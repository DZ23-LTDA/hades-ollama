package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestUnsafeEgressIPBlocksNonGlobalAndSpecialUseRanges(t *testing.T) {
	for _, test := range []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public", ip: "8.8.8.8"},
		{name: "cgnat", ip: "100.64.0.1", want: true},
		{name: "benchmark", ip: "198.18.0.1", want: true},
		{name: "documentation", ip: "203.0.113.1", want: true},
		{name: "ipv6 documentation", ip: "2001:db8::1", want: true},
		{name: "mapped private", ip: "::ffff:10.0.0.1", want: true},
		{name: "mapped public", ip: "::ffff:8.8.8.8", want: false},
		{name: "loopback", ip: "127.0.0.1", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := unsafeEgressIP(net.ParseIP(test.ip)); got != test.want {
				t.Fatalf("unsafeEgressIP(%q)=%v, want %v", test.ip, got, test.want)
			}
		})
	}
}

func TestValidateOutboundPayloadRejectsSensitiveKeysAndValues(t *testing.T) {
	for name, payload := range map[string]any{
		"nested sensitive key":              map[string]any{"request": map[string]any{"api_key": "not-a-real-key"}},
		"hyphenated API key":                map[string]any{"Api-Key": "not-a-real-key"},
		"authorization header":              map[string]any{"Authorization": "Basic dXNlcjpwYXNz"},
		"cookie header":                     map[string]any{"Cookie": "session=example"},
		"opaque basic header":               map[string]any{"value": "Authorization: Basic dXNlcjpwYXNzd29yZA=="},
		"opaque cookie value":               map[string]any{"value": "Cookie: session=opaque-session-value"},
		"opaque session value":              map[string]any{"value": "sid=opaque-session-value"},
		"credential assignment":             map[string]any{"body": "password=example-secret-value"},
		"embedded access token JSON":        map[string]any{"objective": "process webhook payload: {\"access_token\":\"ordinary-secret\"}"},
		"embedded API key JSON":             map[string]any{"objective": "review {\"api_key\":\"ordinary-secret\"}"},
		"embedded authorization JSON":       map[string]any{"objective": "body: {\"authorization\":\"ordinary-secret\"}"},
		"escaped nested JSON key":           map[string]any{"objective": `payload: {\\\"nested\\\":{\\\"refresh_token\\\":\\\"ordinary-secret\\\"}}`},
		"Unicode-escaped sensitive key":     map[string]any{"objective": `payload: {"api\u005fkey":"ordinary-secret"}`},
		"sensitive key after benign key":    map[string]any{"objective": `payload: {"status":"failed","customCredential":"ordinary-secret"}`},
		"nested JSON string Unicode key":    map[string]any{"objective": `\"{\\\"api\\\\u005fkey\\\":\\\"ordinary-secret\\\"}\"`},
		"double-escaped Unicode delimiters": map[string]any{"objective": `body: \\u007b\\u0022api_key\\u0022:\\u0022ordinary-secret\\u0022\\u007d`},
		"provider token":                    map[string]any{"items": []any{"ghp_" + strings.Repeat("A", 30)}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateOutboundPayload(payload); err == nil || !strings.Contains(err.Error(), "DLP") {
				t.Fatalf("validation error = %v, want fail-closed DLP block", err)
			}
		})
	}
	longEmbeddedKey := `{"` + strings.Repeat("a", 300) + "customCredential" + `":"ordinary-secret"}`
	if err := validateOutboundPayload(map[string]any{"objective": longEmbeddedKey}); err == nil || !strings.Contains(err.Error(), "DLP") {
		t.Fatalf("overlong embedded sensitive key error = %v, want fail-closed DLP block", err)
	}
	if err := validateOutboundPayload(map[string]any{"query": "summarize the latest local report", "limit": 10}); err != nil {
		t.Fatalf("ordinary payload rejected: %v", err)
	}
	if err := validateOutboundPayload(map[string]any{"tokenizer": "bpe", "authorization_url": "https://example.test/oauth/authorize", "session": "session-1"}); err != nil {
		t.Fatalf("benign fields rejected: %v", err)
	}
	if err := validateOutboundPayload(map[string]any{"items": make([]string, maxOutboundDLPNodes+1)}); err == nil {
		t.Fatal("excessive collection size was not rejected before encoding")
	}
	if err := validateOutboundPayload(strings.Repeat("x", maxOutboundDLPScanBytes+1)); err == nil {
		t.Fatal("oversized payload was not rejected")
	}
}

func TestConnectorBlocksSensitiveBodyBeforeNetworkRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_test", ConnectorConfig{ID: "egress", Provider: "test", BaseURL: server.URL, Operations: []ConnectorOperation{{Name: "write", Methods: []string{"POST"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	_, _, err := manager.CallForOrganization(context.Background(), "org_test", "egress", "write", "POST", "/resource", []byte(`{"api_key":"example-value"}`))
	if err == nil || !strings.Contains(err.Error(), "DLP") {
		t.Fatalf("connector error = %v, want DLP block", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("connector sent %d request(s) with sensitive payload", requests.Load())
	}
}

func TestMCPManagersBlockSensitiveParamsBeforeEgress(t *testing.T) {
	t.Run("local stdio", func(t *testing.T) {
		t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
		manager := NewMCPManager()
		if err := manager.Register(MCPServerConfig{ID: "echo", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
			t.Fatal(err)
		}
		defer manager.StopAll()
		_, err := manager.CallGlobal(context.Background(), "echo", "echo", map[string]any{"nested": map[string]any{"access_token": "example-value"}})
		if err == nil || !strings.Contains(err.Error(), "DLP") {
			t.Fatalf("local MCP error = %v, want DLP block", err)
		}
	})

	t.Run("remote HTTP", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		manager := NewRemoteMCPManager()
		if err := manager.Register(RemoteMCPServerConfig{ID: "remote", URL: server.URL, AllowedMethods: []string{"tools/call"}}); err != nil {
			t.Fatal(err)
		}
		_, err := manager.CallGlobal(context.Background(), "remote", "tools/call", map[string]any{"secret": "example-value"})
		if err == nil || !strings.Contains(err.Error(), "DLP") {
			t.Fatalf("remote MCP error = %v, want DLP block", err)
		}
		if requests.Load() != 0 {
			t.Fatalf("remote MCP sent %d request(s) with sensitive payload", requests.Load())
		}
	})
}
