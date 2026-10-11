package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Verificação do estágio 13: "branches" está coberto?
//
// Leitura do código: dois passos condition.if IRMÃOS, apontando para o MESMO
// dependente com Expect opostos, são gates que se excluem mutuamente — quem casa
// roda os seus dependentes, quem não casa ignora os seus. Isso É if/else. O que
// estes testes fazem é MEDIR isso, em vez de assumir:
//
//  1. ramo escolhido quando o dependente tem sucesso;
//  2. ramo alternativo quando o dependente NÃO tem sucesso (sem sucesso, sem
//     abortar);
//  3. o limite declarado: falha de um passo com efeito ABORTA o fluxo em vez de
//     desviar para o ramo alternativo. O teste trava esse comportamento para que
//     ele seja uma decisão visível, não folclore.
func branchRuntime(t *testing.T) (*Runtime, *ContextStore, Project) {
	t.Helper()
	return idempotencyRuntime(t)
}

func TestBranchRunsTheSucceededPathWhenDependencySucceeds(t *testing.T) {
	runtime, contextStore, project := branchRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-branch-ok",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		NextRunAt:       time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "portao", Kind: ScheduleStepConditionIf, DependsOn: []string{"gatilho"}, Expect: ScheduleExpectSucceeded},
			{ID: "ramo-sucesso", Kind: ScheduleStepActionMission, DependsOn: []string{"portao"}, Objective: "caminho feliz"},
			{ID: "else", Kind: ScheduleStepConditionIf, DependsOn: []string{"gatilho"}, Expect: ScheduleExpectFailed},
			{ID: "ramo-falha", Kind: ScheduleStepActionMission, DependsOn: []string{"else"}, Objective: "caminho alternativo"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("ramo escolhido não pode falhar: %v", runErr)
	}
	if report.Missions != 1 {
		t.Fatalf("apenas UM ramo pode criar missão, veio %d", report.Missions)
	}
	statuses := map[string]string{}
	for _, step := range report.Steps {
		statuses[step.ID] = step.Status
	}
	if statuses["ramo-sucesso"] != ScheduleStepSucceeded {
		t.Fatalf("o ramo do sucesso deveria rodar: %+v", statuses)
	}
	if statuses["ramo-falha"] != ScheduleStepSkipped {
		t.Fatalf("o ramo alternativo deveria ficar ignorado: %+v", statuses)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].Objective != "caminho feliz" {
		t.Fatalf("a missão criada precisa ser a do ramo escolhido: %+v", missions)
	}
}

func TestBranchRunsTheAlternativePathOnNonSuccess(t *testing.T) {
	runtime, contextStore, project := branchRuntime(t)
	// O dependente é uma aprovação NEGADA: não teve sucesso, mas não é falha
	// técnica — o fluxo continua e o ramo alternativo existe para isso.
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-branch-negado",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		NextRunAt:       time.Date(2026, 10, 11, 21, 30, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "liberar", Kind: ScheduleStepApprovalRequest, DependsOn: []string{"gatilho"}, Objective: "liberar envio"},
			{ID: "portao", Kind: ScheduleStepConditionIf, DependsOn: []string{"liberar"}, Expect: ScheduleExpectSucceeded},
			{ID: "ramo-aprovado", Kind: ScheduleStepActionMission, DependsOn: []string{"portao"}, Objective: "seguir"},
			{ID: "else", Kind: ScheduleStepConditionIf, DependsOn: []string{"liberar"}, Expect: ScheduleExpectFailed},
			{ID: "ramo-negado", Kind: ScheduleStepActionMission, DependsOn: []string{"else"}, Objective: "avisar o solicitante"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	paused, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatal(err)
	}
	// SEGUNDO DEFEITO QUE ESTE TESTE TRAVA: aprovação PENDENTE não é "não teve
	// sucesso". Enquanto ninguém decidiu, o ramo alternativo NÃO pode rodar.
	if paused.Missions != 0 {
		t.Fatalf("com decisão pendente nenhum ramo pode rodar, veio %d missão(ões)", paused.Missions)
	}
	for _, step := range paused.Steps {
		if step.ID == "ramo-negado" && step.Status != ScheduleStepSkipped {
			t.Fatalf("o ramo alternativo não pode rodar antes da decisão: %+v", paused.Steps)
		}
	}
	recorded, err := runtime.RecordScheduleRun(schedule, paused, nil)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := runtime.DecideScheduleApproval(recorded, "liberar", paused.RunKey, false, "ana", "sem verba")
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), denied)
	if runErr != nil {
		t.Fatalf("negativa não é falha técnica: %v", runErr)
	}
	if report.Missions != 1 {
		t.Fatalf("o ramo alternativo deveria criar 1 missão, veio %d (%+v)", report.Missions, report.Steps)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].Objective != "avisar o solicitante" {
		t.Fatalf("a missão precisa ser a do ramo alternativo: %+v", missions)
	}
	for _, step := range report.Steps {
		if step.ID == "ramo-aprovado" && step.Status != ScheduleStepSkipped {
			t.Fatalf("o ramo do sucesso precisa ficar ignorado: %+v", report.Steps)
		}
	}
}

func TestFailingStepAbortsTheFlowInsteadOfBranching(t *testing.T) {
	runtime, contextStore, project := branchRuntime(t)
	// LIMITE DECLARADO: falha de um passo com efeito ABORTA a execução, então o
	// ramo alternativo NÃO é alcançado. Este teste existe para que esse
	// comportamento seja uma decisão explícita e visível, e não uma suposição.
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-branch-aborta",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		IntervalSeconds: 60,
		NextRunAt:       time.Date(2026, 10, 11, 22, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "quebra", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "vai falhar sem projeto"},
			{ID: "else", Kind: ScheduleStepConditionIf, DependsOn: []string{"quebra"}, Expect: ScheduleExpectFailed},
			{ID: "ramo-falha", Kind: ScheduleStepActionMission, DependsOn: []string{"else"}, Objective: "não deve rodar"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = project
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("falha de passo com efeito precisa interromper a execução")
	}
	if !strings.Contains(runErr.Error(), "quebra") {
		t.Fatalf("o erro precisa nomear o passo: %v", runErr)
	}
	if report.Missions != 0 {
		t.Fatalf("o ramo alternativo não pode rodar depois de uma falha: %d", report.Missions)
	}
	// A falha fica visível: fila morta com o passo culpado.
	if len(report.DeadLetters) != 1 || report.DeadLetters[0].StepID != "quebra" {
		t.Fatalf("a falha precisa ficar registrada na fila morta: %+v", report.DeadLetters)
	}
}
