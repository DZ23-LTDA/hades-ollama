package agent

import (
	"testing"
)

func TestModelListOnlyPassIsSelectable(t *testing.T) {
	gate, err := EvaluateGate("model-local", "models", "local-execution", true, true, true, false, "docs/evidencias/local.png", "local ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.Status != GateStatusPass {
		t.Fatalf("expected GateStatusPass, got: %s", gate.Status)
	}

	gateUnconfigured, err := EvaluateGate("model-remote", "models", "remote-api", false, false, false, false, "", "missing key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateUnconfigured.Status != GateStatusNotConfigured {
		t.Fatalf("expected GateStatusNotConfigured, got: %s", gateUnconfigured.Status)
	}
}

func TestModelListExcludesUnreachable(t *testing.T) {
	gateUnreachable, err := EvaluateGate("model-down", "models", "remote-api", true, false, true, false, "", "network timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateUnreachable.Status != GateStatusFail {
		t.Fatalf("expected GateStatusFail, got: %s", gateUnreachable.Status)
	}

	matrix := NewGateMatrix()
	if err := matrix.Add(gateUnreachable); err != nil {
		t.Fatalf("failed to add gate: %v", err)
	}

	summary := matrix.Summary()
	if summary[GateStatusFail] != 1 {
		t.Fatalf("expected 1 FAIL in summary, got: %d", summary[GateStatusFail])
	}
}
