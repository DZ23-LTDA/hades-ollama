package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const rotaDaRecomendacao = "/api/agent/v1/systems/recomendacao"

func pedirRecomendacao(t *testing.T, comSessao bool, query string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if comSessao {
		router.Use(sessaoDeBuild("org-teste"))
	}
	router.GET(rotaDaRecomendacao, (&agentAPI{}).recomendacaoPorVRAM)

	pedido := httptest.NewRequest(http.MethodGet, rotaDaRecomendacao+query, nil)
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var corpo map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e' JSON: %v (%q)", err, resposta.Body.String())
	}
	return resposta.Code, corpo
}

// NEGATIVO: sem sessao nao ha' recomendacao.
func TestRecomendacaoSemSessaoRecusa(t *testing.T) {
	codigo, corpo := pedirRecomendacao(t, false, "?vram_bytes=8589934592&parametros_b=7")
	if codigo != http.StatusUnauthorized {
		t.Fatalf("sem sessao devia dar 401, veio %d (%v)", codigo, corpo)
	}
	if _, existe := corpo["recomendacao"]; existe {
		t.Fatalf("sem sessao a resposta NAO pode trazer recomendacao: %v", corpo)
	}
}

// NEGATIVO, e o mais importante: sem VRAM medida nao se recomenda -- devolve-se o motivo.
func TestRecomendacaoSemVRAMNaoInventa(t *testing.T) {
	codigo, corpo := pedirRecomendacao(t, true, "")
	if codigo != http.StatusBadRequest {
		t.Fatalf("sem vram_bytes devia dar 400, veio %d (%v)", codigo, corpo)
	}
	if _, existe := corpo["recomendacao"]; existe {
		t.Fatalf("sem VRAM a resposta NAO pode trazer recomendacao: %v", corpo)
	}
	if detalhe, _ := corpo["detalhe"].(string); detalhe == "" {
		t.Fatalf("a recusa tem de explicar porque: %v", corpo)
	}
	// E o mesmo para um valor invalido.
	if codigo, _ := pedirRecomendacao(t, true, "?vram_bytes=-1"); codigo != http.StatusBadRequest {
		t.Fatalf("vram negativa devia dar 400, veio %d", codigo)
	}
	if codigo, _ := pedirRecomendacao(t, true, "?vram_bytes=abc"); codigo != http.StatusBadRequest {
		t.Fatalf("vram ilegivel devia dar 400, veio %d", codigo)
	}
}

// POSITIVO: com sessao, VRAM medida e parametros, devolve a recomendacao com numeros.
func TestRecomendacaoComVRAMMedida(t *testing.T) {
	codigo, corpo := pedirRecomendacao(t, true, "?vram_bytes=8589934592&parametros_b=7")
	if codigo != http.StatusOK {
		t.Fatalf("devia dar 200, veio %d (%v)", codigo, corpo)
	}
	rec, ok := corpo["recomendacao"].(map[string]any)
	if !ok {
		t.Fatalf("a resposta tem de trazer 'recomendacao': %v", corpo)
	}
	for _, campo := range []string{"quantizacao", "contexto", "camadas_na_gpu", "motivo"} {
		if _, existe := rec[campo]; !existe {
			t.Fatalf("a recomendacao tem de trazer %q: %v", campo, rec)
		}
	}
	if quantizacao, _ := rec["quantizacao"].(string); quantizacao == "" {
		t.Fatalf("quantizacao vazia numa recomendacao valida: %v", rec)
	}
}
