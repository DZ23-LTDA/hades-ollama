package server

import (
	"net/url"
	"strings"
	"testing"
)

// TestBuiltinOAuthProvidersGenerateValidAuthorizeURL locks in the connector
// OAuth capability: every built-in provider must resolve to usable public
// endpoints and build a valid https authorization URL carrying the client_id,
// so "Conectar com <serviço>" never silently produces a broken login link.
func TestBuiltinOAuthProvidersGenerateValidAuthorizeURL(t *testing.T) {
	for name := range builtinOAuthProviders {
		t.Run(name, func(t *testing.T) {
			provider := oauthProviderFromEnv(name)

			// The operator supplies the client_id/secret; simulate that here.
			t.Setenv(provider.ClientIDEnv, "test-client-id")
			t.Setenv(provider.SecretEnv, "test-secret")

			if provider.AuthorizeURL == "" || provider.TokenURL == "" {
				t.Fatalf("provider %q must have built-in authorize and token URLs", name)
			}
			if !provider.AllowLoopbackRedirect {
				t.Fatalf("provider %q must allow loopback redirect for desktop connect", name)
			}

			authURL, err := provider.AuthorizationURLWithNonce("state-123", "nonce-123", builtinOAuthScopes(name))
			if err != nil {
				t.Fatalf("provider %q authorize URL error: %v", name, err)
			}
			u, parseErr := url.Parse(authURL)
			if parseErr != nil {
				t.Fatalf("provider %q produced an unparsable URL: %v", name, parseErr)
			}
			if u.Scheme != "https" {
				t.Errorf("provider %q authorize URL must be https, got %q", name, u.Scheme)
			}
			if u.Query().Get("client_id") != "test-client-id" {
				t.Errorf("provider %q authorize URL missing client_id", name)
			}
			if !strings.Contains(authURL, "response_type=code") {
				t.Errorf("provider %q authorize URL missing response_type=code", name)
			}
		})
	}
}

// TestConnectorOAuthProviderMapping locks in the connector→provider routing so
// the Google/Microsoft/Meta family share their single OAuth app.
func TestConnectorOAuthProviderMapping(t *testing.T) {
	cases := map[string]string{
		"gmail":          "google",
		"google-drive":   "google",
		"google-cloud":   "google",
		"youtube":        "google",
		"azure":          "microsoft",
		"outlook":        "microsoft",
		"instagram":      "facebook",
		"facebook-pages": "facebook",
		"meta-ads":       "facebook",
		"x-twitter":      "twitter",
		"linkedin":       "linkedin",
		"zoom":           "zoom",
	}
	for connector, want := range cases {
		if got := connectorOAuthProvider(connector); got != want {
			t.Errorf("connectorOAuthProvider(%q) = %q, want %q", connector, got, want)
		}
	}
}
