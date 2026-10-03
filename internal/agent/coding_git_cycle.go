package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GitWorktreeSession represents an isolated worktree for a mission.
type GitWorktreeSession struct {
	MissionID    string    `json:"mission_id"`
	RepoRoot     string    `json:"repo_root"`
	WorktreeDir  string    `json:"worktree_dir"`
	BranchName   string    `json:"branch_name"`
	BaseCommit   string    `json:"base_commit"`
	TargetBranch string    `json:"target_branch"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"` // "active", "merged", "rejected", "removed"
}

// GitMergeApproval contains details for human-in-the-loop merge approval.
type GitMergeApproval struct {
	MissionID    string   `json:"mission_id"`
	RepoRoot     string   `json:"repo_root"`
	BranchName   string   `json:"branch_name"`
	TargetBranch string   `json:"target_branch"`
	BaseCommit   string   `json:"base_commit"`
	HeadCommit   string   `json:"head_commit"`
	Diff         string   `json:"diff"`
	DiffSHA256   string   `json:"diff_sha256"`
	FilesChanged []string `json:"files_changed"`
}

// TestRunOptions configures project test execution.
type TestRunOptions struct {
	SubPath        string   `json:"sub_path,omitempty"`
	Framework      string   `json:"framework,omitempty"` // "go", "npm", "pytest", "custom"
	CustomCommand  []string `json:"custom_command,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	AutoRepair     bool     `json:"auto_repair,omitempty"`
	MaxRepairTries int      `json:"max_repair_tries,omitempty"`
}

// TestRunResult holds the outcome of a project test run.
type TestRunResult struct {
	Passed      bool     `json:"passed"`
	ExitCode    int      `json:"exit_code"`
	Framework   string   `json:"framework"`
	Command     string   `json:"command"`
	Output      string   `json:"output"`
	FailedTests []string `json:"failed_tests,omitempty"`
	DurationMs  int64    `json:"duration_ms"`
}

// runRawGit executes a git command in the target directory with safe environment.
func runRawGit(ctx context.Context, dir string, args ...string) (string, error) {
	gitExec, err := trustedGitExecutable()
	if err != nil {
		gitExec, err = exec.LookPath("git")
		if err != nil {
			return "", fmt.Errorf("git executable not found: %w", err)
		}
	}
	cmd := exec.CommandContext(ctx, gitExec, args...)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + safeToolPath() + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Ollama Agent",
		"GIT_AUTHOR_EMAIL=agent@ollama.local",
		"GIT_COMMITTER_NAME=Ollama Agent",
		"GIT_COMMITTER_EMAIL=agent@ollama.local",
		"GIT_CONFIG_NOSYSTEM=1",
		"LC_ALL=C",
	}
	if home := os.Getenv("HOME"); home != "" {
		cmd.Env = append(cmd.Env, "HOME="+home)
	} else if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		cmd.Env = append(cmd.Env, "USERPROFILE="+userProfile)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("git %s: %s (%w)", strings.Join(args, " "), RedactDLP(msg), err)
		}
		return stdout.String(), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// CreateGitWorktree creates an isolated git worktree with a dedicated branch for the mission.
func CreateGitWorktree(ctx context.Context, repoRoot, dataRoot, missionID, customBranch string) (*GitWorktreeSession, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, errors.New("repo root is required")
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	toplevel, err := runRawGit(ctx, absRepo, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("validate git repository: %w", err)
	}
	absRepo = filepath.Clean(toplevel)

	baseCommit, err := runRawGit(ctx, absRepo, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("get base commit: %w", err)
	}
	targetBranch, _ := runRawGit(ctx, absRepo, "rev-parse", "--abbrev-ref", "HEAD")
	if targetBranch == "" || targetBranch == "HEAD" {
		targetBranch = "main"
	}

	branchName := strings.TrimSpace(customBranch)
	if branchName == "" {
		branchName = "agent/" + missionID
	}

	worktreeDir := filepath.Join(dataRoot, ".agent-worktrees", missionID)
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0o700); err != nil {
		return nil, fmt.Errorf("create worktrees directory: %w", err)
	}
	// If worktree directory already exists, ensure it is clean
	_ = os.RemoveAll(worktreeDir)

	// Create worktree on a new branch from HEAD
	_, err = runRawGit(ctx, absRepo, "worktree", "add", "-b", branchName, worktreeDir, baseCommit)
	if err != nil {
		// If branch already exists, try without -b
		_, err2 := runRawGit(ctx, absRepo, "worktree", "add", worktreeDir, branchName)
		if err2 != nil {
			return nil, fmt.Errorf("create git worktree: %w", err)
		}
	}

	return &GitWorktreeSession{
		MissionID:    missionID,
		RepoRoot:     absRepo,
		WorktreeDir:  worktreeDir,
		BranchName:   branchName,
		BaseCommit:   baseCommit,
		TargetBranch: targetBranch,
		CreatedAt:    time.Now().UTC(),
		Status:       "active",
	}, nil
}

// RemoveGitWorktree safely detaches and removes the worktree.
func RemoveGitWorktree(ctx context.Context, session *GitWorktreeSession) error {
	if session == nil || session.WorktreeDir == "" {
		return nil
	}
	if session.RepoRoot != "" {
		_, _ = runRawGit(ctx, session.RepoRoot, "worktree", "remove", "--force", session.WorktreeDir)
	}
	_ = os.RemoveAll(session.WorktreeDir)
	session.Status = "removed"
	return nil
}

// GetWorktreeDiff computes the diff between the base commit and the worktree changes.
func GetWorktreeDiff(ctx context.Context, session *GitWorktreeSession) (*GitMergeApproval, error) {
	if session == nil {
		return nil, errors.New("worktree session is required")
	}
	// Check if there are uncommitted changes in the worktree; if so, commit them
	status, err := runRawGit(ctx, session.WorktreeDir, "status", "--porcelain")
	if err == nil && strings.TrimSpace(status) != "" {
		_, _ = runRawGit(ctx, session.WorktreeDir, "add", "-A")
		_, _ = runRawGit(ctx, session.WorktreeDir, "commit", "-m", fmt.Sprintf("agent(%s): automated changes for mission", session.MissionID))
	}

	headCommit, err := runRawGit(ctx, session.WorktreeDir, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("get worktree head commit: %w", err)
	}

	diff, err := runRawGit(ctx, session.WorktreeDir, "diff", session.BaseCommit+"...HEAD")
	if err != nil {
		// Fallback to diff against base commit directly
		diff, err = runRawGit(ctx, session.WorktreeDir, "diff", session.BaseCommit, "HEAD")
		if err != nil {
			diff = ""
		}
	}

	filesRaw, _ := runRawGit(ctx, session.WorktreeDir, "diff", "--name-only", session.BaseCommit+"...HEAD")
	var filesChanged []string
	for _, f := range strings.Split(filesRaw, "\n") {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			filesChanged = append(filesChanged, trimmed)
		}
	}

	hash := sha256.Sum256([]byte(diff))
	diffSHA256 := hex.EncodeToString(hash[:])

	return &GitMergeApproval{
		MissionID:    session.MissionID,
		RepoRoot:     session.RepoRoot,
		BranchName:   session.BranchName,
		TargetBranch: session.TargetBranch,
		BaseCommit:   session.BaseCommit,
		HeadCommit:   headCommit,
		Diff:         diff,
		DiffSHA256:   diffSHA256,
		FilesChanged: filesChanged,
	}, nil
}

// MergeWorktreeToOrigin merges the mission's branch into the origin repository working branch.
func MergeWorktreeToOrigin(ctx context.Context, session *GitWorktreeSession) (string, error) {
	if session == nil {
		return "", errors.New("worktree session is required")
	}
	if session.RepoRoot == "" || session.BranchName == "" {
		return "", errors.New("invalid worktree session parameters")
	}

	// Ensure changes in worktree are committed before merge
	status, err := runRawGit(ctx, session.WorktreeDir, "status", "--porcelain")
	if err == nil && strings.TrimSpace(status) != "" {
		_, _ = runRawGit(ctx, session.WorktreeDir, "add", "-A")
		_, _ = runRawGit(ctx, session.WorktreeDir, "commit", "-m", fmt.Sprintf("agent(%s): automated changes for mission", session.MissionID))
	}

	// Perform merge in origin repo
	mergeMsg := fmt.Sprintf("merge branch '%s' for mission %s", session.BranchName, session.MissionID)
	_, err = runRawGit(ctx, session.RepoRoot, "merge", "--no-ff", "-m", mergeMsg, session.BranchName)
	if err != nil {
		// Try fast-forward if no-ff failed
		out, err := runRawGit(ctx, session.RepoRoot, "merge", session.BranchName)
		if err != nil {
			return "", fmt.Errorf("git merge failed: %w (output: %s)", err, out)
		}
	}

	mergeCommit, err := runRawGit(ctx, session.RepoRoot, "rev-parse", "HEAD")
	if err != nil {
		mergeCommit = "merged"
	}
	session.Status = "merged"
	return mergeCommit, nil
}

// RunProjectTests detects and executes the project's tests inside the workspace with no egress.
func RunProjectTests(ctx context.Context, workspacePath string, options TestRunOptions) (TestRunResult, error) {
	if strings.TrimSpace(workspacePath) == "" {
		return TestRunResult{}, errors.New("workspace path is required")
	}
	targetDir := workspacePath
	if options.SubPath != "" {
		// Contain the subpath strictly within the workspace: reject absolute
		// paths and "..", then resolve symlinks and verify the result stays
		// under the workspace root (SEC-02).
		if !filepath.IsLocal(options.SubPath) {
			return TestRunResult{}, fmt.Errorf("invalid sub_path %q: must be a relative path inside the workspace", options.SubPath)
		}
		candidate := filepath.Join(workspacePath, options.SubPath)
		rootReal, err := filepath.EvalSymlinks(workspacePath)
		if err != nil {
			return TestRunResult{}, fmt.Errorf("resolve workspace root: %w", err)
		}
		candidateReal, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			// Target may not exist yet; fall back to the lexically cleaned path.
			candidateReal = filepath.Clean(candidate)
		}
		rel, err := filepath.Rel(rootReal, candidateReal)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return TestRunResult{}, fmt.Errorf("sub_path %q escapes the workspace", options.SubPath)
		}
		targetDir = candidate
	}

	framework := strings.ToLower(strings.TrimSpace(options.Framework))
	var commandArgs []string

	if len(options.CustomCommand) > 0 {
		framework = "custom"
		commandArgs = options.CustomCommand
	} else if framework == "" {
		// Auto-detect framework
		if _, err := os.Stat(filepath.Join(targetDir, "go.mod")); err == nil {
			framework = "go"
		} else if _, err := os.Stat(filepath.Join(targetDir, "package.json")); err == nil {
			framework = "npm"
		} else if hasPythonFiles(targetDir) {
			framework = "pytest"
		} else {
			return TestRunResult{
				Passed:    true,
				ExitCode:  0,
				Framework: "none",
				Command:   "none",
				Output:    "No test runner configuration detected (no go.mod, package.json, or python files).",
			}, nil
		}
	}

	var executable string
	switch framework {
	case "go":
		goExec, err := exec.LookPath("go")
		if err != nil {
			goExec = "/usr/local/go/bin/go"
		}
		executable = goExec
		commandArgs = []string{"test", "-v", "./..."}
	case "npm":
		npmExec, err := exec.LookPath("npm")
		if err != nil {
			npmExec = "/usr/bin/npm"
		}
		executable = npmExec
		commandArgs = []string{"test", "--", "--run"}
	case "pytest", "python":
		pyExec, err := exec.LookPath("pytest")
		if err == nil {
			executable = pyExec
			commandArgs = []string{"-v"}
		} else {
			pythonExec, err := exec.LookPath("python3")
			if err != nil {
				pythonExec = "python"
			}
			executable = pythonExec
			commandArgs = []string{"-m", "unittest", "discover"}
		}
	case "custom":
		if len(options.CustomCommand) == 0 {
			return TestRunResult{}, errors.New("custom command cannot be empty")
		}
		executable = options.CustomCommand[0]
		if len(options.CustomCommand) > 1 {
			commandArgs = options.CustomCommand[1:]
		}
	default:
		return TestRunResult{}, fmt.Errorf("unsupported test framework: %s", framework)
	}

	timeout := 120 * time.Second
	if options.TimeoutSeconds > 0 {
		timeout = time.Duration(options.TimeoutSeconds) * time.Second
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(testCtx, executable, commandArgs...)
	cmd.Dir = targetDir

	// Strict sandbox environment: zero egress
	cmd.Env = []string{
		"PATH=" + safeToolPath() + string(os.PathListSeparator) + os.Getenv("PATH"),
		"GOPROXY=off",
		"GONOSUMDB=*",
		"NODE_ENV=test",
		"PIP_NO_INDEX=1",
		"http_proxy=",
		"https_proxy=",
		"HTTP_PROXY=",
		"HTTPS_PROXY=",
		"ALL_PROXY=",
		"NO_PROXY=*",
		"LC_ALL=C",
	}
	for _, envKey := range []string{
		"SystemRoot", "SYSTEMROOT", "windir", "WINDIR",
		"LOCALAPPDATA", "LocalAppData", "APPDATA",
		"TEMP", "TMP", "TMPDIR", "GOCACHE", "GOPATH", "GOROOT",
		"USERPROFILE", "HOME", "USER", "USERNAME",
	} {
		if val, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(val) != "" {
			cmd.Env = append(cmd.Env, envKey+"="+val)
		}
	}
	if gocache := os.Getenv("GOCACHE"); gocache == "" {
		cacheDir := filepath.Join(os.TempDir(), "agent-go-build-cache")
		_ = os.MkdirAll(cacheDir, 0o700)
		cmd.Env = append(cmd.Env, "GOCACHE="+cacheDir)
	}

	start := time.Now()
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	runErr := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	passed := true
	if runErr != nil {
		passed = false
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	output := outBuf.String()
	failedTests := parseFailedTests(output, framework)

	return TestRunResult{
		Passed:      passed,
		ExitCode:    exitCode,
		Framework:   framework,
		Command:     executable + " " + strings.Join(commandArgs, " "),
		Output:      output,
		FailedTests: failedTests,
		DurationMs:  duration.Milliseconds(),
	}, nil
}

func hasPythonFiles(dir string) bool {
	for _, marker := range []string{"pytest.ini", "pyproject.toml", "setup.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			return true
		}
	}
	return false
}

var (
	goFailPattern  = regexp.MustCompile(`(?m)^--- FAIL: (\w+)`)
	npmFailPattern = regexp.MustCompile(`(?m)FAIL ([\w/.-]+)`)
	pyFailPattern  = regexp.MustCompile(`(?m)^FAILED ([\w/.:]+)`)
)

func parseFailedTests(output, framework string) []string {
	var failed []string
	switch framework {
	case "go":
		matches := goFailPattern.FindAllStringSubmatch(output, -1)
		for _, m := range matches {
			if len(m) > 1 {
				failed = append(failed, m[1])
			}
		}
	case "npm":
		matches := npmFailPattern.FindAllStringSubmatch(output, -1)
		for _, m := range matches {
			if len(m) > 1 {
				failed = append(failed, m[1])
			}
		}
	case "pytest", "python":
		matches := pyFailPattern.FindAllStringSubmatch(output, -1)
		for _, m := range matches {
			if len(m) > 1 {
				failed = append(failed, m[1])
			}
		}
	}
	return failed
}

// ExecuteRepairLoop runs the project tests and, upon failure, attempts fixes up to maxTries,
// recording every attempt via mission SSE events.
func ExecuteRepairLoop(ctx context.Context, r *Runtime, mission Mission, step *Step, testOpts TestRunOptions, repairFn func(attempt int, lastOutput string) error, maxTries int) (TestRunResult, error) {
	if maxTries <= 0 {
		maxTries = 3
	}

	var lastResult TestRunResult
	for attempt := 1; attempt <= maxTries; attempt++ {
		if r != nil {
			_ = r.observeEvent(mission, "mission.repair_attempt", step.ID, map[string]any{
				"attempt":      attempt,
				"max_attempts": maxTries,
				"status":       "running_tests",
			})
		}

		result, err := RunProjectTests(ctx, mission.Workspace, testOpts)
		lastResult = result
		if err == nil && result.Passed {
			if r != nil {
				_ = r.observeEvent(mission, "mission.repair_succeeded", step.ID, map[string]any{
					"attempt":     attempt,
					"duration_ms": result.DurationMs,
					"output":      result.Output,
				})
			}
			return result, nil
		}

		// Tests failed
		if r != nil {
			_ = r.observeEvent(mission, "mission.repair_failed_test", step.ID, map[string]any{
				"attempt":      attempt,
				"exit_code":    result.ExitCode,
				"failed_tests": result.FailedTests,
				"error":        RedactDLP(result.Output),
			})
		}

		if attempt < maxTries && repairFn != nil {
			repairErr := repairFn(attempt, result.Output)
			if repairErr != nil {
				if r != nil {
					_ = r.observeEvent(mission, "mission.repair_error", step.ID, map[string]any{
						"attempt": attempt,
						"error":   RedactDLP(repairErr.Error()),
					})
				}
			}
		}
	}

	if r != nil {
		_ = r.observeEvent(mission, "mission.repair_exhausted", step.ID, map[string]any{
			"attempts":     maxTries,
			"failed_tests": lastResult.FailedTests,
		})
	}
	return lastResult, fmt.Errorf("tests failed after %d repair attempts: %s", maxTries, lastResult.Output)
}

// projectTestRunnerTool exposes project test execution as a Tool.
type projectTestRunnerTool struct{}

func (projectTestRunnerTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:    "project.test.run",
		Version: "1",
		// Honest risk: this runs the project's own test command (go test, npm
		// test, pytest or a custom command), which executes arbitrary project
		// code on the host. It is NOT a real kernel sandbox, so it is treated as
		// an external-side-effect action and requires explicit human approval
		// (SEC-01).
		Description:      "Executar o runner de testes do projeto (Go, Node/npm, Python). Atenção: executa código do projeto no host e exige aprovação.",
		Risk:             RiskExternalSideEffect,
		Scopes:           []string{"workspace:read", "sandbox:execute"},
		RequiresApproval: true,
	}
}

func (projectTestRunnerTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	subPath, _ := input["sub_path"].(string)
	framework, _ := input["framework"].(string)
	opts := TestRunOptions{
		SubPath:   subPath,
		Framework: framework,
	}
	if timeoutVal, ok := input["timeout_seconds"].(float64); ok && timeoutVal > 0 {
		opts.TimeoutSeconds = int(timeoutVal)
	}

	result, err := RunProjectTests(ctx, toolContext.Workspace, opts)
	if err != nil {
		return ToolResult{Value: map[string]any{
			"passed":    false,
			"error":     err.Error(),
			"framework": opts.Framework,
		}}, err
	}

	return ToolResult{
		Value: map[string]any{
			"passed":       result.Passed,
			"exit_code":    result.ExitCode,
			"framework":    result.Framework,
			"command":      result.Command,
			"output":       result.Output,
			"failed_tests": result.FailedTests,
			"duration_ms":  result.DurationMs,
		},
	}, nil
}

// gitMergeOriginTool exposes merging the worktree branch to origin with HITL approval.
type gitMergeOriginTool struct{}

func (gitMergeOriginTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:             "git.merge.origin",
		Version:          "1",
		Description:      "Mesclar o branch de trabalho isolado de volta ao repositório original após aprovação humana explícita",
		Risk:             RiskWrite,
		Scopes:           []string{"workspace:write", "repo:read"},
		RequiresApproval: true,
	}
}

func (gitMergeOriginTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	repoRoot, _ := input["repo_root"].(string)
	branchName, _ := input["branch_name"].(string)
	worktreeDir, _ := input["worktree_dir"].(string)

	if repoRoot == "" || branchName == "" {
		return ToolResult{}, errors.New("repo_root and branch_name are required for merge")
	}

	session := &GitWorktreeSession{
		MissionID:   toolContext.MissionID,
		RepoRoot:    repoRoot,
		BranchName:  branchName,
		WorktreeDir: worktreeDir,
	}

	commitSHA, err := MergeWorktreeToOrigin(ctx, session)
	if err != nil {
		return ToolResult{}, fmt.Errorf("merge to origin: %w", err)
	}

	return ToolResult{
		Value: map[string]any{
			"status":       "merged",
			"branch":       branchName,
			"merge_commit": commitSHA,
		},
	}, nil
}
