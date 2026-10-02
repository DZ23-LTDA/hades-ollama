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
