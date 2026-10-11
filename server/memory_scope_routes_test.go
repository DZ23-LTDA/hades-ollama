package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// Atividade do estágio 15: ESCOPO alcançável pelas rotas, não só na API Go.
// A memória presa a uma missão não pode aparecer em busca/export sem essa missão.
func scopedMemoryRequest(t *testing.T, api *agentAPI, query, actor, projectID string) []agent.Memory {
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
	return payload.Memories
}

func TestMemorySearchRouteHonorsMissionScope(t *testing.T) {
	api, project := newMemoryIsolationAPI(t)
	if _, err := api.context.AddMemoryForActor(t.Context(), agent.Memory{
		ProjectID: project.ID, Kind: "note", Content: "detalhe privado da missão", MissionID: "missao-1",
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	semMissao := scopedMemoryRequest(t, api, "detalhe", "ana", project.ID)
	if len(semMissao) != 0 {
		t.Fatalf("memória de missão não pode aparecer em busca sem missão: %+v", semMissao)
	}
	comMissao := scopedMemoryRequest(t, api, "detalhe&mission_id=missao-1", "ana", project.ID)
	if len(comMissao) != 1 {
		t.Fatalf("a missão dona precisa encontrar a memória: %+v", comMissao)
	}
	// O EXPORT segue a mesma regra — o irmão não pode ficar para trás.
	exportCtx, exportRecorder := memoryIsolationRequest(t, http.MethodGet, project.ID, "", "ana", "org-time")
	exportCtx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+project.ID+"/memories/export", nil)
	exportCtx.Request.Header.Set("X-Ollama-User", "ana")
	api.exportProjectMemories(exportCtx)
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", exportRecorder.Code, exportRecorder.Body.String())
	}
	if strings.Contains(exportRecorder.Body.String(), "detalhe privado da missão") {
		t.Fatalf("o export sem missão não pode entregar memória de missão: %s", exportRecorder.Body.String())
	}
	// E o escopo de AGENTE também é respeitado pela rota.
	if _, err := api.context.AddMemoryForActor(t.Context(), agent.Memory{
		ProjectID: project.ID, Kind: "note", Content: "anotação do agente", AgentID: "agente-x",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	if found := scopedMemoryRequest(t, api, "anotação", "ana", project.ID); len(found) != 0 {
		t.Fatalf("escopo de agente precisa restringir a busca sem escopo: %+v", found)
	}
	if found := scopedMemoryRequest(t, api, "anotação&agent_id=agente-x", "ana", project.ID); len(found) != 1 {
		t.Fatalf("o agente dono precisa ler a memória dele pela rota: %+v", found)
	}
}
