package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Estagio 24 — DEVOLUCAO (E-259). O limite declarado dizia "sem devolucao".
//
// Propriedades travadas: so pedido PAGO/ATENDIDO devolve; SKU precisa pertencer ao pedido; nao devolve mais do que
// comprou; repetir a MESMA devolucao e IDEMPOTENTE; outro tenant recebe a MESMA recusa de inexistente; o dinheiro
// NAO se move (o registro diz que o reembolso exige aprovacao humana); e o estado SOBREVIVE ao reinicio.
func TestCommerceReturnDevuelveEstoqueComIdempotenciaEPersistencia(t *testing.T) {
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
	order, err := store.CreateOrder(CommerceOrderRequest{
		OrganizationID: org, IdempotencyKey: "devolucao-1",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 2}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Pedido NAO pago nao pode ser devolvido.
	if _, err := store.RecordReturn(CommerceReturnRequest{OrganizationID: org, OrderID: order.ID, SKU: "CAM-001", Quantity: 1}, now); !errors.Is(err, ErrCommerceOrderState) {
		t.Fatalf("pedido nao pago precisa recusar devolucao: %v", err)
	}
	if _, err := store.ConfirmPayment(org, order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "p-1"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	saldoAntes := store.Stock(org, "CAM-001").Available

	primeira, err := store.RecordReturn(CommerceReturnRequest{OrganizationID: org, OrderID: order.ID, SKU: "CAM-001", Quantity: 1, Reason: "defeito", ActorID: "ana"}, now)
	if err != nil {
		t.Fatalf("devolucao de pedido pago: %v", err)
	}
	if primeira.MovementID == "" {
		t.Fatal("a devolucao precisa registrar o movimento de estoque")
	}
	if primeira.RefundCents != 7900 {
		t.Fatalf("o reembolso devido precisa ser preco x quantidade: %d", primeira.RefundCents)
	}
	if primeira.FinancialSignal == "" {
		t.Fatal("o dinheiro NAO se move: o registro precisa dizer que o reembolso exige aprovacao")
	}
	if depois := store.Stock(org, "CAM-001").Available; depois != saldoAntes+1 {
		t.Fatalf("a devolucao precisa DEVOLVER estoque: %d -> %d", saldoAntes, depois)
	}

	// IDEMPOTENCIA: repetir a mesma devolucao nao devolve estoque duas vezes.
	repetida, err := store.RecordReturn(CommerceReturnRequest{OrganizationID: org, OrderID: order.ID, SKU: "CAM-001", Quantity: 1, Reason: "defeito", ActorID: "ana"}, now)
	if err != nil {
		t.Fatalf("repetir a mesma devolucao nao pode falhar: %v", err)
	}
	if repetida.ID != primeira.ID {
		t.Fatalf("repetir a mesma devolucao precisa devolver o MESMO registro: %s != %s", repetida.ID, primeira.ID)
	}
	if depois := store.Stock(org, "CAM-001").Available; depois != saldoAntes+1 {
		t.Fatalf("devolucao repetida NAO pode devolver estoque de novo: %d", depois)
	}

	// Nao devolve mais do que comprou (2 comprados, 1 ja devolvido).
	if _, err := store.RecordReturn(CommerceReturnRequest{OrganizationID: org, OrderID: order.ID, SKU: "CAM-001", Quantity: 2, Reason: "outro"}, now); !errors.Is(err, ErrCommerceReturnQuantity) {
		t.Fatalf("quantidade acima do devolvivel precisa ser recusada: %v", err)
	}
	// SKU que nao pertence ao pedido.
	if _, err := store.RecordReturn(CommerceReturnRequest{OrganizationID: org, OrderID: order.ID, SKU: "OUTRO", Quantity: 1}, now); !errors.Is(err, ErrCommerceReturnQuantity) {
		t.Fatalf("SKU fora do pedido precisa ser recusado: %v", err)
	}
	// OUTRO TENANT: a recusa tem de ser a MESMA de pedido inexistente (nao inferencia).
	_, errIntruso := store.RecordReturn(CommerceReturnRequest{OrganizationID: "org-intrusa", OrderID: order.ID, SKU: "CAM-001", Quantity: 1}, now)
	_, errInexistente := store.RecordReturn(CommerceReturnRequest{OrganizationID: "org-intrusa", OrderID: "pedido_inexistente", SKU: "CAM-001", Quantity: 1}, now)
	if errIntruso == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir (intruso=%v, inexistente=%v)", errIntruso, errInexistente)
	}
	// A comparacao NAO pode ser o texto integral: a mensagem embute o ID PEDIDO, que e' do proprio chamador
	// ("commerce order not found: <id>"). O que revelaria outro tenant seria o TIPO de erro ou o prefixo. Comparar
	// texto integral acusaria um oraculo inexistente - o instrumento errado era a ASERCAO.
	if !errors.Is(errIntruso, ErrCommerceOrderMissing) || !errors.Is(errInexistente, ErrCommerceOrderMissing) {
		t.Fatalf("as duas recusas precisam ser do MESMO tipo: intruso=%v inexistente=%v", errIntruso, errInexistente)
	}
	prefixo := func(erro error) string {
		texto := erro.Error()
		if i := strings.Index(texto, ":"); i >= 0 {
			return texto[:i]
		}
		return texto
	}
	if prefixo(errIntruso) != prefixo(errInexistente) {
		t.Fatalf("ORACULO ENTRE TENANTS na devolucao: %q != %q", prefixo(errIntruso), prefixo(errInexistente))
	}

	// PERSISTENCIA: reinicio REAL sobre o mesmo diretorio.
	reaberta, err := NewCommerceStoreAt(root)
	if err != nil {
		t.Fatal(err)
	}
	devolucoes := reaberta.Returns(org)
	if len(devolucoes) != 1 {
		t.Fatalf("a devolucao precisa sobreviver ao reinicio: %d", len(devolucoes))
	}
	if devolucoes[0].ID != primeira.ID || devolucoes[0].MovementID == "" {
		t.Fatalf("a devolucao recuperada esta incompleta: %+v", devolucoes[0])
	}
	if outras := reaberta.Returns("org-intrusa"); len(outras) != 0 {
		t.Fatalf("outro tenant nao pode enxergar a devolucao: %d", len(outras))
	}
}
