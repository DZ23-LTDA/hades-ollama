package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContextStoreRollsBackMutationsWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetWorkspaceRoot(workspace); err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(workspace, "project")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("Original", projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "Original schedule", IntervalSeconds: 60, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemory(Memory{ID: "mem_original", ProjectID: project.ID, Kind: "note", Content: "original", Confidence: 1, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	store.root = unwritableStoreRoot(t, root, "store")
	if _, err := store.UpdateProject(project.ID, "Changed", projectRoot); err == nil {
		t.Fatal("project update unexpectedly succeeded")
	}
	if _, err := store.GetProject(project.ID); err != nil {
		t.Fatal(err)
	} else if got, _ := store.GetProject(project.ID); got.Name != "Original" {
		t.Fatalf("project changed after failed update: %+v", got)
	}
	if _, err := store.UpdateSchedule(schedule.ID, Schedule{Objective: "Changed schedule", IntervalSeconds: 120}); err == nil {
		t.Fatal("schedule update unexpectedly succeeded")
	}
	if got, _ := store.GetSchedule(schedule.ID); got.Objective != "Original schedule" {
		t.Fatalf("schedule changed after failed update: %+v", got)
	}
	if _, err := store.AddMemory(Memory{ID: "mem_failed", ProjectID: project.ID, Kind: "note", Content: "must rollback", Confidence: 1, CreatedAt: time.Now().UTC()}); err == nil {
		t.Fatal("memory write unexpectedly succeeded")
	}
	if memories := store.SearchMemories(project.ID, "", 10); len(memories) != 1 || memories[0].ID != "mem_original" {
		t.Fatalf("memories changed after failed write: %+v", memories)
	}
}

func TestContextStoreRollsBackDeleteScheduleWhenRemoveFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "delete me", IntervalSeconds: 60, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	badRoot := t.TempDir()
	path := filepath.Join(badRoot, "schedules", schedule.ID+".json")
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	store.root = badRoot
	if err := store.DeleteSchedule(schedule.ID); err == nil {
		t.Fatal("schedule delete unexpectedly succeeded")
	}
	if _, err := store.GetSchedule(schedule.ID); err != nil {
		t.Fatalf("schedule disappeared after failed delete: %v", err)
	}
}
