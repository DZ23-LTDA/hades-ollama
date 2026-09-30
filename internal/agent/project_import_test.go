package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func projectImportZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, body := range files {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func newProjectImporterFixture(t *testing.T) (*ProjectImporter, *UploadManager, string) {
	t.Helper()
	root := t.TempDir()
	contextStore, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(root); err != nil {
		t.Fatal(err)
	}
	uploads, err := NewUploadManager(filepath.Join(root, "uploads"), 1<<30, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	ingestion := DocumentIngestor{Context: contextStore, WorkspaceRoot: root}
	return NewProjectImporter(root, root, contextStore, ingestion), uploads, root
}

func TestGitHubImportRejectsUntrustedRepositoryURL(t *testing.T) {
	importer, _, _ := newProjectImporterFixture(t)
	for _, raw := range []string{"http://github.com/a/b", "https://evil.example/a/b", "https://github.com/a/b?token=secret", "https://github.com/a/b/c"} {
		if _, err := importer.ImportGitHub(context.Background(), LocalOrganizationID, ProjectImportRequest{URL: raw}); !errors.Is(err, ErrGitHubURLInvalid) {
			t.Fatalf("URL %q: expected ErrGitHubURLInvalid, got %v", raw, err)
		}
	}
}

func TestPrivateGitHubImportWithoutCredentialsIsNotConfigured(t *testing.T) {
	importer, _, _ := newProjectImporterFixture(t)
	importer.HTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("unauthorized")), Header: make(http.Header)}, nil
	})}
	if _, err := importer.ImportGitHub(context.Background(), LocalOrganizationID, ProjectImportRequest{URL: "https://github.com/example/private"}); !errors.Is(err, ErrGitHubAuthRequired) {
		t.Fatalf("expected private import to be NOT_CONFIGURED, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestGitHubImportUsesBoundedArchiveClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(projectImportZIP(t, map[string]string{"README.md": "imported\n"}))
	}))
	defer server.Close()
	importer, _, _ := newProjectImporterFixture(t)
	data := projectImportZIP(t, map[string]string{"README.md": "imported\n"})
	client := server.Client()
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != len(data) {
		t.Fatalf("fixture archive mismatch: got %d want %d", len(body), len(data))
	}
	if _, _, err := importer.downloadArchive(context.Background(), client, server.URL, ""); err != nil {
		t.Fatalf("bounded archive download: %v", err)
	}
}

func TestLargeUploadProjectImportStreamsZipAndCreatesIsolatedWorktree(t *testing.T) {
	importer, uploads, root := newProjectImporterFixture(t)
	archive := projectImportZIP(t, map[string]string{
		"README.md":   "project documentation\n",
		"src/main.go": "package main\nfunc main() {}\n",
		"run.sh":      "#!/bin/sh\nprintf unsafe\n",
	})
	project, err := importer.Context.CreateProject("incoming", root, LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	session, err := uploads.StartUpload(LocalOrganizationID, project.ID, "project.zip", int64(len(archive)), 7, "")
	if err != nil {
		t.Fatal(err)
	}
	for offset := 0; offset < len(archive); {
		end := offset + 7
		if end > len(archive) {
			end = len(archive)
		}
		if _, err := uploads.AppendChunk(LocalOrganizationID, session.ID, int64(offset), archive[offset:end]); err != nil {
			t.Fatal(err)
		}
		offset = end
	}
	session, err = uploads.FinalizeUpload(LocalOrganizationID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := importer.ImportUpload(context.Background(), LocalOrganizationID, session, "incoming")
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "IMPORTED_INDEXED" || result.IndexedFiles < 2 || result.IndexedMemory == 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if result.Branch == "" || !strings.HasPrefix(result.Branch, "agent/import_") {
		t.Fatalf("import did not create a dedicated branch: %+v", result)
	}
	if result.WorktreePath == project.Root || !strings.Contains(result.WorktreePath, ".agent-worktrees") {
		t.Fatalf("import did not isolate worktree: %q", result.WorktreePath)
	}
	if _, err := os.Stat(filepath.Join(result.WorktreePath, "README.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.WorktreePath, "run.sh")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectImportRejectsTraversalArchive(t *testing.T) {
	importer, _, _ := newProjectImporterFixture(t)
	archive := projectImportZIP(t, map[string]string{"../../escape.txt": "nope"})
	if _, err := importer.importArchive(context.Background(), LocalOrganizationID, "", "unsafe", bytes.NewReader(archive), int64(len(archive)), "hash", "zip", "", ""); !errors.Is(err, ErrImportArchiveUnsafe) {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}
