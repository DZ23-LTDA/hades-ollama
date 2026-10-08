package multillm

import (
	"errors"
	"sort"
	"strings"
)

type ProviderHealth struct {
	Healthy   bool
	LatencyMS int64
	ErrorRate float64
}

// RoutePreference lets the user bias model selection by what they care about
// most. The empty value keeps the balanced default; the others reweight the
// score so the picker can offer "mais rápido / melhor qualidade / mais barato"
// (G9) without changing the baseline behavior.
type RoutePreference string

const (
	RoutePreferenceAuto        RoutePreference = ""
	RoutePreferenceFastest     RoutePreference = "fastest"
	RoutePreferenceBestQuality RoutePreference = "quality"
	RoutePreferenceCheapest    RoutePreference = "cheapest"
)

type RouteRequest struct {
	RequiredCapabilities []string
	Path                 string
	SelectableModels     []Model
	PreferredProvider    string
	MaxLatencyMS         int64
	MaxCostCents         int64
	LocalOnly            bool
	Preference           RoutePreference
	Health               map[string]ProviderHealth
}

type RouteDecision struct {
	Model  Model  `json:"model"`
	Score  int64  `json:"score"`
	Reason string `json:"reason"`
}

var ErrNoRoute = errors.New("no provider route satisfies the requested constraints")

func (r *Registry) Route(request RouteRequest) (RouteDecision, error) {
	candidates := make([]RouteDecision, 0, len(r.models))
	models := r.Models()
	if request.SelectableModels != nil {
		models = append([]Model(nil), request.SelectableModels...)
	}
	for _, model := range models {
		model.Available = r.modelAvailable(model)
		if request.SelectableModels != nil {
			model.Available = true
		}
		if !model.Available || !supports(model, request.RequiredCapabilities) || !r.supportsPath(model, request.Path) {
			continue
		}
		provider, ok := r.providers[model.Provider]
		if !ok || (request.PreferredProvider != "" && provider.Name != request.PreferredProvider) {
			continue
		}
		if request.LocalOnly && !provider.AllowPrivate {
			continue
		}
		cost := model.CostPer1KInputCents + model.CostPer1KOutputCents
		if request.MaxCostCents > 0 && cost > request.MaxCostCents {
			continue
		}
		health := request.Health[provider.Name]
		if request.Health != nil && !health.Healthy {
			continue
		}
		score := int64(model.Priority * 100)
		score += int64(model.QualityScore * 10)
		score -= cost
		if modelIsFree(model) {
			score += 100000
		}
		if health.LatencyMS > 0 {
			if request.MaxLatencyMS > 0 && health.LatencyMS > request.MaxLatencyMS {
				continue
			}
			score -= health.LatencyMS
		}
		if health.ErrorRate > 0.0 {
			score -= int64(health.ErrorRate * 1000)
		}
		if request.PreferredProvider != "" {
			score += 10000
		}
		// Optional user preference reweights the balanced default. The empty
		// (Auto) preference leaves the score unchanged for backward compatibility.
		switch request.Preference {
		case RoutePreferenceBestQuality:
			score += int64(model.QualityScore) * 100
		case RoutePreferenceCheapest:
			score -= cost * 100
		case RoutePreferenceFastest:
			if health.LatencyMS > 0 {
				score -= health.LatencyMS * 20
			}
		}
		candidates = append(candidates, RouteDecision{Model: model, Score: score, Reason: routeReason(model, health, request)})
	}
	if len(candidates) == 0 {
		return RouteDecision{}, ErrNoRoute
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Model.ID < candidates[j].Model.ID
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates[0], nil
}

func routeReason(model Model, health ProviderHealth, request RouteRequest) string {
	parts := []string{"capabilities matched"}
	if modelIsFree(model) {
		parts = append(parts, "free-first: custo zero priorizado")
	} else {
		parts = append(parts, "fallback pago: nenhuma fonte gratuita elegível com pontuação superior")
	}
	if request.PreferredProvider != "" {
		parts = append(parts, "preferred provider")
	}
	if health.LatencyMS > 0 {
		parts = append(parts, "health latency considered")
	}
	if len(model.Capabilities) > 0 {
		parts = append(parts, "model catalog metadata")
	}
	return strings.Join(parts, "; ")
}

func modelIsFree(model Model) bool {
	if strings.TrimSpace(model.CostTag) == "0-assinatura" || strings.TrimSpace(model.CostTag) == "0-local" {
		return true
	}
	return model.CostPer1KInputCents+model.CostPer1KOutputCents == 0
}
