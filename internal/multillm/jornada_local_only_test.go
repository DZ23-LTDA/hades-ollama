package multillm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// E2E nº 11 -- a JORNADA, nao as pecas.
//
// Um pedido HTTP REAL atravessa o middleware do gateway com `LocalOnly=true`. Dois contadores
// dizem o que aconteceu: quantos pedidos o upstream LOCAL recebeu, quantos o provedor REMOTO de
// fachada recebeu, e quantas vezes o pedido "passou adiante" sem decisao (fallthrough).
//
// As tres provas anteriores (transporte, registry, Resolve) mediam partes. Esta mede o caminho.
func TestJornadaLocalOnlyNaoEscapaNemPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var pedidosLocais, pedidosRemotos, passouAdiante int32

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pedidosLocais, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"provider":"local"}`))
	}))
	defer local.Close()

	remoto := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pedidosRemotos, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer remoto.Close()

	registro := &Registry{
		providers: map[string]Provider{
			"local": {
				Name: "local", Type: ProviderTypeOpenAICompatible,
				BaseURL: local.URL, Locality: "local",
				// Loopback exige permissao de rede privada: sem isto o guarda de egresso recusa
				// o proprio upstream local ("resolved to a private or link-local address").
				AllowPrivate: true,
				Paths:        []string{"/v1/chat/completions"},
			},
			"remoto": {
				Name: "remoto", Type: ProviderTypeOpenAICompatible,
				BaseURL: remoto.URL, Locality: "cloud",
				// AllowPrivate=true de proposito: e' o atributo que ANTIGAMENTE deixava passar
				// um remoto num pedido local-only. Agora nao pode fazer diferenca.
				AllowPrivate: true,
				Paths:        []string{"/v1/chat/completions"},
			},
		},
		models: map[string]Model{
			"local/modelo":  {ID: "local/modelo", Provider: "local", Available: true},
			"remoto/modelo": {ID: "remoto/modelo", Provider: "remoto", Available: true},
		},
	}

	gateway := NewGateway(registro, NewProviderSafeHTTPClient(5*time.Second, true))
	gateway.LocalOnly = true

	router := gin.New()
	router.POST("/v1/chat/completions", gateway.Middleware(), func(c *gin.Context) {
		atomic.AddInt32(&passouAdiante, 1)
		c.JSON(http.StatusOK, gin.H{"fallthrough": true})
	})

	pedir := func(modelo string) *httptest.ResponseRecorder {
		corpo := `{"model":"` + modelo + `","messages":[{"role":"user","content":"ola"}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(corpo))
		req.RemoteAddr = "127.0.0.1:54321" // Authorize aceita loopback quando nao ha gateway key
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// NEGATIVO: modelo REMOTO sob LOCAL_ONLY tem de ser recusado, sem sair e sem passar adiante.
	recRemoto := pedir("remoto/modelo")
	if recRemoto.Code != http.StatusServiceUnavailable {
		t.Fatalf("modelo remoto sob LOCAL_ONLY devia ser 503, veio %d (%s)", recRemoto.Code, recRemoto.Body.String())
	}
	if n := atomic.LoadInt32(&pedidosRemotos); n != 0 {
		t.Fatalf("o provedor remoto recebeu %d pedido(s): houve egresso sob LOCAL_ONLY", n)
	}
	if n := atomic.LoadInt32(&passouAdiante); n != 0 {
		t.Fatalf("o pedido passou adiante %d vez(es) em vez de ser recusado", n)
	}
	if n := atomic.LoadInt32(&pedidosLocais); n != 0 {
		t.Fatalf("o upstream local foi contactado %d vez(es) para um modelo remoto", n)
	}

	// POSITIVO: modelo LOCAL sob LOCAL_ONLY e' servido pelo upstream local.
	recLocal := pedir("local/modelo")
	if n := atomic.LoadInt32(&pedidosLocais); n != 1 {
		t.Fatalf("o upstream local devia receber 1 pedido, recebeu %d (status %d)", n, recLocal.Code)
	}
	if recLocal.Code != http.StatusOK {
		t.Fatalf("modelo local devia ser servido, veio %d (%s)", recLocal.Code, recLocal.Body.String())
	}
	if !strings.Contains(recLocal.Body.String(), "local") {
		t.Fatalf("a resposta devia vir do provedor local: %s", recLocal.Body.String())
	}
	if n := atomic.LoadInt32(&pedidosRemotos); n != 0 {
		t.Fatalf("o remoto recebeu %d pedido(s) no total: a jornada vazou", n)
	}
}
