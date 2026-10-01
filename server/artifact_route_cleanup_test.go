package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestArtifactRouteRemovesVerifiedTemporarySnapshotAfterDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	artifactPath := filepath.Join(workspace, "download.txt")
	const content = "verified artifact payload"
	if err := os.WriteFile(artifactPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store := agent.NewMemoryStore()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: store, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "artifact cleanup", Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := agent.BuildArtifactManifest(workspace, mission.ID, "step_1", "download.txt", "download.txt")
	if err != nil {
		t.Fatal(err)
	}
	mission.Artifacts = []agent.ArtifactManifest{manifest}
	mission.Version++
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}
	before, err := filepath.Glob(filepath.Join(os.TempDir(), ".ollama-artifact-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.GET("/missions/:id/artifacts/:artifact_id", api.artifact)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missions/"+mission.ID+"/artifacts/"+manifest.ID, nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != content {
		t.Fatalf("artifact response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	after, err := filepath.Glob(filepath.Join(os.TempDir(), ".ollama-artifact-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	beforeSet := make(map[string]bool, len(before))
	for _, path := range before {
		beforeSet[path] = true
	}
	for _, path := range after {
		if !beforeSet[path] {
			t.Errorf("artifact temporary snapshot leaked after response: %s", path)
		}
	}
}
