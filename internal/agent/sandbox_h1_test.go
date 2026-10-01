package agent

import (
	"runtime"
	"strings"
	"testing"
)

func TestSandboxBestEffortRequiresExplicitOperatorOptIn(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "best-effort")
	t.Setenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT", "")
	if _, err := configuredSandboxMode(); err == nil || !strings.Contains(err.Error(), "explicit approval notice") {
		t.Fatalf("best-effort without explicit operator opt-in must fail closed, got %v", err)
	}
}

func TestStrictSandboxLauncherUsesFailClosedAllowlistAndRlimits(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("seccomp-BPF launcher regression is Linux-specific; non-Linux execution is explicitly NOT_CONFIGURED")
	}
	for _, required := range []string{
		"SECCOMP_RET_KILL_PROCESS",
		"seccomp allowlist installation failed",
		"resource.RLIMIT_CPU",
		"resource.RLIMIT_AS",
		"resource.RLIMIT_NPROC",
		"resource.RLIMIT_NOFILE",
		"resource.RLIMIT_FSIZE",
		"resource.setrlimit",
		"os.execv",
	} {
		if !strings.Contains(strictSandboxLauncher, required) {
			t.Fatalf("strict launcher lost required hardening %q", required)
		}
	}
	if strings.Contains(strictSandboxLauncher, "SECCOMP_RET_ALLOW))\n") {
		t.Fatal("strict launcher must not use allow-all as its default action")
	}
	for _, forbidden := range []string{
		"ptrace", "mount", "umount2", "setns", "unshare", "keyctl", "bpf", "clone", "clone3",
	} {
		if strings.Contains(strictSandboxLauncher, `"`+forbidden+`"`) {
			t.Fatalf("strict launcher allowlist must not include dangerous syscall name %q", forbidden)
		}
	}
}

func TestStrictSandboxReportsNotConfiguredWhenCgroupIsUnavailable(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_SANDBOX_CGROUP_ROOT", t.TempDir())
	if _, err := newSandboxControl("h1_cgroup_unavailable"); err == nil {
		t.Fatal("strict sandbox must fail closed without a delegated cgroup v2 subtree")
	}
}
