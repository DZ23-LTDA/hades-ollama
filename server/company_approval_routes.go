package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// approvalDecision is an explicit command, not a client-controlled projection.
// The server applies it only after validating actor, tenant, nonce, expiry and CAS.
func approvalDecision(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "approve":
		return true, nil
	case "reject":
		return false, nil
	default:
		return false, errors.New("decision must be approve or reject")
	}
}

func (a *agentAPI) decideCompanyApproval(c *gin.Context, resourceType, resourceParam string) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !a.requireApprovalApprover(c) {
		return
	}
	var request struct {
		Decision string `json:"decision"`
		Nonce    string `json:"nonce"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approved, err := approvalDecision(request.Decision)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approval, err := a.runtime.CompanyStore().PendingApproval(c.Param("id"), resourceType, c.Param(resourceParam))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	organizationID := agentOrganizationID(c)
	if organizationID == "" {
		organizationID = company.OrganizationID
	}
	updated, err := a.runtime.CompanyStore().DecideApproval(c.Param("id"), approval.ID, approved, request.Reason, agentActorID(c), organizationID, company.Version, request.Nonce)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (a *agentAPI) decideCompanyApprovalByID(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !a.requireApprovalApprover(c) {
		return
	}
	var request struct {
		Decision string `json:"decision"`
		Nonce    string `json:"nonce"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approved, err := approvalDecision(request.Decision)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	organizationID := agentOrganizationID(c)
	if organizationID == "" {
		organizationID = company.OrganizationID
	}
	updated, err := a.runtime.CompanyStore().DecideApproval(c.Param("id"), c.Param("approval_id"), approved, request.Reason, agentActorID(c), organizationID, company.Version, request.Nonce)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, updated)
}
