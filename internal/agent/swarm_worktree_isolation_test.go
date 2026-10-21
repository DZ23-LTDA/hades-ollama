package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// P0-E / T4 (V2.2 + adendo de paridade): TRÊS ESPECIALISTAS EM WORKTREES ISOLADOS COM MERGE CONTROLADO (E-320).
//
// O adendo exige, textualmente: "três especialistas em worktrees isolados e merge controlado de resultados".
// Este teste usa **git de verdade** (nada de mock): cria um repositório fixture, abre **três worktrees** por missão,
// roda o AgentOrchestrator com três papéis — cada runner gravando seu artefato **dentro do seu próprio worktree** —
// e depois faz o merge **controlado** dos três branches.
//
// As três propriedades medidas aqui:
//  1. ISOLAMENTO REAL: o que um especialista escreve NÃO aparece no worktree do outro nem no repositório base — se
//     isso falhar, três agentes "isolados" estariam pisando no mesmo diretório (o pior desfecho de paralelismo).
//  2. MERGE CONTROLEADO: os três artefatos combinam no branch base quando não há conflito.
//  3. CONFLITO NÃO É RESOLVIDO SOZINHO: dois especialistas mexendo na MESMA linha produzem conflito detectável, e o
//     merge é abortado — *um merge automático silencioso escolheria a versão de alguém sem ninguém decidir*.

func TestTresEspecialistasEmWorktreesIsoladosComMergeControlado(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível neste ambiente")
	}
	ctx := context.Background()
	raiz := t.TempDir()
	repo := filepath.Join(raiz, "repo")
	dataRoot := filepath.Join(raiz, "dados")

	// 1. Repositório fixture de verdade.
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitFixture(t, ctx, repo, "init", "-q", "-b", "main")
	escrever(t, filepath.Join(repo, "README.md"), "base\n")
	gitFixture(t, ctx, repo, "add", "-A")
	gitFixture(t, ctx, repo, "commit", "-q", "-m", "base")

	// 2. TRÊS worktrees isolados, um por missão (a API real do Hades, não uma imitação).
	papeis := []AgentRole{RoleProgram, RoleTesting, RoleSecurity}
	sessoes := map[AgentRole]*GitWorktreeSession{}
	for i, papel := range papeis {
		sessao, err := CreateGitWorktree(ctx, repo, dataRoot, "missao-"+string(papel), "")
		if err != nil {
			t.Fatalf("worktree do papel %s: %v", papel, err)
		}
		sessoes[papel] = sessao
		t.Logf("worktree %d (%s): dir=%s branch=%s", i, papel, sessao.WorktreeDir, sessao.BranchName)
	}
	// Isolamento ESTRUTURAL: diretórios e branches distintos.
	vistosDir := map[string]AgentRole{}
	vistosBranch := map[string]AgentRole{}
	for papel, sessao := range sessoes {
		if outro, repetido := vistosDir[sessao.WorktreeDir]; repetido {
			t.Fatalf("worktrees %s e %s compartilham o diretório %s — não há isolamento", papel, outro, sessao.WorktreeDir)
		}
		if outro, repetido := vistosBranch[sessao.BranchName]; repetido {
			t.Fatalf("branches iguais (%s e %s): %s", papel, outro, sessao.BranchName)
		}
		vistosDir[sessao.WorktreeDir] = papel
		vistosBranch[sessao.BranchName] = papel
	}

	// 3. Orchestrator real com três papéis: cada runner grava o artefato NO SEU worktree (mapa por papel).
	var mu sync.Mutex
	escritas := map[AgentRole]string{}
	runner := func(_ context.Context, task AgentTask) (AgentResult, error) {
		sessao, achou := sessoes[task.Role]
		if !achou {
			return AgentResult{}, nil
		}
		destino := filepath.Join(sessao.WorktreeDir, "artefato-"+string(task.Role)+".txt")
		if err := os.WriteFile(destino, []byte("produzido por "+string(task.Role)+"\n"), 0o644); err != nil {
			return AgentResult{}, err
		}
		gitFixture(t, ctx, sessao.WorktreeDir, "add", "-A")
		gitFixture(t, ctx, sessao.WorktreeDir, "commit", "-q", "-m", "artefato "+string(task.Role))
		mu.Lock()
		escritas[task.Role] = destino
		mu.Unlock()
		return AgentResult{}, nil
	}

	orq, err := NewAgentOrchestrator(filepath.Join(raiz, "orquestrador"), runner)
	if err != nil {
		t.Fatal(err)
	}
	job, err := orq.Plan("produzir três artefatos independentes", repo, "projeto-t4", papeis,
		AgentBudget{MaxAgents: 3, MaxSeconds: 60, MaxOutputBytes: 1 << 16, MaxRetries: 0})
	if err != nil {
		t.Fatalf("planejar com três papéis: %v", err)
	}
	if _, err := orq.Run(ctx, job.ID); err != nil {
		t.Fatalf("executar orquestração: %v", err)
	}
	if len(escritas) != 3 {
		t.Fatalf("os três especialistas precisam ter produzido artefato: %d", len(escritas))
	}

	// 4. ISOLAMENTO MEDIDO: o artefato de um papel NÃO existe no worktree do outro nem no repositório base.
	for papel, sessao := range sessoes {
		meu := filepath.Join(sessao.WorktreeDir, "artefato-"+string(papel)+".txt")
		if _, err := os.Stat(meu); err != nil {
			t.Fatalf("o artefato do próprio papel precisa existir: %s", meu)
		}
		for outro, outra := range sessoes {
			if outro == papel {
				continue
			}
			intruso := filepath.Join(outra.WorktreeDir, "artefato-"+string(papel)+".txt")
			if _, err := os.Stat(intruso); err == nil {
				t.Fatalf("ISOLAMENTO QUEBRADO: artefato de %s apareceu no worktree de %s (%s)", papel, outro, intruso)
			}
		}
		if _, err := os.Stat(filepath.Join(repo, "artefato-"+string(papel)+".txt")); err == nil {
			t.Fatalf("ISOLAMENTO QUEBRADO: o artefato de %s vazou para o repositório base antes do merge", papel)
		}
	}

	// 5. MERGE CONTROLADO dos três branches, em sequência, no branch base.
	gitFixture(t, ctx, repo, "checkout", "-q", "main")
	for _, papel := range papeis {
		gitFixture(t, ctx, repo, "merge", "--no-ff", "-q", "-m", "merge "+string(papel), sessoes[papel].BranchName)
	}
	for _, papel := range papeis {
		if _, err := os.Stat(filepath.Join(repo, "artefato-"+string(papel)+".txt")); err != nil {
			t.Fatalf("após o merge controlado o artefato de %s precisa estar no branch base", papel)
		}
	}

	// 6. CONFLITO NÃO É RESOLVIDO SOZINHO: dois especialistas na MESMA linha.
	conflitoA, err := CreateGitWorktree(ctx, repo, dataRoot, "missao-conflito-a", "")
	if err != nil {
		t.Fatal(err)
	}
	conflitoB, err := CreateGitWorktree(ctx, repo, dataRoot, "missao-conflito-b", "")
	if err != nil {
		t.Fatal(err)
	}
	mesmoArquivoA := filepath.Join(conflitoA.WorktreeDir, "compartilhado.txt")
	mesmoArquivoB := filepath.Join(conflitoB.WorktreeDir, "compartilhado.txt")
	escrever(t, mesmoArquivoA, "versao do especialista A\n")
	escrever(t, mesmoArquivoB, "versao do especialista B\n")
	for _, sessao := range []*GitWorktreeSession{conflitoA, conflitoB} {
		gitFixture(t, ctx, sessao.WorktreeDir, "add", "-A")
		gitFixture(t, ctx, sessao.WorktreeDir, "commit", "-q", "-m", "compartilhado "+sessao.MissionID)
	}
	gitFixture(t, ctx, repo, "merge", "--no-ff", "-q", "-m", "merge A", conflitoA.BranchName)
	saida, errMerge := gitSaida(ctx, repo, "merge", "--no-ff", "-m", "merge B", conflitoB.BranchName)
	if errMerge == nil {
		t.Fatalf("o merge de duas versões da MESMA linha não pode passar em silêncio: %s", saida)
	}
	if !strings.Contains(strings.ToLower(saida), "conflict") && !strings.Contains(strings.ToLower(saida), "conflito") {
		t.Fatalf("a falha do merge precisa ser CONFLITO declarado, e não erro obscuro: %s", saida)
	}
	gitFixture(t, ctx, repo, "merge", "--abort") // o estado volta ao que era: ninguém escolhe sozinho
	if conteudo := ler(t, filepath.Join(repo, "compartilhado.txt")); !strings.Contains(conteudo, "especialista A") {
		t.Fatalf("após abortar o merge conflitante, o base tem de manter a versão já integrada: %q", conteudo)
	}

	// 7. Limpeza: os worktrees sao removidos sem deixar o repositorio preso.
	//
	// FASE 2 do adendo V3: a remocao passou a RECUSAR worktrees com trabalho nao integrado.
	// Os dois worktrees de conflito (`conflitoA`, `conflitoB`) tem exactamente isso -- o merge foi
	// abortado de proposito, e o commit de cada um nunca entrou no alvo. A guarda esta' certa a
	// recusar.
	//
	// Este teste DESCARTA esse trabalho de proposito, porque o conflito ja' foi provado acima.
	// Usar `RemoveGitWorktreeComPerda` com motivo e' a forma honesta de o dizer:
	//
	//	*Um teste que descarta trabalho tem de o declarar, senao amanha ninguem sabe se o
	//	comportamento seguro foi exercitado ou apenas contornado.*
	for _, sessao := range []*GitWorktreeSession{conflitoA, conflitoB} {
		if err := RemoveGitWorktreeComPerda(ctx, sessao, "teste: worktree de conflito descartado apos provar o aborto"); err != nil {
			t.Fatalf("descartar worktree de conflito %s: %v", sessao.MissionID, err)
		}
	}
	for _, sessao := range []*GitWorktreeSession{sessoes[papeis[0]], sessoes[papeis[1]], sessoes[papeis[2]]} {
		// Estes foram integrados pelo merge controlado; a remocao segura tem de os aceitar.
		if err := RemoveGitWorktree(ctx, sessao); err != nil {
			t.Fatalf("remover worktree %s: %v", sessao.MissionID, err)
		}
	}
}

// --- utilitários do teste (git real, sem mock) ---

func gitFixture(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	if _, err := gitSaida(ctx, dir, args...); err != nil {
		t.Fatalf("git %s em %s: %v", strings.Join(args, " "), dir, err)
	}
}

func gitSaida(ctx context.Context, dir string, args ...string) (string, error) {
	completo := append([]string{
		"-c", "user.email=teste@hades.local", "-c", "user.name=Teste Hades",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.CommandContext(ctx, "git", completo...)
	cmd.Dir = dir
	saida, err := cmd.CombinedOutput()
	return string(saida), err
}

func escrever(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ler(t *testing.T, caminho string) string {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(dados)
}
