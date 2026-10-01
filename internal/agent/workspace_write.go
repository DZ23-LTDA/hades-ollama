package agent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const workspaceWriteMaxBytes = 1 << 20

var (
	ErrApprovalPayloadChanged = errors.New("approved action payload changed")
	workspaceWriteMu          sync.Mutex
)

func approvalPayloadSHA256(step Step) (string, error) {
	inputBytes, err := json.Marshal(step.Input)
	if err != nil {
		return "", err
	}
	redactedInputBytes, err := json.Marshal(RedactValue(step.Input))
	if err != nil {
		return "", err
	}
	if !bytes.Equal(inputBytes, redactedInputBytes) {
		return "", errors.New("approval payload contains sensitive data that cannot be persisted safely")
	}
	payload := struct {
		ID                   string         `json:"id"`
		Kind                 string         `json:"kind"`
		Title                string         `json:"title"`
		Input                map[string]any `json:"input"`
		Risk                 RiskClass      `json:"risk"`
		RequiresApproval     bool           `json:"requires_approval"`
		ToolDescriptorSHA256 string         `json:"tool_descriptor_sha256"`
		ToolConfigSHA256     string         `json:"tool_config_sha256"`
	}{step.ID, step.Kind, step.Title, step.Input, step.Risk, step.RequiresApproval, step.ToolDescriptorSHA256, step.ToolConfigSHA256}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func prepareWorkspaceWriteApproval(workspace string, input map[string]any) (map[string]any, error) {
	if input == nil {
		return nil, errors.New("workspace.write input is required")
	}
	path, err := validateWorkspaceWritePath(workspace, stringInput(input, "path", ""))
	if err != nil {
		return nil, err
	}
	content, ok := input["content"].(string)
	if !ok {
		return nil, errors.New("workspace.write content must be a string")
	}
	if len(content) > workspaceWriteMaxBytes {
		return nil, errors.New("workspace.write content exceeds per-file limit")
	}
	if findings := ScanDLP(content); len(findings) > 0 {
		return nil, fmt.Errorf("workspace.write content contains sensitive data (%s)", findings[0].Kind)
	}
	rootPath, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	root, err := openArtifactWorkspaceRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	relative, err := filepath.Rel(rootPath, path)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("workspace.write path escapes workspace")
	}
	if err := rejectSymlinkComponents(rootPath, filepath.Dir(path)); err != nil {
		return nil, err
	}
	prepared := cloneMap(input)
	delete(prepared, "suppress_backup")
	prepared["path"] = filepath.ToSlash(relative)
	prepared["content"] = content
	info, statErr := root.Lstat(filepath.ToSlash(relative))
	switch {
	case statErr == nil:
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("workspace.write target must be a regular non-symlink file")
		}
		before, readErr := readWorkspaceFileLimited(root, filepath.ToSlash(relative), workspaceWriteMaxBytes, "workspace.write existing file exceeds per-file limit")
		if readErr != nil {
			return nil, readErr
		}
		prepared["expected_sha256"] = workspaceContentSHA256(before)
		prepared["expected_absent"] = false
	case errors.Is(statErr, os.ErrNotExist):
		prepared["expected_sha256"] = ""
		prepared["expected_absent"] = true
	default:
		return nil, statErr
	}
	prepared["preview_version"] = 1
	return prepared, nil
}

func validateWorkspaceWritePath(workspace, raw string) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || strings.ContainsRune(raw, '\x00') || strings.HasPrefix(raw, "/") || filepath.IsAbs(raw) || filepath.VolumeName(raw) != "" {
		return "", errors.New("workspace.write path must be relative to workspace")
	}
	for _, component := range strings.Split(raw, "/") {
		if component == ".." {
			return "", errors.New("workspace.write path escapes workspace")
		}
		if strings.EqualFold(component, ".git") || strings.EqualFold(component, ".agent-backups") {
			return "", errors.New("workspace.write path targets protected runtime metadata")
		}
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("workspace.write path escapes workspace")
	}
	return safeWorkspacePath(workspace, clean)
}

func validateAnchoredWorkspaceWritePath(root *os.Root, raw string) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || strings.ContainsRune(raw, '\x00') || strings.HasPrefix(raw, "/") || filepath.IsAbs(raw) || filepath.VolumeName(raw) != "" {
		return "", errors.New("workspace.write path must be relative to workspace")
	}
	for _, component := range strings.Split(raw, "/") {
		if component == ".." {
			return "", errors.New("workspace.write path escapes workspace")
		}
		if strings.EqualFold(component, ".git") || strings.EqualFold(component, ".agent-backups") {
			return "", errors.New("workspace.write path targets protected runtime metadata")
		}
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." || !filepath.IsLocal(clean) {
		return "", errors.New("workspace.write path escapes workspace")
	}
	return safeAnchoredWorkspaceRelative(root, clean, false)
}

func writeWorkspaceFile(toolContext ToolContext, input map[string]any) (ToolResult, error) {
	workspaceWriteMu.Lock()
	defer workspaceWriteMu.Unlock()
	var result ToolResult
	err := withWorkspaceToolMutationLock(toolContext, func() error {
		var writeErr error
		result, writeErr = writeWorkspaceFileUnlocked(toolContext, input)
		return writeErr
	})
	return result, err
}

func writeWorkspaceFileUnlocked(toolContext ToolContext, input map[string]any) (ToolResult, error) {
	return writeWorkspaceFileUnlockedWithOptions(toolContext, input, false)
}

func writeWorkspaceFileUnlockedWithOptions(toolContext ToolContext, input map[string]any, suppressBackup bool) (ToolResult, error) {
	if err := validateStepID(toolContext.StepID); err != nil {
		return ToolResult{}, err
	}
	var path, relative string
	var err error
	if toolContext.WorkspaceRoot != nil {
		relative, err = validateAnchoredWorkspaceWritePath(toolContext.WorkspaceRoot, stringInput(input, "path", ""))
		path = relative
	} else {
		path, err = validateWorkspaceWritePath(toolContext.Workspace, stringInput(input, "path", ""))
		relative = filepath.ToSlash(filepath.Clean(strings.ReplaceAll(stringInput(input, "path", ""), "\\", "/")))
	}
	if err != nil {
		return ToolResult{}, err
	}
	content, ok := input["content"].(string)
	if !ok || len(content) > workspaceWriteMaxBytes {
		return ToolResult{}, errors.New("workspace.write content is missing or exceeds per-file limit")
	}
	if findings := ScanDLP(content); len(findings) > 0 {
		return ToolResult{}, fmt.Errorf("workspace.write content contains sensitive data (%s)", findings[0].Kind)
	}
	expectedHash := strings.ToLower(strings.TrimSpace(stringInput(input, "expected_sha256", "")))
	expectedAbsent, hasAbsent := input["expected_absent"].(bool)
	if !hasAbsent || (expectedAbsent && expectedHash != "") || (!expectedAbsent && len(expectedHash) != 64) {
		return ToolResult{}, errors.New("workspace.write requires an approved target hash or expected_absent")
	}
	rootPath := toolContext.Workspace
	root := toolContext.WorkspaceRoot
	if root == nil {
		rootPath, err = filepath.Abs(toolContext.Workspace)
		if err != nil {
			return ToolResult{}, err
		}
		if err := rejectSymlinkComponents(rootPath, path); err != nil {
			return ToolResult{}, err
		}
		root, err = openArtifactWorkspaceRoot(rootPath)
		if err != nil {
			return ToolResult{}, err
		}
		defer root.Close()
	}
	if err := verifyWorkspaceTarget(root, relative, !expectedAbsent, expectedHash); err != nil {
		return ToolResult{}, err
	}
	var artifacts []ArtifactManifest
	if !expectedAbsent && !suppressBackup {
		before, readErr := readWorkspaceFileLimited(root, relative, workspaceWriteMaxBytes, "workspace.write existing file exceeds per-file limit")
		if readErr != nil {
			return ToolResult{}, readErr
		}
		backupRelative := workspaceBackupRelativePath(toolContext, relative)
		if toolContext.WorkspaceRoot == nil {
			if err := rejectSymlinkComponents(rootPath, filepath.Join(rootPath, filepath.FromSlash(backupRelative))); err != nil {
				return ToolResult{}, err
			}
		}
		if err := atomicWorkspaceWriteChecked(root, backupRelative, before, func() error {
			return verifyWorkspaceTarget(root, backupRelative, false, "")
		}); err != nil {
			return ToolResult{}, fmt.Errorf("write workspace backup: %w", err)
		}
		backupArtifact, manifestErr := buildArtifactManifestFromRoot(root, backupRelative, toolContext.MissionID, toolContext.StepID, "Backup of "+filepath.Base(relative))
		if manifestErr != nil {
			return ToolResult{}, fmt.Errorf("manifest workspace backup: %w", manifestErr)
		}
		artifacts = append(artifacts, backupArtifact)
	}
	data := []byte(content)
	mode := os.FileMode(0o600)
	if !expectedAbsent {
		info, statErr := root.Lstat(relative)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			if statErr != nil {
				return ToolResult{}, statErr
			}
			return ToolResult{}, errors.New("workspace.write target changed type")
		}
		mode = 0o600 | info.Mode().Perm()&0o100
	}
	beforeRename := func() error {
		return verifyWorkspaceTarget(root, relative, !expectedAbsent, expectedHash)
	}
	var writeErr error
	if expectedAbsent {
		writeErr = atomicWorkspaceCreateChecked(root, relative, data, beforeRename, mode)
	} else {
		writeErr = atomicWorkspaceWriteChecked(root, relative, data, beforeRename, mode)
	}
	if writeErr != nil {
		return ToolResult{Artifacts: artifacts}, writeErr
	}
	artifact, err := buildArtifactManifestFromRoot(root, relative, toolContext.MissionID, toolContext.StepID, filepath.Base(relative))
	if err != nil {
		return ToolResult{Artifacts: artifacts}, fmt.Errorf("manifest workspace write: %w", err)
	}
	artifacts = append(artifacts, artifact)
	return ToolResult{Value: map[string]any{"path": relative, "bytes": len(data), "sha256": artifact.SHA256, "backup_created": len(artifacts) > 1}, Artifacts: artifacts}, nil
}

func workspaceBackupRelativePath(toolContext ToolContext, relative string) string {
	mission := sha256.Sum256([]byte(toolContext.MissionID))
	step := sha256.Sum256([]byte(toolContext.StepID))
	name := fmt.Sprintf("backup-%d-%s.bin", time.Now().UTC().UnixNano(), uuid.NewString())
	return filepath.ToSlash(filepath.Join(".agent-backups", hex.EncodeToString(mission[:8]), hex.EncodeToString(step[:8]), name))
}

func withWorkspaceMutationLock(workspace string, run func() error) error {
	if strings.TrimSpace(workspace) == "" || run == nil {
		return errors.New("workspace and mutation callback are required")
	}
	root, err := filepath.Abs(filepath.Clean(workspace))
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	lockDir := filepath.Join(cacheRoot, "ollama-agent", "workspace-locks")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(lockDir, 0o700); err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(root))
	return withFileLock(filepath.Join(lockDir, hex.EncodeToString(digest[:])+".lock"), run)
}

func withWorkspaceToolMutationLock(toolContext ToolContext, run func() error) error {
	if toolContext.WorkspaceRoot == nil {
		return withWorkspaceMutationLock(toolContext.Workspace, run)
	}
	if strings.TrimSpace(toolContext.MissionID) == "" || run == nil {
		return errors.New("mission and mutation callback are required for anchored workspace writes")
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	lockDir := filepath.Join(cacheRoot, "ollama-agent", "workspace-locks")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(lockDir, 0o700); err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(toolContext.MissionID + "\x00" + toolContext.WorkspaceRoot.Name()))
	return withFileLock(filepath.Join(lockDir, hex.EncodeToString(digest[:])+".lock"), run)
}

func withMissionStoreLock(root string, run func() error) error {
	if strings.TrimSpace(root) == "" || run == nil {
		return errors.New("mission store root and callback are required")
	}
	absolute, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("mission store root must be a real directory")
	}
	return withFileLock(filepath.Join(absolute, ".missions.lock"), run)
}

func readWorkspacePathLimited(workspace, path string, limit int, message string) ([]byte, error) {
	if strings.TrimSpace(workspace) == "" {
		return nil, errors.New("workspace is required")
	}
	rootPath, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkComponents(rootPath, path); err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(rootPath, path)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("tool path escapes workspace")
	}
	root, err := openArtifactWorkspaceRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readWorkspaceFileLimited(root, filepath.ToSlash(relative), limit, message)
}

func readWorkspaceFileLimited(root *os.Root, relative string, limit int, message string) ([]byte, error) {
	if limit < 0 {
		return nil, errors.New("workspace file limit is invalid")
	}
	file, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > int64(limit) {
		return nil, errors.New(message)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit || int64(len(data)) != info.Size() {
		return nil, errors.New(message)
	}
	return data, nil
}

func verifyWorkspaceTarget(root *os.Root, relative string, shouldExist bool, expectedHash string) error {
	info, err := root.Lstat(relative)
	if errors.Is(err, os.ErrNotExist) {
		if shouldExist {
			return errors.New("workspace.write target disappeared after approval")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !shouldExist {
		return errors.New("workspace.write target appeared after approval")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("workspace.write target must remain a regular non-symlink file")
	}
	data, err := readWorkspaceFileLimited(root, relative, workspaceWriteMaxBytes, "workspace.write existing file exceeds per-file limit")
	if err != nil {
		return err
	}
	if workspaceContentSHA256(data) != expectedHash {
		return errors.New("workspace.write target changed after approval; inspect and approve a new write")
	}
	return nil
}

func atomicWorkspaceWriteChecked(root *os.Root, relative string, data []byte, beforeRename func() error, requestedMode ...os.FileMode) error {
	return atomicWorkspaceWriteMode(root, relative, data, beforeRename, false, requestedMode...)
}

func atomicWorkspaceCreateChecked(root *os.Root, relative string, data []byte, beforeRename func() error, requestedMode ...os.FileMode) error {
	return atomicWorkspaceWriteMode(root, relative, data, beforeRename, true, requestedMode...)
}

func atomicWorkspaceWriteMode(root *os.Root, relative string, data []byte, beforeRename func() error, exclusive bool, requestedMode ...os.FileMode) error {
	if root == nil || relative == "" || filepath.IsAbs(relative) || len(data) > workspaceWriteMaxBytes {
		return errors.New("atomic workspace write arguments are invalid")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("atomic workspace write path is invalid")
	}
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(clean)))
	if dir != "." && dir != "" {
		if err := root.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	mode := os.FileMode(0o600)
	if len(requestedMode) > 0 {
		mode = requestedMode[0] & 0o700
		if mode&0o600 == 0 {
			mode |= 0o600
		}
	}
	tempName := ".ollama-write-" + uuid.NewString() + ".tmp"
	if dir != "." && dir != "" {
		tempName = filepath.ToSlash(filepath.Join(dir, tempName))
	}
	file, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = root.Remove(tempName)
		}
	}()
	written, err := file.Write(data)
	if err != nil {
		_ = file.Close()
		return err
	}
	if written != len(data) {
		_ = file.Close()
		return io.ErrShortWrite
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if beforeRename != nil {
		if err := beforeRename(); err != nil {
			return err
		}
	}
	if exclusive {
		if err := root.Link(tempName, clean); err != nil {
			return err
		}
		if err := root.Remove(tempName); err != nil {
			_ = root.Remove(clean)
			return err
		}
	} else {
		if err := root.Rename(tempName, clean); err != nil {
			return err
		}
	}
	removeTemp = false
	return nil
}

func workspaceContentSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
