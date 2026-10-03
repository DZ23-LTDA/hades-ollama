package server

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

func TestDownloadBackupStreamsVerifiableArchive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "context"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "context", "note.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A secret-bearing dir that must not be in the backup.
	if err := os.MkdirAll(filepath.Join(dataRoot, "auth"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "auth", "cred.json"), []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: dataRoot})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&agentAPI{runtime: runtime}).downloadBackup(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d", recorder.Code)
	}
	if recorder.Header().Get("X-Backup-SHA256") == "" {
		t.Fatal("missing X-Backup-SHA256 header")
	}

	// The body must be a valid gzip-tar that contains the context file and not
	// the excluded auth secret.
	gz, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatalf("response is not valid gzip: %v", err)
	}
	tr := tar.NewReader(gz)
	names := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names[filepath.ToSlash(hdr.Name)] = true
	}
	if !names["context/note.txt"] {
		t.Fatalf("backup missing context/note.txt; got %v", names)
	}
	for name := range names {
		if len(name) >= 5 && name[:5] == "auth/" {
			t.Fatalf("backup leaked excluded auth dir: %s", name)
		}
	}
}

func TestDownloadBackupRefusedInMultiTenantMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dataRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataRoot, "context"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "context", "note.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: dataRoot})
	if err != nil {
		t.Fatal(err)
	}

	// The full backup covers the whole (multi-tenant) data root, so under auth it
	// must be refused rather than letting any member export every org's data.
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&agentAPI{runtime: runtime, authRequired: true}).downloadBackup(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 in multi-tenant mode, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() > 0 && recorder.Header().Get("X-Backup-SHA256") != "" {
		t.Fatal("multi-tenant backup must not stream archive bytes")
	}
}
