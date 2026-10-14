package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// E-429 / ADV-68 (CRÍTICO) — AS ROTAS DE APROVAÇÃO DOS SERVICE AGENTS EXIGEM PAPEL, E O ATOR É O DA SESSÃO.
//
// **MEDIDO por auditoria adversarial independente** (cabeçalho completo em `server/service_agents_routes.go`):
// com uma sessão `RoleViewer` autenticada, a sonda obteve, no código anterior a esta correção:
//
//	VIEWER decide               -> HTTP 200
//	TURNO SEGUINTE              -> status=executed · "oportunidade op_… criada (new, 79900 centavos)"
//	                               oportunidades: 1 -> 2        ← A ESCRITA EXECUTOU DE VERDADE NO CRM
//	VIEWER redefine aprovadores -> HTTP 200  {"approvers":["eu-mesmo"],"changed_by":"eu-mesmo"}
//	sessão 'u-da-sessao' + actor_id 'presidente-da-empresa' -> 200, decided_by="presidente-da-empresa"
//
// ⇒ ***Um viewer aprovava a própria escrita, redefinia quem pode aprovar, e a trilha gravava o nome que o
// CLIENTE escolhesse.*** E a **mesma** sessão, na rota irmã `decideDeploymentApproval`, recebia **403** ✓ — *o
// portão existia e funcionava; **não havia sido aplicado aqui*** ✗.
//
// `TestServiceAgentApprovalRoutesEnforceRoleAndListExpired` **prometia** medir o papel (`EnforceRole`) e media
// apenas a **lista de aprovadores configurada** ✗ — *a classe "o nome afirma o que o corpo não mede".*
//
// ***Este arquivo mede o que aquele nome prometia.*** O portão tem de disparar **antes** de qualquer busca, então
// o teste **não precisa** de um pedido real: *se a recusa dependesse de o pedido existir, ela não seria um portão.*
func TestRotaDeAprovacaoDeServiceAgentExigePapelOwnerOuAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("api: %v", err)
	}
	// 🔴 **SEM ISTO O TESTE NÃO MEDE NADA.** Com `authRequired == false`, `requireApprovalApprover` devolve
	// `true` na primeira linha e o portão de papel é **no-op** ⇒ *o viewer passaria e o teste "provaria" o
	// contrário do que promete.* É o padrão da casa: `&agentAPI{…, authRequired: true}`.
	api.authRequired = true

	// O MESMO engine, com o papel como parâmetro — *é o papel que varia, nada mais.*
	engineComPapel := func(papel agent.Role) *gin.Engine {
		engine := gin.New()
		group := engine.Group("/api/agent/v1")
		group.Use(func(c *gin.Context) {
			c.Set("agent.membership", agent.Membership{Role: papel, OrganizationID: agent.LocalOrganizationID})
			// O ATOR da sessão: `agentActorID` lê **`agent.user`** (um `agent.User`) ou o header
			// `X-Ollama-User`, caindo em `"local"`. *Eu havia palpitado `agent.actor_id` — a chave errada —
			// e foi este teste que me desmentiu.*
			c.Set("agent.user", agent.User{ID: "ator-da-sessao"})
			c.Next()
		})
		group.POST("/service-agents/approvals/:id/decide", api.serviceAgentDecideApproval)
		group.PUT("/service-agents/approval-approvers", api.serviceAgentApprovalApprovers)
		return engine
	}
	pedir := func(engine *gin.Engine, metodo, alvo, corpo string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(metodo, alvo, strings.NewReader(corpo))
		request.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(recorder, request)
		return recorder
	}
	// Um id que **não existe**: o portão tem de recusar ANTES de procurá-lo. *É isto que separa "portão" de
	// "busca que falhou por outro motivo".*
	const pedidoInexistente = "apr_nao_existe_portao"
	const corpo = `{"approved":true,"actor_id":"presidente-da-empresa","reason":"eu mesmo"}`
	_ = time.Now

	// (a) VIEWER é RECUSADO nas DUAS rotas — *decidir e redefinir quem aprova: as duas são atos de risco.*
	viewer := engineComPapel(agent.RoleViewer)
	casos := []struct {
		nome   string
		metodo string
		alvo   string
		corpo  string
		oque   string
	}{
		{"decidir", http.MethodPost, "/api/agent/v1/service-agents/approvals/" + pedidoInexistente + "/decide", corpo,
			"aprovar a própria escrita"},
		{"redefinir aprovadores", http.MethodPut, "/api/agent/v1/service-agents/approval-approvers",
			`{"actor_id":"presidente-da-empresa","approvers":["presidente-da-empresa"]}`, "definir quem pode aprovar"},
	}
	for _, caso := range casos {
		t.Run("viewer nao pode "+caso.nome, func(t *testing.T) {
			resposta := pedir(viewer, caso.metodo, caso.alvo, caso.corpo)
			if resposta.Code != http.StatusForbidden {
				t.Fatalf("ARMADILHA: um VIEWER conseguiu %s (HTTP %d) — *é exatamente a vulnerabilidade medida pela "+
					"auditoria: aprovação de risco como auto-serviço*.\nCorpo: %s\n"+
					"Se este teste falhou, o portão `requireApprovalApprover` saiu do caminho.",
					caso.oque, resposta.Code, resposta.Body.String())
			}
			// E a recusa NÃO pode ser "ator fora da lista": é recusa de PAPEL, e isso aparece no corpo.
			if !strings.Contains(resposta.Body.String(), "owner or admin") {
				t.Fatalf("a recusa precisa ser por PAPEL (owner/admin), e não por outro motivo: %s", resposta.Body.String())
			}
		})
	}

	// (b) CONTROLE: com papel ADMIN, o portão deixa passar e o erro passa a ser o do PEDIDO inexistente — *sem
	// este controle, um portão que recusasse todo mundo passaria no teste acima.*
	admin := engineComPapel(agent.RoleAdmin)
	passou := pedir(admin, http.MethodPost, "/api/agent/v1/service-agents/approvals/"+pedidoInexistente+"/decide", corpo)
	if passou.Code == http.StatusForbidden {
		t.Fatalf("ARMADILHA: um ADMIN foi recusado pelo portão de papel (HTTP 403) — *o controle do teste não "+
			"distingue 'recusa de papel' de 'recusa de tudo'*: %s", passou.Body.String())
	}
	if !strings.Contains(passou.Body.String(), "não existe") && !strings.Contains(passou.Body.String(), "not found") {
		t.Fatalf("com papel válido, a resposta tem de ser a do PEDIDO (inexistente), e não outra coisa: %s", passou.Body.String())
	}
}
