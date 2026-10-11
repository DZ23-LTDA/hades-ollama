package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Série temporal do painel de negócio (estágio 32).
//
// Mesmo escopo da leitura do painel: vem da SESSÃO via `companyForRequest`. Registrar amostra é
// idempotente por janela de uma hora — o servidor devolve `created=false` quando a hora já tinha
// amostra, em vez de inflar a série com repetições do mesmo instante.

// businessSnapshots lê a série (e a tendência) da organização da empresa da sessão.
func (a *agentAPI) businessSnapshots(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var from time.Time
	if raw := c.Query("from"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			writeAgentError(c, http.StatusBadRequest, parseErr)
			return
		}
		from = parsed
	}
	trend, err := a.runtime.BusinessTrend(company.OrganizationID, from)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, trend)
}

// recordBusinessSnapshot registra a amostra do agora e devolve a série atualizada.
func (a *agentAPI) recordBusinessSnapshot(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	snapshot, created, trend, err := a.runtime.BusinessSnapshot(company.OrganizationID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusOK
	if created {
		// 201 apenas quando a amostra é REALMENTE nova: a janela já registrada não é criação.
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"created": created, "snapshot": snapshot, "trend": trend})
}
