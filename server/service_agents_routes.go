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
		// RequestedTools é o que o agente PEDE para este turno. Pedir não é poder: o runtime confere
		// declaração do agente, executor registrado e política antes de qualquer efeito.
		RequestedTools []string `json:"requested_tools,omitempty"`
		// ToolArguments alimenta as ferramentas de escrita (título, valor, responsável, motivo). O ESCOPO
		// (organização, contato, conversa) vem da sessão e da conversa resolvida, nunca daqui.
		ToolArguments struct {
			Title      string `json:"title,omitempty"`
			ValueCents int64  `json:"value_cents,omitempty"`
			OwnerID    string `json:"owner_id,omitempty"`
			Assignee   string `json:"assignee,omitempty"`
			Reason     string `json:"reason,omitempty"`
		} `json:"tool_arguments,omitempty"`
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
		RequestedTools: input.RequestedTools,
		ToolArguments: agent.ServiceAgentToolRequest{
			Title:      input.ToolArguments.Title,
			ValueCents: input.ToolArguments.ValueCents,
			OwnerID:    input.ToolArguments.OwnerID,
			Assignee:   input.ToolArguments.Assignee,
			Reason:     input.ToolArguments.Reason,
		},
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
