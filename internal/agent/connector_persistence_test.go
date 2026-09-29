package agent

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func persistentConnectorConfig() ConnectorConfig {
	return ConnectorConfig{
		ID:             "github-org-a",
		OrganizationID: "org-a",
		Provider:       "GitHub",
		BaseURL:        "https://api.github.example",
		TokenEnv:       "OLLAMA_GITHUB_TOKEN",
		OAuthProvider:  "github",
		Operations: []ConnectorOperation{{
			Name:         "profile",
			Methods:      []string{"get"},
			PathPrefixes: []string{"/user"},
		}},
	}
}

func TestPersistentConnectorManagerRoundTripsLifecycleWithoutCredentialValues(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "nested", "connectors.json")
	manager, err := NewPersistentConnectorManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	config := persistentConnectorConfig()
	if err := manager.Register(config); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabledForOrganization("org-a", config.ID, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "configured-token") || !strings.Contains(string(data), "OLLAMA_GITHUB_TOKEN") {
		t.Fatalf("manifest leaked or omitted credential reference: %s", data)
	}
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o600 {
		t.Fatalf("manifest permissions=%o, want 600", got)
	}

	reloaded, err := NewPersistentConnectorManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	listed := reloaded.ListForOrganization("org-a")
	// CredentialConfigured is expected to be false because the environment
	// variable is intentionally absent in this test and TokenEnv is redacted.
	if len(listed) != 1 || !listed[0].Disabled || listed[0].TokenEnv != "" || listed[0].CredentialConfigured {
		t.Fatalf("reloaded connector=%+v", listed)
	}
	if err := reloaded.RemoveForOrganization("org-a", config.ID); err != nil {
		t.Fatal(err)
	}
	final, err := NewPersistentConnectorManager(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := final.ListForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("connector survived removal: %+v", got)
	}
}

func TestRegisterForOrganizationBindsAndRejectsConflictingTenant(t *testing.T) {
	manager := NewConnectorManager()
	config := persistentConnectorConfig()
	config.OrganizationID = ""
	if err := manager.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if got := manager.ListForOrganization("org-a"); len(got) != 1 {
		t.Fatalf("org-a connectors=%+v", got)
	}
	if got := manager.ListForOrganization("org-b"); len(got) != 0 {
		t.Fatalf("cross-tenant connector visible=%+v", got)
	}
	conflictingID := persistentConnectorConfig()
	conflictingID.OrganizationID = ""
	if err := manager.RegisterForOrganization("org-b", conflictingID); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("same-id cross-tenant registration error=%v", err)
	}
	config.ID = "github-org-b"
	config.OrganizationID = "org-b"
	if err := manager.RegisterForOrganization("org-a", config); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("conflicting organization error=%v", err)
	}
}

func TestRegisterRejectsEmptyOperationMethod(t *testing.T) {
	manager := NewConnectorManager()
	config := persistentConnectorConfig()
	config.Operations[0].Methods = []string{""}
	if err := manager.Register(config); err == nil || !strings.Contains(err.Error(), "methods") {
		t.Fatalf("expected empty method rejection, got %v", err)
	}
}

func TestPersistentConnectorManagerRejectsTrailingJSON(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "connectors.json")
	if err := os.WriteFile(manifest, []byte("[] {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentConnectorManager(manifest); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}
