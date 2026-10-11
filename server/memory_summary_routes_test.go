package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Atividade do estágio 15: RESUMO alcançável pela rota, declarando o motor usado e
// herdando o isolamento da busca.
func TestMemorySummaryRouteIsFaithfulScopedAndDeclared(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)
	if _, err := api.context.AddMemoryForActor(t.Context(), agent.Memory{
		ProjectID: project.ID, Kind: "decision", Content: "o time escolheu PostgreSQL para o núcleo",
		Source: "ata.md",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.context.AddMemoryForActor(t.Context(), agent.Memory{
		ProjectID: project.ID, Kind: "note", Content: "segredo privado da ana sobre o núcleo",
		Visibility: agent.MemoryVisibilityPrivate,
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	summaryFor := func(actor, query string) (int, string) {
		t.Helper()
		ctx, recorder := memoryIsolationRequest(t, http.MethodGet, project.ID, "", actor, "org-time")
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/summary?q="+query, nil)
		ctx.Request.Header.Set("X-Ollama-User", actor)
		api.memorySummary(ctx)
		return recorder.Code, recorder.Body.String()
	}

	code, body := summaryFor("ana", "núcleo")
	if code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", code, body)
	}
	var summary agent.MemorySummary
	if err := json.Unmarshal([]byte(body), &summary); err != nil {
		t.Fatalf("decode summary: %v body=%s", err, body)
	}
	if summary.Summarizer == "" {
		t.Fatalf("o resumo precisa declarar o motor: %s", body)
	}
	if summary.Count == 0 || len(summary.Citations) == 0 {
		t.Fatalf("o resumo precisa citar as fontes: %+v", summary)
	}
	// Fidelidade: o trecho citado existe na memória de origem.
	for _, citation := range summary.Citations {
		if citation.Snippet == "" || !strings.Contains(summary.Text, citation.Snippet) {
			t.Fatalf("citação fora do texto do resumo: %+v", citation)
		}
	}

	// Terceiro não recebe a memória privada da ana no resumo.
	_, bodyBruno := summaryFor("bruno", "núcleo")
	if strings.Contains(bodyBruno, "segredo privado da ana") {
		t.Fatalf("o resumo por rota VAZOU memória privada de outro usuário: %s", bodyBruno)
	}
}

func TestMemorySummaryRouteIsRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	want := "GET /api/agent/v1/projects/:id/memories/summary"
	for _, route := range engine.Routes() {
		if route.Method+" "+route.Path == want {
			return
		}
	}
	t.Fatalf("rota %s não está registrada", want)
}
