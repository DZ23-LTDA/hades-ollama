package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Atividade do estágio 15: VERSIONAMENTO de memória.
//
// As propriedades travadas aqui:
//  1. alterar preserva a revisão anterior com quem alterou, data e motivo;
//  2. memória privada alheia é recusada com o erro de inexistente (não é oráculo);
//  3. o histórico tem TETO (retenção indefinida não é aceitável);
//  4. o vetor do conteúdo antigo é invalidado (senão a busca devolve a memória errada);
//  5. o histórico sobrevive ao reinício do store.
func versionStore(t *testing.T) (*ContextStore, string) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto versão", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project.ID
}

func TestUpdateMemoryKeepsPreviousRevisionWithProvenance(t *testing.T) {
	store, projectID := versionStore(t)
	original, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: projectID, Kind: "decision", Content: "usar MySQL",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateMemoryForActor(context.Background(), projectID, original.ID, "ana", "usar PostgreSQL", "decisão revista na reunião")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "usar PostgreSQL" {
		t.Fatalf("o conteúdo precisa ser o novo: %q", updated.Content)
	}
	if updated.Revision != 1 {
		t.Fatalf("a revisão precisa subir: %d", updated.Revision)
	}
	if len(updated.Versions) != 1 {
		t.Fatalf("a revisão anterior precisa ser preservada: %+v", updated.Versions)
	}
	previous := updated.Versions[0]
	if previous.Content != "usar MySQL" {
		t.Fatalf("o histórico precisa guardar o texto ANTERIOR: %+v", previous)
	}
	if previous.ActorID != "ana" || previous.Reason != "decisão revista na reunião" || previous.UpdatedAt.IsZero() {
		t.Fatalf("o histórico precisa de quem alterou, motivo e data: %+v", previous)
	}
	// A mesma checagem pelo caminho de leitura do histórico.
	history, err := store.MemoryHistoryForActor(projectID, original.ID, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Content != "usar MySQL" {
		t.Fatalf("histórico inconsistente: %+v", history)
	}
	// Editar com o MESMO conteúdo não cria revisão: histórico é de alterações.
	same, err := store.UpdateMemoryForActor(context.Background(), projectID, original.ID, "ana", "usar PostgreSQL", "sem mudança")
	if err != nil {
		t.Fatal(err)
	}
	if len(same.Versions) != 1 || same.Revision != 1 {
		t.Fatalf("reedição idêntica não pode inventar revisão: %+v", same)
	}
	// Conteúdo vazio é recusado em vez de apagar o conhecimento.
	if _, err := store.UpdateMemoryForActor(context.Background(), projectID, original.ID, "ana", "   ", ""); !errors.Is(err, ErrMemoryContentRequired) {
		t.Fatalf("conteúdo vazio precisa ser recusado, veio %v", err)
	}
}

func TestUpdateMemoryRefusesOtherActorPrivateMemory(t *testing.T) {
	store, projectID := versionStore(t)
	anaMemory, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: projectID, Kind: "note", Content: "segredo da ana", Visibility: MemoryVisibilityPrivate,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	// Bruno não pode alterar nem LER o histórico: mesmo erro de inexistente.
	if _, err := store.UpdateMemoryForActor(context.Background(), projectID, anaMemory.ID, "bruno", "texto do bruno", ""); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("alterar memória privada alheia precisa falhar como inexistente, veio %v", err)
	}
	if _, err := store.MemoryHistoryForActor(projectID, anaMemory.ID, "bruno"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("ler histórico alheio precisa falhar como inexistente, veio %v", err)
	}
	// E a memória da ana continua intacta.
	export, err := store.ExportMemoriesForActor(projectID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 || !strings.Contains(export.Memories[0].Content, "segredo da ana") {
		t.Fatalf("a memória da ana não podia mudar: %+v", export.Memories)
	}
	if export.Memories[0].Revision != 0 {
		t.Fatalf("a revisão não podia avançar: %d", export.Memories[0].Revision)
	}
}

func TestMemoryHistoryIsBoundedAndSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto histórico", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	memory, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "versão 0",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	// Edita além do teto: o histórico precisa parar em MaxMemoryVersions.
	for index := 1; index <= MaxMemoryVersions+5; index++ {
		memory, err = store.UpdateMemoryForActor(context.Background(), project.ID, memory.ID, "ana", "versão "+string(rune('a'+index%26))+"-"+strings.Repeat("x", index), "")
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(memory.Versions) != MaxMemoryVersions {
		t.Fatalf("o histórico precisa respeitar o teto de %d: %d", MaxMemoryVersions, len(memory.Versions))
	}
	// O teto descarta o MAIS ANTIGO: "versão 0" já não está lá.
	if memory.Versions[0].Content == "versão 0" {
		t.Fatal("o teto precisa descartar a revisão mais antiga, não a mais recente")
	}
	// O vetor antigo não pode sobreviver à alteração de conteúdo.
	if len(memory.Embedding) != 0 {
		t.Fatalf("o vetor do texto antigo precisa ser invalidado: %d", len(memory.Embedding))
	}

	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	history, err := restarted.MemoryHistoryForActor(project.ID, memory.ID, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != MaxMemoryVersions {
		t.Fatalf("o histórico precisa sobreviver ao reinício: %d", len(history))
	}
	export, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 || export.Memories[0].Revision != MaxMemoryVersions+5 {
		t.Fatalf("o estado final precisa sobreviver: %+v", export.Memories)
	}
}

func TestUpdateMemoryInvalidatesEmbedding(t *testing.T) {
	store, projectID := versionStore(t)
	memory, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: projectID, Kind: "note", Content: "texto original",
		Embedding: []float32{0.1, 0.2, 0.3},
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateMemoryForActor(context.Background(), projectID, memory.ID, "ana", "texto corrigido", "correção")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Embedding) != 0 {
		t.Fatalf("vetor do texto antigo precisa sair: %+v", updated.Embedding)
	}
}
