package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Razão financeiro e conciliação (estágio 30 — Finance & Operations).
//
// O que já existia: `CompanyBudget` (limite mensal, limiar de aprovação) e
// `RecordSpendRequestWithIdempotency`, que cobre **gasto sob política**. O que
// faltava é o outro lado da operação financeira: **registrar o fato financeiro**
// (receita e despesa), **conciliar** contra uma referência externa e **medir o que
// não conciliou**. Sem isso não há fluxo de caixa nem conciliação — dois itens
// explícitos do estágio.
//
// DECISÕES DELIBERADAS:
//   - **Idempotência por chave, por empresa**: repetir a mesma operação (retry de
//     webhook, reenvio de requisição) devolve o lançamento **original** em vez de
//     cobrar duas vezes. A verificação e a inserção acontecem sob o mesmo lock, então
//     vale também para requisições **concorrentes**;
//   - **O razão NÃO mexe no orçamento**: `SpentCents` é política de gasto, já
//     incrementada por `RecordSpendRequest`; somar de novo aqui **duplicaria** o
//     consumo do limite. São eixos diferentes e o código diz isso;
//   - **Conciliar é idempotente com a mesma referência** e **conflito** com outra:
//     repetir a conciliação não é erro, mas dizer que o mesmo lançamento foi
//     conciliado com dois extratos diferentes precisa ser recusado;
//   - **Valores em centavos e positivos**: sinal vem do tipo (`income`/`expense`).
//     Valor zero ou negativo é erro de entrada, não "despesa negativa".

const (
	// LedgerKindIncome é entrada de dinheiro; LedgerKindExpense é saída.
	LedgerKindIncome  = "income"
	LedgerKindExpense = "expense"

	// LedgerDefaultCurrency é usado quando a moeda não é informada.
	LedgerDefaultCurrency = "BRL"
)

var (
	ErrLedgerEntryInvalid      = errors.New("ledger entry invalid")
	ErrLedgerEntryNotFound     = errors.New("ledger entry not found")
	ErrLedgerAlreadyReconciled = errors.New("ledger entry already reconciled to a different reference")
)

// LedgerEntry é um lançamento financeiro imutável.
type LedgerEntry struct {
	ID           string     `json:"id"`
	CompanyID    string     `json:"company_id"`
	Kind         string     `json:"kind"`
	Category     string     `json:"category,omitempty"`
	AmountCents  int64      `json:"amount_cents"`
	Currency     string     `json:"currency"`
	Description  string     `json:"description,omitempty"`
	Reference    string     `json:"reference,omitempty"`
	Idempotency  string     `json:"idempotency_key,omitempty"`
	OccurredAt   time.Time  `json:"occurred_at"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by,omitempty"`
	ReconciledAt *time.Time `json:"reconciled_at,omitempty"`
	// ReconciledReference é a origem externa (linha de extrato, nota, pedido).
	ReconciledReference string `json:"reconciled_reference,omitempty"`
}

// LedgerSummary é o indicador de fluxo de caixa e conciliação do período.
type LedgerSummary struct {
	CompanyID              string    `json:"company_id"`
	Currency               string    `json:"currency"`
	From                   time.Time `json:"from,omitempty"`
	To                     time.Time `json:"to,omitempty"`
	IncomeCents            int64     `json:"income_cents"`
	ExpenseCents           int64     `json:"expense_cents"`
	NetCents               int64     `json:"net_cents"`
	EntryCount             int       `json:"entry_count"`
	ReconciledCount        int       `json:"reconciled_count"`
	UnreconciledCount      int       `json:"unreconciled_count"`
	UnreconciledIncome     int64     `json:"unreconciled_income_cents"`
	UnreconciledExpense    int64     `json:"unreconciled_expense_cents"`
	ReconciliationCoverage float64   `json:"reconciliation_coverage_pct"`
}

func normalizeLedgerKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case LedgerKindIncome, "receita", "entrada", "in":
		return LedgerKindIncome
	case LedgerKindExpense, "despesa", "saida", "saída", "out":
		return LedgerKindExpense
	default:
		return ""
	}
}

func validateLedgerEntry(entry LedgerEntry) error {
	if strings.TrimSpace(entry.CompanyID) == "" {
		return fmt.Errorf("%w: company is required", ErrLedgerEntryInvalid)
	}
	if normalizeLedgerKind(entry.Kind) == "" {
		return fmt.Errorf("%w: kind must be income or expense", ErrLedgerEntryInvalid)
	}
	if entry.AmountCents <= 0 {
		return fmt.Errorf("%w: amount must be positive (sign comes from the kind)", ErrLedgerEntryInvalid)
	}
	return nil
}

func (s *CompanyStore) ledgerPath(companyID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, "ledger", sanitizeIDComponent(companyID)+".json")
}

// loadLedgerLocked carrega o razão da empresa uma única vez.
func (s *CompanyStore) loadLedgerLocked(companyID string) error {
	if s.ledgerLoaded == nil {
		s.ledgerLoaded = map[string]bool{}
	}
	// O mapa precisa existir ANTES de qualquer leitura: sem isto, a primeira
	// gravação numa empresa sem arquivo ainda escreveria num mapa nil e o
	// processo cairia (achado pelo teste de concorrência).
	if s.ledger == nil {
		s.ledger = map[string][]LedgerEntry{}
	}
	if s.ledgerLoaded[companyID] {
		return nil
	}
	s.ledgerLoaded[companyID] = true
	path := s.ledgerPath(companyID)
	if path == "" {
		return nil
	}
	var stored []LedgerEntry
	if err := readJSON(path, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	s.ledger[companyID] = stored
	return nil
}

func (s *CompanyStore) persistLedgerLocked(companyID string) error {
	path := s.ledgerPath(companyID)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create ledger dir: %w", err)
	}
	return writeJSONAtomic(path, s.ledger[companyID])
}

// RecordLedgerEntry registra o lançamento. Devolve `created=false` quando a chave
// de idempotência já existia — nesse caso o lançamento ORIGINAL é devolvido.
func (s *CompanyStore) RecordLedgerEntry(entry LedgerEntry, now time.Time) (LedgerEntry, bool, error) {
	entry.CompanyID = strings.TrimSpace(entry.CompanyID)
	entry.Idempotency = strings.TrimSpace(entry.Idempotency)
	entry.Kind = normalizeLedgerKind(entry.Kind)
	if err := validateLedgerEntry(entry); err != nil {
		return LedgerEntry{}, false, err
	}
	if strings.TrimSpace(entry.Currency) == "" {
		entry.Currency = LedgerDefaultCurrency
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = now
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	company, err := s.companyLocked(entry.CompanyID)
	if err != nil {
		return LedgerEntry{}, false, err
	}
	if err := s.loadLedgerLocked(entry.CompanyID); err != nil {
		return LedgerEntry{}, false, err
	}
	// IDEMPOTÊNCIA sob o mesmo lock: duas requisições concorrentes com a mesma
	// chave não podem gerar dois lançamentos.
	if entry.Idempotency != "" {
		for _, existing := range s.ledger[entry.CompanyID] {
			if existing.Idempotency == entry.Idempotency {
				return existing, false, nil
			}
		}
	}
	if entry.ID == "" {
		entry.ID = "led_" + uuid.NewString()[:12]
	}
	entry.CreatedAt = now
	entry.CompanyID = company.ID
	s.ledger[entry.CompanyID] = append(s.ledger[entry.CompanyID], entry)
	if err := s.persistLedgerLocked(entry.CompanyID); err != nil {
		// Sem gravação não há lançamento: desfaz para não mentir em memória.
		s.ledger[entry.CompanyID] = s.ledger[entry.CompanyID][:len(s.ledger[entry.CompanyID])-1]
		return LedgerEntry{}, false, err
	}
	return entry, true, nil
}

// ListLedgerEntries devolve os lançamentos do período (limites zero = sem limite).
func (s *CompanyStore) ListLedgerEntries(companyID string, from, to time.Time) ([]LedgerEntry, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("%w: company is required", ErrLedgerEntryInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.companyLocked(companyID); err != nil {
		return nil, err
	}
	if err := s.loadLedgerLocked(companyID); err != nil {
		return nil, err
	}
	entries := make([]LedgerEntry, 0, len(s.ledger[companyID]))
	for _, entry := range s.ledger[companyID] {
		if !from.IsZero() && entry.OccurredAt.Before(from) {
			continue
		}
		if !to.IsZero() && entry.OccurredAt.After(to) {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].OccurredAt.After(entries[j].OccurredAt) })
	return entries, nil
}

// ReconcileLedgerEntry concilia o lançamento com uma referência externa.
// `changed=false` significa que já estava conciliado com a MESMA referência.
func (s *CompanyStore) ReconcileLedgerEntry(companyID, entryID, reference string, now time.Time) (LedgerEntry, bool, error) {
	companyID = strings.TrimSpace(companyID)
	entryID = strings.TrimSpace(entryID)
	reference = strings.TrimSpace(reference)
	if companyID == "" || entryID == "" {
		return LedgerEntry{}, false, fmt.Errorf("%w: company and entry are required", ErrLedgerEntryInvalid)
	}
	if reference == "" {
		return LedgerEntry{}, false, fmt.Errorf("%w: reconciliation requires an external reference", ErrLedgerEntryInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.companyLocked(companyID); err != nil {
		return LedgerEntry{}, false, err
	}
	if err := s.loadLedgerLocked(companyID); err != nil {
		return LedgerEntry{}, false, err
	}
	entries := s.ledger[companyID]
	for index := range entries {
		if entries[index].ID != entryID {
			continue
		}
		if entries[index].ReconciledAt != nil {
			if entries[index].ReconciledReference == reference {
				return entries[index], false, nil
			}
			return LedgerEntry{}, false, fmt.Errorf("%w: entry %s", ErrLedgerAlreadyReconciled, entryID)
		}
		reconciledAt := now
		entries[index].ReconciledAt = &reconciledAt
		entries[index].ReconciledReference = reference
		if err := s.persistLedgerLocked(companyID); err != nil {
			entries[index].ReconciledAt = nil
			entries[index].ReconciledReference = ""
			return LedgerEntry{}, false, err
		}
		return entries[index], true, nil
	}
	return LedgerEntry{}, false, fmt.Errorf("%w: %s", ErrLedgerEntryNotFound, entryID)
}

// LedgerSummary calcula fluxo de caixa e cobertura de conciliação do período.
func (s *CompanyStore) LedgerSummary(companyID string, from, to time.Time) (LedgerSummary, error) {
	entries, err := s.ListLedgerEntries(companyID, from, to)
	if err != nil {
		return LedgerSummary{}, err
	}
	summary := LedgerSummary{CompanyID: strings.TrimSpace(companyID), From: from, To: to, Currency: LedgerDefaultCurrency}
	for _, entry := range entries {
		summary.Currency = entry.Currency
		summary.EntryCount++
		switch entry.Kind {
		case LedgerKindIncome:
			summary.IncomeCents += entry.AmountCents
		case LedgerKindExpense:
			summary.ExpenseCents += entry.AmountCents
		}
		if entry.ReconciledAt != nil {
			summary.ReconciledCount++
			continue
		}
		summary.UnreconciledCount++
		if entry.Kind == LedgerKindIncome {
			summary.UnreconciledIncome += entry.AmountCents
		} else {
			summary.UnreconciledExpense += entry.AmountCents
		}
	}
	summary.NetCents = summary.IncomeCents - summary.ExpenseCents
	if summary.EntryCount > 0 {
		summary.ReconciliationCoverage = float64(summary.ReconciledCount) / float64(summary.EntryCount) * 100
	}
	return summary, nil
}

// UnreconciledLedgerEntries devolve o que ficou sem conciliar além do prazo — a
// fila de trabalho de quem fecha o mês.
func (s *CompanyStore) UnreconciledLedgerEntries(companyID string, olderThanDays int, now time.Time) ([]LedgerEntry, error) {
	entries, err := s.ListLedgerEntries(companyID, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	cutoff := now.AddDate(0, 0, -olderThanDays)
	pending := make([]LedgerEntry, 0)
	for _, entry := range entries {
		if entry.ReconciledAt != nil {
			continue
		}
		if olderThanDays > 0 && entry.OccurredAt.After(cutoff) {
			continue
		}
		pending = append(pending, entry)
	}
	return pending, nil
}

// companyLocked resolve a empresa sem repetir a checagem de organização (a rota já
// autorizou). Devolve erro quando a empresa não existe.
func (s *CompanyStore) companyLocked(companyID string) (Company, error) {
	if company, ok := s.companies[companyID]; ok {
		return company, nil
	}
	return Company{}, fmt.Errorf("%w: %s", ErrCompanyNotFound, companyID)
}
