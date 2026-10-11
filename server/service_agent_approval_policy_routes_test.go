package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// approversRequest chama a rota de configuração de aprovadores com o MÉTODO correto (PUT).
func approversRequest(engine *gin.Engine, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/agent/v1/service-agents/approval-approvers", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

// PRAZO e PAPEL da aprovação pelas ROTAS (estágio 35): a checagem de papel precisa ser CONFIGURÁVEL e a
// recusa precisa ter o status HTTP certo — 403 é "este ator não pode", não "pedido malformado".
func TestServiceAgentApprovalRoutesEnforceRoleAndListExpired(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, true)

	pedido := `{"conversation_id":"` + conversationID + `","intent":"quero o plano",` +
		`"requested_tools":["crm.opportunity.write"],` +
		`"tool_arguments":{"title":"Plano avançado","value_cents":79900,"owner_id":"ana"}}`
	if turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn", pedido); turn.Code != http.StatusOK {
		t.Fatalf("turno precisa responder 200, veio %d: %s", turn.Code, turn.Body.String())
	}
	lista := httptest.NewRecorder()
	engine.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=pending", nil))
	approvalID := extrairApprovalID(t, lista.Body.String())
	if approvalID == "" {
		t.Fatalf("pedido pendente esperado: %s", lista.Body.String())
	}

	// 1. Definir quem aprova SEM ator é recusado: mudar quem autoriza é decisão de risco.
	//    (A rota é PUT; o helper `serviceAgentRequest` é só de POST — usar o certo evita um 404 que parece
	//    defeito de rota e é defeito do teste.)
	semAtor := approversRequest(engine, `{"approvers":["ana"]}`)
	if semAtor.Code != http.StatusBadRequest {
		t.Fatalf("configurar aprovadores sem ator precisa ser 400, veio %d", semAtor.Code)
	}

	// 2. Configuração legítima: a RESPOSTA declara que a decisão passará por checagem de papel.
	config := approversRequest(engine, `{"actor_id":"ana","approvers":["ana"]}`)
	if config.Code != http.StatusOK {
		t.Fatalf("configuração precisa ser aceita (veio %d): %s", config.Code, config.Body.String())
	}
	if !strings.Contains(config.Body.String(), `"role_checked":true`) {
		t.Fatalf("a resposta precisa declarar que o papel é conferido: %s", config.Body.String())
	}

	// 3. Ator FORA da lista: 403 (PROIBIDO), e o pedido continua pendente.
	forasteiro := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide",
		`{"approved":true,"actor_id":"estagiario"}`)
	if forasteiro.Code != http.StatusForbidden {
		t.Fatalf("ator fora da lista precisa ser 403, veio %d: %s", forasteiro.Code, forasteiro.Body.String())
	}
	aindaPendente := httptest.NewRecorder()
	engine.ServeHTTP(aindaPendente, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=pending", nil))
	if !strings.Contains(aindaPendente.Body.String(), approvalID) {
		t.Fatalf("a recusa não pode decidir o pedido: %s", aindaPendente.Body.String())
	}

	// 4. Ator NA lista: decide, e a resposta traz `role_checked` verdadeiro.
	decidido := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide",
		`{"approved":true,"actor_id":"ana","reason":"cliente pediu por escrito"}`)
	if decidido.Code != http.StatusOK || !strings.Contains(decidido.Body.String(), `"role_checked":true`) {
		t.Fatalf("decisão por ator autorizado precisa ser aceita e declarar o papel (veio %d): %s", decidido.Code, decidido.Body.String())
	}
	if !strings.Contains(decidido.Body.String(), "expires_at") {
		t.Fatalf("a aprovação precisa declarar QUANDO vence: %s", decidido.Body.String())
	}

	// 5. O estado `expired` é um filtro VÁLIDO da listagem (e não um erro).
	expiradas := httptest.NewRecorder()
	engine.ServeHTTP(expiradas, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=expired", nil))
	if expiradas.Code != http.StatusOK {
		t.Fatalf("filtro `expired` precisa ser aceito, veio %d: %s", expiradas.Code, expiradas.Body.String())
	}
}
