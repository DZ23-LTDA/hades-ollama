package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 30: rotas financeiras ligadas ao comércio. A receita só entra com pedido
// PAGO, e a lacuna entre comércio e financeiro aparece MEDIDA no relatório.
func setupFinanceAPI(t *testing.T) (*gin.Engine, *agent.Runtime, string) {
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
	now := time.Now().UTC()
	// A empresa é quem recebe os lançamentos; a organização é a do comércio.
	if _, err := runtime.CompanyStore().Create(agent.Company{
		OrganizationID: agent.LocalOrganizationID, Name: "Loja", Currency: "BRL",
	}); err != nil {
		t.Fatalf("create company: %v", err)
	}
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: agent.LocalOrganizationID, SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(agent.LocalOrganizationID, "CAM-001", 5, agent.CommerceMovementEntry, "lote", now); err != nil {
		t.Fatal(err)
	}
	order, err := runtime.Commerce().CreateOrder(agent.CommerceOrderRequest{
		OrganizationID: agent.LocalOrganizationID, IdempotencyKey: "api-fin-1",
		Items: []agent.CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/finance/orders/:id/revenue", api.postOrderRevenue)
	group.POST("/finance/orders/:id/cost", api.postOrderCost)
	group.POST("/finance/orders/:id/reconcile", api.reconcileOrderWithStatement)
	group.GET("/finance/report", api.financeReport)
	return engine, runtime, order.ID
}

func financeRequest(engine *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestFinanceRoutesRefuseUnpaidOrderAndRequirePaymentFirst(t *testing.T) {
	engine, runtime, orderID := setupFinanceAPI(t)

	// Pedido aguardando pagamento: 409, e NADA lançado.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/revenue", ""); got.Code != http.StatusConflict {
		t.Fatalf("receita de pedido não pago precisa ser 409, veio %d (%s)", got.Code, got.Body.String())
	}
	report := financeRequest(engine, http.MethodGet, "/api/agent/v1/finance/report", "")
	var parsed agent.FinanceReport
	if err := json.Unmarshal(report.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Ledger.IncomeCents != 0 {
		t.Fatalf("não pode existir receita de pedido não pago: %+v", parsed.Ledger)
	}
	if parsed.ReceivableCents != 7900 || len(parsed.Receivables) != 1 {
		t.Fatalf("contas a receber precisam aparecer: %+v", parsed)
	}

	// Pagando, a receita entra: 201 na primeira vez e 200 na repetição (idempotente).
	if _, err := runtime.Commerce().ConfirmPayment(agent.LocalOrganizationID, orderID,
		agent.CommercePayment{Method: agent.CommercePaymentPix, Reference: "E123"}, "ana", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	first := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/revenue", "")
	if first.Code != http.StatusCreated {
		t.Fatalf("primeira receita precisa ser 201, veio %d (%s)", first.Code, first.Body.String())
	}
	repeat := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/revenue", "")
	if repeat.Code != http.StatusOK {
		t.Fatalf("repetição precisa ser 200, veio %d (%s)", repeat.Code, repeat.Body.String())
	}
	var posting agent.FinanceOrderPosting
	if err := json.Unmarshal(repeat.Body.Bytes(), &posting); err != nil {
		t.Fatal(err)
	}
	if posting.Created {
		t.Fatal("a repetição não pode criar um segundo lançamento")
	}

	// Custo do pedido: 201 e idempotente.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/cost",
		`{"amount_cents":1500,"category":"frete","reference":"transportadora"}`); got.Code != http.StatusCreated {
		t.Fatalf("custo precisa ser 201, veio %d (%s)", got.Code, got.Body.String())
	}
	// Custo negativo/zero: 400.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/cost",
		`{"amount_cents":0,"category":"frete"}`); got.Code == http.StatusCreated {
		t.Fatalf("custo zero não pode ser aceito: %s", got.Body.String())
	}

	// Conciliação com o extrato: 200; referência divergente é 409.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/reconcile",
		`{"reference":"extrato-1"}`); got.Code != http.StatusOK {
		t.Fatalf("conciliação precisa ser 200, veio %d (%s)", got.Code, got.Body.String())
	}
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/reconcile",
		`{"reference":"extrato-2"}`); got.Code != http.StatusConflict {
		t.Fatalf("conciliação divergente precisa ser 409, veio %d", got.Code)
	}
	// Conciliação sem referência: 400.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/"+orderID+"/reconcile", `{}`); got.Code != http.StatusBadRequest {
		t.Fatalf("conciliação sem referência precisa ser 400, veio %d", got.Code)
	}

	// No fim: receita lançada, custo lançado, conciliado e SEM divergências.
	final := financeRequest(engine, http.MethodGet, "/api/agent/v1/finance/report", "")
	if err := json.Unmarshal(final.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Ledger.IncomeCents != 7900 || parsed.Ledger.ExpenseCents != 1500 {
		t.Fatalf("razão incorreto: %+v", parsed.Ledger)
	}
	if parsed.Ledger.NetCents != 6400 {
		t.Fatalf("saldo incorreto: %+v", parsed.Ledger)
	}
	if len(parsed.Divergences) != 0 || parsed.UnpostedRevenue != 0 {
		t.Fatalf("sem divergências ao final: %+v", parsed)
	}
	// Pedido inexistente: 404 explícito.
	if got := financeRequest(engine, http.MethodPost, "/api/agent/v1/finance/orders/ord_inexistente/revenue", ""); got.Code != http.StatusNotFound {
		t.Fatalf("pedido inexistente precisa ser 404, veio %d", got.Code)
	}
}
