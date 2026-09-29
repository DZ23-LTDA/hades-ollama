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
