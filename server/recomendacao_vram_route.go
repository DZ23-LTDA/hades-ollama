package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// recomendacaoPorVRAM atende `GET /systems/recomendacao?vram_bytes=N&parametros_b=7`.
//
// # PORQUE UM CALCULO, E NAO UM MEDIDOR
//
// `agent.RecomendarParaVRAM` e' uma funcao pura: recebe a VRAM medida e os parametros do modelo e
// devolve quantizacao, contexto e quantas camadas cabem. O servidor **nao mede a VRAM do cliente** --
// quem a mede passa-a. Fingir que o servidor mede seria o mesmo erro que a missao ja' registou sobre
// ferramentas que dizem OK sem ter medido.
//
// # AS RECUSAS SAO A PARTE IMPORTANTE
//
// Sem `vram_bytes` valido, a funcao devolve `ErrVRAMDesconhecida` e esta rota responde **400 com o
// motivo** -- nunca uma recomendacao por omissao. Uma recomendacao inventada gasta a VRAM de alguem.
//
// A identidade vem da SESSAO (`requireOrganizacaoDaSessao`), nunca da query.
func (a *agentAPI) recomendacaoPorVRAM(c *gin.Context) {
	if _, ok := requireOrganizacaoDaSessao(c); !ok {
		return
	}
	vramBruta := strings.TrimSpace(c.Query("vram_bytes"))
	parametrosBrutos := strings.TrimSpace(c.Query("parametros_b"))

	// ParseUint, e nao ParseInt: a funcao do dominio trabalha com bytes (uint64) e um "-1" tem de ser
	// recusado como ilegivel, nao convertido.
	vram, errVRAM := strconv.ParseUint(vramBruta, 10, 64)
	if errVRAM != nil || vram == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "vram_bytes tem de ser um inteiro positivo de bytes medidos",
			"detalhe": "sem VRAM medida nao ha' recomendacao: a funcao recusa em vez de adivinhar",
		})
		return
	}
	parametros := 0.0
	if parametrosBrutos != "" {
		lido, errParametros := strconv.ParseFloat(parametrosBrutos, 64)
		if errParametros != nil || lido <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"erro": "parametros_b tem de ser um numero positivo"})
			return
		}
		parametros = lido
	}

	recomendacao, err := agent.RecomendarParaVRAM(vram, parametros)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recomendacao": recomendacao})
}
