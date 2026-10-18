package server

// FASE 28.6/28.7: testes da rota do plano de release.
//
// O que estes testes travam:
//
//  1. a rota responde e devolve a forma completa;
//  2. `somente_leitura`, `nada_instalado` e `nada_construido` sao TRUE -- a rota nao muta;
//  3. caminho inexistente da' 404; ficheiro da' 400; vazio da' 400;
//  4. projection irreconhecivel da' 422 (nao 404 nem 500);
//  5. `profundidade` invalida da' 400;
//  6. a rota NAO aceita `alvo` -- um projection Go nao produz plano de Windows instalador por
//     parametro;
//  7. `blockers` e `motivos` no topo coincidem com o plano (o painel nao precisa de somar);
//  8. o repo HADES real produz plano pela rota (auto-teste);
//  9. a rota nao escreve nada no projection.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// helper para criar o router com a rota registada.
func routerDePlanoDeBuild(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&agentAPI{}).register(router)
	return router
}

// helper para fazer o GET e devolver o corpo descodificado.
func pedirPlano(t *testing.T, router *gin.Engine, query string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/build/plan?"+query, nil)
	router.ServeHTTP(rec, req)
	var corpo map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
			t.Fatalf("resposta nao e' JSON: %v (corpo: %s)", err, rec.Body.String())
		}
	}
	return rec.Code, corpo
}

func escreverNoProjecto(t *testing.T, raiz, rel, conteudo string) {
	t.Helper()
	caminho := filepath.Join(raiz, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
}

// A rota responde e devolve a forma completa, declarando que so' le'.
func TestRotaPlanoProjetoGo(t *testing.T) {
	raiz := t.TempDir()
	escreverNoProjecto(t, raiz, "go.mod", "module x\n\ngo 1.24\n")

	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(raiz))
	if codigo != http.StatusOK {
		t.Fatalf("status = %d; corpo = %v", codigo, corpo)
	}
	for _, campo := range []string{"somente_leitura", "nada_instalado", "nada_construido"} {
		v, existe := corpo[campo]
		if !existe {
			t.Fatalf("campo %q ausente da resposta", campo)
		}
		if v != true {
			t.Fatalf("%q = %v; esperado true (a rota nao muta nada)", campo, v)
		}
	}
	if _, existe := corpo["plano"]; !existe {
		t.Fatal("campo `plano` ausente")
	}
	if _, existe := corpo["aviso_de_fronteira"]; !existe {
		t.Fatal("campo `aviso_de_fronteira` ausente")
	}
}

// A rota NAO aceita `alvo`: um projection Go produz os alvos do projection, nao os que o cliente pedir.
func TestRotaPlanoIgnoraAlvoNoQuery(t *testing.T) {
	raiz := t.TempDir()
	escreverNoProjecto(t, raiz, "go.mod", "module x\n")

	// Pede um alvo que o projection NAO declara (`android` num repo Go).
	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t),
		"repo="+url.QueryEscape(raiz)+"&alvo=android")
	if codigo != http.StatusOK {
		t.Fatalf("status = %d", codigo)
	}
	plano, _ := corpo["plano"].(map[string]any)
	alvos, _ := plano["alvos"].([]any)
	for _, a := range alvos {
		m, _ := a.(map[string]any)
		if m["alvo"] == "android" {
			t.Fatalf("`alvo=android` foi honrado num projection Go: %v", alvos)
		}
	}
}

// Caminho inexistente: 404.
func TestRotaPlanoRepoInexistente(t *testing.T) {
	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t),
		"repo="+url.QueryEscape(filepath.Join(t.TempDir(), "nao-existe")))
	if codigo != http.StatusNotFound {
		t.Fatalf("status = %d; esperado 404 (corpo %v)", codigo, corpo)
	}
}

// `repo` em falta: 400.
func TestRotaPlanoSemRepo(t *testing.T) {
	codigo, _ := pedirPlano(t, routerDePlanoDeBuild(t), "")
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400", codigo)
	}
}

// `repo` que e' ficheiro: 400.
func TestRotaPlanoRepoQueEFicheiro(t *testing.T) {
	raiz := t.TempDir()
	ficheiro := filepath.Join(raiz, "x.txt")
	if err := os.WriteFile(ficheiro, []byte("x"), 0o600); err != nil {
		t.Fatalf("criar: %v", err)
	}
	codigo, _ := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(ficheiro))
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400", codigo)
	}
}

// Projection irreconhecivel: 422, nao 404 nem 500.
//
// *Um erro que diz "nao sei o que isto e'" nao e' o mesmo que "isto nao existe".*
func TestRotaPlanoProjetoIrreconhecivel(t *testing.T) {
	raiz := t.TempDir()
	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(raiz))
	if codigo != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d; esperado 422 (corpo %v)", codigo, corpo)
	}
}

// `profundidade` invalida: 400.
func TestRotaPlanoProfundidadeInvalida(t *testing.T) {
	raiz := t.TempDir()
	escreverNoProjecto(t, raiz, "go.mod", "module x\n")
	for _, valor := range []string{"0", "99", "abc", "-1", "1.5"} {
		codigo, _ := pedirPlano(t, routerDePlanoDeBuild(t),
			"repo="+url.QueryEscape(raiz)+"&profundidade="+valor)
		if codigo != http.StatusBadRequest {
			t.Fatalf("profundidade=%q: status = %d; esperado 400", valor, codigo)
		}
	}
}

// `blockers` e `motivos` no topo coincidem com o plano -- o painel nao precisa de somar.
func TestRotaPlanoBlockersEMotivosNoTopo(t *testing.T) {
	raiz := t.TempDir()
	escreverNoProjecto(t, raiz, "go.mod", "module x\n")

	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(raiz))
	if codigo != http.StatusOK {
		t.Fatalf("status = %d", codigo)
	}
	for _, campo := range []string{"blockers", "motivos"} {
		if _, existe := corpo[campo]; !existe {
			t.Fatalf("campo %q ausente do topo", campo)
		}
	}
	// Se o plano tem alvos bloqueados, `blockers` nao pode estar vazio.
	plano, _ := corpo["plano"].(map[string]any)
	bloqueados, _ := plano["alvos_bloqueados"].([]any)
	blockers, _ := corpo["blockers"].([]any)
	if len(bloqueados) > 0 && len(blockers) == 0 {
		t.Fatalf("plano tem %d alvo(s) bloqueado(s) mas `blockers` esta' vazio", len(bloqueados))
	}
}

// A rota NAO escreve nada no projection.
func TestRotaPlanoNaoEscreveNoProjecto(t *testing.T) {
	raiz := t.TempDir()
	escreverNoProjecto(t, raiz, "go.mod", "module x\n")

	listar := func() []string {
		var nomes []string
		_ = filepath.WalkDir(raiz, func(c string, e os.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				rel, _ := filepath.Rel(raiz, c)
				nomes = append(nomes, rel)
			}
			return nil
		})
		return nomes
	}
	antes := listar()
	if codigo, _ := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(raiz)); codigo != http.StatusOK {
		t.Fatalf("status = %d", codigo)
	}
	depois := listar()
	if len(antes) != len(depois) {
		t.Fatalf("a rota alterou o projection: antes=%v depois=%v", antes, depois)
	}
}

// Auto-teste: o proprio repo HADES pela rota.
func TestRotaPlanoDoRepoHades(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join(".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.mod")); err != nil {
		t.Skipf("nao estou dentro do repo HADES (%v)", err)
	}
	codigo, corpo := pedirPlano(t, routerDePlanoDeBuild(t), "repo="+url.QueryEscape(raiz))
	if codigo != http.StatusOK {
		t.Fatalf("status = %d; corpo = %v", codigo, corpo)
	}
	plano, _ := corpo["plano"].(map[string]any)
	alvos, _ := plano["alvos"].([]any)
	if len(alvos) == 0 {
		t.Fatal("repo HADES sem alvos pela rota")
	}
	var nomes []string
	for _, a := range alvos {
		m, _ := a.(map[string]any)
		if s, ok := m["alvo"].(string); ok {
			nomes = append(nomes, s)
		}
	}
	t.Logf("repo HADES pela rota: alvos=%v", nomes)
}
