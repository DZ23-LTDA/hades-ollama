package agent

import (
	"testing"
)

// E-467 — **`CreateOrder`: O SILÊNCIO CORRIGIDO — E A REENTREGA QUE AGORA DIZ O QUE ACONTECEU.**
//
// **COMO ISTO FOI ENCONTRADO**: a varredura **exaustiva** do padrão ADV-75/ADV-76 (*comparar um identificador sem o
// escopo*) reduziu **1521** comparações `==` a **55** candidatos, e `company_growth.go:341` estava entre eles.
//
// **MEDIDO, em duas partes**:
//  1. **E-463 (ADV-77)** — a chave **não incluía o `ProductID`** ⇒ *um pedido do produto B **desaparecia** por causa de
//     um pedido do produto A, sem erro, sem aviso* ✓✓. ***O cliente pediu dois produtos e recebeu um.***
//  2. **E-467** — e a reentrega devolvia `nil` **em silêncio**: *nem o pedido existente, nem um sinal de replay* ✗ ⇒
//     ***o chamador que reentrega por queda de rede não sabia se o pedido foi criado agora ou antes.***
//
// **E O CONTRASTE QUE AUTORIZOU A MUDANÇA (E-466)**: `live_commerce.go:611` devolve `anterior` (*"replay: devolve o
// MESMO evento, sem segundo efeito"*) e `sales_workforce.go:512` devolve `existing` ⇒ ***dois vizinhos devolvem o
// recurso existente; `CreateOrder` era a excepção*** ⇒ ***não era questão de opinião: era inconsistência dentro do
// repositório*** ✓. E o próprio repo **já tem** `commerce_restart_idempotency_test.go` ⇒ *o padrão é **testar** replay.*
//
// ⇒ o contrato passa a `(Company, CompanyOrder, error)` com **`CompanyOrder.Replayed`** (*espelhando
// `WorkflowRun.Replayed`*), e este teste mede as **quatro** direcções.
func TestIdempotenciaDoPedidoDeProdutoNaCompanyStore(t *testing.T) {
	store, err := NewCompanyStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	empresa, err := store.Create(Company{Name: "Empresa da varredura", OrganizationID: "org-a"})
	if err != nil {
		t.Fatalf("criar empresa: %v", err)
	}

	adicionarProduto := func(nome string) string {
		// ⚠️ O motor exige `sku`, `name` e `supplier` — *e recusa com o motivo exacto, o que permite corrigir sem
		// adivinhar* (`product sku, name and supplier are required`).
		atualizada, err := store.AddProduct(empresa.ID, CompanyProduct{
			SKU: "SKU-" + nome, Name: nome, Supplier: "fornecedor-do-teste",
			PriceCents: 1000, Inventory: 10, Status: "active",
		})
		if err != nil {
			t.Fatalf("adicionar produto %s: %v", nome, err)
		}
		return atualizada.Products[len(atualizada.Products)-1].ID
	}
	produtoA := adicionarProduto("A")
	produtoB := adicionarProduto("B")
	chave := "k-pedido-do-mesmo-evento"

	pedir := func(produto string) (Company, CompanyOrder) {
		empresaAtual, pedido, err := store.CreateOrder(empresa.ID, CompanyOrder{
			ProductID: produto, CustomerRef: "cliente-1", Quantity: 1, IdempotencyKey: chave,
		})
		if err != nil {
			t.Fatalf("pedido do produto %s: %v", produto, err)
		}
		return empresaAtual, pedido
	}

	// (1) CRIAR: o pedido nasce, e **não** é replay.
	depoisDoPrimeiro, primeiro := pedir(produtoA)
	if len(depoisDoPrimeiro.Orders) != 1 {
		t.Fatalf("depois do primeiro pedido esperava 1, veio %d", len(depoisDoPrimeiro.Orders))
	}
	if primeiro.ID == "" || primeiro.Replayed {
		t.Fatalf("o primeiro pedido tem de vir com ID e SEM `Replayed`: %+v", primeiro)
	}

	// (2) **PRODUTO DIFERENTE, MESMA CHAVE** — a medida do E-463. *Se a chave identifica a intenção, isto é intenção NOVA.*
	depoisDoSegundo, segundo := pedir(produtoB)
	pedidos := len(depoisDoSegundo.Orders)
	switch pedidos {
	case 1:
		t.Fatalf("MEDIDO — O PEDIDO DESAPARECE: o cliente pediu o produto B com a mesma chave do pedido A, e o motor " +
			"ficou com 1 pedido — **sem erro, sem aviso**. ***A chave nao inclui o ProductID.***")
	case 2:
		// *O comportamento correcto.*
	default:
		t.Fatalf("esperava 1 ou 2 pedidos, veio %d", pedidos)
	}
	if segundo.ID == "" || segundo.ID == primeiro.ID || segundo.Replayed {
		t.Fatalf("o pedido de um produto DIFERENTE tem de ser NOVO e sem replay: %+v (primeiro=%q)",
			segundo, primeiro.ID)
	}

	// (3) **REENTREGA DO MESMO pedido** — e agora ela **DIZ** o que aconteceu.
	depoisDaReentrega, reentrega := pedir(produtoA)
	if len(depoisDaReentrega.Orders) != pedidos {
		t.Fatalf("a reentrega NAO pode criar pedido: antes=%d depois=%d", pedidos, len(depoisDaReentrega.Orders))
	}
	if !reentrega.Replayed {
		t.Fatalf("ARMADILHA: a reentrega MESMA intencao NAO foi marcada como `Replayed` — *e' exactamente o silencio "+
			"que o E-467 corrigiu: o chamador que reentrega por queda de rede tem de saber se criou ou se ja' existia* "+
			"(reentrega=%+v)", reentrega)
	}
	if reentrega.ID != primeiro.ID {
		t.Fatalf("a reentrega tem de devolver o MESMO pedido: veio %q, o original e' %q — *idempotencia de verdade e' "+
			"devolver o MESMO RECURSO (E-458)*", reentrega.ID, primeiro.ID)
	}

	// (4) E o CONTRATO com o motor: *o pedido devolvido na criacao tem de estar na lista da empresa* — *a resposta nao
	// pode ser uma segunda verdade.*
	achado := false
	for _, existente := range depoisDaReentrega.Orders {
		if existente.ID == reentrega.ID {
			achado = true
		}
	}
	if !achado {
		t.Fatalf("o pedido devolvido (%q) nao esta' na lista da empresa — *a resposta e o motor tem de contar a MESMA "+
			"historia*", reentrega.ID)
	}
}
