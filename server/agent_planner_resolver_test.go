package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

func TestMultiProviderPlannerResolverSelectsRegisteredModel(t *testing.T) {
	t.Setenv("TEST_PLANNER_KEY", "configured")
	path := filepath.Join(t.TempDir(), "providers.json")
	config := []byte(`{"providers":[{"name":"anthropic","type":"openai-compatible","base_url":"https://example.com/v1","api_key_env":"TEST_PLANNER_KEY","paths":["/api/chat"],"models":[{"id":"claude-sonnet-4-5","capabilities":["chat"]}]}]}`)
	if err := os.WriteFile(path, config, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := multillm.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	planner, err := (multiProviderPlannerResolver{registry: registry}).ResolvePlanner("anthropic", "claude-sonnet-4-5")
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := planner.(agent.OllamaPlanner)
	if !ok {
		t.Fatalf("planner type = %T, want agent.OllamaPlanner", planner)
	}
	if selected.Model != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("planner model = %q", selected.Model)
	}
}

func TestMultiProviderPlannerResolverRejectsMissingModel(t *testing.T) {
	if _, err := (multiProviderPlannerResolver{}).ResolvePlanner("anthropic", ""); err == nil {
		t.Fatal("missing model must fail closed")
	}
}

func TestRoutePreferenceFromEnv(t *testing.T) {
	cases := map[string]multillm.RoutePreference{
		"":             multillm.RoutePreferenceAuto,
		"desconhecido": multillm.RoutePreferenceAuto,
		"fastest":      multillm.RoutePreferenceFastest,
		"rápido":       multillm.RoutePreferenceFastest,
		"velocidade":   multillm.RoutePreferenceFastest,
		"quality":      multillm.RoutePreferenceBestQuality,
		"melhor":       multillm.RoutePreferenceBestQuality,
		"cheapest":     multillm.RoutePreferenceCheapest,
		"barato":       multillm.RoutePreferenceCheapest,
		" CUSTO ":      multillm.RoutePreferenceCheapest,
	}
	for in, want := range cases {
		t.Setenv("OLLAMA_AGENT_ROUTE_PREFERENCE", in)
		if got := routePreferenceFromEnv(); got != want {
			t.Fatalf("routePreferenceFromEnv(%q) = %q, want %q", in, got, want)
		}
	}
}
