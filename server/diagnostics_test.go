package server

import (
	"encoding/json"
	"strings"
	"testing"
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
