package agent

import (
	"errors"
	"testing"
	"time"
)

// E-459 — **A DUPLICATA APÓS CANCELAMENTO: O LIMITE QUE E-458 DECLAROU NÃO MEDIDO.**
//
// **A PERGUNTA, e ela é binária**: o mesmo cliente, com a **mesma `IdempotencyKey`**, consegue **reagendar depois de
// cancelar**?
//
// **O QUE E-458 DECLAROU**: *"`ErrSchedulingDuplicate` só cobre `Status == Booked` — um agendamento já cancelado com a
// mesma chave **não** é reconhecido como duplicata; comportamento **não medido**, e não afirmo se é defeito"* ✗.
//
// **POR QUE A RESPOSTA IMPORTA, e qual é a certa**: a chave identifica *"a MESMA intenção vinda do mesmo canal"*
// (`scheduling.go:266`). **Depois de um cancelamento a intenção é NOVA** — *o cliente quer outro horário, e o
// cancelamento é exactamente o acto que liberta a agenda para ele.* ⇒ se o motor **recusasse**, o cliente ficaria
// **preso**: *cancelou uma vez e nunca mais consegue marcar com aquela intenção.*
//
// ⇒ ***A leitura de `Status == Booked` não é um esquecimento: é a leitura CERTA.*** Este teste mede-o em vez de o supor,
// e mede também o lado oposto — *que a protecção **continua valendo** enquanto o agendamento está vivo.*
func TestReagendarDepoisDeCancelarComAMesmaChave(t *testing.T) {
	raiz := t.TempDir()
	store := NewSchedulingStore(raiz)
	organizacao := LocalOrganizationID

	servico, err := store.AddService(SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := store.AddResource(SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	dia := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	for deslocamento := 0; deslocamento < 7; deslocamento++ {
		if _, err := store.AddShift(SchedulingShift{
			OrganizationID: organizacao, ResourceID: recurso.ID,
			Weekday: int(dia.AddDate(0, 0, deslocamento).Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
		}); err != nil {
			t.Fatalf("criar turno: %v", err)
		}
	}

	pedido := func(inicio time.Time) BookRequest {
		return BookRequest{
			OrganizationID: organizacao, ResourceID: recurso.ID, ServiceID: servico.ID,
			ContactID: "cliente-reagenda", Channel: "voice", StartsAt: inicio, IdempotencyKey: "k-reagenda",
		}
	}
	primeiro, err := store.Book(pedido(time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("primeiro agendamento: %v", err)
	}

	// (1) ENQUANTO ESTÁ VIVO, a protecção vale: *é o que E-458 provou, e tem de continuar provado aqui para o teste
	// medir a DIFERENÇA e não só o caminho novo.*
	if _, err := store.Book(pedido(time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC))); !errors.Is(err, ErrSchedulingDuplicate) {
		t.Fatalf("a reentrega com o agendamento VIVO tinha de dar ErrSchedulingDuplicate, veio: %v", err)
	}

	// (2) CANCELA.
	if _, err := store.Cancel(organizacao, primeiro.ID, "o cliente desmarcou"); err != nil {
		t.Fatalf("cancelar: %v", err)
	}

	// (3) **A MEDIDA**: com o mesmo `IdempotencyKey`, depois do cancelamento, o cliente TEM de conseguir reagendar.
	segundo, err := store.Book(pedido(time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("ARMADILHA: depois de cancelar, a MESMA chave foi recusada (%v) — *o cliente ficaria PRESO: cancelou "+
			"uma vez e nunca mais marca com aquela intenção. A leitura de `Status == Booked` não é esquecimento: é a "+
			"leitura CERTA*", err)
	}
	if segundo.ID == "" || segundo.ID == primeiro.ID {
		t.Fatalf("o reagendamento tem de ser um agendamento NOVO (primeiro=%q segundo=%q)", primeiro.ID, segundo.ID)
	}

	// E o motor tem de contar a verdade: **dois** registos, um cancelado e um agendado.
	agendamentos, err := store.Appointments(organizacao)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	cancelados, agendados := 0, 0
	for _, agendamento := range agendamentos {
		switch agendamento.Status {
		case SchedulingAppointmentCancelled:
			cancelados++
		case SchedulingAppointmentBooked:
			agendados++
		}
	}
	if len(agendamentos) != 2 || cancelados != 1 || agendados != 1 {
		t.Fatalf("esperava 2 registos (1 cancelado, 1 agendado), veio %d (cancelados=%d agendados=%d) — *cancelar não "+
			"apaga história: a agenda precisa de saber que houve um cancelamento*", len(agendamentos), cancelados, agendados)
	}
}
