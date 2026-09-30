package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/multillm"
)

func TestModelListProbeFiltersListHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test server simulating failing remote provider
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer failingServer.Close()

	os.Setenv("TEST_FAILING_KEY", "bad-key")
	defer os.Unsetenv("TEST_FAILING_KEY")

	reg, err := multillm.LoadBytes([]byte(`{
		"providers": [
			{
				"name": "remote-bad",
				"type": "openai-compatible",
				"base_url": "` + failingServer.URL + `",
				"api_key_env": "TEST_FAILING_KEY",
				"allow_insecure_loopback": true,
				"allow_private": true,
				"models": [
					{"id": "bad-model"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	s := &Server{
		multiRegistry: reg,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/tags", nil)

	s.ListHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", w.Code)
	}

	var resp api.ListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Model from remote-bad must have unavailable status/family tag
	found := false
	for _, m := range resp.Models {
		if m.Name == "remote-bad/bad-model" {
			found = true
			if m.Details.Family == "remote-bad" {
				t.Fatalf("expected remote-bad/bad-model to have unavailable tag, got: %s", m.Details.Family)
			}
		}
	}
	if !found {
		t.Fatalf("expected remote-bad/bad-model in tags response")
	}
}

func TestModelListExcludesUnreachable(t *testing.T) {
	// Verifies that multiRegistry Model probe reports failure on unreachable hosts
	reg, err := multillm.LoadBytes([]byte(`{
		"providers": [
			{
				"name": "unreachable",
				"type": "openai-compatible",
				"base_url": "https://127.0.0.1:59999",
				"api_key_env": "OLLAMA_NON_EXISTENT_KEY",
				"allow_private": true,
				"models": [
					{"id": "unreachable-model"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	m, ok := reg.Model("unreachable/unreachable-model")
	if !ok {
		t.Fatalf("expected model to exist in registry")
	}

	probe := reg.ProbeModel(context.Background(), m, nil)
	if probe.Selectable {
		t.Fatalf("unreachable model must not be selectable: %+v", probe)
	}
}

func TestModelListOnlyPassIsSelectable(t *testing.T) {
	reg, err := multillm.LoadBytes([]byte(`{
		"providers": [
			{
				"name": "unconfigured",
				"type": "openai-compatible",
				"base_url": "https://api.example.com",
				"api_key_env": "MISSING_ENV_VAR_ABC",
				"models": [
					{"id": "model-abc"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	m, ok := reg.Model("unconfigured/model-abc")
	if !ok {
		t.Fatalf("expected model in registry")
	}

	probe := reg.ProbeModel(context.Background(), m, nil)
	if probe.Status != multillm.ModelStatusNotConfigured {
		t.Fatalf("expected NOT_CONFIGURED, got: %s", probe.Status)
	}
	if probe.Selectable {
		t.Fatalf("unconfigured model cannot be selectable")
	}
}
