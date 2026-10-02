package server

import (
	"runtime"
	"strings"

	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/version"
)

// diagnosticIntegrationVars are the integration environment variables reported
// in a diagnostic export — as configured/not booleans, never their values.
var diagnosticIntegrationVars = []string{
	"OLLAMA_AGENT_DATABASE_URL",
	"OLLAMA_AGENT_REDIS_URL",
	"OLLAMA_AGENT_OTLP_ENDPOINT",
	"OLLAMA_AGENT_PUSH_ENDPOINT",
	"OLLAMA_AGENT_CONNECTORS",
	"OLLAMA_AGENT_MCP",
	"OLLAMA_AGENT_REMOTE_MCP",
	"OLLAMA_AGENT_MEDIA_BASE_URL",
	"OLLAMA_AGENT_DEPLOYMENTS",
	"OLLAMA_AGENT_EMBED_MODEL",
	"OLLAMA_DZ23_CONFIG",
}

// buildDiagnosticReport assembles a support/troubleshooting snapshot that is
// safe to share. It reports WHICH integrations are configured as booleans
// (never their values) plus build and platform info, and runs free-text fields
// through DLP redaction so secrets can never leak into an exported report (A7).
func buildDiagnosticReport() map[string]any {
	integrations := make(map[string]bool, len(diagnosticIntegrationVars))
	for _, name := range diagnosticIntegrationVars {
		key := strings.ToLower(strings.TrimPrefix(name, "OLLAMA_"))
		integrations[key] = envConfigured(name)
	}
	return map[string]any{
		"product":      "Hades",
		"version":      agent.RedactDLP(version.Version),
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"go":           agent.RedactDLP(runtime.Version()),
		"integrations": integrations,
	}
}
