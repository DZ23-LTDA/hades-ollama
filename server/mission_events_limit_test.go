package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestMissionEventsSupportLimitAndReportTotal asserts that the mission events
// endpoint bounds its response with ?limit (returning the most recent N) while
// reporting the real total, so a long history cannot force an unbounded payload.
func TestMissionEventsSupportLimitAndReportTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	store := agent.NewMemoryStore()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: store, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close(context.Background()) })

	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "eventos", OrganizationID: "org-a", Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		if err := store.AppendEvent(agent.Event{ID: fmt.Sprintf("ev_%d", i), MissionID: mission.ID, OrganizationID: "org-a", Type: "test", CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}

	api := &agentAPI{runtime: runtime, authRequired: true}
	call := func(query string) (int, int) {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		path := "/api/agent/v1/missions/" + mission.ID + "/events" + query
		ctx.Request = httptest.NewRequest(http.MethodGet, path, nil)
		ctx.Params = gin.Params{{Key: "id", Value: mission.ID}}
		ctx.Set("agent.organization", agent.Organization{ID: "org-a"})
		api.events(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("events status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Events []agent.Event `json:"events"`
			Total  int           `json:"total"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode events: %v body=%s", err, recorder.Body.String())
		}
		return len(payload.Events), payload.Total
	}

	fullLen, fullTotal := call("")
	if fullTotal < 5 || fullLen != fullTotal {
		t.Fatalf("unlimited events should return every event: len=%d total=%d", fullLen, fullTotal)
	}

	limitedLen, limitedTotal := call("?limit=2")
	if limitedLen != 2 {
		t.Fatalf("limit=2 should return exactly 2 events, got %d", limitedLen)
	}
	if limitedTotal != fullTotal {
		t.Fatalf("total must report the real count regardless of limit: got %d want %d", limitedTotal, fullTotal)
	}
}
