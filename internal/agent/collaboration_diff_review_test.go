package agent

// FASE 11 do adendo V3: revisao de diffs com feedback ao agente.
//
// O store de colaboracao ja' existia (Comment, CollaborationStore, VinculoDoComentario). O que estes
// testes provam e' a EXTENSAO: ancoragem em diff, resolucao explicita, obsolescencia e -- a regra
// central da fase -- **o agente autor nao pode fechar o proprio achado**.

import (
	"errors"
	"path/filepath"
	"testing"
)

func storeDeRevisaoParaTeste(t *testing.T) *CollaborationStore {
	t.Helper()
	s, err := NewCollaborationStore(filepath.Join(t.TempDir(), "collab"))
	if err != nil {
		t.Fatalf("NewCollaborationStore: %v", err)
	}
	return s
}

// TestComentarioDeDiffGuardaAncoragem prova que a ancoragem sobrevive a' gravacao e a' leitura.
func TestComentarioDeDiffGuardaAncoragem(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	com, err := s.AddCommentWithLink("proj", "humano-1", "esta linha pode nil-deref", VinculoDoComentario{
		File: "internal/agent/x.go", Line: 42, Side: "new", CommitRef: "abc123",
	})
	if err != nil {
		t.Fatalf("AddCommentWithLink: %v", err)
	}
	if com.File != "internal/agent/x.go" || com.Line != 42 || com.Side != "new" || com.CommitRef != "abc123" {
		t.Fatalf("ancoragem perdida no retorno: %+v", com)
	}
	lidos := s.Comments("proj")
	if len(lidos) != 1 || lidos[0].File != "internal/agent/x.go" || lidos[0].Line != 42 {
		t.Fatalf("ancoragem perdida na leitura: %+v", lidos)
	}
}

// TestAncoragemInvalidaERecusadaSemGravar prova as tres formas de alvo invalido.
func TestAncoragemInvalidaERecusadaSemGravar(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	casos := []struct {
		nome    string
		vinculo VinculoDoComentario
	}{
		{"linha sem ficheiro", VinculoDoComentario{Line: 7}},
		{"linha negativa", VinculoDoComentario{File: "a.go", Line: -1}},
		{"lado invalido", VinculoDoComentario{File: "a.go", Line: 1, Side: "meio"}},
	}
	for _, c := range casos {
		if _, err := s.AddCommentWithLink("proj", "humano-1", "corpo", c.vinculo); !errors.Is(err, ErrAncoragemInvalida) {
			t.Errorf("%s: esperado ErrAncoragemInvalida, obtido %v", c.nome, err)
		}
	}
	if lidos := s.Comments("proj"); len(lidos) != 0 {
		t.Fatalf("comentario invalido foi gravado: %+v", lidos)
	}
}

// TestAgenteNaoResolveOProprioComentario e' a regra da FASE 11, e o teste FALHA se a guarda for removida.
func TestAgenteNaoResolveOProprioComentario(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	com, err := s.AddCommentWithLink("proj", "agente-revisor", "achei um problema", VinculoDoComentario{
		File: "a.go", Line: 1, AuthorAgentID: "agente-revisor",
	})
	if err != nil {
		t.Fatalf("AddCommentWithLink: %v", err)
	}
	if _, err := s.ResolveComment("proj", com.ID, "agente-revisor"); !errors.Is(err, ErrAutoResolucaoDeAgente) {
		t.Fatalf("agente resolveu o proprio comentario: esperado ErrAutoResolucaoDeAgente, obtido %v", err)
	}
	lidos := s.Comments("proj")
	if len(lidos) != 1 || lidos[0].Resolved {
		t.Fatalf("comentario foi marcado como resolvido apesar da recusa: %+v", lidos)
	}
}

// TestHumanoResolveComentarioDeAgente prova o caso normal e a trilha de quem resolveu.
func TestHumanoResolveComentarioDeAgente(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	com, err := s.AddCommentWithLink("proj", "agente-revisor", "achei um problema", VinculoDoComentario{
		File: "a.go", Line: 1, AuthorAgentID: "agente-revisor",
	})
	if err != nil {
		t.Fatalf("AddCommentWithLink: %v", err)
	}
	resolvido, err := s.ResolveComment("proj", com.ID, "humano-1")
	if err != nil {
		t.Fatalf("ResolveComment: %v", err)
	}
	if !resolvido.Resolved || resolvido.ResolvedBy != "humano-1" || resolvido.ResolvedAt == nil {
		t.Fatalf("trilha de resolucao incompleta: %+v", resolvido)
	}
	// Idempotente: resolver de novo nao troca quem resolveu.
	deNovo, err := s.ResolveComment("proj", com.ID, "humano-2")
	if err != nil {
		t.Fatalf("ResolveComment idempotente: %v", err)
	}
	if deNovo.ResolvedBy != "humano-1" {
		t.Fatalf("resolucao reexecutada trocou o autor: %+v", deNovo)
	}
}

// TestComentariosObsoletosNaoSaoApagados prova que um diff novo marca sem destruir historia.
func TestComentariosObsoletosNaoSaoApagados(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	if _, err := s.AddCommentWithLink("proj", "humano-1", "no commit velho", VinculoDoComentario{
		File: "a.go", Line: 1, CommitRef: "velho",
	}); err != nil {
		t.Fatalf("AddCommentWithLink velho: %v", err)
	}
	if _, err := s.AddCommentWithLink("proj", "humano-1", "no commit novo", VinculoDoComentario{
		File: "a.go", Line: 2, CommitRef: "novo",
	}); err != nil {
		t.Fatalf("AddCommentWithLink novo: %v", err)
	}
	marcados, err := s.MarkCommentsStale("proj", "velho")
	if err != nil {
		t.Fatalf("MarkCommentsStale: %v", err)
	}
	if marcados != 1 {
		t.Fatalf("esperado 1 marcado, obtido %d", marcados)
	}
	lidos := s.Comments("proj")
	if len(lidos) != 2 {
		t.Fatalf("comentarios foram apagados: %d", len(lidos))
	}
	obsoletos := 0
	for _, c := range lidos {
		if c.Stale {
			obsoletos++
		}
	}
	if obsoletos != 1 {
		t.Fatalf("esperado 1 obsoleto, obtido %d", obsoletos)
	}
}

// TestComentarioSimplesContinuaFuncionando garante compatibilidade com o caminho antigo.
func TestComentarioSimplesContinuaFuncionando(t *testing.T) {
	s := storeDeRevisaoParaTeste(t)
	com, err := s.AddComment("proj", "humano-1", "comentario sem alvo")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if com.File != "" || com.Line != 0 || com.Side != "new" {
		t.Fatalf("comentario simples ganhou ancoragem: %+v", com)
	}
	if _, err := s.ResolveComment("proj", com.ID, "humano-1"); err != nil {
		t.Fatalf("comentario de humano deve poder ser resolvido pelo proprio humano: %v", err)
	}
}
