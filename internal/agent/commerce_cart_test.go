package agent

import (
	"errors"
	"testing"
	"time"
)

// Estagio 24 — CARRINHO PERSISTENTE (E-260). O limite declarado dizia "sem carrinho persistente".
//
// Propriedades travadas: item so entra com SKU do catalogo; quantidade zero REMOVE; carrinho convertido nao aceita
// mais item; a conversao e' IDEMPOTENTE pela chave (repetir nao cria segundo pedido); outro tenant recebe a MESMA
// recusa de inexistente; e o carrinho SOBREVIVE ao reinicio.
func TestCommerceCartPersisteEConverteComIdempotencia(t *testing.T) {
	root := t.TempDir()
	store, err := NewCommerceStoreAt(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	org := LocalOrganizationID
	if _, err := store.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordStock(org, "CAM-001", 10, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}

	// SKU fora do catalogo nao entra no carrinho.
	if _, err := store.UpsertCartItem(org, "", "NAO-EXISTE", 1, now); !errors.Is(err, ErrCommerceProductMissing) {
		t.Fatalf("SKU fora do catalogo precisa ser recusado: %v", err)
	}
	carrinho, err := store.UpsertCartItem(org, "", "CAM-001", 2, now)
	if err != nil {
		t.Fatalf("adicionar item: %v", err)
	}
	if len(carrinho.Items) != 1 || carrinho.Items[0].Quantity != 2 {
		t.Fatalf("carrinho precisa ter o item com quantidade 2: %+v", carrinho.Items)
	}
	if total := store.CartTotalCents(org, carrinho.ID); total != 15800 {
		t.Fatalf("total do carrinho: %d", total)
	}
	// Atualizar a quantidade nao duplica a linha.
	carrinho, err = store.UpsertCartItem(org, carrinho.ID, "CAM-001", 3, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(carrinho.Items) != 1 || carrinho.Items[0].Quantity != 3 {
		t.Fatalf("atualizar quantidade nao pode duplicar a linha: %+v", carrinho.Items)
	}
	// Quantidade ZERO remove.
	carrinho, err = store.UpsertCartItem(org, carrinho.ID, "CAM-001", 0, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(carrinho.Items) != 0 {
		t.Fatalf("quantidade zero precisa remover o item: %+v", carrinho.Items)
	}
	// Carrinho vazio nao converte.
	if _, _, err := store.ConvertCart(org, carrinho.ID, "chave-vazia", now); !errors.Is(err, ErrCommerceOrderState) {
		t.Fatalf("carrinho vazio precisa recusar conversao: %v", err)
	}

	if _, err := store.UpsertCartItem(org, carrinho.ID, "CAM-001", 2, now); err != nil {
		t.Fatal(err)
	}
	pedido, convertido, err := store.ConvertCart(org, carrinho.ID, "conversao-1", now)
	if err != nil {
		t.Fatalf("converter carrinho: %v", err)
	}
	if convertido.Status != CommerceCartConverted || convertido.OrderID != pedido.ID {
		t.Fatalf("carrinho convertido precisa apontar para o pedido: %+v", convertido)
	}
	// IDEMPOTENCIA: repetir a conversao com a MESMA chave nao cria segundo pedido.
	segundo, _, err := store.ConvertCart(org, carrinho.ID, "conversao-1", now)
	if err == nil {
		t.Fatalf("carrinho ja convertido nao pode converter de novo (pedido novo=%s)", segundo.Number)
	}
	if !errors.Is(err, ErrCommerceCartClosed) {
		t.Fatalf("esperava carrinho fechado: %v", err)
	}
	if pedidos := store.Orders(org); len(pedidos) != 1 {
		t.Fatalf("a conversao nao pode duplicar pedido: %d pedidos", len(pedidos))
	}

	// OUTRO TENANT: mesma recusa de inexistente.
	_, errIntruso := store.Cart("org-intrusa", carrinho.ID)
	_, errInexistente := store.Cart("org-intrusa", "cart_inexistente")
	if errIntruso == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir: %v / %v", errIntruso, errInexistente)
	}
	if !errors.Is(errIntruso, ErrCommerceCartMissing) || !errors.Is(errInexistente, ErrCommerceCartMissing) {
		t.Fatalf("as duas recusas precisam ser do MESMO tipo: %v / %v", errIntruso, errInexistente)
	}

	// PERSISTENCIA: reinicio real sobre o mesmo diretorio.
	reaberta, err := NewCommerceStoreAt(root)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, err := reaberta.Cart(org, carrinho.ID)
	if err != nil {
		t.Fatalf("o carrinho precisa sobreviver ao reinicio: %v", err)
	}
	if recuperado.Status != CommerceCartConverted || recuperado.OrderID != pedido.ID {
		t.Fatalf("o carrinho recuperado perdeu estado: %+v", recuperado)
	}
	if len(recuperado.Items) != 1 || recuperado.Items[0].UnitPriceCents != 7900 {
		t.Fatalf("o carrinho recuperado perdeu itens/preco: %+v", recuperado.Items)
	}
	if outros := reaberta.Carts("org-intrusa"); len(outros) != 0 {
		t.Fatalf("outro tenant nao pode enxergar carrinho: %d", len(outros))
	}
}
