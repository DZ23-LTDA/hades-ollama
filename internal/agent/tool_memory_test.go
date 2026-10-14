package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func novaLojaDeMemoria(t *testing.T) (*ContextStore, string) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto compartilhado", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project.ID
}

// A GARANTIA CENTRAL DA FERRAMENTA (D-11/E-618): ela serve o CONHECIMENTO DO PROJETO e ***não***
// a memória `private` de um ator. Não é uma verificação que eu escrevi no Execute e posso
// esquecer: é o isolamento do próprio store a excluí-la, porque o escopo vai SEM ator.
func TestFerramentaDeMemoriaNaoServeMemoriaPrivadaDeOutroAtor(t *testing.T) {
	store, projectID := novaLojaDeMemoria(t)
	ctx := context.Background()

	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_privada", ProjectID: projectID, Kind: "fact",
		Content: "anotação privada da Aline sobre o orçamento", Visibility: MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_equipe", ProjectID: projectID, Kind: "fact",
		Content: "decisão de orçamento da equipe",
	}, "bruno"); err != nil {
		t.Fatal(err)
	}

	ferramenta := memorySearchTool{store: store}
	resultado, err := ferramenta.Execute(ctx, ToolContext{MissionID: "mis_1", ProjectID: projectID}, map[string]any{"query": "orçamento"})
	if err != nil {
		t.Fatalf("a ferramenta recusou uma busca legítima: %v", err)
	}
	serializado := fmt.Sprintf("%+v", resultado.Value)

	if strings.Contains(serializado, "mem_privada") || strings.Contains(serializado, "anotação privada") {
		t.Fatalf("*** a ferramenta SERVIU memória privada de outro ator ***: %s", serializado)
	}
	if !strings.Contains(serializado, "mem_equipe") {
		t.Fatalf("a ferramenta não serviu o conhecimento do projeto: %s", serializado)
	}
	// E diz o que NÃO serve, para quem a usa saber o que não recebe.
	if !strings.Contains(serializado, "privada não é servida") {
		t.Fatalf("a ferramenta não declara o limite de escopo: %s", serializado)
	}
}

// A ferramenta também é servida ao PRÓPRIO ator? Não: ela não conhece ator nenhum. O que isto
// prova é que, para o dono da memória privada, o resultado é o MESMO — o que parece uma perda e
// é a garantia: ***não há caminho em que a ferramenta precise de saber quem pergunta.***
func TestFerramentaDeMemoriaDaOMesmoResultadoIndependentementeDeQuemPergunta(t *testing.T) {
	store, projectID := novaLojaDeMemoria(t)
	ctx := context.Background()
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_privada", ProjectID: projectID, Kind: "fact",
		Content: "anotação privada da Aline sobre o orçamento", Visibility: MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	ferramenta := memorySearchTool{store: store}
	resultado, err := ferramenta.Execute(ctx, ToolContext{ProjectID: projectID}, map[string]any{"query": "orçamento"})
	if err != nil {
		t.Fatal(err)
	}
	if contagem, _ := resultado.Value.(map[string]any)["count"].(int); contagem != 0 {
		t.Fatalf("a ferramenta não pode servir `private` a ninguém: count=%d", contagem)
	}
}

// FALHAS FECHADAS: sem store, sem projeto ou sem pergunta, a ferramenta RECUSA. Devolver vazio em
// silêncio faria o agente concluir "não há nada" — que é uma afirmação diferente de "não
// procurei".
func TestFerramentaDeMemoriaRecusaSemFonteSemProjetoSemPergunta(t *testing.T) {
	store, projectID := novaLojaDeMemoria(t)
	ctx := context.Background()

	casos := []struct {
		nome       string
		ferramenta memorySearchTool
		contexto   ToolContext
		entrada    map[string]any
	}{
		{"sem store", memorySearchTool{}, ToolContext{ProjectID: projectID}, map[string]any{"query": "x"}},
		{"sem projeto", memorySearchTool{store: store}, ToolContext{MissionID: "mis_1"}, map[string]any{"query": "x"}},
		{"sem pergunta", memorySearchTool{store: store}, ToolContext{ProjectID: projectID}, map[string]any{"query": "   "}},
		{"sem entrada", memorySearchTool{store: store}, ToolContext{ProjectID: projectID}, nil},
	}
	for _, caso := range casos {
		if _, err := caso.ferramenta.Execute(ctx, caso.contexto, caso.entrada); err == nil {
			t.Fatalf("%s: a ferramenta tinha de RECUSAR", caso.nome)
		}
	}
}

// O descritor tem de dizer o que a ferramenta NÃO faz, e usar a capacidade que o repositório já
// usa para memória (`workspace:read`), em vez de alargar a política com um escopo novo.
func TestDescritorDaFerramentaDeMemoriaDeclaraEscopoEOLimite(t *testing.T) {
	descritor := memorySearchTool{}.Descriptor()
	if descritor.Name != "memory.search" {
		t.Fatalf("nome inesperado: %q", descritor.Name)
	}
	if descritor.Risk != RiskRead {
		t.Fatalf("ler memória partilhada não muda o mundo: %v", descritor.Risk)
	}
	if len(descritor.Scopes) != 1 || descritor.Scopes[0] != "workspace:read" {
		t.Fatalf("a capacidade tem de ser a que a política já conhece: %v", descritor.Scopes)
	}
	if !strings.Contains(descritor.Description, "private") || !strings.Contains(descritor.Description, "NÃO é servida") {
		t.Fatalf("o descritor tem de declarar que `private` não é servida: %q", descritor.Description)
	}
	if len(descritor.RequiredInputs) != 1 || descritor.RequiredInputs[0] != "query" {
		t.Fatalf("`query` tem de ser entrada obrigatória: %v", descritor.RequiredInputs)
	}
}
