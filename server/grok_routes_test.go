package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/grok"
)

func TestGrokResponsesRejectsStreamBeforeUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	client, err := grok.NewClient(upstream.URL, "key", "grok-4")
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{grok: client}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/grok/responses", strings.NewReader(`{"model":"grok-4","input":"hi","stream":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	api.grokResponses(context)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if upstreamCalled {
		t.Fatal("stream rejection must happen before upstream")
	}
}

func TestGenericExternalExecutionRoutesAreNotRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&agentAPI{}).register(router)
	denied := map[string]bool{
		"POST /api/agent/v1/grok/responses":   false,
		"POST /api/agent/v1/media/image":      false,
		"POST /api/agent/v1/media/video":      false,
		"POST /api/agent/v1/media/speech":     false,
		"POST /api/agent/v1/media/transcribe": false,
		"POST /api/agent/v1/media/vision":     false,
	}
	for _, route := range router.Routes() {
		if _, ok := denied[route.Method+" "+route.Path]; ok {
			denied[route.Method+" "+route.Path] = true
		}
	}
	for route, registered := range denied {
		if registered {
			t.Errorf("generic external execution route %s must remain unavailable; use approved mission tools", route)
		}
	}
}

func TestGrokResponsesBlockSensitivePayloadBeforeUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	client, err := grok.NewClient(upstream.URL, "key", "grok-4")
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{grok: client}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/grok/responses", strings.NewReader(`{"model":"grok-4","input":"ghp_abcdefghijklmnopqrstuvwxyz123456"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	api.grokResponses(context)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sensitive payload status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if upstreamCalled {
		t.Fatal("sensitive payload reached the Grok upstream")
	}
}

func TestGrokResponsesAllowBenignStructPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_test","model":"grok-4","output_text":"provider result {\"api_key\":\"ordinary-secret-value\"}"}`))
	}))
	defer upstream.Close()
	client, err := grok.NewClient(upstream.URL, "key", "grok-4")
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{grok: client}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/grok/responses", strings.NewReader(`{"model":"grok-4","input":"hello safely"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	api.grokResponses(context)
	if recorder.Code != http.StatusOK || !upstreamCalled {
		t.Fatalf("benign struct request status=%d upstream_called=%v body=%s", recorder.Code, upstreamCalled, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "ordinary-secret-value") || strings.Contains(recorder.Body.String(), `"api_key"`) {
		t.Fatalf("Grok response leaked embedded credential: %s", recorder.Body.String())
	}
}

func TestNewAgentGrokClientReadsModelAllowlist(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_GROK_BASE_URL", "https://api.x.ai/v1")
	t.Setenv("OLLAMA_AGENT_GROK_MODEL", "grok-4")
	t.Setenv("OLLAMA_AGENT_GROK_MODELS", "grok-4,grok-4.1")
	client, err := newAgentGrokClient()
	if err != nil {
		t.Fatal(err)
	}
	if len(client.AllowedModels) != 2 || client.AllowedModels[1] != "grok-4.1" {
		t.Fatalf("allowlist=%v", client.AllowedModels)
	}
}
