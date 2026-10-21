package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const rotaDaProposta = "/api/agent/v1/goals/proposta"

func pedirProposta(t *testing.T, comSessao bool, corpo string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if comSessao {
		router.Use(sessaoDeBuild("org-teste"))
	}
	router.POST(rotaDaProposta, (&agentAPI{}).propostaDeGoal)

	pedido := httptest.NewRequest(http.MethodPost, rotaDaProposta, bytes.NewReader([]byte(corpo)))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var saida map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &saida); err != nil {
		t.Fatalf("resposta nao e' JSON: %v (%q)", err, resposta.Body.String())
	}
	return resposta.Code, saida
}

// NEGATIVO: sem sessao nao ha' proposta.
func TestPropostaSemSessaoRecusa(t *testing.T) {
	codigo, corpo := pedirProposta(t, false, `{"mensagens":[{"papel":"user","conteudo":"reduzir a latencia para 200ms"}]}`)
	if codigo != http.StatusUnauthorized {
		t.Fatalf("sem sessao devia dar 401, veio %d (%v)", codigo, corpo)
	}
	if _, existe := corpo["proposta"]; existe {
		t.Fatalf("sem sessao a resposta NAO pode trazer proposta: %v", corpo)
	}
}

// NEGATIVO: conversa vazia nao produz proposta -- inventar um goal a partir de nada e' pior.
func TestPropostaComConversaVaziaRecusa(t *testing.T) {
	for _, corpo := range []string{`{"mensagens":[]}`, `{"mensagens":[{"papel":"user","conteudo":"   "}]}`, `{}`} {
		codigo, saida := pedirProposta(t, true, corpo)
		if codigo != http.StatusBadRequest {
			t.Fatalf("corpo %s devia dar 400, veio %d (%v)", corpo, codigo, saida)
		}
		if _, existe := saida["proposta"]; existe {
			t.Fatalf("corpo %s NAO pode trazer proposta: %v", corpo, saida)
		}
	}
}

// POSITIVO: com conversa, a proposta vem com motivo e com o que falta -- e nao cria nada.
func TestPropostaComConversaCompleta(t *testing.T) {
	codigo, corpo := pedirProposta(t, true, `{"mensagens":[{"papel":"user","conteudo":"Quero reduzir a latencia da API para 200ms neste trimestre."}]}`)
	if codigo != http.StatusOK {
		t.Fatalf("devia dar 200, veio %d (%v)", codigo, corpo)
	}
	proposta, ok := corpo["proposta"].(map[string]any)
	if !ok {
		t.Fatalf("a resposta tem de trazer 'proposta': %v", corpo)
	}
	for _, campo := range []string{"goal", "motivo", "confianca", "faltando"} {
		if _, existe := proposta[campo]; !existe {
			t.Fatalf("a proposta tem de trazer %q: %v", campo, proposta)
		}
	}
	if motivo, _ := proposta["motivo"].(string); !strings.Contains(motivo, "latencia") && !strings.Contains(motivo, "tempo de resposta") {
		t.Fatalf("o motivo tem de citar o que encontrou: %q", motivo)
	}
	goal, _ := proposta["goal"].(map[string]any)
	if metrica, _ := goal["metric"].(string); metrica == "" {
		t.Fatalf("a conversa nomeia a metrica, o goal proposto devia te-la: %v", goal)
	}
}

// E o que a conversa NAO disse aparece em `faltando`, em vez de ser preenchido com um palpite.
func TestPropostaDizOQueFalta(t *testing.T) {
	codigo, corpo := pedirProposta(t, true, `{"mensagens":[{"papel":"user","conteudo":"Precisamos melhorar aquilo tudo."}]}`)
	if codigo != http.StatusOK {
		t.Fatalf("devia dar 200, veio %d", codigo)
	}
	proposta, _ := corpo["proposta"].(map[string]any)
	faltando, _ := proposta["faltando"].([]any)
	if len(faltando) == 0 {
		t.Fatalf("a conversa nao nomeia metrica/alvo/periodo, `faltando` nao pode estar vazio: %v", proposta)
	}
	goal, _ := proposta["goal"].(map[string]any)
	if metrica, _ := goal["metric"].(string); metrica != "" {
		t.Fatalf("metrica NAO pode ser inventada: %q", metrica)
	}
}
