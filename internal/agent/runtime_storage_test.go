package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuntimeDataRootRejectsNestedPaths(t *testing.T) {
	workspace := t.TempDir()
	if _, err := resolveRuntimeDataRoot(workspace, filepath.Join(workspace, "data")); !errors.Is(err, errRuntimeDataRootNested) {
		t.Fatalf("err = %v, want nested data root rejection", err)
	}
}

func TestResolveRuntimeDataRootMigratesLegacyDirectories(t *testing.T) {
	workspace := t.TempDir()
	dataRoot := t.TempDir()
	legacy := filepath.Join(workspace, ".agent-context")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(legacy, "marker")
	if err := os.WriteFile(marker, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := resolveRuntimeDataRoot(workspace, dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != dataRoot {
		t.Fatalf("resolved data root = %q, want %q", resolved, dataRoot)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy directory still exists: %v", err)
	}
	migrated := filepath.Join(dataRoot, ".agent-context", "marker")
	content, err := os.ReadFile(migrated)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "legacy" {
		t.Fatalf("migrated content = %q", content)
	}
}

func TestRuntimeStoresAreOutsideWorkspaceByDefault(t *testing.T) {
	workspace := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: workspace, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if sameOrWithin(runtime.dataRoot, runtime.workspaceRoot) || sameOrWithin(runtime.workspaceRoot, runtime.dataRoot) {
		t.Fatalf("data root %q is not separate from workspace %q", runtime.dataRoot, runtime.workspaceRoot)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".agent-context")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runtime data leaked into workspace: %v", err)
	}
}
