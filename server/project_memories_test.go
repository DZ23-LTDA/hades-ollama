package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func newMemoryGovernanceAPI(t *testing.T, organization string) (*agentAPI, agent.Project) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto memoria", t.TempDir(), organization)
	if err != nil {
		t.Fatal(err)
	}
	older := time.Now().UTC().Add(-60 * 24 * time.Hour)
	for _, memory := range []agent.Memory{
		{ID: "mem-old", ProjectID: project.ID, Kind: "fact", Content: "memória antiga", Source: "import", Confidence: 0.4, CreatedAt: older},
		{ID: "mem-new", ProjectID: project.ID, Kind: "fact", Content: "memória recente", Source: "chat", Confidence: 0.9, CreatedAt: time.Now().UTC()},
	} {
		if _, err := store.AddMemory(memory); err != nil {
			t.Fatal(err)
		}
	}
	return &agentAPI{context: store, authRequired: true}, project
}

// memoryContext monta o contexto gin equivalente ao que o middleware produz:
// projeto na rota e organização AUTENTICADA no contexto (nunca no corpo).
func memoryContext(t *testing.T, method, path, body, projectID, memoryID, organization string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	params := gin.Params{{Key: "id", Value: projectID}}
	if memoryID != "" {
		params = append(params, gin.Param{Key: "memory_id", Value: memoryID})
	}
	ctx.Params = params
	ctx.Set("agent.organization", agent.Organization{ID: organization})
	return ctx, recorder
}

func TestExportProjectMemoriesReturnsProvenanceWithoutEmbeddings(t *testing.T) {
	api, project := newMemoryGovernanceAPI(t, "org-a")

	ctx, recorder := memoryContext(t, http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/export", "", project.ID, "", "org-a")
	api.exportProjectMemories(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Count             int            `json:"count"`
		IncludeEmbeddings bool           `json:"include_embeddings"`
		Memories          []agent.Memory `json:"memories"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export: %v body=%s", err, recorder.Body.String())
	}
	if payload.Count != 2 || len(payload.Memories) != 2 {
		t.Fatalf("export payload = %+v", payload)
	}
	if payload.IncludeEmbeddings {
		t.Fatal("embeddings must be opt-in")
	}
	if payload.Memories[0].ID != "mem-old" || payload.Memories[1].ID != "mem-new" {
		t.Fatalf("export must be chronological: %+v", payload.Memories)
	}
	if payload.Memories[0].Source != "import" || payload.Memories[1].Confidence != 0.9 {
		t.Fatalf("provenance missing from export: %+v", payload.Memories)
	}

	badCtx, badRecorder := memoryContext(t, http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/export?include_embeddings=maybe", "", project.ID, "", "org-a")
	api.exportProjectMemories(badCtx)
	if badRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid include_embeddings status=%d", badRecorder.Code)
	}
}

func TestDeleteProjectMemoryIsScopedAndReportsMissing(t *testing.T) {
	api, project := newMemoryGovernanceAPI(t, "org-a")

	ctx, recorder := memoryContext(t, http.MethodDelete, "/api/agent/v1/projects/"+project.ID+"/memories/mem-new", "", project.ID, "mem-new", "org-a")
	api.deleteProjectMemory(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	remaining := api.context.ExportMemories(project.ID, false)
	if len(remaining) != 1 || remaining[0].ID != "mem-old" {
		t.Fatalf("remaining = %+v", remaining)
	}

	missingCtx, missingRecorder := memoryContext(t, http.MethodDelete, "/api/agent/v1/projects/"+project.ID+"/memories/nao-existe", "", project.ID, "nao-existe", "org-a")
	api.deleteProjectMemory(missingCtx)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("missing memory status=%d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestDeleteProjectMemoryRejectsOtherOrganization(t *testing.T) {
	api, project := newMemoryGovernanceAPI(t, "org-b")

	ctx, recorder := memoryContext(t, http.MethodDelete, "/api/agent/v1/projects/"+project.ID+"/memories/mem-new", "", project.ID, "mem-new", "org-a")
	api.deleteProjectMemory(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := len(api.context.ExportMemories(project.ID, false)); got != 2 {
		t.Fatalf("cross-tenant delete mutated the project: %d memories", got)
	}
}

func TestPruneProjectMemoriesValidatesWindowAndRemovesOld(t *testing.T) {
	api, project := newMemoryGovernanceAPI(t, "org-a")

	for _, body := range []string{`{"older_than_days":0}`, `{"older_than_days":4000}`, `{}`} {
		ctx, recorder := memoryContext(t, http.MethodPost, "/api/agent/v1/projects/"+project.ID+"/memories/prune", body, project.ID, "", "org-a")
		api.pruneProjectMemories(ctx)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %s status=%d", body, recorder.Code)
		}
	}

	ctx, recorder := memoryContext(t, http.MethodPost, "/api/agent/v1/projects/"+project.ID+"/memories/prune", `{"older_than_days":30}`, project.ID, "", "org-a")
	api.pruneProjectMemories(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("prune status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Removed       int `json:"removed"`
		OlderThanDays int `json:"older_than_days"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode prune: %v", err)
	}
	if payload.Removed != 1 || payload.OlderThanDays != 30 {
		t.Fatalf("prune payload = %+v", payload)
	}
	remaining := api.context.ExportMemories(project.ID, false)
	if len(remaining) != 1 || remaining[0].ID != "mem-new" {
		t.Fatalf("remaining after prune = %+v", remaining)
	}
}
