//go:build windows || darwin

package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/internal/multillm"
)

// gatewayRequest issues a request from an explicit address so the local-only
// rule of the gateway key can be exercised.
func gatewayRequest(t *testing.T, method, path, body, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	req.AddCookie(&http.Cookie{Name: "token", Value: "t"})
	rr := httptest.NewRecorder()
	(&Server{Token: "t"}).Handler().ServeHTTP(rr, req)
	return rr
}

func decodeGatewayConnection(t *testing.T, rr *httptest.ResponseRecorder) struct {
	multillm.GatewayInfo
	RestartRequired bool `json:"restart_required"`
} {
	t.Helper()
	var got struct {
		multillm.GatewayInfo
		RestartRequired bool `json:"restart_required"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %q: %v", rr.Body.String(), err)
	}
	return got
}

func TestGatewayConnectionRequiresUIToken(t *testing.T) {
	writeTestProviderConfig(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/connection", nil)
	rr := httptest.NewRecorder()
	(&Server{Token: "t"}).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status without token = %d, want 403", rr.Code)
	}
}

func TestGatewayConnectionHidesTheKeyFromARemoteCaller(t *testing.T) {
	path := setupEmptyProviderConfig(t)
	cfg := `{"gateway_api_key_env":"OLLAMA_DZ23_GATEWAY_KEY","providers":[
	  {"name":"anthropic","type":"anthropic","base_url":"https://api.anthropic.com","api_key_env":"DZ23_TEST_ANTHROPIC_KEY","paths":["/v1/messages"],"models":[{"id":"claude-sonnet"}]}
	]}`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_DZ23_GATEWAY_KEY", "gateway-secret-value")

	rr := gatewayRequest(t, http.MethodGet, "/api/v1/gateway/connection", "", "192.0.2.7:5555")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "gateway-secret-value") {
		t.Fatal("a remote caller received the gateway key")
	}
	got := decodeGatewayConnection(t, rr)
	if got.Key != "" {
		t.Fatalf("key = %q, want empty for a remote caller", got.Key)
	}
	if !got.KeyPresent {
		t.Fatal("the caller must still learn that a key is configured")
	}
	if got.KeyEnv != "OLLAMA_DZ23_GATEWAY_KEY" {
		t.Fatalf("key env = %q", got.KeyEnv)
	}
	if !strings.HasPrefix(got.BaseURL, "http") {
		t.Fatalf("base url = %q", got.BaseURL)
	}
	if !got.Rotation.Enabled || got.Rotation.MaxAttempts != 3 {
		t.Fatalf("rotation = %+v", got.Rotation)
	}
	if len(got.Protocols) != 4 || got.Providers != 1 {
		t.Fatalf("protocols = %d providers = %d", len(got.Protocols), got.Providers)
	}
	if got.Loopback {
		t.Fatal("a configured gateway key means the gateway accepts remote clients")
	}
}

func TestGatewayConnectionRevealsTheKeyToThisComputer(t *testing.T) {
	path := setupEmptyProviderConfig(t)
	cfg := `{"gateway_api_key_env":"OLLAMA_DZ23_GATEWAY_KEY","providers":[]}`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_DZ23_GATEWAY_KEY", "gateway-secret-value")

	for _, addr := range []string{"127.0.0.1:5555", "[::1]:5555"} {
		rr := gatewayRequest(t, http.MethodGet, "/api/v1/gateway/connection", "", addr)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d from %s: %s", rr.Code, addr, rr.Body.String())
		}
		got := decodeGatewayConnection(t, rr)
		if got.Key != "gateway-secret-value" {
			t.Fatalf("key from %s = %q", addr, got.Key)
		}
	}
}

func TestRotateGatewayKeyStoresTheSecretOutsideTheConfig(t *testing.T) {
	path := setupEmptyProviderConfig(t)
	t.Setenv("OLLAMA_DZ23_GATEWAY_KEY", "")
	t.Setenv("OLLAMA_DZ23_GATEWAY_KEY_FILE", "")
	t.Cleanup(func() {
		if dir := multillm.DefaultCredentialDir(); dir != "" {
			_ = secrets.Remove(dir, multillm.DefaultGatewayKeyEnv)
		}
	})

	rr := gatewayRequest(t, http.MethodPost, "/api/v1/gateway/key", "", "127.0.0.1:5555")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeGatewayConnection(t, rr)
	if len(got.Key) < 40 {
		t.Fatalf("generated key looks weak: %q", got.Key)
	}
	if !got.RestartRequired {
		t.Fatal("recording gateway_api_key_env for the first time requires a restart")
	}
	if got.KeyEnv != multillm.DefaultGatewayKeyEnv {
		t.Fatalf("key env = %q", got.KeyEnv)
	}

	// The key must live in the vault, never in the provider config file.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), got.Key) {
		t.Fatal("the generated key was written to the provider config")
	}
	if !strings.Contains(string(raw), `"gateway_api_key_env"`) {
		t.Fatalf("config did not record the gateway env: %s", raw)
	}
	saved, err := multillm.Load(path)
	if err != nil {
		t.Fatalf("the saved config must stay valid: %v", err)
	}
	if saved.GatewayKeyEnv() != multillm.DefaultGatewayKeyEnv {
		t.Fatalf("reloaded env = %q", saved.GatewayKeyEnv())
	}

	// The running server reads the same vault entry on every request.
	if stored := multillm.CredentialValue(multillm.DefaultGatewayKeyEnv); stored != got.Key {
		t.Fatalf("vault holds %q, response held %q", stored, got.Key)
	}
}

func TestRotateGatewayKeyRefusesARemoteCaller(t *testing.T) {
	path := setupEmptyProviderConfig(t)
	rr := gatewayRequest(t, http.MethodPost, "/api/v1/gateway/key", "", "203.0.113.9:4444")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a refused request must not create the provider config")
	}
}
