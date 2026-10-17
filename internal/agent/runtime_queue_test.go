package agent

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCreateMissionAutoRunFailsClosedWhenQueuePersistenceFails(t *testing.T) {
	workspace := t.TempDir()
	queueRoot := t.TempDir()
	queue, err := NewJobQueue(queueRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(queueRoot); err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Queue: queue, Planner: RulePlanner{}, WorkspaceRoot: workspace, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "auto run queue failure", Workspace: workspace, AutoRun: true})
	if err == nil || !strings.Contains(err.Error(), "auto-run enqueue failed") {
		t.Fatalf("CreateMission err=%v, want explicit enqueue failure", err)
	}
	if mission.State != MissionFailed || mission.LastError == "" {
		t.Fatalf("returned mission did not fail closed: %+v", mission)
	}
	persisted, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.State != MissionFailed || !strings.Contains(persisted.LastError, "auto-run enqueue failed") {
		t.Fatalf("persisted mission state=%+v", persisted)
	}
	if jobs := runtime.QueueJobs(""); len(jobs) != 0 {
		t.Fatalf("failed enqueue left jobs: %+v", jobs)
	}
	events, err := runtime.store.ListEvents(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.Type == "mission.queue_failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("queue failure event was not persisted")
	}
}
