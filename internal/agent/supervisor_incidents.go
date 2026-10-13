package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Registro de incidente e recuperação (atividade do estágio 31).
//
// O supervisor já contava falhas em contadores e as devolvia no tick — mas um
// contador **não é um incidente**: ele não tem estado, não tem responsável, não
// distingue "aberto" de "resolvido" e desaparece no reinício. Sem isso não há como
// responder as duas perguntas que o estágio exige: *o que quebrou e ainda não foi
// tratado?* e *o que foi recuperado, e quando?*
//
// DECISÕES DELIBERADAS:
//   - **Idempotente por (tipo, sujeito)**: a mesma falha repetida **incrementa as
//     tentativas** em vez de criar um incidente novo. Um registro que cresce com a
//     repetição vira ruído e esconde o incidente que importa;
//   - **Recuperação é OBSERVADA, não declarada**: quando o mesmo sujeito volta a ter
//     sucesso num tick posterior, o incidente é **resolvido automaticamente** com
//     data. Ninguém precisa "marcar como resolvido" para o registro dizer a verdade;
//   - **Reabre**: se o sujeito falhar de novo depois de resolvido, o incidente
//     reabre somando a tentativa — histórico que volta a ser problema não fica
//     escondido como "resolvido";
//   - **Limitado** a `MaxSupervisorIncidents`, mantendo os mais recentes;
//   - **Persistido** no diretório de dados: incidente que desaparece no reinício não
//     serve para operação.

const (
	// Tipos de incidente observados pelo supervisor.
	SupervisorIncidentScheduleFailure = "schedule_failure"
	SupervisorIncidentRecoveryFailure = "recovery_failure"
	SupervisorIncidentRiskPause       = "risk_pause"

	// Estados do incidente.
	SupervisorIncidentOpen         = "open"
	SupervisorIncidentAcknowledged = "acknowledged"
	SupervisorIncidentResolved     = "resolved"

	// MaxSupervisorIncidents limita o registro.
	MaxSupervisorIncidents = 50
)

var ErrSupervisorIncidentNotFound = errors.New("supervisor incident not found")

// SupervisorIncident é um problema observado, com estado e recuperação.
type SupervisorIncident struct {
	ID               string     `json:"id"`
	Kind             string     `json:"kind"`
	Subject          string     `json:"subject"`
	Detail           string     `json:"detail,omitempty"`
	State            string     `json:"state"`
	OpenedAt         time.Time  `json:"opened_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	RecoveryAttempts int        `json:"recovery_attempts"`
	AcknowledgedBy   string     `json:"acknowledged_by,omitempty"`
	Resolution       string     `json:"resolution,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

// incidentID é estável por (tipo, sujeito): é o que torna o registro idempotente.
func incidentID(kind, subject string) string {
	return "inc_" + kind + "_" + sanitizeIDComponent(subject)
}

func sanitizeIDComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "geral"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(value)
}

// Incidents devolve uma cópia do registro, do mais recente para o mais antigo.
func (s *Supervisor) Incidents() []SupervisorIncident {
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	out := append([]SupervisorIncident(nil), s.incidents...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// recordIncident abre ou atualiza o incidente de (tipo, sujeito).
func (s *Supervisor) recordIncident(kind, subject, detail string, now time.Time) error {
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	id := incidentID(kind, subject)
	for index := range s.incidents {
		incident := &s.incidents[index]
		if incident.ID != id {
			continue
		}
		incident.RecoveryAttempts++
		incident.Detail = detail
		incident.UpdatedAt = now
		// Falha nova depois de resolvido REABRE: histórico que volta a ser problema
		// não pode ficar escondido como resolvido.
		if incident.State == SupervisorIncidentResolved {
			incident.State = SupervisorIncidentOpen
			incident.ResolvedAt = nil
			incident.Resolution = ""
			incident.AcknowledgedBy = ""
		}
		return s.persistIncidentsLocked()
	}
	s.incidents = append(s.incidents, SupervisorIncident{
		ID:               id,
		Kind:             kind,
		Subject:          strings.TrimSpace(subject),
		Detail:           detail,
		State:            SupervisorIncidentOpen,
		OpenedAt:         now,
		UpdatedAt:        now,
		RecoveryAttempts: 1,
	})
	if len(s.incidents) > MaxSupervisorIncidents {
		s.incidents = s.incidents[len(s.incidents)-MaxSupervisorIncidents:]
	}
	return s.persistIncidentsLocked()
}

// resolveIncident fecha o incidente quando a RECUPERAÇÃO é observada. Devolve
// `false` quando não havia incidente aberto — sucesso normal não cria registro.
func (s *Supervisor) resolveIncident(kind, subject, resolution string, now time.Time) (bool, error) {
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	id := incidentID(kind, subject)
	for index := range s.incidents {
		incident := &s.incidents[index]
		if incident.ID != id || incident.State == SupervisorIncidentResolved {
			continue
		}
		resolvedAt := now
		incident.State = SupervisorIncidentResolved
		incident.Resolution = resolution
		incident.ResolvedAt = &resolvedAt
		incident.UpdatedAt = now
		return true, s.persistIncidentsLocked()
	}
	return false, nil
}

// AcknowledgeIncident marca ciência humana sem declarar resolução: quem assume o
// incidente não é quem prova que ele acabou.
func (s *Supervisor) AcknowledgeIncident(id, actorID string, now time.Time) (SupervisorIncident, error) {
	return s.setIncidentState(id, actorID, SupervisorIncidentAcknowledged, "", now)
}

// ResolveIncident fecha o incidente por decisão humana explícita.
func (s *Supervisor) ResolveIncident(id, actorID, resolution string, now time.Time) (SupervisorIncident, error) {
	return s.setIncidentState(id, actorID, SupervisorIncidentResolved, resolution, now)
}

func (s *Supervisor) setIncidentState(id, actorID, state, resolution string, now time.Time) (SupervisorIncident, error) {
	id = strings.TrimSpace(id)
	actorID = strings.TrimSpace(actorID)
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	for index := range s.incidents {
		incident := &s.incidents[index]
		if incident.ID != id {
			continue
		}
		if state == SupervisorIncidentAcknowledged && actorID == "" {
			return SupervisorIncident{}, errors.New("supervisor incident acknowledgement requires an identified actor")
		}
		incident.State = state
		incident.UpdatedAt = now
		if actorID != "" {
			incident.AcknowledgedBy = actorID
		}
		if state == SupervisorIncidentResolved {
			resolvedAt := now
			incident.ResolvedAt = &resolvedAt
			incident.Resolution = strings.TrimSpace(resolution)
		}
		if err := s.persistIncidentsLocked(); err != nil {
			return SupervisorIncident{}, err
		}
		return *incident, nil
	}
	return SupervisorIncident{}, ErrSupervisorIncidentNotFound
}

func (s *Supervisor) incidentsPath() string {
	if s.runtime == nil {
		return ""
	}
	root := strings.TrimSpace(s.runtime.DataRoot())
	if root == "" {
		return ""
	}
	return filepath.Join(root, "supervisor", "incidents.json")
}

func (s *Supervisor) persistIncidentsLocked() error {
	path := s.incidentsPath()
	if path == "" {
		// Sem diretório de dados (testes de unidade do supervisor): o registro
		// continua em memória, e isso é declarado como limite, não escondido.
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create supervisor incident dir: %w", err)
	}
	return writeJSONAtomic(path, s.incidents)
}

// loadIncidents traz o registro do disco. Arquivo ausente é estado normal na
// primeira execução; arquivo corrompido é erro e não pode ser ignorado.
func (s *Supervisor) loadIncidents() error {
	path := s.incidentsPath()
	if path == "" {
		return nil
	}
	var stored []SupervisorIncident
	if err := readJSON(path, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	s.incidents = stored
	return nil
}

// ErrTenantScopeRequired recusa operação sem organização informada (E-337).
var ErrTenantScopeRequired = errors.New("organization scope is required")

// requireOrganization valida a organização CRUA, ANTES de qualquer normalização.
//
// POR QUE ISTO EXISTE (E-337): `consentScope("")` devolve `LocalOrganizationID` — o que é legítimo para uso local de
// usuário único, mas em contexto multi-tenant transforma "organização ausente" em **escopo compartilhado**. As
// validações existentes (`org == ""`) aconteciam DEPOIS de `consentScope`, portanto nunca disparavam. ***Validar
// depois de normalizar é validar o valor que a normalização inventou.***
func requireOrganization(organizationID string) error {
	if strings.TrimSpace(organizationID) == "" {
		return ErrTenantScopeRequired
	}
	return nil
}
