package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// PROVA DA LIGAÇÃO DO E-568 (P0-1 da auditoria de memória).
//
// No commit do E-568 eu declarei, com todas as letras, que NÃO havia teste que
// AFIRMASSE que a decisão de orçamento fica registada no contexto — o teste do
// middleware provava que o caminho CORRIA e não quebrava, não que o campo estava lá.
// Este teste fecha isso: o handler a seguir ao middleware LÊ a decisão e ela tem de
// estar lá, com os valores certos.
//
// E ele distingue os dois caminhos, que é o que impede a tautologia: com um teto
// minúsculo (`OLLAMA_AGENT_CHAT_CONTEXT_TOKENS=10`) a decisão tem de ser COMPACTAR com
// motivo a dizer "acima"; sem teto declarado tem de ser NÃO COMPACTAR com motivo a
// dizer "não declarado". Um teste que só verificasse "existe decisão" passaria mesmo
// se a decisão fosse sempre a mesma.
func TestChatContextMiddlewareRegistraDecisaoDeOrcamento(t *testing.T) {
	gin.SetMode(gin.TestMode)

	correr := func(t *testing.T) (DecisaoDeOrcamento, bool) {
		t.Helper()
		var decisao DecisaoDeOrcamento
		var encontrada bool
		engine := gin.New()
		engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
			c.Set("agent.organization", agent.Organization{ID: "org-a"})
			withAgentChatContext()(c)
		}, func(c *gin.Context) {
			decisao, encontrada = DecisaoDeOrcamentoRegistada(c)
			c.Status(http.StatusOK)
		})
		corpo := `{"model":"qwen2.5:7b","messages":[{"role":"user","content":"` + strings.Repeat("x", 200) + `"}]}`
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(corpo))
		request.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d", recorder.Code)
		}
		return decisao, encontrada
	}

	t.Run("com teto declarado: a decisao chega ao handler e manda compactar", func(t *testing.T) {
		t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "10")
		t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "0")
		decisao, encontrada := correr(t)
		if !encontrada {
			t.Fatal("o middleware NÃO registou a decisão: a ligação do E-568 não aconteceu neste caminho")
		}
		if !decisao.Compactar {
			t.Fatalf("com teto de 10 tokens e 200+ caracteres devia decidir compactar: %+v", decisao)
		}
		if decisao.Uso.Exata {
			t.Fatal("uso marcado como exato: não há tokenizer neste runtime")
		}
		if !strings.Contains(decisao.Motivo, "acima") {
			t.Fatalf("motivo sem a decisão declarada: %q", decisao.Motivo)
		}
		if decisao.Gatilho.OrcamentoTokens != 10 {
			t.Fatalf("o gatilho registado não é o lido do ambiente: %+v", decisao.Gatilho)
		}
	})

	t.Run("sem teto declarado: a decisao chega e NAO manda compactar", func(t *testing.T) {
		t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "")
		t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "")
		decisao, encontrada := correr(t)
		if !encontrada {
			t.Fatal("o middleware deixou de registar a decisão quando não há teto")
		}
		if decisao.Compactar {
			t.Fatalf("decidiu compactar sem orçamento declarado: %+v", decisao)
		}
		if !strings.Contains(decisao.Motivo, "não declarado") {
			t.Fatalf("motivo = %q; ausência de orçamento tem de ser DITA", decisao.Motivo)
		}
	})
}
