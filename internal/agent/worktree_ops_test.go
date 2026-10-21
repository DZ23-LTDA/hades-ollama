package agent

// FASE 2 do adendo V3: testes das operacoes do ciclo de vida e da migracao de estado.
//
// O que estes testes provam:
//
//  1. A maquina de estados esta' LIGADA: `CreateGitWorktree` devolve `preparado`, nunca "active".
//  2. Suspender/retomar/arquivar funcionam e respeitam as transicoes.
//  3. Sessoes antigas com `Status = "active"` continuam a funcionar (migracao).
//  4. `DiagnosticarCicloDeVida` apanha a discordancia entre estado logico e disco.
//  5. `PodeRemoverAgora` exige as DUAS condicoes.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A maquina esta' LIGADA: a criacao devolve um estado da lista fechada.
func TestCriacaoDevolveEstadoDaListaFechada(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-estado-inicial")

	if !EstadoDoWorktree(sessao.Status).Valido() {
		t.Fatalf("estado devolvido %q nao pertence a' lista fechada", sessao.Status)
	}
	if EstadoDoWorktree(sessao.Status) != WorktreePreparado {
		t.Fatalf("estado inicial = %q; esperado preparado (o git worktree add ja' deixa o worktree pronto)",
			sessao.Status)
	}
	// E o valor historico "active" NAO pode aparecer.
	if sessao.Status == "active" {
		t.Fatal("criacao ainda devolve o valor historico \"active\", que nao pertence a' lista fechada")
	}
}

// Sessoes antigas com "active" continuam a funcionar.
//
//	*Um estado antigo que nao se sabe ler e' um worktree que se trata como se nao existisse.*
func TestNormalizarEstadoLegado(t *testing.T) {
	casos := []struct {
		bruto    string
		esperado EstadoDoWorktree
		traduz   bool
	}{
		{"active", WorktreeEmUso, true},
		{"merged", WorktreeIntegrado, true},
		{"rejected", WorktreeArquivado, true},
		{"removed", WorktreeRemovido, true},
		{"preparado", WorktreePreparado, false},
		{"", WorktreeCriado, false},
		{"inventado", WorktreeFalhado, false},
	}
	for _, caso := range casos {
		sessao := &GitWorktreeSession{Status: caso.bruto}
		estado, traduziu := NormalizarEstado(sessao)
		if estado != caso.esperado {
			t.Fatalf("%q -> %q; esperado %q", caso.bruto, estado, caso.esperado)
		}
		if traduziu != caso.traduz {
			t.Fatalf("%q: traduzido = %v; esperado %v", caso.bruto, traduziu, caso.traduz)
		}
		if sessao.Status != string(caso.esperado) {
			t.Fatalf("%q: Status nao foi normalizado (%q)", caso.bruto, sessao.Status)
		}
	}
	if _, traduziu := NormalizarEstado(nil); traduziu {
		t.Fatal("sessao nula marcada como traduzida")
	}
}

// Uma sessao com "active" pode ser usada pelas operacoes.
func TestSessaoLegadaAceitaOperacoes(t *testing.T) {
	sessao := &GitWorktreeSession{MissionID: "antiga", Status: "active"}
	if err := SuspenderWorktree(sessao); err != nil {
		t.Fatalf("suspender sessao legada: %v", err)
	}
	if EstadoDoWorktree(sessao.Status) != WorktreeSuspenso {
		t.Fatalf("estado = %q; esperado suspension", sessao.Status)
	}
	if err := RetomarWorktree(sessao); err != nil {
		t.Fatalf("retomar: %v", err)
	}
	if EstadoDoWorktree(sessao.Status) != WorktreeEmUso {
		t.Fatalf("estado = %q; esperado em_uso", sessao.Status)
	}
}

// Suspender e retomar num worktree REAL nao toca no trabalho.
func TestSuspenderERetomarPreservamTrabalho(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-suspender")
	ficheiro := filepath.Join(sessao.WorktreeDir, "trabalho.txt")
	if err := os.WriteFile(ficheiro, []byte("importante\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	if err := SuspenderWorktree(sessao); err != nil {
		t.Fatalf("suspender: %v", err)
	}
	if _, err := os.Stat(ficheiro); err != nil {
		t.Fatalf("suspender apagou trabalho: %v", err)
	}
	if err := RetomarWorktree(sessao); err != nil {
		t.Fatalf("retomar: %v", err)
	}
	if _, err := os.Stat(ficheiro); err != nil {
		t.Fatalf("retomar apagou trabalho: %v", err)
	}
}

// Arquivar a partir de `em_uso` passa por suspension automaticamente.
//
//	*Nao se arquiva debaixo de um agente a escrever.*
func TestArquivarAPartirDeEmUsoPassaPorSuspenso(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-arquivar")
	// Passar a em_uso.
	if _, _, err := Transitar(sessao, WorktreeEmUso); err != nil {
		t.Fatalf("passar a em_uso: %v", err)
	}
	if err := ArquivarWorktree(sessao); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	if EstadoDoWorktree(sessao.Status) != WorktreeArquivado {
		t.Fatalf("estado = %q; esperado arquivado", sessao.Status)
	}
	// E o diretorio continua la': *arquivar e' o oposto de apagar.*
	if _, err := os.Stat(sessao.WorktreeDir); err != nil {
		t.Fatalf("arquivar apagou o diretorio: %v", err)
	}
}

// Arquivar um worktree `criado` e' recusado (nao esta' na tabela).
func TestArquivarDeCriadoERecusado(t *testing.T) {
	sessao := &GitWorktreeSession{MissionID: "x", Status: string(WorktreeCriado)}
	err := ArquivarWorktree(sessao)
	if err == nil {
		t.Fatal("arquivar a partir de criado foi aceite")
	}
	if !errors.Is(err, ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}

// MarcarIntegrado regista o facto, mas nao salta de criado.
func TestMarcarIntegradoNaoSaltaDeCriado(t *testing.T) {
	sessao := &GitWorktreeSession{MissionID: "x", Status: string(WorktreeCriado)}
	if err := MarcarIntegrado(sessao); err == nil {
		t.Fatal("marcar integrado a partir de criado foi aceite")
	}
	// A partir de em_uso funciona.
	sessao2 := &GitWorktreeSession{MissionID: "y", Status: string(WorktreeEmUso)}
	if err := MarcarIntegrado(sessao2); err != nil {
		t.Fatalf("marcar integrado de em_uso: %v", err)
	}
	if EstadoDoWorktree(sessao2.Status) != WorktreeIntegrado {
		t.Fatalf("estado = %q; esperado integrado", sessao2.Status)
	}
}

// Sessao nula nas operacoes nao rebenta.
func TestOperacoesComSessaoNula(t *testing.T) {
	if err := SuspenderWorktree(nil); err == nil {
		t.Fatal("suspender nil aceite")
	}
	if err := RetomarWorktree(nil); err == nil {
		t.Fatal("retomar nil aceite")
	}
	if err := ArquivarWorktree(nil); err == nil {
		t.Fatal("arquivar nil aceite")
	}
	if err := MarcarIntegrado(nil); err == nil {
		t.Fatal("marcar integrado nil aceite")
	}
}

// O diagnostico de ciclo de vida descreve estado, disco e proximos passos.
func TestDiagnosticoDoCicloDeVida(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-diag-ciclo")

	d := DiagnosticarCicloDeVida(context.Background(), sessao)
	if d.MissionID != "missao-diag-ciclo" {
		t.Fatalf("mission_id = %q", d.MissionID)
	}
	if d.Estado != WorktreePreparado {
		t.Fatalf("estado = %q; esperado preparado", d.Estado)
	}
	if !d.Disco.AlvoSeguro {
		t.Fatalf("disco diz alvo inseguro: %s", d.Disco.MotivoDoAlvo)
	}
	if len(d.Proximos) == 0 {
		t.Fatal("diagnostico sem proximos estados")
	}
	if d.Inconsistente != "" {
		t.Fatalf("worktree recem-criado marcado inconsistente: %s", d.Inconsistente)
	}
}

// O diagnostico APANHA a inconsistencia: estado diz integrado mas ha' trabalho por integrar.
func TestDiagnosticoApanhaEstadoFantasma(t *testing.T) {
	repo := repoFixture(t)
	sessao := criarWorktree(t, repo, t.TempDir(), "missao-fantasma")
	if _, _, err := Transitar(sessao, WorktreeEmUso); err != nil {
		t.Fatalf("em_uso: %v", err)
	}
	if err := MarcarIntegrado(sessao); err != nil {
		t.Fatalf("integrar: %v", err)
	}
	// Depois de "integrado", escrever trabalho novo -- o estado passa a mentir.
	if err := os.WriteFile(filepath.Join(sessao.WorktreeDir, "novo.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	d := DiagnosticarCicloDeVida(context.Background(), sessao)
	if d.Inconsistente == "" {
		t.Fatal("estado diz integrado com trabalho por commitar e o diagnostico nao apontou inconsistencia: " +
			"*um estado que mente e' pior do que nenhum*")
	}
}

// PodeRemoverAgora exige as DUAS condicoes.
func TestPodeRemoverAgora(t *testing.T) {
	repo := repoFixture(t)

	// Caso 1: worktree limpo e estado que permite remover -> SIM.
	limpo := criarWorktree(t, repo, t.TempDir(), "missao-pode-1")
	if ok, motivo := PodeRemoverAgora(context.Background(), limpo); !ok {
		t.Fatalf("worktree limpo recusado: %s", motivo)
	}

	// Caso 2: worktree com trabalho -> NAO.
	sujo := criarWorktree(t, repo, t.TempDir(), "missao-pode-2")
	if err := os.WriteFile(filepath.Join(sujo.WorktreeDir, "x.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if ok, motivo := PodeRemoverAgora(context.Background(), sujo); ok {
		t.Fatal("worktree com trabalho por commitar foi autorizado a remover")
	} else if motivo == "" {
		t.Fatal("recusa sem motivo")
	}

	// Caso 3: estado que NAO permite remover (em_uso) -> NAO, mesmo limpo.
	emUso := criarWorktree(t, repo, t.TempDir(), "missao-pode-3")
	if _, _, err := Transitar(emUso, WorktreeEmUso); err != nil {
		t.Fatalf("em_uso: %v", err)
	}
	if ok, motivo := PodeRemoverAgora(context.Background(), emUso); ok {
		t.Fatal("worktree em_uso foi autorizado a remover: *nao se apaga debaixo de um agente*")
	} else if motivo == "" {
		t.Fatal("recusa sem motivo")
	}

	// Caso 4: sessao nula.
	if ok, _ := PodeRemoverAgora(context.Background(), nil); ok {
		t.Fatal("sessao nula autorizada")
	}
}
