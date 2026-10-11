package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Nó de aprovação (estágio 13): o fluxo PAUSA até decisão humana registrada para
// (passo, execução). As três regras que estes testes travam: sem decisão o
// dependente NÃO roda; a negativa não libera nada e não vira falha técnica; e uma
// aprovação NÃO vaza para uma execução futura.
func approvalSchedule(t *testing.T) (*Runtime, *ContextStore, Schedule) {
	t.Helper()
	runtime, contextStore, project := idempotencyRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-aprovacao",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		NextRunAt:       time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "liberar", Kind: ScheduleStepApprovalRequest, DependsOn: []string{"gatilho"}, Objective: "liberar o envio de {{schedule.id}}"},
			{ID: "enviar", Kind: ScheduleStepActionMission, DependsOn: []string{"liberar"}, Objective: "enviar o comunicado"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime, contextStore, schedule
}

func TestApprovalNodePausesFlowWithoutDecision(t *testing.T) {
	runtime, contextStore, schedule := approvalSchedule(t)
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("espera por aprovação não é falha: %v", runErr)
	}
	if report.Missions != 0 {
		t.Fatalf("sem aprovação nada pode ser criado, veio %d missão(ões)", report.Missions)
	}
	var approvalStep, dependentStep ScheduleStepOutcome
	for _, step := range report.Steps {
		switch step.ID {
		case "liberar":
			approvalStep = step
		case "enviar":
			dependentStep = step
		}
	}
	if approvalStep.Status != ScheduleStepPending {
		t.Fatalf("o passo de aprovação precisa ficar PENDENTE, veio %q", approvalStep.Status)
	}
	if dependentStep.Status != ScheduleStepSkipped {
		t.Fatalf("o dependente não pode rodar sem aprovação, veio %q", dependentStep.Status)
	}
	if len(report.DeadLetters) != 0 {
		t.Fatalf("espera por aprovação não entra na fila morta: %+v", report.DeadLetters)
	}
	if len(report.PendingApprovals) != 1 || report.PendingApprovals[0].StepID != "liberar" {
		t.Fatalf("a pendência precisa ser registrada: %+v", report.PendingApprovals)
	}
	// A pausa SOBREVIVE ao processo: está no schedule.
	stored, err := runtime.RecordScheduleRun(schedule, report, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.PendingApprovals) != 1 {
		t.Fatalf("pendência não persistiu: %+v", stored.PendingApprovals)
	}
	reloaded, err := contextStore.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.PendingApprovals) != 1 || reloaded.PendingApprovals[0].Objective == "" {
		t.Fatalf("a pendência precisa descrever a decisão pedida: %+v", reloaded.PendingApprovals)
	}
}

func TestApprovalDecisionUnlocksAndDenialBlocksWithoutDeadLetter(t *testing.T) {
	runtime, contextStore, schedule := approvalSchedule(t)
	paused, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := runtime.RecordScheduleRun(schedule, paused, nil)
	if err != nil {
		t.Fatal(err)
	}
	runKey := paused.RunKey

	// Decisão NEGADA: o dependente continua barrado, a pendência sai da lista e
	// NADA entra na fila morta (negativa é decisão humana, não falha técnica).
	denied, err := runtime.DecideScheduleApproval(stored, "liberar", runKey, false, "ana", "dados incompletos")
	if err != nil {
		t.Fatal(err)
	}
	if len(denied.PendingApprovals) != 0 {
		t.Fatalf("com decisão não há mais pendência: %+v", denied.PendingApprovals)
	}
	reportDenied, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), denied)
	if runErr != nil {
		t.Fatalf("negativa não é falha técnica: %v", runErr)
	}
	if reportDenied.Missions != 0 {
		t.Fatalf("negativa não pode liberar o dependente: %d missão(ões)", reportDenied.Missions)
	}
	for _, step := range reportDenied.Steps {
		if step.ID == "liberar" {
			if step.Status != ScheduleStepSkipped {
				t.Fatalf("negativa precisa constar como não-liberada, veio %q", step.Status)
			}
			if !strings.Contains(step.Reason, "ana") || !strings.Contains(step.Reason, "dados incompletos") {
				t.Fatalf("a negativa precisa carregar ator e motivo: %q", step.Reason)
			}
		}
	}
	if len(reportDenied.DeadLetters) != 0 {
		t.Fatalf("negativa não entra na fila morta: %+v", reportDenied.DeadLetters)
	}

	// Decisão APROVADA: agora o dependente roda.
	approved, err := runtime.DecideScheduleApproval(stored, "liberar", runKey, true, "bia", "ok")
	if err != nil {
		t.Fatal(err)
	}
	reportApproved, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), approved)
	if runErr != nil {
		t.Fatalf("após aprovação o fluxo deveria rodar: %v", runErr)
	}
	if reportApproved.Missions != 1 {
		t.Fatalf("após aprovação o dependente deveria criar a missão, veio %d", reportApproved.Missions)
	}
	for _, step := range reportApproved.Steps {
		if step.ID == "liberar" && (step.Status != ScheduleStepSucceeded || !strings.Contains(step.Reason, "bia")) {
			t.Fatalf("aprovação precisa constar com quem decidiu: %+v", step)
		}
	}
	// E a decisão ficou auditável no store.
	final, err := contextStore.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Approvals) == 0 {
		t.Fatal("a decisão precisa ficar registrada para auditoria")
	}
	last := final.Approvals[len(final.Approvals)-1]
	if !last.Approved || last.ActorID != "bia" || last.RunKey != runKey {
		t.Fatalf("decisão registrada incorreta: %+v", last)
	}
}

func TestApprovalDoesNotLeakToAFutureExecution(t *testing.T) {
	runtime, _, schedule := approvalSchedule(t)
	paused, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := runtime.RecordScheduleRun(schedule, paused, nil)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := runtime.DecideScheduleApproval(stored, "liberar", paused.RunKey, true, "bia", "ok")
	if err != nil {
		t.Fatal(err)
	}
	// A MESMA aprovação não pode valer para uma execução diferente: o tick seguinte
	// tem outra chave e precisa pedir decisão de novo.
	future := approved
	future.NextRunAt = approved.NextRunAt.Add(time.Hour)
	futureReport, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), future)
	if runErr != nil {
		t.Fatal(runErr)
	}
	if futureReport.RunKey == paused.RunKey {
		t.Fatal("a execução futura precisa ter chave própria")
	}
	for _, step := range futureReport.Steps {
		if step.ID == "liberar" && step.Status != ScheduleStepPending {
			t.Fatalf("aprovação antiga não pode liberar uma execução futura: %+v", step)
		}
	}
	if futureReport.Missions != 0 {
		t.Fatalf("execução futura sem decisão nova não cria missão, veio %d", futureReport.Missions)
	}
}

func TestValidateScheduleApprovalRejectsBadShapes(t *testing.T) {
	base := func(step ScheduleStep) []ScheduleStep {
		return []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			step,
		}
	}
	valid := ScheduleStep{ID: "ap", Kind: ScheduleStepApprovalRequest, DependsOn: []string{"gatilho"}, Objective: "liberar"}
	if err := ValidateScheduleSteps(base(valid)); err != nil {
		t.Fatalf("aprovação válida deveria ser aceita: %v", err)
	}
	semDependencia := valid
	semDependencia.DependsOn = nil
	if err := ValidateScheduleSteps(base(semDependencia)); err == nil {
		t.Fatal("aprovação sem dependência precisa ser recusada")
	}
	semObjetivo := valid
	semObjetivo.Objective = ""
	if err := ValidateScheduleSteps(base(semObjetivo)); err == nil {
		t.Fatal("aprovação sem objetivo precisa ser recusada")
	}
	comItens := valid
	comItens.Items = []string{"x"}
	if err := ValidateScheduleSteps(base(comItens)); err == nil {
		t.Fatal("aprovação não carrega itens de laço")
	}
}
