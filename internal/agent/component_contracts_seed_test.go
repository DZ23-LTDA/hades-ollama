package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// E-277 — SEMEAR OS 16 CONTRATOS CANÔNICOS. Fecha o limite declarado em E-276 ("apenas UM componente registrado até
// agora").
//
// Propriedades travadas:
//  1. semear cobre os 16 canônicos e a completude passa a ser verdadeira;
//  2. semear é IDEMPOTENTE: não duplica e NÃO toca em contrato já promovido (versão e histórico preservados);
//  3. todo contrato semeado tem dono e versão no formato estrito;
//  4. semear em registro já completo não cria nada.
func TestSemearContratosCanonicosCobreOs16EnaoTocaNoQueJaExiste(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "contratos")
	registro, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()

	// 1. Antes de semear, a completude é falsa e nomeia os 16.
	if completo, faltando := CanonicalContractsComplete(registro); completo || len(faltando) != 16 {
		t.Fatalf("registro vazio precisa estar incompleto com 16 faltantes: completo=%v faltando=%d", completo, len(faltando))
	}

	// 2. Um componente JÁ PROMOVIDO antes da semeadura: versão 2.0 com migração registrada.
	if _, err := registro.Register(ComponentContract{Component: "model_gateway", Version: "1.0", Owner: "ia"}, agora); err != nil {
		t.Fatal(err)
	}
	if _, err := registro.Bump("model_gateway", "2.0", ContractChangeQuebra, "remove campo legado", "cto", []string{"product_builder"}, agora); err != nil {
		t.Fatal(err)
	}

	// 3. Semear cria os 15 restantes e deixa o promovido INTACTO.
	criados, err := SeedCanonicalContracts(registro, agora)
	if err != nil {
		t.Fatalf("semear: %v", err)
	}
	if criados != 15 {
		t.Fatalf("esperados 15 contratos criados, vieram %d", criados)
	}
	promovido, ok := registro.Contract("model_gateway")
	if !ok {
		t.Fatal("model_gateway precisa existir")
	}
	if promovido.Version != "2.0" || len(promovido.Migrations) != 1 {
		t.Fatalf("a semeadura NÃO pode tocar em contrato já promovido: %+v", promovido)
	}
	if completo, faltando := CanonicalContractsComplete(registro); !completo || len(faltando) != 0 {
		t.Fatalf("após semear, a completude precisa ser verdadeira: completo=%v faltando=%v", completo, faltando)
	}

	// 4. Todo contrato tem dono e versão estrita.
	for _, contrato := range registro.Contracts() {
		if contrato.Owner == "" {
			t.Fatalf("contrato sem dono não pode existir: %+v", contrato)
		}
		if _, _, err := parseVersion(contrato.Version); err != nil {
			t.Fatalf("versão inválida em %s: %v", contrato.Component, err)
		}
	}

	// 5. IDEMPOTÊNCIA: semear de novo não cria nada e não move versões.
	antes := len(registro.Contracts())
	criadosDeNovo, err := SeedCanonicalContracts(registro, agora.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if criadosDeNovo != 0 || len(registro.Contracts()) != antes {
		t.Fatalf("semear duas vezes precisa ser idempotente: criados=%d antes=%d depois=%d", criadosDeNovo, antes, len(registro.Contracts()))
	}

	// 6. Sobrevive ao reinício já completo.
	reaberto, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if completo, faltando := CanonicalContractsComplete(reaberto); !completo {
		t.Fatalf("após reinício o registro precisa seguir completo: %v", faltando)
	}
	if criados, err := SeedCanonicalContracts(reaberto, agora); err != nil || criados != 0 {
		t.Fatalf("semear registro completo após reinício não pode criar nada: criados=%d err=%v", criados, err)
	}
}
