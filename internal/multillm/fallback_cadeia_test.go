package multillm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// FATIA 2 / F2: quando ha FALLBACK, o cliente tem de saber o que falhou antes -- em ordem.
//
// Dois provedores: o primeiro devolve 500, o segundo serve. A rotação automatica esta' ligada com
// CrossProvider (sem isso `requested == model.ID` volta a tentar o mesmo e nunca ha cadeia).
func TestProxyExplicaCadeiaDeFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	primeiro := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"upstream avariado"}`))
	}))
	defer primeiro.Close()

	segundo := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer segundo.Close()

	t.Setenv("KEY1", "segredo-um")
	t.Setenv("KEY2", "segredo-dois")
	r := &Registry{
		providers: map[string]Provider{
			"p1": {Name: "p1", Type: "openai-compatible", BaseURL: primeiro.URL + "/v1", APIKeyEnv: "KEY1", AllowPrivate: true},
			"p2": {Name: "p2", Type: "openai-compatible", BaseURL: segundo.URL + "/v1", APIKeyEnv: "KEY2", AllowPrivate: true},
		},
		models: map[string]Model{
			"m1": {ID: "m1", UpstreamID: "up1", Provider: "p1", Available: true},
			"m2": {ID: "m2", UpstreamID: "up2", Provider: "p2", Available: true},
		},
		rotation: &RotationConfig{CrossProvider: boolPtr(true), MaxAttempts: 3},
	}
	g := NewGateway(r, primeiro.Client())
	router := gin.New()
	router.Use(g.Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("nao foi proxied") })

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"m1","messages":[]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200 do segundo provedor, veio %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Hades-Attempt"); got != "2" {
		t.Fatalf("X-Hades-Attempt = %q, queria 2 (houve fallback)", got)
	}
	if got := rec.Header().Get("X-Hades-Provider"); got != "p2" {
		t.Fatalf("X-Hades-Provider = %q, queria p2", got)
	}
	cadeia := rec.Header().Get("X-Hades-Fallback")
	if cadeia == "" {
		t.Fatal("X-Hades-Fallback vazio: o cliente nao sabe o que falhou antes")
	}
	if !strings.Contains(cadeia, "p1") || !strings.Contains(cadeia, "m1") {
		t.Fatalf("cadeia nao nomeia o provedor/modelo que falhou: %q", cadeia)
	}
	// F5: a cadeia nao pode vazar credenciais.
	for _, segredo := range []string{"segredo-um", "segredo-dois"} {
		if strings.Contains(cadeia, segredo) {
			t.Fatalf("a cadeia vazou credencial (%s): %q", segredo, cadeia)
		}
	}
	_ = json.Marshal // mantem o import explicito para futuras assercoes de corpo
}
