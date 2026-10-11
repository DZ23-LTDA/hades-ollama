package agent

import (
	"testing"
	"time"
)

// Estágio 32 — série temporal do BI. Propriedades travadas:
//
//  1. UMA AMOSTRA POR JANELA (hora): registrar de novo na mesma hora NÃO cria outra amostra;
//  2. SEM DUAS AMOSTRAS NÃO HÁ TENDÊNCIA: `SemBase` é declarado e **nenhum** delta é emitido —
//     delta zero significaria "não mudou", que é diferente de "não há com o que comparar";
//  3. RETENÇÃO COM LIMITE: o teto é aplicado e o descarte é relatado;
//  4. ISOLAMENTO: a série de uma organização não aparece na outra.
func TestBusinessTrendRequiresTwoSamplesAndComputesRealDeltas(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	organization := LocalOrganizationID

	// Primeira amostra: existe série, mas NÃO existe tendência.
	if _, created, trend, err := runtime.BusinessSnapshot(organization, now); err != nil || !created {
		t.Fatalf("primeira amostra precisa ser criação: created=%v err=%v", created, err)
	} else if !trend.SemBase || trend.Motivo == "" || len(trend.Deltas) != 0 {
		t.Fatalf("com uma amostra a tendência precisa declarar ausência de base e NÃO emitir delta: %+v", trend)
	} else if trend.Snapshots != 1 {
		t.Fatalf("amostras: %d", trend.Snapshots)
	}

	// Move o negócio: mais um pedido pago e mais receita no livro.
	order, err := runtime.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: organization, IdempotencyKey: "trend-1",
		Items: []CommerceOrderItem{{SKU: "BI-1", Quantity: 1}},
	}, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().ConfirmPayment(organization, order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "trend-pay"}, "ana", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	company := runtime.CompanyStore().List(organization)[0]
	if _, _, err := runtime.CompanyStore().RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 5000, Currency: "BRL",
		Description: "receita nova", OccurredAt: now.Add(time.Hour),
	}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	// Segunda amostra, em OUTRA janela.
	_, created, trend, err := runtime.BusinessSnapshot(organization, now.Add(time.Hour))
	if err != nil || !created {
		t.Fatalf("segunda amostra precisa ser criação: created=%v err=%v", created, err)
	}
	if trend.SemBase || trend.Motivo != "" {
		t.Fatalf("com duas amostras a tendência precisa existir: %+v", trend)
	}
	if trend.Snapshots != 2 || trend.First == nil || trend.Last == nil || trend.Interval == "" {
		t.Fatalf("série incompleta: %+v", trend)
	}
	deltas := map[string]int64{}
	for _, delta := range trend.Deltas {
		if delta.Delta != delta.To-delta.From {
			t.Fatalf("delta inconsistente: %+v", delta)
		}
		deltas[delta.Metric] = delta.Delta
	}
	// Os números REAIS: um pedido e uma receita a mais, e a receita de comércio do pedido pago.
	if deltas["pedidos"] != 1 {
		t.Fatalf("delta de pedidos precisa ser 1: %+v", deltas)
	}
	if deltas["receita_financeira_cents"] != 5000 {
		t.Fatalf("delta de receita financeira precisa ser 5000: %+v", deltas)
	}
	if deltas["receita_cents"] <= 0 {
		t.Fatalf("delta de receita de comércio precisa ser positivo: %+v", deltas)
	}
}

func TestBusinessSnapshotIsIdempotentPerWindow(t *testing.T) {
	runtime, _ := businessStatusFixture(t)
	organization := LocalOrganizationID

	// RELÓGIO FIXO — e o motivo é um defeito real que apareceu no CI: usando o relógio de parede,
	// `now.Add(time.Minute)` pode CRUZAR A VIRADA DE HORA, e aí a segunda chamada cria legitimamente
	// uma nova janela. O teste falhava sozinho (CI, 13:59:57 UTC) sem que houvesse nada errado no
	// produto. O teste de ROTA já tinha esse cuidado (E-134); este tinha ficado refém do relógio real.
	// Agora o instante vem do fixture, que usa `businessFixtureClock`.
	base := businessFixtureClock

	first, created, _, err := runtime.BusinessSnapshot(organization, base)
	if err != nil || !created {
		t.Fatalf("primeira: created=%v err=%v", created, err)
	}
	// MESMA janela: não cria outra amostra e devolve a existente.
	again, createdAgain, trend, err := runtime.BusinessSnapshot(organization, base.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if createdAgain {
		t.Fatal("registrar na mesma hora não pode criar outra amostra")
	}
	if again.ID != first.ID {
		t.Fatalf("a amostra reaproveitada precisa ser a mesma: %q vs %q", again.ID, first.ID)
	}
	if trend.Snapshots != 1 {
		t.Fatalf("a série precisa continuar com 1 amostra: %d", trend.Snapshots)
	}
	// A JANELA É DE HORA — e isto agora é exercitado DE PROPÓSITO, em vez de depender de onde o relógio
	// estava: 10:59 ainda é a mesma janela de 10:00...
	if _, createdLate, _, err := runtime.BusinessSnapshot(organization, time.Date(2026, 10, 12, 10, 59, 30, 0, time.UTC)); err != nil || createdLate {
		t.Fatalf("10:59 pertence à janela de 10:00: created=%v err=%v", createdLate, err)
	}
	// ...e 11:00 é a janela seguinte, que cria — comportamento deliberado, não acidente.
	if _, createdNext, _, err := runtime.BusinessSnapshot(organization, time.Date(2026, 10, 12, 11, 0, 30, 0, time.UTC)); err != nil || !createdNext {
		t.Fatalf("nova janela precisa criar: created=%v err=%v", createdNext, err)
	}
	// E registrar de novo DENTRO da hora 11 continua idempotente.
	if _, createdRepeat, _, err := runtime.BusinessSnapshot(organization, time.Date(2026, 10, 12, 11, 45, 0, 0, time.UTC)); err != nil || createdRepeat {
		t.Fatalf("repetir na hora 11 não pode criar: created=%v err=%v", createdRepeat, err)
	}
}

func TestBusinessSnapshotRetentionLimitsAndReportsDiscards(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	organization := LocalOrganizationID
	runtime.businessSnapshots.MaxSnapshots = 3

	for hour := range 5 {
		if _, _, _, err := runtime.BusinessSnapshot(organization, now.Add(time.Duration(hour)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	trend, err := runtime.BusinessTrend(organization, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// RETENÇÃO: o teto vale e as duas mais antigas foram descartadas.
	if trend.Retention != 3 {
		t.Fatalf("retenção declarada: %d", trend.Retention)
	}
	if len(trend.History) != 3 || trend.Snapshots != 3 {
		t.Fatalf("a série precisa respeitar o teto de 3: %d", len(trend.History))
	}
	if trend.History[0].Window != SnapshotWindow(now.Add(2*time.Hour)) {
		t.Fatalf("a amostra mais antiga precisa ser a terceira janela: %s", trend.History[0].Window)
	}
	// Recorte por período relata o que ficou fora.
	trimmed, err := runtime.BusinessTrend(organization, now.Add(4*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.Snapshots != 1 || trimmed.Discarded != 2 {
		t.Fatalf("recorte precisa relatar o descarte: snapshots=%d descartadas=%d", trimmed.Snapshots, trimmed.Discarded)
	}
	// E com uma única amostra no recorte, não há tendência.
	if !trimmed.SemBase {
		t.Fatal("recorte com uma amostra precisa declarar ausência de base")
	}
}

func TestBusinessSnapshotIsScopedByOrganization(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	if _, _, _, err := runtime.BusinessSnapshot(LocalOrganizationID, now); err != nil {
		t.Fatal(err)
	}
	// Outra organização não herda a série alheia.
	other, err := runtime.BusinessTrend("org_bi_outra", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if other.Snapshots != 0 || len(other.History) != 0 || !other.SemBase {
		t.Fatalf("a série não pode vazar entre organizações: %+v", other)
	}
	// E o relatório que a amostra guardou é o da PRÓPRIA organização.
	own, err := runtime.BusinessTrend(LocalOrganizationID, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(own.History) != 1 || own.History[0].Report.OrganizationID != LocalOrganizationID {
		t.Fatalf("amostra precisa guardar o relatório da organização dona: %+v", own.History)
	}
	// Organização ausente é erro declarado.
	if _, err := runtime.BusinessTrend("  ", time.Time{}); err == nil {
		t.Fatal("organização ausente precisa ser erro")
	}
}
