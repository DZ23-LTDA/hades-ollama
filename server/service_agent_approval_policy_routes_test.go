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
	// E-429/ADV-68: a lista é de **pessoas da SESSÃO**, não de nomes que o cliente escreve. Antes, a lista era
	// `["ana"]` e o pedido mandava `actor_id:"ana"` no corpo — *as duas coisas coincidiam porque o corpo ERA a
	// identidade.* Agora quem decide é `agentActorID(c)`; **a lista tem de conter o ator da sessão**.
	config := approversRequest(engine, `{"actor_id":"ana","approvers":["ator-da-sessao"]}`)
	if config.Code != http.StatusOK {
		t.Fatalf("configuração precisa ser aceita (veio %d): %s", config.Code, config.Body.String())
	}
	if !strings.Contains(config.Body.String(), `"role_checked":true`) {
		t.Fatalf("a resposta precisa declarar que o papel é conferido: %s", config.Body.String())
	}

	// 3. 🚨 E-429/ADV-68: **este passo NÃO consegue mais simular "ator fora da lista" pelo corpo.** Antes ele
	// mandava `actor_id:"estagiario"` e o corpo era a identidade ✗; agora o ator é o da **sessão**, então o
	// `actor_id` do corpo é **ignorado** — e o pedido é decidido pelo ator da sessão, que **está** na lista.
	//
	// *Simular "fora da lista" hoje exige uma SESSÃO diferente, não um campo diferente* — e é isso que
	// `service_agent_approval_authorization_test.go` mede (viewer ⇒ 403 pelo **papel**). **A lacuna de não haver
	// aqui uma sessão "fora da lista" fica DECLARADA**, em vez de o teste fingir que o corpo ainda decide.
	//
	// E o que **continua** valendo, e é medido abaixo: uma decisão **negada** não pode decidir o pedido.

	// 4. 🚨 E-429/ADV-68: **este passo codificava a vulnerabilidade.** Ele mandava `actor_id:"ana"` **no corpo**
	// e esperava **200** ⇒ *o corpo era o único ator consultado, e não havia portão de papel* ✗. Medido por
	// auditoria independente: um `RoleViewer` decidia (200), **a escrita executava no CRM**, e a sessão
	// `u-da-sessao` ficava registrada como `presidente-da-empresa` na trilha ✗.
	//
	// ⇒ Com a sessão **owner** que o fixture passou a semear, o caminho autorizado volta a ser exercitado — e o
	// **ator da trilha é o da sessão**, não o do corpo. *A recusa de quem não tem papel é medida em
	// `service_agent_approval_authorization_test.go`.*
	decidido := serviceAgentRequest(engine, "/api/agent/v1/service-agents/approvals/"+approvalID+"/decide",
		`{"approved":true,"actor_id":"ana","reason":"cliente pediu por escrito"}`)
	if decidido.Code != http.StatusOK {
		t.Fatalf("decisão por sessão owner precisa ser aceita (veio %d): %s", decidido.Code, decidido.Body.String())
	}
	if !strings.Contains(decidido.Body.String(), "expires_at") {
		t.Fatalf("a aprovação precisa declarar QUANDO vence: %s", decidido.Body.String())
	}
	// E o ator gravado NÃO pode ser o do corpo.
	if strings.Contains(decidido.Body.String(), `"decided_by":"ana"`) {
		t.Fatalf("ARMADILHA: a trilha gravou `decided_by: ana` — o ator veio do CORPO. *A decisão pertence à "+
			"sessão, não ao que o cliente escreveu.* Corpo: %s", decidido.Body.String())
	}

	// 5. O estado `expired` é um filtro VÁLIDO da listagem (e não um erro).
	expiradas := httptest.NewRecorder()
	engine.ServeHTTP(expiradas, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=expired", nil))
	if expiradas.Code != http.StatusOK {
		t.Fatalf("filtro `expired` precisa ser aceito, veio %d: %s", expiradas.Code, expiradas.Body.String())
	}

	// 6. A DECISÃO humana entrou na trilha durável (§11.1: `/approve` e `/deny` com trilha de auditoria), e a
	//    resposta declara que o registro aconteceu.
	if !strings.Contains(decidido.Body.String(), `"audit_recorded":true`) {
		t.Fatalf("a decisão precisa declarar que foi auditada: %s", decidido.Body.String())
	}
	trilha := httptest.NewRecorder()
	engine.ServeHTTP(trilha, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/audit", nil))
	if !strings.Contains(trilha.Body.String(), "approval.decide") {
		t.Fatalf("o ato humano precisa constar na trilha: %s", trilha.Body.String())
	}
	if !strings.Contains(trilha.Body.String(), "ana") {
		t.Fatalf("quem decidiu precisa constar na trilha: %s", trilha.Body.String())
	}

	// 7. A mudança de QUEM aprova também é ato de política e entra na trilha.
	if !strings.Contains(config.Body.String(), `"audit_recorded":true`) {
		t.Fatalf("a configuração precisa declarar que foi auditada: %s", config.Body.String())
	}
	trilha = httptest.NewRecorder()
	engine.ServeHTTP(trilha, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/audit", nil))
	if !strings.Contains(trilha.Body.String(), "policy.approvers.set") {
		t.Fatalf("a mudança de política precisa constar na trilha: %s", trilha.Body.String())
	}
}

// A FILA de aprovações precisa ser ALCANÇÁVEL: quem opera tem de ver o que falta e há quanto tempo.
func TestServiceAgentApprovalQueueIsVisibleThroughTheRoute(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, true)

	// Um pedido de escrita fica pendente.
	pedido := `{"conversation_id":"` + conversationID + `","intent":"quero o plano",` +
		`"requested_tools":["crm.opportunity.write"],` +
		`"tool_arguments":{"title":"Plano avançado","value_cents":79900,"owner_id":"ana"}}`
	if turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn", pedido); turn.Code != http.StatusOK {
		t.Fatalf("turno precisa responder 200, veio %d: %s", turn.Code, turn.Body.String())
	}

	lista := httptest.NewRecorder()
	engine.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents/approvals?status=pending", nil))
	if lista.Code != http.StatusOK {
		t.Fatalf("listagem precisa responder 200, veio %d: %s", lista.Code, lista.Body.String())
	}
	texto := lista.Body.String()
	// A fila vai junto, com o que falta, a idade e o prazo vigente — sem depender de push que não existe.
	for _, campo := range []string{`"queue"`, `"pending":1`, `"age_seconds"`, `"approvals_needed"`, `"pending_ttl_seconds"`, `"overdue"`} {
		if !strings.Contains(texto, campo) {
			t.Fatalf("a fila precisa trazer %s: %s", campo, texto)
		}
	}
	// E o estado do AVISO: sem destino configurado, quem opera vê que a fila existe mas ninguém é avisado —
	// em vez de supor que o aviso funcionou.
	for _, campo := range []string{`"notification"`, `"configurado":false`, `"entregues":0`} {
		if !strings.Contains(texto, campo) {
			t.Fatalf("o estado do aviso precisa constar (%s): %s", campo, texto)
		}
	}
}
