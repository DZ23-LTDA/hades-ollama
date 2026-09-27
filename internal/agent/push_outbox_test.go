package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPushOutboxRejectsSensitivePayloadBeforePersisting(t *testing.T) {
	root := t.TempDir()
	outbox, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := outbox.Enqueue("org_1", "title", "body", map[string]any{"details": map[string]any{"api_key": "example-secret-value"}}); err == nil {
		t.Fatal("sensitive push payload was accepted")
	}
	if len(outbox.List()) != 0 {
		t.Fatalf("sensitive push payload remains in outbox: %+v", outbox.List())
	}
	data, err := os.ReadFile(filepath.Join(root, "outbox.json"))
	if err == nil && strings.Contains(string(data), "example-secret-value") {
		t.Fatal("sensitive value was persisted")
	}
}

func TestPushOutboxRejectsSensitiveLegacyRecordWithoutRewriting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "outbox.json")
	legacy := map[string]PushOutboxItem{
		"out_legacy": {
			ID:             "out_legacy",
			OrganizationID: "org_1",
			Title:          "title",
			Body:           "body",
			Data:           map[string]any{"nested": map[string]any{"api_key": "legacy-secret-value"}},
			NextAttemptAt:  time.Now().UTC(),
		},
	}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewPushOutbox(root); err == nil {
		t.Fatal("sensitive legacy outbox record was accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("sensitive legacy outbox record was rewritten instead of left for recovery")
	}
}

func TestPushServiceBlocksSensitiveDirectCallBeforeEgress(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-outbox-test-key-with-sufficient-length")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	if _, err := push.Register("ExponentPushToken[test]", "android", "user_1", "org_1"); err != nil {
		t.Fatal(err)
	}
	err = push.NotifyOrganization(context.Background(), "org_1", "title", "body", map[string]any{"api_key": "example-secret-value"})
	if err == nil {
		t.Fatal("sensitive push payload was sent")
	}
	if requests.Load() != 0 {
		t.Fatalf("push made %d requests with sensitive payload", requests.Load())
	}
}

func TestPushOutboxPersistsRetryAndCompletion(t *testing.T) {
	root := filepath.Join(t.TempDir(), "outbox")
	outbox, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Add(time.Second)
	item, err := outbox.Enqueue("org_1", "title", "body", map[string]any{"mission_id": "mis_1"})
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := outbox.ClaimDue(now)
	if err != nil || !ok || claimed.ID != item.ID || claimed.Attempts != 1 {
		t.Fatalf("claim=%+v ok=%v err=%v", claimed, ok, err)
	}
	if err := outbox.failAt(item.ID, claimed.LeaseToken, context.Canceled, now); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	pending := restarted.List()
	if len(pending) != 1 || pending[0].Attempts != 1 || pending[0].LastError == "" {
		t.Fatalf("pending after restart=%+v", pending)
	}
	claimed, ok, err = restarted.ClaimDue(pending[0].NextAttemptAt.Add(time.Second))
	if err != nil || !ok || claimed.Attempts != 2 {
		t.Fatalf("retry claim=%+v ok=%v err=%v", claimed, ok, err)
	}
	if err := restarted.Complete(item.ID, claimed.LeaseToken); err != nil {
		t.Fatal(err)
	}
	if len(restarted.List()) != 0 {
		t.Fatalf("outbox still contains completed item: %+v", restarted.List())
	}
}

func TestPushOutboxFencesStaleWorkerCompletionAndFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "outbox")
	first, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := first.Enqueue("org", "title", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Add(time.Second)
	stale, ok, err := first.ClaimDue(now)
	if err != nil || !ok || stale.LeaseToken == "" {
		t.Fatalf("first claim=%+v ok=%v err=%v", stale, ok, err)
	}
	current, ok, err := second.ClaimDue(now.Add(2 * time.Minute))
	if err != nil || !ok || current.ID != queued.ID || current.LeaseToken == "" || current.LeaseToken == stale.LeaseToken {
		t.Fatalf("reclaimed item=%+v ok=%v err=%v", current, ok, err)
	}
	if err := first.Complete(stale.ID, stale.LeaseToken); !errors.Is(err, errPushOutboxLeaseLost) {
		t.Fatalf("stale completion error=%v, want lease-lost", err)
	}
	if err := first.failAt(stale.ID, stale.LeaseToken, context.Canceled, now.Add(2*time.Minute)); !errors.Is(err, errPushOutboxLeaseLost) {
		t.Fatalf("stale failure error=%v, want lease-lost", err)
	}
	items := second.List()
	if len(items) != 1 || items[0].Attempts != 2 || items[0].LeaseToken != current.LeaseToken {
		t.Fatalf("stale worker mutated current claim: %+v", items)
	}
	if err := second.Complete(current.ID, current.LeaseToken); err != nil {
		t.Fatalf("current claim completion failed: %v", err)
	}
}

func TestPushOutboxConcurrentInstancesPreserveEnqueuesAndClaimsOnce(t *testing.T) {
	root := filepath.Join(t.TempDir(), "outbox")
	first, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan PushOutboxItem, 2)
	errs := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		item, err := first.Enqueue("org_1", "title-a", "body-a", map[string]any{"source": "first"})
		results <- item
		errs <- err
	}()
	go func() {
		defer group.Done()
		item, err := second.Enqueue("org_1", "title-b", "body-b", map[string]any{"source": "second"})
		results <- item
		errs <- err
	}()
	group.Wait()
	close(results)
	close(errs)
	ids := make(map[string]bool)
	for item := range results {
		ids[item.ID] = true
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(ids) != 2 {
		t.Fatalf("concurrent enqueues returned IDs=%v", ids)
	}
	if items := first.List(); len(items) != 2 {
		t.Fatalf("concurrent enqueues lost records: %+v", items)
	}
	now := time.Now().UTC().Add(time.Second)
	claims := make(chan PushOutboxItem, 2)
	claimErrs := make(chan error, 2)
	group.Add(2)
	go func() {
		defer group.Done()
		item, ok, err := first.ClaimDue(now)
		if ok {
			claims <- item
		}
		claimErrs <- err
	}()
	go func() {
		defer group.Done()
		item, ok, err := second.ClaimDue(now)
		if ok {
			claims <- item
		}
		claimErrs <- err
	}()
	group.Wait()
	close(claims)
	close(claimErrs)
	claimedIDs := make(map[string]bool)
	for item := range claims {
		claimedIDs[item.ID] = true
	}
	for err := range claimErrs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(claimedIDs) != 2 {
		t.Fatalf("concurrent claims did not claim each item exactly once: %v", claimedIDs)
	}
}

func TestPushOutboxReturnedDataIsDeepCopied(t *testing.T) {
	outbox, err := NewPushOutbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	item, err := outbox.Enqueue("org_1", "title", "body", map[string]any{"nested": map[string]any{"value": "safe"}})
	if err != nil {
		t.Fatal(err)
	}
	item.Data["nested"].(map[string]any)["value"] = "mutated"
	listed := outbox.List()
	if listed[0].Data["nested"].(map[string]any)["value"] != "safe" {
		t.Fatal("enqueue result mutated outbox state")
	}
	listed[0].Data["nested"].(map[string]any)["value"] = "mutated-again"
	if outbox.List()[0].Data["nested"].(map[string]any)["value"] != "safe" {
		t.Fatal("list result mutated outbox state")
	}
	claimed, ok, err := outbox.ClaimDue(time.Now().UTC().Add(time.Second))
	if err != nil || !ok {
		t.Fatalf("claim ok=%v err=%v", ok, err)
	}
	claimed.Data["nested"].(map[string]any)["value"] = "mutated-claim"
	if outbox.List()[0].Data["nested"].(map[string]any)["value"] != "safe" {
		t.Fatal("claim result mutated outbox state")
	}
}

func TestRuntimeFlushPushOutboxDeliversAndRemovesItem(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-outbox-test-key-with-sufficient-length")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	if _, err := push.Register("ExponentPushToken[test]", "android", "user_1", "org_1"); err != nil {
		t.Fatal(err)
	}
	outbox, err := NewPushOutbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := outbox.Enqueue("org_1", "title", "body", nil); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{push: push, pushOutbox: outbox, metrics: &RuntimeMetrics{}}
	runtime.flushPushOutbox(context.Background())
	if requests.Load() != 1 || len(outbox.List()) != 0 {
		t.Fatalf("requests=%d outbox=%+v", requests.Load(), outbox.List())
	}
	if metrics := runtime.Metrics(); metrics.PushDeliveryFailures != 0 || metrics.PushOutboxFailures != 0 {
		t.Fatalf("metrics=%+v", metrics)
	}
}

func TestRuntimeFlushPushOutboxRetainsFailedDelivery(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-outbox-test-key-with-sufficient-length")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("provider unavailable"))
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	if _, err := push.Register("ExponentPushToken[test]", "ios", "user_1", "org_1"); err != nil {
		t.Fatal(err)
	}
	outbox, err := NewPushOutbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := outbox.Enqueue("org_1", "title", "body", nil); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{push: push, pushOutbox: outbox, metrics: &RuntimeMetrics{}}
	runtime.flushPushOutbox(context.Background())
	pending := outbox.List()
	if len(pending) != 1 || pending[0].LastError == "" || pending[0].LeaseUntil != nil {
		t.Fatalf("pending=%+v", pending)
	}
	if metrics := runtime.Metrics(); metrics.PushDeliveryFailures != 1 || metrics.PushOutboxFailures != 0 {
		t.Fatalf("metrics=%+v", metrics)
	}
}

func TestPushOutboxExpiredLeaseFencesCompleteAndFail(t *testing.T) {
	outbox, err := NewPushOutbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	item, err := outbox.Enqueue("org_1", "title", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	outbox.mu.Lock()
	stored := outbox.items[item.ID]
	expired := time.Now().UTC().Add(-time.Second)
	stored.LeaseUntil = &expired
	stored.LeaseToken = "expired-worker"
	stored.Attempts = 1
	outbox.items[item.ID] = stored
	err = outbox.persistLocked()
	outbox.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := outbox.Complete(item.ID, "expired-worker"); !errors.Is(err, errPushOutboxLeaseLost) {
		t.Fatalf("Complete error=%v, want expired-lease fencing", err)
	}
	if err := outbox.Fail(item.ID, "expired-worker", errors.New("failure")); !errors.Is(err, errPushOutboxLeaseLost) {
		t.Fatalf("Fail error=%v, want expired-lease fencing", err)
	}
}

func TestPushOutboxExhaustedRetryDoesNotStarveDueItems(t *testing.T) {
	root := t.TempDir()
	outbox, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := outbox.Enqueue("org_1", "terminal", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	healthy, err := outbox.Enqueue("org_1", "healthy", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	outbox.mu.Lock()
	terminal = outbox.items[terminal.ID]
	terminal.Attempts = maxPushOutboxAttempts
	terminal.NextAttemptAt = time.Time{}
	outbox.items[terminal.ID] = terminal
	healthy = outbox.items[healthy.ID]
	healthy.NextAttemptAt = time.Time{}
	outbox.items[healthy.ID] = healthy
	err = outbox.persistLocked()
	outbox.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := outbox.ClaimDue(time.Now().UTC())
	if err != nil || !ok || claimed.ID != healthy.ID {
		t.Fatalf("claimed=%+v ok=%v err=%v, want the healthy due notification", claimed, ok, err)
	}
	if items := outbox.List(); len(items) != 1 || items[0].ID != healthy.ID {
		t.Fatalf("expired terminal record was not compacted without losing the healthy job: %+v", items)
	}
}

func TestPushOutboxTerminalPruningRespectsActiveLease(t *testing.T) {
	now := time.Now().UTC()
	leaseUntil := now.Add(time.Minute)
	item := PushOutboxItem{ID: "out_leased_terminal", OrganizationID: "org-a", Title: "title", Body: "body", Attempts: maxPushOutboxAttempts, NextAttemptAt: now.Add(-8 * 24 * time.Hour), LeaseUntil: &leaseUntil, LeaseToken: "active-token"}
	items := map[string]PushOutboxItem{item.ID: item}
	prunePushOutboxTerminals(items, now)
	if _, ok := items[item.ID]; !ok {
		t.Fatal("pruned a terminal item with an active delivery lease")
	}
	prunePushOutboxTerminals(items, leaseUntil.Add(time.Second))
	if _, ok := items[item.ID]; ok {
		t.Fatal("did not prune an expired, unleased terminal item")
	}
}

func TestPushOutboxEnforcesStorageQuotas(t *testing.T) {
	root := t.TempDir()
	outbox, err := NewPushOutbox(root)
	if err != nil {
		t.Fatal(err)
	}
	outbox.mu.Lock()
	outbox.items = make(map[string]PushOutboxItem, maxPushOutboxItems)
	for i := 0; i < maxPushOutboxItems; i++ {
		organizationID := "org-" + strconv.Itoa(i)
		if i < maxPushOutboxItemsPerTenant {
			organizationID = "org-full"
		}
		id := "out_quota_" + strconv.Itoa(i)
		outbox.items[id] = PushOutboxItem{ID: id, OrganizationID: organizationID, Title: "title", Body: "body", NextAttemptAt: time.Now().UTC()}
	}
	err = outbox.persistLocked()
	outbox.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := outbox.Enqueue("org-full", "title", "body", nil); !errors.Is(err, errPushOutboxQuota) {
		t.Fatalf("per-tenant quota error=%v", err)
	}
	if _, err := outbox.Enqueue("new-org", "title", "body", nil); !errors.Is(err, errPushOutboxQuota) {
		t.Fatalf("global quota error=%v", err)
	}
	if _, err := outbox.Enqueue("new-org", strings.Repeat("x", maxPushOutboxItemBytes), "body", nil); !errors.Is(err, errOutboundPayloadLimit) {
		t.Fatalf("per-item quota error=%v", err)
	}

	oversizedRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(oversizedRoot, "outbox.json"), []byte(strings.Repeat(" ", maxPushOutboxFileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPushOutbox(oversizedRoot); !errors.Is(err, errPushOutboxQuota) {
		t.Fatalf("oversized file error=%v", err)
	}
}

func TestRuntimeFlushPushOutboxIsBoundedByBatchSize(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-outbox-test-key-with-sufficient-length")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	if _, err := push.Register("ExponentPushToken[test]", "android", "user", "org"); err != nil {
		t.Fatal(err)
	}
	outbox, err := NewPushOutbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxPushOutboxBatch+1; i++ {
		if _, err := outbox.Enqueue("org", "title", "body", nil); err != nil {
			t.Fatal(err)
		}
	}
	runtime := &Runtime{push: push, pushOutbox: outbox, metrics: &RuntimeMetrics{}}
	runtime.flushPushOutbox(context.Background())
	if requests.Load() != maxPushOutboxBatch || len(outbox.List()) != 1 {
		t.Fatalf("first batch requests=%d items=%d", requests.Load(), len(outbox.List()))
	}
	runtime.flushPushOutbox(context.Background())
	if requests.Load() != maxPushOutboxBatch+1 || len(outbox.List()) != 0 {
		t.Fatalf("second batch requests=%d items=%d", requests.Load(), len(outbox.List()))
	}
	if _, err := outbox.Enqueue("org", "title", "body", nil); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	runtime.flushPushOutbox(canceled)
	if requests.Load() != maxPushOutboxBatch+1 || len(outbox.List()) != 1 {
		t.Fatalf("canceled flush requests=%d items=%d", requests.Load(), len(outbox.List()))
	}
}

func TestPushOutboxRejectsInvalidAttemptMetadata(t *testing.T) {
	root := t.TempDir()
	item := PushOutboxItem{ID: "out_invalid", OrganizationID: "org_1", Title: "title", Body: "body", Attempts: -1, NextAttemptAt: time.Now().UTC()}
	data, err := json.Marshal(map[string]PushOutboxItem{item.ID: item})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "outbox.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPushOutbox(root); err == nil {
		t.Fatal("invalid negative attempt metadata was accepted")
	}
}
