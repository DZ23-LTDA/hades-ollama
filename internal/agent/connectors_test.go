package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConnectorUsesTenantOAuthCredential(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "connector-test-key")
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser("connector@example.com", "Connector User")
	if err != nil {
		t.Fatal(err)
	}
	organization, _, err := store.CreateOrganization("Connector Org", user)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authorize(user.ID, organization.ID, "write"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreOAuthCredential("github", user.ID, organization.ID, map[string]any{"access_token": "tenant-token", "expires_in": float64(3600)}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tenant-token" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	manager.SetOAuthStore(store)
	if err := manager.RegisterForOrganization(organization.ID, ConnectorConfig{ID: "github", Provider: "github", BaseURL: server.URL, OAuthProvider: "github", Operations: []ConnectorOperation{{Name: "profile", Methods: []string{"GET"}, PathPrefixes: []string{"/user"}}}}); err != nil {
		t.Fatal(err)
	}
	status, response, err := manager.CallForOrganization(context.Background(), organization.ID, "github", "profile", "GET", "/user", nil)
	if err != nil || status != http.StatusOK || response != `{"ok":true}` {
		t.Fatalf("status=%d response=%q err=%v", status, response, err)
	}
}

func TestConnectorRejectsMalformedSuccessfulJSON(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":`))
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_a", ConnectorConfig{ID: "provider", Provider: "test", BaseURL: server.URL, Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	status, response, err := manager.CallForOrganization(context.Background(), "org_a", "provider", "read", "GET", "/", nil)
	if status != http.StatusOK || response != "" || err == nil || !strings.Contains(err.Error(), "invalid successful JSON") {
		t.Fatalf("malformed provider success status=%d response=%q err=%v", status, response, err)
	}
}

func TestConnectorRequiresOrganizationScope(t *testing.T) {
	manager := NewConnectorManager()
	if _, _, err := manager.Call(context.Background(), "github", "profile", "GET", "/user", nil); err == nil {
		t.Fatal("expected direct connector call to require organization scope")
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "", "github", "profile", "GET", "/user", nil); err == nil {
		t.Fatal("expected empty organization scope to be rejected")
	}
}

func TestConnectorRejectsCredentialBearingEndpointURL(t *testing.T) {
	manager := NewConnectorManager()
	err := manager.RegisterForOrganization("org_a", ConnectorConfig{ID: "signed", Provider: "test", BaseURL: "https://provider.example.test/api?x.sig=credential-value", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}})
	if err == nil {
		t.Fatal("accepted connector endpoint containing signed query credentials")
	}
}

func TestConnectorRejectsCrossTenantCallBeforeEgress(t *testing.T) {
	var requests int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_owner", ConnectorConfig{ID: "tenant-owned", Provider: "test", BaseURL: server.URL, Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org_attacker", "tenant-owned", "read", "GET", "/", nil); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-tenant error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("cross-tenant call sent %d request(s)", requests)
	}
}

func TestConnectorListReportsCredentialStateWithoutTokenEnvironment(t *testing.T) {
	t.Setenv("CONNECTOR_STATUS_TOKEN", "configured-token")
	manager := NewConnectorManager()
	if err := manager.Register(ConnectorConfig{ID: "status", Provider: "status", BaseURL: "https://example.test", TokenEnv: "CONNECTOR_STATUS_TOKEN", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	listed := manager.List()
	if len(listed) != 1 || !listed[0].CredentialConfigured || listed[0].TokenEnv != "" {
		t.Fatalf("connector status = %+v", listed)
	}
}

func TestConnectorFailsClosedBeforeEgressWhenTokenEnvIsMissing(t *testing.T) {
	t.Setenv("CONNECTOR_MISSING_TOKEN", "")
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_test", ConnectorConfig{ID: "missing-token", Provider: "test", BaseURL: server.URL, TokenEnv: "CONNECTOR_MISSING_TOKEN", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	_, _, err := manager.CallForOrganization(context.Background(), "org_test", "missing-token", "read", "GET", "/resource", nil)
	if err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("missing credential error=%v", err)
	}
	if requests != 0 {
		t.Fatalf("connector made %d request(s) without credential", requests)
	}
}

func TestConnectorPathPrefixMatchesSegments(t *testing.T) {
	if !connectorPathMatches("/users/123", "/users") {
		t.Fatal("expected child path to match")
	}
	if connectorPathMatches("/users-privileged", "/users") {
		t.Fatal("must not match a different path segment")
	}
	operation := ConnectorOperation{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/domains"}}
	for _, requestPath := range []string{"/domains/../admin", "/domains/%2e%2e/admin", "/domains/%252e%252e/admin", "/domains/%255c..%255cadmin"} {
		if _, allowed := findConnectorOperation([]ConnectorOperation{operation}, "read", http.MethodGet, requestPath); allowed {
			t.Errorf("encoded or literal traversal path %q passed the operation allowlist", requestPath)
		}
		if _, err := validateConnectorRequestPath(requestPath); err == nil {
			t.Errorf("expected traversal path %q to be rejected", requestPath)
		}
	}
}

func TestConnectorEnforcesAllowedOrigins(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	op := ConnectorOperation{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}

	blocked := NewConnectorManager()
	blocked.client = server.Client()
	if err := blocked.RegisterForOrganization("org_test", ConnectorConfig{ID: "c", Provider: "test", BaseURL: server.URL, AllowedOrigins: []string{"https://not-allowed.example.com"}, Operations: []ConnectorOperation{op}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := blocked.CallForOrganization(context.Background(), "org_test", "c", "read", "GET", "/resource", nil); err == nil || !strings.Contains(err.Error(), "allowed_origins") {
		t.Fatalf("expected allowed_origins rejection, got %v", err)
	}

	allowed := NewConnectorManager()
	allowed.client = server.Client()
	if err := allowed.RegisterForOrganization("org_test", ConnectorConfig{ID: "c", Provider: "test", BaseURL: server.URL, AllowedOrigins: []string{server.URL}, Operations: []ConnectorOperation{op}}); err != nil {
		t.Fatal(err)
	}
	if status, _, err := allowed.CallForOrganization(context.Background(), "org_test", "c", "read", "GET", "/resource", nil); err != nil || status != http.StatusOK {
		t.Fatalf("status=%d err=%v", status, err)
	}
}

func TestConnectorEgressBlocksRedirectsAndBoundsPayloads(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/provider-error":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("opaque-upstream-secret-diagnostic"))
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", 2<<20+1)))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_test", ConnectorConfig{ID: "egress", Provider: "test", BaseURL: server.URL, Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET", "POST"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org_test", "egress", "read", "GET", "/redirect", nil); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org_test", "egress", "read", "GET", "/large", nil); err == nil || !strings.Contains(err.Error(), "payload") {
		t.Fatalf("expected response limit rejection, got %v", err)
	}
	if status, response, err := manager.CallForOrganization(context.Background(), "org_test", "egress", "read", "GET", "/provider-error", nil); err == nil || status != http.StatusBadGateway || response != "" || strings.Contains(err.Error(), "opaque-upstream-secret-diagnostic") {
		t.Fatalf("non-2xx connector response leaked provider bytes: status=%d response=%q err=%v", status, response, err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org_test", "egress", "read", "POST", "/final", []byte(strings.Repeat("x", 1<<20+1))); err == nil || !strings.Contains(err.Error(), "payload") {
		t.Fatalf("expected request limit rejection, got %v", err)
	}
}

func TestConnectorApprovalFingerprintTracksEffectiveConfig(t *testing.T) {
	manager := NewConnectorManager()
	config := ConnectorConfig{ID: "connector-a", OrganizationID: "org-a", Provider: "test", BaseURL: "https://api.example.test", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/v1"}}}}
	if err := manager.Register(config); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"connector_id": "connector-a", "operation": "read", "method": "GET", "path": "/v1/items"}
	first, err := manager.ApprovalConfigSHA256("org-a", input)
	if err != nil {
		t.Fatal(err)
	}
	config.BaseURL = "https://other.example.test"
	if err := manager.Register(config); err != nil {
		t.Fatal(err)
	}
	second, err := manager.ApprovalConfigSHA256("org-a", input)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("connector destination change did not invalidate approval fingerprint")
	}
	if _, err := manager.ApprovalConfigSHA256("org-b", input); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-organization fingerprint error=%v", err)
	}
}

func TestConnectorApprovalConfigDriftBlocksProviderRequest(t *testing.T) {
	var destinationCalls int
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		destinationCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()
	manager := NewConnectorManager()
	manager.client = destination.Client()
	config := ConnectorConfig{ID: "connector-drift", OrganizationID: "org-a", Provider: "test", BaseURL: "https://initial.example.test", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/v1"}}}}
	if err := manager.Register(config); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"connector_id": config.ID, "operation": "read", "method": "GET", "path": "/v1/items"}
	approvedHash, err := manager.ApprovalConfigSHA256("org-a", input)
	if err != nil {
		t.Fatal(err)
	}
	config.BaseURL = destination.URL
	if err := manager.Register(config); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.CallForOrganizationWithApproval(context.Background(), "org-a", input, nil, approvedHash); !errors.Is(err, ErrApprovalPayloadChanged) {
		t.Fatalf("stale connector approval error=%v, want ErrApprovalPayloadChanged", err)
	}
	if destinationCalls != 0 {
		t.Fatalf("changed connector destination received %d requests before re-approval", destinationCalls)
	}
}

func TestConnectorDialRejectsPrivateResolvedAnswersBeforeTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
		accepted <- struct{}{}
	}()
	lookup := func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("203.0.113.8")}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = connectorDialContextWithResolver(ctx, "tcp", "connector.example:"+portOf(listener.Addr().String()), lookup)
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected pre-resolution private rejection, got %v", err)
	}
	select {
	case <-accepted:
		t.Fatal("private connector destination received TCP before refusal")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestConnectorLoopbackHostnameCannotResolveToPublicAddress(t *testing.T) {
	ctx := context.WithValue(context.Background(), connectorLoopbackContextKey{}, true)
	lookup := func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("203.0.113.8")}, nil
	}
	if _, err := connectorDialContextWithResolver(ctx, "tcp", "localhost:443", lookup); err == nil || !strings.Contains(err.Error(), "outside loopback") {
		t.Fatalf("connector loopback accepted a public DNS answer: %v", err)
	}
}

func TestConnectorRedactsSuccessfulProviderResponse(t *testing.T) {
	const secret = "connector-provider-secret-value"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"api_key":"` + secret + `","nested":{"message":"refresh_token=connector-refresh-secret"}}`))
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org_a", ConnectorConfig{ID: "safe", Provider: "test", BaseURL: server.URL, Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	_, response, err := manager.CallForOrganization(context.Background(), "org_a", "safe", "read", http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response, secret) || strings.Contains(response, "connector-refresh-secret") {
		t.Fatalf("provider secret survived: %s", response)
	}
	if !strings.Contains(response, "[REDACTED]") {
		t.Fatalf("provider response was not redacted: %s", response)
	}
}
