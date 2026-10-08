//go:build linux

package agent

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

type sandboxControl struct {
	dir  string
	file *os.File
}

// trustedSandboxExecutable deliberately does not consult the inherited PATH.
// The absolute result must also be passed to exec.Command: Command resolves
// bare names before command.Env is applied.
func trustedSandboxExecutable(name string) (string, error) {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return "", errors.New("sandbox helper name is invalid")
	}
	for _, directory := range []string{"/usr/bin", "/usr/sbin", "/bin"} {
		candidate := filepath.Join(directory, name)
		info, err := os.Lstat(candidate)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("sandbox helper %q was not found in trusted system directories", name)
}

func newSandboxControl(stepID string) (*sandboxControl, error) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return nil, errors.New("strict sandbox seccomp policy is unavailable for this architecture")
	}
	root := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_CGROUP_ROOT"))
	if root == "" {
		return nil, errors.New("strict sandbox requires OLLAMA_AGENT_SANDBOX_CGROUP_ROOT")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("sandbox cgroup root is invalid: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("sandbox cgroup root is unavailable: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("sandbox cgroup root must be a real directory")
	}
	controllers, err := os.ReadFile(filepath.Join(root, "cgroup.controllers"))
	if err != nil {
		return nil, fmt.Errorf("sandbox cgroup controllers are unavailable: %w", err)
	}
	for _, controller := range []string{"cpu", "memory", "pids"} {
		if !strings.Contains(" "+string(controllers)+" ", " "+controller+" ") {
			return nil, fmt.Errorf("sandbox cgroup controller %q is not delegated", controller)
		}
	}
	for _, name := range []string{"memory.max", "pids.max", "cpu.max"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return nil, fmt.Errorf("sandbox cgroup file %s is unavailable: %w", name, err)
		}
	}
	if _, err := trustedSandboxExecutable("unshare"); err != nil {
		return nil, fmt.Errorf("strict sandbox requires unshare: %w", err)
	}
	if _, err := trustedSandboxExecutable("setpriv"); err != nil {
		return nil, fmt.Errorf("strict sandbox requires setpriv: %w", err)
	}
	prefix := "ollama-" + sanitizeMCPID(stepID) + "-"
	dir, err := os.MkdirTemp(root, prefix)
	if err != nil {
		return nil, fmt.Errorf("create strict sandbox cgroup: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	for name, value := range map[string]string{
		"cpu.max":         "55000 100000\n",
		"memory.max":      "536870912\n",
		"pids.max":        "64\n",
		"memory.swap.max": "0\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600); err != nil {
			if !errors.Is(err, os.ErrNotExist) || name != "memory.swap.max" {
				cleanup()
				return nil, fmt.Errorf("configure strict sandbox cgroup %s: %w", name, err)
			}
		}
	}
	file, err := os.Open(dir)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("open strict sandbox cgroup: %w", err)
	}
	return &sandboxControl{dir: dir, file: file}, nil
}

func configureSandboxCommand(command *exec.Cmd, control *sandboxControl) error {
	if control == nil || control.file == nil {
		return nil
	}
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.Pdeathsig = syscall.SIGKILL
	command.SysProcAttr.UseCgroupFD = true
	command.SysProcAttr.CgroupFD = int(control.file.Fd())
	return nil
}

func closeSandboxControl(control *sandboxControl) {
	if control == nil {
		return
	}
	if control.file != nil {
		_ = control.file.Close()
	}
	_ = os.RemoveAll(control.dir)
}

func killSandboxControl(control *sandboxControl) {
	if control == nil {
		return
	}
	// Falha ao matar o cgroup deixa o sandbox em execução: registre em vez de
	// descartar em silêncio.
	path := filepath.Join(control.dir, "cgroup.kill")
	if err := os.WriteFile(path, []byte("1\n"), 0o600); err != nil {
		slog.Error("failed to kill sandbox cgroup", "path", path, "error", err)
	}
}
