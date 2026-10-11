package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Atividade do estágio 15: VERSIONAMENTO alcançável pelo produto, não só pela API
// Go. A rota de alteração precisa ser escopada por ator como as demais do mesmo
// dado — o irmão esquecido é o caminho mais provável do vazamento.
func memoryItemRequest(t *testing.T, api *agentAPI, method, projectID, memoryID, body, actor string) (int, string) {
	t.Helper()
	ctx, recorder := memoryIsolationRequest(t, method, projectID, body, actor, "org-time")
	ctx.Params = gin.Params{{Key: "id", Value: projectID}, {Key: "memory_id", Value: memoryID}}
	ctx.Request.Header.Set("X-Ollama-User", actor)
	switch method {
	case http.MethodPatch:
		api.updateProjectMemory(ctx)
	case http.MethodGet:
		api.memoryHistory(ctx)
	default:
		t.Fatalf("método não suportado: %s", method)
	}
	return recorder.Code, recorder.Body.String()
}

func TestMemoryUpdateRouteVersionsAndIsolatesActors(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)
	anaMemory := addPrivateMemory(t, api, project.ID, "ana", "decisão antiga da ana")

	// Bruno não altera memória privada da ana: 404, sem revelar existência.
	code, body := memoryItemRequest(t, api, http.MethodPatch, project.ID, anaMemory, `{"content":"texto do bruno","reason":"tentativa"}`, "bruno")
	if code != http.StatusNotFound {
		t.Fatalf("alterar memória privada alheia precisa dar 404, veio %d (%s)", code, body)
	}
	if strings.Contains(body, "decisão antiga da ana") {
		t.Fatalf("a resposta não pode revelar o conteúdo privado: %s", body)
	}

	// A própria ana altera: 200, nova revisão e histórico com o texto anterior.
	code, body = memoryItemRequest(t, api, http.MethodPatch, project.ID, anaMemory, `{"content":"decisão nova da ana","reason":"revisão na reunião"}`, "ana")
	if code != http.StatusOK {
		t.Fatalf("a dona precisa poder alterar: %d (%s)", code, body)
	}
	var updated agent.Memory
	if err := json.Unmarshal([]byte(body), &updated); err != nil {
		t.Fatalf("decode update: %v body=%s", err, body)
	}
	if updated.Content != "decisão nova da ana" || updated.Revision != 1 || len(updated.Versions) != 1 {
		t.Fatalf("a alteração precisa versionar: %+v", updated)
	}
	if updated.Versions[0].Content != "decisão antiga da ana" || updated.Versions[0].Reason != "revisão na reunião" {
		t.Fatalf("o histórico precisa de conteúdo anterior e motivo: %+v", updated.Versions[0])
	}

	// O histórico é legível pela dona e recusado a terceiro.
	code, body = memoryItemRequest(t, api, http.MethodGet, project.ID, anaMemory, "", "ana")
	if code != http.StatusOK || !strings.Contains(body, "decisão antiga da ana") {
		t.Fatalf("a dona precisa ler o histórico: %d (%s)", code, body)
	}
	code, _ = memoryItemRequest(t, api, http.MethodGet, project.ID, anaMemory, "", "bruno")
	if code != http.StatusNotFound {
		t.Fatalf("terceiro não pode ler o histórico: %d", code)
	}

	// Conteúdo vazio é recusado (400) em vez de apagar o conhecimento.
	code, _ = memoryItemRequest(t, api, http.MethodPatch, project.ID, anaMemory, `{"content":"   "}`, "ana")
	if code != http.StatusBadRequest {
		t.Fatalf("conteúdo vazio precisa ser 400, veio %d", code)
	}

	// E o caminho de BUSCA mostra o conteúdo novo, não o antigo.
	ids := searchMemoryIDsByQuery(t, api, project.ID, "ana", "nova")
	if len(ids) != 1 || ids[0] != anaMemory {
		t.Fatalf("a busca precisa encontrar a revisão atual: %v", ids)
	}
}

// A rota nova precisa estar registrada de verdade, não só existir como função.
func TestMemoryVersionRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	found := map[string]bool{}
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{
		"PATCH /api/agent/v1/projects/:id/memories/:memory_id",
		"GET /api/agent/v1/projects/:id/memories/:memory_id/history",
	} {
		if !found[want] {
			t.Fatalf("rota %s não está registrada", want)
		}
	}
}
