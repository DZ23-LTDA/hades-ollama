package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWebhookReplayClaimPersistsAndRejectsReplay(t *testing.T) {
	root := t.TempDir()
	store, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Claim("sch_a", "evt_1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Claim("sch_a", "evt_1"); !errors.Is(err, ErrWebhookReplay) {
		t.Fatalf("same-process replay error = %v", err)
	}
	reloaded, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.Claim("sch_a", "evt_1"); !errors.Is(err, ErrWebhookReplay) {
		t.Fatalf("restart replay error = %v", err)
	}
	if err := reloaded.Claim("sch_b", "evt_1"); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookReplayBeginIsAtomicAcrossStoreInstances(t *testing.T) {
	root := t.TempDir()
	first, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	stores := []*WebhookReplayStore{first, second}
	results := make(chan error, len(stores))
	var wg sync.WaitGroup
	for _, store := range stores {
		wg.Add(1)
		go func(store *WebhookReplayStore) {
			defer wg.Done()
			results <- store.Begin("sch_atomic", "event-atomic")
		}(store)
	}
	wg.Wait()
	close(results)
	var accepted, inProgress int
	for err := range results {
		if err == nil {
			accepted++
		} else if errors.Is(err, ErrWebhookReplayInProgress) {
			inProgress++
		} else {
			t.Fatalf("unexpected Begin error: %v", err)
		}
	}
	if accepted != 1 || inProgress != 1 {
		t.Fatalf("accepted=%d in-progress=%d, want one of each", accepted, inProgress)
	}
}

func TestWebhookReplayStalePendingCanBeRetriedAndMissionIDIsStable(t *testing.T) {
	root := t.TempDir()
	store, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Begin("sch_a", "evt_retry"); err != nil {
		t.Fatal(err)
	}
	var claims map[string]webhookReplayClaim
	data, err := os.ReadFile(filepath.Join(root, "claims.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &claims); err != nil {
		t.Fatal(err)
	}
	for key, claim := range claims {
		claim.UpdatedAt = time.Now().UTC().Add(-webhookPendingLease - time.Second)
		claims[key] = claim
	}
	data, err = json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "claims.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Begin("sch_a", "evt_retry"); err != nil {
		t.Fatalf("stale pending claim was not retried: %v", err)
	}
	first, err := store.MissionID("sch_a", "evt_retry")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.MissionID("sch_a", "evt_retry")
	if err != nil || first != second || !validSnapshotID(first) {
		t.Fatalf("mission IDs are not stable/safe: first=%q second=%q err=%v", first, second, err)
	}
	otherSchedule, err := store.MissionID("sch_b", "evt_retry")
	if err != nil || otherSchedule == first {
		t.Fatalf("different schedules must not share a mission ID: %q %q err=%v", first, otherSchedule, err)
	}
}

func TestWebhookReplayMigratesLegacyAcceptedClaims(t *testing.T) {
	root := t.TempDir()
	legacy, err := json.Marshal(map[string]time.Time{"sch_a\x00evt_legacy": time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "claims.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewWebhookReplayStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Begin("sch_a", "evt_legacy"); !errors.Is(err, ErrWebhookReplay) {
		t.Fatalf("legacy accepted claim was not preserved: %v", err)
	}
}
