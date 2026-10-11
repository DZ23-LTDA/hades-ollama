package agent

import "strings"

// Escopos de memória (atividade do estágio 15).
//
// O estágio pede escopos de **usuário, organização, projeto, workspace, missão,
// agente e equipe**. Organização/projeto já existiam na chave do store, e o escopo
// de usuário veio com o isolamento por ator. Faltavam missão, agente e equipe.
//
// DISTINÇÃO QUE IMPORTA — e que evita um erro clássico:
//
//   - PROPRIEDADE (quem pode ALTERAR/APAGAR): regra por ator (`memoryVisibleTo`).
//     O dono de uma memória continua podendo corrigi-la e removê-la mesmo que ela
//     esteja presa a uma missão antiga. Negar isso deixaria memória órfã e
//     imortal — o oposto de "exclusão" e "retenção" que o estágio pede;
//   - DIVULGAÇÃO (quem pode VER): propriedade **mais** o escopo. Memória presa à
//     missão X não aparece em busca de outro contexto.
//
// Um escopo VAZIO significa "vale para o projeto": é o que preserva todo o
// comportamento anterior. Só memória com escopo explícito fica restrita.

// MemoryScope é o contexto de leitura: quem pergunta e onde.
type MemoryScope struct {
	ActorID   string
	MissionID string
	AgentID   string
	TeamID    string
}

// memoryInScope decide a DIVULGAÇÃO: propriedade primeiro, escopo depois.
func memoryInScope(memory Memory, scope MemoryScope) bool {
	if !memoryVisibleTo(memory, scope.ActorID) {
		return false
	}
	if limit := strings.TrimSpace(memory.MissionID); limit != "" && limit != strings.TrimSpace(scope.MissionID) {
		return false
	}
	if limit := strings.TrimSpace(memory.AgentID); limit != "" && limit != strings.TrimSpace(scope.AgentID) {
		return false
	}
	if limit := strings.TrimSpace(memory.TeamID); limit != "" && limit != strings.TrimSpace(scope.TeamID) {
		return false
	}
	return true
}

func filterMemoriesInScope(memories []Memory, scope MemoryScope) []Memory {
	visible := make([]Memory, 0, len(memories))
	for _, memory := range memories {
		if memoryInScope(memory, scope) {
			visible = append(visible, memory)
		}
	}
	return visible
}
