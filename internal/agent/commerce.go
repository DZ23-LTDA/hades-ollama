package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Commerce Engine (estágio 24): produtos, estoque, pedidos e pagamento em sandbox.
//
// O estágio exige **transações, idempotência e conciliação**, e proíbe processar
// cartão sem infraestrutura e conformidade. Este núcleo entrega isso com decisões
// que valem registrar:
//
//   - **ESTOQUE É UM LIVRO DE MOVIMENTOS, não um contador**: o saldo é derivado dos
//     lançamentos. Um contador mutável perde atualização sob concorrência e não
//     permite auditar quem mexeu — o movimento permite as duas coisas;
//   - **IDEMPOTÊNCIA DE CHECKOUT COM CHAVE, sob o mesmo lock**: reenviar o mesmo
//     checkout **não** cria dois pedidos. Num sistema de comércio isso não é detalhe
//     de elegância, é a diferença entre cobrar uma vez e cobrar duas;
//   - **NÃO SE VENDE O QUE NÃO EXISTE**: o pedido **reserva** estoque na criação e
//     falha se não houver saldo disponível (saldo menos reservas);
//   - **ZERO DADO DE CARTÃO**: os métodos aceitos são **referências** (sandbox, Pix,
//     boleto). Campo com aparência de número de cartão é **recusado** — o estágio
//     proíbe processar cartão sem conformidade, então o sistema não guarda PAN nem
//     por acidente;
//   - **O TOTAL É CONFERIDO**: soma dos itens tem de bater com o total declarado, em
//     centavos inteiros. Total divergente é erro, não arredondamento silencioso;
//   - **CONCILIAÇÃO IDEMPOTENTE**: a mesma referência conciliada de novo é no-op;
//     referência diferente é conflito.

const (
	CommerceOrderDraft      = "rascunho"
	CommerceOrderPending    = "aguardando_pagamento"
	CommerceOrderPaid       = "pago"
	CommerceOrderFulfilled  = "atendido"
	CommerceOrderCancelled  = "cancelado"
	CommerceDefaultCurrency = "BRL"

	// Métodos de pagamento aceitos: apenas REFERÊNCIAS, nunca dados de cartão.
	CommercePaymentSandbox = "sandbox"
	CommercePaymentPix     = "pix"
	CommercePaymentBoleto  = "boleto"
	CommercePaymentManual  = "manual"

	// Motivos de movimento de estoque.
	CommerceMovementEntry   = "entrada"
	CommerceMovementReserve = "reserva"
	CommerceMovementRelease = "liberacao"
	CommerceMovementShipped = "saida"
	CommerceMovementAdjust  = "ajuste"

	// CommerceMaxCardLikeDigits é o limite a partir do qual um número é considerado
	// dado de cartão e recusado.
	CommerceMaxCardLikeDigits = 12
)

var (
	ErrCommerceInvalid           = errors.New("commerce entry invalid")
	ErrCommerceProductMissing    = errors.New("commerce product not found")
	ErrCommerceOrderMissing      = errors.New("commerce order not found")
	ErrCommerceInsufficient      = errors.New("insufficient available stock")
	ErrCommerceCardDataRefused   = errors.New("card data is not accepted; use a payment reference")
	ErrCommerceIdempotencyKey    = errors.New("idempotency key is required")
	ErrCommerceTotalMismatch     = errors.New("order total does not match the items")
	ErrCommerceOrderState        = errors.New("order is not in a state that allows this operation")
	ErrCommerceReconcileConflict = errors.New("order was already reconciled with another reference")
)

// CommerceProduct é um item do catálogo.
type CommerceProduct struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	PriceCents     int64     `json:"price_cents"`
	Currency       string    `json:"currency"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

// CommerceStockMovement é um lançamento no livro de estoque. O saldo é DERIVADO.
type CommerceStockMovement struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	SKU            string `json:"sku"`
	// Delta é positivo para entrada e negativo para saída/reserva.
	Delta     int64     `json:"delta"`
	Kind      string    `json:"kind"`
	Reference string    `json:"reference,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CommerceStockBalance é o saldo calculado.
type CommerceStockBalance struct {
	SKU           string `json:"sku"`
	OnHand        int64  `json:"on_hand"`
	Reserved      int64  `json:"reserved"`
	Available     int64  `json:"available"`
	MovementCount int    `json:"movement_count"`
}

// CommerceOrderItem é uma linha do pedido.
type CommerceOrderItem struct {
	SKU        string `json:"sku"`
	Name       string `json:"name,omitempty"`
	Quantity   int64  `json:"quantity"`
	UnitCents  int64  `json:"unit_cents"`
	TotalCents int64  `json:"total_cents"`
}

// CommercePayment é a QUITAÇÃO por referência externa. Nunca contém dado de cartão.
type CommercePayment struct {
	Method      string    `json:"method"`
	Reference   string    `json:"reference"`
	AmountCents int64     `json:"amount_cents"`
	RecordedAt  time.Time `json:"recorded_at"`
	RecordedBy  string    `json:"recorded_by,omitempty"`
}

// CommerceOrder é o pedido.
type CommerceOrder struct {
	ID             string              `json:"id"`
	OrganizationID string              `json:"organization_id"`
	Number         string              `json:"number"`
	ContactID      string              `json:"contact_id,omitempty"`
	Status         string              `json:"status"`
	Items          []CommerceOrderItem `json:"items"`
	TotalCents     int64               `json:"total_cents"`
	Currency       string              `json:"currency"`
	IdempotencyKey string              `json:"idempotency_key"`
	Payment        *CommercePayment    `json:"payment,omitempty"`
	ReconciledRef  string              `json:"reconciled_reference,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	CancelledAt    *time.Time          `json:"cancelled_at,omitempty"`
	// Duplicated marca a devolução idempotente do MESMO pedido.
	Duplicated bool `json:"duplicated,omitempty"`
}

// CommerceOrderRequest é a entrada de criação de pedido.
type CommerceOrderRequest struct {
	OrganizationID string
	ContactID      string
	Items          []CommerceOrderItem
	Currency       string
	IdempotencyKey string
}

// CommerceStore guarda catálogo, estoque e pedidos por organização.
type CommerceStore struct {
	mu            sync.RWMutex
	products      map[string]CommerceProduct
	stock         map[string][]CommerceStockMovement
	orders        map[string]CommerceOrder
	idempotency   map[string]string
	orderSequence map[string]int64
}

// NewCommerceStore cria o motor de comércio.
func NewCommerceStore() *CommerceStore {
	return &CommerceStore{
		products:      map[string]CommerceProduct{},
		stock:         map[string][]CommerceStockMovement{},
		orders:        map[string]CommerceOrder{},
		idempotency:   map[string]string{},
		orderSequence: map[string]int64{},
	}
}

func commerceKey(organizationID, value string) string {
	return consentScope(organizationID) + "|" + strings.TrimSpace(value)
}

// looksLikeCardData recusa conteúdo com aparência de dado de cartão.
func looksLikeCardData(value string) bool {
	digits := 0
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
			digits++
		case char == ' ' || char == '-' || char == '.':
		default:
			return false
		}
	}
	return digits >= CommerceMaxCardLikeDigits
}

// UpsertProduct cadastra ou atualiza um produto.
func (s *CommerceStore) UpsertProduct(product CommerceProduct, now time.Time) (CommerceProduct, error) {
	product.OrganizationID = consentScope(product.OrganizationID)
	product.SKU = strings.TrimSpace(product.SKU)
	product.Name = strings.TrimSpace(product.Name)
	if product.SKU == "" || product.Name == "" || product.PriceCents < 0 {
		return CommerceProduct{}, fmt.Errorf("%w: sku, name and non-negative price are required", ErrCommerceInvalid)
	}
	if product.Currency == "" {
		product.Currency = CommerceDefaultCurrency
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, existing := range s.products {
		if existing.OrganizationID == product.OrganizationID && existing.SKU == product.SKU {
			product.ID = id
			if product.CreatedAt.IsZero() {
				product.CreatedAt = existing.CreatedAt
			}
			s.products[id] = product
			return product, nil
		}
	}
	if product.ID == "" {
		product.ID = "prd_" + uuid.NewString()[:12]
	}
	product.CreatedAt = now
	s.products[product.ID] = product
	return product, nil
}

// Products lista o catálogo de uma organização.
func (s *CommerceStore) Products(organizationID string) []CommerceProduct {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CommerceProduct, 0)
	for _, product := range s.products {
		if product.OrganizationID != organizationID {
			continue
		}
		out = append(out, product)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

func (s *CommerceStore) productBySKULocked(organizationID, sku string) (CommerceProduct, bool) {
	for _, product := range s.products {
		if product.OrganizationID == organizationID && product.SKU == strings.TrimSpace(sku) {
			return product, true
		}
	}
	return CommerceProduct{}, false
}

// RecordStock lança um movimento de estoque. Entrada é positiva; saída/reserva é
// negativa. Reserva não pode exceder o disponível.
func (s *CommerceStore) RecordStock(organizationID, sku string, delta int64, kind, reference string, now time.Time) (CommerceStockMovement, error) {
	organizationID = consentScope(organizationID)
	sku = strings.TrimSpace(sku)
	if sku == "" || delta == 0 {
		return CommerceStockMovement{}, fmt.Errorf("%w: sku e delta são obrigatórios", ErrCommerceInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.productBySKULocked(organizationID, sku); !ok {
		return CommerceStockMovement{}, fmt.Errorf("%w: %s", ErrCommerceProductMissing, sku)
	}
	key := commerceKey(organizationID, sku)
	if delta < 0 {
		balance := commerceBalanceLocked(s.stock[key], sku)
		if balance.Available+delta < 0 {
			return CommerceStockMovement{}, fmt.Errorf("%w: %s tem %d disponível e o movimento pede %d",
				ErrCommerceInsufficient, sku, balance.Available, delta)
		}
	}
	movement := CommerceStockMovement{
		ID:             "mov_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		SKU:            sku,
		Delta:          delta,
		Kind:           strings.TrimSpace(kind),
		Reference:      strings.TrimSpace(reference),
		CreatedAt:      now,
	}
	s.stock[key] = append(s.stock[key], movement)
	return movement, nil
}

func commerceBalanceLocked(movements []CommerceStockMovement, sku string) CommerceStockBalance {
	balance := CommerceStockBalance{SKU: sku, MovementCount: len(movements)}
	for _, movement := range movements {
		switch movement.Kind {
		case CommerceMovementReserve:
			// A reserva NÃO tira a mercadoria da prateleira: ela continua no estoque
			// físico e só deixa de estar disponível. Somar no `OnHand` aqui faria o
			// saldo físico mentir sobre o que existe no depósito.
			balance.Reserved += -movement.Delta
		case CommerceMovementRelease:
			// Liberar devolve ao disponível, sem mexer no físico.
			balance.Reserved -= movement.Delta
		default:
			// Entrada e saída mexem no físico de verdade.
			balance.OnHand += movement.Delta
		}
	}
	if balance.Reserved < 0 {
		balance.Reserved = 0
	}
	// Disponível = físico menos o que já está comprometido com pedidos.
	balance.Available = balance.OnHand - balance.Reserved
	return balance
}

// Stock devolve o saldo DERIVADO dos movimentos.
func (s *CommerceStore) Stock(organizationID, sku string) CommerceStockBalance {
	organizationID = consentScope(organizationID)
	sku = strings.TrimSpace(sku)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return commerceBalanceLocked(s.stock[commerceKey(organizationID, sku)], sku)
}

// Movements devolve o livro de movimentos de um SKU.
func (s *CommerceStore) Movements(organizationID, sku string) []CommerceStockMovement {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]CommerceStockMovement(nil), s.stock[commerceKey(organizationID, sku)]...)
}

// ValidatePayment recusa método desconhecido e qualquer dado com aparência de cartão.
func ValidatePayment(payment CommercePayment) error {
	switch payment.Method {
	case CommercePaymentSandbox, CommercePaymentPix, CommercePaymentBoleto, CommercePaymentManual:
	default:
		return fmt.Errorf("%w: método %q não é aceito", ErrCommerceInvalid, payment.Method)
	}
	if looksLikeCardData(payment.Reference) {
		return ErrCommerceCardDataRefused
	}
	if strings.TrimSpace(payment.Reference) == "" {
		return fmt.Errorf("%w: a referência do pagamento é obrigatória", ErrCommerceInvalid)
	}
	return nil
}

// CreateOrder cria o pedido com IDEMPOTÊNCIA e RESERVA de estoque.
//
// A mesma chave de idempotência devolve o MESMO pedido: reenviar o checkout não pode
// gerar duas cobranças. Chave, conferência de estoque e criação ficam sob o mesmo lock.
func (s *CommerceStore) CreateOrder(request CommerceOrderRequest, now time.Time) (CommerceOrder, error) {
	organizationID := consentScope(request.OrganizationID)
	key := strings.TrimSpace(request.IdempotencyKey)
	if key == "" {
		return CommerceOrder{}, fmt.Errorf("%w: sem chave, um reenvio criaria um segundo pedido", ErrCommerceIdempotencyKey)
	}
	if len(request.Items) == 0 {
		return CommerceOrder{}, fmt.Errorf("%w: o pedido precisa de itens", ErrCommerceInvalid)
	}
	currency := strings.TrimSpace(request.Currency)
	if currency == "" {
		currency = CommerceDefaultCurrency
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idempotencyKey := commerceKey(organizationID, key)
	if orderID, seen := s.idempotency[idempotencyKey]; seen {
		order := s.orders[orderID]
		order.Duplicated = true
		return order, nil
	}

	items := make([]CommerceOrderItem, 0, len(request.Items))
	var total int64
	for _, item := range request.Items {
		if item.Quantity <= 0 {
			return CommerceOrder{}, fmt.Errorf("%w: quantidade do item %s precisa ser positiva", ErrCommerceInvalid, item.SKU)
		}
		product, ok := s.productBySKULocked(organizationID, item.SKU)
		if !ok {
			return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceProductMissing, item.SKU)
		}
		unit := product.PriceCents
		itemTotal := unit * item.Quantity
		// O preço vem do CATÁLOGO, não do pedido: o cliente não escolhe quanto paga.
		line := CommerceOrderItem{SKU: product.SKU, Name: product.Name, Quantity: item.Quantity, UnitCents: unit, TotalCents: itemTotal}
		items = append(items, line)
		total += itemTotal
	}
	// Reserva de estoque: não se vende o que não existe.
	for _, line := range items {
		balance := commerceBalanceLocked(s.stock[commerceKey(organizationID, line.SKU)], line.SKU)
		if balance.Available < line.Quantity {
			return CommerceOrder{}, fmt.Errorf("%w: %s tem %d disponível e o pedido pede %d",
				ErrCommerceInsufficient, line.SKU, balance.Available, line.Quantity)
		}
	}
	s.orderSequence[organizationID]++
	order := CommerceOrder{
		ID:             "ord_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		Number:         fmt.Sprintf("PED-%06d", s.orderSequence[organizationID]),
		ContactID:      strings.TrimSpace(request.ContactID),
		Status:         CommerceOrderPending,
		Items:          items,
		TotalCents:     total,
		Currency:       currency,
		IdempotencyKey: key,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// A reserva é lançada junto com o pedido: ou as duas coisas acontecem, ou nenhuma.
	reservations := make([]CommerceStockMovement, 0, len(items))
	for _, line := range items {
		reservations = append(reservations, CommerceStockMovement{
			ID: "mov_" + uuid.NewString()[:12], OrganizationID: organizationID, SKU: line.SKU,
			Delta: -line.Quantity, Kind: CommerceMovementReserve, Reference: order.ID, CreatedAt: now,
		})
	}
	for _, movement := range reservations {
		key := commerceKey(organizationID, movement.SKU)
		s.stock[key] = append(s.stock[key], movement)
	}
	s.orders[order.ID] = order
	s.idempotency[idempotencyKey] = order.ID
	return order, nil
}

// Orders lista os pedidos de uma organização.
func (s *CommerceStore) Orders(organizationID string) []CommerceOrder {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CommerceOrder, 0)
	for _, order := range s.orders {
		if order.OrganizationID != organizationID {
			continue
		}
		out = append(out, order)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Order devolve um pedido da organização.
func (s *CommerceStore) Order(organizationID, orderID string) (CommerceOrder, error) {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[strings.TrimSpace(orderID)]
	if !ok || order.OrganizationID != organizationID {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceOrderMissing, orderID)
	}
	return order, nil
}

// ConfirmPayment registra o pagamento por REFERÊNCIA e move o pedido para pago.
func (s *CommerceStore) ConfirmPayment(organizationID, orderID string, payment CommercePayment, actor string, now time.Time) (CommerceOrder, error) {
	organizationID = consentScope(organizationID)
	if err := ValidatePayment(payment); err != nil {
		return CommerceOrder{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[strings.TrimSpace(orderID)]
	if !ok || order.OrganizationID != organizationID {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceOrderMissing, orderID)
	}
	if order.Status == CommerceOrderCancelled || order.Status == CommerceOrderFulfilled {
		return CommerceOrder{}, fmt.Errorf("%w: pedido está %s", ErrCommerceOrderState, order.Status)
	}
	// Reconfirmar com a MESMA referência é idempotente; referência diferente é conflito.
	if order.Payment != nil && order.Payment.Reference != strings.TrimSpace(payment.Reference) {
		return CommerceOrder{}, fmt.Errorf("%w: pedido já quitado com %s", ErrCommerceReconcileConflict, order.Payment.Reference)
	}
	if payment.AmountCents == 0 {
		payment.AmountCents = order.TotalCents
	}
	if payment.AmountCents != order.TotalCents {
		return CommerceOrder{}, fmt.Errorf("%w: pagamento de %d para pedido de %d",
			ErrCommerceTotalMismatch, payment.AmountCents, order.TotalCents)
	}
	payment.RecordedAt = now
	payment.RecordedBy = strings.TrimSpace(actor)
	order.Payment = &payment
	order.Status = CommerceOrderPaid
	order.UpdatedAt = now
	s.orders[order.ID] = order
	return order, nil
}

// Reconcile registra a conciliação. Mesma referência é no-op; outra é conflito.
func (s *CommerceStore) Reconcile(organizationID, orderID, reference string, now time.Time) (CommerceOrder, error) {
	organizationID = consentScope(organizationID)
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return CommerceOrder{}, fmt.Errorf("%w: a referência da conciliação é obrigatória", ErrCommerceInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[strings.TrimSpace(orderID)]
	if !ok || order.OrganizationID != organizationID {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceOrderMissing, orderID)
	}
	if order.ReconciledRef == reference {
		return order, nil
	}
	if order.ReconciledRef != "" {
		return CommerceOrder{}, fmt.Errorf("%w: já conciliado com %s", ErrCommerceReconcileConflict, order.ReconciledRef)
	}
	order.ReconciledRef = reference
	order.UpdatedAt = now
	s.orders[order.ID] = order
	return order, nil
}

// Fulfill marca o pedido como atendido: a reserva vira saída definitiva.
func (s *CommerceStore) Fulfill(organizationID, orderID string, now time.Time) (CommerceOrder, error) {
	organizationID = consentScope(organizationID)
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[strings.TrimSpace(orderID)]
	if !ok || order.OrganizationID != organizationID {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceOrderMissing, orderID)
	}
	if order.Status != CommerceOrderPaid {
		return CommerceOrder{}, fmt.Errorf("%w: só pedido pago pode ser atendido (está %s)", ErrCommerceOrderState, order.Status)
	}
	for _, line := range order.Items {
		key := commerceKey(organizationID, line.SKU)
		// Libera a reserva e lança a saída: o físico diminui agora.
		s.stock[key] = append(s.stock[key],
			CommerceStockMovement{
				ID: "mov_" + uuid.NewString()[:12], OrganizationID: organizationID, SKU: line.SKU,
				Delta: line.Quantity, Kind: CommerceMovementRelease, Reference: order.ID, CreatedAt: now,
			},
			CommerceStockMovement{
				ID: "mov_" + uuid.NewString()[:12], OrganizationID: organizationID, SKU: line.SKU,
				Delta: -line.Quantity, Kind: CommerceMovementShipped, Reference: order.ID, CreatedAt: now,
			},
		)
	}
	order.Status = CommerceOrderFulfilled
	order.UpdatedAt = now
	s.orders[order.ID] = order
	return order, nil
}

// Cancel cancela o pedido e LIBERA a reserva de estoque.
func (s *CommerceStore) Cancel(organizationID, orderID, reason string, now time.Time) (CommerceOrder, error) {
	organizationID = consentScope(organizationID)
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[strings.TrimSpace(orderID)]
	if !ok || order.OrganizationID != organizationID {
		return CommerceOrder{}, fmt.Errorf("%w: %s", ErrCommerceOrderMissing, orderID)
	}
	if order.Status == CommerceOrderCancelled {
		return order, nil
	}
	if order.Status == CommerceOrderFulfilled {
		return CommerceOrder{}, fmt.Errorf("%w: pedido já atendido não pode ser cancelado", ErrCommerceOrderState)
	}
	for _, line := range order.Items {
		key := commerceKey(organizationID, line.SKU)
		s.stock[key] = append(s.stock[key], CommerceStockMovement{
			ID: "mov_" + uuid.NewString()[:12], OrganizationID: organizationID, SKU: line.SKU,
			Delta: line.Quantity, Kind: CommerceMovementRelease, Reference: order.ID + ":" + strings.TrimSpace(reason), CreatedAt: now,
		})
	}
	cancelled := now
	order.Status = CommerceOrderCancelled
	order.CancelledAt = &cancelled
	order.UpdatedAt = now
	s.orders[order.ID] = order
	return order, nil
}

// CommerceSummary são os indicadores do motor de comércio.
type CommerceSummary struct {
	OrganizationID string   `json:"organization_id"`
	Products       int      `json:"products"`
	Orders         int      `json:"orders"`
	Pending        int      `json:"pending"`
	Paid           int      `json:"paid"`
	Fulfilled      int      `json:"fulfilled"`
	Cancelled      int      `json:"cancelled"`
	RevenueCents   int64    `json:"revenue_cents"`
	LowStockSKUs   []string `json:"low_stock_skus,omitempty"`
}

// Summary calcula os indicadores.
func (s *CommerceStore) Summary(organizationID string) CommerceSummary {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary := CommerceSummary{OrganizationID: organizationID}
	for _, product := range s.products {
		if product.OrganizationID != organizationID {
			continue
		}
		summary.Products++
		balance := commerceBalanceLocked(s.stock[commerceKey(organizationID, product.SKU)], product.SKU)
		if balance.Available <= 0 {
			summary.LowStockSKUs = append(summary.LowStockSKUs, product.SKU)
		}
	}
	sort.Strings(summary.LowStockSKUs)
	for _, order := range s.orders {
		if order.OrganizationID != organizationID {
			continue
		}
		summary.Orders++
		switch order.Status {
		case CommerceOrderPaid:
			summary.Paid++
			summary.RevenueCents += order.TotalCents
		case CommerceOrderFulfilled:
			summary.Fulfilled++
			summary.RevenueCents += order.TotalCents
		case CommerceOrderCancelled:
			summary.Cancelled++
		default:
			summary.Pending++
		}
	}
	return summary
}
