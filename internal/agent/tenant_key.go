package agent

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// TenantContextKeyVersionFromEnv reads and validates
// OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION. It returns 1 when unset and rejects
// anything outside the inclusive range 1..999999999. This is the single source
// of truth shared by the CLI and the server, which previously duplicated it.
func TenantContextKeyVersionFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION"))
	if raw == "" {
		return 1, nil
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version < 1 || version > 999999999 {
		return 0, errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION must be an integer between 1 and 999999999")
	}
	return version, nil
}
