package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestJSONStoreRollsBackMissionMutationWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original := Mission{ID: "mis_atomic", Version: 1, Objective: "original", State: MissionReady, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.PutMission(original); err != nil {
		t.Fatal(err)
	}
	store.root = filepath.Join(root, "missing-parent", "store")
	updated := original
	updated.Version = 2
	updated.Objective = "must not remain in memory"
	if err := store.PutMissionIfVersion(updated, original.Version); err == nil {
		t.Fatal("write unexpectedly succeeded")
	}
	store.mu.RLock()
	got := cloneMission(store.missions[original.ID])
	store.mu.RUnlock()
	if got.Version != original.Version || got.Objective != original.Objective {
		t.Fatalf("mission memory diverged: got=%+v original=%+v", got, original)
	}
	reloaded, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reloaded.GetMission(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Version != original.Version || persisted.Objective != original.Objective {
		t.Fatalf("mission disk diverged: got=%+v original=%+v", persisted, original)
	}
	newMission := Mission{ID: "mis_new", Version: 1, Objective: "new", State: MissionReady}
	if err := store.PutMission(newMission); err == nil {
		t.Fatal("new mission write unexpectedly succeeded")
	}
	if _, err := store.GetMission(newMission.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed new mission remained in memory: %v", err)
	}
}

func TestJSONStoreRollsBackEventsWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutMission(Mission{ID: "mis_events", Version: 1, OrganizationID: "org_a", State: MissionReady, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	first := Event{ID: "evt_first", MissionID: "mis_events", OrganizationID: "org_a", Type: "first", CreatedAt: time.Now().UTC()}
	if err := store.AppendEvent(first); err != nil {
		t.Fatal(err)
	}
	store.root = filepath.Join(root, "missing-parent", "store")
	second := Event{ID: "evt_second", MissionID: first.MissionID, OrganizationID: first.OrganizationID, Type: "second", CreatedAt: time.Now().UTC()}
	if err := store.AppendEvent(second); err == nil {
		t.Fatal("event write unexpectedly succeeded")
	}
	store.mu.RLock()
	inMemory := append([]Event(nil), store.events[first.MissionID]...)
	store.mu.RUnlock()
	if len(inMemory) != 1 || inMemory[0].ID != first.ID {
		t.Fatalf("events memory diverged: %+v", inMemory)
	}
	store.root = root
	events, err := store.ListEvents(first.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != first.ID {
		t.Fatalf("events disk diverged: %+v", events)
	}
	reloaded, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	events, err = reloaded.ListEvents(first.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != first.ID {
		t.Fatalf("events disk diverged: %+v", events)
	}
}

func TestJSONStoreCreateMissionCannotOverwriteAcrossInstances(t *testing.T) {
	root := t.TempDir()
	first, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original := Mission{ID: "mis_webhook_stable", Version: 1, Objective: "original", OrganizationID: "org_a", State: MissionPlanning}
	if err := first.CreateMission(original); err != nil {
		t.Fatal(err)
	}
	foreign := original
	foreign.Objective = "must not overwrite"
	foreign.OrganizationID = "org_b"
	if err := second.CreateMission(foreign); !errors.Is(err, ErrMissionAlreadyExists) {
		t.Fatalf("duplicate create error=%v, want ErrMissionAlreadyExists", err)
	}
	reloaded, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.GetMission(original.ID)
	if err != nil || got.Objective != original.Objective || got.OrganizationID != original.OrganizationID {
		t.Fatalf("existing mission was overwritten: got=%+v err=%v", got, err)
	}
}

func TestJSONStoreAppendEventRequiresSameTenantMissionAndJSONPayload(t *testing.T) {
	store := NewMemoryStore()
	mission := Mission{ID: "mis_event_owner", Version: 1, OrganizationID: "org_a", State: MissionReady}
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendEvent(Event{ID: "evt_wrong_tenant", MissionID: mission.ID, OrganizationID: "org_b", Type: "bad"}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("wrong-tenant event error=%v, want permission denied", err)
	}
	if err := store.AppendEvent(Event{ID: "evt_invalid_payload", MissionID: mission.ID, OrganizationID: "org_a", Type: "bad", Payload: map[string]any{"unsupported": func() {}}}); err == nil {
		t.Fatal("non-JSON event payload unexpectedly persisted")
	}
	if events, err := store.ListEvents(mission.ID); err != nil || len(events) != 0 {
		t.Fatalf("rejected events were persisted: events=%+v err=%v", events, err)
	}
}

func TestJSONStoreAggregateEventQuotas(t *testing.T) {
	if err := checkJSONStoreEventQuotaTotals(maxJSONStoreEventFiles+1, 0, 0); err == nil {
		t.Fatal("aggregate event-file count above the limit was accepted")
	}
	if err := checkJSONStoreEventQuotaTotals(1, maxJSONStoreTotalEvents+1, 0); err == nil {
		t.Fatal("aggregate event count above the limit was accepted")
	}
	if err := checkJSONStoreEventQuotaTotals(1, 1, maxJSONStoreTotalEventBytes+1); err == nil {
		t.Fatal("aggregate event bytes above the limit were accepted")
	}
	if err := checkJSONStoreEventQuotaTotals(maxJSONStoreEventFiles, maxJSONStoreTotalEvents, maxJSONStoreTotalEventBytes); err != nil {
		t.Fatalf("exactly-at-limit aggregate was rejected: %v", err)
	}
}

func TestJSONStoreRedactsMissionValuesInMemoryAndDisk(t *testing.T) {
	const secret = "example-secret-value"
	mission := Mission{ID: "mis_redact_store", Version: 1, Objective: `{"api_key":"` + secret + `"}`, Plan: []Step{{Title: "token=ghp_abcdefghijklmnopqrstuvwxyz123456", Input: map[string]any{"credential": secret}}}}
	for _, test := range []struct {
		name string
		root string
	}{
		{name: "memory"},
		{name: "persistent", root: t.TempDir()},
	} {
		t.Run(test.name, func(t *testing.T) {
			var store *JSONStore
			var err error
			if test.root == "" {
				store = NewMemoryStore()
			} else {
				store, err = NewJSONStore(test.root)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := store.CreateMission(mission); err != nil {
				t.Fatal(err)
			}
			got, err := store.GetMission(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(got)
			if err != nil || strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "ghp_abcdefghijklmnopqrstuvwxyz123456") {
				t.Fatalf("mission cache exposed sensitive data: err=%v", err)
			}
			if test.root != "" {
				disk, err := os.ReadFile(filepath.Join(test.root, "missions", mission.ID+".json"))
				if err != nil || strings.Contains(string(disk), secret) {
					t.Fatalf("mission file exposed a credential: err=%v", err)
				}
			}
		})
	}
}

func TestJSONStoreArtifactMetadataAndPutMissionVersions(t *testing.T) {
	const (
		nameSecret      = "artifact-name-secret"
		pathSecret      = "artifact-path-secret"
		mediaTypeSecret = "artifact-media-secret"
	)
	mission := Mission{
		ID:      "mis_artifact_metadata_redact",
		Version: 1,
		Artifacts: []ArtifactManifest{{
			ID:        "artifact_identity",
			MissionID: "mis_artifact_metadata_redact",
			StepID:    "step_identity",
			Name:      "api_key=" + nameSecret,
			Path:      "reports/token=" + pathSecret,
			MediaType: "application/x-secret=" + mediaTypeSecret,
			SHA256:    "integrity-identity",
		}},
	}
	for _, test := range []struct {
		name string
		root string
	}{
		{name: "memory"},
		{name: "persistent", root: t.TempDir()},
	} {
		t.Run("artifact metadata "+test.name, func(t *testing.T) {
			var store *JSONStore
			var err error
			if test.root == "" {
				store = NewMemoryStore()
			} else {
				store, err = NewJSONStore(test.root)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := store.PutMission(mission); err != nil {
				t.Fatal(err)
			}
			got, err := store.GetMission(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{nameSecret, pathSecret, mediaTypeSecret} {
				if strings.Contains(string(encoded), secret) {
					t.Fatalf("returned mission exposed artifact credential %q: %s", secret, encoded)
				}
			}
			if got.Artifacts[0].ID != "artifact_identity" || got.Artifacts[0].MissionID != mission.ID || got.Artifacts[0].StepID != "step_identity" || got.Artifacts[0].SHA256 != "integrity-identity" {
				t.Fatalf("artifact identity/integrity fields changed: %+v", got.Artifacts[0])
			}
			if test.root != "" {
				disk, err := os.ReadFile(filepath.Join(test.root, "missions", mission.ID+".json"))
				if err != nil {
					t.Fatal(err)
				}
				for _, secret := range []string{nameSecret, pathSecret, mediaTypeSecret} {
					if strings.Contains(string(disk), secret) {
						t.Fatalf("mission file exposed artifact credential %q: %s", secret, disk)
					}
				}
			}
		})
	}

	legacyRoot := t.TempDir()
	legacyMission := Mission{
		ID:      "mis_legacy_artifact_metadata",
		Version: 1,
		Artifacts: []ArtifactManifest{{
			ID:        "artifact_identity",
			MissionID: "mis_legacy_artifact_metadata",
			Name:      "credential=legacy-name-secret",
			Path:      "legacy/password=legacy-path-secret",
			MediaType: "application/x-token=legacy-media-secret",
			SHA256:    "integrity-identity",
		}},
	}
	legacyPath := filepath.Join(legacyRoot, "missions", legacyMission.ID+".json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyData, err := json.Marshal(legacyMission)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, legacyData, 0o600); err != nil {
		t.Fatal(err)
	}
	legacyStore, err := NewJSONStore(legacyRoot)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := legacyStore.GetMission(legacyMission.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{loaded.Artifacts[0].Name, loaded.Artifacts[0].Path, loaded.Artifacts[0].MediaType} {
		if strings.Contains(value, "legacy-") {
			t.Fatalf("legacy artifact metadata remained in returned mission: %+v", loaded.Artifacts[0])
		}
	}
	legacyDisk, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"legacy-name-secret", "legacy-path-secret", "legacy-media-secret"} {
		if strings.Contains(string(legacyDisk), secret) {
			t.Fatalf("legacy artifact credential remained in mission file %q: %s", secret, legacyDisk)
		}
	}

	for _, test := range []struct {
		name string
		root string
	}{
		{name: "memory"},
		{name: "persistent", root: t.TempDir()},
	} {
		t.Run("PutMission versions "+test.name, func(t *testing.T) {
			var store *JSONStore
			var err error
			if test.root == "" {
				store = NewMemoryStore()
			} else {
				store, err = NewJSONStore(test.root)
				if err != nil {
					t.Fatal(err)
				}
			}
			base := Mission{ID: "mis_put_version_transition", Version: 1, Objective: "v1"}
			if err := store.PutMission(base); err != nil {
				t.Fatal(err)
			}
			if err := store.PutMission(base); err != nil {
				t.Fatalf("equal idempotent PutMission failed: %v", err)
			}
			jump := base
			jump.Version = 3
			if err := store.PutMission(jump); !errors.Is(err, ErrMissionVersionConflict) {
				t.Fatalf("version jump error=%v, want conflict", err)
			}
			next := base
			next.Version = 2
			next.Objective = "v2"
			if err := store.PutMission(next); err != nil {
				t.Fatalf("exact +1 PutMission failed: %v", err)
			}
			if err := store.PutMission(next); err != nil {
				t.Fatalf("equal idempotent v2 PutMission failed: %v", err)
			}
			jump = next
			jump.Version = 4
			if err := store.PutMission(jump); !errors.Is(err, ErrMissionVersionConflict) {
				t.Fatalf("second version jump error=%v, want conflict", err)
			}
			got, err := store.GetMission(base.ID)
			if err != nil || got.Version != 2 || got.Objective != "v2" {
				t.Fatalf("rejected version jump changed mission: got=%+v err=%v", got, err)
			}
		})
	}

}

func TestOrganizationScopedStoreEnforcesMemoryStoreOwnership(t *testing.T) {
	base := NewMemoryStore()
	for _, mission := range []Mission{{ID: "mis_org_a", Version: 1, OrganizationID: "org_a"}, {ID: "mis_org_b", Version: 1, OrganizationID: "org_b"}} {
		if err := base.CreateMission(mission); err != nil {
			t.Fatal(err)
		}
	}
	scoped := organizationScopedStore{store: base, organizationID: "org_a"}
	if got, err := scoped.ListMissions(); err != nil || len(got) != 1 || got[0].ID != "mis_org_a" {
		t.Fatalf("tenant mission list=%+v err=%v", got, err)
	}
	if _, err := scoped.GetMission("mis_org_b"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign mission lookup error=%v", err)
	}
	if err := scoped.PutMission(Mission{ID: "mis_org_b", Version: 2, OrganizationID: "org_b"}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign mission update error=%v", err)
	}
	if err := scoped.CreateMission(Mission{ID: "mis_wrong_create", Version: 1, OrganizationID: "org_b"}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("foreign mission create error=%v", err)
	}
	if _, err := scoped.ListEvents("mis_org_b"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign event list error=%v", err)
	}
}

func TestJSONStoreScrubsLegacyMissionAndEventFilesOnLoad(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"missions", "events"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	const secret = "legacy-credential-value"
	mission := Mission{ID: "mis_legacy_redact", Version: 1, Objective: `{"api_key":"` + secret + `"}`}
	missionData, err := json.Marshal(mission)
	if err != nil {
		t.Fatal(err)
	}
	missionPath := filepath.Join(root, "missions", mission.ID+".json")
	if err := os.WriteFile(missionPath, missionData, 0o600); err != nil {
		t.Fatal(err)
	}
	events := []Event{{ID: "evt_legacy_redact", MissionID: mission.ID, Type: "legacy", Payload: map[string]any{"api_key": secret}}}
	eventData, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(root, "events", mission.ID+".json")
	if err := os.WriteFile(eventPath, eventData, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	loadedJSON, err := json.Marshal(loaded)
	if err != nil || strings.Contains(string(loadedJSON), secret) {
		t.Fatalf("legacy mission cache retains secret: err=%v", err)
	}
	loadedEvents, err := store.ListEvents(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventJSON, err := json.Marshal(loadedEvents)
	if err != nil || strings.Contains(string(eventJSON), secret) {
		t.Fatalf("legacy event cache retains secret: err=%v", err)
	}
	for _, path := range []string{missionPath, eventPath} {
		disk, err := os.ReadFile(path)
		if err != nil || strings.Contains(string(disk), secret) {
			t.Fatalf("legacy file %s retains secret: err=%v", filepath.Base(path), err)
		}
	}
}

func TestJSONStoreDeepCopiesMissionIngressAndEgress(t *testing.T) {
	const token = "ghp_abcdefghijklmnopqrstuvwxyz123456"
	expiresAt := time.Unix(123, 0).UTC()
	completedAt := time.Unix(456, 0).UTC()
	mission := Mission{
		ID:             "mis_deep_copy",
		Version:        1,
		OrganizationID: "org_a",
		Capabilities:   []string{"workspace:read", "workspace:write"},
		Objective:      "stable objective",
		Plan: []Step{{
			Title: "stable step",
			Input: map[string]any{
				"nested": map[string]any{
					"token": token,
					"items": []any{"keep", map[string]any{"value": "nested"}},
				},
			},
			Result: map[string]any{
				"nested": []any{map[string]any{"api_key": "hidden"}, "result"},
			},
		}},
		Approvals:   []Approval{{ID: "approval", ExpiresAt: &expiresAt}},
		Artifacts:   []ArtifactManifest{{ID: "artifact", Name: "name"}},
		CompletedAt: &completedAt,
	}
	wantInputToken := "[REDACTED]"
	for _, test := range []struct {
		name string
		root string
	}{
		{name: "memory"},
		{name: "persistent", root: t.TempDir()},
	} {
		t.Run(test.name, func(t *testing.T) {
			var store *JSONStore
			var err error
			if test.root == "" {
				store = NewMemoryStore()
			} else {
				store, err = NewJSONStore(test.root)
				if err != nil {
					t.Fatal(err)
				}
			}
			submitted := cloneMission(mission)
			if err := store.PutMission(submitted); err != nil {
				t.Fatal(err)
			}

			// Mutating the caller's graph after ingress must not alter the retained
			// redacted snapshot.
			submitted.Capabilities[0] = "mutated"
			submitted.Plan[0].Input["nested"].(map[string]any)["token"] = "caller mutation"
			submitted.Plan[0].Result.(map[string]any)["nested"].([]any)[1] = "caller mutation"
			*submitted.Approvals[0].ExpiresAt = time.Unix(999, 0).UTC()
			*submitted.CompletedAt = time.Unix(999, 0).UTC()

			got, err := store.GetMission(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Capabilities[0] != "workspace:read" || got.Plan[0].Input["nested"].(map[string]any)["token"] != wantInputToken {
				t.Fatalf("mission ingress mutation leaked into store: %+v", got)
			}
			if got.Plan[0].Result.(map[string]any)["nested"].([]any)[1] != "result" || got.Approvals[0].ExpiresAt.Equal(expiresAt) == false || got.CompletedAt.Equal(completedAt) == false {
				t.Fatalf("mission ingress nested mutation leaked into store: %+v", got)
			}

			// Mutating a returned graph must not alter the cache or the persistent
			// snapshot used by subsequent reads.
			got.Capabilities[1] = "returned mutation"
			got.Plan[0].Input["nested"].(map[string]any)["items"].([]any)[1] = "returned mutation"
			got.Plan[0].Result.(map[string]any)["nested"].([]any)[0].(map[string]any)["api_key"] = "returned mutation"
			got.Approvals[0].ExpiresAt = nil
			got.CompletedAt = nil

			again, err := store.GetMission(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			if again.Capabilities[1] != "workspace:write" || !reflect.DeepEqual(again.Plan[0].Input["nested"].(map[string]any)["items"].([]any)[1], map[string]any{"value": "nested"}) {
				t.Fatalf("mission egress mutation leaked into store: %+v", again)
			}
			if again.Plan[0].Result.(map[string]any)["nested"].([]any)[0].(map[string]any)["api_key"] != "[REDACTED]" || again.Approvals[0].ExpiresAt == nil || again.CompletedAt == nil {
				t.Fatalf("mission egress redaction or pointer isolation failed: %+v", again)
			}
		})
	}
}

func TestJSONStoreDeepCopiesEventPayloadIngressAndEgress(t *testing.T) {
	const token = "ghp_abcdefghijklmnopqrstuvwxyz123456"
	for _, test := range []struct {
		name string
		root string
	}{
		{name: "memory"},
		{name: "persistent", root: t.TempDir()},
	} {
		t.Run(test.name, func(t *testing.T) {
			var store *JSONStore
			var err error
			if test.root == "" {
				store = NewMemoryStore()
			} else {
				store, err = NewJSONStore(test.root)
				if err != nil {
					t.Fatal(err)
				}
			}
			mission := Mission{ID: "mis_event_deep_copy", Version: 1, OrganizationID: "org_a"}
			if err := store.PutMission(mission); err != nil {
				t.Fatal(err)
			}
			payload := map[string]any{
				"nested": map[string]any{"token": token, "items": []any{"keep", map[string]any{"value": "nested"}}},
			}
			event := Event{ID: "evt_deep_copy", MissionID: mission.ID, OrganizationID: mission.OrganizationID, Payload: payload}
			if err := store.AppendEvent(event); err != nil {
				t.Fatal(err)
			}
			payload["nested"].(map[string]any)["items"].([]any)[1] = "caller mutation"

			events, err := store.ListEvents(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			items := events[0].Payload.(map[string]any)["nested"].(map[string]any)["items"].([]any)
			if events[0].Payload.(map[string]any)["nested"].(map[string]any)["token"] != "[REDACTED]" || !reflect.DeepEqual(items[1], map[string]any{"value": "nested"}) {
				t.Fatalf("event ingress mutation or redaction failed: %+v", events)
			}
			events[0].Payload.(map[string]any)["nested"].(map[string]any)["items"].([]any)[1] = "returned mutation"

			again, err := store.ListEvents(mission.ID)
			if err != nil {
				t.Fatal(err)
			}
			againItems := again[0].Payload.(map[string]any)["nested"].(map[string]any)["items"].([]any)
			if !reflect.DeepEqual(againItems[1], map[string]any{"value": "nested"}) || again[0].Payload.(map[string]any)["nested"].(map[string]any)["token"] != "[REDACTED]" {
				t.Fatalf("event egress mutation leaked into store: %+v", again)
			}
		})
	}
}

func TestJSONStorePutMissionRejectsStaleAndForeignPersistentWrites(t *testing.T) {
	root := t.TempDir()
	first, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original := Mission{ID: "mis_stale_put", Version: 1, OrganizationID: "org_a", Objective: "v1"}
	if err := first.PutMission(original); err != nil {
		t.Fatal(err)
	}
	second, err = NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	stale, err := second.GetMission(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := original
	updated.Version = 2
	updated.Objective = "newer"
	if err := first.PutMission(updated); err != nil {
		t.Fatal(err)
	}
	stale.Objective = `{"api_key":"legacy-secret"}`
	if err := second.PutMission(stale); !errors.Is(err, ErrMissionVersionConflict) {
		t.Fatalf("stale persistent PutMission error=%v, want version conflict", err)
	}
	got, err := second.GetMission(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != updated.Version || got.Objective != updated.Objective {
		t.Fatalf("stale write regressed cache: got=%+v", got)
	}
	disk, err := os.ReadFile(filepath.Join(root, "missions", original.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(disk), "legacy-secret") || !strings.Contains(string(disk), "newer") {
		t.Fatalf("stale write regressed disk: %s", disk)
	}
	foreign := updated
	foreign.OrganizationID = "org_b"
	foreign.Version = 3
	foreign.Objective = "foreign overwrite"
	if err := second.PutMission(foreign); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("foreign persistent PutMission error=%v, want permission denied", err)
	}
	reloaded, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err = reloaded.GetMission(original.ID)
	if err != nil || got.Version != updated.Version || got.OrganizationID != updated.OrganizationID || got.Objective != updated.Objective {
		t.Fatalf("foreign write changed persisted mission: got=%+v err=%v", got, err)
	}
}

func TestJSONStoreRejectsPathTraversalIDs(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "escape.json")
	if err := os.WriteFile(sentinel, []byte(`{"untouched":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.PutMission(Mission{ID: "../escape", Version: 1}); err == nil {
		t.Fatal("PutMission accepted traversal ID")
	}
	if err := store.CreateMission(Mission{ID: "../escape", Version: 1}); err == nil {
		t.Fatal("CreateMission accepted traversal ID")
	}
	if err := store.PutMissionIfVersion(Mission{ID: "../escape", Version: 2}, 1); err == nil {
		t.Fatal("PutMissionIfVersion accepted traversal ID")
	}
	if err := store.AppendEvent(Event{ID: "evt_escape", MissionID: "../escape"}); err == nil {
		t.Fatal("AppendEvent accepted traversal mission ID")
	}
	if _, err := store.GetMission("../escape"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GetMission traversal error=%v, want not found", err)
	}
	if _, err := store.ListEvents("../escape"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListEvents traversal error=%v, want not found", err)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != `{"untouched":true}` {
		t.Fatalf("traversal modified sibling file: data=%q err=%v", data, err)
	}
}

func TestJSONStoreRejectsNonJSONMissionGraphsBeforeMutation(t *testing.T) {
	type unsupported struct {
		Mutable []string
		Channel chan int
	}
	store := NewMemoryStore()
	valid := Mission{ID: "mis_json_boundary", Version: 1, Objective: "original"}
	if err := store.CreateMission(valid); err != nil {
		t.Fatal(err)
	}
	bad := valid
	bad.Plan = []Step{{Result: unsupported{Mutable: []string{"before"}, Channel: make(chan int)}}}
	if err := store.PutMission(bad); err == nil {
		t.Fatal("PutMission accepted a non-JSON mission graph")
	}
	if err := store.PutMissionIfVersion(Mission{ID: valid.ID, Version: 2, Plan: bad.Plan}, 1); err == nil {
		t.Fatal("PutMissionIfVersion accepted a non-JSON mission graph")
	}
	if err := store.CreateMission(Mission{ID: "mis_invalid_json", Version: 1, Plan: bad.Plan}); err == nil {
		t.Fatal("CreateMission accepted a non-JSON mission graph")
	}
	got, err := store.GetMission(valid.ID)
	if err != nil || got.Version != valid.Version || got.Objective != valid.Objective || len(got.Plan) != 0 {
		t.Fatalf("rejected graph changed cached mission: got=%+v err=%v", got, err)
	}
}

func TestPersistentJSONStoreRefreshesReadsAcrossInstances(t *testing.T) {
	root := t.TempDir()
	first, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original := Mission{ID: "mis_cross_instance", Version: 1, OrganizationID: "org_a", Objective: "v1"}
	if err := first.CreateMission(original); err != nil {
		t.Fatal(err)
	}
	second, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	updated := original
	updated.Version = 2
	updated.Objective = "v2"
	if err := first.PutMission(updated); err != nil {
		t.Fatal(err)
	}
	got, err := second.GetMission(original.ID)
	if err != nil || got.Version != 2 || got.Objective != "v2" {
		t.Fatalf("cross-instance GetMission returned stale value: got=%+v err=%v", got, err)
	}
	newMission := Mission{ID: "mis_cross_new", Version: 1, OrganizationID: "org_a"}
	if err := first.CreateMission(newMission); err != nil {
		t.Fatal(err)
	}
	listed, err := second.ListMissions()
	if err != nil || len(listed) != 2 {
		t.Fatalf("cross-instance ListMissions=%+v err=%v", listed, err)
	}
	if err := os.Remove(filepath.Join(root, "missions", original.ID+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetMission(original.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted persistent mission remained visible from cache: %v", err)
	}
}

func TestPersistentJSONStoreCASRefreshesStaleCacheBeforeComparingVersion(t *testing.T) {
	root := t.TempDir()
	first, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	original := Mission{ID: "mis_stale_cas", Version: 1, OrganizationID: "org_a", Objective: "v1"}
	if err := first.CreateMission(original); err != nil {
		t.Fatal(err)
	}
	second, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.GetMission(original.ID); err != nil {
		t.Fatal(err)
	}
	updated := original
	updated.Version = 2
	updated.Objective = "v2"
	if err := first.PutMissionIfVersion(updated, 1); err != nil {
		t.Fatal(err)
	}
	final := updated
	final.Version = 3
	final.Objective = "v3"
	if err := second.PutMissionIfVersion(final, 2); err != nil {
		t.Fatalf("CAS rejected current durable version because local cache was stale: %v", err)
	}
	got, err := first.GetMission(original.ID)
	if err != nil || got.Version != 3 || got.Objective != "v3" {
		t.Fatalf("final mission=%+v err=%v", got, err)
	}
}

func TestPersistentListEventsDoesNotMaterializeMissingMissionFiles(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListEvents("mis_never_created"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ListEvents missing mission error=%v, want not found", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "events"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("read-only ListEvents created %d event files", len(entries))
	}
}

func TestPersistentAppendEventRefreshesMissionCreatedByAnotherInstance(t *testing.T) {
	root := t.TempDir()
	first, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewJSONStore(root)
	if err != nil {
		t.Fatal(err)
	}
	mission := Mission{ID: "mis_cross_event", Version: 1, OrganizationID: "org_a", State: MissionReady}
	if err := first.CreateMission(mission); err != nil {
		t.Fatal(err)
	}
	event := Event{ID: "evt_cross_event", MissionID: mission.ID, OrganizationID: "org_a", Type: "mission.created"}
	if err := second.AppendEvent(event); err != nil {
		t.Fatalf("cross-instance AppendEvent rejected newly persisted mission: %v", err)
	}
}

func TestJSONStoreDuplicateEventAtCapacityIsIdempotent(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		t.Run(map[bool]string{false: "memory", true: "persistent"}[persistent], func(t *testing.T) {
			var store *JSONStore
			var err error
			if persistent {
				store, err = NewJSONStore(t.TempDir())
			} else {
				store = NewMemoryStore()
			}
			if err != nil {
				t.Fatal(err)
			}
			mission := Mission{ID: "mis_event_capacity", Version: 1, OrganizationID: "org_a"}
			if err := store.CreateMission(mission); err != nil {
				t.Fatal(err)
			}
			events := make([]Event, maxJSONStoreEventCount)
			for i := range events {
				events[i] = Event{ID: fmt.Sprintf("evt_%05d", i), MissionID: mission.ID, OrganizationID: mission.OrganizationID, Type: "test"}
			}
			if persistent {
				path := filepath.Join(store.root, "events", mission.ID+".json")
				if err := writeJSONAtomicBounded(path, events, maxJSONStoreEventFileBytes, "test event file"); err != nil {
					t.Fatal(err)
				}
			} else {
				store.events[mission.ID] = events
			}
			if err := store.AppendEvent(events[len(events)-1]); err != nil {
				t.Fatalf("duplicate at exact event capacity was not idempotent: %v", err)
			}
		})
	}
}

func TestMissionPersistenceBoundsOversizedAndCyclicGraphs(t *testing.T) {
	store := NewMemoryStore()
	if err := store.CreateMission(Mission{ID: "mis_too_large", Version: 1, Objective: strings.Repeat("x", maxMissionRecordBytes+1)}); err == nil {
		t.Fatal("oversized mission was accepted")
	}
	cycle := map[string]any{}
	cycle["self"] = cycle
	if err := store.CreateMission(Mission{ID: "mis_cycle", Version: 1, Plan: []Step{{ID: "step_1", Input: cycle}}}); err == nil {
		t.Fatal("cyclic mission JSON graph was accepted")
	}
	tooManySteps := make([]Step, maxMissionPlanSteps+1)
	if err := store.CreateMission(Mission{ID: "mis_steps", Version: 1, Plan: tooManySteps}); err == nil {
		t.Fatal("mission with too many plan steps was accepted")
	}
}
