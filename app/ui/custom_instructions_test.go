package ui

import (
	"strings"
	"testing"
)

// TestComposeSystemPrompt verifies that the user's custom instructions are
// combined with (not replaced by) the default chat system prompt, that an empty
// part is handled, and that oversized instructions are bounded.
func TestComposeSystemPrompt(t *testing.T) {
	base := "Responda em português."

	if got := composeSystemPrompt(base, ""); got != base {
		t.Errorf("empty custom should return base, got %q", got)
	}
	if got := composeSystemPrompt("", "Seja breve."); got != "Seja breve." {
		t.Errorf("empty base should return custom, got %q", got)
	}

	combined := composeSystemPrompt(base, "Seja breve.")
	if !strings.Contains(combined, base) || !strings.Contains(combined, "Seja breve.") {
		t.Errorf("combined prompt must contain both parts, got %q", combined)
	}

	huge := strings.Repeat("a", 10000)
	bounded := composeSystemPrompt(base, huge)
	if len(bounded) >= len(base)+10000 {
		t.Errorf("custom instructions must be bounded, got length %d", len(bounded))
	}
}
