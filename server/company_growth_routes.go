package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) companyGrowthReport(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	report, err := a.runtime.CompanyStore().GrowthReport(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (a *agentAPI) addCompanyCampaign(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyCampaign
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddCampaign(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if len(company.Campaigns) > 0 {
		if approval, approvalErr := a.runtime.CompanyStore().PendingApproval(c.Param("id"), "campaign", company.Campaigns[len(company.Campaigns)-1].ID); approvalErr == nil {
			_ = a.runtime.CompanyStore().BindApprovalRequester(c.Param("id"), approval.ID, agentActorID(c))
		}
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) approveCompanyCampaign(c *gin.Context) {
	a.decideCompanyApproval(c, "campaign", "campaign_id")
}

func (a *agentAPI) launchCompanyCampaign(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	company, err := a.runtime.CompanyStore().LaunchCampaign(c.Param("id"), c.Param("campaign_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) pauseCompanyCampaign(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	company, err := a.runtime.CompanyStore().PauseCampaign(c.Param("id"), c.Param("campaign_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) addCompanyAffiliateProgram(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyAffiliateProgram
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddAffiliateProgram(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if len(company.AffiliatePrograms) > 0 {
		if approval, approvalErr := a.runtime.CompanyStore().PendingApproval(c.Param("id"), "affiliate_program", company.AffiliatePrograms[len(company.AffiliatePrograms)-1].ID); approvalErr == nil {
			_ = a.runtime.CompanyStore().BindApprovalRequester(c.Param("id"), approval.ID, agentActorID(c))
		}
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) approveCompanyAffiliateProgram(c *gin.Context) {
	a.decideCompanyApproval(c, "affiliate_program", "program_id")
}

func (a *agentAPI) addCompanyAffiliateLink(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyAffiliateLink
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddAffiliateLink(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) recordCompanyAffiliateConversion(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		RevenueCents int64 `json:"revenue_cents"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().RecordAffiliateConversionWithIdempotency(c.Param("id"), c.Param("link_id"), input.RevenueCents, c.GetHeader("Idempotency-Key"))
	if err != nil {
		if errors.Is(err, agent.ErrCompanyIdempotentReplay) {
			c.JSON(http.StatusOK, company)
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) addCompanyProduct(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyProduct
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddProduct(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) createCompanyOrder(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyOrder
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if input.IdempotencyKey == "" {
		input.IdempotencyKey = c.GetHeader("Idempotency-Key")
	}
	// E-467: `CreateOrder` passou a devolver **o pedido** (criado ou o existente, com `Replayed`) além da empresa ⇒
	// *a superfície deixa de adivinhar qual pedido nasceu, e a reentrega passa a dizer que foi reentrega.*
	company, pedido, err := a.runtime.CompanyStore().CreateOrder(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if pedido.ID != "" {
		if approval, approvalErr := a.runtime.CompanyStore().PendingApproval(c.Param("id"), "order", pedido.ID); approvalErr == nil {
			_ = a.runtime.CompanyStore().BindApprovalRequester(c.Param("id"), approval.ID, agentActorID(c))
		}
	}
	// **409 numa reentrega**, como no agendamento (E-458): *o pedido existe, e a resposta diz qual é.* E `201` quando
	// nasceu agora. *Um cliente que reentrega por queda de rede quer saber se criou ou se já existia — e não uma lição
	// sobre o formato do pedido.*
	codigo := http.StatusCreated
	if pedido.Replayed {
		codigo = http.StatusConflict
	}
	c.JSON(codigo, gin.H{"company": company, "order": pedido, "idempotent_replay": pedido.Replayed})
}

func (a *agentAPI) approveCompanyOrder(c *gin.Context) {
	a.decideCompanyApproval(c, "order", "order_id")
}

func (a *agentAPI) fulfillCompanyOrder(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		TrackingCode string `json:"tracking_code"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().FulfillOrder(c.Param("id"), c.Param("order_id"), input.TrackingCode)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}
