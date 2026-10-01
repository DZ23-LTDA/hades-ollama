package multillm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelListExcludesUnreachable(t *testing.T) {
	// A server that returns 500 error simulating unreachable / malfunctioning upstream
	downServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer downServer.Close()

	t.Setenv("TEST_DOWN_KEY", "secret-key")

	reg := &Registry{
		providers: map[string]Provider{
			"down-provider": {
				Name:                  "down-provider",
				Type:                  ProviderTypeOpenAICompatible,
				BaseURL:               downServer.URL,
				APIKeyEnv:             "TEST_DOWN_KEY",
				AuthStyle:             AuthStyleBearer,
				AllowInsecureLoopback: true,
			},
		},
		models: map[string]Model{
			"down-provider/llama-model": {
				ID:         "down-provider/llama-model",
				UpstreamID: "llama-model",
				Provider:   "down-provider",
			},
		},
	}

	ctx := context.Background()
	probe := reg.ProbeModel(ctx, reg.models["down-provider/llama-model"], downServer.Client())

	if probe.Selectable {
		t.Fatalf("expected unreachable model to be unselectable, got selectable: %+v", probe)
	}
	if probe.Status != ModelStatusFail {
		t.Fatalf("expected status FAIL, got: %s", probe.Status)
	}

	clean := reg.CleanSelectableModels(ctx, downServer.Client())
	if len(clean) != 0 {
		t.Fatalf("expected clean selectable models to exclude unreachable model, got %d models", len(clean))
	}
}

func TestModelListOnlyPassIsSelectable(t *testing.T) {
	// 1. Upstream that responds with valid models list
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-model-good"}]}`))
	}))
	defer goodServer.Close()

	t.Setenv("TEST_GOOD_KEY", "sk-good-key")

	reg := &Registry{
		providers: map[string]Provider{
			"good-provider": {
				Name:                  "good-provider",
				Type:                  ProviderTypeOpenAICompatible,
				BaseURL:               goodServer.URL,
				APIKeyEnv:             "TEST_GOOD_KEY",
				AuthStyle:             AuthStyleBearer,
				AllowInsecureLoopback: true,
			},
			"unconfigured-provider": {
				Name:      "unconfigured-provider",
				Type:      ProviderTypeOpenAICompatible,
				BaseURL:   "https://api.example.com",
				APIKeyEnv: "TEST_MISSING_KEY_XYZ",
			},
		},
		models: map[string]Model{
			"good-provider/gpt-model-good": {
				ID:         "good-provider/gpt-model-good",
				UpstreamID: "gpt-model-good",
				Provider:   "good-provider",
			},
			"unconfigured-provider/unconfigured-model": {
				ID:         "unconfigured-provider/unconfigured-model",
				UpstreamID: "unconfigured-model",
				Provider:   "unconfigured-provider",
			},
		},
	}

	ctx := context.Background()

	// Probe good provider -> PASS
	probeGood := reg.ProbeModel(ctx, reg.models["good-provider/gpt-model-good"], goodServer.Client())
	if probeGood.Status != ModelStatusPass || !probeGood.Selectable {
		t.Fatalf("expected good provider model to have status PASS and be selectable, got: %+v", probeGood)
	}

	// Probe unconfigured provider -> NOT_CONFIGURED
	probeUnconfigured := reg.ProbeModel(ctx, reg.models["unconfigured-provider/unconfigured-model"], nil)
	if probeUnconfigured.Status != ModelStatusNotConfigured || probeUnconfigured.Selectable {
		t.Fatalf("expected unconfigured model to have status NOT_CONFIGURED and be unselectable, got: %+v", probeUnconfigured)
	}

	// Clean selectable models must only contain the good model
	clean := reg.CleanSelectableModels(ctx, goodServer.Client())
	if len(clean) != 1 || clean[0].ID != "good-provider/gpt-model-good" {
		t.Fatalf("expected exactly 1 clean model (good-provider/gpt-model-good), got: %+v", clean)
	}

	// Local models verification
	localProbe := ProbeLocal("llama3:8b", []string{"llama3:8b", "mistral:latest"})
	if localProbe.Status != ModelStatusPass || !localProbe.Selectable {
		t.Fatalf("expected downloaded local model to be PASS, got: %+v", localProbe)
	}

	localMissingProbe := ProbeLocal("non-existent-local", []string{"llama3:8b"})
	if localMissingProbe.Status != ModelStatusNotPresent || localMissingProbe.Selectable {
		t.Fatalf("expected non-downloaded local model to be NOT_PRESENT, got: %+v", localMissingProbe)
	}
}
