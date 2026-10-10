package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Isolamento de memória por USUÁRIO dentro da mesma organização (estágio 15).
//
// O store já isolava por organização e projeto. Faltava o escopo de usuário: duas
// pessoas da MESMA organização, no MESMO projeto, não podem ler a memória privada
// uma da outra. Sem isso, um workspace compartilhado vaza anotação pessoal entre
// colegas — risco ALTO de privacidade.
//
// Regras aplicadas em TODOS os caminhos de leitura (busca e RAG):
//   - memória marcada como `organization` é visível a qualquer ator autorizado;
//   - memória marcada como `private` só é visível ao ator que a criou;
//   - memória antiga (sem visibilidade/ator) continua valendo como `organization`,
//     para não quebrar dados existentes;
//   - `private` sem ator identificado é RECUSADA na gravação: não se cria dado
//     privado que ninguém pode ler.

const (
	// MemoryVisibilityOrganization é o padrão: conhecimento do projeto.
	MemoryVisibilityOrganization = "organization"
	// MemoryVisibilityPrivate restringe a leitura ao ator que criou.
	MemoryVisibilityPrivate = "private"
)

var (
	ErrMemoryVisibilityInvalid = errors.New("memory visibility is invalid")
	ErrMemoryActorRequired     = errors.New("private memory requires an identified actor")
)

func normalizeMemoryVisibility(memory Memory, actorID string) (Memory, error) {
	actorID = strings.TrimSpace(actorID)
	visibility := strings.TrimSpace(strings.ToLower(memory.Visibility))
	switch visibility {
	case "":
		visibility = MemoryVisibilityOrganization
	case MemoryVisibilityOrganization, MemoryVisibilityPrivate:
	default:
		return Memory{}, fmt.Errorf("%w: %q", ErrMemoryVisibilityInvalid, memory.Visibility)
	}
	if visibility == MemoryVisibilityPrivate && actorID == "" {
		return Memory{}, ErrMemoryActorRequired
	}
	memory.Visibility = visibility
	if visibility == MemoryVisibilityPrivate {
		memory.ActorID = actorID
	}
	return memory, nil
}

// memoryVisibleTo decide se um ator autorizado pode ler a memória.
func memoryVisibleTo(memory Memory, actorID string) bool {
	if strings.EqualFold(strings.TrimSpace(memory.Visibility), MemoryVisibilityPrivate) {
		return strings.TrimSpace(memory.ActorID) != "" && strings.TrimSpace(memory.ActorID) == strings.TrimSpace(actorID)
	}
	return true
}

func filterMemoriesForActor(memories []Memory, actorID string) []Memory {
	visible := make([]Memory, 0, len(memories))
	for _, memory := range memories {
		if memoryVisibleTo(memory, actorID) {
			visible = append(visible, memory)
		}
	}
	return visible
}

// AddMemoryForActor grava a memória já associada ao ator autenticado. Memória
// privada de outro ator não pode ser forjada pelo corpo da requisição.
func (s *ContextStore) AddMemoryForActor(ctx context.Context, memory Memory, actorID string) (Memory, error) {
	normalized, err := normalizeMemoryVisibility(memory, actorID)
	if err != nil {
		return Memory{}, err
	}
	return s.AddMemoryContext(ctx, normalized)
}

// SearchMemoriesForActor busca por relevância e devolve apenas o que o ator
// autenticado pode ler: a memória do projeto mais a memória privada dele.
func (s *ContextStore) SearchMemoriesForActor(ctx context.Context, projectID, actorID, query string, limit int) ([]Memory, error) {
	memories, err := s.SearchMemoriesContext(ctx, projectID, query, limit)
	if err != nil {
		return nil, err
	}
	return filterMemoriesForActor(memories, actorID), nil
}

// RetrieveRelevantForActor é o mesmo que RetrieveRelevant, restrito ao que o ator
// pode ler. A filtragem acontece ANTES da formatação de citações, para que uma
// memória privada alheia não apareça nem como referência.
func (s *ContextStore) RetrieveRelevantForActor(ctx context.Context, projectID, actorID, query string, limit int, minScore float64) ([]ScoredMemory, error) {
	hits, err := s.RetrieveRelevant(ctx, projectID, query, limit, minScore)
	if err != nil {
		return nil, err
	}
	visible := make([]ScoredMemory, 0, len(hits))
	for _, hit := range hits {
		if memoryVisibleTo(hit.Memory, actorID) {
			visible = append(visible, hit)
		}
	}
	return visible, nil
}

// GroundedAnswerForActor é o RAG de documentos com isolamento por usuário: uma
// memória privada de outro ator não entra no contexto fundamentado nem nas
// citações.
func (s *ContextStore) GroundedAnswerForActor(ctx context.Context, projectID, actorID, query string, limit int, minScore float64) (string, []Citation, error) {
	hits, err := s.RetrieveRelevantForActor(ctx, projectID, actorID, query, limit, minScore)
	if err != nil {
		return "", nil, err
	}
	text, citations := BuildGroundedContext(query, hits)
	return text, citations, nil
}
