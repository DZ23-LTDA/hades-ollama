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

// Estágio 27 pelas ROTAS: proposta com preço do catálogo, envio condicionado a
// consentimento, decisão com vigência e pós-venda com atraso consultável.
func TestSalesRoutesProposalLifecycleAndFollowUps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := shortServerRoot(t)
	dataRoot := shortServerRoot(t)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: workspace, DataRoot: dataRoot})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/sales/proposals", api.createSalesProposal)
	group.GET("/sales/proposals", api.listSalesProposals)
	group.GET("/sales/proposals/:id", api.getSalesProposal)
	group.POST("/sales/proposals/:id/send", api.sendSalesProposal)
	group.POST("/sales/proposals/:id/decide", api.decideSalesProposal)
	group.POST("/sales/follow-ups", api.createSalesFollowUp)
	group.GET("/sales/follow-ups", api.listSalesFollowUps)
	group.POST("/sales/follow-ups/:id/complete", api.completeSalesFollowUp)

	now := time.Now().UTC()
	organization := agent.LocalOrganizationID
	if _, err := runtime.Commerce().UpsertProduct(agent.CommerceProduct{
		OrganizationID: organization, SKU: "ROTA-1", Name: "Produto de rota", PriceCents: 4200, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}

	// O corpo TENTA mandar preço; o servidor precisa IGNORAR e usar o do catálogo.
	body := `{"contact_id":"contato-rota","items":[{"sku":"ROTA-1","quantity":3,"unit_price_cents":1}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("criação precisa ser 201, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var proposal agent.SalesProposal
	if err := json.Unmarshal(recorder.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	// 3 × 4200 = 12600: o preço do CORPO (1 centavo) foi ignorado.
	if proposal.TotalCents != 12600 {
		t.Fatalf("preço do corpo precisa ser IGNORADO; total veio %d", proposal.TotalCents)
	}

	// SKU desconhecido: 422 (o pedido está bem formado, o conteúdo é que não existe).
	badSKU := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals",
		strings.NewReader(`{"contact_id":"contato-rota","items":[{"sku":"NAO-EXISTE","quantity":1}]}`))
	badRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(badSKU, badRequest)
	if badSKU.Code != http.StatusUnprocessableEntity {
		t.Fatalf("SKU desconhecido precisa ser 422, veio %d (%s)", badSKU.Code, badSKU.Body.String())
	}

	// SEM consentimento: 403.
	send := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals/"+proposal.ID+"/send",
			strings.NewReader(`{"channel":"whatsapp"}`))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := send(); code != http.StatusForbidden {
		t.Fatalf("envio sem consentimento precisa ser 403, veio %d", code)
	}
	if _, err := runtime.Consent().GrantConsent(organization, "whatsapp", "contato-rota", "formulario", "aceite", "operador", nil, now); err != nil {
		t.Fatal(err)
	}
	if code := send(); code != http.StatusOK {
		t.Fatalf("envio com consentimento precisa ser 200, veio %d", code)
	}

	// Decidir com decisão desconhecida: 400.
	bogus := httptest.NewRecorder()
	bogusReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals/"+proposal.ID+"/decide",
		strings.NewReader(`{"decision":"talvez","actor":"cliente"}`))
	bogusReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(bogus, bogusReq)
	if bogus.Code != http.StatusBadRequest {
		t.Fatalf("decisão desconhecida precisa ser 400, veio %d (%s)", bogus.Code, bogus.Body.String())
	}

	// Aceitar: 200 e estado aceita.
	decide := httptest.NewRecorder()
	decideReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals/"+proposal.ID+"/decide",
		strings.NewReader(`{"decision":"aceitar","actor":"cliente"}`))
	decideReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(decide, decideReq)
	if decide.Code != http.StatusOK {
		t.Fatalf("aceitar precisa ser 200, veio %d (%s)", decide.Code, decide.Body.String())
	}
	var decided agent.SalesProposal
	if err := json.Unmarshal(decide.Body.Bytes(), &decided); err != nil {
		t.Fatal(err)
	}
	if decided.Status != agent.SalesProposalAccepted {
		t.Fatalf("proposta precisa ficar aceita: %+v", decided)
	}
	// Decidir DE NOVO: 409 (decisão terminal).
	again := httptest.NewRecorder()
	againReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/proposals/"+proposal.ID+"/decide",
		strings.NewReader(`{"decision":"recusar","actor":"cliente"}`))
	againReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(again, againReq)
	if again.Code != http.StatusConflict {
		t.Fatalf("decisão repetida precisa ser 409, veio %d", again.Code)
	}

	// Proposta inexistente: 404.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/sales/proposals/prop_inexistente", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("proposta inexistente precisa ser 404, veio %d", missing.Code)
	}

	// Listagem com filtro.
	list := httptest.NewRecorder()
	engine.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/agent/v1/sales/proposals?status=aceita", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"count":1`) {
		t.Fatalf("listagem filtrada precisa trazer 1: %d %s", list.Code, list.Body.String())
	}

	// PÓS-VENDA: criar, listar sem atraso, concluir.
	followUpRec := httptest.NewRecorder()
	followUpReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/follow-ups",
		strings.NewReader(`{"contact_id":"contato-rota","order_id":"ord-rota","reason":"confirmar satisfação"}`))
	followUpReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(followUpRec, followUpReq)
	if followUpRec.Code != http.StatusCreated {
		t.Fatalf("pós-venda precisa ser 201, veio %d (%s)", followUpRec.Code, followUpRec.Body.String())
	}
	var followUp agent.SalesFollowUp
	if err := json.Unmarshal(followUpRec.Body.Bytes(), &followUp); err != nil {
		t.Fatal(err)
	}
	// Motivo vazio é 409 pelo domínio (ErrFollowUpInvalid mapeado como conflito de estado
	// de entrada) — o importante é NÃO criar em silêncio.
	noReason := httptest.NewRecorder()
	noReasonReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/follow-ups",
		strings.NewReader(`{"contact_id":"contato-rota","reason":"   "}`))
	noReasonReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(noReason, noReasonReq)
	if noReason.Code != http.StatusBadRequest {
		t.Fatalf("motivo vazio é erro de ENTRADA e precisa ser 400, veio %d (%s)", noReason.Code, noReason.Body.String())
	}
	// Sem atraso ainda.
	overdue := httptest.NewRecorder()
	engine.ServeHTTP(overdue, httptest.NewRequest(http.MethodGet, "/api/agent/v1/sales/follow-ups?overdue=true", nil))
	if !strings.Contains(overdue.Body.String(), `"count":0`) {
		t.Fatalf("não pode haver atraso antes do prazo: %s", overdue.Body.String())
	}
	// Concluir: 200.
	done := httptest.NewRecorder()
	doneReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/follow-ups/"+followUp.ID+"/complete",
		strings.NewReader(`{}`))
	doneReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(done, doneReq)
	if done.Code != http.StatusOK {
		t.Fatalf("concluir pós-venda precisa ser 200, veio %d (%s)", done.Code, done.Body.String())
	}
	// Concluir de novo: 409.
	doneAgain := httptest.NewRecorder()
	doneAgainReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/follow-ups/"+followUp.ID+"/complete",
		strings.NewReader(`{}`))
	doneAgainReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(doneAgain, doneAgainReq)
	if doneAgain.Code != http.StatusConflict {
		t.Fatalf("concluir duas vezes precisa ser 409, veio %d", doneAgain.Code)
	}
	// Acompanhamento inexistente: 404.
	missingFollowUp := httptest.NewRecorder()
	missingReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sales/follow-ups/psf_inexistente/complete",
		strings.NewReader(`{}`))
	missingReq.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(missingFollowUp, missingReq)
	if missingFollowUp.Code != http.StatusNotFound {
		t.Fatalf("pós-venda inexistente precisa ser 404, veio %d", missingFollowUp.Code)
	}
}
