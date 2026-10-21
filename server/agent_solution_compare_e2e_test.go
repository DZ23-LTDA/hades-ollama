package server

// FASE 3: E2E REAL DE "COMPARAR DUAS SOLUCOES E EXIGIR APROVACAO ANTES DE INTEGRAR".
//
// Este e' um dos cenarios E2E minimos que o adendo lista textualmente e que nunca tinha sido
// executado. As pecas existiam soltas:
//
//	E-1239  `agent.CompararSolucoes`            -- a funcao
//	E-1240  POST /api/agent/v1/solutions/compare -- a rota
//	E-1238  harness com git real                 -- o substrate
//
//	*Tres pecas provadas em separado nao sao uma integracao provada.*
//
// O QUE ESTE TESTE FAZ, E PORQUE E' QUE CONTA
//
//  1. cria um REPOSITORIO git REAL e DUAS solucoes em worktrees separados;
//  2. mede mesmo -- `git diff --stat` de verdade, contagem de ficheiros de verdade;
//  3. chama a rota de PRODUCAO com essas medicoes;
//  4. exige `integra_automaticamente: false` na resposta;
//  5. **prova que NADA foi integrado**: a branch principal continua no mesmo commit e nenhuma
//     solucao aparece nela.
//
// O passo 5 e' o que separa "a rota responde" de "a rota nao integra":
//
//	*Uma comparacao que responde `nao integrei` e' uma promessa. Uma comparacao seguida de
//	`git rev-parse main` igual e' uma prova.*

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// gitNoRepo corre git e devolve a saida, falhando o teste com contexto util.
func gitNoRepo(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=HADES", "GIT_AUTHOR_EMAIL=hades@dz23.invalid",
		"GIT_COMMITTER_NAME=HADES", "GIT_COMMITTER_EMAIL=hades@dz23.invalid",
	)
	saida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v falhou: %v\n%s", args, err, saida)
	}
	return strings.TrimSpace(string(saida))
}

// prepararDuasSolucoes cria um repo real e duas solucoes em worktrees separados.
//
// Devolve: caminho do repo, caminho da solucao A, caminho da solucao B, e o commit base.
func prepararDuasSolucoes(t *testing.T) (repo, solucaoA, solucaoB, base string) {
	t.Helper()
	raiz := t.TempDir()
	repo = filepath.Join(raiz, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatalf("criar repo: %v", err)
	}
	gitNoRepo(t, repo, "init", "-b", "main")
	gitNoRepo(t, repo, "config", "user.email", "hades@dz23.invalid")
	gitNoRepo(t, repo, "config", "user.name", "HADES")

	// Ficheiro base: as duas solucoes vao altera-lo de formas diferentes.
	if err := os.WriteFile(filepath.Join(repo, "app.go"), []byte("package app\n"), 0o600); err != nil {
		t.Fatalf("escrever base: %v", err)
	}
	gitNoRepo(t, repo, "add", ".")
	gitNoRepo(t, repo, "commit", "-m", "base")
	base = gitNoRepo(t, repo, "rev-parse", "HEAD")

	// Duas solucoes em worktrees separados: e' isto que impede que uma pise a outra.
	solucaoA = filepath.Join(raiz, "solucao-a")
	solucaoB = filepath.Join(raiz, "solucao-b")
	gitNoRepo(t, repo, "worktree", "add", "-b", "solucao/a", solucaoA, base)
	gitNoRepo(t, repo, "worktree", "add", "-b", "solucao/b", solucaoB, base)

	// A: pequena e correcta (1 linha).
	if err := os.WriteFile(filepath.Join(solucaoA, "app.go"),
		[]byte("package app\n\nfunc Soma(a, b int) int { return a + b }\n"), 0o600); err != nil {
		t.Fatalf("escrever A: %v", err)
	}
	// B: maior, com um ficheiro a mais.
	if err := os.WriteFile(filepath.Join(solucaoB, "app.go"),
		[]byte("package app\n\n// Soma devolve a soma.\nfunc Soma(a, b int) int {\n\treturn a + b\n}\n"), 0o600); err != nil {
		t.Fatalf("escrever B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(solucaoB, "extra.go"),
		[]byte("package app\n\nconst Versao = 2\n"), 0o600); err != nil {
		t.Fatalf("escrever B extra: %v", err)
	}
	for _, wt := range []string{solucaoA, solucaoB} {
		gitNoRepo(t, wt, "add", ".")
		gitNoRepo(t, wt, "commit", "-m", "solucao de "+filepath.Base(wt))
	}
	return repo, solucaoA, solucaoB, base
}

// linhasAlteradas mede o diff real de um worktree contra o commit base.
func linhasAlteradas(t *testing.T, worktree, base string) (linhas, ficheiros float64) {
	t.Helper()
	stat := gitNoRepo(t, worktree, "diff", "--stat", base, "HEAD")
	if stat == "" {
		return 0, 0
	}
	// A ultima linha do --stat e' "N files changed, X insertions(+), Y deletions(-)".
	ultima := stat[strings.LastIndex(stat, "\n")+1:]
	for _, parte := range strings.Split(ultima, ",") {
		parte = strings.TrimSpace(parte)
		campos := strings.Fields(parte)
		if len(campos) == 0 {
			continue
		}
		n, err := strconv.ParseFloat(campos[0], 64)
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(parte, "changed"):
			ficheiros = n
		case strings.Contains(parte, "insertion"), strings.Contains(parte, "deletion"):
			linhas += n
		}
	}
	return linhas, ficheiros
}

// TestE2ECompararDuasSolucoesNaoIntegra e' o cenario minimo do adendo, ponta a ponta.
func TestE2ECompararDuasSolucoesNaoIntegra(t *testing.T) {
	repo, solucaoA, solucaoB, base := prepararDuasSolucoes(t)

	linhasA, ficheirosA := linhasAlteradas(t, solucaoA, base)
	linhasB, ficheirosB := linhasAlteradas(t, solucaoB, base)
	if linhasA == 0 || linhasB == 0 {
		t.Fatalf("medicao invalida: A=%v B=%v linhas", linhasA, linhasB)
	}
	if linhasB <= linhasA {
		t.Logf("AVISO: B tem %v linhas e A tem %v; o teste conta com A mais pequena", linhasB, linhasA)
	}

	// As duas solucoes passam nos testes (a medicao e' entrada do chamador, nao invencao da rota).
	candidatos := []agent.Candidato{
		{
			ID:             "solucao-a",
			Worktree:       solucaoA,
			Branch:         "solucao/a",
			Commit:         gitNoRepo(t, solucaoA, "rev-parse", "HEAD"),
			TestesPassaram: 12,
			TestesFalharam: 0,
			Medicoes: []agent.MedicaoDeCandidato{
				{Criterio: agent.CriterioTempo, Valor: 2, Medido: true, Fonte: "teste"},
				{Criterio: agent.CriterioTamanhoDoDiff, Valor: linhasA, Unidade: "linhas", Medido: true, Fonte: "git diff --stat"},
			},
			Evidencias: []string{"E-1239", "E-1240"},
		},
		{
			ID:             "solucao-b",
			Worktree:       solucaoB,
			Branch:         "solucao/b",
			Commit:         gitNoRepo(t, solucaoB, "rev-parse", "HEAD"),
			TestesPassaram: 12,
			TestesFalharam: 0,
			Medicoes: []agent.MedicaoDeCandidato{
				{Criterio: agent.CriterioTempo, Valor: 5, Medido: true, Fonte: "teste"},
				{Criterio: agent.CriterioTamanhoDoDiff, Valor: linhasB, Unidade: "linhas", Medido: true, Fonte: "git diff --stat"},
			},
			Evidencias: []string{"E-1239", "E-1240"},
		},
	}
	_ = ficheirosA
	_ = ficheirosB

	// === A ROTA DE PRODUCAO ===
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&agentAPI{}).register(router) // o mesmo ponto de entrada do produto

	corpo, err := json.Marshal(map[string]any{
		"missao":     "e2e-compare-duas-solucoes",
		"candidatos": candidatos,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/solutions/compare", bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200. corpo=%s", w.Code, w.Body.String())
	}
	var resposta map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta nao e' JSON: %v", err)
	}

	// === AS GARANTIAS ===
	if resposta["integra_automaticamente"] != false {
		t.Fatalf("integra_automaticamente = %v; a rota NAO pode integrar", resposta["integra_automaticamente"])
	}
	if resposta["somente_leitura"] != true {
		t.Fatalf("somente_leitura = %v; a rota nao muta nada", resposta["somente_leitura"])
	}
	if resposta["exige_aprovacao_humana"] != true {
		t.Fatalf("exige_aprovacao_humana = %v; tem de exigir", resposta["exige_aprovacao_humana"])
	}
	vencedor, _ := resposta["vencedor"].(string)
	if vencedor == "" {
		t.Fatalf("sem vencedor recomendado; resposta=%v", resposta)
	}
	// A vence por tempo (2 < 5) -- o primeiro criterio medido por todos.
	if vencedor != "solucao-a" {
		t.Fatalf("vencedor = %q; esperado solucao-a (tempo 2 < 5)", vencedor)
	}

	// === A PROVA QUE A ROTA NAO INTEGRA ===
	//
	// *Uma comparacao que responde "nao integrei" e' uma promessa. Uma comparacao seguida de
	// `git rev-parse main` igual e' uma prova.*
	depois := gitNoRepo(t, repo, "rev-parse", "main")
	if depois != base {
		t.Fatalf("main mudou de %s para %s: a comparacao INTEGROU alguma coisa", base, depois)
	}
	// Nenhuma branch de solucao esta' merged em main.
	merged := gitNoRepo(t, repo, "branch", "--merged", "main")
	for _, linha := range strings.Split(merged, "\n") {
		if strings.Contains(linha, "solucao/") {
			t.Fatalf("branch de solucao aparece merged em main: %q", linha)
		}
	}
	// O ficheiro novo da solucao B NAO existe no checkout principal.
	if _, err := os.Stat(filepath.Join(repo, "extra.go")); err == nil {
		t.Fatal("extra.go da solucao B aparece no repo principal: houve integracao")
	}
	// As duas branches continuam a existir: nada foi apagado.
	for _, b := range []string{"solucao/a", "solucao/b"} {
		gitNoRepo(t, repo, "rev-parse", "--verify", "refs/heads/"+b)
	}
}

// TestE2ECompararDuasSolucoesComTesteVermelhoNaoRecomenda: a solucao que falha testes nao vence,
// por mais rapida que seja. E' a regra da E-1239 exercitada com git real.
func TestE2ECompararDuasSolucoesComTesteVermelhoNaoRecomenda(t *testing.T) {
	repo, solucaoA, solucaoB, base := prepararDuasSolucoes(t)
	linhasA, _ := linhasAlteradas(t, solucaoA, base)

	candidatos := []agent.Candidato{
		{
			ID: "rapida-e-quebrada", Worktree: solucaoA, Branch: "solucao/a",
			TestesPassaram: 11, TestesFalharam: 1,
			Medicoes: []agent.MedicaoDeCandidato{
				{Criterio: agent.CriterioTempo, Valor: 1, Medido: true},
				{Criterio: agent.CriterioTamanhoDoDiff, Valor: linhasA, Medido: true},
			},
		},
		{
			ID: "lenta-e-correcta", Worktree: solucaoB, Branch: "solucao/b",
			TestesPassaram: 12, TestesFalharam: 0,
			Medicoes: []agent.MedicaoDeCandidato{
				{Criterio: agent.CriterioTempo, Valor: 100, Medido: true},
				{Criterio: agent.CriterioTamanhoDoDiff, Valor: 100, Medido: true},
			},
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&agentAPI{}).register(router)
	corpo, _ := json.Marshal(map[string]any{"candidatos": candidatos})
	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/solutions/compare", bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200", w.Code)
	}
	var resposta map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if resposta["vencedor"] != "lenta-e-correcta" {
		t.Fatalf("vencedor = %v; esperado lenta-e-correcta (a rapida falhou 1 teste)",
			resposta["vencedor"])
	}
	if depois := gitNoRepo(t, repo, "rev-parse", "main"); depois != base {
		t.Fatalf("main mudou: a comparacao integrou")
	}
}
