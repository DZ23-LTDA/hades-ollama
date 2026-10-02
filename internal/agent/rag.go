package agent

import (
	"fmt"
	"strings"
)

// Citation identifies a source used to ground an answer. Index is the [n]
// marker used in the grounded context; Source is the provenance (file#chunk or
// URL) carried from the indexed document.
type Citation struct {
	Index   int     `json:"index"`
	Source  string  `json:"source"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet,omitempty"`
}

const groundedContextSnippetLimit = 1200

// BuildGroundedContext turns retrieved sources into a prompt context block with
// numbered citations plus an instruction to answer strictly from those sources.
// It returns the context text and the ordered citations so the caller can both
// prompt the model and render the references. When there are no sources it
// returns a context that tells the model to say it does not know instead of
// inventing an answer — the anti-hallucination half of document RAG (G1).
func BuildGroundedContext(query string, sources []ScoredMemory) (string, []Citation) {
	query = strings.TrimSpace(query)
	if len(sources) == 0 {
		return "Não há fontes relevantes para esta pergunta no material fornecido. " +
			"Responda honestamente que não encontrou a informação nos documentos; não invente.", nil
	}

	citations := make([]Citation, 0, len(sources))
	var b strings.Builder
	b.WriteString("Responda à pergunta usando SOMENTE os trechos abaixo. ")
	b.WriteString("Cite cada afirmação com o marcador [n] da fonte correspondente. ")
	b.WriteString("Se a resposta não estiver nos trechos, diga que não encontrou nos documentos.\n\n")
	for i, src := range sources {
		index := i + 1
		snippet := strings.TrimSpace(src.Memory.Content)
		if len(snippet) > groundedContextSnippetLimit {
			snippet = snippet[:groundedContextSnippetLimit] + "…"
		}
		source := strings.TrimSpace(src.Memory.Source)
		if source == "" {
			source = "fonte desconhecida"
		}
		fmt.Fprintf(&b, "[%d] (%s)\n%s\n\n", index, source, snippet)
		citations = append(citations, Citation{
			Index:   index,
			Source:  source,
			Score:   src.Score,
			Snippet: snippet,
		})
	}
	if query != "" {
		fmt.Fprintf(&b, "Pergunta: %s\n", query)
	}
	return strings.TrimRight(b.String(), "\n"), citations
}

// BuildWebGroundedContext formats fetched web sources into a cited prompt block,
// reusing BuildGroundedContext. Only sources that were actually fetched (2xx, no
// error, non-empty text) are cited, so the model never cites a page it could not
// read; each citation carries the page title, URL and fetch date. With no usable
// source the model is told it did not find the information (G2).
func BuildWebGroundedContext(query string, sources []ResearchSource) (string, []Citation) {
	usable := make([]ScoredMemory, 0, len(sources))
	for _, s := range sources {
		text := strings.TrimSpace(s.Text)
		if text == "" || strings.TrimSpace(s.Error) != "" {
			continue
		}
		if s.Status != 0 && (s.Status < 200 || s.Status >= 300) {
			continue
		}
		label := strings.TrimSpace(s.Title)
		url := strings.TrimSpace(s.URL)
		if label == "" {
			label = url
		}
		source := label
		if url != "" && url != label {
			source = label + " — " + url
		}
		if !s.FetchedAt.IsZero() {
			source += " (" + s.FetchedAt.UTC().Format("2006-01-02") + ")"
		}
		usable = append(usable, ScoredMemory{Memory: Memory{Content: text, Source: source}, Score: 1})
	}
	return BuildGroundedContext(query, usable)
}
