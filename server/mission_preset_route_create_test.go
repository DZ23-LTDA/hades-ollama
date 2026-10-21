package server

// FASE 23 do adendo V3: E2E de ROTA do preset na criacao de missao.
//
// A UI escolhe um preset, o `AgenticConsole` envia `preset_id` no POST e o servidor tem de
// RECUSAR um preset que nao existe. Sem este teste, a UI poderia enviar o campo e o servidor
// ignora-lo em silencio -- *um campo que o servidor ignora e' um formulario que mente.*

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// createMissionBody usa organizacao VAZIA de proposito: o teste e' sobre o PRESET, nao sobre
// o escopo organizacional. Uma organizacao activa exigiria um project, e misturar as duas
// validacoes num teste so' tornaria a falha ambigua.
func createMissionBody(t *testing.T, api *agentAPI, body map[string]any) (int, string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	ctx, recorder := commandRequest(t, http.MethodPost, string(raw), "")
	api.createMission(ctx)
	return recorder.Code, recorder.Body.String()
}

func TestCreateMissionRouteRecusaPresetInexistente(t *testing.T) {
	api, _, _ := newCommandAPI(t)
	codigo, corpo := createMissionBody(t, api, map[string]any{
		"objective": "tarefa com preset inventado",
		"preset_id": "preset-que-nao-existe",
	})
	if codigo == http.StatusOK || codigo == http.StatusCreated {
		t.Fatalf("preset inexistente foi aceite: %d %s", codigo, corpo)
	}
	// A recusa tem de ser ACIONAVEL: dizer qual id nao existe e listar os validos.
	if !strings.Contains(corpo, "preset-que-nao-existe") {
		t.Fatalf("a recusa nao nomeia o preset: %s", corpo)
	}
	for _, id := range agent.MissionPresetIDs() {
		if !strings.Contains(corpo, id) {
			t.Errorf("a recusa nao lista o preset valido %q", id)
		}
	}
}

func TestCreateMissionRouteAceitaPresetDoCatalogo(t *testing.T) {
	api, _, _ := newCommandAPI(t)
	codigo, corpo := createMissionBody(t, api, map[string]any{
		"objective": "produzir o briefing do dia",
		"preset_id": "briefing-diario",
	})
	if codigo != http.StatusOK && codigo != http.StatusCreated {
		t.Fatalf("preset do catalogo foi recusado: %d %s", codigo, corpo)
	}
	var missao agent.Mission
	if err := json.Unmarshal([]byte(corpo), &missao); err != nil {
		t.Fatalf("decode: %v body=%s", err, corpo)
	}
	if missao.PresetID != "briefing-diario" {
		t.Fatalf("a missao criada nao guardou o preset: %q", missao.PresetID)
	}
}

func TestCreateMissionRouteSemPresetContinuaAIgual(t *testing.T) {
	api, _, _ := newCommandAPI(t)
	codigo, corpo := createMissionBody(t, api, map[string]any{
		"objective": "tarefa sem preset",
	})
	if codigo != http.StatusOK && codigo != http.StatusCreated {
		t.Fatalf("missao sem preset foi recusada: %d %s", codigo, corpo)
	}
}
