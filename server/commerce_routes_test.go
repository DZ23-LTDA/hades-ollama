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

// Estágio 24: rotas do motor de comércio. O checkout exige chave de idempotência e o
// pagamento aceita apenas REFERÊNCIA — nunca dado de cartão.
func setupCommerceAPI(t *testing.T) (*gin.Engine, *agent.Runtime) {
	t.Helper()
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
	group.POST("/commerce/products", api.upsertCommerceProduct)
	group.GET("/commerce/products", api.listCommerceProducts)
	group.POST("/commerce/stock", api.recordCommerceStock)
	group.GET("/commerce/stock/:sku", api.commerceStock)
	group.POST("/commerce/orders", api.createCommerceOrder)
	group.GET("/commerce/orders", api.listCommerceOrders)
	group.GET("/commerce/orders/:id", api.getCommerceOrder)
	group.POST("/commerce/orders/:id/payment", api.confirmCommercePayment)
	group.POST("/commerce/orders/:id/reconcile", api.reconcileCommerceOrder)
	group.POST("/commerce/orders/:id/fulfill", api.fulfillCommerceOrder)
	group.POST("/commerce/orders/:id/cancel", api.cancelCommerceOrder)
	group.GET("/commerce/summary", api.commerceSummary)
	return engine, runtime
}

func commerceRequest(engine *gin.Engine, method, target, body, idempotencyKey string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestCommerceRoutesCoverCatalogStockAndIdempotentCheckout(t *testing.T) {
	engine, runtime := setupCommerceAPI(t)

	// Catálogo + entrada de estoque.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/products",
		`{"sku":"CAM-001","name":"Camiseta","price_cents":7900}`, ""); got.Code != http.StatusCreated {
		t.Fatalf("cadastrar produto: %d (%s)", got.Code, got.Body.String())
	}
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/stock",
		`{"sku":"CAM-001","delta":10,"kind":"entrada","reference":"lote 1"}`, ""); got.Code != http.StatusCreated {
		t.Fatalf("entrada de estoque: %d (%s)", got.Code, got.Body.String())
	}
	var stock struct {
		Balance agent.CommerceStockBalance `json:"balance"`
	}
	balanceBody := commerceRequest(engine, http.MethodGet, "/api/agent/v1/commerce/stock/CAM-001", "", "")
	if err := json.Unmarshal(balanceBody.Body.Bytes(), &stock); err != nil {
		t.Fatal(err)
	}
	if stock.Balance.OnHand != 10 || stock.Balance.Available != 10 {
		t.Fatalf("saldo após entrada: %+v", stock.Balance)
	}

	// Checkout com chave: 201; reenvio com a MESMA chave: 200 com o MESMO pedido.
	body := `{"contact_id":"ct_1","items":[{"sku":"CAM-001","quantity":2}]}`
	first := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders", body, "key-route-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("checkout precisa ser 201, veio %d (%s)", first.Code, first.Body.String())
	}
	var created struct {
		Order      agent.CommerceOrder `json:"order"`
		Duplicated bool                `json:"duplicated"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	repeat := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders", body, "key-route-1")
	if repeat.Code != http.StatusOK {
		t.Fatalf("reenvio do checkout precisa ser 200, veio %d (%s)", repeat.Code, repeat.Body.String())
	}
	var repeated struct {
		Order      agent.CommerceOrder `json:"order"`
		Duplicated bool                `json:"duplicated"`
	}
	if err := json.Unmarshal(repeat.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if !repeated.Duplicated || repeated.Order.ID != created.Order.ID {
		t.Fatalf("o reenvio precisa devolver o MESMO pedido: %+v", repeated)
	}
	// Sem chave: 400, porque não há como garantir a unicidade.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders", body, ""); got.Code != http.StatusBadRequest {
		t.Fatalf("checkout sem chave precisa ser 400, veio %d (%s)", got.Code, got.Body.String())
	}
	// Estoque acima do disponível: 409 (não se vende o que não existe).
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders",
		`{"items":[{"sku":"CAM-001","quantity":99}]}`, "key-route-2"); got.Code != http.StatusConflict {
		t.Fatalf("estoque insuficiente precisa ser 409, veio %d (%s)", got.Code, got.Body.String())
	}
	// Produto inexistente: 404.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders",
		`{"items":[{"sku":"NAO-EXISTE","quantity":1}]}`, "key-route-3"); got.Code != http.StatusNotFound {
		t.Fatalf("produto inexistente precisa ser 404, veio %d", got.Code)
	}

	// Pagamento por referência: aceito.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/payment",
		`{"method":"pix","reference":"E12345678"}`, ""); got.Code != http.StatusOK {
		t.Fatalf("pagamento por referência: %d (%s)", got.Code, got.Body.String())
	}
	// DADO DE CARTÃO: 422, e o pedido não muda de estado.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/payment",
		`{"method":"sandbox","reference":"4111111111111111"}`, ""); got.Code != http.StatusUnprocessableEntity {
		t.Fatalf("dado de cartão precisa ser 422, veio %d (%s)", got.Code, got.Body.String())
	}

	// Conciliação idempotente e conflito.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/reconcile",
		`{"reference":"extrato-1"}`, ""); got.Code != http.StatusOK {
		t.Fatalf("conciliação: %d (%s)", got.Code, got.Body.String())
	}
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/reconcile",
		`{"reference":"extrato-2"}`, ""); got.Code != http.StatusConflict {
		t.Fatalf("conciliação divergente precisa ser 409, veio %d", got.Code)
	}

	// Atendimento e resumo.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/fulfill", "", ""); got.Code != http.StatusOK {
		t.Fatalf("atender pedido: %d (%s)", got.Code, got.Body.String())
	}
	// Pedido atendido não pode ser cancelado: 409.
	if got := commerceRequest(engine, http.MethodPost, "/api/agent/v1/commerce/orders/"+created.Order.ID+"/cancel",
		`{"reason":"tarde"}`, ""); got.Code != http.StatusConflict {
		t.Fatalf("cancelar pedido atendido precisa ser 409, veio %d", got.Code)
	}
	summary := commerceRequest(engine, http.MethodGet, "/api/agent/v1/commerce/summary", "", "")
	var parsed agent.CommerceSummary
	if err := json.Unmarshal(summary.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Products != 1 || parsed.Fulfilled != 1 || parsed.RevenueCents != 15800 {
		t.Fatalf("resumo incorreto: %+v", parsed)
	}
	// O estoque físico caiu e o disponível também.
	after := runtime.Commerce().Stock(agent.LocalOrganizationID, "CAM-001")
	if after.OnHand != 8 || after.Reserved != 0 || after.Available != 8 {
		t.Fatalf("saldo após atendimento: %+v", after)
	}
}

// Com o motor de comércio implementado, o agente de comércio NÃO pode mais declarar
// suas ferramentas de leitura como bloqueadas.
func TestCommerceToolsAreNoLongerBlockedForTheCommerceAgent(t *testing.T) {
	engine, _ := setupCommerceAPI(t)
	t.Setenv("HADES_TEST_COMMERCE_SECRET", "segredo-de-teste-suficiente")
	_ = engine
	for _, tool := range []string{"commerce.order.read", "commerce.stock.read"} {
		if !agent.DefaultServiceAgentTools()[tool] {
			t.Fatalf("%s precisa estar disponível agora que o motor de comércio existe", tool)
		}
	}
	// E o catálogo do agente de comércio não pode mais listá-las como bloqueadas.
	catalog := agent.ServiceAgentCatalog()
	for _, spec := range catalog {
		if spec.ID != agent.ServiceAgentCommerce {
			continue
		}
		for _, tool := range spec.Tools {
			if strings.HasPrefix(tool, "commerce.") && !agent.DefaultServiceAgentTools()[tool] {
				t.Fatalf("ferramenta de comércio ainda bloqueada: %s", tool)
			}
		}
	}
}
