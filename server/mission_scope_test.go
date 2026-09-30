package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestMissionRoutesHideForeignMissionIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "foreign tenant objective must not be disclosed",
		OrganizationID: "org-b",
		Workspace:      workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	for _, handler := range []struct {
		name string
		call func(*gin.Context)
	}{
		{name: "mission", call: api.getMission},
		{name: "events", call: api.events},
	} {
		t.Run(handler.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+foreign.ID+"/"+handler.name, nil)
			ctx.Params = gin.Params{{Key: "id", Value: foreign.ID}}
			ctx.Set("agent.organization", agent.Organization{ID: "org-a"})
			handler.call(ctx)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s; foreign and unknown IDs must both be hidden", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), foreign.Objective) || strings.Contains(recorder.Body.String(), "outside the active organization") {
				t.Fatalf("foreign mission details or existence reason leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestRegisteredMissionRoutesBindBearerTokenToOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "router-scope-test-key-with-sufficient-length")
	auth, err := agent.NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	userA, err := auth.CreateUser("org-a@example.test", "Org A")
	if err != nil {
		t.Fatal(err)
	}
	orgA, _, err := auth.CreateOrganization("Org A", userA)
	if err != nil {
		t.Fatal(err)
	}
	userB, err := auth.CreateUser("org-b@example.test", "Org B")
	if err != nil {
		t.Fatal(err)
	}
	orgB, _, err := auth.CreateOrganization("Org B", userB)
	if err != nil {
		t.Fatal(err)
	}
	tokenA, _, err := auth.IssueToken(userA.ID, orgA.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokenB, _, err := auth.IssueToken(userB.ID, orgB.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: agent.NewMemoryStore(), Planner: agent.RulePlanner{}, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "private org A objective", OrganizationID: orgA.ID, Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, auth: auth, authRequired: true}
	router := gin.New()
	api.register(router)
	serve := func(method, path, token, organizationHeader, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		if organizationHeader != "" {
			request.Header.Set("X-Ollama-Organization", organizationHeader)
		}
		router.ServeHTTP(recorder, request)
		return recorder
	}
	paths := []struct {
		name, method, suffix, body string
	}{
		{"detail", http.MethodGet, "", ""},
		{"events", http.MethodGet, "/events", ""},
		{"event stream", http.MethodGet, "/events/stream", ""},
		{"traces", http.MethodGet, "/traces", ""},
		{"artifact", http.MethodGet, "/artifacts/art_unknown", ""},
		{"run", http.MethodPost, "/run", ""},
		{"cancel", http.MethodPost, "/cancel", ""},
		{"approval", http.MethodPost, "/approvals/apr_unknown", `{"decision":"reject","nonce":"nonce"}`},
	}
	for _, endpoint := range paths {
		t.Run(endpoint.name, func(t *testing.T) {
			foreignPath := "/api/agent/v1/missions/" + foreign.ID + endpoint.suffix
			unknownPath := "/api/agent/v1/missions/mis_unknown" + endpoint.suffix
			foreignResponse := serve(endpoint.method, foreignPath, tokenB, "", endpoint.body)
			unknownResponse := serve(endpoint.method, unknownPath, tokenB, "", endpoint.body)
			if foreignResponse.Code != http.StatusNotFound || unknownResponse.Code != http.StatusNotFound || foreignResponse.Body.String() != unknownResponse.Body.String() {
				t.Fatalf("foreign and unknown IDs differ: foreign=%d %s unknown=%d %s", foreignResponse.Code, foreignResponse.Body.String(), unknownResponse.Code, unknownResponse.Body.String())
			}
			if strings.Contains(foreignResponse.Body.String(), foreign.Objective) {
				t.Fatalf("foreign mission details leaked: %s", foreignResponse.Body.String())
			}
		})
	}
	mismatch := serve(http.MethodGet, "/api/agent/v1/missions/"+foreign.ID, tokenA, orgB.ID, "")
	if mismatch.Code != http.StatusForbidden {
		t.Fatalf("organization header mismatch status=%d body=%s", mismatch.Code, mismatch.Body.String())
	}
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+foreign.ID, nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing bearer token status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestGlobalMetricsHiddenWhenOrganizationAuthenticationIsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: agent.NewMemoryStore(), Planner: agent.RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	for _, handler := range []struct {
		name string
		call func(*gin.Context)
	}{
		{name: "json", call: api.metrics},
		{name: "prometheus", call: api.prometheus},
	} {
		t.Run(handler.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/metrics", nil)
			handler.call(ctx)
			if recorder.Code != http.StatusNotFound || recorder.Body.Len() != 0 {
				t.Fatalf("global metrics leaked through authenticated API: status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}
