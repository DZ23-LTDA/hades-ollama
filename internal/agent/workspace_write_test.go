package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceWriteApprovalBindsPayloadAndCreatesFile(t *testing.T) {
	workspace := t.TempDir()
	input := map[string]any{"path": "nested/result.txt", "content": "approved output"}
	prepared, err := prepareWorkspaceWriteApproval(workspace, input)
	if err != nil {
		t.Fatal(err)
	}
	step := Step{ID: "step_write", Kind: "workspace.write", Title: "write output", Risk: RiskWrite, RequiresApproval: true, Input: prepared}
	hashBefore, err := approvalPayloadSHA256(step)
	if err != nil {
		t.Fatal(err)
	}
	step.Input = cloneMap(prepared)
	step.Input["content"] = "different output"
	hashAfter, err := approvalPayloadSHA256(step)
	if err != nil {
		t.Fatal(err)
	}
	if hashBefore == hashAfter {
		t.Fatal("payload hash did not change when approved content changed")
	}
	result, err := writeWorkspaceFile(ToolContext{MissionID: "mission_test", StepID: "step_write", Workspace: workspace}, prepared)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(workspace, "nested", "result.txt"))
	if err != nil || string(data) != "approved output" {
		t.Fatalf("written=%q err=%v", data, err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].SHA256 != workspaceContentSHA256(data) {
		t.Fatalf("artifacts=%+v", result.Artifacts)
	}
}

func TestWorkspaceWriteRejectsChangedTargetAndPreservesBackup(t *testing.T) {
	workspace := t.TempDir()
	target := filepath.Join(workspace, "existing.txt")
	if err := os.WriteFile(target, []byte("original"), 0o640); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareWorkspaceWriteApproval(workspace, map[string]any{"path": "existing.txt", "content": "replacement"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("concurrent change"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = writeWorkspaceFile(ToolContext{MissionID: "mission_test", StepID: "step_write", Workspace: workspace}, prepared)
	if err == nil || !strings.Contains(err.Error(), "changed after approval") {
		t.Fatalf("write error=%v, want stale target rejection", err)
	}
	current, readErr := os.ReadFile(target)
	if readErr != nil || string(current) != "concurrent change" {
		t.Fatalf("stale write modified target: %q err=%v", current, readErr)
	}

	prepared, err = prepareWorkspaceWriteApproval(workspace, map[string]any{"path": "existing.txt", "content": "approved replacement", "suppress_backup": true})
	if err != nil {
		t.Fatal(err)
	}
	result, err := writeWorkspaceFile(ToolContext{MissionID: "mission_test", StepID: "step_write", Workspace: workspace}, prepared)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(target)
	if err != nil || string(updated) != "approved replacement" {
		t.Fatalf("updated=%q err=%v", updated, err)
	}
	if len(result.Artifacts) != 2 {
		t.Fatalf("overwrite should return backup and output artifacts, got %+v", result.Artifacts)
	}
	backup := result.Artifacts[0]
	if !strings.HasPrefix(backup.Path, ".agent-backups/") {
		t.Fatalf("backup path=%q", backup.Path)
	}
	backupData, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(backup.Path)))
	if err != nil || string(backupData) != "concurrent change" {
		t.Fatalf("backup=%q err=%v", backupData, err)
	}
}

func TestWorkspaceWriteRejectsSensitiveContentAndUnsafePaths(t *testing.T) {
	workspace := t.TempDir()
	for _, input := range []map[string]any{
		{"path": "secret.txt", "content": "api_key=super-secret-token-value"},
		{"path": "../outside.txt", "content": "safe"},
		{"path": ".git/config", "content": "safe"},
	} {
		if _, err := prepareWorkspaceWriteApproval(workspace, input); err == nil {
			t.Errorf("expected input rejection: %+v", input)
		}
	}
	if _, err := writeWorkspaceFile(ToolContext{MissionID: "mission_test", StepID: "step_write", Workspace: workspace}, map[string]any{"path": "new.txt", "content": "unapproved"}); err == nil {
		t.Fatal("expected missing approved hash rejection")
	}
}

func TestWorkspaceCreateFailsClosedWhenTargetAppearsBeforeCommit(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	path := filepath.Join(workspace, "raced.txt")
	err = atomicWorkspaceCreateChecked(root, "raced.txt", []byte("approved"), func() error {
		return os.WriteFile(path, []byte("concurrent"), 0o600)
	})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("create error=%v, want already-exists", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "concurrent" {
		t.Fatalf("concurrent target changed: %q err=%v", got, err)
	}
	entries, err := os.ReadDir(workspace)
	if err != nil || len(entries) != 1 || entries[0].Name() != "raced.txt" {
		t.Fatalf("temporary write leaked: entries=%v err=%v", entries, err)
	}
}

func TestWorkspaceWritePatchCompensatesAndDoesNotLeaveBackups(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "first.txt"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareWorkspacePatchApproval(workspace, map[string]any{"files": []any{
		map[string]any{"path": "first.txt", "content": "after"},
		map[string]any{"path": "second.txt", "content": "second"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	files := prepared["files"].([]any)
	secondInput := files[1].(map[string]any)
	writerCalls := 0
	_, err = writeWorkspacePatchWith(ToolContext{MissionID: "mission_test", StepID: "step_patch", Workspace: workspace}, prepared, func(tc ToolContext, input map[string]any) (ToolResult, error) {
		writerCalls++
		if writerCalls == 2 {
			return ToolResult{}, errors.New("injected failure")
		}
		return writeWorkspaceFileUnlockedWithOptions(tc, input, true)
	})
	if err == nil || !strings.Contains(err.Error(), "compensated") {
		t.Fatalf("patch error=%v, want compensated failure for %+v", err, secondInput)
	}
	first, err := os.ReadFile(filepath.Join(workspace, "first.txt"))
	if err != nil || string(first) != "before" {
		t.Fatalf("first file not restored: %q err=%v", first, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "second.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second file unexpectedly exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".agent-backups")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("compensated patch left backups: %v", err)
	}
}

func TestWorkspacePatchRejectsCaseInsensitiveDuplicatePaths(t *testing.T) {
	workspace := t.TempDir()
	_, err := prepareWorkspacePatchApproval(workspace, map[string]any{"files": []any{
		map[string]any{"path": "A.txt", "content": "first"},
		map[string]any{"path": "a.txt", "content": "second"},
	}})
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("case-insensitive duplicate error=%v", err)
	}
}

func TestApprovalPayloadHashRejectsUnserializableInput(t *testing.T) {
	_, err := approvalPayloadSHA256(Step{ID: "step_write", Input: map[string]any{"bad": make(chan int)}})
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatal("expected JSON serialization error")
	}
}

func TestApprovalPayloadHashRejectsInputChangedByPersistenceRedaction(t *testing.T) {
	step := Step{ID: "step_connector", Kind: "connector.call", Title: "send request", RequiresApproval: true, Input: map[string]any{"params": map[string]any{"api_key": "abcdef0123456789abcdef"}}}
	if _, err := approvalPayloadSHA256(step); err == nil || !strings.Contains(err.Error(), "sensitive data") {
		t.Fatalf("sensitive approval hash error = %v, want fail-closed persistence error", err)
	}
}
