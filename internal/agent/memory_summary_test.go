package agent

import (
	"context"
	"strings"
	"testing"
)

// Atividade do estágio 15: RESUMOS.
//
// As propriedades travadas aqui:
//  1. o resumo é FIEL: todo trecho citado aparece na memória de origem — o digest
//     copia, não interpreta;
//  2. ele DECLARA a origem do texto (`summarizer`), para o produto não sugerir que
//     houve modelo quando não houve;
//  3. o escopo é herdado da busca: memória privada alheia não entra no resumo;
//  4. é DETERMINÍSTICO: mesma entrada, mesmo texto;
//  5. sem fonte relevante ele diz que não encontrou, em vez de inventar;
//  6. o teto de fontes é declarado como truncamento, não escondido.
func summaryStore(t *testing.T) (*ContextStore, Project) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto de resumo", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project
}

func TestMemorySummaryIsFaithfulAndDeclaresItsEngine(t *testing.T) {
	store, project := summaryStore(t)
	conteudo := "a equipe decidiu usar PostgreSQL para o núcleo"
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "decision", Content: conteudo, Source: "ata.md",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	summary, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "PostgreSQL", 8, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Summarizer != (DigestSummarizer{}).Name() {
		t.Fatalf("o resumo precisa declarar o motor usado: %q", summary.Summarizer)
	}
	if len(summary.Citations) != 1 {
		t.Fatalf("o resumo precisa citar a memória: %+v", summary.Citations)
	}
	// FIDELIDADE: cada trecho citado existe na memória de origem.
	citacao := summary.Citations[0]
	if !strings.Contains(conteudo, citacao.Snippet) {
		t.Fatalf("o resumo precisa copiar trecho real: citação=%q conteúdo=%q", citacao.Snippet, conteudo)
	}
	if !strings.Contains(summary.Text, citacao.Snippet) {
		t.Fatal("o texto do resumo precisa conter o trecho citado, não uma paráfrase")
	}
	if summary.Truncated {
		t.Fatalf("com uma fonte só não há truncamento: %+v", summary)
	}
}

func TestMemorySummaryIsDeterministicAndBounded(t *testing.T) {
	store, project := summaryStore(t)
	for index := range 3 {
		if _, err := store.AddMemoryForActor(context.Background(), Memory{
			ProjectID: project.ID, Kind: "note", Content: "anotação número " + string(rune('a'+index)) + " sobre o núcleo",
			Source: "caderno-" + string(rune('a'+index)),
		}, "ana"); err != nil {
			t.Fatal(err)
		}
	}
	primeiro, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "núcleo", 2, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "núcleo", 2, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if primeiro.Text != segundo.Text {
		t.Fatalf("o digest precisa ser determinístico:\n1=%q\n2=%q", primeiro.Text, segundo.Text)
	}
	if primeiro.Count > 2 {
		t.Fatalf("o teto de fontes precisa valer: %d", primeiro.Count)
	}
	if !primeiro.Truncated {
		t.Fatal("cortar material precisa ser declarado como truncamento, não escondido")
	}
	// O teto global é respeitado mesmo se o chamador pedir mais.
	exagerado, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "núcleo", 9999, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if exagerado.Count > MaxMemorySummarySources {
		t.Fatalf("o teto global precisa valer: %d", exagerado.Count)
	}
}

func TestMemorySummaryInheritsScopeIsolation(t *testing.T) {
	store, project := summaryStore(t)
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "segredo privado da ana sobre o núcleo",
		Visibility: MemoryVisibilityPrivate,
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "contexto da missão reservada sobre o núcleo", MissionID: "missao-x",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	// Bruno: sem a memória privada da ana.
	bruno, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "bruno"}, "núcleo", 8, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(bruno.Text, "segredo privado da ana") {
		t.Fatalf("o resumo VAZOU memória privada de outro ator: %s", bruno.Text)
	}
	// Sem missão: sem a memória da missão.
	foraDaMissao, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "núcleo", 8, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(foraDaMissao.Text, "missão reservada") {
		t.Fatalf("o resumo VAZOU memória de outra missão: %s", foraDaMissao.Text)
	}
	// Na missão certa, aparece.
	naMissao, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana", MissionID: "missao-x"}, "núcleo", 8, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(naMissao.Text, "missão reservada") {
		t.Fatalf("a missão dona precisa ver o próprio material: %s", naMissao.Text)
	}
}

func TestMemorySummaryWithoutRelevantSourceDoesNotInvent(t *testing.T) {
	store, project := summaryStore(t)
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "assunto completamente diferente",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	summary, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "tópico inexistente", 8, 0.9, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 0 {
		t.Fatalf("sem fonte relevante não há citação: %+v", summary.Citations)
	}
	if !strings.Contains(summary.Text, "não há fontes relevantes") && !strings.Contains(summary.Text, "não encontrou") {
		t.Fatalf("sem base, o resumo precisa DIZER que não encontrou: %q", summary.Text)
	}
}

// O contrato aceita um resumidor injetado, e o resultado diz qual foi usado: o
// produto não pode sugerir digest quando houve modelo, nem o contrário.
type fakeSummarizer struct{}

func (fakeSummarizer) Name() string { return "modelo-de-teste" }
func (fakeSummarizer) Summarize(_ context.Context, query string, sources []ScoredMemory) (string, []Citation, error) {
	citations := make([]Citation, 0, len(sources))
	for index, source := range sources {
		citations = append(citations, Citation{Index: index + 1, Source: source.Memory.Source, Snippet: source.Memory.Content})
	}
	return "resumo por modelo para: " + query, citations, nil
}

func TestMemorySummaryAcceptsInjectedSummarizer(t *testing.T) {
	store, project := summaryStore(t)
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "material para resumir",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	summary, err := store.SummarizeMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "material", 8, 0, fakeSummarizer{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Summarizer != "modelo-de-teste" {
		t.Fatalf("o resultado precisa declarar o resumidor injetado: %q", summary.Summarizer)
	}
	if !strings.Contains(summary.Text, "resumo por modelo") {
		t.Fatalf("o texto precisa vir do resumidor injetado: %q", summary.Text)
	}
}
