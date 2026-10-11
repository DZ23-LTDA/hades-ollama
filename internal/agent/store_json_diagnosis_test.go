package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

// DIAGNÓSTICO DO MISTÉRIO DE E-166 (E-167): o MESMO comando, o MESMO struct e a MESMA versão de Go produziram
// JSON de 98 KB com 204 alocações no contêiner contra 71 KB com 4 alocações no Windows.
//
// Isso não é curiosidade acadêmica: se o FORMATO gravado depende do ambiente, o arquivo de um Hades pode não ser
// legível por outro — e o Hades grava tudo em JSON (`writeJSONAtomic` é o caminho único). Antes de qualquer
// conclusão, é preciso saber se os BYTES diferem e por quê.
//
// Este diagnóstico imprime os fatos do ambiente e os primeiros bytes do JSON nos dois lados, para a comparação
// ser feita com dados e não com hipótese.

// TestDiagnoseJSONEnvironment imprime ambiente e forma do JSON. Não falha por diferença: o objetivo é MEDIR.
func TestDiagnoseJSONEnvironment(t *testing.T) {
	memorias := cargaDeMemorias("prj_diag")
	dados, err := json.MarshalIndent(memorias, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	alocacoes := testing.AllocsPerRun(20, func() {
		if _, err := json.MarshalIndent(memorias, "", "  "); err != nil {
			t.Fatal(err)
		}
	})
	// Uma segunda codificação SEM indentação, para separar "tamanho por indentação" de "tamanho por conteúdo".
	compacto, err := json.Marshal(memorias)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("AMBIENTE go=%s goos=%s goarch=%s GOEXPERIMENT=%q GOFLAGS=%q GODEBUG=%q",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, os.Getenv("GOEXPERIMENT"), os.Getenv("GOFLAGS"), os.Getenv("GODEBUG"))
	t.Logf("TAMANHO indentado=%d bytes compacto=%d bytes razao=%.3f", len(dados)+1, len(compacto),
		float64(len(dados)+1)/float64(len(compacto)))
	t.Logf("ALOCACOES marshal_indent=%.2f por execucao", alocacoes)
	t.Logf("ESCAPES html_escapado=%v barra_unicode=%d aspas_escapadas=%d",
		strings.Contains(string(dados), `\u003c`) || strings.Contains(string(dados), `\u003e`),
		strings.Count(string(dados), `\u`), strings.Count(string(dados), `\"`))
	// E-167: as invariantes abaixo valem em QUALQUER toolchain — foi a falta delas que deixou a diferença de
	// 4 vs 204 alocações passar por "diferença de plataforma" quando era **diferença de versão do Go**.
	if len(dados)+1 <= len(compacto) {
		t.Fatalf("a indentação precisa aumentar o tamanho: %d vs %d", len(dados)+1, len(compacto))
	}
	var ida []Memory
	if err := json.Unmarshal(dados, &ida); err != nil {
		t.Fatalf("o JSON precisa continuar legível: %v", err)
	}
	if len(ida) != len(memorias) || ida[0].ID != memorias[0].ID || ida[0].Content != memorias[0].Content {
		t.Fatalf("ida e volta divergente: %d registros", len(ida))
	}
	// Determinismo: a MESMA entrada precisa gerar os MESMOS bytes (dois lados já divergiram por toolchain).
	outra, err := json.MarshalIndent(memorias, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(outra) != string(dados) {
		t.Fatal("a serialização precisa ser determinística para a mesma entrada")
	}
	// Os primeiros bytes mostram o formato real: se as plataformas escrevem igual, esta linha é idêntica.
	limite := 220
	if len(dados) < limite {
		limite = len(dados)
	}
	t.Logf("PRIMEIROS_BYTES %q", string(dados[:limite]))
	// E o CONTEÚDO de uma memória, para o tamanho por registro ser comparável.
	t.Logf("REGISTRO bytes_por_memoria=%.1f exemplo=%q", float64(len(dados)+1)/float64(len(memorias)),
		fmt.Sprintf("%+v", memorias[0]))
}
