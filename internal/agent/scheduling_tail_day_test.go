package agent

import "testing"

// E-412 / ADV-61 — A CAUDA DE UM TURNO QUE CRUZA A MEIA-NOITE PERTENCE AO DIA SEGUINTE.
//
// **MEDIDO (6ª auditoria independente)**: `shiftCoversMinute` aplicava o ramo da cauda
// (`fimMinuto <= shift.ClosesAt`) **sem distinguir o dia** ⇒ *`Book(segunda 01:00)` era **ACEITO** pelo turno de
// segunda que só abre às 22:00* ✗ — cliente confirmado com a empresa fechada, divergindo da própria lista de
// `OpenSlots`.
//
// ***O mesmo minuto (60) tem duas respostas opostas:*** 01:00 de **segunda** está **fora** do turno de segunda
// 22:00→02:00; 01:00 de **terça** está **dentro** da cauda desse turno. Quem sabe a diferença é o **dia**, e é
// por isso que `shiftCovering` devolve `diaAnterior` e o chamador soma `1440`.
func TestCaudaDeTurnoQueCruzaPertenceAoDiaSeguinte(t *testing.T) {
	// Turno que cruza a meia-noite: abre 22:00, fecha 02:00.
	turno := SchedulingShift{OpensAt: 22 * 60, ClosesAt: 2 * 60}

	// O MESMO minuto local, nas duas situações.
	const umaDaManha = 60

	// (a) Minuto do PRÓPRIO dia do turno (segunda 01:00): a cauda dele é amanhã, não hoje ⇒ FORA.
	if shiftCoversMinute(turno, umaDaManha, 0) {
		t.Fatal("ARMADILHA: 01:00 do PRÓPRIO dia do turno foi aceito — a cauda pertence ao dia SEGUINTE, e o " +
			"turno que abre às 22:00 não cobre a madrugada anterior. Foi assim que o Book confirmou cliente com a " +
			"empresa fechada (ADV-61).")
	}
	// E o fim do turno também: 23:00 de segunda pertence a ele.
	if !shiftCoversMinute(turno, 23*60, 0) {
		t.Fatal("ARMADILHA: 23:00 do próprio dia deveria estar dentro do turno que abre às 22:00")
	}

	// (b) Minuto do dia SEGUINTE (terça 01:00 = 60 + 1440): aí sim é a cauda ⇒ DENTRO.
	if !shiftCoversMinute(turno, umaDaManha+24*60, 0) {
		t.Fatal("ARMADILHA: 01:00 do dia SEGUINTE é a cauda legítima do turno de 22:00→02:00 e foi recusado — " +
			"é o caso que o teste E-328 defende, e ele tem de continuar aceito")
	}
	// E além do fechamento (terça 03:00 = 180 + 1440) ⇒ FORA.
	if shiftCoversMinute(turno, 3*60+24*60, 0) {
		t.Fatal("ARMADILHA: 03:00 do dia seguinte está além do fechamento (02:00) e foi aceito")
	}

	// (c) Turno que NÃO cruza continua com a checagem simples, sem regressão.
	diurno := SchedulingShift{OpensAt: 9 * 60, ClosesAt: 18 * 60}
	if !shiftCoversMinute(diurno, 12*60, 0) {
		t.Fatal("ARMADILHA: 12:00 deveria estar dentro de um turno 09:00–18:00")
	}
	if shiftCoversMinute(diurno, 8*60, 0) || shiftCoversMinute(diurno, 19*60, 0) {
		t.Fatal("ARMADILHA: horário fora de 09:00–18:00 foi aceito")
	}
	// (d) Duração: um serviço que ultrapassa o fechamento não cabe.
	if shiftCoversMinute(diurno, 17*60+30, 45) {
		t.Fatal("ARMADILHA: serviço que termina 18:15 deveria exceder o fechamento de 18:00")
	}
}
