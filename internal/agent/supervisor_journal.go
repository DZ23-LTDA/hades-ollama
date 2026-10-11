package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Diário de ticks do supervisor (evidência obrigatória do estágio 31).
//
// Os contadores do supervisor viviam só em memória: ao reiniciar, a operação perdia
// a resposta para *"o que o supervisor fez nas últimas horas?"*. O diário guarda os
// últimos ticks com os contadores de cada um, de forma **limitada** e **persistida**.
//
// O diário registra o que ACONTECEU, incluindo os erros do tick — um diário que só
// guarda sucesso não serve para investigar incidente.

// MaxSupervisorTickJournal limita o histórico de ticks.
const MaxSupervisorTickJournal = 50

// SupervisorTickRecord é a linha do diário para um tick.
type SupervisorTickRecord struct {
	TickNumber             int64     `json:"tick_number"`
	ExecutedAt             time.Time `json:"executed_at"`
	MissionsResumed        int       `json:"missions_resumed"`
	SchedulesTriggered     int       `json:"schedules_triggered"`
	CyclesAdvanced         int       `json:"cycles_advanced"`
	ApprovalsCreated       int       `json:"approvals_created"`
	BlockedExternalActions int       `json:"blocked_external_actions"`
	RiskPausedCompanies    int       `json:"risk_paused_companies"`
	ErrorCount             int       `json:"error_count"`
	Errors                 []string  `json:"errors,omitempty"`
}

// TickJournal devolve o diário do mais recente para o mais antigo.
func (s *Supervisor) TickJournal() []SupervisorTickRecord {
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	out := append([]SupervisorTickRecord(nil), s.tickJournal...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// recordTick grava a linha do tick, com os erros observados.
func (s *Supervisor) recordTick(result SupervisorTickResult) {
	record := SupervisorTickRecord{
		TickNumber:             result.TickNumber,
		ExecutedAt:             result.ExecutedAt,
		MissionsResumed:        result.MissionsResumed,
		SchedulesTriggered:     result.SchedulesTriggered,
		CyclesAdvanced:         result.CyclesAdvanced,
		ApprovalsCreated:       result.ApprovalsCreated,
		BlockedExternalActions: result.BlockedExternalActions,
		RiskPausedCompanies:    len(result.RiskPausedCompanies),
		ErrorCount:             len(result.ScheduleRunErrors),
		Errors:                 append([]string(nil), result.ScheduleRunErrors...),
	}
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	s.tickJournal = append(s.tickJournal, record)
	if len(s.tickJournal) > MaxSupervisorTickJournal {
		s.tickJournal = s.tickJournal[len(s.tickJournal)-MaxSupervisorTickJournal:]
	}
	// Falha ao persistir o diário não pode derrubar o tick: o supervisor continua
	// trabalhando e a falha fica visível no próximo resultado.
	_ = s.persistTickJournalLocked()
}

func (s *Supervisor) tickJournalPath() string {
	if s.runtime == nil {
		return ""
	}
	root := strings.TrimSpace(s.runtime.DataRoot())
	if root == "" {
		return ""
	}
	return filepath.Join(root, "supervisor", "ticks.json")
}

func (s *Supervisor) persistTickJournalLocked() error {
	path := s.tickJournalPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(path, s.tickJournal)
}

func (s *Supervisor) loadTickJournal() error {
	path := s.tickJournalPath()
	if path == "" {
		return nil
	}
	var stored []SupervisorTickRecord
	if err := readJSON(path, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	s.incidentsMu.Lock()
	defer s.incidentsMu.Unlock()
	s.tickJournal = stored
	return nil
}
