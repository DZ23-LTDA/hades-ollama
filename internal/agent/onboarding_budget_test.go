package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ORÇAMENTO ACUMULADO DE ONBOARDING — TESTES (E-359).
//
// **MEDIDO antes**: o teto era por plano. Três planos de 4000 linhas passavam um a um e o resultado eram 12000 linhas
// carregadas. ***O limite existia no papel e não no sistema.***
//
// Os testes usam limites CUSTOMIZADOS para medir a acumulação sem importar milhares de linhas: a propriedade é a mesma,
// e a suíte não fica lenta por causa de um número escolhido no teste errado.

func produtoDeTeste(sku string) CommerceProduct {
	// MEDIDO: `UpsertProduct` exige organização ("commerce entry invalid: organization scope is required") — a guarda
	// de tenant do comércio (E-337) funcionando. O produto de teste precisa da organização explícita.
	return CommerceProduct{OrganizationID: "org-a", SKU: sku, Name: "Produto " + sku, PriceCents: 100, Currency: "BRL"}
}

func planoSemOrcamento(skus ...string) CompanyOnboardingPlan {
	linhas := make([]CompanyOnboardingRow, 0, len(skus))
	for _, sku := range skus {
		linhas = append(linhas, CompanyOnboardingRow{SKU: sku, Name: "Produto " + sku, PriceCents: 100, Currency: "BRL"})
	}
	return CompanyOnboardingPlan{OrganizationID: "org-a", Origin: "teste", Rows: linhas}
}

func TestOrcamentoAcumuladoRecusaQuandoOSomatorioPassaDoTeto(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	for _, sku := range []string{"SKU-1", "SKU-2"} {
		if _, err := comercio.UpsertProduct(produtoDeTeste(sku), agora); err != nil {
			t.Fatal(err)
		}
	}
	limites := OnboardingLimits{MaxRows: 3, MaxNameBytes: 512, MaxOriginBytes: 256, MaxTotalBytes: 1 << 20}

	// Já existem 2; o plano acrescenta 2 => 4 > 3: precisa ser RECUSADO.
	errRecusa := ValidarOrcamentoAcumulado(comercio, planoSemOrcamento("SKU-3", "SKU-4"), limites)
	if !errors.Is(errRecusa, ErrOnboardingTooLarge) {
		t.Fatalf("ARMADILHA: 2 existentes + 2 novas com teto 3 foi aceito: %v", errRecusa)
	}
	if !strings.Contains(errRecusa.Error(), "acervo acumulado") {
		t.Fatalf("a recusa precisa explicar que o problema é o ACERVO, não o plano: %v", errRecusa)
	}

	// Já existem 2; o plano acrescenta 1 => 3 <= 3: precisa PASSAR.
	if err := ValidarOrcamentoAcumulado(comercio, planoSemOrcamento("SKU-3"), limites); err != nil {
		t.Fatalf("plano que fecha exatamente o teto precisa passar: %v", err)
	}
}

// Reimportar o MESMO catálogo não pode ser bloqueado: as linhas existentes são ATUALIZADAS, não somadas.
func TestOrcamentoAcumuladoNaoBloqueiaReimportacao(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	for _, sku := range []string{"SKU-1", "SKU-2", "SKU-3"} {
		if _, err := comercio.UpsertProduct(produtoDeTeste(sku), agora); err != nil {
			t.Fatal(err)
		}
	}
	limites := OnboardingLimits{MaxRows: 3, MaxNameBytes: 512, MaxOriginBytes: 256, MaxTotalBytes: 1 << 20}
	// Teto já atingido, mas o plano só repete os SKUs existentes.
	if err := ValidarOrcamentoAcumulado(comercio, planoSemOrcamento("SKU-1", "SKU-2", "SKU-3"), limites); err != nil {
		t.Fatalf("ARMADILHA: reimportar o mesmo catálogo foi bloqueado pelo acervo: %v", err)
	}
}

// Medição do acervo: o número vem do ESTADO REAL, não de um contador paralelo.
func TestMedicaoDoAcervoVemDoEstadoReal(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	if linhas, _ := MedirAcumuladoDoOnboarding(comercio, "org-a"); linhas != 0 {
		t.Fatalf("organização nova começa vazia: %d", linhas)
	}
	for _, sku := range []string{"SKU-1", "SKU-2"} {
		if _, err := comercio.UpsertProduct(produtoDeTeste(sku), agora); err != nil {
			t.Fatal(err)
		}
	}
	linhas, bytes := MedirAcumuladoDoOnboarding(comercio, "org-a")
	if linhas != 2 || bytes <= 0 {
		t.Fatalf("acervo medido errado: linhas=%d bytes=%d", linhas, bytes)
	}
	// E o acervo de OUTRA organização não conta para esta.
	outra, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if linhasOutra, _ := MedirAcumuladoDoOnboarding(outra, "org-a"); linhasOutra != 0 {
		t.Fatalf("ARMADILHA: acervo de uma loja apareceu em outra: %d", linhasOutra)
	}
}

// Integração: aplicar duas vezes passa pelo orçamento e o acervo cresce de verdade.
func TestAplicarDuasVezesPassaPeloOrcamentoEAcervoCresce(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	if _, _, err := ApplyCompanyOnboarding(comercio, planoSemOrcamento("SKU-1", "SKU-2"), agora); err != nil {
		t.Fatalf("primeira aplicação (dentro do teto padrão) precisa passar: %v", err)
	}
	if _, _, err := ApplyCompanyOnboarding(comercio, planoSemOrcamento("SKU-3"), agora); err != nil {
		t.Fatalf("segunda aplicação (dentro do teto padrão) precisa passar: %v", err)
	}
	linhas, _ := MedirAcumuladoDoOnboarding(comercio, "org-a")
	if linhas != 3 {
		t.Fatalf("o acervo precisa somar as DUAS aplicações: %d", linhas)
	}
	// E o plano absurdo continua recusado no volume do próprio plano.
	gigante := onboardingComLinhas(DefaultOnboardingLimits().MaxRows+1, "Produto")
	if _, _, err := ApplyCompanyOnboarding(comercio, gigante, agora); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("plano acima do teto de linhas precisa ser recusado: %v", err)
	}
}
