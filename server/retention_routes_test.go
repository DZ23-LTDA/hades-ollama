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

func newRetentionAPI(t *testing.T) (*agentAPI, agent.Project, agent.Project) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	own, err := store.CreateProject("projeto A", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateProject("projeto B", t.TempDir(), "org-b")
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{context: store, authRequired: true}, own, other
}

// retentionRequest monta o contexto gin para as rotas de retenção.
func retentionRequest(t *testing.T, method, path, body, organization string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("agent.organization", agent.Organization{ID: organization})
	return ctx, recorder
}

func seedRetentionMemoriesForRoute(t *testing.T, store *agent.ContextStore, projectID, prefix string, count int, age time.Duration) {
	t.Helper()
	for index := range count {
		if _, err := store.AddMemory(agent.Memory{
			ID:        prefix + "_" + time.Unix(int64(index), 0).UTC().Format("150405"),
			ProjectID: projectID,
			Kind:      "fact",
			Content:   "memória de retenção por rota",
			CreatedAt: time.Now().UTC().Add(-age - time.Duration(index)*time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRetentionRoutesConfigureApplyAndReportRealCounts(t *testing.T) {
	api, own, other := newRetentionAPI(t)

	// Antes de configurar: configured=false (não é erro).
	getCtx, getRecorder := retentionRequest(t, http.MethodGet, "/api/agent/v1/retention", "", "org-a")
	api.retentionPolicy(getCtx)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var initial struct {
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Configured {
		t.Fatal("policy must start unconfigured")
	}

	// Aplicar sem política: 404 explícito, nada removido.
	seedRetentionMemoriesForRoute(t, api.context, own.ID, "antiga", 2, 90*24*time.Hour)
	applyCtx, applyRecorder := retentionRequest(t, http.MethodPost, "/api/agent/v1/retention/apply", "", "org-a")
	api.applyRetentionPolicy(applyCtx)
	if applyRecorder.Code != http.StatusNotFound {
		t.Fatalf("apply without policy status=%d body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}
	if got := len(api.context.ExportMemories(own.ID, false)); got != 2 {
		t.Fatalf("nothing may be removed without a policy, got %d", got)
	}

	// Política inválida: 400.
	badCtx, badRecorder := retentionRequest(t, http.MethodPut, "/api/agent/v1/retention", `{"max_age_days":0}`, "org-a")
	api.setRetentionPolicy(badCtx)
	if badRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid policy status=%d body=%s", badRecorder.Code, badRecorder.Body.String())
	}

	// Política válida: 200 e a leitura passa a mostrar configured=true.
	putCtx, putRecorder := retentionRequest(t, http.MethodPut, "/api/agent/v1/retention", `{"max_age_days":30,"max_memories_per_project":1}`, "org-a")
	api.setRetentionPolicy(putCtx)
	if putRecorder.Code != http.StatusOK {
		t.Fatalf("set policy status=%d body=%s", putRecorder.Code, putRecorder.Body.String())
	}
	seedRetentionMemoriesForRoute(t, api.context, own.ID, "nova", 3, time.Minute)
	seedRetentionMemoriesForRoute(t, api.context, other.ID, "alheia", 2, 200*24*time.Hour)

	applyCtx2, applyRecorder2 := retentionRequest(t, http.MethodPost, "/api/agent/v1/retention/apply", "", "org-a")
	api.applyRetentionPolicy(applyCtx2)
	if applyRecorder2.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", applyRecorder2.Code, applyRecorder2.Body.String())
	}
	var result agent.RetentionResult
	if err := json.Unmarshal(applyRecorder2.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v body=%s", err, applyRecorder2.Body.String())
	}
	if result.ProjectsScanned != 1 {
		t.Fatalf("only the authenticated organization may be scanned: %+v", result)
	}
	if result.RemovedByAge != 2 {
		t.Fatalf("removed by age = %d, want 2", result.RemovedByAge)
	}
	if result.RemovedByCount != 2 {
		t.Fatalf("removed by count = %d, want 2", result.RemovedByCount)
	}
	if got := len(api.context.ExportMemories(own.ID, false)); got != 1 {
		t.Fatalf("own project must keep the ceiling, got %d", got)
	}
	// A organização B, sem política, permanece intacta.
	if got := len(api.context.ExportMemories(other.ID, false)); got != 2 {
		t.Fatalf("other organization must be untouched, got %d", got)
	}
	// A política de A não vaza para B.
	getBCtx, getBRecorder := retentionRequest(t, http.MethodGet, "/api/agent/v1/retention", "", "org-b")
	api.retentionPolicy(getBCtx)
	if !strings.Contains(getBRecorder.Body.String(), `"configured":false`) {
		t.Fatalf("org-b must not see org-a policy: %s", getBRecorder.Body.String())
	}
}
