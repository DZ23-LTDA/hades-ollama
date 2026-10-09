//go:build windows || darwin

package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/internal/multillm"
)

func writeTestProviderConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "providers.json")
	cfg := `{"providers":[
	  {"name":"openai","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key_env":"DZ23_TEST_OPENAI_KEY","models":[{"id":"a"},{"id":"b"}]},
	  {"name":"local","type":"openai-compatible","base_url":"http://127.0.0.1:1234/v1","models":[{"id":"c"}]}
	]}`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_DZ23_CONFIG", path)
	t.Setenv("DZ23_TEST_OPENAI_KEY", "")
	t.Setenv("DZ23_TEST_OPENAI_KEY_FILE", "")
}

func providerRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	return providerRequestWithHeaders(t, method, path, body, nil)
}

func providerRequestWithHeaders(t *testing.T, method, path, body string, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.AddCookie(&http.Cookie{Name: "token", Value: "t"})
	rr := httptest.NewRecorder()
	(&Server{Token: "t"}).Handler().ServeHTTP(rr, req)
	return rr
}

func TestListProvidersReportsStatusWithoutSecrets(t *testing.T) {
	writeTestProviderConfig(t)
	t.Setenv("DZ23_TEST_OPENAI_KEY", "sk-should-never-be-returned")

	rr := providerRequest(t, http.MethodGet, "/api/v1/providers", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "sk-should-never-be-returned") {
		t.Fatal("provider listing leaked a credential")
	}
	var got struct {
		Providers []ProviderStatus `json:"providers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Providers) != 2 {
		t.Fatalf("providers = %+v", got.Providers)
	}
	openai, local := got.Providers[0], got.Providers[1]
	if !openai.NeedsKey || !openai.Configured || openai.Models != 2 {
		t.Fatalf("openai = %+v", openai)
	}
	if local.NeedsKey || !local.Configured {
		t.Fatalf("keyless provider = %+v", local)
	}
}

func TestProviderKeyRejectsUnknownProviderAndBadKeys(t *testing.T) {
	writeTestProviderConfig(t)

	if rr := providerRequest(t, http.MethodPut, "/api/v1/providers/nope/key", `{"key":"x"}`); rr.Code != http.StatusNotFound {
		t.Fatalf("unknown provider status = %d", rr.Code)
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/providers/local/key", `{"key":"x"}`); rr.Code != http.StatusNotFound {
		t.Fatalf("keyless provider status = %d", rr.Code)
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/providers/openai/key", `{"key":"two\nlines"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("bad key status = %d", rr.Code)
	}
}

func TestProviderEndpointsRequireUIToken(t *testing.T) {
	writeTestProviderConfig(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	rr := httptest.NewRecorder()
	(&Server{Token: "t"}).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status without token = %d, want 403", rr.Code)
	}
}

// setupEmptyProviderConfig points OLLAMA_DZ23_CONFIG at a path that does not
// exist yet and isolates the credential vault inside a temp directory, so a
// create both exercises file creation and never touches the real vault.
func setupEmptyProviderConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	t.Setenv("OLLAMA_DZ23_CONFIG", path)
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "appdata"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "home", "config"))
	return path
}

func TestCreateProviderLocalWithoutKey(t *testing.T) {
	path := setupEmptyProviderConfig(t)

	body := `{"name":"lmstudio","base_url":"http://127.0.0.1:1234/v1","models":["qwen2.5"," qwen2.5 ",""]}`
	rr := providerRequest(t, http.MethodPost, "/api/v1/providers", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var got ProviderStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "lmstudio" || got.NeedsKey || !got.Configured || got.Models != 1 {
		t.Fatalf("status = %+v", got)
	}
	// The missing file must have been created as a valid config.
	saved, err := multillm.Load(path)
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if _, ok := saved.Model("lmstudio/qwen2.5"); !ok {
		t.Fatal("model was not persisted")
	}
}

func TestCreateProviderRemoteWithKeyDoesNotLeak(t *testing.T) {
	path := setupEmptyProviderConfig(t)
	const secret = "sk-super-secret-value-123"
	t.Cleanup(func() {
		if dir := multillm.DefaultCredentialDir(); dir != "" {
			_ = secrets.Remove(dir, "OPENROUTER_API_KEY")
		}
	})

	body := `{"name":"openrouter","base_url":"https://openrouter.ai/api/v1","models":["x-ai/grok","openai/gpt-4o"],"api_key":"` + secret + `"}`
	rr := providerRequest(t, http.MethodPost, "/api/v1/providers", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), secret) {
		t.Fatal("response leaked the API key")
	}
	var got ProviderStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.NeedsKey || !got.Configured || got.APIKeyEnv != "OPENROUTER_API_KEY" || got.Models != 2 {
		t.Fatalf("status = %+v", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatal("config file leaked the API key")
	}
	if !strings.Contains(string(raw), "OPENROUTER_API_KEY") {
		t.Fatal("config file is missing the api_key_env reference")
	}
	if !secrets.Configured("OPENROUTER_API_KEY") {
		t.Fatal("API key was not stored in the credential vault")
	}
}

func TestCreateProviderRejectsDuplicateAndBadInput(t *testing.T) {
	setupEmptyProviderConfig(t)

	ok := `{"name":"dup","base_url":"https://api.example.com/v1","models":["a"]}`
	if rr := providerRequest(t, http.MethodPost, "/api/v1/providers", ok); rr.Code != http.StatusCreated {
		t.Fatalf("first create status %d: %s", rr.Code, rr.Body.String())
	}
	if rr := providerRequest(t, http.MethodPost, "/api/v1/providers", ok); rr.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d", rr.Code)
	}

	for name, body := range map[string]string{
		"external http":  `{"name":"ext","base_url":"http://api.example.com/v1","models":["a"]}`,
		"userinfo url":   `{"name":"ui","base_url":"https://user:pass@api.example.com/v1","models":["a"]}`,
		"no models":      `{"name":"none","base_url":"https://api.example.com/v1","models":[]}`,
		"bad name":       `{"name":"bad/name","base_url":"https://api.example.com/v1","models":["a"]}`,
		"empty base_url": `{"name":"nourl","base_url":"","models":["a"]}`,
	} {
		if rr := providerRequest(t, http.MethodPost, "/api/v1/providers", body); rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d (%s)", name, rr.Code, rr.Body.String())
		}
	}
}

func TestDeriveProviderEnvName(t *testing.T) {
	for in, want := range map[string]string{
		"openai":        "OPENAI_API_KEY",
		"Open Router":   "OPEN_ROUTER_API_KEY",
		"deepseek-chat": "DEEPSEEK_CHAT_API_KEY",
		"123provider":   "PROVIDER_API_KEY",
		"42":            "PROVEDOR_API_KEY",
	} {
		if got := deriveProviderEnvName(in); got != want {
			t.Errorf("deriveProviderEnvName(%q) = %q, want %q", in, got, want)
		}
		if got := deriveProviderEnvName(in); !providerEnvNamePattern.MatchString(got) {
			t.Errorf("deriveProviderEnvName(%q) = %q does not match the required pattern", in, got)
		}
	}
}

func TestMergeProviderModelsKeepsSettingsAndAddsNew(t *testing.T) {
	current := []multillm.ModelConfig{
		{ID: "keep", Capabilities: []string{"chat", "tools"}, Priority: 7},
		{ID: "drop"},
	}
	got, err := mergeProviderModels(current, []string{" new ", "keep", "keep", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "keep" || got[0].Priority != 7 || got[1].ID != "new" || got[1].Capabilities[0] != "chat" {
		t.Fatalf("merged = %+v", got)
	}
	if _, err := mergeProviderModels(current, nil); err == nil {
		t.Fatal("empty selection must be rejected")
	}
}

func TestProviderModelsEndpoints(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			http.Error(w, "no key", http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"data":[{"id":"fresh-model"},{"id":"a"}]}`))
	}))
	defer upstream.Close()

	path := filepath.Join(t.TempDir(), "providers.json")
	cfg := `{"providers":[{"name":"openai","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key_env":"DZ23_TEST_OPENAI_KEY","models":[{"id":"a","capabilities":["chat","tools"]},{"id":"retired"}]}]}`
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_DZ23_CONFIG", path)
	t.Setenv("DZ23_TEST_OPENAI_KEY", "sk-test")
	t.Setenv("DZ23_TEST_OPENAI_KEY_FILE", "")

	// The listing uses the provider's own base_url; point it at the fake.
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "https://api.openai.com/v1", upstream.URL, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	rr := providerRequest(t, http.MethodGet, "/api/v1/providers/openai/models", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rr.Code, rr.Body.String())
	}
	var listed struct {
		Configured []string `json:"configured"`
		Available  []string `json:"available"`
		Error      string   `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Error != "" || strings.Join(listed.Available, ",") != "a,fresh-model" || strings.Join(listed.Configured, ",") != "a,retired" {
		t.Fatalf("listed = %+v", listed)
	}

	// Saving validates the whole config, so restore an https base_url first.
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	rr = providerRequest(t, http.MethodPut, "/api/v1/providers/openai/models", `{"models":["a","fresh-model"]}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("save status %d: %s", rr.Code, rr.Body.String())
	}
	saved, err := multillm.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := saved.Model("openai/fresh-model"); !ok {
		t.Fatal("fresh-model not saved")
	}
	if _, ok := saved.Model("openai/retired"); ok {
		t.Fatal("retired model should be gone")
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/providers/nope/models", `{"models":["x"]}`); rr.Code != http.StatusNotFound {
		t.Fatalf("unknown provider status = %d", rr.Code)
	}
}
