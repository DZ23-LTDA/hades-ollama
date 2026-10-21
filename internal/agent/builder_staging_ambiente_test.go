package agent

// Adendo 2026-10-18, item B -- as fronteiras de execucao do builder de staging.
//
// O achado Semgrep `dangerous-exec-command` apontou `builder_staging.go` e a auditoria
// confirmou DOIS defeitos reais que o scanner nao nomeia:
//
//   1. `exec.CommandContext(ctx, "go", ...)` resolvia `go` pelo PATH do OPERADOR;
//   2. `compilar.Env = os.Environ()` entregava ao filho TODAS as variaveis do operador.
//
// Estes testes recusam aceitar qualquer um dos dois de volta. *Um teste que so' verifica que
// o build funciona nao distingue "funciona" de "funciona com o ambiente de quem lancou".*

import (
	"strings"
	"testing"
)

// TestAmbienteGoNaoHerdaCredenciaisDoOperador e' o discriminador central do item B.
func TestAmbienteGoNaoHerdaCredenciaisDoOperador(t *testing.T) {
	// Marcadores que `governedCLIKeyLooksSecret` reconhece: token, chave, senha.
	for _, nome := range []string{
		"HADES_STAGING_TEST_TOKEN",
		"HADES_STAGING_TEST_API_KEY",
		"HADES_STAGING_TEST_PASSWORD",
		"GITHUB_TOKEN",
		"OPENAI_API_KEY",
	} {
		t.Setenv(nome, "valor-que-nao-pode-chegar-ao-filho")
	}
	// Uma variavel inofensiva para provar que a lista nao e' vazia por acidente.
	t.Setenv("LANG", "pt_BR.UTF-8")

	env := ambienteGoGovernado("/tmp/cache-de-teste")

	junto := strings.Join(env, "\n")
	for _, proibido := range []string{
		"HADES_STAGING_TEST_TOKEN",
		"HADES_STAGING_TEST_API_KEY",
		"HADES_STAGING_TEST_PASSWORD",
		"GITHUB_TOKEN",
		"OPENAI_API_KEY",
	} {
		if strings.Contains(junto, proibido) {
			t.Fatalf("a variavel %q chegou ao filho do build -- *um filho nao deve herdar o que "+
				"nao usa: herdar e' a forma mais silenciosa de vazar*\nambiente=%v", proibido, env)
		}
	}

	// E o que o build PRECISA tem de estar la'.
	achouCache := false
	for _, linha := range env {
		if linha == "GOCACHE=/tmp/cache-de-teste" {
			achouCache = true
		}
	}
	if !achouCache {
		t.Fatalf("o GOCACHE proprio tem de estar no ambiente; veio %v", env)
	}
	// A variavel inofensiva da lista de permissao continua presente.
	if !strings.Contains(junto, "LANG=pt_BR.UTF-8") {
		t.Fatalf("uma variavel da lista de permissao (LANG) devia passar; veio %v", env)
	}
}

// TestAmbienteGoNaoHerdaGOROOTNemGOPATH: herdar o GOROOT de quem INICIOU o processo pode
// apontar para uma instalacao que o filho nao tem. *Deixar o `go` descobrir a propria raiz e'
// mais correcto do que herdar uma que talvez nao exista.*
func TestAmbienteGoNaoHerdaGOROOTNemGOPATH(t *testing.T) {
	t.Setenv("GOROOT", "/goroot-do-operador")
	t.Setenv("GOPATH", "/gopath-do-operador")
	env := ambienteGoGovernado("/tmp/c2")
	junto := strings.Join(env, "\n")
	for _, proibido := range []string{"GOROOT=/goroot-do-operador", "GOPATH=/gopath-do-operador"} {
		if strings.Contains(junto, proibido) {
			t.Fatalf("%q nao pode ser herdado; ambiente=%v", proibido, env)
		}
	}
}
