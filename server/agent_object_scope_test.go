package server

import (
	"bytes"
	"context"
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

	for _, test := range []struct {
		name, method, path, body string
		handler                  func(*gin.Context)
	}{
		{"project get", http.MethodGet, "/api/agent/projects/" + tenantProject.ID, "", api.getProject},
		{"project update", http.MethodPut, "/api/agent/projects/" + tenantProject.ID, `{"name":"changed","root":"` + tenantProject.Root + `"}`, api.updateProject},
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
