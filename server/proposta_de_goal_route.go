package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// pedidoDeProposta e' o corpo aceite por `propostaDeGoal`.
//
// O CORPO TRAZ A CONVERSA, NUNCA A IDENTIDADE. A organizacao vem da sessao, como em todas as rotas
// desta fatia -- aceitar um `organization_id` aqui seria dar ao corpo o poder de escolher o dono.
type pedidoDeProposta struct {
	Mensagens []agent.MensagemDaConversa `json:"mensagens"`
}

// propostaDeGoal atende `POST /goals/proposta`.
//
// PROPOE, NAO CRIA. `ProporGoalDaConversa` devolve um goal proposto com o motivo e, sobretudo, com o
// que **falta** na conversa. Nada e' gravado: criar o goal e' de quem le a proposta.
//
// A recusa e' a parte importante: conversa vazia -> 400 com o motivo, e a resposta nao traz proposta.
func (a *agentAPI) propostaDeGoal(c *gin.Context) {
	if _, ok := requireOrganizacaoDaSessao(c); !ok {
		return
	}
	var pedido pedidoDeProposta
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "corpo invalido: esperado {\"mensagens\": [...]}"})
		return
	}
	proposta, err := agent.ProporGoalDaConversa(pedido.Mensagens)
	if err != nil {
		if errors.Is(err, agent.ErrConversaVazia) {
			c.JSON(http.StatusBadRequest, gin.H{
				"erro":    err.Error(),
				"detalhe": "sem conversa nao ha' proposta: a funcao recusa em vez de inventar um goal",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proposta": proposta})
}
