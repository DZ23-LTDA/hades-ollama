package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// AGENDAMENTO — FATIA 2 (V2.2): EXPEDIENTE, FECHAMENTO E FUSO (E-314).
//
// MEDIDO na fatia 1: `Availability` oferecia horário para **qualquer** instante da janela pedida — inclusive 03:00 de
// um domingo. O V2.2 exige *"disponibilidade consultada em fonte da verdade, SEM INVENTAR HORÁRIOS"*: horário fora do
// expediente não é disponível, é invenção — e um cliente confirmado às 03:00 chega numa porta fechada.
func TestAgendamentoRespeitaExpedienteFechamentoEFuso(t *testing.T) {
	// FIXTURE PRÓPRIA: este teste mede EXPEDIENTE, então não pode herdar o 24/7 do fixture de concorrência (medido:
	// herdar fazia o domingo aparecer aberto e o teste acusava o próprio fixture, não o código).
	store, org, recurso, servico := schedulingFixtureSemTurno(t)
	const fuso = "America/Sao_Paulo"
	if err := store.SetTimeZone(org, fuso); err != nil {
		t.Fatalf("configurar fuso: %v", err)
	}
	loc, err := time.LoadLocation(fuso)
	if err != nil {
		t.Fatal(err)
	}
	// Segunda-feira 09:00–12:00 no horário LOCAL da organização.
	segunda := time.Date(2026, 10, 19, 0, 0, 0, 0, loc) // 2026-10-19 é segunda-feira
	if segunda.Weekday() != time.Monday {
		t.Fatalf("fixture precisa de uma segunda-feira, veio %s", segunda.Weekday())
	}
	if _, err := store.AddShift(SchedulingShift{OrganizationID: org, Weekday: int(time.Monday), OpensAt: 9 * 60, ClosesAt: 12 * 60}); err != nil {
		t.Fatal(err)
	}

	// 1. Fora do turno: nenhum horário. Sem turno no domingo: nenhum horário.
	domingo := segunda.AddDate(0, 0, -1)
	slotsDomingo, err := store.OpenSlots(org, recurso, domingo.Format("2006-01-02"), servico, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(slotsDomingo) != 0 {
		t.Fatalf("domingo sem expediente não pode oferecer horário: %+v", slotsDomingo)
	}
	// 2. Dentro do turno: 09:00, 09:30, 10:00, 10:30, 11:00 e 11:30 (serviço de 30min precisa caber até 12:00).
	slots, err := store.OpenSlots(org, recurso, segunda.Format("2006-01-02"), servico, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 6 {
		t.Fatalf("segunda 09:00-12:00 com serviço de 30min precisa ter 6 horários, veio %d: %+v", len(slots), slots)
	}
	if slots[0].StartsAt.In(loc).Hour() != 9 || slots[len(slots)-1].StartsAt.In(loc).Hour() != 11 {
		t.Fatalf("os horários precisam ficar dentro do turno local: %+v", slots)
	}

	// 3. FECHAMENTO (feriado/folga) no dia: nenhum horário, e o agendamento é recusado pelo motivo certo.
	if _, err := store.AddClosure(SchedulingClosure{OrganizationID: org, Date: segunda.Format("2006-01-02"), Reason: "feriado municipal"}); err != nil {
		t.Fatal(err)
	}
	slotsFechado, err := store.OpenSlots(org, recurso, segunda.Format("2006-01-02"), servico, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(slotsFechado) != 0 {
		t.Fatalf("dia fechado não pode oferecer horário: %+v", slotsFechado)
	}
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 19, 10, 0, 0, 0, loc), Channel: "voice"}); !errors.Is(err, ErrSchedulingClosed) {
		t.Fatalf("agendar em dia fechado precisa ser recusado: %v", err)
	}

	// 4. FUSO: o mesmo "10:00" é 13:00Z em São Paulo. Um pedido às 03:00 LOCAIS é recusado por expediente — mesmo que
	//    em UTC ele pareça "06:00". É a hora LOCAL que decide, e é isto que o teste prova.
	outroDia := segunda.AddDate(0, 0, 7) // segunda seguinte, sem fechamento
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 26, 3, 0, 0, 0, loc), Channel: "voice"}); !errors.Is(err, ErrSchedulingOutsideHours) {
		t.Fatalf("03:00 locais precisam ser recusados por expediente: %v", err)
	}
	agendado, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 26, 10, 0, 0, 0, loc), Channel: "voice", IdempotencyKey: "fuso-1"})
	if err != nil {
		t.Fatalf("10:00 locais precisam ser aceitos: %v", err)
	}
	if agendado.StartsAt.UTC().Hour() != 13 {
		t.Fatalf("10:00 em São Paulo é 13:00Z: %s", agendado.StartsAt.UTC().Format(time.RFC3339))
	}

	// 5. A ocupação continua valendo dentro do turno (fatia 1 + fatia 2 juntas).
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 26, 10, 15, 0, 0, loc), Channel: "whatsapp"}); !errors.Is(err, ErrSchedulingSlotTaken) {
		t.Fatalf("sobreposição dentro do turno precisa continuar recusada: %v", err)
	}
	_ = outroDia
}

// schedulingFixtureSemTurno monta serviço e recurso SEM turno configurado — o estado inicial honesto de uma agenda
// que ainda não teve expediente cadastrado.
func schedulingFixtureSemTurno(t *testing.T) (*SchedulingStore, string, string, string) {
	t.Helper()
	store := NewSchedulingStore(filepath.Join(t.TempDir(), "scheduling"))
	const org = "org-a"
	servico, err := store.AddService(SchedulingService{OrganizationID: org, Name: "Corte", DurationMinutes: 30, BufferMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := store.AddResource(SchedulingResource{OrganizationID: org, Name: "Profissional Ana"})
	if err != nil {
		t.Fatal(err)
	}
	return store, org, recurso.ID, servico.ID
}
