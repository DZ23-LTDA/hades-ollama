package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// TestCheckoutIdempotenteEConciliacaoSobrevivemAoReinicio prova, no motor de comércio PERSISTENTE, que um
// REINÍCIO REAL do processo (novo store sobre o MESMO diretório) não perde o pedido, não perde o livro de
// estoque e NÃO duplica a venda quando o cliente reenvia a MESMA chave de idempotência.
//
// O caso importa porque idempotência só vale de verdade se atravessar o reinício: em memória, a segunda
// tentativa após um restart pareceria nova e geraria cobrança/estoque duplicados.
func TestCheckoutIdempotenteEConciliacaoSobrevivemAoReinicio(t *testing.T) {
	raiz := t.TempDir()
	org := "org-comercio-reinicio"
	agora := time.Now().UTC()

	loja, err := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
	if err != nil {
		t.Fatalf("abrir motor de comércio: %v", err)
	}
	if _, err := loja.UpsertProduct(CommerceProduct{
		OrganizationID: org,
		SKU:            "SKU-CAFE",
		Name:           "Café 500g",
		PriceCents:     2500,
		Currency:       "BRL",
		Active:         true,
	}, agora); err != nil {
		t.Fatalf("cadastrar produto: %v", err)
	}
	if _, err := loja.RecordStock(org, "SKU-CAFE", 10, "entrada", "carga-inicial", agora); err != nil {
		t.Fatalf("lançar estoque: %v", err)
	}

	pedido := CommerceOrderRequest{
		OrganizationID: org,
		Items:          []CommerceOrderItem{{SKU: "SKU-CAFE", Quantity: 2}},
		Currency:       "BRL",
		IdempotencyKey: "checkout-abc-123",
	}
	criado, err := loja.CreateOrder(pedido, agora)
	if err != nil {
		t.Fatalf("criar pedido: %v", err)
	}
	saldoAntes := loja.Stock(org, "SKU-CAFE")
	if saldoAntes.MovementCount == 0 {
		t.Fatal("estoque sem lançamentos: o teste perdeu o propósito")
	}

	// REINÍCIO REAL: novo motor sobre o mesmo diretório de dados.
	reiniciada, err := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
	if err != nil {
		t.Fatalf("reabrir motor (reinício): %v", err)
	}

	// 1. O pedido sobreviveu.
	recuperado, err := reiniciada.Order(org, criado.ID)
	if err != nil {
		t.Fatalf("pedido NÃO sobreviveu ao reinício: %v", err)
	}
	if recuperado.IdempotencyKey != pedido.IdempotencyKey {
		t.Fatalf("chave de idempotência perdida no reinício: %q", recuperado.IdempotencyKey)
	}

	// 2. A MESMA chave, depois do reinício, devolve o MESMO pedido e NÃO cria outro.
	repetido, err := reiniciada.CreateOrder(pedido, agora.Add(time.Minute))
	if err != nil {
		t.Fatalf("repetir checkout após reinício: %v", err)
	}
	if repetido.ID != criado.ID {
		t.Fatalf("checkout repetido após reinício criou OUTRO pedido: %s != %s", repetido.ID, criado.ID)
	}
	if total := len(reiniciada.Orders(org)); total != 1 {
		t.Fatalf("esperava 1 pedido após reinício e reenvio, encontrei %d (venda duplicada)", total)
	}

	// 3. O livro de estoque é idêntico: saldo é derivado e não pode mudar pelo reinício.
	saldoDepois := reiniciada.Stock(org, "SKU-CAFE")
	if saldoDepois != saldoAntes {
		t.Fatalf("livro de estoque mudou com o reinício: antes=%+v depois=%+v", saldoAntes, saldoDepois)
	}

	// 4. Conciliação sobrevive e continua idempotente.
	if _, err := reiniciada.ConfirmPayment(org, criado.ID, CommercePayment{
		Method: "pix", Reference: "PIX-REF-9", AmountCents: recuperado.TotalCents, RecordedAt: agora,
	}, "operador", agora); err != nil {
		t.Fatalf("registrar quitação: %v", err)
	}
	conciliado, err := reiniciada.Reconcile(org, criado.ID, "PIX-REF-9", agora)
	if err != nil {
		t.Fatalf("conciliar: %v", err)
	}
	if conciliado.ReconciledRef == "" {
		t.Fatal("conciliação não registrou referência")
	}

	terceira, err := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
	if err != nil {
		t.Fatalf("terceiro reinício: %v", err)
	}
	final, err := terceira.Order(org, criado.ID)
	if err != nil {
		t.Fatalf("pedido sumiu no terceiro reinício: %v", err)
	}
	if final.ReconciledRef != conciliado.ReconciledRef {
		t.Fatalf("conciliação perdida no reinício: %q != %q", final.ReconciledRef, conciliado.ReconciledRef)
	}
	if final.Payment == nil || final.Payment.Reference != "PIX-REF-9" {
		t.Fatalf("quitação perdida no reinício: %+v", final.Payment)
	}
	if _, err := terceira.Reconcile(org, criado.ID, "PIX-REF-9", agora.Add(2*time.Minute)); err != nil {
		t.Fatalf("reconciliar de novo deveria ser idempotente: %v", err)
	}
	if total := len(terceira.Orders(org)); total != 1 {
		t.Fatalf("reenvio após conciliação duplicou pedido: %d pedidos", total)
	}
	t.Logf("REINÍCIO: pedido=%s chave=%s estoque=%+v conciliado=%q",
		criado.ID, pedido.IdempotencyKey, saldoDepois, final.ReconciledRef)
}
