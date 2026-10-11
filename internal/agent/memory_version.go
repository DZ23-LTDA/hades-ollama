package agent

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Versionamento de memória (atividade do estágio 15).
//
// Antes disto a memória era só append + delete: corrigir uma anotação exigia
// apagar e recriar, e o que foi corrigido se perdia. Sem histórico não há como
// auditar quem reescreveu o conhecimento do projeto, nem desfazer uma alteração
// errada — que é exatamente o risco "memória sem proveniência".
//
// Decisões deliberadas:
//   - o histórico guarda o conteúdo ANTERIOR, com quem alterou, data e motivo;
//   - o histórico é LIMITADO: sem teto, uma memória editada muitas vezes viraria
//     um arquivo sem fim no store e no contexto exportado;
//   - a alteração passa pela MESMA checagem de visibilidade da exclusão: memória
//     privada de outro ator é recusada com o erro de inexistente, para a rota não
//     servir de oráculo;
//   - o vetor de embedding do conteúdo antigo é INVALIDADO: manter vetor de texto
//     que não existe mais faria a busca semântica devolver a memória errada.

// MemoryVersion é uma revisão anterior preservada da memória.
type MemoryVersion struct {
	Revision  int       `json:"revision"`
	Content   string    `json:"content"`
	ActorID   string    `json:"actor_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	// `EncryptedContent` existe SÓ no disco, como no registro principal: o
	// histórico também guarda texto do usuário e não pode vazar em claro.
	EncryptedContent string `json:"encrypted_content,omitempty"`
}

// MaxMemoryVersions limita o histórico por memória. O mais antigo é descartado
// quando o limite é atingido: guardar tudo para sempre é retenção indefinida.
const MaxMemoryVersions = 20

var ErrMemoryContentRequired = errors.New("memory content is required")

// UpdateMemoryForActor altera o conteúdo de uma memória que o ator PODE ler,
// preservando a revisão anterior no histórico.
func (s *ContextStore) UpdateMemoryForActor(ctx context.Context, projectID, memoryID, actorID, content, reason string) (Memory, error) {
	projectID = strings.TrimSpace(projectID)
	memoryID = strings.TrimSpace(memoryID)
	if projectID == "" || memoryID == "" {
		return Memory{}, ErrMemoryNotFound
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return Memory{}, ErrMemoryContentRequired
	}
	if len(content) > 64<<10 {
		return Memory{}, errors.New("memory content is too long")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.memories[projectID]
	index := -1
	for position, memory := range stored {
		if strings.TrimSpace(memory.ID) != memoryID {
			continue
		}
		if !memoryVisibleTo(memory, actorID) {
			// Mesmo erro de inexistente: não se revela a existência de memória
			// privada alheia pela tentativa de alteração.
			return Memory{}, ErrMemoryNotFound
		}
		index = position
		break
	}
	if index < 0 {
		return Memory{}, ErrMemoryNotFound
	}
	current := stored[index]
	if current.Content == content {
		// Sem mudança real não se cria revisão: histórico é de alterações.
		return current, nil
	}

	updated := current
	updated.Versions = append(append([]MemoryVersion(nil), current.Versions...), MemoryVersion{
		Revision:  current.Revision,
		Content:   current.Content,
		ActorID:   strings.TrimSpace(actorID),
		Reason:    strings.TrimSpace(reason),
		UpdatedAt: time.Now().UTC(),
	})
	if len(updated.Versions) > MaxMemoryVersions {
		updated.Versions = updated.Versions[len(updated.Versions)-MaxMemoryVersions:]
	}
	updated.Revision = current.Revision + 1
	updated.Content = content
	// O vetor descrevia o texto ANTIGO: mantê-lo faria a busca devolver a memória
	// errada. Sem embedder disponível o vetor é simplesmente descartado.
	updated.Embedding = nil

	previous := append([]Memory(nil), stored...)
	replaced := append([]Memory(nil), stored[:index]...)
	replaced = append(replaced, updated)
	replaced = append(replaced, stored[index+1:]...)
	s.memories[projectID] = replaced
	if s.root != "" {
		if err := s.writeMemoriesFile(projectID, replaced); err != nil {
			s.memories[projectID] = previous
			return Memory{}, err
		}
	}
	return updated, nil
}

// MemoryHistoryForActor devolve o histórico de uma memória visível ao ator, da
// revisão mais antiga para a mais recente.
func (s *ContextStore) MemoryHistoryForActor(projectID, memoryID, actorID string) ([]MemoryVersion, error) {
	projectID = strings.TrimSpace(projectID)
	memoryID = strings.TrimSpace(memoryID)
	if projectID == "" || memoryID == "" {
		return nil, ErrMemoryNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, memory := range s.memories[projectID] {
		if strings.TrimSpace(memory.ID) != memoryID {
			continue
		}
		if !memoryVisibleTo(memory, actorID) {
			return nil, ErrMemoryNotFound
		}
		history := append([]MemoryVersion(nil), memory.Versions...)
		return history, nil
	}
	return nil, ErrMemoryNotFound
}
