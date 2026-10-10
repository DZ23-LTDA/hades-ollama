package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
)

func chatRequestFixture(messages ...api.Message) api.ChatRequest {
	return api.ChatRequest{Model: "qwen2.5:7b", Messages: messages}
}

func TestInjectContextBootstrapPrependsWithoutMutatingOrDuplicating(t *testing.T) {
	original := []api.Message{
		{Role: "user", Content: "monte um relatório"},
		{Role: "assistant", Content: "certo"},
		{Role: "user", Content: "agora com gráficos"},
	}
	injected := injectContextBootstrap(original, "PREAMBULO")
	if len(injected) != len(original)+1 {
		t.Fatalf("mensagens = %d, quer %d", len(injected), len(original)+1)
	}
	if injected[0].Role != "system" || injected[0].Content != "PREAMBULO" {
		t.Fatalf("primeira mensagem = %+v", injected[0])
	}
	for index, message := range original {
		if injected[index+1].Role != message.Role || injected[index+1].Content != message.Content {
			t.Fatalf("ordem alterada no índice %d: %+v", index, injected[index+1])
		}
	}
	// O slice recebido não pode ser alterado no lugar.
	if len(original) != 3 || original[0].Role != "user" {
		t.Fatalf("slice original foi mutado: %+v", original)
	}

	// Idempotente: reenviar a conversa com o preâmbulo não duplica.
	again := injectContextBootstrap(injected, "PREAMBULO")
	if len(again) != len(injected) {
		t.Fatalf("preâmbulo duplicado: %d mensagens", len(again))
	}
	if empty := injectContextBootstrap(original, "   "); len(empty) != len(original) {
		t.Fatal("preâmbulo vazio não deveria injetar nada")
	}
}

func TestBuildChatContextPreambleDescribesSessionWithoutLeakingIdentifiers(t *testing.T) {
	request := chatRequestFixture(
		api.Message{Role: "user", Content: "primeira pergunta"},
		api.Message{Role: "assistant", Content: "resposta"},
		api.Message{Role: "user", Content: "objetivo atual da sessão"},
	)
	preamble, err := buildChatContextPreamble("org-secreta", "projeto-interno", request, []string{"workspace:read"}, agent.DefaultCapabilityPolicy())
	if err != nil {
		t.Fatalf("preamble: %v", err)
	}
	for _, expected := range []string{"Você opera dentro do Hades", "modo:", "PODE_LER", "objetivo atual da sessão"} {
		if !strings.Contains(preamble, expected) {
			t.Fatalf("preâmbulo sem %q:\n%s", expected, preamble)
		}
	}
	// Identificadores de tenant/projeto ficam do lado do servidor.
	for _, leak := range []string{"org-secreta", "projeto-interno"} {
		if strings.Contains(preamble, leak) {
			t.Fatalf("preâmbulo vazou %q:\n%s", leak, preamble)
		}
	}
	if strings.Contains(preamble, "primeira pergunta") {
		t.Fatal("objetivo precisa ser o último pedido do usuário, não o primeiro")
	}
}

func TestBuildChatContextPreambleRejectsUnknownCapabilityAndEmptyOrganization(t *testing.T) {
	request := chatRequestFixture(api.Message{Role: "user", Content: "oi"})
	if _, err := buildChatContextPreamble("", "", request, nil, agent.DefaultCapabilityPolicy()); err == nil {
		t.Fatal("organização vazia deveria falhar")
	}
	// Escopo desconhecido não vira permissão declarada: falha alto.
	if _, err := buildChatContextPreamble("org-a", "", request, []string{"escopo:inexistente"}, agent.DefaultCapabilityPolicy()); err == nil {
		t.Fatal("escopo desconhecido deveria falhar")
	}
}

func TestBuildChatContextPreambleTruncatesHugeObjective(t *testing.T) {
	huge := strings.Repeat("a", maxChatObjectiveRunes+250)
	request := chatRequestFixture(api.Message{Role: "user", Content: huge})
	preamble, err := buildChatContextPreamble("org-a", "", request, nil, agent.DefaultCapabilityPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(preamble)) > maxChatObjectiveRunes+2000 {
		t.Fatalf("preâmbulo grande demais: %d runas", len([]rune(preamble)))
	}
	// O que importa é o preâmbulo ser LIMITADO: um pedido gigante não pode ser
	// despejado inteiro no contexto do modelo.
	if strings.Contains(preamble, huge) {
		t.Fatal("objetivo gigante não deveria entrar inteiro no preâmbulo")
	}
	if !strings.Contains(preamble, strings.Repeat("a", 50)) {
		t.Fatal("o preâmbulo deveria preservar o início do objetivo")
	}
}

// chatContextRecorder devolve um handler que captura o corpo já processado pelo
// middleware — é assim que provamos a injeção sem invocar inferência.
func chatContextRecorder(captured *[]byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		*captured = body
		c.Status(http.StatusOK)
	}
}

func TestChatContextMiddlewareInjectsForAuthenticatedScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var captured []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: "org-a"})
		withAgentChatContext()(c)
	}, chatContextRecorder(&captured))

	body := `{"model":"qwen2.5:7b","messages":[{"role":"user","content":"oi"}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var received api.ChatRequest
	if err := json.Unmarshal(captured, &received); err != nil {
		t.Fatalf("json: %v (body=%s)", err, string(captured))
	}
	if len(received.Messages) != 2 {
		t.Fatalf("mensagens = %d, quer 2 (preâmbulo + usuário)", len(received.Messages))
	}
	if received.Messages[0].Role != "system" || !strings.Contains(received.Messages[0].Content, "Você opera dentro do Hades") {
		t.Fatalf("preâmbulo não foi injetado: %+v", received.Messages[0])
	}
	if received.Messages[1].Content != "oi" {
		t.Fatalf("mensagem do usuário alterada: %+v", received.Messages[1])
	}
}

func TestChatContextMiddlewareLeavesBodyUntouchedWithoutScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var captured []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", withAgentChatContext(), chatContextRecorder(&captured))

	body := `{"model":"qwen2.5:7b","messages":[{"role":"user","content":"oi"}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(body))
	engine.ServeHTTP(recorder, request)

	if string(captured) != body {
		t.Fatalf("sem escopo o corpo deveria passar intacto:\n%s", string(captured))
	}
}

func TestChatContextMiddlewareSurvivesInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var captured []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: "org-a"})
		withAgentChatContext()(c)
	}, chatContextRecorder(&captured))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader("{nao-e-json"))
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if string(captured) != "{nao-e-json" {
		t.Fatalf("corpo inválido deveria ser repassado ao handler: %s", string(captured))
	}
}
