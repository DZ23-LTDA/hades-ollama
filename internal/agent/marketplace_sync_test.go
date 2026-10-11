package agent

import (
	"errors"
	"testing"
	"time"
)

// Estágio 25 — contrato de sincronização de marketplace contra DUBLE de sandbox.
//
// Propriedades travadas:
//  1. CATALOGAR NÃO É HABILITAR: canal sem credencial é `CONFIG_REQUIRED` com procedimento; com
//     credencial e sem handshake é `UNAUTHORIZED` e `LiveTested=false`;
//  2. a homologação live é declarada `BLOCKED_BY_EXTERNAL_DEPENDENCY` em TODO relatório;
//  3. IDEMPOTÊNCIA por versão: reexecutar não reaplica;
//  4. TETO de operações por execução é recusado ANTES de aplicar efeito;
//  5. CONFLITO de estoque é registrado e NÃO sobrescreve o remoto em silêncio;
//  6. canal desconhecido, direção inválida e pedido vazio são erros declarados.
func marketplaceRuntime(t *testing.T) *Runtime {
	t.Helper()
	return factoryRuntime(t)
}

func TestMarketplaceChannelCatalogDeclaresRealState(t *testing.T) {
	// Sem nenhuma credencial: todo canal é CONFIG_REQUIRED e declara o que falta.
	for _, spec := range MarketplaceChannels(map[string]bool{}) {
		if spec.State != MarketplaceStateConfigRequired {
			t.Fatalf("canal %s sem credencial precisa ser CONFIG_REQUIRED, veio %s", spec.ID, spec.State)
		}
		if spec.LiveTested {
			t.Fatalf("canal %s sem handshake não pode declarar teste live", spec.ID)
		}
		if spec.Unblock == "" {
			t.Fatalf("canal %s precisa declarar o procedimento de desbloqueio", spec.ID)
		}
		if len(spec.Capabilities) == 0 {
			t.Fatalf("canal %s precisa declarar capacidades", spec.ID)
		}
	}
	// Credencial presente NÃO é operação: vira UNAUTHORIZED, ainda sem teste live.
	configured := map[string]bool{"HADES_MARKETPLACE_MERCADO_LIVRE_TOKEN": true}
	for _, spec := range MarketplaceChannels(configured) {
		if spec.ID != "mercado_livre" {
			continue
		}
		if spec.State != MarketplaceStateUnauthorized {
			t.Fatalf("credencial sem handshake precisa ser UNAUTHORIZED, veio %s", spec.State)
		}
		if spec.LiveTested {
			t.Fatalf("credencial presente não equivale a teste live")
		}
	}
	if MarketplaceLiveStatus != "BLOCKED_BY_EXTERNAL_DEPENDENCY" || MarketplaceLiveReason == "" {
		t.Fatalf("a homologação live precisa ser declarada bloqueada com motivo")
	}
}

func TestMarketplaceSyncAppliesIsIdempotentAndDeclaresSandbox(t *testing.T) {
	runtime := marketplaceRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "MKP-1", Name: "Produto de marketplace", PriceCents: 3000, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(organization, "MKP-1", 7, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}

	request := MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "mercado_livre", Direction: "push",
		Items: []MarketplaceSyncItem{
			{Entity: MarketplaceEntityProducts, SKU: "MKP-1", Version: 1, PriceCents: 3000},
			{Entity: MarketplaceEntityStock, SKU: "MKP-1", Version: 1, Quantity: 7},
		},
	}
	report, err := runtime.Marketplace().Sync(request, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Applied != 2 || report.Conflicts != 0 || report.Skipped != 0 {
		t.Fatalf("primeira sincronização precisa aplicar 2: %+v", report)
	}
	// O relatório DECLARA o ambiente e o bloqueio: nada aqui é produção.
	if !report.Sandbox || report.LiveStatus != MarketplaceLiveStatus || report.LiveReason == "" {
		t.Fatalf("relatório precisa declarar sandbox e homologação bloqueada: %+v", report)
	}
	if report.OperationsUsed != 2 || report.OperationLimit != MarketplaceMaxOperationsPerRun {
		t.Fatalf("contagem de operações incorreta: %+v", report)
	}
	// O duble recebeu o estoque.
	if quantity, ok := runtime.Marketplace().RemoteStock("mercado_livre", "MKP-1"); !ok || quantity != 7 {
		t.Fatalf("estoque remoto do duble incorreto: %d %t", quantity, ok)
	}

	// IDEMPOTÊNCIA: a MESMA versão não reaplica.
	repeated, err := runtime.Marketplace().Sync(request, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Applied != 0 || repeated.Skipped != 2 || repeated.OperationsUsed != 0 {
		t.Fatalf("reexecutar a mesma versão precisa apenas ignorar: %+v", repeated)
	}

	// Versão NOVA aplica de novo.
	request.Items = []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "MKP-1", Version: 2, Quantity: 7}}
	fresh, err := runtime.Marketplace().Sync(request, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Applied != 1 {
		t.Fatalf("versão nova precisa aplicar: %+v", fresh)
	}
}

func TestMarketplaceSyncRefusesRateLimitAndInvalidInput(t *testing.T) {
	runtime := marketplaceRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID

	// TETO de operações: recusado ANTES de qualquer efeito.
	items := make([]MarketplaceSyncItem, 0, MarketplaceMaxOperationsPerRun+1)
	for index := 0; index <= MarketplaceMaxOperationsPerRun; index++ {
		items = append(items, MarketplaceSyncItem{Entity: MarketplaceEntityStock, SKU: "MKP-LIM", Version: 1, Quantity: 1})
	}
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "shopee", Items: items,
	}, now); !errors.Is(err, ErrMarketplaceRateLimited) {
		t.Fatalf("acima do teto precisa ser recusado, veio %v", err)
	}
	// E nada foi aplicado, apesar do pedido grande.
	if _, ok := runtime.Marketplace().RemoteStock("shopee", "MKP-LIM"); ok {
		t.Fatal("recusa por teto não pode ter aplicado efeito")
	}

	// Canal desconhecido.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "marketplace_inexistente",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "X", Version: 1, Quantity: 1}},
	}, now); !errors.Is(err, ErrMarketplaceChannelUnknown) {
		t.Fatalf("canal desconhecido precisa falhar, veio %v", err)
	}
	// Pedido vazio.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{OrganizationID: organization, ChannelID: "shopee"}, now); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("pedido vazio precisa falhar, veio %v", err)
	}
	// Direção inválida.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "shopee", Direction: "lateral",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "X", Version: 1, Quantity: 1}},
	}, now); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("direção inválida precisa falhar, veio %v", err)
	}
	// Item sem SKU.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "shopee",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, Version: 1, Quantity: 1}},
	}, now); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("item sem SKU precisa falhar, veio %v", err)
	}
	// Escopo ausente.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		ChannelID: "shopee", Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "X", Version: 1, Quantity: 1}},
	}, now); !errors.Is(err, ErrMarketplaceSyncInvalid) {
		t.Fatalf("organização ausente precisa falhar, veio %v", err)
	}
}

func TestMarketplaceSyncReportsConflictInsteadOfOverwriting(t *testing.T) {
	runtime := marketplaceRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "MKP-C", Name: "Produto em conflito", PriceCents: 5000, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(organization, "MKP-C", 10, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	// Primeira publicação: remoto passa a ter 10.
	if _, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "amazon",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "MKP-C", Version: 1, Quantity: 10}},
	}, now); err != nil {
		t.Fatal(err)
	}

	// O estoque local cai para 5 (venda), e queremos publicar 5. O remoto ainda diz 10 e NÃO foi
	// atualizado por este espelho: sobrescrever esconderia a divergência.
	if _, err := runtime.Commerce().RecordStock(organization, "MKP-C", -5, CommerceMovementShipped, "venda", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	report, err := runtime.Marketplace().Sync(MarketplaceSyncRequest{
		OrganizationID: organization, ChannelID: "amazon",
		Items: []MarketplaceSyncItem{{Entity: MarketplaceEntityStock, SKU: "MKP-C", Version: 2, Quantity: 5}},
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if report.Conflicts != 1 || report.Applied != 0 {
		t.Fatalf("divergência precisa virar CONFLITO sem aplicar: %+v", report)
	}
	if len(report.Outcomes) != 1 || report.Outcomes[0].Result != MarketplaceResultConflict || report.Outcomes[0].Detail == "" {
		t.Fatalf("o conflito precisa ser registrado com detalhe: %+v", report.Outcomes)
	}
	// O remoto NÃO foi sobrescrito.
	if quantity, _ := runtime.Marketplace().RemoteStock("amazon", "MKP-C"); quantity != 10 {
		t.Fatalf("o remoto não pode ser sobrescrito em conflito: %d", quantity)
	}
}
