package proxy

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"
)

type fakeFileInfo struct {
	name string
	size int64
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Now() }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestCliSubscriptionDetection(t *testing.T) {
	ctx := context.Background()

	providers := []CliSubscriptionProvider{
		CliProviderClaudeCode,
		CliProviderCodex,
		CliProviderGemini,
		CliProviderCopilot,
	}

	for _, provider := range providers {
		t.Run(string(provider)+"-Ausente", func(t *testing.T) {
			insp := &SystemInspector{
				LookPath: func(file string) (string, error) {
					return "", errors.New("not found")
				},
				Stat: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
				ReadFile: func(name string) ([]byte, error) {
					return nil, os.ErrNotExist
				},
				Getenv: func(key string) string {
					return ""
				},
				UserHome: func() (string, error) {
					return "/home/user", nil
				},
			}

			detector := NewCliSubscriptionDetector(insp)
			state := detector.Detect(ctx, provider)

			if state.Status != GateStatusNotPresent {
				t.Fatalf("expected status %s, got %s", GateStatusNotPresent, state.Status)
			}
			if state.Selectable {
				t.Fatalf("expected selectable false for missing CLI, got true")
			}
			if !strings.Contains(strings.ToLower(state.Action), "install") {
				t.Fatalf("expected action to contain install instruction, got %q", state.Action)
			}
			if state.CostTag != CostTagSubscription {
				t.Fatalf("expected cost tag %s, got %s", CostTagSubscription, state.CostTag)
			}
		})

		t.Run(string(provider)+"-Deslogado", func(t *testing.T) {
			insp := &SystemInspector{
				LookPath: func(file string) (string, error) {
					return "/usr/local/bin/" + file, nil
				},
				Stat: func(name string) (os.FileInfo, error) {
					// Binary exists, but auth files do not
					if strings.Contains(name, "bin") {
						return fakeFileInfo{name: name, size: 1024}, nil
					}
					return nil, os.ErrNotExist
				},
				ReadFile: func(name string) ([]byte, error) {
					return nil, os.ErrNotExist
				},
				Getenv: func(key string) string {
					return ""
				},
				UserHome: func() (string, error) {
					return "/home/user", nil
				},
			}

			detector := NewCliSubscriptionDetector(insp)
			state := detector.Detect(ctx, provider)

			if state.Status != GateStatusNotConfigured {
				t.Fatalf("expected status %s, got %s", GateStatusNotConfigured, state.Status)
			}
			if state.Selectable {
				t.Fatalf("expected selectable false for unauthenticated CLI, got true")
			}
			if !strings.Contains(strings.ToLower(state.Action), "login") && !strings.Contains(strings.ToLower(state.Action), "auth") {
				t.Fatalf("expected action to contain login instruction, got %q", state.Action)
			}
		})

		t.Run(string(provider)+"-Logado", func(t *testing.T) {
			insp := &SystemInspector{
				LookPath: func(file string) (string, error) {
					return "/usr/local/bin/" + file, nil
				},
				Stat: func(name string) (os.FileInfo, error) {
					return fakeFileInfo{name: name, size: 1024}, nil
				},
				ReadFile: func(name string) ([]byte, error) {
					// Return valid json or auth yaml
					return []byte(`{"oauth_token": "valid_token", "authenticated": true, "session": "active"}`), nil
				},
				Getenv: func(key string) string {
					return ""
				},
				UserHome: func() (string, error) {
					return "/home/user", nil
				},
			}

			detector := NewCliSubscriptionDetector(insp)
			state := detector.Detect(ctx, provider)

			if state.Status != GateStatusPass {
				t.Fatalf("expected status %s, got %s", GateStatusPass, state.Status)
			}
			if !state.Selectable {
				t.Fatalf("expected selectable true for authenticated CLI, got false")
			}
			if state.CostTag != CostTagSubscription {
				t.Fatalf("expected cost tag %s, got %s", CostTagSubscription, state.CostTag)
			}
			if len(state.Models) == 0 {
				t.Fatalf("expected models for provider %s, got 0", provider)
			}
		})
	}

	t.Run("DeslogadoOuAusenteNaoEntraNaListaUsavel", func(t *testing.T) {
		insp := &SystemInspector{
			LookPath: func(file string) (string, error) {
				// Only Claude is installed and logged in
				if file == "claude" {
					return "/usr/local/bin/claude", nil
				}
				if file == "codex" {
					return "/usr/local/bin/codex", nil // installed but not logged in
				}
				return "", errors.New("not found")
			},
			Stat: func(name string) (os.FileInfo, error) {
				if strings.Contains(name, "claude") {
					return fakeFileInfo{name: name, size: 500}, nil
				}
				if strings.Contains(name, "codex") && strings.Contains(name, "bin") {
					return fakeFileInfo{name: name, size: 500}, nil
				}
				return nil, os.ErrNotExist
			},
			ReadFile: func(name string) ([]byte, error) {
				if strings.Contains(name, "claude") {
					return []byte(`{"session_key": "active_test_session"}`), nil
				}
				return nil, os.ErrNotExist
			},
			Getenv: func(key string) string {
				return ""
			},
			UserHome: func() (string, error) {
				return "/home/user", nil
			},
		}

		detector := NewCliSubscriptionDetector(insp)
		usable := detector.GetUsableModels(ctx)

		if len(usable) == 0 {
			t.Fatalf("expected usable models from Claude Code (PASS)")
		}

		for _, m := range usable {
			if m.Provider != CliProviderClaudeCode {
				t.Fatalf("unexpected usable model %s from provider %s (should only be Claude Code)", m.ID, m.Provider)
			}
		}

		// Check catalog entries: all models exist, but unauthenticated/missing have selectable=false
		catalog := detector.GetCatalogEntries(ctx)
		var codexFound, geminiFound bool
		for _, e := range catalog {
			if e.Provider == CliProviderCodex {
				codexFound = true
				if e.Selectable {
					t.Fatalf("expected codex model %s to NOT be selectable when unauthenticated", e.ID)
				}
				if e.Status != GateStatusNotConfigured {
					t.Fatalf("expected codex status NOT_CONFIGURED, got %s", e.Status)
				}
			}
			if e.Provider == CliProviderGemini {
				geminiFound = true
				if e.Selectable {
					t.Fatalf("expected gemini model %s to NOT be selectable when missing", e.ID)
				}
				if e.Status != GateStatusNotPresent {
					t.Fatalf("expected gemini status NOT_PRESENT, got %s", e.Status)
				}
			}
		}
		if !codexFound || !geminiFound {
			t.Fatalf("expected catalog entries for codex and gemini")
		}
	})
}
