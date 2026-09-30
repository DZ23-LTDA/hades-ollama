package agent

import (
	"testing"
)

func TestGateStatusNeverPassWithoutExecution(t *testing.T) {
	// Rule 1: executed=false with passed=true MUST NEVER be PASS; it MUST be NOT_EXECUTED.
	gate, err := EvaluateGate("gate_1", "coding", "worktree_isolation", false, true, true, false, "evidence/test.png", "not yet executed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.Status == GateStatusPass {
		t.Fatalf("VIOLATION: gate marked PASS when executed=false")
	}
	if gate.Status != GateStatusNotExecuted {
		t.Errorf("expected NOT_EXECUTED, got %s", gate.Status)
	}

	// Rule 2: executed=true, passed=true, but evidenceRef="" MUST fail with error.
	_, err = EvaluateGate("gate_2", "coding", "worktree_isolation", true, true, true, false, "", "passed without proof")
	if err == nil {
		t.Fatalf("expected error when evidenceRef is empty for PASS gate, got nil")
	}

	// Rule 3: executed=true, passed=true, evidenceRef valid -> PASS.
	gateValid, err := EvaluateGate("gate_3", "coding", "worktree_isolation", true, true, true, false, "docs/evidencias/test.png", "verified")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateValid.Status != GateStatusPass {
		t.Errorf("expected PASS, got %s", gateValid.Status)
	}

	// Rule 4: external blocked -> BLOCKED_EXTERNAL regardless of execution.
	gateBlocked, err := EvaluateGate("gate_4", "integrations", "tiktok_commerce", false, false, false, true, "", "ADR-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateBlocked.Status != GateStatusBlockedExternal {
		t.Errorf("expected BLOCKED_EXTERNAL, got %s", gateBlocked.Status)
	}

	// Rule 5: not configured -> NOT_CONFIGURED.
	gateUnconfigured, err := EvaluateGate("gate_5", "models", "codex_remote", false, false, false, false, "", "no API key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateUnconfigured.Status != GateStatusNotConfigured {
		t.Errorf("expected NOT_CONFIGURED, got %s", gateUnconfigured.Status)
	}

	// Rule 6: executed=true, passed=false -> FAIL.
	gateFailed, err := EvaluateGate("gate_6", "unit", "syntax_check", true, false, true, false, "", "failed tests")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateFailed.Status != GateStatusFail {
		t.Errorf("expected FAIL, got %s", gateFailed.Status)
	}
}

func TestGateMatrixRejectsFakePass(t *testing.T) {
	matrix := NewGateMatrix()

	// Attempting to inject a fake PASS directly into matrix without execution MUST be rejected.
	fakeGate := CapabilityGate{
		ID:          "fake_1",
		Domain:      "security",
		Capability:  "sandbox",
		Status:      GateStatusPass,
		Executed:    false, // NOT EXECUTED!
		EvidenceRef: "docs/fake.png",
	}
	err := matrix.Add(fakeGate)
	if err == nil {
		t.Fatalf("GateMatrix allowed fake PASS without execution")
	}

	// Valid gate
	validGate := CapabilityGate{
		ID:          "valid_1",
		Domain:      "security",
		Capability:  "sandbox",
		Status:      GateStatusPass,
		Executed:    true,
		EvidenceRef: "docs/evidencias/real.png",
	}
	if err := matrix.Add(validGate); err != nil {
		t.Fatalf("unexpected error adding valid gate: %v", err)
	}

	summary := matrix.Summary()
	if summary[GateStatusPass] != 1 {
		t.Errorf("expected 1 PASS, got %d", summary[GateStatusPass])
	}
}
