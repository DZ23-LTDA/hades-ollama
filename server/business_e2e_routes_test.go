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

// Estágio 38 pela ROTA: a certificação é superfície real do produto, e a resposta
// carrega evidência por etapa, prova de isolamento e o que depende do externo.
func TestBusinessCertificationRouteReturnsPerStepEvidence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/business/certify", api.certifyBusiness)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/business/certify",
		strings.NewReader(`{"other_organization_id":"org-concorrente"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("certificação precisa ser 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var report agent.BusinessJourneyReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Verdict != agent.BusinessVerdictOK {
		t.Fatalf("veredito precisa ser aprovado: %+v", report)
	}
	if len(report.Steps) < 18 {
		t.Fatalf("o relatório precisa trazer as etapas: %d", len(report.Steps))
	}
	if !report.IsolationProven || !report.RecoveryProven {
		t.Fatalf("isolamento e recuperação precisam estar provados: %+v", report)
	}
	if len(report.NotDone) == 0 {
		t.Fatal("o relatório precisa declarar o que não foi feito")
	}
	// Corpo inválido: 400.
	bad := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/business/certify", strings.NewReader(`{"other_organization_id":`))
	badRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(bad, badRequest)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("corpo inválido precisa ser 400, veio %d", bad.Code)
	}
}
