package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func newObjectScopeTestAPI(t *testing.T) (*agentAPI, agent.Project, agent.Project) {
	t.Helper()
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	collaboration, err := agent.NewCollaborationStore("")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Context:       contextStore,
		Collaboration: collaboration,
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	projectA, err := contextStore.CreateProject("A", filepath.Join(root, "project-a"), "org_a")
	if err != nil {
		t.Fatal(err)
	}
	projectB, err := contextStore.CreateProject("B", filepath.Join(root, "project-b"), "org_b")
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{runtime: runtime, context: contextStore}, projectA, projectB
}

func scopedObjectRequest(t *testing.T, method, path, body, organizationID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	return ctx, recorder
}

func TestAgentObjectRoutesRejectCrossOrganizationProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, _, projectB := newObjectScopeTestAPI(t)

	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		handler func(*gin.Context)
	}{
		{name: "project read", method: http.MethodGet, path: "/api/agent/projects/" + projectB.ID, handler: api.getProject},
		{name: "memory write", method: http.MethodPost, path: "/api/agent/projects/" + projectB.ID + "/memories", body: `{"content":"private"}`, handler: api.addMemory},
		{name: "memory search", method: http.MethodGet, path: "/api/agent/projects/" + projectB.ID + "/memories", handler: api.searchMemories},
		{name: "project ingest", method: http.MethodPost, path: "/api/agent/projects/" + projectB.ID + "/ingest", body: `{}`, handler: api.ingestProject},
		{name: "collaboration snapshot", method: http.MethodGet, path: "/api/agent/projects/" + projectB.ID + "/collaboration", handler: api.collabSnapshot},
		{name: "collaboration comment", method: http.MethodPost, path: "/api/agent/projects/" + projectB.ID + "/collaboration/comments", body: `{"body":"private"}`, handler: api.collabComment},
		{name: "collaboration presence", method: http.MethodPost, path: "/api/agent/projects/" + projectB.ID + "/collaboration/presence", body: `{"status":"online"}`, handler: api.collabPresence},
		{name: "mission project", method: http.MethodPost, path: "/api/agent/missions", body: `{"objective":"use project","project_id":"` + projectB.ID + `"}`, handler: api.createMission},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, recorder := scopedObjectRequest(t, test.method, test.path, test.body, "org_a")
			if test.name == "project read" || test.name == "memory write" || test.name == "memory search" || test.name == "project ingest" {
				ctx.Params = gin.Params{{Key: "id", Value: projectB.ID}}
			} else if test.name != "mission project" {
				ctx.Params = gin.Params{{Key: "project_id", Value: projectB.ID}}
			}
			test.handler(ctx)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
			}
		})
	}
}

func TestAgentCreateMissionBindsWorkspaceToProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/missions", `{"objective":"use project","project_id":"`+projectA.ID+`","workspace":"/tmp/other"}`, "org_a")
	api.createMission(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	ctx, recorder = scopedObjectRequest(t, http.MethodPost, "/api/agent/missions", `{"objective":"use project","project_id":"`+projectA.ID+`"}`, "org_a")
	api.createMission(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
}

func TestUnauthenticatedLocalModeCannotAccessTenantContextRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, _, tenantProject := newObjectScopeTestAPI(t)
	t.Cleanup(func() { _ = api.runtime.Close(context.Background()) })
	localProject, err := api.context.CreateProject("local", filepath.Join(api.runtime.WorkspaceRoot(), "local-project"), agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := api.context.RegisterSkillForOrganization("org_b", agent.SkillManifest{ID: "tenant-skill", Version: "1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.context.RegisterSkillForOrganization(agent.LocalOrganizationID, agent.SkillManifest{ID: "local-skill", Version: "1"}); err != nil {
		t.Fatal(err)
	}
	tenantSchedule, err := api.context.CreateSchedule(agent.Schedule{ID: "sch_" + uuid.NewString(), Objective: "tenant schedule", IntervalSeconds: 60, OrganizationID: "org_b"})
	if err != nil {
		t.Fatal(err)
	}
	localSchedule, err := api.context.CreateSchedule(agent.Schedule{ID: "sch_" + uuid.NewString(), Objective: "local schedule", IntervalSeconds: 60, OrganizationID: agent.LocalOrganizationID})
	if err != nil {
		t.Fatal(err)
	}
	localRequest := func(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		return ctx, recorder
	}

	ctx, recorder := localRequest(http.MethodGet, "/api/agent/projects", "")
	api.projects(ctx)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), localProject.ID) || strings.Contains(recorder.Body.String(), tenantProject.ID) {
		t.Fatalf("local project list leaked tenant or omitted local project: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = localRequest(http.MethodGet, "/api/agent/skills", "")
	api.skills(ctx)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "local-skill") || strings.Contains(recorder.Body.String(), "tenant-skill") {
		t.Fatalf("local skill list leaked tenant or omitted local skill: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = localRequest(http.MethodGet, "/api/agent/schedules", "")
	api.schedules(ctx)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), localSchedule.ID) || strings.Contains(recorder.Body.String(), tenantSchedule.ID) {
		t.Fatalf("local schedule list leaked tenant or omitted local schedule: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	projectUpdateBody, err := json.Marshal(map[string]string{"name": "changed", "root": tenantProject.Root})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name, method, path, body string
		handler                  func(*gin.Context)
	}{
		{"project get", http.MethodGet, "/api/agent/projects/" + tenantProject.ID, "", api.getProject},
		{"project update", http.MethodPut, "/api/agent/projects/" + tenantProject.ID, string(projectUpdateBody), api.updateProject},
		{"project delete", http.MethodDelete, "/api/agent/projects/" + tenantProject.ID, "", api.deleteProject},
		{"memory add", http.MethodPost, "/api/agent/projects/" + tenantProject.ID + "/memories", `{"content":"private"}`, api.addMemory},
		{"memory search", http.MethodGet, "/api/agent/projects/" + tenantProject.ID + "/memories", "", api.searchMemories},
		{"schedule update", http.MethodPut, "/api/agent/schedules/" + tenantSchedule.ID, `{"objective":"changed","interval_seconds":60}`, api.updateSchedule},
		{"schedule delete", http.MethodDelete, "/api/agent/schedules/" + tenantSchedule.ID, "", api.deleteSchedule},
		{"tenant webhook", http.MethodPost, "/api/agent/schedules/" + tenantSchedule.ID + "/webhook", `{}`, api.webhook},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, recorder := localRequest(test.method, test.path, test.body)
			if strings.Contains(test.path, "/projects/") {
				ctx.Params = gin.Params{{Key: "id", Value: tenantProject.ID}}
			} else if strings.Contains(test.path, "/schedules/") {
				param := "id"
				if test.name == "tenant webhook" {
					param = "schedule_id"
				}
				ctx.Params = gin.Params{{Key: param, Value: tenantSchedule.ID}}
			}
			test.handler(ctx)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
			}
		})
	}
}

func TestAuthDisabledAlwaysUsesLocalOrganizationAcrossStores(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	companies, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	connectors := agent.NewConnectorManager()
	root := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Context: contextStore, Company: companies, Connectors: connectors,
		Planner: agent.RulePlanner{}, WorkspaceRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })
	api := &agentAPI{runtime: runtime, context: contextStore, authRequired: false}

	tenantCompany, err := companies.Create(agent.Company{OrganizationID: "org_b", Name: "tenant-company"})
	if err != nil {
		t.Fatal(err)
	}
	if err := connectors.Register(agent.ConnectorConfig{
		ID: "tenant-connector", OrganizationID: "org_b", Provider: "private",
		BaseURL:    "https://tenant.example.test",
		Operations: []agent.ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}},
	}); err != nil {
		t.Fatal(err)
	}
	tenantSchedule, err := contextStore.CreateSchedule(agent.Schedule{ID: "sch_" + uuid.NewString(), OrganizationID: "org_b", Objective: "tenant-only", IntervalSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	tenantMission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "tenant-only", Workspace: root, OrganizationID: "org_b"})
	if err != nil {
		t.Fatal(err)
	}

	localRequest := func(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		// A client-controlled organization header must be ignored in local mode.
		ctx.Request.Header.Set("X-Ollama-Organization", "org_b")
		return ctx, recorder
	}

	ctx, recorder := localRequest(http.MethodGet, "/api/agent/missions", "")
	api.authMiddleware(ctx)
	api.missions(ctx)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), tenantMission.ID) {
		t.Fatalf("local missions leaked tenant data: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = localRequest(http.MethodGet, "/api/agent/companies", "")
	api.authMiddleware(ctx)
	api.companies(ctx)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), tenantCompany.ID) {
		t.Fatalf("local companies leaked tenant data: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = localRequest(http.MethodGet, "/api/agent/connectors", "")
	api.authMiddleware(ctx)
	api.connectors(ctx)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "tenant-connector") {
		t.Fatalf("local connectors leaked tenant data: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = localRequest(http.MethodGet, "/api/agent/schedules", "")
	api.authMiddleware(ctx)
	api.schedules(ctx)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), tenantSchedule.ID) {
		t.Fatalf("local schedules leaked tenant data: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	localMissionBody, err := json.Marshal(map[string]string{
		"objective":       "must stay local",
		"workspace":       root,
		"organization_id": "org_b",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, recorder = localRequest(http.MethodPost, "/api/agent/missions", string(localMissionBody))
	api.authMiddleware(ctx)
	api.createMission(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("local mission write status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var created agent.Mission
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.OrganizationID != agent.LocalOrganizationID {
		t.Fatalf("local write accepted foreign organization: got %q", created.OrganizationID)
	}

	ctx, recorder = localRequest(http.MethodGet, "/api/agent/companies/"+tenantCompany.ID, "")
	ctx.Params = gin.Params{{Key: "id", Value: tenantCompany.ID}}
	api.authMiddleware(ctx)
	api.getCompany(ctx)
	if recorder.Code != http.StatusForbidden && recorder.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant company read status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
