package agent

import (
	"fmt"
	"strings"
	"time"
)

// AGENDAMENTO — FATIA 3: REMARCAÇÃO ATÔMICA (V2.2 incremento I / E-319).
//
// O V2.2 exige, entre as operações de agenda: "criar/confirmar/cancelar/**reagendar** … por chat, WhatsApp, telefone e web".
//
// A PROPRIEDADE QUE IMPORTA (e que quase todo sistema erra):
//
//	uma remarcação que FALHA não pode destruir o agendamento original.
//
// O caminho ingênuo — cancelar e depois tentar agendar — deixa o cliente **sem horário nenhum** quando o novo horário
// está ocupado: o cliente perde a vaga antiga e não ganha a nova. ***Isso é pior do que recusar remarcar.***
//
// Aqui a operação é TUDO-OU-NADA sob o mesmo lock: valida expediente e sobreposição para o NOVO intervalo considerando
// os OUTROS agendamentos (o próprio é ignorado), e só então grava. Em qualquer recusa, o agendamento original fica
// exatamente como estava — e o teste prova isso em cada direção de falha.
//
// LIMITE DECLARADO: a remarcação não preserva "histórico de tentativas" (não há tabela de auditoria de remarcação
// ainda); a lista de espera e os lembretes pertencem às próximas fatias.

// Reschedule move um agendamento para outro horário, recusando quando o novo horário não serve.
//
// Em QUALQUER recusa o agendamento ORIGINAL permanece íntegro: horário, buffer e status não mudam.
func (s *SchedulingStore) Reschedule(organizationID, appointmentID string, newStart time.Time) (SchedulingAppointment, error) {
	if s == nil {
		return SchedulingAppointment{}, ErrSchedulingInvalid
	}
	if strings.TrimSpace(appointmentID) == "" || newStart.IsZero() {
		return SchedulingAppointment{}, fmt.Errorf("%w: agendamento e novo horário são obrigatórios", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return SchedulingAppointment{}, err
	}
	org := s.byOrg[organizationID]

	indice := -1
	for i := range org.Appointments {
		if org.Appointments[i].ID == appointmentID {
			indice = i
			break
		}
	}
	if indice < 0 {
		return SchedulingAppointment{}, fmt.Errorf("%w: %s", ErrSchedulingNotFound, appointmentID)
	}
	atual := org.Appointments[indice]
	if atual.Status != SchedulingAppointmentBooked {
		return SchedulingAppointment{}, fmt.Errorf("%w: %s está %s e não pode ser remarcado",
			ErrSchedulingInvalid, appointmentID, atual.Status)
	}
	servico, achou := schedulingService(org, atual.ServiceID)
	if !achou {
		return SchedulingAppointment{}, fmt.Errorf("%w: serviço %s não existe mais nesta organização",
			ErrSchedulingInvalid, atual.ServiceID)
	}

	// 1. Expediente/fechamento do NOVO horário (a regra da fatia 2 vale igual para remarcar).
	if err := s.appointmentAllowedLocked(organizationID, atual.ResourceID, newStart, servico.DurationMinutes); err != nil {
		return SchedulingAppointment{}, err
	}

	inicio := newStart.UTC()
	fim := inicio.Add(time.Duration(servico.DurationMinutes) * time.Minute)
	ocupadoAte := fim.Add(time.Duration(servico.BufferMinutes) * time.Minute)

	// 2. Sobreposição considerando os OUTROS agendamentos (o próprio é ignorado) — inclusive ele mesmo no horário novo
	//    é uma remarcação para o MESMO lugar, que é permitida (idempotência natural da operação).
	for i, outro := range org.Appointments {
		if i == indice || outro.Status != SchedulingAppointmentBooked || outro.ResourceID != atual.ResourceID {
			continue
		}
		if inicio.Before(outro.BusyUntil) && outro.StartsAt.Before(ocupadoAte) {
			return SchedulingAppointment{}, fmt.Errorf("%w: recurso %s ocupado até %s",
				ErrSchedulingSlotTaken, atual.ResourceID, outro.BusyUntil.Format(time.RFC3339))
		}
	}

	// 3. Só agora grava — tudo-ou-nada.
	anterior := org.Appointments[indice]
	org.Appointments[indice].StartsAt = inicio
	org.Appointments[indice].EndsAt = fim
	org.Appointments[indice].BusyUntil = ocupadoAte
	if err := s.saveLocked(organizationID); err != nil {
		org.Appointments[indice] = anterior // rollback em memória se a gravação falhar
		return SchedulingAppointment{}, err
	}
	return org.Appointments[indice], nil
}

// FreeSlotsForReschedule devolve os horários livres de um dia para um agendamento EXISTENTE, já considerando que o
// próprio agendamento pode sair do lugar (por isso ele é ignorado no cálculo). É esta lista que os agentes de voz e de
// chat devem oferecer ao cliente — nunca uma lista que inclua o horário que ele já ocupa como se fosse "novo".
func (s *SchedulingStore) FreeSlotsForReschedule(organizationID, appointmentID, date string, stepMinutes int) ([]AvailabilitySlot, error) {
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
	indice := -1
	for i := range org.Appointments {
		if org.Appointments[i].ID == appointmentID {
			indice = i
			break
		}
	}
	if indice < 0 {
		return nil, fmt.Errorf("%w: %s", ErrSchedulingNotFound, appointmentID)
	}
	atual := org.Appointments[indice]
	loc, _, err := s.schedulingLocationLocked(organizationID)
	if err != nil {
		return nil, err
	}
	if _, fechado := closedOn(org, atual.ResourceID, dia.Format("2006-01-02")); fechado {
		return nil, nil
	}
	shift, temTurno := shiftFor(org, atual.ResourceID, dia.Weekday())
	if !temTurno {
		return nil, nil
	}
	servico, _ := schedulingService(org, atual.ServiceID)
	passo := time.Duration(stepMinutes) * time.Minute
	var livres []AvailabilitySlot
	for minuto := shift.OpensAt; minuto+servico.DurationMinutes <= shift.ClosesAt; minuto += stepMinutes {
		inicio := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, minuto, 0, 0, loc)
		fim := inicio.Add(time.Duration(servico.DurationMinutes) * time.Minute)
		ocupadoAte := fim.Add(time.Duration(servico.BufferMinutes) * time.Minute)
		livre := true
		for i, outro := range org.Appointments {
			if i == indice || outro.Status != SchedulingAppointmentBooked || outro.ResourceID != atual.ResourceID {
				continue
			}
			if inicio.Before(outro.BusyUntil) && outro.StartsAt.Before(ocupadoAte) {
				livre = false
				break
			}
		}
		livres = append(livres, AvailabilitySlot{StartsAt: inicio, EndsAt: inicio.Add(passo), Free: livre})
	}
	return livres, nil
}
