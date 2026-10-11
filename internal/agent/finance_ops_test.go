package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 30 — Finance & Operations ligado ao comércio.
//
// As propriedades travadas aqui:
//  1. PEDIDO NÃO PAGO NÃO É RECEITA (receita fictícia é o defeito a evitar);
//  2. a receita é lançada UMA vez por pedido (idempotente);
//  3. a receita bate com o total do pedido e carrega a moeda;
//  4. LACUNA é MEDIDA: pedido pago sem lançamento aparece no relatório;
//  5. cancelado com receita lançada gera **estorno devido**, e o lançamento NÃO é apagado;
//  6. conciliação cruzada liga pedido e extrato no mesmo lançamento, e referência
//     divergente é conflito;
//  7. contas a receber = pedidos aguardando pagamento;
//  8. isolamento por organização e recusa quando a organização não tem empresa.
func financeFixture(t *testing.T) (*CommerceStore, *CompanyStore, *FinanceOps, string) {
	t.Helper()
	root := t.TempDir()
	companies, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	now := businessFixtureClock
	company, err := companies.Create(Company{ID: "emp-1", OrganizationID: "org-a", Name: "Loja da Ana", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	commerce, err := NewCommerceStoreAt(filepath.Join(root, "commerce"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: "org-a", SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.RecordStock("org-a", "CAM-001", 10, CommerceMovementEntry, "lote 1", now); err != nil {
		t.Fatal(err)
	}
	// A empresa criada é a que recebe os lançamentos; o ID é o da empresa, e a
	// organização é a do comércio.
	_ = company
	return commerce, companies, NewFinanceOps(commerce, companies), company.ID
}

func TestFinanceDoesNotPostRevenueForUnpaidOrder(t *testing.T) {
	commerce, _, finance, _ := financeFixture(t)
	now := time.Now().UTC()
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "fin-1",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Pedido aguardando pagamento NÃO é receita: lançar criaria receita fictícia.
	if _, err := finance.PostOrderRevenue("org-a", order.ID, "ana", now); !errors.Is(err, ErrFinanceOrderNotPayable) {
		t.Fatalf("pedido não pago precisa ser recusado, veio %v", err)
	}
	report, err := finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ledger.IncomeCents != 0 {
		t.Fatalf("não pode existir receita de pedido não pago: %+v", report.Ledger)
	}
	// Depois do pagamento, a receita entra.
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentPix, Reference: "E123"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	posting, err := finance.PostOrderRevenue("org-a", order.ID, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	if !posting.Created {
		t.Fatal("o primeiro lançamento precisa ser criado")
	}
	if posting.Entry.AmountCents != 7900 || posting.Entry.Kind != LedgerKindIncome || posting.Entry.Currency != CommerceDefaultCurrency {
		t.Fatalf("o lançamento precisa refletir o pedido: %+v", posting.Entry)
	}
	if posting.Entry.Reference != order.Number {
		t.Fatalf("a referência do lançamento precisa ser o pedido: %+v", posting.Entry)
	}
}

func TestFinanceRevenueIsPostedOncePerOrder(t *testing.T) {
	commerce, companies, finance, companyID := financeFixture(t)
	now := time.Now().UTC()
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "fin-2",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 2}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "ref-2"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	first, err := finance.PostOrderRevenue("org-a", order.ID, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	// Repetir não pode duplicar receita.
	second, err := finance.PostOrderRevenue("org-a", order.ID, "bruno", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.Created {
		t.Fatal("repetir o lançamento não pode criar um segundo")
	}
	if second.Entry.ID != first.Entry.ID {
		t.Fatalf("a repetição precisa devolver o MESMO lançamento: %s vs %s", second.Entry.ID, first.Entry.ID)
	}
	entries, err := companies.ListLedgerEntries(companyID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	income := 0
	var total int64
	for _, entry := range entries {
		if entry.Kind == LedgerKindIncome {
			income++
			total += entry.AmountCents
		}
	}
	if income != 1 || total != 15800 {
		t.Fatalf("precisa haver UMA receita de 15800: %d lançamentos, total %d", income, total)
	}
	// Custo do pedido também é idempotente por pedido e categoria.
	if _, err := finance.PostOrderCost("org-a", order.ID, 1500, "frete", "transportadora-x", "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := finance.PostOrderCost("org-a", order.ID, 1500, "frete", "transportadora-x", "ana", now); err != nil {
		t.Fatal(err)
	}
	report, err := finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ledger.ExpenseCents != 1500 {
		t.Fatalf("o custo não pode duplicar: %+v", report.Ledger)
	}
	if report.Ledger.NetCents != 15800-1500 {
		t.Fatalf("saldo incorreto: %+v", report.Ledger)
	}
}

func TestFinanceMeasuresUnpostedRevenueAndRefunds(t *testing.T) {
	commerce, _, finance, companyID := financeFixture(t)
	now := time.Now().UTC()
	paid, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "fin-3",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", paid.ID, CommercePayment{Method: CommercePaymentPix, Reference: "E999"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	// Pedido pago e SEM lançamento: a lacuna precisa aparecer medida.
	report, err := finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.UnpostedRevenue != 7900 {
		t.Fatalf("a receita não lançada precisa ser medida: %+v", report)
	}
	if len(report.Divergences) != 1 || report.Divergences[0].Kind != FinanceDivergenceUnposted {
		t.Fatalf("a divergência precisa ser declarada: %+v", report.Divergences)
	}
	// Lançando a receita, a lacuna fecha mas fica a falta de conciliação.
	if _, err := finance.PostOrderRevenue("org-a", paid.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	report, err = finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.UnpostedRevenue != 0 {
		t.Fatalf("após lançar, não há lacuna: %+v", report)
	}
	if len(report.Divergences) != 1 || report.Divergences[0].Kind != FinanceDivergenceUnreconciled {
		t.Fatalf("falta de conciliação precisa aparecer: %+v", report.Divergences)
	}
	// Conciliando com o extrato, as divergências zeram.
	if _, err := finance.ReconcileOrderWithStatement("org-a", paid.ID, "extrato-2026-10", "ana", now); err != nil {
		t.Fatal(err)
	}
	report, err = finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Divergences) != 0 {
		t.Fatalf("conciliado não pode ter divergência: %+v", report.Divergences)
	}
	if report.Ledger.ReconciledCount != 1 {
		t.Fatalf("a conciliação precisa contar no razão: %+v", report.Ledger)
	}
	// Referência divergente é conflito, não sobrescrita.
	if _, err := finance.ReconcileOrderWithStatement("org-a", paid.ID, "extrato-2026-11", "ana", now); !errors.Is(err, ErrLedgerAlreadyReconciled) {
		t.Fatalf("conciliação divergente precisa ser conflito, veio %v", err)
	}

	// Pedido pago, lançado e depois CANCELADO: existe estorno DEVIDO, e o lançamento
	// permanece — apagar esconderia a devolução.
	cancelled, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "fin-4",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", cancelled.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "ref-4"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := finance.PostOrderRevenue("org-a", cancelled.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.Cancel("org-a", cancelled.ID, "cliente desistiu", now); err != nil {
		t.Fatal(err)
	}
	report, err = finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.RefundDue != 7900 {
		t.Fatalf("o estorno devido precisa ser medido: %+v", report)
	}
	foundRefund := false
	for _, divergence := range report.Divergences {
		if divergence.Kind == FinanceDivergenceRefund {
			foundRefund = true
		}
	}
	if !foundRefund {
		t.Fatalf("a divergência de estorno precisa aparecer: %+v", report.Divergences)
	}
	// O lançamento continua no razão.
	entries, err := finance.companies.ListLedgerEntries(companyID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("os dois lançamentos precisam continuar: %d", len(entries))
	}
}

func TestFinanceReportsReceivablesAndIsScopedByOrganization(t *testing.T) {
	commerce, _, finance, _ := financeFixture(t)
	now := time.Now().UTC()
	// Dois pedidos aguardando pagamento: contas a receber.
	for _, key := range []string{"recv-1", "recv-2"} {
		if _, err := commerce.CreateOrder(CommerceOrderRequest{
			OrganizationID: "org-a", IdempotencyKey: key,
			Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
		}, now); err != nil {
			t.Fatal(err)
		}
	}
	report, err := finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReceivableCents != 15800 || len(report.Receivables) != 2 || report.OrdersPending != 2 {
		t.Fatalf("contas a receber incorretas: %+v", report)
	}
	if report.Ledger.IncomeCents != 0 {
		t.Fatalf("nada entrou ainda: %+v", report.Ledger)
	}
	// Organização SEM empresa: recusa explícita, em vez de lançar no vazio.
	if _, err := finance.PostOrderRevenue("org-b", "ord_inexistente", "ana", now); !errors.Is(err, ErrFinanceCompanyMissing) && !errors.Is(err, ErrFinanceOrderMissing) {
		t.Fatalf("organização sem empresa/pedido precisa falhar de forma explícita, veio %v", err)
	}
	// Pedido de outra organização não é visível pelo financeiro da org-a.
	if _, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "recv-3",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := finance.orderForCompany("org-b", commerce.Orders("org-a")[0].ID); err == nil {
		t.Fatal("pedido de outra organização não pode ser usado no financeiro")
	}
}

// O comércio precisa sobreviver ao reinício: pedidos, estoque, catálogo e a chave de
// idempotência (para um reenvio após o reinício não criar um segundo pedido).
func TestCommercePersistsAcrossRestart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "commerce")
	now := time.Now().UTC()
	commerce, err := NewCommerceStoreAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: "org-a", SKU: "CAN-002", Name: "Caneca", PriceCents: 3500, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.RecordStock("org-a", "CAN-002", 5, CommerceMovementEntry, "lote", now); err != nil {
		t.Fatal(err)
	}
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "persist-1",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 2}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "ref-p"}, "ana", now); err != nil {
		t.Fatal(err)
	}

	restarted, err := NewCommerceStoreAt(root)
	if err != nil {
		t.Fatal(err)
	}
	// Pedido recuperado com o estado correto.
	reloaded, err := restarted.Order("org-a", order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != CommerceOrderPaid || reloaded.TotalCents != 7000 || reloaded.Payment == nil {
		t.Fatalf("o pedido precisa sobreviver ao reinício: %+v", reloaded)
	}
	// Estoque recuperado.
	balance := restarted.Stock("org-a", "CAN-002")
	if balance.OnHand != 5 || balance.Reserved != 2 || balance.Available != 3 {
		t.Fatalf("o estoque precisa sobreviver ao reinício: %+v", balance)
	}
	// A chave de idempotência vale depois do reinício: reenviar devolve o MESMO pedido.
	repeat, err := restarted.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "persist-1",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 2}},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if repeat.ID != order.ID || !repeat.Duplicated {
		t.Fatalf("a idempotência precisa valer após o reinício: %+v", repeat)
	}
	if orders := restarted.Orders("org-a"); len(orders) != 1 {
		t.Fatalf("um pedido apenas após o reinício: %d", len(orders))
	}
	// E o catálogo também.
	if products := restarted.Products("org-a"); len(products) != 1 || products[0].SKU != "CAN-002" {
		t.Fatalf("o catálogo precisa sobreviver: %+v", products)
	}
}
