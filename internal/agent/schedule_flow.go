package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ScheduleStepKind is the closed vocabulary of steps a persisted schedule flow
// may contain. The values mirror the node types published by the flow editor
// (app/ui/app/src/lib/flowGraph.ts), so a stored graph round-trips without a
// translation table and without vocabulary drift.
type ScheduleStepKind string

const (
	// ScheduleStepTriggerInterval starts the flow when the schedule interval
	// elapses. The interval itself stays in Schedule.IntervalSeconds/NextRunAt;
	// the step exists so the persisted graph keeps the editor's shape.
	ScheduleStepTriggerInterval ScheduleStepKind = "trigger.interval"
	// ScheduleStepTriggerWebhook starts the flow from an authenticated webhook
	// delivery instead of the interval.
	ScheduleStepTriggerWebhook ScheduleStepKind = "trigger.webhook"
	// ScheduleStepActionMission instantiates exactly one mission with the step
	// objective. It is the only step kind with an external side effect.
	ScheduleStepActionMission ScheduleStepKind = "action.mission"
	// ScheduleStepConditionIf gates its dependents on the recorded outcome of
	// one dependency, without any expression language to evaluate.
	ScheduleStepConditionIf ScheduleStepKind = "condition.if"
)

// Outcomes a condition.if step can expect from its single dependency.
const (
	ScheduleExpectSucceeded = "succeeded"
	ScheduleExpectFailed    = "failed"
)

// maxScheduleFlowSteps bounds one persisted graph so a stored flow stays
// reviewable and a dispatch tick keeps a predictable cost.
const maxScheduleFlowSteps = 12

// Retry por passo. O padrão é 2 (uma repetição) porque a criação da missão é
// idempotente — repetir não cria missão duplicada — e uma falha transitória não
// deve abortar o fluxo inteiro. O teto impede laço longo num motor que roda a
// cada tick.
const (
	ScheduleStepDefaultAttempts = 2
	ScheduleStepMaxAttempts     = 5
	// scheduleStepRetryBaseDelay é o atraso base entre tentativas (linear e
	// limitado): o motor é local e roda em tick, então esperas longas travariam
	// o despacho.
	scheduleStepRetryBaseDelay = 50 * time.Millisecond
	scheduleStepRetryMaxDelay  = 500 * time.Millisecond
)

// scheduleStepAttempts resolve o número de tentativas declarado, aplicando o
// padrão e o teto. Valor acima do teto é recusado na validação, não silenciado.
func scheduleStepAttempts(step ScheduleStep) int {
	if step.MaxAttempts <= 0 {
		return ScheduleStepDefaultAttempts
	}
	if step.MaxAttempts > ScheduleStepMaxAttempts {
		return ScheduleStepMaxAttempts
	}
	return step.MaxAttempts
}

// runWithRetry executa a ação até maxAttempts e devolve quantas tentativas fez.
// O erro devolvido é o da ÚLTIMA tentativa: o relatório mostra o motivo real da
// falha final, e não o primeiro erro de uma sequência que se recuperou.
func runWithRetry(maxAttempts int, attempt func() error) (int, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var err error
	for try := 1; try <= maxAttempts; try++ {
		err = attempt()
		if err == nil {
			return try, nil
		}
		if try < maxAttempts {
			delay := time.Duration(try) * scheduleStepRetryBaseDelay
			if delay > scheduleStepRetryMaxDelay {
				delay = scheduleStepRetryMaxDelay
			}
			time.Sleep(delay)
		}
	}
	return maxAttempts, err
}

// ErrUnknownScheduleStep reports a step kind outside the closed vocabulary. It
// is fail-closed: an unrecognised node never executes.
var ErrUnknownScheduleStep = errors.New("unknown schedule step kind")

// ScheduleStep is one node of a persisted schedule flow. An empty Steps slice on
// Schedule keeps the historical behaviour: one mission carrying the schedule
// objective.
type ScheduleStep struct {
	ID string `json:"id"`
	// Kind is validated against the closed vocabulary before the graph is
	// stored and again before it runs.
	Kind ScheduleStepKind `json:"kind"`
	// DependsOn lists the steps that must have succeeded before this one runs.
	DependsOn []string `json:"depends_on,omitempty"`
	// Objective is the mission objective for kind action.mission.
	Objective string `json:"objective,omitempty"`
	// Expect is the required dependency outcome for kind condition.if, either
	// ScheduleExpectSucceeded or ScheduleExpectFailed.
	Expect string `json:"expect,omitempty"`
	// MaxAttempts é quantas vezes um passo action.mission pode tentar antes de
	// falhar o fluxo. Vazio/0 usa o padrão do motor (ScheduleStepDefaultAttempts);
	// o teto ScheduleStepMaxAttempts é validado antes de persistir. Retry é seguro
	// porque a criação da missão é idempotente: repetir não duplica.
	MaxAttempts int `json:"max_attempts,omitempty"`
}

// ValidateScheduleSteps accepts an empty graph (legacy single-objective
// schedule) and otherwise rejects every malformed graph: unknown kinds, missing
// or duplicated identifiers, more than one trigger, unreachable steps and
// cycles. Validation is fail-closed so nothing partial is ever persisted.
func ValidateScheduleSteps(steps []ScheduleStep) error {
	if len(steps) == 0 {
		return nil
	}
	if len(steps) > maxScheduleFlowSteps {
		return fmt.Errorf("schedule flow accepts at most %d steps, got %d", maxScheduleFlowSteps, len(steps))
	}
	byID := make(map[string]ScheduleStep, len(steps))
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		if id == "" {
			return errors.New("schedule step id is required")
		}
		if _, exists := byID[id]; exists {
			return fmt.Errorf("schedule step id %q is duplicated", id)
		}
		if err := validateScheduleStepShape(step); err != nil {
			return err
		}
		byID[id] = step
	}
	triggers := 0
	for _, step := range steps {
		if isScheduleTrigger(step.Kind) {
			triggers++
		}
	}
	if triggers != 1 {
		return fmt.Errorf("schedule flow requires exactly one trigger, got %d", triggers)
	}
	for _, step := range steps {
		for _, dependency := range step.DependsOn {
			dep := strings.TrimSpace(dependency)
			if dep == strings.TrimSpace(step.ID) {
				return fmt.Errorf("schedule step %q depends on itself", strings.TrimSpace(step.ID))
			}
			if _, exists := byID[dep]; !exists {
				return fmt.Errorf("schedule step %q depends on unknown step %q", strings.TrimSpace(step.ID), dep)
			}
		}
	}
	if _, err := orderScheduleSteps(steps); err != nil {
		return err
	}
	return nil
}

func validateScheduleStepShape(step ScheduleStep) error {
	id := strings.TrimSpace(step.ID)
	switch step.Kind {
	case ScheduleStepTriggerInterval, ScheduleStepTriggerWebhook:
		if len(step.DependsOn) != 0 {
			return fmt.Errorf("schedule trigger %q cannot depend on another step", id)
		}
		if strings.TrimSpace(step.Objective) != "" || strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule trigger %q cannot carry an objective or an expectation", id)
		}
	case ScheduleStepActionMission:
		if strings.TrimSpace(step.Objective) == "" {
			return fmt.Errorf("schedule action %q requires an objective", id)
		}
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule action %q requires at least one dependency", id)
		}
		if strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule action %q cannot carry an expectation", id)
		}
		if step.MaxAttempts < 0 || step.MaxAttempts > ScheduleStepMaxAttempts {
			return fmt.Errorf("schedule action %q accepts at most %d attempts, got %d", id, ScheduleStepMaxAttempts, step.MaxAttempts)
		}
	case ScheduleStepConditionIf:
		if len(step.DependsOn) != 1 {
			return fmt.Errorf("schedule condition %q requires exactly one dependency, got %d", id, len(step.DependsOn))
		}
		if strings.TrimSpace(step.Objective) != "" {
			return fmt.Errorf("schedule condition %q cannot carry an objective", id)
		}
		switch strings.TrimSpace(step.Expect) {
		case ScheduleExpectSucceeded, ScheduleExpectFailed:
		default:
			return fmt.Errorf("schedule condition %q requires expect %q or %q", id, ScheduleExpectSucceeded, ScheduleExpectFailed)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnknownScheduleStep, step.Kind)
	}
	return nil
}

func isScheduleTrigger(kind ScheduleStepKind) bool {
	return kind == ScheduleStepTriggerInterval || kind == ScheduleStepTriggerWebhook
}

// orderScheduleSteps returns a deterministic topological order of the graph and
// fails when the graph contains a cycle or a step unreachable from the trigger.
func orderScheduleSteps(steps []ScheduleStep) ([]ScheduleStep, error) {
	byID := make(map[string]ScheduleStep, len(steps))
	pending := make(map[string]int, len(steps))
	dependents := make(map[string][]string, len(steps))
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		byID[id] = step
		pending[id] = 0
	}
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		for _, dependency := range step.DependsOn {
			dep := strings.TrimSpace(dependency)
			if _, exists := byID[dep]; !exists {
				return nil, fmt.Errorf("schedule step %q depends on unknown step %q", id, dep)
			}
			pending[id]++
			dependents[dep] = append(dependents[dep], id)
		}
	}
	ready := make([]string, 0, len(steps))
	for id, remaining := range pending {
		if remaining == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]ScheduleStep, 0, len(steps))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, byID[id])
		for _, dependent := range dependents[id] {
			pending[dependent]--
			if pending[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(steps) {
		return nil, errors.New("schedule flow must be acyclic and every step must depend on the flow trigger")
	}
	return order, nil
}

// Status possíveis de um passo em uma execução. "skipped" existe para que um nó
// que não rodou (condição falsa, dependência que não teve sucesso) apareça no
// relatório: o estágio 13 proíbe declarar sucesso com nó ignorado.
const (
	ScheduleStepSucceeded = "succeeded"
	ScheduleStepSkipped   = "skipped"
	ScheduleStepFailed    = "failed"
)

// ScheduleStepOutcome é o resultado de UM passo da execução.
type ScheduleStepOutcome struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	// Attempts é quantas tentativas o passo consumiu (retry por nó). 0 significa
	// "não se aplica" (gatilho/condição, que não tentam de novo).
	Attempts int `json:"attempts,omitempty"`
}

// ScheduleRunReport é o resultado auditável de uma execução de fluxo. Ele é
// persistido junto do schedule (Schedule.LastRun) tanto em sucesso quanto em
// falha, para que uma execução parcial seja visível.
type ScheduleRunReport struct {
	FinishedAt time.Time             `json:"finished_at"`
	Missions   int                   `json:"missions"`
	Steps      []ScheduleStepOutcome `json:"steps,omitempty"`
	// RunKey identifica a execução que produziu este relatório. É o que permite
	// reprocessar depois SEM adivinhar: o reprocessamento reusa a mesma chave, e
	// assim os passos que já criaram missão continuam apontando para ela.
	RunKey string `json:"run_key,omitempty"`
}

// SkippedSteps devolve os identificadores ignorados na execução. Vazio significa
// que todo passo do grafo executou.
func (r ScheduleRunReport) SkippedSteps() []string {
	skipped := make([]string, 0)
	for _, step := range r.Steps {
		if step.Status == ScheduleStepSkipped {
			skipped = append(skipped, step.ID)
		}
	}
	sort.Strings(skipped)
	return skipped
}

// FailedStep devolve o identificador do passo que falhou, se houve.
func (r ScheduleRunReport) FailedStep() string {
	for _, step := range r.Steps {
		if step.Status == ScheduleStepFailed {
			return step.ID
		}
	}
	return ""
}

// ExecuteScheduleFlow runs a persisted schedule flow and returns how many
// missions it created. An empty graph keeps the legacy behaviour of a single
// mission carrying the schedule objective.
//
// Every mission created here runs with the baseline read-only capabilities that
// CreateMission applies by default: a stored flow can never grant an elevated
// scope, matching the autorun restriction enforced by the HTTP layer. The flow
// stops at the first action that cannot create its mission and reports that
// error so the caller can record the schedule failure.
func (r *Runtime) ExecuteScheduleFlow(ctx context.Context, schedule Schedule) (int, error) {
	report, err := r.ExecuteScheduleFlowReport(ctx, schedule)
	return report.Missions, err
}

// ExecuteScheduleFlowReport executa o grafo e devolve o resultado POR NÓ, mesmo
// quando falha — o relatório parcial é o que permite registrar quais passos
// rodaram e quais foram ignorados. Erro continua sendo erro: o relatório não
// transforma falha em sucesso.
func (r *Runtime) ExecuteScheduleFlowReport(ctx context.Context, schedule Schedule) (ScheduleRunReport, error) {
	report := ScheduleRunReport{Steps: make([]ScheduleStepOutcome, 0, len(schedule.Steps)), RunKey: ScheduleFlowRunKey(schedule)}
	if len(schedule.Steps) == 0 {
		if _, err := r.createScheduledMission(ctx, schedule, schedule.Objective); err != nil {
			report.FinishedAt = time.Now().UTC()
			report.Steps = append(report.Steps, ScheduleStepOutcome{
				ID:     "legacy-objective",
				Kind:   string(ScheduleStepActionMission),
				Status: ScheduleStepFailed,
				Reason: err.Error(),
			})
			return report, err
		}
		report.Missions = 1
		report.FinishedAt = time.Now().UTC()
		report.Steps = append(report.Steps, ScheduleStepOutcome{
			ID:     "legacy-objective",
			Kind:   string(ScheduleStepActionMission),
			Status: ScheduleStepSucceeded,
		})
		return report, nil
	}
	if err := ValidateScheduleSteps(schedule.Steps); err != nil {
		return report, err
	}
	order, err := orderScheduleSteps(schedule.Steps)
	if err != nil {
		return report, err
	}
	outcomes := make(map[string]bool, len(order))
	runKey := report.RunKey
	for _, step := range order {
		id := strings.TrimSpace(step.ID)
		if !scheduleDependenciesSatisfied(step, outcomes) {
			outcomes[id] = false
			report.Steps = append(report.Steps, ScheduleStepOutcome{
				ID:     id,
				Kind:   string(step.Kind),
				Status: ScheduleStepSkipped,
				Reason: "dependência não teve sucesso",
			})
			continue
		}
		switch step.Kind {
		case ScheduleStepTriggerInterval, ScheduleStepTriggerWebhook:
			outcomes[id] = true
			report.Steps = append(report.Steps, ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded})
		case ScheduleStepActionMission:
			// Retry por nó: a chave de idempotência é a MESMA em todas as
			// tentativas, então uma repetição que encontra a missão já criada não
			// duplica nem conta como missão nova.
			created := false
			attempts, err := runWithRetry(scheduleStepAttempts(step), func() error {
				_, wasCreated, createErr := r.createScheduledMissionIdempotent(ctx, schedule, strings.TrimSpace(step.Objective), scheduleStepIdempotencyKey(runKey, id))
				if createErr == nil {
					created = wasCreated
				}
				return createErr
			})
			if err != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:       id,
					Kind:     string(step.Kind),
					Status:   ScheduleStepFailed,
					Reason:   err.Error(),
					Attempts: attempts,
				})
				return report, fmt.Errorf("schedule step %q: %w", id, err)
			}
			outcome := ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded, Attempts: attempts}
			if created {
				report.Missions++
			} else {
				// Repetição idempotente: nada foi criado agora e o relatório diz
				// isso, em vez de contar uma missão que já existia.
				outcome.Reason = "missão já existia para esta execução (idempotente)"
			}
			outcomes[id] = true
			report.Steps = append(report.Steps, outcome)
		case ScheduleStepConditionIf:
			matched := scheduleConditionMatches(step, outcomes)
			outcomes[id] = matched
			outcome := ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded}
			if !matched {
				outcome.Status = ScheduleStepSkipped
				outcome.Reason = "condição não satisfeita pelos dependentes"
			}
			report.Steps = append(report.Steps, outcome)
		default:
			return report, fmt.Errorf("%w: %s", ErrUnknownScheduleStep, step.Kind)
		}
	}
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

// scheduleDependenciesSatisfied reports whether every dependency already ran and
// succeeded. A skipped dependency therefore blocks its dependents instead of
// letting them run on a partial flow.
func scheduleDependenciesSatisfied(step ScheduleStep, outcomes map[string]bool) bool {
	for _, dependency := range step.DependsOn {
		outcome, ran := outcomes[strings.TrimSpace(dependency)]
		if !ran || !outcome {
			return false
		}
	}
	return true
}

// scheduleConditionMatches evaluates a condition.if step against the recorded
// outcome of its single dependency.
func scheduleConditionMatches(step ScheduleStep, outcomes map[string]bool) bool {
	if len(step.DependsOn) != 1 {
		return false
	}
	outcome, ran := outcomes[strings.TrimSpace(step.DependsOn[0])]
	if !ran {
		return false
	}
	if strings.TrimSpace(step.Expect) == ScheduleExpectFailed {
		return !outcome
	}
	return outcome
}

func (r *Runtime) createScheduledMission(ctx context.Context, schedule Schedule, objective string) (Mission, error) {
	return r.CreateMission(ctx, CreateMissionRequest{
		Objective:      objective,
		Model:          schedule.Model,
		Workspace:      schedule.Workspace,
		ProjectID:      schedule.ProjectID,
		OrganizationID: schedule.OrganizationID,
		AutoRun:        true,
	})
}

// ScheduleFlowRunKey identifica UMA execução do fluxo. Ele é estável durante a
// execução (o NextRunAt só é reescrito depois, ao registrar o resultado) e muda
// entre execuções — é o que permite repetir um passo com segurança: a chave de
// idempotência do passo é derivada daqui, então uma repetição DENTRO da mesma
// execução aponta para a mesma missão em vez de criar outra.
func ScheduleFlowRunKey(schedule Schedule) string {
	return strings.TrimSpace(schedule.ID) + "@" + schedule.NextRunAt.UTC().Format(time.RFC3339Nano)
}

// scheduleStepIdempotencyKey isola cada passo dentro da execução.
func scheduleStepIdempotencyKey(runKey, stepID string) string {
	return runKey + "/" + strings.TrimSpace(stepID)
}

// createScheduledMissionIdempotent cria a missão do passo sem duplicar quando a
// mesma execução rodar de novo. Devolve created=false quando a missão já existia,
// para o relatório não afirmar que criou o que não criou.
func (r *Runtime) createScheduledMissionIdempotent(ctx context.Context, schedule Schedule, objective, key string) (Mission, bool, error) {
	if existing, err := r.getMission(idempotentMissionID(schedule.OrganizationID, key)); err == nil {
		if missionScopeMatches(existing, schedule.OrganizationID, r.organizationScope) {
			return existing, false, nil
		}
	}
	mission, err := r.CreateMission(ctx, CreateMissionRequest{
		Objective:      objective,
		Model:          schedule.Model,
		Workspace:      schedule.Workspace,
		ProjectID:      schedule.ProjectID,
		OrganizationID: schedule.OrganizationID,
		AutoRun:        true,
		IdempotencyKey: key,
	})
	if err != nil {
		return Mission{}, false, err
	}
	return mission, true, nil
}
