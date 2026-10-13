package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ONBOARDING COM VOLUME — LIMITES DECLARADOS (E-353).
//
// **MEDIDO antes**: `PlanCompanyOnboarding` recusava apenas o **vazio** e não tinha teto de volume; `ApplyCompanyOnboarding`
// passa por ele. Logo, um payload de **milhões de linhas** (ou um nome de gigabytes) seria aceito e consumiria memória
// e disco sem que nada no código dissesse qual era o teto.
//
// ***Limite que não está escrito não é limite: é surpresa.***

func onboardingComLinhas(quantidade int, nome string) CompanyOnboardingPlan {
	linhas := make([]CompanyOnboardingRow, 0, quantidade)
	for i := 0; i < quantidade; i++ {
		linhas = append(linhas, CompanyOnboardingRow{SKU: fmt.Sprintf("SKU-%06d", i), Name: nome,
			PriceCents: int64(100 + i), Currency: "BRL", InitialStock: 1})
	}
	return CompanyOnboardingPlan{OrganizationID: "org-a", Origin: "teste", Rows: linhas}
}

func TestOnboardingRecusaAcimaDoLimiteDeLinhas(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	limites := DefaultOnboardingLimits()
	plano := onboardingComLinhas(limites.MaxRows+1, "Produto")
	if _, err := PlanCompanyOnboarding(comercio, plano); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("ARMADILHA: plano com %d linhas (limite %d) foi aceito: %v", limites.MaxRows+1, limites.MaxRows, err)
	}
	// E o limite é conferido ANTES de trabalhar: nada foi criado no comércio.
	if produtos := comercio.Products("org-a"); len(produtos) != 0 {
		t.Fatalf("plano recusado não pode ter criado produto algum: %d", len(produtos))
	}
}

func TestOnboardingRecusaNomeEOrigemGigantes(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	limites := DefaultOnboardingLimits()
	gigante := strings.Repeat("x", limites.MaxNameBytes+1)
	if _, err := PlanCompanyOnboarding(comercio, onboardingComLinhas(1, gigante)); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("ARMADILHA: nome com %d bytes (limite %d) foi aceito: %v", len(gigante), limites.MaxNameBytes, err)
	}
	plano := onboardingComLinhas(1, "Produto")
	plano.Origin = strings.Repeat("o", limites.MaxOriginBytes+1)
	if _, err := PlanCompanyOnboarding(comercio, plano); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("ARMADILHA: origem com %d bytes (limite %d) foi aceita: %v", len(plano.Origin), limites.MaxOriginBytes, err)
	}
}

func TestLimitesDeVolumeSaoDeclaradosEConfiguraveis(t *testing.T) {
	padrao := DefaultOnboardingLimits()
	if padrao.MaxRows <= 0 || padrao.MaxNameBytes <= 0 || padrao.MaxOriginBytes <= 0 || padrao.MaxTotalBytes <= 0 {
		t.Fatalf("os limites padrão precisam existir e ser positivos: %+v", padrao)
	}
	// Limite próprio e mais apertado: a validação é parametrizável (não está soldada no caminho).
	justo := OnboardingLimits{MaxRows: 2, MaxNameBytes: 8, MaxOriginBytes: 8, MaxTotalBytes: 1024}
	if err := ValidarVolumeDoPlano(onboardingComLinhas(3, "ok"), justo); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("com limite de 2 linhas, um plano de 3 precisa ser recusado: %v", err)
	}
	if err := ValidarVolumeDoPlano(onboardingComLinhas(2, "ok"), justo); err != nil {
		t.Fatalf("plano dentro do limite precisa passar: %v", err)
	}
	// Aplicar os limites padrão a um plano vazio não pode explodir.
	if err := ValidarVolumeDoPlano(CompanyOnboardingPlan{OrganizationID: "org-a"}, padrao); err != nil {
		t.Fatalf("plano vazio precisa passar na validação de volume (a exigência de linhas é outra regra): %v", err)
	}
}

// MEDIÇÃO DE VOLUME: um plano grande MAS dentro do limite precisa ser aplicado de verdade — e o tempo é medido.
func TestOnboardingComVolumeDentroDoLimiteAplica(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelar()
	_ = ctx
	raiz := t.TempDir()
	comercio, err := NewCommerceStoreAt(raiz)
	if err != nil {
		t.Fatal(err)
	}
	limites := DefaultOnboardingLimits()
	quantidade := 2000
	if quantidade > limites.MaxRows {
		quantidade = limites.MaxRows
	}
	plano := onboardingComLinhas(quantidade, "Produto de volume")
	inicio := time.Now()
	preview, rollback, err := ApplyCompanyOnboarding(comercio, plano, time.Now().UTC())
	if err != nil {
		t.Fatalf("plano de %d linhas dentro do limite precisa ser aplicado: %v", quantidade, err)
	}
	decorrido := time.Since(inicio)
	// MEDIDO: a prévia tem ToCreate/ToUpdate/Unchanged/Conflicts/Duplicates/InvalidRows/ConflictKnown, e o rollback tem
	// OrganizationID/Origin/Before/CreatedSKUs/AppliedAt. *Campos medidos, não supostos.*
	if len(preview.ToCreate) != quantidade {
		t.Fatalf("a prévia precisa reportar %d criações, reportou %d (%+v)", quantidade, len(preview.ToCreate), preview)
	}
	if len(rollback.CreatedSKUs) != quantidade {
		t.Fatalf("o rollback precisa listar %d SKUs criados, listou %d", quantidade, len(rollback.CreatedSKUs))
	}
	if !rollback.AppliedAt.Equal(time.Time{}) == false && rollback.AppliedAt.IsZero() {
		t.Fatal("o rollback precisa registrar quando foi aplicado")
	}
	produtos := comercio.Products("org-a")
	if len(produtos) != quantidade {
		t.Fatalf("esperava %d produtos aplicados, medido %d", quantidade, len(produtos))
	}
	// E-364: PROVA EM DISCO. Medir 0,01s para 2000 linhas é bom demais para aceitar sem verificar — e o teste antigo
	// só olhava o estado em MEMÓRIA. *Resultado excelente sem verificação é o lugar onde o defeito se esconde.*
	arquivos, err := filepath.Glob(filepath.Join(raiz, "*.json"))
	if err != nil || len(arquivos) == 0 {
		t.Fatalf("a gravação precisa existir em disco: %v %v", arquivos, err)
	}
	conteudo, err := os.ReadFile(arquivos[0])
	if err != nil {
		t.Fatal(err)
	}
	var gravado struct {
		Products []struct {
			SKU string `json:"sku"`
		} `json:"products"`
	}
	if err := json.Unmarshal(conteudo, &gravado); err != nil {
		t.Fatalf("arquivo do comércio ilegível: %v", err)
	}
	if len(gravado.Products) != quantidade {
		t.Fatalf("ARMADILHA: %d produtos em memória mas %d no DISCO — o lote não foi gravado",
			quantidade, len(gravado.Products))
	}
	// E-365: MEDIDA ESTRUTURAL, imune ao ruído da máquina: quantas vezes o arquivo foi GRAVADO para aplicar o lote.
	// *Contar é medir o custo do caminho; o cronômetro mede a máquina.*
	if gravacoes := comercio.Gravacoes(); gravacoes > 10 {
		t.Fatalf("ARMADILHA: %d linhas causaram %d gravações — o lote não está sendo usado", quantidade, gravacoes)
	} else {
		t.Logf("MEDIDA ESTRUTURAL: %d linhas aplicadas com %d gravação(ões) de arquivo", quantidade, gravacoes)
	}
	t.Logf("onboarding com volume medido: %d linhas aplicadas em %.2fs e CONFIRMADAS em disco (%d produtos no arquivo); limite %d linhas, %d bytes",
		quantidade, decorrido.Seconds(), len(gravado.Products), limites.MaxRows, limites.MaxTotalBytes)
}
