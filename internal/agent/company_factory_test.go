package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Estágio 23 — fábrica de empresa em sandbox.
//
// As propriedades travadas aqui:
//  1. o lançamento cria a empresa E MATERIALIZA artefatos em disco (plano, marca,
//     precificação, site), com caminho e tamanho reais no relatório;
//  2. o preço é DERIVADO de custo e margem, com a conta conferida; margem de 100% ou mais
//     é recusada;
//  3. a fábrica NÃO inventa clientes: o CRM fica VAZIO depois do lançamento;
//  4. o checklist separa o que foi FEITO do que depende de terceiros, com o
//     PROCEDIMENTO de desbloqueio declarado;
//  5. relançar não duplica produto (`UpsertProduct` é idempotente por SKU);
//  6. o relatório é escopado por organização.
func factoryRuntime(t *testing.T) *Runtime {
	t.Helper()
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func factoryRequest(organizationID string) CompanyFactoryRequest {
	return CompanyFactoryRequest{
		OrganizationID: organizationID,
		Company: CompanyCreateRequest{
			Name: "Camisaria Sandbox", Mission: "Vender camisetas sob demanda",
			Positioning: "Camisetas locais com estampa exclusiva", BusinessModel: "venda direta e assinatura",
			TargetAudience: "pessoas de 20 a 40 anos", Offer: "camiseta personalizada em 5 dias",
			Currency: "BRL",
		},
		Tagline: "sua estampa, sua história",
		Palette: []string{"#0b3d2e", "#f2e9dc"},
		Products: []CompanyPricingInput{
			{SKU: "CAM-001", Name: "Camiseta básica", CostCents: 3500, MarginPercent: 60, InitialStock: 10},
			{SKU: "CAM-002", Name: "Camiseta premium", CostCents: 5200, MarginPercent: 50, InitialStock: 5},
		},
	}
}

func TestCompanyFactoryMaterializesRealArtifacts(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	report, err := runtime.LaunchCompany(factoryRequest(LocalOrganizationID), now)
	if err != nil {
		t.Fatal(err)
	}
	if report.CompanyID == "" || report.Company.Name != "Camisaria Sandbox" {
		t.Fatalf("empresa não criada: %+v", report.Company)
	}
	// PREÇO DERIVADO: 3500 × 1,60 = 5600 e 5200 × 1,50 = 7800.
	prices := map[string]int64{}
	for _, product := range report.Products {
		prices[product.SKU] = product.PriceCents
	}
	if prices["CAM-001"] != 5600 || prices["CAM-002"] != 7800 {
		t.Fatalf("preço derivado de custo e margem incorreto: %+v", prices)
	}
	// Valor de tabela: 5600×10 + 7800×5 = 95000.
	if report.CatalogValueCents != 95000 {
		t.Fatalf("valor de catálogo incorreto: %d", report.CatalogValueCents)
	}
	// ARTEFATOS REAIS: existem em disco, com tamanho medido e conteúdo não vazio.
	kinds := map[string]CompanyArtifact{}
	for _, artifact := range report.Artifacts {
		kinds[artifact.Kind] = artifact
		if artifact.Bytes <= 0 {
			t.Fatalf("artefato %s precisa ter tamanho real: %+v", artifact.Kind, artifact)
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatalf("artefato %s precisa existir no caminho informado: %v", artifact.Kind, err)
		}
	}
	for _, kind := range []string{CompanyArtifactPlan, CompanyArtifactBrand, CompanyArtifactPricing, CompanyArtifactSite} {
		if _, ok := kinds[kind]; !ok {
			t.Fatalf("artefato %s ausente: %+v", kind, report.Artifacts)
		}
	}
	// O SITE é HTML de verdade e contém os dados REAIS da empresa.
	site, err := os.ReadFile(kinds[CompanyArtifactSite].Path)
	if err != nil {
		t.Fatal(err)
	}
	page := string(site)
	if !strings.Contains(page, "<!doctype html>") || !strings.Contains(page, "Camisaria Sandbox") {
		t.Fatalf("o site precisa ser HTML com os dados da empresa: %q", page[:min(120, len(page))])
	}
	if !strings.Contains(page, "5600 centavos") {
		t.Fatal("o site precisa refletir o preço real do catálogo")
	}
	// A PRECIFICAÇÃO mostra a conta, para o operador conferir.
	pricing, err := os.ReadFile(kinds[CompanyArtifactPricing].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pricing), "preço = custo") {
		t.Fatal("a precificação precisa declarar a fórmula")
	}

	// A FÁBRICA NÃO INVENTA CLIENTES.
	if report.CustomersCreated != 0 {
		t.Fatalf("a fábrica não pode criar clientes: %d", report.CustomersCreated)
	}
	if contacts := runtime.CRM().ListContacts(LocalOrganizationID, ""); len(contacts) != 0 {
		t.Fatalf("o CRM precisa ficar VAZIO: a fábrica não inventa clientes: %+v", contacts)
	}

	// CHECKLIST: o que foi feito e o que depende de terceiros, COM procedimento.
	blocked := 0
	done := 0
	for _, item := range report.Checklist {
		switch item.Status {
		case CompanyChecklistDone:
			done++
			if item.Evidence == "" {
				t.Fatalf("item feito sem evidência: %+v", item)
			}
		case CompanyChecklistBlocked:
			blocked++
			if item.Unblock == "" {
				t.Fatalf("item bloqueado precisa do PROCEDIMENTO de desbloqueio: %+v", item)
			}
		}
	}
	if done < 3 || blocked < 4 {
		t.Fatalf("checklist incompleto: %d feitos, %d bloqueados", done, blocked)
	}
	// O registro legal NÃO é emitido: isso precisa estar explícito.
	legal := false
	for _, item := range report.Checklist {
		if strings.Contains(item.Item, "CNPJ") && item.Status == CompanyChecklistBlocked {
			legal = true
		}
	}
	if !legal {
		t.Fatal("o checklist precisa declarar que CNPJ/registro legal depende de terceiro")
	}

	// O relatório reconstruído a partir do ESTADO REAL aponta os mesmos artefatos.
	reloaded, err := runtime.CompanyFactoryReportFor(LocalOrganizationID, report.CompanyID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Artifacts) != len(report.Artifacts) {
		t.Fatalf("o relatório reconstruído precisa achar os artefatos: %d vs %d",
			len(reloaded.Artifacts), len(report.Artifacts))
	}
	if len(reloaded.Products) != 2 {
		t.Fatalf("o relatório precisa trazer o catálogo real: %+v", reloaded.Products)
	}
}

func TestCompanyFactoryRejectsInvalidMarginAndPrice(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	// Margem de 100% não tem preço finito frente à margem: recusada.
	request := factoryRequest(LocalOrganizationID)
	request.Products = []CompanyPricingInput{{SKU: "X-1", Name: "Inválido", CostCents: 1000, MarginPercent: 100}}
	if _, err := runtime.LaunchCompany(request, now); !errors.Is(err, ErrCompanyPricingInvalid) {
		t.Fatalf("margem de 100%% precisa ser recusada, veio %v", err)
	}
	// Custo negativo é recusado.
	request.Products = []CompanyPricingInput{{SKU: "X-2", Name: "Inválido", CostCents: -1, MarginPercent: 10}}
	if _, err := runtime.LaunchCompany(request, now); !errors.Is(err, ErrCompanyPricingInvalid) {
		t.Fatalf("custo negativo precisa ser recusado, veio %v", err)
	}
	// Organização ausente é recusada.
	request = factoryRequest("")
	request.Products = nil
	if _, err := runtime.LaunchCompany(request, now); err == nil {
		t.Fatal("lançamento sem organização precisa falhar")
	}
}

func TestCompanyFactoryIsIdempotentPerSKUAndScopedByOrganization(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	first, err := runtime.LaunchCompany(factoryRequest(LocalOrganizationID), now)
	if err != nil {
		t.Fatal(err)
	}
	// Relançar NÃO duplica produto: o SKU é a chave.
	if _, err := runtime.LaunchCompany(factoryRequest(LocalOrganizationID), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	products := runtime.Commerce().Products(LocalOrganizationID)
	if len(products) != 2 {
		t.Fatalf("relançar não pode duplicar o catálogo: %d", len(products))
	}
	// ESCOPO: outra organização não vê os artefatos nem o catálogo da primeira.
	if _, err := runtime.CompanyFactoryReportFor("org-vizinho", first.CompanyID); err == nil {
		t.Fatal("o relatório de outra organização precisa ser recusado")
	}
	if products := runtime.Commerce().Products("org-vizinho"); len(products) != 0 {
		t.Fatalf("o catálogo precisa ser por organização: %+v", products)
	}
}
