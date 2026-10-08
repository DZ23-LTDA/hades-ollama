package agent

import (
	"context"
	"testing"
	"time"
)

// TestWatchDurableCancellationPropagatesAcrossInstances proves that a
// cancellation persisted by one runtime instance aborts the in-flight context
// of a DIFFERENT instance running the mission. The watcher polls the durable
// store, so the running instance reacts even though it never received the
// Cancel() call itself and holds no in-memory cancel func for the mission.
func TestWatchDurableCancellationPropagatesAcrossInstances(t *testing.T) {
	store := NewMemoryStore()
	workspace := t.TempDir()
	running, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	running.cancelPollInterval = 20 * time.Millisecond
	other, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	mission, err := running.CreateMission(context.Background(), CreateMissionRequest{Objective: "tarefa longa", OrganizationID: "org-a", Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the running instance holding an in-flight step context.
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go running.watchDurableCancellation(runCtx, mission.ID, cancel, done)

	// A different instance cancels the mission; it only persists the durable
	// marker (it holds no in-memory cancel func for this mission).
	if _, err := other.Cancel(mission.ID); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runCtx.Done():
		// Success: the durable cancellation propagated to the local context.
	case <-time.After(3 * time.Second):
		t.Fatal("durable cancellation from another instance did not abort the running context")
	}
}
