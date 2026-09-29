package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildArtifactManifestRejectsSymlinkOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("not an artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(workspace, "escape.txt")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("creating symlinks requires Developer Mode or elevation on Windows: %v", err)
		}
		t.Fatal(err)
	}
	if _, err := BuildArtifactManifest(workspace, "mission", "step_1", "escape.txt", "escape.txt"); err == nil {
		t.Fatal("expected symlink artifact rejection")
	}
}

func TestBuildArtifactManifestAcceptsRegularFile(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "result.txt")
	if err := os.WriteFile(path, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildArtifactManifest(workspace, "mission", "step_1", "result.txt", "result.txt")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Path != "result.txt" || manifest.Size != 4 || manifest.SHA256 == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
}
