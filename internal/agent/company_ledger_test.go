package agent

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// EstÃ¡gio 30: razÃ£o financeiro e conciliaÃ§Ã£o.
//
// As propriedades travadas aqui:
//  1. IDEMPOTÃŠNCIA vale tambÃ©m sob CONCORRÃŠNCIA: a mesma operaÃ§Ã£o reenviada nÃ£o
//     pode virar duas cobranÃ§as;
//  2. entrada invÃ¡lida Ã© recusada (valor zero/negativo, tipo desconhecido, empresa
//     inexistente) em vez de virar lanÃ§amento torto;
//  3. conciliar Ã© idempotente com a MESMA referÃªncia e CONFLITO com outra;
//  4. o resumo mostra fluxo de caixa e o que AINDA NÃƒO foi conciliado;
//  5. o razÃ£o SOBREVIVE ao reinÃcio;
//  6. o razÃ£o NÃƒO duplica o consumo do orÃ§amento: `SpentCents` Ã© polÃtica de gasto,
//     jÃ¡ contabilizada por `RecordSpendRequest`.
func ledgerStore(t *testing.T) (*CompanyStore, Company) {
	t.Helper()
	store, err := NewCompanyStore(filepath.Join(t.TempDir(), "companies"))
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{
		Name: "Loja de teste", OrganizationID: "org-a",
		Budget: CompanyBudget{MonthlyLimitCents: 500000, Currency: "BRL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, company
}

func TestLedgerIdempotencyHoldsUnderConcurrency(t *testing.T) {
	store, company := ledgerStore(t)
	now := time.Now().UTC()
	const workers = 12

	var wg sync.WaitGroup
	results := make([]bool, workers)
	errorsSeen := make([]error, workers)
	for index := range workers {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			_, created, err := store.RecordLedgerEntry(LedgerEntry{
				CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 15000,
				Category: "infra", Description: "assinatura mensal", Idempotency: "pix-2026-10-001",
			}, now)
			results[slot] = created
			errorsSeen[slot] = err
		}(index)
	}
	wg.Wait()

	for slot, err := range errorsSeen {
		if err != nil {
			t.Fatalf("worker %d falhou: %v", slot, err)
		}
	}
	created := 0
	for _, wasCreated := range results {
		if wasCreated {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("a mesma chave de idempotÃªncia precisa criar UM lanÃ§amento, criou %d", created)
	}
	entries, err := store.ListLedgerEntries(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("o razÃ£o precisa ter um lanÃ§amento: %d", len(entries))
	}
	if entries[0].AmountCents != 15000 {
		t.Fatalf("o valor precisa ser o original: %+v", entries[0])
	}
}

func TestLedgerRejectsInvalidEntries(t *testing.T) {
	store, company := ledgerStore(t)
	now := time.Now().UTC()
	casos := []struct {
		nome  string
		entry LedgerEntry
	}{
		{"sem empresa", LedgerEntry{Kind: LedgerKindIncome, AmountCents: 100}},
		{"tipo desconhecido", LedgerEntry{CompanyID: company.ID, Kind: "transferencia", AmountCents: 100}},
		{"valor zero", LedgerEntry{CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 0}},
		{"valor negativo", LedgerEntry{CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: -500}},
		{"empresa inexistente", LedgerEntry{CompanyID: "nao-existe", Kind: LedgerKindIncome, AmountCents: 100}},
	}
	for _, caso := range casos {
		if _, _, err := store.RecordLedgerEntry(caso.entry, now); err == nil {
			t.Fatalf("caso %q precisava ser recusado", caso.nome)
		}
	}
	// Nada foi gravado.
	entries, err := store.ListLedgerEntries(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entrada invÃ¡lida nÃ£o pode virar lanÃ§amento: %+v", entries)
	}
}

func TestLedgerReconciliationIsIdempotentWithSameReferenceAndConflictsWithOther(t *testing.T) {
	store, company := ledgerStore(t)
	now := time.Now().UTC()
	entry, _, err := store.RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 250000,
		Description: "pedido 1042", Reference: "order-1042",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	reconciled, changed, err := store.ReconcileLedgerEntry(company.ID, entry.ID, "extrato-2026-10-linha-7", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !changed || reconciled.ReconciledAt == nil {
		t.Fatalf("a primeira conciliaÃ§Ã£o precisa marcar data: %+v", reconciled)
	}
	// Repetir com a MESMA referÃªncia nÃ£o Ã© erro nem muda nada.
	_, changedAgain, err := store.ReconcileLedgerEntry(company.ID, entry.ID, "extrato-2026-10-linha-7", now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if changedAgain {
		t.Fatal("repetir a mesma conciliaÃ§Ã£o nÃ£o pode alterar o lanÃ§amento")
	}
	// ReferÃªncia DIFERENTE Ã© conflito: o mesmo lanÃ§amento nÃ£o pode ter sido
	// conciliado com duas origens distintas.
	if _, _, err := store.ReconcileLedgerEntry(company.ID, entry.ID, "outro-extrato", now.Add(3*time.Hour)); !errors.Is(err, ErrLedgerAlreadyReconciled) {
		t.Fatalf("referÃªncia diferente precisa ser conflito, veio %v", err)
	}
	// Exigir referÃªncia externa.
	if _, _, err := store.ReconcileLedgerEntry(company.ID, entry.ID, "  ", now); err == nil {
		t.Fatal("conciliaÃ§Ã£o sem referÃªncia precisa falhar")
	}
	// LanÃ§amento inexistente.
	if _, _, err := store.ReconcileLedgerEntry(company.ID, "led_inexistente", "extrato", now); !errors.Is(err, ErrLedgerEntryNotFound) {
		t.Fatalf("lanÃ§amento inexistente precisa falhar, veio %v", err)
	}
}

func TestLedgerSummaryShowsCashFlowAndUnreconciledWork(t *testing.T) {
	store, company := ledgerStore(t)
	now := time.Now().UTC()
	mustRecord := func(entry LedgerEntry) LedgerEntry {
		t.Helper()
		recorded, _, err := store.RecordLedgerEntry(entry, now)
		if err != nil {
			t.Fatal(err)
		}
		return recorded
	}
	mustRecord(LedgerEntry{CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 300000, Description: "venda"})
	mustRecord(LedgerEntry{CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 100000, Description: "venda 2"})
	mustRecord(LedgerEntry{CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 50000, Description: "frete"})
	expense := mustRecord(LedgerEntry{CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 20000, Description: "embalagem"})
	if _, _, err := store.ReconcileLedgerEntry(company.ID, expense.ID, "nota-77", now); err != nil {
		t.Fatal(err)
	}

	summary, err := store.LedgerSummary(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.IncomeCents != 400000 || summary.ExpenseCents != 70000 {
		t.Fatalf("fluxo de caixa incorreto: %+v", summary)
	}
	if summary.NetCents != 330000 {
		t.Fatalf("lÃquido incorreto: %+v", summary)
	}
	if summary.EntryCount != 4 || summary.ReconciledCount != 1 || summary.UnreconciledCount != 3 {
		t.Fatalf("contagem de conciliaÃ§Ã£o incorreta: %+v", summary)
	}
	if summary.UnreconciledIncome != 400000 || summary.UnreconciledExpense != 50000 {
		t.Fatalf("pendÃªncias de conciliaÃ§Ã£o incorretas: %+v", summary)
	}
	if summary.ReconciliationCoverage != 25 {
		t.Fatalf("cobertura precisa ser 25%%, veio %v", summary.ReconciliationCoverage)
	}

	// Fila de conciliaÃ§Ã£o atrasada: o lanÃ§amento antigo e nÃ£o conciliado aparece.
	old, _, err := store.RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 9000,
		OccurredAt: now.AddDate(0, 0, -40), Description: "cobranÃ§a antiga",
	}, now.AddDate(0, 0, -40))
	if err != nil {
		t.Fatal(err)
	}
	pending, err := store.UnreconciledLedgerEntries(company.ID, 30, now)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range pending {
		if entry.ID == old.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("o lanÃ§amento antigo sem conciliaÃ§Ã£o precisa entrar na fila: %+v", pending)
	}

	// Recorte por perÃodo: sÃ³ o intervalo pedido entra no resumo.
	from := now.AddDate(0, 0, -1)
	period, err := store.LedgerSummary(company.ID, from, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if period.EntryCount != 4 {
		t.Fatalf("o recorte por perÃodo precisa excluir o lanÃ§amento antigo: %+v", period)
	}
}

func TestLedgerSurvivesRestartAndDoesNotDoubleBudget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "companies")
	store, err := NewCompanyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{
		Name: "Loja persistente", OrganizationID: "org-a",
		Budget: CompanyBudget{MonthlyLimitCents: 100000, Currency: "BRL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, _, err := store.RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 12000,
		Category: "marketing", Idempotency: "ad-2026-10",
	}, now); err != nil {
		t.Fatal(err)
	}
	// O razÃ£o registra o fato financeiro; o ORÃ‡AMENTO tem o prÃ³prio caminho
	// (`RecordSpendRequest`) e nÃ£o pode ser incrementado por tabela.
	reloadedCompany, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedCompany.Budget.SpentCents != 0 {
		t.Fatalf("o razÃ£o nÃ£o pode mexer no orÃ§amento: %d", reloadedCompany.Budget.SpentCents)
	}

	restarted, err := NewCompanyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := restarted.ListLedgerEntries(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("o razÃ£o precisa sobreviver ao reinÃcio: %d", len(entries))
	}
	if entries[0].Idempotency != "ad-2026-10" || entries[0].AmountCents != 12000 {
		t.Fatalf("o lanÃ§amento precisa vir Ãntegro: %+v", entries[0])
	}
	// E a idempotÃªncia continua valendo depois do reinÃcio: repetir a chave nÃ£o
	// cria um segundo lanÃ§amento no arquivo recÃ©m-carregado.
	if _, created, err := restarted.RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 12000,
		Category: "marketing", Idempotency: "ad-2026-10",
	}, now.Add(time.Hour)); err != nil || created {
		t.Fatalf("idempotÃªncia precisa valer apÃ³s o reinÃcio (created=%v err=%v)", created, err)
	}
	afterRestart, err := restarted.ListLedgerEntries(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(afterRestart) != 1 {
		t.Fatalf("nÃ£o pode haver lanÃ§amento duplicado apÃ³s o reinÃcio: %d", len(afterRestart))
	}
}

func TestLedgerIsScopedPerCompany(t *testing.T) {
	store, first := ledgerStore(t)
	second, err := store.Create(Company{
		Name: "Outra loja", OrganizationID: "org-b",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, _, err := store.RecordLedgerEntry(LedgerEntry{
		CompanyID: first.ID, Kind: LedgerKindIncome, AmountCents: 1000, Description: "da primeira",
	}, now); err != nil {
		t.Fatal(err)
	}
	entries, err := store.ListLedgerEntries(second.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("o razÃ£o de uma empresa nÃ£o pode aparecer em outra: %+v", entries)
	}
	// A mesma chave de idempotÃªncia em empresas diferentes Ã© operaÃ§Ã£o diferente.
	key := "compartilhada-001"
	if _, _, err := store.RecordLedgerEntry(LedgerEntry{CompanyID: first.ID, Kind: LedgerKindIncome, AmountCents: 500, Idempotency: key}, now); err != nil {
		t.Fatal(err)
	}
	if _, created, err := store.RecordLedgerEntry(LedgerEntry{CompanyID: second.ID, Kind: LedgerKindIncome, AmountCents: 700, Idempotency: key}, now); err != nil || !created {
		t.Fatalf("a chave Ã© por empresa; na segunda precisa criar (created=%v err=%v)", created, err)
	}
	if strings.TrimSpace(second.ID) == strings.TrimSpace(first.ID) {
		t.Fatal("empresas distintas precisam de IDs distintos")
	}
}
