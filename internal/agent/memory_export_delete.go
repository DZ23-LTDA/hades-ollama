package agent

import (
	"errors"
	"strings"
	"time"
)

// Evidência obrigatória do estágio 15: EXPORTAÇÃO e EXCLUSÃO da memória, com o
// mesmo isolamento por ator que as leituras já respeitam.
//
// Duas decisões de privacidade deliberadas:
//
//  1. A exportação devolve SOMENTE o que o ator pode ler. Exportar "tudo do
//     projeto" vazaria anotação privada de colega no arquivo exportado — o risco
//     exato que o escopo por usuário existe para evitar.
//  2. A exclusão de memória que o ator NÃO pode ler falha com o MESMO erro de
//     memória inexistente (`ErrMemoryNotFound`). Distinguir os casos deixaria
//     qualquer um descobrir a existência de dado privado alheio apenas tentando
//     apagar.
//
// A persistência NÃO é reimplementada aqui: a exclusão reusa `RemoveMemory`, que
// já faz a escrita atômica com rollback. Duplicar esse núcleo criaria dois
// caminhos que podem divergir.

// MemoryExport é o snapshot exportável da memória visível a UM ator.
type MemoryExport struct {
	ProjectID   string    `json:"project_id"`
	ActorID     string    `json:"actor_id,omitempty"`
	GeneratedAt time.Time `json:"generated_at"`
	Count       int       `json:"count"`
	Memories    []Memory  `json:"memories"`
}

// ExportMemoriesForActor devolve a memória do projeto que o ator autenticado pode
// ler: conhecimento do projeto mais a memória privada dele. `includeEmbeddings`
// decide se os vetores saem no resultado — exportar vetor por padrão incharia o
// arquivo sem servir ao usuário.
func (s *ContextStore) ExportMemoriesForActor(projectID, actorID string, includeEmbeddings bool) (MemoryExport, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return MemoryExport{}, errors.New("memory export requires a project")
	}
	s.mu.RLock()
	stored := append([]Memory(nil), s.memories[projectID]...)
	s.mu.RUnlock()
	visible := filterMemoriesForActor(stored, actorID)
	if !includeEmbeddings {
		// Cópia sem vetores: o export não deve devolver o embedding por acidente.
		stripped := make([]Memory, 0, len(visible))
		for _, memory := range visible {
			memory.Embedding = nil
			stripped = append(stripped, memory)
		}
		visible = stripped
	}
	return MemoryExport{
		ProjectID:   projectID,
		ActorID:     strings.TrimSpace(actorID),
		GeneratedAt: time.Now().UTC(),
		Count:       len(visible),
		Memories:    visible,
	}, nil
}

// DeleteMemoryForActor remove uma memória do projeto quando o ator PODE lê-la.
//
// A checagem de visibilidade acontece antes da remoção e a recusa usa o mesmo
// erro de memória inexistente, para que a resposta não sirva de oráculo sobre a
// existência de dado privado alheio.
func (s *ContextStore) DeleteMemoryForActor(projectID, memoryID, actorID string) error {
	projectID = strings.TrimSpace(projectID)
	memoryID = strings.TrimSpace(memoryID)
	if projectID == "" || memoryID == "" {
		return ErrMemoryNotFound
	}
	s.mu.RLock()
	allowed := false
	for _, memory := range s.memories[projectID] {
		if strings.TrimSpace(memory.ID) != memoryID {
			continue
		}
		allowed = memoryVisibleTo(memory, actorID)
		break
	}
	s.mu.RUnlock()
	if !allowed {
		return ErrMemoryNotFound
	}
	return s.RemoveMemory(projectID, memoryID)
}
