package agent

import (
	"fmt"
	"strings"
)

// CliSubscriptionModelInfo holds status and metadata for a CLI subscription model within the agent.
type CliSubscriptionModelInfo struct {
	ModelID     string     `json:"model_id"`
	Provider    string     `json:"provider"`
	DisplayName string     `json:"display_name"`
	Status      GateStatus `json:"status"`
	CostTag     string     `json:"cost_tag"`
	Selectable  bool       `json:"selectable"`
	Reason      string     `json:"reason,omitempty"`
	Action      string     `json:"action,omitempty"`
}

// KnownCliSubscriptionModels maps known model IDs to their CLI subscription provider.
var KnownCliSubscriptionModels = map[string]string{
	"claude-3-7-sonnet":         "claude_code",
	"claude-3-5-sonnet":         "claude_code",
	"claude-3-5-haiku":          "claude_code",
	"o3-mini":                   "codex",
	"gpt-4o":                    "codex",
	"gpt-4o-mini":               "codex",
	"gemini-2.5-pro":            "gemini",
	"gemini-2.5-flash":          "gemini",
	"gemini-2.0-flash":          "gemini",
	"copilot/claude-3.5-sonnet": "copilot",
	"copilot/gpt-4o":            "copilot",
	"copilot/o3-mini":           "copilot",
}

// IsCliSubscriptionModel returns true if the model name belongs to a CLI subscription.
func IsCliSubscriptionModel(modelName string) bool {
	norm := strings.TrimSpace(strings.ToLower(modelName))
	_, exists := KnownCliSubscriptionModels[norm]
	return exists
}

// ValidateCliSubscriptionSelection checks whether a CLI subscription model can be used by an agent mission.
func ValidateCliSubscriptionSelection(modelName string, status GateStatus, reason, action string) (CliSubscriptionModelInfo, error) {
	norm := strings.TrimSpace(strings.ToLower(modelName))
	provider, exists := KnownCliSubscriptionModels[norm]
	if !exists {
		provider = "unknown"
	}

	info := CliSubscriptionModelInfo{
		ModelID:    modelName,
		Provider:   provider,
		Status:     status,
		CostTag:    "0-assinatura",
		Selectable: status == GateStatusPass,
		Reason:     reason,
		Action:     action,
	}

	if status != GateStatusPass {
		msg := fmt.Sprintf("modelo de assinatura %q (provedor %s) não pode ser selecionado com status honesto %s", modelName, provider, status)
		if reason != "" {
			msg += fmt.Sprintf(": %s", reason)
		}
		if action != "" {
			msg += fmt.Sprintf(" (ação requerida: %s)", action)
		}
		return info, fmt.Errorf("%s", msg)
	}

	return info, nil
}

// ResolveAgentModelCost returns the cost description for display and accounting.
func ResolveAgentModelCost(modelName string, isSubscription bool) string {
	if isSubscription || IsCliSubscriptionModel(modelName) {
		return "0-assinatura"
	}
	return "padrao"
}
