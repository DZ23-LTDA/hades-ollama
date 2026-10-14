package agent

import (
	"testing"
	"time"
)

// E-413 / ADV-60 — `Availability` NÃO CONSULTA EXPEDIENTE, E AGORA ISSO ESTÁ DECLARADO E FIXADO.
//
// **MEDIDO (6ª auditoria independente)**: `Availability` devolvia **4 slots livres** para uma organização **sem
// turno nenhum** num **domingo 03:00**, enquanto `OpenSlots` no mesmo dia devolvia **0** e `Book` recusava com
// `ErrSchedulingOutsideHours` — *e o comentário da função mandava os agentes de voz/WhatsApp consultarem **ela**
// antes de oferecer horário* ✗. A E-314 chegou a registrar que o buraco tinha sido fechado; foi fechado em
// `OpenSlots`, função nova.
//
// ***Este teste fixa o LIMITE em vez de fingir que não existe:*** *a função não mudou de comportamento — mudou
// de promessa (E-413). Quem oferece horário ao cliente é `OpenSlots`.*
func TestAvailabilityNaoConsultaExpedienteEIssoEstaDeclarado(t *testing.T) {
	store, org, recurso, servico := schedulingFixtureSemTurno(t)
	loc, _, err := store.schedulingLocationLocked(org)
	if err != nil {
		t.Fatal(err)
	}
	// Domingo 03:00 local — o instante exato que a auditoria usou.
	domingo := time.Date(2026, 10, 18, 3, 0, 0, 0, loc)
	if domingo.Weekday() != time.Sunday {
		t.Fatalf("o cenário precisa ser um domingo, veio %s", domingo.Weekday())
	}

	// (a) O LIMITE, fixado: sem expediente, `Availability` AINDA devolve slots — e é isso que o comentário declara.
	slots, err := store.Availability(org, recurso, domingo, domingo.Add(2*time.Hour), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("ARMADILHA: se `Availability` passou a respeitar expediente, o LIMITE DECLARADO em scheduling.go " +
			"ficou obsoleto — atualize o comentário (e a E-413) em vez de deixar a documentação mentir para o outro " +
			"lado")
	}
	// E ela não esconde o que faz: os slots vêm sem marcação de expediente.
	livres := 0
	for _, slot := range slots {
		if slot.Free {
			livres++
		}
	}
	if livres == 0 {
		t.Fatal("o cenário esperava slots marcados como livres (a função decide só por agendamentos)")
	}

	// (b) A FUNÇÃO QUE DEVE SER USADA PARA OFERECER HORÁRIO: `OpenSlots` recusa o mesmo instante.
	abertos, err := store.OpenSlots(org, recurso, domingo.Format("2006-01-02"), servico, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(abertos) != 0 {
		t.Fatalf("ARMADILHA: organização sem turno não pode ter horário aberto no domingo, veio %d: %+v",
			len(abertos), abertos)
	}

	// (c) E o `Book` recusa — a ponta que o cliente sente.
	if _, err := store.Book(BookRequest{
		OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		StartsAt: domingo, Channel: "voice", IdempotencyKey: "madrugada-sem-turno",
	}); err == nil {
		t.Fatal("ARMADILHA: `Book` aceitou um domingo 03:00 sem expediente — a ponta que importa tem de recusar")
	}
}
