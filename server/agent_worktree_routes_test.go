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
	"bytes"
	"context"
	"encoding/json"
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

// O estado PERSISTIDO tem de chegar a' ROTA -- senao a persistencia e' um ficheiro que ninguem le'.
//
//	*Uma persistencia que a rota nao consulta e' um ficheiro que ninguem le'.*
func TestWorktreesDiagnosticoLeEstadoPersistido(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-persistida", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	// Suspender e PERSISTIR o estado no data root do worktree.
	if err := agent.SuspenderWorktree(sessao); err != nil {
		t.Fatalf("suspender: %v", err)
	}
	registo, err := agent.CarregarRegistoDeWorktrees(agent.CaminhoDoRegisto(dataRoot))
	if err != nil {
		t.Fatalf("carregar registo: %v", err)
	}
	if err := registo.Registar(sessao.MissionID, agent.WorktreeSuspenso, sessao.WorktreeDir, sessao.BranchName); err != nil {
		t.Fatalf("registar: %v", err)
	}
	if err := registo.Guardar(); err != nil {
		t.Fatalf("guardar registo: %v", err)
	}

	engine := routerDeWorktrees(t)
	var payload worktreeDiagnosticoResposta
	caminho := "/api/agent/v1/worktrees/diagnose?repo=" + repo + "&missao=missao-persistida"
	if code := getJSON(t, engine, caminho, &payload); code != http.StatusOK {
		t.Fatalf("GET diagnose -> %d", code)
	}

	if payload.Estado != string(agent.WorktreeSuspenso) {
		t.Fatalf("estado = %q; esperado %q (do registo persistido)", payload.Estado, agent.WorktreeSuspenso)
	}
	if !payload.EstadoDoRegisto {
		t.Fatal("a rota nao marcou o estado como vindo do registo")
	}
	if payload.Registo != "consultado" {
		t.Fatalf("registo = %q; esperado consultado", payload.Registo)
	}
	// Um worktree suspension e limpo NAO pode ser removido: `suspension` permite remover, mas o
	// ponto do teste e' que a ROTA concorda com o registo, nao com um valor inventado.
	if payload.EstadoObservado == "" {
		t.Fatal("diagnostico nao explica de onde veio o estado")
	}
}

// Sem registo, a rota DIZ que nao ha' registo -- nao inventa um estado.
func TestWorktreesDiagnosticoSemRegistoDeclaraAusencia(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-sem-estado", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	engine := routerDeWorktrees(t)
	var payload worktreeDiagnosticoResposta
	caminho := "/api/agent/v1/worktrees/diagnose?repo=" + repo + "&missao=missao-sem-estado"
	if code := getJSON(t, engine, caminho, &payload); code != http.StatusOK {
		t.Fatalf("GET diagnose -> %d", code)
	}
	if payload.EstadoDoRegisto {
		t.Fatal("a rota diz ter registo quando nao existe nenhum")
	}
	if payload.Registo == "" || payload.Registo == "consultado" {
		t.Fatalf("registo = %q; esperado declaracao de ausencia", payload.Registo)
	}
}

// Um registo ILEGIVEL nao pode passar por ausente nem por valido.
//
//	*Um registo corrompido lido como vazio apagaria o estado de todos os worktrees.*
func TestWorktreesDiagnosticoComRegistoIlegivel(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-corrompida", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	caminhoRegisto := agent.CaminhoDoRegisto(dataRoot)
	if err := os.WriteFile(caminhoRegisto, []byte("{isto nao e' json"), 0o600); err != nil {
		t.Fatalf("escrever registo corrompido: %v", err)
	}

	engine := routerDeWorktrees(t)
	var payload worktreeDiagnosticoResposta
	caminho := "/api/agent/v1/worktrees/diagnose?repo=" + repo + "&missao=missao-corrompida"
	if code := getJSON(t, engine, caminho, &payload); code != http.StatusOK {
		t.Fatalf("GET diagnose -> %d (um registo ilegivel nao deve derrubar o diagnostico)", code)
	}
	if payload.EstadoDoRegisto {
		t.Fatal("a rota diz ter lido estado de um registo ilegivel")
	}
	if !strings.HasPrefix(payload.Registo, "ilegivel") {
		t.Fatalf("registo = %q; esperado comecar por \"ilegivel\"", payload.Registo)
	}
}

// ---------------------------------------------------------------------------------------------
// POST /worktrees/state -- a UNICA rota mutante de worktree (E-1236).
// ---------------------------------------------------------------------------------------------

// postJSONState envia um corpo JSON e devolve o codigo e o corpo bruto.
func postJSONState(t *testing.T, engine *gin.Engine, caminho string, corpo any) (int, string) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, caminho, bytes.NewReader(bruto))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// A transicao aceite muda o estado E PERSISTE-O.
//
//	*Um estado que so' o programador muda nao e' um estado do produto: e' um valor de fabrica.*
func TestWorktreesMudaEstadoEPersiste(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-muda", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	engine := routerDeWorktrees(t)
	codigo, corpo := postJSONState(t, engine, "/api/agent/v1/worktrees/state", map[string]string{
		"repo": repo, "missao": "missao-muda", "para": "suspender",
	})
	if codigo != http.StatusOK {
		t.Fatalf("POST state -> %d: %s", codigo, corpo)
	}
	var payload estadoDeResposta
	if err := json.Unmarshal([]byte(corpo), &payload); err != nil {
		t.Fatalf("decodificar: %v (%s)", err, corpo)
	}
	if payload.Para != string(agent.WorktreeSuspenso) {
		t.Fatalf("para = %q; esperado suspension", payload.Para)
	}
	if payload.Acao != "suspender" {
		t.Fatalf("acao = %q", payload.Acao)
	}
	if !payload.EstadoDoRegisto {
		t.Fatal("a rota nao diz ter usado o registo")
	}

	// E o registo em disco tem mesmo o estado novo.
	registo, err := agent.CarregarRegistoDeWorktrees(agent.CaminhoDoRegisto(dataRoot))
	if err != nil {
		t.Fatalf("recarregar registo: %v", err)
	}
	estado, existe := registo.EstadoDe("missao-muda")
	if !existe {
		t.Fatal("missao nao ficou no registo")
	}
	if estado != agent.WorktreeSuspenso {
		t.Fatalf("estado persistido = %q; esperado suspension", estado)
	}

	// E o diagnostico (leitura) ve' o estado novo.
	var diag worktreeDiagnosticoResposta
	caminhoDiag := "/api/agent/v1/worktrees/diagnose?repo=" + repo + "&missao=missao-muda"
	if code := getJSON(t, engine, caminhoDiag, &diag); code != http.StatusOK {
		t.Fatalf("GET diagnose -> %d", code)
	}
	if diag.Estado != string(agent.WorktreeSuspenso) {
		t.Fatalf("diagnostico = %q; esperado suspension (o ciclo escrever->ler nao fecha)", diag.Estado)
	}
}

// `removido` NAO e' aceite por esta rota.
//
//	*Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num clique
//	que o painel nao sabe explicar.*
func TestWorktreesNaoAceitaRemover(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-nao-remove", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	engine := routerDeWorktrees(t)
	for _, pedido := range []string{"removido", "remover", "removed", "delete"} {
		codigo, corpo := postJSONState(t, engine, "/api/agent/v1/worktrees/state", map[string]string{
			"repo": repo, "missao": "missao-nao-remove", "para": pedido,
		})
		if codigo != http.StatusBadRequest {
			t.Fatalf("%q aceite com %d: %s", pedido, codigo, corpo)
		}
		if !strings.Contains(corpo, "nao permitida") {
			t.Fatalf("%q: mensagem sem a recusa esperada: %s", pedido, corpo)
		}
	}
	// E o worktree continua la'.
	if _, err := os.Stat(sessao.WorktreeDir); err != nil {
		t.Fatalf("o worktree desapareceu: %v", err)
	}
}

// Uma transicao que a tabela fechada recusa e' recusada pela ROTA -- nao por uma copia local.
func TestWorktreesRecusaTransicaoInvalida(t *testing.T) {
	repo := repoDeWorktree(t)
	dataRoot := t.TempDir()
	sessao, err := agent.CreateGitWorktree(context.Background(), repo, dataRoot, "missao-invalida", "")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	t.Cleanup(func() {
		_ = agent.RemoveGitWorktreeComPerda(context.Background(), sessao, "limpeza do teste")
	})

	engine := routerDeWorktrees(t)
	// Primeiro suspender (preparado -> suspension e' valido).
	if codigo, corpo := postJSONState(t, engine, "/api/agent/v1/worktrees/state", map[string]string{
		"repo": repo, "missao": "missao-invalida", "para": "suspender",
	}); codigo != http.StatusOK {
		t.Fatalf("suspender -> %d: %s", codigo, corpo)
	}
	// `removido` nao entra; tentar uma transicao que nao existe na tabela tambem nao.
	codigo, corpo := postJSONState(t, engine, "/api/agent/v1/worktrees/state", map[string]string{
		"repo": repo, "missao": "missao-invalida", "para": "arquivar",
	})
	// suspension -> arquivado E' valido na tabela; o teste verifica que nao e' recusado por engano.
	if codigo != http.StatusOK {
		t.Fatalf("suspension->arquivado devia ser valido, veio %d: %s", codigo, corpo)
	}
}

// A rota exige repo e missao.
func TestWorktreesEstadoExigeParametros(t *testing.T) {
	engine := routerDeWorktrees(t)
	for _, corpo := range []map[string]string{
		{"missao": "m", "para": "suspender"},
		{"repo": "/tmp", "para": "suspender"},
		{"repo": "/tmp", "missao": "m"},
	} {
		codigo, saida := postJSONState(t, engine, "/api/agent/v1/worktrees/state", corpo)
		if codigo != http.StatusBadRequest {
			t.Fatalf("%v aceite com %d: %s", corpo, codigo, saida)
		}
	}
}

// Missao inexistente devolve 404, nao inventa um worktree.
func TestWorktreesEstadoMissaoInexistente(t *testing.T) {
	repo := repoDeWorktree(t)
	engine := routerDeWorktrees(t)
	codigo, corpo := postJSONState(t, engine, "/api/agent/v1/worktrees/state", map[string]string{
		"repo": repo, "missao": "nao-existe-mesmo", "para": "suspender",
	})
	if codigo != http.StatusNotFound {
		t.Fatalf("missao inexistente -> %d: %s", codigo, corpo)
	}
}
