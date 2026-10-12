package agent

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPullDePedidosDoCanalEhIdempotenteEReservaEstoque exercita o caminho que faltava no Marketplace Hub:
// PUXAR pedido do canal. O pedido local nasce pelo motor de comércio (preço vem do CATÁLOGO, estoque fica
// RESERVADO) e a idempotência é por ID EXTERNO — reenviar o mesmo pedido do canal NÃO cria outro pedido.
func TestPullDePedidosDoCanalEhIdempotenteEReservaEstoque(t *testing.T) {
	raiz := t.TempDir()
	org := "org-marketplace-pull"
	canal := "mercado_livre"
	agora := time.Now().UTC()

	comercio, err := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
	if err != nil {
		t.Fatalf("motor de comércio: %v", err)
	}
	if _, err := comercio.UpsertProduct(CommerceProduct{
		OrganizationID: org, SKU: "SKU-ML", Name: "Produto do canal", PriceCents: 3990, Currency: "BRL", Active: true,
	}, agora); err != nil {
		t.Fatalf("cadastrar produto: %v", err)
	}
	if _, err := comercio.RecordStock(org, "SKU-ML", 10, "entrada", "carga-inicial", agora); err != nil {
		t.Fatalf("lançar estoque: %v", err)
	}
	sincronizacao := NewMarketplaceSyncStore(filepath.Join(raiz, "sync"), comercio)

	puxar := func(externalID string, versao int64) MarketplaceSyncReport {
		t.Helper()
		relatorio, err := sincronizacao.Sync(MarketplaceSyncRequest{
			OrganizationID: org,
			ChannelID:      canal,
			Direction:      "pull",
			Items: []MarketplaceSyncItem{{
				Entity: MarketplaceEntityOrders, SKU: "SKU-ML", ExternalID: externalID, Version: versao, Quantity: 2,
			}},
		}, agora)
		if err != nil {
			t.Fatalf("pull do pedido %s: %v", externalID, err)
		}
		return relatorio
	}

	// 1. Primeiro pull: pedido entra e vira pedido LOCAL, com estoque reservado pelo motor.
	primeiro := puxar("ML-100", 1)
	if primeiro.Applied != 1 {
		t.Fatalf("esperava 1 aplicado, veio %+v", primeiro)
	}
	if !strings.Contains(primeiro.Outcomes[0].Detail, "pedido local") {
		t.Fatalf("resultado deveria apontar o pedido local criado: %+v", primeiro.Outcomes[0])
	}
	pedidos := comercio.Orders(org)
	if len(pedidos) != 1 {
		t.Fatalf("esperava 1 pedido local após o pull, veio %d", len(pedidos))
	}
	if pedidos[0].TotalCents != 2*3990 {
		t.Fatalf("preço deveria vir do CATÁLOGO (2 x 3990), veio %d", pedidos[0].TotalCents)
	}
	if saldo := comercio.Stock(org, "SKU-ML"); saldo.Reserved != 2 || saldo.Available != 8 {
		t.Fatalf("estoque não foi reservado pelo pull: %+v", saldo)
	}

	// 2. Mesmo id externo com VERSÃO DIFERENTE: não pode duplicar (idempotência é por id, não por versão).
	segundo := puxar("ML-100", 2)
	if segundo.Skipped != 1 || segundo.Applied != 0 {
		t.Fatalf("reenvio do MESMO pedido deveria ser ignorado: %+v", segundo)
	}
	if len(comercio.Orders(org)) != 1 {
		t.Fatalf("reenvio duplicou o pedido: %d", len(comercio.Orders(org)))
	}

	// 3. Id externo diferente entra como pedido novo.
	terceiro := puxar("ML-101", 1)
	if terceiro.Applied != 1 {
		t.Fatalf("pedido novo deveria ser aplicado: %+v", terceiro)
	}
	if len(comercio.Orders(org)) != 2 {
		t.Fatalf("esperava 2 pedidos locais, veio %d", len(comercio.Orders(org)))
	}

	// 4. Pedido de canal NÃO se publica por push: é entrada, não saída.
	if _, err := sincronizacao.Sync(MarketplaceSyncRequest{
		OrganizationID: org, ChannelID: canal, Direction: "push",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityOrders, SKU: "SKU-ML", ExternalID: "ML-102", Version: 1, Quantity: 1}},
	}, agora); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("push de pedido deveria ser recusado, veio: %v", err)
	}

	// 5. Sem id externo não há idempotência possível: recusar é o certo.
	if _, err := sincronizacao.Sync(MarketplaceSyncRequest{
		OrganizationID: org, ChannelID: canal, Direction: "pull",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityOrders, SKU: "SKU-ML", Version: 1, Quantity: 1}},
	}, agora); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("pedido sem id externo deveria ser recusado, veio: %v", err)
	}

	// 6. O teto por execução continua valendo: recusar em vez de violar o rate limit da plataforma.
	excesso := make([]MarketplaceSyncItem, 0, MarketplaceMaxOperationsPerRun+1)
	for i := 0; i <= MarketplaceMaxOperationsPerRun; i++ {
		excesso = append(excesso, MarketplaceSyncItem{
			Entity: MarketplaceEntityOrders, SKU: "SKU-ML", ExternalID: "ML-EXCESSO", Version: 1, Quantity: 1,
		})
	}
	if _, err := sincronizacao.Sync(MarketplaceSyncRequest{
		OrganizationID: org, ChannelID: canal, Direction: "pull", Items: excesso,
	}, agora); !errors.Is(err, ErrMarketplaceRateLimited) {
		t.Fatalf("excesso de operações deveria ser recusado, veio: %v", err)
	}

	t.Logf("PULL DE PEDIDOS: pedidos_locais=%d reservado=%d disponivel=%d primeiro=%s ignorado=%s",
		len(comercio.Orders(org)), comercio.Stock(org, "SKU-ML").Reserved, comercio.Stock(org, "SKU-ML").Available,
		primeiro.Outcomes[0].Detail, segundo.Outcomes[0].Detail)
}
