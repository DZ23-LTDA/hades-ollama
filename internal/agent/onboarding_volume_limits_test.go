package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	for i := range quantidade {
		linhas = append(linhas, CompanyOnboardingRow{
			SKU: fmt.Sprintf("SKU-%06d", i), Name: nome,
			PriceCents: int64(100 + i), Currency: "BRL", InitialStock: 1,
		})
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
	// 🚨 ADV-57/4 (4ª auditoria independente, MEDIDO): aqui havia `ctx, cancelar := context.WithTimeout(…, 5*time.Minute)`
	// seguido de `_ = ctx` — **código MORTO**: o contexto nunca era passado a `ApplyCompanyOnboarding`, então a guarda
	// de cinco minutos não protegia nada, só parecia proteger. ***Guarda que não pode disparar não é proteção: é
	// sinalizador de intenção.*** REMOVIDO. *O limite real do teste é o timeout do próprio `go test`.*
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
	//
	// 🚨 ADV-57/6 (4ª auditoria independente): antes isto era um `glob("*.json")` + `arquivos[0]`, **sem conferir que o
	// arquivo lido era o da organização** — *prova de disco que lê "algum" arquivo não é prova de disco do estado.*
	caminho := comercio.commercePath("org-a")
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("a gravação da organização precisa existir em disco (%s): %v", caminho, err)
	}
	var gravado struct {
		Products []struct {
			SKU string `json:"sku"`
		} `json:"products"`
		// MEDIDO DUAS VEZES: primeiro como `map[string]int` deu `cannot unmarshal array`, depois como `[]map[string]any`
		// deu `cannot unmarshal object` ⇒ *a forma do estoque não é nenhuma das duas que eu supus.* Em vez de adivinhar
		// uma terceira, decodifico **agnóstico** (`json.RawMessage`) e afirmo o que importa: **o estoque chegou ao
		// arquivo, não-vazio**. *Prova que só vale para a forma que o autor supôs não é prova do conteúdo.*
		Stock json.RawMessage `json:"stock"`
	}
	if err := json.Unmarshal(conteudo, &gravado); err != nil {
		t.Fatalf("arquivo do comércio ilegível: %v", err)
	}
	if len(gravado.Products) != quantidade {
		t.Fatalf("ARMADILHA: %d produtos em memória mas %d no DISCO — o lote não foi gravado",
			quantidade, len(gravado.Products))
	}
	// 🚨 ADV-57/1 (4ª auditoria independente, MEDIDO): a asserção era `Gravacoes() <= 10` e **passava com UMA
	// gravação** — desligando o lote de ESTOQUE a contagem caía para 1 e `1 > 10` é falso ⇒ *a alegação publicada
	// ("2000 linhas com 2 gravações") NÃO era verificada por asserção nenhuma*, e a prova em disco só contava
	// `products`, **nunca** `stock`. ***A parte da alegação que ninguém mede é a parte que envelhece sem avisar.***
	// E o ESTOQUE também passa a ser conferido NO DISCO:
	if len(gravado.Stock) == 0 || string(gravado.Stock) == "null" || string(gravado.Stock) == "{}" {
		t.Fatalf("ARMADILHA: o ESTOQUE inicial não chegou ao DISCO (%s) — a alegação de '2 gravações' incluía o lote de "+
			"estoque, e ele não está no arquivo", string(gravado.Stock))
	}
	// E-365: MEDIDA ESTRUTURAL, imune ao ruído da máquina: quantas vezes o arquivo foi GRAVADO para aplicar o lote.
	// *Contar é medir o custo do caminho; o cronômetro mede a máquina.*
	// 🚨 ADV-57/2: o limiar `> 10` também não distinguia **1 lote de N lotes** (fatias de 250 sobre 1000 davam 4
	// gravações e passavam). O número EXATO é o que prova o lote: **2** — um lote de produtos e um de estoque.
	if gravacoes := comercio.Gravacoes(); gravacoes != 2 {
		t.Fatalf("ARMADILHA: %d linhas causaram %d gravações — o esperado é EXATAMENTE 2 (um lote de produtos e um de "+
			"estoque). Mais que isso: o lote foi fragmentado; menos: uma das gravações não aconteceu, e a alegação "+
			"publicada não vale.", quantidade, gravacoes)
	} else {
		t.Logf("MEDIDA ESTRUTURAL: %d linhas aplicadas com %d gravação(ões) de arquivo (1 produtos + 1 estoque)",
			quantidade, gravacoes)
	}
	t.Logf("onboarding com volume medido: %d linhas aplicadas em %.2fs e CONFIRMADAS em disco (%d produtos no arquivo); limite %d linhas, %d bytes",
		quantidade, decorrido.Seconds(), len(gravado.Products), limites.MaxRows, limites.MaxTotalBytes)
}
