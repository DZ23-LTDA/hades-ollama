package agent

import (
	"context"
	"testing"
)

// DLQ por nó (estágio 13): o passo que esgota as tentativas fica registrado com o
// motivo final, o registro é LIMITADO e sai da lista quando o passo se recupera —
// inclusive via reprocessamento.
func TestFailedStepIsRegisteredInDeadLetters(t *testing.T) {
	runtime, contextStore := scheduleFlowRuntime(t)
	// Sem projeto a criação falha de forma determinística.
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-dlq",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		IntervalSeconds: 60,
		Enabled:         true,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "vai falhar", MaxAttempts: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("esperava falha")
	}
	if len(report.DeadLetters) != 1 {
		t.Fatalf("o passo que esgotou as tentativas precisa entrar na DLQ: %+v", report.DeadLetters)
	}
	letter := report.DeadLetters[0]
	if letter.StepID != "acao" || letter.Reason == "" || letter.Attempts != 2 {
		t.Fatalf("entrada de DLQ incompleta: %+v", letter)
	}
	if letter.FailedAt.IsZero() {
		t.Fatal("a entrada precisa registrar QUANDO falhou")
	}

	// E o gravador PERSISTE a DLQ no schedule, junto do resto da evidência.
	stored, err := runtime.RecordScheduleRun(schedule, report, runErr)
	if err != nil {
		t.Fatalf("registro: %v", err)
	}
	if len(stored.DeadLetters) != 1 || stored.DeadLetters[0].StepID != "acao" {
		t.Fatalf("DLQ não persistiu no schedule: %+v", stored.DeadLetters)
	}
}

func TestDeadLettersAreBoundedAndKeepTheNewest(t *testing.T) {
	existing := make([]ScheduleStepDeadLetter, 0, maxScheduleDeadLetters)
	added := make([]ScheduleStepDeadLetter, 0, 5)
	for i := range 5 {
		existing = append(existing, ScheduleStepDeadLetter{StepID: "antigo" + string(rune('a'+i))})
	}
	for i := range 5 {
		added = append(added, ScheduleStepDeadLetter{StepID: "novo" + string(rune('a'+i))})
	}
	merged := mergeScheduleDeadLetters(existing, added, nil)
	if len(merged) != 10 {
		t.Fatalf("sem excedente nada pode ser descartado, veio %d", len(merged))
	}
	if merged[0].StepID != "novoa" {
		t.Fatalf("as falhas mais recentes precisam vir primeiro, veio %q", merged[0].StepID)
	}

	// Excedente é cortado pelo teto, sem crescer sem fim.
	big := make([]ScheduleStepDeadLetter, 0, maxScheduleDeadLetters+10)
	for i := range maxScheduleDeadLetters + 10 {
		big = append(big, ScheduleStepDeadLetter{StepID: "passo" + string(rune('a'+i%26)) + string(rune('0'+i/26))})
	}
	capped := mergeScheduleDeadLetters(big, nil, nil)
	if len(capped) > maxScheduleDeadLetters {
		t.Fatalf("a DLQ precisa ser limitada a %d, veio %d", maxScheduleDeadLetters, len(capped))
	}
}

func TestRecoveredStepLeavesTheDeadLetterQueue(t *testing.T) {
	runtime, contextStore, project := idempotencyRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-dlq-recupera",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Enabled:         true,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "missão"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 1) Falha: o schedule perde o projeto e a criação falha.
	broken := schedule
	broken.ProjectID = ""
	if _, err := contextStore.UpdateSchedule(schedule.ID, broken); err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), broken)
	if runErr == nil {
		t.Fatal("esperava falha sem projeto")
	}
	stored, err := runtime.RecordScheduleRun(broken, report, runErr)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.DeadLetters) != 1 {
		t.Fatalf("a falha deveria entrar na DLQ: %+v", stored.DeadLetters)
	}

	// 2) O operador corrige o schedule e o reprocessamento recupera o passo: a
	// entrada PRECISA sair, senão a DLQ mentiria sobre o estado atual.
	fixed := stored
	fixed.ProjectID = project.ID
	if _, err := contextStore.UpdateSchedule(fixed.ID, fixed); err != nil {
		t.Fatal(err)
	}
	report2, err := runtime.ReprocessScheduleFlow(context.Background(), fixed)
	if err != nil {
		t.Fatalf("reprocessamento: %v", err)
	}
	if len(report2.DeadLetters) != 0 {
		t.Fatalf("execução recuperada não pode produzir entrada de DLQ: %+v", report2.DeadLetters)
	}
	recovered, err := runtime.RecordScheduleRun(fixed, report2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.DeadLetters) != 0 {
		t.Fatalf("passo recuperado precisa SAIR da DLQ, veio %+v", recovered.DeadLetters)
	}
}
