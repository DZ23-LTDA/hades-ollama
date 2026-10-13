package agent

import (
	"errors"
	"testing"
	"time"
)

// LISTA DE ESPERA, LEMBRETES E NO-SHOW — TESTES (E-329; V2.2 incremento I, fatia 6).
//
// A armadilha mais importante está no teste da OFERTA: oferecer **não** pode agendar. O V2.2 proíbe contato sem
// consentimento; confirmar por conta própria o horário de quem nem respondeu é agir em nome do cliente.

func waitlistFixture(t *testing.T) (*SchedulingStore, string, string, string, *time.Location) {
	t.Helper()
	store, org, recurso, servico := schedulingFixture(t) // expediente 24/7 declarado
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	return store, org, recurso, servico, loc
}

func TestOfertaDaFilaOfereceMasNaoAgenda(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 14, 10, 0, 0, 0, loc) // segunda-feira
	janela := [2]time.Time{slot.Add(-2 * time.Hour), slot.Add(4 * time.Hour)}

	primeiro, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000001", Channel: "whatsapp", From: janela[0], To: janela[1],
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Entrada mais NOVA na mesma janela: a FIFO precisa escolher a primeira.
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000002", Channel: "voice", From: janela[0], To: janela[1],
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC())
	if err != nil || !achou {
		t.Fatalf("deveria haver oferta: achou=%v err=%v", achou, err)
	}
	if oferta.ID != primeiro.ID {
		t.Fatalf("FIFO precisa oferecer ao PRIMEIRO da fila: esperado %s, veio %s", primeiro.ID, oferta.ID)
	}
	if oferta.Status != WaitlistOffered || oferta.OfferedSlot == nil || oferta.OfferExpiresAt == nil {
		t.Fatalf("a entrada precisa ficar em oferta COM prazo: %+v", oferta)
	}
	// ARMADILHA CENTRAL: nenhum agendamento pode ter nascido da oferta.
	agendamentos, err := store.Appointments(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(agendamentos) != 0 {
		t.Fatalf("ARMADILHA: a oferta AGENDOU sozinha — oferecer é pedir confirmação, não agir em nome do cliente: %+v", agendamentos)
	}
}

func TestFilaRespeitaJanelaDesejada(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 14, 18, 0, 0, 0, loc)
	// Janela que NÃO contém o horário oferecido.
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, From: slot.Add(-8 * time.Hour), To: slot.Add(-4 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC()); err != nil || achou {
		t.Fatalf("janela que não contém o horário NÃO pode receber oferta: achou=%v err=%v", achou, err)
	}
	// Janela que contém.
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC()); err != nil || !achou {
		t.Fatalf("janela que contém o horário precisa receber oferta: achou=%v err=%v", achou, err)
	}
}

func TestConfirmacaoReconfereDisponibilidadeEExpira(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 14, 11, 0, 0, 0, loc)
	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000003", Channel: "web",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, agora); err != nil || !achou {
		t.Fatalf("oferta: achou=%v err=%v", achou, err)
	}
	// Alguém agenda o MESMO horário por outro canal antes da confirmação.
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000004", Channel: "voice", StartsAt: slot, IdempotencyKey: "corrida-1",
	}); err != nil {
		t.Fatal(err)
	}
	// ARMADILHA: confirmar precisa passar pelo portão e ser RECUSADO.
	if _, err := store.ConfirmWaitlistOffer(org, entrada.ID, agora); !errors.Is(err, ErrSchedulingSlotTaken) {
		t.Fatalf("confirmar depois de o horário ser ocupado precisa ser recusado: %v", err)
	}

	// Oferta VENCIDA: recusa explícita e o expirado NÃO volta para a fila.
	outroSlot := time.Date(2026, 12, 14, 15, 0, 0, 0, loc)
	segunda, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000005", Channel: "web",
		From: outroSlot.Add(-time.Hour), To: outroSlot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, outroSlot, agora); err != nil || !achou {
		t.Fatalf("oferta para o segundo: achou=%v err=%v", achou, err)
	}
	depoisDoPrazo := agora.Add(WaitlistOfferWindow + time.Minute)
	if _, err := store.ConfirmWaitlistOffer(org, segunda.ID, depoisDoPrazo); !errors.Is(err, ErrWaitlistOfferExpired) {
		t.Fatalf("oferta vencida precisa ser recusada explicitamente: %v", err)
	}
	fila, err := store.Waitlist(org)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range fila {
		if item.ID == segunda.ID && item.Status == WaitlistWaiting {
			t.Fatal("quem perdeu o prazo NÃO pode voltar para a fila de espera como se nada tivesse acontecido")
		}
		if item.ID == segunda.ID && item.Status != WaitlistExpired {
			t.Fatalf("o estado precisa ser expired: %+v", item)
		}
	}
}

func TestConfirmacaoBemSucedidaCriaAgendamentoEChamaDeNovoNaoDuplica(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 14, 16, 0, 0, 0, loc)
	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, ContactID: "+5511900000006", Channel: "whatsapp",
		From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	if _, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, agora); err != nil || !achou {
		t.Fatalf("oferta: achou=%v err=%v", achou, err)
	}
	criado, err := store.ConfirmWaitlistOffer(org, entrada.ID, agora)
	if err != nil {
		t.Fatalf("confirmar: %v", err)
	}
	if !criado.StartsAt.Equal(slot.UTC()) {
		t.Fatalf("o agendamento precisa ficar no horário oferecido: %+v", criado)
	}
	// Chamar de novo devolve o MESMO agendamento (idempotência) e não cria segundo.
	denovo, err := store.ConfirmWaitlistOffer(org, entrada.ID, agora)
	if err != nil {
		t.Fatalf("confirmar duas vezes precisa ser idempotente: %v", err)
	}
	if denovo.ID != criado.ID {
		t.Fatalf("a segunda confirmação devolveu agendamento diferente: %s != %s", denovo.ID, criado.ID)
	}
	agendamentos, _ := store.Appointments(org)
	if len(agendamentos) != 1 {
		t.Fatalf("confirmar duas vezes não pode criar dois agendamentos: %d", len(agendamentos))
	}
}

func TestLembreteNaoSaiDuasVezes(t *testing.T) {
	store, org, recurso, servico, _ := waitlistFixture(t)
	agora := time.Now().UTC()
	marcado := agora.Add(90 * time.Minute)
	agendamento, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000007", Channel: "voice", StartsAt: marcado, IdempotencyKey: "lembrete-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	devidos, err := store.DueReminders(org, "2h", 2*time.Hour, agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(devidos) != 1 || devidos[0].ID != agendamento.ID {
		t.Fatalf("o lembrete está devido e precisa aparecer: %+v", devidos)
	}
	if err := store.MarkReminderSent(org, agendamento.ID, "2h", agora); err != nil {
		t.Fatalf("marcar envio: %v", err)
	}
	// ARMADILHA: o MESMO lembrete não pode ser devido de novo.
	devidos, err = store.DueReminders(org, "2h", 2*time.Hour, agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(devidos) != 0 {
		t.Fatalf("ARMADILHA: o mesmo lembrete saiu duas vezes (lembrete repetido é spam): %+v", devidos)
	}
	// Marcar de novo é recusado explicitamente.
	if err := store.MarkReminderSent(org, agendamento.ID, "2h", agora); !errors.Is(err, ErrSchedulingInvalid) {
		t.Fatalf("marcar o mesmo lembrete duas vezes precisa ser recusado: %v", err)
	}
	// Um tipo DIFERENTE de lembrete continua devido (a chave é agendamento + tipo).
	if devidos, _ := store.DueReminders(org, "30min", 2*time.Hour, agora); len(devidos) != 1 {
		t.Fatalf("outro tipo de lembrete precisa continuar devido: %+v", devidos)
	}
	// Antecedência menor que a distância até o início: nada devido.
	if devidos, _ := store.DueReminders(org, "10min", 10*time.Minute, agora); len(devidos) != 0 {
		t.Fatalf("fora da antecedência não há lembrete devido: %+v", devidos)
	}
}

func TestNoShowEhRegistradoEEstadoMuda(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	agendamento, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000008", Channel: "voice",
		StartsAt: time.Date(2026, 12, 14, 9, 0, 0, 0, loc), IdempotencyKey: "falta-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	marcado, err := store.MarkNoShow(org, agendamento.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("marcar falta: %v", err)
	}
	if marcado.Status != SchedulingAppointmentNoShow {
		t.Fatalf("o estado precisa ser no_show: %+v", marcado)
	}
	// Marcar falta de novo no mesmo agendamento é recusado (não se registra falta duas vezes).
	if _, err := store.MarkNoShow(org, agendamento.ID, time.Now().UTC()); !errors.Is(err, ErrSchedulingInvalid) {
		t.Fatalf("marcar falta duas vezes precisa ser recusado: %v", err)
	}
	// O agendamento continua no histórico (não é removido).
	agendamentos, _ := store.Appointments(org)
	if len(agendamentos) != 1 {
		t.Fatalf("a falta não remove o agendamento do histórico: %d", len(agendamentos))
	}
}

func TestFilaNaoMisturaOrganizacoes(t *testing.T) {
	store, org, recurso, servico, loc := waitlistFixture(t)
	slot := time.Date(2026, 12, 14, 17, 0, 0, 0, loc)
	if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
		OrganizationID: org, ResourceID: recurso,
		ServiceID: servico, From: slot.Add(-time.Hour), To: slot.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	vizinha, err := store.Waitlist("org-vizinha")
	if err != nil {
		t.Fatal(err)
	}
	if len(vizinha) != 0 {
		t.Fatalf("a fila da organização vizinha precisa estar vazia: %+v", vizinha)
	}
	if _, achou, err := store.OfferNextFromWaitlist("org-vizinha", recurso, slot, time.Now().UTC()); err != nil || achou {
		t.Fatalf("a vizinha não pode receber oferta da fila alheia: achou=%v err=%v", achou, err)
	}
}
