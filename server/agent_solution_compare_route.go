package server

// FASE 3 (E-1240): a rota que expoe a COMPARACAO OBJECTIVA DE SOLUCOES.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1239 implementou `agent.CompararSolucoes` e provou-a com 10 testes. Mas **nenhuma rota e
// nenhum painel a consumiam**: na pratica, a comparacao existia para quem escrevesse Go.
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma
//	funcao.*
//
// O QUE ESTA ROTA E', E O QUE NAO E'
//
// E' uma rota de CALCULO: recebe os candidatos com as suas medicoes e devolve o ranking, os
// rejeitados e a recomendacao. Nao le' worktrees, nao mede nada e **nao integra nada**.
//
//	*As medicoes sao ENTRADAS. A rota nao inventa o que nao mediu -- quem mede e' o chamador, e o
//	campo `medido` obriga-o a dizer se mediu.*
//
// PORQUE E' POST SE NAO MUTA NADA
//
// O corpo pode ter dezenas de candidatos com dezenas de medicoes cada. Isso nao cabe num URL.
// POST sem efeito secundario e' o compromises: o campo `somente_leitura` da resposta diz
// explicitamente que nada foi alterado, para o cliente nao ter de inferir do metodo.

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// maxCandidatosNaComparacao limita o tamanho do pedido.
//
//	*Uma rota que aceita candidatos sem limite e' um vector de esgotamento de memoria com nome de
//	funcionalidade.*
const maxCandidatosNaComparacao = 32

// comparacaoDeSolucoesPedido e' o corpo de POST /solutions/compare.
type comparacaoDeSolucoesPedido struct {
	// Missao liga a comparacao a um /goal, quando aplicavel.
	Missao string `json:"missao,omitempty"`
	// Candidatos sao as solucoes a comparar.
	Candidatos []agent.Candidato `json:"candidatos"`
}

// comparacaoDeSolucoesResposta e' a resposta de POST /solutions/compare.
type comparacaoDeSolucoesResposta struct {
	agent.ResultadoDaComparacao
	// Missao ecoa o pedido, quando veio.
	Missao string `json:"missao,omitempty"`
	// SomenteLeitura diz explicitamente que NADA foi alterado.
	//
	//	*O metodo e' POST, mas o efeito e' nenhum; o cliente nao deve ter de inferir isso do metodo.*
	SomenteLeitura bool `json:"somente_leitura"`
	// Avisos explica o que pode surpreender quem le'.
	Avisos []string `json:"avisos"`
}

// comparacaoDeSolucoes trata POST /solutions/compare.
func (a *agentAPI) comparacaoDeSolucoes(c *gin.Context) {
	var pedido comparacaoDeSolucoesPedido
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "corpo JSON invalido",
			"detalhe": err.Error(),
		})
		return
	}
	if len(pedido.Candidatos) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o campo 'candidatos' e' obrigatorio e nao pode ser vazio",
		})
		return
	}
	if len(pedido.Candidatos) > maxCandidatosNaComparacao {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"erro":      "demasiados candidatos",
			"recebidos": len(pedido.Candidatos),
			"maximo":    maxCandidatosNaComparacao,
		})
		return
	}

	resultado, err := agent.CompararSolucoes(pedido.Candidatos)
	if err != nil {
		// `ErrComparacaoSemCandidatos` ja' foi apanhado acima; qualquer outro erro e' do chamador.
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	resp := comparacaoDeSolucoesResposta{
		ResultadoDaComparacao: resultado,
		Missao:                strings.TrimSpace(pedido.Missao),
		SomenteLeitura:        true,
		Avisos:                []string{},
	}

	// Avisos derivados do RESULTADO, nao de constantes.
	if resultado.Empate {
		resp.Avisos = append(resp.Avisos,
			"ha' empate nos criterios medidos: nenhum vencedor foi escolhido, a decisao e' humana")
	}
	if len(resultado.Rejeitados) > 0 {
		resp.Avisos = append(resp.Avisos,
			"candidatos rejeitados nao entram no ranking; o motivo de cada um esta' em 'rejeitados'")
	}
	if len(resultado.NaoMedidos) > 0 {
		resp.Avisos = append(resp.Avisos,
			"criterios em 'nao_medidos' nao separaram os candidatos porque ninguem os mediu")
	}
	if len(resultado.MedidosParcialmente) > 0 {
		resp.Avisos = append(resp.Avisos,
			"criterios em 'medidos_parcialmente' foram medidos em alguns candidatos e nao noutros")
	}
	if resultado.Vencedor == "" && !resultado.Empate && len(resultado.Rejeitados) > 0 {
		resp.Avisos = append(resp.Avisos,
			"nenhum candidato passou nos testes: nao ha' recomendacao")
	}
	resp.Avisos = append(resp.Avisos,
		"esta rota NAO integra nada: 'integra_automaticamente' e' sempre falso e a aprovacao "+
			"humana e' obrigatoria")

	c.JSON(http.StatusOK, resp)
}
