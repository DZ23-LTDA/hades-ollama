package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestStudioBuilderRoutesInteractiveAndTraceableExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	builder, err := agent.NewBuilderService(tmpDir)
	if err != nil {
		t.Fatalf("failed to create builder service: %v", err)
	}

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: tmpDir,
		Builder:       builder,
	})
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}

	api := &agentAPI{runtime: runtime}
	router := gin.New()
	group := router.Group("/api/agent/v1")
	{
		group.GET("/builders", api.builders)
		group.GET("/builders/:id", api.getBuilder)
		group.POST("/builders", api.createBuilder)
		group.POST("/builders/:id/visual", api.updateBuilderVisual)
		group.POST("/builders/:id/undo", api.undoBuilder)
		group.POST("/builders/:id/redo", api.redoBuilder)
		group.POST("/builders/:id/preview", api.previewBuilder)
		group.POST("/builders/:id/export", api.exportBuilder)
		group.GET("/builders/:id/download", api.downloadBuilder)
	}

	// 1. Create a project via POST /builders
	createBody := `{"name":"Studio Test Site","kind":"website"}`
	recCreate := httptest.NewRecorder()
	reqCreate, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders", strings.NewReader(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	var createdProject agent.BuilderProject
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createdProject); err != nil {
		t.Fatalf("failed to decode created project: %v", err)
	}
	projectID := createdProject.ID

	// 2. Fetch project via GET /builders/:id
	recGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/builders/"+projectID, nil)
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 3. Update components via POST /builders/:id/visual
	visualBody := `{
		"components": [
			{
				"id": "hero_text",
				"type": "heading",
				"props": {"text": "Studio Live Test", "level": "h1"},
				"x": 30,
				"y": 50,
				"width": 500,
				"height": 60
			}
		]
	}`
	recVisual := httptest.NewRecorder()
	reqVisual, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+projectID+"/visual", strings.NewReader(visualBody))
	reqVisual.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recVisual, reqVisual)

	if recVisual.Code != http.StatusOK {
		t.Fatalf("expected status 200 on visual update, got %d: %s", recVisual.Code, recVisual.Body.String())
	}
	var updatedProject agent.BuilderProject
	if err := json.Unmarshal(recVisual.Body.Bytes(), &updatedProject); err != nil {
		t.Fatalf("failed to decode updated project: %v", err)
	}
	if len(updatedProject.Components) != 1 || updatedProject.Components[0].Props["text"] != "Studio Live Test" {
		t.Errorf("unexpected components: %+v", updatedProject.Components)
	}

	// 4. Test Undo via POST /builders/:id/undo
	recUndo := httptest.NewRecorder()
	reqUndo, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+projectID+"/undo", nil)
	router.ServeHTTP(recUndo, reqUndo)

	if recUndo.Code != http.StatusOK {
		t.Fatalf("expected status 200 on undo, got %d: %s", recUndo.Code, recUndo.Body.String())
	}

	// 5. Test Redo via POST /builders/:id/redo
	recRedo := httptest.NewRecorder()
	reqRedo, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+projectID+"/redo", nil)
	router.ServeHTTP(recRedo, reqRedo)

	if recRedo.Code != http.StatusOK {
		t.Fatalf("expected status 200 on redo, got %d: %s", recRedo.Code, recRedo.Body.String())
	}

	// 6. Test Preview via POST /builders/:id/preview
	recPreview := httptest.NewRecorder()
	reqPreview, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+projectID+"/preview", nil)
	router.ServeHTTP(recPreview, reqPreview)

	if recPreview.Code != http.StatusOK {
		t.Fatalf("expected status 200 on preview, got %d: %s", recPreview.Code, recPreview.Body.String())
	}

	// 7. Test Export via POST /builders/:id/export
	recExport := httptest.NewRecorder()
	reqExport, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+projectID+"/export", nil)
	router.ServeHTTP(recExport, reqExport)

	if recExport.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 on export, got %d: %s", recExport.Code, recExport.Body.String())
	}
	var exportResp struct {
		Project     agent.BuilderProject `json:"project"`
		ArchivePath string               `json:"archive_path"`
		Checksum    string               `json:"checksum"`
		SHA256      string               `json:"sha256"`
		DownloadURL string               `json:"download_url"`
	}
	if err := json.Unmarshal(recExport.Body.Bytes(), &exportResp); err != nil {
		t.Fatalf("failed to decode export response: %v", err)
	}
	if exportResp.Checksum == "" || exportResp.SHA256 == "" {
		t.Errorf("expected traceable checksum in export response, got empty")
	}
	if exportResp.DownloadURL != "/api/agent/v1/builders/"+projectID+"/download" {
		t.Errorf("unexpected download URL: %s", exportResp.DownloadURL)
	}

	// 8. Test Download via GET /builders/:id/download
	recDownload := httptest.NewRecorder()
	reqDownload, _ := http.NewRequest(http.MethodGet, "/api/agent/v1/builders/"+projectID+"/download", nil)
	router.ServeHTTP(recDownload, reqDownload)

	if recDownload.Code != http.StatusOK {
		t.Fatalf("expected status 200 on download, got %d: %s", recDownload.Code, recDownload.Body.String())
	}
	if recDownload.Header().Get("X-Checksum-SHA256") != exportResp.Checksum {
		t.Errorf("header checksum mismatch: expected %s, got %s", exportResp.Checksum, recDownload.Header().Get("X-Checksum-SHA256"))
	}
	if !strings.Contains(recDownload.Header().Get("Content-Disposition"), "attachment; filename=") {
		t.Errorf("missing Content-Disposition attachment header: %s", recDownload.Header().Get("Content-Disposition"))
	}

	// Verify the downloaded bytes are a valid zip archive
	zipReader, err := zip.NewReader(bytes.NewReader(recDownload.Body.Bytes()), int64(recDownload.Body.Len()))
	if err != nil {
		t.Fatalf("downloaded body is not a valid zip: %v", err)
	}
	if len(zipReader.File) == 0 {
		t.Errorf("expected files in zip archive, got 0")
	}
}

func TestStudioDeployRejectsWithoutCredentialsHonestGateStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	builder, _ := agent.NewBuilderService(tmpDir)
	project, _ := builder.Create(context.Background(), agent.BuilderSpec{
		Name: "Deploy Test",
		Kind: agent.BuilderWebsite,
	})

	runtime, _ := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: tmpDir,
		Builder:       builder,
		// No deployment manager configured -> honest fail-closed
	})

	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.POST("/api/agent/v1/builders/:id/deploy/:provider", api.deployBuilder)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+project.ID+"/deploy/vercel", strings.NewReader(`{"target":"production"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	// Must fail honestly (501 or 400), NEVER 200 OK pretending to be published!
	if rec.Code == http.StatusOK {
		t.Fatalf("HONEST GATE VIOLATION: deploy succeeded without credentials!")
	}
}

func TestStudioPreviewAndDownloadRequireBearerWhenAuthEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	builder, err := agent.NewBuilderService(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: tmpDir, Builder: builder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })
	auth, err := agent.NewAuthStore("")
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.CreateUser("studio-owner@example.test", "Studio Owner")
	if err != nil {
		t.Fatal(err)
	}
	organization, _, err := auth.CreateOrganization("Studio Org", user)
	if err != nil {
		t.Fatal(err)
	}
	project, err := builder.Create(context.Background(), agent.BuilderSpec{
		Name: "Authenticated Studio", OrganizationID: organization.ID, Kind: agent.BuilderWebsite,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawToken, _, err := auth.IssueToken(user.ID, organization.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, auth: auth, authRequired: true}
	router := gin.New()
	group := router.Group("/api/agent/v1")
	group.Use(api.authMiddleware)
	group.POST("/builders/:id/preview", api.previewBuilder)
	group.GET("/builders/:id/preview/*path", api.builderPreviewFile)
	group.POST("/builders/:id/export", api.exportBuilder)
	group.GET("/builders/:id/download", api.downloadBuilder)

	for _, path := range []string{
		"/api/agent/v1/builders/" + project.ID + "/preview",
		"/api/agent/v1/builders/" + project.ID + "/preview/index.html",
		"/api/agent/v1/builders/" + project.ID + "/download",
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if strings.HasSuffix(path, "/preview") {
			req.Method = http.MethodPost
		}
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}

	authed := func(method, path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)
		router.ServeHTTP(recorder, req)
		return recorder
	}
	preview := authed(http.MethodPost, "/api/agent/v1/builders/"+project.ID+"/preview")
	if preview.Code != http.StatusOK {
		t.Fatalf("authenticated preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	previewFile := authed(http.MethodGet, "/api/agent/v1/builders/"+project.ID+"/preview/index.html")
	// net/http ServeFile canonically redirects index.html to the directory;
	// the browser/fetch client follows this same-origin redirect.
	if previewFile.Code != http.StatusOK && previewFile.Code != http.StatusMovedPermanently {
		t.Fatalf("authenticated preview file status=%d body=%s", previewFile.Code, previewFile.Body.String())
	}
	export := authed(http.MethodPost, "/api/agent/v1/builders/"+project.ID+"/export")
	if export.Code != http.StatusAccepted {
		t.Fatalf("authenticated export status=%d body=%s", export.Code, export.Body.String())
	}
	download := authed(http.MethodGet, "/api/agent/v1/builders/"+project.ID+"/download")
	if download.Code != http.StatusOK || download.Header().Get("X-Checksum-SHA256") == "" {
		t.Fatalf("authenticated download status=%d checksum=%q", download.Code, download.Header().Get("X-Checksum-SHA256"))
	}
}
