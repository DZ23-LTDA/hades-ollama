package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	workspacePatchMaxFiles      = 16
	workspacePatchMaxTotalBytes = 4 << 20
)

type workspacePatchFile struct {
	input   map[string]any
	path    string
	content []byte
	before  []byte
	existed bool
}

type workspacePatchWriter func(ToolContext, map[string]any) (ToolResult, error)

func prepareWorkspacePatchApproval(workspace string, input map[string]any) (map[string]any, error) {
	if input == nil {
		return nil, errors.New("workspace.patch input is required")
	}
	rawFiles, ok := input["files"].([]any)
	if !ok || len(rawFiles) == 0 || len(rawFiles) > workspacePatchMaxFiles {
		return nil, fmt.Errorf("workspace.patch requires between 1 and %d files", workspacePatchMaxFiles)
	}
	prepared := make([]any, 0, len(rawFiles))
	seen := make(map[string]struct{}, len(rawFiles))
	totalBytes := 0
	for index, raw := range rawFiles {
		fileInput, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workspace.patch file %d must be an object", index)
		}
		path := strings.TrimSpace(stringInput(fileInput, "path", ""))
		content, contentOK := fileInput["content"].(string)
		if path == "" || !contentOK {
			return nil, fmt.Errorf("workspace.patch file %d requires path and string content", index)
		}
		if len(content) > workspaceWriteMaxBytes {
			return nil, fmt.Errorf("workspace.patch file %q exceeds per-file limit", path)
		}
		totalBytes += len(content)
		if totalBytes > workspacePatchMaxTotalBytes {
			return nil, errors.New("workspace.patch total content limit exceeded")
		}
		key := filepath.ToSlash(filepath.Clean(path))
		key = strings.ToLower(key)
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf("workspace.patch contains duplicate path %q", path)
		}
		seen[key] = struct{}{}
		preparedFile, err := prepareWorkspaceWriteApproval(workspace, fileInput)
		if err != nil {
			return nil, fmt.Errorf("prepare patch file %q: %w", path, err)
		}
		prepared = append(prepared, preparedFile)
	}
	result := cloneMap(input)
	result["files"] = prepared
	result["preview_version"] = 1
	return result, nil
}

func writeWorkspacePatch(toolContext ToolContext, input map[string]any) (ToolResult, error) {
	workspaceWriteMu.Lock()
	defer workspaceWriteMu.Unlock()
	var result ToolResult
	err := withWorkspaceToolMutationLock(toolContext, func() error {
		var writeErr error
		result, writeErr = writeWorkspacePatchWith(toolContext, input, func(tc ToolContext, file map[string]any) (ToolResult, error) {
			return writeWorkspaceFileUnlockedWithOptions(tc, file, true)
		})
		return writeErr
	})
	return result, err
}

func writeWorkspacePatchWith(toolContext ToolContext, input map[string]any, writer workspacePatchWriter) (ToolResult, error) {
	if err := validateStepID(toolContext.StepID); err != nil {
		return ToolResult{}, err
	}
	if writer == nil {
		return ToolResult{}, errors.New("workspace.patch writer is unavailable")
	}
	workspace := toolContext.Workspace
	rootPath := workspace
	var err error
	if toolContext.WorkspaceRoot == nil {
		rootPath, err = filepath.Abs(workspace)
		if err != nil {
			return ToolResult{}, err
		}
	}
	rawFiles, ok := input["files"].([]any)
	if !ok || len(rawFiles) == 0 || len(rawFiles) > workspacePatchMaxFiles {
		return ToolResult{}, fmt.Errorf("workspace.patch requires between 1 and %d prepared files", workspacePatchMaxFiles)
	}
	files := make([]workspacePatchFile, 0, len(rawFiles))
	seen := make(map[string]struct{}, len(rawFiles))
	totalBytes := 0
	root := toolContext.WorkspaceRoot
	if root == nil {
		root, err = openArtifactWorkspaceRoot(rootPath)
		if err != nil {
			return ToolResult{}, err
		}
		defer root.Close()
	}

	// Preflight every target before the first mutation: containment, uniqueness,
	// regular-file type, current hash and bounded payload must all be valid.
	for index, raw := range rawFiles {
		fileInput, ok := raw.(map[string]any)
		if !ok {
			return ToolResult{}, fmt.Errorf("workspace.patch prepared file %d is invalid", index)
		}
		path := filepath.ToSlash(strings.TrimSpace(stringInput(fileInput, "path", "")))
		content := []byte(stringInput(fileInput, "content", ""))
		if path == "" || len(content) > workspaceWriteMaxBytes {
			return ToolResult{}, fmt.Errorf("workspace.patch prepared file %d is invalid", index)
		}
		totalBytes += len(content)
		if totalBytes > workspacePatchMaxTotalBytes {
			return ToolResult{}, errors.New("workspace.patch total content limit exceeded")
		}
		key := path
		key = strings.ToLower(key)
		if _, duplicate := seen[key]; duplicate {
			return ToolResult{}, fmt.Errorf("workspace.patch contains duplicate path %q", path)
		}
		seen[key] = struct{}{}
		if toolContext.WorkspaceRoot != nil {
			anchoredPath, pathErr := validateAnchoredWorkspaceWritePath(root, path)
			if pathErr != nil {
				return ToolResult{}, pathErr
			}
			path = anchoredPath
		} else {
			target, pathErr := safeWorkspacePath(rootPath, path)
			if pathErr != nil {
				return ToolResult{}, pathErr
			}
			if pathErr = rejectSymlinkComponents(rootPath, filepath.Dir(target)); pathErr != nil {
				return ToolResult{}, pathErr
			}
		}
		expected := strings.ToLower(strings.TrimSpace(stringInput(fileInput, "expected_sha256", "")))
		var before []byte
		info, statErr := root.Lstat(path)
		existed := statErr == nil
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return ToolResult{}, statErr
		}
		if existed {
			if !info.Mode().IsRegular() {
				return ToolResult{}, fmt.Errorf("workspace.patch target %q must be a regular file", path)
			}
			if len(expected) != 64 {
				return ToolResult{}, fmt.Errorf("workspace.patch target %q is missing its approved base hash", path)
			}
			before, err = readWorkspaceFileLimited(root, path, workspaceWriteMaxBytes, "workspace.patch existing-file limit exceeded")
			if err != nil {
				return ToolResult{}, err
			}
			if workspaceContentSHA256(before) != expected {
				return ToolResult{}, fmt.Errorf("workspace.patch target %q changed after approval; inspect and approve a new patch", path)
			}
		} else if expected != "" {
			return ToolResult{}, fmt.Errorf("workspace.patch target %q appeared/disappeared after approval; inspect and approve a new patch", path)
		}
		files = append(files, workspacePatchFile{input: fileInput, path: path, content: content, before: before, existed: existed})
	}

	applied := make([]int, 0, len(files))
	var artifacts []ArtifactManifest
	results := make([]map[string]any, 0, len(files))
	for index, file := range files {
		result, writeErr := writer(toolContext, file.input)
		if result.Artifacts != nil {
			artifacts = appendUniqueArtifacts(artifacts, result.Artifacts)
		}
		if writeErr != nil {
			rollbackErr := rollbackWorkspacePatch(root, files, applied)
			if rollbackErr != nil {
				return ToolResult{Artifacts: artifacts}, fmt.Errorf("workspace.patch failed for %q (%v); compensation incomplete: %w; inspect the failed target before retrying", file.path, writeErr, rollbackErr)
			}
			artifacts = withoutCompensatedTargetArtifacts(artifacts, files, applied)
			return ToolResult{Artifacts: artifacts}, fmt.Errorf("workspace.patch failed for %q; earlier successful writes were compensated; failed target may have changed and must be inspected before retrying: %w", file.path, writeErr)
		}
		applied = append(applied, index)
		results = append(results, map[string]any{"path": file.path, "bytes": len(file.content), "previous_sha256": func() string {
			if file.existed {
				return workspaceContentSHA256(file.before)
			}
			return ""
		}()})
	}
	return ToolResult{Value: map[string]any{"files": results, "count": len(results), "rollback_on_observed_failure": true, "crash_atomic": false}, Artifacts: artifacts}, nil
}

func patchTargetMatches(root *os.Root, relative string, expected []byte) bool {
	info, err := root.Lstat(relative)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	content, err := readWorkspaceFileLimited(root, relative, workspaceWriteMaxBytes, "workspace.patch target exceeds per-file limit")
	return err == nil && workspaceContentSHA256(content) == workspaceContentSHA256(expected)
}

func rollbackWorkspacePatch(root *os.Root, files []workspacePatchFile, applied []int) error {
	var failures []string
	for index := len(applied) - 1; index >= 0; index-- {
		file := files[applied[index]]
		if !patchTargetMatches(root, file.path, file.content) {
			failures = append(failures, file.path+": changed concurrently; left untouched")
			continue
		}
		if file.existed {
			err := atomicWorkspaceWriteChecked(root, file.path, file.before, func() error {
				return verifyWorkspaceTarget(root, file.path, true, workspaceContentSHA256(file.content))
			})
			if err != nil {
				failures = append(failures, file.path+": "+err.Error())
			}
			continue
		}
		if err := root.Remove(file.path); err != nil {
			failures = append(failures, file.path+": "+err.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func withoutCompensatedTargetArtifacts(artifacts []ArtifactManifest, files []workspacePatchFile, applied []int) []ArtifactManifest {
	compensated := make(map[string]struct{}, len(applied))
	for _, index := range applied {
		if index >= 0 && index < len(files) {
			compensated[filepath.ToSlash(files[index].path)] = struct{}{}
		}
	}
	filtered := artifacts[:0]
	for _, artifact := range artifacts {
		if _, ok := compensated[filepath.ToSlash(filepath.Clean(artifact.Path))]; ok {
			continue
		}
		filtered = append(filtered, artifact)
	}
	return filtered
}
