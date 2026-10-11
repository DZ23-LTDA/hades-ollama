package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Aprovação humana pela ROTA (estágio 35): o humano precisa conseguir VER o pedido e DECIDIR.
//
// Uma aprovação que só existe no código Go não é aprovação: sem listagem e sem decisão alcançáveis, a
// escrita ficaria pendente para sempre — e um agente travado é indistinguível de um agente quebrado.
func TestServiceAgentApprovalRoutesListAndDecide(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, true)

	// 1. Um turno pede uma ESCRITA: fica pendente de decisão.
	pedido := `{"conversation_id":"` + conversationID + `","intent":"quero o plano",` +
		`"requested_tools":["crm.opportunity.write"],` +
		`"tool_arguments":{"title":"Plano avançado","value_cents":79900,"owner_id":"ana"}}`
	turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn", pedido)
	if turn.Code != http.StatusOK {
		t.Fatalf("turno precisa responder 200, veio %d: %s", turn.Code, turn.Body.String())
	}
	if !strings.Contains(turn.Body.String(), "approval_required") {
		t.Fatalf("a escrita sem aprovação precisa ficar pendente: %s", turn.Body.String())
	}

	// 2. O humano VÊ o pedido pendente.
	lista := httptest.NewRecorder()
	engine.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=pending", nil))
	if lista.Code != http.StatusOK || !strings.Contains(lista.Body.String(), "crm.opportunity.write") {
		t.Fatalf("a listagem precisa mostrar o pedido pendente (veio %d): %s", lista.Code, lista.Body.String())
	}
	approvalID := extrairApprovalID(t, lista.Body.String())
	if approvalID == "" {
		t.Fatalf("o pedido precisa trazer id: %s", lista.Body.String())
	}

	// Status inválido é ERRO, não lista vazia: vazio pareceria "nada pendente".
	invalido := httptest.NewRecorder()
	engine.ServeHTTP(invalido, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=inventado", nil))
	if invalido.Code != http.StatusBadRequest {
		t.Fatalf("status inválido precisa ser 400, veio %d", invalido.Code)
	}

	// 3. Decidir SEM ator é recusado: aprovação anônima não é aprovação.
	anonimo := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide", `{"approved":true}`)
	if anonimo.Code != http.StatusBadRequest {
		t.Fatalf("decisão sem ator precisa ser 400, veio %d: %s", anonimo.Code, anonimo.Body.String())
	}

	// 4. Pedido inexistente é 404.
	fantasma := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/apr_nao_existe/decide", `{"approved":true,"actor_id":"ana"}`)
	if fantasma.Code != http.StatusNotFound {
		t.Fatalf("pedido inexistente precisa ser 404, veio %d: %s", fantasma.Code, fantasma.Body.String())
	}

	// 5. Decisão humana legítima.
	decidido := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide",
		`{"approved":true,"actor_id":"ana","reason":"cliente pediu por escrito"}`)
	if decidido.Code != http.StatusOK || !strings.Contains(decidido.Body.String(), "approved") {
		t.Fatalf("decisão precisa ser aceita (veio %d): %s", decidido.Code, decidido.Body.String())
	}

	// 6. Repetir o turno agora EXECUTA a escrita aprovada.
	turn = serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn", pedido)
	if turn.Code != http.StatusOK || !strings.Contains(turn.Body.String(), "executed") {
		t.Fatalf("com aprovação o turno precisa executar (veio %d): %s", turn.Code, turn.Body.String())
	}

	// 7. Decidir de novo sobre a mesma aprovação já CONSUMIDA é conflito, não sucesso silencioso.
	repetida := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide",
		`{"approved":true,"actor_id":"ana"}`)
	if repetida.Code != http.StatusConflict {
		t.Fatalf("decisão repetida precisa ser 409, veio %d: %s", repetida.Code, repetida.Body.String())
	}
}

// extrairApprovalID pega o primeiro id de aprovação do corpo (formato estável: "id":"apr_...").
func extrairApprovalID(t *testing.T, body string) string {
	t.Helper()
	const marca = `"id":"`
	i := strings.Index(body, marca)
	if i < 0 {
		return ""
	}
	resto := body[i+len(marca):]
	fim := strings.Index(resto, `"`)
	if fim < 0 {
		return ""
	}
	return resto[:fim]
}
