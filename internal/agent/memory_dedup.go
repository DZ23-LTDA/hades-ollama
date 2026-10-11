package agent

import "strings"

// Deduplicação de memória (atividade do estágio 15).
//
// O risco que ela mitiga é declarado no próprio estágio: "contexto inflado". Sem
// dedup, registrar a mesma decisão duas vezes enche o preâmbulo do modelo com a
// mesma linha e consome o orçamento de contexto que outras memórias usariam.
//
// A regra é CONSERVADORA: só é duplicata quando conteúdo, tipo, visibilidade/ator
// e origem coincidem. `kind` diferente é conhecimento diferente; ator ou
// visibilidade diferentes são fronteiras de privacidade; `source` diferente é
// proveniência diferente. Fundir o que não é idêntico perderia informação — e
// perder memória é pior que guardar uma linha repetida.

func (s *ContextStore) duplicateMemory(memory Memory) (Memory, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.duplicateMemoryLocked(memory)
}

func (s *ContextStore) duplicateMemoryLocked(memory Memory) (Memory, bool) {
	for _, existing := range s.memories[memory.ProjectID] {
		if !sameMemoryIdentity(existing, memory) {
			continue
		}
		// Devolve a memória ORIGINAL: quem grava precisa saber que já existia.
		return existing, true
	}
	return Memory{}, false
}

func sameMemoryIdentity(a, b Memory) bool {
	if strings.TrimSpace(a.Content) != strings.TrimSpace(b.Content) {
		return false
	}
	if strings.TrimSpace(a.Kind) != strings.TrimSpace(b.Kind) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(a.Visibility), strings.TrimSpace(b.Visibility)) {
		return false
	}
	if strings.TrimSpace(a.ActorID) != strings.TrimSpace(b.ActorID) {
		return false
	}
	return strings.TrimSpace(a.Source) == strings.TrimSpace(b.Source)
}
