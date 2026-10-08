package multillm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolvedPrivateProviderHostIsRejected(t *testing.T) {
	_, err := resolveProviderDestination(context.Background(), Provider{BaseURL: "https://localhost/v1"})
	if err == nil {
		t.Fatal("expected hostname resolving to loopback to be rejected")
	}
	if _, err := resolveProviderDestination(context.Background(), Provider{BaseURL: "https://localhost/v1", AllowPrivate: true}); err != nil {
		t.Fatalf("explicitly trusted private provider was rejected: %v", err)
	}
}

func TestProviderTransportDialsOnlyApprovedIP(t *testing.T) {
	var dialed string
	base := &http.Transport{DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = address
		return nil, errors.New("test stop")
	}}
	roundTripper, err := pinnedProviderTransport(base, []net.IP{net.ParseIP("192.0.2.10")})
	if err != nil {
		t.Fatal(err)
	}
	transport := roundTripper.(*http.Transport)
	_, _ = transport.DialContext(context.Background(), "tcp", "provider.example:443")
	if dialed != "192.0.2.10:443" {
		t.Fatalf("dialed %q instead of the approved address", dialed)
	}
}

func TestProviderTransportDisablesTLSHooksThatBypassPinnedDial(t *testing.T) {
	base := &http.Transport{
		DialTLS: func(_, _ string) (net.Conn, error) { return nil, errors.New("unsafe legacy hook") },
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("unsafe context hook")
		},
	}
	roundTripper, err := pinnedProviderTransport(base, []net.IP{net.ParseIP("192.0.2.10")})
	if err != nil {
		t.Fatal(err)
	}
	transport := roundTripper.(*http.Transport)
	if transport.DialTLS != nil || transport.DialTLSContext != nil {
		t.Fatal("custom TLS dial hooks can bypass destination pinning")
	}
}

func TestProviderRedirectRejectsHTTPSDowngrade(t *testing.T) {
	check := sameHostRedirects("provider.example", nil)
	next := httptest.NewRequest(http.MethodGet, "http://provider.example/v1", nil)
	if err := check(next, nil); err == nil {
		t.Fatal("expected plaintext redirect to be rejected")
	}
}

func TestProxyRoutesConfiguredModelAndRedactsClientAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotModel, gotAuthorization string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotModel, _ = body["model"].(string)
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
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("request was not proxied") })

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"remote/model","messages":[]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer client-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || gotModel != "upstream-model" {
		t.Fatalf("status=%d upstream model=%q body=%s", rec.Code, gotModel, rec.Body.String())
	}
	if gotAuthorization != "Bearer provider-secret" {
		t.Fatalf("upstream authorization = %q", gotAuthorization)
	}
}

func TestProxyRoutesLocalOmniRouteGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotModel, gotAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"omniroute-test","choices":[{"message":{"role":"assistant","content":"local gateway answer"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	t.Setenv("OMNIROUTE_TEST_KEY", "omniroute-secret")
	r := &Registry{
		providers: map[string]Provider{"omniroute": {
			Name:                  "omniroute",
			Type:                  ProviderTypeOpenAICompatible,
			BaseURL:               upstream.URL + "/v1",
			APIKeyEnv:             "OMNIROUTE_TEST_KEY",
			AllowPrivate:          true,
			AllowInsecureLoopback: true,
		}},
		models: map[string]Model{"omniroute/auto": {ID: "omniroute/auto", UpstreamID: "auto", Provider: "omniroute", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, nil).Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("request was not proxied") })

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"omniroute/auto","messages":[{"role":"user","content":"hello"}]}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer client-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || gotPath != "/v1/chat/completions" || gotModel != "auto" {
		t.Fatalf("status=%d path=%q model=%q body=%s", rec.Code, gotPath, gotModel, rec.Body.String())
	}
	if gotAuthorization != "Bearer omniroute-secret" {
		t.Fatalf("upstream authorization = %q", gotAuthorization)
	}
}

func TestProxyInjectsHarnessRouterMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var metadata map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		metadata, _ = body["metadata"].(map[string]any)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"harnessrouter-test","object":"response","status":"completed"}`))
	}))
	defer upstream.Close()

	r := &Registry{
		providers: map[string]Provider{"harnessrouter": {Name: "harnessrouter", Type: ProviderTypeOpenAICompatible, BaseURL: upstream.URL + "/v1", Paths: []string{"/v1/responses"}, AllowPrivate: true, AllowInsecureLoopback: true}},
		models:    map[string]Model{"harnessrouter/codex": {ID: "harnessrouter/codex", UpstreamID: "gpt-test", HarnessID: "codex", Provider: "harnessrouter", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, nil).Middleware())
	router.POST("/v1/responses", func(c *gin.Context) { t.Fatal("request was not proxied") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"harnessrouter/codex","metadata":{"trace_id":"trace-test"},"input":"hello"}`))
	request.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if metadata["harness_id"] != "codex" || metadata["trace_id"] != "trace-test" {
		t.Fatalf("metadata=%v", metadata)
	}
}

func TestProxyPassesThroughResponsesProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotModel, gotAuthorization string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModel = ""
		gotAuthorization = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp-xai-test","object":"response","status":"completed","output":[]}`))
	}))
	defer upstream.Close()
	t.Setenv("XAI_TEST_KEY", "xai-secret")
	r := &Registry{
		providers: map[string]Provider{"xai": {Name: "xai", Type: ProviderTypeOpenAICompatible, BaseURL: upstream.URL + "/v1", APIKeyEnv: "XAI_TEST_KEY", Paths: []string{"/v1/responses"}, AllowPrivate: true}},
		models:    map[string]Model{"xai/grok-4.7": {ID: "xai/grok-4.7", UpstreamID: "grok-4.7", Provider: "xai", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, upstream.Client()).Middleware())
	router.POST("/v1/responses", func(c *gin.Context) { t.Fatal("request was not proxied") })
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"xai/grok-4.7","input":"hello","tools":[{"type":"web_search"}]}`))
	request.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || gotModel != "grok-4.7" || gotAuthorization != "Bearer xai-secret" {
		t.Fatalf("status=%d model=%q authorization=%q body=%s", recorder.Code, gotModel, gotAuthorization, recorder.Body.String())
	}
}

func TestCLIProviderRequiresExplicitExecutionAndReturnsCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DZ23_CLI_HELPER", "1")
	r := &Registry{
		providers: map[string]Provider{"codex": {Name: "codex", Type: "cli", Executable: os.Args[0], Args: []string{"-test.run=TestCLIHelperProcess"}, AllowExecution: true, PassthroughEnv: []string{"DZ23_CLI_HELPER"}}},
		models:    map[string]Model{"codex/cli": {ID: "codex/cli", UpstreamID: "cli", Provider: "codex", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, nil).Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("request was not executed") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"codex/cli","messages":[{"role":"user","content":"hello"}]}`))
	request.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "assistant: user: hello") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCLIProviderAcceptsNativeGeneratePrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DZ23_CLI_HELPER", "1")
	r := &Registry{
		providers: map[string]Provider{"cli": {Name: "cli", Type: ProviderTypeCLI, Executable: os.Args[0], Args: []string{"-test.run=TestCLIHelperProcess"}, AllowExecution: true, PassthroughEnv: []string{"DZ23_CLI_HELPER"}}},
		models:    map[string]Model{"cli/default": {ID: "cli/default", UpstreamID: "default", Provider: "cli", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, nil).Middleware())
	router.POST("/api/generate", func(c *gin.Context) { t.Fatal("request was not executed") })
	request := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{"model":"cli/default","stream":false,"prompt":"hello generate"}`))
	request.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"response":"assistant: hello generate"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAutoRoutingSkipsProviderWithoutRequestedPath(t *testing.T) {
	r := &Registry{
		providers: map[string]Provider{
			"chat-only": {Name: "chat-only", Type: ProviderTypeOpenAICompatible, Paths: []string{"/v1/chat/completions"}},
			"responses": {Name: "responses", Type: ProviderTypeOpenAICompatible, Paths: []string{"/v1/responses"}},
		},
		models: map[string]Model{
			"chat-only/model": {ID: "chat-only/model", Provider: "chat-only", Available: true, Priority: 100, Capabilities: []string{"coding"}},
			"responses/model": {ID: "responses/model", Provider: "responses", Available: true, Priority: 10, Capabilities: []string{"coding"}},
		},
	}
	model, ok := r.Resolve("auto/coding", Policy{Path: "/v1/responses"})
	if !ok || model.ID != "responses/model" {
		t.Fatalf("resolved %#v, ok=%v", model, ok)
	}
}

func TestBoundedReaderReturnsExplicitLimitError(t *testing.T) {
	reader := &boundedReader{reader: strings.NewReader("abcd"), remaining: 3}
	_, err := io.ReadAll(reader)
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("error = %v", err)
	}
}

func TestOversizedSSEEmitsClientVisibleErrorEvent(t *testing.T) {
	var output strings.Builder
	err := copySSEBounded(&output, strings.NewReader("1234"), 3)
	if !errors.Is(err, errResponseLimit) {
		t.Fatalf("error = %v", err)
	}
	if output.String() != "123\nevent: error\ndata: {\"error\":{\"message\":\"provider response exceeded limit\",\"type\":\"response_too_large\"}}\n\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestGenericAPIKeyAuthDoesNotInjectAnthropicHeader(t *testing.T) {
	t.Setenv("REMOTE_KEY", "secret")
	req := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	applyProviderAuth(req, Provider{APIKeyEnv: "REMOTE_KEY", AuthStyle: AuthStyleAPIKey})
	if req.Header.Get("X-API-Key") != "secret" || req.Header.Get("Anthropic-Version") != "" {
		t.Fatalf("headers = %#v", req.Header)
	}
	applyProviderAuth(req, Provider{APIKeyEnv: "REMOTE_KEY", AuthStyle: AuthStyleAnthropic})
	if req.Header.Get("Anthropic-Version") == "" {
		t.Fatal("Anthropic auth did not add required version header")
	}
}

func TestOpenAIToolCallsAreConvertedToNativeArguments(t *testing.T) {
	calls, err := normalizeOpenAIToolCalls(json.RawMessage(`[{"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Recife\"}"}}]`))
	if err != nil || len(calls) != 1 {
		t.Fatalf("calls=%#v err=%v", calls, err)
	}
	function := calls[0]["function"].(map[string]any)
	arguments := function["arguments"].(map[string]any)
	if arguments["city"] != "Recife" {
		t.Fatalf("arguments=%#v", arguments)
	}

	accumulator := make(map[int]*openAIToolCall)
	accumulateOpenAIToolCalls(accumulator, json.RawMessage(`[{"index":0,"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":"}}]`))
	accumulateOpenAIToolCalls(accumulator, json.RawMessage(`[{"index":0,"function":{"arguments":"\"Recife\"}"}}]`))
	completed, err := normalizeOpenAIToolCalls(normalizedAccumulatedToolCalls(accumulator))
	if err != nil || completed[0]["function"].(map[string]any)["arguments"].(map[string]any)["city"] != "Recife" {
		t.Fatalf("completed=%#v err=%v", completed, err)
	}
}

func TestNativeToolHistoryIsConvertedToOpenAIShape(t *testing.T) {
	messages, err := openAICompatibleMessages(json.RawMessage(`[
      {"role":"assistant","content":"","tool_calls":[{"function":{"name":"weather","arguments":{"city":"Recife"}}}]},
      {"role":"tool","tool_name":"weather","content":"sunny"}
    ]`))
	if err != nil {
		t.Fatal(err)
	}
	calls := messages[0]["tool_calls"].([]any)
	call := calls[0].(map[string]any)
	function := call["function"].(map[string]any)
	if function["arguments"] != `{"city":"Recife"}` || call["id"] == "" {
		t.Fatalf("call=%#v", call)
	}
	if messages[1]["tool_call_id"] != call["id"] || messages[1]["tool_name"] != nil {
		t.Fatalf("tool result=%#v", messages[1])
	}
}

func TestRemoteCallerCannotSpendProviderKeyWithoutGatewayAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := &Registry{
		providers: map[string]Provider{"remote": {Name: "remote", Type: "openai-compatible", BaseURL: "https://example.com/v1"}},
		models:    map[string]Model{"remote/model": {ID: "remote/model", UpstreamID: "model", Provider: "remote", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, nil).Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { t.Fatal("request bypassed authorization") })
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"remote/model","messages":[]}`))
	request.RemoteAddr = "192.0.2.10:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("DZ23_CLI_HELPER") != "1" {
		return
	}
	b, _ := io.ReadAll(os.Stdin)
	_, _ = os.Stdout.WriteString("assistant: " + strings.TrimSpace(string(b)))
	os.Exit(0)
}

func TestProxyLeavesLocalModelsForOllama(t *testing.T) {
	r := NewGateway(&Registry{providers: map[string]Provider{}, models: map[string]Model{}}, http.DefaultClient)
	router := gin.New()
	router.Use(r.Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"qwen:latest"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNativeChatRoutesRemoteModelAndReturnsOllamaShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"remote answer"}}]}`))
	}))
	defer upstream.Close()
	r := &Registry{
		providers: map[string]Provider{"remote": {Name: "remote", Type: "openai-compatible", BaseURL: upstream.URL + "/v1", AllowPrivate: true}},
		models:    map[string]Model{"remote/model": {ID: "remote/model", UpstreamID: "upstream-model", Provider: "remote", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, upstream.Client()).Middleware())
	router.POST("/api/chat", func(c *gin.Context) { t.Fatal("request was not proxied") })
	request := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"model":"remote/model","stream":false,"messages":[{"role":"user","content":"hello"}]}`))
	request.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"content":"remote answer"`) || !strings.Contains(recorder.Body.String(), `"done":true`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAnthropicProviderTranslatesNativeChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotModel, gotSystem string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Header.Get("Anthropic-Version") == "" || r.Header.Get("X-API-Key") != "anthropic-secret" {
			t.Fatalf("missing Anthropic authentication headers: %#v", r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotModel, _ = body["model"].(string)
		gotSystem, _ = body["system"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","model":"claude-test","content":[{"type":"text","text":"anthropic answer"}],"stop_reason":"end_turn"}`))
	}))
	defer upstream.Close()
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-secret")
	r := &Registry{
		providers: map[string]Provider{"anthropic": {Name: "anthropic", Type: ProviderTypeAnthropic, BaseURL: upstream.URL, APIKeyEnv: "ANTHROPIC_API_KEY", AllowPrivate: true}},
		models:    map[string]Model{"anthropic/claude-test": {ID: "anthropic/claude-test", UpstreamID: "claude-test", Provider: "anthropic", Available: true}},
	}
	router := gin.New()
	router.Use(NewGateway(r, upstream.Client()).Middleware())
	router.POST("/api/chat", func(c *gin.Context) { t.Fatal("request was not proxied") })
	request := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"model":"anthropic/claude-test","stream":false,"messages":[{"role":"system","content":"be concise"},{"role":"user","content":"hello"}]}`))
	request.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || gotPath != "/v1/messages" || gotModel != "claude-test" || gotSystem != "be concise" || !strings.Contains(recorder.Body.String(), `"content":"anthropic answer"`) {
		t.Fatalf("status=%d path=%q model=%q system=%q body=%s", recorder.Code, gotPath, gotModel, gotSystem, recorder.Body.String())
	}
}

func TestProxyTranslatesOpenAIStreamingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk-one\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\" chunk-two\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	t.Setenv("STREAM_KEY", "stream-secret")
	t.Setenv("GATEWAY_KEY", "gateway-secret")
	registry := &Registry{
		providers:        map[string]Provider{"stream": {Name: "stream", Type: ProviderTypeOpenAICompatible, BaseURL: upstream.URL + "/v1", APIKeyEnv: "STREAM_KEY", Paths: []string{"/api/chat"}, AllowPrivate: true, AllowInsecureLoopback: true}},
		models:           map[string]Model{"stream/model": {ID: "stream/model", UpstreamID: "stream-model", Provider: "stream", Available: true}},
		gatewayAPIKeyEnv: "GATEWAY_KEY",
	}
	router := gin.New()
	router.Use(NewGateway(registry, upstream.Client()).Middleware())
	router.POST("/api/chat", func(c *gin.Context) { t.Fatal("stream request was not proxied") })
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"model":"stream/model","stream":true,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Authorization", "Bearer gateway-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || gotAuthorization != "Bearer stream-secret" {
		t.Fatalf("status=%d authorization=%q body=%s", rec.Code, gotAuthorization, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "chunk-one") || !strings.Contains(rec.Body.String(), "chunk-two") || !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Fatalf("translated stream missing expected chunks: %s", rec.Body.String())
	}
}
