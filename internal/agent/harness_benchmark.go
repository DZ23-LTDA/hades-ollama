package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P0-B / T2 — BENCHMARK DE TAREFA EQUIVALENTE: RED → GREEN EM REPO FIXTURE (E-343).
//
// **MEDIDO ANTES DE ESCREVER**: o repositório já tem `benchmark_suite.go`, mas ele mede **jornadas de negócio
// internas** (`BenchmarkTaskResult` do próprio sistema) — **não** mede a capacidade do harness de **consertar um bug
// real** num repositório. E a máquina TEM Ollama local no ar (`/api/tags` HTTP 200) com `qwen2.5:7b` (ferramentas
// nativas) — então uma medição **honesta** é possível **sem** depender de login externo.
//
// O QUE ESTE INSTRUMENTO FAZ, PASSO A PASSO (e o que ele NÃO faz):
//
//  1. Escreve um **fixture** (módulo Go pequeno) com um bug **real** e um teste que **falha**.
//  2. Prova que o fixture está **VERMELHO** (`go test` ⇒ saída ≠ 0). *Fixture que já nasce verde não mede nada.*
//  3. Pede a correção ao harness (modelo local, via Ollama) com **observação mínima**: o teste que falha, a saída do
//     erro e os arquivos do fixture.
//  4. Aplica a resposta, roda de novo e exige **VERDE** (`go test` ⇒ saída 0).
//  5. Registra: harness, modelo, tentativas, `red_exit`, `green_exit`, **diff por hunk** e duração.
//
// LIMITES DECLARADOS (nada de "paridade" por decreto): isto mede **um** fixture pequeno, com **um** harness caseiro de
// laço mínimo sobre o Ollama **local** — *não* mede o agente completo do Hades nem Claude Code/Codex/Manus, que
// dependem de credencial e ficam `BLOCKED_BY_EXTERNAL_DEPENDENCY`. Resultado de harness externo = `NAO_MEDIDO` até
// existir login, e `NAO_MEDIDO` **não** é "ok".

var (
	// ErrBenchmarkInvalid recusa benchmark malformado (fixture verde, patch vazio).
	ErrBenchmarkInvalid = errors.New("harness benchmark is invalid")
	// ErrBenchmarkModelUnavailable declara a ausência do modelo em vez de inventar resultado.
	ErrBenchmarkModelUnavailable = errors.New("harness benchmark model is unavailable")
)

// HarnessBenchmarkTask descreve o fixture e a correção de REFERÊNCIA.
type HarnessBenchmarkTask struct {
	Name string
	// Files são os arquivos SEM o bug plantado (usados para provar que a tarefa é solucionável).
	Files map[string]string
	// BuggyFiles são os arquivos COM o bug.
	BuggyFiles map[string]string
	// ReferenceFix é o conteúdo correto — só para prova de que o benchmark é solucionável sem modelo.
	ReferenceFix map[string]string
}

// HarnessBenchmarkResult é o resultado MEDIDO.
type HarnessBenchmarkResult struct {
	Task            string   `json:"task"`
	Harness         string   `json:"harness"`
	Model           string   `json:"model,omitempty"`
	RedExit         int      `json:"red_exit"`
	GreenExit       int      `json:"green_exit"`
	Attempts        int      `json:"attempts"`
	PatchFiles      []string `json:"patch_files,omitempty"`
	DiffHunks       int      `json:"diff_hunks"`
	DiffLines       int      `json:"diff_lines"`
	DurationSeconds float64  `json:"duration_seconds"`
	Verdict         string   `json:"verdict"`
	Reason          string   `json:"reason,omitempty"`
}

// HarnessProposer é a fronteira com o harness (modelo local hoje; outro amanhã).
type HarnessProposer interface {
	Name() string
	Model() string
	ProposeFix(ctx context.Context, task HarnessBenchmarkTask, observacao string) (map[string]string, error)
}

// FixtureT2 devolve a tarefa de referência: um bug de soma em `calc.Sum` (laço começa em 1 e ignora o primeiro item).
func FixtureT2() HarnessBenchmarkTask {
	correto := `package calc

// Sum devolve a soma de todos os valores.
func Sum(valores []int) int {
	total := 0
	for _, valor := range valores {
		total += valor
	}
	return total
}
`
	comBug := `package calc

// Sum devolve a soma de todos os valores.
func Sum(valores []int) int {
	total := 0
	for i := 1; i < len(valores); i++ {
		total += valores[i]
	}
	return total
}
`
	teste := `package calc

import "testing"

func TestSumIncluiTodosOsValores(t *testing.T) {
	if got := Sum([]int{1, 2, 3}); got != 6 {
		t.Fatalf("Sum([1 2 3]) = %d, queria 6", got)
	}
	if got := Sum([]int{5}); got != 5 {
		t.Fatalf("Sum([5]) = %d, queria 5", got)
	}
	if got := Sum(nil); got != 0 {
		t.Fatalf("Sum(nil) = %d, queria 0", got)
	}
}
`
	modulo := "module fixturet2\n\ngo 1.21\n"
	base := map[string]string{"go.mod": modulo, "calc_test.go": teste}
	comBugCompleto := map[string]string{"go.mod": modulo, "calc.go": comBug, "calc_test.go": teste}
	corretoCompleto := map[string]string{"go.mod": modulo, "calc.go": correto, "calc_test.go": teste}
	_ = base
	return HarnessBenchmarkTask{Name: "fixture-t2-soma", Files: corretoCompleto, BuggyFiles: comBugCompleto,
		ReferenceFix: corretoCompleto}
}

// WriteFixtureT2 escreve os arquivos do fixture num diretório limpo.
func WriteFixtureT2(dir string, arquivos map[string]string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	nomes := make([]string, 0, len(arquivos))
	for nome := range arquivos {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		caminho := filepath.Join(dir, nome)
		if strings.Contains(nome, "..") {
			return fmt.Errorf("%w: nome de fixture inseguro %q", ErrBenchmarkInvalid, nome)
		}
		if err := os.WriteFile(caminho, []byte(arquivos[nome]), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// RunFixtureTests roda `go test ./...` no fixture e devolve o código de saída e a saída combinada.
func RunFixtureTests(ctx context.Context, dir string) (int, string) {
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = dir
	// Ambiente do próprio toolchain, sem depender de módulos externos nem de rede.
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOPROXY=off")
	var saida bytes.Buffer
	cmd.Stdout = &saida
	cmd.Stderr = &saida
	err := cmd.Run()
	if err == nil {
		return 0, saida.String()
	}
	var saidaErr *exec.ExitError
	if errors.As(err, &saidaErr) {
		return saidaErr.ExitCode(), saida.String()
	}
	return -1, saida.String() + "\n" + err.Error()
}

// ContarHunksDoDiff conta hunks e linhas alteradas comparando dois conjuntos de arquivos.
//
// Feito em Go, sem depender de `git` no ambiente do teste: é uma contagem **declarada** de blocos alterados, não um
// diff unificado completo — e o registro diz isso.
func ContarHunksDoDiff(antes, depois map[string]string) (hunks, linhas int, arquivos []string) {
	nomes := make([]string, 0, len(depois))
	for nome := range depois {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		anterior, tinha := antes[nome]
		atual := depois[nome]
		if tinha && anterior == atual {
			continue
		}
		arquivos = append(arquivos, nome)
		// Um "hunk" = bloco contíguo de linhas alteradas.
		linhasAntes := strings.Split(anterior, "\n")
		linhasDepois := strings.Split(atual, "\n")
		maximo := len(linhasAntes)
		if len(linhasDepois) > maximo {
			maximo = len(linhasDepois)
		}
		emBloco := false
		for i := 0; i < maximo; i++ {
			var a, b string
			if i < len(linhasAntes) {
				a = linhasAntes[i]
			}
			if i < len(linhasDepois) {
				b = linhasDepois[i]
			}
			if a != b {
				linhas++
				if !emBloco {
					hunks++
					emBloco = true
				}
				continue
			}
			emBloco = false
		}
	}
	return hunks, linhas, arquivos
}

// RunHarnessBenchmark executa a medição completa: vermelho obrigatório, correção do harness, verde obrigatório.
func RunHarnessBenchmark(ctx context.Context, harness HarnessProposer, task HarnessBenchmarkTask, tentativas int) (HarnessBenchmarkResult, error) {
	inicio := time.Now()
	// GreenExit comeca em -1 (NAO EXECUTADO). Zero é do `go test` que PASSOU — deixar o valor zero significar "passou"
	// faria um teste que nunca rodou parecer aprovado. ***Marcador ambíguo é como a categoria "ok" que este projeto já
	// proibiu: ausência de medição não é aprovação.***
	resultado := HarnessBenchmarkResult{Task: task.Name, Harness: harness.Name(), Model: harness.Model(), GreenExit: -1}
	if tentativas <= 0 {
		tentativas = 1
	}
	dir, err := os.MkdirTemp("", "hades-t2-")
	if err != nil {
		return resultado, err
	}
	defer os.RemoveAll(dir)

	if err := WriteFixtureT2(dir, task.BuggyFiles); err != nil {
		return resultado, err
	}
	// 1. O fixture PRECISA estar vermelho: se já nasce verde, a medição não prova nada.
	codigoVermelho, saidaVermelha := RunFixtureTests(ctx, dir)
	resultado.RedExit = codigoVermelho
	if codigoVermelho == 0 {
		resultado.Verdict = "FIXTURE_INVALIDO"
		resultado.Reason = "o fixture já passa sem correção: não há o que medir"
		return resultado, fmt.Errorf("%w: %s", ErrBenchmarkInvalid, resultado.Reason)
	}

	observacao := "Testes que falham:\n" + saidaVermelha + "\n\nArquivos do projeto:\n"
	nomes := make([]string, 0, len(task.BuggyFiles))
	for nome := range task.BuggyFiles {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		if strings.HasSuffix(nome, "_test.go") {
			continue // o teste é o enunciado, não o alvo da correção
		}
		observacao += "\n=== " + nome + " ===\n" + task.BuggyFiles[nome]
	}

	// 2. O harness propõe; aplicamos e medimos. Tentativas são CONTADAS, nunca escondidas.
	var proposta map[string]string
	for tentativa := 1; tentativa <= tentativas; tentativa++ {
		resultado.Attempts = tentativa
		// MEDIDO (E-344): na matriz, `llama3.2:3b` gastou **1202 s** num único fixture — 2 tentativas contra o timeout
		// de 10 min do cliente HTTP. Um limite POR TENTATIVA é o que impede que um modelo travado consuma a janela
		// inteira: ***tempo de espera não declarado é custo invisível***.
		ctxTentativa, cancelarTentativa := context.WithTimeout(ctx, limitesDeTentativa())
		proposta, err = harness.ProposeFix(ctxTentativa, task, observacao)
		excedeuTempo := errors.Is(err, context.DeadlineExceeded) || errors.Is(ctxTentativa.Err(), context.DeadlineExceeded)
		cancelarTentativa()
		if excedeuTempo {
			resultado.Verdict = "TIMEOUT"
			resultado.Reason = fmt.Sprintf("tentativa %d excedeu o limite de %s", tentativa, limitesDeTentativa())
			continue
		}
		if err != nil {
			resultado.Reason = err.Error()
			continue
		}
		if len(proposta) == 0 {
			resultado.Reason = "harness não propôs nenhuma alteração"
			continue
		}
		// O teste do fixture NUNCA é alvo: harness que "conserta" o teste está trapaceando.
		//
		// E-343: trapaça é MEDIÇÃO, não erro de execução (mesma filosofia do vermelho). Ela é REGISTRADA no veredito e o
		// laço continua, com feedback explícito na tentativa seguinte — *um modelo que altera o teste precisa ter a
		// chance de fazer o certo depois de ser avisado; esconder a primeira tentativa falsearia o resultado*.
		trapacou := ""
		for nome := range proposta {
			if strings.HasSuffix(nome, "_test.go") {
				trapacou = nome
				break
			}
		}
		if trapacou != "" {
			resultado.Verdict = "HARNESS_TRAFICOU"
			resultado.Reason = "o harness tentou alterar o arquivo de teste: " + trapacou
			proposta = nil
			// feedback para a próxima tentativa
			observacao += "\n\nATENÇÃO: a tentativa anterior foi REJEITADA porque alterou " + trapacou +
				", que é o enunciado do problema. Corrija SOMENTE o arquivo calc.go e NÃO inclua o arquivo de teste."
			continue
		}
		if err := WriteFixtureT2(dir, proposta); err != nil {
			resultado.Reason = err.Error()
			continue
		}
		codigoVerde, saidaVerde := RunFixtureTests(ctx, dir)
		resultado.GreenExit = codigoVerde
		if codigoVerde == 0 {
			hunks, linhas, arquivos := ContarHunksDoDiff(task.BuggyFiles, proposta)
			resultado.DiffHunks, resultado.DiffLines, resultado.PatchFiles = hunks, linhas, arquivos
			resultado.DurationSeconds = time.Since(inicio).Seconds()
			resultado.Verdict = "GREEN"
			return resultado, nil
		}
		resultado.Reason = "tentativa " + fmt.Sprint(tentativa) + " não ficou verde: " + saidaVerde
	}
	resultado.DurationSeconds = time.Since(inicio).Seconds()
	if resultado.Verdict != "HARNESS_TRAFICOU" && resultado.Verdict != "TIMEOUT" {
		resultado.Verdict = "RED"
	}
	return resultado, nil
}

// OllamaHarness é o harness MÍNIMO sobre o Ollama LOCAL (sem login externo).
//
// Declarado exatamente como é: laço curto, uma chamada por tentativa, sem ferramentas nativas — *mede "um modelo local
// com um laço mínimo", não "o agente Hades completo"*.
type OllamaHarness struct {
	BaseURL string
	Modelo  string
	Client  *http.Client
}

// Name identifica o harness no resultado.
func (h OllamaHarness) Name() string { return "ollama-local-minimo" }

// Model devolve o modelo medido.
func (h OllamaHarness) Model() string { return h.Modelo }

// ProposeFix pede a correção e extrai os blocos de arquivo da resposta.
func (h OllamaHarness) ProposeFix(ctx context.Context, task HarnessBenchmarkTask, observacao string) (map[string]string, error) {
	base := strings.TrimSpace(h.BaseURL)
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	modelo := strings.TrimSpace(h.Modelo)
	if modelo == "" {
		return nil, fmt.Errorf("%w: modelo não informado", ErrBenchmarkModelUnavailable)
	}
	cliente := h.Client
	if cliente == nil {
		cliente = &http.Client{Timeout: 3 * time.Minute}
	}

	prompt := "Você vai corrigir um bug em Go. Responda SOMENTE com o conteúdo COMPLETO do arquivo corrigido, " +
		"dentro de um bloco delimitado por <<<ARQUIVO nome>>> e <<<FIM>>>. Não altere o arquivo de teste.\n\n" + observacao
	// E-349: UMA implementação da chamada ao modelo. Duas cópias divergiriam — e a divergência apareceria como
	// diferença de resultado entre o benchmark e o especialista, não como bug de código.
	resposta, err := OllamaGenerate(ctx, base, modelo, prompt, cliente)
	if err != nil {
		return nil, err
	}
	return ExtrairArquivosDaResposta(resposta)

}

// ExtrairArquivosDaResposta lê os blocos <<<ARQUIVO nome>>> ... <<<FIM>>>.
//
// Se o modelo não usar os delimitadores, devolve vazio — *e o benchmark registra RED em vez de fingir sucesso*.
func ExtrairArquivosDaResposta(texto string) (map[string]string, error) {
	arquivos := map[string]string{}
	restante := texto
	for {
		inicio := strings.Index(restante, "<<<ARQUIVO ")
		if inicio < 0 {
			break
		}
		fimCabecalho := strings.Index(restante[inicio:], ">>>")
		if fimCabecalho < 0 {
			break
		}
		nome := strings.TrimSpace(restante[inicio+len("<<<ARQUIVO ") : inicio+fimCabecalho])
		corpo := restante[inicio+fimCabecalho+3:]
		fim := strings.Index(corpo, "<<<FIM>>>")
		if fim < 0 {
			break
		}
		conteudo := strings.TrimPrefix(corpo[:fim], "\n")
		if !strings.HasSuffix(conteudo, "\n") {
			conteudo += "\n"
		}
		arquivos[nome] = conteudo
		restante = corpo[fim+len("<<<FIM>>>"):]
	}
	return arquivos, nil
}

// OllamaDisponivel mede se o Ollama local responde (sem presumir).
func OllamaDisponivel(ctx context.Context, baseURL string) bool {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	cliente := &http.Client{Timeout: 3 * time.Second}
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/tags", nil)
	if err != nil {
		return false
	}
	resposta, err := cliente.Do(requisicao)
	if err != nil {
		return false
	}
	defer resposta.Body.Close()
	return resposta.StatusCode == http.StatusOK
}

// limitesDeTentativa é o tempo máximo de UMA tentativa de correção.
//
// Declarado e ajustável por ambiente (`HADES_T2_ATTEMPT_TIMEOUT`, em segundos) para que a medição seja reproduzível
// em máquinas diferentes — *limite escondido em código não é limite, é surpresa*.
func limitesDeTentativa() time.Duration {
	if bruto := strings.TrimSpace(os.Getenv("HADES_T2_ATTEMPT_TIMEOUT")); bruto != "" {
		if segundos, err := time.ParseDuration(bruto + "s"); err == nil && segundos > 0 {
			return segundos
		}
	}
	return 4 * time.Minute
}
