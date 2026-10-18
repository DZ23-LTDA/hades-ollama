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
	// 🚨 E-428 (Estágio 43 do complemento, estratégia `MANUAL`): **PIN de MODELO por tarefa**.
	//
	// *Isto completa o que já existia — e a correção do meu próprio diagnóstico importa:* `PreferredProvider`
	// (`:32`) **já é restrição RÍGIDA por PROVEDOR** (`:67` faz `continue` em quem não bate), *não* um bônus
	// solto. **O que faltava era fixar o MODELO**: *"quero ESTE modelo nesta tarefa"*, e não "quero qualquer um
	// deste provedor".
	//
	// ***Uma tarefa que fixa um modelo e recebe outro silenciosamente é a mesma traição de confiar numa aprovação
	// que não foi dada — só muda o objeto.***
	//
	// Um pin que **não pode ser honrado** (modelo indisponível, capability ausente, teto de custo, `LocalOnly`
	// incompatível) devolve `ErrNoRoute`: *falhar é a resposta honesta; escolher outro por baixo é que não é.*
	PinnedModelID string
}

type RouteDecision struct {
	Model  Model  `json:"model"`
	Score  int64  `json:"score"`
	Reason string `json:"reason"`
	// Factors sao os componentes que SOMAM o Score. Existem porque o adendo proibe uma
	// pontuacao opaca como unica justificativa: *um score sem factors obriga quem o le' a
	// confiar em quem o escreveu -- e confianca nao e' um formato de evidencia.*
	//
	// Invariante testada: a soma dos factors de um candidato e' igual ao seu Score.
	Factors map[string]int64 `json:"factors,omitempty"`
}

var ErrNoRoute = errors.New("no provider route satisfies the requested constraints")

// RouteCandidates devolve todos os candidatos elegíveis em ordem, do melhor
// para o pior. Route usa o primeiro; o gateway usa a lista inteira para
// rotacionar quando a primeira tentativa falha. Retorna ErrNoRoute quando
// nenhum candidato satisfaz as restrições.
func (r *Registry) RouteCandidates(request RouteRequest) ([]RouteDecision, error) {
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
		// E-428: o PIN de modelo é restrição RÍGIDA, como `LocalOnly` — *os outros não são candidatos.* Um pin que
		// não pode ser honrado cai no `ErrNoRoute` do fim da função, **nunca** num "escolhi outro" silencioso.
		if request.PinnedModelID != "" && model.ID != request.PinnedModelID {
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
		// Cada parcela e' registada no mapa `factors` E somada ao score na mesma linha.
		// *Somar num sitio e explicar noutro e' como um relatorio que se desmente a si
		// proprio: cedo ou tarde alguem compara os dois.*
		score := int64(0)
		factors := map[string]int64{}
		somar := func(nome string, valor int64) {
			if valor != 0 {
				factors[nome] = valor
			}
			score += valor
		}
		somar("prioridade", int64(model.Priority*100))
		somar("qualidade", int64(model.QualityScore*10))
		somar("custo", -cost)
		if modelIsFree(model) {
			somar("gratuito", 100000)
		}
		if health.LatencyMS > 0 {
			if request.MaxLatencyMS > 0 && health.LatencyMS > request.MaxLatencyMS {
				continue
			}
			somar(ParetoLatency, -health.LatencyMS)
		}
		if health.ErrorRate > 0.0 {
			somar("taxa_erro", -int64(health.ErrorRate*1000))
		}
		if request.PreferredProvider != "" {
			somar("provedor_preferido", 10000)
		}
		// Optional user preference reweights the balanced default. The empty
		// (Auto) preference leaves the score unchanged for backward compatibility.
		switch request.Preference {
		case RoutePreferenceBestQuality:
			somar("preferencia_qualidade", int64(model.QualityScore)*100)
		case RoutePreferenceCheapest:
			somar("preferencia_custo", -cost*100)
		case RoutePreferenceFastest:
			if health.LatencyMS > 0 {
				somar("preferencia_latencia", -health.LatencyMS*20)
			}
		case RoutePreferenceMostPrivate:
			// FASE 19: PRIVACIDADE e' um eixo, nao um filtro. O local sobe; o remoto nao
			// desaparece. `LocalOnly` continua a ser o filtro rigido, e sao coisas
			// diferentes de proposito.
			if provider.AllowPrivate {
				somar("privacidade_local", 50000)
			}
		}
		candidates = append(candidates, RouteDecision{Model: model, Score: score, Reason: routeReason(model, health, request), Factors: factors})
	}
	if len(candidates) == 0 {
		return nil, ErrNoRoute
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Model.ID < candidates[j].Model.ID
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates, nil
}

// Route mantém o contrato histórico: o melhor candidato ou ErrNoRoute.
func (r *Registry) Route(request RouteRequest) (RouteDecision, error) {
	candidates, err := r.RouteCandidates(request)
	if err != nil {
		return RouteDecision{}, err
	}
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
