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

	ctx := newOriginTestContext(t, http.MethodPost, "")
	ctx.Request.RemoteAddr = "127.0.0.1:54321"
	if !isDevTokenRequestAllowed(ctx) {
		t.Fatal("loopback direto deveria continuar emitindo token de desenvolvimento")
	}

	for _, header := range []string{"X-Forwarded-For", "X-Real-Ip", "Forwarded", "X-Forwarded-Host", "X-Client-Ip"} {
		proxied := newOriginTestContext(t, http.MethodPost, "")
		proxied.Request.RemoteAddr = "127.0.0.1:54321"
		proxied.Request.Header.Set(header, "203.0.113.10")
		if isDevTokenRequestAllowed(proxied) {
			t.Fatalf("request com %s deveria ser recusado: loopback vem do proxy, não do cliente", header)
		}
	}
}

func TestCompanionSecureRequestDoesNotTrustClientHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/agent/v1/devices/dev-1/connect", nil)
	request.Header.Set("X-Forwarded-Proto", "https")

	t.Setenv("OLLAMA_AGENT_TRUSTED_TLS_PROXY", "")
	if companionSecureRequest(request) {
		t.Fatal("X-Forwarded-Proto enviado pelo cliente não prova TLS")
	}

	t.Setenv("OLLAMA_AGENT_TRUSTED_TLS_PROXY", "1")
	if !companionSecureRequest(request) {
		t.Fatal("com proxy TLS declarado pelo operador, o header deve valer")
	}
}
