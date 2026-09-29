package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTrustedExecutablePathsRejectWritableDirectories(t *testing.T) {
	writable := t.TempDir()
	if err := os.Chmod(writable, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, ok := trustedSystemExecutableDirectory(writable); ok {
		t.Fatal("group/world-writable directory accepted as trusted")
	}
	if _, err := trustedGitExecutable(); err != nil {
		t.Fatalf("trusted system Git could not be resolved: %v", err)
	}
}

func TestGitReadersIgnoreAmbientPATH(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ambient executable hijack regression uses a POSIX script")
	}
	root := initGitRepository(t)
	fakeBin := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ambient-git-was-run")
	fake := "#!/bin/sh\nprintf hijacked > " + marker + "\nexit 97\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "git"), []byte(fake), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin)
	if _, _, err := runGitRepoCommand(context.Background(), root, "rev-parse", "--show-toplevel"); err != nil {
		t.Fatalf("descriptor-bound Git reader failed: %v", err)
	}
	workspaceRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer workspaceRoot.Close()
	readView, err := newGitReadView(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer readView.Close()
	directory, err := workspaceRoot.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if _, err := runWorkspaceSnapshotGitAtRoot(context.Background(), root, directory, readView, "rev-parse", "--show-toplevel"); err != nil {
		t.Fatalf("snapshot Git reader failed: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("ambient fake git was executed (stat error %v)", err)
	}
}

func TestGitReadViewOmitsConfigAndHooks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("private descriptor-bound Git views are Linux-specific")
	}
	root := initGitRepository(t)
	hook := filepath.Join(root, ".git", "hooks", "post-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	workspaceRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer workspaceRoot.Close()
	view, err := newGitReadView(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer view.Close()
	for _, omitted := range []string{"config", "hooks/post-commit"} {
		if _, err := os.Stat(filepath.Join(view.gitDir, omitted)); !os.IsNotExist(err) {
			t.Errorf("private Git view unexpectedly contains %q: stat error=%v", omitted, err)
		}
	}
	for _, required := range []string{"HEAD", "objects"} {
		if _, err := os.Stat(filepath.Join(view.gitDir, required)); err != nil {
			t.Errorf("private Git view omitted required metadata %q: %v", required, err)
		}
	}
}

func TestGitRepoInspectReturnsBoundedReadOnlySnapshot(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	root := initGitRepository(t)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "tracked.txt")
	runFixtureGit(t, root, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "staged.txt"), []byte("staged content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("untracked content\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", result.Value)
	}
	if value["read_only"] != true || value["repository_root"] != root || strings.TrimSpace(value["head"].(string)) == "" {
		t.Fatalf("repository metadata = %#v", value)
	}
	status := value["status"].(string)
	unstaged := value["unstaged_diff"].(string)
	staged := value["staged_diff"].(string)
	for _, expected := range []string{"tracked.txt", "staged.txt", "untracked.txt"} {
		if !strings.Contains(status, expected) {
			t.Errorf("status %q omits %q", status, expected)
		}
	}
	if !strings.Contains(unstaged, "changed") || !strings.Contains(staged, "staged content") {
		t.Fatalf("diffs do not match working tree: unstaged=%q staged=%q", unstaged, staged)
	}
}

func TestGitRepoInspectDoesNotExecuteConfigInjectedBeforeDiff(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("private descriptor-bound Git view regression requires Linux")
	}
	root := initGitRepository(t)
	tracked := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("tracked.txt filter=evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "tracked.txt", ".gitattributes")
	runFixtureGit(t, root, "commit", "-m", "tracked filter attribute")
	if err := os.WriteFile(tracked, []byte("modified\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "filter-executed")
	filter := filepath.Join(t.TempDir(), "evil-filter.sh")
	if err := os.WriteFile(filter, []byte("#!/bin/sh\nprintf executed >> "+marker+"\ncat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	injected := false
	gitRepoCommandHook = func(_ string, args []string) {
		if injected || len(args) == 0 || args[0] != "diff" {
			return
		}
		injected = true
		configPath := filepath.Join(root, ".git", "config")
		config, err := os.ReadFile(configPath)
		if err != nil {
			t.Errorf("read live Git config: %v", err)
			return
		}
		config = append(config, []byte("\n[filter \"evil\"]\n\tclean = /bin/sh "+filter+"\n")...)
		if err := os.WriteFile(configPath, config, 0o600); err != nil {
			t.Errorf("inject live Git filter config: %v", err)
		}
	}
	defer func() { gitRepoCommandHook = nil }()
	_, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil)
	if !injected {
		t.Fatal("Git config mutation hook did not reach diff")
	}
	if err == nil || !strings.Contains(err.Error(), "Git configuration changed or became unsafe") {
		t.Fatalf("inspection result error=%v, want fail-closed post-command config validation", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("attacker-controlled clean filter executed; marker stat error=%v", statErr)
	}
}

func TestGitRepoInspectAcceptsPinnedNonIsolatedWorkspaceRoot(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	root := initGitRepository(t)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "tracked.txt")
	runFixtureGit(t, root, "commit", "-m", "initial")
	workspaceRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer workspaceRoot.Close()
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root, WorkspaceRoot: workspaceRoot}, nil); err != nil {
		t.Fatalf("pinned normal workspace Git inspection failed: %v", err)
	}
}

func TestGitRepoInspectRequiresWorkspaceToBeRepositoryRoot(t *testing.T) {
	workspace := t.TempDir()
	_ = initGitRepositoryAt(t, filepath.Join(workspace, "nested"))
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: workspace}, nil); err == nil || !strings.Contains(err.Error(), "not a supported Git repository") {
		t.Fatalf("nested repository result error = %v", err)
	}
}

func TestGitRepoInspectRejectsGitWorktreePointer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: ../outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "worktrees") {
		t.Fatalf("worktree pointer error = %v", err)
	}
}

func TestGitRepoInspectRejectsGitMetadataSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink may require elevated privileges on Windows")
	}
	root := initGitRepository(t)
	outside := filepath.Join(t.TempDir(), "git-config")
	if err := os.WriteFile(outside, []byte("[core]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, ".git", "config")
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, config); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "unsafe Git metadata") {
		t.Fatalf("symlink metadata error = %v", err)
	}
}

func TestGitRepoInspectRejectsSymlinkedObjectStore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink may require elevated privileges on Windows")
	}
	root := initGitRepository(t)
	objects := filepath.Join(root, ".git", "objects")
	if err := os.RemoveAll(objects); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), objects); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "unsafe Git metadata") {
		t.Fatalf("symlinked Git object store error = %v, want metadata rejection", err)
	}
}

func TestGitRepoInspectRejectsExternalObjectAlternates(t *testing.T) {
	root := initGitRepository(t)
	alternates := filepath.Join(root, ".git", "objects", "info", "alternates")
	if err := os.WriteFile(alternates, []byte(t.TempDir()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "Git metadata pointer") {
		t.Fatalf("external object alternate error = %v", err)
	}
}

func TestGitRepoInspectRejectsLocalAndConditionalConfigIncludes(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	for _, config := range []string{
		"[include]\n\tpath = /tmp/attacker-git-config\n",
		"[include] # external config\n\tpath = /tmp/attacker-git-config\n",
		"[includeIf \"gitdir:/tmp/\"]\n\tpath = ../attacker-git-config\n",
		"[includeIf \"gitdir:/tmp/\"] ; conditional include\n\tpath = ../attacker-git-config\n",
		"[core]\n\tinclude.path = /tmp/attacker-git-config\n",
	} {
		t.Run(strings.ReplaceAll(strings.TrimSpace(config), "\n", "/"), func(t *testing.T) {
			root := initGitRepository(t)
			if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "include/includeIf") {
				t.Fatalf("Git config include error=%v, want fail-closed rejection", err)
			}
		})
	}

	root := initGitRepository(t)
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("safe config test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "README")
	runFixtureGit(t, root, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0\n# include.path = this-is-only-a-comment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err != nil {
		t.Fatalf("comment-only include mention rejected: %v", err)
	}
}

func TestGitRepoInspectRejectsConfiguredCleanFilterBeforeGitRuns(t *testing.T) {
	root := initGitRepository(t)
	tracked := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("safe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "tracked.txt")
	runFixtureGit(t, root, "commit", "-m", "initial")
	marker := filepath.Join(t.TempDir(), "git-clean-filter-executed")
	if err := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("*.txt filter=evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "config", "filter.evil.clean", "sh -c 'touch "+marker+"; cat'")
	if err := os.WriteFile(tracked, []byte("modified\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "filter configuration") {
		t.Fatalf("filter config error=%v, want fail-closed rejection", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repository filter command executed before rejection (stat error %v)", err)
	}
	configPath := filepath.Join(root, ".git", "config")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configText := strings.Replace(string(config), `[filter "evil"]`, `[filter.evil]`, 1)
	if configText == string(config) {
		t.Fatalf("test fixture did not contain Git filter subsection: %s", config)
	}
	if err := os.WriteFile(configPath, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), "external filter configuration") {
		t.Fatalf("dot-form filter config error=%v, want fail-closed rejection", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repository dot-form filter command executed before rejection (stat error %v)", err)
	}
}

func TestGitRepoInspectRejectsAliasesAndWorktreeRedirectBeforeGitRuns(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "git-alias-executed")
	outside := t.TempDir()
	for _, tc := range []struct {
		name   string
		config string
		want   string
	}{
		{name: "alias", config: "[alias]\n\tstatus = !touch " + marker + "\n", want: "Git aliases"},
		{name: "core worktree", config: "[core]\n\tworktree = " + outside + "\n", want: "core.worktree"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := initGitRepository(t)
			if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0\n"+tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsafe Git config error=%v, want rejection containing %q", err, tc.want)
			}
		})
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("repository-local alias command executed: %v", err)
	}
}

func TestParseGitConfigRejectsSectionWhitespaceAndSubsectionBypasses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		want   string
	}{
		{name: "filter tab subsection", config: "[filter\t\"evil\"]\n\tclean = !touch /tmp/filter\n", want: "filter configuration"},
		{name: "filter repeated whitespace", config: "[filter   \"evil\"]\n\tclean = !touch /tmp/filter\n", want: "filter configuration"},
		{name: "filter dot subsection", config: "[filter.evil]\n\tclean = !touch /tmp/filter\n", want: "filter configuration"},
		{name: "includeIf tab subsection", config: "[includeIf\t\"gitdir:/tmp/\"]\n\tpath = /tmp/attacker\n", want: "include/includeIf"},
		{name: "includeIf repeated whitespace", config: "[includeIf    \"gitdir:/tmp/\"]\n\tpath = /tmp/attacker\n", want: "include/includeIf"},
		{name: "alias tab subsection", config: "[alias\t\"evil\"]\n\tstatus = !touch /tmp/alias\n", want: "Git aliases"},
		{name: "alias space subsection", config: "[alias \"evil\"]\n\tstatus = !touch /tmp/alias\n", want: "Git aliases"},
		{name: "alias dot subsection", config: "[alias.evil]\n\tstatus = !touch /tmp/alias\n", want: "Git aliases"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseGitConfigForInspection([]byte(tc.config)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("config error=%v, want rejection containing %q", err, tc.want)
			}
		})
	}
}

func TestGitRepoInspectBoundsDiffOutput(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	root := initGitRepository(t)
	content := strings.Repeat("line of tracked content\n", 8000)
	path := filepath.Join(root, "large.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "large.txt")
	runFixtureGit(t, root, "commit", "-m", "large base")
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "line of tracked content", "different tracked content")), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{Workspace: root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	value := result.Value.(map[string]any)
	if value["truncated"] != true {
		t.Fatalf("large diff was not marked truncated: truncated=%v unstaged_bytes=%d staged_bytes=%d status_bytes=%d", value["truncated"], len(value["unstaged_diff"].(string)), len(value["staged_diff"].(string)), len(value["status"].(string)))
	}
	if got := len(value["unstaged_diff"].(string)); got > gitRepoOutputLimit {
		t.Fatalf("unstaged diff length %d exceeds limit %d", got, gitRepoOutputLimit)
	}
}

func initGitRepository(t *testing.T) string {
	t.Helper()
	return initGitRepositoryAt(t, t.TempDir())
}

func initGitRepositoryAt(t *testing.T, root string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "init", "-q")
	runFixtureGit(t, root, "config", "user.name", "Ollama Full test")
	runFixtureGit(t, root, "config", "user.email", "ollama-full-test@example.invalid")
	return root
}

func runFixtureGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestGitRepoInspectWithoutWorkspaceFailsClosed(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := (gitRepoInspectTool{}).Execute(context.Background(), ToolContext{WorkspaceRoot: root}, nil); err == nil || !strings.Contains(err.Error(), "workspace is required") {
		t.Fatalf("Git inspection without workspace path error = %v", err)
	}
}

func TestGitRepoCommandUsesOpenedWorkspaceAfterPathReplacement(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	initGitRepositoryAt(t, root)
	handle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	moved := filepath.Join(parent, "authorized-workspace")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	initGitRepositoryAt(t, root)
	output, _, err := runGitRepoCommandAtRoot(context.Background(), root, handle, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output); !samePath(got, moved) {
		t.Fatalf("Git inspected replacement path %q; want opened workspace %q", got, moved)
	}
}
