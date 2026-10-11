package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 34: o webhook precisa RECUSAR o que não é autenticado e nunca vazar segredo.
func setupOmnichannelAPI(t *testing.T) (*gin.Engine, *agent.Runtime) {
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
	group.POST("/omnichannel/connectors", api.registerOmnichannelConnector)
	group.GET("/omnichannel/connectors", api.listOmnichannelConnectors)
	group.POST("/omnichannel/webhooks/:channel", api.ingestOmnichannelWebhook)
	group.GET("/omnichannel/events", api.listOmnichannelEvents)
	group.GET("/omnichannel/dlq", api.listOmnichannelDeadLetters)
	group.POST("/omnichannel/dlq/:id/retry", api.retryOmnichannelDeadLetter)
	return engine, runtime
}

func TestOmnichannelWebhookRoutesRejectUnsignedAndAcceptSigned(t *testing.T) {
	engine, runtime := setupOmnichannelAPI(t)
	t.Setenv("HADES_TEST_ROUTE_SECRET", "segredo-de-rota-com-tamanho-suficiente")

	// Registrar o conector pela ROTA: só o NOME da variável aparece na resposta.
	register := httptest.NewRecorder()
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/connectors",
		bytes.NewBufferString(`{"channel":"whatsapp","name":"WhatsApp","secret_env":"HADES_TEST_ROUTE_SECRET"}`))
	registerRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(register, registerRequest)
	if register.Code != http.StatusCreated && register.Code != http.StatusOK {
		t.Fatalf("registrar conector: %d (%s)", register.Code, register.Body.String())
	}
	var connector agent.OmnichannelConnector
	if err := json.Unmarshal(register.Body.Bytes(), &connector); err != nil {
		t.Fatal(err)
	}
	secret := "segredo-de-rota-com-tamanho-suficiente"
	if bytes.Contains(register.Body.Bytes(), []byte(secret)) {
		t.Fatal("a resposta NÃO pode conter o valor do segredo")
	}

	body := []byte(`{"external_message_id":"wamid.rota","contact_id":"+5511999998888","message_type":"text","preview":"olá"}`)
	now := time.Now().UTC()

	// Sem assinatura: 401.
	unsigned := httptest.NewRecorder()
	unsignedRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/webhooks/whatsapp", bytes.NewReader(body))
	unsignedRequest.Header.Set("Content-Type", "application/json")
	unsignedRequest.Header.Set(agent.OmnichannelTimestampHeader, strconv.FormatInt(now.Unix(), 10))
	engine.ServeHTTP(unsigned, unsignedRequest)
	if unsigned.Code != http.StatusUnauthorized {
		t.Fatalf("webhook sem assinatura precisa ser 401, veio %d (%s)", unsigned.Code, unsigned.Body.String())
	}

	// Com assinatura válida: aceito.
	signature, err := runtime.Omnichannel().SignWebhook(connector.ID, now.Unix(), body)
	if err != nil {
		t.Fatal(err)
	}
	signed := httptest.NewRecorder()
	signedRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/webhooks/whatsapp", bytes.NewReader(body))
	signedRequest.Header.Set("Content-Type", "application/json")
	signedRequest.Header.Set(agent.OmnichannelTimestampHeader, strconv.FormatInt(now.Unix(), 10))
	signedRequest.Header.Set(agent.OmnichannelSignatureHeader, signature)
	engine.ServeHTTP(signed, signedRequest)
	if signed.Code != http.StatusAccepted {
		t.Fatalf("webhook assinado precisa ser 202, veio %d (%s)", signed.Code, signed.Body.String())
	}
	if bytes.Contains(signed.Body.Bytes(), []byte(secret)) || bytes.Contains(signed.Body.Bytes(), []byte(signature)) {
		t.Fatal("a resposta não pode devolver segredo nem assinatura")
	}
	var accepted struct {
		Event agent.OmnichannelEvent `json:"event"`
	}
	if err := json.Unmarshal(signed.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Event.Deduplicated || accepted.Event.ConversationID == "" {
		t.Fatalf("o evento aceito precisa apontar para a conversa: %+v", accepted.Event)
	}

	// Reentrega: 200 com deduplicado.
	replay := httptest.NewRecorder()
	replayRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/webhooks/whatsapp", bytes.NewReader(body))
	replayRequest.Header.Set("Content-Type", "application/json")
	replayRequest.Header.Set(agent.OmnichannelTimestampHeader, strconv.FormatInt(now.Unix(), 10))
	replayRequest.Header.Set(agent.OmnichannelSignatureHeader, signature)
	engine.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusOK {
		t.Fatalf("reentrega precisa ser 200, veio %d (%s)", replay.Code, replay.Body.String())
	}
	var repeated struct {
		Deduplicated bool `json:"deduplicated"`
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if !repeated.Deduplicated {
		t.Fatal("a reentrega precisa ser marcada como deduplicada")
	}

	// Carimbo fora da janela: 409 (replay).
	oldTimestamp := now.Add(-2 * agent.OmnichannelReplayWindow).Unix()
	oldSignature, err := runtime.Omnichannel().SignWebhook(connector.ID, oldTimestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	replayOld := httptest.NewRecorder()
	replayOldRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/webhooks/whatsapp", bytes.NewReader(body))
	replayOldRequest.Header.Set("Content-Type", "application/json")
	replayOldRequest.Header.Set(agent.OmnichannelTimestampHeader, strconv.FormatInt(oldTimestamp, 10))
	replayOldRequest.Header.Set(agent.OmnichannelSignatureHeader, oldSignature)
	engine.ServeHTTP(replayOld, replayOldRequest)
	if replayOld.Code != http.StatusConflict {
		t.Fatalf("evento fora da janela precisa ser 409, veio %d (%s)", replayOld.Code, replayOld.Body.String())
	}

	// Canal SEM conector: 404 explícito, não 500 nem aceite.
	unknown := httptest.NewRecorder()
	unknownRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/webhooks/telegram", bytes.NewReader(body))
	unknownRequest.Header.Set("Content-Type", "application/json")
	unknownRequest.Header.Set(agent.OmnichannelTimestampHeader, strconv.FormatInt(now.Unix(), 10))
	unknownRequest.Header.Set(agent.OmnichannelSignatureHeader, signature)
	engine.ServeHTTP(unknown, unknownRequest)
	if unknown.Code == http.StatusAccepted || unknown.Code == http.StatusOK {
		t.Fatalf("canal sem conector não pode ser aceito, veio %d", unknown.Code)
	}

	// Listagens escopadas.
	events := httptest.NewRecorder()
	engine.ServeHTTP(events, httptest.NewRequest(http.MethodGet, "/api/agent/v1/omnichannel/events", nil))
	if events.Code != http.StatusOK {
		t.Fatalf("listar eventos: %d", events.Code)
	}
	connectors := httptest.NewRecorder()
	engine.ServeHTTP(connectors, httptest.NewRequest(http.MethodGet, "/api/agent/v1/omnichannel/connectors", nil))
	var listed struct {
		Connectors []agent.OmnichannelConnector `json:"connectors"`
		Count      int                          `json:"count"`
	}
	if err := json.Unmarshal(connectors.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Count != 1 {
		t.Fatalf("a listagem precisa trazer o conector: %+v", listed)
	}
	// O retry de DLQ inexistente é 404 explícito.
	retry := httptest.NewRecorder()
	retryRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/omnichannel/dlq/dlq_inexistente/retry", nil)
	engine.ServeHTTP(retry, retryRequest)
	if retry.Code != http.StatusNotFound {
		t.Fatalf("retry de DLQ inexistente precisa ser 404, veio %d", retry.Code)
	}
}
