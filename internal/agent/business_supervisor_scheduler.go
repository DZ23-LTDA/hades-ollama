package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------------------------------------------
// AGENDADOR PRÓPRIO DO SUPERVISOR (E-274)
// ---------------------------------------------------------------------------------------------------------------
//
// O lease e o ciclo supervisionado já existiam; faltava QUEM CHAMA. Este agendador decide se o ciclo está vencido,
// chama o ciclo sob lease e registra o disparo. Ele não faz o trabalho: ele decide a HORA.

// SupervisorIntervaloPadrao é o intervalo entre ciclos quando o chamador não informa um.
const SupervisorIntervaloPadrao = 15 * time.Minute

// ErrSupervisorScheduleInvalid recusa intervalo inválido, organização ausente ou supervisor ausente.
var ErrSupervisorScheduleInvalid = errors.New("supervisor schedule invalid")

// supervisorSchedule é o arquivo que guarda o último disparo.
type supervisorSchedule struct {
	LastRun  time.Time `json:"last_run"`
	LastTick int64     `json:"last_tick"`
	LastNote string    `json:"last_note,omitempty"`
	Runs     int64     `json:"runs"`
}

// SupervisorScheduler dispara o ciclo supervisionado no intervalo configurado.
type SupervisorScheduler struct {
	supervisor     *BusinessSupervisor
	organizationID string
	interval       time.Duration
	ttl            time.Duration
	root           string
	lastRun        time.Time
	lastTick       int64
	lastNote       string
	runs           int64
}

// NewSupervisorScheduler cria o agendador. `root` vazio mantém o relógio em memória (declarado).
func NewSupervisorScheduler(root string, supervisor *BusinessSupervisor, organizationID string, interval time.Duration) (*SupervisorScheduler, error) {
	if supervisor == nil {
		return nil, fmt.Errorf("%w: sem supervisor não há ciclo para agendar", ErrSupervisorScheduleInvalid)
	}
	// VALIDAR A ENTRADA CRUA: consentScope("") devolve a organização PADRÃO. Aceitar isso aqui faria o agendador
	// disparar o ciclo no tenant ERRADO sem ninguém pedir - omissão viraria acesso a dado alheio.
	if strings.TrimSpace(organizationID) == "" {
		return nil, fmt.Errorf("%w: organização é obrigatória", ErrSupervisorScheduleInvalid)
	}
	organizationID = consentScope(organizationID)
	if interval <= 0 {
		if interval == 0 {
			interval = SupervisorIntervaloPadrao
		} else {
			return nil, fmt.Errorf("%w: intervalo negativo (%s)", ErrSupervisorScheduleInvalid, interval)
		}
	}
	agendador := &SupervisorScheduler{
		supervisor:     supervisor,
		organizationID: organizationID,
		interval:       interval,
		ttl:            SupervisorLeaseTTLPadrao,
		root:           strings.TrimSpace(root),
	}
	if err := agendador.load(); err != nil {
		return nil, err
	}
	return agendador, nil
}

func (s *SupervisorScheduler) path() string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, "supervisor.schedule.json")
}

func (s *SupervisorScheduler) load() error {
	caminho := s.path()
	if caminho == "" {
		return nil
	}
	stored := supervisorSchedule{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("ler agenda do supervisor: %w", err)
	}
	s.lastRun = stored.LastRun
	s.lastTick = stored.LastTick
	s.lastNote = stored.LastNote
	s.runs = stored.Runs
	return nil
}

func (s *SupervisorScheduler) persist() error {
	caminho := s.path()
	if caminho == "" {
		return nil
	}
	return writeJSONAtomic(caminho, supervisorSchedule{
		LastRun: s.lastRun, LastTick: s.lastTick, LastNote: s.lastNote, Runs: s.runs,
	})
}

// Due diz se o próximo ciclo está vencido. Nunca rodou => vencido.
func (s *SupervisorScheduler) Due(now time.Time) bool {
	if s.lastRun.IsZero() {
		return true
	}
	return !now.Before(s.lastRun.Add(s.interval))
}

// LastRun devolve o último disparo registrado (zero quando nunca rodou).
func (s *SupervisorScheduler) LastRun() time.Time { return s.lastRun }

// Runs conta os ciclos disparados por este agendador.
func (s *SupervisorScheduler) Runs() int64 { return s.runs }

// RunDue roda UM ciclo se estiver vencido. Devolve `false` quando não era a hora ou quando OUTRA instância tem o
// lease — nos dois casos o relógio NÃO avança.
func (s *SupervisorScheduler) RunDue(holder string, now time.Time) (BusinessTickResult, bool, error) {
	if !s.Due(now) {
		return BusinessTickResult{}, false, nil
	}
	result, err := s.supervisor.RunSupervisedTick(holder, s.organizationID, now, s.ttl)
	if errors.Is(err, ErrSupervisorLeaseHeld) {
		// O ciclo NÃO rodou: avançar o relógio aqui faria o intervalo ser PULADO em silêncio.
		return BusinessTickResult{}, false, nil
	}
	// CATCH-UP COLAPSADO: o relógio vai para AGORA, não para o instante perdido. Depois de uma queda longa, rodar um
	// tick por intervalo perdido seria uma tempestade de efeitos externos.
	s.lastRun = now
	s.runs++
	if err != nil {
		// Falha AVANÇA o relógio de propósito: um ciclo que falha sempre não pode virar laço apertado.
		s.lastNote = "falha no ciclo: " + err.Error()
		_ = s.persist()
		return result, true, err
	}
	s.lastTick = result.TickNumber
	s.lastNote = ""
	if persistErr := s.persist(); persistErr != nil {
		return result, true, persistErr
	}
	return result, true, nil
}
