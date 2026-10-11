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

// Estágio 31 pela ROTA: o ciclo roda, a ação irreversível NÃO executa sozinha e a
// decisão humana fica registrada.
func setupBusinessSupervisorAPI(t *testing.T) (*gin.Engine, *agent.Runtime) {
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
	if _, err := runtime.CompanyStore().Create(agent.Company{
		OrganizationID: agent.LocalOrganizationID, Name: "Loja", Currency: "BRL",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: agent.LocalOrganizationID, SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(agent.LocalOrganizationID, "CAM-001", 10, agent.CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	// Pedido pago, receita lançada e cancelado: gera ESTORNO DEVIDO, que exige humano.
	order, err := runtime.Commerce().CreateOrder(agent.CommerceOrderRequest{
		OrganizationID: agent.LocalOrganizationID, IdempotencyKey: "api-sup-1",
		Items: []agent.CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().ConfirmPayment(agent.LocalOrganizationID, order.ID,
		agent.CommercePayment{Method: agent.CommercePaymentSandbox, Reference: "r1"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Finance().PostOrderRevenue(agent.LocalOrganizationID, order.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().Cancel(agent.LocalOrganizationID, order.ID, "desistiu", now); err != nil {
		t.Fatal(err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/business/tick", api.businessSupervisorTick)
	group.GET("/business/actions", api.listBusinessActions)
	group.POST("/business/actions/:id/decide", api.decideBusinessAction)
	return engine, runtime
}

func businessRequest(engine *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestBusinessSupervisorRoutesRunTickAndRequireHumanDecision(t *testing.T) {
	engine, runtime := setupBusinessSupervisorAPI(t)

	// Ciclo: observa os domínios e deixa o estorno aguardando decisão.
	tick := businessRequest(engine, http.MethodPost, "/api/agent/v1/business/tick", "")
	if tick.Code != http.StatusOK {
		t.Fatalf("ciclo precisa ser 200, veio %d (%s)", tick.Code, tick.Body.String())
	}
	var result agent.BusinessTickResult
	if err := json.Unmarshal(tick.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Signals) == 0 {
		t.Fatalf("o ciclo precisa trazer sinais: %+v", result)
	}
	if result.AwaitingApproval == 0 {
		t.Fatalf("o estorno devido precisa aguardar aprovação: %+v", result.Actions)
	}
	// A listagem de pendências traz a ação e a política vigente.
	list := businessRequest(engine, http.MethodGet, "/api/agent/v1/business/actions", "")
	if list.Code != http.StatusOK {
		t.Fatalf("listar pendências: %d", list.Code)
	}
	var pending struct {
		Pending []agent.BusinessAction `json:"pending"`
		Count   int                    `json:"count"`
		Policy  agent.SupervisorPolicy `json:"policy"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if pending.Count == 0 || pending.Policy.BudgetCents == 0 {
		t.Fatalf("pendências e política precisam vir na listagem: %+v", pending)
	}
	actionID := pending.Pending[0].ID

	// NENHUM estorno foi lançado antes da decisão humana.
	ledger := runtime.CompanyStore().List(agent.LocalOrganizationID)
	entries, err := runtime.CompanyStore().ListLedgerEntries(ledger[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Category == "estorno" {
			t.Fatalf("nada pode ser estornado sem decisão: %+v", entry)
		}
	}

	// Recusar não executa e fica registrado.
	deny := businessRequest(engine, http.MethodPost, "/api/agent/v1/business/actions/"+actionID+"/decide",
		`{"decision":"RECUSAR","reason":"não autorizado"}`)
	if deny.Code != http.StatusOK {
		t.Fatalf("recusar precisa ser 200, veio %d (%s)", deny.Code, deny.Body.String())
	}
	var denied agent.BusinessAction
	if err := json.Unmarshal(deny.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Status != agent.BusinessActionDenied {
		t.Fatalf("a recusa precisa mudar o estado: %+v", denied)
	}
	// Decisão inválida e ação inexistente têm códigos próprios.
	if got := businessRequest(engine, http.MethodPost, "/api/agent/v1/business/actions/"+actionID+"/decide",
		`{"decision":"TALVEZ"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("decisão inválida precisa ser 400, veio %d", got.Code)
	}
	if got := businessRequest(engine, http.MethodPost, "/api/agent/v1/business/actions/bact_inexistente/decide",
		`{"decision":"APROVAR"}`); got.Code != http.StatusNotFound {
		t.Fatalf("ação inexistente precisa ser 404, veio %d", got.Code)
	}
	if got := businessRequest(engine, http.MethodPost, "/api/agent/v1/business/actions/"+actionID+"/decide", `{"decision":`); got.Code != http.StatusBadRequest {
		t.Fatalf("corpo inválido precisa ser 400, veio %d", got.Code)
	}
}
