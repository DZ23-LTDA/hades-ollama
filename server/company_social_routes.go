package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) companySocialReport(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	report, err := a.runtime.CompanyStore().SocialReport(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (a *agentAPI) addCompanySocialAccount(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanySocialAccount
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddSocialAccount(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) createCompanySocialDraft(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanySocialDraft
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().CreateSocialDraft(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if len(company.SocialDrafts) > 0 {
		if approval, approvalErr := a.runtime.CompanyStore().PendingApproval(c.Param("id"), "social_draft", company.SocialDrafts[len(company.SocialDrafts)-1].ID); approvalErr == nil {
			_ = a.runtime.CompanyStore().BindApprovalRequester(c.Param("id"), approval.ID, agentActorID(c))
		}
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) approveCompanySocialDraft(c *gin.Context) {
	a.decideCompanyApproval(c, "social_draft", "draft_id")
}

func (a *agentAPI) publishCompanySocialDraft(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	company, err := a.runtime.CompanyStore().PublishSocialDraft(c.Param("id"), c.Param("draft_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) recordCompanySocialMetric(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanySocialMetric
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().RecordSocialMetricWithIdempotency(c.Param("id"), input, c.GetHeader("Idempotency-Key"))
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
