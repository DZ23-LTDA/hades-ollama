package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// E-280 — VERIFICAÇÃO POR SÍMBOLO. O limite de E-278/E-279 era "a auditoria prova existência de arquivo, não
// conformidade semântica". Este degrau confere que o SÍMBOLO DECLARADO aparece no arquivo.
//
// ESCopo HONESTO: é mais forte que "o arquivo existe" (o arquivo pode existir sem o componente) e mais fraco que
// "o comportamento cumpre a versão" (exigiria teste de contrato por componente). O degrau é rotulado.
func TestVerificacaoPorSimboloEncontraOComponenteEAcusaSimboloErrado(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "contratos")
	registro, err := NewContractRegistry(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SeedCanonicalContracts(registro, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	// 1. Todo contrato tem arquivo E símbolo declarados.
	pontos := ComponentSymbolPoints()
	for _, contrato := range registro.Contracts() {
		ponto, ok := pontos[contrato.Component]
		if !ok || ponto.File == "" || ponto.Symbol == "" {
			t.Fatalf("contrato sem arquivo/símbolo declarado: %s", contrato.Component)
		}
		// Símbolo genérico passaria sempre: a verificação precisa de identificador próprio do componente.
		if len(ponto.Symbol) < 4 {
			t.Fatalf("símbolo frouxo declarado para %s: %q", contrato.Component, ponto.Symbol)
		}
	}

	// 2. Os símbolos aparecem nos arquivos reais.
	relatorio, err := AuditContractSymbols(registro)
	if err != nil {
		t.Fatal(err)
	}
	if relatorio.Audited != len(registro.Contracts()) {
		t.Fatalf("precisa auditar todos os contratos: %d de %d", relatorio.Audited, len(registro.Contracts()))
	}
	// O que o mapa DECLARA precisa estar presente. Componente sem símbolo verificado aparece em Missing e é
	// REPORTADO no log: limite declarado, não escondido.
	if relatorio.Present != len(pontos) {
		t.Fatalf("todo símbolo declarado precisa estar presente: declarados=%d presentes=%d ausentes=%+v", len(pontos), relatorio.Present, relatorio.Missing)
	}
	if len(relatorio.Missing) > 0 {
		t.Logf("LIMITE DECLARADO (E-280): %d componente(s) sem símbolo verificado: %+v", len(relatorio.Missing), relatorio.Missing)
	}

	// 3. CAMINHO NEGATIVO: símbolo errado é ACUSADO, mesmo com o arquivo existindo.
	umContrato := &ContractRegistry{contracts: map[string]ComponentContract{
		"hades_core": {Component: "hades_core", Version: "1.0", Owner: "cto"},
	}}
	negativo, err := auditContractSymbolsAt(umContrato, map[string]ComponentPoint{
		"hades_core": {File: "service_agents.go", Symbol: "SimboloQueNaoExisteEmLugarNenhum"},
	}, packageDir())
	if err != nil {
		t.Fatal(err)
	}
	if negativo.Present != 0 || len(negativo.Missing) != 1 {
		t.Fatalf("símbolo ausente precisa ser ACUSADO: %+v", negativo)
	}
	if negativo.Missing[0].Note == "" {
		t.Fatal("a acusação precisa dizer POR QUE (arquivo existe, símbolo ausente)")
	}
}
