package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func uploadTestCtx(method, target, org string, params gin.Params, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Params = params
	ctx.Set("agent.organization", agent.Organization{ID: org})
	return ctx, rec
}

func TestUploadRoutesHappyPathAndCrossTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}

	content := []byte("payload-http-upload-test")
	sum := sha256.Sum256(content)
	startBody, _ := json.Marshal(map[string]any{
		"project_id": "p", "filename": "f.bin",
		"total_size": len(content), "chunk_size": 8, "sha256": hex.EncodeToString(sum[:]),
	})

	// Start (org-a) -> 201.
	ctx, rec := uploadTestCtx(http.MethodPost, "/api/agent/v1/uploads", "org-a", nil, startBody)
	api.startUpload(ctx)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start code=%d body=%s", rec.Code, rec.Body.String())
	}
	var session agent.UploadSession
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil || session.ID == "" {
		t.Fatalf("start decode: %v session=%+v", err, session)
	}
	id := session.ID
	p := gin.Params{{Key: "id", Value: id}}

	// Cross-tenant GET (org-b) -> 403.
	ctx, rec = uploadTestCtx(http.MethodGet, "/api/agent/v1/uploads/"+id, "org-b", p, nil)
	api.getUpload(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant get code=%d", rec.Code)
	}

	// Chunk (org-a, sequencial) -> 200.
	for off := 0; off < len(content); off += 8 {
		end := off + 8
		if end > len(content) {
			end = len(content)
		}
		ctx, rec = uploadTestCtx(http.MethodPut, "/api/agent/v1/uploads/"+id+"/chunk?offset="+itoaTest(off), "org-a", p, content[off:end])
		api.uploadChunk(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("chunk off=%d code=%d body=%s", off, rec.Code, rec.Body.String())
		}
	}

	// Finalize (org-a) -> 200 completed.
	ctx, rec = uploadTestCtx(http.MethodPost, "/api/agent/v1/uploads/"+id+"/finalize", "org-a", p, nil)
	api.finalizeUpload(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("finalize code=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil || session.State != agent.UploadCompleted {
		t.Fatalf("finalize decode: %v state=%s", err, session.State)
	}
	if strings.Contains(rec.Body.String(), "final_path") {
		t.Fatalf("finalize response disclosed a server filesystem path: %s", rec.Body.String())
	}
}

func itoaTest(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func TestUploadRouteSanitizesFilesystemErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataRoot := t.TempDir()
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: workspace, DataRoot: dataRoot})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}
	session, err := runtime.Uploads().StartUpload("org-a", "p", "f.bin", 4, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Uploads().AppendChunk("org-a", session.ID, 0, []byte("data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dataRoot, ".agent-uploads", session.ID+".part")); err != nil {
		t.Fatal(err)
	}
	params := gin.Params{{Key: "id", Value: session.ID}}
	ctx, recorder := uploadTestCtx(http.MethodPost, "/api/agent/v1/uploads/"+session.ID+"/finalize", "org-a", params, nil)
	api.finalizeUpload(ctx)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want 500", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), dataRoot) || strings.Contains(recorder.Body.String(), ".part") {
		t.Fatalf("filesystem details leaked in response: %s", recorder.Body.String())
	}
}

func TestWriteAgentErrorRedactsEmbeddedCredentialJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writeAgentError(ctx, http.StatusBadGateway, errors.New(`provider rejected request: {"api_key":"plain-secret-value"}`))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusBadGateway)
	}
	if strings.Contains(recorder.Body.String(), "plain-secret-value") || strings.Contains(recorder.Body.String(), "api_key") {
		t.Fatalf("API error leaked embedded credential: %s", recorder.Body.String())
	}
}
