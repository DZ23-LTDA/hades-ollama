//go:build linux

package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const workspaceSnapshotMaxRepositoryAncestors = 256

func openWorkspaceSnapshotRepositoryRoot(ctx context.Context, requestedRoot string) (string, *os.Root, error) {
	if ctx == nil {
		return "", nil, errors.New("snapshot context is required")
	}
	canonical, err := canonicalExistingDirectory(requestedRoot)
	if err != nil {
		return "", nil, err
	}
	currentPath := filepath.Clean(canonical)
	pathInfo, err := os.Lstat(currentPath)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.IsDir() {
		return "", nil, errors.New("snapshot source root must be a real directory")
	}
	currentRoot, err := os.OpenRoot(currentPath)
	if err != nil {
		return "", nil, err
	}
	currentFile, err := currentRoot.Open(".")
	if err != nil {
		_ = currentRoot.Close()
		return "", nil, err
	}
	currentInfo, err := currentFile.Stat()
	if err != nil {
		_ = currentFile.Close()
		_ = currentRoot.Close()
		return "", nil, err
	}
	openedPathInfo, err := os.Lstat(currentPath)
	if err != nil || openedPathInfo.Mode()&os.ModeSymlink != 0 || !openedPathInfo.IsDir() || !os.SameFile(pathInfo, currentInfo) || !os.SameFile(currentInfo, openedPathInfo) {
		_ = currentFile.Close()
		_ = currentRoot.Close()
		return "", nil, ErrWorkspaceSnapshotChanged
	}

	for depth := 0; ; depth++ {
		if err := ctx.Err(); err != nil {
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, err
		}
		if depth == 0 && workspaceSnapshotRepositoryDiscoveryHook != nil {
			workspaceSnapshotRepositoryDiscoveryHook(currentPath)
		}
		gitInfo, statErr := currentRoot.Lstat(".git")
		if statErr == nil {
			if !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
				_ = currentFile.Close()
				_ = currentRoot.Close()
				return "", nil, errors.New("workspace snapshots require a real .git directory; linked worktrees are unsupported")
			}
			if err := validateGitConfigForInspection(currentRoot); err != nil {
				_ = currentFile.Close()
				_ = currentRoot.Close()
				return "", nil, fmt.Errorf("unsafe Git configuration: %w", err)
			}
			if err := validateGitMetadataTree(currentRoot); err != nil {
				_ = currentFile.Close()
				_ = currentRoot.Close()
				return "", nil, fmt.Errorf("unsafe Git metadata: %w", err)
			}
			if err := verifyWorkspaceSnapshotRootIdentity(currentPath, currentRoot); err != nil {
				_ = currentFile.Close()
				_ = currentRoot.Close()
				return "", nil, err
			}
			_ = currentFile.Close()
			return currentPath, currentRoot, nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, statErr
		}
		if depth >= workspaceSnapshotMaxRepositoryAncestors {
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, errors.New("workspace repository ancestor limit exceeded")
		}
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, errors.New("workspace is not inside a supported Git repository")
		}
		parentFD, err := syscall.Openat(int(currentFile.Fd()), "..", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
		if err != nil {
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, fmt.Errorf("open workspace parent by descriptor: %w", err)
		}
		parentFile := os.NewFile(uintptr(parentFD), "workspace-parent")
		parentInfo, err := parentFile.Stat()
		if err != nil {
			_ = parentFile.Close()
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, err
		}
		parentPathInfo, err := os.Lstat(parentPath)
		if err != nil || parentPathInfo.Mode()&os.ModeSymlink != 0 || !parentPathInfo.IsDir() || !os.SameFile(parentInfo, parentPathInfo) {
			_ = parentFile.Close()
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, ErrWorkspaceSnapshotChanged
		}
		parentRoot, err := os.OpenRoot(fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), parentFile.Fd()))
		if err != nil {
			_ = parentFile.Close()
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, fmt.Errorf("open descriptor-pinned workspace parent: %w", err)
		}
		openedParentInfo, err := parentRoot.Stat(".")
		if err != nil || !os.SameFile(parentInfo, openedParentInfo) {
			_ = parentRoot.Close()
			_ = parentFile.Close()
			_ = currentFile.Close()
			_ = currentRoot.Close()
			return "", nil, ErrWorkspaceSnapshotChanged
		}
		_ = currentFile.Close()
		_ = currentRoot.Close()
		currentPath = parentPath
		currentFile = parentFile
		currentRoot = parentRoot
	}
}
