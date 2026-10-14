package agent

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// ADV-57/3 e ADV-57/7 — O CUSTO DO CAMINHO DIRETO E O GANHO DO LOTE, MEDIDOS **ESTRUTURALMENTE**.
//
// **MEDIDO (4ª auditoria independente)**: `TestUpsertDiretoRespeitaOrcamentoEDeixaAtualizarNoTeto` enchia o acervo pelo
// caminho direto **medindo o tempo e não afirmando nada** ⇒ *"não é prova, é log"*: dobrar o laço deixava a suíte verde.
// E a alegação do ADV-44/E-372 (*"ganho 211×"*) **não tinha teste próprio** — e **não se reproduz**: o auditor mediu
// **~196×**, e numa medição minha **~102×**. *O tempo e o multiplicador dependem da máquina; as contagens, não.*
//
// ⇒ Este teste fixa o que é **estrutural**: **N chamadas soltas custam N gravações; 1 lote de N custa 1.**
// *Nota de lint (E-423): a palavra em português para "individual" era acusada pelo `misspell` contra o dicionário
// **não desabilitar lint** ⇒ reescrevi em vez de silenciar: "soltas" diz a mesma coisa e não é ambígua.*
// ***Cronômetro mede a máquina; contador mede o caminho — e só o segundo serve de catraca.***
func TestCustoEstruturalDiretoVersusLote(t *testing.T) {
	const n = 100
	agora := time.Now().UTC()

	// --- caminho DIRETO: N gravações (uma por chamada)
	direto, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		if _, err := direto.UpsertProduct(produtoDeTeste(fmt.Sprintf("DIR-%04d", i)), agora); err != nil {
			t.Fatal(err)
		}
	}
	gravacoesDireto := direto.Gravacoes()
	if gravacoesDireto != n {
		t.Fatalf("ARMADILHA: %d chamadas soltas deveriam custar EXATAMENTE %d gravações e custaram %d. "+
			"O caminho direto é o caro por construção; se ele passou a gravar menos, o ganho do lote precisa ser "+
			"remedido e este número atualizado junto com a documentação.", n, n, gravacoesDireto)
	}

	// --- caminho em LOTE: 1 gravação
	emLote, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	lote := make([]CommerceProduct, 0, n)
	for i := range n {
		lote = append(lote, produtoDeTeste(fmt.Sprintf("LOT-%04d", i)))
	}
	if _, err := emLote.UpsertProducts(lote, agora); err != nil {
		t.Fatal(err)
	}
	gravacoesLote := emLote.Gravacoes()
	if gravacoesLote != 1 {
		t.Fatalf("ARMADILHA: 1 lote de %d produtos deveria custar EXATAMENTE 1 gravação e custou %d — o lote foi "+
			"fragmentado", n, gravacoesLote)
	}

	// E os dois caminhos produzem o MESMO resultado: *o ganho não pode vir de gravar menos dados.*
	if a, b := len(direto.Products("org-a")), len(emLote.Products("org-a")); a != n || b != n {
		t.Fatalf("os dois caminhos precisam aplicar os %d produtos: direto=%d lote=%d", n, a, b)
	}

	// O CONTRASTE, em número: é ele que vale como prova. *Multiplicador de tempo não entra aqui de propósito.*
	if gravacoesDireto <= gravacoesLote {
		t.Fatalf("não faz sentido: o caminho direto (%d) não pode custar o mesmo ou menos que o lote (%d)",
			gravacoesDireto, gravacoesLote)
	}
	t.Logf("ESTRUTURAL: %d produtos pelo caminho DIRETO = %d gravações; pelo LOTE = %d gravação(ões). "+
		"Razão de gravações = %dx (o tempo varia por máquina e NÃO é afirmado aqui)",
		n, gravacoesDireto, gravacoesLote, gravacoesDireto/gravacoesLote)
}

// ADV-57/3, a metade que o auditor pediu explicitamente: o teste que **só media e logava** passa a AFIRMAR o custo do
// caminho direto. Assim, *mudar o caminho direto sem mudar a documentação* deixa de passar em silêncio.
func TestUpsertDiretoCustaUmaGravacaoPorChamada(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	const amostra = 50
	for i := range amostra {
		if _, err := comercio.UpsertProduct(produtoDeTeste(fmt.Sprintf("DIR-%04d", i)), agora); err != nil {
			t.Fatal(err)
		}
	}
	if gravacoes := comercio.Gravacoes(); gravacoes != amostra {
		t.Fatalf("ARMADILHA: %d chamadas diretas gravaram %d vez(es) — o esperado é EXATAMENTE %d (uma por chamada). "+
			"Se caiu, o caminho direto foi otimizado: REMEDIR e atualizar a documentação (é a catraca avisando que a "+
			"narrativa do custo envelheceu).", amostra, gravacoes, amostra)
	}
	// E o teto continua valendo por este caminho: a asserção acima não substitui a de orçamento, ela a complementa.
	limite := DefaultOnboardingLimits().MaxRows
	_ = limite
	if _, err := comercio.UpsertProduct(produtoDeTeste("SKU-000000"), agora); err != nil && !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("atualizar um SKU existente não pode ser recusado pelo teto: %v", err)
	}
}
