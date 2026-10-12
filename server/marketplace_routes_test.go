package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 25 pelas ROTAS: catálogo declarando estado real, sincronização contra o duble de sandbox,
// teto de operações como 429 e nenhuma publicação externa.
func TestMarketplaceRoutesCatalogSyncAndMirror(t *testing.T) {
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
	group := engine.Group("/api/agent/v1")
	group.GET("/companies/:id/marketplace/channels", api.marketplaceChannels)
	group.POST("/companies/:id/marketplace/sync", api.marketplaceSync)
	group.GET("/companies/:id/marketplace/mirror/:channel_id/:sku", api.marketplaceMirror)

	company, err := runtime.CompanyStore().CreateRequest(agent.CompanyCreateRequest{
		Name: "Loja de Marketplace", Mission: "vender", Currency: "BRL",
	}, agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/agent/v1/companies/" + company.ID + "/marketplace"

	// CATÁLOGO: os canais existem e declaram estado real, sem "pronto para usar" inventado.
	catalogRecorder := httptest.NewRecorder()
	engine.ServeHTTP(catalogRecorder, httptest.NewRequest(http.MethodGet, base+"/channels", nil))
	if catalogRecorder.Code != http.StatusOK {
		t.Fatalf("catálogo precisa ser 200, veio %d (%s)", catalogRecorder.Code, catalogRecorder.Body.String())
	}
	var catalog struct {
		Channels []agent.MarketplaceChannelSpec `json:"channels"`
	}
	if err := json.Unmarshal(catalogRecorder.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Channels) == 0 {
		t.Fatal("catálogo vazio: os canais precisam ser listados")
	}
	for _, spec := range catalog.Channels {
		if spec.LiveTested {
			t.Fatalf("canal %s não pode declarar teste live sem credencial", spec.ID)
		}
		if spec.State != agent.MarketplaceStateConfigRequired && spec.State != agent.MarketplaceStateUnauthorized {
			t.Fatalf("canal %s com estado inesperado: %s", spec.ID, spec.State)
		}
		if spec.Unblock == "" || len(spec.Capabilities) == 0 {
			t.Fatalf("canal %s precisa de procedimento e capacidades: %+v", spec.ID, spec)
		}
	}

	// Produto e estoque locais para a sincronização ter o que espelhar.
	now := time.Now().UTC()
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: company.OrganizationID, SKU: "R-1", Name: "Produto de rota", PriceCents: 2500, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(company.OrganizationID, "R-1", 4, agent.CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}

	// SINCRONIZAÇÃO VÁLIDA: 200 e o relatório DECLARA sandbox e homologação bloqueada.
	syncBody := `{"channel_id":"mercado_livre","direction":"push","items":[{"entity":"estoque","sku":"R-1","version":1,"quantity":4}]}`
	syncRecorder := httptest.NewRecorder()
	syncRequest := httptest.NewRequest(http.MethodPost, base+"/sync", strings.NewReader(syncBody))
	syncRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(syncRecorder, syncRequest)
	if syncRecorder.Code != http.StatusOK {
		t.Fatalf("sincronização válida precisa ser 200, veio %d (%s)", syncRecorder.Code, syncRecorder.Body.String())
	}
	var report agent.MarketplaceSyncReport
	if err := json.Unmarshal(syncRecorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Applied != 1 || report.Conflicts != 0 {
		t.Fatalf("sincronização precisa aplicar 1: %+v", report)
	}
	if !report.Sandbox || report.LiveStatus != agent.MarketplaceLiveStatus || report.LiveReason == "" {
		t.Fatalf("relatório precisa declarar sandbox e bloqueio: %+v", report)
	}

	// IDEMPOTÊNCIA pela rota: reexecutar a mesma versão apenas ignora.
	repeatRecorder := httptest.NewRecorder()
	repeatRequest := httptest.NewRequest(http.MethodPost, base+"/sync", strings.NewReader(syncBody))
	repeatRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(repeatRecorder, repeatRequest)
	var repeated agent.MarketplaceSyncReport
	if err := json.Unmarshal(repeatRecorder.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if repeated.Applied != 0 || repeated.Skipped != 1 {
		t.Fatalf("reexecutar precisa ignorar: %+v", repeated)
	}

	// ESPELHO: mostra o que o duble recebeu.
	mirrorRecorder := httptest.NewRecorder()
	engine.ServeHTTP(mirrorRecorder, httptest.NewRequest(http.MethodGet, base+"/mirror/mercado_livre/R-1", nil))
	if mirrorRecorder.Code != http.StatusOK {
		t.Fatalf("espelho precisa ser 200, veio %d", mirrorRecorder.Code)
	}
	var mirror map[string]any
	if err := json.Unmarshal(mirrorRecorder.Body.Bytes(), &mirror); err != nil {
		t.Fatal(err)
	}
	if quantity, _ := mirror["quantity"].(float64); quantity != 4 {
		t.Fatalf("espelho precisa refletir o duble: %+v", mirror)
	}
	if sandbox, _ := mirror["sandbox"].(bool); !sandbox {
		t.Fatalf("o espelho precisa declarar sandbox: %+v", mirror)
	}

	// CANAL DESCONHECIDO: 404.
	unknownRecorder := httptest.NewRecorder()
	unknownRequest := httptest.NewRequest(http.MethodPost, base+"/sync",
		strings.NewReader(`{"channel_id":"marketplace_inexistente","items":[{"entity":"estoque","sku":"R-1","version":9,"quantity":1}]}`))
	unknownRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(unknownRecorder, unknownRequest)
	if unknownRecorder.Code != http.StatusNotFound {
		t.Fatalf("canal desconhecido precisa ser 404, veio %d (%s)", unknownRecorder.Code, unknownRecorder.Body.String())
	}

	// PEDIDO INVÁLIDO: 400 (sem itens).
	emptyRecorder := httptest.NewRecorder()
	emptyRequest := httptest.NewRequest(http.MethodPost, base+"/sync", strings.NewReader(`{"channel_id":"shopee","items":[]}`))
	emptyRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(emptyRecorder, emptyRequest)
	if emptyRecorder.Code != http.StatusBadRequest {
		t.Fatalf("pedido vazio precisa ser 400, veio %d", emptyRecorder.Code)
	}

	// TETO DE OPERAÇÕES: 429 — é limitação de taxa, não erro do pedido.
	var items []string
	for index := 0; index <= agent.MarketplaceMaxOperationsPerRun; index++ {
		items = append(items, fmt.Sprintf(`{"entity":"estoque","sku":"R-%d","version":1,"quantity":1}`, index))
	}
	overRecorder := httptest.NewRecorder()
	overRequest := httptest.NewRequest(http.MethodPost, base+"/sync",
		strings.NewReader(`{"channel_id":"shopee","items":[`+strings.Join(items, ",")+`]}`))
	overRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(overRecorder, overRequest)
	if overRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("acima do teto precisa ser 429, veio %d (%s)", overRecorder.Code, overRecorder.Body.String())
	}
	// E NADA foi aplicado, apesar do pedido grande.
	if quantity, present := runtime.Marketplace().RemoteStock("shopee", "R-0"); present {
		t.Fatalf("a recusa por teto não pode ter aplicado efeito (quantidade %d)", quantity)
	}

	// Empresa inexistente: 404.
	missingRecorder := httptest.NewRecorder()
	engine.ServeHTTP(missingRecorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/co_inexistente/marketplace/channels", nil))
	// NÃO INFERÊNCIA (E-190): a empresa inexistente recebia 404 e a empresa de OUTRO tenant recebia 403 — a
	// diferença revelava a existência do ID. A recusa agora é a MESMA (403) para os dois casos; a propriedade é
	// provada de forma genérica em `company_scope_non_inference_test.go`.
	if missingRecorder.Code != http.StatusForbidden {
		t.Fatalf("empresa inexistente precisa ser a recusa unificada 403, veio %d", missingRecorder.Code)
	}
}
