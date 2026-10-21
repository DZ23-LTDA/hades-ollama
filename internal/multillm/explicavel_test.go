package multillm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// FATIA 2 / F1: o cliente tem de saber QUEM serviu e em que TENTATIVA, sem ler logs.
func TestProxyExplicaQuemServiu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer upstream.Close()

	t.Setenv("REMOTE_KEY", "provider-secret")
	r := &Registry{
		providers: map[string]Provider{"remote": {Name: "remote", Type: "openai-compatible", BaseURL: upstream.URL + "/v1", APIKeyEnv: "REMOTE_KEY", AllowPrivate: true}},
		models:    map[string]Model{"remote/model": {ID: "remote/model", UpstreamID: "upstream-model", Provider: "remote", Available: true}},
	}
	g := NewGateway(r, upstream.Client())
	router := gin.New()
	router.Use(g.Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("nao foi proxied") })

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"remote/model","messages":[]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Hades-Provider"); got != "remote" {
		t.Fatalf("X-Hades-Provider = %q, queria remote", got)
	}
	if got := rec.Header().Get("X-Hades-Model"); got != "remote/model" {
		t.Fatalf("X-Hades-Model = %q, queria remote/model", got)
	}
	if got := rec.Header().Get("X-Hades-Attempt"); got != "1" {
		t.Fatalf("X-Hades-Attempt = %q, queria 1", got)
	}
	// Sem falha anterior nao ha cadeia -- e o cabecalho nao deve aparecer vazio.
	if got := rec.Header().Get("X-Hades-Fallback"); got != "" {
		t.Fatalf("X-Hades-Fallback devia estar ausente, veio %q", got)
	}
	// F5: a explicacao nao pode vazar a credencial.
	if corpo := rec.Body.String() + rec.Header().Get("X-Hades-Fallback"); strings.Contains(corpo, "provider-secret") {
		t.Fatal("a explicacao vazou a credencial")
	}
}

// FATIA 2 / F3: provedor sem credencial NAO pode servir em silencio nem cair para outro sem avisar.
func TestProxySemCredencialFalhaAlto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chamadas := 0
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamadas++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"nao-devia-chegar"}`))
	}))
	defer upstream.Close()

	// APIKeyEnv aponta para uma variavel que NAO existe.
	t.Setenv("REMOTE_KEY_AUSENTE", "")
	r := &Registry{
		providers: map[string]Provider{"remote": {Name: "remote", Type: "openai-compatible", BaseURL: upstream.URL + "/v1", APIKeyEnv: "REMOTE_KEY_AUSENTE", AllowPrivate: true}},
		models:    map[string]Model{"remote/model": {ID: "remote/model", UpstreamID: "upstream-model", Provider: "remote", Available: true}},
	}
	g := NewGateway(r, upstream.Client())
	router := gin.New()
	router.Use(g.Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("nao foi proxied") })

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"remote/model","messages":[]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("sem credencial devia falhar, veio 200: %s", rec.Body.String())
	}
	// Contagem medida para o relatorio: se o upstream foi chamado sem chave, a falha nao e' clara,
	// e' so' um erro do provedor. O teste regista o valor para nao fingir que provou mais do que prova.
	t.Logf("chamadas ao upstream sem credencial = %d (esperado 0 se a credencial for checada antes)", chamadas)
}
