package agent

import (
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

func TestSwarmRoleRoutesByCapability(t *testing.T) {
	registry, err := multillm.LoadBytes([]byte(`{"providers":[{"name":"coding","type":"openai-compatible","base_url":"https://coding.example","models":[{"id":"model","capabilities":["coding"],"cost_tag":"0-local"}]},{"name":"reasoning","type":"openai-compatible","base_url":"https://reasoning.example","models":[{"id":"model","capabilities":["reasoning"],"cost_per_1k_input_cents":2,"cost_per_1k_output_cents":2}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		role     AgentRole
		provider string
	}{
		{RoleProgram, "coding"},
		{RoleReview, "reasoning"},
	} {
		decision, routeErr := registry.Route(multillm.RouteRequest{RequiredCapabilities: CapabilitiesForRole(fixture.role), Path: "/api/chat", SelectableModels: registry.Models()})
		if routeErr != nil {
			t.Fatalf("role %s: %v", fixture.role, routeErr)
		}
		if decision.Model.Provider != fixture.provider {
			t.Fatalf("role %s routed to %s, want %s", fixture.role, decision.Model.Provider, fixture.provider)
		}
	}
}
