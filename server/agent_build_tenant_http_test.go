package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// routerDeBuildsPara monta as rotas de build com uma sessao de uma organizacao concreta, partilhando
// o MESMO api (logo o mesmo store) -- e' isso que permite perguntar a duas organizacoes pelo mesmo
// build.
func routerDeBuildsPara(api *agentAPI, organizacao string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessaoDeBuild(organizacao))
	g := r.Group("/api/agent/v1")
	g.GET("/build/status/:id", api.estadoDeBuild)
	g.GET("/build/registos", api.registosDeBuild)
	return r
}

func pedirBuild(t *testing.T, r *gin.Engine, caminho string) (int, string) {
	t.Helper()
	pedido := httptest.NewRequest(http.MethodGet, caminho, nil)
	resposta := httptest.NewRecorder()
	r.ServeHTTP(resposta, pedido)
	return resposta.Code, resposta.Body.String()
}

// TestBuildDeOutroTenantNaoVazaNemTemOracle prova, ao nivel HTTP, a fronteira que o `E-1470` fechou
// no store: outra organizacao nao le o build E nao consegue distinguir "existe mas nao e' teu" de
// "nao existe".
//
//	*Uma negacao que muda de forma conforme o motivo e' uma fuga: o atacante aprende a existencia do
//	objecto sem nunca o ler.*
func TestBuildDeOutroTenantNaoVazaNemTemOracle(t *testing.T) {
	api := &agentAPI{}
	id, err := api.storeDeBuilds().RegistarDoTenant("windows", "raiz-secreta", "destino-secreto", "org-a")
	if err != nil {
		t.Fatalf("registar build: %v", err)
	}

	daOrganizacaoA := routerDeBuildsPara(api, "org-a")
	deOutraOrganizacao := routerDeBuildsPara(api, "org-b")

	// POSITIVO: o dono le o proprio build.
	codigoDono, corpoDono := pedirBuild(t, daOrganizacaoA, "/api/agent/v1/build/status/"+id)
	if codigoDono != http.StatusOK {
		t.Fatalf("o dono devia ler o build, veio %d: %s", codigoDono, corpoDono)
	}
	if !strings.Contains(corpoDono, "raiz-secreta") {
		t.Fatalf("o dono devia ver os campos do registo; corpo: %s", corpoDono)
	}

	// NEGATIVO: outra organizacao recebe 404 e NENHUM campo do registo.
	codigoAlheio, corpoAlheio := pedirBuild(t, deOutraOrganizacao, "/api/agent/v1/build/status/"+id)
	if codigoAlheio != http.StatusNotFound {
		t.Fatalf("outra organizacao devia receber 404, veio %d: %s", codigoAlheio, corpoAlheio)
	}
	for _, proibido := range []string{"raiz-secreta", "destino-secreto", "org-a", "windows"} {
		if strings.Contains(corpoAlheio, proibido) {
			t.Fatalf("a negacao vazou %q: %s", proibido, corpoAlheio)
		}
	}

	// SEM ORACULO: a resposta e' a MESMA de um id que nunca existiu (a unica diferenca admitida e' o
	// eco do id que o proprio chamador enviou).
	codigoInexistente, corpoInexistente := pedirBuild(t, deOutraOrganizacao, "/api/agent/v1/build/status/nao-existe-xyz")
	if codigoInexistente != http.StatusNotFound {
		t.Fatalf("id inexistente devia dar 404, veio %d", codigoInexistente)
	}
	normalizado := strings.ReplaceAll(corpoAlheio, id, "<id>")
	normalizado = strings.ReplaceAll(normalizado, "nao-existe-xyz", "<id>")
	esperado := strings.ReplaceAll(corpoInexistente, "nao-existe-xyz", "<id>")
	if normalizado != esperado {
		t.Fatalf("a negacao de build alheio tem de ser IDENTICA a de id inexistente\n  alheio:      %s\n  inexistente: %s",
			normalizado, esperado)
	}

	// E a LISTA nao mostra o build de outra organizacao.
	codigoListaAlheia, corpoListaAlheia := pedirBuild(t, deOutraOrganizacao, "/api/agent/v1/build/registos")
	if codigoListaAlheia != http.StatusOK {
		t.Fatalf("a lista devia dar 200, veio %d", codigoListaAlheia)
	}
	if strings.Contains(corpoListaAlheia, id) || strings.Contains(corpoListaAlheia, "raiz-secreta") {
		t.Fatalf("a lista de org-b mostrou build de org-a: %s", corpoListaAlheia)
	}
	if !strings.Contains(corpoListaAlheia, `"total":0`) {
		t.Fatalf("a lista de org-b devia estar vazia: %s", corpoListaAlheia)
	}
	_, corpoListaDono := pedirBuild(t, daOrganizacaoA, "/api/agent/v1/build/registos")
	if !strings.Contains(corpoListaDono, id) {
		t.Fatalf("a lista do dono devia incluir o build: %s", corpoListaDono)
	}
}
