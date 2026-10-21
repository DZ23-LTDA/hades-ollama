package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestCompanionRouteReportsHonestState prova o elo que faltava no item 46: a rota
// devolve o estado do Companion DERIVADO dos eventos persistidos da missao - e o
// declara com a razao e o escopo, sem inventar nada.
//
// O fixture e' o mesmo da rota vizinha (`mission_events_limit_test.go`): *o elo novo
// e' testado no mesmo molde do que ja' existia, e nao num molde so' dele.*
func TestCompanionRouteReportsHonestState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	store := agent.NewMemoryStore()
	// ***E-1087: o `DataRoot` e' TEMPORARIO.*** *Sem ele, o `NewRuntime` resolve
	// para o `%APPDATA%` real do utilizador (`resolveRuntimeDataRoot`), e o teste
	// **escreve na maquina** -- medido aqui: `mkdir ...\ollama-agent\<hash>: Acesso negado`
	// num ambiente confinado, enquanto no CI (sem confinamento) passa.* => *um teste que
	// depende de onde corre nao testa o que diz testar.*
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: store, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "companion", OrganizationID: "org-a", Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}

	api := &agentAPI{runtime: runtime, authRequired: true}
	chamar := func(organizationID, missaoID string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/missions/"+missaoID+"/companion", nil)
		ctx.Params = gin.Params{{Key: "id", Value: missaoID}}
		ctx.Set("agent.organization", agent.Organization{ID: organizationID})
		api.companion(ctx)
		var corpo map[string]any
		if recorder.Code == http.StatusOK {
			if err := json.Unmarshal(recorder.Body.Bytes(), &corpo); err != nil {
				t.Fatalf("decode companion: %v body=%s", err, recorder.Body.String())
			}
		}
		return recorder, corpo
	}

	// 1. Missao recem-criada: o estado honesto e' `a_planear`, porque o runtime emite
	//    `mission.planning` (do vocabulario). O `mission.planned` que ele tambem emite
	//    NAO pertence ao vocabulario e e' descartado pelo conversor - *antes deste
	//    filtro, a rota respondia 400 aqui. Este caso e' a regressao do defeito.*
	recorder, corpo := chamar("org-a", mission.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("missao recem-criada: status=%d body=%s (era aqui que o defeito dava 400)", recorder.Code, recorder.Body.String())
	}
	if corpo["estado"] != "a_planear" {
		t.Fatalf("missao recem-criada devia estar a_planear, veio %v (body=%s)", corpo["estado"], recorder.Body.String())
	}
	if razao, _ := corpo["razao"].(string); razao == "" {
		t.Fatal("o estado honesto tem de vir COM a razao; veio vazia")
	}
	// O escopo e' declarado: um agregado sem escopo e' lido como global.
	if escopo, _ := corpo["scope"].(string); escopo == "" {
		t.Fatal("a resposta tem de declarar o escopo")
	}

	// 2. Um evento REAL e persistido muda o estado. `mission.cancelled` pertence ao
	//    vocabulario fechado do nucleo (`companion_estado.go`).
	if err := store.AppendEvent(agent.Event{
		ID: "ev_cancel", MissionID: mission.ID, OrganizationID: "org-a",
		Type: "mission.cancelled", CreatedAt: time.Now().UTC().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	recorder, corpo = chamar("org-a", mission.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("com evento: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if corpo["estado"] != "cancelado" {
		t.Fatalf("com mission.cancelled o estado devia ser cancelado, veio %v (body=%s)", corpo["estado"], recorder.Body.String())
	}

	// 3. Escopo: outra organizacao NAO recebe estado nenhum desta missao.
	recorder, _ = chamar("org-b", mission.ID)
	if recorder.Code == http.StatusOK {
		t.Fatalf("missao alheia devia ser recusada, veio 200 com %s", recorder.Body.String())
	}
}
