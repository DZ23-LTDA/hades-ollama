package server

// FASE 28.6-A / 28.7: testes das rotas do instalador modular.
//
// O que estes testes travam:
//
//  1. o catalogo e os perfis chegam ao consumidor completos e com a auditoria feita;
//  2. o plano de `CUSTOM` nao inclui nada por omissao;
//  3. perfil desconhecido devolve 400 -- nao 200 com plano vazio;
//  4. nenhuma das rotas MUTA o sistema (POST recusado);
//  5. o plano nunca diz "executa algo";
//  6. `bloqueios` nunca e' `null`;
//  7. o perfil com `containers` declara aprovacoes necessarias mesmo quando a maquina tem Docker.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// routerDoInstalador monta o mesmo engine que os testes de toolchain usam: o `agentAPI` real,
// com as rotas reais. *Testar o handler sem o router deixaria o registo da rota por provar.*
func routerDoInstalador(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	return engine
}

// getRaw devolve o corpo CRU: e' a unica forma de provar que `bloqueios` nao e' `null`.
func getRaw(t *testing.T, engine *gin.Engine, caminho string) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, caminho, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s -> %d", caminho, recorder.Code)
	}
	return recorder.Body.String()
}

// postJSON tenta um POST e devolve o codigo, para provar que a rota e' so' leitura.
func postJSON(t *testing.T, engine *gin.Engine, caminho string) int {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, caminho, strings.NewReader("{}")))
	return recorder.Code
}

var _ = json.Marshal

// O catalogo chega com os 9 perfis e os 18 pacotes.
func TestInstallProfilesDevolvePerfisECatalogo(t *testing.T) {
	engine := routerDoInstalador(t)
	var payload struct {
		Perfis  []installPerfilResposta `json:"perfis"`
		Pacotes []installPacoteResposta `json:"pacotes"`
		Avisos  []string                `json:"avisos"`
		Count   int                     `json:"count"`
		Host    string                  `json:"host"`
	}
	if code := getJSON(t, engine, "/api/agent/v1/install/profiles", &payload); code != http.StatusOK {
		t.Fatalf("GET /install/profiles -> %d", code)
	}
	if len(payload.Perfis) != 9 {
		t.Fatalf("%d perfis; esperados 9", len(payload.Perfis))
	}
	if payload.Count != len(payload.Perfis) {
		t.Fatalf("count = %d, perfis = %d", payload.Count, len(payload.Perfis))
	}
	if len(payload.Pacotes) != 18 {
		t.Fatalf("%d pacotes; esperados 18", len(payload.Pacotes))
	}
	if payload.Host == "" {
		t.Fatal("host vazio: o consumidor nao sabe em que maquina o catalogo foi lido")
	}
	// A auditoria do catalogo chegou ao consumidor.
	if payload.Avisos == nil {
		t.Fatal("avisos = null (tem de ser lista, mesmo vazia)")
	}
	if len(payload.Avisos) != 0 {
		for _, a := range payload.Avisos {
			t.Logf("  aviso: %s", a)
		}
		t.Fatalf("%d aviso(s) no catalogo entregue", len(payload.Avisos))
	}
	// Cada perfil tem descricao legivel (nao uma string vazia).
	for _, p := range payload.Perfis {
		if strings.TrimSpace(p.Descricao) == "" {
			t.Fatalf("perfil %q sem descricao", p.ID)
		}
	}
	// CUSTOM e' o unico que escolhe pacotes a mao.
	for _, p := range payload.Perfis {
		querEscolher := p.ID == "CUSTOM"
		if p.EscolhePacotes != querEscolher {
			t.Fatalf("perfil %q EscolhePacotes=%v", p.ID, p.EscolhePacotes)
		}
		if querEscolher && len(p.Pacotes) != 0 {
			t.Fatalf("CUSTOM veio com %d pacotes por omissao", len(p.Pacotes))
		}
	}
}

// Um pacote com impacto tem de o declarar -- senao a UI mostra-o como inofensivo.
func TestInstallPacotesDeclaramImpacto(t *testing.T) {
	engine := routerDoInstalador(t)
	var payload struct {
		Pacotes []installPacoteResposta `json:"pacotes"`
	}
	if code := getJSON(t, engine, "/api/agent/v1/install/profiles", &payload); code != http.StatusOK {
		t.Fatalf("GET /install/profiles -> %d", code)
	}
	for _, p := range payload.Pacotes {
		if p.ExigeAprovacao && len(p.Impacto) == 0 {
			t.Fatalf("pacote %q exige aprovacao sem declarar impacto", p.ID)
		}
		if len(p.Impacto) > 0 && !p.ExigeAprovacao {
			t.Fatalf("pacote %q declara impacto mas nao exige aprovacao", p.ID)
		}
		if strings.TrimSpace(p.Licenca) == "" {
			t.Fatalf("pacote %q sem licenca", p.ID)
		}
	}
	// `containers` e' o caso que o adendo nomeia: servico + elevacao + reinicio + porta.
	var achou bool
	for _, p := range payload.Pacotes {
		if p.ID != "containers" {
			continue
		}
		achou = true
		if !p.ExigeAprovacao {
			t.Fatal("containers nao exige aprovacao")
		}
		juntas := strings.Join(p.Impacto, " | ")
		for _, palavra := range []string{"elevacao", "servico", "reinicio", "porta"} {
			if !strings.Contains(juntas, palavra) {
				t.Fatalf("impacto de containers sem %q: %s", palavra, juntas)
			}
		}
	}
	if !achou {
		t.Fatal("containers ausente do catalogo")
	}
}

// O plano de CUSTOM nao inclui nada por omissao.
func TestInstallPlanCustomVazio(t *testing.T) {
	engine := routerDoInstalador(t)
	var payload installPlanoResposta
	if code := getJSON(t, engine, "/api/agent/v1/install/plan/CUSTOM", &payload); code != http.StatusOK {
		t.Fatalf("GET /install/plan/CUSTOM -> %d", code)
	}
	if len(payload.Itens) != 0 {
		t.Fatalf("CUSTOM trouxe %d itens", len(payload.Itens))
	}
	if payload.TamanhoTotalMB != 0 {
		t.Fatalf("CUSTOM com tamanho %d", payload.TamanhoTotalMB)
	}
	if payload.ExecutaAlgo {
		t.Fatal("plano declara que executa algo")
	}
}

// Perfil desconhecido -> 400, nao 200 com plano vazio.
func TestInstallPlanPerfilDesconhecido400(t *testing.T) {
	engine := routerDoInstalador(t)
	for _, invalido := range []string{"TUDO", "basic-plus", "BASIC2", "xyz"} {
		code := getJSON(t, engine, "/api/agent/v1/install/plan/"+invalido, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("perfil %q -> %d; esperado 400", invalido, code)
		}
	}
}

// O perfil e' aceite em minusculas (o path e' humano).
func TestInstallPlanAceitaMinusculas(t *testing.T) {
	engine := routerDoInstalador(t)
	var payload installPlanoResposta
	if code := getJSON(t, engine, "/api/agent/v1/install/plan/basic", &payload); code != http.StatusOK {
		t.Fatalf("GET /install/plan/basic -> %d", code)
	}
	if payload.Perfil != "BASIC" {
		t.Fatalf("perfil normalizado = %q; esperado BASIC", payload.Perfil)
	}
}

// O plano de BASIC inclui o core e nada mais (nesta maquina, o core nao tem comando verificavel).
func TestInstallPlanBasicSoCore(t *testing.T) {
	engine := routerDoInstalador(t)
	var payload installPlanoResposta
	if code := getJSON(t, engine, "/api/agent/v1/install/plan/BASIC", &payload); code != http.StatusOK {
		t.Fatalf("GET /install/plan/BASIC -> %d", code)
	}
	if len(payload.Itens) != 1 {
		t.Fatalf("%d itens em BASIC; esperado 1 (so' o core)", len(payload.Itens))
	}
	if payload.Itens[0].PacoteID != "hades-core" {
		t.Fatalf("item = %q; esperado hades-core", payload.Itens[0].PacoteID)
	}
	// `hades-core` nao tem ferramenta representativa -> UNKNOWN -> PULAR, NUNCA instalar.
	if payload.Itens[0].Acao != "PULAR" {
		t.Fatalf("hades-core acao = %q; esperado PULAR (*UNKNOWN nao e' ABSENT*)",
			payload.Itens[0].Acao)
	}
	if len(payload.Bloqueios) == 0 {
		t.Fatal("sem bloqueios apesar de o core nao ter sido avaliado")
	}
	if payload.Pronto {
		t.Fatal("plano declarado pronto com bloqueio")
	}
}

// O plano declara no payload que nao executa nada.
func TestInstallPlanNaoExecutaNada(t *testing.T) {
	engine := routerDoInstalador(t)
	for _, perfil := range []string{"BASIC", "RECOMMENDED", "CONTAINERS_AND_CLOUD"} {
		var payload installPlanoResposta
		if code := getJSON(t, engine, "/api/agent/v1/install/plan/"+perfil, &payload); code != http.StatusOK {
			t.Fatalf("GET /install/plan/%s -> %d", perfil, code)
		}
		if payload.ExecutaAlgo {
			t.Fatalf("perfil %s declara executa_algo=true", perfil)
		}
		if strings.TrimSpace(payload.Resumo) == "" {
			t.Fatalf("perfil %s sem resumo", perfil)
		}
	}
}

// `bloqueios` nunca e' null no JSON.
func TestInstallPlanBloqueiosNuncaNull(t *testing.T) {
	engine := routerDoInstalador(t)
	// CUSTOM nao tem itens: bloqueios tem de ser lista vazia, nao null.
	corpo := getRaw(t, engine, "/api/agent/v1/install/plan/CUSTOM")
	if strings.Contains(corpo, `"bloqueios":null`) {
		t.Fatal("*um null faz o consumidor tratar 'sem bloqueios' como 'campo ausente'*")
	}
	if !strings.Contains(corpo, `"bloqueios":[]`) {
		t.Fatalf("bloqueios nao e' lista vazia: %s", corpo)
	}
}

// As rotas recusam POST: planejar nao e' instalar.
func TestInstallRotasSaoSoLeitura(t *testing.T) {
	engine := routerDoInstalador(t)
	for _, caminho := range []string{
		"/api/agent/v1/install/profiles",
		"/api/agent/v1/install/plan/BASIC",
		"/api/agent/v1/install/plan/CUSTOM",
	} {
		code := postJSON(t, engine, caminho)
		if code == http.StatusOK {
			t.Fatalf("POST %s devolveu 200: *uma rota que planear e executar na mesma chamada "+
				"poupa um clique e gasta uma decisao*", caminho)
		}
	}
}

// O servidor nao devolve material secreto no plano.
func TestInstallPlanNaoExpoeSegredos(t *testing.T) {
	engine := routerDoInstalador(t)
	for _, perfil := range []string{"BASIC", "RECOMMENDED", "DEVELOPER_COMPLETE"} {
		corpo := strings.ToLower(getRaw(t, engine, "/api/agent/v1/install/plan/"+perfil))
		for _, proibido := range []string{
			"private_key", "-----begin", "storepassword", "keypassword", "password",
		} {
			if strings.Contains(corpo, proibido) {
				t.Fatalf("perfil %s contem %q", perfil, proibido)
			}
		}
	}
}

// Os planos sao deterministicos entre duas chamadas seguidas.
func TestInstallPlanDeterministico(t *testing.T) {
	engine := routerDoInstalador(t)
	var a, b installPlanoResposta
	if code := getJSON(t, engine, "/api/agent/v1/install/plan/RECOMMENDED", &a); code != http.StatusOK {
		t.Fatalf("1a chamada -> %d", code)
	}
	if code := getJSON(t, engine, "/api/agent/v1/install/plan/RECOMMENDED", &b); code != http.StatusOK {
		t.Fatalf("2a chamada -> %d", code)
	}
	if len(a.Itens) != len(b.Itens) {
		t.Fatalf("itens: %d vs %d", len(a.Itens), len(b.Itens))
	}
	for i := range a.Itens {
		if a.Itens[i] != b.Itens[i] {
			t.Fatalf("item %d difere entre chamadas", i)
		}
	}
}
