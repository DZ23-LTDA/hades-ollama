package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/multillm"
)

func TestCliSubscriptionCatalogList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg, _ := multillm.Load("")
	s := &Server{multiRegistry: reg}
	router := gin.New()
	router.GET("/api/tags", s.ListHandler)

	req, err := http.NewRequest(http.MethodGet, "/api/tags", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.ListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	var cliModels []api.ListModelResponse
	for _, m := range resp.Models {
		if m.Details.Format == "cli_subscription" {
			cliModels = append(cliModels, m)
		}
	}

	if len(cliModels) == 0 {
		t.Fatalf("expected at least one model with format 'cli_subscription' in /api/tags")
	}

	// Verify known providers are present
	providersFound := make(map[string]bool)
	for _, m := range cliModels {
		if !strings.HasPrefix(m.Digest, "cli_subscription:") {
			t.Fatalf("expected digest to start with 'cli_subscription:', got %q", m.Digest)
		}
		provider := strings.TrimPrefix(m.Digest, "cli_subscription:")
		providersFound[provider] = true
	}

	expectedProviders := []string{"claude_code", "codex", "gemini", "copilot"}
	for _, exp := range expectedProviders {
		if !providersFound[exp] {
			t.Fatalf("expected provider %q in /api/tags cli_subscription models", exp)
		}
	}
}
