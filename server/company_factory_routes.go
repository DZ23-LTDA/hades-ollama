package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas da fábrica de empresa (estágio 23). O lançamento materializa artefatos reais
// (plano, marca, precificação e site) e devolve o checklist com o que depende de
// terceiros — a fábrica NÃO cria cliente nem emite registro legal.
func (a *agentAPI) launchCompanyFactory(c *gin.Context) {
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrCompanyNotFound)
		return
	}
	var input struct {
		Company struct {
			Name           string `json:"name"`
			Mission        string `json:"mission"`
			Positioning    string `json:"positioning"`
			BusinessModel  string `json:"business_model"`
			TargetAudience string `json:"target_audience"`
			Offer          string `json:"offer"`
			Currency       string `json:"currency"`
		} `json:"company"`
		Tagline  string   `json:"tagline"`
		Palette  []string `json:"palette"`
		Products []struct {
			SKU           string `json:"sku"`
			Name          string `json:"name"`
			CostCents     int64  `json:"cost_cents"`
			MarginPercent int64  `json:"margin_percent"`
			InitialStock  int64  `json:"initial_stock"`
		} `json:"products"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	products := make([]agent.CompanyPricingInput, 0, len(input.Products))
	for _, item := range input.Products {
		products = append(products, agent.CompanyPricingInput{
			SKU: item.SKU, Name: item.Name, CostCents: item.CostCents,
			MarginPercent: item.MarginPercent, InitialStock: item.InitialStock,
		})
	}
	report, err := a.runtime.LaunchCompany(agent.CompanyFactoryRequest{
		OrganizationID: agentOrganizationID(c),
		Company: agent.CompanyCreateRequest{
			Name: input.Company.Name, Mission: input.Company.Mission, Positioning: input.Company.Positioning,
			BusinessModel: input.Company.BusinessModel, TargetAudience: input.Company.TargetAudience,
			Offer: input.Company.Offer, Currency: input.Company.Currency,
		},
		Tagline:  input.Tagline,
		Palette:  input.Palette,
		Products: products,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, report)
}

// companyFactoryReport devolve o relatório reconstruído do ESTADO REAL (empresa,
// catálogo e arquivos em disco).
func (a *agentAPI) companyFactoryReport(c *gin.Context) {
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrCompanyNotFound)
		return
	}
	report, err := a.runtime.CompanyFactoryReportFor(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}
