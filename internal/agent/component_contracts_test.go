package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 3 — REGISTRO DE CONTRATOS VERSIONADOS (E-276). O limite declarado dizia que existem motores reais "mas sem
// registro de contrato versionado por módulo" (verificado: não havia ContractRegistry nem ComponentContract).
//
// As propriedades travadas:
//  1. o registro cobre os 16 componentes canônicos e SABE dizer quais faltam (registro parcial não se diz completo);
//  2. subir versão SEM registrar a migração é RECUSADO;
//  3. voltar versão é RECUSADO;
//  4. mudança que QUEBRA precisa declarar quem quebra;
//  5. o histórico de migração é APPEND-ONLY e sobrevive ao reinício.
func TestRegistroDeContratosExigeMigracaoParaSubirVersao(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "contratos")
	registro, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()

	// 1. Registro vazio sabe dizer o que falta: os 16 canônicos.
	if faltando := registro.MissingCanonicalComponents(); len(faltando) != 16 {
		t.Fatalf("registro vazio precisa apontar os 16 canônicos, veio %d", len(faltando))
	}

	// 2. Registro válido.
	contrato, err := registro.Register(ComponentContract{
		Component: "model_gateway", Version: "1.0", Owner: "cto",
	}, agora)
	if err != nil {
		t.Fatalf("registrar: %v", err)
	}
	if contrato.Version != "1" && contrato.Version != "1.0" {
		t.Fatalf("versão gravada: %q", contrato.Version)
	}
	// Duplicata e não-canônico são recusados.
	if _, err := registro.Register(ComponentContract{Component: "model_gateway", Version: "1.1", Owner: "x"}, agora); !errors.Is(err, ErrContractInvalid) {
		t.Fatalf("duplicata precisa ser recusada: %v", err)
	}
	if _, err := registro.Register(ComponentContract{Component: "componente_inventado", Version: "1.0", Owner: "x"}, agora); !errors.Is(err, ErrContractInvalid) {
		t.Fatalf("componente fora da lista canônica precisa ser recusado: %v", err)
	}
	// Versão malformada é recusada: formato frouxo impede comparar, e é a comparação que sustenta a compatibilidade.
	if _, err := registro.Register(ComponentContract{Component: "skill_manager", Version: "v1", Owner: "x"}, agora); !errors.Is(err, ErrContractInvalid) {
		t.Fatalf("versão fora de maior.menor precisa ser recusada: %v", err)
	}

	// 3. SUBIR SEM MIGRAÇÃO É RECUSADO (a propriedade central).
	if _, err := registro.Bump("model_gateway", "1.1", ContractChangeAditivo, "", "cto", nil, agora); !errors.Is(err, ErrContractMigrationRequired) {
		t.Fatalf("bump sem motivo precisa ser recusado: %v", err)
	}
	if _, err := registro.Bump("model_gateway", "1.1", ContractChangeAditivo, "novo provedor", "", nil, agora); !errors.Is(err, ErrContractMigrationRequired) {
		t.Fatalf("bump sem ator precisa ser recusado: %v", err)
	}
	// 4. QUEBRA sem declarar quem quebra é recusada.
	if _, err := registro.Bump("model_gateway", "2.0", ContractChangeQuebra, "remove campo", "cto", nil, agora); !errors.Is(err, ErrContractBreakingWithoutImpact) {
		t.Fatalf("quebra sem afetados precisa ser recusada: %v", err)
	}
	// 5. Voltar versão é recusado.
	if _, err := registro.Bump("model_gateway", "0.9", ContractChangeCorrecao, "voltar", "cto", nil, agora); !errors.Is(err, ErrContractVersionRegression) {
		t.Fatalf("regressão de versão precisa ser recusada: %v", err)
	}

	// 6. Bump válido: versão sobe e a migração fica registrada.
	subiu, err := registro.Bump("model_gateway", "1.1", ContractChangeAditivo, "novo provedor compatível", "cto", nil, agora)
	if err != nil {
		t.Fatalf("bump válido: %v", err)
	}
	if subiu.Version != "1.1" || len(subiu.Migrations) != 1 {
		t.Fatalf("bump precisa subir versão e registrar migração: %+v", subiu)
	}
	quebra, err := registro.Bump("model_gateway", "2.0", ContractChangeQuebra, "remove campo legado", "cto", []string{"product_builder"}, agora)
	if err != nil {
		t.Fatalf("bump de quebra com afetados: %v", err)
	}
	// 7. APPEND-ONLY: a migração anterior é PRESERVADA.
	if len(quebra.Migrations) != 2 || quebra.Migrations[0].ToVersion != "1.1" {
		t.Fatalf("histórico precisa ser append-only: %+v", quebra.Migrations)
	}

	// 8. REINÍCIO: contratos, versões e histórico sobrevivem.
	reaberto, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, ok := reaberto.Contract("model_gateway")
	if !ok {
		t.Fatal("contrato precisa sobreviver ao reinício")
	}
	if recuperado.Version != "2.0" || len(recuperado.Migrations) != 2 {
		t.Fatalf("versão e histórico precisam sobreviver: %+v", recuperado)
	}
	if len(reaberto.MissingCanonicalComponents()) != 15 {
		t.Fatalf("com um contrato registrado faltam 15: %d", len(reaberto.MissingCanonicalComponents()))
	}
}
