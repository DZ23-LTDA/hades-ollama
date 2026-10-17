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
	for i := range quantidade {
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

// E-394 — E o cenário que faltava: **catálogo ANTERIOR**. O E-392 declarou este limite em voz alta:
// *"o cenário do teste é empresa nova; com catálogo anterior o custo é 2 gravações e o teste não cobre"* ✗.
//
// Aqui, o plano tem **uma linha que reaproveita um SKU existente** (⇒ o `token.Before` **não** é vazio) e **uma linha
// nova** (⇒ há o que remover) ⇒ o desfazer tem de custar **EXATAMENTE 2**: **um lote** para restaurar o anterior e
// **um lote** para remover o criado. *O número exato depende do cenário — e um limite declarado em voz alta só está
// fechado quando existe teste que o cobre.*
func TestRollbackComCatalogoAnteriorGravaEmDoisLotes(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	// Catálogo ANTERIOR: um produto que já existe, com preço que o plano vai alterar.
	existente := produtoDeTeste("PRE-0001")
	existente.PriceCents = 500
	if _, err := comercio.UpsertProduct(existente, agora); err != nil {
		t.Fatal(err)
	}

	plano := []CompanyOnboardingRow{
		{SKU: "PRE-0001", Name: "Produto que já existia", PriceCents: 1500},
		{SKU: "NOVO-0002", Name: "Produto novo desta carga", PriceCents: 2500},
	}
	_, token, err := ApplyCompanyOnboarding(comercio, CompanyOnboardingPlan{
		OrganizationID: "org-a", Origin: "teste de catálogo anterior", Rows: plano,
		// MEDIDO: sem `Overwrite` explícito o plano com SKU existente e preço divergente é **RECUSADO**
		// (`company onboarding has unresolved conflicts: PRE-0001`) — *conflito tem de ser decisão, não efeito
		// colateral* ✓. E é essa recusa que faz o `token.Before` existir.
		Overwrite: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(token.Before) == 0 {
		t.Fatal("o cenário PRECISA de catálogo anterior (token.Before não pode ser vazio) — senão o teste não cobre o caso")
	}
	if len(token.CreatedSKUs) == 0 {
		t.Fatal("o cenário PRECISA de pelo menos um SKU criado — senão não há o que remover")
	}

	antes := comercio.Gravacoes()
	if err := RollbackCompanyOnboarding(comercio, token, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	gravacoes := comercio.Gravacoes() - antes
	if gravacoes != 2 {
		t.Fatalf("ARMADILHA: desfazer com catálogo anterior custou %d gravação(ões) — o esperado é EXATAMENTE 2 "+
			"(um lote de restauração e um de remoção). Se subiu, alguém voltou a gravar POR LINHA ou POR SKU.",
			gravacoes)
	}
	// E o estado anterior foi RESTAURADO de verdade — não basta gravar pouco, tem de desfazer.
	restaurado, existe := comercio.ProductBySKU("org-a", "PRE-0001")
	if !existe || restaurado.PriceCents != 500 {
		t.Fatalf("ARMADILHA: o desfazer não restaurou o preço anterior (%+v, existe=%v)", restaurado, existe)
	}
	if _, criadoFicou := comercio.ProductBySKU("org-a", "NOVO-0002"); criadoFicou {
		t.Fatal("ARMADILHA: o produto criado pela carga continuou no acervo depois do desfazer")
	}
	t.Logf("MEDIDA ESTRUTURAL: catálogo anterior ⇒ desfazer com %d gravações, preço restaurado e SKU novo removido",
		gravacoes)
}
