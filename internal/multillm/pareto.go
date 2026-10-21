package multillm

// FASE 19 do adendo V3: roteamento multiobjectivo e fronteira de Pareto.
//
// O QUE FALTAVA, medido antes de escrever isto:
//
//   - `RouteDecision` tinha apenas `Score` e `Reason`. *Uma pontuacao opaca como unica
//     justificativa e' exactamente o que o adendo proibe*: nao havia como saber PORQUE um
//     modelo ganhou, nem comparar alternativas que nao fossem "a que tem score maior".
//   - Nao existia fronteira de Pareto: quando um modelo e' melhor em qualidade e outro em
//     custo, o router escolhia um e escondia o outro. *Escolher bem quando ha' um vencedor
//     claro e' facil; escolher quando nao ha' e' que precisa de ser explicado.*
//   - Nao havia politica `mais privado`: `LocalOnly` e' uma RESTRICAO RIGIDA (filtra), e
//     nao uma PREFERENCIA (ordena). Sao coisas diferentes e estavam confundidas.

import "sort"

// RoutePreferenceMostPrivate prefere o candidato mais local sem exigir que todos sejam
// locais. A diferenca para `LocalOnly` e' a diferenca entre ORDENAR e FILTRAR:
// `LocalOnly` recusa o que nao e' local; `MostPrivate` ordena o local primeiro e admite o
// remoto como fallback visivel. *Confundir as duas faz um utilizador que queria "preferir
// local" ficar sem servico quando o local cai -- sem saber porque.*
const RoutePreferenceMostPrivate RoutePreference = "private"

// Eixos de comparacao da fronteira. Os nomes sao os do adendo.
const (
	ParetoQuality = "qualidade"
	ParetoCost    = "custo"
	ParetoLatency = "latencia"
	ParetoPrivacy = "privacidade"
)

// ParetoScores sao os eixos de um candidato, ja' orientados para "maior e' melhor".
//
// A orientacao acontece AQUI e nao no chamador: *um eixo em que menor e' melhor e' a
// origem classica do bug em que a fronteira devolve o pior candidato.* Custo e latencia
// entram negativos para que a comparacao seja sempre "dominado = todos os eixos menores".
type ParetoScores struct {
	Quality int64
	Cost    int64 // ja' NEGADO: custo menor vira valor MAIOR
	Latency int64 // ja' NEGADO: latencia menor vira valor MAIOR
	Privacy int64 // 1 = provider local, 0 = remoto
}

// ScoresDoCandidato extrai os eixos de um candidato. `local` diz se o provider do modelo
// permite rede privada -- o mesmo criterio que `LocalOnly` usa para filtrar, aqui usado
// para ordenar.
func ScoresDoCandidato(decision RouteDecision, local bool) ParetoScores {
	scores := ParetoScores{
		Quality: int64(decision.Model.QualityScore),
		Cost:    -(decision.Model.CostPer1KInputCents + decision.Model.CostPer1KOutputCents),
	}
	if latencia, tem := decision.Factors[ParetoLatency]; tem && latencia != 0 {
		scores.Latency = -latencia
	}
	if local {
		scores.Privacy = 1
	}
	return scores
}

// Dominado diz se `a` domina `b`: e' >= em TODOS os eixos e > em pelo menos um.
//
// Nao ha' pesos nesta funcao, e' deliberado: *uma fronteira de Pareto com pesos nao e'
// uma fronteira de Pareto, e' outra pontuacao opaca com outro nome.*
func Dominado(a, b ParetoScores) bool {
	maiorEmAlgum := false
	pares := [][2]int64{
		{a.Quality, b.Quality},
		{a.Cost, b.Cost},
		{a.Latency, b.Latency},
		{a.Privacy, b.Privacy},
	}
	for _, par := range pares {
		if par[0] < par[1] {
			return false
		}
		if par[0] > par[1] {
			maiorEmAlgum = true
		}
	}
	return maiorEmAlgum
}

// ParetoFrontier devolve os candidatos NAO-DOMINADOS, preservando a ordem recebida.
//
// Um candidato fica na fronteira se nenhum outro for melhor ou igual em todos os eixos e
// estritamente melhor em pelo menos um. Quando todos sao iguais nos eixos, todos ficam:
// *empate nao e' dominancia, e esconder o empate faz o utilizador achar que a escolha foi
// melhor do que foi.*
func ParetoFrontier(candidatos []RouteDecision, localDe func(RouteDecision) bool) []RouteDecision {
	if len(candidatos) <= 1 {
		return append([]RouteDecision(nil), candidatos...)
	}
	scores := make([]ParetoScores, len(candidatos))
	for i, c := range candidatos {
		scores[i] = ScoresDoCandidato(c, localDe != nil && localDe(c))
	}
	fronteira := make([]RouteDecision, 0, len(candidatos))
	for i, candidato := range candidatos {
		dominado := false
		for j := range candidatos {
			if i == j {
				continue
			}
			if Dominado(scores[j], scores[i]) {
				dominado = true
				break
			}
		}
		if !dominado {
			fronteira = append(fronteira, candidato)
		}
	}
	return fronteira
}

// ParetoResumo devolve a fronteira com os eixos que a definem, para o Mission Control
// poder mostrar ALTERNATIVAS e nao apenas o vencedor.
func ParetoResumo(candidatos []RouteDecision, localDe func(RouteDecision) bool) []ParetoAlternative {
	fronteira := ParetoFrontier(candidatos, localDe)
	alternativas := make([]ParetoAlternative, 0, len(fronteira))
	for _, c := range fronteira {
		alternativas = append(alternativas, ParetoAlternative{
			ModelID: c.Model.ID,
			Score:   c.Score,
			Eixos:   ScoresDoCandidato(c, localDe != nil && localDe(c)),
		})
	}
	// Ordem estavel por score: a fronteira e' um conjunto, mas a lista que o utilizador le
	// tem de ser deterministica. *Uma lista que muda de ordem entre duas leituras iguais
	// faz duvidar do dado.*
	sort.SliceStable(alternativas, func(i, j int) bool {
		if alternativas[i].Score == alternativas[j].Score {
			return alternativas[i].ModelID < alternativas[j].ModelID
		}
		return alternativas[i].Score > alternativas[j].Score
	})
	return alternativas
}

// ParetoAlternative e' um candidato da fronteira com os eixos expostos.
type ParetoAlternative struct {
	ModelID string       `json:"model_id"`
	Score   int64        `json:"score"`
	Eixos   ParetoScores `json:"eixos"`
}
