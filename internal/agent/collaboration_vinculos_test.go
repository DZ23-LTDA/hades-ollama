package agent

import (
	"errors"
	"testing"
)

// Complemento 48 / FASE 2 e FASE 7 — RESPOSTAS VINCULADAS E COMENTARIO EM ENTREGAVEL.
//
// O que existia ANTES desta fatia (medido em 2026-10-15, commit 93f12ac19): `Comment` era uma lista PLANA
// (`collaboration.go:16-22`) — sem pai e sem entregavel. Estes testes fixam o comportamento NOVO: o vinculo
// e' gravado, **persiste**, e o vinculo que aponta para o nada e' **RECUSADO**.

func TestRespostaVinculadaAoPaiEPersiste(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewCollaborationStore(raiz)
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	pai, err := store.AddComment("proj-1", "humano:ana", "precisamos do relatorio")
	if err != nil {
		t.Fatalf("comentario pai: %v", err)
	}
	resposta, err := store.AddCommentWithLink("proj-1", "agente:pesquisador", "assumo: entrego hoje",
		VinculoDoComentario{ParentID: pai.ID})
	if err != nil {
		t.Fatalf("resposta: %v", err)
	}
	if resposta.ParentID != pai.ID {
		t.Fatalf("a resposta tem de apontar para o pai: esperado %q, veio %q", pai.ID, resposta.ParentID)
	}
	respostas := store.Respostas("proj-1", pai.ID)
	if len(respostas) != 1 || respostas[0].ID != resposta.ID {
		t.Fatalf("Respostas tem de devolver a resposta direta, veio %d item(s)", len(respostas))
	}
	// PERSISTENCIA: um store NOVO, lendo o MESMO diretorio, tem de ver o vinculo.
	reaberto, err := NewCollaborationStore(raiz)
	if err != nil {
		t.Fatalf("reabrir store: %v", err)
	}
	recarregados := reaberto.Comments("proj-1")
	if len(recarregados) != 2 {
		t.Fatalf("esperado pai+resposta apos reabrir, veio %d", len(recarregados))
	}
	if recarregados[1].ParentID != pai.ID {
		t.Fatalf("o vinculo NAO sobreviveu a' releitura: %q", recarregados[1].ParentID)
	}
	if len(reaberto.Respostas("proj-1", pai.ID)) != 1 {
		t.Fatal("o filtro de respostas tem de funcionar tambem no store reaberto")
	}
}

func TestRespostaOrfaERecusadaSemGravarNada(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	_, err = store.AddCommentWithLink("proj-1", "humano:ana", "resposta solta no ar",
		VinculoDoComentario{ParentID: "com_nao_existe"})
	if !errors.Is(err, ErrComentarioSemPai) {
		t.Fatalf("resposta orfa tem de ser RECUSADA com ErrComentarioSemPai, veio: %v", err)
	}
	// E NAO PODE DEIXAR RASTRO: recusar depois de gravar seria pior do que nao gravar.
	if comentarios := store.Comments("proj-1"); len(comentarios) != 0 {
		t.Fatalf("a recusa nao pode gravar nada, mas ficaram %d comentario(s)", len(comentarios))
	}
}

func TestRespostaNaoAtravessaProjeto(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	doOutro, err := store.AddComment("proj-A", "humano:ana", "assunto do A")
	if err != nil {
		t.Fatalf("comentario do projeto A: %v", err)
	}
	_, err = store.AddCommentWithLink("proj-B", "humano:bob", "respondendo o A",
		VinculoDoComentario{ParentID: doOutro.ID})
	if !errors.Is(err, ErrComentarioSemPai) {
		t.Fatalf("responder comentario de OUTRO projeto tem de ser recusado, veio: %v", err)
	}
}

func TestComentarioEmEntregavelFicaDeclarado(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	comentario, err := store.AddCommentWithLink("proj-1", "agente:designer", "ajustei o logo",
		VinculoDoComentario{ArtifactID: "art_123"})
	if err != nil {
		t.Fatalf("comentario em entregavel: %v", err)
	}
	if comentario.ArtifactID != "art_123" {
		t.Fatalf("o entregavel tem de ser gravado: %q", comentario.ArtifactID)
	}
	// LIMITE DECLARADO (nao escondido): este store NAO valida o artefato - nao ha registro de entregaveis aqui.
	// O teste FIXA o comportamento atual para que ninguem presuma validacao que nao existe.
	aceito, err := store.AddCommentWithLink("proj-1", "agente:designer", "id que ninguem confere",
		VinculoDoComentario{ArtifactID: "art_que_nao_existe_mesmo"})
	if err != nil || aceito.ArtifactID == "" {
		t.Fatalf("hoje o id e' DECLARADO, nao verificado (err=%v); quando a validacao existir, este teste muda", err)
	}
}

func TestAddCommentSimplesNaoGanhaVinculo(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	simples, err := store.AddComment("proj-1", "humano:ana", "sem vinculo nenhum")
	if err != nil {
		t.Fatalf("comentario simples: %v", err)
	}
	if simples.ParentID != "" || simples.ArtifactID != "" {
		t.Fatalf("o caminho antigo nao pode inventar vinculo (pai=%q entregavel=%q)", simples.ParentID, simples.ArtifactID)
	}
}
