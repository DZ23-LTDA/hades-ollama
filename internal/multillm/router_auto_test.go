package multillm

import (
	"fmt"
	"path/filepath"
	"testing"
)

func testRouteRegistry(t *testing.T) *Registry {
	t.Helper()
	// Absolute executable path valid on every OS (ToSlash keeps it JSON-safe on
	// Windows, where filepath.IsAbs rejects POSIX-style paths like /usr/bin/true).
	exe := filepath.ToSlash(filepath.Join(t.TempDir(), "subscription-cli"))
	registry, err := LoadBytes([]byte(fmt.Sprintf(`{"providers":[{"name":"local","type":"openai-compatible","base_url":"https://local.example","priority":1,"models":[{"id":"coder","capabilities":["coding"],"cost_tag":"0-local","quality_score":1}]},{"name":"subscription","type":"cli","executable":%q,"allow_execution":true,"priority":2,"models":[{"id":"coder","capabilities":["coding"],"cost_tag":"0-assinatura","quality_score":2}]},{"name":"paid","type":"openai-compatible","base_url":"https://paid.example","priority":100,"models":[{"id":"coder","capabilities":["coding"],"cost_per_1k_input_cents":10,"cost_per_1k_output_cents":10,"quality_score":10}]}]}`, exe)))
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestAutoRoutePrefersFreeOverPaid(t *testing.T) {
	registry := testRouteRegistry(t)
	decision, err := registry.Route(RouteRequest{RequiredCapabilities: []string{"coding"}, Path: "/api/chat", SelectableModels: registry.Models()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Model.Provider == "paid" || decision.Model.CostTag == "" {
		t.Fatalf("expected free candidate, got %+v", decision)
	}
	if decision.Model.CostTag != "0-assinatura" && decision.Model.CostTag != "0-local" {
		t.Fatalf("expected zero-cost tag, got %q", decision.Model.CostTag)
	}
}

func TestAutoRouteRespectsManualOverride(t *testing.T) {
	registry := testRouteRegistry(t)
	decision, err := registry.Route(RouteRequest{RequiredCapabilities: []string{"coding"}, Path: "/api/chat", PreferredProvider: "paid", SelectableModels: registry.Models()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Model.Provider != "paid" {
		t.Fatalf("manual provider override was ignored: %+v", decision)
	}
}

func TestAutoRouteOnlyPassCandidates(t *testing.T) {
	registry := testRouteRegistry(t)
	all := registry.Models()
	pass := make([]Model, 0, len(all))
	for _, model := range all {
		if model.Provider != "paid" {
			pass = append(pass, model)
		}
	}
	decision, err := registry.Route(RouteRequest{RequiredCapabilities: []string{"coding"}, Path: "/api/chat", SelectableModels: pass})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Model.Provider == "paid" {
		t.Fatalf("unavailable/non-PASS candidate was selected: %+v", decision)
	}
}
