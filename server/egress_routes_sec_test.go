package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestEgressLogsOrganizationScoped proves SEC-10(a) at the HTTP boundary: the
// egress-logs endpoint only returns records owned by the requesting
// organization and never leaks another tenant's destinations.
func TestEgressLogsOrganizationScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := &agentAPI{authRequired: true}
	group := router.Group("/api/agent/v1")
	// In production these routes sit behind authMiddleware, which derives the
	// organization from the verified bearer token and stores it in the context;
	// agentOrganizationID no longer trusts a client-supplied header (that was a
	// tenant-spoofing vector). Inject the org into the context the same way here
	// so the test exercises the real scoping path.
	group.Use(func(c *gin.Context) {
		if org := strings.TrimSpace(c.GetHeader("X-Ollama-Organization")); org != "" {
			c.Set("agent.organization", agent.Organization{ID: org})
		}
		c.Next()
	})
	{
		group.GET("/egress/logs", api.getEgressLogs)
		group.GET("/egress/status", api.getEgressStatus)
	}

	agent.DefaultEgressAuditStore.Clear()
	t.Cleanup(agent.DefaultEgressAuditStore.Clear)
	agent.DefaultEgressAuditStore.Record(agent.EgressDecision{OrganizationID: "org-a", Callsite: "connectors", Destination: "https://a.example.com", Host: "a.example.com", Allowed: true})
	agent.DefaultEgressAuditStore.Record(agent.EgressDecision{OrganizationID: "org-b", Callsite: "connectors", Destination: "https://b.example.com", Host: "b.example.com", Allowed: false})

	// org-a request sees only its single record.
	recLogs := httptest.NewRecorder()
	reqLogs, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/egress/logs", nil)
	reqLogs.Header.Set("X-Ollama-Organization", "org-a")
	router.ServeHTTP(recLogs, reqLogs)

	if recLogs.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recLogs.Code, recLogs.Body.String())
	}
	body := recLogs.Body.String()
	if strings.Contains(body, "b.example.com") {
		t.Fatalf("org-a response leaked org-b destination: %s", body)
	}
	var logsResp map[string]any
	if err := json.Unmarshal([]byte(body), &logsResp); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if count, ok := logsResp["count"].(float64); !ok || count != 1 {
		t.Fatalf("expected count 1 for org-a, got %v", logsResp["count"])
	}

	// org-a status only counts its own decisions (1 allowed, 0 blocked).
	recStatus := httptest.NewRecorder()
	reqStatus, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/egress/status", nil)
	reqStatus.Header.Set("X-Ollama-Organization", "org-a")
	router.ServeHTTP(recStatus, reqStatus)

	var statusResp map[string]any
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if total, ok := statusResp["total_decisions"].(float64); !ok || total != 1 {
		t.Fatalf("expected total_decisions 1 for org-a, got %v", statusResp["total_decisions"])
	}
	if blocked, ok := statusResp["blocked_count"].(float64); !ok || blocked != 0 {
		t.Fatalf("expected blocked_count 0 for org-a, got %v", statusResp["blocked_count"])
	}
}
