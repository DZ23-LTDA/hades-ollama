package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func signWhatsAppBody(body, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func setupWhatsAppTestServer(t *testing.T) (*gin.Engine, *agent.Runtime, *agent.WhatsAppGateway) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	gw := agent.NewWhatsAppGateway(agent.WhatsAppGatewayConfig{
		PrimaryBackend: agent.WhatsAppBackendCloudAPI,
		CloudAPI: agent.CloudAPIConfig{
			PhoneNumberID: "phone-test-001",
			VerifyToken:   "my_secret_token",
			AppSecret:     "app-secret-test",
		},
		Allowlist: []agent.WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: agent.ContactRoleOwner, Allowed: true},
		},
	}, runtime)
	runtime.SetWhatsApp(gw)

	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}

	group := r.Group("/api/agent/v1")
	group.POST("/whatsapp/webhook", api.whatsappWebhook)
	group.GET("/whatsapp/webhook", api.whatsappWebhookVerify)
	group.GET("/whatsapp/status", api.whatsappStatus)
	group.POST("/whatsapp/send", api.whatsappSend)
	group.GET("/whatsapp/dlq", api.whatsappDLQ)
	group.DELETE("/whatsapp/dlq", api.whatsappClearDLQ)
	group.GET("/whatsapp/allowlist", api.whatsappAllowlist)
	group.POST("/whatsapp/allowlist", api.whatsappSetContactPolicy)
	group.DELETE("/whatsapp/allowlist/:phone", api.whatsappRemoveContactPolicy)
	group.POST("/whatsapp/config", api.whatsappConfig)

	return r, runtime, gw
}

func TestWhatsAppRoutesWebhookVerify(t *testing.T) {
	srv, _, _ := setupWhatsAppTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=my_secret_token&hub.challenge=test_challenge_1234", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for challenge verify, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "test_challenge_1234" {
		t.Fatalf("expected challenge echo, got %s", w.Body.String())
	}
}

func TestWhatsAppRoutesWebhookRejectsInvalidSignatureAndIdentity(t *testing.T) {
	srv, _, _ := setupWhatsAppTestServer(t)
	body := `{"object":"whatsapp","entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"phone-test-001"}}}]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/whatsapp/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-Hub-Signature-256", signWhatsAppBody(body, "wrong-secret"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid HMAC must return 401, got %d: %s", w.Code, w.Body.String())
	}

	badObject := `{"object":"wrong","entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"phone-test-001"}}}]}]}`
	req = httptest.NewRequest(http.MethodPost, "/api/agent/v1/whatsapp/webhook", bytes.NewBufferString(badObject))
	req.Header.Set("X-Hub-Signature-256", signWhatsAppBody(badObject, "app-secret-test"))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("wrong object must return 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWhatsAppRoutesStatus(t *testing.T) {
	srv, _, _ := setupWhatsAppTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/whatsapp/status", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal status response: %v", err)
	}

	if res["gate_status"] != "NOT_CONFIGURED" {
		t.Fatalf("expected unconfigured gateway to have gate_status NOT_CONFIGURED, got %v", res["gate_status"])
	}
}

func TestWhatsAppRoutesAllowlistCRUD(t *testing.T) {
	srv, _, _ := setupWhatsAppTestServer(t)

	// 1. Get initial allowlist
	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/whatsapp/allowlist", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get allowlist failed: %d", w.Code)
	}

	// 2. Add new contact policy
	newContact := agent.WhatsAppContactPolicy{
		PhoneNumber: "5511777777777",
		Name:        "New Tester",
		Role:        agent.ContactRoleOperator,
		Allowed:     true,
	}
	body, _ := json.Marshal(newContact)

	postReq := httptest.NewRequest(http.MethodPost, "/api/agent/v1/whatsapp/allowlist", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	srv.ServeHTTP(wPost, postReq)

	if wPost.Code != http.StatusOK {
		t.Fatalf("post contact policy failed: %d: %s", wPost.Code, wPost.Body.String())
	}

	// 3. Remove contact policy
	delReq := httptest.NewRequest(http.MethodDelete, "/api/agent/v1/whatsapp/allowlist/5511777777777", nil)
	wDel := httptest.NewRecorder()
	srv.ServeHTTP(wDel, delReq)

	if wDel.Code != http.StatusOK {
		t.Fatalf("delete contact policy failed: %d: %s", wDel.Code, wDel.Body.String())
	}
}

func TestWhatsAppRoutesSendWithoutCredentials(t *testing.T) {
	srv, _, _ := setupWhatsAppTestServer(t)

	sendBody := map[string]string{
		"to":   "5511999999999",
		"text": "teste de envio",
	}
	b, _ := json.Marshal(sendBody)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/whatsapp/send", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Expect StatusFailedDependency (424) because adapter is NOT_CONFIGURED
	if w.Code != http.StatusFailedDependency {
		t.Fatalf("expected 424 Failed Dependency when sending without credentials, got %d: %s", w.Code, w.Body.String())
	}
}
