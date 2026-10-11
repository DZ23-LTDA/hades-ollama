package agent

import (
	"context"
	"testing"
)

// Atividade do estágio 15: DEDUPLICAÇÃO de memória. A implementação está em
// `memory_dedup.go`; aqui ficam as propriedades que os testes travam.
func dedupStore(t *testing.T) (*ContextStore, string) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto dedup", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project.ID
}

func TestAddMemoryDeduplicatesIdenticalContent(t *testing.T) {
	store, projectID := dedupStore(t)
	first, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: projectID, Kind: "decision", Content: "usar Go no backend",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: projectID, Kind: "decision", Content: "  usar Go no backend  ",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	// Devolve a memória ORIGINAL, não uma cópia nova: quem grava precisa saber que
	// a memória já existia.
	if second.ID != first.ID {
		t.Fatalf("duplicata deve devolver a memória existente: %s != %s", second.ID, first.ID)
	}
	export, err := store.ExportMemoriesForActor(projectID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 {
		t.Fatalf("o contexto não pode inchar com a mesma decisão duas vezes: %d", export.Count)
	}
}

func TestAddMemoryKeepsDifferentKindVisibilityAndSourceApart(t *testing.T) {
	store, projectID := dedupStore(t)
	base := Memory{ProjectID: projectID, Kind: "note", Content: "mesmo texto", Source: "doc.md"}
	if _, err := store.AddMemoryForActor(context.Background(), base, "ana"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		memory Memory
		actor  string
	}{
		{"tipo diferente", Memory{ProjectID: projectID, Kind: "fact", Content: "mesmo texto", Source: "doc.md"}, "ana"},
		{"origem diferente", Memory{ProjectID: projectID, Kind: "note", Content: "mesmo texto", Source: "outro.md"}, "ana"},
		{"privada do mesmo ator", Memory{ProjectID: projectID, Kind: "note", Content: "mesmo texto", Source: "doc.md", Visibility: MemoryVisibilityPrivate}, "ana"},
		{"privada de outro ator", Memory{ProjectID: projectID, Kind: "note", Content: "mesmo texto", Source: "doc.md", Visibility: MemoryVisibilityPrivate}, "bruno"},
	}
	for _, testCase := range cases {
		if _, err := store.AddMemoryForActor(context.Background(), testCase.memory, testCase.actor); err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}
	}
	// Ana vê: o original e TRÊS diferenças (tipo, origem e a privada dela). A
	// privada do bruno não é visível a ela.
	export, err := store.ExportMemoriesForActor(projectID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 4 {
		t.Fatalf("esperava o original mais três diferenças visíveis à ana, veio %d", export.Count)
	}
	// Bruno vê: o original, o tipo diferente, a origem diferente e a privada DELE
	// — as três primeiras são conhecimento do projeto e continuam compartilhadas.
	bruno, err := store.ExportMemoriesForActor(projectID, "bruno", false)
	if err != nil {
		t.Fatal(err)
	}
	if bruno.Count != 4 {
		t.Fatalf("o bruno vê o conhecimento do projeto e a memória dele: %d", bruno.Count)
	}
	// Nenhuma diferença foi FUNDIDA: quatro gravações distintas geraram quatro
	// memórias, e nenhuma delas virou duplicata por engano.
	if export.Count != bruno.Count {
		t.Fatalf("contagens divergentes entre atores: ana=%d bruno=%d", export.Count, bruno.Count)
	}
}

func TestMemoryDedupSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto dedup reinício", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := store.AddMemoryForActor(context.Background(), Memory{
			ProjectID: project.ID, Kind: "decision", Content: "decisão única",
		}, "ana"); err != nil {
			t.Fatal(err)
		}
	}
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	// Repetir a gravação DEPOIS do reinício também precisa deduplicar: a regra
	// vive no dado persistido, não na memória do processo.
	if _, err := restarted.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "decision", Content: "decisão única",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	export, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 {
		t.Fatalf("três gravações iguais e uma após reinício devem ser UMA memória: %d", export.Count)
	}
}
