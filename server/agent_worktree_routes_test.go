package server

// FASE 2 do adendo V3: testes da superficie de worktrees.
//
// O que estes testes provam:
//
//  1. O inventor mostra worktrees REAIS do repositorio, com git de verdade.
//  2. Worktree externo (criado fora do HADES) aparece e e' marcado como tal.
//  3. `repo` ausente e' recusado -- nao se inventaria o checkout errado.
//  4. O diagnostico usa o INVENTOR, nao um caminho construido (licao da E-1231).
//  5. Missao inexistente da' 404 com dica, nao um estado inventado.
//  6. As rotas sao SO' LEITURA: POST e' recusado.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// routerDeWorktrees monta o mesmo engine usado pelos testes de toolchain/install: o `agentAPI`
// real, com as rotas reais. *Testar o handler sem o router deixaria o registo da rota por provar.*
func routerDeWorktrees(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	return engine
}

// getCorpo lê o corpo CRU de qualquer status -- `getJSON` so' descodifica em 200, e os casos de
// erro sao exactamente os que precisam de ser lidos.
func getCorpo(t *testing.T, engine *gin.Engine, caminho string) (int, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, caminho, nil))
	return recorder.Code, recorder.Body.String()
}

// postSemCorpo prova que a rota recusa mutacao.
func postSemCorpo(t *testing.T, engine *gin.Engine, caminho string) int {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, caminho, nil))
	return recorder.Code
}

// repoDeWorktree cria um repositorio git real com um commit.
func repoDeWorktree(t *testing.T) string {
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
		if saida, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, saida)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "inicial"}} {
		if saida, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, saida)
		}
	}
	return dir
}

// POST e' recusado: as rotas sao so' leitura.
//
//	*Remover exige uma decisao sobre trabalho nao integrado; nao pode ser efeito lateral de um GET
//	-- e muito menos de um POST que ninguem desenhou.*
func TestWorktreesRecusaPost(t *testing.T) {
	engine := routerDeWorktrees(t)
	for _, caminho := range []string{"/api/agent/v1/worktrees", "/api/agent/v1/worktrees/diagnose"} {
		code := postSemCorpo(t, engine, caminho)
		if code == http.StatusOK {
			t.Fatalf("POST %s foi aceite; as rotas de worktree sao so' leitura", caminho)
		}
	}
}

// `repo` ausente e' recusado com 400 e explicacao.
func TestWorktreesExigeRepo(t *testing.T) {
	engine := routerDeWorktrees(t)
	code, corpo := getCorpo(t, engine, "/api/agent/v1/worktrees")
	if code != http.StatusBadRequest {
		t.Fatalf("GET sem repo -> %d; esperado 400", code)
	}
	if !strings.Contains(corpo, "erro") {
		t.Fatalf("400 sem mensagem de erro: %s", corpo)
	}
}

// Um diretorio que nao e' repositorio e' recusado.
func TestWorktreesRecusaNaoRepo(t *testing.T) {
	engine := routerDeWorktrees(t)
	var payload map[string]any
	code := getJSON(t, engine, "/api/agent/v1/worktrees?repo="+t.TempDir(), &payload)
	if code != http.StatusBadRequest {
		t.Fatalf("repo invalido -> %d; esperado 400", code)
	}
}

// O inventor mostra o worktree PRINCIPAL e os criados pelo HADES.
func TestWorktreesInventariaRepositorioReal(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	if _, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-rota", ""); err != nil {
		t.Fatalf("criar worktree: %v", err)
	}

	engine := routerDeWorktrees(t)
	var payload worktreeInventarioResposta
	if code := getJSON(t, engine, "/api/agent/v1/worktrees?repo="+repo, &payload); code != http.StatusOK {
		t.Fatalf("GET worktrees -> %d", code)
	}
	if payload.Total < 2 {
		t.Fatalf("total = %d; esperado >= 2 (principal + o criado)", payload.Total)
	}
	if payload.DoHades < 1 {
		t.Fatal("nenhum worktree marcado como criado pelo HADES")
	}
	if !payload.SomenteLeitura {
		t.Fatal("resposta nao se declara so' leitura")
	}
	if payload.Lista == nil {
		t.Fatal("lista nula; devia ser []")
	}
	// Exatamente um principal.
	principais := 0
	for _, w := range payload.Lista {
		if w.Principal {
			principais++
		}
	}
	if principais != 1 {
		t.Fatalf("%d worktrees principais; esperado 1", principais)
	}
	if len(payload.Avisos) == 0 {
		t.Fatal("sem avisos apesar de haver worktrees do HADES")
	}
}

// Um worktree criado FORA do HADES aparece e e' marcado como externo.
func TestWorktreesMostraExterno(t *testing.T) {
	repo := repoDeWorktree(t)
	externo := filepath.Join(t.TempDir(), "externo")
	if saida, err := exec.Command("git", "-C", repo, "worktree", "add", "-b", "externo/rota", externo, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("criar worktree externo: %v (%s)", err, saida)
	}
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", repo, "worktree", "remove", "--force", externo).Run()
	})

	engine := routerDeWorktrees(t)
	var payload worktreeInventarioResposta
	if code := getJSON(t, engine, "/api/agent/v1/worktrees?repo="+repo, &payload); code != http.StatusOK {
		t.Fatalf("GET worktrees -> %d", code)
	}
	if payload.Externo < 1 {
		t.Fatal("worktree externo nao contado: *um worktree externo que nao se ve' e' um worktree que se apaga por engano*")
	}
	achou := false
	for _, w := range payload.Lista {
		if filepath.Base(w.CaminhoResolvido) == "externo" {
			achou = true
			if w.CriadoPeloHades {
				t.Fatal("worktree externo marcado como criado pelo HADES")
			}
		}
	}
	if !achou {
		t.Fatal("worktree externo ausente do inventor")
	}
}

// O diagnostico encontra a missao PELO INVENTOR e devolve o estado.
func TestWorktreesDiagnosticaMissao(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-diag", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	engine := routerDeWorktrees(t)
	var payload worktreeDiagnosticoResposta
	if code := getJSON(t, engine, "/api/agent/v1/worktrees/diagnose?repo="+repo+"&missao=missao-diag", &payload); code != http.StatusOK {
		t.Fatalf("GET diagnose -> %d", code)
	}
	if payload.MissionID != "missao-diag" {
		t.Fatalf("mission_id = %q", payload.MissionID)
	}
	if !payload.EstadoValido {
		t.Fatalf("estado %q nao pertence a' lista fechada", payload.Estado)
	}
	if len(payload.Proximos) == 0 {
		t.Fatal("diagnostico sem proximos estados")
	}
	if !payload.AlvoSeguro {
		t.Fatal("alvo legitimo marcado inseguro")
	}
	if !payload.PodeRemover {
		t.Fatalf("worktree limpo nao pode ser removido: %s", payload.MotivoRemover)
	}
	if payload.EstadoObservado == "" {
		t.Fatal("diagnostico nao declara que o estado logico pertence ao processo que criou a sessao")
	}
}

// Missao inexistente da' 404 com dica -- nao um estado inventado.
func TestWorktreesDiagnosticoMissaoInexistente(t *testing.T) {
	repo := repoDeWorktree(t)
	engine := routerDeWorktrees(t)
	code, corpo := getCorpo(t, engine, "/api/agent/v1/worktrees/diagnose?repo="+repo+"&missao=nao-existe")
	if code != http.StatusNotFound {
		t.Fatalf("missao inexistente -> %d; esperado 404", code)
	}
	if !strings.Contains(corpo, "dica") {
		t.Fatalf("404 sem dica de proximo passo: %s", corpo)
	}
}

// O diagnostico exige repo E missao.
func TestWorktreesDiagnosticoExigeParametros(t *testing.T) {
	engine := routerDeWorktrees(t)
	for _, url := range []string{
		"/api/agent/v1/worktrees/diagnose",
		"/api/agent/v1/worktrees/diagnose?repo=x",
		"/api/agent/v1/worktrees/diagnose?missao=y",
	} {
		var payload map[string]any
		if code := getJSON(t, engine, url, &payload); code != http.StatusBadRequest {
			t.Fatalf("%s -> %d; esperado 400", url, code)
		}
	}
}
