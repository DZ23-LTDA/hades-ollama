package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestDevTokenUsesStrictJSONDecoder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_AUTH_DEV", "true")
	t.Setenv("OLLAMA_AGENT_AUTH_DEV_SECRET", "dev-secret")
	auth, err := agent.NewAuthStore("")
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{auth: auth}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/auth/dev/token", strings.NewReader(`{"email":"dev@example.com","name":"Dev","organization":"Dev Org","unexpected":true}`))
	context.Request.RemoteAddr = "127.0.0.1:12345"
	context.Request.Header.Set("X-Ollama-Agent-Dev-Secret", "dev-secret")

	api.devToken(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("dev token status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
