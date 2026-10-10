package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Registry struct {
	tools map[string]Tool
}

var ErrCapabilityDenied = errors.New("tool capability is not granted to this mission")

func capabilityAllowed(descriptor ToolDescriptor, granted []string) bool {
	return DefaultCapabilityPolicy().Allows(descriptor, granted)
}

func NewRegistry() *Registry {
	registry := &Registry{tools: make(map[string]Tool)}
	registry.Register(workspaceListTool{})
	registry.Register(workspaceReadTool{})
	registry.Register(workspaceWriteTool{})
	registry.Register(workspacePatchTool{})
	registry.Register(gitRepoInspectTool{})
	registry.Register(terminalExecTool{allowed: map[string]bool{"pwd": true, "ls": true}})
	registry.Register(sandboxExecTool{})
	registry.Register(browserOperatorTool{})
	registry.Register(desktopCompanionTool{})
	registry.Register(projectTestRunnerTool{})
	registry.Register(gitMergeOriginTool{})
	registry.Register(imTelegramSendTool{})
	registry.Register(imTelegramVerifyTool{})
	return registry
}

func (r *Registry) Register(tool Tool) {
	if r == nil || tool == nil {
		return
	}
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	r.tools[tool.Descriptor().Name] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Descriptors() []ToolDescriptor {
	if r == nil {
		return nil
	}
	result := make([]ToolDescriptor, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool.Descriptor())
	}
	return result
}

type workspaceListTool struct{}

func (workspaceListTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.list", Version: "1", Description: "Listar entradas do workspace autorizado", Risk: RiskRead, Scopes: []string{"workspace:read"}}
}

func (workspaceListTool) Execute(_ context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	rawPath := stringInput(input, "path", ".")
	var path, relative string
	var err error
	if toolContext.WorkspaceRoot != nil {
		relative, err = safeAnchoredWorkspaceRelative(toolContext.WorkspaceRoot, rawPath, true)
		path = relative
	} else {
		path, err = safeWorkspacePath(toolContext.Workspace, rawPath)
	}
	if err != nil {
		return ToolResult{}, err
	}
	maxEntries := intInput(input, "max_entries", 100)
	if maxEntries < 1 {
		maxEntries = 1
	}
	if maxEntries > 500 {
		maxEntries = 500
	}
	root := toolContext.WorkspaceRoot
	if root == nil {
		if err := rejectSymlinkComponents(toolContext.Workspace, path); err != nil {
			return ToolResult{}, err
		}
		rootPath, absErr := filepath.Abs(toolContext.Workspace)
		if absErr != nil {
			return ToolResult{}, absErr
		}
		root, err = os.OpenRoot(rootPath)
		if err != nil {
			return ToolResult{}, err
		}
		defer root.Close()
		if relative, err = filepath.Rel(rootPath, path); err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return ToolResult{}, errors.New("tool path escapes workspace")
		}
	}
	info, err := root.Lstat(filepath.ToSlash(relative))
	if err != nil {
		return ToolResult{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ToolResult{}, errors.New("workspace.list target must be a real directory")
	}
	directory, err := root.Open(filepath.ToSlash(relative))
	if err != nil {
		return ToolResult{}, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(maxEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return ToolResult{}, err
	}
	truncated := len(entries) > maxEntries
	if truncated {
		entries = entries[:maxEntries]
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		result = append(result, map[string]any{"name": entry.Name(), "directory": entry.IsDir()})
	}
	return ToolResult{Value: map[string]any{"path": path, "entries": result, "truncated": truncated}}, nil
}

type workspaceReadTool struct{}

func safeAnchoredWorkspaceRelative(root *os.Root, raw string, allowRoot bool) (string, error) {
	if root == nil || strings.TrimSpace(raw) == "" {
		return "", errors.New("anchored workspace path is required")
	}
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if strings.HasPrefix(raw, "/") || filepath.IsAbs(raw) || filepath.VolumeName(raw) != "" || strings.ContainsRune(raw, '\x00') {
		return "", errors.New("tool path must be relative to workspace")
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." {
		if allowRoot {
			return ".", nil
		}
		return "", errors.New("tool path must identify a workspace file")
	}
	if !filepath.IsLocal(clean) {
		return "", errors.New("tool path escapes workspace")
	}
	current := ""
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("tool path contains a symlink")
		}
	}
	return filepath.ToSlash(clean), nil
}

func (workspaceReadTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.read", Version: "1", Description: "Ler um arquivo do workspace autorizado", Risk: RiskRead, Scopes: []string{"workspace:read"}}
}

func (workspaceReadTool) Execute(_ context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	var path string
	var data []byte
	var err error
	if toolContext.WorkspaceRoot != nil {
		path, err = safeAnchoredWorkspaceRelative(toolContext.WorkspaceRoot, stringInput(input, "path", ""), false)
		if err == nil {
			data, err = readWorkspaceFileLimited(toolContext.WorkspaceRoot, path, 1<<20, "workspace.read limit exceeded")
		}
	} else {
		path, err = safeWorkspacePath(toolContext.Workspace, stringInput(input, "path", ""))
		if err == nil {
			err = rejectSymlinkComponents(toolContext.Workspace, path)
		}
		if err == nil {
			data, err = readWorkspacePathLimited(toolContext.Workspace, path, 1<<20, "workspace.read limit exceeded")
		}
	}
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Value: map[string]any{"path": path, "content": string(data), "bytes": len(data), "sha256": workspaceContentSHA256(data)}}, nil
}

type workspaceWriteTool struct{}

func (workspaceWriteTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.write", Version: "2", Description: "Criar ou atualizar atomicamente arquivo do workspace após approval; overwrite exige SHA-256 observado e produz backup", Risk: RiskWrite, RequiresApproval: true, Scopes: []string{"workspace:write"}}
}

func (workspaceWriteTool) Execute(_ context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	return writeWorkspaceFile(toolContext, input)
}

type workspacePatchTool struct{}

func (workspacePatchTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.patch", Version: "1", Description: "Aplicar patch limitado de múltiplos arquivos após approval; cada arquivo exibe diff e usa SHA-256/CAS, com compensação em falha observada", Risk: RiskWrite, RequiresApproval: true, Scopes: []string{"workspace:write"}}
}

func (workspacePatchTool) Execute(_ context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	return writeWorkspacePatch(toolContext, input)
}

type terminalExecTool struct {
	allowed map[string]bool
}

func (t terminalExecTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "terminal.exec", Version: "1", Description: "Executar um binário explicitamente permitido no workspace", Risk: RiskWrite, RequiresApproval: true, Scopes: []string{"terminal:allowlisted"}}
}

func (t terminalExecTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	executable := filepath.Base(stringInput(input, "executable", ""))
	if executable == "" || !t.allowed[executable] {
		return ToolResult{}, fmt.Errorf("terminal executable %q is not allowlisted", executable)
	}
	if executable == "git" {
		return ToolResult{}, errors.New("Git is available only through the hardened git.repo.inspect tool")
	}
	if runtime.GOOS != "linux" || toolContext.WorkspaceRoot == nil {
		return ToolResult{}, errors.New("terminal execution requires a pinned Linux workspace directory")
	}
	args := stringSliceInput(input, "args")
	for _, arg := range args {
		if strings.ContainsAny(arg, "\x00\r\n") {
			return ToolResult{}, errors.New("terminal argument contains a control character")
		}
	}
	if err := validateTerminalArguments(executable, args, toolContext.Workspace); err != nil {
		return ToolResult{}, err
	}
	if executable == "ls" {
		return executeRootedTerminalLS(toolContext.WorkspaceRoot)
	}
	resolvedExecutable, err := trustedToolExecutable(executable)
	if err != nil {
		return ToolResult{}, err
	}
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cwdFile, err := toolContext.WorkspaceRoot.Open(".")
	if err != nil {
		return ToolResult{}, fmt.Errorf("open pinned terminal workspace: %w", err)
	}
	defer cwdFile.Close()
	command := exec.Command(resolvedExecutable, args...)
	command.Dir = fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), cwdFile.Fd())
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=/", "PWD=" + command.Dir}
	configureToolProcess(command)
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedBuffer{Buffer: &stdout, Limit: 64 << 10}
	command.Stderr = &limitedBuffer{Buffer: &stderr, Limit: 64 << 10}
	if err := runToolCommand(deadline, command); err != nil {
		return ToolResult{Value: map[string]any{"stdout": RedactDLP(stdout.String()), "stderr": RedactDLP(stderr.String()), "execution_isolation": "best-effort-process-group", "resource_limits": "context-timeout-and-output-bounded"}}, err
	}
	return ToolResult{Value: map[string]any{"stdout": RedactDLP(stdout.String()), "stderr": RedactDLP(stderr.String()), "exit_code": 0, "execution_isolation": "best-effort-process-group", "resource_limits": "context-timeout-and-output-bounded"}}, nil
}

func validateTerminalArguments(executable string, args []string, workspace string) error {
	switch executable {
	case "pwd":
		if len(args) != 0 {
			return errors.New("pwd does not accept arguments in the default terminal policy")
		}
	case "git":
		return errors.New("Git commands are available only through git.repo.inspect")
	case "ls":
		allowedFlags := map[string]bool{"-a": true, "-A": true, "-l": true, "-la": true, "-al": true, "--all": true, "--almost-all": true, "--format=long": true}
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				if !allowedFlags[arg] {
					return fmt.Errorf("ls flag %q is not allowlisted", arg)
				}
				continue
			}
			if filepath.Clean(filepath.FromSlash(strings.ReplaceAll(arg, "\\", "/"))) != "." {
				return errors.New("ls accepts only the pinned workspace root")
			}
		}
	}
	return nil
}

func executeRootedTerminalLS(root *os.Root) (ToolResult, error) {
	if root == nil {
		return ToolResult{}, errors.New("ls requires a pinned workspace directory")
	}
	directory, err := root.Open(".")
	if err != nil {
		return ToolResult{}, fmt.Errorf("open pinned ls workspace: %w", err)
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return ToolResult{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var stdout strings.Builder
	for _, entry := range entries {
		stdout.WriteString(entry.Name())
		stdout.WriteByte('\n')
	}
	return ToolResult{Value: map[string]any{
		"stdout":              RedactDLP(stdout.String()),
		"stderr":              "",
		"exit_code":           0,
		"execution_isolation": "descriptor-rooted-read",
		"resource_limits":     "bounded-directory-read",
	}}, nil
}

type limitedBuffer struct {
	Buffer *bytes.Buffer
	Limit  int
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if b.Buffer.Len()+len(data) > b.Limit {
		return 0, errors.New("tool output limit exceeded")
	}
	return b.Buffer.Write(data)
}

func safeWorkspacePath(workspace, relative string) (string, error) {
	if strings.TrimSpace(workspace) == "" {
		return "", errors.New("workspace is required")
	}
	if filepath.IsAbs(relative) {
		return "", errors.New("tool path must be relative to workspace")
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(root, relative))
	if err != nil {
		return "", err
	}
	if !isWithin(root, candidate) {
		return "", errors.New("tool path escapes workspace")
	}
	if err := rejectSymlinkComponents(root, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func rejectSymlinkComponents(root, candidate string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return err
	}
	if !isWithin(root, candidate) {
		return errors.New("tool path escapes workspace")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("tool path contains a symlink")
		}
		resolved, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr != nil {
			return resolveErr
		}
		if !isWithin(realRoot, resolved) {
			return errors.New("tool path resolves outside workspace")
		}
	}
	return nil
}

var safeStepIDPattern = regexp.MustCompile(`^step_[A-Za-z0-9_-]{1,100}$`)

func validateStepID(stepID string) error {
	if !safeStepIDPattern.MatchString(strings.TrimSpace(stepID)) {
		return errors.New("step id is invalid")
	}
	return nil
}

func stringInput(input map[string]any, key, fallback string) string {
	value, ok := input[key].(string)
	if !ok {
		return fallback
	}
	return value
}

func intInput(input map[string]any, key string, fallback int) int {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	case string:
		parsed, err := strconv.Atoi(number)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func stringSliceInput(input map[string]any, key string) []string {
	value, ok := input[key]
	if !ok {
		return nil
	}
	var result []string
	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		for _, item := range items {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type sandboxExecTool struct{}

func (sandboxExecTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "sandbox.exec", Version: "4", Description: "Executar Python ou Node no sandbox Linux, com filesystem mínimo e modo vinculado ao approval", Risk: RiskWrite, RequiresApproval: true, Scopes: []string{"sandbox:execute"}}
}

func resolveSandboxInterpreter(language string) (string, error) {
	if runtime.GOOS != "linux" {
		return "", errors.New("sandbox interpreter resolution is supported only on Linux trusted system paths")
	}
	var name string
	switch language {
	case "python", "python3":
		name = "python3"
	case "node":
		name = "node"
	default:
		return "", errors.New("sandbox language must be python or node")
	}
	resolved, err := trustedToolExecutable(name)
	if err != nil {
		return "", fmt.Errorf("sandbox interpreter unavailable for %s", language)
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve sandbox interpreter for %s: %w", language, err)
	}
	if !isWithin("/usr", resolved) && !isWithin("/usr/local", resolved) {
		return "", errors.New("sandbox interpreter resolves outside trusted system directories")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", errors.New("sandbox interpreter is not a trusted executable file")
	}
	return resolved, nil
}

func sandboxPythonStdlibPath(ctx context.Context, interpreter string) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, interpreter, "-I", "-S", "-c", "import sysconfig; print(sysconfig.get_path('stdlib'))")
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=/tmp", "PYTHONDONTWRITEBYTECODE=1"}
	output := &boundedGitBuffer{limit: 512}
	command.Stdout = output
	command.Stderr = io.Discard
	if err := command.Run(); err != nil || output.truncated {
		return "", errors.New("could not securely resolve the Python standard library")
	}
	path, err := filepath.Abs(strings.TrimSpace(output.String()))
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !isWithin("/usr/lib", path) && !isWithin("/usr/local/lib", path) {
		return "", errors.New("Python standard library resolves outside trusted system directories")
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", errors.New("Python standard library is not a trusted directory")
	}
	return path, nil
}

func sandboxSharedLibraryPath() (string, error) {
	var triplet string
	switch runtime.GOARCH {
	case "amd64":
		triplet = "x86_64-linux-gnu"
	case "arm64":
		triplet = "aarch64-linux-gnu"
	default:
		return "", errors.New("sandbox shared-library layout is unsupported on this architecture")
	}
	path := filepath.Join("/usr/lib", triplet)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !isWithin("/usr/lib", resolved) {
		return "", errors.New("sandbox shared-library directory resolves outside trusted system directories")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("sandbox shared-library path is not a trusted directory")
	}
	return resolved, nil
}

func configuredSandboxMode() (string, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_MODE")))
	if mode == "" {
		mode = "strict"
	}
	if mode != "best-effort" && mode != "strict" {
		return "", errors.New("OLLAMA_AGENT_SANDBOX_MODE must be best-effort or strict")
	}
	if mode == "best-effort" && strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT")) != "true" {
		return "", fmt.Errorf("sandbox best-effort is disabled by default: set OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT=true only with explicit approval notice (%s)", sandboxBestEffortApprovalNotice)
	}
	return mode, nil
}

const sandboxBestEffortApprovalNotice = "strong process isolation is unavailable; this run is NOT_CONFIGURED and only permitted by explicit operator approval"

func (sandboxExecTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if err := validateStepID(toolContext.StepID); err != nil {
		return ToolResult{}, err
	}
	if runtime.GOOS != "linux" {
		return ToolResult{}, errors.New("sandbox.exec requires the Linux namespace executor; host-process fallback is disabled")
	}
	mode, err := configuredSandboxMode()
	if err != nil {
		return ToolResult{}, err
	}
	approvedMode := strings.ToLower(strings.TrimSpace(stringInput(input, "sandbox_mode", "")))
	if approvedMode == "" || approvedMode != mode {
		return ToolResult{}, fmt.Errorf("%w: sandbox isolation mode changed or is not bound to approval", ErrApprovalPayloadChanged)
	}
	if mode == "best-effort" {
		if stringInput(input, "sandbox_gate_status", "") != string(GateStatusNotConfigured) || stringInput(input, "sandbox_approval_notice", "") != sandboxBestEffortApprovalNotice {
			return ToolResult{Value: map[string]any{"gate_status": GateStatusNotConfigured, "execution_isolation": "best-effort-not-approved"}}, fmt.Errorf("sandbox best-effort requires explicit approval notice (%s)", GateStatusNotConfigured)
		}
	}
	language := strings.ToLower(strings.TrimSpace(stringInput(input, "language", "")))
	interpreter, err := resolveSandboxInterpreter(language)
	if err != nil {
		return ToolResult{}, err
	}
	pythonInterpreter, err := resolveSandboxInterpreter("python")
	if err != nil {
		return ToolResult{}, fmt.Errorf("sandbox Python launcher unavailable: %w", err)
	}
	pythonStdlibPath, err := sandboxPythonStdlibPath(ctx, pythonInterpreter)
	if err != nil {
		return ToolResult{}, err
	}
	sharedLibraryPath, err := sandboxSharedLibraryPath()
	if err != nil {
		return ToolResult{}, err
	}
	dashPath, err := trustedSandboxExecutable("dash")
	if err != nil {
		return ToolResult{}, fmt.Errorf("sandbox shell unavailable: %w", err)
	}
	code := stringInput(input, "code", "")
	if strings.TrimSpace(code) == "" {
		return ToolResult{}, errors.New("sandbox code is required")
	}
	if len(code) > 512<<10 {
		return ToolResult{}, errors.New("sandbox code limit exceeded")
	}
	strict := mode == "strict"
	runID := "run_" + uuid.NewString()
	var control *sandboxControl
	if strict {
		var err error
		control, err = newSandboxControl(runID)
		if err != nil {
			return ToolResult{Value: map[string]any{"gate_status": GateStatusNotConfigured, "execution_isolation": "unavailable"}}, fmt.Errorf("strict sandbox unavailable (%s): %w", GateStatusNotConfigured, err)
		}
		defer closeSandboxControl(control)
	}
	workspaceRoot := toolContext.WorkspaceRoot
	ownsWorkspaceRoot := false
	if workspaceRoot == nil {
		workspaceRoot, err = os.OpenRoot(toolContext.Workspace)
		if err != nil {
			return ToolResult{}, err
		}
		ownsWorkspaceRoot = true
	}
	if ownsWorkspaceRoot {
		defer workspaceRoot.Close()
	}
	executionDirName := ".agent-sandbox-run-" + uuid.NewString()
	if err := workspaceRoot.Mkdir(executionDirName, 0o700); err != nil {
		return ToolResult{}, err
	}
	extension := ".py"
	if language == "node" {
		extension = ".js"
	}
	codeRelative := filepath.Join(executionDirName, "main"+extension)
	launcherRelative := ""
	defer func() {
		if err := workspaceRoot.RemoveAll(executionDirName); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Error("agent sandbox cleanup failed", "run_id", runID, "error", err)
		}
	}()
	if err := writeContainedSandboxFile(workspaceRoot, codeRelative, []byte(code)); err != nil {
		return ToolResult{}, err
	}
	if strict {
		launcherRelative = filepath.Join(executionDirName, "launcher.py")
		if err := writeContainedSandboxFile(workspaceRoot, launcherRelative, []byte(strictSandboxLauncher)); err != nil {
			return ToolResult{}, err
		}
	}
	deadline, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	codeInSandbox := "/workspace/" + filepath.ToSlash(codeRelative)
	launcherInSandbox := ""
	if strict {
		launcherInSandbox = "/workspace/" + filepath.ToSlash(launcherRelative)
	}
	unsharePath, err := trustedSandboxExecutable("unshare")
	if err != nil {
		return ToolResult{}, err
	}
	setprivPath, err := trustedSandboxExecutable("setpriv")
	if err != nil {
		return ToolResult{}, err
	}
	chrootPath, err := trustedSandboxExecutable("chroot")
	if err != nil {
		return ToolResult{}, err
	}
	mountScript := `set -eu
ulimit -t 55
ulimit -v 524288
ulimit -n 256
ulimit -f 1048576
mount --make-rprivate /
mount -t tmpfs -o size=256m,nosuid,nodev tmpfs /home
ROOT=/home/sandbox-root
INTERPRETER_ROOT=/usr/bin/python3
if [ "$8" = "node" ]; then INTERPRETER_ROOT=/usr/bin/node; fi
mkdir -p "$ROOT"/workspace "$ROOT"/tmp "$ROOT"/usr/bin "$ROOT"/usr/lib "$ROOT"/bin "$ROOT"/dev
mkdir -p "$ROOT$9" "${ROOT}${10}"
mount --bind "$9" "$ROOT$9"
mount -o remount,bind,ro "$ROOT$9"
mount --bind "${10}" "${ROOT}${10}"
mount -o remount,bind,ro "${ROOT}${10}"
: > "$ROOT/usr/bin/python3"
mount --bind "${11}" "$ROOT/usr/bin/python3"
mount -o remount,bind,ro "$ROOT/usr/bin/python3"
if [ "$8" = "node" ]; then
  : > "$ROOT/usr/bin/node"
  mount --bind "$2" "$ROOT/usr/bin/node"
  mount -o remount,bind,ro "$ROOT/usr/bin/node"
fi
: > "$ROOT/bin/sh"
mount --bind "${12}" "$ROOT/bin/sh"
mount -o remount,bind,ro "$ROOT/bin/sh"
ln -s "${10}" "$ROOT/lib"
ln -s "${10}" "$ROOT/lib64"
mount --bind "$1" "$ROOT"/workspace
if [ "$4" = "strict" ]; then mount -o remount,bind,ro "$ROOT"/workspace; fi
mount -t tmpfs -o size=64m,nosuid,nodev tmpfs "$ROOT"/tmp
mount -t tmpfs -o size=1m,nosuid,noexec tmpfs "$ROOT"/dev
for device in null zero urandom; do : > "$ROOT/dev/$device"; mount --bind "/dev/$device" "$ROOT/dev/$device"; done
exec 3<&-

if [ "$4" = "strict" ]; then
  exec "$6" --no-new-privs --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all "$7" "$ROOT" /bin/sh -c 'cd /workspace && exec /usr/bin/python3 -I -S "$@"' sandbox "$5" "$INTERPRETER_ROOT" "$3"
fi
if [ "$8" != "node" ]; then
  exec "$7" "$ROOT" /bin/sh -c 'cd /workspace && exec /usr/bin/python3 -I -S "$@"' sandbox "$3"
fi
exec "$7" "$ROOT" /bin/sh -c 'cd /workspace && exec /usr/bin/node "$@"' sandbox "$3"`
	workspaceDirectory, err := workspaceRoot.Open(".")
	if err != nil {
		return ToolResult{}, fmt.Errorf("open anchored workspace directory: %w", err)
	}
	defer workspaceDirectory.Close()
	args := []string{"--user", "--map-root-user", "--mount", "--pid", "--fork", "--mount-proc", "--net", "--kill-child", "--propagation", "private", "/bin/sh", "-c", mountScript, "sandbox", "/proc/self/fd/3", interpreter, codeInSandbox, mode, launcherInSandbox, setprivPath, chrootPath, language, pythonStdlibPath, sharedLibraryPath, pythonInterpreter, dashPath}
	command := exec.Command(unsharePath, args...)
	command.ExtraFiles = []*os.File{workspaceDirectory}
	command.Dir = "/"
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=/home/workspace", "PWD=/home/workspace"}
	configureToolProcess(command)
	if err := configureSandboxCommand(command, control); err != nil {
		return ToolResult{}, fmt.Errorf("configure strict sandbox: %w", err)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedBuffer{Buffer: &stdout, Limit: 128 << 10}
	command.Stderr = &limitedBuffer{Buffer: &stderr, Limit: 128 << 10}
	if err := runToolCommand(deadline, command, control); err != nil {
		isolation := "best-effort-unshare"
		limits := "ulimit-context-timeout-output-bounded"
		if strict {
			isolation = "strict-linux-user-mount-pid-net-seccomp-cgroupv2"
			limits = "cgroup-v2-cpu-memory-pids-swap-timeout-output-bounded"
		}
		return ToolResult{Value: map[string]any{"stdout": RedactDLP(stdout.String()), "stderr": RedactDLP(stderr.String()), "execution_isolation": isolation, "resource_limits": limits}}, err
	}
	isolation := "best-effort-unshare"
	limits := "ulimit-context-timeout-output-bounded"
	if strict {
		isolation = "strict-linux-user-mount-pid-net-seccomp-cgroupv2"
		limits = "cgroup-v2-cpu-memory-pids-swap-timeout-output-bounded"
	}
	gateStatus := GateStatusPass
	if !strict {
		gateStatus = GateStatusNotConfigured
	}
	return ToolResult{Value: map[string]any{"stdout": RedactDLP(stdout.String()), "stderr": RedactDLP(stderr.String()), "exit_code": 0, "execution_isolation": isolation, "resource_limits": limits, "gate_status": gateStatus}}, nil
}

func writeContainedSandboxFile(root *os.Root, relative string, data []byte) error {
	file, err := root.OpenFile(relative, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return closeErr
}

func runToolCommand(ctx context.Context, command *exec.Cmd, controls ...*sandboxControl) error {
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if len(controls) > 0 {
			killSandboxControl(controls[0])
		}
		terminateToolProcess(command)
		<-done
		return ctx.Err()
	}
}
