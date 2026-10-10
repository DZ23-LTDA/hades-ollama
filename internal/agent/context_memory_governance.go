package agent

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Governança de memória (estágio 15): exclusão, exportação com proveniência e
// retenção. O `ContextStore` já indexava e recuperava memória por projeto; o
// que faltava era o ciclo de vida — direito de apagar, direito de levar o dado
// embora e limite de retenção.
//
// Regras aplicadas aqui, iguais às do resto do store:
//   - toda mutação passa por `writeJSONAtomic` e faz rollback do estado em
//     memória quando a persistência falha (nunca "meio gravado");
//   - o escopo é sempre o PROJETO informado pelo servidor, nunca um campo vindo
//     do corpo da requisição;
//   - memória sem data de criação é PRESERVADA na retenção: apagar o que não se
//     sabe quando foi criado seria destrutivo por suposição.

// ErrMemoryNotFound indica que a memória não existe NO PROJETO informado.
var ErrMemoryNotFound = errors.New("memory not found")

// RemoveMemory apaga uma memória do projeto e persiste a lista restante.
func (s *ContextStore) RemoveMemory(projectID, memoryID string) error {
	projectID = strings.TrimSpace(projectID)
	memoryID = strings.TrimSpace(memoryID)
	if projectID == "" || memoryID == "" {
		return ErrMemoryNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	memories := s.memories[projectID]
	index := -1
	for position, memory := range memories {
		if memory.ID == memoryID {
			index = position
			break
		}
	}
	if index < 0 {
		return ErrMemoryNotFound
	}
	previous := append([]Memory(nil), memories...)
	remaining := make([]Memory, 0, len(memories)-1)
	remaining = append(remaining, memories[:index]...)
	remaining = append(remaining, memories[index+1:]...)
	if len(remaining) == 0 {
		delete(s.memories, projectID)
	} else {
		s.memories[projectID] = remaining
	}
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "memories", projectID+".json"), remaining); err != nil {
			if previous == nil {
				delete(s.memories, projectID)
			} else {
				s.memories[projectID] = previous
			}
			return err
		}
	}
	return nil
}

// ExportMemories devolve uma cópia cronológica das memórias do projeto. Os
// vetores de embedding são omitidos por padrão porque são dados derivados e
// volumosos; quem precisa deles pede explicitamente.
func (s *ContextStore) ExportMemories(projectID string, includeEmbeddings bool) []Memory {
	projectID = strings.TrimSpace(projectID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	source := s.memories[projectID]
	exported := make([]Memory, 0, len(source))
	for _, memory := range source {
		copied := memory
		if includeEmbeddings {
			copied.Embedding = append([]float32(nil), memory.Embedding...)
		} else {
			copied.Embedding = nil
		}
		exported = append(exported, copied)
	}
	sort.SliceStable(exported, func(i, j int) bool {
		return exported[i].CreatedAt.Before(exported[j].CreatedAt)
	})
	return exported
}

// PruneMemories aplica retenção: apaga memórias criadas ANTES do corte e
// devolve quantas saíram. Memória com data zero é preservada.
func (s *ContextStore) PruneMemories(projectID string, olderThan time.Time) (int, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return 0, errors.New("project id is required")
	}
	if olderThan.IsZero() {
		return 0, errors.New("retention cutoff is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	memories := s.memories[projectID]
	kept := make([]Memory, 0, len(memories))
	removed := 0
	for _, memory := range memories {
		if !memory.CreatedAt.IsZero() && memory.CreatedAt.Before(olderThan) {
			removed++
			continue
		}
		kept = append(kept, memory)
	}
	if removed == 0 {
		return 0, nil
	}
	previous := append([]Memory(nil), memories...)
	if len(kept) == 0 {
		delete(s.memories, projectID)
	} else {
		s.memories[projectID] = kept
	}
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "memories", projectID+".json"), kept); err != nil {
			if previous == nil {
				delete(s.memories, projectID)
			} else {
				s.memories[projectID] = previous
			}
			return 0, err
		}
	}
	return removed, nil
}
