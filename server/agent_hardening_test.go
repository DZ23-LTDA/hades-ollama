package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newOriginTestContext(t *testing.T, method, origin string) *gin.Context {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, "/api/agent/v1/missions", nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	ctx.Request = request
	return ctx
}

func TestAgentOriginAllowedRejectsOpaqueSchemes(t *testing.T) {
	t.Setenv("OLLAMA_ORIGINS", "")
	for _, origin := range []string{"file://", "app://obsidian.md", "tauri://localhost", "vscode-webview://abc"} {
		if agentOriginAllowed(newOriginTestContext(t, http.MethodPost, origin)) {
			t.Fatalf("origem opaca %q deveria ser recusada na superfície agêntica", origin)
		}
	}
}

func TestAgentOriginAllowedChecksReadsToo(t *testing.T) {
	t.Setenv("OLLAMA_ORIGINS", "https://console.example.com")
	if agentOriginAllowed(newOriginTestContext(t, http.MethodGet, "https://attacker.example")) {
		t.Fatal("GET cross-origin de origem não listada deveria ser recusado")
	}
	if !agentOriginAllowed(newOriginTestContext(t, http.MethodGet, "https://console.example.com")) {
		t.Fatal("GET da origem listada deveria ser permitido")
	}
	if !agentOriginAllowed(newOriginTestContext(t, http.MethodGet, "")) {
		t.Fatal("cliente sem header Origin (CLI/SDK) deveria ser permitido")
	}
}

func TestIsDevTokenRequestAllowedRejectsProxiedRequests(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_AUTH_DEV", "true")
	t.Setenv("OLLAMA_AGENT_AUTH_DEV_SECRET", "dev-secret")

	ctx := newOriginTestContext(t, http.MethodPost, "")
	ctx.Request.RemoteAddr = "127.0.0.1:54321"
	ctx.Request.Header.Set("X-Ollama-Agent-Dev-Secret", "dev-secret")
	if !isDevTokenRequestAllowed(ctx) {
		t.Fatal("loopback direto com segredo correto deveria emitir token de desenvolvimento")
	}

	for _, header := range []string{"X-Forwarded-For", "X-Real-Ip", "Forwarded", "X-Forwarded-Host", "X-Client-Ip"} {
		proxied := newOriginTestContext(t, http.MethodPost, "")
		proxied.Request.RemoteAddr = "127.0.0.1:54321"
		proxied.Request.Header.Set("X-Ollama-Agent-Dev-Secret", "dev-secret")
		proxied.Request.Header.Set(header, "203.0.113.10")
		if isDevTokenRequestAllowed(proxied) {
			t.Fatalf("request com %s deveria ser recusado: loopback vem do proxy, não do cliente", header)
		}
	}
}

func TestDevTokenRequiresOperatorSecret(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_AUTH_DEV", "true")

	// Sem segredo configurado, o endpoint é fail-closed: fecha o caminho do
	// proxy same-host que encaminha para loopback sem adicionar cabeçalhos.
	t.Setenv("OLLAMA_AGENT_AUTH_DEV_SECRET", "")
	clean := newOriginTestContext(t, http.MethodPost, "")
	clean.Request.RemoteAddr = "127.0.0.1:54321"
	if isDevTokenRequestAllowed(clean) {
		t.Fatal("sem segredo configurado, o token de dev deve ser recusado mesmo em loopback limpo")
	}

	// Com segredo configurado mas ausente ou errado no request, recusar.
	t.Setenv("OLLAMA_AGENT_AUTH_DEV_SECRET", "dev-secret")
	missing := newOriginTestContext(t, http.MethodPost, "")
	missing.Request.RemoteAddr = "127.0.0.1:54321"
	if isDevTokenRequestAllowed(missing) {
		t.Fatal("sem o header do segredo, o request deve ser recusado")
	}
	wrong := newOriginTestContext(t, http.MethodPost, "")
	wrong.Request.RemoteAddr = "127.0.0.1:54321"
	wrong.Request.Header.Set("X-Ollama-Agent-Dev-Secret", "errado")
	if isDevTokenRequestAllowed(wrong) {
		t.Fatal("segredo incorreto deve ser recusado")
	}
}

func TestCompanionSecureRequestDoesNotTrustClientHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/agent/v1/devices/dev-1/connect", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	request.RemoteAddr = "127.0.0.1:54321"

	t.Setenv("OLLAMA_AGENT_TRUSTED_TLS_PROXY", "")
	if companionSecureRequest(request) {
		t.Fatal("X-Forwarded-Proto enviado pelo cliente não prova TLS")
	}

	t.Setenv("OLLAMA_AGENT_TRUSTED_TLS_PROXY", "1")
	if !companionSecureRequest(request) {
		t.Fatal("com proxy TLS declarado e peer loopback, o header deve valer")
	}

	// Peer remoto direto: nem com a flag o header forjado pode valer.
	remote := httptest.NewRequest(http.MethodGet, "/api/agent/v1/devices/dev-1/connect", nil)
	remote.Header.Set("X-Forwarded-Proto", "https")
	remote.RemoteAddr = "203.0.113.10:4444"
	if companionSecureRequest(remote) {
		t.Fatal("cliente remoto direto não pode forjar TLS via X-Forwarded-Proto")
	}
}
