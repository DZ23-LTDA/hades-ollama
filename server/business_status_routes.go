package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Painel de BI / status de negócio (estágio 32).
//
// O escopo vem da SESSÃO via `companyForRequest`, nunca do corpo ou da query: um painel é
// exatamente o lugar onde um parâmetro de organização vindo do cliente viraria vazamento entre
// organizações. O relatório devolve, além dos indicadores, a PROVENIÊNCIA de cada bloco e a lista
// do que NÃO é medido.
func (a *agentAPI) businessStatus(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	report, err := a.runtime.BusinessStatus(company.OrganizationID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}
