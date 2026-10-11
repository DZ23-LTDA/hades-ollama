package agent

import (
	"context"
	"strings"
	"time"
)

// Resumos de memória (atividade do estágio 15).
//
// DECISÃO DE HONESTIDADE: o resumo padrão é um **digest determinístico**, não um
// texto gerado por modelo. A missão proíbe inventar conclusões, e um resumo por LLM
// pode afirmar o que não está na memória. O digest **copia trechos com referência**
// — se ele diz algo, aquilo está na memória; se não encontra, ele **diz que não
// encontrou** em vez de preencher com suposição.
//
// Para quem quiser resumo por modelo, existe o CONTRATO (`MemorySummarizer`): o
// motor aceita um resumidor injetado e **declara qual usou** (`Name()`), para o
// produto poder ser honesto sobre a origem do texto (§11.3 — catalogar não é
// habilitar).
//
// O digest NÃO reimplementa formatação: reusa `BuildGroundedContext`, que já limita
// tamanho, numera citações e trata o trecho como dado não confiável. Um segundo
// caminho de formatação divergiria do primeiro — o defeito de "irmão esquecido".

// MemorySummarizer produz o texto de um resumo a partir de fontes JÁ filtradas por
// escopo. Uma implementação por modelo deve devolver erro em vez de inventar quando
// não houver base.
type MemorySummarizer interface {
	Name() string
	Summarize(ctx context.Context, query string, sources []ScoredMemory) (string, []Citation, error)
}

// DigestSummarizer é o padrão: determinístico e fiel ao material.
type DigestSummarizer struct{}

func (DigestSummarizer) Name() string { return "digest-deterministico" }

func (DigestSummarizer) Summarize(_ context.Context, query string, sources []ScoredMemory) (string, []Citation, error) {
	text, citations := BuildGroundedContext(query, sources)
	return text, citations, nil
}

// MemorySummary é o resultado do resumo, com a origem declarada.
type MemorySummary struct {
	ProjectID   string     `json:"project_id"`
	Query       string     `json:"query"`
	Summarizer  string     `json:"summarizer"`
	Text        string     `json:"text"`
	Count       int        `json:"count"`
	Citations   []Citation `json:"citations,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	// Truncated avisa que o teto de fontes cortou material: silêncio aqui viraria
	// "resumo do que existe" quando é "resumo do que coube".
	Truncated bool `json:"truncated"`
}

// SummarizeMemoriesInScope resume a memória que o escopo pode DIVULGAR. O escopo é
// aplicado pela mesma camada da busca (`RetrieveRelevantInScope`), então o resumo
// herda o isolamento em vez de ter uma segunda regra para divergir.
func (s *ContextStore) SummarizeMemoriesInScope(ctx context.Context, projectID string, scope MemoryScope, query string, limit int, minScore float64, summarizer MemorySummarizer) (MemorySummary, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return MemorySummary{}, ErrMemoryNotFound
	}
	if summarizer == nil {
		summarizer = DigestSummarizer{}
	}
	if limit <= 0 {
		limit = DefaultMemorySummarySources
	}
	if limit > MaxMemorySummarySources {
		limit = MaxMemorySummarySources
	}
	hits, err := s.RetrieveRelevantInScope(ctx, projectID, scope, query, limit, minScore)
	if err != nil {
		return MemorySummary{}, err
	}
	text, citations, err := summarizer.Summarize(ctx, query, hits)
	if err != nil {
		return MemorySummary{}, err
	}
	return MemorySummary{
		ProjectID:   projectID,
		Query:       strings.TrimSpace(query),
		Summarizer:  summarizer.Name(),
		Text:        text,
		Count:       len(citations),
		Citations:   citations,
		GeneratedAt: time.Now().UTC(),
		Truncated:   len(hits) >= limit,
	}, nil
}

const (
	// DefaultMemorySummarySources é o teto padrão de fontes no resumo.
	DefaultMemorySummarySources = 8
	// MaxMemorySummarySources limita o teto: resumo com dezenas de fontes deixa de
	// ser resumo e vira despejo de conteúdo no contexto.
	MaxMemorySummarySources = 32
)
