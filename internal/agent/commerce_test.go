package agent

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// Estágio 24 — motor de comércio.
//
// As propriedades travadas aqui:
//  1. IDEMPOTÊNCIA de checkout: a mesma chave devolve o MESMO pedido (reenviar não
//     pode virar duas cobranças);
//  2. NÃO SE VENDE O QUE NÃO EXISTE: o pedido reserva e falha sem saldo disponível;
//  3. o estoque é LIVRO DE MOVIMENTOS: reserva não reduz o físico, saída reduz;
//  4. ZERO DADO DE CARTÃO: referência com aparência de número de cartão é recusada;
//  5. o preço vem do CATÁLOGO, não do pedido;
//  6. conciliação idempotente;
//  7. cancelamento devolve o estoque e pedido atendido não pode ser cancelado;
//  8. isolamento por organização.
func commerceStore(t *testing.T) *CommerceStore {
	t.Helper()
	store := NewCommerceStore()
	now := time.Now().UTC()
	for _, product := range []CommerceProduct{
		{OrganizationID: "org-a", SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true},
		{OrganizationID: "org-a", SKU: "CAN-002", Name: "Caneca", PriceCents: 3500, Active: true},
	} {
		if _, err := store.UpsertProduct(product, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.RecordStock("org-a", "CAM-001", 10, CommerceMovementEntry, "compra inicial", now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordStock("org-a", "CAN-002", 4, CommerceMovementEntry, "compra inicial", now); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCommerceCheckoutIsIdempotent(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	request := CommerceOrderRequest{
		OrganizationID: "org-a", ContactID: "ct_1", IdempotencyKey: "checkout-abc",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 2}},
	}
	first, err := store.CreateOrder(request, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Duplicated {
		t.Fatal("o primeiro pedido não é duplicado")
	}
	// Reenvio do MESMO checkout: precisa devolver o MESMO pedido, não um segundo.
	second, err := store.CreateOrder(request, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || !second.Duplicated {
		t.Fatalf("a mesma chave precisa devolver o MESMO pedido marcado como duplicado: %+v", second)
	}
	if orders := store.Orders("org-a"); len(orders) != 1 {
		t.Fatalf("reenviar o checkout não pode criar um segundo pedido: %d", len(orders))
	}
	// E o estoque foi reservado UMA vez.
	balance := store.Stock("org-a", "CAM-001")
	if balance.OnHand != 10 || balance.Reserved != 2 || balance.Available != 8 {
		t.Fatalf("a reserva precisa acontecer uma única vez: %+v", balance)
	}
	// Sem chave de idempotência o pedido é recusado: não dá para garantir unicidade.
	if _, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now); !errors.Is(err, ErrCommerceIdempotencyKey) {
		t.Fatalf("sem chave o pedido precisa ser recusado, veio %v", err)
	}
	// Chave diferente para o mesmo conteúdo é um pedido NOVO (decisão do cliente).
	outro, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "checkout-def",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if outro.ID == first.ID {
		t.Fatal("chave diferente é um pedido diferente")
	}
}

func TestCommerceNeverSellsWhatItDoesNotHave(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	// Há 4 canecas: pedir 5 precisa falhar.
	if _, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "k1",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 5}},
	}, now); !errors.Is(err, ErrCommerceInsufficient) {
		t.Fatalf("vender mais do que existe precisa falhar, veio %v", err)
	}
	// Pedir as 4 funciona e esgota o disponível.
	if _, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "k2",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 4}},
	}, now); err != nil {
		t.Fatal(err)
	}
	balance := store.Stock("org-a", "CAN-002")
	if balance.Available != 0 || balance.OnHand != 4 || balance.Reserved != 4 {
		t.Fatalf("reservar tudo precisa zerar o DISPONÍVEL sem zerar o físico: %+v", balance)
	}
	// Com o disponível zerado, outro pedido falha mesmo sendo de 1.
	if _, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "k3",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 1}},
	}, now); !errors.Is(err, ErrCommerceInsufficient) {
		t.Fatalf("sem disponível o pedido precisa falhar, veio %v", err)
	}
	// Movimento negativo direto também respeita o disponível.
	if _, err := store.RecordStock("org-a", "CAN-002", -1, CommerceMovementReserve, "teste", now); !errors.Is(err, ErrCommerceInsufficient) {
		t.Fatalf("movimento além do disponível precisa falhar, veio %v", err)
	}
}

func TestCommerceStockLedgerKeepsPhysicalAndReservedSeparate(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	order, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "full-cycle",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 3}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Cancelar devolve o disponível e mantém o físico.
	if _, err := store.Cancel("org-a", order.ID, "cliente desistiu", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	balance := store.Stock("org-a", "CAM-001")
	if balance.OnHand != 10 || balance.Reserved != 0 || balance.Available != 10 {
		t.Fatalf("cancelar precisa devolver o disponível: %+v", balance)
	}
	// Ciclo completo: pedido, pagamento e atendimento reduzem o FÍSICO.
	paid, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "ship-it",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 3}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfirmPayment("org-a", paid.ID, CommercePayment{Method: CommercePaymentPix, Reference: "pix-ref-1"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Fulfill("org-a", paid.ID, now); err != nil {
		t.Fatal(err)
	}
	balance = store.Stock("org-a", "CAM-001")
	if balance.OnHand != 7 || balance.Reserved != 0 || balance.Available != 7 {
		t.Fatalf("atender precisa reduzir o FÍSICO e liberar a reserva: %+v", balance)
	}
	// O livro de movimentos registra TUDO, e é auditável.
	movements := store.Movements("org-a", "CAM-001")
	if len(movements) < 5 {
		t.Fatalf("o livro precisa registrar entrada, reservas, liberação e saída: %+v", movements)
	}
	kinds := map[string]int{}
	for _, movement := range movements {
		kinds[movement.Kind]++
	}
	for _, expected := range []string{CommerceMovementEntry, CommerceMovementReserve, CommerceMovementRelease, CommerceMovementShipped} {
		if kinds[expected] == 0 {
			t.Fatalf("o livro precisa ter movimento do tipo %s: %+v", expected, kinds)
		}
	}
	// Pedido atendido não pode ser cancelado.
	if _, err := store.Cancel("org-a", paid.ID, "tarde demais", now); !errors.Is(err, ErrCommerceOrderState) {
		t.Fatalf("pedido atendido não pode ser cancelado, veio %v", err)
	}
}

func TestCommerceRefusesCardDataAndUnknownMethods(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	order, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "pay-1",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Número de cartão (mesmo sem separadores) é RECUSADO: o estágio proíbe processar
	// cartão sem conformidade, então o sistema não guarda PAN nem por acidente.
	for _, reference := range []string{"4111111111111111", "4111 1111 1111 1111", "4111-1111-1111-1111"} {
		err := ValidatePayment(CommercePayment{Method: CommercePaymentSandbox, Reference: reference})
		if !errors.Is(err, ErrCommerceCardDataRefused) {
			t.Fatalf("referência %q precisa ser recusada como dado de cartão, veio %v", reference, err)
		}
		if _, err := store.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: reference}, "ana", now); !errors.Is(err, ErrCommerceCardDataRefused) {
			t.Fatalf("o pedido não pode aceitar dado de cartão (%q), veio %v", reference, err)
		}
	}
	// Método desconhecido é recusado.
	if _, err := store.ConfirmPayment("org-a", order.ID, CommercePayment{Method: "cartao_credito", Reference: "ref-1"}, "ana", now); !errors.Is(err, ErrCommerceInvalid) {
		t.Fatalf("método não aceito precisa falhar, veio %v", err)
	}
	// Referência comum funciona.
	if _, err := store.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentPix, Reference: "E12345678"}, "ana", now); err != nil {
		t.Fatalf("referência Pix precisa ser aceita: %v", err)
	}
	// Valor divergente do pedido é recusado.
	other, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "pay-2",
		Items: []CommerceOrderItem{{SKU: "CAN-002", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfirmPayment("org-a", other.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "ref-x", AmountCents: 1}, "ana", now); !errors.Is(err, ErrCommerceTotalMismatch) {
		t.Fatalf("valor divergente precisa falhar, veio %v", err)
	}
}

func TestCommercePriceComesFromCatalogAndReconciliationIsIdempotent(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	// O pedido manda um preço FALSO de 1 centavo: o motor precisa cobrar o catálogo.
	order, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "price-guard",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 2, UnitCents: 1, TotalCents: 2}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if order.Items[0].UnitCents != 7900 || order.TotalCents != 15800 {
		t.Fatalf("o preço precisa vir do CATÁLOGO, não do pedido: %+v", order)
	}
	if _, err := store.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "ref-cat"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	// Conciliação: primeira vez grava; repetir a MESMA referência é no-op; outra é conflito.
	reconciled, err := store.Reconcile("org-a", order.ID, "extrato-2026-10", now)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.ReconciledRef != "extrato-2026-10" {
		t.Fatalf("a conciliação precisa ser registrada: %+v", reconciled)
	}
	if again, err := store.Reconcile("org-a", order.ID, "extrato-2026-10", now); err != nil || again.ReconciledRef != "extrato-2026-10" {
		t.Fatalf("repetir a mesma conciliação precisa ser no-op: %v", err)
	}
	if _, err := store.Reconcile("org-a", order.ID, "extrato-2026-11", now); !errors.Is(err, ErrCommerceReconcileConflict) {
		t.Fatalf("conciliação com outra referência precisa ser conflito, veio %v", err)
	}
	// Resumo: receita contabilizada.
	summary := store.Summary("org-a")
	if summary.Products != 2 || summary.Paid != 1 || summary.RevenueCents != 15800 {
		t.Fatalf("resumo incorreto: %+v", summary)
	}
}

func TestCommerceIsScopedByOrganizationAndRejectsUnknownProduct(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	// SKU de outra organização (ou inexistente) falha.
	if _, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-b", IdempotencyKey: "kb",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now); !errors.Is(err, ErrCommerceProductMissing) {
		t.Fatalf("o catálogo precisa ser por organização, veio %v", err)
	}
	order, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "ka",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Pedido da org-a não pode ser lido nem alterado pela org-b.
	if _, err := store.Order("org-b", order.ID); !errors.Is(err, ErrCommerceOrderMissing) {
		t.Fatalf("pedido de outra organização não pode ser lido, veio %v", err)
	}
	if _, err := store.ConfirmPayment("org-b", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r"}, "ana", now); !errors.Is(err, ErrCommerceOrderMissing) {
		t.Fatalf("pedido de outra organização não pode ser pago, veio %v", err)
	}
	if orders := store.Orders("org-b"); len(orders) != 0 {
		t.Fatalf("a listagem precisa ser escopada: %+v", orders)
	}
	// O estoque da outra organização é próprio: o SKU dela não existe aqui.
	if _, err := store.RecordStock("org-b", "CAM-001", 5, CommerceMovementEntry, "x", now); !errors.Is(err, ErrCommerceProductMissing) {
		t.Fatalf("estoque precisa ser por organização, veio %v", err)
	}
}

// O checkout idempotente precisa valer também sob CONCORRÊNCIA: dez requisições
// simultâneas do mesmo checkout não podem gerar dez pedidos.
func TestCommerceIdempotencyHoldsUnderConcurrency(t *testing.T) {
	store := commerceStore(t)
	now := time.Now().UTC()
	var wg sync.WaitGroup
	ids := make([]string, 10)
	for index := range 10 {
		wg.Add(1)
		go func(position int) {
			defer wg.Done()
			order, err := store.CreateOrder(CommerceOrderRequest{
				OrganizationID: "org-a", IdempotencyKey: "concurrent-checkout",
				Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
			}, now)
			if err == nil {
				ids[position] = order.ID
			}
		}(index)
	}
	wg.Wait()
	unique := map[string]bool{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			t.Fatal("toda requisição concorrente precisa devolver um pedido")
		}
		unique[id] = true
	}
	if len(unique) != 1 {
		t.Fatalf("a mesma chave sob concorrência precisa gerar UM pedido: %d", len(unique))
	}
	if orders := store.Orders("org-a"); len(orders) != 1 {
		t.Fatalf("um pedido apenas: %d", len(orders))
	}
	balance := store.Stock("org-a", "CAM-001")
	if balance.Reserved != 1 {
		t.Fatalf("a reserva precisa ser de uma unidade só: %+v", balance)
	}
}
