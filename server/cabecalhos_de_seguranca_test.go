package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// A CSP minima existe e tem as tres directivas que NAO restringem o que a SPA carrega.
func TestCSPEmitidaComAsTresDirectivas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	garantirCabecalhosDeSeguranca(r)
	r.GET("/qualquer", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	pedido := httptest.NewRequest(http.MethodGet, "/qualquer", nil)
	resposta := httptest.NewRecorder()
	r.ServeHTTP(resposta, pedido)

	csp := resposta.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("a resposta tem de trazer Content-Security-Policy")
	}
	for _, directiva := range []string{"object-src 'none'", "base-uri 'self'", "frame-ancestors 'self'"} {
		if !strings.Contains(csp, directiva) {
			t.Fatalf("a CSP tem de conter %q, veio: %q", directiva, csp)
		}
	}
	if nosniff := resposta.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Fatalf("X-Content-Type-Options devia ser nosniff, veio %q", nosniff)
	}
}

// A politica NAO pode restringir scripts/estilos/ligacoes sem medicao: se alguem acrescentar
// `script-src 'self'` sem verificar a app, este teste obriga a decisao a ser consciente.
func TestCSPNaoRestringeScriptsSemDecisaoDeclarada(t *testing.T) {
	for _, proibida := range []string{"script-src", "style-src", "connect-src", "default-src"} {
		if strings.Contains(PoliticaDeSegurancaDeConteudo, proibida) {
			t.Fatalf("a politica passou a conter %q: isso exige medir o que a UI carrega (bundles, "+
				"websockets, workers) antes de ligar -- e actualizar este teste com a medicao", proibida)
		}
	}
}

// O middleware aplica-se a TODAS as respostas (nao so' ao HTML): a CSP em JSON e' ignorada pelo
// browser, e evita ter de saber qual rota serve o documento.
func TestCSPAplicadaATodasAsRotas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	garantirCabecalhosDeSeguranca(r)
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "<html></html>") })
	r.GET("/api/version", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"version": "x"}) })

	for _, caminho := range []string{"/", "/api/version"} {
		pedido := httptest.NewRequest(http.MethodGet, caminho, nil)
		resposta := httptest.NewRecorder()
		r.ServeHTTP(resposta, pedido)
		if resposta.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("%s saiu sem CSP", caminho)
		}
	}
}
