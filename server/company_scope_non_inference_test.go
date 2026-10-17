package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// NÃO INFERÊNCIA NA SUPERFÍCIE DE EMPRESA (E-190, fechamento de B-15).
//
// O achado B-15: o guarda compartilhado `companyForRequest` devolvia **404** para empresa inexistente e **403** para
// empresa de OUTRO tenant — a diferença de status revelava, a qualquer tenant autenticado, quais IDs de empresa
// existem. Este teste prova a propriedade de forma GENÉRICA em várias rotas: para cada uma, a recusa de "empresa
// inexistente" e a recusa de "empresa de outro tenant" precisam ser **idênticas** em status e corpo.
//
// A asserção NÃO fixa o número do status: o que importa é a INDISTINGUIBILIDADE. Há rotas cuja recusa unificada é 403
// (as que passam pelo guarda) e uma cuja recusa unificada é 404 (`CompanyFactoryReportFor`, que já unificava). Fixar
// um número específico seria trocar um acoplamento por outro.
func TestCompanyScopedRoutesDoNotRevealExistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	empresas, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	daOrgA, err := empresas.Create(agent.Company{OrganizationID: "org-a", Name: "Empresa da Org A"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: empresas, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}

	// A organização "org-intrusa" é quem tenta; ela NUNCA é dona de nada aqui.
	const intrusa = "org-intrusa"

	rotas := []struct {
		nome    string
		caminho func(id string) string
		handler func(*gin.Context)
	}{
		{
			nome:    "tel-agent history",
			caminho: func(id string) string { return "/companies/" + id + "/tel-agent/history" },
			handler: api.companyTelAgentHistory,
		},
		{
			nome:    "marketplace channels",
			caminho: func(id string) string { return "/companies/" + id + "/marketplace/channels" },
			handler: api.marketplaceChannels,
		},
		{
			nome:    "company factory report",
			caminho: func(id string) string { return "/company/factory/report/" + id },
			handler: api.companyFactoryReport,
		},
	}

	for _, rota := range rotas {
		t.Run(rota.nome, func(t *testing.T) {
			executar := func(companyID string) *httptest.ResponseRecorder {
				caminho := rota.caminho(companyID)
				ctx, gravador := orgContext(t, http.MethodGet, caminho, intrusa)
				ctx.Params = gin.Params{{Key: "id", Value: companyID}}
				ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: intrusa})
				ctx.Set("agent.user", agent.User{ID: "intruso"})
				ctx.Request = httptest.NewRequest(http.MethodGet, caminho, nil)
				rota.handler(ctx)
				return gravador
			}

			deOutroTenant := executar(daOrgA.ID)
			inexistente := executar("cmp_que_nao_existe_em_lugar_nenhum")

			// (1) as duas precisam ser RECUSA (nunca sucesso).
			if deOutroTenant.Code == http.StatusOK {
				t.Fatalf("empresa de outro tenant não pode responder 200 (veio %d: %s)", deOutroTenant.Code, deOutroTenant.Body.String())
			}
			// (2) e precisam ser INDISTINGUÍVEIS — é isto que impede a inferência.
			if deOutroTenant.Code != inexistente.Code {
				t.Fatalf("ORÁCULO DE ENUMERAÇÃO em %s: outro tenant -> %d, inexistente -> %d",
					rota.nome, deOutroTenant.Code, inexistente.Code)
			}
			// A comparação NORMALIZA o ID que o próprio chamador informou: uma mensagem que ecoa o ID pedido não é
			// inferência (ele já sabe o que perguntou). Inferência seria revelar algo que ele NÃO informou.
			corpoOutro := normalizarID(deOutroTenant.Body.String(), daOrgA.ID)
			corpoInexistente := normalizarID(inexistente.Body.String(), "cmp_que_nao_existe_em_lugar_nenhum")
			t.Logf("BRUTO outro=%s | inexistente=%s", deOutroTenant.Body.String(), inexistente.Body.String())
			if corpoOutro != corpoInexistente {
				t.Fatalf("ORÁCULO DE ENUMERAÇÃO em %s: corpos diferentes:\n  outro tenant: %s\n  inexistente:  %s",
					rota.nome, corpoOutro, corpoInexistente)
			}
			// (3) a recusa precisa ter corpo: negativa vazia também é observável (e não informa o operador).
			if deOutroTenant.Body.Len() == 0 {
				t.Fatalf("recusa sem corpo em %s", rota.nome)
			}
			t.Logf("%s: recusa unificada status=%d corpo=%q", rota.nome, deOutroTenant.Code, deOutroTenant.Body.String())
		})
	}
}

// CONTROLE POSITIVO: a propriedade acima só vale se o dono CONSEGUE acessar a própria empresa — sem isto, um
// "sempre nega" passaria no teste de indistinguibilidade e a prova seria vazia.
func TestCompanyScopedRoutesAllowTheOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	empresas, err := agent.NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	empresa, err := empresas.Create(agent.Company{OrganizationID: "org-a", Name: "Empresa da Org A"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Company: empresas, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, authRequired: true}
	caminho := "/companies/" + empresa.ID + "/tel-agent/history"
	ctx, gravador := orgContext(t, http.MethodGet, caminho, "org-a")
	ctx.Params = gin.Params{{Key: "id", Value: empresa.ID}}
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleViewer, OrganizationID: "org-a"})
	ctx.Set("agent.user", agent.User{ID: "viewer-a"})
	ctx.Request = httptest.NewRequest(http.MethodGet, caminho, nil)
	api.companyTelAgentHistory(ctx)
	if gravador.Code != http.StatusOK {
		t.Fatalf("o dono precisa acessar a própria empresa: status=%d body=%s", gravador.Code, gravador.Body.String())
	}
}

// normalizarID troca o ID informado pelo chamador por um marcador: a comparação precisa medir a FORMA da recusa, não
// o eco do que o chamador já sabe.
func normalizarID(corpo, id string) string {
	return strings.ReplaceAll(corpo, id, "<id>")
}
