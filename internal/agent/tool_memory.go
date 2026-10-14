package agent

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// FERRAMENTA DE MEMÓRIA PARA OS AGENTES (D-11 / E-618).
//
// A LACUNA QUE ELA FECHA: até aqui a memória entrava no contexto por INJECÇÃO (o preâmbulo), e
// um agente **não podia pedir memória** — não havia ferramenta. O mandato pede integração a
// MCPs/skills; sem ferramenta, não há pedido possível.
//
// E ELA NASCE COM UM LIMITE, DITO NA DESCRIÇÃO: ***serve o conhecimento PARTILHADO do projeto
// (visibilidade `organization`) e NÃO serve memória `private`.*** A razão é medida, não
// estética: a ferramenta corre no runtime da missão, e o runtime **não conhece o ator**
// (`ToolContext` não tem `ActorID` e o registo da missão não tem criador — D-12). Servir
// `private` exigiria um filtro por ator que aqui não existe.
//
// PORQUE ISTO É SEGURO, E NÃO APENAS HONESTO: com um escopo SEM ator, o isolamento do próprio
// store exclui a memória privada por construção (`memory_isolation.go` — `memoryInScope` exige
// `actorID != "" && memory.ActorID == actorID`). Não é uma verificação que eu escrevi e posso
// esquecer: é o mesmo caminho que o contexto já usa. *Não há como vazar o que nem se procura.*
//
// A CAPACIDADE É `workspace:read`, E NÃO UMA NOVA: a política de capacidades tem LISTA FECHADA
// (`capability_policy.go`), e o próprio repositório espera que a recuperação de memória seja
// governada por `workspace:read` (`evaluation.go`, caso `memory.context`). Inventar um
// `memory:read` seria alargar a política para servir uma ferramenta — o contrário do que se quer.

// LimiteDeMemoriasPorFerramenta é o teto de resultados devolvidos a um agente. Um agente que
// peça "tudo" enche o contexto e empurra para fora o que interessa.
const LimiteDeMemoriasPorFerramenta = 8

type memorySearchTool struct {
	store *ContextStore
}

func (t memorySearchTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:    "memory.search",
		Version: "1",
		Description: "Buscar o conhecimento PARTILHADO do projeto (memória com visibilidade " +
			"`organization`). Memória `private` NÃO é servida por esta ferramenta: ela corre no " +
			"runtime da missão, que não conhece o ator, e servir `private` exigiria um filtro por " +
			"ator que aqui não existe. Devolve até " +
			strconv.Itoa(LimiteDeMemoriasPorFerramenta) + " trechos, já sem credenciais.",
		// Ler memória partilhada não muda nada no mundo: risco de LEITURA.
		Risk: RiskRead,
		// A capacidade existente que o repositório já usa para contexto de memória.
		Scopes:         []string{"workspace:read"},
		RequiredInputs: []string{"query"},
	}
}

func (t memorySearchTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if t.store == nil {
		// FALHA FECHADO: sem fonte não se serve memória. Devolver vazio em silêncio faria o
		// agente concluir que "não há nada", que é uma afirmação diferente de "não consultei".
		return ToolResult{}, errors.New("memory.search sem store: recuso servir sem fonte")
	}
	query, _ := input["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return ToolResult{}, errors.New("memory.search exige `query`: uma busca sem pergunta não é uma busca")
	}
	projectID := strings.TrimSpace(toolContext.ProjectID)
	if projectID == "" {
		// SEM PROJETO NÃO SE PROCURA — a mesma regra do bootstrap ("sem projeto declarado,
		// nenhuma memória é injetada"). Aceitar o projeto pelo `input` seria pior: um agente
		// poderia pedir a memória de OUTRO projeto.
		return ToolResult{}, errors.New("memory.search sem projeto no escopo: recuso procurar sem saber de quem")
	}
	// ESCOPO VAZIO, DE PROPÓSITO: sem ator, o isolamento exclui `private` por construção.
	memorias, err := t.store.SearchMemoriesInScope(ctx, projectID, MemoryScope{}, query, LimiteDeMemoriasPorFerramenta)
	if err != nil {
		return ToolResult{}, err
	}
	type trecho struct {
		ID      string `json:"id"`
		Kind    string `json:"kind,omitempty"`
		Content string `json:"content"`
	}
	// REDAÇÃO NA SAÍDA: o mesmo tratamento que o contexto entregue ao modelo já recebe. Uma
	// ferramenta nova não pode ser a porta por onde entra o que a outra filtra.
	trechos := make([]trecho, 0, len(memorias))
	for _, memoria := range memorias {
		trechos = append(trechos, trecho{
			ID:      memoria.ID,
			Kind:    memoria.Kind,
			Content: RedactDLP(memoria.Content),
		})
	}
	return ToolResult{Value: map[string]any{
		"memories": trechos,
		"count":    len(trechos),
		"scope":    "conhecimento partilhado do projeto (memória privada não é servida por aqui)",
	}}, nil
}
