package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResumePendingRecordsBoundedScheduleFailures(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "will fail planner", IntervalSeconds: 60, OrganizationID: "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	schedule.NextRunAt = time.Now().UTC().Add(-time.Second)
	schedule, err = store.UpdateSchedule(schedule.ID, schedule)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Context: store, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	orgRuntime := runtime.WithOrganization("org-a")
	if err := orgRuntime.resumePending(context.Background()); err != nil {
		t.Fatal(err)
	}
	failed, err := store.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.FailureCount != 1 || failed.LastFailureCode != "mission_creation_failed" || !failed.Enabled || !failed.NextRunAt.After(time.Now().UTC()) {
		t.Fatalf("first failure state = %+v", failed)
	}
	for failed.FailureCount < maxScheduleFailures {
		failed, err = orgRuntime.recordScheduleFailure(failed)
		if err != nil {
			t.Fatal(err)
		}
	}
	if failed.Enabled || failed.FailureCount != maxScheduleFailures {
		t.Fatalf("schedule was not disabled after bounded failures: %+v", failed)
	}
	failed.Enabled = true
	failed, err = orgRuntime.recordScheduleSuccess(failed)
	if err != nil {
		t.Fatal(err)
	}
	if failed.FailureCount != 0 || failed.LastFailureCode != "" || !failed.Enabled {
		t.Fatalf("success did not reset failure state: %+v", failed)
	}
}

func TestClaimDueSchedulesRollsBackMemoryOnPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "claim rollback", IntervalSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	schedule.NextRunAt = time.Now().UTC().Add(-time.Second)
	if _, err := store.UpdateSchedule(schedule.ID, schedule); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "schedules")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "schedules"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if due := store.ClaimDueSchedules(time.Now().UTC()); len(due) != 0 {
		t.Fatalf("claim unexpectedly succeeded: %+v", due)
	}
	current, err := store.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !current.NextRunAt.Before(time.Now().UTC()) || current.LastRunAt != nil {
		t.Fatalf("claim did not rollback memory: %+v", current)
	}
}
