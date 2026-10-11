package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 35: as rotas dos agentes precisam entregar contexto real, procedência do
// texto e escalada com motivo — e continuar respeitando o consentimento.
func setupServiceAgentAPI(t *testing.T, grantConsent bool) (*gin.Engine, *agent.Runtime, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	now := time.Now().UTC()
	phone := "+5511999998888"
	if _, _, err := runtime.CRM().UpsertContact(agent.CRMContact{OrganizationID: agent.LocalOrganizationID, Name: "Ana Souza", Phone: phone}, now); err != nil {
		t.Fatal(err)
	}
	contact := runtime.CRM().ListContacts(agent.LocalOrganizationID, "")[0]
	if _, err := runtime.CRM().CreateOpportunity(agent.LocalOrganizationID, contact.ID, "Plano avançado", 49900, "ana", now); err != nil {
		t.Fatal(err)
	}
	if grantConsent {
		if _, err := runtime.Consent().GrantConsent(agent.LocalOrganizationID, agent.ConsentChannelWhatsApp, phone, "site", "checkbox", "ana", nil, now); err != nil {
			t.Fatal(err)
		}
	}
	conversation, _, err := runtime.Inbox().AppendInbound(agent.LocalOrganizationID, "whatsapp", phone, "Ana Souza", "quero saber do plano", "wamid.sa.route", now)
	if err != nil {
		t.Fatal(err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.GET("/service-agents", api.listServiceAgents)
	group.POST("/service-agents/:id/turn", api.serviceAgentTurn)
	return engine, runtime, conversation.ID
}

func serviceAgentRequest(engine *gin.Engine, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestServiceAgentRoutesExposeCatalogAndDeclareProvenance(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, true)

	catalog := httptest.NewRecorder()
	engine.ServeHTTP(catalog, httptest.NewRequest(http.MethodGet, "/api/agent/v1/service-agents", nil))
	if catalog.Code != http.StatusOK {
		t.Fatalf("catálogo: %d", catalog.Code)
	}
	var listed struct {
		Agents []agent.ServiceAgentSpec `json:"agents"`
		Count  int                      `json:"count"`
	}
	if err := json.Unmarshal(catalog.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Count != 7 || len(listed.Agents) != 7 {
		t.Fatalf("o catálogo precisa trazer os sete agentes: %+v", listed)
	}

	turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn",
		`{"conversation_id":"`+conversationID+`","intent":"quero o plano","model_id":"qwen2.5:7b"}`)
	if turn.Code != http.StatusOK {
		t.Fatalf("turno precisa ser 200, veio %d (%s)", turn.Code, turn.Body.String())
	}
	var result agent.ServiceAgentTurn
	if err := json.Unmarshal(turn.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Decision != "responder" || result.Escalated {
		t.Fatalf("com consentimento o agente responde: %+v", result)
	}
	// PROCEDÊNCIA declarada: o cliente não pode achar que houve modelo quando não houve.
	if result.ReplySource != agent.ReplySourceTemplate {
		t.Fatalf("a procedência precisa ser declarada: %q", result.ReplySource)
	}
	// Contexto REAL: contato do CRM, oportunidade e histórico.
	if result.Context.Contact == nil || len(result.Context.Opportunities) != 1 || len(result.Context.History) == 0 {
		t.Fatalf("o contexto precisa ser real: %+v", result.Context)
	}
	if result.Context.ModelID != "qwen2.5:7b" {
		t.Fatalf("o modelo escolhido precisa aparecer: %+v", result.Context)
	}
	if len(result.Context.Policies) < 3 {
		t.Fatalf("as políticas aplicadas precisam estar declaradas: %+v", result.Context.Policies)
	}

	// Agente inexistente: erro explícito, não 200 com resposta vazia.
	if unknown := serviceAgentRequest(engine, "/api/agent/v1/service-agents/fantasma/turn",
		`{"conversation_id":"`+conversationID+`"}`); unknown.Code == http.StatusOK {
		t.Fatalf("agente inexistente não pode responder 200: %s", unknown.Body.String())
	}
	// Corpo inválido: 400.
	if bad := serviceAgentRequest(engine, "/api/agent/v1/service-agents/sales/turn", `{"conversation_id":`); bad.Code != http.StatusBadRequest {
		t.Fatalf("corpo inválido precisa ser 400, veio %d", bad.Code)
	}
}

func TestServiceAgentRouteEscalatesWithoutConsent(t *testing.T) {
	engine, _, conversationID := setupServiceAgentAPI(t, false)
	turn := serviceAgentRequest(engine, "/api/agent/v1/service-agents/support/turn",
		`{"conversation_id":"`+conversationID+`","intent":"meu pedido não chegou"}`)
	// Escalada é resposta legítima: 202 com o MOTIVO, nunca 200 fingindo atendimento.
	if turn.Code != http.StatusAccepted {
		t.Fatalf("escalada precisa ser 202, veio %d (%s)", turn.Code, turn.Body.String())
	}
	var result agent.ServiceAgentTurn
	if err := json.Unmarshal(turn.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Escalated || result.Escalation != agent.ServiceAgentEscalateConsent {
		t.Fatalf("a escalada precisa vir com o motivo: %+v", result)
	}
	if result.Message != nil || result.Reply != "" {
		t.Fatalf("escalar não pode responder: %+v", result)
	}
}
