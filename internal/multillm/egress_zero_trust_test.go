package multillm

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
)

func TestProviderEgressSSRFBlocksPrivateAndMetadata(t *testing.T) {
	blockedIPs := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254", // Cloud metadata
		"::1",
		"::ffff:127.0.0.1",
		"100.64.0.1", // CGNAT
	}

	for _, ipStr := range blockedIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("failed to parse IP: %s", ipStr)
		}
		blocked, reason := ClassifyProviderEgressIP(ip)
		if !blocked {
			t.Errorf("PROVIDER SSRF BUG: IP %s was NOT blocked! (reason: %s)", ipStr, reason)
		}
	}

	// ValidateProviderEgressURL fail-closed
	err := ValidateProviderEgressURL("http://169.254.169.254/v1", false)
	if err == nil {
		t.Error("expected error for metadata provider URL, got nil")
	}

	err = ValidateProviderEgressURL("http://127.0.0.1:8000/v1", false)
	if err == nil {
		t.Error("expected error for loopback provider URL, got nil")
	}

	err = ValidateProviderEgressURL("http://10.1.2.3:8000/v1", false)
	if err == nil {
		t.Error("expected error for private network provider URL, got nil")
	}
}

func TestProviderEgressDNSRebindingRejected(t *testing.T) {
	mixedResolver := func(_ context.Context, _ string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("93.184.216.34"),
			net.ParseIP("127.0.0.1"),
		}, nil
	}

	_, err := ResolveProviderPublicIPsWithResolver(context.Background(), "rebind.provider.com", mixedResolver)
	if err == nil {
		t.Fatal("PROVIDER DNS REBINDING BUG: mixed public/loopback IP was accepted!")
	}
	if !errors.Is(err, ErrProviderBlockedDNSRebind) {
		t.Fatalf("expected ErrProviderBlockedDNSRebind, got: %v", err)
	}
}

func TestProviderEgressRedirectPolicyStripsCredentials(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer sk-secret-12345")
	req.Header.Set("X-Api-Key", "my-key")

	StripProviderCredentials(req)

	if req.Header.Get("Authorization") != "" {
		t.Error("PROVIDER CREDENTIAL LEAK BUG: Authorization was not stripped!")
	}
	if req.Header.Get("X-Api-Key") != "" {
		t.Error("PROVIDER CREDENTIAL LEAK BUG: X-Api-Key was not stripped!")
	}

	// Cross-host redirect rejection
	policy := CheckProviderRedirectPolicy("api.openai.com")
	reqVia, _ := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	reqTarget, _ := http.NewRequest(http.MethodGet, "https://malicious.attacker.com/steal", nil)

	err = policy(reqTarget, []*http.Request{reqVia})
	if err == nil {
		t.Fatal("PROVIDER REDIRECT BYPASS BUG: cross-host redirect was accepted!")
	}
}

func TestProviderEgressAuditStoreRecordsDecisions(t *testing.T) {
	store := NewProviderEgressAuditStore(50)
	store.Record(ProviderEgressDecision{
		Provider:    "groq",
		Destination: "https://api.groq.com/openai/v1",
		Host:        "api.groq.com",
		Allowed:     true,
		Reason:      "approved provider public destination",
	})
	store.Record(ProviderEgressDecision{
		Provider:    "attacker",
		Destination: "http://169.254.169.254/v1",
		Host:        "169.254.169.254",
		Allowed:     false,
		Reason:      "blocked cloud metadata",
	})

	list := store.List(10)
	if len(list) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(list))
	}
	if !list[0].Allowed || list[1].Allowed {
		t.Errorf("unexpected decision states: %+v", list)
	}
}
