package launch

import (
	"slices"
	"testing"
)

// TestCliSubscriptionLaunchCompatibility verifies that ollama launch integrations
// are fully preserved and registered, ensuring the CLI subscription feature in the agent
// does not break or alter the launcher tools flow.
func TestCliSubscriptionLaunchCompatibility(t *testing.T) {
	requiredLaunchers := []string{"claude", "codex", "copilot", "gemini"}

	infos := ListIntegrationInfos()
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}

	for _, req := range requiredLaunchers {
		if !slices.Contains(names, req) {
			t.Fatalf("expected launcher %q to be registered in cmd/launch, got: %v", req, names)
		}
	}

	// Verify Gemini runner
	gemini := &Gemini{}
	if gemini.String() != "Gemini CLI" {
		t.Fatalf("expected 'Gemini CLI', got %q", gemini.String())
	}
	args := gemini.args("llama3:8b", []string{"--debug"})
	if !slices.Contains(args, "--model") || !slices.Contains(args, "llama3:8b") || !slices.Contains(args, "--debug") {
		t.Fatalf("unexpected gemini args: %v", args)
	}
}
