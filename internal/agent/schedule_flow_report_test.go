package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Achado do estágio 13 ("não declarar sucesso com nós ignorados" + "logs por
// nó"): `ExecuteScheduleFlow` devolvia apenas a CONTAGEM de missões. Um fluxo
// cujo nó condicional era falso reportava sucesso sem registrar que passos não
// rodaram, e a execução bem-sucedida nem era persistida
// (`recordScheduleSuccess` saía cedo sem gravar `last_run_at`). O supervisor,
// no outro ponto de disparo, não gravava nada — uma falha de fluxo desaparecia.
//
// Estes testes travam o relatório por nó, a persistência em SUCESSO e a
// nomeação do passo que falhou.
func scheduleFlowRuntime(t *testing.T) (*Runtime, *ContextStore) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Context: store, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return runtime.WithOrganization("org-a"), store
}

func TestScheduleFlowReportMarksSkippedSteps(t *testing.T) {
	runtime, _ := scheduleFlowRuntime(t)
	// O gatilho sempre roda; a condição espera FALHA do gatilho (que teve
	// sucesso), então fica IGNORADA; a ação depende da condição e também é
	// ignorada. Antes, isso era "sucesso" sem nenhum registro.
	schedule := Schedule{
		ID:             "sched-skip",
		OrganizationID: "org-a",
		Objective:      "corpo",
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "condicao", Kind: ScheduleStepConditionIf, DependsOn: []string{"gatilho"}, Expect: ScheduleExpectFailed},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"condicao"}, Objective: "nunca roda"},
		},
	}
	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatalf("fluxo sem ação executada não deveria falhar: %v", err)
	}
	if report.Missions != 0 {
		t.Fatalf("nenhuma missão deveria ser criada, veio %d", report.Missions)
	}
	status := map[string]string{}
	for _, step := range report.Steps {
		status[step.ID] = step.Status
	}
	if status["gatilho"] != ScheduleStepSucceeded {
		t.Fatalf("gatilho deveria ter sucesso: %+v", report.Steps)
	}
	if status["condicao"] != ScheduleStepSkipped || status["acao"] != ScheduleStepSkipped {
		t.Fatalf("passos sem execução precisam aparecer como ignorados: %+v", report.Steps)
	}
	if skipped := report.SkippedSteps(); len(skipped) != 2 || skipped[0] != "acao" || skipped[1] != "condicao" {
		t.Fatalf("SkippedSteps = %v", skipped)
	}
	for _, step := range report.Steps {
		if step.Status == ScheduleStepSkipped && strings.TrimSpace(step.Reason) == "" {
			t.Fatalf("passo ignorado precisa de motivo: %+v", step)
		}
	}
}

func TestScheduleFlowReportKeepsFailureEvidence(t *testing.T) {
	runtime, _ := scheduleFlowRuntime(t)
	// Sem planejador, a criação da missão falha — é o caminho de erro real.
	schedule := Schedule{
		ID:             "sched-falha",
		OrganizationID: "org-a",
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "vai falhar"},
		},
	}
	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("ação que não cria missão precisa devolver erro")
	}
	if report.FailedStep() != "acao" {
		t.Fatalf("relatório deveria apontar o passo que falhou: %+v", report.Steps)
	}
	if report.FinishedAt.IsZero() {
		t.Fatal("relatório de falha precisa de FinishedAt")
	}
	for _, step := range report.Steps {
		if step.ID == "acao" {
			if step.Status != ScheduleStepFailed || strings.TrimSpace(step.Reason) == "" {
				t.Fatalf("passo falho precisa de status e motivo: %+v", step)
			}
		}
	}
	// O wrapper histórico continua retornando apenas a contagem.
	if missions, wrapperErr := runtime.ExecuteScheduleFlow(context.Background(), schedule); wrapperErr == nil || missions != 0 {
		t.Fatalf("wrapper = (%d,%v), esperado (0, erro)", missions, wrapperErr)
	}
}

func TestRecordScheduleRunPersistsSuccessEvidence(t *testing.T) {
	runtime, store := scheduleFlowRuntime(t)
	schedule, err := store.CreateSchedule(Schedule{ID: "sched-ok", Objective: "objetivo", OrganizationID: "org-a", IntervalSeconds: 60, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	report := ScheduleRunReport{
		FinishedAt: time.Now().UTC(),
		Missions:   1,
		Steps: []ScheduleStepOutcome{
			{ID: "gatilho", Kind: string(ScheduleStepTriggerInterval), Status: ScheduleStepSucceeded},
			{ID: "condicao", Kind: string(ScheduleStepConditionIf), Status: ScheduleStepSkipped, Reason: "condição não satisfeita"},
		},
	}
	updated, err := runtime.recordScheduleRun(schedule, report, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.LastRunAt == nil {
		t.Fatal("execução bem-sucedida PRECISA registrar last_run_at (antes não registrava nada)")
	}
	if updated.LastRun == nil || len(updated.LastRun.Steps) != 2 {
		t.Fatalf("relatório por nó não persistiu: %+v", updated.LastRun)
	}
	if skipped := updated.LastRun.SkippedSteps(); len(skipped) != 1 || skipped[0] != "condicao" {
		t.Fatalf("passo ignorado não persistiu: %v", skipped)
	}

	// E sobrevive ao round-trip do store.
	reloaded, err := store.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.LastRun == nil || reloaded.LastRunAt == nil {
		t.Fatalf("evidência da execução se perdeu ao recarregar: %+v", reloaded)
	}
}

func TestRecordScheduleRunNamesTheFailingStep(t *testing.T) {
	runtime, store := scheduleFlowRuntime(t)
	schedule, err := store.CreateSchedule(Schedule{
		ID:              "sched-nome",
		OrganizationID:  "org-a",
		Objective:       "objetivo",
		IntervalSeconds: 60,
		Enabled:         true,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := ScheduleRunReport{
		FinishedAt: time.Now().UTC(),
		Steps: []ScheduleStepOutcome{
			{ID: "acao", Kind: string(ScheduleStepActionMission), Status: ScheduleStepFailed, Reason: "planner indisponível"},
		},
	}
	updated, err := runtime.recordScheduleRun(schedule, report, context.DeadlineExceeded)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FailureCount != 1 {
		t.Fatalf("FailureCount = %d", updated.FailureCount)
	}
	if updated.LastFailureCode != "mission_creation_failed:acao" {
		t.Fatalf("código de falha deveria nomear o passo, veio %q", updated.LastFailureCode)
	}
	if !updated.NextRunAt.After(time.Now().UTC()) {
		t.Fatal("falha deveria reagendar com atraso")
	}
	// O caminho legado (sem grafo) mantém o código histórico, sem sufixo.
	legacy, err := store.CreateSchedule(Schedule{ID: "sched-legado", Objective: "objetivo legado", OrganizationID: "org-a", IntervalSeconds: 60, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	legacyReport := ScheduleRunReport{Steps: []ScheduleStepOutcome{{ID: "legacy-objective", Kind: string(ScheduleStepActionMission), Status: ScheduleStepFailed}}}
	legacyUpdated, err := runtime.recordScheduleRun(legacy, legacyReport, context.DeadlineExceeded)
	if err != nil {
		t.Fatal(err)
	}
	if legacyUpdated.LastFailureCode != "mission_creation_failed" {
		t.Fatalf("caminho legado deveria manter o código histórico, veio %q", legacyUpdated.LastFailureCode)
	}
}
