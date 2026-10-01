//go:build windows || darwin

package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

func TestAttachmentCharBudgetUsesModelContext(t *testing.T) {
	details := &api.ShowResponse{ModelInfo: map[string]any{"gemma3.context_length": float64(262144)}}
	if got := attachmentCharBudget(details); got != 629145 {
		t.Fatalf("budget = %d, want 629145", got)
	}
	if got := attachmentCharBudget(nil); got != 78643 {
		t.Fatalf("default budget = %d, want 78643", got)
	}
}

func TestSplitAttachmentsFitsBudgetAndDropsNoise(t *testing.T) {
	files := []attachmentText{
		{"p.zip/src/a.ts", strings.Repeat("a", 100)},
		{"p.zip/package-lock.json", strings.Repeat("x", 10000)},
		{"p.zip/src/big.ts", strings.Repeat("b", 1000)},
		{"p.zip/src/c.ts", strings.Repeat("c", 100)},
	}
	kept, noisy, overflow := splitAttachments(files, 400)
	if got := strings.Join(fileNames(kept), ","); got != "p.zip/src/a.ts,p.zip/src/c.ts" {
		t.Fatalf("kept = %s", got)
	}
	if len(noisy) != 1 || noisy[0] != "p.zip/package-lock.json" {
		t.Fatalf("noisy = %v", noisy)
	}
	if len(overflow) != 1 || overflow[0].name != "p.zip/src/big.ts" {
		t.Fatalf("overflow = %v", fileNames(overflow))
	}
	if note := attachmentNote(nil, nil, ""); note != "" {
		t.Fatalf("empty note = %q", note)
	}
}

func TestBatchAttachmentsRespectsBudget(t *testing.T) {
	files := []attachmentText{
		{"a", strings.Repeat("a", 150)},
		{"b", strings.Repeat("b", 150)},
		{"c", strings.Repeat("c", 150)},
		{"huge", strings.Repeat("h", 5000)},
	}
	batches := batchAttachments(files, 400)
	if len(batches) != 3 {
		t.Fatalf("batches = %d, want 3", len(batches))
	}
	for i, batch := range batches {
		total := 0
		for _, f := range batch {
			total += f.size()
		}
		if total > 400+40 {
			t.Fatalf("batch %d is %d chars, over budget", i, total)
		}
	}
	if !strings.Contains(batches[2][0].content, "truncado") {
		t.Fatal("oversized file should be truncated")
	}
}

func TestDigestAttachmentsAnalyzesEveryBatch(t *testing.T) {
	var prompts []string
	fake := func(_ context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
		prompts = append(prompts, req.Messages[0].Content)
		if strings.Contains(req.Messages[0].Content, "parte 2 de") {
			return errors.New("boom")
		}
		return fn(api.ChatResponse{Message: api.Message{Content: "achado"}})
	}
	var statuses []string
	files := []attachmentText{{"a", strings.Repeat("a", 300)}, {"b", strings.Repeat("b", 300)}, {"c", strings.Repeat("c", 300)}}
	digest := digestAttachments(context.Background(), fake, "m", "auditoria", files, 400, func(s string) { statuses = append(statuses, s) })

	if len(prompts) != 3 || len(statuses) != 3 {
		t.Fatalf("calls = %d, statuses = %d", len(prompts), len(statuses))
	}
	if !strings.Contains(prompts[0], "Pedido do usuário: auditoria") || !strings.Contains(prompts[0], "--- File: a ---") {
		t.Fatalf("prompt = %q", prompts[0])
	}
	if strings.Count(digest.text, "achado") != 2 || !strings.Contains(digest.text, "Falha ao analisar esta parte: boom") {
		t.Fatalf("digest = %q", digest.text)
	}
	if note := attachmentNote(nil, digest.omitted, digest.text); !strings.Contains(note, "analisados em lotes") {
		t.Fatalf("note = %q", note)
	}
}

func TestDigestAttachmentsCapsBatches(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, _ *api.ChatRequest, fn api.ChatResponseFunc) error {
		calls++
		return fn(api.ChatResponse{Message: api.Message{Content: "ok"}})
	}
	var files []attachmentText
	for i := range maxDigestBatches + 5 {
		files = append(files, attachmentText{name: fmt.Sprintf("f%d", i), content: strings.Repeat("x", 300)})
	}
	digest := digestAttachments(context.Background(), fake, "m", "q", files, 400, nil)
	if calls != maxDigestBatches || len(digest.omitted) != 5 {
		t.Fatalf("calls = %d omitted = %d", calls, len(digest.omitted))
	}
}
