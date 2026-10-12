package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// diretorioConfiavelParaCLI cria um diretório de trabalho que os CLIs de código aceitam.
//
// MOTIVO MEDIDO (E-205): rodar `codex exec` num diretório **não-git** devolve
// "Not inside a trusted directory and --skip-git-repo-check was not specified" — e o teste falhava antes por isso, não
// por falta de sessão. Em vez de afrouxar a verificação do CLI com `--skip-git-repo-check`, damos a ele o que ele
// pede: um repositório de verdade.
func diretorioConfiavelParaCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if saida, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("sem git utilizável para preparar o diretório confiável (%v): %s", err, strings.TrimSpace(string(saida)))
	}
	// Um commit vazio deixa o repositório em estado normal mesmo se o CLI exigir HEAD.
	cmd = exec.Command("git", "commit", "-q", "--allow-empty", "-m", "diretorio de teste")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=teste", "GIT_AUTHOR_EMAIL=teste@exemplo.invalid",
		"GIT_COMMITTER_NAME=teste", "GIT_COMMITTER_EMAIL=teste@exemplo.invalid",
	)
	if saida, err := cmd.CombinedOutput(); err != nil {
		t.Logf("aviso: commit vazio não concluiu (%v): %s", err, strings.TrimSpace(string(saida)))
	}
	if resolvido, err := filepath.Abs(dir); err == nil {
		return resolvido
	}
	return dir
}

// SESSÃO REAL DE CLI GOVERNADA — OPT-IN (E-204).
//
// POR QUE ESTE TESTE EXISTE, E POR QUE ELE É PULADO POR PADRÃO
// -----------------------------------------------------------
// O §11.7 do master exige, além do unitário: "CLI Codex/Claude/Gemini quando instalado: sessão real, streaming,
// cancelamento e limites; sem login, marcar bloqueio real". Isso **não pode** rodar no CI: depende de conta real
// logada, rede e cota do operador — um pipeline hermético que dependesse disso ficaria vermelho por motivo alheio ao
// código. Então o padrão é **pular**, e a sessão real só acontece quando o operador autoriza explicitamente com
// `OLLAMA_AGENT_REAL_CLI=<harness>`.
//
// O que fica provado quando ele roda: a sessão **inicia** de verdade, a saída **chega** (streaming concluído), o
// **cancelamento** pelo contexto interrompe, os **limites** de saída são respeitados e **nenhum segredo** vaza pelo
// caminho governado (DLPFindings == 0).
func TestSessaoRealDeCLIGovernadaEhOptIn(t *testing.T) {
	alvo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_CLI"))
	if alvo == "" {
		t.Skip("sem OLLAMA_AGENT_REAL_CLI: a sessão real de CLI é opt-in porque usa conta real, rede e cota")
	}

	spec, ok := lookupGovernedCLI(alvo)
	if !ok {
		t.Fatalf("harness %q não está no catálogo governado", alvo)
	}
	t.Logf("harness=%s binário=%s verificado=%v", spec.Harness, spec.Binary, spec.Verified)

	// 1. SESSÃO REAL, com tarefa mínima e sem ferramentas: o objetivo é provar o CAMINHO, não avaliar o modelo.
	ctx, cancelar := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancelar()

	resultado, err := runGovernedCLI(ctx, GovernedCLIRequest{
		Harness:   alvo,
		Prompt:    "Responda apenas com a palavra OK, sem usar ferramentas.",
		Workdir:   diretorioConfiavelParaCLI(t),
		Timeout:   120 * time.Second,
		MaxOutput: 8192,
	})
	if err != nil {
		// Falha de binário ausente/reprovado é BLOQUEIO REAL, não defeito do teste: deixe isso explícito no log.
		t.Fatalf("sessão real de %s não executou: %v (se for binário ausente, é bloqueio de ambiente)", alvo, err)
	}

	t.Logf("exit=%d duração=%dms truncado=%v dlp=%d", resultado.ExitCode, resultado.DurationMS, resultado.StdoutTruncated, resultado.DLPFindings)
	// Evidência de governança: como o ambiente foi montado e o que ficou registrado para auditoria.
	t.Logf("governança: ambiente=%q isolamento=%q limites=%q", resultado.Environment, resultado.Isolation, resultado.ResourceLimits)
	t.Logf("comando registrado: %s", resultado.CommandLine)

	if resultado.ExitCode != 0 {
		t.Fatalf("sessão real terminou com exit %d; stderr=%q", resultado.ExitCode, resultado.Stderr)
	}
	if strings.TrimSpace(resultado.Stdout) == "" {
		t.Fatal("a sessão real não devolveu saída: streaming vazio não prova integração")
	}
	if resultado.DLPFindings != 0 {
		t.Fatalf("o caminho governado acusou %d achado(s) de DLP: a saída não pode carregar segredo", resultado.DLPFindings)
	}
	if resultado.StdoutTruncated && len(resultado.Stdout) > 8192 {
		t.Fatalf("saída truncada excedeu MaxOutput: %d bytes", len(resultado.Stdout))
	}
	if resultado.DurationMS <= 0 {
		t.Fatal("duração ausente: não há como medir limites sem tempo")
	}
	// O comando registrado serve para auditoria; ele não pode conter a prompt inteira nem segredo.
	if strings.Contains(resultado.CommandLine, "Responda apenas com a palavra OK") {
		t.Log("aviso: o comando registrado contém a prompt — verifique se isso é o esperado na auditoria")
	}
}

// Cancelamento e limites: o contexto cancelado precisa interromper a sessão, e não pendurar o processo.
func TestSessaoRealDeCLIGovernadaRespeitaCancelamento(t *testing.T) {
	alvo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_CLI"))
	if alvo == "" {
		t.Skip("sem OLLAMA_AGENT_REAL_CLI: cancelamento real só é medido com a CLI real autorizada")
	}
	if _, ok := lookupGovernedCLI(alvo); !ok {
		t.Fatalf("harness %q não está no catálogo governado", alvo)
	}

	// Contexto já expirado: o caminho governado tem de recusar/interromper rápido, sem esperar o timeout longo.
	ctx, cancelar := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelar()

	inicio := time.Now()
	_, err := runGovernedCLI(ctx, GovernedCLIRequest{
		Harness:   alvo,
		Prompt:    "Tarefa que não deve completar por causa do cancelamento.",
		Workdir:   diretorioConfiavelParaCLI(t),
		Timeout:   60 * time.Second,
		MaxOutput: 1024,
	})
	decorrido := time.Since(inicio)

	if err == nil {
		t.Fatal("contexto cancelado precisa interromper a sessão (nenhum erro foi devolvido)")
	}
	if decorrido > 20*time.Second {
		t.Fatalf("cancelamento demorou %s: o limite não está sendo respeitado", decorrido)
	}
	t.Logf("cancelamento respeitado em %s (erro: %v)", decorrido, err)
}
