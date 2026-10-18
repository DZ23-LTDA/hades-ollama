package server

// FASE 2 (E-1238): HARNESS de servidor REAL para o E2E de browser.
//
// O PROBLEMA QUE ISTO RESOLVE
//
// O spec `e2e/worktrees-panel.spec.ts` (E-1237) tem dois modos declarados. O modo
// `UI_FIXTURE` correu; o modo `BACKEND_REAL_AGENT_RUNTIME` ficou pronto mas **nao exercitado**,
// porque nao havia um servidor Hades a servir um repositorio git real neste ambiente.
//
// UM FIXTURE NAO RESOLVE ISSO
//
//	*Um fixture que se faz passar por backend real e' pior do que nenhum fixture: prova
//	integracao que nao aconteceu.*
//
// O QUE ESTE FICHEIRO E'
//
// Um teste que, quando `HADES_E2E_HARNESS_PORT` esta' definido, monta o **router de PRODUCAO**
// (`(&agentAPI{}).register`, exactamente o que os testes de rota usam) num `httptest.Server` com
// **TCP real e porto fixo**, cria **repositorios git reais** com worktrees reais, e bloqueia.
//
//	*O servidor e' o de producao. O repositorio e' git de verdade. O unico atalho e' o processo
//	que o hospeda -- e esse atalho e' declarado aqui, nao escondido.*
//
// O QUE ELE NAO E'
//
// Nao e' um segundo backend: e' o mesmo `register` que serve o produto. Nao ha' mock de HTTP: o
// Playwright fala com um socket. As rotas que respondem sao as mesmas `agent_worktree_routes.go` e
// `agent_worktree_state_route.go`.
//
// COMO CORRER
//
//	HADES_E2E_HARNESS_PORT=43217 go test ./server/ -run TestHarnessE2EBackendReal -v
//
// e, noutra consola:
//
//	HADES_E2E_BACKEND_URL=http://127.0.0.1:43217 HADES_E2E_REPO=<caminho> npx playwright test
//
// Sem a variavel, o teste e' pulado: `go test ./server/` continua a correr toda a suite sem
// abrir portas nem bloquear.

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestHarnessE2EBackendReal sobe o router de PRODUCAO num porto fixo e serve repositorios git
// reais ate' o processo ser morto.
//
// Sem `HADES_E2E_HARNESS_PORT`, salta -- *um teste que abre portas sem ser pedido e' um efeito
// secundario que ninguem autorizou.*
func TestHarnessE2EBackendReal(t *testing.T) {
	portaTexto := os.Getenv("HADES_E2E_HARNESS_PORT")
	if portaTexto == "" {
		t.Skip("HADES_E2E_HARNESS_PORT nao definido: harness de backend real nao pedido")
	}
	porta, err := strconv.Atoi(portaTexto)
	if err != nil || porta <= 0 || porta > 65535 {
		t.Fatalf("HADES_E2E_HARNESS_PORT invalido: %q", portaTexto)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponivel: sem repositorio real o harness nao prova nada")
	}

	raiz := os.Getenv("HADES_E2E_REPO_ROOT")
	if raiz == "" {
		raiz = filepath.Join(os.TempDir(), "hades-e2e-real")
	}
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatalf("criar raiz: %v", err)
	}

	// Repositorio git REAL com um worktree REAL.
	repo, worktree, err := prepararRepositorioReal(t, raiz)
	if err != nil {
		t.Fatalf("preparar repositorio real: %v", err)
	}
	// O Playwright precisa de saber onde estao' -- e o teste imprime, para o operador poder
	// exportar `HADES_E2E_REPO` sem adivinhar.
	t.Logf("HADES_E2E_REPO=%s", repo)
	t.Logf("worktree=%s", worktree)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	// O MESMO registo que serve o produto.
	(&agentAPI{}).register(engine)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", porta))
	if err != nil {
		t.Fatalf("escutar em 127.0.0.1:%d: %v", porta, err)
	}
	servidor := httptest.NewUnstartedServer(engine)
	servidor.Listener.Close() // libertar o porto aleatorio que o httptest escolheu
	servidor.Listener = listener
	servidor.Start()
	defer servidor.Close()
	t.Logf("harness a servir em %s", servidor.URL)

	// Bloquear ate' prazo ou sinal. Um prazo evita um processo orfao a segurar o porto.
	prazo := 30 * time.Minute
	if v := os.Getenv("HADES_E2E_HARNESS_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			prazo = d
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), prazo)
	defer cancelar()
	<-ctx.Done()
	t.Logf("harness terminou apos %s", prazo)
}

// prepararRepositorioReal cria um repo git e um worktree, ambos reais.
func prepararRepositorioReal(t *testing.T, raiz string) (repo, worktree string, err error) {
	t.Helper()
	repo = filepath.Join(raiz, "repo")
	worktree = filepath.Join(repo, ".agent-worktrees", "missao-real")

	ctx := context.Background()
	git := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
		if saida, e := cmd.CombinedOutput(); e != nil {
			return fmt.Errorf("git %v: %w (%s)", args, e, saida)
		}
		return nil
	}

	// IDEMPOTENTE: uma segunda execucao sobre o mesmo `HADES_E2E_REPO_ROOT` nao pode falhar.
	//
	//	*Um harness que so funciona na primeira vez obriga a apagar estado a mao -- e apagar
	//	estado a mao e' exactamente o que nao se quer num instrumento de medicao.*
	if _, err := os.Stat(filepath.Join(repo, ".git")); err == nil {
		if _, err := os.Stat(worktree); err == nil {
			return repo, worktree, nil
		}
		// Repo existe mas o worktree nao: recriar so' o worktree.
		if err := git("worktree", "add", "-q", "-b", "agent/missao-real", worktree, "HEAD"); err != nil {
			// A branch pode ja' existir sem o worktree; tentar sem `-b`.
			if err2 := git("worktree", "add", "-q", worktree, "agent/missao-real"); err2 != nil {
				return "", "", err
			}
		}
		return repo, worktree, nil
	}
	if err := os.MkdirAll(repo, 0o700); err != nil {
		return "", "", err
	}
	if err := git("init", "-q"); err != nil {
		return "", "", err
	}
	for _, passo := range [][]string{
		{"config", "user.email", "e2e@hades.local"},
		{"config", "user.name", "E2E HADES"},
		{"config", "commit.gpgsign", "false"},
	} {
		if err := git(passo...); err != nil {
			return "", "", err
		}
	}
	// Um ficheiro e um commit, para o repo nao ser vazio.
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		return "", "", err
	}
	if err := git("add", "-A"); err != nil {
		return "", "", err
	}
	if err := git("commit", "-q", "-m", "inicial"); err != nil {
		return "", "", err
	}
	// Worktree REAL, na convencao que o HADES usa.
	if err := git("worktree", "add", "-q", "-b", "agent/missao-real", worktree, "HEAD"); err != nil {
		return "", "", err
	}
	return repo, worktree, nil
}
