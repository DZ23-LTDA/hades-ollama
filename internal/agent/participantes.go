package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Complemento 48 / FASE 2 e FASE 6 - MEMBROS: humanos e AGENTES na MESMA lista.
//
// O QUE FOI MEDIDO ANTES DE ESCREVER (2026-10-15, commit 93f12ac19): nao existia membro nenhum; a presenca era
// chaveada por `ProjectID+UserID`, o que **excluia o agente por construcao** - *um agente nao tem "user id" de
// pessoa*. Sem uma lista unica, "o time" e' so' a soma das pessoas, e a FASE 2 ("humanos e multiplos agentes na
// mesma sala") nao tem onde existir.
//
// DECISAO DE DESENHO: o tipo e' **explicito** (`human`/`agent`) e nao um booleano ou um campo livre.
// *Quem for decidir orcamento, permissao ou aprovacao precisa distinguir os dois SEM comparar strings.*
//
// NOTA DE ESTILO (aprendida com o lint, nao com a opiniao): os identificadores desta API sao em INGLES, como os
// vizinhos (`Comment`, `Presence`, `AddComment`, `SetPresence`). *A primeira versao usou nomes noutra lingua e o
// `misspell` acusou 14 ocorrencias - o dicionario do linter e' en-US, e a consistencia com a API existente ganha
// dos dois lados.* **Os comentarios continuam em pt-BR.**
//
// NOTA DE CODIFICACAO (esta custou um CI vermelho, e fica registada): editar este ficheiro com
// `Set-Content -Encoding UTF8` do PowerShell 5.1 **duplica a codificacao** - o `Get-Content -Raw` le o UTF-8 como
// cp1252 e a gravacao transforma o texto acentuado num monstro. **O arbitro e' o checker do repositorio**
// (`python scripts/hades_encoding_check_test.py`), que roda no CI: *editar texto por shell neste host exige
// verificar a codificacao depois.*
//
// LIMITE DECLARADO: entrar na lista **nao autentica** ninguem - mesmo limite da presenca (item 4). A `Origin`
// diz QUEM declarou a entrada; quem autentica e' a sessao/credencial, que nao vive neste arquivo.

// ParticipantKind separa quem e' gente de quem e' programa.
type ParticipantKind string

const (
	// ParticipantHuman: uma pessoa.
	ParticipantHuman ParticipantKind = "humano"
	// ParticipantAgent: um agente/executor.
	ParticipantAgent ParticipantKind = "agente"
)

// ErrUnknownParticipantKind: fora de `human`/`agent`. *Um tipo aberto faria "o time" virar qualquer coisa que o
// autor quisesse chamar de membro.*
var ErrUnknownParticipantKind = errors.New("unknown participant kind")

// ErrParticipantWithoutOrigin: entrada sem quem a declarou.
var ErrParticipantWithoutOrigin = errors.New("participant requires an origin")

// Participant e' um membro da sala - pessoa ou agente.
type Participant struct {
	ProjectID string          `json:"project_id"`
	ID        string          `json:"id"`
	Kind      ParticipantKind `json:"tipo"`
	Name      string          `json:"nome,omitempty"`
	// Origin e' quem DECLAROU a entrada (nao necessariamente o proprio membro).
	Origin  string    `json:"origem"`
	Entered time.Time `json:"entrou_em"`
}

// JoinProject poe um humano OU um agente na lista de membros do projeto.
//
// Idempotente por `ID`: entrar de novo ATUALIZA o tipo/nome/origem em vez de duplicar - *uma lista com a mesma
// pessoa tres vezes nao e' um time, e' um bug de contagem.*
func (s *CollaborationStore) JoinProject(projectID, participantID string, kind ParticipantKind, name, origin string) (Participant, error) {
	projectID = strings.TrimSpace(projectID)
	participantID = strings.TrimSpace(participantID)
	name = strings.TrimSpace(name)
	origin = strings.TrimSpace(origin)
	if projectID == "" || participantID == "" {
		return Participant{}, errors.New("project and participant are required")
	}
	if kind != ParticipantHuman && kind != ParticipantAgent {
		return Participant{}, fmt.Errorf("%w: %q", ErrUnknownParticipantKind, string(kind))
	}
	if origin == "" {
		return Participant{}, ErrParticipantWithoutOrigin
	}
	joined := Participant{
		ProjectID: projectID, ID: participantID, Kind: kind, Name: name,
		Origin: origin, Entered: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]Participant(nil), s.participants[projectID]...)
	replaced := false
	list := make([]Participant, 0, len(previous)+1)
	for _, existing := range previous {
		if existing.ID == participantID {
			list = append(list, joined)
			replaced = true
			continue
		}
		list = append(list, existing)
	}
	if !replaced {
		list = append(list, joined)
	}
	s.participants[projectID] = list
	if err := s.persistLocked(); err != nil {
		if len(previous) == 0 {
			delete(s.participants, projectID)
		} else {
			s.participants[projectID] = previous
		}
		return Participant{}, err
	}
	return joined, nil
}

// Participants devolve a lista ordenada por ID - **humanos e agentes juntos**, sem separacao por tipo.
// *Quem quiser separar, filtra; quem quiser "o time", usa isto.*
func (s *CollaborationStore) Participants(projectID string) []Participant {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := append([]Participant(nil), s.participants[projectID]...)
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list
}

// ProjectAgents devolve APENAS os agentes. Existe porque a mencao `@agente` precisa de resolver contra
// **agentes de verdade**, e nao contra pessoas cujo nome por acaso coincide.
func (s *CollaborationStore) ProjectAgents(projectID string) []Participant {
	agents := make([]Participant, 0)
	for _, member := range s.Participants(projectID) {
		if member.Kind == ParticipantAgent {
			agents = append(agents, member)
		}
	}
	return agents
}
