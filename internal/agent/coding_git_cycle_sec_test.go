package agent

import (
	"context"
	"path/filepath"
	"testing"
)

// TestProjectTestRunDescriptorRequiresApproval locks in SEC-01: running the
// project's test command executes arbitrary project code on the host, so the
// tool must be an external-side-effect action that requires human approval and
// the sandbox:execute scope — never an auto-run RiskRead tool.
func TestProjectTestRunDescriptorRequiresApproval(t *testing.T) {
	d := projectTestRunnerTool{}.Descriptor()
	if !d.RequiresApproval {
		t.Error("project.test.run must require approval (SEC-01)")
	}
	if d.Risk != RiskExternalSideEffect {
		t.Errorf("project.test.run risk = %v, want RiskExternalSideEffect", d.Risk)
	}
	hasExec := false
	for _, s := range d.Scopes {
		if s == "sandbox:execute" {
			hasExec = true
		}
	}
	if !hasExec {
		t.Errorf("project.test.run must require sandbox:execute, got scopes %v", d.Scopes)
	}
}

// TestRunProjectTestsRejectsSubPathEscape locks in SEC-02: a sub_path that is
// absolute or escapes the workspace (via "..") must be rejected before any
// command runs; a normal relative subpath is accepted.
func TestRunProjectTestsRejectsSubPathEscape(t *testing.T) {
	ws := t.TempDir()

	for _, sub := range []string{
		"../escape",
		filepath.Join("..", "..", "etc"),
		ws, // absolute path, even inside, is not a valid relative sub_path
	} {
		if _, err := RunProjectTests(context.Background(), ws, TestRunOptions{SubPath: sub}); err == nil {
			t.Errorf("sub_path %q should be rejected as escaping/absolute", sub)
		}
	}

	// A plain relative subpath is contained and accepted (no runner detected is
	// a normal, non-error result).
	if _, err := RunProjectTests(context.Background(), ws, TestRunOptions{SubPath: "subdir"}); err != nil {
		t.Errorf("valid relative sub_path should be accepted, got: %v", err)
	}
}
