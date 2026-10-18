package server

// Testes das rotas do Signing Identity Center (FASE 28.5).
//
// O que estes testes protegem:
//   1. as rotas existem e respondem 200 mesmo quando o registo nao esta' configurado --
//      *um 404 faria a UI mostrar erro de rede onde o que falta e' accao do proprietario*;
//   2. a resposta NAO contem nenhum campo com material secreto, verificado por varredura do JSON;
//   3. `problemas` e' sempre lista, nunca nil;
//   4. a rota e' de LEITURA: POST recusado.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func routerDeSigning(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	return engine
}

func TestSigningIdentitiesRespondeMesmoSemRegisto(t *testing.T) {
	engine := routerDeSigning(t)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/signing/identities", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /signing/identities -> %d; esperado 200 (nao configurado e' estado, nao falha)", recorder.Code)
	}

	var payload struct {
		Estado struct {
			Configurado bool   `json:"configurado"`
			Carregado   bool   `json:"carregado"`
			Mensagem    string `json:"mensagem"`
		} `json:"estado"`
		Identidades []json.RawMessage      `json:"identidades"`
		Problemas   []json.RawMessage      `json:"problemas"`
		Resumo      map[string]interface{} `json:"resumo"`
		Aviso       string                 `json:"aviso"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON invalido: %v (corpo: %s)", err, recorder.Body.String())
	}
	if payload.Estado.Mensagem == "" {
		t.Fatal("estado sem mensagem: a UI nao tem o que mostrar")
	}
	// `identidades` e `problemas` tem de ser listas, nao null.
	if payload.Identidades == nil {
		t.Fatal("identidades veio null; o contrato promete lista")
	}
	if payload.Problemas == nil {
		t.Fatal("problemas veio null; o contrato promete lista")
	}
	if payload.Resumo == nil {
		t.Fatal("resumo veio null")
	}
	// O aviso e' parte do contrato: o consumidor nao deve ter de adivinhar.
	if !strings.Contains(strings.ToLower(payload.Aviso), "metadados") {
		t.Fatalf("aviso nao explica que so' ha' metadados: %q", payload.Aviso)
	}
}

// A varredura negativa: nenhuma chave do JSON pode parecer material secreto.
//
// Este teste e' o que impede a rota de crescer para o lado errado no futuro.
func TestSigningIdentitiesNaoExpoeMaterialSecreto(t *testing.T) {
	engine := routerDeSigning(t)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/signing/identities", nil))
	corpo := strings.ToLower(recorder.Body.String())

	for _, proibido := range []string{
		"private_key", "privatekey", "private key", "password", "senha",
		"-----begin", "storepassword", "keypassword", "client_email",
		"secret_value", "pepem", "p12_content", "keystore_content",
	} {
		if strings.Contains(corpo, proibido) {
			t.Fatalf("a resposta contem %q; a rota so' pode devolver metadados publicos", proibido)
		}
	}
}

func TestSigningSummaryResponde(t *testing.T) {
	engine := routerDeSigning(t)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/signing/summary", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /signing/summary -> %d", recorder.Code)
	}
	var payload struct {
		Configurado bool   `json:"configurado"`
		Carregado   bool   `json:"carregado"`
		Total       int    `json:"total"`
		Mensagem    string `json:"mensagem"`
		Resumo      struct {
			Total       int            `json:"total"`
			PorEstado   map[string]int `json:"por_estado"`
			PorAmbiente map[string]int `json:"por_ambiente"`
			Problemas   int            `json:"problemas"`
		} `json:"resumo"`
		Problemas int `json:"problemas"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON invalido: %v", err)
	}
	if payload.Mensagem == "" {
		t.Fatal("mensagem vazia")
	}
	// Coerencia: o total do resumo tem de bater com o total do estado.
	if payload.Resumo.Total != payload.Total {
		t.Fatalf("resumo.total=%d mas estado.total=%d", payload.Resumo.Total, payload.Total)
	}
	// Coerencia entre `carregado` e a existencia de contagens.
	if !payload.Carregado && payload.Total != 0 {
		t.Fatalf("nao carregado mas total=%d", payload.Total)
	}
}

// As rotas sao de LEITURA: um POST tem de ser recusado.
//
// *Uma rota que assina por HTTP transformaria um pedido em acto, e um acto precisa de um rosto.*
func TestSigningE_SoLeitura(t *testing.T) {
	engine := routerDeSigning(t)
	for _, caminho := range []string{
		"/api/agent/v1/signing/identities",
		"/api/agent/v1/signing/summary",
	} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, caminho, nil))
		if recorder.Code == http.StatusOK {
			t.Fatalf("POST %s devolveu 200; o Signing Identity Center e' so' leitura", caminho)
		}
	}
}
