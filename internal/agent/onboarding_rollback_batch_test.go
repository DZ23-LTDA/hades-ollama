package agent

import (
	"fmt"
	"testing"
	"time"
)

// E-392 / ADV-57/5b — O DESFAZER DO ONBOARDING NÃO PODE GRAVAR POR SKU.
//
// **MEDIDO (4ª auditoria independente)**: `RollbackCompanyOnboarding` chamava `RemoveProduct` **por SKU**, e
// `RemoveProduct` grava o arquivo inteiro **por chamada** ⇒ desfazer 500 linhas custava **1000 gravações em 2,72s**
// (antes do E-390) e ainda **500 gravações em 1,37s** (depois do E-390, que loteou só a restauração).
// ***Faltava a API, não faltava frouxidão no teste.***
//
// Este teste é a **prova permanente** que o E-390 declarou faltar: ele mede o **custo estrutural** (gravações, não
// tempo) do desfazer e **falha** se alguém voltar a gravar por SKU. *Cronômetro mede a máquina; contador mede o
// caminho.*
func TestRollbackDoOnboardingNaoGravaPorSku(t *testing.T) {
	const quantidade = 100
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	linhas := make([]CompanyOnboardingRow, 0, quantidade)
	for i := 0; i < quantidade; i++ {
		linhas = append(linhas, CompanyOnboardingRow{
			SKU: fmt.Sprintf("RB-%04d", i), Name: fmt.Sprintf("produto %d", i), PriceCents: 1000,
		})
	}
	_, token, err := ApplyCompanyOnboarding(comercio, CompanyOnboardingPlan{
		OrganizationID: "org-a", Origin: "teste de lote no desfazer", Rows: linhas,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if aplicados := len(comercio.Products("org-a")); aplicados != quantidade {
		t.Fatalf("o cenário precisa de %d produtos aplicados, medido %d", quantidade, aplicados)
	}

	antes := comercio.Gravacoes()
	if err := RollbackCompanyOnboarding(comercio, token, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	gravacoes := comercio.Gravacoes() - antes

	// EXATO, não "<= N": o limiar frouxo foi o defeito do ADV-57/1 — *um teto que aceita 1 também aceita o caminho
	// errado*. **MEDIDO: 1 gravação** — *o cenário é uma empresa NOVA, então `token.Before` está vazio e só o lote de
	// REMOÇÃO grava.* Num cenário com catálogo anterior seriam **2** (um lote de restauração + um de remoção) ⇒ *o
	// número exato depende do cenário, e é isso que o teste tem de dizer em vez de aceitar um teto.*
	if gravacoes != 1 {
		t.Fatalf("ARMADILHA: desfazer %d linhas de uma empresa NOVA custou %d gravação(ões) — o esperado é EXATAMENTE "+
			"1 (só o lote de remoção; `token.Before` está vazio). Se subiu, alguém voltou a gravar POR SKU.",
			quantidade, gravacoes)
	}
	if restantes := len(comercio.Products("org-a")); restantes != 0 {
		t.Fatalf("ARMADILHA: o desfazer deixou %d produto(s) — o lote economizou gravações mas não desfez", restantes)
	}
	t.Logf("MEDIDA ESTRUTURAL: desfazer %d linhas com %d gravação(ões) e 0 produtos restantes", quantidade, gravacoes)
}
