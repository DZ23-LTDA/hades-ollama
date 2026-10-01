package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPersistentMCPManagerRoundTripAndTenantCollision(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "mcp.json")
	command := filepath.Join(root, "mcp-server")
	if runtime.GOOS == "windows" {
		// Windows identifies executables by extension, not by mode bits.
		command += ".exe"
	}
	if err := os.WriteFile(command, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	manager, err := NewPersistentMCPManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	config := MCPServerConfig{
		ID:              "local-tools",
		OrganizationID:  "org-a",
		Command:         command,
		Args:            []string{"-c", "cat"},
		AllowedMethods:  []string{"tools/list"},
		EnvironmentVars: []string{"MCP_TEST_TOKEN"},
	}
	if err := manager.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if err := manager.RegisterForOrganization("org-b", config); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("expected cross-tenant collision rejection, got %v", err)
	}
	var saved []MCPServerConfig
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0].WorkingDirectory != "" {
		t.Fatalf("temporary workspace must not be persisted: %#v", saved)
	}
	if mode := fileMode(t, manifest); runtime.GOOS != "windows" && mode.Perm() != 0o600 {
		t.Fatalf("manifest permissions = %o, want 600", mode.Perm())
	}
	if err := manager.SetEnabledForOrganization("org-a", "local-tools", false); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewPersistentMCPManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	loaded := reloaded.ListForOrganization("org-a")
	if len(loaded) != 1 || !loaded[0].Disabled || loaded[0].OrganizationID != "org-a" {
		t.Fatalf("unexpected reloaded MCP config: %#v", loaded)
	}
	if err := reloaded.RemoveForOrganization("org-a", "local-tools"); err != nil {
		t.Fatal(err)
	}
	if got := len(reloaded.List()); got != 0 {
		t.Fatalf("MCP list after remove = %d, want 0", got)
	}
}

func TestPersistentRemoteMCPManagerRoundTripAndTenantCollision(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "remote-mcp.json")
	manager, err := NewPersistentRemoteMCPManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	config := RemoteMCPServerConfig{
		ID:             "remote-tools",
		OrganizationID: "org-a",
		URL:            "https://example.com/mcp",
		TokenEnv:       "REMOTE_MCP_TOKEN",
		HeadersEnv:     map[string]string{"X-Org": "REMOTE_MCP_ORG"},
		AllowedMethods: []string{"tools/list"},
	}
	if err := manager.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if err := manager.RegisterForOrganization("org-b", config); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("expected cross-tenant collision rejection, got %v", err)
	}
	if err := manager.SetEnabledForOrganization("org-a", "remote-tools", false); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewPersistentRemoteMCPManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	reloaded.mu.RLock()
	persisted := reloaded.servers["remote-tools"]
	reloaded.mu.RUnlock()
	if persisted.TokenEnv != "REMOTE_MCP_TOKEN" || persisted.HeadersEnv["X-Org"] != "REMOTE_MCP_ORG" || persisted.URL != "https://example.com/mcp" {
		t.Fatalf("private persisted configuration did not round-trip: %#v", persisted)
	}
	loaded := reloaded.ListForOrganization("org-a")
	if len(loaded) != 1 || !loaded[0].Disabled || loaded[0].TokenEnv != "" || loaded[0].URL != "https://example.com" {
		t.Fatalf("unexpected reloaded Remote MCP config: %#v", loaded)
	}
	if len(loaded[0].HeadersEnv) != 0 {
		t.Fatalf("tenant-facing config exposed header env mappings: %#v", loaded[0].HeadersEnv)
	}
}

func TestPersistentSkillManifestRoundTripIsNeverTrusted(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-a", SkillManifest{
		ID:             "research-skill",
		OrganizationID: "org-a",
		Version:        "1.0.0",
		Description:    "Research tools",
		Scopes:         []string{"research:read"},
		Tools:          []string{"search"},
		Trusted:        true,
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "skills", "research-skill.json")
	if mode := fileMode(t, path); runtime.GOOS != "windows" && mode.Perm() != 0o600 {
		t.Fatalf("skill manifest permissions = %o, want 600", mode.Perm())
	}
	reloaded, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded := reloaded.SkillsForOrganization("org-a")
	if len(loaded) != 1 || loaded[0].Trusted || !loaded[0].Enabled {
		t.Fatalf("skill trust/enabled state = %#v", loaded)
	}
	if err := reloaded.SetSkillEnabledForOrganization("org-a", "research-skill", false); err != nil {
		t.Fatal(err)
	}
	secondReload, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	loaded = secondReload.SkillsForOrganization("org-a")
	if len(loaded) != 1 || loaded[0].Enabled || loaded[0].Trusted {
		t.Fatalf("disabled skill was not persisted fail-closed: %#v", loaded)
	}
	if err := secondReload.RemoveSkillForOrganization("org-a", "research-skill"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("skill manifest remains after remove: %v", err)
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode()
}

func TestPersistentMCPManagerRejectsTrailingJSON(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(manifest, []byte("[] {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentMCPManager(manifest); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}

func TestPersistentRemoteMCPManagerRejectsTrailingJSON(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "remote-mcp.json")
	if err := os.WriteFile(manifest, []byte("[] {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentRemoteMCPManager(manifest); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}

func TestLoadSkillsRejectsUnknownAndTrailingJSON(t *testing.T) {
	unknownDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(unknownDir, "unknown.json"), []byte(`{"id":"unknown","version":"1","unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LoadSkillsForOrganization(unknownDir, "org-a"); err == nil {
		t.Fatal("expected unknown skill field rejection")
	}

	trailingDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(trailingDir, "trailing.json"), []byte(`{} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.LoadSkillsForOrganization(trailingDir, "org-a"); err == nil {
		t.Fatal("expected trailing skill JSON rejection")
	}
}
