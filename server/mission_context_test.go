package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/version"
)

func newMissionContextFixture(t *testing.T, organization string) (*agentAPI, agent.Mission, agent.Project) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	store := agent.NewMemoryStore()
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         store,
		Context:       contextStore,
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	project, err := contextStore.CreateProject("projeto e2e", workspace, organization)
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "corrigir falha de estoque",
		OrganizationID: organization,
		ProjectID:      project.ID,
		Workspace:      workspace,
		Capabilities:   []string{"workspace:read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{runtime: runtime, authRequired: true, context: contextStore}, mission, project
}

// TestMissionContextReturnsServerAuthorizedBootstrap prova que a rota devolve
// o Context Bootstrap da missão com permissões DECLARADAS (nunca concedidas por
// este endpoint), memória redigida do projeto e nenhum identificador no
// preâmbulo que vai ao modelo.
func TestMissionContextReturnsServerAuthorizedBootstrap(t *testing.T) {
	api, mission, project := newMissionContextFixture(t, "org-a")

	rawKey := "sk-live-abcdefghijklmnopqrstuvwxyz012345"
	if _, err := api.context.AddMemory(agent.Memory{
		ID:        "mem-1",
		ProjectID: project.ID,
		Kind:      "fact",
		Content:   "corrigir falha de estoque: causa raiz conhecida OPENAI_API_KEY=" + rawKey,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+mission.ID+"/context", nil)
	ctx.Params = gin.Params{{Key: "id", Value: mission.ID}}
	ctx.Set("agent.organization", agent.Organization{ID: "org-a"})
	api.missionContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), rawKey) {
		t.Fatal("context route leaked a credential-shaped value")
	}
	var bootstrap agent.ContextBootstrap
	if err := json.Unmarshal(recorder.Body.Bytes(), &bootstrap); err != nil {
		t.Fatalf("decode bootstrap: %v body=%s", err, recorder.Body.String())
	}
	if bootstrap.Identity != "Hades" || bootstrap.Version != version.Version {
		t.Fatalf("identity/version = %q/%q", bootstrap.Identity, bootstrap.Version)
	}
	if bootstrap.MissionID != mission.ID || bootstrap.ProjectID != project.ID {
		t.Fatalf("scope = %+v", bootstrap)
	}
	if got := bootstrap.Permissions["workspace:read"]; got != agent.PermissionRead {
		t.Errorf("workspace:read = %q, want %q", got, agent.PermissionRead)
	}
	if got := bootstrap.Permissions["workspace:write"]; got != agent.PermissionForbidden {
		t.Errorf("workspace:write = %q, want %q", got, agent.PermissionForbidden)
	}
	if got := bootstrap.Permissions["mcp:call"]; got != agent.PermissionForbidden {
		t.Errorf("mcp:call = %q, want %q", got, agent.PermissionForbidden)
	}
	if got := bootstrap.Permissions["escopo:inventado"]; got != "" {
		// A criação da missão já recusa capacidade desconhecida
		// (`unknown mission capability`), então este caminho é impossível pela
		// API — a asserção só garante que nada inventado apareceu no documento.
		t.Errorf("invented scope reached the bootstrap: %q", got)
	}
	if len(bootstrap.MemoryRefs) != 1 || bootstrap.MemoryRefs[0].ID != "mem-1" {
		t.Fatalf("memory refs = %+v", bootstrap.MemoryRefs)
	}
	if !strings.Contains(bootstrap.MemoryRefs[0].Snippet, "causa raiz conhecida") {
		t.Fatalf("memory snippet lost its content: %q", bootstrap.MemoryRefs[0].Snippet)
	}
	for _, secret := range []string{mission.ID, project.ID, mission.Workspace} {
		if secret != "" && strings.Contains(bootstrap.SystemPreamble, secret) {
			t.Errorf("preamble leaked %q", secret)
		}
	}
	if !strings.Contains(bootstrap.SystemPreamble, "Permissões declaradas pelo servidor") {
		t.Error("preamble must declare the permission table")
	}
}

// TestMissionContextRespectsOrganizationIsolation garante que a rota não
// entrega contexto de outra organização: o handler reusa o isolamento do
// `missionForRequest`, que responde 404 fora do tenant autenticado.
func TestMissionContextRespectsOrganizationIsolation(t *testing.T) {
	api, mission, _ := newMissionContextFixture(t, "org-b")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+mission.ID+"/context", nil)
	ctx.Params = gin.Params{{Key: "id", Value: mission.ID}}
	ctx.Set("agent.organization", agent.Organization{ID: "org-a"})
	api.missionContext(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant context status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), mission.ID) {
		t.Fatal("cross-tenant response echoed the mission identifier")
	}
}

// TestMissionContextWithoutProjectSkipsMemory confirma que uma missão sem
// projeto não recebe memória nenhuma — na dúvida, não se injeta contexto.
func TestMissionContextWithoutProjectSkipsMemory(t *testing.T) {
	api, mission, project := newMissionContextFixture(t, "org-a")
	if _, err := api.context.AddMemory(agent.Memory{
		ID: "mem-1", ProjectID: project.ID, Kind: "fact", Content: "falha de estoque",
	}); err != nil {
		t.Fatal(err)
	}
	missionWithoutProject, err := api.runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "corrigir falha de estoque",
		OrganizationID: "org-a",
		Workspace:      mission.Workspace,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+missionWithoutProject.ID+"/context", nil)
	ctx.Params = gin.Params{{Key: "id", Value: missionWithoutProject.ID}}
	ctx.Set("agent.organization", agent.Organization{ID: "org-a"})
	api.missionContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var bootstrap agent.ContextBootstrap
	if err := json.Unmarshal(recorder.Body.Bytes(), &bootstrap); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	if len(bootstrap.MemoryRefs) != 0 {
		t.Fatalf("mission without project must not receive memory: %+v", bootstrap.MemoryRefs)
	}
	if strings.Contains(bootstrap.SystemPreamble, "Memória autorizada") {
		t.Fatal("preamble must not announce memory that was not injected")
	}
}
