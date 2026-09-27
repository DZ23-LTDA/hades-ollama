package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	ErrArtifactIntegrity = errors.New("artifact integrity check failed")
	ErrArtifactTooLarge  = errors.New("artifact exceeds the 1 GiB processing limit")
)

const artifactMaxBytes int64 = 1 << 30

var artifactHandoffs sync.Map

func BuildArtifactManifest(workspace, missionID, stepID, name, relativePath string) (ArtifactManifest, error) {
	if strings.TrimSpace(workspace) == "" {
		return ArtifactManifest{}, errors.New("workspace is required")
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return ArtifactManifest{}, err
	}
	candidate, err := filepath.Abs(filepath.Join(root, relativePath))
	if err != nil {
		return ArtifactManifest{}, err
	}
	if !isWithin(root, candidate) {
		return ArtifactManifest{}, errors.New("artifact path escapes workspace")
	}
	if err := rejectSymlinkComponents(root, candidate); err != nil {
		return ArtifactManifest{}, err
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return ArtifactManifest{}, err
	}
	workspaceRoot, err := openArtifactWorkspaceRoot(root)
	if err != nil {
		return ArtifactManifest{}, err
	}
	defer workspaceRoot.Close()
	return buildArtifactManifestFromRoot(workspaceRoot, relative, missionID, stepID, name)
}

func buildArtifactManifestFromRoot(workspaceRoot *os.Root, relativePath, missionID, stepID, name string) (ArtifactManifest, error) {
	if workspaceRoot == nil || !filepath.IsLocal(filepath.FromSlash(relativePath)) || filepath.FromSlash(relativePath) == "." {
		return ArtifactManifest{}, errors.New("artifact path must be a local workspace-relative file")
	}
	relative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	file, err := workspaceRoot.Open(filepath.FromSlash(relative))
	if err != nil {
		return ArtifactManifest{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return ArtifactManifest{}, err
	}
	if !info.Mode().IsRegular() {
		return ArtifactManifest{}, errors.New("artifact path must be a regular file")
	}
	if info.Size() < 0 || info.Size() > artifactMaxBytes {
		return ArtifactManifest{}, ErrArtifactTooLarge
	}
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(file, artifactMaxBytes+1))
	if err != nil {
		return ArtifactManifest{}, err
	}
	if read > artifactMaxBytes {
		return ArtifactManifest{}, ErrArtifactTooLarge
	}
	if read != info.Size() {
		return ArtifactManifest{}, fmt.Errorf("artifact changed while its manifest was being built")
	}
	mediaType := mime.TypeByExtension(filepath.Ext(relative))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(relative)
	}
	return ArtifactManifest{
		ID:        fmt.Sprintf("art_%d", time.Now().UnixNano()),
		MissionID: missionID,
		StepID:    stepID,
		Name:      name,
		Path:      filepath.ToSlash(relative),
		MediaType: mediaType,
		Size:      info.Size(),
		SHA256:    hex.EncodeToString(hash.Sum(nil)),
		CreatedAt: time.Now().UTC(),
	}, nil
}

func createVerifiedArtifactSnapshot(workspace, relativePath string, manifest ArtifactManifest) (string, error) {
	workspaceRoot, err := openArtifactWorkspaceRoot(workspace)
	if err != nil {
		return "", fmt.Errorf("%w: open workspace root: %v", ErrArtifactIntegrity, err)
	}
	defer workspaceRoot.Close()
	return createVerifiedArtifactSnapshotFromRoot(workspaceRoot, relativePath, manifest)
}

func createVerifiedArtifactSnapshotFromRoot(workspaceRoot *os.Root, relativePath string, manifest ArtifactManifest) (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("%w: descriptor-backed artifact delivery is unsupported on this platform", ErrArtifactIntegrity)
	}
	if workspaceRoot == nil || !filepath.IsLocal(filepath.FromSlash(relativePath)) {
		return "", fmt.Errorf("%w: artifact path is not workspace-relative", ErrArtifactIntegrity)
	}
	file, err := workspaceRoot.Open(filepath.FromSlash(relativePath))
	if err != nil {
		return "", fmt.Errorf("%w: open artifact: %v", ErrArtifactIntegrity, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return "", fmt.Errorf("%w: stat artifact: %v", ErrArtifactIntegrity, err)
	}
	if !info.Mode().IsRegular() || info.Size() != manifest.Size {
		_ = file.Close()
		return "", fmt.Errorf("%w: artifact size or file type changed", ErrArtifactIntegrity)
	}
	if manifest.Size < 0 || manifest.Size > artifactMaxBytes {
		_ = file.Close()
		return "", fmt.Errorf("%w: %w", ErrArtifactIntegrity, ErrArtifactTooLarge)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(hash, io.LimitReader(file, artifactMaxBytes+1))
	if copyErr != nil {
		_ = file.Close()
		return "", fmt.Errorf("%w: read artifact: %v", ErrArtifactIntegrity, copyErr)
	}
	if written > artifactMaxBytes {
		_ = file.Close()
		return "", fmt.Errorf("%w: %w", ErrArtifactIntegrity, ErrArtifactTooLarge)
	}
	if written != manifest.Size || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimSpace(manifest.SHA256)) {
		_ = file.Close()
		return "", fmt.Errorf("%w: artifact size or content hash changed", ErrArtifactIntegrity)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("%w: rewind verified artifact: %v", ErrArtifactIntegrity, err)
	}
	handoffPath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), file.Fd())
	artifactHandoffs.Store(handoffPath, file)
	return handoffPath, nil
}

// CloseArtifactSnapshot releases the descriptor retained for an artifact
// delivery handoff. The server must call this after serving the same open
// descriptor whose content was verified.
func CloseArtifactSnapshot(path string) error {
	if value, ok := artifactHandoffs.LoadAndDelete(path); ok {
		return value.(*os.File).Close()
	}
	return nil
}

// OpenArtifactSnapshot returns the exact descriptor that was hashed by
// createVerifiedArtifactSnapshotFromRoot. Callers must close it via
// CloseArtifactSnapshot after serving the response.
func OpenArtifactSnapshot(path string) (*os.File, error) {
	value, ok := artifactHandoffs.Load(path)
	if !ok {
		return nil, fmt.Errorf("%w: artifact handoff is unavailable", ErrArtifactIntegrity)
	}
	file, ok := value.(*os.File)
	if !ok || file == nil {
		return nil, fmt.Errorf("%w: artifact handoff is invalid", ErrArtifactIntegrity)
	}
	return file, nil
}

func openArtifactWorkspaceRoot(workspace string) (*os.Root, error) {
	rootPath, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	before, err := os.Lstat(rootPath)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, errors.New("artifact workspace root must be a non-symlink directory")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	opened, err := root.Stat(".")
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	if !os.SameFile(before, opened) {
		_ = root.Close()
		return nil, errors.New("artifact workspace root changed while opening")
	}
	return root, nil
}

func isWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
