package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	if len(schedule.Steps) == 0 {
		if _, err := r.createScheduledMission(ctx, schedule, schedule.Objective); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err := ValidateScheduleSteps(schedule.Steps); err != nil {
		return 0, err
	}
	order, err := orderScheduleSteps(schedule.Steps)
	if err != nil {
		return 0, err
	}
	outcomes := make(map[string]bool, len(order))
	missions := 0
	for _, step := range order {
		id := strings.TrimSpace(step.ID)
		if !scheduleDependenciesSatisfied(step, outcomes) {
			outcomes[id] = false
			continue
		}
		switch step.Kind {
		case ScheduleStepTriggerInterval, ScheduleStepTriggerWebhook:
			outcomes[id] = true
		case ScheduleStepActionMission:
			if _, err := r.createScheduledMission(ctx, schedule, strings.TrimSpace(step.Objective)); err != nil {
				outcomes[id] = false
				return missions, fmt.Errorf("schedule step %q: %w", id, err)
			}
			missions++
			outcomes[id] = true
		case ScheduleStepConditionIf:
			outcomes[id] = scheduleConditionMatches(step, outcomes)
		default:
			return missions, fmt.Errorf("%w: %s", ErrUnknownScheduleStep, step.Kind)
		}
	}
	return missions, nil
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
