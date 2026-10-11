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

// DEFEITO CORRIGIDO (estágio 15): a busca de memória era escopada por ator, mas o
// EXPORT e o DELETE chamavam as funções SEM ator. Efeito medido: a busca escondia
// a memória privada de um colega e o export a entregava, e qualquer usuário podia
// apagar a memória privada de outro. Mesmo dado, duas respostas — o clássico
// "protegi um caminho e esqueci o irmão".
func addPrivateMemory(t *testing.T, api *agentAPI, projectID, actor, content string) string {
	t.Helper()
	ctx, recorder := memoryIsolationRequest(t, http.MethodPost, projectID, `{"kind":"note","content":"`+content+`","visibility":"private"}`, actor, "org-time")
	api.addMemory(ctx)
	if recorder.Code != http.StatusCreated && recorder.Code != http.StatusOK {
		t.Fatalf("addMemory status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload agent.Memory
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode add: %v body=%s", err, recorder.Body.String())
	}
	if payload.ID == "" {
		t.Fatalf("memória sem id: %s", recorder.Body.String())
	}
	return payload.ID
}

func TestMemoryExportRouteDoesNotLeakOtherUserPrivateMemory(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)
	anaMemory := addPrivateMemory(t, api, project.ID, "ana", "segredo da ana")

	// Bruno exporta o projeto: a memória privada da ana NÃO pode aparecer.
	ctx, recorder := memoryIsolationRequest(t, http.MethodGet, project.ID, "", "bruno", "org-time")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/export", nil)
	ctx.Request.Header.Set("X-Ollama-User", "bruno")
	api.exportProjectMemories(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "segredo da ana") {
		t.Fatalf("o export VAZOU memória privada de outro usuário: %s", recorder.Body.String())
	}
	var payload struct {
		ActorID  string         `json:"actor_id"`
		Count    int            `json:"count"`
		Memories []agent.Memory `json:"memories"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, memory := range payload.Memories {
		if memory.ID == anaMemory {
			t.Fatalf("a memória privada da ana apareceu no export do bruno: %+v", memory)
		}
	}
	if payload.ActorID != "bruno" {
		t.Fatalf("o export precisa registrar quem exportou, veio %q", payload.ActorID)
	}

	// A própria ana exporta e ENCONTRA a memória dela.
	anaCtx, anaRecorder := memoryIsolationRequest(t, http.MethodGet, project.ID, "", "ana", "org-time")
	anaCtx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/export", nil)
	anaCtx.Request.Header.Set("X-Ollama-User", "ana")
	api.exportProjectMemories(anaCtx)
	if anaRecorder.Code != http.StatusOK {
		t.Fatalf("export da ana status=%d body=%s", anaRecorder.Code, anaRecorder.Body.String())
	}
	if !strings.Contains(anaRecorder.Body.String(), "segredo da ana") {
		t.Fatalf("a dona da memória precisa exportá-la: %s", anaRecorder.Body.String())
	}
}

func TestMemoryDeleteRouteRefusesOtherUserPrivateMemory(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)
	anaMemory := addPrivateMemory(t, api, project.ID, "ana", "não apague isto")

	// Bruno tenta apagar a memória privada da ana: 404, sem revelar que existe.
	deleteCtx, deleteRecorder := memoryIsolationRequest(t, http.MethodDelete, project.ID, "", "bruno", "org-time")
	deleteCtx.Params = gin.Params{{Key: "id", Value: project.ID}, {Key: "memory_id", Value: anaMemory}}
	deleteCtx.Request.Header.Set("X-Ollama-User", "bruno")
	api.deleteProjectMemory(deleteCtx)
	if deleteRecorder.Code != http.StatusNotFound {
		t.Fatalf("apagar memória privada alheia precisa dar 404, veio %d (%s)", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	// E a memória da ana continua lá (na busca dela).
	if ids := searchMemoryIDsByQuery(t, api, project.ID, "ana", "apague"); len(ids) != 1 {
		t.Fatalf("a memória da ana não podia ser apagada por outro: %v", ids)
	}

	// A própria ana apaga a dela: 200 e desaparece.
	ownCtx, ownRecorder := memoryIsolationRequest(t, http.MethodDelete, project.ID, "", "ana", "org-time")
	ownCtx.Params = gin.Params{{Key: "id", Value: project.ID}, {Key: "memory_id", Value: anaMemory}}
	ownCtx.Request.Header.Set("X-Ollama-User", "ana")
	api.deleteProjectMemory(ownCtx)
	if ownRecorder.Code != http.StatusOK {
		t.Fatalf("a dona precisa poder apagar: %d (%s)", ownRecorder.Code, ownRecorder.Body.String())
	}
	if ids := searchMemoryIDsByQuery(t, api, project.ID, "ana", "apague"); len(ids) != 0 {
		t.Fatalf("após apagar, a memória não pode continuar aparecendo: %v", ids)
	}
}

// searchMemoryIDsByQuery é o mesmo caminho de busca do teste existente, com a
// consulta de filtro explicitada.
func searchMemoryIDsByQuery(t *testing.T, api *agentAPI, projectID, actor, query string) []string {
	t.Helper()
	ctx, recorder := memoryIsolationRequest(t, http.MethodGet, projectID, "", actor, "org-time")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+projectID+"/memories?q="+query, nil)
	ctx.Request.Header.Set("X-Ollama-User", actor)
	api.searchMemories(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Memories []agent.Memory `json:"memories"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(payload.Memories))
	for _, memory := range payload.Memories {
		ids = append(ids, memory.ID)
	}
	return ids
}
