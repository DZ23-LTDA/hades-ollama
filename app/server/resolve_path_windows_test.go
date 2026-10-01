//go:build windows

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathPrefersBundledExe(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bundled := filepath.Join(filepath.Dir(exe), "ollama.exe")
	if _, err := os.Stat(bundled); err == nil {
		t.Skip("a real ollama.exe already sits next to the test binary")
	}
	if err := os.WriteFile(bundled, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(bundled) })

	// A decoy on PATH must not win over the binary shipped with the app.
	decoyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(decoyDir, "ollama.exe"), []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", decoyDir)

	if got := resolvePath("ollama"); got != bundled {
		t.Fatalf("resolvePath(ollama) = %q, want bundled %q", got, bundled)
	}
}
