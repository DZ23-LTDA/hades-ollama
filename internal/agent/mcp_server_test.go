package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func novoServidorDeTeste(t *testing.T, projetosPermitidos ...string) (*ServidorMCP, *ContextStore, string) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("projeto partilhado", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	permitidos := map[string]bool{projeto.ID: true}
	for _, id := range projetosPermitidos {
		permitidos[id] = true
	}
	servidor := NovoServidorMCP(store, "org-a", func(projectID string) bool { return permitidos[projectID] })
	return servidor, store, projeto.ID
}

func chamarFerramentaDeMemoria(t *testing.T, servidor *ServidorMCP, projectID, query string) (any, *ErroMCP) {
	t.Helper()
	params, err := json.Marshal(map[string]any{
		"name":      NomeDaFerramentaDeMemoria,
		"arguments": map[string]any{"project_id": projectID, "query": query},
	})
	if err != nil {
		t.Fatal(err)
	}
	return servidor.Responder(context.Background(), "tools/call", params)
}

// A HONESTIDADE DO `initialize`: capacidades declaradas, e a ORGANIZAÇÃO NÃO VAI NA RESPOSTA —
// é identidade de sessão, não informação pública.
func TestMCPInitializeDeclaraCapacidadesESemOrganizacao(t *testing.T) {
	servidor, _, _ := novoServidorDeTeste(t)
	resultado, erro := servidor.Responder(context.Background(), "initialize", nil)
	if erro != nil {
		t.Fatalf("initialize devia correr: %v", erro)
	}
	serializado, _ := json.Marshal(resultado)
	if !strings.Contains(string(serializado), ProtocoloMCP) {
		t.Fatalf("devia declarar a versão do protocolo: %s", serializado)
	}
	if !strings.Contains(string(serializado), "hades-memory") {
		t.Fatalf("devia declarar o serverInfo: %s", serializado)
	}
	if strings.Contains(string(serializado), "org-a") {
		t.Fatalf("*** a organização não pode ir na resposta do initialize ***: %s", serializado)
	}
}

// A FERRAMENTA É UMA SÓ, e declara os inputs obrigatórios — o cliente sabe o que tem de enviar.
func TestMCPListaAFerramentaDeMemoriaComInputsObrigatorios(t *testing.T) {
	servidor, _, _ := novoServidorDeTeste(t)
	resultado, erro := servidor.Responder(context.Background(), "tools/list", nil)
	if erro != nil {
		t.Fatal(erro)
	}
	serializado, _ := json.Marshal(resultado)
	if !strings.Contains(string(serializado), NomeDaFerramentaDeMemoria) {
		t.Fatalf("a ferramenta de memória devia estar listada: %s", serializado)
	}
	if !strings.Contains(string(serializado), "project_id") || !strings.Contains(string(serializado), "query") {
		t.Fatalf("os inputs obrigatórios deviam estar declarados: %s", serializado)
	}
}

// A GARANTIA CENTRAL: serve o conhecimento do PROJETO e não a memória `private` de outro ator.
func TestMCPNaoServeMemoriaPrivadaDeOutroAtor(t *testing.T) {
	servidor, store, projectID := novoServidorDeTeste(t)
	ctx := context.Background()
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_privada", ProjectID: projectID, Kind: "fact",
		Content: "anotação privada sobre o orçamento", Visibility: MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_equipe", ProjectID: projectID, Kind: "fact",
		Content: "decisão de orçamento da equipe",
	}, "bruno"); err != nil {
		t.Fatal(err)
	}

	resultado, erro := chamarFerramentaDeMemoria(t, servidor, projectID, "orçamento")
	if erro != nil {
		t.Fatalf("a chamada devia correr: %v", erro)
	}
	serializado, _ := json.Marshal(resultado)
	if strings.Contains(string(serializado), "mem_privada") || strings.Contains(string(serializado), "anotação privada") {
		t.Fatalf("*** o servidor MCP serviu memória privada de outro ator ***: %s", serializado)
	}
	if !strings.Contains(string(serializado), "mem_equipe") {
		t.Fatalf("devia servir o conhecimento do projeto: %s", serializado)
	}
}

// ***O PROJETO NÃO PODE SER ESCOLHIDO PELO CLIENTE.*** Um projeto que o verificador recusa devolve
// "unknown project" — e a mesma resposta para "não existe" e "não é seu", para não dar pistas.
func TestMCPRecusaProjetoForaDoEscopo(t *testing.T) {
	servidor, _, _ := novoServidorDeTeste(t)
	if _, erro := chamarFerramentaDeMemoria(t, servidor, "prj_de_outra_organizacao", "qualquer"); erro == nil {
		t.Fatal("*** um projeto de outra organização não pode ser aceite ***")
	} else if !strings.Contains(erro.Mensagem, "unknown project") {
		t.Fatalf("mensagem inesperada: %v", erro)
	}
}

// E SEM VERIFICADOR O SERVIDOR RECUSA PROCURAR — não há caminho em que o `project_id` entre por
// esquecimento.
func TestMCPSemVerificadorDeProjetoRecusa(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("projeto", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	servidor := NovoServidorMCP(store, "org-a", nil)
	if _, erro := chamarFerramentaDeMemoria(t, servidor, projeto.ID, "x"); erro == nil {
		t.Fatal("sem verificador de projeto o servidor tem de RECUSAR")
	}
}

// Ferramenta desconhecida e método desconhecido são erros de protocolo, não silêncios.
func TestMCPRecusaFerramentaEMetodoDesconhecidos(t *testing.T) {
	servidor, _, projectID := novoServidorDeTeste(t)
	params, _ := json.Marshal(map[string]any{"name": "outra.ferramenta", "arguments": map[string]any{}})
	if _, erro := servidor.Responder(context.Background(), "tools/call", params); erro == nil {
		t.Fatal("ferramenta desconhecida devia ser recusada")
	}
	if _, erro := servidor.Responder(context.Background(), "resources/read", nil); erro == nil || erro.Codigo != -32601 {
		t.Fatalf("método desconhecido devia dar -32601: %+v", erro)
	}
	// E uma procura sem pergunta é recusada.
	if _, erro := chamarFerramentaDeMemoria(t, servidor, projectID, "   "); erro == nil {
		t.Fatal("procura sem pergunta devia ser recusada")
	}
}
