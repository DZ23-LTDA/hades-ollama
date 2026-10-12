package agent

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TelAgentSessionState e a maquina de estados do handoff humano do Tel-Agent.
// SIP/PBX fisico continua fora (BLOCKED_BY_EXTERNAL_DEPENDENCY); isto e o
// contrato textual de sessao/handoff, testavel localmente.
type TelAgentSessionState string

const (
	TelSessionActiveAI     TelAgentSessionState = "ACTIVE_AI"
	TelSessionWaitingHuman TelAgentSessionState = "WAITING_HUMAN"
	TelSessionHumanActive  TelAgentSessionState = "HUMAN_ACTIVE"
	TelSessionResumedAI    TelAgentSessionState = "RESUMED_AI"
	TelSessionClosed       TelAgentSessionState = "CLOSED"
)

// TelAgentSession representa uma conversa persistente por tenant.
type TelAgentSession struct {
	ID             string               `json:"id"`
	OrganizationID string               `json:"organization_id"`
	ActorID        string               `json:"actor_id"` // identidade do cliente/caller
	Channel        string               `json:"channel"`
	State          TelAgentSessionState `json:"state"`
	AssignedAgent  string               `json:"assigned_agent,omitempty"`
	HumanOwner     string               `json:"human_owner,omitempty"`
	ContextRefs    []string             `json:"context_refs,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	LastActivityAt time.Time            `json:"last_activity_at"`
}

var (
	ErrTelSessionNotFound      = errors.New("tel-agent session not found")
	ErrTelSessionClosed        = errors.New("tel-agent session is closed")
	ErrTelSessionInvalidState  = errors.New("tel-agent session state transition not allowed")
	ErrTelSessionHumanMissing  = errors.New("tel-agent handoff requires a human owner")
	ErrTelSessionAlreadyHuman  = errors.New("tel-agent session is already owned by a human")
	ErrTelSessionActorRequired = errors.New("tel-agent session requires an actor id")
)

const maxTelAgentSessions = 200

// StartTelAgentSession cria uma sessao ACTIVE_AI para um tenant.
func (s *CompanyStore) StartTelAgentSession(id, organizationID, actorID, channel string) (Company, TelAgentSession, error) {
	organizationID = strings.TrimSpace(organizationID)
	actorID = strings.TrimSpace(RedactDLP(actorID))
	if actorID == "" {
		return Company{}, TelAgentSession{}, ErrTelSessionActorRequired
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "tel-agent.text"
	}
	now := time.Now().UTC()
	session := TelAgentSession{
		ID:             "tas_" + uuid.NewString(),
		OrganizationID: organizationID,
		ActorID:        actorID,
		Channel:        channel,
		State:          TelSessionActiveAI,
		AssignedAgent:  "ai",
		CreatedAt:      now,
		UpdatedAt:      now,
		LastActivityAt: now,
	}
	updated, err := s.mutate(id, func(company *Company) error {
		if strings.TrimSpace(company.OrganizationID) != organizationID {
			return ErrTelAgentOrganizationMismatch
		}
		company.TelAgentSessions = append(company.TelAgentSessions, session)
		if len(company.TelAgentSessions) > maxTelAgentSessions {
			company.TelAgentSessions = company.TelAgentSessions[len(company.TelAgentSessions)-maxTelAgentSessions:]
		}
		return nil
	})
	if err != nil {
		return Company{}, TelAgentSession{}, err
	}
	return updated, session, nil
}

// mutateTelSession aplica fn na sessao (org-scoped), atualizando timestamps.
func (s *CompanyStore) mutateTelSession(id, organizationID, sessionID string, fn func(*TelAgentSession) error) (Company, TelAgentSession, error) {
	organizationID = strings.TrimSpace(organizationID)
	sessionID = strings.TrimSpace(sessionID)
	var out TelAgentSession
	updated, err := s.mutate(id, func(company *Company) error {
		if strings.TrimSpace(company.OrganizationID) != organizationID {
			return ErrTelAgentOrganizationMismatch
		}
		idx := -1
		for i := range company.TelAgentSessions {
			if company.TelAgentSessions[i].ID == sessionID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return ErrTelSessionNotFound
		}
		session := company.TelAgentSessions[idx]
		if session.State == TelSessionClosed {
			return ErrTelSessionClosed
		}
		if err := fn(&session); err != nil {
			return err
		}
		now := time.Now().UTC()
		session.UpdatedAt = now
		session.LastActivityAt = now
		company.TelAgentSessions[idx] = session
		out = session
		return nil
	})
	if err != nil {
		return Company{}, TelAgentSession{}, err
	}
	return updated, out, nil
}

// RequestTelAgentHandoff: ACTIVE_AI/RESUMED_AI -> WAITING_HUMAN.
func (s *CompanyStore) RequestTelAgentHandoff(id, organizationID, sessionID string) (Company, TelAgentSession, error) {
	return s.mutateTelSession(id, organizationID, sessionID, func(session *TelAgentSession) error {
		if session.State != TelSessionActiveAI && session.State != TelSessionResumedAI {
			return ErrTelSessionInvalidState
		}
		session.State = TelSessionWaitingHuman
		return nil
	})
}

// TelAgentHumanTakeOver: WAITING_HUMAN -> HUMAN_ACTIVE (humanOwner obrigatorio).
func (s *CompanyStore) TelAgentHumanTakeOver(id, organizationID, sessionID, humanOwner string) (Company, TelAgentSession, error) {
	humanOwner = strings.TrimSpace(RedactDLP(humanOwner))
	if humanOwner == "" {
		return Company{}, TelAgentSession{}, ErrTelSessionHumanMissing
	}
	return s.mutateTelSession(id, organizationID, sessionID, func(session *TelAgentSession) error {
		if session.State == TelSessionHumanActive {
			return ErrTelSessionAlreadyHuman
		}
		if session.State != TelSessionWaitingHuman {
			return ErrTelSessionInvalidState
		}
		session.State = TelSessionHumanActive
		session.HumanOwner = humanOwner
		session.AssignedAgent = "human"
		return nil
	})
}

// TelAgentHumanReturn: HUMAN_ACTIVE -> RESUMED_AI (devolve para a IA).
func (s *CompanyStore) TelAgentHumanReturn(id, organizationID, sessionID string) (Company, TelAgentSession, error) {
	return s.mutateTelSession(id, organizationID, sessionID, func(session *TelAgentSession) error {
		if session.State != TelSessionHumanActive {
			return ErrTelSessionInvalidState
		}
		session.State = TelSessionResumedAI
		session.HumanOwner = ""
		session.AssignedAgent = "ai"
		return nil
	})
}

// CloseTelAgentSession: qualquer estado nao-fechado -> CLOSED.
func (s *CompanyStore) CloseTelAgentSession(id, organizationID, sessionID string) (Company, TelAgentSession, error) {
	return s.mutateTelSession(id, organizationID, sessionID, func(session *TelAgentSession) error {
		session.State = TelSessionClosed
		return nil
	})
}

// TelAgentSessionForOrganization le uma sessao (org-scoped, sem vazamento).
func (s *CompanyStore) TelAgentSessionForOrganization(id, organizationID, sessionID string) (TelAgentSession, error) {
	organizationID = strings.TrimSpace(organizationID)
	sessionID = strings.TrimSpace(sessionID)
	company, err := s.Get(id)
	if err != nil {
		// NÃO INFERÊNCIA (E-192, fechamento de B-16): mesma correção do B-15 — a sessão de outro tenant e a sessão de
		// empresa inexistente precisam devolver a MESMA recusa.
		return TelAgentSession{}, ErrTelAgentOrganizationMismatch
	}
	if strings.TrimSpace(company.OrganizationID) != organizationID {
		return TelAgentSession{}, ErrTelAgentOrganizationMismatch
	}
	for _, session := range company.TelAgentSessions {
		if session.ID == sessionID {
			return session, nil
		}
	}
	return TelAgentSession{}, ErrTelSessionNotFound
}
