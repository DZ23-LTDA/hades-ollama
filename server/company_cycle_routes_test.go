package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestCompanyCycleHTTPIsIdempotentAndDoesNotDuplicateSchedule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	company, err := companies.Create(agent.Company{OrganizationID: "org-a", Name: "Cycle HTTP"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: companies, Context: contextStore, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore, authRequired: true}
	path := "/companies/" + company.ID + "/cycles"
	body := `{"name":"Daily","objective":"Review backlog","frequency":"daily","interval_seconds":86400}`
	request := func(payload string) (*gin.Context, *httptest.ResponseRecorder) {
		ctx, recorder := orgContext(t, http.MethodPost, path, "org-a")
		ctx.Params = gin.Params{{Key: "id", Value: company.ID}}
		ctx.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
		ctx.Set("agent.user", agent.User{ID: "operator-a"})
		ctx.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(payload))
		ctx.Request.Header.Set("Idempotency-Key", "cycle-http-key")
		return ctx, recorder
	}
	first, firstRecorder := request(body)
	api.addCompanyCycle(first)
	if firstRecorder.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", firstRecorder.Code, firstRecorder.Body.String())
	}
	var created agent.Company
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(created.Cycles) != 1 || created.Cycles[0].ScheduleID == "" || len(contextStore.ListSchedulesForOrganization("org-a")) != 1 {
		t.Fatalf("first cycle/schedule=%+v schedules=%+v", created.Cycles, contextStore.ListSchedulesForOrganization("org-a"))
	}
	replay, replayRecorder := request(body)
	api.addCompanyCycle(replay)
	if replayRecorder.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}
	if got := contextStore.ListSchedulesForOrganization("org-a"); len(got) != 1 {
		t.Fatalf("replay duplicated schedules=%+v", got)
	}
	conflict, conflictRecorder := request(`{"name":"Different","objective":"Review backlog","frequency":"daily","interval_seconds":86400}`)
	api.addCompanyCycle(conflict)
	if conflictRecorder.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflictRecorder.Code, conflictRecorder.Body.String())
	}
	if got, err := companies.Get(company.ID); err != nil || len(got.Cycles) != 1 {
		t.Fatalf("conflict changed cycles=%+v err=%v", got, err)
	}
}

func TestCompanyCycleHTTPRollsBackWhenSchedulePersistenceFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	contextRoot := t.TempDir()
	contextStore, err := agent.NewContextStore(contextRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(contextRoot, "schedules")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextRoot, "schedules"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	company, err := companies.Create(agent.Company{OrganizationID: "org-a", Name: "Rollback HTTP"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: companies, Context: contextStore, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore, authRequired: true}
	path := "/companies/" + company.ID + "/cycles"
	ctx, recorder := orgContext(t, http.MethodPost, path, "org-a")
	ctx.Params = gin.Params{{Key: "id", Value: company.ID}}
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleOperator, OrganizationID: "org-a"})
	ctx.Set("agent.user", agent.User{ID: "operator-a"})
	ctx.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"name":"Daily","objective":"Review backlog","interval_seconds":60}`))
	ctx.Request.Header.Set("Idempotency-Key", "rollback-key")
	api.addCompanyCycle(ctx)
	if recorder.Code == http.StatusCreated {
		t.Fatalf("rollback request unexpectedly succeeded: %s", recorder.Body.String())
	}
	updated, err := companies.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Cycles) != 0 || len(updated.Idempotency) != 0 || len(contextStore.ListSchedules()) != 0 {
		t.Fatalf("orphaned cycle/schedule after failure: company=%+v schedules=%+v", updated, contextStore.ListSchedules())
	}
}
