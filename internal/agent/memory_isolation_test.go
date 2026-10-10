package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func newIsolationStore(t *testing.T) (*ContextStore, Project) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto compartilhado", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project
}

func TestPrivateMemoryIsInvisibleToOtherActorInSameOrganization(t *testing.T) {
	store, project := newIsolationStore(t)
	ctx := context.Background()

	// Aline cria uma memória privada; Bruno cria uma do projeto.
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_aline", ProjectID: project.ID, Kind: "fact",
		Content: "anotação privada da Aline sobre o orçamento", Visibility: MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_equipe", ProjectID: project.ID, Kind: "fact",
		Content: "decisão de orçamento da equipe",
	}, "bruno"); err != nil {
		t.Fatal(err)
	}

	aline, err := store.SearchMemoriesForActor(ctx, project.ID, "aline", "orçamento", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(aline) != 2 {
		t.Fatalf("owner must see both memories, got %d", len(aline))
	}
	bruno, err := store.SearchMemoriesForActor(ctx, project.ID, "bruno", "orçamento", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(bruno) != 1 || bruno[0].ID != "mem_equipe" {
		t.Fatalf("other actor must not see the private memory: %+v", bruno)
	}
	// Sem ator identificado, só o conhecimento do projeto aparece.
	anon, err := store.SearchMemoriesForActor(ctx, project.ID, "", "orçamento", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(anon) != 1 || anon[0].ID != "mem_equipe" {
		t.Fatalf("anonymous reader must not see private memory: %+v", anon)
	}
	// O dono da memória não pode ser forjado pelo corpo: privada do Bruno não
	// vira visível para a Aline.
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_forjada", ProjectID: project.ID, Kind: "fact",
		Content: "tentativa de forjar autoria", Visibility: MemoryVisibilityPrivate, ActorID: "aline",
	}, "bruno"); err != nil {
		t.Fatal(err)
	}
	aline2, err := store.SearchMemoriesForActor(ctx, project.ID, "aline", "forjar", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(aline2) != 0 {
		t.Fatalf("actor id from the payload must be ignored: %+v", aline2)
	}
	bruno2, err := store.SearchMemoriesForActor(ctx, project.ID, "bruno", "forjar", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(bruno2) != 1 {
		t.Fatalf("writer must own what it wrote: %+v", bruno2)
	}
}

func TestMemoryVisibilityRejectsInvalidAndAnonymousPrivate(t *testing.T) {
	store, project := newIsolationStore(t)
	ctx := context.Background()

	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_sem_ator", ProjectID: project.ID, Kind: "fact", Content: "privada sem dono",
		Visibility: MemoryVisibilityPrivate,
	}, ""); !errors.Is(err, ErrMemoryActorRequired) {
		t.Fatalf("private without actor error = %v", err)
	}
	if _, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_invalida", ProjectID: project.ID, Kind: "fact", Content: "visibilidade inventada",
		Visibility: "publica-para-o-mundo",
	}, "aline"); !errors.Is(err, ErrMemoryVisibilityInvalid) {
		t.Fatalf("invalid visibility error = %v", err)
	}
	// Memória antiga, sem visibilidade declarada, continua sendo do projeto.
	created, err := store.AddMemoryForActor(ctx, Memory{
		ID: "mem_legada", ProjectID: project.ID, Kind: "fact", Content: "memória legada",
	}, "aline")
	if err != nil {
		t.Fatal(err)
	}
	if created.Visibility != MemoryVisibilityOrganization || created.ActorID != "" {
		t.Fatalf("default visibility = %+v", created)
	}
	if got, err := store.SearchMemoriesForActor(ctx, project.ID, "bruno", "legada", 20); err != nil || len(got) != 1 {
		t.Fatalf("legacy memory must stay visible to the organization: %+v err=%v", got, err)
	}
	// Nada disso foi gravado: a leitura depois de reabrir o store não encontra as
	// memórias recusadas.
	reopened, err := NewContextStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := reopened.SearchMemoriesForActor(ctx, project.ID, "", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, memory := range stored {
		if memory.ID == "mem_sem_ator" || memory.ID == "mem_invalida" {
			t.Fatalf("rejected memory must not be persisted: %+v", memory)
		}
	}
}

func TestPrivateMemorySurvivesRestartAndKeepsOwner(t *testing.T) {
	store, project := newIsolationStore(t)
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ID: "mem_persistida", ProjectID: project.ID, Kind: "fact",
		Content: "privada persistida", Visibility: MemoryVisibilityPrivate,
	}, "aline"); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewContextStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	memories, err := reopened.SearchMemoriesForActor(context.Background(), project.ID, "aline", "persistida", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(memories) != 1 || memories[0].Visibility != MemoryVisibilityPrivate || memories[0].ActorID != "aline" {
		t.Fatalf("reloaded memory = %+v", memories)
	}
	aline, err := reopened.SearchMemoriesForActor(context.Background(), project.ID, "aline", "persistida", 20)
	if err != nil || len(aline) != 1 {
		t.Fatalf("owner must still see it after restart: %+v err=%v", aline, err)
	}
	bruno, err := reopened.SearchMemoriesForActor(context.Background(), project.ID, "bruno", "persistida", 20)
	if err != nil || len(bruno) != 0 {
		t.Fatalf("other actor must not see it after restart: %+v err=%v", bruno, err)
	}
}

func TestGroundedAnswerRespectsUserIsolation(t *testing.T) {
	store, project := newIsolationStore(t)
	ctx := context.Background()
	// Duas memórias privadas com o mesmo assunto, donos diferentes.
	for _, fixture := range []struct {
		id    string
		actor string
		text  string
	}{
		{"mem_a", "aline", "orçamento secreto da Aline"},
		{"mem_b", "bruno", "orçamento secreto do Bruno"},
	} {
		if _, err := store.AddMemoryForActor(ctx, Memory{
			ID: fixture.id, ProjectID: project.ID, Kind: "fact", Content: fixture.text,
			Source: "nota.txt", Visibility: MemoryVisibilityPrivate, Confidence: 0.9,
		}, fixture.actor); err != nil {
			t.Fatal(err)
		}
	}
	text, citations, err := store.GroundedAnswerForActor(ctx, project.ID, "aline", "orçamento secreto", 5, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if len(citations) != 1 {
		t.Fatalf("citations = %+v", citations)
	}
	if !strings.Contains(text, "Aline") || strings.Contains(text, "Bruno") {
		t.Fatalf("grounded context leaked another actor's memory: %s", text)
	}
	// O mesmo vale para a recuperação pura.
	hits, err := store.RetrieveRelevantForActor(ctx, project.ID, "bruno", "orçamento secreto", 5, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Memory.ID != "mem_b" {
		t.Fatalf("retrieval must be actor scoped: %+v", hits)
	}
}
