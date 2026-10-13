package agent

import (
	"context"
	"os"
	"testing"
	"time"
)

// P0-B / T2 — MATRIZ: TESTES (E-344).

func TestTodosOsFixturesSaoVermelhosComBugEVerdesComReferencia(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancelar()
	fixtures := AllT2Fixtures()
	if len(fixtures) < 4 {
		t.Fatalf("a matriz precisa de mais de uma classe de bug: %d", len(fixtures))
	}
	for _, tarefa := range fixtures {
		tarefa := tarefa
		t.Run(tarefa.Name, func(t *testing.T) {
			// COM o bug ⇒ VERMELHO (senão o fixture não mede nada).
			dirBug := t.TempDir()
			if err := WriteFixtureT2(dirBug, tarefa.BuggyFiles); err != nil {
				t.Fatal(err)
			}
			codigo, saida := RunFixtureTests(ctx, dirBug)
			if codigo < 0 {
				t.Skipf("toolchain Go indisponível: %s", saida)
			}
			if codigo == 0 {
				t.Fatalf("ARMADILHA: fixture %q já passa com o bug — não mede nada", tarefa.Name)
			}
			// COM a correção de referência ⇒ VERDE (a tarefa é solucionável).
			dirOk := t.TempDir()
			if err := WriteFixtureT2(dirOk, tarefa.ReferenceFix); err != nil {
				t.Fatal(err)
			}
			if codigoOk, saidaOk := RunFixtureTests(ctx, dirOk); codigoOk != 0 {
				t.Fatalf("a correção de referência de %q não fica verde (benchmark insolúvel):\n%s", tarefa.Name, saidaOk)
			}
		})
	}
}

func TestMatrizComCorrecaoDeReferenciaDaVerdeEmTodas(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancelar()
	resultados := RunT2Matrix(ctx, []HarnessProposer{harnessFixo{}}, AllT2Fixtures(), 2)
	if len(resultados) != 4 {
		t.Fatalf("a matriz precisa cobrir os 4 fixtures: %d", len(resultados))
	}
	resumo := SummarizeT2Matrix(resultados)
	if resumo.Green != resumo.Total {
		t.Fatalf("com a referência, todas precisam ficar verdes: %+v", resumo)
	}
	if resumo.GreenPrimeira != resumo.Total {
		t.Fatalf("a referência resolve na primeira tentativa: %+v", resumo)
	}
	if resumo.TaxaSucesso != "4/4" {
		t.Fatalf("taxa de sucesso medida: %q", resumo.TaxaSucesso)
	}
	// O diff precisa estar medido em TODOS, não presumido.
	for _, resultado := range resultados {
		if resultado.DiffHunks == 0 || len(resultado.PatchFiles) == 0 {
			t.Fatalf("diff não medido em %q: %+v", resultado.Task, resultado)
		}
		if resultado.RedExit == 0 {
			t.Fatalf("o vermelho inicial precisa constar em %q", resultado.Task)
		}
	}
}

func TestResumoDaMatrizNaoContaNaoMedidoComoSucesso(t *testing.T) {
	resultados := []HarnessBenchmarkResult{
		{Task: "a", Verdict: "GREEN", Attempts: 1, Model: "m1"},
		{Task: "b", Verdict: "GREEN", Attempts: 2, Model: "m1"},
		{Task: "c", Verdict: "HARNESS_TRAFICOU", Attempts: 2, Model: "m2"},
		{Task: "d", Verdict: "RED", Attempts: 2, Model: "m2"},
		{Task: "e", Verdict: "NAO_MEDIDO", GreenExit: -1},
	}
	resumo := SummarizeT2Matrix(resultados)
	if resumo.Total != 5 || resumo.Green != 2 {
		t.Fatalf("resumo: %+v", resumo)
	}
	// ARMADILHA: NAO_MEDIDO não pode entrar como verde nem como vermelho — é ausência.
	if resumo.Vermelho != 1 {
		t.Fatalf("NAO_MEDIDO não é vermelho medido: %+v", resumo)
	}
	if resumo.Trapacou != 1 {
		t.Fatalf("trapaça precisa ser contada à parte: %+v", resumo)
	}
	if resumo.GreenPrimeira != 1 {
		t.Fatalf("só uma resolução na primeira tentativa: %+v", resumo)
	}
	if resumo.TaxaSucesso != "2/5" {
		t.Fatalf("taxa de sucesso: %q", resumo.TaxaSucesso)
	}
}

// harnessTravado nunca responde; serve para provar que contexto cancelado vira NAO_MEDIDO, não sucesso nem erro.
type harnessTravado struct{}

func (harnessTravado) Name() string  { return "travado" }
func (harnessTravado) Model() string { return "nenhum" }
func (harnessTravado) ProposeFix(ctx context.Context, _ HarnessBenchmarkTask, _ string) (map[string]string, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestMatrizComContextoCanceladoRegistraNaoMedido(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar() // já cancelado: nada pode ser medido
	resultados := RunT2Matrix(ctx, []HarnessProposer{harnessTravado{}}, AllT2Fixtures(), 1)
	if len(resultados) != 4 {
		t.Fatalf("a matriz precisa registrar todas as combinações: %d", len(resultados))
	}
	for _, resultado := range resultados {
		if resultado.Verdict != "NAO_MEDIDO" {
			t.Fatalf("sem execução o veredito é NAO_MEDIDO, nunca outro: %+v", resultado)
		}
		if resultado.GreenExit != -1 {
			t.Fatalf("verde não executado precisa ser -1: %+v", resultado)
		}
	}
	if resumo := SummarizeT2Matrix(resultados); resumo.Green != 0 {
		t.Fatalf("NAO_MEDIDO não pode contar como verde: %+v", resumo)
	}
}

// MATRIZ REAL com modelos locais: roda quando o Ollama responde; caso contrário PULA com motivo declarado.
func TestMatrizRealComModelosLocais(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancelar()
	if !OllamaDisponivel(ctx, "") {
		t.Skip("Ollama local não respondeu: matriz com modelos = NAO_MEDIDO (nunca 'ok' por ausência)")
	}
	modelos := []string{"qwen2.5:7b", "llama3.2:3b"}
	if escolhido := os.Getenv("HADES_T2_MODELS"); escolhido != "" {
		modelos = []string{escolhido}
	}
	harnesses := make([]HarnessProposer, 0, len(modelos))
	for _, modelo := range modelos {
		harnesses = append(harnesses, OllamaHarness{Modelo: modelo})
	}
	resultados := RunT2Matrix(ctx, harnesses, AllT2Fixtures(), 2)
	for _, resultado := range resultados {
		t.Logf("T2 matriz: modelo=%-14s fixture=%-24s red=%d green=%d tentativas=%d hunks=%d linhas=%d dur=%.1fs veredito=%s",
			resultado.Model, resultado.Task, resultado.RedExit, resultado.GreenExit, resultado.Attempts,
			resultado.DiffHunks, resultado.DiffLines, resultado.DurationSeconds, resultado.Verdict)
	}
	resumo := SummarizeT2Matrix(resultados)
	t.Logf("T2 resumo: total=%d verde=%d primeira_tentativa=%d trapaceou=%d vermelho=%d taxa=%s",
		resumo.Total, resumo.Green, resumo.GreenPrimeira, resumo.Trapacou, resumo.Vermelho, resumo.TaxaSucesso)
	// EVIDÊNCIA REPRODUZÍVEL: a matriz medida vira ARTEFATO em disco, com data, modelos e o resultado por combinação.
	// *Número que só existe no log de um terminal não é evidência — é lembrança.*
	EscreverArtefatoDaMatrizT2(t, resultados, resumo)
	// O teste NÃO falha por resultado ruim: a matriz é MEDIÇÃO. Ele falha apenas se a medição for impossível.
	if resumo.Total != len(modelos)*len(AllT2Fixtures()) {
		t.Fatalf("a matriz precisa cobrir todas as combinações: %d", resumo.Total)
	}
}

// harnessLento trava até o contexto vencer — prova que tentativa lenta vira TIMEOUT (e não consome a janela inteira).
type harnessLento struct{}

func (harnessLento) Name() string  { return "lento" }
func (harnessLento) Model() string { return "nenhum" }
func (harnessLento) ProposeFix(ctx context.Context, _ HarnessBenchmarkTask, _ string) (map[string]string, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestTentativaQueExcedeOTempoViraTimeout(t *testing.T) {
	// ACHADO DA MATRIZ (E-344): llama3.2:3b gastou 1202s num fixture. Limite curto aqui para o teste ser rápido, e
	// declarado por ambiente para a medição ser reproduzível.
	t.Setenv("HADES_T2_ATTEMPT_TIMEOUT", "1")
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	resultado, err := RunHarnessBenchmark(ctx, harnessLento{}, FixtureT2(), 2)
	if err != nil {
		t.Fatalf("timeout é MEDIÇÃO (veredito), não erro de execução: %v", err)
	}
	if resultado.Verdict != "TIMEOUT" {
		t.Fatalf("tentativa que excede o limite precisa virar TIMEOUT: %+v", resultado)
	}
	// ARMADILHA: verde NÃO pode ter sido executado (o modelo nunca respondeu).
	if resultado.GreenExit != -1 {
		t.Fatalf("sem resposta não há execução de teste verde (esperado -1): %d", resultado.GreenExit)
	}
	if resultado.Attempts != 2 {
		t.Fatalf("as duas tentativas precisam ser contadas: %d", resultado.Attempts)
	}
	if resultado.Reason == "" {
		t.Fatal("o motivo do timeout precisa ficar registrado")
	}
}
