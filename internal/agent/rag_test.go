package agent

import (
	"strings"
	"testing"
)

func TestBuildGroundedContextCitesSources(t *testing.T) {
	sources := []ScoredMemory{
		{Memory: Memory{Content: "O gato é um mamífero.", Source: "gatos.txt#1"}, Score: 0.98},
		{Memory: Memory{Content: "Gatos dormem muito.", Source: "gatos.txt#2"}, Score: 0.91},
	}
	ctx, citations := BuildGroundedContext("o que é um gato?", sources)

	if len(citations) != 2 {
		t.Fatalf("citations = %d, want 2", len(citations))
	}
	if citations[0].Index != 1 || citations[0].Source != "gatos.txt#1" {
		t.Fatalf("first citation = %+v", citations[0])
	}
	for _, want := range []string{"[1]", "[2]", "gatos.txt#1", "O gato é um mamífero.", "Pergunta: o que é um gato?"} {
		if !strings.Contains(ctx, want) {
			t.Fatalf("grounded context missing %q:\n%s", want, ctx)
		}
	}
}

func TestBuildGroundedContextWithNoSourcesInstructsHonesty(t *testing.T) {
	ctx, citations := BuildGroundedContext("qualquer coisa", nil)
	if citations != nil {
		t.Fatalf("citations = %+v, want nil", citations)
	}
	// Must instruct the model to admit it does not know rather than invent.
	if !strings.Contains(strings.ToLower(ctx), "não") || !strings.Contains(strings.ToLower(ctx), "invente") {
		t.Fatalf("no-source context should tell the model not to invent:\n%s", ctx)
	}
}

func TestBuildGroundedContextTruncatesLongSnippet(t *testing.T) {
	long := strings.Repeat("a", groundedContextSnippetLimit+500)
	ctx, _ := BuildGroundedContext("q", []ScoredMemory{{Memory: Memory{Content: long, Source: "big.txt"}, Score: 1}})
	// The snippet must be cut to the limit: a run longer than the limit means
	// it was not truncated (other prose in the block also contains 'a', so we
	// check the contiguous run rather than a raw count).
	if strings.Contains(ctx, strings.Repeat("a", groundedContextSnippetLimit+1)) {
		t.Fatalf("snippet was not truncated to the limit")
	}
	if !strings.Contains(ctx, "…") {
		t.Fatalf("truncated snippet should end with an ellipsis")
	}
}
