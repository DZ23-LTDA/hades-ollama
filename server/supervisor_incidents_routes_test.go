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

// Estágio 31: o registro de incidente precisa ser ALCANÇÁVEL pela operação, com a
// MESMA guarda do supervisor (SEC-09) e sem resposta que pareça sucesso quando não é.
func setupSupervisorIncidentAPI(t *testing.T) (*agentAPI, *agent.Supervisor) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	supervisor := runtime.Supervisor()
	if supervisor == nil {
		t.Fatal("o runtime precisa ter supervisor")
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	return api, supervisor
}

func incidentContext(t *testing.T, method, target, body, id string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if id != "" {
		ctx.Params = gin.Params{{Key: "id", Value: id}}
	}
	return ctx, recorder
}

func TestSupervisorIncidentRoutesListAndReportMissingExplicitly(t *testing.T) {
	api, supervisor := setupSupervisorIncidentAPI(t)

	// LISTAGEM: responde mesmo vazia, sem inventar incidente.
	listCtx, listRecorder := incidentContext(t, http.MethodGet, "/api/agent/v1/supervisor/incidents", "", "")
	api.supervisorIncidents(listCtx)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var payload struct {
		Count     int                        `json:"count"`
		Incidents []agent.SupervisorIncident `json:"incidents"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != len(payload.Incidents) {
		t.Fatalf("a contagem precisa bater com a lista: %+v", payload)
	}

	// Incidente real pelo caminho de produção: pausa por risco de uma empresa.
	if _, err := supervisor.AcknowledgeIncident("inc_nao_existe", "ana", time.Now().UTC()); err == nil {
		t.Fatal("ciência de incidente inexistente precisa falhar no motor")
	}

	// ACKNOWLEDGE de inexistente: 404 explícito, não 200 vazio.
	ackCtx, ackRecorder := incidentContext(t, http.MethodPost, "/api/agent/v1/supervisor/incidents/inc_nao_existe/acknowledge", "", "inc_nao_existe")
	api.supervisorAcknowledgeIncident(ackCtx)
	if ackRecorder.Code != http.StatusNotFound {
		t.Fatalf("acknowledge de inexistente precisa ser 404, veio %d (%s)", ackRecorder.Code, ackRecorder.Body.String())
	}

	// RESOLVE de inexistente: 404 e a resposta NÃO pode parecer resolvida.
	resolveCtx, resolveRecorder := incidentContext(t, http.MethodPost, "/api/agent/v1/supervisor/incidents/inc_nao_existe/resolve", `{"resolution":"nada"}`, "inc_nao_existe")
	api.supervisorResolveIncident(resolveCtx)
	if resolveRecorder.Code != http.StatusNotFound {
		t.Fatalf("resolve de inexistente precisa ser 404, veio %d (%s)", resolveRecorder.Code, resolveRecorder.Body.String())
	}
	if strings.Contains(resolveRecorder.Body.String(), "resolved") {
		t.Fatalf("resposta de erro não pode parecer sucesso: %s", resolveRecorder.Body.String())
	}
}

func TestSupervisorIncidentRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	found := map[string]bool{}
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{
		"GET /api/agent/v1/supervisor/incidents",
		"POST /api/agent/v1/supervisor/incidents/:id/acknowledge",
		"POST /api/agent/v1/supervisor/incidents/:id/resolve",
	} {
		if !found[want] {
			t.Fatalf("rota %s não está registrada", want)
		}
	}
}
