package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	workspaceSnapshotSchemaVersion = 1
	workspaceSnapshotGitOutputMax  = 32 << 20
	workspaceSnapshotIndexMax      = 32 << 20
	workspaceSnapshotDefaultFiles  = 10_000
	workspaceSnapshotDefaultDepth  = 64
	workspaceSnapshotDefaultFile   = int64(32 << 20)
	workspaceSnapshotDefaultTotal  = int64(512 << 20)
	workspaceSnapshotOrphanGrace   = 24 * time.Hour
	workspaceSnapshotRetention     = 30 * 24 * time.Hour
)

var (
	ErrWorkspaceSnapshotLimit   = errors.New("workspace snapshot resource limit exceeded")
	ErrWorkspaceSnapshotChanged = errors.New("workspace changed while snapshot was being created")
	snapshotCredentialPattern   = regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?token|secret|password|private[_ -]?key)\s*[:=]\s*["']?[a-z0-9_./+=-]{12,}`)
	snapshotPrivateKeyPattern   = regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |PGP )?PRIVATE KEY-----`)

	// workspaceSnapshotGitCommandHook is set only by package tests to orchestrate
	// a source pathname replacement at a precise Git command boundary.
	workspaceSnapshotGitCommandHook          func(string, []string)
	workspaceSnapshotRepositoryDiscoveryHook func(string)
	// workspaceSnapshotAfterFileReadHook deterministically mutates a test fixture between reads.
	workspaceSnapshotAfterFileReadHook func(string)
)

type WorkspaceSnapshotRequest struct {
	SourceRoot              string
	ExpectedProjectRoot     string
	ExpectedProjectRootInfo os.FileInfo
	DataRoot                string
	MissionID               string
	OrganizationID          string
	ProjectID               string
	MaxFiles                int
	MaxDepth                int
	MaxFileBytes            int64
	MaxTotalBytes           int64
}

type WorkspaceSnapshotIndexEntry struct {
	Path     string `json:"path"`
	Mode     uint32 `json:"mode"`
	ObjectID string `json:"object_id"`
	Stage    int    `json:"stage"`
}

type WorkspaceSnapshotFile struct {
	Path         string                        `json:"path"`
	Source       string                        `json:"source"`
	Kind         string                        `json:"kind"`
	OriginalMode uint32                        `json:"original_mode,omitempty"`
	Present      bool                          `json:"present"`
	Included     bool                          `json:"included"`
	Size         int64                         `json:"size,omitempty"`
	SHA256       string                        `json:"sha256,omitempty"`
	SkipReason   string                        `json:"skip_reason,omitempty"`
	Index        []WorkspaceSnapshotIndexEntry `json:"index,omitempty"`
}

type WorkspaceSnapshotManifest struct {
	SchemaVersion  int                           `json:"schema_version"`
	SnapshotID     string                        `json:"snapshot_id"`
	MissionID      string                        `json:"mission_id"`
	OrganizationID string                        `json:"organization_id,omitempty"`
	ProjectID      string                        `json:"project_id,omitempty"`
	SourceRoot     string                        `json:"source_root"`
	CreatedAt      time.Time                     `json:"created_at"`
	GitHead        string                        `json:"git_head,omitempty"`
	GitBranch      string                        `json:"git_branch,omitempty"`
	GitIndexSHA256 string                        `json:"git_index_sha256,omitempty"`
	GitStatus      []string                      `json:"git_status"`
	IndexEntries   []WorkspaceSnapshotIndexEntry `json:"index_entries"`
	Files          []WorkspaceSnapshotFile       `json:"files"`
	CopiedFiles    int                           `json:"copied_files"`
	TotalBytes     int64                         `json:"total_bytes"`
	Policy         string                        `json:"policy"`
	SnapshotRoot   string                        `json:"-"`
	TreeRoot       string                        `json:"-"`
}

// CreateWorkspaceSnapshot copies the current non-ignored Git working tree into a
// private DataRoot-owned directory. It never changes the source tree or Git index.
// Ignored files, symlinks, submodules, and credential-looking files are omitted.
func CreateWorkspaceSnapshot(ctx context.Context, request WorkspaceSnapshotRequest) (WorkspaceSnapshotManifest, error) {
	manifest, handles, err := createWorkspaceSnapshotWithHandles(ctx, request)
	if handles != nil {
		_ = handles.Close()
	}
	return manifest, err
}

type workspaceSnapshotHandles struct {
	dataRoot       *os.Root
	dataRootPath   string
	snapshotRoot   *os.Root
	treeRoot       *os.Root
	snapshotRel    string
	createdParents []string
}

func (h *workspaceSnapshotHandles) Close() error {
	if h == nil {
		return nil
	}
	var errs []error
	for _, root := range []*os.Root{h.treeRoot, h.snapshotRoot, h.dataRoot} {
		if root != nil {
			errs = append(errs, root.Close())
		}
	}
	return errors.Join(errs...)
}

func (h *workspaceSnapshotHandles) Remove() error {
	if h == nil || h.dataRoot == nil || h.snapshotRel == "" {
		return errors.New("snapshot cleanup handle is incomplete")
	}
	removeErr := h.dataRoot.RemoveAll(h.snapshotRel)
	parentsErr := removeEmptyWorkspaceSnapshotParents(h.dataRoot, h.createdParents)
	return errors.Join(removeErr, parentsErr)
}

func prepareWorkspaceSnapshotHandoff(manifest WorkspaceSnapshotManifest, handles *workspaceSnapshotHandles, relativeWorkspace string) (string, string, error) {
	if handles == nil || handles.dataRoot == nil || handles.snapshotRoot == nil || handles.treeRoot == nil {
		return "", "", errors.New("workspace snapshot handoff handles are incomplete")
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(manifest.SnapshotRoot, handles.snapshotRoot); err != nil {
		return "", "", err
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(manifest.TreeRoot, handles.treeRoot); err != nil {
		return "", "", err
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(handles.dataRootPath, handles.dataRoot); err != nil {
		return "", "", err
	}
	workspace := manifest.TreeRoot
	if relativeWorkspace != "" && relativeWorkspace != "." {
		if !filepath.IsLocal(relativeWorkspace) || filepath.IsAbs(relativeWorkspace) {
			return "", "", errors.New("isolated project workspace escapes snapshot")
		}
		workspace = filepath.Join(manifest.TreeRoot, relativeWorkspace)
		if !isWithin(manifest.TreeRoot, workspace) {
			return "", "", errors.New("isolated project workspace escapes snapshot")
		}
		workspaceRoot, err := openOrCreateWorkspaceSnapshotSubdirectory(handles.treeRoot, manifest.TreeRoot, relativeWorkspace)
		if err != nil {
			return "", "", fmt.Errorf("create isolated project subdirectory: %w", err)
		}
		if err := workspaceRoot.Close(); err != nil {
			return "", "", fmt.Errorf("close isolated project subdirectory: %w", err)
		}
	}
	manifestFile, err := handles.snapshotRoot.Open("manifest.json")
	if err != nil {
		return "", "", fmt.Errorf("open workspace snapshot manifest: %w", err)
	}
	info, statErr := manifestFile.Stat()
	if statErr != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > 64<<20 {
		_ = manifestFile.Close()
		if statErr != nil {
			return "", "", statErr
		}
		return "", "", errors.New("workspace snapshot manifest is unsafe or exceeds the size limit")
	}
	manifestBytes, readErr := io.ReadAll(io.LimitReader(manifestFile, 64<<20+1))
	closeErr := manifestFile.Close()
	if readErr != nil {
		return "", "", readErr
	}
	if closeErr != nil {
		return "", "", closeErr
	}
	if len(manifestBytes) > 64<<20 {
		return "", "", errors.New("workspace snapshot manifest exceeds the size limit")
	}
	digest := sha256.Sum256(manifestBytes)
	return workspace, hex.EncodeToString(digest[:]), nil
}

func createWorkspaceSnapshotWithHandles(ctx context.Context, request WorkspaceSnapshotRequest) (WorkspaceSnapshotManifest, *workspaceSnapshotHandles, error) {
	if ctx == nil {
		return WorkspaceSnapshotManifest{}, nil, errors.New("snapshot context is required")
	}
	if !validSnapshotID(request.MissionID) {
		return WorkspaceSnapshotManifest{}, nil, errors.New("valid mission ID is required")
	}
	if request.OrganizationID != "" && (!utf8.ValidString(request.OrganizationID) || len(request.OrganizationID) > 256 || strings.ContainsRune(request.OrganizationID, '\x00')) {
		return WorkspaceSnapshotManifest{}, nil, errors.New("organization ID is invalid")
	}
	if request.ProjectID != "" && (!utf8.ValidString(request.ProjectID) || len(request.ProjectID) > 256 || strings.ContainsRune(request.ProjectID, '\x00')) {
		return WorkspaceSnapshotManifest{}, nil, errors.New("project ID is invalid")
	}
	limits, err := normalizeWorkspaceSnapshotLimits(request)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}

	sourceRoot, sourceHandle, err := openWorkspaceSnapshotRepositoryRoot(ctx, request.SourceRoot)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("resolve snapshot Git repository root: %w", err)
	}
	defer sourceHandle.Close()
	openedSourceInfo, err := sourceHandle.Stat(".")
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	sourceDirectory, err := sourceHandle.Open(".")
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("open snapshot source directory: %w", err)
	}
	defer sourceDirectory.Close()
	openedDirectoryInfo, err := sourceDirectory.Stat()
	if err != nil || !os.SameFile(openedSourceInfo, openedDirectoryInfo) {
		return WorkspaceSnapshotManifest{}, nil, ErrWorkspaceSnapshotChanged
	}
	if err := validateGitMetadataTree(sourceHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("unsafe Git metadata: %w", err)
	}
	if err := verifyWorkspaceSnapshotRootIdentity(sourceRoot, sourceHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if err := verifyExpectedWorkspaceProjectRoot(sourceRoot, request.ExpectedProjectRoot, request.ExpectedProjectRootInfo); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	gitView, err := newGitReadView(sourceHandle)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	defer gitView.Close()
	dataRoot := strings.TrimSpace(request.DataRoot)
	if dataRoot == "" {
		return WorkspaceSnapshotManifest{}, nil, errors.New("snapshot data root is required")
	}
	dataRoot, err = canonicalPathAllowMissing(dataRoot)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("resolve snapshot data root: %w", err)
	}
	if sameOrWithin(sourceRoot, dataRoot) || sameOrWithin(dataRoot, sourceRoot) {
		return WorkspaceSnapshotManifest{}, nil, errors.New("snapshot data root must be separate from the source workspace")
	}
	gitInfo, err := sourceHandle.Lstat(".git")
	if err != nil || !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
		return WorkspaceSnapshotManifest{}, nil, errors.New("snapshot source must be a Git repository root with a real .git directory")
	}

	before, err := readWorkspaceSnapshotGitState(ctx, sourceRoot, sourceHandle, sourceDirectory, gitView)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if err := verifyWorkspaceSnapshotRootIdentity(sourceRoot, sourceHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if !samePath(before.RepositoryRoot, sourceRoot) {
		return WorkspaceSnapshotManifest{}, nil, errors.New("snapshot source must exactly match the Git repository root")
	}
	dataRootHandle, err := openOrCreateWorkspaceSnapshotDirectoryRoot(dataRoot)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("open snapshot data root: %w", err)
	}
	keepDataRootHandle := false
	defer func() {
		if !keepDataRootHandle {
			_ = dataRootHandle.Close()
		}
	}()
	if err := secureWorkspaceSnapshotDirectory(dataRoot, dataRootHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("secure snapshot data root: %w", err)
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(dataRoot, dataRootHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}

	snapshotID := "snp_" + uuid.NewString()
	tenantDigest := sha256.Sum256([]byte(request.OrganizationID))
	tenantKey := hex.EncodeToString(tenantDigest[:12])
	snapshotRoot, snapshotRelative, snapshotRootHandle, createdParents, err := createWorkspaceSnapshotDirectory(dataRootHandle, dataRoot, tenantKey, request.MissionID, snapshotID)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	keepSnapshotRootHandle := false
	defer func() {
		if !keepSnapshotRootHandle {
			_ = snapshotRootHandle.Close()
		}
	}()
	completed := false
	defer func() {
		if !completed {
			_ = dataRootHandle.RemoveAll(snapshotRelative)
			_ = removeEmptyWorkspaceSnapshotParents(dataRootHandle, createdParents)
		}
	}()
	treeRoot := filepath.Join(snapshotRoot, "tree")
	treeRootHandle, err := createWorkspaceSnapshotChildRoot(snapshotRootHandle, snapshotRoot, "tree", 0o700)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("create snapshot tree: %w", err)
	}
	keepTreeRootHandle := false
	defer func() {
		if !keepTreeRootHandle {
			_ = treeRootHandle.Close()
		}
	}()

	manifest := WorkspaceSnapshotManifest{
		SchemaVersion:  workspaceSnapshotSchemaVersion,
		SnapshotID:     snapshotID,
		MissionID:      request.MissionID,
		OrganizationID: request.OrganizationID,
		ProjectID:      request.ProjectID,
		SourceRoot:     sourceRoot,
		CreatedAt:      time.Now().UTC(),
		GitHead:        before.Head,
		GitBranch:      before.Branch,
		GitIndexSHA256: before.IndexSHA256,
		GitStatus:      before.Status,
		IndexEntries:   before.IndexEntries,
		Files:          make([]WorkspaceSnapshotFile, 0, len(before.Paths)),
		Policy:         "tracked-working-tree-plus-nonignored-untracked; ignored/symlink/submodule/credential-like files excluded; source read-only",
		SnapshotRoot:   snapshotRoot,
		TreeRoot:       treeRoot,
	}
	var totalBytes int64
	for _, sourcePath := range before.Paths {
		if err := ctx.Err(); err != nil {
			return WorkspaceSnapshotManifest{}, nil, err
		}
		if len(manifest.Files) >= limits.maxFiles {
			return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("%w: file count exceeds %d", ErrWorkspaceSnapshotLimit, limits.maxFiles)
		}
		if err := validateWorkspaceSnapshotPath(sourcePath, limits.maxDepth); err != nil {
			return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("unsafe Git path %q: %w", sourcePath, err)
		}
		entry := WorkspaceSnapshotFile{Path: sourcePath, Source: before.SourceByPath[sourcePath], Kind: "file", Present: false, Index: before.IndexByPath[sourcePath]}
		if err := copyWorkspaceSnapshotFile(ctx, sourceRoot, sourceHandle, treeRootHandle, treeRoot, sourcePath, before.ModeByPath[sourcePath], limits, &totalBytes, &entry); err != nil {
			return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("snapshot %q: %w", sourcePath, err)
		}
		if entry.Included {
			manifest.CopiedFiles++
			manifest.TotalBytes += entry.Size
		}
		manifest.Files = append(manifest.Files, entry)
	}

	after, err := readWorkspaceSnapshotGitState(ctx, sourceRoot, sourceHandle, sourceDirectory, gitView)
	if err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if err := verifyWorkspaceSnapshotRootIdentity(sourceRoot, sourceHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if request.ExpectedProjectRoot != "" {
		projectRoot, projectErr := canonicalExistingDirectory(request.ExpectedProjectRoot)
		if projectErr != nil || !isWithin(sourceRoot, projectRoot) {
			return WorkspaceSnapshotManifest{}, nil, errors.New("authorized project root is outside the snapshot source repository")
		}
		projectInfo, statErr := os.Lstat(projectRoot)
		if statErr != nil || projectInfo.Mode()&os.ModeSymlink != 0 || !projectInfo.IsDir() || request.ExpectedProjectRootInfo == nil || !os.SameFile(request.ExpectedProjectRootInfo, projectInfo) {
			return WorkspaceSnapshotManifest{}, nil, ErrWorkspaceSnapshotChanged
		}
	}
	if !sameWorkspaceSnapshotGitState(before, after) {
		return WorkspaceSnapshotManifest{}, nil, ErrWorkspaceSnapshotChanged
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(snapshotRoot, snapshotRootHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if err := verifyWorkspaceSnapshotDirectoryIdentity(treeRoot, treeRootHandle); err != nil {
		return WorkspaceSnapshotManifest{}, nil, err
	}
	if request.ExpectedProjectRoot != "" {
		projectRoot, projectErr := canonicalExistingDirectory(request.ExpectedProjectRoot)
		if projectErr != nil {
			return WorkspaceSnapshotManifest{}, nil, ErrWorkspaceSnapshotChanged
		}
		projectInfo, statErr := os.Lstat(projectRoot)
		if statErr != nil || request.ExpectedProjectRootInfo == nil || !os.SameFile(request.ExpectedProjectRootInfo, projectInfo) {
			return WorkspaceSnapshotManifest{}, nil, ErrWorkspaceSnapshotChanged
		}
	}
	if err := writeJSONAtomicWorkspaceRoot(snapshotRootHandle, "manifest.json", manifest); err != nil {
		return WorkspaceSnapshotManifest{}, nil, fmt.Errorf("write workspace snapshot manifest: %w", err)
	}
	completed = true
	keepDataRootHandle = true
	keepSnapshotRootHandle = true
	keepTreeRootHandle = true
	return manifest, &workspaceSnapshotHandles{dataRoot: dataRootHandle, dataRootPath: dataRoot, snapshotRoot: snapshotRootHandle, treeRoot: treeRootHandle, snapshotRel: snapshotRelative, createdParents: createdParents}, nil
}

func resolveWorkspaceSnapshotRepositoryRoot(ctx context.Context, requestedRoot string) (string, error) {
	root, handle, err := openWorkspaceSnapshotRepositoryRoot(ctx, requestedRoot)
	if err != nil {
		return "", err
	}
	if err := handle.Close(); err != nil {
		return "", err
	}
	return root, nil
}

func verifyExpectedWorkspaceProjectRoot(repositoryRoot, expectedProjectRoot string, expectedInfo os.FileInfo) error {
	if expectedProjectRoot == "" {
		return nil
	}
	if expectedInfo == nil {
		return ErrWorkspaceSnapshotChanged
	}
	projectRoot, err := canonicalExistingDirectory(expectedProjectRoot)
	if err != nil || !isWithin(repositoryRoot, projectRoot) {
		return errors.New("authorized project root is outside the snapshot source repository")
	}
	if err := rejectSymlinkComponents(repositoryRoot, projectRoot); err != nil {
		return fmt.Errorf("unsafe authorized project path: %w", err)
	}
	projectHandle, err := os.OpenRoot(projectRoot)
	if err != nil {
		return ErrWorkspaceSnapshotChanged
	}
	defer projectHandle.Close()
	openedInfo, err := projectHandle.Stat(".")
	if err != nil || !os.SameFile(expectedInfo, openedInfo) {
		return ErrWorkspaceSnapshotChanged
	}
	pathInfo, err := os.Lstat(projectRoot)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.IsDir() || !os.SameFile(openedInfo, pathInfo) {
		return ErrWorkspaceSnapshotChanged
	}
	return nil
}

// verifyWorkspaceSnapshotRootIdentity ensures the canonical path still names the
// directory pinned by OpenRoot. Git subprocesses use the path, while file reads
// use the handle; this prevents those two views from diverging after replacement.
func verifyWorkspaceSnapshotRootIdentity(path string, root *os.Root) error {
	if root == nil {
		return errors.New("snapshot source root handle is required")
	}
	opened, err := root.Stat(".")
	if err != nil {
		return fmt.Errorf("stat opened snapshot source root: %w", err)
	}
	current, err := os.Lstat(path)
	if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, current) {
		return ErrWorkspaceSnapshotChanged
	}
	return nil
}

// validateMissionWorkspaceSnapshot binds an isolated mission to its tenant-owned
// snapshot and verifies the persisted manifest digest before any execution.
func validateMissionWorkspaceSnapshot(dataRoot string, mission Mission) error {
	if !mission.WorkspaceIsolated {
		return nil
	}
	if !validSnapshotID(mission.ID) || !validSnapshotID(mission.WorkspaceSnapshotID) || strings.TrimSpace(mission.OrganizationID) == "" || strings.TrimSpace(mission.ProjectID) == "" {
		return errors.New("isolated mission has incomplete snapshot identity")
	}
	expectedDigest, err := hex.DecodeString(mission.WorkspaceSnapshotSHA256)
	if err != nil || len(expectedDigest) != sha256.Size {
		return errors.New("isolated mission has an invalid snapshot manifest digest")
	}
	tenantDigest := sha256.Sum256([]byte(mission.OrganizationID))
	tenantKey := hex.EncodeToString(tenantDigest[:12])
	base := filepath.Join(dataRoot, ".agent-workspace-snapshots", "snapshots")
	snapshotRoot := filepath.Join(base, tenantKey, mission.ID, mission.WorkspaceSnapshotID)
	if err := rejectSymlinkComponents(base, snapshotRoot); err != nil {
		return fmt.Errorf("unsafe persisted workspace snapshot path: %w", err)
	}
	root, err := os.OpenRoot(snapshotRoot)
	if err != nil {
		return fmt.Errorf("open persisted workspace snapshot: %w", err)
	}
	defer root.Close()
	if err := verifyWorkspaceSnapshotRootIdentity(snapshotRoot, root); err != nil {
		return err
	}
	manifestInfo, err := root.Lstat("manifest.json")
	if err != nil || !manifestInfo.Mode().IsRegular() {
		return errors.New("persisted workspace snapshot manifest is missing or unsafe")
	}
	manifestFile, err := root.Open("manifest.json")
	if err != nil {
		return fmt.Errorf("open persisted workspace snapshot manifest: %w", err)
	}
	openedInfo, statErr := manifestFile.Stat()
	if statErr != nil || !os.SameFile(manifestInfo, openedInfo) {
		_ = manifestFile.Close()
		return errors.New("persisted workspace snapshot manifest changed while opening")
	}
	manifestBytes, readErr := io.ReadAll(io.LimitReader(manifestFile, 64<<20+1))
	closeErr := manifestFile.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if len(manifestBytes) > 64<<20 {
		return errors.New("persisted workspace snapshot manifest exceeds the size limit")
	}
	actualDigest := sha256.Sum256(manifestBytes)
	if !bytes.Equal(expectedDigest, actualDigest[:]) {
		return errors.New("persisted workspace snapshot manifest digest mismatch")
	}
	var manifest WorkspaceSnapshotManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode persisted workspace snapshot manifest: %w", err)
	}
	if manifest.SchemaVersion != workspaceSnapshotSchemaVersion || manifest.SnapshotID != mission.WorkspaceSnapshotID || manifest.MissionID != mission.ID || manifest.OrganizationID != mission.OrganizationID || manifest.ProjectID != mission.ProjectID {
		return errors.New("persisted workspace snapshot identity does not match mission")
	}
	treeRoot := filepath.Join(snapshotRoot, "tree")
	if err := rejectSymlinkComponents(snapshotRoot, treeRoot); err != nil {
		return fmt.Errorf("unsafe persisted workspace tree: %w", err)
	}
	treeInfo, err := os.Lstat(treeRoot)
	if err != nil || !treeInfo.IsDir() || treeInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("persisted workspace snapshot tree is missing or unsafe")
	}
	workspace, err := filepath.Abs(mission.Workspace)
	if err != nil || !isWithin(treeRoot, workspace) {
		return errors.New("isolated mission workspace escapes its snapshot")
	}
	if err := rejectSymlinkComponents(treeRoot, workspace); err != nil {
		return fmt.Errorf("unsafe isolated mission workspace: %w", err)
	}
	workspaceInfo, err := os.Stat(workspace)
	if err != nil || !workspaceInfo.IsDir() {
		return errors.New("isolated mission workspace is missing or not a directory")
	}
	return nil
}

// openMissionWorkspaceSnapshot returns a descriptor anchored at the mission's
// workspace directory. All subsequent workspace tools use this handle rather
// than resolving mission.Workspace again, so a pathname replacement cannot
// redirect an in-flight mission to another tenant's directory.
func openMissionWorkspaceSnapshot(dataRoot string, mission Mission) (*os.Root, error) {
	if !mission.WorkspaceIsolated {
		if strings.TrimSpace(mission.WorkspaceIdentity) == "" {
			return nil, errors.New("mission workspace identity is missing; recreate the mission to reauthorize its workspace")
		}
		root, err := os.OpenRoot(mission.Workspace)
		if err != nil {
			return nil, fmt.Errorf("open mission workspace: %w", err)
		}
		identity, identityErr := workspaceDirectoryIdentity(mission.Workspace)
		if identityErr != nil || identity != mission.WorkspaceIdentity {
			_ = root.Close()
			return nil, errors.New("mission workspace changed after authorization")
		}
		return root, nil
	}
	if err := validateMissionWorkspaceSnapshot(dataRoot, mission); err != nil {
		return nil, err
	}
	tenantDigest := sha256.Sum256([]byte(mission.OrganizationID))
	tenantKey := hex.EncodeToString(tenantDigest[:12])
	base := filepath.Join(dataRoot, ".agent-workspace-snapshots", "snapshots")
	snapshotPath := filepath.Join(base, tenantKey, mission.ID, mission.WorkspaceSnapshotID)
	snapshotRoot, err := os.OpenRoot(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("open isolated snapshot root: %w", err)
	}
	closeSnapshot := true
	defer func() {
		if closeSnapshot {
			_ = snapshotRoot.Close()
		}
	}()
	if err := verifyWorkspaceSnapshotRootIdentity(snapshotPath, snapshotRoot); err != nil {
		return nil, err
	}
	manifestBytes, err := snapshotRoot.ReadFile("manifest.json")
	if err != nil || len(manifestBytes) > 64<<20 {
		if err == nil {
			err = errors.New("persisted workspace snapshot manifest exceeds the size limit")
		}
		return nil, err
	}
	actualDigest := sha256.Sum256(manifestBytes)
	expectedDigest, err := hex.DecodeString(mission.WorkspaceSnapshotSHA256)
	if err != nil || !bytes.Equal(expectedDigest, actualDigest[:]) {
		return nil, errors.New("persisted workspace snapshot manifest digest mismatch")
	}
	var manifest WorkspaceSnapshotManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode persisted workspace snapshot manifest: %w", err)
	}
	if manifest.SchemaVersion != workspaceSnapshotSchemaVersion || manifest.SnapshotID != mission.WorkspaceSnapshotID || manifest.MissionID != mission.ID || manifest.OrganizationID != mission.OrganizationID || manifest.ProjectID != mission.ProjectID {
		return nil, errors.New("persisted workspace snapshot identity does not match mission")
	}
	treeRoot, err := snapshotRoot.OpenRoot("tree")
	if err != nil {
		return nil, fmt.Errorf("open isolated snapshot tree: %w", err)
	}
	workspacePath, err := filepath.Abs(mission.Workspace)
	if err != nil {
		_ = treeRoot.Close()
		return nil, err
	}
	treePath := filepath.Join(snapshotPath, "tree")
	workspaceRelative, err := filepath.Rel(treePath, workspacePath)
	if err != nil || !filepath.IsLocal(workspaceRelative) {
		_ = treeRoot.Close()
		return nil, errors.New("isolated mission workspace escapes its snapshot")
	}
	if err := verifyWorkspaceSnapshotContents(treeRoot, manifest, mission, workspaceRelative); err != nil {
		_ = treeRoot.Close()
		return nil, fmt.Errorf("verify isolated snapshot contents: %w", err)
	}
	workspaceInfo, err := treeRoot.Lstat(filepath.ToSlash(workspaceRelative))
	if err != nil || !workspaceInfo.IsDir() || workspaceInfo.Mode()&os.ModeSymlink != 0 {
		_ = treeRoot.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("isolated mission workspace is missing or unsafe")
	}
	workspaceRoot, err := treeRoot.OpenRoot(filepath.ToSlash(workspaceRelative))
	_ = treeRoot.Close()
	if err != nil {
		return nil, fmt.Errorf("open isolated mission workspace: %w", err)
	}
	openedInfo, err := workspaceRoot.Stat(".")
	if err != nil || !openedInfo.IsDir() || !os.SameFile(workspaceInfo, openedInfo) {
		_ = workspaceRoot.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("isolated mission workspace changed while opening")
	}
	workspaceManifest := manifest
	workspaceManifest.Files = nil
	for _, entry := range manifest.Files {
		entryPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))
		if workspaceRelative == "." {
			workspaceManifest.Files = append(workspaceManifest.Files, entry)
			continue
		}
		if !strings.HasPrefix(entryPath, filepath.ToSlash(filepath.Clean(workspaceRelative))+"/") {
			continue
		}
		entry.Path = strings.TrimPrefix(entryPath, filepath.ToSlash(filepath.Clean(workspaceRelative))+"/")
		workspaceManifest.Files = append(workspaceManifest.Files, entry)
	}
	if err := verifyWorkspaceSnapshotContents(workspaceRoot, workspaceManifest, mission, "."); err != nil {
		_ = workspaceRoot.Close()
		return nil, fmt.Errorf("verify isolated mission workspace contents: %w", err)
	}
	closeSnapshot = false
	_ = snapshotRoot.Close()
	return workspaceRoot, nil
}

// readMissionWorkspaceSnapshotManifest returns Git baseline metadata bound to
// the persisted mission digest. The relative prefix identifies a nested project
// workspace without exposing its host filesystem path.
func readMissionWorkspaceSnapshotManifest(dataRoot string, mission Mission) (WorkspaceSnapshotManifest, string, error) {
	if !mission.WorkspaceIsolated {
		return WorkspaceSnapshotManifest{}, "", errors.New("Git snapshot metadata requires an isolated mission")
	}
	if err := validateMissionWorkspaceSnapshot(dataRoot, mission); err != nil {
		return WorkspaceSnapshotManifest{}, "", err
	}
	tenantDigest := sha256.Sum256([]byte(mission.OrganizationID))
	tenantKey := hex.EncodeToString(tenantDigest[:12])
	snapshotPath := filepath.Join(dataRoot, ".agent-workspace-snapshots", "snapshots", tenantKey, mission.ID, mission.WorkspaceSnapshotID)
	snapshotRoot, err := os.OpenRoot(snapshotPath)
	if err != nil {
		return WorkspaceSnapshotManifest{}, "", fmt.Errorf("open mission snapshot metadata: %w", err)
	}
	defer snapshotRoot.Close()
	if err := verifyWorkspaceSnapshotRootIdentity(snapshotPath, snapshotRoot); err != nil {
		return WorkspaceSnapshotManifest{}, "", err
	}
	manifestBytes, err := snapshotRoot.ReadFile("manifest.json")
	if err != nil || len(manifestBytes) > 64<<20 {
		if err == nil {
			err = errors.New("persisted workspace snapshot manifest exceeds the size limit")
		}
		return WorkspaceSnapshotManifest{}, "", err
	}
	digest := sha256.Sum256(manifestBytes)
	expectedDigest, err := hex.DecodeString(mission.WorkspaceSnapshotSHA256)
	if err != nil || !bytes.Equal(expectedDigest, digest[:]) {
		return WorkspaceSnapshotManifest{}, "", errors.New("persisted workspace snapshot manifest digest mismatch")
	}
	var manifest WorkspaceSnapshotManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return WorkspaceSnapshotManifest{}, "", fmt.Errorf("decode persisted workspace snapshot manifest: %w", err)
	}
	if manifest.SchemaVersion != workspaceSnapshotSchemaVersion || manifest.SnapshotID != mission.WorkspaceSnapshotID || manifest.MissionID != mission.ID || manifest.OrganizationID != mission.OrganizationID || manifest.ProjectID != mission.ProjectID {
		return WorkspaceSnapshotManifest{}, "", errors.New("persisted workspace snapshot identity does not match mission")
	}
	workspacePath, err := filepath.Abs(mission.Workspace)
	if err != nil {
		return WorkspaceSnapshotManifest{}, "", err
	}
	workspacePrefix, err := filepath.Rel(filepath.Join(snapshotPath, "tree"), workspacePath)
	if err != nil || !filepath.IsLocal(workspacePrefix) {
		return WorkspaceSnapshotManifest{}, "", errors.New("isolated mission workspace escapes its snapshot")
	}
	if workspacePrefix == "" {
		workspacePrefix = "."
	}
	return manifest, filepath.ToSlash(workspacePrefix), nil
}

func verifyWorkspaceSnapshotContents(root *os.Root, manifest WorkspaceSnapshotManifest, mission Mission, workspacePrefix string) error {
	if root == nil {
		return errors.New("snapshot tree root is required")
	}
	original := make(map[string]string, len(manifest.Files))
	for _, entry := range manifest.Files {
		if !entry.Included || !entry.Present || entry.Kind != "file" {
			continue
		}
		if err := validateWorkspaceSnapshotPath(entry.Path, workspaceSnapshotDefaultDepth); err != nil {
			return fmt.Errorf("unsafe path in snapshot manifest: %w", err)
		}
		original[filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))] = strings.ToLower(entry.SHA256)
	}
	approved := make(map[string]ArtifactManifest, len(mission.Artifacts))
	for _, artifact := range mission.Artifacts {
		if artifact.MissionID != mission.ID || artifact.Path == "" || artifact.Size < 0 || artifact.Size > artifactMaxBytes {
			continue
		}
		if !filepath.IsLocal(filepath.FromSlash(artifact.Path)) {
			continue
		}
		artifactPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(artifact.Path)))
		if workspacePrefix != "." && workspacePrefix != "" {
			artifactPath = filepath.ToSlash(filepath.Join(filepath.FromSlash(workspacePrefix), filepath.FromSlash(artifactPath)))
		}
		approved[artifactPath] = artifact
	}
	seen := make(map[string]struct{}, len(original))
	err := fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("snapshot contains symlink %q", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("snapshot contains non-regular file %q", path)
		}
		path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		file, err := root.Open(filepath.FromSlash(path))
		if err != nil {
			return err
		}
		hash := sha256.New()
		size, copyErr := io.Copy(hash, io.LimitReader(file, artifactMaxBytes+1))
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if size > artifactMaxBytes || size != info.Size() {
			return fmt.Errorf("snapshot file %q changed while validating", path)
		}
		digest := hex.EncodeToString(hash.Sum(nil))
		if expected, ok := original[path]; ok {
			if strings.EqualFold(expected, digest) {
				seen[path] = struct{}{}
				return nil
			}
		}
		artifact, ok := approved[path]
		if !ok || artifact.Size != size || !strings.EqualFold(strings.TrimSpace(artifact.SHA256), digest) {
			return fmt.Errorf("snapshot file %q is not covered by the original manifest or a persisted artifact", path)
		}
		seen[path] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}
	for path := range original {
		if _, ok := seen[path]; !ok {
			if _, artifactOK := approved[path]; !artifactOK {
				return fmt.Errorf("original snapshot file %q is missing", path)
			}
		}
	}
	return nil
}

// sweepOrphanedWorkspaceSnapshots reclaims unreferenced leaf directories after
// a grace window and snapshots referenced by terminal missions after retention.
// organizationScope limits cleanup to one tenant in request-scoped runtimes.
func sweepOrphanedWorkspaceSnapshots(dataRoot, organizationScope string, missions []Mission, now time.Time) error {
	if strings.TrimSpace(dataRoot) == "" {
		return errors.New("snapshot data root is required for orphan cleanup")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	dataRootHandle, err := openExistingWorkspaceSnapshotDirectoryRoot(dataRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer dataRootHandle.Close()
	storageRoot, err := openExistingWorkspaceSnapshotChildRoot(dataRootHandle, ".agent-workspace-snapshots")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open snapshot storage root: %w", err)
	}
	defer storageRoot.Close()
	root, err := openExistingWorkspaceSnapshotChildRoot(storageRoot, "snapshots")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open snapshot orphan cleanup root: %w", err)
	}
	defer root.Close()
	references := make(map[string]Mission, len(missions))
	for _, mission := range missions {
		if !mission.WorkspaceIsolated || !validSnapshotID(mission.ID) || !validSnapshotID(mission.WorkspaceSnapshotID) || strings.TrimSpace(mission.OrganizationID) == "" {
			continue
		}
		digest := sha256.Sum256([]byte(mission.OrganizationID))
		tenantKey := hex.EncodeToString(digest[:12])
		references[filepath.Join(tenantKey, mission.ID, mission.WorkspaceSnapshotID)] = mission
	}
	var onlyTenant string
	if organizationScope = strings.TrimSpace(organizationScope); organizationScope != "" {
		digest := sha256.Sum256([]byte(organizationScope))
		onlyTenant = hex.EncodeToString(digest[:12])
	}
	tenantDirs, err := readWorkspaceSnapshotDirectory(root, ".")
	if err != nil {
		return err
	}
	var cleanupErrors []error
	for _, tenant := range tenantDirs {
		if !validSnapshotTenantKey(tenant.Name()) || (onlyTenant != "" && tenant.Name() != onlyTenant) || !tenant.IsDir() {
			continue
		}
		tenantRoot, openErr := openExistingWorkspaceSnapshotChildRoot(root, tenant.Name())
		if openErr != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("open snapshot tenant %q: %w", tenant.Name(), openErr))
			continue
		}
		missionDirs, readErr := readWorkspaceSnapshotDirectory(tenantRoot, ".")
		if readErr != nil {
			_ = tenantRoot.Close()
			cleanupErrors = append(cleanupErrors, readErr)
			continue
		}
		for _, missionDir := range missionDirs {
			if !strings.HasPrefix(missionDir.Name(), "mis_") || !validSnapshotID(missionDir.Name()) || !missionDir.IsDir() {
				continue
			}
			relativeMission := filepath.Join(tenant.Name(), missionDir.Name())
			missionRoot, openErr := openExistingWorkspaceSnapshotChildRoot(tenantRoot, missionDir.Name())
			if openErr != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("open snapshot mission %q: %w", relativeMission, openErr))
				continue
			}
			snapshotDirs, readErr := readWorkspaceSnapshotDirectory(missionRoot, ".")
			if readErr != nil {
				_ = missionRoot.Close()
				cleanupErrors = append(cleanupErrors, readErr)
				continue
			}
			for _, snapshotDir := range snapshotDirs {
				if !strings.HasPrefix(snapshotDir.Name(), "snp_") || !validSnapshotID(snapshotDir.Name()) || !snapshotDir.IsDir() {
					continue
				}
				relative := filepath.Join(relativeMission, snapshotDir.Name())
				info, statErr := missionRoot.Lstat(snapshotDir.Name())
				if errors.Is(statErr, os.ErrNotExist) {
					continue
				}
				if statErr != nil {
					cleanupErrors = append(cleanupErrors, statErr)
					continue
				}
				if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					continue
				}
				if mission, referenced := references[relative]; referenced {
					terminal := mission.State == MissionCompleted || mission.State == MissionCancelled || mission.State == MissionFailed
					if !terminal || mission.UpdatedAt.IsZero() || now.Sub(mission.UpdatedAt) < workspaceSnapshotRetention {
						continue
					}
				} else if now.Sub(info.ModTime()) < workspaceSnapshotOrphanGrace {
					continue
				}
				if err := missionRoot.RemoveAll(snapshotDir.Name()); err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Errorf("remove orphan workspace snapshot %q: %w", relative, err))
				}
			}
			_ = missionRoot.Close()
		}
		_ = tenantRoot.Close()
	}
	return errors.Join(cleanupErrors...)
}

func validSnapshotTenantKey(value string) bool {
	if len(value) != 24 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 12
}

type workspaceSnapshotLimits struct {
	maxFiles     int
	maxDepth     int
	maxFileBytes int64
	maxTotal     int64
}

func normalizeWorkspaceSnapshotLimits(request WorkspaceSnapshotRequest) (workspaceSnapshotLimits, error) {
	limits := workspaceSnapshotLimits{
		maxFiles:     request.MaxFiles,
		maxDepth:     request.MaxDepth,
		maxFileBytes: request.MaxFileBytes,
		maxTotal:     request.MaxTotalBytes,
	}
	if limits.maxFiles == 0 {
		limits.maxFiles = workspaceSnapshotDefaultFiles
	}
	if limits.maxDepth == 0 {
		limits.maxDepth = workspaceSnapshotDefaultDepth
	}
	if limits.maxFileBytes == 0 {
		limits.maxFileBytes = workspaceSnapshotDefaultFile
	}
	if limits.maxTotal == 0 {
		limits.maxTotal = workspaceSnapshotDefaultTotal
	}
	if limits.maxFiles < 1 || limits.maxFiles > workspaceSnapshotDefaultFiles || limits.maxDepth < 1 || limits.maxDepth > workspaceSnapshotDefaultDepth || limits.maxFileBytes < 1 || limits.maxFileBytes > workspaceSnapshotDefaultFile || limits.maxTotal < 1 || limits.maxTotal > workspaceSnapshotDefaultTotal {
		return workspaceSnapshotLimits{}, errors.New("snapshot limits must be positive and cannot exceed platform maxima")
	}
	return limits, nil
}

func validSnapshotID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func createWorkspaceSnapshotDirectory(dataRootHandle *os.Root, dataRoot, tenantKey, missionID, snapshotID string) (string, string, *os.Root, []string, error) {
	if dataRootHandle == nil {
		return "", "", nil, nil, errors.New("snapshot data root handle is required")
	}
	currentPath := dataRoot
	currentRoot := dataRootHandle
	closeCurrent := false
	createdParents := make([]string, 0, 3)
	relativeParent := ""
	for _, component := range []string{"snapshots", tenantKey, missionID} {
		childPath := filepath.Join(currentPath, component)
		child, created, err := createWorkspaceSnapshotChildRootTracked(currentRoot, currentPath, component, 0o700)
		if closeCurrent {
			_ = currentRoot.Close()
		}
		if err != nil {
			cleanupErr := removeEmptyWorkspaceSnapshotParents(dataRootHandle, createdParents)
			return "", "", nil, nil, errors.Join(fmt.Errorf("create snapshot directory: %w", err), cleanupErr)
		}
		relativeParent = filepath.Join(relativeParent, component)
		if created {
			createdParents = append(createdParents, filepath.ToSlash(relativeParent))
		}
		currentPath = childPath
		currentRoot = child
		closeCurrent = true
	}
	snapshotRoot, createdLeaf, err := createWorkspaceSnapshotChildRootTracked(currentRoot, currentPath, snapshotID, 0o700)
	if closeCurrent {
		_ = currentRoot.Close()
	}
	if err != nil {
		cleanupErr := removeEmptyWorkspaceSnapshotParents(dataRootHandle, createdParents)
		return "", "", nil, nil, errors.Join(fmt.Errorf("create isolated snapshot directory: %w", err), cleanupErr)
	}
	snapshotPath := filepath.Join(currentPath, snapshotID)
	relative, err := filepath.Rel(dataRoot, snapshotPath)
	if err != nil || filepath.IsAbs(relative) || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		_ = snapshotRoot.Close()
		var leafErr error
		if createdLeaf {
			leafErr = dataRootHandle.RemoveAll(filepath.ToSlash(filepath.Join(relativeParent, snapshotID)))
		}
		cleanupErr := removeEmptyWorkspaceSnapshotParents(dataRootHandle, createdParents)
		return "", "", nil, nil, errors.Join(errors.New("isolated snapshot path escapes data root"), leafErr, cleanupErr)
	}
	return snapshotPath, filepath.ToSlash(relative), snapshotRoot, createdParents, nil
}

func removeEmptyWorkspaceSnapshotParents(dataRoot *os.Root, parents []string) error {
	if dataRoot == nil {
		return errors.New("snapshot data root handle is required for parent cleanup")
	}
	var cleanupErrors []error
	for index := len(parents) - 1; index >= 0; index-- {
		parent := filepath.ToSlash(filepath.Clean(filepath.FromSlash(parents[index])))
		if parent == "." || parent == ".." || strings.HasPrefix(parent, "../") || filepath.IsAbs(parent) {
			cleanupErrors = append(cleanupErrors, errors.New("snapshot cleanup parent escapes data root"))
			continue
		}
		entries, err := readWorkspaceSnapshotDirectory(dataRoot, filepath.FromSlash(parent))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		if len(entries) != 0 {
			continue
		}
		if err := dataRoot.Remove(filepath.FromSlash(parent)); err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func openOrCreateWorkspaceSnapshotDirectoryRoot(path string) (*os.Root, error) {
	return openWorkspaceSnapshotDirectoryRoot(path, true)
}

func openExistingWorkspaceSnapshotDirectoryRoot(path string) (*os.Root, error) {
	return openWorkspaceSnapshotDirectoryRoot(path, false)
}

func openWorkspaceSnapshotDirectoryRoot(path string, create bool) (*os.Root, error) {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	volume := filepath.VolumeName(absolute)
	rootPath := volume + string(filepath.Separator)
	relative := strings.TrimPrefix(absolute, rootPath)
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		info, statErr := root.Lstat(component)
		if create && errors.Is(statErr, os.ErrNotExist) {
			if err := root.Mkdir(component, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				_ = root.Close()
				return nil, err
			}
			info, statErr = root.Lstat(component)
		}
		if statErr != nil {
			_ = root.Close()
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			_ = root.Close()
			return nil, errors.New("snapshot data root contains a symlink or non-directory")
		}
		child, err := root.OpenRoot(component)
		if err != nil {
			_ = root.Close()
			return nil, err
		}
		opened, err := child.Stat(".")
		if err != nil || !os.SameFile(info, opened) {
			_ = child.Close()
			_ = root.Close()
			return nil, ErrWorkspaceSnapshotChanged
		}
		_ = root.Close()
		root = child
	}
	return root, nil
}

func readWorkspaceSnapshotDirectory(root *os.Root, relative string) ([]os.DirEntry, error) {
	if root == nil {
		return nil, errors.New("snapshot directory handle is required")
	}
	directory, err := root.Open(relative)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	return directory.ReadDir(-1)
}

func openExistingWorkspaceSnapshotChildRoot(parent *os.Root, name string) (*os.Root, error) {
	if parent == nil || name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return nil, errors.New("snapshot directory component is invalid")
	}
	before, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, errors.New("snapshot directory contains a symlink or non-directory")
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := child.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = child.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrWorkspaceSnapshotChanged
	}
	return child, nil
}

func createWorkspaceSnapshotChildRoot(parent *os.Root, parentPath, name string, mode os.FileMode) (*os.Root, error) {
	root, _, err := createWorkspaceSnapshotChildRootTracked(parent, parentPath, name, mode)
	return root, err
}

func createWorkspaceSnapshotChildRootTracked(parent *os.Root, parentPath, name string, mode os.FileMode) (*os.Root, bool, error) {
	if parent == nil || name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return nil, false, errors.New("snapshot directory component is invalid")
	}
	created := false
	if err := parent.Mkdir(name, mode); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, false, err
		}
	} else {
		created = true
	}
	before, err := parent.Lstat(name)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		if created {
			_ = parent.Remove(name)
		}
		return nil, false, errors.New("snapshot storage path contains a symlink or non-directory")
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		if created {
			_ = parent.Remove(name)
		}
		return nil, false, err
	}
	opened, err := child.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		_ = child.Close()
		if created {
			_ = parent.Remove(name)
		}
		return nil, false, ErrWorkspaceSnapshotChanged
	}
	if err := secureWorkspaceSnapshotDirectory(filepath.Join(parentPath, name), child); err != nil {
		_ = child.Close()
		if created {
			_ = parent.Remove(name)
		}
		return nil, false, err
	}
	return child, created, nil
}

func verifyWorkspaceSnapshotDirectoryIdentity(path string, root *os.Root) error {
	if root == nil {
		return errors.New("snapshot directory handle is required")
	}
	opened, err := root.Stat(".")
	if err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.IsDir() || !os.SameFile(opened, current) {
		return ErrWorkspaceSnapshotChanged
	}
	return nil
}

func writeJSONAtomicWorkspaceRoot(root *os.Root, name string, value any) error {
	if root == nil || name == "" || strings.ContainsAny(name, `/\\`) {
		return errors.New("anchored JSON file destination is invalid")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporaryName := ".snapshot-manifest-" + uuid.NewString() + ".tmp"
	temporary, err := root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = root.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := root.Rename(temporaryName, name); err != nil {
		return err
	}
	keep = true
	return nil
}

type workspaceSnapshotGitState struct {
	RepositoryRoot string
	Branch         string
	Head           string
	IndexSHA256    string
	Status         []string
	IndexEntries   []WorkspaceSnapshotIndexEntry
	IndexByPath    map[string][]WorkspaceSnapshotIndexEntry
	ModeByPath     map[string]uint32
	SourceByPath   map[string]string
	Paths          []string
	IndexOutput    []byte
	Untracked      []byte
}

func readWorkspaceSnapshotGitState(ctx context.Context, root string, sourceHandle *os.Root, sourceDirectory *os.File, gitView *gitReadView) (workspaceSnapshotGitState, error) {
	var state workspaceSnapshotGitState
	if sourceHandle == nil || sourceDirectory == nil || gitView == nil {
		return state, errors.New("descriptor-bound source root, directory, and private Git view are required for snapshot inspection")
	}
	if err := validateGitConfigForInspection(sourceHandle); err != nil {
		return state, fmt.Errorf("unsafe Git configuration: %w", err)
	}
	if err := validateGitMetadataTree(sourceHandle); err != nil {
		return state, fmt.Errorf("unsafe Git metadata: %w", err)
	}
	runGit := func(args ...string) ([]byte, error) {
		output, commandErr := runWorkspaceSnapshotGitAtRoot(ctx, root, sourceDirectory, gitView, args...)
		if commandErr != nil {
			return output, commandErr
		}
		if err := validateGitMetadataTree(sourceHandle); err != nil {
			return output, fmt.Errorf("Git metadata changed or became unsafe during snapshot inspection: %w", err)
		}
		if err := validateGitConfigForInspection(sourceHandle); err != nil {
			return output, fmt.Errorf("Git configuration changed or became unsafe during snapshot inspection: %w", err)
		}
		return output, nil
	}
	rootOutput, err := runGit("rev-parse", "--show-toplevel")
	if err != nil {
		return state, err
	}
	state.RepositoryRoot = strings.TrimSpace(string(rootOutput))
	branch, err := runGit("branch", "--show-current")
	if err != nil {
		return state, err
	}
	state.Branch = strings.TrimSpace(string(branch))
	status, err := runGit("status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all", "--ignore-submodules=all")
	if err != nil {
		return state, err
	}
	state.Status, err = splitSnapshotNULRecords(status)
	if err != nil {
		return state, fmt.Errorf("Git status contains an unsupported path encoding: %w", err)
	}
	head, headErr := runGit("rev-parse", "--verify", "--quiet", "HEAD")
	if headErr != nil {
		if !bytes.Contains(status, []byte("# branch.oid (initial)")) {
			return state, headErr
		}
	} else {
		state.Head = strings.TrimSpace(string(head))
	}
	state.IndexSHA256, err = hashWorkspaceGitIndex(sourceHandle)
	if err != nil {
		return state, err
	}
	indexOutput, err := runGit("ls-files", "--stage", "-z")
	if err != nil {
		return state, err
	}
	state.IndexOutput = indexOutput
	state.IndexEntries, state.IndexByPath, state.ModeByPath, err = parseWorkspaceSnapshotIndex(indexOutput)
	if err != nil {
		return state, err
	}
	untracked, err := runGit("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return state, err
	}
	state.Untracked = untracked
	untrackedPaths, err := splitSnapshotNULRecords(untracked)
	if err != nil {
		return state, fmt.Errorf("Git untracked list contains an unsupported path encoding: %w", err)
	}
	state.SourceByPath = make(map[string]string, len(state.IndexByPath)+len(untrackedPaths))
	for path := range state.IndexByPath {
		state.SourceByPath[path] = "tracked"
	}
	for _, path := range untrackedPaths {
		if _, exists := state.SourceByPath[path]; !exists {
			state.SourceByPath[path] = "untracked"
		}
	}
	state.Paths = make([]string, 0, len(state.SourceByPath))
	for path := range state.SourceByPath {
		state.Paths = append(state.Paths, path)
	}
	sort.Strings(state.Paths)
	return state, nil
}

func sameWorkspaceSnapshotGitState(before, after workspaceSnapshotGitState) bool {
	return before.RepositoryRoot == after.RepositoryRoot && before.Branch == after.Branch && before.Head == after.Head && before.IndexSHA256 == after.IndexSHA256 && bytes.Equal(before.IndexOutput, after.IndexOutput) && bytes.Equal(before.Untracked, after.Untracked) && equalStrings(before.Status, after.Status)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// runWorkspaceSnapshotGitAtRoot executes Git with cwd pinned to the directory
// descriptor used by snapshot file reads. On Linux, command.Dir is resolved
// before ExtraFiles are remapped in the child, so use the still-open parent
// process descriptor path rather than /proc/self/fd/<parent-fd> in the child.
// Other platforms fail closed because they do not provide the same
// descriptor-backed cwd contract.
func runWorkspaceSnapshotGitAtRoot(ctx context.Context, root string, directory *os.File, gitView *gitReadView, args ...string) ([]byte, error) {
	if directory == nil || gitView == nil || gitView.gitDir == "" {
		return nil, errors.New("descriptor-bound Git workspace directory and private view are required")
	}
	if runtime.GOOS == "linux" {
		return runWorkspaceSnapshotGitCommand(ctx, root, fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), directory.Fd()), nil, gitView, args...)
	}
	// macOS and Windows do not expose Linux's proc-fd cwd contract. The
	// repository root was canonicalized and opened by the caller; use that
	// stable path while keeping Git metadata in the private read view.
	return runWorkspaceSnapshotGitCommand(ctx, root, root, nil, gitView, args...)
}

func runWorkspaceSnapshotGitCommand(ctx context.Context, root, commandDir string, extraFiles []*os.File, gitView *gitReadView, args ...string) ([]byte, error) {
	if gitView == nil || gitView.gitDir == "" {
		return nil, errors.New("private Git read view is required")
	}
	if runtime.GOOS == "linux" && !strings.HasPrefix(commandDir, fmt.Sprintf("/proc/%d/fd/", os.Getpid())) {
		return nil, errors.New("private descriptor-bound Git view is required")
	}
	if runtime.GOOS != "linux" && !filepath.IsAbs(commandDir) {
		return nil, errors.New("canonical Git workspace directory is required")
	}
	gitExecutable, err := trustedGitExecutable()
	if err != nil {
		return nil, err
	}
	commandCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	gitArgs := []string{"--no-pager"}
	pinnedDescriptorCWD := runtime.GOOS == "linux" && strings.HasPrefix(commandDir, fmt.Sprintf("/proc/%d/fd/", os.Getpid()))
	if !pinnedDescriptorCWD {
		gitArgs = append(gitArgs, "-C", root)
	}
	gitArgs = append(gitArgs, "-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-c", "diff.external=", "-c", "core.pager=cat")
	gitArgs = append(gitArgs, args...)
	command := exec.CommandContext(commandCtx, gitExecutable, gitArgs...)
	command.Dir = commandDir
	command.ExtraFiles = extraFiles
	command.Env = []string{
		"PATH=" + safeToolPath(),
		"HOME=" + gitView.directory,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_SYSTEM=",
		"GIT_CONFIG_GLOBAL=",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_DIR=" + gitView.gitDir,
		"GIT_WORK_TREE=" + commandDir,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
		"LC_ALL=C",
	}
	configureToolProcess(command)
	if workspaceSnapshotGitCommandHook != nil {
		workspaceSnapshotGitCommandHook(root, args)
	}
	stdout := &boundedGitBuffer{limit: workspaceSnapshotGitOutputMax}
	stderr := &boundedGitBuffer{limit: 8 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	err = command.Run()
	if commandCtx.Err() != nil {
		return nil, fmt.Errorf("Git snapshot command timed out: %w", commandCtx.Err())
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return nil, fmt.Errorf("Git snapshot command failed: %s", RedactDLP(message))
		}
		return nil, fmt.Errorf("Git snapshot command failed: %w", err)
	}
	if stdout.truncated || stderr.truncated {
		return nil, fmt.Errorf("%w: Git metadata exceeds %d bytes", ErrWorkspaceSnapshotLimit, workspaceSnapshotGitOutputMax)
	}
	if !utf8.Valid(stdout.data) {
		return nil, errors.New("Git metadata is not valid UTF-8")
	}
	return append([]byte(nil), stdout.data...), nil
}

func splitSnapshotNULRecords(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	if data[len(data)-1] != 0 {
		return nil, errors.New("NUL-delimited Git output is incomplete")
	}
	parts := bytes.Split(data[:len(data)-1], []byte{0})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if !utf8.Valid(part) {
			return nil, errors.New("path is not valid UTF-8")
		}
		result = append(result, string(part))
	}
	return result, nil
}

func parseWorkspaceSnapshotIndex(data []byte) ([]WorkspaceSnapshotIndexEntry, map[string][]WorkspaceSnapshotIndexEntry, map[string]uint32, error) {
	entries := make([]WorkspaceSnapshotIndexEntry, 0)
	byPath := make(map[string][]WorkspaceSnapshotIndexEntry)
	modeByPath := make(map[string]uint32)
	if len(data) == 0 {
		return entries, byPath, modeByPath, nil
	}
	if data[len(data)-1] != 0 {
		return nil, nil, nil, errors.New("Git index output is incomplete")
	}
	for _, record := range bytes.Split(data[:len(data)-1], []byte{0}) {
		separator := bytes.IndexByte(record, '\t')
		if separator <= 0 {
			return nil, nil, nil, errors.New("Git index entry is malformed")
		}
		metadata := strings.Fields(string(record[:separator]))
		if len(metadata) != 3 {
			return nil, nil, nil, errors.New("Git index metadata is malformed")
		}
		modeValue, err := strconv.ParseUint(metadata[0], 8, 32)
		if err != nil {
			return nil, nil, nil, errors.New("Git index file mode is invalid")
		}
		stage, err := strconv.Atoi(metadata[2])
		if err != nil || stage < 0 || stage > 3 || metadata[1] == "" {
			return nil, nil, nil, errors.New("Git index object or stage is invalid")
		}
		path := string(record[separator+1:])
		if err := validateWorkspaceSnapshotPath(path, workspaceSnapshotDefaultDepth); err != nil {
			return nil, nil, nil, fmt.Errorf("unsafe Git index path: %w", err)
		}
		entry := WorkspaceSnapshotIndexEntry{Path: path, Mode: uint32(modeValue), ObjectID: metadata[1], Stage: stage}
		entries = append(entries, entry)
		byPath[path] = append(byPath[path], entry)
		if previous, ok := modeByPath[path]; !ok || stage == 0 {
			modeByPath[path] = entry.Mode
		} else if stage < 0 {
			modeByPath[path] = previous
		}
	}
	return entries, byPath, modeByPath, nil
}

func validateWorkspaceSnapshotPath(path string, maxDepth int) error {
	if path == "" || !utf8.ValidString(path) || strings.ContainsRune(path, '\x00') || strings.Contains(path, `\`) {
		return errors.New("path is empty, malformed, or not portable")
	}
	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" || strings.HasPrefix(path, "/") || path == "." || path == ".." || strings.HasPrefix(path, "../") {
		return errors.New("path is absolute or escapes the workspace")
	}
	depth := strings.Count(path, "/") + 1
	if depth > maxDepth {
		return fmt.Errorf("path depth exceeds %d", maxDepth)
	}
	for _, component := range strings.Split(path, "/") {
		if component == "" || component == "." || component == ".." {
			return errors.New("path contains an unsafe component")
		}
		if !portableSnapshotPathComponent(component) {
			return errors.New("path contains a non-portable component")
		}
	}
	return nil
}

func portableSnapshotPathComponent(component string) bool {
	if component == "" || strings.TrimRight(component, ". ") != component || strings.ContainsAny(component, `<>:"|?*`) {
		return false
	}
	for _, character := range component {
		if character < 0x20 {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return false
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return false
	}
	return true
}

func hashWorkspaceGitIndex(sourceRoot *os.Root) (string, error) {
	if sourceRoot == nil {
		return "", errors.New("Git index source root handle is required")
	}
	gitInfo, err := sourceRoot.Lstat(".git")
	if err != nil || !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Git metadata directory is not a real directory")
	}
	gitRoot, err := sourceRoot.OpenRoot(".git")
	if err != nil {
		return "", err
	}
	defer gitRoot.Close()
	openedGitInfo, err := gitRoot.Stat(".")
	if err != nil || !os.SameFile(gitInfo, openedGitInfo) {
		return "", ErrWorkspaceSnapshotChanged
	}
	before, err := gitRoot.Lstat("index")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() > workspaceSnapshotIndexMax {
		return "", errors.New("Git index is symlinked, non-regular, or too large")
	}
	file, err := gitRoot.Open("index")
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", ErrWorkspaceSnapshotChanged
	}
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(file, workspaceSnapshotIndexMax+1))
	if err != nil {
		return "", err
	}
	if read > workspaceSnapshotIndexMax || read != before.Size() {
		return "", ErrWorkspaceSnapshotChanged
	}
	after, err := gitRoot.Lstat("index")
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return "", ErrWorkspaceSnapshotChanged
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyWorkspaceSnapshotFile(ctx context.Context, sourceRoot string, sourceHandle, treeRoot *os.Root, treeRootPath, relative string, indexedMode uint32, limits workspaceSnapshotLimits, totalBytes *int64, entry *WorkspaceSnapshotFile) error {
	if indexedMode == 0o160000 {
		entry.Kind = "submodule"
		entry.SkipReason = "submodule_not_copied"
		return nil
	}
	if err := rejectWorkspaceSnapshotParentSymlinks(sourceRoot, relative); err != nil {
		if errors.Is(err, os.ErrNotExist) && entry.Source == "tracked" {
			entry.SkipReason = "deleted"
			return nil
		}
		return err
	}
	relativeOS := filepath.FromSlash(relative)
	info, err := sourceHandle.Lstat(relativeOS)
	if errors.Is(err, os.ErrNotExist) && entry.Source == "tracked" {
		entry.SkipReason = "deleted"
		return nil
	}
	if err != nil {
		return err
	}
	entry.Present = true
	entry.OriginalMode = uint32(info.Mode().Perm())
	if info.Mode()&os.ModeSymlink != 0 {
		entry.Kind = "symlink"
		entry.SkipReason = "symlink_not_copied"
		return nil
	}
	if info.IsDir() {
		entry.Kind = "directory"
		entry.SkipReason = "directory_or_submodule_not_copied"
		return nil
	}
	if !info.Mode().IsRegular() {
		entry.Kind = "unsupported"
		entry.SkipReason = "non_regular_file_not_copied"
		return nil
	}
	if isSensitiveSnapshotFilename(relative) {
		entry.SkipReason = "credential_like_filename"
		return nil
	}
	if info.Size() < 0 || info.Size() > limits.maxFileBytes {
		return fmt.Errorf("%w: file %q exceeds %d bytes", ErrWorkspaceSnapshotLimit, relative, limits.maxFileBytes)
	}
	if *totalBytes > limits.maxTotal-info.Size() {
		return fmt.Errorf("%w: total copied bytes exceed %d", ErrWorkspaceSnapshotLimit, limits.maxTotal)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rejectWorkspaceSnapshotParentSymlinks(sourceRoot, relative); err != nil {
		return err
	}
	source, err := sourceHandle.Open(relativeOS)
	if err != nil {
		return err
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return ErrWorkspaceSnapshotChanged
	}
	var copied bytes.Buffer
	written, err := copySnapshotWithContext(ctx, &copied, source, limits.maxFileBytes)
	if err != nil {
		return err
	}
	if written != info.Size() {
		return ErrWorkspaceSnapshotChanged
	}
	if workspaceSnapshotAfterFileReadHook != nil {
		workspaceSnapshotAfterFileReadHook(filepath.Join(sourceRoot, relativeOS))
	}
	if err := verifyWorkspaceSnapshotSourceContent(ctx, source, copied.Bytes(), limits.maxFileBytes); err != nil {
		return err
	}
	if snapshotBytesContainCredentialSignal(copied.Bytes()) {
		entry.SkipReason = "credential_like_content"
		return nil
	}
	current, err := sourceHandle.Lstat(relativeOS)
	if err != nil || !os.SameFile(info, current) || info.Size() != current.Size() || !info.ModTime().Equal(current.ModTime()) {
		return ErrWorkspaceSnapshotChanged
	}
	openedAfterCopy, err := source.Stat()
	if err != nil || openedAfterCopy.Size() != opened.Size() || !openedAfterCopy.ModTime().Equal(opened.ModTime()) {
		return ErrWorkspaceSnapshotChanged
	}
	parentRelative := filepath.ToSlash(filepath.Dir(relativeOS))
	parentRoot := treeRoot
	closeParent := false
	if parentRelative != "." {
		parentRoot, err = openOrCreateWorkspaceSnapshotSubdirectory(treeRoot, treeRootPath, parentRelative)
		if err != nil {
			return fmt.Errorf("create snapshot parent directory: %w", err)
		}
		closeParent = true
	}
	if closeParent {
		defer parentRoot.Close()
	}
	destinationName := filepath.Base(relativeOS)
	temporaryName := ".snapshot-" + uuid.NewString() + ".tmp"
	temporary, err := parentRoot.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer temporary.Close()
	keepTemporary := false
	defer func() {
		if !keepTemporary {
			_ = parentRoot.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(copied.Bytes()); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	temporaryInfo, err := temporary.Stat()
	if err != nil || !temporaryInfo.Mode().IsRegular() || temporaryInfo.Size() != written {
		return ErrWorkspaceSnapshotChanged
	}
	mode := os.FileMode(0o600)
	if info.Mode().Perm()&0o111 != 0 {
		mode = 0o700
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := parentRoot.Rename(temporaryName, destinationName); err != nil {
		return err
	}
	keepTemporary = true
	entry.Kind = "file"
	entry.Included = true
	entry.Size = written
	digest := sha256.Sum256(copied.Bytes())
	entry.SHA256 = hex.EncodeToString(digest[:])
	*totalBytes += written
	return nil
}

// verifyWorkspaceSnapshotSourceContent confirms that the captured bytes still
// match a fresh read from the same descriptor. Size and mtime are not enough:
// a writer can replace bytes in place and restore both metadata values.
func verifyWorkspaceSnapshotSourceContent(ctx context.Context, source *os.File, copied []byte, maxBytes int64) error {
	if source == nil {
		return ErrWorkspaceSnapshotChanged
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	expected := sha256.Sum256(copied)
	actual := sha256.New()
	read, err := copySnapshotWithContext(ctx, actual, source, maxBytes)
	if err != nil || read != int64(len(copied)) || !bytes.Equal(actual.Sum(nil), expected[:]) {
		return ErrWorkspaceSnapshotChanged
	}
	return nil
}

func openOrCreateWorkspaceSnapshotSubdirectory(root *os.Root, rootPath, relative string) (*os.Root, error) {
	if root == nil {
		return nil, errors.New("snapshot tree root handle is required")
	}
	current := root
	currentPath := rootPath
	closeCurrent := false
	for _, component := range strings.Split(filepath.ToSlash(relative), "/") {
		if component == "" || component == "." {
			continue
		}
		child, err := createWorkspaceSnapshotChildRoot(current, currentPath, component, 0o700)
		if closeCurrent {
			_ = current.Close()
		}
		if err != nil {
			return nil, err
		}
		current = child
		currentPath = filepath.Join(currentPath, component)
		closeCurrent = true
	}
	if !closeCurrent {
		return nil, errors.New("snapshot file parent path is empty")
	}
	return current, nil
}

func rejectWorkspaceSnapshotParentSymlinks(sourceRoot, relative string) error {
	absolute := filepath.Join(sourceRoot, filepath.FromSlash(relative))
	return rejectSymlinkComponents(sourceRoot, filepath.Dir(absolute))
}

func snapshotBytesContainCredentialSignal(data []byte) bool {
	return snapshotCredentialPattern.Match(data) || snapshotPrivateKeyPattern.Match(data) || len(ScanDLP(string(data))) > 0
}

func copySnapshotWithContext(ctx context.Context, destination io.Writer, source io.Reader, maxBytes int64) (int64, error) {
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			total += int64(count)
			if total > maxBytes {
				return total, fmt.Errorf("%w: source file exceeds %d bytes", ErrWorkspaceSnapshotLimit, maxBytes)
			}
			written, writeErr := destination.Write(buffer[:count])
			if writeErr != nil {
				return total, writeErr
			}
			if written != count {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

func isSensitiveSnapshotFilename(relative string) bool {
	base := strings.ToLower(filepath.Base(relative))
	if base == ".env" || strings.HasPrefix(base, ".env.") || base == ".npmrc" || base == ".pypirc" || base == ".netrc" || base == "credentials" || base == "credentials.json" || base == "secrets" || strings.HasPrefix(base, "secret.") || strings.HasPrefix(base, "credentials.") || strings.HasPrefix(base, "id_rsa") || strings.HasPrefix(base, "id_ed25519") {
		return true
	}
	extension := strings.ToLower(filepath.Ext(base))
	return extension == ".pem" || extension == ".key" || extension == ".p12" || extension == ".pfx" || extension == ".keystore"
}

func containsSnapshotCredentialSignal(reader io.Reader) (bool, error) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	const overlapSize = 512
	buffer := make([]byte, 64<<10)
	var overlap []byte
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			window := make([]byte, 0, len(overlap)+count)
			window = append(window, overlap...)
			window = append(window, buffer[:count]...)
			if snapshotCredentialPattern.Match(window) || snapshotPrivateKeyPattern.Match(window) {
				return true, nil
			}
			if len(window) > overlapSize {
				overlap = append(overlap[:0], window[len(window)-overlapSize:]...)
			} else {
				overlap = append(overlap[:0], window...)
			}
		}
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
}
