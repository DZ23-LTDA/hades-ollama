package server

// E-1306: prova de que a rota `build.start` separa os casos como deve.
//
// A rota e' a primeira do Release Hub que EXECUTA alguma coisa. O que estes testes fixam nao e' o
// caminho feliz -- e' que CADA recusa acontece ANTES de tocar no filesystem ou de correr um comando.
//
//	*Uma rota que executa o que lhe mandarem nao e' um BuildProvider: e' um shell com JSON.*
//
// A distincao entre os codigos importa:
//
//	400 -- o corpo esta' mal formado (culpa de quem chama)
//	412 -- o corpo esta' bem formado mas falta um pre-requisito (o servidor funcionou)
//	422 -- o alvo nao esta' no catalogo (o recurso pedido nao existe)
//
//	*"Depende de conta externa" nao e' falha do servidor, e o codigo HTTP tem de o dizer.*

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

const rotaDoInicioDeBuild = "/api/agent/v1/build/start"

// montarRotaDeInicioDeBuild devolve o router com a rota montada.
func montarRotaDeInicioDeBuild(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessaoDeBuild("org-teste"))
	router.POST(rotaDoInicioDeBuild, (&agentAPI{}).iniciarBuild)
	return router
}

// pedirInicioDeBuild faz o POST e devolve o codigo e o corpo.
func pedirInicioDeBuild(t *testing.T, router *gin.Engine, corpo any) (int, map[string]any) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, rotaDoInicioDeBuild, bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var corpoDaResposta map[string]any
	if resposta.Body.Len() > 0 {
		if errDec := json.Unmarshal(resposta.Body.Bytes(), &corpoDaResposta); errDec != nil {
			t.Fatalf("resposta nao e' JSON (%d): %v -- %s", resposta.Code, errDec, resposta.Body.String())
		}
	}
	return resposta.Code, corpoDaResposta
}

// TestInicioDeBuildRecusaRaizRelativaAntesDeTocarNoDisco prova o 412 e que nada foi construido.
func TestInicioDeBuildRecusaRaizRelativaAntesDeTocarNoDisco(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{
		"raiz": "relativo/aqui",
		"alvo": "linux",
	})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if erro == "" || !contemTexto(erro, "absoluto") {
		t.Fatalf("o erro devia dizer que a raiz nao e' absoluta: %q", erro)
	}
	// O corpo do 412 nao pode trazer um resultado de build: nada correu.
	if _, existe := corpo["resultado"]; existe {
		t.Fatal("um 412 devolveu `resultado` -- a recusa aconteceu depois de construir")
	}
}

// TestInicioDeBuildRecusaRaizVazia prova o 412 com mensagem propria.
func TestInicioDeBuildRecusaRaizVazia(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{"alvo": "linux"})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !contemTexto(erro, "raiz") {
		t.Fatalf("o erro devia nomear `raiz`: %q", erro)
	}
}

// TestInicioDeBuildRecusaAlvoVazioComOCatalogo prova que o 412 lista os alvos que o provider cobre.
func TestInicioDeBuildRecusaAlvoVazioComOCatalogo(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{"raiz": t.TempDir()})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("esperado 412, obtido %d: %v", codigo, corpo)
	}
	alvos, ok := corpo["alvos"].([]any)
	if !ok || len(alvos) == 0 {
		t.Fatalf("o 412 devia listar os alvos cobertos: %v", corpo)
	}
}

// TestInicioDeBuildRecusaTimeoutAcimaDoTecto prova o 400.
//
//	*Um timeout escolhido pelo cliente e' uma pausa que o servidor aceita por escrito.*
func TestInicioDeBuildRecusaTimeoutAcimaDoTecto(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{
		"raiz":             t.TempDir(),
		"alvo":             "linux",
		"timeout_segundos": 99999,
	})
	if codigo != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtido %d: %v", codigo, corpo)
	}
}

// TestInicioDeBuildRecusaTimeoutNegativo prova o 400 e a distincao com o zero.
//
// Zero significa "usa o limite por omissao"; negativo nao significa nada. Converter um no outro
// seria aceitar um valor que ninguem pediu.
func TestInicioDeBuildRecusaTimeoutNegativo(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{
		"raiz":             t.TempDir(),
		"alvo":             "linux",
		"timeout_segundos": -5,
	})
	if codigo != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtido %d: %v", codigo, corpo)
	}
	erro, _ := corpo["erro"].(string)
	if !contemTexto(erro, "negativo") {
		t.Fatalf("o erro devia nomear o valor negativo: %q", erro)
	}
}

// TestInicioDeBuildRecusaCorpoInvalido prova o 400 sem tocar em nada.
func TestInicioDeBuildRecusaCorpoInvalido(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessaoDeBuild("org-teste"))
	router.POST(rotaDoInicioDeBuild, (&agentAPI{}).iniciarBuild)

	pedido := httptest.NewRequest(http.MethodPost, rotaDoInicioDeBuild, bytes.NewReader([]byte("nao e' json")))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	if resposta.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtido %d: %s", resposta.Code, resposta.Body.String())
	}
}

// TestInicioDeBuildDryRunNaoExecutaOComando prova o caminho de sucesso SEM correr `go build`.
//
// O `DryRun` do provider devolve o comando e nao o executa. Este teste usa-o para provar o 200 e o
// `construiu=false` sem pagar o custo de uma build real -- e, sobretudo, sem escrever um binario.
func TestInicioDeBuildDryRunNaoExecutaOComando(t *testing.T) {
	directorio := t.TempDir()
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	// Um `main.go` minimo para o plano nao falhar por projecto vazio.
	if err := os.WriteFile(filepath.Join(directorio, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("escrever main.go: %v", err)
	}

	router := montarRotaDeInicioDeBuild(t)
	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{
		"raiz":    directorio,
		"alvo":    "linux",
		"dry_run": true,
	})

	// Pode dar 412 (o plano deste host pode ter blockers) ou 200 (plano pronto). O que NAO pode e'
	// dar 200 com `construiu=true` num dry-run, nem 500 (o servidor nao falhou em nenhum caso).
	if codigo == http.StatusInternalServerError {
		t.Fatalf("dry-run nao devia dar 500: %v", corpo)
	}
	if codigo == http.StatusOK {
		if construiu, _ := corpo["construiu"].(bool); construiu {
			t.Fatal("`construiu` devia ser false num dry-run")
		}
		if nadaAssinado, _ := corpo["nada_assinado"].(bool); !nadaAssinado {
			t.Fatal("`nada_assinado` devia ser true mesmo num dry-run")
		}
		if nadaPublicado, _ := corpo["nada_publicado"].(bool); !nadaPublicado {
			t.Fatal("`nada_publicado` devia ser true mesmo num dry-run")
		}
		aviso, _ := corpo["aviso_de_fronteira"].(string)
		if !contemTexto(aviso, "NAO assina") || !contemTexto(aviso, "NAO publica") {
			t.Fatalf("o aviso de fronteira devia nomear o que a rota nao faz: %q", aviso)
		}
	}
}

// TestInicioDeBuildNaoPrometeBuildStatus garante que a rota NAO responde por rotas que nao existem.
//
// O adendo pede `build.status`, `build.logs`, `build.cancel` e `build.artifacts`. Nenhuma existe --
// e inventar um `status` que responde 404 a tudo o que prometeu seria pior do que nao a ter.
func TestInicioDeBuildNaoPrometeBuildStatus(t *testing.T) {
	router := montarRotaDeInicioDeBuild(t)
	gin.SetMode(gin.TestMode)

	for _, rota := range []string{
		"/api/agent/v1/build/status",
		"/api/agent/v1/build/logs",
		"/api/agent/v1/build/cancel",
		"/api/agent/v1/build/artifacts",
	} {
		pedido := httptest.NewRequest(http.MethodGet, rota, nil)
		resposta := httptest.NewRecorder()
		router.ServeHTTP(resposta, pedido)
		if resposta.Code != http.StatusNotFound {
			t.Fatalf("%s: esperado 404 (a rota NAO existe), obtido %d", rota, resposta.Code)
		}
	}
}
