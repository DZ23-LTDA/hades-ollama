package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

// plannerEgressClient builds the HTTP client used to call the planner model.
// Planning legitimately takes much longer than an ordinary API call: a local
// model may be cold-loading into VRAM on the first mission, and a cloud model
// may stream a longer plan. The shared 30s egress timeout was aborting missions
// with "context deadline exceeded (Client.Timeout exceeded while awaiting
// headers)" on first run, so the planner gets a generous timeout (default
// 3 minutes, override with OLLAMA_AGENT_PLANNER_TIMEOUT_MS).
func plannerEgressClient() *http.Client {
	timeout := 180 * time.Second
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_PLANNER_TIMEOUT_MS")); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}
	return agent.NewSafeEgressHTTPClient(agent.EgressOptions{
		Callsite:      "server.planner.local",
		AllowLoopback: true,
		Timeout:       timeout,
		MaxBodyBytes:  20 << 20,
	})
}

type multiProviderPlannerResolver struct {
	registry *multillm.Registry
	client   *api.Client
	spend    *multillm.SpendLedger
}

// Nominal per-routing token estimate used to pre-authorize a paid model against
// the organization's spend cap. It is a conservative hold, not exact per-token
// accounting: free/local models always cost zero and are never blocked.
const (
	spendEstimateInputTokens  = 1000
	spendEstimateOutputTokens = 1000
)

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
			client = api.NewClient(envconfig.ConnectableHost(), plannerEgressClient())
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
		client = api.NewClient(envconfig.ConnectableHost(), plannerEgressClient())
	}
	return agent.OllamaPlanner{Client: client, Model: resolved.ID}, nil
}

func (r multiProviderPlannerResolver) ResolvePlannerForMission(ctx context.Context, provider, model, organizationID string, capabilities []string) (agent.Planner, agent.PlannerResolution, error) {
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
	spendCapReached := false
	if r.registry != nil {
		selectable := r.registry.CleanSelectableModels(ctx, newServerEgressClient("server.planner.discovery", false))
		decision, routeErr := r.registry.Route(multillm.RouteRequest{RequiredCapabilities: required, Path: "/api/chat", SelectableModels: selectable, Preference: routePreferenceFromEnv()})
		if routeErr == nil {
			if r.spendAllows(organizationID, decision.Model) {
				planner, resolveErr := r.ResolvePlanner(decision.Model.Provider, decision.Model.ID)
				if resolveErr == nil {
					return planner, agent.PlannerResolution{Provider: decision.Model.Provider, Model: decision.Model.ID, Reason: decision.Reason, CostTag: decision.Model.CostTag}, nil
				}
			} else {
				// The org is at/over its spend cap: do not route to the paid
				// model; fall back to a local/free planner with an honest reason.
				spendCapReached = true
			}
		}
	}

	capNote := ""
	if spendCapReached {
		capNote = " (limite de gasto da organização atingido; usando modelo local gratuito)"
	}
	localModel := strings.TrimSpace(os.Getenv("OLLAMA_DZ23_LOCAL_MODEL"))
	if localModel != "" {
		client := r.client
		if client == nil {
			client = api.NewClient(envconfig.ConnectableHost(), plannerEgressClient())
		}
		return agent.OllamaPlanner{Client: client, Model: localModel}, agent.PlannerResolution{Provider: "ollama-local", Model: localModel, Reason: "fallback local-first: nenhuma rota remota PASS elegível" + capNote, CostTag: "0-local"}, nil
	}
	return agent.RulePlanner{}, agent.PlannerResolution{Provider: "ollama-local", Reason: "fallback local-first: planner de regras local sem modelo remoto elegível" + capNote, CostTag: "0-local"}, nil
}

// spendAllows enforces the per-organization spend cap for a paid model. Free or
// local models (zero estimated cost) always pass. With a ledger configured, it
// pre-authorizes a nominal per-routing estimate and, when within the cap,
// records it so repeated routings accumulate toward the daily/monthly limit;
// when it would exceed the cap it returns false so the caller falls back to a
// local/free planner. This is a conservative hold, not exact token accounting.
func (r multiProviderPlannerResolver) spendAllows(organizationID string, model multillm.Model) bool {
	if r.spend == nil {
		return true
	}
	estimate := model.CostCents(spendEstimateInputTokens, spendEstimateOutputTokens)
	if estimate <= 0 {
		return true
	}
	now := time.Now().UTC()
	if err := r.spend.Authorize(organizationID, estimate, now); err != nil {
		return false
	}
	_ = r.spend.Record(organizationID, model.ID, estimate, now)
	return true
}

// newAgentSpendLedger builds a spend ledger from environment caps. With no caps
// configured it returns nil so routing behaves exactly as before.
func newAgentSpendLedger() *multillm.SpendLedger {
	daily := spendCapFromEnv("OLLAMA_AGENT_SPEND_DAILY_CAP_CENTS")
	monthly := spendCapFromEnv("OLLAMA_AGENT_SPEND_MONTHLY_CAP_CENTS")
	if daily <= 0 && monthly <= 0 {
		return nil
	}
	ledger, err := multillm.NewSpendLedger(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SPEND_LEDGER_PATH")), daily, monthly)
	if err != nil {
		return nil
	}
	return ledger
}

func spendCapFromEnv(name string) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
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
