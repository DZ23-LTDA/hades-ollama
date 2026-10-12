package agent

import (
	"errors"
	"testing"
	"time"
)

// Estagio 29 — BASE DE CONHECIMENTO DO INBOX (E-265). O limite declarado dizia "sem base de conhecimento nem
// resposta automatica por IA na conversa".
//
// Enquadramento: a BASE e a SUGESTAO entram; o ENVIO automatico NAO. O teste prova que sugerir e LEITURA PURA.
func TestInboxKnowledgeSugereSemEnviarEIsolaOrganizacoes(t *testing.T) {
	root := t.TempDir()
	store, err := NewInboxStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	org := LocalOrganizationID

	// Artigo sem titulo/conteudo e' recusado: base de conhecimento com artigo vazio nao ajuda ninguem.
	if _, err := store.UpsertKnowledgeArticle(InboxKnowledgeArticle{OrganizationID: org, Title: "  "}, now); !errors.Is(err, ErrInboxKnowledgeInvalid) {
		t.Fatalf("artigo sem titulo precisa ser recusado: %v", err)
	}
	entrega, err := store.UpsertKnowledgeArticle(InboxKnowledgeArticle{
		OrganizationID: org, Title: "Politica de entrega",
		Content: "Entregamos em ate 7 dias uteis para todo o Brasil.",
		Tags:    []string{"entrega", "prazo"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	devolucao, err := store.UpsertKnowledgeArticle(InboxKnowledgeArticle{
		OrganizationID: org, Title: "Como devolver um produto",
		Content: "O cliente tem 7 dias para solicitar devolucao pelo atendimento.",
		Tags:    []string{"devolucao"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Atualizar o MESMO artigo nao duplica.
	atualizado, err := store.UpsertKnowledgeArticle(InboxKnowledgeArticle{
		ID: entrega.ID, OrganizationID: org, Title: "Politica de entrega",
		Content: "Entregamos em ate 5 dias uteis para todo o Brasil.", Tags: []string{"entrega"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if atualizado.ID != entrega.ID {
		t.Fatalf("atualizar precisa manter o ID: %s != %s", atualizado.ID, entrega.ID)
	}
	artigos, err := store.KnowledgeArticles(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(artigos) != 2 {
		t.Fatalf("atualizar nao pode duplicar artigo: %d", len(artigos))
	}

	// BUSCA: titulo pesa mais que conteudo.
	hits, err := store.SearchKnowledge(org, "politica entrega", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].Article.ID != entrega.ID {
		t.Fatalf("a busca precisa achar o artigo de titulo correspondente: %+v", hits)
	}
	// Busca VAZIA nao inventa resultado.
	if vazios, err := store.SearchKnowledge(org, "   ", 5); err != nil || len(vazios) != 0 {
		t.Fatalf("consulta vazia nao pode retornar artigo: %v %+v", err, vazios)
	}

	// SUGESTAO a partir da mensagem recebida.
	sugestao, err := store.SuggestReply(org, "meu produto chegou errado, quero devolucao", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !sugestao.Found || sugestao.ArticleID != devolucao.ID {
		t.Fatalf("a sugestao precisa vir do artigo de devolucao: %+v", sugestao)
	}
	if sugestao.Suggestion == "" {
		t.Fatal("a sugestao precisa trazer o conteudo do artigo")
	}
	// LEITURA PURA: sugerir NAO altera a base (nem contagem, nem UpdatedAt).
	depois, err := store.KnowledgeArticles(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(depois) != len(artigos) {
		t.Fatalf("sugerir nao pode alterar a base: %d -> %d", len(artigos), len(depois))
	}
	for i := range artigos {
		if !artigos[i].UpdatedAt.Equal(depois[i].UpdatedAt) || artigos[i].Content != depois[i].Content {
			t.Fatalf("sugerir alterou o artigo %s", artigos[i].ID)
		}
	}
	// Sem correspondencia: Found=false (nao se inventa resposta).
	semCasar, err := store.SuggestReply(org, "assunto completamente fora do catalogo xyz", 3)
	if err != nil {
		t.Fatal(err)
	}
	if semCasar.Found {
		t.Fatalf("sem correspondencia a sugestao precisa ser Found=false: %+v", semCasar)
	}

	// ISOLAMENTO: outra organizacao nao ve nem busca os artigos.
	if outros, err := store.KnowledgeArticles("org-intrusa"); err != nil || len(outros) != 0 {
		t.Fatalf("artigos nao podem vazar entre organizacoes: %v %+v", err, outros)
	}
	if alheios, err := store.SearchKnowledge("org-intrusa", "entrega", 5); err != nil || len(alheios) != 0 {
		t.Fatalf("busca de outra organizacao nao pode achar artigo: %v %+v", err, alheios)
	}
	if err := store.DeleteKnowledgeArticle("org-intrusa", entrega.ID); !errors.Is(err, ErrInboxKnowledgeMissing) {
		t.Fatalf("apagar artigo alheio precisa responder como inexistente: %v", err)
	}

	// PERSISTENCIA: reinicio REAL sobre o mesmo diretorio.
	reaberta, err := NewInboxStore(root)
	if err != nil {
		t.Fatal(err)
	}
	recuperados, err := reaberta.KnowledgeArticles(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(recuperados) != 2 {
		t.Fatalf("a base precisa sobreviver ao reinicio: %d", len(recuperados))
	}
	// APAGAR persiste.
	if err := reaberta.DeleteKnowledgeArticle(org, entrega.ID); err != nil {
		t.Fatal(err)
	}
	terceira, err := NewInboxStore(root)
	if err != nil {
		t.Fatal(err)
	}
	restantes, err := terceira.KnowledgeArticles(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(restantes) != 1 || restantes[0].ID != devolucao.ID {
		t.Fatalf("apagar precisa persistir: %+v", restantes)
	}
}
