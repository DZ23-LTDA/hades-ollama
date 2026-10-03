package agent

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestEgressAuditOrganizationIsolation proves SEC-10(a): audit records carry an
// organization_id and reads scoped to one organization never expose another
// organization's egress destinations.
func TestEgressAuditOrganizationIsolation(t *testing.T) {
	store := NewEgressAuditStore(100)
	store.Record(EgressDecision{OrganizationID: "org-a", Callsite: "connectors", Destination: "https://a.example.com", Allowed: true})
	store.Record(EgressDecision{OrganizationID: "org-b", Callsite: "connectors", Destination: "https://b.example.com", Allowed: true})
	store.Record(EgressDecision{OrganizationID: "", Callsite: "connectors", Destination: "https://legacy.example.com", Allowed: true})

	orgA := store.ListForOrganization("org-a", 100)
	if len(orgA) != 1 || orgA[0].Destination != "https://a.example.com" {
		t.Fatalf("org-a should see only its own record, got %+v", orgA)
	}
	for _, entry := range orgA {
		if strings.Contains(entry.Destination, "b.example.com") {
			t.Fatalf("org-a leaked org-b destination: %q", entry.Destination)
		}
	}

	orgB := store.ListForOrganization("org-b", 100)
	if len(orgB) != 1 || orgB[0].Destination != "https://b.example.com" {
		t.Fatalf("org-b should see only its own record, got %+v", orgB)
	}

	// The reserved local scope additionally sees ownerless (legacy) records.
	local := store.ListForOrganization(LocalOrganizationID, 100)
	if len(local) != 1 || local[0].Destination != "https://legacy.example.com" {
		t.Fatalf("local scope should see only the ownerless record, got %+v", local)
	}

	// Callsite filtering is also organization-scoped.
	if got := store.FilterForOrganization("org-a", "connectors", 100); len(got) != 1 || got[0].OrganizationID != "org-a" {
		t.Fatalf("FilterForOrganization leaked cross-org records: %+v", got)
	}
}

// TestSanitizeEgressURL proves SEC-10(b): the sanitizer strips userinfo, query
// and fragment, keeping only scheme://host/path.
func TestSanitizeEgressURL(t *testing.T) {
	cases := map[string]string{
		"https://user:secret@api.example.com/v1/x?token=abc123&k=v#frag": "https://api.example.com/v1/x",
		"https://api.example.com/path?apikey=leak":                       "https://api.example.com/path",
		"https://api.example.com":                                        "https://api.example.com",
	}
	for raw, want := range cases {
		got := sanitizeEgressURL(raw)
		if got != want {
			t.Fatalf("sanitizeEgressURL(%q) = %q, want %q", raw, got, want)
		}
		if strings.Contains(got, "secret") || strings.Contains(got, "token=") || strings.Contains(got, "apikey=") || strings.Contains(got, "#") {
			t.Fatalf("sanitizeEgressURL(%q) leaked sensitive data: %q", raw, got)
		}
	}
}

// TestEgressRedirectRecordsSanitizedDestination proves SEC-10(b) end-to-end: the
// redirect audit path logs only scheme://host/path, never the credentials or
// query string present in the redirect target URL.
func TestEgressRedirectRecordsSanitizedDestination(t *testing.T) {
	DefaultEgressAuditStore.Clear()
	t.Cleanup(DefaultEgressAuditStore.Clear)

	check := NewEgressCheckRedirect("connectors", false)

	origin, _ := http.NewRequest(http.MethodGet, "https://good.example.com/start", nil)
	target, _ := http.NewRequest(http.MethodGet, "https://evil.example.com/steal?token=secret123#frag", nil)
	target.URL.User = url.UserPassword("user", "hunter2")

	if err := check(target, []*http.Request{origin}); err == nil {
		t.Fatal("expected cross-host redirect to be blocked")
	}

	entries := DefaultEgressAuditStore.List(10)
	if len(entries) == 0 {
		t.Fatal("expected a recorded egress decision for the blocked redirect")
	}
	last := entries[len(entries)-1]
	if last.Destination != "https://evil.example.com/steal" {
		t.Fatalf("recorded destination = %q, want sanitized scheme://host/path", last.Destination)
	}
	if strings.Contains(last.Destination, "token=") || strings.Contains(last.Destination, "secret123") ||
		strings.Contains(last.Destination, "hunter2") || strings.Contains(last.Destination, "user:") ||
		strings.Contains(last.Destination, "#") {
		t.Fatalf("recorded destination leaked sensitive data: %q", last.Destination)
	}
}
