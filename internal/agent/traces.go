package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TraceSpan struct {
	TraceID        string         `json:"trace_id"`
	OrganizationID string         `json:"organization_id,omitempty"`
	SpanID         string         `json:"span_id"`
	ParentID       string         `json:"parent_id,omitempty"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	StartAt        time.Time      `json:"start_at"`
	EndAt          *time.Time     `json:"end_at,omitempty"`
	Duration       time.Duration  `json:"duration_ns,omitempty"`
	Attributes     map[string]any `json:"attributes,omitempty"`
	Error          string         `json:"error,omitempty"`
}

type TraceStore struct {
	mu    sync.RWMutex
	root  string
	spans []TraceSpan
}

const (
	maxTraceSpans          = 10_000
	maxTraceSpanRecordSize = 64 << 10
	maxTraceFileBytes      = 20 << 20
)

func NewTraceStore(root string) (*TraceStore, error) {
	store := &TraceStore{root: root, spans: []TraceSpan{}}
	if root == "" {
		return store, nil
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(root, "spans.json")
	if err := withFileLock(path+".lock", func() error {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		if err := readJSONBounded(path, &store.spans, maxTraceFileBytes); err != nil {
			return err
		}
		return validateTraceSpanCollection(store.spans)
	}); err != nil {
		return nil, err
	}
	return store, nil
}

type SpanHandle struct {
	store  *TraceStore
	index  int
	spanID string
	ended  bool
}

func (h *SpanHandle) ID() string {
	if h == nil || h.store == nil {
		return ""
	}
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()
	if h.spanID != "" {
		return h.spanID
	}
	if h.index < 0 || h.index >= len(h.store.spans) {
		return ""
	}
	return h.store.spans[h.index].SpanID
}

func (s *TraceStore) Start(traceID, parentID, name string, attributes map[string]any) *SpanHandle {
	return s.StartForOrganization("local", traceID, parentID, name, attributes)
}

func (s *TraceStore) StartForOrganization(organizationID, traceID, parentID, name string, attributes map[string]any) *SpanHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.spans) >= maxTraceSpans {
		return &SpanHandle{store: s, index: -1, ended: true}
	}
	budget := missionJSONBudget{}
	if err := validateBoundedJSONValue(reflect.ValueOf(attributes), 0, &budget); err != nil || budget.bytes > maxTraceSpanRecordSize {
		return &SpanHandle{store: s, index: -1, ended: true}
	}
	if traceID == "" {
		traceID = "tr_" + uuid.NewString()
	}
	redactedAttributes, _ := RedactValue(attributes).(map[string]any)
	span := TraceSpan{TraceID: limitError(traceID, 128), OrganizationID: limitError(normalizedOrganizationID(organizationID), 256), SpanID: "sp_" + uuid.NewString(), ParentID: limitError(parentID, 128), Name: RedactDLP(limitError(name, 1024)), Status: "running", StartAt: time.Now().UTC(), Attributes: redactedAttributes}
	candidate := append(append([]TraceSpan(nil), s.spans...), span)
	if err := validateTraceSpanCollection(candidate); err != nil {
		return &SpanHandle{store: s, index: -1, ended: true}
	}
	s.spans = candidate
	return &SpanHandle{store: s, index: len(s.spans) - 1, spanID: span.SpanID}
}

func (h *SpanHandle) End(status string, runErr error) {
	if h == nil || h.store == nil || h.ended {
		return
	}
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	if h.ended {
		return
	}
	index := h.index
	if h.spanID != "" {
		index = -1
		for candidate := range h.store.spans {
			if h.store.spans[candidate].SpanID == h.spanID {
				index = candidate
				break
			}
		}
	}
	if index < 0 || index >= len(h.store.spans) {
		return
	}
	now := time.Now().UTC()
	span := &h.store.spans[index]
	previous := *span
	span.Status = RedactDLP(limitError(status, 128))
	span.EndAt = &now
	span.Duration = now.Sub(span.StartAt)
	if runErr != nil {
		span.Status = "error"
		span.Error = RedactDLP(limitError(runErr.Error(), 2000))
	}
	if err := validateTraceSpanCollection(h.store.spans); err != nil {
		*span = previous
		slog.Error("agent trace span exceeded persistence limits", "trace_id", previous.TraceID, "span_id", previous.SpanID, "error", err)
		h.ended = true
		return
	}
	if err := h.store.persistLocked(index); err != nil {
		h.store.spans[index] = previous
		slog.Error("agent trace persistence failed", "trace_id", previous.TraceID, "span_id", previous.SpanID, "error", err)
		return
	}
	h.ended = true
}

func (s *TraceStore) List(traceID string, limit int) []TraceSpan {
	return s.ListForOrganization("local", traceID, limit)
}

func (s *TraceStore) ListForOrganization(organizationID, traceID string, limit int) []TraceSpan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	result := make([]TraceSpan, 0, len(s.spans))
	for _, span := range s.spans {
		if normalizedOrganizationID(span.OrganizationID) == normalizedOrganizationID(organizationID) && (traceID == "" || span.TraceID == traceID) {
			result = append(result, cloneTraceSpan(span))
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].StartAt.Before(result[j].StartAt) })
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (s *TraceStore) persistLocked(index int) error {
	if s.root == "" {
		return nil
	}
	if len(s.spans) > maxTraceSpans {
		return errors.New("trace span count exceeds persistence limit")
	}
	if err := validateTraceSpanCollection(s.spans); err != nil {
		return err
	}
	path := filepath.Join(s.root, "spans.json")
	if index < 0 || index >= len(s.spans) {
		return errors.New("trace span index is invalid")
	}
	spanToPersist := cloneTraceSpan(s.spans[index])
	return withFileLock(path+".lock", func() error {
		var disk []TraceSpan
		if err := readJSONBounded(path, &disk, maxTraceFileBytes); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := validateTraceSpanCollection(disk); err != nil {
			return err
		}
		found := false
		for i := range disk {
			if disk[i].SpanID == spanToPersist.SpanID {
				disk[i] = spanToPersist
				found = true
				break
			}
		}
		if !found {
			disk = append(disk, spanToPersist)
		}
		if err := validateTraceSpanCollection(disk); err != nil {
			return err
		}
		byID := make(map[string]int, len(disk))
		for i := range disk {
			byID[disk[i].SpanID] = i
		}
		merged := make([]TraceSpan, 0, len(disk)+len(s.spans))
		for _, persisted := range disk {
			merged = append(merged, cloneTraceSpan(persisted))
		}
		for _, local := range s.spans {
			if _, exists := byID[local.SpanID]; !exists {
				merged = append(merged, cloneTraceSpan(local))
			}
		}
		if err := validateTraceSpanCollection(merged); err != nil {
			return fmt.Errorf("cross-instance trace merge exceeds persistence limits: %w", err)
		}
		if err := writeJSONAtomicBounded(path, disk, maxTraceFileBytes, "trace file"); err != nil {
			return err
		}
		s.spans = merged
		return nil
	})
}

func validateTraceSpanCollection(spans []TraceSpan) error {
	if len(spans) > maxTraceSpans {
		return errors.New("trace span count exceeds persistence limit")
	}
	for _, span := range spans {
		encoded, err := json.Marshal(span)
		if err != nil {
			return fmt.Errorf("marshal trace span: %w", err)
		}
		if len(encoded) > maxTraceSpanRecordSize {
			return errors.New("trace span record exceeds persistence limit")
		}
	}
	encoded, err := json.MarshalIndent(spans, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trace file: %w", err)
	}
	if len(encoded)+1 > maxTraceFileBytes {
		return errors.New("trace file exceeds persistence limit")
	}
	return nil
}

func cloneTraceSpan(span TraceSpan) TraceSpan {
	if span.EndAt != nil {
		endAt := *span.EndAt
		span.EndAt = &endAt
	}
	if attributes, ok := RedactValue(span.Attributes).(map[string]any); ok {
		span.Attributes = attributes
	}
	return span
}
