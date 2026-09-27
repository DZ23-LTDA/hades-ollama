package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestMediaApprovalBindsInputBytes(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "input.wav")
	if err := os.WriteFile(path, []byte("approved audio bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareMediaProcessApproval(workspace, map[string]any{"operation": "audio.transcribe", "input_path": "input.wav"})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := approvalPayloadSHA256(Step{ID: "step_media", Kind: "media.process", Risk: RiskExternalSideEffect, RequiresApproval: true, Input: prepared})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed audio bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = readApprovedMediaInput(context.Background(), workspace, prepared, 100<<20)
	if !errors.Is(err, ErrMediaInputChanged) {
		t.Fatalf("changed media input error=%v, want ErrMediaInputChanged", err)
	}
	prepared["prompt"] = "different approved prompt"
	current, err := approvalPayloadSHA256(Step{ID: "step_media", Kind: "media.process", Risk: RiskExternalSideEffect, RequiresApproval: true, Input: prepared})
	if err != nil || current == approved {
		t.Fatalf("payload hash did not bind prepared media content: old=%s new=%s err=%v", approved, current, err)
	}
}

func TestMediaDLPBlocksSensitiveTextBeforeProviderRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "fixture", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	if _, err := manager.GenerateImage(context.Background(), t.TempDir(), "api_key=super-secret-token-value", ""); err == nil {
		t.Fatal("sensitive media prompt was accepted")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("DLP rejection still caused provider requests: %d", got)
	}
	if _, err := prepareMediaProcessApproval(t.TempDir(), map[string]any{"operation": "speech.generate", "text": "password=super-secret-password-value"}); err == nil {
		t.Fatal("sensitive speech text passed approval preparation")
	}
}

func TestApprovedMediaInputReadsFromPinnedWorkspaceAfterPathReplacement(t *testing.T) {
	workspace := t.TempDir()
	inputPath := filepath.Join(workspace, "image.png")
	approvedBytes := []byte("approved image bytes")
	if err := os.WriteFile(inputPath, approvedBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareMediaProcessApproval(workspace, map[string]any{"operation": "image.analyze", "input_path": "image.png", "prompt": "describe this image"})
	if err != nil {
		t.Fatal(err)
	}
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
	if err := os.WriteFile(filepath.Join(workspace, "image.png"), []byte("replacement image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	relative, data, err := readApprovedMediaInput(context.Background(), workspace, prepared, 25<<20, root)
	if err != nil {
		t.Fatal(err)
	}
	if relative != "image.png" || string(data) != string(approvedBytes) {
		t.Fatalf("approved read followed replacement path: relative=%q data=%q", relative, data)
	}
}

func TestMediaProcessDoesNotEgressInputChangedAfterApproval(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "fixture", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	workspace := t.TempDir()
	path := filepath.Join(workspace, "image.png")
	if err := os.WriteFile(path, []byte("approved image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareMediaProcessApproval(workspace, map[string]any{"operation": "image.analyze", "input_path": "image.png", "prompt": "describe this image"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	workspaceRoot, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer workspaceRoot.Close()
	tool := mediaProcessTool{manager: manager}
	_, err = tool.Execute(context.Background(), ToolContext{MissionID: "mission_media", StepID: "step_media", Workspace: workspace, WorkspaceRoot: workspaceRoot}, prepared)
	if !errors.Is(err, ErrMediaInputChanged) {
		t.Fatalf("changed image error=%v, want ErrMediaInputChanged", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("changed approved input reached provider: requests=%d", got)
	}
}

func TestMediaApprovalCanonicalizesToneAndRejectsInvalidInputs(t *testing.T) {
	prepared, err := prepareMediaProcessApproval(t.TempDir(), map[string]any{"operation": "tone.generate"})
	if err != nil {
		t.Fatal(err)
	}
	if prepared["frequency_hz"] != float64(440) || prepared["duration_ms"] != 1000 {
		t.Fatalf("tone defaults are not explicit in approval payload: %#v", prepared)
	}

	invalidInputs := []map[string]any{
		{"operation": "tone.generate", "frequency_hz": float64(20001)},
		{"operation": "tone.generate", "frequency_hz": "not-a-number"},
		{"operation": "tone.generate", "duration_ms": 30001},
		{"operation": "tone.generate", "duration_ms": 1.5},
		{"operation": "image.generate", "prompt": "  "},
		{"operation": "video.generate", "prompt": "  "},
		{"operation": "speech.generate", "text": "  "},
		{"operation": "image.analyze", "input_path": "missing.png", "prompt": "  "},
	}
	for _, input := range invalidInputs {
		if _, err := prepareMediaProcessApproval(t.TempDir(), input); err == nil {
			t.Errorf("invalid media approval input was accepted: %#v", input)
		}
	}
}
