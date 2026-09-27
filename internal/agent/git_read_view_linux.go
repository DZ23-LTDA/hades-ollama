//go:build linux

package agent

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	gitReadViewMaxEntries = 100_000
	gitReadViewMaxFile    = int64(2 << 30)
	gitReadViewMaxBytes   = int64(4 << 30)
)

// gitReadView is a private snapshot of only the Git metadata required by
// read-only inspection. Repository-local config, hooks, and worktree pointers
// are intentionally never copied into this view.
type gitReadView struct {
	directory string
	gitDir    string
}

func (v *gitReadView) Close() error {
	if v == nil || v.directory == "" {
		return nil
	}
	return os.RemoveAll(v.directory)
}

func newGitReadView(source *os.Root) (*gitReadView, error) {
	if source == nil {
		return nil, errors.New("Git workspace root handle is required")
	}
	if err := validateGitMetadataTree(source); err != nil {
		return nil, fmt.Errorf("unsafe Git metadata: %w", err)
	}
	if err := validateGitConfigForInspection(source); err != nil {
		return nil, fmt.Errorf("unsafe Git configuration: %w", err)
	}
	directory, err := os.MkdirTemp("", "ollama-git-read-view-")
	if err != nil {
		return nil, fmt.Errorf("create private Git read view: %w", err)
	}
	view := &gitReadView{directory: directory, gitDir: filepath.Join(directory, ".git")}
	cleanup := true
	defer func() {
		if cleanup {
			_ = view.Close()
		}
	}()
	viewRoot, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer viewRoot.Close()
	if err := viewRoot.Mkdir(".git", 0o700); err != nil {
		return nil, err
	}
	state := gitReadViewCopyState{}
	for _, relative := range []string{"HEAD", "index", "packed-refs", "shallow", "objects", "refs", "info/exclude"} {
		if err := copyGitReadViewPath(source, viewRoot, ".git/"+relative, &state); err != nil {
			return nil, err
		}
	}
	if state.entries == 0 {
		return nil, errors.New("Git read view contains no repository metadata")
	}
	cleanup = false
	return view, nil
}

type gitReadViewCopyState struct {
	entries int
	bytes   int64
}

func copyGitReadViewPath(source, destination *os.Root, sourcePath string, state *gitReadViewCopyState) error {
	if isGitReadViewPointerPath(sourcePath) {
		if _, err := source.Lstat(sourcePath); err == nil {
			return fmt.Errorf("Git metadata pointer %q is not supported in a private read view", sourcePath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	info, err := source.Lstat(sourcePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return fmt.Errorf("Git read view source %q is not a regular file or directory", sourcePath)
	}
	if info.IsDir() {
		return fs.WalkDir(source.FS(), sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if isGitReadViewPointerPath(path) {
				return fmt.Errorf("Git metadata pointer %q is not supported in a private read view", path)
			}
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if entryInfo.Mode()&os.ModeSymlink != 0 || (!entryInfo.IsDir() && !entryInfo.Mode().IsRegular()) {
				return fmt.Errorf("Git read view entry %q is not a regular file or directory", path)
			}
			if entry.IsDir() {
				if path == sourcePath {
					return nil
				}
				target := strings.TrimPrefix(path, ".git/")
				return destination.MkdirAll(filepath.ToSlash(filepath.Join(".git", target)), 0o700)
			}
			return copyGitReadViewFile(source, destination, path, entryInfo, state)
		})
	}
	return copyGitReadViewFile(source, destination, sourcePath, info, state)
}

func isGitReadViewPointerPath(path string) bool {
	path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	switch path {
	case ".git/commondir", ".git/objects/info/alternates", ".git/objects/info/http-alternates":
		return true
	default:
		return false
	}
}

func copyGitReadViewFile(source, destination *os.Root, sourcePath string, expectedInfo os.FileInfo, state *gitReadViewCopyState) error {
	if expectedInfo.Size() < 0 || expectedInfo.Size() > gitReadViewMaxFile || expectedInfo.Size() > gitReadViewMaxBytes-state.bytes {
		return fmt.Errorf("Git read view byte limit exceeded at %q", sourcePath)
	}
	state.entries++
	if state.entries > gitReadViewMaxEntries {
		return errors.New("Git read view entry limit exceeded")
	}
	input, err := source.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	openedInfo, err := input.Stat()
	if err != nil || !os.SameFile(expectedInfo, openedInfo) || !openedInfo.Mode().IsRegular() || openedInfo.Size() != expectedInfo.Size() {
		return fmt.Errorf("Git read view source %q changed while opening", sourcePath)
	}
	destinationPath := filepath.ToSlash(filepath.Join(".git", strings.TrimPrefix(sourcePath, ".git/")))
	if err := destination.MkdirAll(filepath.Dir(destinationPath), 0o700); err != nil {
		return err
	}
	output, err := destination.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, expectedInfo.Size()+1))
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != expectedInfo.Size() {
		return fmt.Errorf("Git read view source %q changed while copying", sourcePath)
	}
	state.bytes += written
	return nil
}
