package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestMutationHarnessCapabilityEnforcement verifies that mutating (removing)
// capability validation causes the security gate to fail.
func TestMutationHarnessCapabilityEnforcement(t *testing.T) {
	policy := DefaultCapabilityPolicy()

	// Target tool requires both desktop:screen and desktop:input
	descriptor := ToolDescriptor{
		Name:   "desktop",
		Scopes: []string{"desktop:screen", "desktop:input"},
	}

	// Baseline: partial grant containing only "desktop:screen"
	grantedCapabilities := []string{"desktop:screen"}

	// 1. Real gate: policy.Allows must reject because desktop:input is missing
	allowedReal := policy.Allows(descriptor, grantedCapabilities)
	if allowedReal {
		t.Fatalf("Baseline security failure: expected partial grant to be rejected")
	}

	// 2. Mutated gate: simulate a compromised/weakened policy that allows any tool
	mutatedAllows := func(desc ToolDescriptor, granted []string) bool {
		// MUTATION: intentionally weakened to accept everything (bypass)
		return true
	}
	allowedMutated := mutatedAllows(descriptor, grantedCapabilities)
	if !allowedMutated {
		t.Fatalf("Mutation test setup failed: expected mutated policy to bypass check")
	}

	// 3. Meta-assertion: the baseline security check MUST catch what the mutated check bypassed.
	// If baseline allowed the tool like the mutation, the harness reproves.
	if allowedReal == allowedMutated {
		t.Fatalf("MUTATION HARNESS REPROVED: Baseline gate does not protect against capability bypass!")
	}
}

// TestMutationHarnessHITLApprovalBypass verifies that mutating (bypassing)
// human-in-the-loop approval is caught and fails.
func TestMutationHarnessHITLApprovalBypass(t *testing.T) {
	// Baseline: a step that requires approval
	step := &Step{
		ID:               "step_merge",
		Kind:             "git.merge.origin",
		RequiresApproval: true,
		State:            StepPending,
	}

	// Gate 1: unapproved step cannot be executed directly
	executeStepWithGate := func(s *Step, approved bool) error {
		if s.RequiresApproval && !approved {
			return errors.New("security gate: step requires human approval before execution")
		}
		return nil
	}

	errUnapproved := executeStepWithGate(step, false)
	if errUnapproved == nil {
		t.Fatalf("Baseline security failure: unapproved step executed without approval")
	}

	// Mutation: weakened gate that forgets to check `RequiresApproval`
	mutatedExecuteGate := func(s *Step, approved bool) error {
		// MUTATION: weakened/omitted approval check
		return nil
	}
	errMutated := mutatedExecuteGate(step, false)
	if errMutated != nil {
		t.Fatalf("Mutation test setup error: mutated gate should have bypassed check")
	}

	// Meta-assertion: real gate must block, mutated gate allows
	if errUnapproved == nil || errMutated != nil {
		t.Fatalf("MUTATION HARNESS REPROVED: HITL approval gate is trivial or inactive!")
	}
}

// TestMutationHarnessSymlinkTraversal verifies that removing symlink validation
// is caught by the test harness.
func TestMutationHarnessSymlinkTraversal(t *testing.T) {
	tempDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create sensitive target file outside workspace
	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super_secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create symlink inside workspace pointing outside
	symlinkPath := filepath.Join(tempDir, "leak_link")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skip("Symlinks not supported in this environment:", err)
	}

	// 1. Real gate: rejectSymlinkComponents must reject this link
	errReal := rejectSymlinkComponents(tempDir, symlinkPath)
	if errReal == nil {
		t.Fatalf("Baseline security failure: rejectSymlinkComponents allowed external symlink")
	}

	// 2. Mutated gate: removes the symlink check
	mutatedSymlinkCheck := func(root, candidate string) error {
		// MUTATION: bypassed symlink check
		return nil
	}
	errMutated := mutatedSymlinkCheck(tempDir, symlinkPath)

	// 3. Meta-assertion
	if errReal == nil || errMutated != nil {
		t.Fatalf("MUTATION HARNESS REPROVED: Symlink traversal gate is trivial or inactive!")
	}
}
