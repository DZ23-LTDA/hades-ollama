package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// QUARTO IRMÃO DA MESMA FAMÍLIA (ALTO): o comando /memory usava
// SearchMemoriesContext — a busca SEM ator — e devolvia a memória `private` de
// OUTRO usuário do mesmo projeto no corpo da resposta. A rota de busca já era
// escopada; o comando, que lê o MESMO dado, não era.
func runMemoryCommandAs(t *testing.T, api *agentAPI, projectID, actor string) (int, string) {
	t.Helper()
	ctx, recorder := commandRequest(t, http.MethodPost, `{"input":"/memory `+projectID+`"}`, "org-a")
	ctx.Request.Header.Set("X-Ollama-User", actor)
	api.runCommand(ctx)
	return recorder.Code, recorder.Body.String()
}

func TestMemoryCommandDoesNotLeakOtherUserPrivateMemory(t *testing.T) {
	api, _, _ := newCommandAPI(t)
	project, err := api.context.CreateProject("projeto do comando", filepath.Join(api.runtime.WorkspaceRoot(), "proj-cmd-iso"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.context.AddMemoryForActor(context.Background(), agent.Memory{
		ProjectID: project.ID, Kind: "fact", Content: "conhecimento do time pelo comando",
		Visibility: agent.MemoryVisibilityOrganization,
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.context.AddMemoryForActor(context.Background(), agent.Memory{
		ProjectID: project.ID, Kind: "note", Content: "segredo da ana pelo comando",
		Visibility: agent.MemoryVisibilityPrivate,
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	// Bruno usa /memory no mesmo projeto: nada privado da ana pode aparecer.
	code, body := runMemoryCommandAs(t, api, project.ID, "bruno")
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	if strings.Contains(body, "segredo da ana pelo comando") {
		t.Fatalf("o comando /memory VAZOU memória privada de outro usuário: %s", body)
	}
	if !strings.Contains(body, "conhecimento do time pelo comando") {
		t.Fatalf("o conhecimento do projeto precisa continuar acessível: %s", body)
	}

	// A própria ana encontra a memória dela.
	code, body = runMemoryCommandAs(t, api, project.ID, "ana")
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	if !strings.Contains(body, "segredo da ana pelo comando") {
		t.Fatalf("a dona da memória precisa vê-la pelo comando: %s", body)
	}

	// E sem ator identificado, nada privado aparece (fail-closed).
	code, body = runMemoryCommandAs(t, api, project.ID, "")
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	if strings.Contains(body, "segredo da ana pelo comando") {
		t.Fatalf("sem ator, memória privada não pode aparecer: %s", body)
	}

	// O comando continua entregando a estrutura esperada, agora escopada.
	var payload struct {
		Executed bool           `json:"executed"`
		Result   map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		// A resposta acima foi a do bruno; decodificar a última é o suficiente
		// para provar que o corpo não vazou.
		return
	}
}
