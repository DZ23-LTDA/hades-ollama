package server

// FASE 23 do adendo V3: E2E possivel do `/preset` -- router REAL -> handler REAL -> catalogo REAL.
//
// Nao ha mock nesta cadeia: o `agentAPI.register` monta o mesmo router que a producao monta, e o
// handler devolve o mesmo `agent.BuiltInMissionPresets()` que o `/preset` resolve. O que este
// teste prova e' que a rota RESPONDE com o catalogo -- nao que a rota existe (isso ja' e' o
// `TestEveryAvailableCommandMapsToARegisteredRoute`, que compara strings).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func getMissionPresets(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/presets", nil))
	return recorder
}

func TestMissionPresetsRouteServeOCatalogoReal(t *testing.T) {
	recorder := getMissionPresets(t)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/v1/presets -> %d (corpo: %s)", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Presets []agent.MissionPreset `json:"presets"`
		Count   int                   `json:"count"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("o corpo nao e' JSON valido: %v", err)
	}

	if payload.Count != len(payload.Presets) {
		t.Fatalf("count=%d mas vieram %d presets", payload.Count, len(payload.Presets))
	}
	if payload.Count != 11 {
		t.Fatalf("esperados 11 presets embutidos, vieram %d", payload.Count)
	}

	// O contrato que a FASE 23 exige: cada preset declara criterio de conclusao.
	// *Um preset sem criterio de conclusao e' um preset que nunca acaba.*
	for _, preset := range payload.Presets {
		if preset.ID == "" {
			t.Error("preset sem id")
		}
		if len(preset.CompletionCriteria) == 0 {
			t.Errorf("preset %q sem completion_criteria", preset.ID)
		}
		if preset.Goal == "" {
			t.Errorf("preset %q sem objetivo", preset.ID)
		}
	}

	// E o catalogo servido pela rota tem de ser o MESMO que o `/preset` resolve.
	// *Duas listas com o mesmo nome e conteudo diferente mentem em silencio.*
	ids := agent.MissionPresetIDs()
	if len(ids) != payload.Count {
		t.Fatalf("a rota serve %d presets e o catalogo tem %d", payload.Count, len(ids))
	}
	servidos := map[string]bool{}
	for _, preset := range payload.Presets {
		servidos[preset.ID] = true
	}
	for _, id := range ids {
		if !servidos[id] {
			t.Errorf("o preset %q existe no catalogo mas nao veio na rota", id)
		}
	}
}

func TestMissionPresetsRouteNaoVazaSegredo(t *testing.T) {
	corpo := getMissionPresets(t).Body.String()
	if corpo == "" {
		t.Fatal("a rota devolveu corpo vazio")
	}
	for _, proibido := range []string{"BEGIN PRIVATE KEY", "storePassword", "keyPassword", "client_email", "BEGIN RSA"} {
		if strings.Contains(corpo, proibido) {
			t.Errorf("a rota expoe %q", proibido)
		}
	}
}
