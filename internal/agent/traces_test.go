package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTraceStoreOrganizationScope(t *testing.T) {
	store, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	span := store.StartForOrganization("org-a", "tr_scope", "", "tenant span", nil)
	span.End("ok", nil)
	if got := store.ListForOrganization("org-b", "", 100); len(got) != 0 {
		t.Fatalf("cross-tenant trace list returned %+v", got)
	}
	if got := store.ListForOrganization("org-a", "tr_scope", 100); len(got) != 1 || got[0].OrganizationID != "org-a" {
		t.Fatalf("same-tenant trace list=%+v", got)
	}
}

func TestTraceStoreRollsBackSpanWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewTraceStore(root)
	if err != nil {
		t.Fatal(err)
	}
	span := store.StartForOrganization("org-a", "tr_rollback", "", "operation", nil)
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	span.End("error", os.ErrPermission)
	spans := store.ListForOrganization("org-a", "tr_rollback", 10)
	if len(spans) != 1 || spans[0].Status != "running" || spans[0].EndAt != nil {
		t.Fatalf("span changed after persistence failure: %+v", spans)
	}
}

func TestTraceStoreRejectsOversizedPersistenceFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "spans.json"), []byte(strings.Repeat(" ", maxTraceFileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTraceStore(root); err == nil {
		t.Fatal("oversized trace file was loaded")
	}
}

func TestTraceStoreBoundsSpanRecordsAndAggregateCount(t *testing.T) {
	if err := validateTraceSpanCollection([]TraceSpan{{SpanID: "sp_large", Attributes: map[string]any{"payload": strings.Repeat("x", maxTraceSpanRecordSize)}}}); err == nil {
		t.Fatal("oversized trace span record was accepted")
	}
	spans := make([]TraceSpan, maxTraceSpans+1)
	if err := validateTraceSpanCollection(spans); err == nil {
		t.Fatal("trace span count above the configured bound was accepted")
	}
	store, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	tooLarge := store.Start("tr_large", "", "oversized", map[string]any{"payload": strings.Repeat("x", maxTraceSpanRecordSize)})
	if tooLarge.ID() != "" || len(store.List("tr_large", 100)) != 0 {
		t.Fatal("oversized span grew in-memory trace storage")
	}
}

func TestTraceStoreListReturnsDeepCopies(t *testing.T) {
	store, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	span := store.Start("tr_copy", "", "operation", map[string]any{"safe": "value"})
	span.End("ok", nil)
	listed := store.List("tr_copy", 10)
	listed[0].Attributes["safe"] = "mutated"
	if got := store.List("tr_copy", 10)[0].Attributes["safe"]; got != "value" {
		t.Fatalf("caller mutated internal trace state: %v", got)
	}
}

func TestTraceStoreQuotaValidatorRejectsUnserializableSpan(t *testing.T) {
	if err := validateTraceSpanCollection([]TraceSpan{{SpanID: "sp_invalid", Attributes: map[string]any{"unsupported": func() {}}}}); err == nil {
		t.Fatalf("unserializable span error=%v", err)
	}
}

func TestTraceStoreMergesConcurrentInstances(t *testing.T) {
	root := t.TempDir()
	first, err := NewTraceStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTraceStore(root)
	if err != nil {
		t.Fatal(err)
	}
	spanA := first.StartForOrganization("org-a", "tr_concurrent", "", "first", nil)
	spanB := second.StartForOrganization("org-a", "tr_concurrent", "", "second", nil)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() { defer wait.Done(); spanA.End("ok", nil) }()
	go func() { defer wait.Done(); spanB.End("ok", nil) }()
	wait.Wait()
	reloaded, err := NewTraceStore(root)
	if err != nil {
		t.Fatal(err)
	}
	spans := reloaded.ListForOrganization("org-a", "tr_concurrent", 10)
	if len(spans) != 2 {
		t.Fatalf("concurrent trace writes lost spans: %+v", spans)
	}
}

func TestTraceStoreRejectsOverLimitCrossInstanceMerge(t *testing.T) {
	root := t.TempDir()
	store, err := NewTraceStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const targetID = "sp_shared_target"
	local := make([]TraceSpan, maxTraceSpans)
	disk := make([]TraceSpan, maxTraceSpans)
	now := time.Now().UTC()
	for index := 0; index < maxTraceSpans; index++ { //nolint:intrange // index is part of deterministic fixture IDs
		localID := fmt.Sprintf("sp_local_%05d", index)
		diskID := fmt.Sprintf("sp_disk_%05d", index)
		if index == 0 {
			localID = targetID
			diskID = targetID
		}
		local[index] = TraceSpan{SpanID: localID, TraceID: "tr_local", Name: "local", Status: "running", StartAt: now}
		disk[index] = TraceSpan{SpanID: diskID, TraceID: "tr_disk", Name: "disk", Status: "ok", StartAt: now}
	}
	store.spans = local
	path := filepath.Join(root, "spans.json")
	if err := writeJSONAtomicBounded(path, disk, maxTraceFileBytes, "trace file test fixture"); err != nil {
		t.Fatal(err)
	}
	handle := &SpanHandle{store: store, index: 0, spanID: targetID}
	handle.End("ok", nil)
	if len(store.spans) != maxTraceSpans || store.spans[0].Status != "running" || store.spans[0].EndAt != nil {
		t.Fatalf("failed merge changed local bounded state: count=%d target=%+v", len(store.spans), store.spans[0])
	}
	var persisted []TraceSpan
	if err := readJSONBounded(path, &persisted, maxTraceFileBytes); err != nil {
		t.Fatal(err)
	}
	if len(persisted) != maxTraceSpans || persisted[0].Status != "ok" {
		t.Fatalf("failed merge mutated persisted state: count=%d first=%+v", len(persisted), persisted[0])
	}
}

func TestTraceStoreRejectsCyclicAndDeepAttributesBeforeGrowth(t *testing.T) {
	store, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	cycle := map[string]any{}
	cycle["self"] = cycle
	if span := store.Start("tr_cycle", "", "cycle", cycle); span.ID() != "" {
		t.Fatal("cyclic trace attributes were accepted")
	}
	var nested any = "leaf"
	for i := 0; i <= maxMissionRecordDepth; i++ {
		nested = []any{nested}
	}
	if span := store.Start("tr_deep", "", "deep", map[string]any{"nested": nested}); span.ID() != "" {
		t.Fatal("deep trace attributes were accepted")
	}
	if len(store.List("", 100)) != 0 {
		t.Fatal("rejected trace attributes grew internal storage")
	}
}
