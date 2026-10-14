package agent

import (
	"testing"
	"time"
)

// E-420 — O FIM DO HORÁRIO DE VERÃO: A HORA QUE SE REPETE.
//
// No **spring-forward** (E-419) o problema era um horário **inexistente** normalizado para o mesmo instante de
// outro, e a correção deduplicou por instante. No **fall-back** o caso é o **oposto**: *01:00–01:59 acontece
// **duas vezes** — uma em EDT e outra em EST — e são **instantes distintos e ambos reais***.
//
// ⚠️ **Este teste existe para medir uma DECLARAÇÃO, não para confirmar uma suposição.** No E-419 eu escrevi que a
// deduplicação por instante "**preserva** a hora legitimamente repetida do fim do horário de verão". Isso é uma
// **intenção**: *o laço anda por deslocamento de **relógio de parede** e `time.Date` resolve cada horário de
// parede para **um** instante* ⇒ *é perfeitamente possível que ele ofereça a hora repetida **uma única vez**, e a
// minha declaração estaria errada* ✗.
//
// ***Declaração não é medição — e é exatamente por isso que ela vira teste.***
//
// O teste **não afirma** que os dois instantes aparecem (isso seria escrever o resultado antes de medir). Ele
// **mede** e diz, no log, o que o produto faz hoje.
func TestOpenSlotsNoFimDoHorarioDeVerao(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("fuso America/New_York indisponível: %v — *um teste que se pula sozinho é verde sem medir nada; "+
			"instale o banco de fusos (tzdata) em vez de deixar isto passar em silêncio*", err)
	}
	// 2026-11-01 é o domingo do fall-back nos EUA: 02:00 EDT volta para 01:00 EST.
	dia := time.Date(2026, 11, 1, 0, 0, 0, 0, loc)
	if dia.Weekday() != time.Sunday {
		t.Fatalf("o cenário precisa ser domingo, veio %s", dia.Weekday())
	}

	loja, org, recurso, _ := schedulingFixtureSemTurno(t)
	if err := loja.SetTimeZone(org, "America/New_York"); err != nil {
		t.Fatalf("definir o fuso: %v", err)
	}
	if _, err := loja.AddShift(SchedulingShift{OrganizationID: org, Weekday: 0, OpensAt: 0, ClosesAt: 3 * 60}); err != nil {
		t.Fatalf("turno de domingo 00:00–03:00: %v", err)
	}
	servico, err := loja.AddService(SchedulingService{OrganizationID: org, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}

	slots, err := loja.OpenSlots(org, recurso, "2026-11-01", servico.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	// (a) NENHUM instante repetido — a mesma propriedade do E-419, agora no outro lado da virada.
	vistos := map[int64]int{}
	for _, slot := range slots {
		vistos[slot.StartsAt.UTC().Unix()]++
	}
	for instante, quantas := range vistos {
		if quantas > 1 {
			t.Fatalf("ARMADILHA: o instante %s foi oferecido %d vezes no fall-back — a deduplicação por instante "+
				"não pode colapsar dois instantes IGUAIS (se são iguais, é duplicata; se são distintos, devem ambos "+
				"aparecer)", time.Unix(instante, 0).UTC().Format(time.RFC3339), quantas)
		}
	}

	// (b) QUANTOS instantes com o MESMO horário de parede de 01:00? *Esta é a medição da minha declaração.*
	comUmaDaManha := 0
	for instante := range vistos {
		local := time.Unix(instante, 0).In(loc)
		if local.Hour() == 1 && local.Minute() == 0 {
			comUmaDaManha++
		}
	}
	t.Logf("MEDIDA ESTRUTURAL: %d slot(s) no fall-back; instantes distintos com horário de parede 01:00 = %d "+
		"(2 significaria que os dois lados da hora repetida são oferecidos; 1 significa que o laço por relógio de "+
		"parede resolve 01:00 para um único instante)", len(slots), comUmaDaManha)

	// (c) E o que é oferecido tem de ser COERENTE: o horário de parede informado reconstrói o mesmo instante.
	for _, slot := range slots {
		local := slot.StartsAt.In(loc)
		reconstruido := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, loc)
		if !reconstruido.Equal(slot.StartsAt) {
			t.Fatalf("ARMADILHA: o slot reporta %s mas o relógio local reconstrói %s — o horário de parede "+
				"informado não é o que o instante representa",
				slot.StartsAt.Format(time.RFC3339), reconstruido.Format(time.RFC3339))
		}
	}
}
