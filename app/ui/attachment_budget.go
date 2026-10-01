//go:build windows || darwin

package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/app/attachments"
)

const (
	// defaultContextTokens is assumed when the model does not report one.
	defaultContextTokens = 32768
	// charsPerToken is a conservative average for code and prose.
	charsPerToken = 4
	// attachmentShare of the context is available to file contents; the rest
	// is left for the conversation, instructions and the reply.
	attachmentShare = 0.6
	// maxDigestBatches bounds how many model calls one attachment set can
	// trigger when it does not fit the context window.
	maxDigestBatches = 24
)

// attachmentCharBudget returns how many characters of attached file text fit
// the model's context window.
func attachmentCharBudget(details *api.ShowResponse) int {
	tokens := 0
	if details != nil {
		for key, value := range details.ModelInfo {
			if !strings.HasSuffix(key, ".context_length") {
				continue
			}
			switch v := value.(type) {
			case float64:
				tokens = int(v)
			case int:
				tokens = v
			case int64:
				tokens = int(v)
			}
		}
	}
	if tokens <= 0 {
		tokens = defaultContextTokens
	}
	return int(float64(tokens*charsPerToken) * attachmentShare)
}

type attachmentText struct {
	name    string
	content string
}

func (a attachmentText) size() int { return len(a.content) + len(a.name)*2 + 40 }

// splitAttachments keeps files in their original order while they fit the
// budget. Noisy files (lockfiles, dependencies, build output) are dropped and
// the rest that does not fit is returned as overflow.
func splitAttachments(files []attachmentText, budget int) (kept []attachmentText, noisy []string, overflow []attachmentText) {
	used := 0
	for _, f := range files {
		if attachments.Noisy(f.name) {
			noisy = append(noisy, f.name)
			continue
		}
		if used+f.size() > budget {
			overflow = append(overflow, f)
			continue
		}
		used += f.size()
		kept = append(kept, f)
	}
	return kept, noisy, overflow
}

// batchAttachments groups files into batches that each fit the budget; a
// single file larger than the budget is truncated to fit its own batch.
func batchAttachments(files []attachmentText, budget int) [][]attachmentText {
	var batches [][]attachmentText
	var current []attachmentText
	used := 0
	for _, f := range files {
		if f.size() > budget {
			limit := budget - len(f.name)*2 - 80
			if limit < 0 {
				limit = 0
			}
			if limit < len(f.content) {
				f.content = f.content[:limit] + "\n[... arquivo truncado ...]"
			}
		}
		if used+f.size() > budget && len(current) > 0 {
			batches = append(batches, current)
			current, used = nil, 0
		}
		current = append(current, f)
		used += f.size()
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

func writeFiles(sb *strings.Builder, files []attachmentText) {
	for _, f := range files {
		fmt.Fprintf(sb, "\n--- File: %s ---\n%s\n--- End of %s ---", f.name, f.content, f.name)
	}
}

func fileNames(files []attachmentText) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.name
	}
	return names
}

// attachmentNote tells the model what it is not seeing in full, so answers
// about a large project are explicit about their coverage.
func attachmentNote(noisy []string, omitted []string, digest string) string {
	if len(noisy) == 0 && len(omitted) == 0 && digest == "" {
		return ""
	}
	var note strings.Builder
	note.WriteString("\n--- Nota sobre os anexos ---\n")
	if len(noisy) > 0 {
		fmt.Fprintf(&note, "%d arquivo(s) de dependências, build ou minificados foram omitidos: %s\n", len(noisy), summarizeNames(noisy))
	}
	if digest != "" {
		note.WriteString("O projeto não cabe inteiro no contexto do modelo. Os arquivos restantes foram analisados em lotes; os achados de cada lote estão abaixo e devem ser considerados na resposta.\n")
		note.WriteString(digest)
		note.WriteString("\n")
	}
	if len(omitted) > 0 {
		fmt.Fprintf(&note, "%d arquivo(s) não couberam e não foram analisados: %s\n", len(omitted), summarizeNames(omitted))
	}
	note.WriteString("--- Fim da nota ---")
	return note.String()
}

func summarizeNames(names []string) string {
	const shown = 40
	if len(names) <= shown {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s e mais %d", strings.Join(names[:shown], ", "), len(names)-shown)
}

// attachmentDigest is the batched analysis of files that did not fit.
type attachmentDigest struct {
	text    string
	omitted []string
}

// digestAttachments analyzes overflow files in context-sized batches with the
// user's request and returns the combined findings. Batches beyond
// maxDigestBatches are reported as not analyzed rather than silently dropped.
func digestAttachments(ctx context.Context, chat func(context.Context, *api.ChatRequest, api.ChatResponseFunc) error, model, request string, overflow []attachmentText, budget int, progress func(string)) attachmentDigest {
	batches := batchAttachments(overflow, budget)
	var result attachmentDigest
	if len(batches) > maxDigestBatches {
		for _, b := range batches[maxDigestBatches:] {
			result.omitted = append(result.omitted, fileNames(b)...)
		}
		batches = batches[:maxDigestBatches]
	}
	var sb strings.Builder
	stream := false
	for i, batch := range batches {
		if ctx.Err() != nil {
			for _, b := range batches[i:] {
				result.omitted = append(result.omitted, fileNames(b)...)
			}
			break
		}
		if progress != nil {
			progress(fmt.Sprintf("Analisando parte %d de %d do projeto (%d arquivos)…\n", i+1, len(batches), len(batch)))
		}
		var prompt strings.Builder
		fmt.Fprintf(&prompt, "Pedido do usuário: %s\n\nVocê está analisando a parte %d de %d de um projeto maior. Liste de forma objetiva os achados relevantes para o pedido nestes arquivos (problemas, riscos, bugs, segurança, qualidade), citando o arquivo de cada achado. Não repita o código.\n", request, i+1, len(batches))
		writeFiles(&prompt, batch)

		var answer strings.Builder
		err := chat(ctx, &api.ChatRequest{
			Model:    model,
			Messages: []api.Message{{Role: "user", Content: prompt.String()}},
			Stream:   &stream,
		}, func(res api.ChatResponse) error {
			answer.WriteString(res.Message.Content)
			return nil
		})
		fmt.Fprintf(&sb, "\n### Parte %d de %d (%s)\n", i+1, len(batches), summarizeNames(fileNames(batch)))
		if err != nil {
			fmt.Fprintf(&sb, "Falha ao analisar esta parte: %v\n", err)
			continue
		}
		sb.WriteString(strings.TrimSpace(answer.String()))
		sb.WriteString("\n")
	}
	result.text = sb.String()
	return result
}
