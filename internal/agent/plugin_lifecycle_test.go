package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConnectorLifecycleDisablesCalls(t *testing.T) {
	manager := NewConnectorManager()
	if err := manager.RegisterForOrganization("org_test", ConnectorConfig{ID: "c", Provider: "test", BaseURL: "https://example.test", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabledForOrganization("org_test", "c", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org_test", "c", "read", "GET", "/", nil); !errors.Is(err, ErrConnectorDisabled) {
		t.Fatalf("expected disabled connector, got %v", err)
	}
	if err := manager.SetEnabledForOrganization("org_test", "c", true); err != nil {
		t.Fatal(err)
	}
	if err := manager.RemoveForOrganization("org_test", "c"); err != nil {
		t.Fatal(err)
	}
}

func TestMCPAndSkillsLifecycle(t *testing.T) {
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "echo", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}}); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabled("echo", false); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.CallGlobal(context.Background(), "echo", "echo", map[string]any{}); err == nil {
		t.Fatal("expected disabled MCP error")
	}
	if err := manager.Remove("echo"); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skill.json"), []byte(`{"id":"skill","version":"1","description":"test","trusted":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LoadSkills(dir, true); err != nil {
		t.Fatal(err)
	}
	if len(store.Skills()) != 1 || !store.Skills()[0].Enabled || store.Skills()[0].Trusted {
		t.Fatalf("skills=%+v", store.Skills())
	}
	if err := store.SetSkillEnabled("skill", false); err != nil {
		t.Fatal(err)
	}
	if store.Skills()[0].Enabled {
		t.Fatal("skill should be disabled")
	}
	if err := store.RemoveSkill("skill"); err != nil {
		t.Fatal(err)
	}
}

func TestRawPluginLifecycleCannotMutateTenantOwnedResources(t *testing.T) {
	connector := NewConnectorManager()
	if err := connector.RegisterForOrganization("org_owner", ConnectorConfig{ID: "tenant-connector", Provider: "test", BaseURL: "https://example.test", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	if err := connector.SetEnabled("tenant-connector", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw connector disable error = %v", err)
	}
	if err := connector.Remove("tenant-connector"); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw connector remove error = %v", err)
	}
	if items := connector.ListForOrganization("org_owner"); len(items) != 1 || items[0].Disabled {
		t.Fatalf("tenant connector was changed through local API: %+v", items)
	}

	mcp := NewMCPManager()
	if err := mcp.RegisterForOrganization("org_owner", MCPServerConfig{ID: "tenant-mcp", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}}); err != nil {
		t.Fatal(err)
	}
	if err := mcp.SetEnabled("tenant-mcp", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw MCP disable error = %v", err)
	}
	if err := mcp.Remove("tenant-mcp"); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw MCP remove error = %v", err)
	}
	if items := mcp.ListForOrganization("org_owner"); len(items) != 1 || items[0].Disabled {
		t.Fatalf("tenant MCP was changed through local API: %+v", items)
	}

	remote := NewRemoteMCPManager()
	if err := remote.RegisterForOrganization("org_owner", RemoteMCPServerConfig{ID: "tenant-remote", URL: "https://example.test/mcp", AllowedMethods: []string{"tools/call"}}); err != nil {
		t.Fatal(err)
	}
	if err := remote.SetEnabled("tenant-remote", false); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw remote MCP disable error = %v", err)
	}
	if err := remote.Remove("tenant-remote"); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("raw remote MCP remove error = %v", err)
	}
	if items := remote.ListForOrganization("org_owner"); len(items) != 1 || items[0].Disabled {
		t.Fatalf("tenant remote MCP was changed through local API: %+v", items)
	}
}
