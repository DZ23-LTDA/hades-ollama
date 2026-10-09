package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func attachmentRequest(t *testing.T, files map[string]string, organizationID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("name", "Documentos da missão"); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/projects/import/attachments", &body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	return ctx, recorder
}

func TestImportMissionAttachmentsCreatesIndexedOrgProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: workspace, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}
	ctx, recorder := attachmentRequest(t, map[string]string{"notes.txt": "A política prevê 30 dias de férias."}, "org-attachments")

	api.importMissionAttachments(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var result agent.ProjectImportResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Source != "attachments" || result.Project.ID == "" || result.Project.OrganizationID != "org-attachments" {
		t.Fatalf("unexpected imported project: %+v", result)
	}
	if result.IndexedFiles != 1 || result.IndexedMemory < 1 {
		t.Fatalf("attachments were not indexed: %+v", result)
	}
	if result.State != "IMPORTED_INDEXED" {
		t.Fatalf("state=%q, want IMPORTED_INDEXED", result.State)
	}
}

func TestImportMissionAttachmentsRejectsUnsupportedFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}
	ctx, recorder := attachmentRequest(t, map[string]string{"photo.png": "not an OCR document"}, agent.LocalOrganizationID)

	api.importMissionAttachments(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", recorder.Code, recorder.Body.String())
	}
	if len(runtime.Context().ListProjects()) != 0 {
		t.Fatal("unsupported attachment must not create a project")
	}
}

func TestValidateAutoRunCapabilitiesIsReadOnly(t *testing.T) {
	if err := validateAutoRunCapabilities([]string{"workspace:read"}); err != nil {
		t.Fatalf("read-only autorun rejected: %v", err)
	}
	if err := validateAutoRunCapabilities(nil); err != nil {
		t.Fatalf("implicit read-only autorun rejected: %v", err)
	}
	for _, capabilities := range [][]string{{"workspace:read", "workspace:write"}, {"browser:navigate"}, {"terminal:allowlisted"}} {
		if err := validateAutoRunCapabilities(capabilities); err == nil {
			t.Fatalf("elevated autorun capabilities accepted: %v", capabilities)
		}
	}
}
