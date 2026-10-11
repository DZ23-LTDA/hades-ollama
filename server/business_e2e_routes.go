package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Certificação E2E da empresa em sandbox (estágio 38). É uma superfície REAL: o
// operador roda a jornada completa e recebe o relatório com evidência por etapa, a
// prova de isolamento e a lista do que depende de conta/provedor externo.
func (a *agentAPI) certifyBusiness(c *gin.Context) {
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrFinanceCompanyMissing)
		return
	}
	var input struct {
		OtherOrganizationID string `json:"other_organization_id"`
		CheckoutKey         string `json:"checkout_key"`
		StatementReference  string `json:"statement_reference"`
		SKU                 string `json:"sku"`
		ProductName         string `json:"product_name"`
		Quantity            int64  `json:"quantity"`
		InitialStock        int64  `json:"initial_stock"`
		PriceCents          int64  `json:"price_cents"`
	}
	if c.Request.ContentLength > 0 {
		if err := decodeJSON(c, &input); err != nil {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
	}
	organizationID := agentOrganizationID(c)
	otherOrganizationID := input.OtherOrganizationID
	if otherOrganizationID == "" {
		// A prova de isolamento sempre precisa de uma SEGUNDA organização; sem uma
		// informada, usamos um identificador determinístico derivado do tenant.
		otherOrganizationID = "certificacao-outra-organizacao"
	}
	journey := agent.DefaultBusinessJourneyInput(organizationID, otherOrganizationID)
	if input.CheckoutKey != "" {
		journey.CheckoutKey = input.CheckoutKey
	}
	if input.StatementReference != "" {
		journey.StatementRef = input.StatementReference
	}
	if input.SKU != "" {
		journey.SKU = input.SKU
	}
	if input.ProductName != "" {
		journey.ProductName = input.ProductName
	}
	if input.Quantity > 0 {
		journey.Quantity = input.Quantity
	}
	if input.InitialStock > 0 {
		journey.InitialStock = input.InitialStock
	}
	if input.PriceCents > 0 {
		journey.PriceCents = input.PriceCents
	}
	report, err := a.runtime.CertifyBusinessJourney(journey, time.Now().UTC())
	if err != nil {
		// O relatório VAI junto mesmo em falha: é ele que diz qual etapa quebrou.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"report": report, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
