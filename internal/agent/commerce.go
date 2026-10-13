package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// commerceFile é o formato em disco de UMA organização. Guardar por organização
// mantém o isolamento no próprio armazenamento, e não só na consulta.
type commerceFile struct {
	Products    []CommerceProduct                  `json:"products"`
	Stock       map[string][]CommerceStockMovement `json:"stock"`
	Orders      []CommerceOrder                    `json:"orders"`
	Returns     []CommerceReturn                   `json:"returns,omitempty"`
	Carts       []CommerceCart                     `json:"carts,omitempty"`
	Idempotency map[string]string                  `json:"idempotency"`
	Sequence    int64                              `json:"sequence"`
}

// CommerceStore guarda catálogo, estoque e pedidos por organização.
type CommerceStore struct {
	mu            sync.RWMutex
	returns       map[string]CommerceReturn
	carts         map[string]CommerceCart
	products      map[string]CommerceProduct
	stock         map[string][]CommerceStockMovement
	orders        map[string]CommerceOrder
	idempotency   map[string]string
	orderSequence map[string]int64
	// E-363: ÍNDICE por (organização, SKU). MEDIDO antes: 10,03 ms por produto e enchimento QUADRÁTICO, porque cada
	// operação varria o mapa inteiro (para saber se o SKU existe E para contar o acervo). O índice responde as duas
	// perguntas em O(1) e o tamanho do mapa de uma organização É a contagem do acervo — *uma fonte, não duas*.
	produtosPorChave map[string]map[string]string
	// root vazio mantém o motor em memória (usado em testes e cenários efêmeros).
	root string
}

// NewCommerceStore cria o motor de comércio EM MEMÓRIA.
func NewCommerceStore() *CommerceStore {
	return &CommerceStore{
		products:      map[string]CommerceProduct{},
		stock:         map[string][]CommerceStockMovement{},
		orders:        map[string]CommerceOrder{},
		idempotency:   map[string]string{},
		orderSequence: map[string]int64{},
	}
}

// NewCommerceStoreAt cria o motor de comércio com PERSISTÊNCIA, um arquivo por
// organização, e carrega o que já existe. Raiz vazia devolve o motor em memória.
func NewCommerceStoreAt(root string) (*CommerceStore, error) {
	store := NewCommerceStore()
	store.root = strings.TrimSpace(root)
	if store.root == "" {
		return store, nil
	}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var stored commerceFile
		if err := readJSON(filepath.Join(store.root, entry.Name()), &stored); err != nil {
			return nil, fmt.Errorf("load commerce from %s: %w", entry.Name(), err)
		}
		for _, product := range stored.Products {
			store.products[product.ID] = product
			if stored.Sequence > store.orderSequence[consentScope(product.OrganizationID)] {
				store.orderSequence[consentScope(product.OrganizationID)] = stored.Sequence
			}
		}
		for orgKey, movements := range stored.Stock {
			store.stock[orgKey] = append(store.stock[orgKey], movements...)
		}
		for _, order := range stored.Orders {
			store.orders[order.ID] = order
			if order.IdempotencyKey != "" {
				store.idempotency[commerceKey(order.OrganizationID, order.IdempotencyKey)] = order.ID
			}
			if stored.Sequence > store.orderSequence[consentScope(order.OrganizationID)] {
				store.orderSequence[consentScope(order.OrganizationID)] = stored.Sequence
			}
		}
		for _, carrinho := range stored.Carts {
			if store.carts == nil {
				store.carts = map[string]CommerceCart{}
			}
			store.carts[carrinho.ID] = carrinho
		}
		for _, devolucao := range stored.Returns {
			if store.returns == nil {
				store.returns = map[string]CommerceReturn{}
			}
			store.returns[devolucao.ID] = devolucao
		}
		for key, orderID := range stored.Idempotency {
			store.idempotency[key] = orderID
		}
	}
	return store, nil
}

// commercePath devolve o arquivo da organização. O nome é sanitizado, então uma
// organização não consegue escrever no arquivo de outra nem escapar da raiz.
func (s *CommerceStore) commercePath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(consentScope(organizationID))+".json")
}

// persistLocked grava SOMENTE a organização indicada.
func (s *CommerceStore) persistLocked(organizationID string) error {
	path := s.commercePath(organizationID)
	if path == "" {
		return nil
	}
	organizationID = consentScope(organizationID)
	payload := commerceFile{
		Products:    []CommerceProduct{},
		Returns:     []CommerceReturn{},
		Carts:       []CommerceCart{},
		Stock:       map[string][]CommerceStockMovement{},
		Orders:      []CommerceOrder{},
		Idempotency: map[string]string{},
		Sequence:    s.orderSequence[organizationID],
	}
	for _, product := range s.products {
		if product.OrganizationID == organizationID {
			payload.Products = append(payload.Products, product)
		}
	}
	for _, devolucao := range s.returns {
		if consentScope(devolucao.OrganizationID) == organizationID {
			payload.Returns = append(payload.Returns, devolucao)
		}
	}
	for _, carrinho := range s.carts {
		if consentScope(carrinho.OrganizationID) == organizationID {
			payload.Carts = append(payload.Carts, carrinho)
		}
	}
	for key, movements := range s.stock {
		if strings.HasPrefix(key, organizationID+"|") {
			payload.Stock[key] = movements
		}
	}
	for _, order := range s.orders {
		if order.OrganizationID == organizationID {
			payload.Orders = append(payload.Orders, order)
		}
	}
	for key, orderID := range s.idempotency {
		if strings.HasPrefix(key, organizationID+"|") {
			payload.Idempotency[key] = orderID
		}
	}
	sort.SliceStable(payload.Products, func(i, j int) bool { return payload.Products[i].ID < payload.Products[j].ID })
	sort.SliceStable(payload.Orders, func(i, j int) bool { return payload.Orders[i].ID < payload.Orders[j].ID })
	return writeJSONAtomic(path, payload)
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
	if err := requireOrganization(product.OrganizationID); err != nil {
		return CommerceProduct{}, fmt.Errorf("%w: %v", ErrCommerceInvalid, err)
	}
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
	// E-360: O ORÇAMENTO VALE TAMBÉM NO CAMINHO DIRETO. Em E-359 o teto acumulado vivia só no onboarding, e eu
	// declarei esse limite: uma chamada direta a `UpsertProduct` não consumia orçamento. ***Limite que depende do
	// chamador lembrar de usar o caminho certo não é limite.*** A contagem é feita DENTRO do lock para ser consistente
	// com o que está sendo gravado. Atualizar SKU existente NÃO é bloqueado: o acervo não cresce.
	if !existeSKULocked(s, product.OrganizationID, product.SKU) {
		// O tamanho do mapa da organização É a contagem do acervo (E-363): uma fonte, não duas.
		acervo := s.AcervoLocked(product.OrganizationID)
		if limite := DefaultOnboardingLimits().MaxRows; acervo+1 > limite {
			return CommerceProduct{}, fmt.Errorf("%w: acervo da organização chegaria a %d produtos (limite %d); "+
				"use o onboarding com um plano menor ou remova produtos antes de acrescentar", ErrOnboardingTooLarge, acervo+1, limite)
		}
	}
	// E-363: O(1) de verdade. A primeira versão desta mudança ADICIONOU o índice mas MANTEVE esta varredura — e a
	// medição mostrou o custo SUBIR (10,03 -> 13,56 ms por produto). ***Instrumento ao lado do defeito não é correção;
	// a medição é que decide.***
	if id, existe := s.IDPorSKULocked(product.OrganizationID, product.SKU); existe {
		existing := s.products[id]
		product.ID = id
		if product.CreatedAt.IsZero() {
			product.CreatedAt = existing.CreatedAt
		}
		s.products[id] = product
		s.indexarProdutoLocked(product)
		if err := s.persistLocked(product.OrganizationID); err != nil {
			return CommerceProduct{}, err
		}
		return product, nil
	}
	if product.ID == "" {
		product.ID = "prd_" + uuid.NewString()[:12]
	}
	product.CreatedAt = now
	s.products[product.ID] = product
	s.indexarProdutoLocked(product)
	if err := s.persistLocked(product.OrganizationID); err != nil {
		return CommerceProduct{}, err
	}
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
	if err := s.persistLocked(organizationID); err != nil {
		s.stock[key] = s.stock[key][:len(s.stock[key])-1]
		return CommerceStockMovement{}, err
	}
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
	if err := s.persistLocked(organizationID); err != nil {
		// Sem gravação o pedido não existe: desfaz pedido, reservas e chave para não
		// deixar memória e disco discordando.
		delete(s.orders, order.ID)
		delete(s.idempotency, idempotencyKey)
		for _, movement := range reservations {
			key := commerceKey(organizationID, movement.SKU)
			if len(s.stock[key]) > 0 {
				s.stock[key] = s.stock[key][:len(s.stock[key])-1]
			}
		}
		return CommerceOrder{}, err
	}
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
	if err := s.persistLocked(organizationID); err != nil {
		return CommerceOrder{}, err
	}
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
	if err := s.persistLocked(organizationID); err != nil {
		return CommerceOrder{}, err
	}
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
	// Os dois movimentos por linha vão juntos para o disco: o saldo em memória e o do
	// arquivo não podem divergir.
	order.Status = CommerceOrderFulfilled
	order.UpdatedAt = now
	s.orders[order.ID] = order
	if err := s.persistLocked(organizationID); err != nil {
		return CommerceOrder{}, err
	}
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
	// A liberação acompanha o cancelamento: os dois vão para o disco juntos.
	cancelled := now
	order.Status = CommerceOrderCancelled
	order.CancelledAt = &cancelled
	order.UpdatedAt = now
	s.orders[order.ID] = order
	if err := s.persistLocked(organizationID); err != nil {
		return CommerceOrder{}, err
	}
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

// ErrCommerceReturnQuantity cobre devolucao de SKU ausente no pedido, quantidade invalida ou acima do comprado.
var ErrCommerceCartMissing = errors.New("commerce cart not found")

var ErrCommerceReturnQuantity = errors.New("return quantity is not allowed for this order")

// CommerceReturnRequest descreve uma devolucao pedida pelo operador.
type CommerceReturnRequest struct {
	OrganizationID string `json:"organization_id"`
	OrderID        string `json:"order_id"`
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	Reason         string `json:"reason"`
	ActorID        string `json:"actor_id"`
}

// CommerceReturn e o REGISTRO de uma devolucao. O campo FinancialSignal diz explicitamente que o dinheiro NAO se
// moveu: reembolso exige aprovacao humana, e prometer o contrario seria mentir.
type CommerceReturn struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	OrderID         string    `json:"order_id"`
	OrderNumber     string    `json:"order_number"`
	SKU             string    `json:"sku"`
	Quantity        int64     `json:"quantity"`
	Reason          string    `json:"reason"`
	ActorID         string    `json:"actor_id"`
	MovementID      string    `json:"movement_id"`
	RefundCents     int64     `json:"refund_cents"`
	FinancialSignal string    `json:"financial_signal"`
	CreatedAt       time.Time `json:"created_at"`
}

// RecordReturn registra a devolucao de itens de um pedido PAGO, devolve o estoque e sinaliza o reembolso devido.
//
// Propriedades travadas por teste: o pedido precisa ser do MESMO tenant (recusa identica a de inexistente),
// so pedido pago/atendido pode devolver, a quantidade nao pode passar do comprado menos o ja devolvido, e repetir
// a MESMA devolucao e IDEMPOTENTE (nao devolve estoque duas vezes).
func (s *CommerceStore) RecordReturn(request CommerceReturnRequest, now time.Time) (CommerceReturn, error) {
	organizationID := consentScope(request.OrganizationID)
	order, err := s.Order(organizationID, strings.TrimSpace(request.OrderID))
	if err != nil {
		return CommerceReturn{}, err
	}
	if order.Status != CommerceOrderPaid && order.Status != CommerceOrderFulfilled {
		return CommerceReturn{}, fmt.Errorf("%w: pedido em %q nao pode ser devolvido", ErrCommerceOrderState, order.Status)
	}
	sku := strings.TrimSpace(request.SKU)
	if request.Quantity <= 0 {
		return CommerceReturn{}, fmt.Errorf("%w: quantidade %d", ErrCommerceReturnQuantity, request.Quantity)
	}
	comprado := int64(0)
	var unitario int64
	for _, item := range order.Items {
		if item.SKU != sku {
			continue
		}
		comprado += item.Quantity
	}
	if comprado == 0 {
		return CommerceReturn{}, fmt.Errorf("%w: sku %q nao pertence ao pedido %s", ErrCommerceReturnQuantity, sku, order.Number)
	}
	for _, product := range s.Products(organizationID) {
		if product.SKU == sku {
			unitario = product.PriceCents
			break
		}
	}
	jaDevolvido := int64(0)
	s.mu.RLock()
	for _, existente := range s.returns {
		if existente.OrderID != order.ID || existente.SKU != sku {
			continue
		}
		// IDEMPOTENCIA: repetir a MESMA devolucao devolve o registro existente, sem novo movimento.
		if existente.Quantity == request.Quantity && existente.Reason == strings.TrimSpace(request.Reason) {
			s.mu.RUnlock()
			return existente, nil
		}
		jaDevolvido += existente.Quantity
	}
	s.mu.RUnlock()
	if request.Quantity > comprado-jaDevolvido {
		return CommerceReturn{}, fmt.Errorf("%w: %d excede o devolvivel (%d de %d)", ErrCommerceReturnQuantity, request.Quantity, comprado-jaDevolvido, comprado)
	}
	devolucao := CommerceReturn{
		ID:              "ret_" + uuid.NewString(),
		OrganizationID:  organizationID,
		OrderID:         order.ID,
		OrderNumber:     order.Number,
		SKU:             sku,
		Quantity:        request.Quantity,
		Reason:          strings.TrimSpace(request.Reason),
		ActorID:         strings.TrimSpace(request.ActorID),
		RefundCents:     unitario * request.Quantity,
		FinancialSignal: "reembolso_devido_requer_aprovacao_humana",
		CreatedAt:       now,
	}
	// ORDEM CORRETA (e a licao de E-258 aplicada com criterio): primeiro o MOVIMENTO de estoque, depois o registro
	// COMPLETO e UMA gravacao. Inserir o registro antes do movimento fazia o estado duravel sair sem MovementID -
	// devolucao sem vinculo auditavel. Devolucao e acao RARA e humana, nao hot path: uma gravacao explicita aqui e'
	// barata, diferente da gravacao por CICLO que instabilizou a suite.
	movement, err := s.RecordStock(organizationID, sku, request.Quantity, CommerceMovementEntry, "devolucao:"+order.Number, now)
	if err != nil {
		return CommerceReturn{}, err
	}
	devolucao.MovementID = movement.ID
	s.mu.Lock()
	if s.returns == nil {
		s.returns = map[string]CommerceReturn{}
	}
	s.returns[devolucao.ID] = devolucao
	persistErr := s.persistLocked(organizationID)
	if persistErr != nil {
		// O estoque JA voltou (movimento duravel). O registro nao gravou: devolver o erro e dizer o que aconteceu,
		// em vez de deixar a devolucao invisivel.
		delete(s.returns, devolucao.ID)
	}
	s.mu.Unlock()
	if persistErr != nil {
		return CommerceReturn{}, fmt.Errorf("movimento %s registrado, mas a devolucao nao foi gravada: %w", movement.ID, persistErr)
	}
	return devolucao, nil
}

// Returns devolve as devolucoes registradas da organizacao (copia).
func (s *CommerceStore) Returns(organizationID string) []CommerceReturn {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := []CommerceReturn{}
	for _, devolucao := range s.returns {
		if consentScope(devolucao.OrganizationID) == organizationID {
			resultado = append(resultado, devolucao)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].CreatedAt.Before(resultado[j].CreatedAt) })
	return resultado
}

// Status do carrinho. Um carrinho convertido NAO aceita mais item: aceitar seria vender duas vezes o mesmo carrinho.
const (
	CommerceCartOpen      = "aberto"
	CommerceCartConverted = "convertido"
)

// ErrCommerceCartClosed cobre mutacao de carrinho ja convertido.
var ErrCommerceCartClosed = errors.New("commerce cart is not open")

// CommerceCartItem e uma linha do carrinho, com o PRECO DO MOMENTO em que foi adicionado.
type CommerceCartItem struct {
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

// CommerceCart e o carrinho persistido de uma organizacao.
type CommerceCart struct {
	ID             string             `json:"id"`
	OrganizationID string             `json:"organization_id"`
	Status         string             `json:"status"`
	Items          []CommerceCartItem `json:"items"`
	OrderID        string             `json:"order_id,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// Cart devolve o carrinho da organizacao (recusa identica a de inexistente para outro tenant).
func (s *CommerceStore) Cart(organizationID, cartID string) (CommerceCart, error) {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	carrinho, ok := s.carts[strings.TrimSpace(cartID)]
	if !ok || consentScope(carrinho.OrganizationID) != organizationID {
		return CommerceCart{}, fmt.Errorf("%w: carrinho %s", ErrCommerceCartMissing, strings.TrimSpace(cartID))
	}
	return carrinho, nil
}

// Carts lista os carrinhos da organizacao (copia).
func (s *CommerceStore) Carts(organizationID string) []CommerceCart {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := []CommerceCart{}
	for _, carrinho := range s.carts {
		if consentScope(carrinho.OrganizationID) == organizationID {
			resultado = append(resultado, carrinho)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].CreatedAt.Before(resultado[j].CreatedAt) })
	return resultado
}

// UpsertCartItem adiciona/atualiza um item do carrinho e cria o carrinho quando o ID vem vazio.
//
// Quantidade ZERO remove o item - remover com o mesmo caminho de adicionar evita uma segunda API para a mesma
// intencao. O SKU precisa existir no catalogo e estar ATIVO: carrinho nao vende o que a loja nao vende.
func (s *CommerceStore) UpsertCartItem(organizationID, cartID, sku string, quantity int64, now time.Time) (CommerceCart, error) {
	organizationID = consentScope(organizationID)
	sku = strings.TrimSpace(sku)
	if quantity < 0 {
		return CommerceCart{}, fmt.Errorf("%w: quantidade negativa (%d)", ErrCommerceReturnQuantity, quantity)
	}
	var unitario int64
	encontrado := false
	for _, product := range s.Products(organizationID) {
		if product.SKU == sku {
			unitario, encontrado = product.PriceCents, true
			break
		}
	}
	if !encontrado {
		return CommerceCart{}, fmt.Errorf("%w: sku %q", ErrCommerceProductMissing, sku)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.carts == nil {
		s.carts = map[string]CommerceCart{}
	}
	carrinho, ok := s.carts[strings.TrimSpace(cartID)]
	if !ok {
		if strings.TrimSpace(cartID) != "" {
			// ID pedido que nao existe NESTA organizacao: mesma recusa de inexistente.
			return CommerceCart{}, fmt.Errorf("%w: carrinho %s", ErrCommerceCartMissing, strings.TrimSpace(cartID))
		}
		carrinho = CommerceCart{ID: "cart_" + uuid.NewString(), OrganizationID: organizationID, Status: CommerceCartOpen, CreatedAt: now}
	}
	if consentScope(carrinho.OrganizationID) != organizationID {
		return CommerceCart{}, fmt.Errorf("%w: carrinho %s", ErrCommerceCartMissing, strings.TrimSpace(cartID))
	}
	if carrinho.Status != CommerceCartOpen {
		return CommerceCart{}, fmt.Errorf("%w: carrinho %s esta %s", ErrCommerceCartClosed, carrinho.ID, carrinho.Status)
	}
	atualizado := false
	itens := make([]CommerceCartItem, 0, len(carrinho.Items)+1)
	for _, item := range carrinho.Items {
		if item.SKU != sku {
			itens = append(itens, item)
			continue
		}
		atualizado = true
		if quantity > 0 {
			itens = append(itens, CommerceCartItem{SKU: sku, Quantity: quantity, UnitPriceCents: unitario})
		}
	}
	if !atualizado && quantity > 0 {
		itens = append(itens, CommerceCartItem{SKU: sku, Quantity: quantity, UnitPriceCents: unitario})
	}
	carrinho.Items = itens
	carrinho.UpdatedAt = now
	s.carts[carrinho.ID] = carrinho
	if err := s.persistLocked(organizationID); err != nil {
		delete(s.carts, carrinho.ID)
		return CommerceCart{}, err
	}
	return carrinho, nil
}

// CartTotalCents soma o carrinho.
func (s *CommerceStore) CartTotalCents(organizationID, cartID string) int64 {
	carrinho, err := s.Cart(organizationID, cartID)
	if err != nil {
		return 0
	}
	var total int64
	for _, item := range carrinho.Items {
		total += item.UnitPriceCents * item.Quantity
	}
	return total
}

// ConvertCart transforma o carrinho em pedido com CHECKOUT IDEMPOTENTE pela chave informada.
//
// A ordem importa: o PEDIDO e criado primeiro (efeito, com a idempotencia do proprio checkout) e o carrinho so
// entao e marcado convertido. Se a marcacao falhar, repetir a conversao com a MESMA chave devolve o pedido
// existente em vez de criar um segundo.
func (s *CommerceStore) ConvertCart(organizationID, cartID, idempotencyKey string, now time.Time) (CommerceOrder, CommerceCart, error) {
	carrinho, err := s.Cart(organizationID, cartID)
	if err != nil {
		return CommerceOrder{}, CommerceCart{}, err
	}
	if carrinho.Status != CommerceCartOpen {
		return CommerceOrder{}, carrinho, fmt.Errorf("%w: carrinho %s esta %s", ErrCommerceCartClosed, carrinho.ID, carrinho.Status)
	}
	if len(carrinho.Items) == 0 {
		return CommerceOrder{}, carrinho, fmt.Errorf("%w: carrinho %s esta vazio", ErrCommerceOrderState, carrinho.ID)
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CommerceOrder{}, carrinho, fmt.Errorf("%w: conversao de carrinho exige chave de idempotencia", ErrCommerceOrderState)
	}
	itens := make([]CommerceOrderItem, 0, len(carrinho.Items))
	for _, item := range carrinho.Items {
		itens = append(itens, CommerceOrderItem{SKU: item.SKU, Quantity: item.Quantity})
	}
	pedido, err := s.CreateOrder(CommerceOrderRequest{
		OrganizationID: consentScope(organizationID), IdempotencyKey: idempotencyKey, Items: itens,
	}, now)
	if err != nil {
		return CommerceOrder{}, carrinho, err
	}
	s.mu.Lock()
	carrinho.Status = CommerceCartConverted
	carrinho.OrderID = pedido.ID
	carrinho.UpdatedAt = now
	s.carts[carrinho.ID] = carrinho
	persistErr := s.persistLocked(organizationID)
	s.mu.Unlock()
	if persistErr != nil {
		// O pedido EXISTE (efeito real). Devolver o erro dizendo isso e' melhor do que fingir que nada aconteceu.
		return pedido, carrinho, fmt.Errorf("pedido %s criado, mas o carrinho nao foi marcado convertido: %w", pedido.Number, persistErr)
	}
	return pedido, carrinho, nil
}

// ProductBySKU devolve o produto da organização pela SKU (chave natural vinda de sistemas de origem).
func (s *CommerceStore) ProductBySKU(organizationID, sku string) (CommerceProduct, bool) {
	organizationID = consentScope(organizationID)
	alvo := strings.TrimSpace(sku)
	s.mu.RLock()
	defer s.mu.RUnlock()
	// E-363: O(1). A chave natural (organização, SKU) tem índice; varrer o mapa inteiro era a outra metade do custo.
	if id, existe := s.IDPorSKULocked(organizationID, alvo); existe {
		return s.products[id], true
	}
	return CommerceProduct{}, false
}

// RemoveProduct retira o produto do catálogo da organização, persistindo a remoção.
//
// Usado pelo ROLLBACK do onboarding de empresa existente (E-327): uma importação precisa poder ser desfeita, e
// desfazer significa o produto deixar de existir — não apenas ficar inativo.
func (s *CommerceStore) RemoveProduct(organizationID, sku string) error {
	organizationID = consentScope(organizationID)
	alvo := strings.TrimSpace(sku)
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, existe := s.IDPorSKULocked(organizationID, alvo); existe {
		produto := s.products[id]
		{
			s.desindexarProdutoLocked(organizationID, alvo)
			delete(s.products, id)
			if err := s.persistLocked(organizationID); err != nil {
				s.products[id] = produto // rollback em memória se a gravação falhar
				s.indexarProdutoLocked(produto)
				return err
			}
			return nil
		}
	}
	return nil
}

// existeSKULocked informa se o SKU já existe para a organização. Exige o lock de escrita (ou leitura) já tomado.
func existeSKULocked(s *CommerceStore, organizationID, sku string) bool {
	// E-363: O(1). Era varredura do mapa inteiro — a metade quadrática do enchimento medido em E-360.
	_, existe := s.IDPorSKULocked(organizationID, strings.TrimSpace(sku))
	return existe
}

// indexarProdutoLocked registra o SKU no índice. Exige o lock de escrita tomado.
func (s *CommerceStore) indexarProdutoLocked(produto CommerceProduct) {
	if s.produtosPorChave == nil {
		s.produtosPorChave = map[string]map[string]string{}
	}
	porSKU := s.produtosPorChave[produto.OrganizationID]
	if porSKU == nil {
		porSKU = map[string]string{}
		s.produtosPorChave[produto.OrganizationID] = porSKU
	}
	porSKU[produto.SKU] = produto.ID
}

// desindexarProdutoLocked remove o SKU do índice. Exige o lock de escrita tomado.
func (s *CommerceStore) desindexarProdutoLocked(organizacao, sku string) {
	porSKU := s.produtosPorChave[organizacao]
	if porSKU == nil {
		return
	}
	delete(porSKU, sku)
	if len(porSKU) == 0 {
		delete(s.produtosPorChave, organizacao)
	}
}

// IDPorSKULocked devolve o id do produto em O(1). Exige o lock (leitura ou escrita) tomado.
func (s *CommerceStore) IDPorSKULocked(organizacao, sku string) (string, bool) {
	porSKU := s.produtosPorChave[organizacao]
	if porSKU == nil {
		return "", false
	}
	id, ok := porSKU[sku]
	return id, ok
}

// AcervoLocked devolve quantos produtos a organização tem, em O(1). Exige o lock tomado.
func (s *CommerceStore) AcervoLocked(organizacao string) int {
	return len(s.produtosPorChave[organizacao])
}
