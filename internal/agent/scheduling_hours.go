package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// AGENDAMENTO — FATIA 2: JANELAS DE FUNCIONAMENTO, FECHAMENTOS E FUSO (V2.2 incremento I / E-314).
//
// MEDIDO na fatia 1: `Availability` devolvia slot para **qualquer** horário da janela pedida — inclusive 03:00 da
// manhã de um domingo. O V2.2 exige *"disponibilidade consultada em FONTE DA VERDADE, SEM INVENTAR HORÁRIOS"*: um
// horário fora do expediente não é disponível, é invenção. Esta fatia fecha esse buraco:
//
//   - **turnos** por dia da semana, por recurso ou para todos (expediente real da empresa);
//   - **fechamentos** (feriado, folga, férias, manutenção) por data, por recurso ou gerais;
//   - **fuso da organização**: o expediente é escrito em hora LOCAL e convertido para o instante real — *o mesmo
//     "09:00" é 12:00Z em São Paulo e 09:00Z em Lisboa*;
//   - `Book` passa a RECUSAR fora do expediente e em dia fechado, com erro próprio em vez de aceitar em silêncio.
//
// LIMITE DECLARADO: feriados **nacionais** não vêm embutidos — não se inventa calendário de feriado por país. O
// operador (ou uma integração futura) registra as datas; o motor apenas as respeita.

var (
	// ErrSchedulingOutsideHours recusa agendamento fora do expediente configurado.
	ErrSchedulingOutsideHours = fmt.Errorf("%w: fora do expediente configurado", ErrSchedulingInvalid)
	// ErrSchedulingClosed recusa agendamento em data fechada (feriado, folga, férias, manutenção).
	ErrSchedulingClosed = fmt.Errorf("%w: data fechada", ErrSchedulingInvalid)
)

// SchedulingShift é a janela de funcionamento de um dia da semana.
//
// `ResourceID` vazio significa **vale para todos os recursos** (expediente da empresa). Com valor, vale só para aquele
// recurso — é assim que se representa o profissional que atende só de manhã sem inventar um segundo motor.
type SchedulingShift struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ResourceID     string `json:"resource_id,omitempty"`
	// Weekday segue time.Weekday: 0 = domingo … 6 = sábado, em hora LOCAL da organização.
	Weekday int `json:"weekday"`
	// OpensAt e ClosesAt são minutos desde a meia-noite local (ex.: 09:00 = 540).
	OpensAt  int `json:"opens_at_minute"`
	ClosesAt int `json:"closes_at_minute"`
}

// SchedulingClosure é o fechamento de uma DATA (feriado, folga, férias, manutenção).
type SchedulingClosure struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ResourceID     string `json:"resource_id,omitempty"`
	// Date é a data LOCAL no formato AAAA-MM-DD.
	Date   string `json:"date"`
	Reason string `json:"reason,omitempty"`
}

// SchedulingSettings guarda o fuso da organização (obrigatório para interpretar expediente e fechamento).
type SchedulingSettings struct {
	OrganizationID string `json:"organization_id"`
	TimeZone       string `json:"time_zone"`
}

const schedulingDefaultTimeZone = "America/Sao_Paulo"

// SetTimeZone define o fuso da organização. Fuso inválido é RECUSADO (não se assume UTC em silêncio).
func (s *SchedulingStore) SetTimeZone(organizationID, timeZone string) error {
	if s == nil {
		return ErrSchedulingInvalid
	}
	organizationID = consentScope(organizationID)
	timeZone = strings.TrimSpace(timeZone)
	if organizationID == "" || timeZone == "" {
		return fmt.Errorf("%w: organização e fuso são obrigatórios", ErrSchedulingInvalid)
	}
	if _, err := time.LoadLocation(timeZone); err != nil {
		return fmt.Errorf("%w: fuso desconhecido %q", ErrSchedulingInvalid, timeZone)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(organizationID); err != nil {
		return err
	}
	org := s.byOrg[organizationID]
	anterior := org.TimeZone
	org.TimeZone = timeZone
	if err := s.saveLocked(organizationID); err != nil {
		org.TimeZone = anterior
		return err
	}
	return nil
}

// TimeZone devolve o fuso configurado (ou o padrão declarado) da organização.
func (s *SchedulingStore) TimeZone(organizationID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return "", err
	}
	if tz := s.byOrg[organizationID].TimeZone; tz != "" {
		return tz, nil
	}
	return schedulingDefaultTimeZone, nil
}

// AddShift registra uma janela de funcionamento.
func (s *SchedulingStore) AddShift(shift SchedulingShift) (SchedulingShift, error) {
	if s == nil {
		return SchedulingShift{}, ErrSchedulingInvalid
	}
	shift.OrganizationID = consentScope(shift.OrganizationID)
	if shift.OrganizationID == "" || shift.Weekday < 0 || shift.Weekday > 6 {
		return SchedulingShift{}, fmt.Errorf("%w: organização e dia da semana (0..6) são obrigatórios", ErrSchedulingInvalid)
	}
	// E-328: `ClosesAt < OpensAt` significa turno que CRUZA a meia-noite (22:00->02:00). Igualdade continua
	// recusada: janela de duração zero não é janela.
	if shift.OpensAt < 0 || shift.ClosesAt > 24*60 || shift.ClosesAt == shift.OpensAt {
		return SchedulingShift{}, fmt.Errorf("%w: janela inválida (%d..%d)", ErrSchedulingInvalid, shift.OpensAt, shift.ClosesAt)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(shift.OrganizationID); err != nil {
		return SchedulingShift{}, err
	}
	if shift.ID == "" {
		shift.ID = "shift_" + sanitizeIDComponent(fmt.Sprintf("%d-%d-%d", shift.Weekday, shift.OpensAt, shift.ClosesAt))
	}
	org := s.byOrg[shift.OrganizationID]
	org.Shifts = append(org.Shifts, shift)
	if err := s.saveLocked(shift.OrganizationID); err != nil {
		org.Shifts = org.Shifts[:len(org.Shifts)-1]
		return SchedulingShift{}, err
	}
	return shift, nil
}

// AddClosure registra um fechamento por data.
func (s *SchedulingStore) AddClosure(closure SchedulingClosure) (SchedulingClosure, error) {
	if s == nil {
		return SchedulingClosure{}, ErrSchedulingInvalid
	}
	closure.OrganizationID = consentScope(closure.OrganizationID)
	closure.Date = strings.TrimSpace(closure.Date)
	if closure.OrganizationID == "" {
		return SchedulingClosure{}, fmt.Errorf("%w: organização é obrigatória", ErrSchedulingInvalid)
	}
	if _, err := time.Parse("2006-01-02", closure.Date); err != nil {
		return SchedulingClosure{}, fmt.Errorf("%w: data precisa ser AAAA-MM-DD", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(closure.OrganizationID); err != nil {
		return SchedulingClosure{}, err
	}
	if closure.ID == "" {
		closure.ID = "clo_" + sanitizeIDComponent(closure.Date+closure.ResourceID)
	}
	org := s.byOrg[closure.OrganizationID]
	org.Closures = append(org.Closures, closure)
	if err := s.saveLocked(closure.OrganizationID); err != nil {
		org.Closures = org.Closures[:len(org.Closures)-1]
		return SchedulingClosure{}, err
	}
	return closure, nil
}

// schedulingLocation devolve o fuso da organização (padrão declarado quando não configurado).
func (s *SchedulingStore) schedulingLocationLocked(organizationID string) (*time.Location, string, error) {
	tz := s.byOrg[organizationID].TimeZone
	if tz == "" {
		tz = schedulingDefaultTimeZone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, tz, fmt.Errorf("%w: fuso %q inválido", ErrSchedulingInvalid, tz)
	}
	return loc, tz, nil
}

// shiftFor devolve a janela aplicável naquele dia local (recurso específico tem precedência sobre o geral).
func shiftFor(org *schedulingOrg, resourceID string, weekday time.Weekday) (SchedulingShift, bool) {
	especifico, achouEspecifico := SchedulingShift{}, false
	geral, achouGeral := SchedulingShift{}, false
	for _, shift := range org.Shifts {
		if shift.Weekday != int(weekday) {
			continue
		}
		if shift.ResourceID == resourceID {
			if !achouEspecifico || shift.OpensAt < especifico.OpensAt {
				especifico, achouEspecifico = shift, true
			}
			continue
		}
		if shift.ResourceID == "" {
			if !achouGeral || shift.OpensAt < geral.OpensAt {
				geral, achouGeral = shift, true
			}
		}
	}
	if achouEspecifico {
		return especifico, true
	}
	return geral, achouGeral
}

// closedOn diz se a data local está fechada para o recurso (fechamento específico ou geral).
func closedOn(org *schedulingOrg, resourceID, data string) (SchedulingClosure, bool) {
	for _, closure := range org.Closures {
		if closure.Date != data {
			continue
		}
		if closure.ResourceID == resourceID || closure.ResourceID == "" {
			return closure, true
		}
	}
	return SchedulingClosure{}, false
}

// OpenSlots devolve os horários REALMENTE livres do recurso num dia local: dentro do turno, fora de fechamento e
// descontando o que já está ocupado (a ocupação vem da fatia 1, dos agendamentos persistidos).
//
// O passo é o "granulado" da agenda (ex.: 30min). Um horário que não caiba inteiro no turno **não** é oferecido.
func (s *SchedulingStore) OpenSlots(organizationID, resourceID, date, serviceID string, stepMinutes int) ([]AvailabilitySlot, error) {
	if s == nil || stepMinutes <= 0 {
		return nil, fmt.Errorf("%w: passo precisa ser positivo", ErrSchedulingInvalid)
	}
	dia, err := time.Parse("2006-01-02", strings.TrimSpace(date))
	if err != nil {
		return nil, fmt.Errorf("%w: data precisa ser AAAA-MM-DD", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	org := s.byOrg[organizationID]
	loc, _, err := s.schedulingLocationLocked(organizationID)
	if err != nil {
		return nil, err
	}
	if _, fechado := closedOn(org, resourceID, dia.Format("2006-01-02")); fechado {
		return nil, nil // dia fechado não tem horário: devolver nada é a resposta honesta
	}
	shift, temTurno := shiftFor(org, resourceID, dia.Weekday())
	if !temTurno {
		return nil, nil // sem expediente nesse dia, também não há horário
	}
	duracao := 0
	if servico, achou := schedulingService(org, serviceID); achou {
		duracao = servico.DurationMinutes
	}
	passo := time.Duration(stepMinutes) * time.Minute
	var livres []AvailabilitySlot
	// E-328: a janela pode CRUZAR a meia-noite, então o laço anda por DESLOCAMENTO a partir da abertura e o instante
	// real é calculado com o dia correto — senão um turno 22:00->02:00 ofereceria 01:00 do dia errado (no passado).
	duracaoJanela := shiftDurationMinutes(shift)
	for deslocamento := 0; deslocamento+duracao <= duracaoJanela; deslocamento += stepMinutes {
		inicio := shiftStartInstant(dia, shift, deslocamento, loc)
		// INTERVALO: almoço/descanso não é horário disponível; oferecer seria o mesmo que inventar expediente.
		if _, emIntervalo := breakCovers(org, resourceID, inicio.Weekday(), inicio.Hour()*60+inicio.Minute(), duracao); emIntervalo {
			continue
		}
		ocupado := false
		for _, agendamento := range org.Appointments {
			if agendamento.Status != SchedulingAppointmentBooked || agendamento.ResourceID != resourceID {
				continue
			}
			if inicio.Before(agendamento.BusyUntil) && agendamento.StartsAt.Before(inicio.Add(passo)) {
				ocupado = true
				break
			}
		}
		livres = append(livres, AvailabilitySlot{StartsAt: inicio, EndsAt: inicio.Add(passo), Free: !ocupado})
	}
	sort.Slice(livres, func(i, j int) bool { return livres[i].StartsAt.Before(livres[j].StartsAt) })
	return livres, nil
}

// appointmentAllowedLocked confere expediente/fechamento para o instante pedido — é o portão que `Book` usa.
func (s *SchedulingStore) appointmentAllowedLocked(organizationID, resourceID string, startsAt time.Time, durationMinutes int) error {
	org := s.byOrg[organizationID]
	loc, tz, err := s.schedulingLocationLocked(organizationID)
	if err != nil {
		return err
	}
	local := startsAt.In(loc)
	if closure, fechado := closedOn(org, resourceID, local.Format("2006-01-02")); fechado {
		motivo := strings.TrimSpace(closure.Reason)
		if motivo == "" {
			motivo = "fechado"
		}
		return fmt.Errorf("%w: %s (%s, %s)", ErrSchedulingClosed, motivo, local.Format("2006-01-02"), tz)
	}
	// E-328b: às 01:00 de terça, quem cobre pode ser o turno de SEGUNDA que cruza a meia-noite (22:00->02:00).
	// Procurar só o dia corrente recusaria a madrugada de um turno que existe — medido pelo teste.
	shift, temTurno := shiftCovering(org, resourceID, local)
	if !temTurno {
		return fmt.Errorf("%w: %s não tem expediente em %s (%s)", ErrSchedulingOutsideHours,
			resourceID, local.Format("2006-01-02"), tz)
	}
	inicioMinuto := local.Hour()*60 + local.Minute()
	// E-328: a checagem de janela considera turno que CRUZA a meia-noite.
	if !shiftCoversMinute(shift, inicioMinuto, durationMinutes) {
		return fmt.Errorf("%w: %s está fora da janela %02d:%02d-%02d:%02d (%s)",
			ErrSchedulingOutsideHours, local.Format(time.RFC3339), shift.OpensAt/60, shift.OpensAt%60,
			shift.ClosesAt/60, shift.ClosesAt%60, tz)
	}
	// INTERVALO: almoço/descanso recusa o agendamento com erro PRÓPRIO — a causa é outra, e o cliente merece saber.
	if br, emIntervalo := breakCovers(org, resourceID, local.Weekday(), inicioMinuto, durationMinutes); emIntervalo {
		rotulo := strings.TrimSpace(br.Label)
		if rotulo == "" {
			rotulo = "intervalo"
		}
		return fmt.Errorf("%w: %s cai em %s (%02d:%02d-%02d:%02d)",
			ErrSchedulingOnBreak, local.Format(time.RFC3339), rotulo, br.StartsAtMinute/60, br.StartsAtMinute%60,
			br.EndsAtMinute/60, br.EndsAtMinute%60)
	}
	return nil
}
