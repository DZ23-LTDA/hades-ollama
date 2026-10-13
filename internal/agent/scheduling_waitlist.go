package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AGENDAMENTO — FATIA 6: LISTA DE ESPERA, LEMBRETES E NO-SHOW (V2.2 incremento I / E-329).
//
// O V2.2 exige, no mesmo item: *"lista de espera, confirmação, cancelamento e remarcação"*. As fatias anteriores
// fecharam criação, cancelamento e remarcação. Esta fecha o ciclo com as três peças que faltavam — e cada uma tem uma
// armadilha própria:
//
//  1. **LISTA DE ESPERA EM FIFO, MAS SEM AGENDAR SOZINHO.** Quando um horário é liberado, o motor **oferece** ao
//     primeiro da fila (ordem de chegada, que é justa e explicável) — e **NÃO** cria o agendamento. O V2.2 proíbe
//     contato sem consentimento; ***confirmar por conta própria o horário de quem nem respondeu é agir em nome do
//     cliente.*** O teste confere que **nenhum** agendamento nasceu da oferta.
//  2. **OFERTA EXPIRA.** Sem prazo, a fila ficaria travada por alguém que nunca responde. A oferta tem validade; ao
//     expirar, passa ao **próximo** — e o expirado **não** volta para a fila (*quem perdeu o turno não fura a fila de
//     novo por acidente*).
//  3. **CONFIRMAR RECONFERE A DISPONIBILIDADE.** Entre a oferta e a confirmação o mundo muda: outra pessoa pode ter
//     agendado pelo WhatsApp. A confirmação passa pelo **mesmo portão** de criação (expediente, intervalo,
//     sobreposição) e pode ser **recusada**.
//  4. **LEMBRETE NÃO SAI DUAS VEZES.** Um lembrete repetido é spam, e spam queima o canal do cliente.
//  5. **NO-SHOW É REGISTRADO, NÃO INVENTADO.** Marcar falta é decisão do operador; o motor só registra com trilha.

var (
	// ErrWaitlistInvalid recusa entrada malformada na fila.
	ErrWaitlistInvalid = errors.New("waitlist entry is invalid")
	// ErrWaitlistNotFound recusa operação sobre entrada inexistente.
	ErrWaitlistNotFound = errors.New("waitlist entry not found")
	// ErrWaitlistOfferExpired recusa confirmação de oferta vencida.
	ErrWaitlistOfferExpired = errors.New("waitlist offer has expired")
	// ErrWaitlistNotOffered recusa confirmação de entrada que não está em oferta.
	ErrWaitlistNotOffered = errors.New("waitlist entry is not in offer state")
)

// Estados da fila de espera.
const (
	WaitlistWaiting   = "waiting"
	WaitlistOffered   = "offered"
	WaitlistConfirmed = "confirmed"
	WaitlistExpired   = "expired"
)

// WaitlistOfferWindow é o prazo padrão de uma oferta. Declarado como constante para ser auditável.
const WaitlistOfferWindow = 30 * time.Minute

// SchedulingWaitlistEntry é uma pessoa esperando um horário numa janela.
type SchedulingWaitlistEntry struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ResourceID     string     `json:"resource_id"`
	ServiceID      string     `json:"service_id"`
	ContactID      string     `json:"contact_id,omitempty"`
	Channel        string     `json:"channel,omitempty"`
	From           time.Time  `json:"from"`
	To             time.Time  `json:"to"`
	CreatedAt      time.Time  `json:"created_at"`
	Status         string     `json:"status"`
	OfferedSlot    *time.Time `json:"offered_slot,omitempty"`
	OfferExpiresAt *time.Time `json:"offer_expires_at,omitempty"`
	AppointmentID  string     `json:"appointment_id,omitempty"`
}

// SchedulingReminder registra que um lembrete FOI enviado (para nunca sair duas vezes).
type SchedulingReminder struct {
	OrganizationID string    `json:"organization_id"`
	AppointmentID  string    `json:"appointment_id"`
	Kind           string    `json:"kind"`
	SentAt         time.Time `json:"sent_at"`
}

// AddToWaitlist coloca alguém na fila. A janela precisa fazer sentido e o recurso/serviço precisam existir.
func (s *SchedulingStore) AddToWaitlist(entrada SchedulingWaitlistEntry) (SchedulingWaitlistEntry, error) {
	if s == nil {
		return SchedulingWaitlistEntry{}, ErrWaitlistInvalid
	}
	entrada.OrganizationID = consentScope(entrada.OrganizationID)
	entrada.ResourceID = strings.TrimSpace(entrada.ResourceID)
	entrada.ServiceID = strings.TrimSpace(entrada.ServiceID)
	if entrada.OrganizationID == "" || entrada.ResourceID == "" || entrada.ServiceID == "" {
		return SchedulingWaitlistEntry{}, fmt.Errorf("%w: organização, recurso e serviço são obrigatórios", ErrWaitlistInvalid)
	}
	if entrada.From.IsZero() || entrada.To.IsZero() || !entrada.To.After(entrada.From) {
		return SchedulingWaitlistEntry{}, fmt.Errorf("%w: a janela desejada precisa ter início e fim válidos", ErrWaitlistInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(entrada.OrganizationID); err != nil {
		return SchedulingWaitlistEntry{}, err
	}
	org := s.byOrg[entrada.OrganizationID]
	if _, achou := schedulingService(org, entrada.ServiceID); !achou {
		return SchedulingWaitlistEntry{}, fmt.Errorf("%w: serviço %s não existe nesta organização", ErrWaitlistInvalid, entrada.ServiceID)
	}
	if !schedulingResourceExists(org, entrada.ResourceID) {
		return SchedulingWaitlistEntry{}, fmt.Errorf("%w: recurso %s não existe nesta organização", ErrWaitlistInvalid, entrada.ResourceID)
	}
	if entrada.ID == "" {
		entrada.ID = "wl_" + uuid.NewString()
	}
	if entrada.CreatedAt.IsZero() {
		entrada.CreatedAt = time.Now().UTC()
	}
	entrada.Status = WaitlistWaiting
	org.Waitlist = append(org.Waitlist, entrada)
	if err := s.saveLocked(entrada.OrganizationID); err != nil {
		org.Waitlist = org.Waitlist[:len(org.Waitlist)-1]
		return SchedulingWaitlistEntry{}, err
	}
	return entrada, nil
}

// Waitlist devolve a fila da organização em ordem de chegada (FIFO).
func (s *SchedulingStore) Waitlist(organizationID string) ([]SchedulingWaitlistEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	fila := append([]SchedulingWaitlistEntry(nil), s.byOrg[organizationID].Waitlist...)
	sort.SliceStable(fila, func(i, j int) bool { return fila[i].CreatedAt.Before(fila[j].CreatedAt) })
	return fila, nil
}

// OfferNextFromWaitlist OFERECE o horário liberado ao primeiro da fila cuja janela o contenha.
//
// **Não cria agendamento.** Oferecer é pedir confirmação; confirmar sozinho seria agir em nome do cliente.
func (s *SchedulingStore) OfferNextFromWaitlist(organizationID, resourceID string, slot time.Time, now time.Time) (SchedulingWaitlistEntry, bool, error) {
	if s == nil {
		return SchedulingWaitlistEntry{}, false, ErrWaitlistInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return SchedulingWaitlistEntry{}, false, err
	}
	org := s.byOrg[organizationID]
	// Expira ofertas vencidas ANTES de escolher: quem não respondeu não pode travar a fila.
	expirou := false
	for i := range org.Waitlist {
		entrada := &org.Waitlist[i]
		if entrada.Status == WaitlistOffered && entrada.OfferExpiresAt != nil && !entrada.OfferExpiresAt.After(now) {
			entrada.Status = WaitlistExpired
			expirou = true
		}
	}
	indice := -1
	for i := range org.Waitlist {
		entrada := org.Waitlist[i]
		if entrada.Status != WaitlistWaiting || entrada.ResourceID != strings.TrimSpace(resourceID) {
			continue
		}
		// A janela desejada precisa CONTER o horário oferecido, em instante absoluto.
		if slot.Before(entrada.From) || !slot.Before(entrada.To) {
			continue
		}
		if indice < 0 || entrada.CreatedAt.Before(org.Waitlist[indice].CreatedAt) {
			indice = i
		}
	}
	if indice < 0 {
		if expirou {
			if err := s.saveLocked(organizationID); err != nil {
				return SchedulingWaitlistEntry{}, false, err
			}
		}
		return SchedulingWaitlistEntry{}, false, nil
	}
	expiracao := now.Add(WaitlistOfferWindow).UTC()
	copia := slot.UTC()
	org.Waitlist[indice].Status = WaitlistOffered
	org.Waitlist[indice].OfferedSlot = &copia
	org.Waitlist[indice].OfferExpiresAt = &expiracao
	if err := s.saveLocked(organizationID); err != nil {
		org.Waitlist[indice].Status = WaitlistWaiting
		org.Waitlist[indice].OfferedSlot = nil
		org.Waitlist[indice].OfferExpiresAt = nil
		return SchedulingWaitlistEntry{}, false, err
	}
	return org.Waitlist[indice], true, nil
}

// ConfirmWaitlistOffer confirma a oferta e CRIA o agendamento — passando pelo mesmo portão da criação normal.
//
// É idempotente por entrada: confirmar duas vezes não cria dois agendamentos.
func (s *SchedulingStore) ConfirmWaitlistOffer(organizationID, entryID string, now time.Time) (SchedulingAppointment, error) {
	if s == nil {
		return SchedulingAppointment{}, ErrWaitlistInvalid
	}
	organizationID = consentScope(organizationID)

	// 1. Lê a entrada sob a trava e valida o estado (oferta existente e não vencida).
	s.mu.Lock()
	if err := s.loadLocked(organizationID); err != nil {
		s.mu.Unlock()
		return SchedulingAppointment{}, err
	}
	org := s.byOrg[organizationID]
	indice := -1
	for i := range org.Waitlist {
		if org.Waitlist[i].ID == entryID {
			indice = i
			break
		}
	}
	if indice < 0 {
		s.mu.Unlock()
		return SchedulingAppointment{}, fmt.Errorf("%w: %s", ErrWaitlistNotFound, entryID)
	}
	entrada := org.Waitlist[indice]
	if entrada.Status == WaitlistConfirmed && entrada.AppointmentID != "" {
		// Idempotência: já confirmado devolve o MESMO agendamento.
		var existente SchedulingAppointment
		for _, agendamento := range org.Appointments {
			if agendamento.ID == entrada.AppointmentID {
				existente = agendamento
				break
			}
		}
		s.mu.Unlock()
		if existente.ID == "" {
			return SchedulingAppointment{}, fmt.Errorf("%w: agendamento da entrada %s não encontrado", ErrWaitlistInvalid, entryID)
		}
		return existente, nil
	}
	if entrada.Status != WaitlistOffered || entrada.OfferedSlot == nil {
		s.mu.Unlock()
		return SchedulingAppointment{}, fmt.Errorf("%w: %s está %s", ErrWaitlistNotOffered, entryID, entrada.Status)
	}
	if entrada.OfferExpiresAt != nil && !entrada.OfferExpiresAt.After(now) {
		org.Waitlist[indice].Status = WaitlistExpired
		_ = s.saveLocked(organizationID)
		s.mu.Unlock()
		return SchedulingAppointment{}, fmt.Errorf("%w: %s venceu em %s", ErrWaitlistOfferExpired, entryID,
			entrada.OfferExpiresAt.Format(time.RFC3339))
	}
	slot := *entrada.OfferedSlot
	servicoID := entrada.ServiceID
	recursoID := entrada.ResourceID
	contato := entrada.ContactID
	canal := entrada.Channel
	s.mu.Unlock()

	// 2. Cria pelo MESMO caminho do Book (expediente, intervalo, sobreposição) — o mundo pode ter mudado desde a
	//    oferta, e é por isso que a confirmação pode ser recusada.
	agendamento, err := s.Book(BookRequest{
		OrganizationID: organizationID,
		ResourceID:     recursoID,
		ServiceID:      servicoID,
		ContactID:      contato,
		Channel:        canal,
		StartsAt:       slot,
		IdempotencyKey: "waitlist:" + entryID,
	})
	if err != nil {
		return SchedulingAppointment{}, err
	}

	// 3. Marca a entrada como confirmada.
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(organizationID); err != nil {
		return SchedulingAppointment{}, err
	}
	for i := range s.byOrg[organizationID].Waitlist {
		if s.byOrg[organizationID].Waitlist[i].ID == entryID {
			s.byOrg[organizationID].Waitlist[i].Status = WaitlistConfirmed
			s.byOrg[organizationID].Waitlist[i].AppointmentID = agendamento.ID
			if err := s.saveLocked(organizationID); err != nil {
				s.byOrg[organizationID].Waitlist[i].Status = WaitlistOffered
				s.byOrg[organizationID].Waitlist[i].AppointmentID = ""
				return SchedulingAppointment{}, err
			}
			break
		}
	}
	return agendamento, nil
}

// DueReminders devolve os agendamentos cujo lembrete está na hora e AINDA NÃO FOI ENVIADO.
//
// `kind` identifica o tipo (ex.: "24h", "2h") — a checagem é por (agendamento, tipo), então o mesmo lembrete nunca sai
// duas vezes.
func (s *SchedulingStore) DueReminders(organizationID string, kind string, lead time.Duration, now time.Time) ([]SchedulingAppointment, error) {
	if s == nil || strings.TrimSpace(kind) == "" {
		return nil, fmt.Errorf("%w: tipo de lembrete é obrigatório", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	org := s.byOrg[organizationID]
	jaEnviados := map[string]bool{}
	for _, lembrete := range org.Reminders {
		if lembrete.Kind == kind {
			jaEnviados[lembrete.AppointmentID] = true
		}
	}
	var devidos []SchedulingAppointment
	for _, agendamento := range org.Appointments {
		if agendamento.Status != SchedulingAppointmentBooked || jaEnviados[agendamento.ID] {
			continue
		}
		// Devido quando o início já está dentro da antecedência e ainda não passou.
		if !agendamento.StartsAt.After(now) {
			continue
		}
		if agendamento.StartsAt.Sub(now) > lead {
			continue
		}
		devidos = append(devidos, agendamento)
	}
	sort.SliceStable(devidos, func(i, j int) bool { return devidos[i].StartsAt.Before(devidos[j].StartsAt) })
	return devidos, nil
}

// MarkReminderSent registra o envio. Registrar de novo o MESMO lembrete é recusado (não se marca duas vezes).
func (s *SchedulingStore) MarkReminderSent(organizationID, appointmentID, kind string, now time.Time) error {
	if s == nil {
		return ErrSchedulingInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return err
	}
	org := s.byOrg[organizationID]
	for _, lembrete := range org.Reminders {
		if lembrete.AppointmentID == appointmentID && lembrete.Kind == kind {
			return fmt.Errorf("%w: lembrete %q do agendamento %s já foi enviado", ErrSchedulingInvalid, kind, appointmentID)
		}
	}
	org.Reminders = append(org.Reminders, SchedulingReminder{OrganizationID: organizationID,
		AppointmentID: appointmentID, Kind: kind, SentAt: now.UTC()})
	if err := s.saveLocked(organizationID); err != nil {
		org.Reminders = org.Reminders[:len(org.Reminders)-1]
		return err
	}
	return nil
}

// MarkNoShow registra falta do cliente. Não remove o agendamento: o histórico fica.
func (s *SchedulingStore) MarkNoShow(organizationID, appointmentID string, now time.Time) (SchedulingAppointment, error) {
	if s == nil {
		return SchedulingAppointment{}, ErrSchedulingInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return SchedulingAppointment{}, err
	}
	org := s.byOrg[organizationID]
	for i := range org.Appointments {
		if org.Appointments[i].ID != appointmentID {
			continue
		}
		if org.Appointments[i].Status != SchedulingAppointmentBooked {
			return SchedulingAppointment{}, fmt.Errorf("%w: %s está %s", ErrSchedulingInvalid, appointmentID, org.Appointments[i].Status)
		}
		anterior := org.Appointments[i].Status
		org.Appointments[i].Status = SchedulingAppointmentNoShow
		if err := s.saveLocked(organizationID); err != nil {
			org.Appointments[i].Status = anterior
			return SchedulingAppointment{}, err
		}
		return org.Appointments[i], nil
	}
	return SchedulingAppointment{}, fmt.Errorf("%w: %s", ErrSchedulingNotFound, appointmentID)
}
