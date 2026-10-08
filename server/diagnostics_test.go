package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildDiagnosticReportReportsBooleansWithoutLeakingValues(t *testing.T) {
	const secret = "postgres://user:SUPER-SECRET-PASSWORD@db.internal/agent"
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", secret)
	t.Setenv("OLLAMA_AGENT_REDIS_URL", "")

	report := buildDiagnosticReport()

	integrations, ok := report["integrations"].(map[string]bool)
	if !ok {
		t.Fatalf("integrations missing or wrong type: %T", report["integrations"])
	}
	if !integrations["agent_database_url"] {
		t.Fatal("configured database should report true")
	}
	if integrations["agent_redis_url"] {
		t.Fatal("unset redis should report false")
	}
	if report["product"] != "Hades" {
		t.Fatalf("product = %v, want Hades", report["product"])
	}

	// Critically: the secret value must never appear anywhere in the export.
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SUPER-SECRET-PASSWORD") || strings.Contains(string(data), secret) {
		t.Fatalf("diagnostic report leaked a secret value: %s", data)
	}
}

func TestDiagnosticsEndpointReturnsReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", "postgres://u:SECRETPW@db/x")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&agentAPI{}).diagnostics(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "\"product\":\"Hades\"") {
		t.Fatalf("missing product: %s", body)
	}
	if strings.Contains(body, "SECRETPW") {
		t.Fatalf("diagnostics endpoint leaked a secret: %s", body)
	}
}
