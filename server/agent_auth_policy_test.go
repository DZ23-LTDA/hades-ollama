package server

import (
	"encoding/base32"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestAgentAuthRequiredDefaultsToLoopbackOnly(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "127.0.0.1:11434")
	t.Setenv("OLLAMA_AGENT_AUTH_REQUIRED", "")
	required, err := agentAuthRequired()
	if err != nil {
		t.Fatal(err)
	}
	if required {
		t.Fatal("loopback default should allow local-first mode without auth")
	}

	t.Setenv("OLLAMA_HOST", "0.0.0.0:11434")
	required, err = agentAuthRequired()
	if err != nil {
		t.Fatal(err)
	}
	if !required {
		t.Fatal("non-loopback bind must require auth by default")
	}
}

func TestAgentAuthCannotBeDisabledOnNonLoopbackBind(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "0.0.0.0:11434")
	t.Setenv("OLLAMA_AGENT_AUTH_REQUIRED", "false")
	required, err := agentAuthRequired()
	if err != nil {
		t.Fatal(err)
	}
	if !required {
		t.Fatal("non-loopback bind must not accept auth=false")
	}
}

func TestOnlyCompanionConnectRouteUsesDeviceHandshake(t *testing.T) {
	if !isCompanionConnectRoute("/api/agent/v1/devices/:id/connect") {
		t.Fatal("device connect route should use the device handshake")
	}
	for _, path := range []string{
		"/api/agent/v1/missions/:id/connect",
		"/api/agent/v1/connect",
		"/api/agent/v1/devices/:id/connect/extra",
	} {
		if isCompanionConnectRoute(path) {
			t.Fatalf("unexpected auth bypass for %q", path)
		}
	}
}

func TestPublicSSORouteDoesNotBypassAuthenticatedOAuthLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{
		"/api/agent/v1/auth/oauth/github/start",
		"/api/agent/v1/auth/oauth/github/callback",
		"/api/agent/v1/auth/saml/enterprise/start",
		"/api/agent/v1/auth/saml/enterprise/metadata",
		"/api/agent/v1/auth/saml/enterprise/acs",
	} {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("GET", path, nil)
		if !isPublicSSORoute(context) {
			t.Fatalf("expected public SSO route: %s", path)
		}
	}
	for _, path := range []string{
		"/api/agent/v1/auth/oauth/github/refresh",
		"/api/agent/v1/auth/oauth/github/revoke",
		"/api/agent/v1/auth/saml/enterprise/disable",
	} {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("POST", path, nil)
		if isPublicSSORoute(context) {
			t.Fatalf("unexpected public SSO bypass: %s", path)
		}
	}
}

func TestAgentOriginPolicyBlocksCrossSiteMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name   string
		method string
		origin string
		want   bool
	}{
		{name: "same local web app", method: "POST", origin: "http://localhost:3000", want: true},
		{name: "native bearer client", method: "POST", origin: "", want: true},
		{name: "cross site mutation", method: "POST", origin: "https://evil.example", want: false},
		// Leitura cross-origin não é inofensiva nesta API: /missions, /traces,
		// /connectors e /egress/logs expõem estado operacional. A origem passa
		// a valer para GET também.
		{name: "cross site read", method: "GET", origin: "https://evil.example", want: false},
		{name: "preflight", method: "OPTIONS", origin: "https://evil.example", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(test.method, "/api/agent/v1/missions", nil)
			if test.origin != "" {
				context.Request.Header.Set("Origin", test.origin)
			}
			if got := agentOriginAllowed(context); got != test.want {
				t.Fatalf("agentOriginAllowed(%s, %s) = %v, want %v", test.method, test.origin, got, test.want)
			}
		})
	}
}

func TestDevTokenOnlyAllowsActualLoopback(t *testing.T) {
	for _, test := range []struct {
		name string
		addr string
		want bool
	}{
		{name: "ipv4 loopback", addr: "127.0.0.1:1234", want: true},
		{name: "ipv6 loopback", addr: "[::1]:1234", want: true},
		{name: "private network is not loopback", addr: "192.168.1.10:1234", want: false},
		{name: "missing peer is denied", addr: "", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isLoopbackRemoteAddr(test.addr); got != test.want {
				t.Fatalf("isLoopbackRemoteAddr(%q) = %v, want %v", test.addr, got, test.want)
			}
		})
	}
}

func TestApprovalApproverRequiresOwnerOrAdminWhenAuthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: true}
	for _, test := range []struct {
		role agent.Role
		want bool
	}{
		{role: agent.RoleViewer, want: false},
		{role: agent.RoleOperator, want: false},
		{role: agent.RoleAdmin, want: true},
		{role: agent.RoleOwner, want: true},
	} {
		t.Run(string(test.role), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Set("agent.membership", agent.Membership{Role: test.role})
			if got := api.requireApprovalApprover(context); got != test.want {
				t.Fatalf("role %s approval access=%v want=%v", test.role, got, test.want)
			}
		})
	}
}

func TestAgentAuthMiddlewareThrottlesMFAFailuresAndPersistsLockout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "server-mfa-throttle-test-key-long-enough")
	root := t.TempDir()
	auth, err := agent.NewAuthStore(root)
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.CreateUser("mfa-middleware@example.com", "MFA Middleware")
	if err != nil {
		t.Fatal(err)
	}
	organization, _, err := auth.CreateOrganization("MFA Org", user)
	if err != nil {
		t.Fatal(err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("0123456789012345"))
	if _, err := auth.EnableMFA(user.ID, secret); err != nil {
		t.Fatal(err)
	}
	rawToken, _, err := auth.IssueToken(user.ID, organization.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{auth: auth, authRequired: true}
	for attempt := 1; attempt <= 5; attempt++ {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/missions", nil)
		ctx.Request.Header.Set("Authorization", "Bearer "+rawToken)
		ctx.Request.Header.Set("X-Ollama-MFA-Code", "000000")
		ctx.Request.RemoteAddr = "127.0.0.1:4567"
		api.authMiddleware(ctx)
		want := http.StatusUnauthorized
		if attempt == 5 {
			want = http.StatusTooManyRequests
			if recorder.Header().Get("Retry-After") == "" {
				t.Fatal("throttled MFA response omitted Retry-After")
			}
		}
		if recorder.Code != want || !ctx.IsAborted() {
			t.Fatalf("attempt %d status=%d want=%d aborted=%v body=%s", attempt, recorder.Code, want, ctx.IsAborted(), recorder.Body.String())
		}
	}
	reloaded, err := agent.NewAuthStore(root)
	if err != nil {
		t.Fatal(err)
	}
	api.auth = reloaded
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/missions", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+rawToken)
	ctx.Request.Header.Set("X-Ollama-MFA-Code", "000000")
	ctx.Request.RemoteAddr = "127.0.0.1:4567"
	api.authMiddleware(ctx)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("persisted lockout status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLocalAuthMiddlewareAppliesOriginPolicyToMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: false}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/api/agent/v1/missions", nil)
	context.Request.Header.Set("Origin", "https://evil.example")

	api.authMiddleware(context)

	if recorder.Code != 403 || !context.IsAborted() {
		t.Fatalf("local cross-site mutation status=%d aborted=%v body=%s", recorder.Code, context.IsAborted(), recorder.Body.String())
	}
}

func TestAgentOriginWildcardMatchesPortOnlyOnExactHost(t *testing.T) {
	t.Setenv("OLLAMA_ORIGINS", "https://trusted.example:*")
	for _, test := range []struct {
		origin string
		want   bool
	}{
		{origin: "https://trusted.example:8443", want: true},
		{origin: "https://trusted.example.evil:8443", want: false},
		{origin: "https://trusted.example", want: false},
	} {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("POST", "/api/agent/v1/missions", nil)
		context.Request.Header.Set("Origin", test.origin)
		if got := agentOriginAllowed(context); got != test.want {
			t.Fatalf("agentOriginAllowed(%q)=%v, want %v", test.origin, got, test.want)
		}
	}
}

func TestConfigureAgentTLSRequiresTLSForNonLoopbackBind(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_TLS_CERT_FILE", "")
	t.Setenv("OLLAMA_AGENT_TLS_KEY_FILE", "")
	t.Setenv("OLLAMA_AGENT_REQUIRE_MTLS", "")

	for _, address := range []net.Addr{
		&net.TCPAddr{IP: net.IPv4zero, Port: 11434},
		&net.TCPAddr{IP: net.IPv6unspecified, Port: 11434},
		&net.TCPAddr{IP: net.ParseIP("192.168.1.10"), Port: 11434},
	} {
		server := &http.Server{}
		if _, err := configureAgentTLS(server, address); err == nil {
			t.Fatalf("configureAgentTLS(%s) accepted a non-loopback listener without TLS", address)
		}
	}
}

func TestConfigureAgentTLSAllowsLoopbackWithoutTLS(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_TLS_CERT_FILE", "")
	t.Setenv("OLLAMA_AGENT_TLS_KEY_FILE", "")
	t.Setenv("OLLAMA_AGENT_REQUIRE_MTLS", "")

	for _, address := range []net.Addr{
		&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 11434},
		&net.TCPAddr{IP: net.ParseIP("::1"), Port: 11434},
	} {
		secure, err := configureAgentTLS(&http.Server{}, address)
		if err != nil || secure {
			t.Fatalf("configureAgentTLS(%s) = (%v, %v), want (false, nil)", address, secure, err)
		}
	}
}
