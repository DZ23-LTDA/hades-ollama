package server

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 32 pela ROTA: o export precisa baixar como CSV, ser parseável e respeitar o escopo da sessão.
func TestBusinessExportRouteDownloadsValidCSV(t *testing.T) {
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
	engine.GET("/api/agent/v1/companies/:id/business-export", api.businessExport)

	company, err := runtime.CompanyStore().CreateRequest(agent.CompanyCreateRequest{
		Name: "Loja de export", Mission: "vender", Currency: "BRL",
	}, agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: company.OrganizationID, SKU: "EXP-1", Name: "Produto", PriceCents: 1000, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/"+company.ID+"/business-export", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("export precisa ser 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	// Cabeçalhos de download: sem esses dois o navegador abriria o texto em vez de baixar o arquivo.
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/csv") {
		t.Fatalf("Content-Type precisa ser CSV: %q", contentType)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); !strings.Contains(disposition, "attachment") || !strings.Contains(disposition, ".csv") {
		t.Fatalf("Content-Disposition precisa ser anexo de CSV: %q", disposition)
	}

	// O corpo precisa ser CSV real com o esquema uniforme e as lacunas declaradas.
	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("corpo do export precisa ser CSV válido: %v", err)
	}
	if len(records) < 10 {
		t.Fatalf("export curto demais: %d linhas", len(records))
	}
	for index, record := range records {
		if len(record) != 4 {
			t.Fatalf("linha %d com %d campos (esperado 4): %+v", index, len(record), record)
		}
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "nao_medido") {
		t.Fatal("o export precisa levar a lista do que NÃO é medido")
	}
	if !strings.Contains(body, "indicador,produtos,1,") {
		t.Fatalf("o export precisa trazer os indicadores da empresa da sessão:\n%s", body)
	}

	// Empresa inexistente: 404, sem exportar dado de outra organização.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/co_inexistente/business-export", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("empresa inexistente precisa ser 404, veio %d", missing.Code)
	}
}
