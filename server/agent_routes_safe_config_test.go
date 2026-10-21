package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestSafeConfigRedactsValuesAndReportsConfiguredStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_MODEL", "qwen-test")
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", "postgres://secret-user:secret-pass@db.internal/agent")
	t.Setenv("OLLAMA_AGENT_REDIS_URL", "redis://secret:secret@redis.internal/0")
	t.Setenv("OLLAMA_AGENT_CONNECTORS", "/private/connectors.json")
	t.Setenv("OLLAMA_AGENT_MEDIA_BASE_URL", "https://media.internal")

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	(&agentAPI{authRequired: true}).safeConfig(context)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["store"] != "postgres" || body["queue"] != "redis" || body["planner_model_configured"] != true {
		t.Fatalf("unexpected safe config: %#v", body)
	}
	for _, secret := range []string{"secret-user", "secret-pass", "private/connectors.json", "media.internal", "qwen-test"} {
		if containsString(recorder.Body.String(), secret) {
			t.Fatalf("safe config leaked %q: %s", secret, recorder.Body.String())
		}
	}
}

func containsString(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestSafeConfigReportsDurableManifestAsConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, name := range []string{"OLLAMA_AGENT_CONNECTORS", "OLLAMA_AGENT_MCP", "OLLAMA_AGENT_REMOTE_MCP", "OLLAMA_AGENT_MEDIA_BASE_URL", "OLLAMA_AGENT_DEPLOYMENTS"} {
		t.Setenv(name, "")
	}
	manager := agent.NewConnectorManager()
	if err := manager.Register(agent.ConnectorConfig{
		ID:       "durable-test",
		Provider: "fixture",
		BaseURL:  "https://fixture.example.test",
		Operations: []agent.ConnectorOperation{{
			Name:         "health",
			Methods:      []string{"GET"},
			PathPrefixes: []string{"/health"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), Connectors: manager, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	(&agentAPI{runtime: runtime}).safeConfig(context)
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connectors_configured"] != true {
		t.Fatalf("durable connector not reported configured: %#v", body)
	}
	if containsString(recorder.Body.String(), "fixture.example.test") {
		t.Fatalf("safe config leaked connector endpoint: %s", recorder.Body.String())
	}
}
