package agent

import (
	"errors"
	"testing"
	"time"
)

// AGENDAMENTO — FATIA 3: REMARCAÇÃO ATÔMICA (V2.2, E-319).
//
// A PROPRIEDADE QUE IMPORTA: **uma remarcação que FALHA não pode destruir o agendamento original.**
// O caminho ingênuo (cancelar e depois tentar agendar) deixa o cliente **sem horário nenhum** quando o novo horário já
// está ocupado — pior do que recusar remarcar. Cada direção de falha abaixo confere que o original segue **intacto**.
func TestRemarcacaoEhAtomicaENaoDeixaBuraco(t *testing.T) {
	store, org, recurso, servico := schedulingFixture(t) // expediente 24/7 declarado no fixture
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 11, 2, 9, 0, 0, 0, loc) // segunda-feira, 09:00 local

	original, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000001", Channel: "voice", StartsAt: base, IdempotencyKey: "orig-1"})
	if err != nil {
		t.Fatal(err)
	}
	vizinho, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000002", Channel: "whatsapp", StartsAt: base.Add(time.Hour), IdempotencyKey: "viz-1"})
	if err != nil {
		t.Fatal(err)
	}
	estadoOriginal := original

	// 1. FALHA por sobreposição: o novo horário colide com o vizinho ⇒ recusa e o ORIGINAL fica intacto.
	if _, err := store.Reschedule(org, original.ID, vizinho.StartsAt.Add(5*time.Minute)); !errors.Is(err, ErrSchedulingSlotTaken) {
		t.Fatalf("remarcar para horário ocupado precisa ser recusado: %v", err)
	}
	agendamentos, err := store.Appointments(org)
	if err != nil {
		t.Fatal(err)
	}
	atual := encontrarAgendamento(t, agendamentos, original.ID)
	if !atual.StartsAt.Equal(estadoOriginal.StartsAt) || !atual.BusyUntil.Equal(estadoOriginal.BusyUntil) || atual.Status != SchedulingAppointmentBooked {
		t.Fatalf("APÓS FALHA o original tem de ficar intacto: antes=%+v depois=%+v", estadoOriginal, atual)
	}

	// 2. FALHA por expediente: o fixture é 24/7, então usamos uma data FECHADA para provar a recusa da fatia 2.
	fechado := base.AddDate(0, 0, 1)
	if _, err := store.AddClosure(SchedulingClosure{OrganizationID: org, Date: fechado.Format("2006-01-02"), Reason: "feriado"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reschedule(org, original.ID, fechado.Add(10*time.Hour)); !errors.Is(err, ErrSchedulingClosed) {
		t.Fatalf("remarcar para data fechada precisa ser recusado: %v", err)
	}
	agendamentos, _ = store.Appointments(org)
	atual = encontrarAgendamento(t, agendamentos, original.ID)
	if !atual.StartsAt.Equal(estadoOriginal.StartsAt) {
		t.Fatalf("recusa por data fechada não pode mover o original: %+v", atual)
	}

	// 3. FALHA por inexistente.
	if _, err := store.Reschedule(org, "appt_nao_existe", base.Add(2*time.Hour)); !errors.Is(err, ErrSchedulingNotFound) {
		t.Fatalf("remarcar inexistente precisa ser recusado: %v", err)
	}

	// 4. SUCESSO: horário livre ⇒ o original se move, o horário antigo liberta e o vizinho NÃO é afetado.
	novo := base.Add(2 * time.Hour)
	movido, err := store.Reschedule(org, original.ID, novo)
	if err != nil {
		t.Fatalf("remarcar para horário livre precisa funcionar: %v", err)
	}
	if !movido.StartsAt.Equal(novo.UTC()) {
		t.Fatalf("o agendamento precisa estar no novo horário: %+v", movido)
	}
	if !movido.EndsAt.Equal(novo.UTC().Add(30 * time.Minute)) {
		t.Fatalf("a duração do serviço precisa ser preservada: %+v", movido)
	}
	agendamentos, _ = store.Appointments(org)
	if !encontrarAgendamento(t, agendamentos, vizinho.ID).StartsAt.Equal(vizinho.StartsAt) {
		t.Fatal("o agendamento vizinho não pode ser tocado pela remarcação de outro")
	}
	// O horário ANTIGO volta a ser agendável (não ficou "buraco" nem vaga fantasma).
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900000003", Channel: "web", StartsAt: base, IdempotencyKey: "novo-no-antigo"}); err != nil {
		t.Fatalf("o horário liberado precisa voltar a ser agendável: %v", err)
	}

	// 5. Remarcar para o MESMO horário é permitido (idempotência natural), sem criar segundo agendamento.
	// MEDIDO: eu capturei a contagem ANTES de um agendamento novo entrar (passo 4) e comparei depois - de novo
	// aritmética em instante errado. A contagem tem de ser lida IMEDIATAMENTE antes da operação que está sendo medida.
	agendamentos, _ = store.Appointments(org)
	quantosAntes := len(agendamentos)
	if _, err := store.Reschedule(org, original.ID, novo); err != nil {
		t.Fatalf("remarcar para o mesmo horário não deve falhar: %v", err)
	}
	agendamentos, _ = store.Appointments(org)
	if len(agendamentos) != quantosAntes {
		t.Fatalf("remarcar não pode criar agendamento novo: %d != %d", len(agendamentos), quantosAntes)
	}

	// 6. A lista oferecida para remarcar considera que o PRÓPRIO agendamento pode sair do lugar.
	livres, err := store.FreeSlotsForReschedule(org, original.ID, novo.Format("2006-01-02"), 30)
	if err != nil {
		t.Fatal(err)
	}
	achouProprio := false
	for _, slot := range livres {
		if slot.StartsAt.Equal(novo.UTC()) {
			achouProprio = true
			if !slot.Free {
				t.Fatal("o horário que o próprio agendamento ocupa precisa aparecer como LIVRE para remarcação")
			}
		}
	}
	if !achouProprio {
		t.Fatalf("a lista de remarcação precisa conter o horário atual do próprio agendamento: %+v", livres)
	}
}

func encontrarAgendamento(t *testing.T, agendamentos []SchedulingAppointment, id string) SchedulingAppointment {
	t.Helper()
	for _, agendamento := range agendamentos {
		if agendamento.ID == id {
			return agendamento
		}
	}
	t.Fatalf("agendamento %s não encontrado", id)
	return SchedulingAppointment{}
}
