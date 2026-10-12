package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estagio 25 — PULL DE PEDIDO MULTI-SKU (E-270). O limite declarado dizia "o pull cobre UM item por pedido
// (pedido multi-SKU exige uma chamada por item)".
//
// O defeito que o teste trava: cada item criava uma ORDEM LOCAL separada. Um pedido do canal com dois SKUs virava
// dois pedidos locais - o cliente apareceria com dois pedidos e a logistica com dois embarques.
func TestPullDePedidoMultiSKUCriaUmUnicoPedidoLocal(t *testing.T) {
	root := t.TempDir()
	commerce, err := NewCommerceStoreAt(filepath.Join(root, "commerce"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	org := LocalOrganizationID
	for _, sku := range []string{"CAM-1", "BON-1"} {
		if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: sku, Name: sku, PriceCents: 5000, Active: true}, now); err != nil {
			t.Fatal(err)
		}
		if _, err := commerce.RecordStock(org, sku, 10, CommerceMovementEntry, "carga", now); err != nil {
			t.Fatal(err)
		}
	}
	sincronizador := NewMarketplaceSyncStore(filepath.Join(root, "sync"), commerce)
	pull := func(ext string, itens ...MarketplaceSyncItem) MarketplaceSyncReport {
		report, err := sincronizador.Sync(MarketplaceSyncRequest{
			OrganizationID: org, ChannelID: "mercado_livre", Direction: "pull", Items: itens,
		}, now)
		if err != nil {
			t.Fatalf("pull %s: %v", ext, err)
		}
		return report
	}

	// 1. UM pedido do canal com DOIS SKUs vira UM pedido local com DUAS linhas.
	report := pull("EXT-1",
		MarketplaceSyncItem{Entity: "pedidos", SKU: "CAM-1", Quantity: 1, ExternalID: "EXT-1", Version: 1},
		MarketplaceSyncItem{Entity: "pedidos", SKU: "BON-1", Quantity: 2, ExternalID: "EXT-1", Version: 1},
	)
	if report.Applied != 2 {
		t.Fatalf("as DUAS linhas precisam ser aplicadas: %+v", report)
	}
	pedidos := commerce.Orders(org)
	if len(pedidos) != 1 {
		t.Fatalf("pedido multi-SKU precisa virar UM pedido local, veio %d", len(pedidos))
	}
	completo, err := commerce.Order(org, pedidos[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(completo.Items) != 2 {
		t.Fatalf("o pedido local precisa ter as DUAS linhas: %+v", completo.Items)
	}

	// 2. REENTREGA do mesmo pedido do canal: tudo ignorado e NENHUM pedido novo.
	repetido := pull("EXT-1",
		MarketplaceSyncItem{Entity: "pedidos", SKU: "CAM-1", Quantity: 1, ExternalID: "EXT-1", Version: 1},
		MarketplaceSyncItem{Entity: "pedidos", SKU: "BON-1", Quantity: 2, ExternalID: "EXT-1", Version: 1},
	)
	if repetido.Skipped != 2 || repetido.Applied != 0 {
		t.Fatalf("reentrega precisa ser ignorada por completo: %+v", repetido)
	}
	if len(commerce.Orders(org)) != 1 {
		t.Fatalf("reentrega criou pedido novo: %d", len(commerce.Orders(org)))
	}

	// 3. SKU desconhecido em UMA linha: o pedido NAO entra pela metade.
	recusado := pull("EXT-2",
		MarketplaceSyncItem{Entity: "pedidos", SKU: "CAM-1", Quantity: 1, ExternalID: "EXT-2", Version: 1},
		MarketplaceSyncItem{Entity: "pedidos", SKU: "NAO-EXISTE", Quantity: 1, ExternalID: "EXT-2", Version: 1},
	)
	if recusado.Conflicts != 1 || recusado.Applied != 0 {
		t.Fatalf("pedido com linha invalida precisa ser recusado por inteiro: %+v", recusado)
	}
	if len(commerce.Orders(org)) != 1 {
		t.Fatalf("pedido parcial nao pode existir: %d pedidos", len(commerce.Orders(org)))
	}

	// 4. ISOLAMENTO: a MESMA identificacao externa em OUTRA organizacao e outro pedido.
	outra := "org-b"
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: outra, SKU: "CAM-1", Name: "Camiseta", PriceCents: 5000, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.RecordStock(outra, "CAM-1", 5, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	if _, err := sincronizador.Sync(MarketplaceSyncRequest{
		OrganizationID: outra, ChannelID: "mercado_livre", Direction: "pull",
		Items: []MarketplaceSyncItem{{Entity: "pedidos", SKU: "CAM-1", Quantity: 1, ExternalID: "EXT-1", Version: 1}},
	}, now); err != nil {
		t.Fatalf("pull da outra organizacao: %v", err)
	}
	if len(commerce.Orders(outra)) != 1 {
		t.Fatalf("a idempotencia do canal NAO pode ser compartilhada entre organizacoes: %d pedidos", len(commerce.Orders(outra)))
	}
}
