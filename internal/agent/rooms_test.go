package agent

import (
	"errors"
	"testing"
)

// Complemento 48 / FASE 2 — A SALA, com a regra que lhe da' valor:
// **so' entra na sala quem ja' e' membro do projeto.** *Sem isso, a sala seria uma via paralela de permissao.*

func storeComMembros(t *testing.T) *CollaborationStore {
	t.Helper()
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantHuman, "Ana", "ana"); err != nil {
		t.Fatalf("membro humano: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "agente-pesquisador", ParticipantAgent, "Pesquisador", "ana"); err != nil {
		t.Fatalf("membro agente: %v", err)
	}
	return store
}

func TestSalaJuntaHumanoEAgente(t *testing.T) {
	store := storeComMembros(t)
	sala, err := store.CreateRoom("proj-1", "sala-produto", "Produto", "ana")
	if err != nil {
		t.Fatalf("criar sala: %v", err)
	}
	// Quem cria ja' entra: *sala vazia com o autor de fora seria um artefacto, nao uma sala.*
	if len(sala.Members) != 1 || sala.Members[0] != "ana" {
		t.Fatalf("o criador tem de entrar: %#v", sala.Members)
	}
	if _, err := store.JoinRoom("proj-1", "sala-produto", "agente-pesquisador"); err != nil {
		t.Fatalf("entrar na sala: %v", err)
	}
	membros := store.RoomMembers("proj-1", "sala-produto")
	if len(membros) != 2 {
		t.Fatalf("a sala tem de ter a pessoa E o agente, veio %#v", membros)
	}
	tipos := map[ParticipantKind]int{}
	for _, membro := range membros {
		tipos[membro.Kind]++
	}
	if tipos[ParticipantHuman] != 1 || tipos[ParticipantAgent] != 1 {
		t.Fatalf("humano e agente na MESMA sala: %#v", tipos)
	}
}

func TestSalaRecusaQuemNaoEhMembroDoProjeto(t *testing.T) {
	store := storeComMembros(t)
	if _, err := store.CreateRoom("proj-1", "sala-1", "Sala", "ana"); err != nil {
		t.Fatalf("criar sala: %v", err)
	}
	// *O caso que importa:* um id que nao esta' na lista de membros do projeto NAO entra - nem que a sala exista.
	_, err := store.JoinRoom("proj-1", "sala-1", "invasor")
	if !errors.Is(err, ErrRoomMemberNotInProject) {
		t.Fatalf("nao-membro tem de ser RECUSADO, veio: %v", err)
	}
	if membros := store.RoomMembers("proj-1", "sala-1"); len(membros) != 1 {
		t.Fatalf("a recusa nao pode gravar nada, veio %#v", membros)
	}
}

func TestSalaRecusaCriadorQueNaoEhMembro(t *testing.T) {
	store := storeComMembros(t)
	// Nem quem cria a sala fica fora da regra.
	if _, err := store.CreateRoom("proj-1", "sala-x", "X", "estranho"); !errors.Is(err, ErrRoomMemberNotInProject) {
		t.Fatalf("criador que nao e' membro tem de ser recusado, veio: %v", err)
	}
	if salas := store.Rooms("proj-1"); len(salas) != 0 {
		t.Fatalf("a recusa nao pode criar sala, veio %#v", salas)
	}
}

func TestSalaDuplicadaEhRecusada(t *testing.T) {
	store := storeComMembros(t)
	if _, err := store.CreateRoom("proj-1", "sala-1", "Sala", "ana"); err != nil {
		t.Fatalf("primeira: %v", err)
	}
	if _, err := store.CreateRoom("proj-1", "sala-1", "Outra", "ana"); !errors.Is(err, ErrRoomAlreadyExists) {
		t.Fatalf("sala duplicada tem de ser recusada, veio: %v", err)
	}
	if salas := store.Rooms("proj-1"); len(salas) != 1 || salas[0].Name != "Sala" {
		t.Fatalf("a sala original nao pode ser alterada pela tentativa: %#v", salas)
	}
}

func TestEntrarNaSalaInexistenteEhRecusado(t *testing.T) {
	store := storeComMembros(t)
	if _, err := store.JoinRoom("proj-1", "sala-que-nao-existe", "ana"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("sala inexistente tem de ser recusada, veio: %v", err)
	}
}

func TestEntrarDeNovoNaSalaNaoDuplica(t *testing.T) {
	store := storeComMembros(t)
	if _, err := store.CreateRoom("proj-1", "sala-1", "Sala", "ana"); err != nil {
		t.Fatalf("criar: %v", err)
	}
	for i := range 3 {
		if _, err := store.JoinRoom("proj-1", "sala-1", "agente-pesquisador"); err != nil {
			t.Fatalf("entrar (%d): %v", i, err)
		}
	}
	if membros := store.RoomMembers("proj-1", "sala-1"); len(membros) != 2 {
		t.Fatalf("entrar tres vezes tem de deixar DOIS membros, veio %d", len(membros))
	}
}

func TestSalaPersisteComMembros(t *testing.T) {
	root := t.TempDir()
	store, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "ana", ParticipantHuman, "Ana", "ana"); err != nil {
		t.Fatalf("membro: %v", err)
	}
	if _, err := store.JoinProject("proj-1", "agente-1", ParticipantAgent, "Agente", "ana"); err != nil {
		t.Fatalf("membro: %v", err)
	}
	if _, err := store.CreateRoom("proj-1", "sala-1", "Sala", "ana"); err != nil {
		t.Fatalf("criar: %v", err)
	}
	if _, err := store.JoinRoom("proj-1", "sala-1", "agente-1"); err != nil {
		t.Fatalf("entrar: %v", err)
	}
	reaberto, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	salas := reaberto.Rooms("proj-1")
	if len(salas) != 1 || salas[0].Name != "Sala" || len(salas[0].Members) != 2 {
		t.Fatalf("a sala e os membros tem de sobreviver a' releitura: %#v", salas)
	}
	if membros := reaberto.RoomMembers("proj-1", "sala-1"); len(membros) != 2 {
		t.Fatalf("os membros tem de resolver apos reabrir: %#v", membros)
	}
}
