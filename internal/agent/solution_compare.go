package agent

// FASE 3 do adendo V3: COMPARACAO OBJECTIVA DE SOLUCOES.
//
// O QUE JA' EXISTIA, E O QUE FALTAVA
//
// O `swarm_worktree_isolation_test.go` ja' provava que tres especialistas em worktrees separados
// nao se pisam e que o merge e' controlado. O `AgentOrchestrator` ja' tem fan-out com
// `MaxAgents`, papeis e orcamento.
//
// O que NAO existia era a segunda metade da FASE 3: com varias solucoes na mao, **qual escolher,
// com que criterio, e quem decide**.
//
//	*Paralelismo sem comparacao produz mais trabalho, nao melhor trabalho.*
//
// AS QUATRO REGRAS QUE ESTE FICHEIRO IMPOE
//
//  1. **Teste que falha desqualifica.** Nao se compara o que nao passa. Uma solucao com testes
//     vermelhos nao entra no ranking por muito rapida que seja.
//
//  2. **Medicao ausente NAO e' zero.** Um criterio nao medido fica `Medido: false` e o candidato
//     nao ganha nem perde pontos por ele -- e a comparacao DIZ que ficou por medir.
//
//		*Marcar como zero o que nao foi medido e' inventar uma derrota.*
//
//  3. **Empate nao se desempata por sorte.** Se dois candidatos ficam iguais nos criterios
//     medidos, o resultado e' `Empate: true` e o vencedor fica vazio. Escolher um deles seria
//     decidir por um criterio que nao esta' escrito.
//
//  4. **A comparacao NUNCA integra.** `IntegraAutomaticamente` e' constante `false` -- e' um campo
//     do contrato precisamente para que um cliente nao possa inferir o contrario.
//
//		*O sistema pode recomendar um vencedor; nao pode escolher por quem assina o merge.*

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// CriterioDeComparacao e' um eixo verificavel da FASE 3 do adendo.
type CriterioDeComparacao string

const (
	// CriterioTestes conta suites que passaram.
	CriterioTestes CriterioDeComparacao = "testes"
	// CriterioGates conta gates (lint, typecheck, build) que passaram.
	CriterioGates CriterioDeComparacao = "gates"
	// CriterioSeguranca conta verificacoes de seguranca que passaram.
	CriterioSeguranca CriterioDeComparacao = "seguranca"
	// CriterioRegressao conta regressoes DETECTADAS (menos e' melhor).
	CriterioRegressao CriterioDeComparacao = "regressoes"
	// CriterioTamanhoDoDiff e' o numero de linhas alteradas (menos e' melhor, com ressalva).
	CriterioTamanhoDoDiff CriterioDeComparacao = "tamanho_do_diff"
	// CriterioTempo e' a duracao em segundos (menos e' melhor).
	CriterioTempo CriterioDeComparacao = "tempo"
	// CriterioCusto e' o custo em unidades monetarias (menos e' melhor).
	CriterioCusto CriterioDeComparacao = "custo"
	// CriterioEvidencias conta evidencias produzidas (mais e' melhor).
	CriterioEvidencias CriterioDeComparacao = "evidencias"
)

// CriteriosDaFase3 devolve todos os criterios, na ordem em que sao avaliados.
//
// A ordem importa: os criterios de CORRECTUDE vem antes dos de eficiencia.
//
//	*Uma solucao rapida que nao funciona nao e' uma solucao rapida: e' uma solucao que falhou
//	depressa.*
func CriteriosDaFase3() []CriterioDeComparacao {
	return []CriterioDeComparacao{
		CriterioTestes,
		CriterioGates,
		CriterioSeguranca,
		CriterioRegressao,
		CriterioEvidencias,
		CriterioTamanhoDoDiff,
		CriterioTempo,
		CriterioCusto,
	}
}

// MedicaoDeCandidato e' o valor de UM criterio para UM candidato.
type MedicaoDeCandidato struct {
	Criterio CriterioDeComparacao `json:"criterio"`
	// Valor e' o numero medido. Ignorado quando `Medido` e' falso.
	Valor float64 `json:"valor"`
	// Unidade e' o que o numero significa ("linhas", "segundos", "contagem", ...).
	Unidade string `json:"unidade,omitempty"`
	// Medido distingue "medi e deu zero" de "nao medi".
	//
	//	*Marcar como zero o que nao foi medido e' inventar uma derrota.*
	Medido bool `json:"medido"`
	// Fonte diz de onde veio o numero, para o auditor poder refazer a medicao.
	Fonte string `json:"fonte,omitempty"`
}

// Candidato e' UMA solucao candidata, num worktree proprio.
type Candidato struct {
	// ID identifica a solucao no relatorio.
	ID string `json:"id"`
	// Worktree e' onde a solucao vive. Vazio significa que a comparacao nao sabe onde ela esta'.
	Worktree string `json:"worktree,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Commit   string `json:"commit,omitempty"`
	// Agente e' quem a produziu, quando aplicavel.
	Agente string `json:"agente,omitempty"`
	// TestesPassaram e TestesFalharam sao a contagem CRUA, nao uma medicao.
	TestesPassaram int `json:"testes_passaram"`
	TestesFalharam int `json:"testes_falharam"`
	// Medicoes sao os criterios que foram efectivamente medidos.
	Medicoes []MedicaoDeCandidato `json:"medicoes"`
	// Evidencias sao IDs do indice canonico, nao texto livre.
	Evidencias []string `json:"evidencias,omitempty"`
}

// MotivoDeRejeicao explica porque um candidato ficou fora do ranking.
type MotivoDeRejeicao struct {
	CandidatoID string `json:"candidato_id"`
	// Motivo e' legivel e diz o numero, nao uma impressao.
	Motivo string `json:"motivo"`
}

// ResultadoDaComparacao e' o veredicto da comparacao.
type ResultadoDaComparacao struct {
	Candidatos []Candidato `json:"candidatos"`
	// Ranking sao os candidatos ACEITES, ordenados do melhor para o pior.
	Ranking []string `json:"ranking"`
	// Vencedor fica vazio quando ha' empate ou quando ninguem foi aceite.
	Vencedor string `json:"vencedor,omitempty"`
	// Empate diz que os criterios MEDIDOS nao separam os primeiros.
	Empate bool `json:"empate"`
	// Rejeitados sao os candidatos que nao entraram no ranking, com motivo.
	Rejeitados []MotivoDeRejeicao `json:"rejeitados"`
	// NaoMedidos sao os criterios que NENHUM candidato conseguiu medir.
	NaoMedidos []CriterioDeComparacao `json:"nao_medidos"`
	// MedidosParcialmente sao os criterios que ALGUNS candidatos mediram e outros nao.
	//
	//	*Um criterio medido num candidato e nao noutro nao compara: informa.*
	//
	// Sem este campo, um criterio medido em metade dos candidatos desaparecia do relatorio e quem
	// auditasse nao saberia que a comparacao foi feita sem ele.
	MedidosParcialmente []CriterioDeComparacao `json:"medidos_parcialmente"`
	// IntegraAutomaticamente e' SEMPRE falso.
	//
	//	*O sistema pode recomendar um vencedor; nao pode escolher por quem assina o merge.*
	IntegraAutomaticamente bool `json:"integra_automaticamente"`
	// ExigeAprovacaoHumana e' o espelho de `IntegraAutomaticamente`: true quando ha' vencedor ou
	// empate, porque ha' algo a decidir.
	ExigeAprovacaoHumana bool `json:"exige_aprovacao_humana"`
	// Explicacao diz em uma frase o que o resultado significa.
	Explicacao string `json:"explicacao"`
}

// ErrComparacaoSemCandidatos e' devolvido quando nao ha' nada a comparar.
var ErrComparacaoSemCandidatos = errors.New("comparacao sem candidatos")

// criterioEhMenorMelhor diz quais criterios beneficiam de valores menores.
func criterioEhMenorMelhor(c CriterioDeComparacao) bool {
	switch c {
	case CriterioRegressao, CriterioTamanhoDoDiff, CriterioTempo, CriterioCusto:
		return true
	}
	return false
}

// CompararSolucoes compara candidatos por criterios verificaveis e devolve uma recomendacao.
//
// NUNCA integra nada: o campo `IntegraAutomaticamente` do resultado e' constante `false`.
func CompararSolucoes(candidatos []Candidato) (ResultadoDaComparacao, error) {
	if len(candidatos) == 0 {
		return ResultadoDaComparacao{}, ErrComparacaoSemCandidatos
	}

	res := ResultadoDaComparacao{
		Candidatos:             append([]Candidato(nil), candidatos...),
		Ranking:                []string{},
		Rejeitados:             []MotivoDeRejeicao{},
		NaoMedidos:             []CriterioDeComparacao{},
		MedidosParcialmente:    []CriterioDeComparacao{},
		IntegraAutomaticamente: false,
	}

	// 1. DESQUALIFICAR. Teste vermelho nao entra no ranking.
	aceites := make([]Candidato, 0, len(candidatos))
	for _, c := range candidatos {
		switch {
		case strings.TrimSpace(c.ID) == "":
			res.Rejeitados = append(res.Rejeitados, MotivoDeRejeicao{
				CandidatoID: "(sem id)",
				Motivo:      "candidato sem identificador nao e' comparavel",
			})
		case c.TestesFalharam > 0:
			res.Rejeitados = append(res.Rejeitados, MotivoDeRejeicao{
				CandidatoID: c.ID,
				Motivo: fmt.Sprintf("%d teste(s) falharam: nao se compara o que nao passa",
					c.TestesFalharam),
			})
		case c.TestesPassaram == 0 && c.TestesFalharam == 0:
			res.Rejeitados = append(res.Rejeitados, MotivoDeRejeicao{
				CandidatoID: c.ID,
				Motivo:      "nenhum teste executado: sem teste nao ha' evidencia de que funcione",
			})
		default:
			aceites = append(aceites, c)
		}
	}

	if len(aceites) == 0 {
		res.Explicacao = "nenhum candidato passou nos testes; nada a recomendar"
		res.ExigeAprovacaoHumana = true
		return res, nil
	}

	// 2. QUE CRITERIOS FORAM MEDIDOS POR TODOS? So' esses separam.
	//
	//	*Um criterio medido num candidato e nao noutro nao compara: informa.*
	medidosPorTodos := make([]CriterioDeComparacao, 0, len(CriteriosDaFase3()))
	for _, criterio := range CriteriosDaFase3() {
		todos := true
		algum := false
		for _, c := range aceites {
			m, ok := medicaoDe(c, criterio)
			if ok && m.Medido {
				algum = true
				continue
			}
			todos = false
		}
		switch {
		case !algum:
			res.NaoMedidos = append(res.NaoMedidos, criterio)
		case !todos:
			// Alguns mediran, outros nao: o criterio NAO separa, mas tem de ser reportado.
			res.MedidosParcialmente = append(res.MedidosParcialmente, criterio)
		default:
			medidosPorTodos = append(medidosPorTodos, criterio)
		}
	}

	// 3. ORDENAR por comparacao lexicografica nos criterios medidos por todos.
	ordenados := append([]Candidato(nil), aceites...)
	sort.SliceStable(ordenados, func(i, j int) bool {
		for _, criterio := range medidosPorTodos {
			vi, _ := valorDe(ordenados[i], criterio)
			vj, _ := valorDe(ordenados[j], criterio)
			if vi == vj {
				continue
			}
			if criterioEhMenorMelhor(criterio) {
				return vi < vj
			}
			return vi > vj
		}
		// Empate: a ordem e' estavel, nao aleatoria.
		return false
	})

	for _, c := range ordenados {
		res.Ranking = append(res.Ranking, c.ID)
	}

	// 4. EMPATE: se os criterios medidos nao separam os primeiros, nao se inventa desempate.
	if len(res.Ranking) > 1 && empatam(ordenados[0], ordenados[1], medidosPorTodos) {
		res.Empate = true
		res.Explicacao = fmt.Sprintf(
			"empate entre %q e %q nos %d criterio(s) medido(s) por todos; nenhum vencedor escolhido",
			ordenados[0].ID, ordenados[1].ID, len(medidosPorTodos))
	} else if len(res.Ranking) == 1 {
		res.Vencedor = res.Ranking[0]
		res.Explicacao = "unico candidato que passou nos testes; recomendado sem comparacao"
	} else {
		res.Vencedor = res.Ranking[0]
		res.Explicacao = fmt.Sprintf(
			"%q lidera em %d criterio(s) medido(s) por todos; recomendacao sujeita a aprovacao humana",
			res.Vencedor, len(medidosPorTodos))
	}
	res.ExigeAprovacaoHumana = true

	// Garantia de contrato: nunca integra.
	res.IntegraAutomaticamente = false
	return res, nil
}

// medicaoDe procura a medicao de um criterio num candidato.
func medicaoDe(c Candidato, criterio CriterioDeComparacao) (MedicaoDeCandidato, bool) {
	for _, m := range c.Medicoes {
		if m.Criterio == criterio {
			return m, true
		}
	}
	return MedicaoDeCandidato{}, false
}

// valorDe devolve o valor medido, e se ele existe.
func valorDe(c Candidato, criterio CriterioDeComparacao) (float64, bool) {
	m, ok := medicaoDe(c, criterio)
	if !ok || !m.Medido {
		return 0, false
	}
	return m.Valor, true
}

// empatam diz se dois candidatos sao iguais em todos os criterios comparaveis.
func empatam(a, b Candidato, criterios []CriterioDeComparacao) bool {
	for _, criterio := range criterios {
		va, _ := valorDe(a, criterio)
		vb, _ := valorDe(b, criterio)
		if va != vb {
			return false
		}
	}
	return true
}
