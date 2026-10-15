package agent

import (
	"errors"
	"testing"
)

// Complemento 48 / FASE 2 + FASE 6 â€” membros: *humanos e agentes na MESMA lista.*
// O defeito medido: a presenca era chaveada por `ProjectID+UserID`, o que excluia o agente por construcao.

func TestHumanoEAgenteFicamNaMesmaLista(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantHuman, "Ana", "ana"); err != nil {
		t.Fatalf("entrar humano: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "agente-pesquisador", ParticipantAgent, "Pesquisador", "ana"); err != nil {
		t.Fatalf("entrar agente: %v", err)
	}
	// *A ASSERTIVA QUE IMPORTA:* "o time" contem os DOIS na mesma lista, sem separacao por tipo.
	list := store.Participants("proj-1")
	if len(list) != 2 {
		t.Fatalf("o time tem de ter humano E agente, veio %#v", list)
	}
	if list[0].ID != "agente-pesquisador" || list[1].ID != "ana" {
		t.Fatalf("lista ordenada por ID: veio %#v", []string{list[0].ID, list[1].ID})
	}
	// E o filtro de agentes devolve so' o agente - *e' ele que a mencao resolve.*
	agents := store.ProjectAgents("proj-1")
	if len(agents) != 1 || agents[0].ID != "agente-pesquisador" {
		t.Fatalf("ProjectAgents veio %#v", agents)
	}
}

func TestParticipantPersisteComTipoEOrigem(t *testing.T) {
	root := t.TempDir()
	store, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "agente-1", ParticipantAgent, "Agente Um", "supervisor:empresa"); err != nil {
		t.Fatalf("entrar: %v", err)
	}
	reopened, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	list := reopened.Participants("proj-1")
	if len(list) != 1 || list[0].Kind != ParticipantAgent || list[0].Origin != "supervisor:empresa" {
		t.Fatalf("tipo e origem tem de sobreviver a' releitura: %#v", list)
	}
}

func TestTipoDesconhecidoEhRecusado(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "x", "robo", "X", "ana"); !errors.Is(err, ErrUnknownParticipantKind) {
		t.Fatalf("tipo fora de humano/agente tem de ser recusado, veio: %v", err)
	}
	if list := store.Participants("proj-1"); len(list) != 0 {
		t.Fatalf("a recusa nao pode gravar nada, veio %#v", list)
	}
}

func TestParticipantSemOrigemEhRecusado(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantHuman, "Ana", "   "); !errors.Is(err, ErrParticipantWithoutOrigin) {
		t.Fatalf("entrada sem quem declarou tem de ser recusada, veio: %v", err)
	}
}

func TestEntrarDeNovoAtualizaEmVezDeDuplicar(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantHuman, "Ana", "ana"); err != nil {
		t.Fatalf("primeira entrada: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantAgent, "Ana (bot)", "sistema"); err != nil {
		t.Fatalf("segunda entrada: %v", err)
	}
	list := store.Participants("proj-1")
	if len(list) != 1 {
		t.Fatalf("entrar de novo ATUALIZA, nao duplica: veio %d", len(list))
	}
	if list[0].Kind != ParticipantAgent || list[0].Name != "Ana (bot)" {
		t.Fatalf("a entrada tem de reflectir a actualizacao: %#v", list[0])
	}
}

func TestParticipantsSaoPorProjeto(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-A", "ana", ParticipantHuman, "Ana", "ana"); err != nil {
		t.Fatalf("proj-A: %v", err)
	}
	if list := store.Participants("proj-B"); len(list) != 0 {
		t.Fatalf("membro de um projeto nao pode aparecer noutro: %#v", list)
	}
}
