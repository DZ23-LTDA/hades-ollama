package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const gitRepoOutputLimit = 32 << 10

var gitRepoCommandHook func(string, []string)

type gitRepoInspectTool struct{}

func (gitRepoInspectTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:        "git.repo.inspect",
		Version:     "4",
		Description: "Inspecionar branch e status do repositório; no snapshot isolado, listar apenas alterações registradas como artefatos da missão",
		Risk:        RiskRead,
		Scopes:      []string{"repo:read"},
	}
}

func (gitRepoInspectTool) Execute(ctx context.Context, toolContext ToolContext, _ map[string]any) (ToolResult, error) {
	if toolContext.WorkspaceRoot != nil && toolContext.WorkspaceSnapshotManifest != nil {
		return inspectIsolatedGitSnapshot(toolContext)
	}
	root, err := safeWorkspacePath(toolContext.Workspace, ".")
	if err != nil {
		return ToolResult{}, err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return ToolResult{}, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return ToolResult{}, errors.New("git workspace root must be a real directory")
	}
	gitDir := filepath.Join(root, ".git")
	gitInfo, err := os.Lstat(gitDir)
	if err != nil {
		return ToolResult{}, errors.New("workspace root is not a supported Git repository")
	}
	if !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
		return ToolResult{}, errors.New("Git worktrees and linked .git files are not supported by this read-only inspector")
	}
	if err := rejectSymlinkComponents(root, gitDir); err != nil {
		return ToolResult{}, err
	}
	for _, relative := range []string{".git/config", ".git/HEAD", ".git/index"} {
		path := filepath.Join(root, relative)
		if _, err := os.Lstat(path); err == nil {
			if err := rejectSymlinkComponents(root, path); err != nil {
				return ToolResult{}, fmt.Errorf("unsafe Git metadata: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return ToolResult{}, err
		}
	}
	workspaceRoot := toolContext.WorkspaceRoot
	ownsWorkspaceRoot := false
	if workspaceRoot == nil {
		workspaceRoot, err = os.OpenRoot(root)
		if err != nil {
			return ToolResult{}, err
		}
		ownsWorkspaceRoot = true
	}
	if ownsWorkspaceRoot {
		defer workspaceRoot.Close()
	}
	openedInfo, err := workspaceRoot.Stat(".")
	if err != nil || !os.SameFile(rootInfo, openedInfo) {
		return ToolResult{}, errors.New("Git workspace changed while it was being authorized")
	}
	if err := validateGitMetadataTree(workspaceRoot); err != nil {
		return ToolResult{}, fmt.Errorf("unsafe Git metadata: %w", err)
	}
	if err := validateGitConfigForInspection(workspaceRoot); err != nil {
		return ToolResult{}, fmt.Errorf("unsafe Git configuration: %w", err)
	}
	readView, err := newGitReadView(workspaceRoot)
	if err != nil {
		return ToolResult{}, err
	}
	defer readView.Close()
	run := func(args ...string) (string, bool, error) {
		return runGitRepoCommandAtRootWithView(ctx, root, workspaceRoot, readView, args...)
	}

	rootOutput, truncated, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return ToolResult{}, err
	}
	topLevel := strings.TrimSpace(rootOutput)
	if truncated || !samePath(topLevel, root) {
		return ToolResult{}, errors.New("Git repository root must exactly match the authorized workspace")
	}

	branch, branchTruncated, err := run("branch", "--show-current")
	if err != nil {
		return ToolResult{}, err
	}
	head, headTruncated, err := run("rev-parse", "--short=12", "HEAD")
	if err != nil {
		return ToolResult{}, err
	}
	status, statusTruncated, err := run("status", "--porcelain=v1", "--branch", "--untracked-files=normal", "--ignore-submodules=all")
	if err != nil {
		return ToolResult{}, err
	}
	unstaged, unstagedTruncated, err := run("diff", "--no-ext-diff", "--no-textconv", "--no-color", "--no-renames", "--unified=3", "--")
	if err != nil {
		return ToolResult{}, err
	}
	staged, stagedTruncated, err := run("diff", "--cached", "--no-ext-diff", "--no-textconv", "--no-color", "--no-renames", "--unified=3", "--")
	if err != nil {
		return ToolResult{}, err
	}

	value := map[string]any{
		"repository_root": root,
		"branch":          strings.TrimSpace(branch),
		"head":            strings.TrimSpace(head),
		"status":          RedactDLP(strings.TrimSpace(status)),
		"unstaged_diff":   RedactDLP(unstaged),
		"staged_diff":     RedactDLP(staged),
		"truncated":       branchTruncated || headTruncated || statusTruncated || unstagedTruncated || stagedTruncated,
		"read_only":       true,
	}
	return ToolResult{Value: value}, nil
}

func validateGitMetadataTree(root *os.Root) error {
	if root == nil {
		return errors.New("Git workspace root handle is required")
	}
	for _, pointer := range []string{".git/objects/info/alternates", ".git/objects/info/http-alternates", ".git/commondir"} {
		if _, err := root.Lstat(pointer); err == nil {
			return fmt.Errorf("Git metadata pointer %q is not supported", pointer)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	entries := 0
	err := fs.WalkDir(root.FS(), ".git", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entries++
		if entries > 100_000 {
			return errors.New("Git metadata entry limit exceeded")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("Git metadata entry %q is not a regular file or directory", path)
		}
		return nil
	})
	return err
}

// validateGitConfigForInspection prevents local config from loading external
// files or invoking configured filter processes during status/diff. Global and
// system configuration are disabled in the child process.
func validateGitConfigForInspection(root *os.Root) error {
	if root == nil {
		return errors.New("Git workspace root handle is required")
	}
	worktreeConfig, err := validateGitConfigFile(root, ".git/config")
	if err != nil {
		return err
	}
	if worktreeConfig {
		if _, err := validateGitConfigFile(root, ".git/config.worktree"); err != nil {
			return fmt.Errorf("validate worktree Git config: %w", err)
		}
	}
	return nil
}

func validateGitConfigFile(root *os.Root, path string) (bool, error) {
	info, err := root.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > 8<<20 {
		return false, errors.New("Git config is not a regular file or exceeds the size limit")
	}
	config, err := root.Open(path)
	if err != nil {
		return false, err
	}
	defer config.Close()
	openedInfo, err := config.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return false, errors.New("Git config changed while it was being opened")
	}
	data, err := io.ReadAll(io.LimitReader(config, 8<<20+1))
	if err != nil {
		return false, err
	}
	if len(data) > 8<<20 {
		return false, errors.New("Git config exceeds the size limit")
	}
	return parseGitConfigForInspection(data)
}

func parseGitConfigForInspection(data []byte) (bool, error) {
	section := ""
	worktreeConfig := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(stripGitConfigComment(raw))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return false, errors.New("ambiguous Git config section header is not supported")
			}
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			switch gitConfigSectionBase(section) {
			case "alias":
				return false, errors.New("repository-local Git aliases are not supported")
			case "filter":
				return false, errors.New("repository-local external filter configuration is not supported")
			case "include", "includeif":
				return false, errors.New("repository-local include/includeIf directives are not supported")
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		if strings.HasPrefix(key, "include.path") || strings.HasPrefix(key, "includeif.") {
			return false, errors.New("repository-local include/includeIf directives are not supported")
		}
		if section == "core" && key == "worktree" {
			return false, errors.New("repository-local core.worktree redirection is not supported")
		}
		if section == "extensions" && key == "worktreeconfig" {
			if !found {
				worktreeConfig = true
				continue
			}
			value = strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
			switch value {
			case "true", "yes", "on", "1":
				worktreeConfig = true
			case "false", "no", "off", "0":
			default:
				return false, errors.New("Git worktreeConfig extension has an invalid boolean value")
			}
		}
	}
	return worktreeConfig, nil
}

func gitConfigSectionBase(section string) string {
	section = strings.ToLower(strings.TrimSpace(section))
	if end := strings.IndexAny(section, " \t\r\n."); end >= 0 {
		section = section[:end]
	}
	return section
}

func stripGitConfigComment(line string) string {
	quoted := false
	escaped := false
	for index, character := range line {
		if escaped {
			escaped = false
			continue
		}
		if quoted && character == '\\' {
			escaped = true
			continue
		}
		if character == '"' {
			quoted = !quoted
			continue
		}
		if !quoted && (character == '#' || character == ';') && (index == 0 || line[index-1] == ' ' || line[index-1] == '\t') {
			return line[:index]
		}
	}
	return line
}

func inspectIsolatedGitSnapshot(toolContext ToolContext) (ToolResult, error) {
	manifest := toolContext.WorkspaceSnapshotManifest
	if manifest == nil || toolContext.WorkspaceRoot == nil {
		return ToolResult{}, errors.New("validated isolated Git snapshot is required")
	}
	prefix := filepath.ToSlash(filepath.Clean(filepath.FromSlash(toolContext.WorkspaceSnapshotPrefix)))
	if prefix == "" {
		prefix = "."
	}
	if prefix != "." && !filepath.IsLocal(filepath.FromSlash(prefix)) {
		return ToolResult{}, errors.New("isolated Git workspace prefix is unsafe")
	}

	baseline := make(map[string]string)
	baselineStatus := make(map[string]int)
	for _, entry := range manifest.Files {
		if !entry.Included || !entry.Present || entry.Kind != "file" || isSensitiveSnapshotFilename(entry.Path) {
			continue
		}
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))
		if prefix != "." {
			prefixPath := prefix + "/"
			if !strings.HasPrefix(path, prefixPath) {
				continue
			}
			path = strings.TrimPrefix(path, prefixPath)
		}
		if path == "" || path == "." || !filepath.IsLocal(filepath.FromSlash(path)) {
			continue
		}
		baseline[path] = strings.ToLower(entry.SHA256)
		baselineStatus[entry.Source]++
	}

	current := make(map[string]ArtifactManifest)
	for _, artifact := range toolContext.MissionArtifacts {
		if artifact.MissionID != toolContext.MissionID || artifact.Path == "" || artifact.Size < 0 || artifact.Size > artifactMaxBytes {
			continue
		}
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(artifact.Path)))
		if !filepath.IsLocal(filepath.FromSlash(path)) || strings.HasPrefix(path, ".agent-backups/") || isSensitiveSnapshotFilename(path) {
			continue
		}
		if _, ok := current[path]; !ok {
			current[path] = artifact
			continue
		}
		if artifact.CreatedAt.After(current[path].CreatedAt) {
			current[path] = artifact
		}
	}

	paths := make([]string, 0, len(current))
	for path := range current {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	changes := make([]map[string]any, 0)
	lines := make([]string, 0)
	for _, path := range paths {
		artifact := current[path]
		baselineHash, existed := baseline[path]
		if existed && strings.EqualFold(baselineHash, artifact.SHA256) {
			continue
		}
		change := "added"
		if existed {
			change = "modified"
		}
		changes = append(changes, map[string]any{"change": change, "path": path, "baseline_sha256": baselineHash, "artifact_sha256": strings.ToLower(artifact.SHA256)})
		lines = append(lines, fmt.Sprintf("%s %s", map[string]string{"added": "A", "modified": "M"}[change], path))
	}
	statusSummary := fmt.Sprintf("isolated snapshot baseline: %d included files; %d tracked, %d untracked", len(baseline), baselineStatus["tracked"], baselineStatus["untracked"])
	return ToolResult{Value: map[string]any{
		"repository_root":           ".",
		"branch":                    RedactDLP(manifest.GitBranch),
		"head":                      RedactDLP(manifest.GitHead),
		"status":                    statusSummary,
		"unstaged_diff":             strings.Join(lines, "\n"),
		"staged_diff":               "",
		"staged_diff_available":     false,
		"staged_diff_note":          "isolated snapshot does not expose the source Git index",
		"changes":                   changes,
		"change_scope":              "persisted_mission_artifacts_only",
		"changes_complete":          false,
		"truncated":                 false,
		"read_only":                 true,
		"baseline_is_mission_start": true,
	}}, nil
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func runGitRepoCommand(ctx context.Context, root string, args ...string) (string, bool, error) {
	if runtime.GOOS != "linux" {
		return "", false, errors.New("descriptor-bound Git execution is unsupported on this platform")
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		return "", false, err
	}
	defer handle.Close()
	return runGitRepoCommandAtRoot(ctx, root, handle, args...)
}

func runGitRepoCommandAtRoot(ctx context.Context, root string, handle *os.Root, args ...string) (string, bool, error) {
	view, err := newGitReadView(handle)
	if err != nil {
		return "", false, err
	}
	defer view.Close()
	return runGitRepoCommandAtRootWithView(ctx, root, handle, view, args...)
}

func runGitRepoCommandAtRootWithView(ctx context.Context, root string, handle *os.Root, view *gitReadView, args ...string) (string, bool, error) {
	if handle == nil {
		return "", false, errors.New("Git workspace handle is required")
	}
	if view == nil || view.gitDir == "" {
		return "", false, errors.New("private Git metadata view is required")
	}
	if runtime.GOOS != "linux" {
		return "", false, errors.New("descriptor-bound Git execution is unsupported on this platform")
	}
	if err := validateGitMetadataTree(handle); err != nil {
		return "", false, fmt.Errorf("unsafe Git metadata: %w", err)
	}
	if err := validateGitConfigForInspection(handle); err != nil {
		return "", false, fmt.Errorf("unsafe Git configuration: %w", err)
	}
	file, err := handle.Open(".")
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	commandDir := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), file.Fd())
	output, truncated, commandErr := runGitRepoCommandWithDir(ctx, root, commandDir, view, args...)
	if err := validateGitMetadataTree(handle); err != nil {
		return output, truncated, fmt.Errorf("Git metadata changed or became unsafe during inspection: %w", err)
	}
	if err := validateGitConfigForInspection(handle); err != nil {
		return output, truncated, fmt.Errorf("Git configuration changed or became unsafe during inspection: %w", err)
	}
	return output, truncated, commandErr
}

func runGitRepoCommandWithDir(ctx context.Context, root, commandDir string, view *gitReadView, args ...string) (string, bool, error) {
	if runtime.GOOS != "linux" {
		return "", false, errors.New("descriptor-bound Git execution is unsupported on this platform")
	}
	if view == nil || view.gitDir == "" || !strings.HasPrefix(commandDir, fmt.Sprintf("/proc/%d/fd/", os.Getpid())) {
		return "", false, errors.New("descriptor-bound Git command directory is required")
	}
	gitExecutable, err := trustedGitExecutable()
	if err != nil {
		return "", false, err
	}
	commandContext, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	gitArgs := []string{
		"-c", "core.fsmonitor=false",
		"-c", "core.pager=cat",
		"-c", "color.ui=false",
		"-c", "diff.external=",
	}
	gitArgs = append(gitArgs, args...)
	command := exec.CommandContext(commandContext, gitExecutable, gitArgs...)
	command.Dir = commandDir
	command.Env = []string{
		"PATH=" + safeToolPath(),
		"HOME=" + view.directory,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_DIR=" + view.gitDir,
		"GIT_WORK_TREE=" + commandDir,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_LAZY_FETCH=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
		"LC_ALL=C",
	}
	configureToolProcess(command)
	if gitRepoCommandHook != nil {
		gitRepoCommandHook(root, args)
	}
	stdoutBuffer := &boundedGitBuffer{limit: gitRepoOutputLimit}
	stderrBuffer := &boundedGitBuffer{limit: 8 << 10}
	command.Stdout = stdoutBuffer
	command.Stderr = stderrBuffer
	waitErr := command.Run()
	if commandContext.Err() != nil {
		return stdoutBuffer.String(), stdoutBuffer.truncated, fmt.Errorf("Git read command timed out: %w", commandContext.Err())
	}
	if waitErr != nil {
		message := strings.TrimSpace(stderrBuffer.String())
		if message != "" {
			message = RedactDLP(message)
			return stdoutBuffer.String(), stdoutBuffer.truncated, fmt.Errorf("Git read command failed: %s", message)
		}
		return stdoutBuffer.String(), stdoutBuffer.truncated, fmt.Errorf("Git read command failed: %w", waitErr)
	}
	return stdoutBuffer.String(), stdoutBuffer.truncated || stderrBuffer.truncated, nil
}

type boundedGitBuffer struct {
	data      []byte
	limit     int
	truncated bool
}

func (b *boundedGitBuffer) Write(data []byte) (int, error) {
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		b.data = append(b.data, data[:remaining]...)
	}
	if len(data) > remaining {
		b.truncated = true
	}
	return len(data), nil
}

func (b *boundedGitBuffer) String() string { return string(b.data) }

// trustedGitExecutable resolves Git only beneath fixed system directories. Do
// this before setting command.Env: exec.Command resolves bare names against the
// parent process PATH at construction time.
func trustedGitExecutable() (string, error) {
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, directory := range strings.Split(safeToolPath(), string(os.PathListSeparator)) {
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", errors.New("Git executable was not found in trusted system directories")
}

func trustedToolExecutable(name string) (string, error) {
	if name == "git" {
		return trustedGitExecutable()
	}
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return "", errors.New("terminal executable name is invalid")
	}
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}
	for _, directory := range strings.Split(safeToolPath(), string(os.PathListSeparator)) {
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("terminal executable %q was not found in trusted system directories", name)
}

func safeToolPath() string {
	if runtime.GOOS == "windows" {
		// Never trust ambient PATH entries inherited by the service: a user may
		// write a fake git.exe there and influence the command executed by tools.
		return `C:\Program Files\Git\cmd;C:\Program Files\Git\bin;C:\Windows\System32;C:\Windows`
	}
	return "/usr/local/bin:/usr/bin:/bin"
}
