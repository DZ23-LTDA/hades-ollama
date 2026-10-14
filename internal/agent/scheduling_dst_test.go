package agent

import (
	"testing"
	"time"
)

// E-419 / ADV-64 — HORÁRIO DE VERÃO: `OpenSlots` NÃO PODE OFERECER O MESMO INSTANTE DUAS VEZES.
//
// **MEDIDO (6ª auditoria independente)**: com a organização em `America/New_York` e um turno de domingo
// 01:00–05:00, em **2026-03-08** (*o salto de 02:00 para 03:00*) o `OpenSlots` devolvia **8 slots com 01:00 e 01:30
// duplicados** ✗ — *o mesmo instante UTC duas vezes*. **CAUSA, lida no código**: o laço
// (`scheduling_hours.go:259-260`) anda por **deslocamento de relógio de parede** e calcula cada início com
// `time.Date(..., loc)` ⇒ *`time.Date` **normaliza** horários que não existem* ✗ — *02:00 e 02:30 caem ambos em
// 03:00, e a lista final apenas **ordena** (`:277`), sem deduplicar* ✓.
//
// ***A máquina desta base é `America/Sao_Paulo`, que não tem horário de verão desde 2019 — foi por isso que
// nenhum teste pegou.*** E o defeito **não** é do relógio da máquina: é da lista que o produto publica.
func TestOpenSlotsNaoDuplicaInstanteNaViradaDeHorarioDeVerao(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("fuso America/New_York indisponível nesta máquina: %v", err)
	}
	// 2026-03-08 é o domingo do spring-forward nos EUA: 02:00 vira 03:00 (horário local inexistente).
	dia := time.Date(2026, 3, 8, 0, 0, 0, 0, loc)
	if dia.Weekday() != time.Sunday {
		t.Fatalf("o cenário precisa ser domingo, veio %s", dia.Weekday())
	}

	loja, org, recurso, _ := schedulingFixtureSemTurno(t)
	if err := loja.SetTimeZone(org, "America/New_York"); err != nil {
		t.Fatalf("definir o fuso da organização: %v", err)
	}
	// Turno APENAS no domingo, 01:00–05:00 — a janela que contém o horário inexistente.
	if _, err := loja.AddShift(SchedulingShift{OrganizationID: org, Weekday: 0, OpensAt: 1 * 60, ClosesAt: 5 * 60}); err != nil {
		t.Fatalf("turno de domingo: %v", err)
	}
	servico, err := loja.AddService(SchedulingService{OrganizationID: org, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}

	slots, err := loja.OpenSlots(org, recurso, "2026-03-08", servico.ID, 30)
	if err != nil {
		t.Fatal(err)
	}

	// (a) NENHUM INSTANTE REPETIDO. É a asserção que o auditor mediu como falhando.
	vistos := map[int64]int{}
	for _, slot := range slots {
		vistos[slot.StartsAt.UTC().Unix()]++
	}
	for instante, quantas := range vistos {
		if quantas > 1 {
			t.Fatalf("ARMADILHA: o instante %s (UTC) foi oferecido %d vezes em %s — *um horário que não existe "+
				"foi normalizado para o mesmo instante de outro*. Slots: %v",
				time.Unix(instante, 0).UTC().Format(time.RFC3339), quantas,
				dia.Format("2006-01-02"), slots)
		}
	}

	// (b) E O QUE SOBRA É HONESTO: os horários de parede oferecidos têm de **existir** e cair no domingo local.
	for _, slot := range slots {
		local := slot.StartsAt.In(loc)
		if local.Weekday() != time.Sunday {
			t.Fatalf("ARMADILHA: o slot %s caiu em %s, não no domingo — a virada de dia foi mal calculada",
				local.Format(time.RFC3339), local.Weekday())
		}
		// `time.Date` com o mesmo horário de parede tem de reproduzir o MESMO instante: se não reproduz, o
		// horário de parede informado não existe nesse dia (foi normalizado).
		reconstruido := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, loc)
		if !reconstruido.Equal(slot.StartsAt) {
			t.Fatalf("ARMADILHA: o slot reporta %s mas o relógio local reconstrói %s — *o horário de parede "+
				"informado não existe nesse dia* (normalização de horário de verão)",
				slot.StartsAt.Format(time.RFC3339), reconstruido.Format(time.RFC3339))
		}
	}
	t.Logf("MEDIDA ESTRUTURAL: %d slot(s) distintos na virada de horário de verão (%s)", len(slots), dia.Format("2006-01-02"))
}
