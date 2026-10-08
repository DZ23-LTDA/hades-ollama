package agent

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAgentOrchestratorRunsSpecialistsAndSynthesizes(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	var attempts atomic.Int32
	runner := func(ctx context.Context, task AgentTask) (AgentResult, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		if task.Role == RoleSecurity && attempts.Add(1) == 1 {
			return AgentResult{}, errors.New("transient")
		}
		select {
		case <-ctx.Done():
			return AgentResult{}, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
		return AgentResult{Output: string(task.Role) + " result", Evidence: []AgentEvidence{{URL: "https://example.test/source", SHA256: string(task.Role)}}}, nil
	}
	orchestrator, err := NewAgentOrchestrator(t.TempDir(), runner)
	if err != nil {
		t.Fatal(err)
	}
	job, err := orchestrator.Plan("pesquisar e revisar arquitetura", t.TempDir(), "project-1", []AgentRole{RoleResearch, RoleSecurity, RoleReview}, AgentBudget{MaxAgents: 2, MaxRetries: 1, MaxSeconds: 10})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := orchestrator.Run(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != OrchestrationCompleted || completed.Summary == "" {
		t.Fatalf("job=%+v", completed)
	}
	if peak.Load() > 2 {
		t.Fatalf("parallelism exceeded budget: %d", peak.Load())
	}
	if !strings.Contains(completed.Summary, "research") || len(completed.Conflicts) == 0 {
		t.Fatalf("summary=%q conflicts=%v", completed.Summary, completed.Conflicts)
	}
}

func TestAgentOrchestratorPersistsAndCancelsPlannedJob(t *testing.T) {
	root := t.TempDir()
	orchestrator, err := NewAgentOrchestrator(root, func(context.Context, AgentTask) (AgentResult, error) { return AgentResult{Output: "unused"}, nil })
	if err != nil {
		t.Fatal(err)
	}
	job, err := orchestrator.Plan("test", root, "", []AgentRole{RoleTesting}, AgentBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewAgentOrchestrator(root, func(context.Context, AgentTask) (AgentResult, error) { return AgentResult{}, nil })
	if err != nil {
		t.Fatal(err)
	}
	stored, err := reloaded.Get(job.ID)
	if err != nil || stored.State != OrchestrationCancelled {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestAgentOrchestratorCancelsRunningJob(t *testing.T) {
	root := t.TempDir()
	started := make(chan struct{})
	runner := func(ctx context.Context, task AgentTask) (AgentResult, error) {
		close(started) // single task -> called once
		<-ctx.Done()   // block until the orchestration is cancelled
		return AgentResult{}, ctx.Err()
	}
	orchestrator, err := NewAgentOrchestrator(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	job, err := orchestrator.Plan("cancel running", root, "", []AgentRole{RoleTesting}, AgentBudget{MaxAgents: 1})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan OrchestrationJob, 1)
	go func() {
		result, _ := orchestrator.Run(context.Background(), job.ID)
		done <- result
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not start")
	}

	cancelled, err := orchestrator.Cancel(job.ID)
	if err != nil {
		t.Fatalf("cancel running job: %v", err)
	}
	if cancelled.State != OrchestrationCancelled {
		t.Fatalf("Cancel state = %v, want %v", cancelled.State, OrchestrationCancelled)
	}

	select {
	case result := <-done:
		if result.State != OrchestrationCancelled {
			t.Fatalf("Run final state = %v, want %v", result.State, OrchestrationCancelled)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancellation (job was not interrupted)")
	}
}

func TestAgentOrchestratorOrganizationScope(t *testing.T) {
	orchestrator, err := NewAgentOrchestrator(t.TempDir(), func(context.Context, AgentTask) (AgentResult, error) { return AgentResult{}, nil })
	if err != nil {
		t.Fatal(err)
	}
	job, err := orchestrator.PlanForOrganization("org-a", "tenant A", t.TempDir(), "", []AgentRole{RoleTesting}, AgentBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.GetForOrganization(job.ID, "org-b"); !errors.Is(err, ErrOrchestrationForbidden) {
		t.Fatalf("cross-tenant orchestration read err=%v", err)
	}
	if _, err := orchestrator.CancelForOrganization(job.ID, "org-b"); !errors.Is(err, ErrOrchestrationForbidden) {
		t.Fatalf("cross-tenant orchestration cancel err=%v", err)
	}
	stored, err := orchestrator.GetForOrganization(job.ID, "org-a")
	if err != nil || stored.State != OrchestrationPlanned {
		t.Fatalf("cross-tenant cancel mutated job=%+v err=%v", stored, err)
	}
}

func TestAgentOrchestratorRollsBackRunAndCancelOnPersistenceFailure(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		root := t.TempDir()
		orchestrator, err := NewAgentOrchestrator(root, func(context.Context, AgentTask) (AgentResult, error) {
			return AgentResult{Output: "unused"}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		job, err := orchestrator.Plan("run rollback", root, "", []AgentRole{RoleTesting}, AgentBudget{})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(root); err != nil {
			t.Fatal(err)
		}
		if _, err := orchestrator.Run(context.Background(), job.ID); err == nil {
			t.Fatal("run unexpectedly succeeded without a writable ledger")
		}
		stored, err := orchestrator.Get(job.ID)
		if err != nil || stored.State != OrchestrationPlanned {
			t.Fatalf("run mutated memory after persistence failure: stored=%+v err=%v", stored, err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		root := t.TempDir()
		orchestrator, err := NewAgentOrchestrator(root, func(context.Context, AgentTask) (AgentResult, error) {
			return AgentResult{}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		job, err := orchestrator.Plan("cancel rollback", root, "", []AgentRole{RoleTesting}, AgentBudget{})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(root); err != nil {
			t.Fatal(err)
		}
		if _, err := orchestrator.Cancel(job.ID); err == nil {
			t.Fatal("cancel unexpectedly succeeded without a writable ledger")
		}
		stored, err := orchestrator.Get(job.ID)
		if err != nil || stored.State != OrchestrationPlanned {
			t.Fatalf("cancel mutated memory after persistence failure: stored=%+v err=%v", stored, err)
		}
	})
}
