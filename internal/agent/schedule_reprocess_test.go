package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Reprocessamento (estágio 13): reexecutar APENAS o que ficou ignorado ou falhou,
// reusando a chave da execução anterior. Os passos que já tiveram sucesso são
// preservados e não criam missão nova — a idempotência é o que permite repetir sem
// duplicar.
func reprocessSchedule(t *testing.T, expectSkippedBranch string) (Schedule, ScheduleRunReport) {
	t.Helper()
	return Schedule{
		ID:             "sched-reproc",
		OrganizationID: "org-a",
		NextRunAt:      time.Date(2026, 10, 11, 6, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "primeira missão"},
			// A condição espera FALHA de um passo que teve SUCESSO: fica ignorada, e
			// o passo dependente dela fica bloqueado.
			{ID: "condicao", Kind: ScheduleStepConditionIf, DependsOn: []string{"acao"}, Expect: expectSkippedBranch},
			{ID: "acao2", Kind: ScheduleStepActionMission, DependsOn: []string{"condicao"}, Objective: "segunda missão"},
		},
	}, ScheduleRunReport{}
}

func TestReprocessScheduleFlowRequiresPreviousRun(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	schedule, _ := reprocessSchedule(t, ScheduleExpectFailed)
	schedule.ProjectID = project.ID
	if _, err := runtime.ReprocessScheduleFlow(context.Background(), schedule); !errors.Is(err, ErrScheduleReprocessUnavailable) {
		t.Fatalf("sem execução anterior o reprocessamento precisa ser recusado, veio %v", err)
	}
	// E nada foi criado pela tentativa recusada.
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 0 {
		t.Fatalf("reprocessamento recusado não pode criar missão: %d", len(missions))
	}
}

func TestReprocessScheduleFlowPreservesSucceededAndRetriesSkipped(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	schedule, _ := reprocessSchedule(t, ScheduleExpectFailed)
	schedule.ProjectID = project.ID

	first, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatalf("primeira execução: %v", err)
	}
	if first.Missions != 1 {
		t.Fatalf("primeira execução deveria criar 1 missão, veio %d (%+v)", first.Missions, first.Steps)
	}
	if len(first.SkippedSteps()) != 2 {
		t.Fatalf("condição e ação dependente deveriam ficar ignoradas: %+v", first.Steps)
	}
	schedule.LastRun = &first
	schedule.LastRunAt = &first.FinishedAt

	// Sem mudar o grafo, o reprocessamento NÃO cria missão nova: a ação já teve
	// sucesso (preservada) e o ramo ignorado continua ignorado.
	second, err := runtime.ReprocessScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("reprocessamento: %v", err)
	}
	if second.Missions != 0 {
		t.Fatalf("reprocessamento sem mudança não pode criar missão, veio %d", second.Missions)
	}
	if second.RunKey != first.RunKey {
		t.Fatalf("o reprocessamento precisa REUSAR a chave da execução (%q != %q)", second.RunKey, first.RunKey)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 {
		t.Fatalf("a ação preservada não pode ser duplicada: %d missões", len(missions))
	}
	var acaoPreservada bool
	for _, step := range second.Steps {
		if step.ID == "acao" {
			acaoPreservada = step.Status == ScheduleStepSucceeded
		}
	}
	if !acaoPreservada {
		t.Fatalf("o passo já bem-sucedido precisa aparecer preservado no novo relatório: %+v", second.Steps)
	}

	// Agora o operador corrige a condição: o passo que estava ignorado passa a
	// poder rodar, e o reprocessamento o executa.
	schedule.Steps[2].Expect = ScheduleExpectSucceeded
	schedule.LastRun = &second
	third, err := runtime.ReprocessScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("reprocessamento após correção: %v", err)
	}
	if third.Missions != 1 {
		t.Fatalf("o passo antes ignorado deveria rodar agora, veio %d missão(ões) (%+v)", third.Missions, third.Steps)
	}
	missions, err = runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 2 {
		t.Fatalf("esperava 2 missões (a original + a do ramo corrigido), veio %d", len(missions))
	}
	if third.RunKey != first.RunKey {
		t.Fatal("a chave da execução precisa continuar sendo a mesma durante o reprocessamento")
	}
}

func TestReprocessScheduleFlowRunsStepsAddedAfterTheRun(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	schedule, _ := reprocessSchedule(t, ScheduleExpectSucceeded)
	schedule.ProjectID = project.ID
	// Só o gatilho e uma ação, para a execução inicial ser simples.
	schedule.Steps = []ScheduleStep{
		{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
		{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "original"},
	}
	first, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatal(err)
	}
	schedule.LastRun = &first

	// Grafo editado: um passo NOVO, sem resultado anterior, precisa rodar.
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "acao-nova", Kind: ScheduleStepActionMission, DependsOn: []string{"acao"}, Objective: "nova",
	})
	second, err := runtime.ReprocessScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("reprocessamento com passo novo: %v", err)
	}
	if second.Missions != 1 {
		t.Fatalf("o passo novo deveria criar 1 missão, veio %d (%+v)", second.Missions, second.Steps)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 2 {
		t.Fatalf("esperava as 2 missões (original preservada + nova), veio %d", len(missions))
	}
}
