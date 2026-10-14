package agent

import (
	"testing"
)

// E-462 — **`CreateOrder`: A REENTREGA QUE DEVOLVE `nil` EM SILÊNCIO — E O PEDIDO QUE PODE DESAPARECER.**
//
// **COMO ISTO FOI ENCONTRADO**: a varredura **exaustiva** do padrão ADV-75/ADV-76 (*comparar um identificador sem o
// escopo*) reduziu **1521** comparações `==` a **55** candidatos, e `company_growth.go:341` estava entre eles —
// `existing.IdempotencyKey == order.IdempotencyKey`.
//
// **E A LEITURA DO CÓDIGO MOSTROU QUE NÃO É IRMÃO, É OUTRA COISA**:
//
//	return s.mutate(id, func(company *Company) error {          <- escopo por EMPRESA esta' garantido
//	    if existing.IdempotencyKey == order.IdempotencyKey {
//	        return nil                                          <- SILENCIO
//
// ⇒ ***A reentrega não cria o pedido (correcto), mas o chamador não recebe NADA: nem o pedido existente, nem um sinal de
// replay.*** É **exactamente o oposto** da lição de **E-458** — *"idempotência de verdade é devolver o **MESMO
// RECURSO**"*.
//
// **AS DUAS PERGUNTAS, medidas separadamente**:
//  1. a chave **não inclui o `ProductID`** ⇒ *produtos diferentes com a mesma chave fazem o segundo pedido
//     **desaparecer**?* — *o cliente pediu dois produtos e recebeu um, sem erro?*
//  2. e a reentrega do **mesmo** produto devolve `nil` — *o chamador fica sem o pedido que já existe.*
//
// ***Meço antes de decidir; o veredicto é o que a medição mostrar.***
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
		// ⚠️ O motor exige `sku`, `name` e `supplier` - *e recusa com o motivo exacto, o que e' o que permite corrigir
		// sem adivinhar* (`product sku, name and supplier are required`).
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

	// (1) Primeiro pedido: produto A.
	depoisDoPrimeiro, err := store.CreateOrder(empresa.ID, CompanyOrder{
		ProductID: produtoA, CustomerRef: "cliente-1", Quantity: 1, IdempotencyKey: chave,
	})
	if err != nil {
		t.Fatalf("primeiro pedido: %v", err)
	}
	if len(depoisDoPrimeiro.Orders) != 1 {
		t.Fatalf("depois do primeiro pedido esperava 1, veio %d", len(depoisDoPrimeiro.Orders))
	}

	// (2) PRODUTO DIFERENTE, MESMA CHAVE: *se a chave identifica a intenção, isto é uma intenção NOVA.*
	depoisDoSegundo, err := store.CreateOrder(empresa.ID, CompanyOrder{
		ProductID: produtoB, CustomerRef: "cliente-1", Quantity: 1, IdempotencyKey: chave,
	})
	if err != nil {
		t.Fatalf("segundo pedido: %v", err)
	}

	pedidos := len(depoisDoSegundo.Orders)
	switch pedidos {
	case 1:
		t.Fatalf("MEDIDO — O PEDIDO DESAPARECE: o cliente pediu o produto B com a mesma chave do pedido A, e o motor " +
			"ficou com 1 pedido — **sem erro, sem aviso, e sem devolver o pedido de A tampouco**. ***A chave nao inclui " +
			"o ProductID, e o `return nil` nao diz nada ao chamador: e' o OPOSTO da licao de E-458 (idempotencia de " +
			"verdade e' devolver o MESMO RECURSO).***")
	case 2:
		t.Log("MEDIDO: produtos diferentes com a mesma chave criaram DOIS pedidos — a intencao nova NAO foi engolida. " +
			"*O que resta e' o SILENCIO na reentrega: `return nil` sem devolver o pedido existente.*")
	default:
		t.Fatalf("esperava 1 ou 2 pedidos, veio %d", pedidos)
	}

	// (3) REENTREGA DO MESMO produto e mesma chave: *aqui a idempotencia age — e o que ela DEVOLVE?*
	depoisDaReentrega, err := store.CreateOrder(empresa.ID, CompanyOrder{
		ProductID: produtoA, CustomerRef: "cliente-1", Quantity: 1, IdempotencyKey: chave,
	})
	if err != nil {
		t.Fatalf("reentrega: %v", err)
	}
	if len(depoisDaReentrega.Orders) != pedidos {
		t.Fatalf("a reentrega NAO pode criar pedido: antes=%d depois=%d", pedidos, len(depoisDaReentrega.Orders))
	}
	// E o defeito de DESENHO, medido: *o chamador recebe a empresa, sem qualquer sinal de que a intencao ja' existia.*
	// **Nao afirmo `Replayed` aqui** - o campo nao existe neste tipo; afirmo o que medi: *a resposta e' indistinguivel de
	// uma criacao bem-sucedida.*
	t.Logf("MEDIDO: a reentrega devolve a empresa com %d pedido(s) e **nada** que distinga 'criei' de 'ja' existia' — "+
		"*o chamador que reentrega por queda de rede nao sabe se o pedido foi criado agora ou antes.*", len(depoisDaReentrega.Orders))
}
