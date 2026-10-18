package agent

// FASE 2 do adendo V3: testes do ciclo de vida de worktree com protecao de trabalho.
//
// Os testes que importam sao os NEGATIVOS: os que provam que trabalho NAO e' perdido.
//
//   *Apagar trabalho que ninguem integrou nao e' limpeza: e' perda de dados com nome tecnico.*
//
// Todos correm com **git de verdade** (nada de mock), porque a perda acontece no disco.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoFixture cria um repositorio git real com um commit inicial.
func repoFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponivel neste ambiente")
	}
	dir := t.TempDir()
	ctx := context.Background()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "teste@hades.local"},
		{"config", "user.name", "Teste HADES"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runRawGit(ctx, dir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatalf("escrever base: %v", err)
	}
	if _, err := runRawGit(ctx, dir, "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runRawGit(ctx, dir, "commit", "-q", "-m", "inicial"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return dir
}

// criarWorktree abre um worktree pelo caminho canonico.
func criarWorktree(t *testing.T, repo, dataRoot, missionID string) *GitWorktreeSession {
	t.Helper()
	sessao, err := CreateGitWorktree(context.Background(), repo, dataRoot, missionID, "")
	if err != nil {
		t.Fatalf("CreateGitWorktree: %v", err)
	}
	t.Cleanup(func() {
		// Limpeza do teste: descarta o que restar, com motivo.
		_ = RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})
	return sessao
}

// O DIAGNOSTICO recusa um caminho que nao segue a convencao.
func TestDiagnosticoRecusaCaminhoForaDaConvencao(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		nome   string
		sess   *GitWorktreeSession
		motivo string
	}{
		{"sessao nula", nil, "sessao ou caminho vazio"},
		{"caminho vazio", &GitWorktreeSession{WorktreeDir: ""}, "sessao ou caminho vazio"},
		{
			"caminho sem a convencao",
			&GitWorktreeSession{WorktreeDir: filepath.Join(t.TempDir(), "qualquer-coisa")},
			"nao segue a convencao",
		},
		{
			"raiz do volume",
			&GitWorktreeSession{WorktreeDir: filepath.VolumeName(t.TempDir()) + string(filepath.Separator)},
			"raiz do volume",
		},
	}
	for _, caso := range casos {
		diag := DiagnosticarWorktree(ctx, caso.sess)
		if diag.AlvoSeguro {
			t.Fatalf("%s: alvo aceite; devia ter sido recusado", caso.nome)
		}
		if !strings.Contains(diag.MotivoDoAlvo, caso.motivo) {
			t.Fatalf("%s: motivo = %q; esperado conter %q", caso.nome, diag.MotivoDoAlvo, caso.motivo)
		}
		if diag.PodeRemover() {
			t.Fatalf("%s: PodeRemover verdadeiro com alvo inseguro", caso.nome)
		}
	}
}

// O DIAGNOSTICO recusa um caminho que e' o proprio repositorio.
func TestDiagnosticoRecusaOProprioRepositorio(t *testing.T) {
	repo := repoFixture(t)
	// Construir um path que contenha a convencao mas aponte para o repo.
	falso := filepath.Join(repo, SegmentoDeWorktrees, "x")
	diag := DiagnosticarWorktree(context.Background(), &GitWorktreeSession{
		RepoRoot:    repo,
		WorktreeDir: falso,
	})
	// Este path esta' DENTRO do repo, portanto e' seguro quanto a ancestralidade -- o que se
	// testa aqui e' que o repo em si nao e' aceite como worktree.
	if diag.AlvoSeguro {
		// E' aceitavel: o path contem a convencao e nao e' ancestral do repo.
		t.Logf("path interno aceite (esperado): %s", diag.CaminhoResolvido)
	}
	diagRepo := DiagnosticarWorktree(context.Background(), &GitWorktreeSession{
		RepoRoot:    repo,
		WorktreeDir: repo,
	})
	if diagRepo.AlvoSeguro {
		t.Fatal("o proprio repositorio foi aceite como worktree removivel")
	}
}

// A PROTECAO CENTRAL: worktree com commit nao integrado NAO e' removido.
func TestWorktreeComCommitNaoIntegradoNaoERemovido(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-com-trabalho")

	// Escrever e COMMITAR dentro do worktree -- trabalho que ninguem integrou.
	if err := os.WriteFile(filepath.Join(sessao.WorktreeDir, "novo.txt"), []byte("trabalho\n"), 0o600); err != nil {
		t.Fatalf("escrever no worktree: %v", err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "trabalho do agente"}} {
		if _, err := runRawGit(context.Background(), sessao.WorktreeDir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	diag := DiagnosticarWorktree(context.Background(), sessao)
	if !diag.TemTrabalhoNaoIntegrado {
		t.Fatalf("diagnostico nao detectou trabalho; detalhe=%q", diag.Detalhe)
	}
	if diag.CommitsNaoIntegrados < 1 {
		t.Fatalf("commits nao integrados = %d; esperado >= 1", diag.CommitsNaoIntegrados)
	}

	err := RemoveGitWorktree(context.Background(), sessao)
	if !errors.Is(err, ErrTrabalhoNaoIntegrado) {
		t.Fatalf("erro = %v; esperado ErrTrabalhoNaoIntegrado", err)
	}

	// O worktree TEM de continuar no disco.
	if _, statErr := os.Stat(sessao.WorktreeDir); statErr != nil {
		t.Fatalf("worktree desapareceu apesar do trabalho nao integrado: %v", statErr)
	}
	// E o ficheiro do agente tem de estar la'.
	if _, statErr := os.Stat(filepath.Join(sessao.WorktreeDir, "novo.txt")); statErr != nil {
		t.Fatalf("o ficheiro do agente desapareceu: %v", statErr)
	}
	// O estado NAO pode dizer "removed": *um estado que mente e' pior do que nenhum.*
	if sessao.Status == "removed" {
		t.Fatal("estado diz 'removed' com o worktree ainda no disco")
	}
}

// A PROTECAO vale tambem para ficheiros por commitar.
func TestWorktreeComFicheirosPorCommparseIntNaoERemovido(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-por-commitar")

	if err := os.WriteFile(filepath.Join(sessao.WorktreeDir, "solto.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	diag := DiagnosticarWorktree(context.Background(), sessao)
	if diag.FicheirosModificados < 1 {
		t.Fatalf("ficheiros modificados = %d; esperado >= 1", diag.FicheirosModificados)
	}
	if err := RemoveGitWorktree(context.Background(), sessao); !errors.Is(err, ErrTrabalhoNaoIntegrado) {
		t.Fatalf("erro = %v; esperado ErrTrabalhoNaoIntegrado", err)
	}
	if _, statErr := os.Stat(filepath.Join(sessao.WorktreeDir, "solto.txt")); statErr != nil {
		t.Fatalf("ficheiro por commitar desapareceu: %v", statErr)
	}
}

// Um worktree LIMPO e' removido sem erro -- a guarda nao pode bloquear o caso normal.
//
// *Uma guarda que recusa tudo e' tao inutil como uma que aceita tudo.*
func TestWorktreeLimpoERemovido(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-limpa")

	diag := DiagnosticarWorktree(context.Background(), sessao)
	if !diag.AlvoSeguro {
		t.Fatalf("alvo recusado num worktree legitimo: %s", diag.MotivoDoAlvo)
	}
	if diag.TemTrabalhoNaoIntegrado {
		t.Fatalf("worktree recem-criado com trabalho? %s", diag.Detalhe)
	}

	if err := RemoveGitWorktree(context.Background(), sessao); err != nil {
		t.Fatalf("remocao de worktree limpo falhou: %v", err)
	}
	if _, statErr := os.Stat(sessao.WorktreeDir); !os.IsNotExist(statErr) {
		t.Fatalf("worktree limpo nao foi removido: %v", statErr)
	}
	if sessao.Status != "removed" {
		t.Fatalf("estado = %q; esperado removed", sessao.Status)
	}
	if strings.TrimSpace(sessao.CaminhoRemovido) == "" {
		t.Fatal("CaminhoRemovido vazio depois de remover")
	}
}

// O DESCARTE deliberado exige motivo e deixa rastro.
func TestDescarteDeliberadoExigeMotivo(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-descarte")

	if err := os.WriteFile(filepath.Join(sessao.WorktreeDir, "perder.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	// Sem motivo: recusa.
	if err := RemoveGitWorktreeComPerda(context.Background(), sessao, ""); err == nil {
		t.Fatal("descarte sem motivo foi aceite")
	} else if !strings.Contains(err.Error(), "motivo") {
		t.Fatalf("erro pouco claro: %v", err)
	}
	// E o worktree continua la'.
	if _, statErr := os.Stat(filepath.Join(sessao.WorktreeDir, "perder.txt")); statErr != nil {
		t.Fatalf("ficheiro desapareceu no descarte sem motivo: %v", statErr)
	}

	// Com motivo: remove e regista.
	if err := RemoveGitWorktreeComPerda(context.Background(), sessao, "trabalho obsoleto, revisto pelo proprietario"); err != nil {
		t.Fatalf("descarte com motivo falhou: %v", err)
	}
	if sessao.MotivoDoDescarte == "" {
		t.Fatal("MotivoDoDescarte vazio depois do descarte")
	}
	if sessao.TrabalhoDescartado == "" {
		t.Fatal("TrabalhoDescartado vazio: nao ficou rastro do que se perdeu")
	}
	if _, statErr := os.Stat(sessao.WorktreeDir); !os.IsNotExist(statErr) {
		t.Fatalf("worktree ainda existe depois do descarte: %v", statErr)
	}
}

// O branch NAO e' apagado pela remocao.
//
// *Um branch e' o unico sitio onde o trabalho nao integrado ainda pode estar, e apaga-lo seria
// perder a ultima copia.*
func TestRemocaoNaoApagaOBranch(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-branch")
	branch := sessao.BranchName
	if branch == "" {
		t.Skip("sessao sem nome de branch")
	}

	if err := RemoveGitWorktree(context.Background(), sessao); err != nil {
		t.Fatalf("remocao: %v", err)
	}

	// O branch tem de continuar a existir no repositorio.
	if _, err := runRawGit(context.Background(), repo, "rev-parse", "--verify", branch); err != nil {
		t.Fatalf("branch %q desapareceu depois da remocao: %v", branch, err)
	}
}

// Sessao nula nao rebenta.
func TestRemocaoDeSessaoNulaNaoRebenta(t *testing.T) {
	if err := RemoveGitWorktree(context.Background(), nil); err != nil {
		t.Fatalf("sessao nula devolveu erro: %v", err)
	}
	if err := RemoveGitWorktreeComPerda(context.Background(), nil, "x"); err != nil {
		t.Fatalf("sessao nula no descarte devolveu erro: %v", err)
	}
}

// DiagnosticarERemover devolve o diagnostico mesmo quando recusa.
//
// *Um diagnostico que so' aparece no caso feliz nao serve para decidir o caso dificil.*
func TestDiagnosticarERemoverDevolveDiagnosticoNaRecusa(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-diag")

	if err := os.WriteFile(filepath.Join(sessao.WorktreeDir, "y.txt"), []byte("y\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	diag, err := DiagnosticarERemover(context.Background(), sessao)
	if !errors.Is(err, ErrTrabalhoNaoIntegrado) {
		t.Fatalf("erro = %v; esperado ErrTrabalhoNaoIntegrado", err)
	}
	if !diag.TemTrabalhoNaoIntegrado {
		t.Fatal("diagnostico nao descreve o trabalho que bloqueou a remocao")
	}
	if strings.TrimSpace(diag.Detalhe) == "" {
		t.Fatal("diagnostico sem detalhe legivel")
	}
}
