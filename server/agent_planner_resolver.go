package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

type multiProviderPlannerResolver struct {
	registry *multillm.Registry
	client   *api.Client
}

func (r multiProviderPlannerResolver) ResolvePlanner(provider, model string) (agent.Planner, error) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	if provider == "" {
		return nil, errors.New("planner provider is required")
	}
	if provider == "ollama-local" {
		if model == "" {
			return nil, errors.New("local provider requires an explicit model")
		}
		client := r.client
		if client == nil {
			client = api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.planner.local", true))
		}
		return agent.OllamaPlanner{Client: client, Model: model}, nil
	}
	if r.registry == nil {
		return nil, fmt.Errorf("provider %q is not configured", provider)
	}
	if model == "" {
		return nil, fmt.Errorf("provider %q requires an explicit model", provider)
	}
	modelID := model
	if !strings.Contains(modelID, "/") {
		modelID = provider + "/" + modelID
	}
	resolved, ok := r.registry.Resolve(modelID, multillm.Policy{Path: "/api/chat"})
	if !ok || resolved.Provider != provider {
		return nil, fmt.Errorf("provider model %q is unavailable or does not support /api/chat", modelID)
	}
	client := r.client
	if client == nil {
		client = api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.planner.local", true))
	}
	return agent.OllamaPlanner{Client: client, Model: resolved.ID}, nil
}

func (r multiProviderPlannerResolver) ResolvePlannerForMission(ctx context.Context, provider, model string, capabilities []string) (agent.Planner, agent.PlannerResolution, error) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	if model != "" && !strings.HasPrefix(model, "auto/") && model != "auto" {
		planner, err := r.ResolvePlanner(provider, model)
		return planner, agent.PlannerResolution{Provider: provider, Model: model, Reason: "override manual respeitado"}, err
	}

	required := append([]string(nil), capabilities...)
	switch model {
	case "auto/coding":
		required = []string{"coding"}
	case "auto/reasoning":
		required = []string{"reasoning"}
	case "auto/vision":
		required = []string{"vision"}
	default:
		filtered := required[:0]
		for _, capability := range required {
			if !strings.HasPrefix(strings.TrimSpace(capability), "workspace:") {
				filtered = append(filtered, capability)
			}
		}
		required = filtered
	}
	if r.registry != nil {
		selectable := r.registry.CleanSelectableModels(ctx, newServerEgressClient("server.planner.discovery", false))
		decision, err := r.registry.Route(multillm.RouteRequest{RequiredCapabilities: required, Path: "/api/chat", SelectableModels: selectable, Preference: routePreferenceFromEnv()})
		if err == nil {
			planner, resolveErr := r.ResolvePlanner(decision.Model.Provider, decision.Model.ID)
			if resolveErr == nil {
				return planner, agent.PlannerResolution{Provider: decision.Model.Provider, Model: decision.Model.ID, Reason: decision.Reason, CostTag: decision.Model.CostTag}, nil
			}
		}
	}

	localModel := strings.TrimSpace(os.Getenv("OLLAMA_DZ23_LOCAL_MODEL"))
	if localModel != "" {
		client := r.client
		if client == nil {
			client = api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.planner.local", true))
		}
		return agent.OllamaPlanner{Client: client, Model: localModel}, agent.PlannerResolution{Provider: "ollama-local", Model: localModel, Reason: "fallback local-first: nenhuma rota remota PASS elegível", CostTag: "0-local"}, nil
	}
	return agent.RulePlanner{}, agent.PlannerResolution{Provider: "ollama-local", Reason: "fallback local-first: planner de regras local sem modelo remoto elegível", CostTag: "0-local"}, nil
}

var _ agent.PlannerResolver = multiProviderPlannerResolver{}

// routePreferenceFromEnv maps OLLAMA_AGENT_ROUTE_PREFERENCE (with pt-BR aliases)
// to a router preference so the user can bias model selection toward speed,
// quality or cost without code changes (G9). An unset/unknown value keeps the
// balanced default.
func routePreferenceFromEnv() multillm.RoutePreference {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_ROUTE_PREFERENCE"))) {
	case "fastest", "rapido", "rápido", "speed", "velocidade":
		return multillm.RoutePreferenceFastest
	case "quality", "qualidade", "best", "melhor":
		return multillm.RoutePreferenceBestQuality
	case "cheapest", "barato", "cost", "custo":
		return multillm.RoutePreferenceCheapest
	default:
		return multillm.RoutePreferenceAuto
	}
}
