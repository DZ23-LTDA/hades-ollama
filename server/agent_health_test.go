package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestHealthReportsRealQueueDepth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	(&agentAPI{runtime: runtime}).health(context)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Status     string `json:"status"`
		Subsystems struct {
			Queue struct {
				Health     string `json:"health"`
				Pending    *int   `json:"pending"`
				Running    *int   `json:"running"`
				DeadLetter *int   `json:"dead_letter"`
			} `json:"queue"`
		} `json:"subsystems"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health: %v (body=%s)", err, recorder.Body.String())
	}

	// A fresh runtime has an empty, healthy queue. The counts must be present
	// (real numbers from the queue), not a static placeholder label.
	if body.Subsystems.Queue.Pending == nil || body.Subsystems.Queue.Running == nil || body.Subsystems.Queue.DeadLetter == nil {
		t.Fatalf("queue depth fields missing: %s", recorder.Body.String())
	}
	if *body.Subsystems.Queue.DeadLetter != 0 {
		t.Fatalf("fresh queue dead_letter = %d, want 0", *body.Subsystems.Queue.DeadLetter)
	}
	if body.Subsystems.Queue.Health != "ok" {
		t.Fatalf("fresh queue health = %q, want ok", body.Subsystems.Queue.Health)
	}
	if body.Status != "ok" {
		t.Fatalf("overall status = %q, want ok", body.Status)
	}
}
