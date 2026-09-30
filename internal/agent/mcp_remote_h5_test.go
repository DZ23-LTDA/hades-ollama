package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteMCPOAuthRefreshAndSessionResumption(t *testing.T) {
	var refreshes int
	var sawAuth, sawSession bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			refreshes++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"fresh-token","refresh_token":"refresh-2","expires_in":300}`))
		case "/mcp":
			sawAuth = r.Header.Get("Authorization") == "Bearer fresh-token"
			sawSession = r.Header.Get("Mcp-Session-Id") == "session-1"
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Mcp-Session-Id", "session-1")
			var request struct {
				ID int64 `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"ok": true}})
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(h)
	defer server.Close()
	t.Setenv("MCP_CLIENT_ID", "client-1")
	manager := NewRemoteMCPManager()
	config := RemoteMCPServerConfig{ID: "oauth", URL: server.URL + "/mcp", AllowedMethods: []string{"ping"}, OAuth: &RemoteMCPOAuthConfig{AuthorizationURL: server.URL + "/authorize", TokenURL: server.URL + "/token", ClientIDEnv: "MCP_CLIENT_ID", RedirectURI: "http://127.0.0.1/callback"}}
	if err := manager.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginOAuth("oauth", "org-a"); err != nil {
		t.Fatal(err)
	}
	// A valid authorization-code exchange stores the access/refresh pair in memory only.
	// The local test endpoint is intentionally used; production endpoints must be HTTPS.
	// State is discovered from the manager only inside this package test.
	manager.mu.RLock()
	var state string
	for key := range manager.oauthStates {
		state = key
	}
	manager.mu.RUnlock()
	if err := manager.CompleteOAuth(context.Background(), state, "auth-code"); err != nil {
		t.Fatal(err)
	}
	manager.setOAuthTokenForTest("oauth", remoteMCPOAuthToken{AccessToken: "expired", RefreshToken: "refresh-1", ExpiresAt: time.Now().Add(-time.Minute)})
	result, err := manager.CallForOrganization(context.Background(), "org-a", "oauth", "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"ok":true}` {
		t.Fatalf("result=%s", result)
	}
	if refreshes == 0 || !sawAuth {
		t.Fatalf("OAuth refresh/auth not observed: refreshes=%d auth=%v", refreshes, sawAuth)
	}
	_, err = manager.CallForOrganization(context.Background(), "org-a", "oauth", "ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !sawSession {
		t.Fatal("second call did not resume MCP session")
	}
}

func TestRemoteMCPPairingIsAuthenticatedOneShotAndBoundToChallenge(t *testing.T) {
	t.Setenv("MCP_PAIRING_SECRET", "pair-secret")
	manager := NewRemoteMCPManager()
	if err := manager.RegisterForOrganization("org-a", RemoteMCPServerConfig{ID: "pair", URL: "https://mcp.example.test/mcp", AllowedMethods: []string{"ping"}, PairingTokenEnv: "MCP_PAIRING_SECRET"}); err != nil {
		t.Fatal(err)
	}
	challenge, err := manager.BeginPairing("pair", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.CompletePairing("pair", "org-a", "pair-secret", "wrong"); err != ErrRemoteMCPPairingUnauthorized {
		t.Fatalf("wrong challenge err=%v", err)
	}
	challenge, err = manager.BeginPairing("pair", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.CompletePairing("pair", "org-a", "wrong-secret", challenge); err != ErrRemoteMCPPairingUnauthorized {
		t.Fatalf("wrong secret err=%v", err)
	}
	challenge, err = manager.BeginPairing("pair", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.CompletePairing("pair", "org-a", "pair-secret", challenge); err != nil {
		t.Fatal(err)
	}
	if err := manager.CompletePairing("pair", "org-a", "pair-secret", challenge); err != ErrRemoteMCPPairingUnauthorized {
		t.Fatalf("pairing was reusable: %v", err)
	}
}

func TestRemoteMCPNotConfiguredStatusAndExpiredSession(t *testing.T) {
	manager := NewRemoteMCPManager()
	if err := manager.Register(RemoteMCPServerConfig{ID: "unconfigured", URL: "https://mcp.example.test/mcp", AllowedMethods: []string{"ping"}, TokenEnv: "MISSING_MCP_TOKEN"}); err != nil {
		t.Fatal(err)
	}
	if got := manager.Status("unconfigured"); got != GateStatusNotConfigured {
		t.Fatalf("status=%s", got)
	}
	manager.rememberSession("unconfigured", "", "expired", time.Now().Add(-time.Second))
	if _, err := manager.sessionFor("unconfigured", "", "expired"); err != ErrRemoteMCPSessionInvalid {
		t.Fatalf("session err=%v", err)
	}
	if manager.Status("unconfigured") == GateStatusPass {
		t.Fatal("unconfigured server reported PASS")
	}
}
