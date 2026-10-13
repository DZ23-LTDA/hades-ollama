package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// QUARTO LOTE DE MEDIÇÃO POR COMPORTAMENTO (E-342): marketplace_sync e company.Create.
//
// HIPÓTESE LEVANTADA NA LEITURA: `MarketplaceSyncStore.Sync` **já** valida a organização crua (linha 245:
// `if organizationID == "" || channelID == ""`), ANTES de qualquer sanitização. Se o comportamento confirmar, este
// arquivo é uma **falsa pendência** — a catraca acusa porque só procura a string `requireOrganization(`, e um arquivo
// pode estar correto por outro caminho. *Instrumento que só reconhece um padrão transforma código certo em dívida.*

func TestQuartoLoteMedeEscopoAusente(t *testing.T) {
	t.Run("marketplace_sync", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "marketplace")
		// MEDIDO: NewMarketplaceSyncStore(root, *CommerceStore) - dois parâmetros.
		comercio, errComercio := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
		if errComercio != nil {
			t.Fatal(errComercio)
		}
		store := NewMarketplaceSyncStore(raiz, comercio)
		// Organização vazia: precisa recusar. Canal vazio também; itens vazios também.
		if _, err := store.Sync(MarketplaceSyncRequest{OrganizationID: "", ChannelID: "ch-1",
			Items: []MarketplaceSyncItem{{SKU: "SKU-1"}}}, time.Now().UTC()); err == nil {
			t.Fatal("ARMADILHA: sincronização sem organização foi aceita (gravaria em arquivo compartilhado)")
		}
		if _, err := store.Sync(MarketplaceSyncRequest{OrganizationID: "org-a", ChannelID: "",
			Items: []MarketplaceSyncItem{{SKU: "SKU-1"}}}, time.Now().UTC()); err == nil {
			t.Fatal("sincronização sem canal precisa ser recusada")
		}
		// Organização só com espaços também é vazia (trim antes da decisão).
		if _, err := store.Sync(MarketplaceSyncRequest{OrganizationID: "   ", ChannelID: "ch-1",
			Items: []MarketplaceSyncItem{{SKU: "SKU-1"}}}, time.Now().UTC()); err == nil {
			t.Fatal("ARMADILHA: organização só com espaços foi aceita")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("company_create", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "company")
		store, err := NewCompanyStore(raiz)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Create(Company{Name: "Empresa"}); err == nil {
			t.Fatal("ARMADILHA: criação de empresa sem organização foi aceita")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})
}

// O inverso: com organização válida, os dois caminhos funcionam.
func TestQuartoLoteComOrganizacaoValidaFunciona(t *testing.T) {
	// marketplace: canal desconhecido falha por OUTRO motivo (canal), não por escopo — prova que o escopo passou.
	raiz := filepath.Join(t.TempDir(), "marketplace")
	comercio, errComercio := NewCommerceStoreAt(filepath.Join(raiz, "commerce"))
	if errComercio != nil {
		t.Fatal(errComercio)
	}
	store := NewMarketplaceSyncStore(raiz, comercio)
	if _, err := store.Sync(MarketplaceSyncRequest{OrganizationID: "org-a", ChannelID: "canal-inexistente",
		Items: []MarketplaceSyncItem{{SKU: "SKU-1"}}}, time.Now().UTC()); errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("com organização válida a recusa precisa ser outra (canal), não escopo: %v", err)
	}
	// company: criação válida devolve empresa com identificador.
	raizEmpresa := filepath.Join(t.TempDir(), "company")
	companhia, err := NewCompanyStore(raizEmpresa)
	if err != nil {
		t.Fatal(err)
	}
	criada, err := companhia.Create(Company{Name: "Empresa Válida", OrganizationID: "org-a"})
	if err != nil {
		t.Fatalf("criação com organização válida precisa funcionar: %v", err)
	}
	if criada.ID == "" {
		t.Fatal("a empresa criada precisa receber identificador")
	}
}
