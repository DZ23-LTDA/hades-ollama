package agent

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// ORÇAMENTO NO CAMINHO DIRETO (E-360).
//
// **MEDIDO antes** (limite declarado em E-359): o teto acumulado vivia **só no onboarding** — uma chamada direta a
// `UpsertProduct` não consumia orçamento.
//
// ***Limite que depende do chamador lembrar de usar o caminho certo não é limite.***
//
// Aqui o mesmo teto é conferido no `UpsertProduct`, DENTRO do lock (contagem consistente com o que se grava), e a
// atualização de SKU existente continua permitida — o acervo não cresce.

func TestUpsertDiretoRespeitaOrcamentoEDeixaAtualizarNoTeto(t *testing.T) {
	comercio, err := NewCommerceStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	limite := DefaultOnboardingLimits().MaxRows

	// Enche o acervo ATÉ o teto pelo caminho DIRETO — e o tempo disso é medido, porque o custo do caminho direto é
	// informação de operação, não detalhe de teste.
	inicio := time.Now()
	for i := 0; i < limite; i++ {
		produto := produtoDeTeste(fmt.Sprintf("SKU-%06d", i))
		if _, err := comercio.UpsertProduct(produto, agora); err != nil {
			t.Fatalf("produto %d dentro do teto precisa ser aceito: %v", i, err)
		}
	}
	decorrido := time.Since(inicio)

	// ARMADILHA 1: um SKU NOVO acima do teto precisa ser RECUSADO no caminho direto.
	novo := produtoDeTeste("SKU-ACIMA-DO-TETO")
	if _, err := comercio.UpsertProduct(novo, agora); !errors.Is(err, ErrOnboardingTooLarge) {
		t.Fatalf("ARMADILHA: UpsertProduct direto aceitou o produto %d, acima do teto de %d: %v",
			limite+1, limite, err)
	}
	// ARMADILHA 2: ATUALIZAR um SKU existente continua permitido — o acervo não cresce.
	atualizado := produtoDeTeste("SKU-000000")
	atualizado.PriceCents = 999
	atualizado.Name = "Produto renomeado"
	if _, err := comercio.UpsertProduct(atualizado, agora); err != nil {
		t.Fatalf("ARMADILHA: atualizar SKU existente no teto foi bloqueado: %v", err)
	}
	// E o acervo continua exatamente no teto (nem um a mais, nem um a menos).
	linhas, _ := MedirAcumuladoDoOnboarding(comercio, "org-a")
	if linhas != limite {
		t.Fatalf("acervo depois do teste precisa ser exatamente o teto: %d (teto %d)", linhas, limite)
	}
	t.Logf("volume do caminho DIRETO medido: %d produtos em %.2fs (%.2f ms por produto); teto respeitado e atualização permitida",
		limite, decorrido.Seconds(), decorrido.Seconds()*1000/float64(limite))
}
