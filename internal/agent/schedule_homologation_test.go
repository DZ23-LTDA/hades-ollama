package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HOMOLOGAÇÃO DO ESTÁGIO 13 — critério do estágio: "Workflow real ponta a ponta
// com falha simulada, retry, retomada e evidência por nó".
//
// Este teste usa o caminho REAL, não peças isoladas:
//   - ContextStore e JSONStore em DISCO (o estado sobrevive ao processo);
//   - o store de missões falha de forma TRANSITÓRIA (falha simulada no limite de
//     persistência, o tipo de falha que retry existe para absorver);
//   - um SEGUNDO runtime é construído sobre os mesmos diretórios, simulando o
//     reinício do worker, e retoma pelo reprocessamento.
//
// O que este teste trava: retry recupera, a evidência sai POR NÓ, nada duplica e
// a retomada funciona depois do "restart".

// flakyMissionStore falha as primeiras `failures` criações cujo objetivo contenha
// `objectiveFragment`, simulando indisponibilidade transitória do meio de
// persistência. As demais operações delegam ao store real.
type flakyMissionStore struct {
	Store
	objectiveFragment string
	failures          int
	calls             int
}

func (s *flakyMissionStore) CreateMission(mission Mission) error {
	s.calls++
	if s.failures > 0 && strings.Contains(mission.Objective, s.objectiveFragment) {
		s.failures--
		return errors.New("persistência temporariamente indisponível (falha simulada)")
	}
	return s.Store.CreateMission(mission)
}

func TestScheduleFlowHomologationFailureRetryAndWorkerRestart(t *testing.T) {
	contextRoot := t.TempDir()
	storeRoot := t.TempDir()
	workspace := t.TempDir()

	contextStore, err := NewContextStore(contextRoot)
	if err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("projeto de homologação", workspace, "org-a")
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-homolog",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Enabled:         true,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "primeira", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "primeira missão"},
			{
				ID: "segunda", Kind: ScheduleStepActionMission, DependsOn: []string{"primeira"},
				Objective: "segunda missão (transitória)", MaxAttempts: 3,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if schedule.FlowVersion != 1 {
		t.Fatalf("o schedule precisa nascer na versão 1 do fluxo, veio %d", schedule.FlowVersion)
	}

	// ---- ETAPA 1: falha SIMULADA no segundo passo, absorvida pelo retry ----
	flaky := &flakyMissionStore{objectiveFragment: "transitória", failures: 2}
	baseStore, err := NewJSONStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	flaky.Store = baseStore
	workerA, err := NewRuntime(RuntimeConfig{
		Store:         flaky,
		Context:       contextStore,
		Planner:       RulePlanner{},
		WorkspaceRoot: workspace, DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	workerA = workerA.WithOrganization("org-a")

	report, runErr := workerA.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("o retry deveria absorver a falha transitória: %v", runErr)
	}
	if report.FlowVersion != schedule.FlowVersion {
		t.Fatalf("o relatório precisa registrar a versão do fluxo executada (%d != %d)", report.FlowVersion, schedule.FlowVersion)
	}
	var segunda ScheduleStepOutcome
	for _, step := range report.Steps {
		if step.ID == "segunda" {
			segunda = step
		}
	}
	if segunda.Status != ScheduleStepSucceeded {
		t.Fatalf("o passo com retry deveria concluir: %+v", report.Steps)
	}
	if segunda.Attempts != 3 {
		t.Fatalf("a evidência precisa mostrar as 3 tentativas (2 falhas simuladas + 1 sucesso), veio %d", segunda.Attempts)
	}
	if flaky.calls == 0 || flaky.failures != 0 {
		t.Fatalf("a falha simulada não foi exercitada: calls=%d restantes=%d", flaky.calls, flaky.failures)
	}
	if len(report.DeadLetters) != 0 {
		t.Fatalf("passo recuperado NÃO pode ficar na fila morta: %+v", report.DeadLetters)
	}

	// Grava o resultado como o despacho real faz: a evidência vai para o disco.
	recorded, err := workerA.RecordScheduleRun(schedule, report, nil)
	if err != nil {
		t.Fatal(err)
	}
	if recorded.LastRun == nil || len(recorded.LastRun.Steps) != 3 {
		t.Fatalf("evidência por nó não persistiu: %+v", recorded.LastRun)
	}

	// ---- ETAPA 2: REINÍCIO do worker (novos runtime/store sobre o MESMO disco) ----
	restartedContext, err := NewContextStore(contextRoot)
	if err != nil {
		t.Fatal(err)
	}
	restartedStore, err := NewJSONStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	workerB, err := NewRuntime(RuntimeConfig{
		Store:         restartedStore,
		Context:       restartedContext,
		Planner:       RulePlanner{},
		WorkspaceRoot: workspace, DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	workerB = workerB.WithOrganization("org-a")

	reloaded, err := restartedContext.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatalf("o schedule precisa sobreviver ao reinício: %v", err)
	}
	if reloaded.LastRun == nil || reloaded.FlowVersion != schedule.FlowVersion {
		t.Fatalf("estado do fluxo não sobreviveu ao reinício: %+v", reloaded.LastRun)
	}

	// ---- ETAPA 3: RETOMADA pelo reprocessamento, sem duplicar o que já rodou ----
	resumed, err := workerB.ReprocessScheduleFlow(context.Background(), reloaded)
	if err != nil {
		t.Fatalf("retomada após reinício: %v", err)
	}
	if resumed.Missions != 0 {
		t.Fatalf("retomada não pode recriar o que já foi criado, veio %d", resumed.Missions)
	}
	if resumed.RunKey != reloaded.LastRun.RunKey {
		t.Fatalf("a retomada precisa reusar a execução original (%q != %q)", resumed.RunKey, reloaded.LastRun.RunKey)
	}
	for _, step := range resumed.Steps {
		if step.Status != ScheduleStepSucceeded {
			t.Fatalf("após a retomada todo passo precisa constar como bem-sucedido: %+v", resumed.Steps)
		}
	}
	// O store REAL sobreviveu ao reinício: as missões estão lá, sem duplicata.
	missions, err := restartedStore.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 2 {
		t.Fatalf("a homologação exige 2 missões (uma por passo), veio %d", len(missions))
	}

	// ---- ETAPA 4: a versão do fluxo muda SÓ quando o grafo muda ----
	edited := reloaded
	edited.Steps = append(edited.Steps, ScheduleStep{
		ID: "terceira", Kind: ScheduleStepActionMission, DependsOn: []string{"segunda"}, Objective: "terceira",
	})
	updated, err := restartedContext.UpdateSchedule(edited.ID, edited)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FlowVersion != schedule.FlowVersion+1 {
		t.Fatalf("mudar o grafo precisa subir a versão (%d -> %d)", schedule.FlowVersion, updated.FlowVersion)
	}
	// Edição que NÃO muda o grafo não cria versão nova.
	toggled := updated
	toggled.Objective = "corpo editado"
	toggled.Enabled = false
	sameFlow, err := restartedContext.UpdateSchedule(toggled.ID, toggled)
	if err != nil {
		t.Fatal(err)
	}
	if sameFlow.FlowVersion != updated.FlowVersion {
		t.Fatalf("editar objetivo/estado NÃO é mudança de fluxo (%d -> %d)", updated.FlowVersion, sameFlow.FlowVersion)
	}

	// ---- ETAPA 5: nenhum arquivo de estado ficou para trás ----
	for _, name := range []string{"schedules/" + schedule.ID + ".json"} {
		if _, statErr := os.Stat(filepath.Join(contextRoot, filepath.FromSlash(name))); statErr != nil {
			t.Fatalf("estado do schedule ausente no disco: %v", statErr)
		}
	}
}
