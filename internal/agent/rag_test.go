package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBuildWebGroundedContextCitesOnlyFetchedSources(t *testing.T) {
	fetched := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	sources := []ResearchSource{
		{URL: "https://ex.com/a", Title: "Página A", Status: 200, Text: "conteúdo A", FetchedAt: fetched},
		{URL: "https://ex.com/404", Title: "Faltante", Status: 404, Text: "", Error: "not found"},
		{URL: "https://ex.com/b", Title: "Página B", Status: 200, Text: "conteúdo B", FetchedAt: fetched},
	}
	ctx, citations := BuildWebGroundedContext("pergunta", sources)
	if len(citations) != 2 {
		t.Fatalf("citations = %d, want 2 (failed fetch excluded)", len(citations))
	}
	for _, want := range []string{"https://ex.com/a", "Página A", "2026-10-02", "conteúdo B"} {
		if !strings.Contains(ctx, want) {
			t.Fatalf("web context missing %q:\n%s", want, ctx)
		}
	}
	if strings.Contains(ctx, "ex.com/404") || strings.Contains(ctx, "Faltante") {
		t.Fatalf("failed fetch must not be cited:\n%s", ctx)
	}
}

func TestBuildWebGroundedContextNoUsableSourcesIsHonest(t *testing.T) {
	sources := []ResearchSource{{URL: "https://ex.com/x", Status: 500, Error: "boom"}}
	_, citations := BuildWebGroundedContext("q", sources)
	if citations != nil {
		t.Fatalf("citations = %+v, want nil when nothing was fetched", citations)
	}
}

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

func TestGroundedAnswerContextEndToEnd(t *testing.T) {
	store, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	store.SetEmbedder(testEmbedder{
		"a politica de ferias da empresa e de 30 dias": {1, 0},
		"ferias":  {1, 0},
		"salario": {0, 1},
	})
	mem := Memory{ProjectID: "p", Kind: "doc", Content: "a politica de ferias da empresa e de 30 dias", Source: "rh.pdf#3", Confidence: 1}
	if _, err := store.AddMemoryContext(context.Background(), mem); err != nil {
		t.Fatal(err)
	}

	text, citations, err := store.GroundedAnswerContext(context.Background(), "p", "ferias", 5, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(citations) != 1 || citations[0].Source != "rh.pdf#3" {
		t.Fatalf("citations = %+v, want one citing rh.pdf#3", citations)
	}
	if !strings.Contains(text, "rh.pdf#3") || !strings.Contains(text, "30 dias") {
		t.Fatalf("grounded context missing source/content:\n%s", text)
	}

	// A question the documents cannot answer yields no citations and an honest
	// instruction instead of a fabricated answer.
	_, none, err := store.GroundedAnswerContext(context.Background(), "p", "salario", 5, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if none != nil {
		t.Fatalf("unanswerable query returned citations: %+v", none)
	}
}
