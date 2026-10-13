package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// P0-E / T4 — LIGAÇÃO AO LAÇO REAL: TESTES (E-348).
//
// O teste usa o **fixture do T2** como carga de trabalho: git de verdade, `go test` de verdade dentro da worktree, e a
// correção escrita pelo fixer. *Se qualquer elo estiver fingido, o teste não fica verde.*

func repoComFixtureT2(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@hades.local"}, {"config", "user.name", "T"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if saida, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git indisponível: %s (%v)", saida, err)
		}
	}
	tarefa := FixtureT2()
	for nome, conteudo := range tarefa.BuggyFiles {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte(conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "base com bug"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if saida, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("não consegui preparar o repositório: %s (%v)", saida, err)
		}
	}
	return dir
}

func TestEspecialistaConsertaNoLacoRealEMerge(t *testing.T) {
	repo := repoComFixtureT2(t)
	tarefa := FixtureT2()
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelar()

	// O fixer escreve a correção DE REFERÊNCIA: o laço roda os testes de verdade dentro da worktree.
	fixer := &fixerDeArquivos{arquivos: map[string]string{"calc.go": tarefa.ReferenceFix["calc.go"]}}
	relatorio, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(),
		[]SpecialistTask{{Name: "calc", Segment: []string{"calc.go"}, Instruction: "consertar a soma"}},
		fixer, DefaultSpecialistLoopOptions())
	if err != nil {
		t.Fatalf("execução ligada ao laço real: %v", err)
	}
	if len(relatorio.Outcomes) != 1 {
		t.Fatalf("precisa haver um desfecho: %+v", relatorio)
	}
	desfecho := relatorio.Outcomes[0]
	if !desfecho.Passed {
		t.Fatalf("o laço PRECISA ficar verde com a correção de referência (medido: exit=%d, erro=%q, falhas=%v)",
			desfecho.ExitCode, desfecho.Error, desfecho.FailedTest)
	}
	if fixer.chamadas == 0 {
		t.Fatal("o laço não chamou o fixer: a ligação não funciona")
	}
	// O diff precisa vir da MEDIÇÃO existente (GetWorktreeDiff), não de suposição.
	if len(desfecho.FilesDiff) == 0 {
		t.Fatalf("o diff por arquivo precisa ser medido: %+v", desfecho)
	}
	// E o merge precisa ter acontecido na árvore de integração.
	if relatorio.Verdict != "MESCLADO" {
		t.Fatalf("veredito: %+v", relatorio)
	}
	dados, err := os.ReadFile(filepath.Join(repo, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(dados) != tarefa.ReferenceFix["calc.go"] {
		t.Fatalf("o arquivo integrado precisa ser o corrigido:\n%s", dados)
	}
}

func TestEspecialistasConflitantesNoLacoRealSaoRecusados(t *testing.T) {
	repo := repoComFixtureT2(t)
	tarefa := FixtureT2()
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelar()

	// DOIS especialistas no MESMO arquivo: cada um conserta na sua worktree, e a coordenação recusa ANTES do merge.
	fixer := &fixerDeArquivos{arquivos: map[string]string{"calc.go": tarefa.ReferenceFix["calc.go"]}}
	relatorio, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(),
		[]SpecialistTask{
			{Name: "a", Segment: []string{"calc.go"}},
			{Name: "b", Segment: []string{"calc.go"}},
		}, fixer, DefaultSpecialistLoopOptions())
	if !errors.Is(err, ErrSpecialistConflict) {
		t.Fatalf("dois especialistas no mesmo arquivo precisam ser RECUSADOS: %v", err)
	}
	if relatorio.Verdict != "RECUSADO" {
		t.Fatalf("veredito precisa declarar a recusa: %+v", relatorio)
	}
	// ARMADILHA: recusa significa que NADA foi integrado — o arquivo do repositório continua com o bug.
	dados, err := os.ReadFile(filepath.Join(repo, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(dados) == tarefa.ReferenceFix["calc.go"] {
		t.Fatal("ARMADILHA: o merge aconteceu apesar do conflito — recusar depois de mesclar não é recusar")
	}
}

// fixerQueNaoConserta nunca escreve nada: o laço tem de terminar VERMELHO e ser declarado como tal.
type fixerQueNaoConserta struct{ chamadas int }

func (f *fixerQueNaoConserta) Fix(_ context.Context, _ string, _ SpecialistTask, _ int, _ string) error {
	f.chamadas++
	return nil
}

func TestEspecialistaQueNaoConsertaEhDeclaradoVermelho(t *testing.T) {
	repo := repoComFixtureT2(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelar()
	opcoes := DefaultSpecialistLoopOptions()
	opcoes.MaxRepairTries = 2
	fixer := &fixerQueNaoConserta{}
	relatorio, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(),
		[]SpecialistTask{{Name: "inutil", Segment: []string{"calc.go"}}}, fixer, opcoes)
	if err != nil {
		t.Fatalf("não ficar verde é RESULTADO, não exceção: %v", err)
	}
	if len(relatorio.Outcomes) != 1 || relatorio.Outcomes[0].Passed {
		t.Fatalf("o desfecho precisa declarar que NÃO passou: %+v", relatorio.Outcomes)
	}
	// MEDIDO (E-348): com `maxTries=2`, o laço EXISTENTE chama o fixer **1 vez**. Eu havia suposto que chamaria 2 —
	// *a contagem é semântica do laço, não do meu desejo*. A propriedade que importa é: o fixer é chamado (a ligação
	// funciona) e o laço NÃO ultrapassa o teto declarado.
	if fixer.chamadas < 1 {
		t.Fatal("o laço precisa ter chamado o fixer ao menos uma vez (senão a ligação não existe)")
	}
	if fixer.chamadas > opcoes.MaxRepairTries {
		t.Fatalf("o laço ultrapassou o teto declarado: %d chamadas para maxTries=%d", fixer.chamadas, opcoes.MaxRepairTries)
	}
	t.Logf("laço de reparo medido: %d chamada(s) ao fixer com maxTries=%d", fixer.chamadas, opcoes.MaxRepairTries)
	if relatorio.Verdict != "PARCIAL" {
		t.Fatalf("veredito precisa ser PARCIAL com especialista vermelho: %+v", relatorio)
	}
}

func TestLigacaoRecusaConfiguracaoInvalida(t *testing.T) {
	repo := repoComFixtureT2(t)
	ctx := context.Background()
	if _, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(), nil, &fixerDeArquivos{}, DefaultSpecialistLoopOptions()); !errors.Is(err, ErrSpecialistInvalid) {
		t.Fatalf("sem especialista precisa ser recusado: %v", err)
	}
	if _, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(),
		[]SpecialistTask{{Name: "sem-segmento"}}, &fixerDeArquivos{}, DefaultSpecialistLoopOptions()); !errors.Is(err, ErrSpecialistInvalid) {
		t.Fatalf("especialista sem segmento precisa ser recusado: %v", err)
	}
	if _, err := RunSpecialistRepairLoops(ctx, "", t.TempDir(),
		[]SpecialistTask{{Name: "a", Segment: []string{"a"}}}, &fixerDeArquivos{}, DefaultSpecialistLoopOptions()); !errors.Is(err, ErrSpecialistFixerInvalid) {
		t.Fatalf("repositório vazio precisa ser recusado: %v", err)
	}
}
