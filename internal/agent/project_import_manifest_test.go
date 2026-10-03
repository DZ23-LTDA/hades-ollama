package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestImportedFileManifestReportsPerFileStatus asserts that an imported project
// exposes, for every file, whether it was indexed and — when skipped — why, so
// a user is never left guessing about a document that was silently ignored.
func TestImportedFileManifestReportsPerFileStatus(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string, content []byte) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("notes.txt", []byte("conteúdo de texto"))
	mustWrite("docs/readme.md", []byte("# titulo"))
	mustWrite("diagram.png", []byte("binary-not-indexable"))
	mustWrite("huge.txt", bytes.Repeat([]byte("a"), maxIndexedImportFileBytes+1))
	mustWrite(".git/config", []byte("[core]")) // control plane must never be listed

	manifest := importedFileManifest(root)
	byPath := map[string]ProjectImportFile{}
	for _, file := range manifest {
		byPath[file.Path] = file
	}

	if _, listed := byPath[".git/config"]; listed {
		t.Fatalf("manifest leaked the .git control plane: %+v", manifest)
	}

	indexed := []string{"notes.txt", "docs/readme.md"}
	for _, path := range indexed {
		file, ok := byPath[path]
		if !ok {
			t.Fatalf("expected %q in manifest: %+v", path, manifest)
		}
		if !file.Indexed || file.Reason != "" {
			t.Fatalf("%q should be indexed without a reason, got %+v", path, file)
		}
	}

	png, ok := byPath["diagram.png"]
	if !ok || png.Indexed || png.Reason == "" {
		t.Fatalf("diagram.png must be reported as skipped with a reason, got %+v (ok=%v)", png, ok)
	}

	huge, ok := byPath["huge.txt"]
	if !ok || huge.Indexed || huge.Reason == "" {
		t.Fatalf("huge.txt must be reported as skipped (over size limit), got %+v (ok=%v)", huge, ok)
	}
	if huge.SizeBytes <= maxIndexedImportFileBytes {
		t.Fatalf("huge.txt size should exceed the index limit, got %d", huge.SizeBytes)
	}

	// importedTextFiles must agree with the manifest's indexed subset.
	if got := len(importedTextFiles(root)); got != len(indexed) {
		t.Fatalf("importedTextFiles returned %d indexed files, want %d", got, len(indexed))
	}
}
