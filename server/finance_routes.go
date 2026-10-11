package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas financeiras (estágio 30). Derivam lançamentos do comércio no livro existente;
// NÃO movem dinheiro — registram e sinalizam divergências.
func (a *agentAPI) finance() *agent.FinanceOps {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.Finance()
}

var errFinanceUnavailable = agent.ErrFinanceCompanyMissing

// postOrderRevenue lança a receita de um pedido PAGO. Pedido aguardando pagamento é
// recusado (409): lançar receita dele seria receita fictícia.
func (a *agentAPI) postOrderRevenue(c *gin.Context) {
	finance := a.finance()
	if finance == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errFinanceUnavailable)
		return
	}
	posting, err := finance.PostOrderRevenue(agentOrganizationID(c), c.Param("id"), agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// 201 quando cria; 200 quando a receita já existia (idempotente).
	status := http.StatusCreated
	if !posting.Created {
		status = http.StatusOK
	}
	c.JSON(status, posting)
}

func (a *agentAPI) postOrderCost(c *gin.Context) {
	finance := a.finance()
	if finance == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errFinanceUnavailable)
		return
	}
	var input struct {
		AmountCents int64  `json:"amount_cents"`
		Category    string `json:"category"`
		Reference   string `json:"reference"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	entry, err := finance.PostOrderCost(agentOrganizationID(c), c.Param("id"), input.AmountCents, input.Category, input.Reference, agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, entry)
}

// reconcileOrderWithStatement liga o pedido à linha do extrato no MESMO lançamento.
func (a *agentAPI) reconcileOrderWithStatement(c *gin.Context) {
	finance := a.finance()
	if finance == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errFinanceUnavailable)
		return
	}
	var input struct {
		Reference string `json:"reference"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	entry, err := finance.ReconcileOrderWithStatement(agentOrganizationID(c), c.Param("id"), input.Reference, agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, entry)
}

// financeReport devolve o panorama com as divergências MEDIDAS.
func (a *agentAPI) financeReport(c *gin.Context) {
	finance := a.finance()
	if finance == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errFinanceUnavailable)
		return
	}
	report, err := finance.Report(agentOrganizationID(c), time.Time{}, time.Time{}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}
