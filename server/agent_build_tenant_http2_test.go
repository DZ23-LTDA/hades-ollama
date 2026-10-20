package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// routerDeBuildsCompleto monta as SEIS rotas de build com a sessao de uma organizacao.
func routerDeBuildsCompleto(api *agentAPI, organizacao string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(sessaoDeBuild(organizacao))
	g := r.Group("/api/agent/v1")
	g.GET("/build/status/:id", api.estadoDeBuild)
	g.GET("/build/registos", api.registosDeBuild)
	g.GET("/build/logs/:id", api.saidaDoBuild)
	g.GET("/build/artifacts/:id", api.artefactosDoBuild)
	g.POST("/build/cancel/:id", api.cancelarBuild)
	return r
}

func pedirBuildMetodo(t *testing.T, r *gin.Engine, metodo, caminho string) (int, string) {
	t.Helper()
	pedido, err := http.NewRequest(metodo, caminho, nil)
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	resposta := httptest.NewRecorder()
	r.ServeHTTP(resposta, pedido)
	return resposta.Code, resposta.Body.String()
}

// TestTodosOsHandlersDeBuildRespeitamOTenant fecha a lacuna declarada no `E-1477`: o teste anterior
// cobria `status` e `registos`; este cobre `logs`, `artifacts` e `cancel`.
//
//	*Um handler a mais sem prova e' um handler a menos na fronteira -- e o defeito original existia em
//	SEIS handlers, nao num.*
func TestTodosOsHandlersDeBuildRespeitamOTenant(t *testing.T) {
	api := &agentAPI{}
	id, err := api.storeDeBuilds().RegistarDoTenant("windows", "/raiz-do-dono", "/destino-do-dono", "org-a")
	if err != nil {
		t.Fatalf("registar build: %v", err)
	}
	// Escrever saida e resultado, para que um vazamento seja visivel no corpo.
	if err := api.storeDeBuilds().Terminar(id, nil, nil, "saida-secretissima-do-build"); err != nil {
		t.Fatalf("terminar build: %v", err)
	}

	dono := routerDeBuildsCompleto(api, "org-a")
	alheio := routerDeBuildsCompleto(api, "org-b")

	casos := []struct{ nome, metodo, caminho, proibido string }{
		{"logs", http.MethodGet, "/api/agent/v1/build/logs/", "saida-secretissima-do-build"},
		{"artifacts", http.MethodGet, "/api/agent/v1/build/artifacts/", "raiz-do-dono"},
		{"cancel", http.MethodPost, "/api/agent/v1/build/cancel/", "cancelado"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			// NEGATIVO: outra organizacao nao le nem cancela.
			codigoAlheio, corpoAlheio := pedirBuildMetodo(t, alheio, caso.metodo, caso.caminho+id)
			if codigoAlheio != http.StatusNotFound {
				t.Fatalf("%s alheio devia dar 404, veio %d: %s", caso.nome, codigoAlheio, corpoAlheio)
			}
			if strings.Contains(corpoAlheio, caso.proibido) {
				t.Fatalf("%s alheio vazou %q: %s", caso.nome, caso.proibido, corpoAlheio)
			}
			// SEM ORACULO: identico a um id inexistente.
			codigoInexistente, corpoInexistente := pedirBuildMetodo(t, alheio, caso.metodo, caso.caminho+"nao-existe-xyz")
			if codigoInexistente != http.StatusNotFound {
				t.Fatalf("%s de id inexistente devia dar 404, veio %d", caso.nome, codigoInexistente)
			}
			normalizado := strings.ReplaceAll(corpoAlheio, id, "<id>")
			esperado := strings.ReplaceAll(corpoInexistente, "nao-existe-xyz", "<id>")
			if normalizado != esperado {
				t.Fatalf("%s: negacao de build alheio difere da de id inexistente\n  alheio:      %s\n  inexistente: %s",
					caso.nome, normalizado, esperado)
			}
		})
	}

	// POSITIVO: o dono le a saida e cancela.
	codigoLogs, corpoLogs := pedirBuildMetodo(t, dono, http.MethodGet, "/api/agent/v1/build/logs/"+id)
	if codigoLogs != http.StatusOK {
		t.Fatalf("o dono devia ler a saida, veio %d: %s", codigoLogs, corpoLogs)
	}
	if !strings.Contains(corpoLogs, "saida-secretissima-do-build") {
		t.Fatalf("o dono devia ver a saida; corpo: %s", corpoLogs)
	}
	codigoArtefactos, corpoArtefactos := pedirBuildMetodo(t, dono, http.MethodGet, "/api/agent/v1/build/artifacts/"+id)
	if codigoArtefactos != http.StatusOK {
		t.Fatalf("o dono devia ler os artefactos, veio %d: %s", codigoArtefactos, corpoArtefactos)
	}
	// O build ja' terminou => cancelar devolve 409 ao dono (a maquina de estados, nao a fronteira).
	codigoCancelar, corpoCancelar := pedirBuildMetodo(t, dono, http.MethodPost, "/api/agent/v1/build/cancel/"+id)
	if codigoCancelar != http.StatusConflict {
		t.Fatalf("cancelar build terminado devia dar 409, veio %d: %s", codigoCancelar, corpoCancelar)
	}
}
