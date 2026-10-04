package agent

import "testing"

// TestNormalizeRedirectURILoopback covers the desktop loopback redirect rule:
// a loopback callback URI is accepted without an allowlist when loopback
// redirects are enabled, while non-loopback URIs still require the allowlist
// and loopback URIs are rejected when the option is off.
func TestNormalizeRedirectURILoopback(t *testing.T) {
	loopback := OAuthProvider{Name: "google", AllowLoopbackRedirect: true}

	for _, uri := range []string{
		"http://127.0.0.1:53170/api/agent/v1/connectors/gmail/oauth/callback",
		"http://localhost:8080/cb",
		"http://[::1]:9000/cb",
	} {
		got, err := loopback.NormalizeRedirectURI(uri)
		if err != nil {
			t.Fatalf("expected loopback %q to be accepted, got error: %v", uri, err)
		}
		if got == "" {
			t.Fatalf("expected a canonical URI for %q", uri)
		}
	}

	// Non-loopback is rejected without an allowlist even when loopback is
	// allowed.
	if _, err := loopback.NormalizeRedirectURI("https://example.com/cb"); err == nil {
		t.Fatal("expected non-loopback redirect to be rejected without an allowlist")
	}

	// A non-loopback URI that IS on the allowlist still works (regression).
	allowed := OAuthProvider{Name: "google", AllowLoopbackRedirect: true, RedirectURIs: []string{"https://app.example.com/cb"}}
	if _, err := allowed.NormalizeRedirectURI("https://app.example.com/cb"); err != nil {
		t.Fatalf("expected allowlisted https redirect to be accepted: %v", err)
	}

	// With loopback disabled, a loopback http URI is rejected at the syntax gate.
	disabled := OAuthProvider{Name: "google", AllowLoopbackRedirect: false}
	if _, err := disabled.NormalizeRedirectURI("http://127.0.0.1:53170/cb"); err == nil {
		t.Fatal("expected loopback redirect to be rejected when AllowLoopbackRedirect is false")
	}
}
