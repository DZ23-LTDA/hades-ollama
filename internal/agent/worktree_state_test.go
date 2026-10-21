package agent

// FASE 2 do adendo V3: testes da maquina de estados e do inventor de worktrees.
//
// Os invariantes:
//
//   1. A lista de estados e' FECHADA -- um estado inventado e' recusado.
//   2. As transicoes sao declaradas -- `criado -> removido` e `em_uso -> removido` NAO existem.
//   3. O inventor VE os worktrees externos (criados fora do HADES).
//   4. O worktree principal nunca e' marcado como removivel.
//   5. O inventor e' deterministico e nunca altera o repositorio.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A lista de estados e' fechada.
func TestEstadosDoWorktreeSaoFechados(t *testing.T) {
	for _, e := range EstadosDoWorktree() {
		if !e.Valido() {
			t.Fatalf("estado declarado rejeitado: %q", e)
		}
	}
	if EstadoDoWorktree("quase_pronto").Valido() {
		t.Fatal("estado inventado aceite: *um estado que ninguem pode nomear invalido nao e' um\n" +
			"estado, e' um campo de texto*")
	}
	if len(EstadosDoWorktree()) != 9 {
		t.Fatalf("%d estados; esperados 9", len(EstadosDoWorktree()))
	}
}

// As transicoes proibidas NAO existem.
//
// *Uma transicao que nao existe obriga a quem a quer a perguntar por que a quer.*
func TestTransicoesProibidasSaoRecusadas(t *testing.T) {
	proibidas := []struct {
		de     EstadoDoWorktree
		para   EstadoDoWorktree
		porque string
	}{
		{WorktreeCriado, WorktreeRemovido, "remover o que nunca correu esconde erro de orquestracao"},
		{WorktreeEmUso, WorktreeRemovido, "apagar com agente a escrever e' a receita da perda (E-1226)"},
		{WorktreeCriado, WorktreeIntegrado, "nao se integra o que nunca foi preparado"},
		{WorktreeRemovido, WorktreeEmUso, "removido e' terminal"},
		{WorktreeRemovido, WorktreeCriado, "removido e' terminal"},
	}
	for _, caso := range proibidas {
		sessao := &GitWorktreeSession{Status: string(caso.de)}
		_, _, err := Transitar(sessao, caso.para)
		if !errors.Is(err, ErrTransicaoInvalida) {
			t.Fatalf("%s -> %s: erro = %v; esperado ErrTransicaoInvalida (%s)",
				caso.de, caso.para, err, caso.porque)
		}
		if EstadoDoWorktree(sessao.Status) != caso.de {
			t.Fatalf("%s -> %s: o estado mudou apesar da recusa (%q)", caso.de, caso.para, sessao.Status)
		}
	}
}

// As transicoes declaradas funcionam.
func TestTransicoesValidasFuncionam(t *testing.T) {
	caminho := []EstadoDoWorktree{
		WorktreeCriado, WorktreePreparado, WorktreeEmUso,
		WorktreeSuspenso, WorktreeEmUso, WorktreeIntegrado,
		WorktreeArquivado, WorktreeRemovido,
	}
	sessao := &GitWorktreeSession{}
	for i, destino := range caminho {
		de, para, err := Transitar(sessao, destino)
		if err != nil {
			t.Fatalf("passo %d (%s): %v", i, destino, err)
		}
		if para != destino {
			t.Fatalf("passo %d: devolveu %q; esperado %q", i, para, destino)
		}
		if EstadoDoWorktree(sessao.Status) != destino {
			t.Fatalf("passo %d: Status = %q; esperado %q", i, sessao.Status, destino)
		}
		if i > 0 && de != caminho[i-1] {
			t.Fatalf("passo %d: origem devolvida = %q; esperado %q", i, de, caminho[i-1])
		}
	}
}

// Repetir a mesma transicao e' idempotente.
func TestTransicaoRepetidaEIdempotente(t *testing.T) {
	sessao := &GitWorktreeSession{Status: string(WorktreePreparado)}
	de, para, err := Transitar(sessao, WorktreePreparado)
	if err != nil {
		t.Fatalf("transicao repetida devolveu erro: %v", err)
	}
	if de != WorktreePreparado || para != WorktreePreparado {
		t.Fatalf("de=%q para=%q; esperado ambos preparado", de, para)
	}
}

// Uma sessao sem estado comeca em `criado`.
//
// *Nao tratar o vazio como erro obrigaria todos os chamadores a inicializar o campo.*
func TestSessaoSemEstadoComecaCriado(t *testing.T) {
	sessao := &GitWorktreeSession{}
	de, _, err := Transitar(sessao, WorktreePreparado)
	if err != nil {
		t.Fatalf("sessao sem estado recusada: %v", err)
	}
	if de != WorktreeCriado {
		t.Fatalf("origem = %q; esperado criado", de)
	}
}

// O estado de origem desconhecido e' recusado.
func TestEstadoDeOrigemDesconhecidoERecusado(t *testing.T) {
	sessao := &GitWorktreeSession{Status: "estado-inventado"}
	_, _, err := Transitar(sessao, WorktreePreparado)
	if !errors.Is(err, ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}

// Sessao nula nao rebenta.
func TestTransitarSessaoNulaNaoRebenta(t *testing.T) {
	_, _, err := Transitar(nil, WorktreePreparado)
	if err == nil {
		t.Fatal("sessao nula aceite")
	}
	if !errors.Is(err, ErrTransicaoInvalida) {
		t.Fatalf("erro = %v; esperado ErrTransicaoInvalida", err)
	}
}

// `TransicoesDe` devolve copia, nao a fatia interna.
func TestTransicoesDeDevolveCopia(t *testing.T) {
	a := TransicoesDe(WorktreeEmUso)
	if len(a) == 0 {
		t.Fatal("nenhuma transicao a partir de em_uso")
	}
	a[0] = "corrompido"
	b := TransicoesDe(WorktreeEmUso)
	if b[0] == "corrompido" {
		t.Fatal("TransicoesDe devolveu a fatia interna: *um catalogo que o chamador pode mutar\n" +
			"deixa de ser um catalogo*")
	}
}

// ---------------------------------------------------------------------------------------------
// INVENTOR
// ---------------------------------------------------------------------------------------------

// O inventor ve o worktree principal e os criados pelo HADES.
func TestInventarioVePrincipalEOsDoHades(t *testing.T) {
	repo := repoFixture(t)
	dataRoot := t.TempDir()
	sessao := criarWorktree(t, repo, dataRoot, "missao-inventor")

	lista, err := InventariarWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	if len(lista) < 2 {
		t.Fatalf("%d worktrees; esperados >= 2 (principal + o criado)", len(lista))
	}
	var principais, doHades int
	for _, w := range lista {
		if w.Principal {
			principais++
		}
		if w.CriadoPeloHades {
			doHades++
		}
	}
	if principais != 1 {
		t.Fatalf("%d principais; esperado exatamente 1", principais)
	}
	if doHades < 1 {
		t.Fatal("o worktree criado pelo HADES nao foi reconhecido como tal")
	}
	// E o nosso worktree tem de estar na lista, com o caminho resolvido.
	achou := false
	for _, w := range lista {
		// Comparar caminhos REAIS: no Windows o runner do GitHub tem nome curto 8.3
		// (`RUNNER~1`) e o git devolve o nome longo (`runneradmin`).
		//	*A correccao existia -- `caminhoReal` -- e o teste nao a usava.*
		if caminhoReal(w.CaminhoResolvido) == caminhoReal(sessao.WorktreeDir) {
			achou = true
			if w.Branch != sessao.BranchName {
				t.Fatalf("branch do inventor = %q; esperado %q", w.Branch, sessao.BranchName)
			}
		}
	}
	if !achou {
		t.Fatal("worktree criado ausente do inventor")
	}
}

// O inventor VE um worktree externo criado fora do HADES.
//
// *Um worktree externo que nao se ve' e' um worktree que se apaga por engano.*
func TestInventarioVeWorktreeExterno(t *testing.T) {
	repo := repoFixture(t)
	externo := filepath.Join(t.TempDir(), "worktree-externo")
	if _, err := runRawGit(context.Background(), repo, "worktree", "add", "-b", "externo/um", externo, "HEAD"); err != nil {
		t.Fatalf("criar worktree externo: %v", err)
	}
	t.Cleanup(func() {
		_, _ = runRawGit(context.Background(), repo, "worktree", "remove", "--force", externo)
	})

	lista, err := InventariarWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("inventariar: %v", err)
	}
	achou := false
	for _, w := range lista {
		if caminhoReal(w.CaminhoResolvido) == caminhoReal(externo) {
			achou = true
			if w.CriadoPeloHades {
				t.Fatal("worktree externo marcado como criado pelo HADES")
			}
			if w.Principal {
				t.Fatal("worktree externo marcado como principal")
			}
		}
	}
	if !achou {
		t.Fatal("worktree externo NAO foi descoberto: *um worktree externo que nao se ve' e' um\n" +
			"worktree que se apaga por engano*")
	}
}

// O inventor aceita um repo cujo caminho tem symlinks.
//
// No macOS `/var` e' symlink de `/private/var`, e o `git` devolve o caminho RESOLVIDO. Sem
// comparar caminhos reais, um repositorio valido era recusado no CI do macOS.
//
//	*Um caminho absoluto nao e' o mesmo que um caminho real.*
func TestInventarioAceitaRepoComSymlinkNoCaminho(t *testing.T) {
	repo := repoFixture(t)
	base := t.TempDir()
	link := filepath.Join(base, "atalho-para-repo")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("symlink indisponivel neste ambiente: %v", err)
	}

	lista, err := InventariarWorktrees(context.Background(), link)
	if err != nil {
		t.Fatalf("repo acessivel por symlink foi recusado: %v\n"+
			"*um caminho absoluto nao e' o mesmo que um caminho real*", err)
	}
	if len(lista) == 0 {
		t.Fatal("inventor vazio para repo valido via symlink")
	}
	if caminhoReal(link) != caminhoReal(repo) {
		t.Fatalf("caminhoReal nao resolveu o symlink: %q vs %q", caminhoReal(link), caminhoReal(repo))
	}
}

// caminhoReal devolve o caminho limpo quando o alvo nao existe.
func TestCaminhoRealComAlvoInexistente(t *testing.T) {
	inexistente := filepath.Join(t.TempDir(), "nao-existe")
	if got := caminhoReal(inexistente); got != filepath.Clean(inexistente) {
		t.Fatalf("caminhoReal(%q) = %q; esperado o caminho limpo", inexistente, got)
	}
}

// O inventor e' deterministico.
func TestInventarioDeterministico(t *testing.T) {
	repo := repoFixture(t)
	_ = criarWorktree(t, repo, t.TempDir(), "missao-det")

	a, err := InventariarWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("1a: %v", err)
	}
	b, err := InventariarWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("2a: %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("tamanhos diferentes: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("entrada %d difere entre chamadas", i)
		}
	}
}

// O inventor recusa repo vazio.
func TestInventarioRecusaRepoVazio(t *testing.T) {
	if _, err := InventariarWorktrees(context.Background(), ""); err == nil {
		t.Fatal("repo vazio aceite")
	}
}

// Um diretorio que nao e' repositorio devolve erro, nao lista vazia.
//
// *Uma lista vazia diria "nao ha' worktrees"; o erro diz "isto nao e' um repositorio" -- e sao
// coisas diferentes.*
func TestInventarioEmDiretorioNaoGitDaErro(t *testing.T) {
	dir := t.TempDir()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponivel")
	}
	if _, err := InventariarWorktrees(context.Background(), dir); err == nil {
		t.Fatal("diretorio nao-git devolveu lista em vez de erro")
	}
}

// O resumo conta o que interessa.
func TestResumoDoInventario(t *testing.T) {
	lista := []WorktreeDescoberto{
		{CaminhoResolvido: "/a", Commit: "", Principal: true},
		{CaminhoResolvido: "/b", CriadoPeloHades: true},
		{CaminhoResolvido: "/c", CriadoPeloHades: true},
	}
	resumo := ResumoDoInventario(lista)
	for _, palavra := range []string{"3 worktree", "2 do HADES", "1 externo"} {
		if !strings.Contains(resumo, palavra) {
			t.Fatalf("resumo sem %q: %q", palavra, resumo)
		}
	}
}
