package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// servidorQR monta o handler isolado com uma instancia Evolution falsa.
func servidorQR(t *testing.T, evolutionURL string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_HOST", "127.0.0.1:11434") // modo local: sem auth, admin passa

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	gw := agent.NewWhatsAppGateway(agent.WhatsAppGatewayConfig{
		PrimaryBackend: agent.WhatsAppBackendEvolution,
		Evolution: agent.EvolutionConfig{
			BaseURL: evolutionURL, APIKey: "chave-secreta", Instance: "hades",
			WebhookSecret: "segredo-webhook",
		},
	}, runtime)
	runtime.SetWhatsApp(gw)

	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	r := gin.New()
	r.GET("/api/agent/v1/whatsapp/qr", api.whatsappQR)
	return r
}

// POSITIVO: devolve o QR como data-URL e NUNCA expoe credenciais.
func TestWhatsAppQRDevolveQRSemVazarCredenciais(t *testing.T) {
	evolution := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "chave-secreta" {
			t.Errorf("api key nao enviada no cabecalho apikey")
		}
		if strings.Contains(r.URL.Path, "/instance/connectionState/") {
			_, _ = w.Write([]byte(`{"instance":{"state":"connecting"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"base64":"iVBORw0KGgo=","pairingCode":"XY-99"}`))
	}))
	defer evolution.Close()

	r := servidorQR(t, evolution.URL)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agent/v1/whatsapp/qr", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}
	corpo := w.Body.String()
	if !strings.Contains(corpo, "data:image/png;base64,") {
		t.Fatalf("resposta sem QR: %s", corpo)
	}
	for _, segredo := range []string{"chave-secreta", "segredo-webhook"} {
		if strings.Contains(corpo, segredo) {
			t.Fatalf("DEFEITO: credencial %q vazou na resposta", segredo)
		}
	}
}

// NEGATIVO: instancia inacessivel -> 502 com motivo, e nenhum QR inventado.
func TestWhatsAppQRErroDaInstancia(t *testing.T) {
	evolution := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer evolution.Close()

	r := servidorQR(t, evolution.URL)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agent/v1/whatsapp/qr", nil))

	if w.Code != http.StatusBadGateway {
		t.Fatalf("esperava 502, veio %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "data:image") {
		t.Fatal("DEFEITO: devolveu QR quando a instancia falhou")
	}
}
