package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Complemento 48 / FASE 2 â€” A SALA (room).
//
// O QUE FOI MEDIDO ANTES DE ESCREVER (2026-10-15, commits 93f12ac19 e 0a0e4f5f3): nao existia conceito de sala
// em NENHUMA das tres camadas (runtime, backend, UI) - zero ocorrencias de `room`/`sala` como conversa; o unico
// `rooms` era escopo do conector LiveKit, declarado `operator_setup_required` (nao configurado). A partir de
// `0a0e4f5f3` ja' existiam MEMBROS (humanos e agentes numa lista unica), mas **sem lugar onde se reunirem**.
//
// A REGRA QUE DA' VALOR A ISTO (fail-closed): **so' entra na sala quem ja' e' membro do projeto.** *Uma sala que
// aceita qualquer id vira um canal paralelo de permissao* - e o complemento proibe criar um segundo motor de
// permissao (regra â‘ ). Por isso `JoinRoom` **recusa** um id que nao esteja em `Participants`.
//
// LIMITES DECLARADOS: â‘  a sala **nao** tem visibilidade publica/privada nem papeis (nao invento vocabulario que
// ninguem pediu); â‘¡ **nao** ha canal de mensagens ligado a ela ainda - o chat existente (`inbox`, `/api/agent/v1/chat`)
// continua a ser o unico caminho de mensagem, *e a sala aponta para ele, nao o substitui*; â‘¢ **nenhuma UI**.

// Room e' um lugar de trabalho com membros (pessoas e agentes).
type Room struct {
	ProjectID string    `json:"project_id"`
	ID        string    `json:"id"`
	Name      string    `json:"nome,omitempty"`
	CreatedBy string    `json:"criada_por"`
	CreatedAt time.Time `json:"criada_em"`
	// Members sao ids de PARTICIPANTS do projeto (a ordem e' de entrada; a leitura devolve resolvidos).
	Members []string `json:"membros"`
}

// ErrRoomAlreadyExists: a sala ja' existe. *Sala duplicada nao e' idempotencia: e' duas listas a divergir.*
var ErrRoomAlreadyExists = errors.New("room already exists")

// ErrRoomNotFound: sala inexistente.
var ErrRoomNotFound = errors.New("room not found")

// ErrRoomMemberNotInProject: o id nao e' membro do projeto. *E' esta recusa que impede a sala de virar uma via
// paralela de permissao.*
var ErrRoomMemberNotInProject = errors.New("room member is not a participant of this project")

// CreateRoom cria a sala. O criador tem de ser um membro do projeto - *nem quem cria a sala fica fora da regra.*
func (s *CollaborationStore) CreateRoom(projectID, roomID, name, createdBy string) (Room, error) {
	projectID = strings.TrimSpace(projectID)
	roomID = strings.TrimSpace(roomID)
	name = strings.TrimSpace(name)
	createdBy = strings.TrimSpace(createdBy)
	if projectID == "" || roomID == "" {
		return Room{}, errors.New("project and room are required")
	}
	if createdBy == "" {
		return Room{}, errors.New("room requires who created it")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roomExistsLocked(projectID, roomID) {
		return Room{}, fmt.Errorf("%w: %s", ErrRoomAlreadyExists, roomID)
	}
	if !s.participantxistsLocked(projectID, createdBy) {
		return Room{}, fmt.Errorf("%w: %s", ErrRoomMemberNotInProject, createdBy)
	}
	room := Room{
		ProjectID: projectID, ID: roomID, Name: name,
		CreatedBy: createdBy, CreatedAt: time.Now().UTC(),
		// Quem cria ja' entra: *sala vazia com o autor de fora seria um artefacto, nao uma sala.*
		Members: []string{createdBy},
	}
	previous := append([]Room(nil), s.rooms[projectID]...)
	s.rooms[projectID] = append(s.rooms[projectID], room)
	if err := s.persistLocked(); err != nil {
		if len(previous) == 0 {
			delete(s.rooms, projectID)
		} else {
			s.rooms[projectID] = previous
		}
		return Room{}, err
	}
	return room, nil
}

// JoinRoom poe um MEMBRO DO PROJETO na sala. Idempotente: entrar de novo nao duplica.
func (s *CollaborationStore) JoinRoom(projectID, roomID, memberID string) (Room, error) {
	projectID = strings.TrimSpace(projectID)
	roomID = strings.TrimSpace(roomID)
	memberID = strings.TrimSpace(memberID)
	if projectID == "" || roomID == "" || memberID == "" {
		return Room{}, errors.New("project, room and member are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	indice := s.roomIndexLocked(projectID, roomID)
	if indice < 0 {
		return Room{}, fmt.Errorf("%w: %s", ErrRoomNotFound, roomID)
	}
	// A REGRA: membro do PROJETO primeiro. Nao ha caminho que contorne esta checagem.
	if !s.participantxistsLocked(projectID, memberID) {
		return Room{}, fmt.Errorf("%w: %s", ErrRoomMemberNotInProject, memberID)
	}
	atual := s.rooms[projectID][indice]
	for _, existente := range atual.Members {
		if existente == memberID {
			return atual, nil // ja' esta': nao duplica e nao grava
		}
	}
	anterior := append([]Room(nil), s.rooms[projectID]...)
	atualizado := atual
	atualizado.Members = append(append([]string(nil), atual.Members...), memberID)
	s.rooms[projectID][indice] = atualizado
	if err := s.persistLocked(); err != nil {
		s.rooms[projectID] = anterior
		return Room{}, err
	}
	return atualizado, nil
}

// Rooms devolve as salas do projeto, ordenadas por id.
func (s *CollaborationStore) Rooms(projectID string) []Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	salas := append([]Room(nil), s.rooms[projectID]...)
	sort.Slice(salas, func(i, j int) bool { return salas[i].ID < salas[j].ID })
	return salas
}

// RoomMembers devolve os MEMBROS da sala ja' resolvidos contra a lista de participants - pessoas e agentes
// juntos. *Quem quiser so' os agentes filtra por `Kind`; a sala nao esconde ninguem.*
func (s *CollaborationStore) RoomMembers(projectID, roomID string) []Participant {
	projectID = strings.TrimSpace(projectID)
	roomID = strings.TrimSpace(roomID)
	s.mu.Lock()
	indice := s.roomIndexLocked(projectID, roomID)
	if indice < 0 {
		s.mu.Unlock()
		return nil
	}
	ids := append([]string(nil), s.rooms[projectID][indice].Members...)
	s.mu.Unlock()
	membros := make([]Participant, 0, len(ids))
	for _, id := range ids {
		for _, participant := range s.Participants(projectID) {
			if participant.ID == id {
				membros = append(membros, participant)
				break
			}
		}
	}
	return membros
}

func (s *CollaborationStore) roomExistsLocked(projectID, roomID string) bool {
	return s.roomIndexLocked(projectID, roomID) >= 0
}

func (s *CollaborationStore) roomIndexLocked(projectID, roomID string) int {
	for i, sala := range s.rooms[projectID] {
		if sala.ID == roomID {
			return i
		}
	}
	return -1
}

func (s *CollaborationStore) participantxistsLocked(projectID, participantID string) bool {
	for _, participant := range s.participants[projectID] {
		if participant.ID == participantID {
			return true
		}
	}
	return false
}
