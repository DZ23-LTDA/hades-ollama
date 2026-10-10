package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// memoryIsolationRequest monta o contexto gin das rotas de memória com um ATOR
// explícito (cabeçalho X-Ollama-User) na mesma organização/projeto.
func memoryIsolationRequest(t *testing.T, method, projectID, body, actor, org string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, "/api/agent/v1/projects/"+projectID, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("X-Ollama-User", actor)
	ctx.Params = gin.Params{{Key: "id", Value: projectID}}
	ctx.Set("agent.organization", agent.Organization{ID: org})
	return ctx, recorder
}

func newMemoryIsolationAPI(t *testing.T) (*agentAPI, agent.Project) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Planner: agent.RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("Projeto compartilhado", filepath.Join(root, "proj"), "org-time")
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{runtime: runtime, context: contextStore}, project
}

func searchMemoryIDs(t *testing.T, api *agentAPI, projectID, actor string) []string {
	t.Helper()
	ctx, recorder := memoryIsolationRequest(t, http.MethodGet, projectID, "", actor, "org-time")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+projectID+"/memories?q=", nil)
	ctx.Request.Header.Set("X-Ollama-User", actor)
	api.searchMemories(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Memories []agent.Memory `json:"memories"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode search: %v body=%s", err, recorder.Body.String())
	}
	ids := make([]string, 0, len(payload.Memories))
	for _, memory := range payload.Memories {
		ids = append(ids, memory.ID)
	}
	return ids
}

func TestMemoryRoutesIsolatePrivateMemoryBetweenUsers(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)

	// Aline grava uma memória PRIVADA e uma do projeto; Bruno grava uma do projeto.
	privateBody := `{"id":"mem_aline","kind":"fact","content":"rascunho privado da aline","visibility":"private"}`
	ctxAline, recAline := memoryIsolationRequest(t, http.MethodPost, project.ID, privateBody, "aline", "org-time")
	api.addMemory(ctxAline)
	if recAline.Code != http.StatusCreated {
		t.Fatalf("private add status=%d body=%s", recAline.Code, recAline.Body.String())
	}
	var created agent.Memory
	if err := json.Unmarshal(recAline.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ActorID != "aline" || created.Visibility != agent.MemoryVisibilityPrivate {
		t.Fatalf("created memory = %+v", created)
	}

	for _, fixture := range []struct{ id, actor, body string }{
		{"mem_projeto_aline", "aline", `{"id":"mem_projeto_aline","kind":"fact","content":"decisao do time sobre rascunho"}`},
		{"mem_projeto_bruno", "bruno", `{"id":"mem_projeto_bruno","kind":"fact","content":"outra decisao do time sobre rascunho"}`},
	} {
		ctx, recorder := memoryIsolationRequest(t, http.MethodPost, project.ID, fixture.body, fixture.actor, "org-time")
		api.addMemory(ctx)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("add %s status=%d body=%s", fixture.id, recorder.Code, recorder.Body.String())
		}
	}

	// Aline enxerga as três; Bruno não enxerga a privada dela.
	alineIDs := searchMemoryIDs(t, api, project.ID, "aline")
	if len(alineIDs) != 3 {
		t.Fatalf("aline must see her private memory and the project ones: %v", alineIDs)
	}
	brunoIDs := searchMemoryIDs(t, api, project.ID, "bruno")
	if len(brunoIDs) != 2 {
		t.Fatalf("bruno must not see aline's private memory: %v", brunoIDs)
	}
	for _, id := range brunoIDs {
		if id == "mem_aline" {
			t.Fatal("private memory leaked to another user through the search route")
		}
	}
}

func TestMemoryRouteIgnoresForgedActorAndRejectsAnonymousPrivate(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)

	// Bruno tenta se passar por Aline no corpo da requisição: a autoria vem da
	// sessão, então a memória é do Bruno.
	forged := `{"id":"mem_forjada","kind":"fact","content":"tentativa de forjar","visibility":"private","actor_id":"aline"}`
	ctx, recorder := memoryIsolationRequest(t, http.MethodPost, project.ID, forged, "bruno", "org-time")
	api.addMemory(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("forged add status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var created agent.Memory
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ActorID != "bruno" {
		t.Fatalf("actor must come from the session, got %+v", created)
	}
	if ids := searchMemoryIDs(t, api, project.ID, "aline"); len(ids) != 0 {
		t.Fatalf("forged authorship leaked to aline: %v", ids)
	}

	// Sem usuário autenticado nem cabeçalho, o ator é o sintético `local` (modo
	// de um único usuário). A memória privada existe, mas continua invisível a
	// qualquer ator nomeado — que é o efeito de segurança que importa.
	anonymous := `{"id":"mem_anonima","kind":"fact","content":"privada do modo local","visibility":"private"}`
	ctxAnon, recorderAnon := memoryIsolationRequest(t, http.MethodPost, project.ID, anonymous, "", "org-time")
	api.addMemory(ctxAnon)
	if recorderAnon.Code != http.StatusCreated {
		t.Fatalf("local-mode private memory status=%d body=%s", recorderAnon.Code, recorderAnon.Body.String())
	}
	var localMemory agent.Memory
	if err := json.Unmarshal(recorderAnon.Body.Bytes(), &localMemory); err != nil {
		t.Fatal(err)
	}
	if localMemory.ActorID != agent.LocalActorID {
		t.Fatalf("local-mode memory must belong to the local actor, got %+v", localMemory)
	}
	for _, id := range searchMemoryIDs(t, api, project.ID, "bruno") {
		if id == "mem_anonima" {
			t.Fatal("named actor must not read local-mode private memory")
		}
	}
	// E o ator local não enxerga a memória privada do Bruno.
	for _, id := range searchMemoryIDs(t, api, project.ID, "") {
		if id == "mem_forjada" {
			t.Fatal("local actor must not read another actor's private memory")
		}
	}
}

func TestAskProjectDocumentsDoesNotCiteOtherUserPrivateMemory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	contextStore.SetEmbedder(fakeAskEmbedder{
		"orçamento secreto": {1, 0},
		"orcamento":         {1, 0},
	})
	root := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Planner: agent.RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("Financeiro", filepath.Join(root, "fin"), "org-fin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contextStore.AddMemoryForActor(context.Background(), agent.Memory{
		ID: "mem_secreta", ProjectID: project.ID, Kind: "doc",
		Content: "orçamento secreto", Source: "aline.txt#1", Confidence: 1,
		Visibility: agent.MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore}

	// Aline recebe a citação da própria memória.
	ctxAline, recAline := askRequest(t, project.ID, "orcamento", "org-fin")
	ctxAline.Request.Header.Set("X-Ollama-User", "aline")
	api.askProjectDocuments(ctxAline)
	if recAline.Code != http.StatusOK {
		t.Fatalf("aline ask status=%d body=%s", recAline.Code, recAline.Body.String())
	}
	if !strings.Contains(recAline.Body.String(), "aline.txt#1") {
		t.Fatalf("owner must see her own source: %s", recAline.Body.String())
	}

	// Bruno não pode vê-la nem como referência.
	ctxBruno, recBruno := askRequest(t, project.ID, "orcamento", "org-fin")
	ctxBruno.Request.Header.Set("X-Ollama-User", "bruno")
	api.askProjectDocuments(ctxBruno)
	if recBruno.Code != http.StatusOK {
		t.Fatalf("bruno ask status=%d body=%s", recBruno.Code, recBruno.Body.String())
	}
	if strings.Contains(recBruno.Body.String(), "aline.txt#1") {
		t.Fatalf("private memory of another user must not be cited: %s", recBruno.Body.String())
	}
	var none struct {
		Grounded  bool          `json:"grounded"`
		Citations []interface{} `json:"citations"`
	}
	if err := json.Unmarshal(recBruno.Body.Bytes(), &none); err != nil {
		t.Fatal(err)
	}
	if none.Grounded || len(none.Citations) != 0 {
		t.Fatalf("bruno must have no grounded sources: %s", recBruno.Body.String())
	}
}
