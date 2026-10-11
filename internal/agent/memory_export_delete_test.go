package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Estágio 15 — evidência obrigatória: exportação e exclusão da memória.
//
// As propriedades que estes testes travam:
//  1. exportar devolve SOMENTE o que o ator pode ler;
//  2. a exclusão sobrevive ao REINÍCIO do store (critério do estágio);
//  3. apagar memória privada alheia falha com o MESMO erro de inexistente, para
//     não revelar a existência de dado privado.
func memoryStoreWithActors(t *testing.T) (*ContextStore, string) {
	t.Helper()
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto de memória", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "conhecimento do time", Visibility: MemoryVisibilityOrganization,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "anotação privada da ana", Visibility: MemoryVisibilityPrivate,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "anotação privada do bruno", Visibility: MemoryVisibilityPrivate,
	}, "bruno")
	if err != nil {
		t.Fatal(err)
	}
	return store, project.ID
}

func TestExportMemoriesForActorNeverLeaksPrivateMemory(t *testing.T) {
	store, projectID := memoryStoreWithActors(t)
	export, err := store.ExportMemoriesForActor(projectID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 2 || len(export.Memories) != 2 {
		t.Fatalf("ana deveria exportar o conhecimento do projeto e a memória dela: %+v", export)
	}
	for _, memory := range export.Memories {
		if strings.Contains(memory.Content, "bruno") {
			t.Fatalf("a exportação vazou memória privada de outro ator: %+v", memory)
		}
	}
	// O export carrega a identidade de quem exportou e quando, para auditoria.
	if export.ActorID != "ana" || export.GeneratedAt.IsZero() {
		t.Fatalf("o export precisa registrar o ator e a data de geração: %+v", export)
	}
	// Sem projeto, o export é recusado em vez de devolver tudo.
	if _, err := store.ExportMemoriesForActor("  ", "ana", false); err == nil {
		t.Fatal("export sem projeto precisa ser recusado")
	}
}

func TestDeleteMemoryForActorPersistsAcrossRestart(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto de exclusão", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	mine, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "apagar isto", Visibility: MemoryVisibilityPrivate,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "memória do bruno", Visibility: MemoryVisibilityPrivate,
	}, "bruno")
	if err != nil {
		t.Fatal(err)
	}
	shared, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "conhecimento do time", Visibility: MemoryVisibilityOrganization,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}

	// 1) O ator apaga a PRÓPRIA memória.
	if err := store.DeleteMemoryForActor(project.ID, mine.ID, "ana"); err != nil {
		t.Fatalf("apagar a própria memória deveria funcionar: %v", err)
	}
	// 2) Apagar memória privada ALHEIA falha como inexistente — sem revelar nada.
	if err := store.DeleteMemoryForActor(project.ID, other.ID, "ana"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("memória privada alheia precisa falhar como inexistente, veio %v", err)
	}
	// 3) ID inexistente dá o MESMO erro: as duas respostas são indistinguíveis.
	if err := store.DeleteMemoryForActor(project.ID, "mem_nao_existe", "ana"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("memória inexistente precisa dar o mesmo erro, veio %v", err)
	}
	// 4) Conhecimento do projeto é compartilhado: o ator autorizado pode remover.
	if err := store.DeleteMemoryForActor(project.ID, shared.ID, "ana"); err != nil {
		t.Fatalf("conhecimento do projeto deveria ser removível: %v", err)
	}

	// 5) REINÍCIO: um store novo sobre o mesmo diretório vê exatamente o que sobrou.
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	export, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 0 {
		t.Fatalf("nada deveria sobrar para ana, veio %+v", export.Memories)
	}
	exportBruno, err := restarted.ExportMemoriesForActor(project.ID, "bruno", false)
	if err != nil {
		t.Fatal(err)
	}
	if exportBruno.Count != 1 || !strings.Contains(exportBruno.Memories[0].Content, "bruno") {
		t.Fatalf("a memória do bruno precisa sobreviver intacta: %+v", exportBruno.Memories)
	}
	// E a exclusão recusada realmente NÃO apagou nada, mesmo após o reinício.
	if err := restarted.DeleteMemoryForActor(project.ID, other.ID, "ana"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("a recusa precisa continuar valendo após o reinício, veio %v", err)
	}
	if export, err := restarted.ExportMemoriesForActor(project.ID, "bruno", false); err != nil || export.Count != 1 {
		t.Fatalf("o bruno ainda precisa ter a memória dele: %v %+v", err, export.Memories)
	}
}

func TestExportSurvivesRestartWithSameContent(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto reinício", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "persistir isto", Visibility: MemoryVisibilityPrivate,
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	before, err := store.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	after, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if before.Count != after.Count || after.Count != 1 {
		t.Fatalf("o conteúdo exportado precisa sobreviver ao reinício: antes=%d depois=%d", before.Count, after.Count)
	}
	if before.Memories[0].ID != after.Memories[0].ID || before.Memories[0].Content != after.Memories[0].Content {
		t.Fatalf("a memória mudou no reinício: antes=%+v depois=%+v", before.Memories[0], after.Memories[0])
	}
}

func TestExportOmitsEmbeddingsUnlessRequested(t *testing.T) {
	store, projectID := memoryStoreWithActors(t)
	without, err := store.ExportMemoriesForActor(projectID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, memory := range without.Memories {
		if len(memory.Embedding) != 0 {
			t.Fatalf("o export padrão não deve carregar vetores: %+v", memory)
		}
	}
	with, err := store.ExportMemoriesForActor(projectID, "ana", true)
	if err != nil {
		t.Fatal(err)
	}
	if with.Count != without.Count {
		t.Fatalf("incluir embedding não muda a quantidade: %d != %d", with.Count, without.Count)
	}
}
