package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// §9 — A VERDADE DE CAPACIDADE NA SUPERFICIE (E-654).
//
// A garantia que estes testes travam NAO e' "a rota responde 200": e' que ***um modelo NAO MEDIDO
// nao aparece como suportado***. Confundir «nao medido» com «nao suporta» faria o produto recusar um
// modelo que talvez funcione; confundir com «suporta» faria prometer o que nao acontece.

func pedidoDeCapacidades(t *testing.T, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, caminho, nil)
	api := &agentAPI{authRequired: true}
	api.capabilities(ctx)
	return recorder
}

func TestCapacidadesListaOsFatosComOrigem(t *testing.T) {
	recorder := pedidoDeCapacidades(t, "/api/agent/v1/capabilities")
	if recorder.Code != http.StatusOK {
		t.Fatalf("devia dar 200, deu %d", recorder.Code)
	}
	var resposta struct {
		Models []struct {
			Model  string `json:"model"`
			Source string `json:"source"`
		} `json:"models"`
		Policy string `json:"policy"`
		Scope  string `json:"scope"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta ilegivel: %v (%s)", err, recorder.Body.String())
	}
	if len(resposta.Models) == 0 {
		t.Fatal("o registo devia trazer modelos conhecidos")
	}
	for _, modelo := range resposta.Models {
		// ***CAPACIDADE SEM ORIGEM NAO E' CAPACIDADE*** - e' a regra do proprio modulo.
		if modelo.Source != "medido" && modelo.Source != "declarado" {
			t.Fatalf("modelo %q sem origem declarada: %q", modelo.Model, modelo.Source)
		}
	}
	if !strings.Contains(resposta.Policy, "NAO_MEDIDO") {
		t.Fatalf("a politica para o que nao esta' na lista tem de estar dita: %q", resposta.Policy)
	}
	if resposta.Scope == "" {
		t.Fatal("um dado sem escopo e' lido como global")
	}
}

func TestCapacidadesDeModeloConhecidoDizemOModoEMedeOrigem(t *testing.T) {
	recorder := pedidoDeCapacidades(t, "/api/agent/v1/capabilities?model=qwen2.5:7b")
	var resposta struct {
		Model        string `json:"model"`
		Known        bool   `json:"known"`
		ToolMode     string `json:"tool_mode"`
		Capabilities struct {
			Source string `json:"source"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatal(err)
	}
	if !resposta.Known || resposta.Model != "qwen2.5:7b" {
		t.Fatalf("modelo medido devia ser conhecido: %+v", resposta)
	}
	if resposta.ToolMode == "" {
		t.Fatal("o modo de ferramentas tem de ser dito")
	}
	if resposta.Capabilities.Source == "" {
		t.Fatal("a capacidade devia trazer a origem")
	}
}

func TestCapacidadesDeModeloDesconhecidoNaoAfirmamSuporte(t *testing.T) {
	recorder := pedidoDeCapacidades(t, "/api/agent/v1/capabilities?model=modelo-que-ninguem-sondou:99b")
	if recorder.Code != http.StatusOK {
		t.Fatalf("desconhecido nao e' erro de protocolo: %d", recorder.Code)
	}
	corpo := recorder.Body.String()
	var resposta struct {
		Known    bool   `json:"known"`
		ToolMode string `json:"tool_mode"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatal(err)
	}
	if resposta.Known {
		t.Fatal("*** modelo nao sondado NAO pode aparecer como conhecido ***")
	}
	if resposta.ToolMode != "unknown" {
		t.Fatalf("modo devia ser unknown, veio %q", resposta.ToolMode)
	}
	if !strings.Contains(resposta.Reason, "NAO MEDIDA") {
		t.Fatalf("o motivo tem de dizer que nao foi medido: %q", resposta.Reason)
	}
	// E o corpo NAO pode afirmar suporte: nada de `tool_calling: true` nem de capacidades.
	if strings.Contains(corpo, `"tool_calling":true`) || strings.Contains(corpo, `"capabilities"`) {
		t.Fatalf("*** o desconhecido nao pode trazer afirmacao de capacidade ***: %s", corpo)
	}
}
