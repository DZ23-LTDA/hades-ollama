package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// P0-B / T2 — BENCHMARK RED → GREEN: TESTES (E-343).
//
// O núcleo é DETERMINÍSTICO (não depende de modelo): prova que o fixture está **vermelho** com o bug e **verde** com a
// correção de referência. Sem isso, um benchmark pode parecer funcionar e não medir nada.
//
// A parte que usa modelo **pula com motivo declarado** quando o Ollama não responde — *nunca vira "ok" por ausência*.

func TestFixtureT2EstaVermelhoComBugEVerdeComCorrecaoDeReferencia(t *testing.T) {
	tarefa := FixtureT2()
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()

	// 1. COM o bug: precisa ficar VERMELHO.
	dirBug, err := os.MkdirTemp("", "t2-bug-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirBug)
	if err := WriteFixtureT2(dirBug, tarefa.BuggyFiles); err != nil {
		t.Fatal(err)
	}
	codigoVermelho, saidaVermelha := RunFixtureTests(ctx, dirBug)
	if codigoVermelho == 0 {
		t.Fatalf("ARMADILHA: o fixture com bug passou — o benchmark mediria nada. Saída:\n%s", saidaVermelha)
	}
	if codigoVermelho < 0 {
		t.Skipf("toolchain Go indisponível para rodar o fixture: %s", saidaVermelha)
	}

	// 2. COM a correção de REFERÊNCIA: precisa ficar VERDE — prova que a tarefa é solucionável.
	dirCorreto, err := os.MkdirTemp("", "t2-ok-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirCorreto)
	if err := WriteFixtureT2(dirCorreto, tarefa.ReferenceFix); err != nil {
		t.Fatal(err)
	}
	if codigoVerde, saidaVerde := RunFixtureTests(ctx, dirCorreto); codigoVerde != 0 {
		t.Fatalf("a correção de referência precisa deixar o fixture VERDE (benchmark insolúvel). Saída:\n%s", saidaVerde)
	}
}

func TestHunkContaAlteracoesPorBloco(t *testing.T) {
	antes := map[string]string{"a.go": "linha1\nlinha2\nlinha3\n"}
	// Um bloco contíguo de 1 linha alterada.
	depois := map[string]string{"a.go": "linha1\nALTERADA\nlinha3\n"}
	hunks, linhas, arquivos := ContarHunksDoDiff(antes, depois)
	if hunks != 1 || linhas != 1 || len(arquivos) != 1 {
		t.Fatalf("um bloco de uma linha: hunks=%d linhas=%d arquivos=%v", hunks, linhas, arquivos)
	}
	// Dois blocos separados por linha igual ⇒ dois hunks.
	depois2 := map[string]string{"a.go": "MUDOU1\nlinha2\nMUDOU3\n"}
	if hunks2, linhas2, _ := ContarHunksDoDiff(antes, depois2); hunks2 != 2 || linhas2 != 2 {
		t.Fatalf("dois blocos separados: hunks=%d linhas=%d", hunks2, linhas2)
	}
	// Sem alteração ⇒ nada.
	if hunks3, linhas3, arquivos3 := ContarHunksDoDiff(antes, antes); hunks3 != 0 || linhas3 != 0 || len(arquivos3) != 0 {
		t.Fatalf("sem alteração não pode contar nada: %d/%d/%v", hunks3, linhas3, arquivos3)
	}
	// Arquivo NOVO conta como alteração.
	if hunks4, _, arquivos4 := ContarHunksDoDiff(antes, map[string]string{"a.go": "linha1\nlinha2\nlinha3\n", "b.go": "novo\n"}); hunks4 != 1 || len(arquivos4) != 1 {
		t.Fatalf("arquivo novo precisa contar: hunks=%d arquivos=%v", hunks4, arquivos4)
	}
}

// harnessTrapaca tenta "consertar" o arquivo de teste — atalho que o benchmark precisa RECUSAR.
type harnessTrapaca struct{}

func (harnessTrapaca) Name() string  { return "trapaca" }
func (harnessTrapaca) Model() string { return "nenhum" }
func (harnessTrapaca) ProposeFix(context.Context, HarnessBenchmarkTask, string) (map[string]string, error) {
	return map[string]string{"calc_test.go": "package calc\n\nimport \"testing\"\n\nfunc TestSumIncluiTodosOsValores(t *testing.T) {}\n"}, nil
}

func TestHarnessNaoPodeTrapacearAlterandoOTeste(t *testing.T) {
	tarefa := FixtureT2()
	resultado, err := RunHarnessBenchmark(context.Background(), harnessTrapaca{}, tarefa, 1)
	if err != nil {
		t.Fatalf("trapaça e' medição (veredito), não erro de execução: %v", err)
	}
	if resultado.Verdict != "HARNESS_TRAFICOU" {
		t.Fatalf("o veredito precisa registrar a trapaça: %+v", resultado)
	}
	// `-1` = NÃO EXECUTADO. Zero significaria `go test` aprovado, e aqui o verde nunca foi alcançado.
	if resultado.GreenExit == 0 {
		t.Fatal("o harness trapaceiro NÃO pode terminar verde: o teste não é o alvo da correção")
	}
	if resultado.GreenExit != -1 {
		t.Fatalf("sem correção legítima o verde não pode ter sido executado (esperado -1): %d", resultado.GreenExit)
	}
	if resultado.Attempts != 1 {
		t.Fatalf("a única tentativa precisa ser contada: %d", resultado.Attempts)
	}
}

// harnessFixo devolve a correção de referência — prova que o caminho completo (vermelho → patch → verde) funciona.
type harnessFixo struct{}

func (harnessFixo) Name() string  { return "fixo-referencia" }
func (harnessFixo) Model() string { return "nenhum" }
func (harnessFixo) ProposeFix(_ context.Context, task HarnessBenchmarkTask, _ string) (map[string]string, error) {
	proposta := map[string]string{}
	for nome, conteudo := range task.ReferenceFix {
		if nome == "calc.go" {
			proposta[nome] = conteudo
		}
	}
	return proposta, nil
}

func TestBenchmarkCompletaVermelhoParaVerde(t *testing.T) {
	tarefa := FixtureT2()
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	resultado, err := RunHarnessBenchmark(ctx, harnessFixo{}, tarefa, 2)
	if err != nil {
		t.Fatalf("caminho completo com correção correta: %v", err)
	}
	if resultado.Verdict != "GREEN" || resultado.GreenExit != 0 {
		t.Fatalf("precisa terminar GREEN: %+v", resultado)
	}
	if resultado.RedExit == 0 {
		t.Fatal("o resultado precisa registrar que o fixture começou VERMELHO")
	}
	// O diff por hunk precisa estar medido, não presumido.
	if resultado.DiffHunks != 1 || resultado.DiffLines == 0 || len(resultado.PatchFiles) != 1 {
		t.Fatalf("diff por hunk precisa ser medido: %+v", resultado)
	}
	if resultado.Attempts != 1 {
		t.Fatalf("a correção veio na primeira tentativa: %+v", resultado)
	}
	if resultado.DurationSeconds <= 0 {
		t.Fatal("a duração precisa ser medida")
	}
}

func TestExtratorDeArquivosRespeitaDelimitadores(t *testing.T) {
	texto := "bla\n<<<ARQUIVO calc.go>>>\npackage calc\n<<<FIM>>>\nresto"
	arquivos, err := ExtrairArquivosDaResposta(texto)
	if err != nil {
		t.Fatal(err)
	}
	if arquivos["calc.go"] != "package calc\n" {
		t.Fatalf("extração incorreta: %q", arquivos["calc.go"])
	}
	// Sem delimitadores ⇒ vazio (e o benchmark registrará RED, não sucesso).
	vazio, err := ExtrairArquivosDaResposta("desculpe, não sei")
	if err != nil {
		t.Fatal(err)
	}
	if len(vazio) != 0 {
		t.Fatalf("resposta sem delimitadores não pode virar patch: %+v", vazio)
	}
}

// MEDIÇÃO COM MODELO LOCAL: roda quando o Ollama responde; caso contrário, PULA com motivo declarado.
func TestBenchmarkComModeloLocalOllama(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancelar()
	if !OllamaDisponivel(ctx, "") {
		t.Skip("Ollama local não respondeu em /api/tags: benchmark com modelo = NAO_MEDIDO (nunca 'ok' por ausência)")
	}
	modelo := os.Getenv("HADES_T2_MODEL")
	if modelo == "" {
		modelo = "qwen2.5:7b"
	}
	harness := OllamaHarness{Modelo: modelo}
	resultado, err := RunHarnessBenchmark(ctx, harness, FixtureT2(), 2)
	if err != nil {
		t.Fatalf("execução do benchmark com modelo local: %v", err)
	}
	// O resultado é REGISTRADO, seja qual for: verde prova capacidade; vermelho é resultado honesto, não erro de teste.
	t.Logf("benchmark T2: harness=%s modelo=%s red=%d green=%d tentativas=%d hunks=%d linhas=%d duracao=%.1fs veredito=%s motivo=%s",
		resultado.Harness, resultado.Model, resultado.RedExit, resultado.GreenExit, resultado.Attempts,
		resultado.DiffHunks, resultado.DiffLines, resultado.DurationSeconds, resultado.Verdict, resultado.Reason)
	// O veredito é REGISTRADO, inclusive HARNESS_TRAFICOU: alterar o teste é resultado medido, não erro de teste.
	// *Um instrumento que falha quando o resultado é ruim não é instrumento — é opinião.*
	if resultado.Model == "" {
		t.Fatal("o modelo medido precisa aparecer no resultado")
	}
	_ = filepath.Join
}
