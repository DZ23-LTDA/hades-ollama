package agent

import (
	"fmt"
	"testing"
	"time"
)

// TestSAMLSweepRemovesExpiredRequests proves SEC-07: abandoned (expired) SAML
// requests are collected by the sweep instead of lingering in memory forever.
func TestSAMLSweepRemovesExpiredRequests(t *testing.T) {
	s := &SAMLService{requests: make(map[string]trackedSAMLRequest)}
	now := time.Now().UTC()

	for i := range 50 {
		s.requests[fmt.Sprintf("expired-%d", i)] = trackedSAMLRequest{
			RequestID: fmt.Sprintf("id-%d", i),
			ExpiresAt: now.Add(-time.Minute),
		}
	}
	s.requests["still-valid"] = trackedSAMLRequest{RequestID: "valid", ExpiresAt: now.Add(time.Minute)}

	s.mu.Lock()
	s.sweepExpiredRequestsLocked(now)
	s.mu.Unlock()

	if len(s.requests) != 1 {
		t.Fatalf("expected sweep to leave 1 valid request, got %d", len(s.requests))
	}
	if _, ok := s.requests["still-valid"]; !ok {
		t.Fatalf("sweep removed the still-valid request")
	}
}

// TestSAMLRequestQuotaBoundsGrowth proves SEC-07: exceeding the pending-request
// quota does not grow the map without bound; the capacity stays capped.
func TestSAMLRequestQuotaBoundsGrowth(t *testing.T) {
	s := &SAMLService{requests: make(map[string]trackedSAMLRequest)}
	now := time.Now().UTC()

	// Insert far more live requests than the quota allows; none expire yet, so
	// only the eviction path can keep the map bounded.
	total := maxPendingSAMLRequests + 500
	for i := range total {
		s.trackRequest(fmt.Sprintf("relay-%d", i), fmt.Sprintf("id-%d", i), "/", now)
	}

	if len(s.requests) > maxPendingSAMLRequests {
		t.Fatalf("pending SAML requests exceeded quota: got %d, cap %d", len(s.requests), maxPendingSAMLRequests)
	}
}

// TestSAMLTrackRequestSweepsOnInsert proves that inserting a fresh request also
// reclaims entries that have already expired, keeping memory bounded under a
// steady stream of abandoned logins.
func TestSAMLTrackRequestSweepsOnInsert(t *testing.T) {
	s := &SAMLService{requests: make(map[string]trackedSAMLRequest)}
	base := time.Now().UTC()

	for i := range 10 {
		s.requests[fmt.Sprintf("old-%d", i)] = trackedSAMLRequest{ExpiresAt: base.Add(-time.Second)}
	}

	// A new insert happening after the old entries expired should sweep them.
	s.trackRequest("new", "new-id", "/", base)

	if len(s.requests) != 1 {
		t.Fatalf("expected expired entries to be swept on insert, got %d entries", len(s.requests))
	}
	if _, ok := s.requests["new"]; !ok {
		t.Fatalf("new request missing after insert")
	}
}
