package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestEgressSSRFBlocksInternalIPs proves that SSRF attempts against loopback, private ranges,
// cloud metadata (169.254.169.254), IPv4-mapped IPv6, and CGNAT are rejected fail-closed.
func TestEgressSSRFBlocksInternalIPs(t *testing.T) {
	blockedIPs := []string{
		"127.0.0.1",
		"127.0.0.2",
		"::1",
		"10.0.0.1",
		"10.254.0.1",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.1.1",
		"192.168.0.254",
		"169.254.169.254", // Cloud metadata
		"169.254.1.1",     // Link-local
		"fe80::1",         // IPv6 link-local
		"::ffff:127.0.0.1",
		"::ffff:10.0.0.1",
		"::ffff:169.254.169.254",
		"100.64.0.1",      // CGNAT
		"100.127.255.254", // CGNAT
		"0.0.0.0",
		"::",
		"255.255.255.255",
	}

	for _, ipStr := range blockedIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("failed to parse test IP: %s", ipStr)
		}
		blocked, reason := ClassifyEgressIP(ip)
		if !blocked {
			t.Errorf("SSRF BYPASS BUG: IP %s was NOT blocked! (reason: %s)", ipStr, reason)
		}

		// Also verify via ResolveAllPublicIPs
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := ResolveAllPublicIPs(ctx, ipStr)
		cancel()
		if err == nil {
			t.Errorf("SSRF BYPASS BUG: ResolveAllPublicIPs allowed IP %s", ipStr)
		}
	}
}

// TestEgressDNSRebindingMixedRecordsRejected verifies that if a DNS query returns both a public IP
// and an internal/private IP, the entire host is rejected fail-closed to prevent DNS rebinding attacks.
func TestEgressDNSRebindingMixedRecordsRejected(t *testing.T) {
	ctx := context.Background()

	// Simulated DNS lookup returning a public IP AND a loopback/private IP
	mixedResolver := func(_ context.Context, host string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("93.184.216.34"), // public (example.com)
			net.ParseIP("127.0.0.1"),     // malicious rebinding target
		}, nil
	}

	_, err := ResolveAllPublicIPsWithResolver(ctx, "rebind.attacker.com", mixedResolver)
	if err == nil {
		t.Fatal("DNS REBINDING BYPASS BUG: host with mixed public and loopback IP was accepted!")
	}
	if !errors.Is(err, ErrEgressBlockedDNSRebind) {
		t.Fatalf("expected ErrEgressBlockedDNSRebind, got: %v", err)
	}

	// Also test mixed with cloud metadata 169.254.169.254
	metadataResolver := func(_ context.Context, host string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("93.184.216.34"),
			net.ParseIP("169.254.169.254"),
		}, nil
	}

	_, err = ResolveAllPublicIPsWithResolver(ctx, "metadata-rebind.attacker.com", metadataResolver)
	if err == nil {
		t.Fatal("DNS REBINDING BYPASS BUG: host with metadata IP was accepted!")
	}
	if !errors.Is(err, ErrEgressBlockedDNSRebind) {
		t.Fatalf("expected ErrEgressBlockedDNSRebind, got: %v", err)
	}
}

// TestEgressRedirectBlocksUnapprovedHost verifies that redirects to unapproved external or internal
// hosts are strictly blocked by the egress redirect policy.
func TestEgressRedirectBlocksUnapprovedHost(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("sensitive internal data"))
	}))
	defer targetServer.Close()

	// Redirecting server attempts to bounce client to targetServer
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL+"/exfiltrate", http.StatusFound)
	}))
	defer redirectServer.Close()

	client := NewSafeEgressHTTPClient(EgressOptions{
		Callsite:      "test_redirect",
		Timeout:       5 * time.Second,
		AllowLoopback: true,
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, redirectServer.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer super-secret-token")

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		t.Fatal("REDIRECT BYPASS BUG: redirect to different host was followed!")
	}
	if !errors.Is(err, ErrEgressRedirectDisallowed) && !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect disallowed error, got: %v", err)
	}
}

// TestEgressRedirectStripsSensitiveCredentials proves that any redirect attempt immediately strips
// credentials (Authorization, Cookie, X-Api-Key) before any subsequent dispatch.
func TestEgressRedirectStripsSensitiveCredentials(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://approved.example.com/start", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-12345")
	req.Header.Set("Cookie", "session=abcdef")
	req.Header.Set("X-Api-Key", "api-key-999")
	req.Header.Set("Custom-Header", "preserve-this")

	StripSensitiveEgressHeaders(req)

	if req.Header.Get("Authorization") != "" {
		t.Errorf("CREDENTIAL LEAK BUG: Authorization header was not stripped!")
	}
	if req.Header.Get("Cookie") != "" {
		t.Errorf("CREDENTIAL LEAK BUG: Cookie header was not stripped!")
	}
	if req.Header.Get("X-Api-Key") != "" {
		t.Errorf("CREDENTIAL LEAK BUG: X-Api-Key header was not stripped!")
	}
	if req.Header.Get("Custom-Header") != "preserve-this" {
		t.Errorf("expected non-sensitive header to be preserved")
	}
}

// TestEgressResourceExhaustionPayloadBounded verifies that reading a payload larger than the configured
// limit fails closed, protecting the agent against memory exhaustion (DoS).
func TestEgressResourceExhaustionPayloadBounded(t *testing.T) {
	// Generate an oversized stream of 5MB
	oversizedStream := strings.NewReader(strings.Repeat("A", 5*1024*1024))
	maxLimit := int64(1024 * 1024) // 1MB limit

	_, err := ReadBoundedBody(oversizedStream, maxLimit)
	if err == nil {
		t.Fatal("RESOURCE EXHAUSTION BUG: oversized payload was accepted!")
	}
	if !errors.Is(err, ErrEgressPayloadExceedsLimit) {
		t.Fatalf("expected ErrEgressPayloadExceedsLimit, got: %v", err)
	}

	// Normal payload under limit succeeds
	validStream := strings.NewReader("hello bounded world")
	data, err := ReadBoundedBody(validStream, maxLimit)
	if err != nil {
		t.Fatalf("unexpected error on valid payload: %v", err)
	}
	if string(data) != "hello bounded world" {
		t.Fatalf("unexpected payload content: %s", string(data))
	}
}

// TestEgressAuditLogRecordsDecisions verifies that all egress decisions are recorded with full audit trail.
func TestEgressAuditLogRecordsDecisions(t *testing.T) {
	auditor := NewEgressAuditStore(100)
	auditor.Record(EgressDecision{
		Callsite:    "connectors",
		Destination: "https://api.github.com/repos",
		Host:        "api.github.com",
		Allowed:     true,
		Reason:      "approved public destination",
	})
	auditor.Record(EgressDecision{
		Callsite:    "whatsapp",
		Destination: "http://169.254.169.254/latest/meta-data",
		Host:        "169.254.169.254",
		Allowed:     false,
		Reason:      "blocked cloud metadata service",
	})

	list := auditor.List(10)
	if len(list) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(list))
	}
	if list[0].Allowed != true || list[0].Callsite != "connectors" {
		t.Errorf("unexpected first entry: %+v", list[0])
	}
	if list[1].Allowed != false || list[1].Callsite != "whatsapp" {
		t.Errorf("unexpected second entry: %+v", list[1])
	}

	filtered := auditor.Filter("whatsapp", 10)
	if len(filtered) != 1 || filtered[0].Allowed != false {
		t.Errorf("expected 1 filtered entry for whatsapp, got: %+v", filtered)
	}
}

// TestEgressPeerVerificationRejectsHijackedSocket proves that if the underlying connection dials
// an IP that is restricted, the connection is closed and rejected even if initial DNS looked benign.
func TestEgressPeerVerificationRejectsHijackedSocket(t *testing.T) {
	dialer := NewEgressTransport(EgressOptions{
		Callsite:      "test_peer",
		AllowLoopback: false,
		Lookup: func(_ context.Context, _ string) ([]net.IP, error) {
			// Malicious resolver returns loopback
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		},
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://fake.safe.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{Transport: dialer, Timeout: 2 * time.Second}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("PEER HIJACK BUG: connection to loopback without explicit permission was allowed!")
	}
}
