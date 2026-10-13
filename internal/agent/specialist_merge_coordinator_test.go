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

// P0-E / T4 — COORDENADOR DE MERGE: TESTES (E-346).

func aprovacaoComArquivos(arquivos ...string) *GitMergeApproval {
	return &GitMergeApproval{FilesChanged: arquivos}
}

func TestPlanoRecusaConflitoNoMesmoArquivo(t *testing.T) {
	planos := []SpecialistMergePlan{
		{SessionID: "esp-a", Segment: []string{"x.go", "y.go"}, Approval: aprovacaoComArquivos("x.go")},
		{SessionID: "esp-b", Segment: []string{"x.go"}, Approval: aprovacaoComArquivos("x.go")},
	}
	err := ValidateSpecialistPlan(planos, DefaultSpecialistPlanLimits())
	if !errors.Is(err, ErrSpecialistConflict) {
		t.Fatalf("dois especialistas no mesmo arquivo precisam ser RECUSADOS antes do merge: %v", err)
	}
	if err == nil || !containsAll(err.Error(), "x.go") {
		t.Fatalf("a recusa precisa nomear o arquivo em disputa: %v", err)
	}
}

func TestPlanoRecusaArquivoForaDoSegmento(t *testing.T) {
	planos := []SpecialistMergePlan{
		{SessionID: "esp-a", Segment: []string{"permitido.go"}, Approval: aprovacaoComArquivos("proibido.go")},
	}
	if err := ValidateSpecialistPlan(planos, DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistOutOfSegment) {
		t.Fatalf("alteração fora do segmento precisa ser recusada: %v", err)
	}
}

func TestPlanoRespeitaLimitesDeclarados(t *testing.T) {
	limites := DefaultSpecialistPlanLimits()
	limites.MaxSpecialists = 2
	tres := []SpecialistMergePlan{
		{SessionID: "a", Segment: []string{"a"}, Approval: aprovacaoComArquivos("a")},
		{SessionID: "b", Segment: []string{"b"}, Approval: aprovacaoComArquivos("b")},
		{SessionID: "c", Segment: []string{"c"}, Approval: aprovacaoComArquivos("c")},
	}
	if err := ValidateSpecialistPlan(tres, limites); !errors.Is(err, ErrSpecialistPlanLimit) {
		t.Fatalf("acima do número de especialistas precisa ser recusado: %v", err)
	}
	limites = DefaultSpecialistPlanLimits()
	limites.MaxFilesPerSpecialist = 1
	if err := ValidateSpecialistPlan([]SpecialistMergePlan{
		{SessionID: "a", Segment: []string{"a", "b"}, Approval: aprovacaoComArquivos("a", "b")},
	}, limites); !errors.Is(err, ErrSpecialistPlanLimit) {
		t.Fatalf("acima do número de arquivos precisa ser recusado: %v", err)
	}
}

func TestPlanoInvalidoEhRecusado(t *testing.T) {
	// Sem especialista.
	if err := ValidateSpecialistPlan(nil, DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistPlanInvalid) {
		t.Fatalf("plano vazio precisa ser recusado: %v", err)
	}
	// Sem segmento: especialista sem fronteira não é especialista.
	if err := ValidateSpecialistPlan([]SpecialistMergePlan{{SessionID: "a"}}, DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistPlanInvalid) {
		t.Fatalf("especialista sem segmento precisa ser recusado: %v", err)
	}
	// Sem identificador.
	if err := ValidateSpecialistPlan([]SpecialistMergePlan{{Segment: []string{"a"}}}, DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistPlanInvalid) {
		t.Fatalf("especialista sem identificador precisa ser recusado: %v", err)
	}
	// Segmento com traversal.
	if err := ValidateSpecialistPlan([]SpecialistMergePlan{{SessionID: "a", Segment: []string{"../fuga"}}},
		DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistPlanInvalid) {
		t.Fatalf("segmento com traversal precisa ser recusado: %v", err)
	}
}

func TestPlanoLimpoPassaERecusaAntesDeMesclar(t *testing.T) {
	limpo := []SpecialistMergePlan{
		{SessionID: "esp-a", Segment: []string{"a.go"}, Approval: aprovacaoComArquivos("a.go")},
		{SessionID: "esp-b", Segment: []string{"b.go"}, Approval: aprovacaoComArquivos("b.go")},
	}
	if err := ValidateSpecialistPlan(limpo, DefaultSpecialistPlanLimits()); err != nil {
		t.Fatalf("segmentos disjuntos precisam passar: %v", err)
	}
	// Sem diff medido ainda: passa a validação estrutural (o chamador mede e revalida).
	semDiff := []SpecialistMergePlan{{SessionID: "esp-a", Segment: []string{"a.go"}}}
	if err := ValidateSpecialistPlan(semDiff, DefaultSpecialistPlanLimits()); err != nil {
		t.Fatalf("plano sem diff ainda precisa passar a validação estrutural: %v", err)
	}
	// ARMADILHA: com conflito, NADA é mesclado — o coordenador precisa recusar ANTES de tocar na árvore.
	comConflito := []SpecialistMergePlan{
		{SessionID: "esp-a", Segment: []string{"mesmo.go"}, Approval: aprovacaoComArquivos("mesmo.go")},
		{SessionID: "esp-b", Segment: []string{"mesmo.go"}, Approval: aprovacaoComArquivos("mesmo.go")},
	}
	sessoes := map[string]*GitWorktreeSession{"esp-a": {}, "esp-b": {}}
	if _, err := CoordinateSpecialistMerges(context.Background(), sessoes, comConflito, DefaultSpecialistPlanLimits()); !errors.Is(err, ErrSpecialistConflict) {
		t.Fatalf("o coordenador precisa recusar o plano conflitante antes de mesclar: %v", err)
	}
}

// Caminho REAL com a API EXISTENTE: duas worktrees de verdade, mudanças isoladas, merge coordenado.
func TestCoordenadorMesclaWorktreesReais(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	repo := t.TempDir()
	passos := [][]string{{"init", "-q"}, {"config", "user.email", "t@hades.local"}, {"config", "user.name", "T"}}
	for _, args := range passos {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if saida, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git indisponível: %s (%v)", saida, err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "base"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if saida, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("não consegui preparar o repositório: %s (%v)", saida, err)
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	dados := t.TempDir()

	sessaoA, err := CreateGitWorktree(ctx, repo, dados, "missao-a", "hades/esp-a")
	if err != nil {
		t.Fatalf("worktree A: %v", err)
	}
	sessaoB, err := CreateGitWorktree(ctx, repo, dados, "missao-b", "hades/esp-b")
	if err != nil {
		t.Fatalf("worktree B: %v", err)
	}
	// Cada especialista altera SÓ o seu arquivo, DENTRO da própria worktree.
	for _, par := range []struct {
		dir, arquivo, conteudo string
	}{{sessaoA.WorktreeDir, "a.txt", "de A\n"}, {sessaoB.WorktreeDir, "b.txt", "de B\n"}} {
		if err := os.WriteFile(filepath.Join(par.dir, par.arquivo), []byte(par.conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	diffA, err := GetWorktreeDiff(ctx, sessaoA)
	if err != nil {
		t.Fatalf("diff A: %v", err)
	}
	diffB, err := GetWorktreeDiff(ctx, sessaoB)
	if err != nil {
		t.Fatalf("diff B: %v", err)
	}
	planos := []SpecialistMergePlan{
		{SessionID: "esp-a", Segment: diffA.FilesChanged, Approval: diffA},
		{SessionID: "esp-b", Segment: diffB.FilesChanged, Approval: diffB},
	}
	sessoes := map[string]*GitWorktreeSession{"esp-a": sessaoA, "esp-b": sessaoB}
	resultados, err := CoordinateSpecialistMerges(ctx, sessoes, planos, DefaultSpecialistPlanLimits())
	if err != nil {
		t.Fatalf("plano limpo precisa mesclar: %v", err)
	}
	mesclados := 0
	for _, resultado := range resultados {
		if resultado.Merged {
			mesclados++
		}
	}
	if mesclados != 2 {
		t.Fatalf("os dois especialistas precisam ser mesclados: %+v", resultados)
	}
	for _, arquivo := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(repo, arquivo)); err != nil {
			t.Fatalf("arquivo %s precisa estar na árvore de integração: %v", arquivo, err)
		}
	}
}

func containsAll(texto, alvo string) bool {
	for i := 0; i+len(alvo) <= len(texto); i++ {
		if texto[i:i+len(alvo)] == alvo {
			return true
		}
	}
	return false
}
