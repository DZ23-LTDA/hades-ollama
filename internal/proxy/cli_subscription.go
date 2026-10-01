package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// CliSubscriptionProvider identifies an external CLI subscription provider.
type CliSubscriptionProvider string

const (
	CliProviderClaudeCode CliSubscriptionProvider = "claude_code"
	CliProviderCodex      CliSubscriptionProvider = "codex"
	CliProviderGemini     CliSubscriptionProvider = "gemini"
	CliProviderCopilot    CliSubscriptionProvider = "copilot"
)

// Standard GateStatus values matching internal/agent/gate_status.go.
const (
	GateStatusPass          = "PASS"
	GateStatusNotConfigured = "NOT_CONFIGURED"
	GateStatusNotPresent    = "NOT_PRESENT"
	CostTagSubscription     = "0-assinatura"
)

// CliSubscriptionModel represents a model provided under a CLI subscription.
type CliSubscriptionModel struct {
	ID          string                  `json:"id"`
	DisplayName string                  `json:"display_name"`
	Provider    CliSubscriptionProvider `json:"provider"`
}

// CliSubscriptionState represents the honest evaluation state for a CLI tool.
type CliSubscriptionState struct {
	Provider    CliSubscriptionProvider `json:"provider"`
	DisplayName string                  `json:"display_name"`
	Binary      string                  `json:"binary"`
	BinaryPath  string                  `json:"binary_path,omitempty"`
	Status      string                  `json:"status"` // GateStatus: PASS, NOT_CONFIGURED, NOT_PRESENT
	CostTag     string                  `json:"cost_tag"`
	Selectable  bool                    `json:"selectable"`
	Reason      string                  `json:"reason,omitempty"`
	Action      string                  `json:"action,omitempty"`
	Models      []CliSubscriptionModel  `json:"models"`
}

// CliCatalogEntry represents a single model entry formatted for client consumption.
type CliCatalogEntry struct {
	ID          string                  `json:"id"`
	DisplayName string                  `json:"display_name"`
	Provider    CliSubscriptionProvider `json:"provider"`
	Status      string                  `json:"status"`
	CostTag     string                  `json:"cost_tag"`
	Selectable  bool                    `json:"selectable"`
	Reason      string                  `json:"reason,omitempty"`
	Action      string                  `json:"action,omitempty"`
}

// SystemInspector provides hooks for filesystem, environment and process queries.
type SystemInspector struct {
	LookPath func(file string) (string, error)
	Stat     func(name string) (os.FileInfo, error)
	ReadFile func(name string) ([]byte, error)
	Getenv   func(key string) string
	UserHome func() (string, error)
}

func defaultInspector() *SystemInspector {
	return &SystemInspector{
		LookPath: exec.LookPath,
		Stat:     os.Stat,
		ReadFile: os.ReadFile,
		Getenv:   os.Getenv,
		UserHome: os.UserHomeDir,
	}
}

// DefaultModels returns the default available models for each supported CLI subscription.
func DefaultModels(provider CliSubscriptionProvider) []CliSubscriptionModel {
	switch provider {
	case CliProviderClaudeCode:
		return []CliSubscriptionModel{
			{ID: "claude-3-7-sonnet", DisplayName: "Claude 3.7 Sonnet (Assinatura CLI)", Provider: CliProviderClaudeCode},
			{ID: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet (Assinatura CLI)", Provider: CliProviderClaudeCode},
			{ID: "claude-3-5-haiku", DisplayName: "Claude 3.5 Haiku (Assinatura CLI)", Provider: CliProviderClaudeCode},
		}
	case CliProviderCodex:
		return []CliSubscriptionModel{
			{ID: "o3-mini", DisplayName: "o3-mini (Assinatura CLI)", Provider: CliProviderCodex},
			{ID: "gpt-4o", DisplayName: "GPT-4o (Assinatura CLI)", Provider: CliProviderCodex},
			{ID: "gpt-4o-mini", DisplayName: "GPT-4o mini (Assinatura CLI)", Provider: CliProviderCodex},
		}
	case CliProviderGemini:
		return []CliSubscriptionModel{
			{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro (Assinatura CLI)", Provider: CliProviderGemini},
			{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash (Assinatura CLI)", Provider: CliProviderGemini},
			{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash (Assinatura CLI)", Provider: CliProviderGemini},
		}
	case CliProviderCopilot:
		return []CliSubscriptionModel{
			{ID: "copilot/claude-3.5-sonnet", DisplayName: "Copilot Claude 3.5 Sonnet (Assinatura CLI)", Provider: CliProviderCopilot},
			{ID: "copilot/gpt-4o", DisplayName: "Copilot GPT-4o (Assinatura CLI)", Provider: CliProviderCopilot},
			{ID: "copilot/o3-mini", DisplayName: "Copilot o3-mini (Assinatura CLI)", Provider: CliProviderCopilot},
		}
	default:
		return nil
	}
}

// CliSubscriptionDetector performs detection and model resolution.
type CliSubscriptionDetector struct {
	inspector *SystemInspector
	mu        sync.RWMutex //nolint:unused // compatibility/security surface retained for future adapter wiring
}

// NewCliSubscriptionDetector creates a detector with the given inspector (or default).
func NewCliSubscriptionDetector(insp *SystemInspector) *CliSubscriptionDetector {
	if insp == nil {
		insp = defaultInspector()
	}
	return &CliSubscriptionDetector{
		inspector: insp,
	}
}

var (
	defaultDetectorInstance *CliSubscriptionDetector
	defaultDetectorOnce     sync.Once
)

// GetDefaultCliSubscriptionDetector returns the singleton detector for production.
func GetDefaultCliSubscriptionDetector() *CliSubscriptionDetector {
	defaultDetectorOnce.Do(func() {
		defaultDetectorInstance = NewCliSubscriptionDetector(nil)
	})
	return defaultDetectorInstance
}

// Detect evaluates the state for a single provider.
func (d *CliSubscriptionDetector) Detect(ctx context.Context, provider CliSubscriptionProvider) CliSubscriptionState {
	insp := d.inspector
	if insp == nil {
		insp = defaultInspector()
	}

	models := DefaultModels(provider)
	switch provider {
	case CliProviderClaudeCode:
		return d.detectClaude(insp, models)
	case CliProviderCodex:
		return d.detectCodex(insp, models)
	case CliProviderGemini:
		return d.detectGemini(insp, models)
	case CliProviderCopilot:
		return d.detectCopilot(insp, models)
	default:
		return CliSubscriptionState{
			Provider:   provider,
			Status:     GateStatusNotPresent,
			CostTag:    CostTagSubscription,
			Selectable: false,
			Reason:     fmt.Sprintf("Provedor CLI desconhecido: %s", provider),
			Action:     "Configurar provedor suportado",
		}
	}
}

// DetectAll returns detection states for all supported CLI providers.
func (d *CliSubscriptionDetector) DetectAll(ctx context.Context) []CliSubscriptionState {
	providers := []CliSubscriptionProvider{
		CliProviderClaudeCode,
		CliProviderCodex,
		CliProviderGemini,
		CliProviderCopilot,
	}
	results := make([]CliSubscriptionState, 0, len(providers))
	for _, p := range providers {
		results = append(results, d.Detect(ctx, p))
	}
	return results
}

// GetUsableModels returns only models from subscriptions that have PASS status.
func (d *CliSubscriptionDetector) GetUsableModels(ctx context.Context) []CliSubscriptionModel {
	all := d.DetectAll(ctx)
	var usable []CliSubscriptionModel
	for _, state := range all {
		if state.Status == GateStatusPass && state.Selectable {
			usable = append(usable, state.Models...)
		}
	}
	return usable
}

// GetCatalogEntries returns all models with their GateStatus, cost tag, and availability.
func (d *CliSubscriptionDetector) GetCatalogEntries(ctx context.Context) []CliCatalogEntry {
	all := d.DetectAll(ctx)
	var entries []CliCatalogEntry
	for _, state := range all {
		for _, m := range state.Models {
			entries = append(entries, CliCatalogEntry{
				ID:          m.ID,
				DisplayName: m.DisplayName,
				Provider:    state.Provider,
				Status:      state.Status,
				CostTag:     state.CostTag,
				Selectable:  state.Selectable,
				Reason:      state.Reason,
				Action:      state.Action,
			})
		}
	}
	return entries
}

// Detection logic for Claude Code
func (d *CliSubscriptionDetector) detectClaude(insp *SystemInspector, models []CliSubscriptionModel) CliSubscriptionState {
	state := CliSubscriptionState{
		Provider:    CliProviderClaudeCode,
		DisplayName: "Claude Code",
		Binary:      "claude",
		CostTag:     CostTagSubscription,
		Models:      models,
	}

	binPath := d.findBinary(insp, "claude", []string{
		filepath.Join(".local", "bin", "claude"),
		filepath.Join(".claude", "local", "claude"),
	})
	if binPath == "" {
		state.Status = GateStatusNotPresent
		state.Selectable = false
		state.Reason = "CLI 'claude' não encontrada no sistema"
		state.Action = "npm install -g @anthropic-ai/claude-code"
		return state
	}
	state.BinaryPath = binPath

	// Check authentication
	if insp.Getenv("ANTHROPIC_API_KEY") != "" || insp.Getenv("CLAUDE_SESSION_KEY") != "" {
		state.Status = GateStatusPass
		state.Selectable = true
		return state
	}

	home, _ := insp.UserHome()
	if home != "" {
		for _, rel := range []string{
			".claude.json",
			filepath.Join(".claude", "auth.json"),
			filepath.Join(".claude", "session.json"),
			filepath.Join(".claude", "credentials.json"),
		} {
			p := filepath.Join(home, rel)
			if fi, err := insp.Stat(p); err == nil && fi.Size() > 2 {
				if d.hasValidJSONContent(insp, p) {
					state.Status = GateStatusPass
					state.Selectable = true
					return state
				}
			}
		}
	}

	state.Status = GateStatusNotConfigured
	state.Selectable = false
	state.Reason = "Sessão da CLI 'claude' não autenticada"
	state.Action = "fazer login com 'claude login'"
	return state
}

// Detection logic for Codex / ChatGPT
func (d *CliSubscriptionDetector) detectCodex(insp *SystemInspector, models []CliSubscriptionModel) CliSubscriptionState {
	state := CliSubscriptionState{
		Provider:    CliProviderCodex,
		DisplayName: "Codex / ChatGPT",
		Binary:      "codex",
		CostTag:     CostTagSubscription,
		Models:      models,
	}

	binPath := d.findBinary(insp, "codex", []string{
		filepath.Join(".local", "bin", "codex"),
		filepath.Join(".codex", "bin", "codex"),
	})
	if binPath == "" {
		// Fallback check for 'chatgpt'
		binPath = d.findBinary(insp, "chatgpt", []string{
			filepath.Join(".local", "bin", "chatgpt"),
		})
	}

	if binPath == "" {
		state.Status = GateStatusNotPresent
		state.Selectable = false
		state.Reason = "CLI 'codex' não encontrada no sistema"
		state.Action = "npm install -g @openai/codex"
		return state
	}
	state.BinaryPath = binPath

	// Check authentication
	if insp.Getenv("OPENAI_API_KEY") != "" || insp.Getenv("CODEX_API_KEY") != "" {
		state.Status = GateStatusPass
		state.Selectable = true
		return state
	}

	home, _ := insp.UserHome()
	if home != "" {
		for _, rel := range []string{
			filepath.Join(".codex", "auth.json"),
			filepath.Join(".chatgpt", "session.json"),
			filepath.Join(".openai", "credentials.json"),
		} {
			p := filepath.Join(home, rel)
			if fi, err := insp.Stat(p); err == nil && fi.Size() > 2 {
				if d.hasValidJSONContent(insp, p) {
					state.Status = GateStatusPass
					state.Selectable = true
					return state
				}
			}
		}
	}

	state.Status = GateStatusNotConfigured
	state.Selectable = false
	state.Reason = "Sessão da CLI 'codex' não autenticada"
	state.Action = "fazer login com 'codex auth login'"
	return state
}

// Detection logic for Gemini CLI
func (d *CliSubscriptionDetector) detectGemini(insp *SystemInspector, models []CliSubscriptionModel) CliSubscriptionState {
	state := CliSubscriptionState{
		Provider:    CliProviderGemini,
		DisplayName: "Gemini CLI",
		Binary:      "gemini",
		CostTag:     CostTagSubscription,
		Models:      models,
	}

	binPath := d.findBinary(insp, "gemini", []string{
		filepath.Join(".local", "bin", "gemini"),
	})
	if binPath == "" {
		state.Status = GateStatusNotPresent
		state.Selectable = false
		state.Reason = "CLI 'gemini' não encontrada no sistema"
		state.Action = "npm install -g @google/gemini-cli"
		return state
	}
	state.BinaryPath = binPath

	// Check authentication
	if insp.Getenv("GEMINI_API_KEY") != "" || insp.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		state.Status = GateStatusPass
		state.Selectable = true
		return state
	}

	home, _ := insp.UserHome()
	if home != "" {
		for _, rel := range []string{
			filepath.Join(".config", "gemini", "credentials.json"),
			filepath.Join(".gemini", "token.json"),
			filepath.Join(".gemini", "oauth_credentials.json"),
		} {
			p := filepath.Join(home, rel)
			if fi, err := insp.Stat(p); err == nil && fi.Size() > 2 {
				if d.hasValidJSONContent(insp, p) {
					state.Status = GateStatusPass
					state.Selectable = true
					return state
				}
			}
		}
	}

	state.Status = GateStatusNotConfigured
	state.Selectable = false
	state.Reason = "Sessão da CLI 'gemini' não autenticada"
	state.Action = "fazer login com 'gemini auth login'"
	return state
}

// Detection logic for GitHub Copilot
func (d *CliSubscriptionDetector) detectCopilot(insp *SystemInspector, models []CliSubscriptionModel) CliSubscriptionState {
	state := CliSubscriptionState{
		Provider:    CliProviderCopilot,
		DisplayName: "GitHub Copilot",
		Binary:      "copilot",
		CostTag:     CostTagSubscription,
		Models:      models,
	}

	binPath := d.findBinary(insp, "copilot", []string{
		filepath.Join(".local", "bin", "copilot"),
	})
	if binPath == "" {
		// Also check if `gh` is available (gh copilot extension)
		if ghPath, err := insp.LookPath("gh"); err == nil && ghPath != "" {
			binPath = ghPath
		}
	}

	if binPath == "" {
		state.Status = GateStatusNotPresent
		state.Selectable = false
		state.Reason = "GitHub Copilot CLI não encontrada no sistema"
		state.Action = "gh extension install github/gh-copilot"
		return state
	}
	state.BinaryPath = binPath

	// Check authentication
	if insp.Getenv("GITHUB_TOKEN") != "" || insp.Getenv("GH_TOKEN") != "" || insp.Getenv("COPILOT_TOKEN") != "" {
		state.Status = GateStatusPass
		state.Selectable = true
		return state
	}

	home, _ := insp.UserHome()
	if home != "" {
		for _, rel := range []string{
			filepath.Join(".config", "github-copilot", "hosts.json"),
			filepath.Join(".config", "gh", "hosts.yml"),
			filepath.Join(".copilot", "credentials.json"),
		} {
			p := filepath.Join(home, rel)
			if fi, err := insp.Stat(p); err == nil && fi.Size() > 2 {
				data, rerr := insp.ReadFile(p)
				if rerr == nil && len(data) > 0 {
					// Check for token indicator in json or yaml
					content := string(data)
					if strings.Contains(content, "oauth_token") || strings.Contains(content, "token") || strings.Contains(content, "user") {
						state.Status = GateStatusPass
						state.Selectable = true
						return state
					}
				}
			}
		}
	}

	state.Status = GateStatusNotConfigured
	state.Selectable = false
	state.Reason = "Sessão do GitHub Copilot não autenticada"
	state.Action = "fazer login com 'gh auth login' ou 'copilot auth'"
	return state
}

func (d *CliSubscriptionDetector) findBinary(insp *SystemInspector, name string, relativeFallbacks []string) string {
	if p, err := insp.LookPath(name); err == nil && p != "" {
		return p
	}

	home, err := insp.UserHome()
	if err != nil || home == "" {
		return ""
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	for _, rel := range relativeFallbacks {
		candidate := filepath.Join(home, rel+ext)
		if fi, err := insp.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
		// Try without extension too
		candidateNoExt := filepath.Join(home, rel)
		if fi, err := insp.Stat(candidateNoExt); err == nil && !fi.IsDir() {
			return candidateNoExt
		}
	}

	return ""
}

func (d *CliSubscriptionDetector) hasValidJSONContent(insp *SystemInspector, path string) bool {
	data, err := insp.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	return len(raw) > 0
}
