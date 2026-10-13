package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// ONBOARDING DE EMPRESA JÁ EXISTENTE — TESTES (E-327; V2.2 / adendo P1-B).
//
// Cada propriedade tem direção negativa: preview que grava, dedupe que não deduplica, conflito sobrescrito em silêncio
// ou rollback que deixa resíduo seriam defeitos piores do que não ter importação nenhuma.

func onboardingFixture(t *testing.T) (*CommerceStore, string) {
	t.Helper()
	// MEDIDO por `go vet`: NewCommerceStoreAt devolve DOIS valores. Medir a assinatura evitou supor.
	store, err := NewCommerceStoreAt(filepath.Join(t.TempDir(), "commerce"))
	if err != nil {
		t.Fatal(err)
	}
	return store, "org-existente"
}

func TestPreviewNaoEscreveNadaNoCatalogo(t *testing.T) {
	comercio, org := onboardingFixture(t)
	// Empresa que JÁ opera: um produto existe antes de qualquer importação.
	if _, err := comercio.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: "SKU-1", Name: "Produto antigo",
		PriceCents: 1000, Currency: "BRL"}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "erp-x", Rows: []CompanyOnboardingRow{
		{SKU: "SKU-1", Name: "Produto antigo", PriceCents: 1000, Currency: "BRL"},
		{SKU: "SKU-2", Name: "Novo produto", PriceCents: 2500, Currency: "BRL", InitialStock: 5},
	}}
	preview, err := PlanCompanyOnboarding(comercio, plano)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.ToCreate) != 1 || preview.ToCreate[0] != "SKU-2" || len(preview.Unchanged) != 1 {
		t.Fatalf("preview incorreto: %+v", preview)
	}
	// ARMADILHA: o preview NÃO pode ter criado o produto.
	if _, achou := comercio.ProductBySKU(org, "SKU-2"); achou {
		t.Fatal("ARMADILHA: o preview ESCREVEU no catálogo — um preview que grava deixou de ser preview")
	}
	if produtos := comercio.Products(org); len(produtos) != 1 {
		t.Fatalf("o preview não pode alterar o catálogo: %d produtos", len(produtos))
	}
}

func TestConflitoDePrecoNaoEhSobrescritoEmSilencio(t *testing.T) {
	comercio, org := onboardingFixture(t)
	if _, err := comercio.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: "SKU-1", Name: "Produto",
		PriceCents: 1000, Currency: "BRL"}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "erp-x", Rows: []CompanyOnboardingRow{
		{SKU: "SKU-1", Name: "Produto", PriceCents: 9999, Currency: "BRL"}, // preço divergente do cliente
	}}
	preview, err := PlanCompanyOnboarding(comercio, plano)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Conflicts) != 1 || preview.Conflicts[0] != "SKU-1" {
		t.Fatalf("preço divergente precisa virar conflito DECLARADO: %+v", preview)
	}
	// Aplicar sem Overwrite é RECUSADO — o conflito tem de ser decisão, não efeito colateral.
	if _, _, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); !errors.Is(err, ErrOnboardingConflict) {
		t.Fatalf("aplicar com conflito e sem Overwrite precisa ser recusado: %v", err)
	}
	atual, _ := comercio.ProductBySKU(org, "SKU-1")
	if atual.PriceCents != 1000 {
		t.Fatalf("ARMADILHA: o preço do cliente foi sobrescrito sem autorização: %d", atual.PriceCents)
	}

	// Com Overwrite explícito, aplica.
	plano.Overwrite = true
	if _, token, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatalf("com Overwrite explícito precisa aplicar: %v", err)
	} else if len(token.CreatedSKUs) != 0 {
		t.Fatalf("atualizar não cria SKU novo: %+v", token.CreatedSKUs)
	}
	if atual, _ := comercio.ProductBySKU(org, "SKU-1"); atual.PriceCents != 9999 {
		t.Fatalf("com Overwrite o preço precisa ser atualizado: %d", atual.PriceCents)
	}
}

func TestDuplicatasDoPayloadContamUmaVezEDedupeEhRegistrado(t *testing.T) {
	comercio, org := onboardingFixture(t)
	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "planilha", Rows: []CompanyOnboardingRow{
		{SKU: "SKU-A", Name: "Produto A", PriceCents: 100, Currency: "BRL"},
		{SKU: "SKU-A", Name: "Produto A repetido", PriceCents: 100, Currency: "BRL"},
		{SKU: "SKU-A", Name: "Produto A repetido de novo", PriceCents: 100, Currency: "BRL"},
		{SKU: "SKU-B", Name: "Produto B", PriceCents: 200, Currency: "BRL"},
	}}
	preview, err := PlanCompanyOnboarding(comercio, plano)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Duplicates) != 2 {
		t.Fatalf("as duas repetições precisam ser CONTADAS como duplicata: %+v", preview.Duplicates)
	}
	if len(preview.ToCreate) != 2 {
		t.Fatalf("duplicata não pode gerar produto extra: %+v", preview.ToCreate)
	}
	if _, _, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if produtos := comercio.Products(org); len(produtos) != 2 {
		t.Fatalf("importar duas vezes a mesma linha duplicaria o catálogo do cliente: %d", len(produtos))
	}
	// Reaplicação IDEMPOTENTE: nada muda, nada duplica.
	repetido, err := PlanCompanyOnboarding(comercio, plano)
	if err != nil {
		t.Fatal(err)
	}
	if len(repetido.ToCreate) != 0 || len(repetido.Unchanged) != 2 {
		t.Fatalf("reaplicar precisa ser idempotente: %+v", repetido)
	}
}

func TestEstoqueInicialNaoInflaNaReaplicacao(t *testing.T) {
	comercio, org := onboardingFixture(t)
	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "erp-x", Rows: []CompanyOnboardingRow{
		{SKU: "SKU-E", Name: "Produto com estoque", PriceCents: 500, Currency: "BRL", InitialStock: 12},
	}}
	if _, _, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	saldo := comercio.Stock(org, "SKU-E")
	if saldo.OnHand != 12 {
		t.Fatalf("estoque inicial reaplicado inflou o saldo (a lição de E-305): %+v", saldo)
	}
	if movimentos := comercio.Movements(org, "SKU-E"); len(movimentos) != 1 {
		t.Fatalf("a carga inicial precisa ser UM movimento: %d", len(movimentos))
	}
}

func TestRollbackRestauraEstadoAnteriorExato(t *testing.T) {
	comercio, org := onboardingFixture(t)
	if _, err := comercio.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: "SKU-1", Name: "Antigo",
		PriceCents: 1000, Currency: "BRL"}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	antes := comercio.Products(org)

	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "erp-x", Overwrite: true, Rows: []CompanyOnboardingRow{
		{SKU: "SKU-1", Name: "Renomeado", PriceCents: 1111, Currency: "BRL"},
		{SKU: "SKU-3", Name: "Criado pela carga", PriceCents: 300, Currency: "BRL"},
	}}
	if _, token, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatal(err)
	} else {
		if len(token.CreatedSKUs) != 1 || token.CreatedSKUs[0] != "SKU-3" {
			t.Fatalf("o token precisa registrar o que foi CRIADO: %+v", token.CreatedSKUs)
		}
		if err := RollbackCompanyOnboarding(comercio, token, time.Now().UTC()); err != nil {
			t.Fatalf("rollback: %v", err)
		}
	}
	depois := comercio.Products(org)
	if len(depois) != len(antes) {
		t.Fatalf("rollback precisa deixar o catálogo como estava: antes %d, depois %d", len(antes), len(depois))
	}
	restaurado, _ := comercio.ProductBySKU(org, "SKU-1")
	if restaurado.Name != "Antigo" || restaurado.PriceCents != 1000 {
		t.Fatalf("rollback precisa restaurar o produto anterior EXATO: %+v", restaurado)
	}
	if _, achou := comercio.ProductBySKU(org, "SKU-3"); achou {
		t.Fatal("rollback precisa remover o que a carga criou")
	}
}

func TestOnboardingNaoMisturaOrganizacoes(t *testing.T) {
	comercio, org := onboardingFixture(t)
	plano := CompanyOnboardingPlan{OrganizationID: org, Origin: "erp-x", Rows: []CompanyOnboardingRow{
		{SKU: "SKU-Z", Name: "Produto da org A", PriceCents: 700, Currency: "BRL"},
	}}
	if _, _, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if produtos := comercio.Products("org-vizinha"); len(produtos) != 0 {
		t.Fatalf("a organização vizinha não pode receber catálogo alheio: %+v", produtos)
	}
	if _, achou := comercio.ProductBySKU("org-vizinha", "SKU-Z"); achou {
		t.Fatal("a SKU da organização A não pode existir na B")
	}
}
