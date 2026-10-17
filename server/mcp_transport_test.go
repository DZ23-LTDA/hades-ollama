package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TRANSPORTE MCP: o que so' a ROTA pode provar (E-647).
//
// O respondedor tem os seus testes no pacote do agente (E-646). Aqui prova-se o TRANSPORTE: que os
// erros sao do PROTOCOLO (JSON-RPC em 200) e que um corpo invalido vira -32700, em vez de um 400
// que o cliente MCP nao sabe ler.

func pedidoMCP(t *testing.T, corpo string, organizacao string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/mcp/rpc", strings.NewReader(corpo))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("agent.organization", agent.Organization{ID: organizacao})
	return ctx, recorder
}

func TestTransporteMCPRespondeInitializeComProtocolo(t *testing.T) {
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{context: store, authRequired: true}
	ctx, recorder := pedidoMCP(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, "org-a")
	api.mcpRPC(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("initialize devia dar 200, deu %d", recorder.Code)
	}
	var resposta struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta ilegivel: %v (%s)", err, recorder.Body.String())
	}
	if resposta.JSONRPC != "2.0" || resposta.Result.ProtocolVersion != agent.ProtocoloMCP {
		t.Fatalf("resposta fora do protocolo: %s", recorder.Body.String())
	}
	// E a organizacao NAO vai na resposta: e' identidade de sessao, nao informacao publica.
	if strings.Contains(recorder.Body.String(), "org-a") {
		t.Fatalf("*** a organizacao nao pode ir na resposta ***: %s", recorder.Body.String())
	}
}

// ERRO DE PROTOCOLO VAI EM 200: quem fala MCP le JSON-RPC, e um 4xx obrigaria o cliente a adivinhar.
func TestTransporteMCPDevolveErroDeProtocoloEm200(t *testing.T) {
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{context: store, authRequired: true}

	ctx, recorder := pedidoMCP(t, `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{}}`, "org-a")
	api.mcpRPC(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("metodo desconhecido devia dar 200 com erro JSON-RPC, deu %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "-32601") {
		t.Fatalf("devia ser -32601: %s", recorder.Body.String())
	}

	// Corpo invalido => -32700 (parse error), tambem em 200.
	ctxInvalido, recorderInvalido := pedidoMCP(t, `{isto nao e' json`, "org-a")
	api.mcpRPC(ctxInvalido)
	if recorderInvalido.Code != http.StatusOK || !strings.Contains(recorderInvalido.Body.String(), "-32700") {
		t.Fatalf("corpo invalido devia dar -32700 em 200: %d %s", recorderInvalido.Code, recorderInvalido.Body.String())
	}
}
