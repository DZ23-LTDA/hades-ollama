package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestCompanyTelAgentEnforcesRoleAndReturnsHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := companies.Create(agent.Company{OrganizationID: "org-a", Name: "Tel Agent Co"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: companies, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	path := "/companies/" + company.ID + "/tel-agent"

	viewer, viewerRecorder := orgContext(t, http.MethodPost, path, "org-a")
	viewer.Params = gin.Params{{Key: "id", Value: company.ID}}
	viewer.Set("agent.membership", agent.Membership{Role: agent.RoleViewer, OrganizationID: "org-a"})
	viewer.Set("agent.user", agent.User{ID: "viewer-a"})
	viewer.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"message":"create","operation":"backlog.create","title":"blocked"}`))
	api.companyTelAgent(viewer)
	if viewerRecorder.Code != http.StatusForbidden {
		t.Fatalf("viewer status=%d body=%s", viewerRecorder.Code, viewerRecorder.Body.String())
	}

	operator, operatorRecorder := orgContext(t, http.MethodPost, path, "org-a")
	operator.Params = gin.Params{{Key: "id", Value: company.ID}}
	operator.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
	operator.Set("agent.user", agent.User{ID: "operator-a"})
	operator.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"message":"create","operation":"backlog.create","title":"Prepare launch","priority":10}`))
	operator.Request.Header.Set("Idempotency-Key", "http-tel-agent-1")
	api.companyTelAgent(operator)
	if operatorRecorder.Code != http.StatusOK {
		t.Fatalf("operator status=%d body=%s", operatorRecorder.Code, operatorRecorder.Body.String())
	}
	var response struct {
		Company  agent.Company          `json:"company"`
		Exchange agent.TelAgentExchange `json:"exchange"`
	}
	if err := json.Unmarshal(operatorRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Exchange.CreatedResourceID == "" || len(response.Company.Backlog) != 1 {
		t.Fatalf("unexpected response=%+v", response)
	}
	replay, replayRecorder := orgContext(t, http.MethodPost, path, "org-a")
	replay.Params = gin.Params{{Key: "id", Value: company.ID}}
	replay.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
	replay.Set("agent.user", agent.User{ID: "operator-a"})
	replay.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"message":"create","operation":"backlog.create","title":"Prepare launch","priority":10}`))
	replay.Request.Header.Set("Idempotency-Key", "http-tel-agent-1")
	api.companyTelAgent(replay)
	if replayRecorder.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}
	conflict, conflictRecorder := orgContext(t, http.MethodPost, path, "org-a")
	conflict.Params = gin.Params{{Key: "id", Value: company.ID}}
	conflict.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
	conflict.Set("agent.user", agent.User{ID: "operator-a"})
	conflict.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"message":"create","operation":"backlog.create","title":"Different","priority":10}`))
	conflict.Request.Header.Set("Idempotency-Key", "http-tel-agent-1")
	api.companyTelAgent(conflict)
	if conflictRecorder.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflictRecorder.Code, conflictRecorder.Body.String())
	}

	history, historyRecorder := orgContext(t, http.MethodGet, "/companies/"+company.ID+"/tel-agent/history", "org-a")
	history.Params = gin.Params{{Key: "id", Value: company.ID}}
	history.Set("agent.membership", agent.Membership{Role: agent.RoleViewer, OrganizationID: "org-a"})
	history.Set("agent.user", agent.User{ID: "viewer-a"})
	history.Request = httptest.NewRequest(http.MethodGet, "/companies/"+company.ID+"/tel-agent/history", nil)
	api.companyTelAgentHistory(history)
	if historyRecorder.Code != http.StatusOK || len(historyRecorder.Body.Bytes()) == 0 {
		t.Fatalf("history status=%d body=%s", historyRecorder.Code, historyRecorder.Body.String())
	}

	crossTenant, crossRecorder := orgContext(t, http.MethodGet, "/companies/"+company.ID+"/tel-agent/history", "org-b")
	crossTenant.Params = gin.Params{{Key: "id", Value: company.ID}}
	crossTenant.Set("agent.membership", agent.Membership{Role: agent.RoleViewer, OrganizationID: "org-b"})
	crossTenant.Request = httptest.NewRequest(http.MethodGet, "/companies/"+company.ID+"/tel-agent/history", nil)
	api.companyTelAgentHistory(crossTenant)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross tenant status=%d body=%s", crossRecorder.Code, crossRecorder.Body.String())
	}
}
