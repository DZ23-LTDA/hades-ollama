package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SERVIDOR MCP DO HADES — EXPOR A MEMÓRIA PELO PROTOCOLO (E-646).
//
// A AUDITORIA DO E-645 mediu o que faltava: o Hades é um CLIENTE MCP solido (stdio + remoto, OAuth,
// anti-SSRF, segredos por organização) e **não é servidor** — nenhum respondedor de `initialize`,
// `tools/list` ou `tools/call` existia. Logo nenhum cliente MCP (Claude Desktop, Cursor, outro
// harness) podia consultar a memória do Hades.
//
// DUAS DECISÕES DE SEGURANÇA, E SÃO O CENTRO DESTE FICHEIRO:
//
//  1. ***A ORGANIZAÇÃO VEM DA CONSTRUÇÃO, NUNCA DOS ARGUMENTOS.*** O cliente MCP não tem como pedir
//     outra organização: não existe campo para isso. Um campo de organização no `tools/call` seria
//     um pedido de tenant escrito pelo atacante.
//  2. ***O PROJETO É VERIFICADO POR CONSTRUÇÃO.*** A procura é por projeto, e o servidor exige um
//     verificador (`verificarProjeto`) fornecido por quem monta a superfície autenticada. Sem
//     verificador, a ferramenta RECUSA — *não há caminho em que um `project_id` de outra
//     organização seja aceite por esquecimento.*
//
// E a visibilidade segue a regra da casa: o escopo vai SEM ator, pelo que a memória `private` é
// excluída por construção (`memory_isolation.go`) — o mesmo mecanismo das outras superfícies.

// ProtocoloMCP é a versão do protocolo que este servidor responde.
const ProtocoloMCP = "2025-06-18"

// NomeDaFerramentaDeMemoria é o nome exposto aos clientes MCP.
const NomeDaFerramentaDeMemoria = "hades.memory.search"

// LimiteDeMemoriasNoMCP é o teto de resultados devolvidos a um cliente MCP.
const LimiteDeMemoriasNoMCP = 8

var (
	ErrMCPMetodoDesconhecido      = errors.New("mcp method not supported")
	ErrMCPFerramentaDesconhecida  = errors.New("mcp tool not found")
	ErrMCPSemVerificadorDeProjeto = errors.New("mcp server requires a project verifier")
)

// ServidorMCP responde a pedidos MCP para UMA organização, fixada na construção.
type ServidorMCP struct {
	store            *ContextStore
	organizationID   string
	verificarProjeto func(projectID string) bool
}

// NovoServidorMCP cria o respondedor. `verificarProjeto` diz se o projeto pertence à organização
// do chamador; sem ele o servidor recusa procurar (fail-closed).
func NovoServidorMCP(store *ContextStore, organizationID string, verificarProjeto func(projectID string) bool) *ServidorMCP {
	return &ServidorMCP{store: store, organizationID: strings.TrimSpace(organizationID), verificarProjeto: verificarProjeto}
}

// ErroMCP é o erro no formato do protocolo.
type ErroMCP struct {
	Codigo   int    `json:"code"`
	Mensagem string `json:"message"`
}

func (e *ErroMCP) Error() string { return fmt.Sprintf("mcp %d: %s", e.Codigo, e.Mensagem) }

type definicaoDeFerramenta struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Responder trata UM pedido MCP. `metodo` é o método JSON-RPC; `params` são os parâmetros crus.
//
// Devolve `(resultado, nil)` quando corre, e `(nil, ErroMCP)` quando o protocolo pede erro.
func (s *ServidorMCP) Responder(ctx context.Context, metodo string, params json.RawMessage) (any, *ErroMCP) {
	switch metodo {
	case "initialize":
		return map[string]any{
			"protocolVersion": ProtocoloMCP,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "hades-memory", "version": "1"},
			// A organização NÃO vai na resposta: é identidade de sessão, não informação pública.
		}, nil
	case "tools/list":
		return map[string]any{"tools": []definicaoDeFerramenta{{
			Name: NomeDaFerramentaDeMemoria,
			Description: "Busca o conhecimento PARTILHADO do projeto (memória com visibilidade " +
				"`organization`). Memória `private` NÃO é servida por aqui: o escopo vai sem ator, e " +
				"o isolamento exclui-a por construção. Devolve até 8 trechos, já sem credenciais.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string", "description": "projeto da organização da sessão"},
					"query":      map[string]any{"type": "string"},
				},
				"required": []string{"project_id", "query"},
			},
		}}}, nil
	case "tools/call":
		return s.chamarFerramenta(ctx, params)
	default:
		return nil, &ErroMCP{Codigo: -32601, Mensagem: ErrMCPMetodoDesconhecido.Error() + ": " + metodo}
	}
}

func (s *ServidorMCP) chamarFerramenta(ctx context.Context, params json.RawMessage) (any, *ErroMCP) {
	var pedido struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &pedido); err != nil {
		return nil, &ErroMCP{Codigo: -32602, Mensagem: "invalid params"}
	}
	if pedido.Name != NomeDaFerramentaDeMemoria {
		return nil, &ErroMCP{Codigo: -32602, Mensagem: ErrMCPFerramentaDesconhecida.Error() + ": " + pedido.Name}
	}
	var argumentos struct {
		ProjectID string `json:"project_id"`
		Query     string `json:"query"`
	}
	if err := json.Unmarshal(pedido.Arguments, &argumentos); err != nil {
		return nil, &ErroMCP{Codigo: -32602, Mensagem: "invalid arguments"}
	}
	query := strings.TrimSpace(argumentos.Query)
	if query == "" {
		return nil, &ErroMCP{Codigo: -32602, Mensagem: "query is required"}
	}
	projectID := strings.TrimSpace(argumentos.ProjectID)
	if projectID == "" {
		return nil, &ErroMCP{Codigo: -32602, Mensagem: "project_id is required"}
	}
	if s.store == nil {
		return nil, &ErroMCP{Codigo: -32603, Mensagem: "no memory store"}
	}
	// ***O PROJETO É VERIFICADO ANTES DE PROCURAR.*** Sem verificador, recusa: aceitar o projeto
	// pelos argumentos sem o confirmar seria deixar o cliente MCP escolher de quem lê.
	if s.verificarProjeto == nil {
		return nil, &ErroMCP{Codigo: -32603, Mensagem: ErrMCPSemVerificadorDeProjeto.Error()}
	}
	if !s.verificarProjeto(projectID) {
		// A mesma resposta para "não existe" e "não é seu": distinguir seria uma pista.
		return nil, &ErroMCP{Codigo: -32602, Mensagem: "unknown project"}
	}
	memorias, err := s.store.SearchMemoriesInScope(ctx, projectID, MemoryScope{}, query, LimiteDeMemoriasNoMCP)
	if err != nil {
		return nil, &ErroMCP{Codigo: -32603, Mensagem: "search failed"}
	}
	type trecho struct {
		ID      string `json:"id"`
		Kind    string `json:"kind,omitempty"`
		Content string `json:"content"`
	}
	trechos := make([]trecho, 0, len(memorias))
	for _, memoria := range memorias {
		trechos = append(trechos, trecho{ID: memoria.ID, Kind: memoria.Kind, Content: RedactDLP(memoria.Content)})
	}
	texto, err := json.Marshal(map[string]any{
		"memories": trechos,
		"count":    len(trechos),
		"scope":    "conhecimento partilhado do projeto (memoria privada nao e' servida aqui)",
	})
	if err != nil {
		return nil, &ErroMCP{Codigo: -32603, Mensagem: "encode failed"}
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(texto)}},
		"isError": false,
	}, nil
}
