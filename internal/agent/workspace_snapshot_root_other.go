//go:build !linux

package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const workspaceSnapshotMaxRepositoryAncestors = 256

// openWorkspaceSnapshotRepositoryRoot uses canonical, non-symlink paths on
// platforms without Linux's /proc descriptor namespace. The returned os.Root
// still makes all subsequent reads relative and rejects symlinked metadata;
// identity is rechecked before and during snapshot inspection.
func openWorkspaceSnapshotRepositoryRoot(ctx context.Context, requestedRoot string) (string, *os.Root, error) {
	if ctx == nil {
		return "", nil, errors.New("snapshot context is required")
	}
	currentPath, err := canonicalExistingDirectory(requestedRoot)
	if err != nil {
		return "", nil, err
	}
	for depth := 0; ; depth++ {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		if depth == 0 && workspaceSnapshotRepositoryDiscoveryHook != nil {
			workspaceSnapshotRepositoryDiscoveryHook(currentPath)
		}
		info, err := os.Lstat(currentPath)
		if err != nil {
			return "", nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", nil, errors.New("snapshot source root must be a real directory")
		}
		root, err := os.OpenRoot(currentPath)
		if err != nil {
			return "", nil, err
		}
		opened, statErr := root.Stat(".")
		if statErr != nil || !os.SameFile(info, opened) {
			_ = root.Close()
			if statErr != nil {
				return "", nil, statErr
			}
			return "", nil, ErrWorkspaceSnapshotChanged
		}
		gitInfo, gitErr := root.Lstat(".git")
		if gitErr == nil {
			if !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
				_ = root.Close()
				return "", nil, errors.New("workspace snapshots require a real .git directory; linked worktrees are unsupported")
			}
			if err := validateGitConfigForInspection(root); err != nil {
				_ = root.Close()
				return "", nil, fmt.Errorf("unsafe Git configuration: %w", err)
			}
			if err := validateGitMetadataTree(root); err != nil {
				_ = root.Close()
				return "", nil, fmt.Errorf("unsafe Git metadata: %w", err)
			}
			if err := verifyWorkspaceSnapshotRootIdentity(currentPath, root); err != nil {
				_ = root.Close()
				return "", nil, err
			}
			return currentPath, root, nil
		}
		_ = root.Close()
		if !errors.Is(gitErr, os.ErrNotExist) {
			return "", nil, gitErr
		}
		if depth >= workspaceSnapshotMaxRepositoryAncestors {
			return "", nil, errors.New("workspace repository ancestor limit exceeded")
		}
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			return "", nil, errors.New("workspace is not inside a supported Git repository")
		}
		currentPath, err = canonicalExistingDirectory(parent)
		if err != nil {
			return "", nil, err
		}
	}
}
