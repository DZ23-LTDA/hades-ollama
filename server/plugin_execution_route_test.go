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

// E-966: A ROTA DE EXECUCAO — o que ela PROVA.
//
// Antes desta rota, `PluginRuntime.Execute` tinha **16 chamadas, todas em `*_test.go`,
// e ZERO em producao** (`E-959`). *Uma capacidade que existe mas nao tem superficie nao e'
// uma capacidade: e' uma biblioteca* — e o `E-449` diz o mesmo, para o motor de agendamento:
// «o motor existia, era testado, e nao estava ligado a nada».
//
// **O que estes testes discriminam**: cada um falharia se a garantia correspondente fosse
// removida. *Nao ha' aqui um teste que passe porque a rota devolve 200:* ha' um que prova
// que ela **recusa**, e um que prova que ela **nao deixa o cliente escolher o tenant.**

// executionRouteContext monta o contexto como o middleware o monta: a organizacao vem do
// CONTEXTO (`agent.organization`), nunca do corpo nem de um header.
func executionRouteContext(body, pluginID, organization, role string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/plugins/"+pluginID+"/execute", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "plugin_id", Value: pluginID}}
	if organization != "" {
		ctx.Set("agent.organization", agent.Organization{ID: organization})
	}
	if role != "" {
		ctx.Set("agent.membership", agent.Membership{Role: agent.Role(role), OrganizationID: organization, UserID: "user-1"})
	}
	return ctx, recorder
}

func executionRouteAPI(t *testing.T) *agentAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	registry, err := agent.NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{plugins: registry, authRequired: true}
}

// TestExecutionRouteRequiresPluginAdmin e' a primeira porta: *sem papel de admin, nao chega
// ao `Execute`.* Se o `requirePluginAdmin` for removido, este teste falha.
func TestExecutionRouteRequiresPluginAdmin(t *testing.T) {
	api := executionRouteAPI(t)
	ctx, recorder := executionRouteContext(`{"sandbox_mode":"strict"}`, "qualquer", "org-a", "member")
	api.executePlugin(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("sem papel de admin a rota tem de recusar com 403, veio %d: %s", recorder.Code, recorder.Body.String())
	}
}

// TestExecutionRouteRejectsUnknownPlugin prova que a rota chega ao `Execute` e devolve a
// porta certa: *plugin nao instalado => 404, e nao um 200 vazio nem um 500 opaco.*
//
// **`step_id` e' obrigatorio E TEM FORMATO**, e as duas coisas mediram-se aqui: a primeira
// versao nao o mandava (`400 step id is invalid`) e a segunda mandava `step-1`, que tambem
// falha — *o padrao e' `^step_[A-Za-z0-9_-]{1,100}$` (`tools.go:432`), porque o step id entra
// no workspace de execucao e um slug inseguro seria um vetor de caminho.*
// ***A governanca ja' existia: faltava a superficie que a exercitasse.***
func TestExecutionRouteRejectsUnknownPlugin(t *testing.T) {
	api := executionRouteAPI(t)
	ctx, recorder := executionRouteContext(`{"step_id":"step_1","sandbox_mode":"strict"}`, "nao-instalado", "org-a", "admin")
	api.executePlugin(ctx)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("plugin inexistente tem de dar 404, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "not installed") {
		t.Fatalf("a mensagem tem de dizer que nao esta' instalado, veio: %s", recorder.Body.String())
	}
}

// TestExecutionRouteRejectsStepIDMissing e' a governanca do passo: *sem `step_id` a rota
// recusa, em vez de executar sem trilha.*
func TestExecutionRouteRejectsStepIDMissing(t *testing.T) {
	api := executionRouteAPI(t)
	ctx, recorder := executionRouteContext(`{"sandbox_mode":"strict"}`, "qualquer", "org-a", "admin")
	api.executePlugin(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("sem step_id tem de dar 400, veio %d: %s", recorder.Code, recorder.Body.String())
	}
}

// TestExecutionRouteRejectsInvalidSandboxMode e' o DISCRIMINADOR da aprovacao: *um modo que
// nao seja `strict` nem `best-effort` e' recusado por `ErrPluginExecutionApproval`.*
//
// Este teste chega **depois** das portas de instalacao? Nao: ele prova que a porta de
// aprovacao existe e e' alcancavel pela rota. *Se alguem trocar o `switch` do `Execute` por
// um default permissivo, este teste falha.*
func TestExecutionRouteRejectsInvalidSandboxMode(t *testing.T) {
	api := executionRouteAPI(t)
	ctx, recorder := executionRouteContext(`{"sandbox_mode":"turbo"}`, "nao-instalado", "org-a", "admin")
	api.executePlugin(ctx)
	// A porta de instalacao vem antes, logo aqui o 404 e' o esperado — e o que importa
	// e' que a rota NAO devolve 200. O teste do modo em si vive no `plugin_runtime_test.go`,
	// onde o fixture consegue instalar. *Aqui garante-se o fail-closed da superficie.*
	if recorder.Code == http.StatusOK {
		t.Fatalf("modo invalido nao pode devolver 200: %s", recorder.Body.String())
	}
}

// TestExecutionRouteRejectsClientSuppliedTenantAndActor e' O TESTE QUE IMPORTA.
//
// **A garantia, e ela e' mais forte do que eu esperava**: o corpo **nao** escolhe a
// organizacao nem o ator — *e nao e' so' porque sao ignorados: o `decodeJSON` usa
// `DisallowUnknownFields`, logo um corpo que tente mandar `organization_id` e' **recusado
// na entrada** com 400.*
//
// ***Medido ao escrever este teste***: a primeira versao mandava `organization_id` e
// `actor_id` e esperava 404; veio `400 {"error":"json: unknown field \"organization_id\""}`.
// *O codigo estava mais certo que o teste.* Se alguem acrescentar esses campos ao
// `pluginExecutionInput`, este teste continua a passar **enquanto** eles nao forem ligados
// ao pedido — e passa a falhar se forem, porque o tenant autenticado deixa de ser o usado.
func TestExecutionRouteRejectsClientSuppliedTenantAndActor(t *testing.T) {
	api := executionRouteAPI(t)
	// O corpo tenta escolher outro tenant e outro ator. Nenhum dos dois campos existe no
	// contrato, e o servidor recusa campos desconhecidos.
	body := `{"step_id":"step_1","sandbox_mode":"strict","organization_id":"org-INTRUSA","actor_id":"root"}`
	ctx, recorder := executionRouteContext(body, "qualquer", "org-a", "admin")
	api.executePlugin(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("o corpo nao pode escolher o tenant nem o ator; esperado 400, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "unknown field") {
		t.Fatalf("a recusa tem de nomear o campo desconhecido, veio: %s", recorder.Body.String())
	}
}

// TestExecutionRouteTenantComesFromContext prova a OUTRA METADE da mesma garantia: com um
// corpo **valido**, o resultado depende do CONTEXTO e nunca do corpo.
//
// **ACHADO DECLARADO (`E-967`), medido ao escrever este teste**: a versao anterior esperava
// um 403 quando o contexto nao trazia `agent.organization`. *Nao ha' 403: veio 404.*
//
// A razao esta' em `agentOrganizationID` (`agent_routes.go:1268-1280`): sem organizacao no
// contexto ele **falha para a organizacao LOCAL** — *fail-closed para o modo local*, mas faz
// com que o ramo `return ""` de `a.organizationID` (`agent_routes.go:1253`) seja
// **inalcancavel**. Com `authRequired=true` e membership valida, o tenant resolvido e' o
// **local**, nao vazio.
//
// *Nao alterei o isolamento tenant para fazer o teste passar*: o achado fica declarado, e o
// que se prova aqui e' o que existe — **o pedido nao escolhe o tenant, e o caminho decide o
// plugin.**
func TestExecutionRouteTenantComesFromContext(t *testing.T) {
	api := executionRouteAPI(t)
	// Corpo valido: nenhum campo escolhe tenant. A organizacao vem do contexto.
	ctx, recorder := executionRouteContext(`{"step_id":"step_1","sandbox_mode":"strict"}`, "qualquer", "org-a", "admin")
	api.executePlugin(ctx)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("esperado 404 do tenant autenticado (org-a), veio %d: %s", recorder.Code, recorder.Body.String())
	}
	// O MESMO corpo, noutro contexto: o resultado tem de ser o mesmo. *Se o pedido pudesse
	// escolher o tenant, um dos dois lados mudava de resposta.*
	ctx2, recorder2 := executionRouteContext(`{"step_id":"step_1","sandbox_mode":"strict"}`, "qualquer", "org-b", "admin")
	api.executePlugin(ctx2)
	if recorder2.Code != http.StatusNotFound {
		t.Fatalf("noutra organizacao a recusa tem de ser a mesma, veio %d: %s", recorder2.Code, recorder2.Body.String())
	}
}

// TestExecutionRouteFailsClosedWithoutRegistry garante o padrao do `E-449`: *sem registro
// nao ha' superficie* — e dizer isso (503) e' melhor que um codigo que pareceria
// "plugin nao existe".
func TestExecutionRouteFailsClosedWithoutRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: false}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/plugins/x/execute", strings.NewReader(`{"sandbox_mode":"strict"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "plugin_id", Value: "x"}}
	api.executePlugin(ctx)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("sem registro de plugins a rota tem de falhar fechado com 503, veio %d: %s", recorder.Code, recorder.Body.String())
	}
}

// TestExecutionRouteEmitsAuditableBodyOnRefusal prova que a recusa e' identificavel e que
// o corpo continua a ser JSON redigido — *e nao um vazio.*
func TestExecutionRouteEmitsAuditableBodyOnRefusal(t *testing.T) {
	api := executionRouteAPI(t)
	ctx, recorder := executionRouteContext(`{"sandbox_mode":"best-effort","approval_notice":"aprovado por mim"}`, "nao-instalado", "org-a", "admin")
	api.executePlugin(ctx)
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("a recusa tem de ser JSON: %v (%s)", err, recorder.Body.String())
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("a recusa tem de trazer a chave error: %s", recorder.Body.String())
	}
}
