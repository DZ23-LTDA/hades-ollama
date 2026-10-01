package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginManagersEnforceOrganizationOwnership(t *testing.T) {
	connector := NewConnectorManager()
	for _, organizationID := range []string{"org-a", "org-b"} {
		if err := connector.Register(ConnectorConfig{ID: "connector-" + organizationID, OrganizationID: organizationID, Provider: "test", BaseURL: "https://api.example.test", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/v1"}}}}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(connector.ListForOrganization("org-a")); got != 1 {
		t.Fatalf("connector org-a count=%d", got)
	}
	if err := connector.SetEnabledForOrganization("org-a", "connector-org-b", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-tenant connector mutation err=%v", err)
	}

	mcp := NewMCPManager()
	command := os.Args[0]
	for _, organizationID := range []string{"org-a", "org-b"} {
		if err := mcp.Register(MCPServerConfig{ID: "mcp-" + organizationID, OrganizationID: organizationID, Command: command, Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(mcp.ListForOrganization("org-a")); got != 1 {
		t.Fatalf("MCP org-a count=%d", got)
	}
	if err := mcp.RemoveForOrganization("org-a", "mcp-org-b"); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-tenant MCP mutation err=%v", err)
	}
	mcp.StopAll()

	remote := NewRemoteMCPManager()
	for _, organizationID := range []string{"org-a", "org-b"} {
		if err := remote.Register(RemoteMCPServerConfig{ID: "remote-" + organizationID, OrganizationID: organizationID, URL: "https://mcp.example.test", AllowedMethods: []string{"echo"}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := remote.SetEnabledForOrganization("org-a", "remote-org-b", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-tenant remote MCP mutation err=%v", err)
	}

	contextStore, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), []byte(`{"id":"skill","version":"1","description":"test"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := contextStore.LoadSkillsForOrganization(skillDir, "org-a"); err != nil {
		t.Fatal(err)
	}
	if err := contextStore.RegisterSkill(SkillManifest{ID: "skill-local-only", Version: "1", Description: "local manifest"}); err != nil {
		t.Fatal(err)
	}
	if got := len(contextStore.SkillsForOrganization("org-b")); got != 0 {
		t.Fatalf("skill leaked to org-b: %d", got)
	}
	if got := contextStore.SkillsForOrganization("org-a"); len(got) != 1 || got[0].ID != "skill" {
		t.Fatalf("org-a skills include an ownerless manifest or omit its own skill: %+v", got)
	}
	if got := contextStore.SkillsForOrganization(LocalOrganizationID); len(got) != 1 || got[0].ID != "skill-local-only" {
		t.Fatalf("local skills should include only the local ownerless manifest: %+v", got)
	}
	if err := contextStore.SetSkillEnabledForOrganization("org-b", "skill", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-tenant skill mutation err=%v", err)
	}
}
