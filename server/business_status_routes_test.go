package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 32 pela ROTA: o painel devolve indicadores reais, a PROVENIÊNCIA de cada bloco e a lista
// do que NÃO é medido — e o escopo vem da sessão, não do cliente.
func TestBusinessStatusRouteReportsIndicatorsAndGaps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: shortServerRoot(t),
		DataRoot:      shortServerRoot(t),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	engine.GET("/api/agent/v1/companies/:id/business-status", api.businessStatus)

	company, err := runtime.CompanyStore().CreateRequest(agent.CompanyCreateRequest{
		Name: "Loja de BI", Mission: "vender", Currency: "BRL",
	}, agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: company.OrganizationID, SKU: "ROTA-1", Name: "Produto", PriceCents: 1000, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/"+company.ID+"/business-status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("painel precisa ser 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var report agent.BusinessStatusReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OrganizationID != company.OrganizationID {
		t.Fatalf("o escopo precisa vir da empresa da sessão: %q", report.OrganizationID)
	}
	if report.Commerce.Products != 1 {
		t.Fatalf("produtos incorretos: %+v", report.Commerce)
	}
	// PROVENIÊNCIA e LACUNAS DECLARADAS fazem parte do contrato da resposta.
	if len(report.Sources) < 5 {
		t.Fatalf("a resposta precisa declarar a origem dos blocos: %+v", report.Sources)
	}
	if len(report.NotMeasured) < 4 {
		t.Fatalf("a resposta precisa declarar o que NÃO mede: %+v", report.NotMeasured)
	}
	if !report.Sandbox {
		t.Fatal("o painel precisa declarar que opera em sandbox")
	}
	if report.Campaigns.PublicacaoExterna != agent.CampaignExternalPublication {
		t.Fatalf("publicação externa precisa vir declarada como bloqueada: %q", report.Campaigns.PublicacaoExterna)
	}

	// Empresa inexistente: 404, sem vazar indicadores de outra organização.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/co_inexistente/business-status", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("empresa inexistente precisa ser 404, veio %d", missing.Code)
	}
}
