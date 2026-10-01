package agent

import (
	"runtime"
	"testing"
)

// requireDescriptorBoundWorkspaceIsolation skips tests that depend on
// descriptor-bound Git repository discovery and metadata views. Those
// primitives exist only on Linux (see workspace_snapshot_root_linux.go and
// git_read_view_linux.go); the non-Linux implementation deliberately fails
// closed with "unsupported on this platform". The Windows CI job runs the
// whole agent package, so these isolation regressions must be skipped there
// instead of failing on a capability the platform intentionally lacks.
func requireDescriptorBoundWorkspaceIsolation(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-bound workspace isolation requires Linux")
	}
}

// requireLinuxSandboxExecutor skips tests that exercise sandbox.exec. The tool
// is intentionally Linux-only ("host-process fallback is disabled"), so its
// approval binding and interpreter resolution cannot be asserted on Windows.
func requireLinuxSandboxExecutor(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("sandbox.exec requires the Linux namespace executor")
	}
}
