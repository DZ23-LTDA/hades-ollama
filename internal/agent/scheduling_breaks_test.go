package agent

import (
	"errors"
	"testing"
	"time"
)

// AGENDAMENTO — FATIA 5: INTERVALOS E TURNO QUE CRUZA A MEIA-NOITE (E-328).
//
// Dois limites declarados nas fatias anteriores: *sem intervalos de almoço* e *sem turno que cruza a meia-noite*.
// ***Oferecer horário que a empresa não tem é a mesma classe de erro que inventar expediente.***

func TestIntervaloDentroDoTurnoNaoEhOferecidoNemAceito(t *testing.T) {
	store, org, recurso, servico := schedulingFixtureSemTurno(t)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	segunda := time.Date(2026, 10, 19, 0, 0, 0, 0, loc) // segunda-feira
	if segunda.Weekday() != time.Monday {
		t.Fatalf("fixture precisa de segunda-feira, veio %s", segunda.Weekday())
	}
	if _, err := store.AddShift(SchedulingShift{OrganizationID: org, Weekday: int(time.Monday), OpensAt: 9 * 60, ClosesAt: 18 * 60}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddBreak(SchedulingBreak{
		OrganizationID: org, Weekday: int(time.Monday),
		StartsAtMinute: 12 * 60, EndsAtMinute: 13 * 60, Label: "almoço",
	}); err != nil {
		t.Fatal(err)
	}

	slots, err := store.OpenSlots(org, recurso, segunda.Format("2006-01-02"), servico, 30)
	if err != nil {
		t.Fatal(err)
	}
	for _, slot := range slots {
		local := slot.StartsAt.In(loc)
		minuto := local.Hour()*60 + local.Minute()
		// 12:00 e 12:30 caem no intervalo; 11:30 (30min) invade o almoço e TAMBÉM não pode ser oferecido.
		if minuto == 12*60 || minuto == 12*60+30 {
			t.Fatalf("horário dentro do intervalo foi oferecido: %s", local.Format("15:04"))
		}
		// MEDIDO: eu afirmei que 11:30 invadiria o almoço, mas 11:30 + 30min termina EXATAMENTE às 12:00 — tocar a
		// borda não é invadir. A asserção estava errada, não o motor. A colisão parcial real é testada abaixo com o
		// pedido de 11:45 (que terminaria 12:15).
	}

	// Agendar dentro do intervalo é RECUSADO com erro PRÓPRIO (a causa é outra, e o cliente merece saber).
	_, err = store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 19, 12, 15, 0, 0, loc), Channel: "voice",
	})
	if !errors.Is(err, ErrSchedulingOnBreak) {
		t.Fatalf("agendar no intervalo precisa ser recusado por INTERVALO: %v", err)
	}
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 19, 11, 45, 0, 0, loc), Channel: "voice",
	}); !errors.Is(err, ErrSchedulingOnBreak) {
		t.Fatalf("colisão PARCIAL com o intervalo precisa ser recusada: %v", err)
	}
	// Fora do intervalo, o mesmo dia continua aceitando.
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 19, 14, 0, 0, 0, loc), Channel: "voice",
	}); err != nil {
		t.Fatalf("fora do intervalo precisa continuar aceitando: %v", err)
	}

	// Intervalo que CRUZA a meia-noite é recusado no registro (não se adivinha o dia).
	if _, err := store.AddBreak(SchedulingBreak{
		OrganizationID: org, Weekday: int(time.Monday),
		StartsAtMinute: 23 * 60, EndsAtMinute: 1 * 60,
	}); !errors.Is(err, ErrSchedulingInvalid) {
		t.Fatalf("intervalo que cruza a meia-noite precisa ser recusado: %v", err)
	}
}

func TestIntervaloDeUmRecursoNaoAfetaOutro(t *testing.T) {
	store, org, recursoA, servico := schedulingFixtureSemTurno(t)
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	recursoB, err := store.AddResource(SchedulingResource{OrganizationID: org, Name: "Profissional Bia"})
	if err != nil {
		t.Fatal(err)
	}
	for dia := range 7 {
		if _, err := store.AddShift(SchedulingShift{OrganizationID: org, Weekday: dia, OpensAt: 0, ClosesAt: 24 * 60}); err != nil {
			t.Fatal(err)
		}
	}
	// Intervalo só da profissional Ana.
	if _, err := store.AddBreak(SchedulingBreak{
		OrganizationID: org, ResourceID: recursoA, Weekday: int(time.Monday),
		StartsAtMinute: 12 * 60, EndsAtMinute: 13 * 60, Label: "almoço",
	}); err != nil {
		t.Fatal(err)
	}
	horario := time.Date(2026, 10, 19, 12, 15, 0, 0, loc)
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recursoA, ServiceID: servico,
		StartsAt: horario, Channel: "voice",
	}); !errors.Is(err, ErrSchedulingOnBreak) {
		t.Fatalf("a profissional com intervalo precisa recusar: %v", err)
	}
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recursoB.ID, ServiceID: servico,
		StartsAt: horario, Channel: "voice",
	}); err != nil {
		t.Fatalf("o outro recurso NÃO pode ser afetado pelo intervalo alheio: %v", err)
	}
}

func TestTurnoQueCruzaAMeiaNoiteUsaODiaLocalCorreto(t *testing.T) {
	store, org, recurso, servico := schedulingFixtureSemTurno(t)
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	// Barbearia que abre 22:00 e fecha 02:00 (o turno ATRAVESSA a meia-noite).
	if _, err := store.AddShift(SchedulingShift{OrganizationID: org, Weekday: int(time.Monday), OpensAt: 22 * 60, ClosesAt: 2 * 60}); err != nil {
		t.Fatalf("turno que cruza a meia-noite precisa ser ACEITO no registro: %v", err)
	}
	segunda := time.Date(2026, 10, 19, 0, 0, 0, 0, loc)
	slots, err := store.OpenSlots(org, recurso, segunda.Format("2006-01-02"), servico, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 4 {
		t.Fatalf("22:00->02:00 com passo de 60min e serviço de 30min precisa ter 4 horários, veio %d: %+v", len(slots), slots)
	}
	// O PRIMEIRO é na segunda (22:00) e os de madrugada caem na TERÇA (00:00 e 01:00) — dia local correto.
	if slots[0].StartsAt.In(loc).Day() != 19 || slots[0].StartsAt.In(loc).Hour() != 22 {
		t.Fatalf("primeiro horário precisa ser 19/10 22:00 local: %s", slots[0].StartsAt.In(loc))
	}
	if slots[2].StartsAt.In(loc).Day() != 20 || slots[2].StartsAt.In(loc).Hour() != 0 {
		t.Fatalf("o horário de 00:00 precisa cair no DIA SEGUINTE (20/10), veio %s", slots[2].StartsAt.In(loc))
	}
	if slots[3].StartsAt.In(loc).Hour() != 1 || slots[3].StartsAt.In(loc).Day() != 20 {
		t.Fatalf("o horário de 01:00 precisa ser 20/10 01:00 local: %s", slots[3].StartsAt.In(loc))
	}

	// Agendar 01:00 é aceito e grava o instante LOCAL do dia seguinte; 03:00 é recusado (fora do turno).
	agendado, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 20, 1, 0, 0, 0, loc), Channel: "voice", IdempotencyKey: "madrugada-1",
	})
	if err != nil {
		t.Fatalf("01:00 dentro do turno que cruza precisa ser aceito: %v", err)
	}
	if agendado.StartsAt.In(loc).Day() != 20 || agendado.StartsAt.In(loc).Hour() != 1 {
		t.Fatalf("o agendamento precisa ficar no dia local correto: %s", agendado.StartsAt.In(loc))
	}
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 20, 3, 0, 0, 0, loc), Channel: "voice",
	}); !errors.Is(err, ErrSchedulingOutsideHours) {
		t.Fatalf("03:00 está fora do turno e precisa ser recusado: %v", err)
	}
	// E 12:00 do dia seguinte (fora do turno da madrugada) também é recusado.
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: time.Date(2026, 10, 20, 12, 0, 0, 0, loc), Channel: "voice",
	}); !errors.Is(err, ErrSchedulingOutsideHours) {
		t.Fatalf("12:00 não pertence ao turno da madrugada: %v", err)
	}
}

func TestTurnoDeDuracaoZeroContinuaRecusado(t *testing.T) {
	store, org, _, _ := schedulingFixtureSemTurno(t)
	if _, err := store.AddShift(SchedulingShift{OrganizationID: org, Weekday: 1, OpensAt: 9 * 60, ClosesAt: 9 * 60}); !errors.Is(err, ErrSchedulingInvalid) {
		t.Fatalf("janela de duração zero precisa continuar recusada: %v", err)
	}
}
