package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const rotaDoEstadoDoPare = "/api/agent/v1/build/pare"

func pedirEstadoDoPare(t *testing.T, comSessao bool) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if comSessao {
		router.Use(sessaoDeBuild("org-teste"))
	}
	router.GET(rotaDoEstadoDoPare, (&agentAPI{}).estadoDoPare)

	pedido := httptest.NewRequest(http.MethodGet, rotaDoEstadoDoPare, nil)
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var corpo map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e' JSON: %v (corpo %q)", err, resposta.Body.String())
	}
	return resposta.Code, corpo
}

// NEGATIVO: sem sessao nao ha' estado. Nao e' um 200 com dados vazios -- e' recusa.
func TestEstadoDoPareSemSessaoRecusa(t *testing.T) {
	codigo, corpo := pedirEstadoDoPare(t, false)
	if codigo != http.StatusUnauthorized {
		t.Fatalf("sem sessao devia dar 401, veio %d (%v)", codigo, corpo)
	}
	if _, existe := corpo["pare"]; existe {
		t.Fatalf("sem sessao a resposta NAO pode trazer o estado: %v", corpo)
	}
}

// POSITIVO: com sessao, o estado e' legivel e diz que esta' desarmado.
func TestEstadoDoPareDesarmado(t *testing.T) {
	codigo, corpo := pedirEstadoDoPare(t, true)
	if codigo != http.StatusOK {
		t.Fatalf("com sessao devia dar 200, veio %d (%v)", codigo, corpo)
	}
	pare, ok := corpo["pare"].(map[string]any)
	if !ok {
		t.Fatalf("a resposta tem de trazer o objecto 'pare': %v", corpo)
	}
	if acionado, _ := pare["acionado"].(bool); acionado {
		t.Fatalf("o pare comeca desarmado, veio acionado=%v", pare["acionado"])
	}
}

// E quando esta' acionado, o estado DIZ porque -- e nao expoe segredos.
func TestEstadoDoPareAcionadoMostraMotivo(t *testing.T) {
	if err := pareDoServidor.Acionar("motivo observavel no estado", "teste"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	t.Cleanup(func() { _ = pareDoServidor.Rearmar("teste", "limpeza do teste") })

	codigo, corpo := pedirEstadoDoPare(t, true)
	if codigo != http.StatusOK {
		t.Fatalf("devia dar 200, veio %d", codigo)
	}
	pare, _ := corpo["pare"].(map[string]any)
	if acionado, _ := pare["acionado"].(bool); !acionado {
		t.Fatalf("devia estar acionado: %v", pare)
	}
	if motivo, _ := pare["motivo"].(string); !strings.Contains(motivo, "motivo observavel") {
		t.Fatalf("o estado tem de citar o motivo: %v", pare)
	}
	// Nada de credenciais no corpo do estado.
	bruto, _ := json.Marshal(corpo)
	for _, proibido := range []string{"senha", "password", "token", "secret", "api_key"} {
		if strings.Contains(strings.ToLower(string(bruto)), proibido) {
			t.Fatalf("o estado do pare nao pode expor %q: %s", proibido, bruto)
		}
	}
}
