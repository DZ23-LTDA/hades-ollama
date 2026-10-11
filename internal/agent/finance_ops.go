package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Finance & Operations (estágio 30): liga o comércio ao LIVRO FINANCEIRO existente.
//
// O estágio exige lançamentos, conciliação e autorização, sem pagamento de alto
// impacto sem autorização. Este módulo **não cria um segundo razão**: ele deriva
// lançamentos do comércio no livro que já existe (`company_ledger.go`), com decisões
// que valem registrar:
//
//   - **PEDIDO NÃO PAGO NÃO É RECEITA**: lançar receita de pedido aguardando pagamento
//     criaria receita fictícia. A receita só entra quando o pagamento existe;
//   - **IDEMPOTÊNCIA POR PEDIDO**: postar duas vezes o mesmo pedido **não** duplica
//     receita — a chave é o próprio pedido;
//   - **LACUNA É MEDIDA, NÃO ESCONDIDA**: pedido pago sem lançamento aparece em
//     `UnpostedRevenueCents`. É o detector de receita que "sumiu" entre o comércio e o
//     financeiro;
//   - **DIVERGÊNCIA É DECLARADA**: pedido cancelado que já tem receita lançada vira
//     **estorno devido** (`RefundDueCents`) — o lançamento **não** é apagado, porque
//     dinheiro que entrou e depois foi devolvido tem história, e apagar esconderia a
//     devolução;
//   - **CONCILIAÇÃO CRUZADA**: a referência do extrato é registrada no **lançamento**,
//     ligando comércio e banco pelo mesmo documento;
//   - **NENHUM DINHEIRO SE MOVE**: não há pagamento, transferência nem estorno
//     automático — apenas registro e sinalização. Mover dinheiro exige autorização.

var (
	ErrFinanceOrderNotPayable = errors.New("order is not paid; a pending order is not revenue")
	ErrFinanceOrderMissing    = errors.New("finance: order not found")
	ErrFinanceCompanyMissing  = errors.New("finance: company not found")
)

// FinanceOrderPosting registra a receita de um pedido no livro da empresa.
type FinanceOrderPosting struct {
	OrderID     string      `json:"order_id"`
	OrderNumber string      `json:"order_number"`
	CompanyID   string      `json:"company_id"`
	Entry       LedgerEntry `json:"entry"`
	// Created false significa que a receita JÁ existia: a operação foi idempotente.
	Created bool `json:"created"`
}

// FinanceReceivable é uma conta a receber: pedido ainda não pago.
type FinanceReceivable struct {
	OrderID     string    `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	ContactID   string    `json:"contact_id,omitempty"`
	TotalCents  int64     `json:"total_cents"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

// FinanceDivergence é uma inconsistência MEDIDA entre comércio e financeiro.
type FinanceDivergence struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	Kind        string `json:"kind"`
	AmountCents int64  `json:"amount_cents"`
	Detail      string `json:"detail"`
}

const (
	// FinanceDivergenceUnposted: pedido pago e sem receita lançada.
	FinanceDivergenceUnposted = "receita_nao_lancada"
	// FinanceDivergenceRefund: pedido cancelado com receita lançada.
	FinanceDivergenceRefund = "estorno_devido"
	// FinanceDivergenceUnreconciled: receita lançada sem conciliação com o extrato.
	FinanceDivergenceUnreconciled = "receita_sem_conciliacao"
)

// FinanceReport é o panorama financeiro derivado.
type FinanceReport struct {
	CompanyID string        `json:"company_id"`
	From      time.Time     `json:"from,omitempty"`
	To        time.Time     `json:"to,omitempty"`
	Ledger    LedgerSummary `json:"ledger"`
	// ReceivableCents é o que ainda NÃO entrou (pedidos aguardando pagamento).
	ReceivableCents int64               `json:"receivable_cents"`
	Receivables     []FinanceReceivable `json:"receivables,omitempty"`
	UnpostedRevenue int64               `json:"unposted_revenue_cents"`
	RefundDue       int64               `json:"refund_due_cents"`
	Divergences     []FinanceDivergence `json:"divergences,omitempty"`
	OrdersPaid      int                 `json:"orders_paid"`
	OrdersFulfilled int                 `json:"orders_fulfilled"`
	OrdersPending   int                 `json:"orders_pending"`
	OrdersCancelled int                 `json:"orders_cancelled"`
}

// FinanceOps liga o comércio ao livro financeiro.
type FinanceOps struct {
	commerce  *CommerceStore
	companies *CompanyStore
}

// NewFinanceOps cria o módulo financeiro sobre o comércio e o livro existentes.
func NewFinanceOps(commerce *CommerceStore, companies *CompanyStore) *FinanceOps {
	return &FinanceOps{commerce: commerce, companies: companies}
}

// orderForCompany carrega o pedido garantindo que ele é da empresa pedida.
func (f *FinanceOps) orderForCompany(companyID, orderID string) (CommerceOrder, error) {
	if f == nil || f.commerce == nil {
		return CommerceOrder{}, fmt.Errorf("%w: motor de comércio indisponível", ErrFinanceOrderMissing)
	}
	order, err := f.commerce.Order(consentScope(companyID), orderID)
	if err != nil {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrFinanceOrderMissing, orderID)
	}
	return order, nil
}

// companyForOrganization resolve a empresa da organização. O razão é por EMPRESA, e o
// comércio é por ORGANIZAÇÃO: sem uma empresa, não há onde lançar — e inventar um
// lançamento sem empresa seria pior do que recusar.
func (f *FinanceOps) companyForOrganization(organizationID string) (Company, error) {
	if f == nil || f.companies == nil {
		return Company{}, fmt.Errorf("%w: livro financeiro indisponível", ErrFinanceCompanyMissing)
	}
	for _, company := range f.companies.List(consentScope(organizationID)) {
		return company, nil
	}
	return Company{}, fmt.Errorf("%w: a organização %s não tem empresa criada", ErrFinanceCompanyMissing, consentScope(organizationID))
}

// ledgerIdempotencyKey é a chave que garante UMA receita por pedido.
func financeOrderKey(orderID string) string {
	return "commerce-order:" + strings.TrimSpace(orderID)
}

// PostOrderRevenue lança a receita de um pedido PAGO no livro da empresa.
//
// Pedido aguardando pagamento é recusado: lançar receita dele criaria receita
// fictícia. Repetir a operação devolve o lançamento existente sem duplicar.
func (f *FinanceOps) PostOrderRevenue(companyID, orderID, actor string, now time.Time) (FinanceOrderPosting, error) {
	order, err := f.orderForCompany(companyID, orderID)
	if err != nil {
		return FinanceOrderPosting{}, err
	}
	if order.Status != CommerceOrderPaid && order.Status != CommerceOrderFulfilled {
		return FinanceOrderPosting{}, fmt.Errorf("%w: pedido %s está %s", ErrFinanceOrderNotPayable, order.Number, order.Status)
	}
	if f.companies == nil {
		return FinanceOrderPosting{}, fmt.Errorf("%w: livro financeiro indisponível", ErrFinanceCompanyMissing)
	}
	company, err := f.companyForOrganization(order.OrganizationID)
	if err != nil {
		return FinanceOrderPosting{}, err
	}
	occurred := order.UpdatedAt
	if order.Payment != nil && !order.Payment.RecordedAt.IsZero() {
		occurred = order.Payment.RecordedAt
	}
	entry, created, err := f.companies.RecordLedgerEntry(LedgerEntry{
		CompanyID:   company.ID,
		Kind:        LedgerKindIncome,
		Category:    "vendas",
		AmountCents: order.TotalCents,
		Currency:    order.Currency,
		Description: "receita do pedido " + order.Number,
		Reference:   order.Number,
		// A chave de idempotência é o PEDIDO: uma receita por pedido, sempre.
		Idempotency: financeOrderKey(order.ID),
		OccurredAt:  occurred,
		CreatedBy:   strings.TrimSpace(actor),
	}, now)
	if err != nil {
		return FinanceOrderPosting{}, err
	}
	return FinanceOrderPosting{
		OrderID:     order.ID,
		OrderNumber: order.Number,
		// CompanyID é o da EMPRESA (foi para ela que o lançamento foi feito); a
		// organização do pedido continua no próprio pedido.
		CompanyID: company.ID,
		Entry:     entry,
		Created:   created,
	}, nil
}

// PostOrderCost lança um custo do pedido (frete, embalagem, taxa) no livro.
func (f *FinanceOps) PostOrderCost(companyID, orderID string, amountCents int64, category, reference, actor string, now time.Time) (LedgerEntry, error) {
	order, err := f.orderForCompany(companyID, orderID)
	if err != nil {
		return LedgerEntry{}, err
	}
	if amountCents <= 0 {
		return LedgerEntry{}, fmt.Errorf("%w: o custo precisa ser positivo", ErrCommerceInvalid)
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "custo_do_pedido"
	}
	company, err := f.companyForOrganization(order.OrganizationID)
	if err != nil {
		return LedgerEntry{}, err
	}
	entry, _, err := f.companies.RecordLedgerEntry(LedgerEntry{
		CompanyID:   company.ID,
		Kind:        LedgerKindExpense,
		Category:    category,
		AmountCents: amountCents,
		Currency:    order.Currency,
		Description: "custo do pedido " + order.Number,
		Reference:   strings.TrimSpace(reference),
		// Custo também é idempotente por pedido e categoria: repetir não duplica.
		Idempotency: "commerce-order-cost:" + order.ID + ":" + category,
		OccurredAt:  now,
		CreatedBy:   strings.TrimSpace(actor),
	}, now)
	if err != nil {
		return LedgerEntry{}, err
	}
	return entry, nil
}

// ReconcileOrderWithStatement liga o pedido à linha do extrato: garante a receita
// lançada (idempotente) e registra a conciliação no PRÓPRIO lançamento.
func (f *FinanceOps) ReconcileOrderWithStatement(companyID, orderID, statementReference, actor string, now time.Time) (LedgerEntry, error) {
	statementReference = strings.TrimSpace(statementReference)
	if statementReference == "" {
		return LedgerEntry{}, fmt.Errorf("%w: a referência do extrato é obrigatória", ErrCommerceInvalid)
	}
	posting, err := f.PostOrderRevenue(companyID, orderID, actor, now)
	if err != nil {
		return LedgerEntry{}, err
	}
	entry, _, err := f.companies.ReconcileLedgerEntry(posting.CompanyID, posting.Entry.ID, statementReference, now)
	if err != nil {
		return LedgerEntry{}, err
	}
	return entry, nil
}

// Report calcula o panorama financeiro e as divergências MEDIDAS.
func (f *FinanceOps) Report(companyID string, from, to time.Time, now time.Time) (FinanceReport, error) {
	companyID = consentScope(companyID)
	if f == nil || f.commerce == nil || f.companies == nil {
		return FinanceReport{}, fmt.Errorf("%w: comércio ou livro indisponível", ErrFinanceCompanyMissing)
	}
	company, err := f.companyForOrganization(companyID)
	if err != nil {
		return FinanceReport{}, err
	}
	report := FinanceReport{CompanyID: company.ID, From: from, To: to}
	summary, err := f.companies.LedgerSummary(company.ID, from, to)
	if err != nil {
		return FinanceReport{}, err
	}
	report.Ledger = summary

	entries, err := f.companies.ListLedgerEntries(company.ID, from, to)
	if err != nil {
		return FinanceReport{}, err
	}
	incomeByOrder := map[string]LedgerEntry{}
	refundedOrders := map[string]LedgerEntry{}
	for _, entry := range entries {
		// O ESTORNO registrado precisa ser reconhecido: sem isso o estorno continuaria
		// aparecendo como DEVIDO e o supervisor proporia a mesma ação a cada ciclo,
		// para sempre — o defeito clássico de uma ação que não fecha o que abriu.
		if strings.HasPrefix(entry.Idempotency, "refund:") {
			refundedOrders[strings.TrimPrefix(entry.Idempotency, "refund:")] = entry
			continue
		}
		if entry.Kind != LedgerKindIncome {
			continue
		}
		if strings.HasPrefix(entry.Idempotency, "commerce-order:") {
			incomeByOrder[strings.TrimPrefix(entry.Idempotency, "commerce-order:")] = entry
		}
	}

	for _, order := range f.commerce.Orders(companyID) {
		switch order.Status {
		case CommerceOrderPending:
			report.OrdersPending++
			report.ReceivableCents += order.TotalCents
			report.Receivables = append(report.Receivables, FinanceReceivable{
				OrderID: order.ID, OrderNumber: order.Number, ContactID: order.ContactID,
				TotalCents: order.TotalCents, Currency: order.Currency, CreatedAt: order.CreatedAt,
			})
			continue
		case CommerceOrderPaid:
			report.OrdersPaid++
		case CommerceOrderFulfilled:
			report.OrdersFulfilled++
		case CommerceOrderCancelled:
			report.OrdersCancelled++
		}
		entry, posted := incomeByOrder[order.ID]
		if order.Status == CommerceOrderCancelled {
			// Cancelado com receita lançada: existe estorno DEVIDO. O lançamento NÃO é
			// apagado, porque dinheiro que entrou e foi devolvido tem história.
			if !posted {
				continue
			}
			// Se o estorno JÁ foi registrado, a pendência está LIQUIDADA: continuar
			// marcando como devido faria o supervisor repropor a mesma decisão em todo
			// ciclo, mesmo depois de a pessoa ter aprovado.
			if _, settled := refundedOrders[order.ID]; settled {
				continue
			}
			report.RefundDue += order.TotalCents
			report.Divergences = append(report.Divergences, FinanceDivergence{
				OrderID: order.ID, OrderNumber: order.Number, Kind: FinanceDivergenceRefund,
				AmountCents: order.TotalCents, Detail: "pedido cancelado com receita lançada (lançamento " + entry.ID + ")",
			})
			continue
		}
		if !posted {
			// Pago e SEM receita: lacuna entre comércio e financeiro.
			report.UnpostedRevenue += order.TotalCents
			report.Divergences = append(report.Divergences, FinanceDivergence{
				OrderID: order.ID, OrderNumber: order.Number, Kind: FinanceDivergenceUnposted,
				AmountCents: order.TotalCents, Detail: "pedido pago sem receita lançada",
			})
			continue
		}
		if entry.ReconciledAt == nil {
			report.Divergences = append(report.Divergences, FinanceDivergence{
				OrderID: order.ID, OrderNumber: order.Number, Kind: FinanceDivergenceUnreconciled,
				AmountCents: order.TotalCents, Detail: "receita lançada sem conciliação com o extrato",
			})
		}
	}
	return report, nil
}
