package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// AUDITORIA ADVERSARIAL — SEGUNDA ONDA (E-335).
//
// Ataques declarados em E-330: **fila de espera × cancelamento simultâneos** e **`Reschedule` durante oferta de fila**.
// A pergunta de fundo: *o motor pode oferecer a um cliente um horário que já não existe?* Se puder, a fila vira uma
// promessa falsa — e o cliente só descobre ao confirmar.

// ACHADO CANDIDATO 1: `OfferNextFromWaitlist` não confere se o horário está LIVRE antes de oferecer. Um horário já
// ocupado (por agendamento normal, por outro canal) não pode ser oferecido.
func TestAuditoriaFilaNaoOfereceHorarioJaOcupado(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 15, 10, 0, 0, 0, loc)

	// O horário JÁ está ocupado por um agendamento comum.
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000001", Channel: "voice", StartsAt: slot, IdempotencyKey: "ocupado-antes",
	}); err != nil {
		t.Fatal(err)
	}
	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000002", Channel: "whatsapp",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC())
	if err != nil {
		t.Fatalf("a consulta da fila não deve explodir: %v", err)
	}
	if achou {
		t.Fatalf("ACHADO CONFIRMADO: a fila ofereceu um horário JÁ OCUPADO (entrada %s, slot %s) — "+
			"oferecer o que não existe é promessa falsa ao cliente", oferta.ID, slot.Format(time.RFC3339))
	}
	fila, err := store.Waitlist(org)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range fila {
		if item.ID == entrada.ID && item.Status != WaitlistWaiting {
			t.Fatalf("a entrada precisa continuar aguardando, não virar oferta de horário inexistente: %+v", item)
		}
	}
}

// ACHADO CANDIDATO 2: horário fora do expediente ou em data fechada também não pode ser oferecido.
func TestAuditoriaFilaNaoOfereceForaDoExpediente(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	// O fixture da fila usa expediente 24/7; aqui fechamos um dia inteiro e tentamos oferecer nele.
	diaFechado := time.Date(2026, 12, 16, 0, 0, 0, 0, loc)
	if _, err := store.AddClosure(SchedulingClosure{
		OrganizationID: org, Date: diaFechado.Format("2006-01-02"),
		Reason: "feriado",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, From: diaFechado, To: diaFechado.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	slot := diaFechado.Add(10 * time.Hour)
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC()); err != nil {
		t.Fatalf("consulta da fila: %v", err)
	} else if achou {
		t.Fatal("ACHADO CONFIRMADO: a fila ofereceu horário em DIA FECHADO — o motor não pode inventar expediente")
	}
}

// ACHADO CANDIDATO 3: cancelamento libera horário e a fila deve ser capaz de oferecê-lo — o caminho feliz precisa
// continuar funcionando (senão eu teria "corrigido" transformando a fila em enfeite).
func TestAuditoriaCancelamentoLiberaEAFilaOfereceDeVerdade(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 17, 14, 0, 0, 0, loc)
	agendamento, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000003", Channel: "voice", StartsAt: slot, IdempotencyKey: "vai-cancelar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000004", Channel: "web",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// Antes do cancelamento não se oferece; depois, sim.
	if _, achou, _ := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC()); achou {
		t.Fatal("não pode oferecer horário ainda ocupado")
	}
	if _, err := store.Cancel(org, agendamento.ID, "desistiu"); err != nil {
		t.Fatal(err)
	}
	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC())
	if err != nil || !achou {
		t.Fatalf("após o cancelamento a fila PRECISA oferecer: achou=%v err=%v", achou, err)
	}
	// E a confirmação funciona.
	if _, err := store.ConfirmWaitlistOffer(org, oferta.ID, time.Now().UTC()); err != nil {
		t.Fatalf("confirmar oferta legítima: %v", err)
	}
}

// ACHADO CANDIDATO 4: `Reschedule` durante oferta de fila. O horário oferecido NÃO está reservado; quem remarcar
// primeiro leva. O que o motor NÃO pode fazer é deixar isso silencioso: a confirmação precisa falhar de forma
// explícita e a entrada não pode ficar presa em estado de oferta para sempre.
func TestAuditoriaRescheduleDuranteOfertaNaoPassaEmSilencio(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 18, 9, 0, 0, 0, loc)
	outro, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000005", Channel: "voice",
		StartsAt: time.Date(2026, 12, 18, 15, 0, 0, 0, loc), IdempotencyKey: "vizinho",
	})
	if err != nil {
		t.Fatal(err)
	}
	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000006", Channel: "whatsapp",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC())
	if err != nil || !achou {
		t.Fatalf("oferta: achou=%v err=%v", achou, err)
	}
	// Enquanto a oferta está de pé, o horário é remarcado para outra pessoa (a oferta NÃO reserva — limite declarado).
	if _, err := store.Reschedule(org, outro.ID, slot); err != nil {
		t.Fatalf("remarcar para o horário oferecido (não reservado) precisa ser permitido: %v", err)
	}
	// CONFIRMAR precisa ser RECUSADO, com erro explícito — nada de silêncio.
	if _, err := store.ConfirmWaitlistOffer(org, oferta.ID, time.Now().UTC()); !errors.Is(err, ErrSchedulingSlotTaken) {
		t.Fatalf("confirmar oferta cujo horário foi tomado precisa ser recusado explicitamente: %v", err)
	}
	// E a entrada NÃO pode ficar presa: precisa voltar a ser agendável (waiting) para receber nova oferta.
	fila, err := store.Waitlist(org)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range fila {
		if item.ID != entrada.ID {
			continue
		}
		if item.Status == WaitlistOffered {
			t.Fatalf("ACHADO CONFIRMADO: entrada ficou PRESA em oferta de horário que já não existe: %+v", item)
		}
	}
}

// ACHADO CANDIDATO 5: concorrência real entre confirmação de fila e outro agendamento no MESMO horário. Exatamente um
// ganhador — nunca dois.
func TestAuditoriaFilaEOutroCanalDisputandoOMesmoHorario(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 19, 11, 0, 0, 0, loc)
	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000007", Channel: "web",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, agora)
	if err != nil || !achou {
		t.Fatalf("oferta: achou=%v err=%v", achou, err)
	}

	var wg sync.WaitGroup
	erros := make([]error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); _, erros[0] = store.ConfirmWaitlistOffer(org, oferta.ID, agora) }()
	go func() {
		defer wg.Done()
		_, erros[1] = store.Book(BookRequest{
			OrganizationID: org, ResourceID: recurso, ServiceID: servico,
			ContactID: "+5511900000008", Channel: "voice", StartsAt: slot, IdempotencyKey: "corrida-fila",
		})
	}()
	wg.Wait()

	sucessos := 0
	for _, err := range erros {
		if err == nil {
			sucessos++
		}
	}
	if sucessos != 1 {
		t.Fatalf("o mesmo horário precisa ter EXATAMENTE um ganhador entre fila e canal direto: %v", erros)
	}
	agendamentos, _ := store.Appointments(org)
	ativos := 0
	for _, a := range agendamentos {
		if a.Status == SchedulingAppointmentBooked {
			ativos++
		}
	}
	if ativos != 1 {
		t.Fatalf("não pode haver dois agendamentos ativos no mesmo recurso: %d", ativos)
	}
	_ = entrada
	_ = context.Background()
}
