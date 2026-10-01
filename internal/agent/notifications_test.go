package agent

import (
	"testing"
	"time"
)

func TestRuntimeListNotificationsUsesPersistedMissionEvents(t *testing.T) {
	store := NewMemoryStore()
	mission := Mission{ID: "mis_notifications", OrganizationID: LocalOrganizationID, Objective: "validar notificações", State: MissionReady, Version: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.CreateMission(mission); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC()
	if err := store.AppendEvent(Event{ID: "evt_notifications", MissionID: mission.ID, OrganizationID: LocalOrganizationID, Type: "mission.completed", CreatedAt: created}); err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := runtime.ListNotifications(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "evt_notifications" || got[0].Title != "Missão concluída" {
		t.Fatalf("notifications = %+v, want one completed mission event", got)
	}
	if got[0].Body != mission.Objective || !got[0].CreatedAt.Equal(created) {
		t.Fatalf("notification projection = %+v, want mission objective and event timestamp", got[0])
	}
}
