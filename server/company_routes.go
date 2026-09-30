package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func companyOrganizationID(c *gin.Context) string {
	if organizationID := agentOrganizationID(c); organizationID != "" {
		return organizationID
	}
	return "local"
}

func (a *agentAPI) companyForRequest(c *gin.Context) (agent.Company, error) {
	company, err := a.runtime.CompanyStore().Get(c.Param("id"))
	if err != nil {
		return agent.Company{}, err
	}
	if organizationID := companyOrganizationID(c); company.OrganizationID != organizationID {
		return agent.Company{}, errAgentForbidden
	}
	return company, nil
}

func (a *agentAPI) companies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"companies": a.runtime.CompanyStore().List(companyOrganizationID(c))})
}

func (a *agentAPI) createCompany(c *gin.Context) {
	var input agent.CompanyCreateRequest
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().CreateRequest(input, companyOrganizationID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) getCompany(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) updateCompany(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyUpdate
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().Update(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) companyReport(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	report, err := a.runtime.CompanyStore().Report(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (a *agentAPI) companyTelAgent(c *gin.Context) {
	_, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request agent.TelAgentRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	request.IdempotencyKey = c.GetHeader("Idempotency-Key")
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
	if operation != "report.read" && !a.companyTelAgentWriteAllowed(c) {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	organizationID := companyOrganizationID(c)
	updated, result, err := a.runtime.CompanyStore().ExecuteTelAgent(c.Param("id"), organizationID, agentActorID(c), request)
	if err != nil {
		status := statusForAgentError(err)
		if errors.Is(err, agent.ErrTelAgentOrganizationMismatch) {
			status = http.StatusForbidden
		} else if errors.Is(err, agent.ErrCompanyIdempotentReplay) || errors.Is(err, agent.ErrCompanyIdempotencyConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, agent.ErrTelAgentActorRequired) || errors.Is(err, agent.ErrTelAgentMessageRequired) || errors.Is(err, agent.ErrTelAgentMessageTooLong) || errors.Is(err, agent.ErrTelAgentUnsupportedOperation) || errors.Is(err, agent.ErrTelAgentIdempotencyKeyTooLong) {
			status = http.StatusBadRequest
		}
		writeAgentError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"company": updated, "exchange": result.Exchange, "report": result.Report})
}

func (a *agentAPI) companyTelAgentHistory(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	history, err := a.runtime.CompanyStore().TelAgentHistoryForOrganization(c.Param("id"), companyOrganizationID(c), 100)
	if err != nil {
		status := statusForAgentError(err)
		if errors.Is(err, agent.ErrTelAgentOrganizationMismatch) {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history, "channel": "tel-agent.text", "telephony": "not_configured"})
}

func (a *agentAPI) companyTelAgentWriteAllowed(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, ok := c.Get("agent.membership")
	if !ok {
		return false
	}
	membership, ok := value.(agent.Membership)
	if !ok {
		return false
	}
	switch membership.Role {
	case agent.RoleOwner, agent.RoleAdmin, agent.RoleOperator:
		return true
	default:
		return false
	}
}

func (a *agentAPI) addCompanyRoadmap(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyRoadmapItem
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddRoadmap(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) addCompanyGoal(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyGoal
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddGoal(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) addCompanyBacklog(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CompanyBacklogItem
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().AddBacklog(c.Param("id"), input)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, company)
}

func (a *agentAPI) addCompanyCycle(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if company.Status == agent.CompanyPaused || company.Risk.Paused {
		writeAgentError(c, http.StatusLocked, agent.ErrCompanyPaused)
		return
	}
	var input agent.CompanyCycle
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	created, cycle, replayed, err := a.runtime.CompanyStore().AddCycleWithIdempotency(c.Param("id"), input, c.GetHeader("Idempotency-Key"))
	if err != nil {
		if errors.Is(err, agent.ErrCompanyIdempotencyConflict) {
			writeAgentError(c, http.StatusConflict, err)
			return
		}
		if errors.Is(err, agent.ErrCompanyCycleIdempotencyKeyTooLong) {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if replayed {
		if cycle.ScheduleID == "" {
			writeAgentError(c, http.StatusConflict, agent.ErrCompanyCycleReplayMissing)
			return
		}
		c.JSON(http.StatusOK, created)
		return
	}
	if cycle.ID == "" {
		writeAgentError(c, http.StatusInternalServerError, errors.New("company cycle was not persisted"))
		return
	}
	schedule, err := a.context.CreateSchedule(agent.Schedule{
		Objective:       "Company OS / " + company.Name + ": " + cycle.Objective,
		Model:           "",
		Workspace:       "company://" + company.ID,
		OrganizationID:  company.OrganizationID,
		IntervalSeconds: cycle.IntervalSeconds,
		Enabled:         true,
		NextRunAt:       cycle.NextRunAt,
	})
	if err != nil {
		_, _ = a.runtime.CompanyStore().RemoveCycle(c.Param("id"), cycle.ID)
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	updated, err := a.runtime.CompanyStore().SetCycleSchedule(c.Param("id"), cycle.ID, schedule.ID)
	if err != nil {
		_ = a.context.DeleteSchedule(schedule.ID)
		_, _ = a.runtime.CompanyStore().RemoveCycle(c.Param("id"), cycle.ID)
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, updated)
}

func (a *agentAPI) pauseCompany(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	company, err := a.runtime.CompanyStore().Pause(c.Param("id"), input.Reason)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) resumeCompany(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	company, err := a.runtime.CompanyStore().Resume(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) recordCompanyAnomaly(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Severity string `json:"severity"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().RecordAnomaly(c.Param("id"), input.Severity, input.Reason)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func (a *agentAPI) recordCompanySpend(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Category    string `json:"category"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	company, err := a.runtime.CompanyStore().RecordSpendRequestWithIdempotency(c.Param("id"), input.Category, input.AmountCents, c.GetHeader("Idempotency-Key"))
	if err != nil {
		if errors.Is(err, agent.ErrCompanyIdempotentReplay) {
			c.JSON(http.StatusOK, company)
			return
		}
		if errors.Is(err, agent.ErrCompanySpendApprovalPending) {
			for _, pending := range company.Approvals {
				if pending.ResourceType == "spend" && pending.Status == agent.CompanyApprovalPending && pending.Category == input.Category && pending.AmountCents == input.AmountCents {
					_ = a.runtime.CompanyStore().BindApprovalRequester(c.Param("id"), pending.ID, agentActorID(c))
					break
				}
			}
			c.JSON(http.StatusAccepted, company)
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

func companyErrorStatus(err error) int {
	switch {
	case errors.Is(err, agent.ErrCompanyNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrCompanyBudgetExceeded), errors.Is(err, agent.ErrCompanyAgentBudgetExceeded), errors.Is(err, agent.ErrCompanyApprovalRequired), errors.Is(err, agent.ErrCompanyApprovalConflict), errors.Is(err, agent.ErrCompanyApprovalNonce), errors.Is(err, agent.ErrCompanyApprovalSelf), errors.Is(err, agent.ErrCompanySpendApprovalPending), errors.Is(err, agent.ErrCompanyIdempotencyConflict):
		return http.StatusConflict
	case errors.Is(err, agent.ErrCompanyCampaignNotFound), errors.Is(err, agent.ErrCompanyAffiliateNotFound), errors.Is(err, agent.ErrCompanyProductNotFound), errors.Is(err, agent.ErrCompanyOrderNotFound), errors.Is(err, agent.ErrCompanyApprovalNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrCompanySocialNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrCompanyAgentNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrCompanyApprovalRequiredForExternal):
		return http.StatusConflict
	case errors.Is(err, agent.ErrCompanySocialOAuthRequired), errors.Is(err, agent.ErrCompanySocialExternalUnavailable):
		return http.StatusConflict
	case errors.Is(err, agent.ErrCompanyInvalidExternalURL):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrCompanySocialProviderUnsupported):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrCompanyPaused):
		return http.StatusLocked
	case strings.Contains(strings.ToLower(err.Error()), "company"):
		return http.StatusBadRequest
	default:
		return 0
	}
}
