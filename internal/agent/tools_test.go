package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTerminalToolRedactsStderrAndReportsBestEffortIsolation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-bound terminal execution is Linux-only")
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "visible.txt"), []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	result, err := (terminalExecTool{allowed: map[string]bool{"ls": true}}).Execute(context.Background(), ToolContext{Workspace: workspace, WorkspaceRoot: root}, map[string]any{"executable": "ls"})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result value = %#v", result.Value)
	}
	stdout, _ := value["stdout"].(string)
	if !strings.Contains(stdout, "visible.txt") {
		t.Fatalf("pinned terminal output = %q", stdout)
	}
	if value["execution_isolation"] != "descriptor-rooted-read" {
		t.Fatalf("execution isolation = %#v", value["execution_isolation"])
	}
}

func TestTerminalGitIgnoresAmbientPATHReplacement(t *testing.T) {
	fakeDir := t.TempDir()
	marker := filepath.Join(fakeDir, "executed")
	fakeGit := filepath.Join(fakeDir, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nprintf executed > \"$OLLAMA_TEST_GIT_MARKER\"\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_TEST_GIT_MARKER", marker)
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := (terminalExecTool{allowed: map[string]bool{"git": true}}).Execute(context.Background(), ToolContext{Workspace: t.TempDir()}, map[string]any{
		"executable": "git",
		"args":       []string{"status"},
	})
	if err == nil || !strings.Contains(err.Error(), "git.repo.inspect") {
		t.Fatalf("terminal Git was not denied in favor of the dedicated inspector: %v", err)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("ambient fake git executed, marker stat error=%v", statErr)
	}
}

func TestTerminalUsesPinnedWorkspaceAfterPathReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor cwd regression is Linux-only")
	}
	parent := t.TempDir()
	workspace := filepath.Join(parent, "workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "authorized.txt"), []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	moved := filepath.Join(parent, "authorized-original")
	if err := os.Rename(workspace, moved); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "foreign-secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, workspace); err != nil {
		t.Fatal(err)
	}
	result, err := (terminalExecTool{allowed: map[string]bool{"ls": true}}).Execute(context.Background(), ToolContext{Workspace: workspace, WorkspaceRoot: root}, map[string]any{"executable": "ls"})
	if err != nil {
		t.Fatal(err)
	}
	stdout := result.Value.(map[string]any)["stdout"].(string)
	if !strings.Contains(stdout, "authorized.txt") || strings.Contains(stdout, "foreign-secret.txt") {
		t.Fatalf("terminal escaped pinned workspace: %q", stdout)
	}
}

func TestTerminalFailsClosedWithoutPinnedWorkspace(t *testing.T) {
	_, err := (terminalExecTool{allowed: map[string]bool{"pwd": true}}).Execute(context.Background(), ToolContext{Workspace: t.TempDir()}, map[string]any{"executable": "pwd"})
	if err == nil || !strings.Contains(err.Error(), "pinned Linux workspace") {
		t.Fatalf("unanchored terminal execution error = %v", err)
	}
}

// sleepHelperEnv ativa o processo auxiliar usado pelo teste de cancelamento.
const sleepHelperEnv = "HADES_TEST_SLEEP_HELPER"

// TestSleepHelperProcess NÃO testa nada por si: quando a variável de ambiente está
// definida, ele dorme 10 segundos para servir de processo-alvo do teste de
// cancelamento. Sem a variável, ele é ignorado.
//
// O teste de cancelamento antes dependia do `sh` do sistema, o que o tornava
// **impossível de verificar nesta máquina** (sem `sh` no PATH) e diferente entre host e
// CI. Usar o PRÓPRIO binário de teste como alvo deixa o teste portátil e verificável.
func TestSleepHelperProcess(t *testing.T) {
	if os.Getenv(sleepHelperEnv) != "1" {
		t.Skip("processo auxiliar do teste de cancelamento")
	}
	time.Sleep(10 * time.Second)
}

func TestRunToolCommandKillsProcessGroupOnCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// O alvo é o binário de teste em modo auxiliar: portátil em qualquer sistema, sem
	// depender de `sh`, `sleep` ou de utilitários do host.
	command := exec.Command(os.Args[0], "-test.run=TestSleepHelperProcess")
	command.Env = append(os.Environ(), sleepHelperEnv+"=1")
	configureToolProcess(command)
	started := time.Now()
	err := runToolCommand(ctx, command)
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("runToolCommand error = %v", err)
	}
	// A PROPRIEDADE é o cancelamento ser EFETIVO: o processo de 10 segundos precisa ser
	// morto bem antes de terminar sozinho. O limite é metade do próprio sono, então
	// continua reprovando qualquer cancelamento que apenas "espere" o processo.
	//
	// O limite era 2s e o runner Windows de CI o excedeu (2,37s em
	// `dz23-agentic-quality`, run 38125950314): derrubar o grupo de tarefas no Windows
	// depende do sistema e custa mais sob carga. Ligar o limite à propriedade medida
	// mantém a verificação e remove um número que media a carga da máquina.
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancellation took too long (o processo de 10s deveria ter sido morto bem antes): %s", elapsed)
	}
}

func TestTerminalArgumentPolicyRejectsEscapeAndUnsupportedFlags(t *testing.T) {
	workspace := t.TempDir()
	if err := validateTerminalArguments("ls", []string{"."}, workspace); err != nil {
		t.Fatal(err)
	}
	if err := validateTerminalArguments("ls", []string{"../"}, workspace); err == nil {
		t.Fatal("expected ls path escape rejection")
	}
	if err := validateTerminalArguments("ls", []string{"--color=always"}, workspace); err == nil {
		t.Fatal("expected unsupported ls flag rejection")
	}
	if err := validateTerminalArguments("pwd", []string{"."}, workspace); err == nil {
		t.Fatal("expected pwd argument rejection")
	}
	if err := validateTerminalArguments("git", []string{"status", "--short"}, workspace); err == nil {
		t.Fatal("expected git argument rejection")
	}
}

func TestSandboxStrictModeFailsClosedWithoutDelegatedCgroup(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_SANDBOX_CGROUP_ROOT", t.TempDir())
	_, err := newSandboxControl("step_strict_test")
	if err == nil {
		t.Fatalf("strict sandbox error = %v", err)
	}
}

func TestSandboxRejectsUnknownMode(t *testing.T) {
	requireLinuxSandboxExecutor(t)
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "unsafe")
	_, err := (sandboxExecTool{}).Execute(context.Background(), ToolContext{Workspace: t.TempDir(), StepID: "step_mode_test"}, map[string]any{
		"language": "python",
		"code":     "print('must not run')",
	})
	if err == nil || !strings.Contains(err.Error(), "must be best-effort or strict") {
		t.Fatalf("unknown sandbox mode error = %v", err)
	}
}

func TestResolveSandboxInterpreterUsesSupportedLanguageOnly(t *testing.T) {
	requireLinuxSandboxExecutor(t)
	resolved, err := resolveSandboxInterpreter("python")
	if err != nil || resolved == "" {
		t.Fatalf("python interpreter=%q err=%v", resolved, err)
	}
	if _, err := resolveSandboxInterpreter("ruby"); err == nil {
		t.Fatal("expected unsupported language rejection")
	}
}
