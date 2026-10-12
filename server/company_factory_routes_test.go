package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 23 pela ROTA: o lançamento devolve artefatos gravados em disco, preço derivado
// e checklist honesto — e NÃO cria cliente.
func TestCompanyFactoryRouteMaterializesArtifactsAndDeclaresBlockers(t *testing.T) {
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
	group.POST("/company/factory/launch", api.launchCompanyFactory)
	group.GET("/company/factory/report/:id", api.companyFactoryReport)

	body := `{
		"company": {"name":"Camisaria Sandbox","mission":"vender camisetas","positioning":"local",
			"business_model":"venda direta","target_audience":"adultos","offer":"camiseta em 5 dias","currency":"BRL"},
		"tagline":"sua estampa",
		"palette":["#0b3d2e"],
		"products":[{"sku":"CAM-001","name":"Camiseta","cost_cents":3500,"margin_percent":60,"initial_stock":10}]
	}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/company/factory/launch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("lançamento precisa ser 201, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var report agent.CompanyFactoryReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.CompanyID == "" || len(report.Products) != 1 || report.Products[0].PriceCents != 5600 {
		t.Fatalf("relatório incorreto: %+v", report)
	}
	if len(report.Artifacts) < 4 {
		t.Fatalf("os quatro artefatos precisam existir: %+v", report.Artifacts)
	}
	for _, artifact := range report.Artifacts {
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatalf("artefato %s precisa existir: %v", artifact.Path, err)
		}
	}
	// A ROTA não cria cliente: o CRM fica vazio.
	if contacts := runtime.CRM().ListContacts(agent.LocalOrganizationID, ""); len(contacts) != 0 {
		t.Fatalf("a fábrica não pode inventar clientes: %+v", contacts)
	}
	// Checklist com bloqueio e procedimento declarados.
	blocked := 0
	for _, item := range report.Checklist {
		if item.Status == agent.CompanyChecklistBlocked {
			blocked++
			if item.Unblock == "" {
				t.Fatalf("bloqueio sem procedimento: %+v", item)
			}
		}
	}
	if blocked == 0 {
		t.Fatalf("o checklist precisa declarar o que depende de terceiros: %+v", report.Checklist)
	}

	// Relatório reconstruído do estado real.
	reload := httptest.NewRecorder()
	engine.ServeHTTP(reload, httptest.NewRequest(http.MethodGet, "/api/agent/v1/company/factory/report/"+report.CompanyID, nil))
	if reload.Code != http.StatusOK {
		t.Fatalf("relatório precisa ser 200, veio %d (%s)", reload.Code, reload.Body.String())
	}
	// Empresa inexistente: 404.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/company/factory/report/co_inexistente", nil))
	// NÃO INFERÊNCIA (E-190): a empresa inexistente recebia 404 e a empresa de OUTRO tenant recebia 403 — a
	// diferença revelava a existência do ID. A recusa agora é a MESMA (403) para os dois casos; a propriedade é
	// provada de forma genérica em `company_scope_non_inference_test.go`.
	if missing.Code != http.StatusNotFound {
		t.Fatalf("empresa inexistente precisa da recusa unificada (404 nesta rota), veio %d", missing.Code)
	}
	// Margem inválida: 400 (erro do cliente, não do servidor).
	bad := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/company/factory/launch",
		strings.NewReader(`{"company":{"name":"X"},"products":[{"sku":"A","name":"A","cost_cents":100,"margin_percent":100}]}`))
	badRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(bad, badRequest)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("margem de 100%% precisa ser 400, veio %d (%s)", bad.Code, bad.Body.String())
	}
	// Corpo inválido: 400.
	bogus := httptest.NewRecorder()
	bogusRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/company/factory/launch", strings.NewReader(`{"company":`))
	bogusRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(bogus, bogusRequest)
	if bogus.Code != http.StatusBadRequest {
		t.Fatalf("corpo inválido precisa ser 400, veio %d", bogus.Code)
	}
}
