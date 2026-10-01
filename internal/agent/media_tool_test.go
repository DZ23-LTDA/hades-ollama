package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestMediaProcessToolDescriptorAndLocalTone(t *testing.T) {
	tool := mediaProcessTool{}
	descriptor := tool.Descriptor()
	if descriptor.Name != "media.process" || descriptor.Risk != RiskExternalSideEffect || !descriptor.RequiresApproval || len(descriptor.Scopes) != 1 || descriptor.Scopes[0] != "media:execute" {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	workspace := t.TempDir()
	workspaceRoot, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer workspaceRoot.Close()
	result, err := tool.Execute(context.Background(), ToolContext{MissionID: "mission_tone", StepID: "step_tone", Workspace: workspace, WorkspaceRoot: workspaceRoot}, map[string]any{"operation": "tone.generate", "frequency_hz": 440, "duration_ms": 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].MissionID != "mission_tone" || result.Artifacts[0].StepID != "step_tone" || result.Artifacts[0].MediaType != "audio/wav" {
		t.Fatalf("result=%+v", result)
	}
	if _, err := tool.Execute(context.Background(), ToolContext{StepID: "step_tone", Workspace: workspace, WorkspaceRoot: workspaceRoot}, map[string]any{"operation": "arbitrary.http"}); err == nil {
		t.Fatal("unlisted operation was accepted")
	}
}

func TestMediaProcessOutputUsesPinnedWorkspaceAfterPathReplacement(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	movedWorkspace := workspace + "-pinned"
	if err := os.Rename(workspace, movedWorkspace); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, ".agent-media")); err != nil {
		t.Fatal(err)
	}

	result, err := (mediaProcessTool{}).Execute(context.Background(), ToolContext{
		MissionID: "mission_pinned_media", StepID: "step_pinned_media", Workspace: workspace, WorkspaceRoot: root,
	}, map[string]any{"operation": "tone.generate", "duration_ms": 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected one pinned artifact, got %+v", result.Artifacts)
	}
	file, err := root.Open(filepath.FromSlash(result.Artifacts[0].Path))
	if err != nil {
		t.Fatalf("artifact is missing from the pinned workspace: %v", err)
	}
	_ = file.Close()
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("media output escaped through replacement symlink: %v", entries)
	}
	if got := result.Value.(map[string]any)["path"]; got != result.Artifacts[0].Path {
		t.Fatalf("result path=%v is not the pinned relative artifact path %q", got, result.Artifacts[0].Path)
	}
}

func TestMediaProcessDoesNotCallProviderBeforeApproval(t *testing.T) {
	var requests atomic.Int32
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'p', 'n', 'g'}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/images/generations" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(png)}}})
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "fixture", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	workspace := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       fixedPlanner{steps: []Step{{ID: "step_media", Kind: "media.process", Title: "generate image", Risk: RiskExternalSideEffect, State: StepPending, Input: map[string]any{"operation": "image.generate", "prompt": "fixture"}}}},
		WorkspaceRoot: workspace,
		Media:         manager,
	})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "generate test image", Capabilities: []string{"media:execute"}})
	if err != nil {
		t.Fatal(err)
	}
	if mission.State != MissionAwaitingApproval || len(mission.Approvals) != 1 {
		t.Fatalf("mission=%+v", mission)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("provider called before approval: requests=%d", got)
	}
	mission, err = runtime.DecideApproval(mission.ID, mission.Approvals[0].ID, true, "provider fixture authorized")
	if err != nil || mission.State != MissionReady {
		t.Fatalf("approval mission=%+v err=%v", mission, err)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != MissionCompleted || len(completed.Artifacts) != 1 || completed.Artifacts[0].MissionID != mission.ID || completed.Artifacts[0].StepID != "step_1" {
		t.Fatalf("completed=%+v", completed)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("provider requests=%d, want exactly one after approval", got)
	}
}

func TestMediaProcessRequiresScopeAndConfiguredProvider(t *testing.T) {
	if _, err := (mediaProcessTool{}).Execute(context.Background(), ToolContext{StepID: "step_media", Workspace: t.TempDir()}, map[string]any{"operation": "tone.generate"}); err == nil || err.Error() != "pinned media workspace is required" {
		t.Fatalf("missing pinned root error = %v", err)
	}
	if _, err := DefaultCapabilityPolicy().ValidateMissionCapabilities([]string{"media:execute"}); err != nil {
		t.Fatalf("known capability unexpectedly rejected: %v", err)
	}
	tool := mediaProcessTool{}
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, err = tool.Execute(context.Background(), ToolContext{StepID: "step_media", Workspace: workspace, WorkspaceRoot: root}, map[string]any{"operation": "image.generate", "prompt": "test"})
	if err == nil || err.Error() != "media provider is not configured" {
		t.Fatalf("unconfigured provider error=%v", err)
	}
}
