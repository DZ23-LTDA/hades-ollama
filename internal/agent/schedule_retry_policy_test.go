package agent

import (
	"context"
	"errors"
	"testing"
)

// Retry por nó (estágio 13). Só é seguro DEPOIS da idempotência da criação de
// missão: repetir o passo reusa a mesma chave, então uma repetição que encontra a
// missão já criada não duplica. Estes testes travam a política de tentativas, o
// limite declarado e a evidência no relatório.
func TestRunWithRetryStopsAtFirstSuccess(t *testing.T) {
	calls := 0
	attempts, err := runWithRetry(3, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("sucesso na primeira tentativa não deveria errar: %v", err)
	}
	if attempts != 1 || calls != 1 {
		t.Fatalf("esperava 1 tentativa, veio attempts=%d calls=%d", attempts, calls)
	}
}

func TestRunWithRetryRecoversFromTransientFailures(t *testing.T) {
	calls := 0
	attempts, err := runWithRetry(3, func() error {
		calls++
		if calls < 3 {
			return errors.New("falha transitória")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("deveria se recuperar na terceira tentativa: %v", err)
	}
	if attempts != 3 || calls != 3 {
		t.Fatalf("esperava 3 tentativas, veio attempts=%d calls=%d", attempts, calls)
	}
}

func TestRunWithRetryIsBoundedAndReportsTheLastError(t *testing.T) {
	calls := 0
	attempts, err := runWithRetry(3, func() error {
		calls++
		return errors.New("falha permanente " + string(rune('0'+calls)))
	})
	if err == nil {
		t.Fatal("falha permanente precisa devolver erro")
	}
	if attempts != 3 || calls != 3 {
		t.Fatalf("tentativas precisam ser LIMITADAS: attempts=%d calls=%d", attempts, calls)
	}
	// O motivo relatado é o da ÚLTIMA tentativa, não o da primeira.
	if err.Error() != "falha permanente 3" {
		t.Fatalf("erro final deveria ser o da última tentativa, veio %q", err.Error())
	}
}

func TestRunWithRetryTreatsNonPositiveAsSingleAttempt(t *testing.T) {
	for _, declared := range []int{0, -1} {
		calls := 0
		attempts, _ := runWithRetry(declared, func() error {
			calls++
			return errors.New("falha")
		})
		if attempts != 1 || calls != 1 {
			t.Fatalf("maxAttempts=%d deveria virar 1 tentativa, veio attempts=%d calls=%d", declared, attempts, calls)
		}
	}
}

func TestScheduleStepAttemptsDefaultsAndCaps(t *testing.T) {
	if got := scheduleStepAttempts(ScheduleStep{}); got != ScheduleStepDefaultAttempts {
		t.Fatalf("sem declaração o padrão é %d, veio %d", ScheduleStepDefaultAttempts, got)
	}
	if got := scheduleStepAttempts(ScheduleStep{MaxAttempts: 4}); got != 4 {
		t.Fatalf("declaração explícita deveria valer, veio %d", got)
	}
	if got := scheduleStepAttempts(ScheduleStep{MaxAttempts: ScheduleStepMaxAttempts + 9}); got != ScheduleStepMaxAttempts {
		t.Fatalf("acima do teto precisa ser limitado a %d, veio %d", ScheduleStepMaxAttempts, got)
	}
}

func TestValidateScheduleStepsRejectsAttemptsOutsideRange(t *testing.T) {
	base := func(attempts int) []ScheduleStep {
		return []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "x", MaxAttempts: attempts},
		}
	}
	for _, valid := range []int{0, 1, ScheduleStepMaxAttempts} {
		if err := ValidateScheduleSteps(base(valid)); err != nil {
			t.Fatalf("max_attempts=%d deveria ser aceito: %v", valid, err)
		}
	}
	for _, invalid := range []int{-1, ScheduleStepMaxAttempts + 1} {
		if err := ValidateScheduleSteps(base(invalid)); err == nil {
			t.Fatalf("max_attempts=%d deveria ser recusado antes de persistir", invalid)
		}
	}
}

func TestScheduleFlowRetriesFailingActionAndRecordsAttempts(t *testing.T) {
	runtime, _ := scheduleFlowRuntime(t)
	// Sem projeto, a criação falha de forma determinística: serve para medir a
	// política de tentativas e a evidência no relatório.
	schedule := Schedule{
		ID:             "sched-retry",
		OrganizationID: "org-a",
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "vai falhar", MaxAttempts: 3},
		},
	}
	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("ação que não cria missão precisa devolver erro")
	}
	var attempts int
	for _, step := range report.Steps {
		if step.ID == "acao" {
			attempts = step.Attempts
			if step.Status != ScheduleStepFailed {
				t.Fatalf("passo deveria constar como falho: %+v", step)
			}
			if step.Reason == "" {
				t.Fatal("passo falho precisa de motivo")
			}
		}
	}
	if attempts != 3 {
		t.Fatalf("o relatório precisa registrar as 3 tentativas, veio %d", attempts)
	}

	// Sem declaração, o padrão do motor é aplicado.
	schedule.Steps[1].MaxAttempts = 0
	report, err = runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("esperava falha")
	}
	for _, step := range report.Steps {
		if step.ID == "acao" && step.Attempts != ScheduleStepDefaultAttempts {
			t.Fatalf("padrão de tentativas deveria ser %d, veio %d", ScheduleStepDefaultAttempts, step.Attempts)
		}
	}
}
