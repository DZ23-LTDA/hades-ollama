package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestRunMissionReportsQueuedNotRunning asserts that POST /missions/:id/run
// reports the mission as queued (its real pre-claim state) instead of a
// premature RUNNING. The transition to RUNNING only happens when a worker
// claims the job; reporting RUNNING on enqueue misleads the client.
func TestRunMissionReportsQueuedNotRunning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "deve ficar enfileirada, não running",
		OrganizationID: "org-a",
		Workspace:      workspace,
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &agentAPI{runtime: runtime, authRequired: true}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/missions/"+mission.ID+"/run", nil)
	ctx.Params = gin.Params{{Key: "id", Value: mission.ID}}
	ctx.Set("agent.organization", agent.Organization{ID: "org-a"})

	api.runMission(ctx)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("run status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		MissionID string         `json:"mission_id"`
		State     string         `json:"state"`
		Queued    bool           `json:"queued"`
		Job       agent.QueueJob `json:"job"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode run response: %v body=%s", err, recorder.Body.String())
	}
	if payload.State == string(agent.MissionRunning) {
		t.Fatalf("run response claims RUNNING before a worker claimed the job: %s", recorder.Body.String())
	}
	if !payload.Queued {
		t.Fatalf("run response must mark the mission as queued: %s", recorder.Body.String())
	}
	if payload.Job.ID == "" {
		t.Fatalf("run response must include the queued job: %s", recorder.Body.String())
	}
}
