package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas do supervisor autônomo de negócio (estágio 31). O ciclo é acionado pelo
// operador ou pelo agendador existente; as ações irreversíveis ficam aguardando
// decisão humana explícita.
func (a *agentAPI) businessSupervisor() *agent.BusinessSupervisor {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.BusinessSupervisor()
}

var errBusinessSupervisorUnavailable = agent.ErrBusinessActionNotFound

// businessSupervisorTick roda um ciclo de supervisão sobre os domínios.
func (a *agentAPI) businessSupervisorTick(c *gin.Context) {
	supervisor := a.businessSupervisor()
	if supervisor == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errBusinessSupervisorUnavailable)
		return
	}
	result, err := supervisor.BusinessTick(agentOrganizationID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// O resultado VAI junto mesmo com erro de domínio declarado: é ele que mostra o que
	// foi observado, executado, bloqueado e o que aguarda decisão.
	c.JSON(http.StatusOK, result)
}

// listBusinessActions devolve as ações aguardando decisão humana.
func (a *agentAPI) listBusinessActions(c *gin.Context) {
	supervisor := a.businessSupervisor()
	if supervisor == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errBusinessSupervisorUnavailable)
		return
	}
	pending := supervisor.PendingActions(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"pending": pending, "count": len(pending), "policy": supervisor.Policy()})
}

// decideBusinessAction aplica a decisão humana: aprovar EXECUTA, recusar não executa.
func (a *agentAPI) decideBusinessAction(c *gin.Context) {
	supervisor := a.businessSupervisor()
	if supervisor == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errBusinessSupervisorUnavailable)
		return
	}
	var input struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	action, err := supervisor.Decide(agentOrganizationID(c), c.Param("id"), input.Decision, agentActorID(c), input.Reason, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, action)
}
