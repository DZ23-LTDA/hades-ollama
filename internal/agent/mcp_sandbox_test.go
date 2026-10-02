package agent

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestMCPOrganizationCallFailsClosedWithoutStrictSandbox(t *testing.T) {
	manager := NewMCPManager()
	if err := manager.RegisterForOrganization("org_a", MCPServerConfig{
		ID: "tenant-mcp", Command: os.Args[0],
		Args:           []string{"-test.run=TestMCPHelperProcess"},
		AllowedMethods: []string{"echo"},
	}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	_, err := manager.CallForOrganization(context.Background(), "org_a", "tenant-mcp", "echo", map[string]any{"value": "must-not-run"})
	if !errors.Is(err, ErrMCPStrictSandboxNotConfigured) {
		t.Fatalf("tenant MCP call error = %v, want strict sandbox fail-closed", err)
	}
}
