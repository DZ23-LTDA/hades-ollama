package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// P0-B / T2 — MATRIZ DE FIXTURES E MODELOS (E-344).
//
// A primeira medição (E-343) provou o caminho RED → GREEN com **um** fixture e **um** modelo. Uma medição só não é
// evidência de capacidade: é uma **observação**. Aqui a medição ganha **variedade** (quatro classes de bug) e
// **repetição** (mais de um modelo), com **taxa de sucesso** explícita — e com `NAO_MEDIDO` onde não houver modelo.
//
// ***Um número sem repetição mede o dia; uma matriz medida mede a capacidade.***

// t2Modulo é o cabeçalho de módulo comum aos fixtures (sem dependências externas, sem rede).
const t2Modulo = "module fixturet2\n\ngo 1.21\n"

// FixtureT2Comparacao: erro de comparação — maioridade por `>` em vez de `>=`.
func FixtureT2Comparacao() HarnessBenchmarkTask {
	comBug := `package calc

// MaiorDeIdade diz se a pessoa pode responder por si.
func MaiorDeIdade(idade int) bool {
	return idade > 18
}
`
	correto := `package calc

// MaiorDeIdade diz se a pessoa pode responder por si.
func MaiorDeIdade(idade int) bool {
	return idade >= 18
}
`
	teste := `package calc

import "testing"

func TestMaiorDeIdadeInclui18(t *testing.T) {
	if !MaiorDeIdade(18) {
		t.Fatal("18 anos precisa contar como maior de idade")
	}
	if MaiorDeIdade(17) {
		t.Fatal("17 anos não pode contar como maior de idade")
	}
	if !MaiorDeIdade(40) {
		t.Fatal("40 anos precisa contar como maior de idade")
	}
}
`
	buggy := map[string]string{"go.mod": t2Modulo, "idade.go": comBug, "idade_test.go": teste}
	ok := map[string]string{"go.mod": t2Modulo, "idade.go": correto, "idade_test.go": teste}
	return HarnessBenchmarkTask{Name: "fixture-t2-comparacao", Files: ok, BuggyFiles: buggy, ReferenceFix: ok}
}

// FixtureT2DivisaoPorZero: caso-limite ausente — média de lista vazia divide por zero.
func FixtureT2DivisaoPorZero() HarnessBenchmarkTask {
	comBug := `package calc

// Media devolve a média dos valores.
func Media(valores []int) int {
	total := 0
	for _, valor := range valores {
		total += valor
	}
	return total / len(valores)
}
`
	correto := `package calc

// Media devolve a média dos valores; lista vazia devolve 0.
func Media(valores []int) int {
	if len(valores) == 0 {
		return 0
	}
	total := 0
	for _, valor := range valores {
		total += valor
	}
	return total / len(valores)
}
`
	teste := `package calc

import "testing"

func TestMediaDeListaVaziaEhZero(t *testing.T) {
	if got := Media(nil); got != 0 {
		t.Fatalf("Media(nil) = %d, queria 0", got)
	}
	if got := Media([]int{2, 4}); got != 3 {
		t.Fatalf("Media([2 4]) = %d, queria 3", got)
	}
}
`
	buggy := map[string]string{"go.mod": t2Modulo, "media.go": comBug, "media_test.go": teste}
	ok := map[string]string{"go.mod": t2Modulo, "media.go": correto, "media_test.go": teste}
	return HarnessBenchmarkTask{Name: "fixture-t2-caso-limite", Files: ok, BuggyFiles: buggy, ReferenceFix: ok}
}

// FixtureT2Fatiamento: erro de índice — devolve a cauda em vez do começo.
func FixtureT2Fatiamento() HarnessBenchmarkTask {
	comBug := `package calc

// Primeiros devolve os n primeiros valores.
func Primeiros(valores []int, n int) []int {
	if n > len(valores) {
		n = len(valores)
	}
	return valores[n:]
}
`
	correto := `package calc

// Primeiros devolve os n primeiros valores.
func Primeiros(valores []int, n int) []int {
	if n > len(valores) {
		n = len(valores)
	}
	return valores[:n]
}
`
	teste := `package calc

import (
	"reflect"
	"testing"
)

func TestPrimeirosPegaOComeco(t *testing.T) {
	if got := Primeiros([]int{1, 2, 3, 4}, 2); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("Primeiros = %v, queria [1 2]", got)
	}
	if got := Primeiros([]int{1, 2}, 9); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("pedido maior que a lista = %v, queria [1 2]", got)
	}
}
`
	buggy := map[string]string{"go.mod": t2Modulo, "fatia.go": comBug, "fatia_test.go": teste}
	ok := map[string]string{"go.mod": t2Modulo, "fatia.go": correto, "fatia_test.go": teste}
	return HarnessBenchmarkTask{Name: "fixture-t2-fatiamento", Files: ok, BuggyFiles: buggy, ReferenceFix: ok}
}

// AllT2Fixtures devolve o conjunto medido — quatro classes de bug distintas.
func AllT2Fixtures() []HarnessBenchmarkTask {
	return []HarnessBenchmarkTask{
		FixtureT2(),               // laço que ignora o primeiro item
		FixtureT2Comparacao(),     // comparação errada em caso-limite
		FixtureT2DivisaoPorZero(), // caso-limite ausente (divisão por zero)
		FixtureT2Fatiamento(),     // erro de índice em fatiamento
	}
}

// T2MatrixSummary resume a matriz SEM esconder o que não foi medido.
type T2MatrixSummary struct {
	Total         int      `json:"total"`
	Green         int      `json:"green"`
	GreenPrimeira int      `json:"green_primeira_tentativa"`
	Trapacou      int      `json:"harness_trapacou"`
	Vermelho      int      `json:"permaneceu_vermelho"`
	Modelos       []string `json:"modelos"`
	TaxaSucesso   string   `json:"taxa_sucesso"`
	NaoMedido     []string `json:"nao_medido,omitempty"`
}

// SummarizeT2Matrix calcula o resumo a partir dos resultados MEDIDOS.
func SummarizeT2Matrix(resultados []HarnessBenchmarkResult) T2MatrixSummary {
	resumo := T2MatrixSummary{Total: len(resultados)}
	modelos := map[string]bool{}
	for _, resultado := range resultados {
		if resultado.Model != "" {
			modelos[resultado.Model] = true
		}
		switch resultado.Verdict {
		case "GREEN":
			resumo.Green++
			if resultado.Attempts == 1 {
				resumo.GreenPrimeira++
			}
		case "HARNESS_TRAFICOU":
			resumo.Trapacou++
		case "RED":
			resumo.Vermelho++
		}
	}
	nomes := make([]string, 0, len(modelos))
	for nome := range modelos {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	resumo.Modelos = nomes
	if resumo.Total > 0 {
		resumo.TaxaSucesso = fmt.Sprintf("%d/%d", resumo.Green, resumo.Total)
	}
	return resumo
}

// RunT2Matrix roda a matriz completa: cada harness contra cada fixture, com tentativas contadas.
func RunT2Matrix(ctx context.Context, harnesses []HarnessProposer, tasks []HarnessBenchmarkTask, tentativas int) []HarnessBenchmarkResult {
	resultados := make([]HarnessBenchmarkResult, 0, len(harnesses)*len(tasks))
	for _, harness := range harnesses {
		for _, task := range tasks {
			if ctx.Err() != nil {
				// Cancelou: NÃO inventa resultado para o que não rodou.
				resultados = append(resultados, HarnessBenchmarkResult{Task: task.Name, Harness: harness.Name(),
					Model: harness.Model(), GreenExit: -1, Verdict: "NAO_MEDIDO", Reason: "contexto cancelado"})
				continue
			}
			inicio := time.Now()
			resultado, err := RunHarnessBenchmark(ctx, harness, task, tentativas)
			if err != nil {
				resultado = HarnessBenchmarkResult{Task: task.Name, Harness: harness.Name(),
					Model: harness.Model(), GreenExit: -1, Verdict: "NAO_MEDIDO", Reason: err.Error(),
					DurationSeconds: time.Since(inicio).Seconds()}
			}
			resultados = append(resultados, resultado)
		}
	}
	return resultados
}

// EscreverArtefatoDaMatrizT2 grava a matriz MEDIDA em `docs/parity/HADES_T2_MATRIX.md`.
//
// POR QUE UM ARTEFATO (E-344): a matriz depende de modelos locais; no CI ela PULA. Sem artefato, o número medido fica
// só no terminal de quem rodou — *e o que não fica registrado envelhece como boato*. O arquivo registra modelos, data e
// o resultado por combinação, inclusive os que ficaram vermelhos ou trapacearam.
func EscreverArtefatoDaMatrizT2(t interface {
	Logf(string, ...any)
	Fatalf(string, ...any)
}, resultados []HarnessBenchmarkResult, resumo T2MatrixSummary) {
	raiz, achou := raizDoRepositorioParaArtefato()
	if !achou {
		t.Logf("artefato da matriz NÃO gravado: docs/parity não encontrado a partir do arquivo-fonte")
		return
	}
	var b strings.Builder
	b.WriteString("# HADES — MATRIZ T2 (medida)\n\n")
	b.WriteString("**Gerado por:** `TestMatrizRealComModelosLocais` · **Data:** " + time.Now().UTC().Format(time.RFC3339) + "\n\n")
	b.WriteString("Medição de **RED → GREEN** em fixtures próprios, com harness mínimo sobre Ollama local. ")
	b.WriteString("Harness externo (Claude Code/Codex/Manus) exige credencial ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒ ")
	b.WriteString("`NAO_MEDIDO`. **Nenhuma alegação de paridade** sai deste arquivo.\n\n")
	b.WriteString("| modelo | fixture | red | green | tentativas | hunks | linhas | duração (s) | veredito |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range resultados {
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %d | %.1f | %s |\n",
			r.Model, r.Task, r.RedExit, r.GreenExit, r.Attempts, r.DiffHunks, r.DiffLines, r.DurationSeconds, r.Verdict))
	}
	b.WriteString("\n**Resumo medido:** total " + fmt.Sprint(resumo.Total) + " · verde " + fmt.Sprint(resumo.Green) +
		" · verde na 1ª tentativa " + fmt.Sprint(resumo.GreenPrimeira) + " · trapaceou " + fmt.Sprint(resumo.Trapacou) +
		" · permaneceu vermelho " + fmt.Sprint(resumo.Vermelho) + " · **taxa " + resumo.TaxaSucesso + "**\n\n")
	b.WriteString("**Limites:** modelos locais apenas; `veredito=HARNESS_TRAFICOU` significa que o modelo tentou alterar o ")
	b.WriteString("arquivo de TESTE (o anti-trapaça registra e o laço continua com feedback); `green=-1` significa ")
	b.WriteString("**não executado** (ausência de medição não é aprovação).\n")
	if err := os.WriteFile(filepath.Join(raiz, "HADES_T2_MATRIX.md"), []byte(b.String()), 0o600); err != nil {
		t.Logf("artefato da matriz não gravado: %v", err)
	}
}

// raizDoRepositorioParaArtefato localiza `docs/parity` A PARTIR DO ARQUIVO-FONTE, não do diretório de trabalho.
//
// POR QUE ISSO IMPORTA (E-345): `go test` roda com o **diretório do pacote** como diretório de trabalho. Um caminho
// relativo `docs/parity` NÃO existe ali — o artefato era silenciosamente pulado, e o registro ficava sem evidência.
// ***Artefato que "às vezes" é gravado é pior que artefato nenhum: parece evidência.***
func raizDoRepositorioParaArtefato() (string, bool) {
	_, arquivo, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	dir := filepath.Dir(arquivo)
	for i := 0; i < 6; i++ {
		candidato := filepath.Join(dir, "docs", "parity")
		if info, err := os.Stat(candidato); err == nil && info.IsDir() {
			return candidato, true
		}
		pai := filepath.Dir(dir)
		if pai == dir {
			break
		}
		dir = pai
	}
	return "", false
}
