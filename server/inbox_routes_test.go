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

// Estágio 29: rotas da caixa de entrada, com deduplicação de webhook, handoff e
// consentimento no envio.
func setupInboxAPI(t *testing.T) (*gin.Engine, *agent.Runtime) {
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
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.GET("/inbox/conversations", api.listInboxConversations)
	group.POST("/inbox/messages", api.appendInboxMessage)
	group.GET("/inbox/conversations/:id", api.getInboxConversation)
	group.POST("/inbox/conversations/:id/reply", api.replyInboxConversation)
	group.POST("/inbox/conversations/:id/assign", api.assignInboxConversation)
	group.POST("/inbox/conversations/:id/handoff", api.handoffInboxConversation)
	group.POST("/inbox/conversations/:id/close", api.closeInboxConversation)
	group.GET("/inbox/metrics", api.inboxMetrics)
	return engine, runtime
}

func inboxRequest(engine *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestInboxRoutesDeduplicateHandoffAndMeasureSLA(t *testing.T) {
	engine, _ := setupInboxAPI(t)
	inbound := `{"channel":"whatsapp","contact_id":"+5511999998888","contact_name":"Ana","body":"preciso de ajuda","external_id":"wamid.http.1"}`

	first := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/messages", inbound)
	if first.Code != http.StatusCreated {
		t.Fatalf("primeira mensagem precisa ser 201, veio %d (%s)", first.Code, first.Body.String())
	}
	var created struct {
		Conversation agent.InboxConversation `json:"conversation"`
		Message      agent.InboxMessage      `json:"message"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// Reentrega do MESMO evento: 200 com a mensagem ORIGINAL e marca de deduplicação.
	repeat := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/messages", inbound)
	if repeat.Code != http.StatusOK {
		t.Fatalf("reentrega precisa ser 200, veio %d (%s)", repeat.Code, repeat.Body.String())
	}
	var repeated struct {
		Message      agent.InboxMessage `json:"message"`
		Deduplicated bool               `json:"deduplicated"`
	}
	if err := json.Unmarshal(repeat.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if !repeated.Deduplicated || repeated.Message.ID != created.Message.ID {
		t.Fatalf("a reentrega precisa devolver a mesma mensagem marcada: %+v", repeated)
	}

	// A conversa tem UMA mensagem e o detalhe expõe o estado de resposta.
	detail := inboxRequest(engine, http.MethodGet, "/api/agent/v1/inbox/conversations/"+created.Conversation.ID, "")
	if detail.Code != http.StatusOK {
		t.Fatalf("detalhe precisa ser 200, veio %d", detail.Code)
	}
	var view struct {
		Conversation agent.InboxConversation `json:"conversation"`
		Messages     []agent.InboxMessage    `json:"messages"`
		ReplyBlocked bool                    `json:"reply_blocked"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Messages) != 1 {
		t.Fatalf("webhook duplicado não pode duplicar mensagem: %d", len(view.Messages))
	}
	// Sem consentimento registrado, a visão já avisa que responder está bloqueado.
	if !view.ReplyBlocked {
		t.Fatal("sem consentimento o detalhe precisa indicar resposta bloqueada")
	}
	// E a resposta é RECUSADA pelo servidor (não só avisada).
	blocked := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/conversations/"+created.Conversation.ID+"/reply", `{"body":"resposta"}`)
	if blocked.Code == http.StatusCreated {
		t.Fatalf("sem consentimento a resposta precisa ser recusada, veio %d (%s)", blocked.Code, blocked.Body.String())
	}

	// Handoff exige motivo.
	noReason := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/conversations/"+created.Conversation.ID+"/handoff", `{}`)
	if noReason.Code != http.StatusBadRequest {
		t.Fatalf("handoff sem motivo precisa ser 400, veio %d (%s)", noReason.Code, noReason.Body.String())
	}
	handoff := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/conversations/"+created.Conversation.ID+"/handoff", `{"reason":"cliente pediu humano","assignee":"ana"}`)
	if handoff.Code != http.StatusOK {
		t.Fatalf("handoff precisa ser 200, veio %d (%s)", handoff.Code, handoff.Body.String())
	}
	var handed struct {
		Conversation agent.InboxConversation `json:"conversation"`
	}
	if err := json.Unmarshal(handoff.Body.Bytes(), &handed); err != nil {
		t.Fatal(err)
	}
	if handed.Conversation.Status != agent.InboxStatusPendingHuman || handed.Conversation.AssignedTo != "ana" {
		t.Fatalf("a conversa precisa ir para humano com responsável: %+v", handed.Conversation)
	}

	// Indicadores: fila humana contabilizada e SLA aguardando resposta.
	metrics := inboxRequest(engine, http.MethodGet, "/api/agent/v1/inbox/metrics", "")
	if metrics.Code != http.StatusOK {
		t.Fatalf("métricas precisam ser 200, veio %d", metrics.Code)
	}
	var summary agent.InboxMetrics
	if err := json.Unmarshal(metrics.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.PendingHuman != 1 || summary.AwaitingFirstReply != 1 {
		t.Fatalf("indicadores incorretos: %+v", summary)
	}

	// Filtro por status e encerramento.
	pending := inboxRequest(engine, http.MethodGet, "/api/agent/v1/inbox/conversations?status=pending_human", "")
	var listed struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(pending.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Count != 1 {
		t.Fatalf("o filtro por status precisa encontrar a conversa: %+v", listed)
	}
	closed := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/conversations/"+created.Conversation.ID+"/close", `{"reason":"resolvido"}`)
	if closed.Code != http.StatusOK {
		t.Fatalf("encerrar precisa ser 200, veio %d (%s)", closed.Code, closed.Body.String())
	}
	// Mensagem inválida é 400.
	if bad := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/messages", `{"channel":"whatsapp","contact_id":"x"}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("mensagem sem corpo precisa ser 400, veio %d", bad.Code)
	}
}

func TestInboxRoutesAreScopedByOrganization(t *testing.T) {
	engine, runtime := setupInboxAPI(t)
	// Conversa de OUTRA organização criada direto no store.
	other, _, err := runtime.Inbox().AppendInbound("org-terceira", "whatsapp", "+5511988887777", "Zoe", "de outra org", "wamid.other", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := inboxRequest(engine, http.MethodGet, "/api/agent/v1/inbox/conversations/"+other.ID, ""); got.Code == http.StatusOK {
		t.Fatalf("conversa de outra organização não pode ser lida, veio 200 (%s)", got.Body.String())
	}
	list := inboxRequest(engine, http.MethodGet, "/api/agent/v1/inbox/conversations", "")
	var payload struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != 0 {
		t.Fatalf("a listagem não pode trazer conversa de outra organização: %d", payload.Count)
	}
	if got := inboxRequest(engine, http.MethodPost, "/api/agent/v1/inbox/conversations/"+other.ID+"/handoff", `{"reason":"x"}`); got.Code == http.StatusOK {
		t.Fatalf("handoff em conversa de outra organização precisa ser recusado, veio %d", got.Code)
	}
}

// Os três comandos que dependiam da caixa de entrada precisam ter saído do estado
// indisponível — e a guarda do §11.1 já verifica que a rota existe de verdade.
func TestInboxBackedCommandsAreAvailable(t *testing.T) {
	for _, name := range []string{"/inbox", "/handoff", "/support"} {
		spec, ok := agent.SlashCommandSpecByName(name)
		if !ok {
			t.Fatalf("comando %s precisa existir no registro", name)
		}
		if !spec.Available {
			t.Fatalf("%s precisa estar disponível agora que a caixa de entrada existe: %+v", name, spec)
		}
		if !strings.Contains(spec.Backend, "/inbox/") {
			t.Fatalf("%s precisa apontar para as rotas da caixa de entrada, veio %q", name, spec.Backend)
		}
	}
}
