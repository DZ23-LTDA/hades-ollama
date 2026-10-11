package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Trilha DURÁVEL pela ROTA (estágio 35): a auditoria precisa ser ALCANÇÁVEL, e a resposta precisa declarar
// o estado real da persistência — uma trilha "em memória" apresentada como durável seria pior que a ausência.
func TestServiceAgentAuditRouteExposesDurableTrail(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, true)

	// Um turno com duas tentativas: uma que executa e uma BLOQUEADA (o recepcionista não declara
	// `inbox.assign`) — a tentativa bloqueada também é evidência de auditoria.
	body := `{"conversation_id":"` + conversationID + `","intent":"quero saber do pedido",` +
		`"requested_tools":["crm.contact.read","inbox.assign"]}`
	turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/receptionist/turn", body)
	if turn.Code != http.StatusOK {
		t.Fatalf("turno precisa responder 200, veio %d: %s", turn.Code, turn.Body.String())
	}

	audit := httptest.NewRecorder()
	engine.ServeHTTP(audit, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/audit", nil))
	if audit.Code != http.StatusOK {
		t.Fatalf("auditoria precisa responder 200, veio %d: %s", audit.Code, audit.Body.String())
	}
	texto := audit.Body.String()
	if !strings.Contains(texto, "crm.contact.read") || !strings.Contains(texto, "inbox.assign") {
		t.Fatalf("a trilha precisa registrar as duas tentativas: %s", texto)
	}
	if !strings.Contains(texto, "blocked") {
		t.Fatalf("a tentativa bloqueada precisa constar: %s", texto)
	}
	if !strings.Contains(texto, "\"persisted\":true") || !strings.Contains(texto, "\"storage\"") {
		t.Fatalf("a resposta precisa declarar o estado da persistência: %s", texto)
	}
}
