package agent

import (
	"context"
	"testing"
)

func TestSlashGoalCreatesMissionAndDelegates(t *testing.T) {
	parsed, err := ParseSlashCommand("/goal construir um relatório de qualidade")
	if err != nil {
		t.Fatalf("parse /goal: %v", err)
	}
	if parsed.Name != "goal" || parsed.Objective == "" {
		t.Fatalf("parsed=%+v", parsed)
	}

	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: parsed.Objective, Provider: "ollama-local", AutoRun: false})
	if err != nil {
		t.Fatalf("create mission: %v", err)
	}
	if mission.ID == "" || mission.Objective != parsed.Objective {
		t.Fatalf("mission=%+v", mission)
	}

	orchestrator, err := NewAgentOrchestrator(t.TempDir(), func(context.Context, AgentTask) (AgentResult, error) { return AgentResult{}, nil })
	if err != nil {
		t.Fatal(err)
	}
	job, err := orchestrator.PlanForOrganization("local", parsed.Objective, "", "", SlashGoalRoles(), AgentBudget{MaxAgents: 3, MaxSeconds: 60})
	if err != nil {
		t.Fatalf("plan delegated job: %v", err)
	}
	if len(job.Tasks) != 5 {
		t.Fatalf("expected five delegated roles, got %d", len(job.Tasks))
	}
	for _, task := range job.Tasks {
		if task.State != AgentTaskPending || task.Objective == "" {
			t.Fatalf("invalid delegated task=%+v", task)
		}
	}
}

func TestSlashCommandRejectsUnknownCommand(t *testing.T) {
	if _, err := ParseSlashCommand("/does-not-exist objetivo"); err == nil {
		t.Fatal("unknown slash command was accepted")
	}
}
