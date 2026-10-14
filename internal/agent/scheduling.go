package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AGENDAMENTO UNIVERSAL — NÚCLEO ANTI-DOUBLE-BOOKING (V2.2, incremento I / E-313).
//
// O V2.2 exige um Scheduling Engine multiempresa/multiunidade/multiprofissional com:
//
//	"transações e locks/constraints contra double booking sob concorrência; idempotência em chat, WhatsApp,
//	 telefone e web; disponibilidade consultada em fonte da verdade, sem inventar horários".
//
// MEDIDO antes de implementar: **não existia nenhum motor de agendamento** — os arquivos `schedule*` do repositório
// são o scheduler de **workflow** (outro conceito). Esta é a PRIMEIRA fatia: o núcleo que impede dois agendamentos no
// mesmo recurso, sob concorrência real, com recusa explícita, idempotência por canal e persistência por tenant.
//
// O QUE ESTA FATIA NÃO FAZ (declarado, não fingido): turnos/folgas/férias/feriados/fusos, lista de espera, lembretes,
// no-show, sincronização Google/Microsoft, pacotes de nicho e interface. Ficam para as próximas fatias, com o mesmo
// critério: teste antes de promessa.
//
// A DISPONIBILIDADE VEM DA FONTE DA VERDADE: `Availability` é derivada dos agendamentos PERSISTIDOS da organização —
// não existe horário inventado nem agenda paralela em memória.

var (
	// ErrSchedulingSlotTaken é a recusa EXPLÍCITA quando o recurso já está ocupado naquele intervalo.
	ErrSchedulingSlotTaken = errors.New("scheduling slot already taken")
	// ErrSchedulingInvalid recusa pedido malformado (sem serviço, sem recurso, início zero, duração inválida).
	ErrSchedulingInvalid = errors.New("scheduling request is invalid")
	// ErrSchedulingDuplicate é a recusa explícita do replay idempotente (mesma chave de canal).
	ErrSchedulingDuplicate = errors.New("scheduling idempotent replay")
	// ErrSchedulingNotFound recusa operação sobre agendamento inexistente.
	ErrSchedulingNotFound = errors.New("scheduling appointment not found")
)

// SchedulingService é o serviço agendável: duração própria e intervalo de preparo entre atendimentos.
type SchedulingService struct {
	ID              string `json:"id"`
	OrganizationID  string `json:"organization_id"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
	// BufferMinutes é o preparo/limpeza APÓS o atendimento: o recurso só volta a ficar livre depois dele. Sem buffer,
	// dois atendimentos colados viram atraso em cascata — e o sistema diria que havia vaga onde não havia.
	BufferMinutes int `json:"buffer_minutes,omitempty"`
}

// SchedulingResource é quem executa: profissional, sala, equipamento ou veículo.
type SchedulingResource struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

// SchedulingAppointment é o agendamento confirmado.
type SchedulingAppointment struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ResourceID     string    `json:"resource_id"`
	ServiceID      string    `json:"service_id"`
	ContactID      string    `json:"contact_id,omitempty"`
	Channel        string    `json:"channel,omitempty"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	// BusyUntil inclui o buffer: é o intervalo REAL em que o recurso não pode receber outro atendimento.
	BusyUntil      time.Time `json:"busy_until"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

const (
	// SchedulingAppointmentBooked é o estado de um agendamento ativo.
	SchedulingAppointmentBooked = "booked"
	// SchedulingAppointmentCancelled é o estado de um agendamento cancelado (o horário volta a ficar livre).
	SchedulingAppointmentCancelled = "cancelled"
	// SchedulingAppointmentNoShow (E-329) registra FALTA do cliente.
	SchedulingAppointmentNoShow = "no_show"
)

// SchedulingStore guarda serviços, recursos e agendamentos por organização, com escrita atômica.
type SchedulingStore struct {
	mu     sync.Mutex
	root   string
	byOrg  map[string]*schedulingOrg
	loaded map[string]bool
}

type schedulingOrg struct {
	Services     []SchedulingService     `json:"services"`
	Resources    []SchedulingResource    `json:"resources"`
	Appointments []SchedulingAppointment `json:"appointments"`
	// Shifts, Closures e TimeZone entram na FATIA 2 (E-314): sem eles a disponibilidade oferecia horário fora do
	// expediente, o que o V2.2 chama de "inventar horários".
	Shifts   []SchedulingShift   `json:"shifts,omitempty"`
	Closures []SchedulingClosure `json:"closures,omitempty"`
	// Breaks (E-328): intervalos dentro do turno (almoço, descanso, manutenção).
	Breaks []SchedulingBreak `json:"breaks,omitempty"`
	// Waitlist e Reminders (E-329): fila de espera FIFO com oferta que expira, e trilha de lembretes ja
	// enviados (para o MESMO lembrete nunca sair duas vezes).
	Waitlist  []SchedulingWaitlistEntry `json:"waitlist,omitempty"`
	Reminders []SchedulingReminder      `json:"reminders,omitempty"`
	TimeZone  string                    `json:"time_zone,omitempty"`
}

// NewSchedulingStore cria o store no diretório informado.
func NewSchedulingStore(root string) *SchedulingStore {
	return &SchedulingStore{root: strings.TrimSpace(root), byOrg: map[string]*schedulingOrg{}, loaded: map[string]bool{}}
}

// Root devolve o diretório do store.
func (s *SchedulingStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *SchedulingStore) path(organizationID string) string {
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// loadLocked carrega a organização UMA vez. Falha de leitura é ERRO (não se inventa agenda vazia em cima de erro).
func (s *SchedulingStore) loadLocked(organizationID string) error {
	if s.loaded[organizationID] {
		return nil
	}
	s.loaded[organizationID] = true
	if s.root == "" {
		s.byOrg[organizationID] = &schedulingOrg{}
		return nil
	}
	dados, err := os.ReadFile(s.path(organizationID))
	if err != nil {
		if os.IsNotExist(err) {
			s.byOrg[organizationID] = &schedulingOrg{}
			return nil
		}
		return err
	}
	org := &schedulingOrg{}
	if len(dados) > 0 {
		if err := json.Unmarshal(dados, org); err != nil {
			return err
		}
	}
	s.byOrg[organizationID] = org
	return nil
}

// saveLocked grava a organização com escrita atômica.
func (s *SchedulingStore) saveLocked(organizationID string) error {
	if s.root == "" {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	org := s.byOrg[organizationID]
	if org == nil {
		org = &schedulingOrg{}
	}
	return writeJSONAtomic(s.path(organizationID), org)
}

// AddService registra um serviço agendável da organização.
func (s *SchedulingStore) AddService(service SchedulingService) (SchedulingService, error) {
	if s == nil {
		return SchedulingService{}, ErrSchedulingInvalid
	}
	// E-337: valida a organização CRUA antes de consentScope (que tem fallback para "local").
	if err := requireOrganization(service.OrganizationID); err != nil {
		return SchedulingService{}, fmt.Errorf("%w: %v", ErrSchedulingInvalid, err)
	}
	service.OrganizationID = consentScope(service.OrganizationID)
	service.Name = strings.TrimSpace(service.Name)
	if service.OrganizationID == "" || service.Name == "" || service.DurationMinutes <= 0 || service.BufferMinutes < 0 {
		return SchedulingService{}, fmt.Errorf("%w: organização, nome, duração > 0 e buffer >= 0 são obrigatórios", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(service.OrganizationID); err != nil {
		return SchedulingService{}, err
	}
	if service.ID == "" {
		service.ID = "svc_" + uuid.NewString()
	}
	org := s.byOrg[service.OrganizationID]
	org.Services = append(org.Services, service)
	if err := s.saveLocked(service.OrganizationID); err != nil {
		org.Services = org.Services[:len(org.Services)-1]
		return SchedulingService{}, err
	}
	return service, nil
}

// AddResource registra quem executa (profissional, sala, equipamento).
func (s *SchedulingStore) AddResource(resource SchedulingResource) (SchedulingResource, error) {
	if s == nil {
		return SchedulingResource{}, ErrSchedulingInvalid
	}
	if err := requireOrganization(resource.OrganizationID); err != nil {
		return SchedulingResource{}, fmt.Errorf("%w: %v", ErrSchedulingInvalid, err)
	}
	resource.OrganizationID = consentScope(resource.OrganizationID)
	resource.Name = strings.TrimSpace(resource.Name)
	if resource.OrganizationID == "" || resource.Name == "" {
		return SchedulingResource{}, fmt.Errorf("%w: organização e nome do recurso são obrigatórios", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(resource.OrganizationID); err != nil {
		return SchedulingResource{}, err
	}
	if resource.ID == "" {
		resource.ID = "res_" + uuid.NewString()
	}
	org := s.byOrg[resource.OrganizationID]
	org.Resources = append(org.Resources, resource)
	if err := s.saveLocked(resource.OrganizationID); err != nil {
		org.Resources = org.Resources[:len(org.Resources)-1]
		return SchedulingResource{}, err
	}
	return resource, nil
}

// Services devolve os serviços da organização.
func (s *SchedulingStore) Services(organizationID string) ([]SchedulingService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]SchedulingService(nil), s.byOrg[organizationID].Services...), nil
}

// Appointments devolve os agendamentos da organização (todos os estados).
func (s *SchedulingStore) Appointments(organizationID string) ([]SchedulingAppointment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]SchedulingAppointment(nil), s.byOrg[organizationID].Appointments...), nil
}

// BookRequest é o pedido de agendamento, venha ele de chat, WhatsApp, telefone ou web.
type BookRequest struct {
	OrganizationID string
	ResourceID     string
	ServiceID      string
	ContactID      string
	Channel        string
	StartsAt       time.Time
	// IdempotencyKey identifica a MESMA intenção vinda do mesmo canal: reentrega não cria segundo agendamento.
	IdempotencyKey string
}

// Book cria o agendamento recusando sobreposição NO MESMO RECURSO.
//
// O ponto crítico é a CONCORRÊNCIA: dois agentes (voz e WhatsApp, por exemplo) disputando o último horário devem
// resultar em **um** agendamento e **uma recusa explícita** — nunca em dois. Por isso a checagem e a gravação
// acontecem sob o MESMO lock, e a disponibilidade sai dos agendamentos persistidos (fonte da verdade).
func (s *SchedulingStore) bookLocked(request BookRequest) (SchedulingAppointment, error) {
	if s == nil {
		return SchedulingAppointment{}, ErrSchedulingInvalid
	}
	request.OrganizationID = consentScope(request.OrganizationID)
	if request.OrganizationID == "" || strings.TrimSpace(request.ResourceID) == "" ||
		strings.TrimSpace(request.ServiceID) == "" || request.StartsAt.IsZero() {
		return SchedulingAppointment{}, fmt.Errorf("%w: organização, recurso, serviço e início são obrigatórios", ErrSchedulingInvalid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(request.OrganizationID); err != nil {
		return SchedulingAppointment{}, err
	}
	org := s.byOrg[request.OrganizationID]

	// Idempotência ANTES de qualquer efeito: reentrega do mesmo canal não cria segundo agendamento.
	if chave := strings.TrimSpace(request.IdempotencyKey); chave != "" {
		for _, existente := range org.Appointments {
			if existente.IdempotencyKey == chave && existente.Status == SchedulingAppointmentBooked {
				return SchedulingAppointment{}, fmt.Errorf("%w: %s", ErrSchedulingDuplicate, existente.ID)
			}
		}
	}

	servico, achou := schedulingService(org, request.ServiceID)
	if !achou {
		return SchedulingAppointment{}, fmt.Errorf("%w: serviço %s não existe nesta organização", ErrSchedulingInvalid, request.ServiceID)
	}
	if !schedulingResourceExists(org, request.ResourceID) {
		return SchedulingAppointment{}, fmt.Errorf("%w: recurso %s não existe nesta organização", ErrSchedulingInvalid, request.ResourceID)
	}

	// PORTÃO DE EXPEDIENTE (E-314): fora do turno ou em dia fechado o agendamento é RECUSADO antes de qualquer efeito.
	// Validar aqui (e não só na consulta) é o que impede que um cliente seja confirmado às 03:00 por um caminho que
	// não passou pela disponibilidade.
	if err := s.appointmentAllowedLocked(request.OrganizationID, strings.TrimSpace(request.ResourceID), request.StartsAt, servico.DurationMinutes); err != nil {
		return SchedulingAppointment{}, err
	}

	inicio := request.StartsAt.UTC()
	fim := inicio.Add(time.Duration(servico.DurationMinutes) * time.Minute)
	ocupadoAte := fim.Add(time.Duration(servico.BufferMinutes) * time.Minute)

	// SOBREPOSIÇÃO: o recurso está ocupado de `StartsAt` até `BusyUntil`, com o buffer incluído.
	for _, existente := range org.Appointments {
		if existente.Status != SchedulingAppointmentBooked || existente.ResourceID != request.ResourceID {
			continue
		}
		if inicio.Before(existente.BusyUntil) && existente.StartsAt.Before(ocupadoAte) {
			return SchedulingAppointment{}, fmt.Errorf("%w: recurso %s ocupado até %s",
				ErrSchedulingSlotTaken, request.ResourceID, existente.BusyUntil.Format(time.RFC3339))
		}
	}

	agendamento := SchedulingAppointment{
		ID:             "appt_" + uuid.NewString(),
		OrganizationID: request.OrganizationID,
		ResourceID:     strings.TrimSpace(request.ResourceID),
		ServiceID:      servico.ID,
		ContactID:      strings.TrimSpace(request.ContactID),
		Channel:        strings.ToLower(strings.TrimSpace(request.Channel)),
		StartsAt:       inicio,
		EndsAt:         fim,
		BusyUntil:      ocupadoAte,
		IdempotencyKey: strings.TrimSpace(request.IdempotencyKey),
		Status:         SchedulingAppointmentBooked,
		CreatedAt:      time.Now().UTC(),
	}
	org.Appointments = append(org.Appointments, agendamento)
	if err := s.saveLocked(request.OrganizationID); err != nil {
		org.Appointments = org.Appointments[:len(org.Appointments)-1]
		return SchedulingAppointment{}, err
	}
	return agendamento, nil
}

// Cancel libera o horário (o agendamento fica no histórico, cancelado).
func (s *SchedulingStore) cancelLocked(organizationID, appointmentID, reason string) (SchedulingAppointment, error) {
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
			return SchedulingAppointment{}, fmt.Errorf("%w: %s já está %s", ErrSchedulingInvalid, appointmentID, org.Appointments[i].Status)
		}
		org.Appointments[i].Status = SchedulingAppointmentCancelled
		if err := s.saveLocked(organizationID); err != nil {
			org.Appointments[i].Status = SchedulingAppointmentBooked
			return SchedulingAppointment{}, err
		}
		return org.Appointments[i], nil
	}
	return SchedulingAppointment{}, fmt.Errorf("%w: %s", ErrSchedulingNotFound, appointmentID)
}

// AvailabilitySlot é um horário livre derivado dos agendamentos persistidos (fonte da verdade).
type AvailabilitySlot struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Free     bool      `json:"free"`
}

// Availability devolve os slots de uma JANELA de relógio, marcando o que está livre e o que está ocupado.
//
// ⚠️ LIMITE DECLARADO (E-413 / ADV-60, 6ª auditoria independente): **esta função NÃO consulta expediente.**
// Ela decide **apenas por agendamentos** — não olha turno, fechamento por data, intervalo nem fuso. *Um slot
// marcado `Free` aqui pode estar às 03:00 de um domingo, com a empresa fechada.*
//
// 🚨 E a versão anterior deste comentário dizia o contrário: *"A resposta NÃO inventa horários … É esta função
// que os agentes de voz/WhatsApp/Telegram/recepção devem consultar antes de oferecer horário"* ✗ — **promessa
// que o corpo não cumpria**, e a **E-314** chegou a registrar que o buraco tinha sido fechado. Ele foi fechado
// em **`OpenSlots`** (`scheduling_hours.go:224`), função **nova**; esta ficou intacta.
//
// ⇒ **PARA OFERECER HORÁRIO AO CLIENTE, USE `OpenSlots`** (*que aplica turno, fechamento, intervalo, fuso e
// antecedência*). Esta função serve para **cruzar** uma janela com a agenda já marcada.
// ***A função não mudou de comportamento — mudou de promessa.***
//
// O contrato real é o do teste que a exercita (`scheduling_test.go:120`): *"a disponibilidade precisa derivar
// dos agendamentos persistidos"*.
func (s *SchedulingStore) Availability(organizationID, resourceID string, from, to time.Time, step time.Duration) ([]AvailabilitySlot, error) {
	if s == nil || step <= 0 || !to.After(from) {
		return nil, fmt.Errorf("%w: janela e passo precisam ser válidos", ErrSchedulingInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	org := s.byOrg[organizationID]
	var slots []AvailabilitySlot
	for inicio := from.UTC(); inicio.Before(to.UTC()); inicio = inicio.Add(step) {
		livre := true
		for _, agendamento := range org.Appointments {
			if agendamento.Status != SchedulingAppointmentBooked || agendamento.ResourceID != resourceID {
				continue
			}
			if agendamento.StartsAt.Equal(inicio) {
				livre = false
				break
			}
			if inicio.After(agendamento.StartsAt) && inicio.Before(agendamento.BusyUntil) {
				livre = false
				break
			}
		}
		slots = append(slots, AvailabilitySlot{StartsAt: inicio, EndsAt: inicio.Add(step), Free: livre})
	}
	return slots, nil
}

func schedulingService(org *schedulingOrg, id string) (SchedulingService, bool) {
	for _, servico := range org.Services {
		if servico.ID == id {
			return servico, true
		}
	}
	return SchedulingService{}, false
}

func schedulingResourceExists(org *schedulingOrg, id string) bool {
	for _, recurso := range org.Resources {
		if recurso.ID == id {
			return true
		}
	}
	return false
}

// SchedulingResources devolve os recursos da organização em ordem estável.
func (s *SchedulingStore) SchedulingResources(organizationID string) ([]SchedulingResource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	recursos := append([]SchedulingResource(nil), s.byOrg[organizationID].Resources...)
	sort.Slice(recursos, func(i, j int) bool { return recursos[i].ID < recursos[j].ID })
	return recursos, nil
}
