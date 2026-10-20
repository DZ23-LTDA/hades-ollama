package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// montarRotaDoVault cria um router MÍNIMO com o middleware de sessão simulado (o `authMiddleware`
// real faz exactamente `c.Set("agent.user"/"agent.organization")` depois de verificar o token) e as
// duas rotas reais. O objectivo é exercer handler -> serviço -> disco, não o auth.
func montarRotaDoVault(t *testing.T, utilizador, organizacao string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &agentAPI{}
	grupo := router.Group("/api/agent/v1")
	grupo.Use(func(c *gin.Context) {
		if utilizador != "" {
			c.Set("agent.user", agent.User{ID: utilizador})
		}
		if organizacao != "" {
			c.Set("agent.organization", agent.Organization{ID: organizacao})
		}
		c.Next()
	})
	grupo.POST("/memory/vault/append", api.appendMemoriaNoVault)
	grupo.GET("/memory/vault/read", api.lerMemoriaDoVault)
	return router
}

func prepararVaultDeTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	t.Setenv("HADES_VAULT_ROOT", raiz)
	t.Setenv("HADES_VAULT_ACL", `[{"identidade":"user-permitido","escopo":"TENANT","permissoes":["ler","escrever"]}]`)
	return raiz
}

func pedirAppend(t *testing.T, router *gin.Engine, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/memory/vault/append", strings.NewReader(corpo))
	pedido.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, pedido)
	return rec
}

func pedirLeitura(t *testing.T, router *gin.Engine, relativo string) *httptest.ResponseRecorder {
	t.Helper()
	pedido := httptest.NewRequest(http.MethodGet, "/api/agent/v1/memory/vault/read?relativo="+relativo, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, pedido)
	return rec
}

// NEGATIVO: sem sessão, 401 e NENHUM ficheiro criado.
func TestHTTPAppendSemSessaoNaoEscreve(t *testing.T) {
	raiz := prepararVaultDeTeste(t)
	router := montarRotaDoVault(t, "", "")

	rec := pedirAppend(t, router, `{"relativo":"notas/a.md","conteudo":"x","rotulo":"FACT_VERIFIED","fonte":"teste"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401, veio %d (%s)", rec.Code, rec.Body.String())
	}
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entradas) != 0 {
		t.Fatalf("a recusa criou %d entrada(s) no vault", len(entradas))
	}
}

// NEGATIVO: identidade fora da ACL, 403 e nenhum ficheiro.
func TestHTTPAppendSemConcessaoNaoEscreve(t *testing.T) {
	raiz := prepararVaultDeTeste(t)
	router := montarRotaDoVault(t, "intruso", "org-1")

	rec := pedirAppend(t, router, `{"relativo":"notas/a.md","conteudo":"x","rotulo":"FACT_VERIFIED","fonte":"teste"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperava 403, veio %d (%s)", rec.Code, rec.Body.String())
	}
	entradas, _ := os.ReadDir(raiz)
	if len(entradas) != 0 {
		t.Fatalf("a recusa criou %d entrada(s)", len(entradas))
	}
}

// NEGATIVO: corpo que tenta impor identidade -> 400 e nenhum ficheiro.
func TestHTTPAppendComIdentidadeNoCorpoERecusado(t *testing.T) {
	raiz := prepararVaultDeTeste(t)
	router := montarRotaDoVault(t, "user-permitido", "org-1")

	rec := pedirAppend(t, router, `{"relativo":"notas/a.md","conteudo":"x","identidade":"user-permitido"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d (%s)", rec.Code, rec.Body.String())
	}
	entradas, _ := os.ReadDir(raiz)
	if len(entradas) != 0 {
		t.Fatalf("o corpo recusado criou %d entrada(s)", len(entradas))
	}
}

// POSITIVO ponta a ponta no processo: POST grava, GET devolve o mesmo conteudo com fonte.
func TestHTTPAppendELeituraDevolvemOMesmoConteudo(t *testing.T) {
	raiz := prepararVaultDeTeste(t)
	router := montarRotaDoVault(t, "user-permitido", "org-1")

	rec := pedirAppend(t, router, `{"relativo":"notas/uma.md","conteudo":"decisao registada","rotulo":"FACT_VERIFIED","fonte":"chat-A"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperava 201, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "uma.md")); err != nil {
		t.Fatalf("a nota nao ficou no disco: %v", err)
	}

	recLeitura := pedirLeitura(t, router, "notas/uma.md")
	if recLeitura.Code != http.StatusOK {
		t.Fatalf("esperava 200 na leitura, veio %d (%s)", recLeitura.Code, recLeitura.Body.String())
	}
	var memoria agent.Memoria
	if err := json.Unmarshal(recLeitura.Body.Bytes(), &memoria); err != nil {
		t.Fatalf("resposta nao e' uma memoria: %v (%s)", err, recLeitura.Body.String())
	}
	if memoria.Conteudo != "decisao registada" {
		t.Fatalf("conteudo divergente: %q", memoria.Conteudo)
	}
	if memoria.Fonte != "chat-A" {
		t.Fatalf("a fonte (proveniencia) perdeu-se: %q", memoria.Fonte)
	}
}

// NEGATIVO de isolamento no HTTP: outro tenant e nota inexistente devolvem a MESMA resposta.
func TestHTTPLeituraNaoDistingueOutroTenantDeInexistente(t *testing.T) {
	prepararVaultDeTeste(t)
	routerDono := montarRotaDoVault(t, "user-permitido", "org-DONA")
	if rec := pedirAppend(t, routerDono, `{"relativo":"notas/da-outra.md","conteudo":"segredo de outro tenant","rotulo":"FACT_VERIFIED","fonte":"chat-A"}`); rec.Code != http.StatusCreated {
		t.Fatalf("preparacao falhou: %d (%s)", rec.Code, rec.Body.String())
	}

	routerIntruso := montarRotaDoVault(t, "user-permitido", "org-1")
	recOutroTenant := pedirLeitura(t, routerIntruso, "notas/da-outra.md")
	recInexistente := pedirLeitura(t, routerIntruso, "notas/nao-existe.md")

	if recOutroTenant.Code != http.StatusForbidden || recInexistente.Code != http.StatusForbidden {
		t.Fatalf("esperava 403 nos dois: %d e %d", recOutroTenant.Code, recInexistente.Code)
	}
	if recOutroTenant.Body.String() != recInexistente.Body.String() {
		t.Fatalf("respostas distinguem existencia: %q vs %q", recOutroTenant.Body.String(), recInexistente.Body.String())
	}
	if strings.Contains(recOutroTenant.Body.String(), "segredo") {
		t.Fatal("a resposta vazou conteudo de outro tenant")
	}
}

// NEGATIVO: sem HADES_VAULT_ROOT a rota fica desligada.
func TestHTTPSemRaizConfiguradaResponde503(t *testing.T) {
	t.Setenv("HADES_VAULT_ROOT", "")
	t.Setenv("HADES_VAULT_ACL", "")
	router := montarRotaDoVault(t, "user-permitido", "org-1")
	if rec := pedirAppend(t, router, `{"relativo":"a.md"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", rec.Code)
	}
	if rec := pedirLeitura(t, router, "a.md"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503 na leitura, veio %d", rec.Code)
	}
}

// NEGATIVO: configuracao de ACL ilegivel nao vira "sem concessoes" -- 503 e nada criado.
func TestHTTPACLInvalidaResponde503ENaoEscreve(t *testing.T) {
	raiz := t.TempDir()
	t.Setenv("HADES_VAULT_ROOT", raiz)
	t.Setenv("HADES_VAULT_ACL", `{isto nao e' json`)
	router := montarRotaDoVault(t, "user-permitido", "org-1")
	if rec := pedirAppend(t, router, `{"relativo":"a.md","conteudo":"x"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d (%s)", rec.Code, rec.Body.String())
	}
	entradas, _ := os.ReadDir(raiz)
	if len(entradas) != 0 {
		t.Fatalf("com ACL invalida foram criadas %d entrada(s)", len(entradas))
	}
}
