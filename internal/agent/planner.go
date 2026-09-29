package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
)

type Planner interface {
	Plan(ctx context.Context, mission Mission) ([]Step, error)
}

type PlannerResolver interface {
	// ResolvePlanner must return an executable planner for the requested provider.
	// It must not silently substitute a different provider or a rule-only plan.
	ResolvePlanner(provider, model string) (Planner, error)
}

type plannerChatClient interface {
	Chat(ctx context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error
}

type RulePlanner struct{}

func (RulePlanner) Plan(_ context.Context, mission Mission) ([]Step, error) {
	objective := strings.ToLower(mission.Objective)
	if strings.Contains(objective, "escrever") || strings.Contains(objective, "criar arquivo") || strings.Contains(objective, "editar") {
		return []Step{{
			ID:               "step_1",
			Kind:             "workspace.write",
			Title:            "Escrever o resultado solicitado no workspace",
			Risk:             RiskWrite,
			RequiresApproval: true,
			State:            StepPending,
			Input:            map[string]any{"path": "agent-output.txt", "content": mission.Objective + "\n"},
		}}, nil
	}
	return []Step{{
		ID:    "step_1",
		Kind:  "workspace.list",
		Title: "Inspecionar o workspace autorizado",
		Risk:  RiskRead,
		State: StepPending,
		Input: map[string]any{"path": ".", "max_entries": 100},
	}}, nil
}

type UnconfiguredPlanner struct{}

func (UnconfiguredPlanner) Plan(_ context.Context, _ Mission) ([]Step, error) {
	return nil, errors.New("planner model is not configured")
}

type OllamaPlanner struct {
	Client plannerChatClient
	Model  string
}

func (p OllamaPlanner) Plan(ctx context.Context, mission Mission) ([]Step, error) {
	model := strings.TrimSpace(mission.Model)
	if model == "" {
		model = strings.TrimSpace(p.Model)
	}
	if p.Client == nil {
		return nil, errors.New("planner provider client is unavailable")
	}
	if model == "" {
		return nil, errors.New("planner provider model is required")
	}
	stream := false
	format := json.RawMessage(`"json"`)
	request := &api.ChatRequest{
		Model:  model,
		Stream: &stream,
		Format: format,
		Messages: []api.Message{
			{Role: "system", Content: "You are a mission planner. Return only JSON with a top-level steps array. Each step must have kind, title, risk, requires_approval, and input. Allowed kinds are workspace.list, workspace.read, git.repo.inspect, workspace.write, terminal.exec, sandbox.exec, browser.operator, desktop.companion, mcp.call, mcp.remote.call, connector.http, and media.process only when a media provider is configured. Never invent completed results. Use read risk for inspection, write risk for filesystem changes, external_side_effect for browser, desktop, MCP, connectors, and media provider actions, and require approval for write, terminal, sandbox, browser, desktop, MCP, connector, and media steps."},
			{Role: "user", Content: fmt.Sprintf("Objective: %s", mission.Objective)},
		},
	}
	var response string
	err := p.Client.Chat(ctx, request, func(chatResponse api.ChatResponse) error {
		response = chatResponse.Message.Content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("planner provider request failed: %w", err)
	}
	steps, err := parsePlan(response)
	if err != nil {
		return nil, fmt.Errorf("planner returned invalid plan: %w", err)
	}
	return normalizeSteps(steps)
}

func parsePlan(content string) ([]Step, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var payload struct {
		Steps []map[string]any `json:"steps"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, err
	}
	if len(payload.Steps) == 0 {
		return nil, errors.New("planner returned no steps")
	}
	steps := make([]Step, 0, len(payload.Steps))
	for i, raw := range payload.Steps {
		// Smaller local models often return "input" as a plain string; keep
		// it as text instead of rejecting the whole plan. Validation of kinds
		// and risks still happens in normalizeSteps.
		switch input := raw["input"].(type) {
		case string:
			raw["input"] = map[string]any{"text": input}
		case nil, map[string]any:
		default:
			raw["input"] = map[string]any{"value": input}
		}
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var step Step
		if err := json.Unmarshal(data, &step); err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func normalizeSteps(steps []Step) ([]Step, error) {
	if len(steps) == 0 || len(steps) > 32 {
		return nil, errors.New("planner step count is outside the allowed range")
	}
	allowed := map[string]RiskClass{
		"workspace.list":    RiskRead,
		"workspace.read":    RiskRead,
		"git.repo.inspect":  RiskRead,
		"workspace.write":   RiskWrite,
		"terminal.exec":     RiskWrite,
		"sandbox.exec":      RiskWrite,
		"browser.operator":  RiskExternalSideEffect,
		"desktop.companion": RiskExternalSideEffect,
		"mcp.call":          RiskExternalSideEffect,
		"mcp.remote.call":   RiskExternalSideEffect,
		"connector.http":    RiskExternalSideEffect,
		"media.process":     RiskExternalSideEffect,
	}
	for i := range steps {
		if _, ok := allowed[steps[i].Kind]; !ok {
			return nil, fmt.Errorf("planner returned unsupported tool %q", steps[i].Kind)
		}
		steps[i].ID = fmt.Sprintf("step_%d", i+1)
		if steps[i].Title == "" {
			steps[i].Title = steps[i].Kind
		}
		if steps[i].Risk == "" {
			steps[i].Risk = allowed[steps[i].Kind]
		}
		if steps[i].Risk != RiskRead {
			steps[i].RequiresApproval = true
		}
		steps[i].State = StepPending
	}
	return steps, nil
}
