package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Agentes de atendimento e vendas (estágio 35). As rotas expõem o catálogo e o turno.
// O turno SEMPRE declara a procedência do texto e o motivo de eventual escalada.
func (a *agentAPI) serviceAgents() *agent.ServiceAgentRuntime {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.ServiceAgents()
}

func (a *agentAPI) listServiceAgents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"agents": agent.ServiceAgentCatalog(), "count": len(agent.ServiceAgentCatalog())})
}

// serviceAgentTurn executa um turno do agente sobre uma conversa da caixa de entrada.
func (a *agentAPI) serviceAgentTurn(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	var input struct {
		ConversationID string `json:"conversation_id"`
		Intent         string `json:"intent"`
		ModelID        string `json:"model_id"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	turn, err := runtime.RunTurn(agent.ServiceAgentTurnRequest{
		OrganizationID: agentOrganizationID(c),
		AgentID:        c.Param("id"),
		ConversationID: input.ConversationID,
		Intent:         input.Intent,
		ModelID:        input.ModelID,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// Escalada é uma resposta legítima e vem com o motivo; responder traz a mensagem.
	status := http.StatusOK
	if turn.Escalated {
		status = http.StatusAccepted
	}
	c.JSON(status, turn)
}

var errServiceAgentsUnavailable = agent.ErrServiceAgentTurnInvalid
