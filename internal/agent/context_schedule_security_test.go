package agent

import (
	"os"
	"testing"
)

func TestScheduleRejectsPathTraversalIdentifiers(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../escape", `..\\escape`, "/tmp/escape", "sch/escape"} {
		if _, err := store.CreateSchedule(Schedule{ID: id, Objective: "must reject", IntervalSeconds: 60}); err == nil {
			t.Fatalf("CreateSchedule accepted unsafe id %q", id)
		}
		if _, err := store.GetSchedule(id); !os.IsNotExist(err) {
			t.Fatalf("GetSchedule(%q) error=%v, want not-exist", id, err)
		}
		if err := store.DeleteSchedule(id); err == nil {
			t.Fatalf("DeleteSchedule accepted unsafe id %q", id)
		}
	}
}
