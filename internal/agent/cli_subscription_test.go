package agent

import (
	"strings"
	"testing"
)

func TestCliSubscriptionSelection(t *testing.T) {
	// Test that PASS allows selection with cost tag "0-assinatura"
	t.Run("StatusPassPermiteSelecaoComCustoZero", func(t *testing.T) {
		info, err := ValidateCliSubscriptionSelection("claude-3-7-sonnet", GateStatusPass, "", "")
		if err != nil {
			t.Fatalf("expected nil error for PASS status, got %v", err)
		}
		if !info.Selectable {
			t.Fatalf("expected selectable true for PASS status")
		}
		if info.CostTag != "0-assinatura" {
			t.Fatalf("expected cost tag '0-assinatura', got %q", info.CostTag)
		}
		if info.Provider != "claude_code" {
			t.Fatalf("expected provider 'claude_code', got %q", info.Provider)
		}
	})

	// Test that NOT_CONFIGURED rejects selection and includes login action
	t.Run("StatusNotConfiguredRejeitaSelecaoComAcaoDeLogin", func(t *testing.T) {
		info, err := ValidateCliSubscriptionSelection("gpt-4o", GateStatusNotConfigured, "Sessão não autenticada", "fazer login com 'codex auth login'")
		if err == nil {
			t.Fatalf("expected error for NOT_CONFIGURED status, got nil")
		}
		if info.Selectable {
			t.Fatalf("expected selectable false for NOT_CONFIGURED status")
		}
		if !strings.Contains(err.Error(), "fazer login") {
			t.Fatalf("expected error to contain login instruction, got %q", err.Error())
		}
	})

	// Test that NOT_PRESENT rejects selection and includes install action
	t.Run("StatusNotPresentRejeitaSelecaoComAcaoDeInstalar", func(t *testing.T) {
		info, err := ValidateCliSubscriptionSelection("gemini-2.5-pro", GateStatusNotPresent, "CLI não encontrada", "npm install -g @google/gemini-cli")
		if err == nil {
			t.Fatalf("expected error for NOT_PRESENT status, got nil")
		}
		if info.Selectable {
			t.Fatalf("expected selectable false for NOT_PRESENT status")
		}
		if !strings.Contains(err.Error(), "npm install") {
			t.Fatalf("expected error to contain install instruction, got %q", err.Error())
		}
	})

	// Test that Copilot models are recognized
	t.Run("CopilotModelIsRecognized", func(t *testing.T) {
		if !IsCliSubscriptionModel("copilot/claude-3.5-sonnet") {
			t.Fatalf("expected copilot/claude-3.5-sonnet to be recognized as CLI subscription model")
		}
		info, err := ValidateCliSubscriptionSelection("copilot/claude-3.5-sonnet", GateStatusPass, "", "")
		if err != nil {
			t.Fatalf("expected nil error for PASS status, got %v", err)
		}
		if info.CostTag != "0-assinatura" {
			t.Fatalf("expected cost tag '0-assinatura', got %q", info.CostTag)
		}
	})
}
