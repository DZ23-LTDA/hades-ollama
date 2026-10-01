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

func TestCompanyCampaignApprovalHTTPUsesNonceAndOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := companies.Create(agent.Company{OrganizationID: "org-a", Name: "HTTP Company"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = companies.AddCampaign(company.ID, agent.CompanyCampaign{Name: "Campaign", Objective: "Objective"})
	if err != nil {
		t.Fatal(err)
	}
	campaignID := company.Campaigns[0].ID
	approval, err := companies.PendingApproval(company.ID, "campaign", campaignID)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: companies, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	ctx, recorder := orgContext(t, http.MethodPost, "/companies/"+company.ID+"/campaigns/"+campaignID+"/approve", "org-a")
	ctx.Params = gin.Params{{Key: "id", Value: company.ID}, {Key: "campaign_id", Value: campaignID}}
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	ctx.Set("agent.user", agent.User{ID: "admin-a"})
	ctx.Request = httptest.NewRequest(http.MethodPost, "/companies/"+company.ID+"/campaigns/"+campaignID+"/approve", bytes.NewBufferString(`{"decision":"approve","nonce":"`+approval.Nonce+`","reason":"approved by policy"}`))
	api.approveCompanyCampaign(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("approval status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var updated agent.Company
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if !updated.Campaigns[0].Approved || updated.Approvals[0].ActorID != "admin-a" {
		t.Fatalf("updated company=%+v", updated)
	}

	replay, replayRecorder := orgContext(t, http.MethodPost, "/companies/"+company.ID+"/campaigns/"+campaignID+"/approve", "org-a")
	replay.Params = ctx.Params
	replay.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	replay.Set("agent.user", agent.User{ID: "admin-a"})
	replay.Request = httptest.NewRequest(http.MethodPost, "/companies/"+company.ID+"/campaigns/"+campaignID+"/approve", bytes.NewBufferString(`{"decision":"approve","nonce":"`+approval.Nonce+`","reason":"replay"}`))
	api.approveCompanyCampaign(replay)
	if replayRecorder.Code != http.StatusNotFound {
		t.Fatalf("replay status=%d body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}
}

func TestCompanySpendHTTPRequiresDecisionBeforeDebit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := companies.Create(agent.Company{OrganizationID: "org-a", Name: "Spend HTTP", Budget: agent.CompanyBudget{MonthlyLimitCents: 10000, ApprovalThresholdCents: 1000}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: companies, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	spend, spendRecorder := orgContext(t, http.MethodPost, "/companies/"+company.ID+"/spend", "org-a")
	spend.Params = gin.Params{{Key: "id", Value: company.ID}}
	spend.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
	spend.Set("agent.user", agent.User{ID: "operator-a"})
	spend.Request = httptest.NewRequest(http.MethodPost, "/companies/"+company.ID+"/spend", bytes.NewBufferString(`{"category":"ads","amount_cents":2500}`))
	api.recordCompanySpend(spend)
	if spendRecorder.Code != http.StatusAccepted {
		t.Fatalf("spend status=%d body=%s", spendRecorder.Code, spendRecorder.Body.String())
	}
	created, err := companies.Get(company.ID)
	if err != nil || created.Budget.SpentCents != 0 || len(created.Approvals) != 1 {
		t.Fatalf("spend was not pending: company=%+v err=%v", created, err)
	}
	approval := created.Approvals[0]
	decide, decideRecorder := orgContext(t, http.MethodPost, "/companies/"+company.ID+"/approvals/"+approval.ID+"/decide", "org-a")
	decide.Params = gin.Params{{Key: "id", Value: company.ID}, {Key: "approval_id", Value: approval.ID}}
	decide.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	decide.Set("agent.user", agent.User{ID: "admin-a"})
	decide.Request = httptest.NewRequest(http.MethodPost, "/companies/"+company.ID+"/approvals/"+approval.ID+"/decide", bytes.NewBufferString(`{"decision":"approve","nonce":"`+approval.Nonce+`","reason":"approved by policy"}`))
	api.decideCompanyApprovalByID(decide)
	if decideRecorder.Code != http.StatusOK {
		t.Fatalf("decide status=%d body=%s", decideRecorder.Code, decideRecorder.Body.String())
	}
	final, err := companies.Get(company.ID)
	if err != nil || final.Budget.SpentCents != 2500 {
		t.Fatalf("approved spend not debited atomically: company=%+v err=%v", final, err)
	}
}
