package agent

import (
	"testing"
	"time"
)

// E-421 — INSTANTE DE TESTE NÃO SE HERDA DO RELÓGIO: ESCOLHE-SE.
//
// **Duas vezes** nesta missão um teste ficou vermelho por derivar um **instante de negócio** do relógio da
// máquina, e as duas só apareceram perto da meia-noite:
//
//   - `scheduling_test.go:17` — `agora := time.Now().UTC().Add(24 * time.Hour)` + `agora.Add(90 * time.Minute)`
//     ⇒ *às 23:41 o agendamento caía em **01:11 do dia seguinte***, fora de um turno `00:00–24:00` (E-417);
//   - `scheduling_waitlist_test.go:187` (*este deixou o **CI vermelho**, em `Go agentic and server gates`* ✗) —
//     `agora := time.Now().UTC()` + `agora.Add(90 * time.Minute)` ⇒ *o log dizia
//     `fora do expediente configurado: 2026-10-13T23:53:07-03:00 está fora da janela 00:00-24:00`* (E-421).
//
// **Nos dois casos o produto estava CERTO** — *um serviço que termina depois da meia-noite **não cabe** num turno
// que fecha 24:00* — e o defeito era do **fixture**, que **herdava** o instante do relógio em vez de **escolhê-lo**.
//
// ⇒ `instanteDeTeste` devolve **amanhã às 10:00 UTC**: *longe de qualquer virada de dia, de semana e de horário de
// verão, e um dia à frente para que "amanhã" nunca seja "hoje" no meio da execução.*
//
// ***O relógio do teste deve derivar do instante escolhido — nunca o contrário.*** *Se um teste precisa de "90
// minutos antes", escreva `instanteDeTeste(t).Add(-90 * time.Minute)`: a **intenção** fica idêntica e a hora da
// máquina deixa de importar.*
//
// **A guarda abaixo é o que impede isto de voltar em silêncio**: se alguém rodar a suíte num horário em que o
// instante escolhido cruzaria uma fronteira, o teste **falha dizendo por quê** em vez de acusar o produto.
func instanteDeTeste(t *testing.T) time.Time {
	t.Helper()
	// E-527 (CORRIGIDO, com a causa medida no CI): era `Add(24 * time.Hour)` - e a **truncagem para as 10:00**
	// faz com que a distancia real fique entre **10 h e 34 h**, enquanto a guarda exige **>= 20 h**.
	// ⇒ *sempre que "agora" esteja entre 10:00 e 24:00 UTC, o instante fica a <= 20 h e a guarda DISPARA.*
	//
	// ***Foi o que aconteceu em 2026-10-14 (14:46 UTC): o truncamento deu 2026-10-15T10:00Z, a 19 h 14 min.***
	// *A proteccao contra o relogio tornou-se, ela propria, dependente do relogio - e com PRAZO DE VALIDADE.*
	//
	// +48 h: o **minimo** passa a **34 h > 20 h**, e a guarda deixa de poder disparar por causa do horario do dia
	// - SEM afrouxar o criterio. *Continua a usar aritmetica RELATIVA (`Now() + offset`), logo NAO depende do
	// relogio ABSOLUTO da maquina.*
	base := time.Now().UTC().Add(48 * time.Hour)
	instante := time.Date(base.Year(), base.Month(), base.Day(), 10, 0, 0, 0, time.UTC)

	// Guarda: 10:00 UTC tem de estar a pelo menos 2 h de qualquer virada de dia. *É esta asserção que transforma
	// "escolhemos um horário seguro" em propriedade verificada.*
	local := instante.UTC()
	if hora := local.Hour(); hora < 2 || hora > 22 {
		t.Fatalf("instanteDeTeste devolveu %s, a menos de 2 h de uma virada de dia — *o helper perdeu a "+
			"propriedade que o justifica*", local.Format(time.RFC3339))
	}
	if instante.Sub(time.Now().UTC()) < 20*time.Hour {
		t.Fatalf("instanteDeTeste devolveu %s, a menos de 20 h de agora — *o teste deixaria de ser independente do "+
			"relógio da máquina*", instante.Format(time.RFC3339))
	}
	return instante
}
