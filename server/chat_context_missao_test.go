package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
)

// Testes do D-5: identidade de missão no funil do chat.
//
// A PROPRIEDADE DE SEGURANÇA é o que mais importa aqui: o identificador VEM DO PEDIDO, logo
// passa pelo MESMO guard de missão (`missionForRequest`). Sem isso, um cabeçalho controlado
// pelo cliente seria injecção de contexto entre inquilinos — exactamente o que o E-597 fechou
// do outro lado do §4.

func pedidoDeChatComMissao(t *testing.T, apiHades *agentAPI, organizationID, missionID string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var capturado []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: organizationID})
		withAgentChatContext(apiHades, "")(c)
	}, chatContextRecorder(&capturado))

	corpo := `{"model":"qwen2.5:7b","messages":[{"role":"user","content":"oi"}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(corpo))
	request.Header.Set("Content-Type", "application/json")
	if missionID != "" {
		request.Header.Set("X-Hades-Mission", missionID)
	}
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var recebido api.ChatRequest
	if err := json.Unmarshal(capturado, &recebido); err != nil {
		t.Fatalf("json: %v (body=%s)", err, string(capturado))
	}
	juntas := ""
	for _, m := range recebido.Messages {
		juntas += m.Content + "\n"
	}
	return juntas
}

func TestChatComMissaoDoProprioInquilinoTrazHandoffRedigido(t *testing.T) {
	api, mission, project := newMissionContextFixture(t, "org-a")
	preamble := pedidoDeChatComMissao(t, api, "org-a", mission.ID)

	// (1) O HANDOFF CHEGA — é o que o D-5 veio permitir.
	if !strings.Contains(preamble, agent.MarcadorDeHandoff) {
		t.Fatal("o handoff não chegou ao preâmbulo do chat")
	}
	// (2) E CHEGA REDIGIDO: identificadores internos e caminhos locais NÃO aparecem.
	for _, proibido := range []string{mission.ID, project.ID} {
		if proibido != "" && strings.Contains(preamble, proibido) {
			t.Fatalf("o preâmbulo do chat vazou o identificador %q", proibido)
		}
	}
	if strings.Contains(preamble, "Projeto:") || strings.Contains(preamble, "Missão:") {
		t.Fatal("o preâmbulo traz os rótulos de topologia que a versão redigida remove")
	}
}

// Sem cabeçalho, o chat continua como sempre: nada de handoff, e o preâmbulo normal.
func TestChatSemMissaoNaoInventaHandoff(t *testing.T) {
	api, _, _ := newMissionContextFixture(t, "org-a")
	preamble := pedidoDeChatComMissao(t, api, "org-a", "")
	if strings.Contains(preamble, agent.MarcadorDeHandoff) {
		t.Fatal("apareceu handoff sem o pedido identificar missão")
	}
	if !strings.Contains(preamble, "Você opera dentro do Hades") {
		t.Fatal("o preâmbulo normal desapareceu")
	}
}

// O GUARD: uma missão de OUTRA organização (ou inexistente) não pode fornecer contexto.
// Esta é a asserção que impede o cabeçalho controlado pelo cliente de virar injecção.
func TestChatRecusaMissaoDeOutroInquilino(t *testing.T) {
	api, mission, _ := newMissionContextFixture(t, "org-a")
	// Mesma missão, pedido autenticado noutra organização.
	preamble := pedidoDeChatComMissao(t, api, "org-b", mission.ID)
	if strings.Contains(preamble, agent.MarcadorDeHandoff) {
		t.Fatal("SERVIDOR ENTREGOU HANDOFF DE MISSÃO DE OUTRO INQUILINO")
	}
	// E um id inexistente também não fornece nada.
	preamble = pedidoDeChatComMissao(t, api, "org-a", "mis_inexistente_000")
	if strings.Contains(preamble, agent.MarcadorDeHandoff) {
		t.Fatal("entregou handoff para um id inexistente")
	}
}

// Com API nula (o caso dos testes que só exercitam o preâmbulo) o middleware não pode
// explodir: a degradação é declarada, não silenciosa.
func TestChatSemAPIResolvePreambleSemMissao(t *testing.T) {
	preamble := pedidoDeChatComMissao(t, nil, "org-a", "mis_qualquer")
	if !strings.Contains(preamble, "Você opera dentro do Hades") {
		t.Fatal("sem API o preâmbulo normal devia continuar a ser injetado")
	}
	if strings.Contains(preamble, agent.MarcadorDeHandoff) {
		t.Fatal("sem API não há como resolver missão — não pode aparecer handoff")
	}
}
