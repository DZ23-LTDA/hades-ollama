package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestEgressRoutesAuditAndStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := &agentAPI{}
	group := router.Group("/api/agent/v1")
	{
		group.GET("/egress/logs", api.getEgressLogs)
		group.GET("/egress/status", api.getEgressStatus)
	}

	// Seed some audit decisions
	agent.DefaultEgressAuditStore.Clear()
	agent.DefaultEgressAuditStore.Record(agent.EgressDecision{
		Callsite:    "connectors",
		Destination: "https://api.github.com",
		Host:        "api.github.com",
		Allowed:     true,
		Reason:      "approved public destination",
	})
	agent.DefaultEgressAuditStore.Record(agent.EgressDecision{
		Callsite:    "whatsapp",
		Destination: "http://169.254.169.254",
		Host:        "169.254.169.254",
		Allowed:     false,
		Reason:      "blocked cloud metadata service",
	})

	// Test GET /egress/status
	recStatus := httptest.NewRecorder()
	reqStatus, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/egress/status", nil)
	router.ServeHTTP(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp map[string]any
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if statusResp["status"] != "PASS" {
		t.Errorf("expected status PASS, got %v", statusResp["status"])
	}
	if allowed, ok := statusResp["allowed_count"].(float64); !ok || allowed != 1 {
		t.Errorf("expected allowed_count 1, got %v", statusResp["allowed_count"])
	}
	if blocked, ok := statusResp["blocked_count"].(float64); !ok || blocked != 1 {
		t.Errorf("expected blocked_count 1, got %v", statusResp["blocked_count"])
	}

	// Test GET /egress/logs
	recLogs := httptest.NewRecorder()
	reqLogs, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/egress/logs", nil)
	router.ServeHTTP(recLogs, reqLogs)

	if recLogs.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recLogs.Code, recLogs.Body.String())
	}
	var logsResp map[string]any
	if err := json.Unmarshal(recLogs.Body.Bytes(), &logsResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if count, ok := logsResp["count"].(float64); !ok || count != 2 {
		t.Errorf("expected count 2, got %v", logsResp["count"])
	}

	// Test GET /egress/logs?callsite=whatsapp
	recFiltered := httptest.NewRecorder()
	reqFiltered, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/egress/logs?callsite=whatsapp", nil)
	router.ServeHTTP(recFiltered, reqFiltered)

	if recFiltered.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recFiltered.Code, recFiltered.Body.String())
	}
	var filteredResp map[string]any
	if err := json.Unmarshal(recFiltered.Body.Bytes(), &filteredResp); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if count, ok := filteredResp["count"].(float64); !ok || count != 1 {
		t.Errorf("expected count 1 for whatsapp filter, got %v", filteredResp["count"])
	}
}
