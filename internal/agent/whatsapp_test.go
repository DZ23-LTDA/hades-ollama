package agent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"
)

func signedWhatsAppPayload(body, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// MockWhatsAppAdapter implements WhatsAppAdapter for deterministic testing.
type MockWhatsAppAdapter struct {
	backend       WhatsAppBackendType
	status        GateStatus
	sentMessages  []WhatsAppOutboundMessage
	parseFunc     func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error)
	statusDetails string
}

func NewMockWhatsAppAdapter(backend WhatsAppBackendType, status GateStatus) *MockWhatsAppAdapter {
	return &MockWhatsAppAdapter{
		backend:       backend,
		status:        status,
		sentMessages:  make([]WhatsAppOutboundMessage, 0),
		statusDetails: "mock adapter",
	}
}

func (m *MockWhatsAppAdapter) Backend() WhatsAppBackendType {
	return m.backend
}

func (m *MockWhatsAppAdapter) Status() GateStatus {
	return m.status
}

func (m *MockWhatsAppAdapter) StatusDetails() string {
	return m.statusDetails
}

func (m *MockWhatsAppAdapter) VerifyWebhook(req *http.Request) ([]byte, bool) {
	return []byte("challenge_ok"), true
}

func (m *MockWhatsAppAdapter) ParseWebhook(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
	if m.parseFunc != nil {
		return m.parseFunc(body, header)
	}
	// Default mock parser: expect JSON or return simple inbound
	return []WhatsAppInboundMessage{
		{
			ID:        "msg_mock_1",
			From:      "5511999999999",
			Text:      string(body),
			Type:      "text",
			FromMe:    false,
			Timestamp: time.Now().Unix(),
			Backend:   m.backend,
		},
	}, nil
}

func (m *MockWhatsAppAdapter) SendMessage(ctx context.Context, msg WhatsAppOutboundMessage) (WhatsAppSendResult, error) {
	m.sentMessages = append(m.sentMessages, msg)
	return WhatsAppSendResult{
		MessageID: "sent_" + msg.To,
		Status:    "delivered",
	}, nil
}

// 1. Sem credencial = NOT_CONFIGURED
func TestWhatsAppAdapterStatusWithoutCredentials(t *testing.T) {
	// Evolution API without config
	evo := NewEvolutionAdapter(EvolutionConfig{})
	if evo.Status() != GateStatusNotConfigured {
		t.Fatalf("expected evolution adapter status %s without config, got %s", GateStatusNotConfigured, evo.Status())
	}
	if !strings.Contains(evo.StatusDetails(), "não configurada") {
		t.Fatalf("expected clear details for unconfigured evolution, got %s", evo.StatusDetails())
	}

	// Cloud API without config
	cloud := NewCloudAPIAdapter(CloudAPIConfig{})
	if cloud.Status() != GateStatusNotConfigured {
		t.Fatalf("expected cloud adapter status %s without config, got %s", GateStatusNotConfigured, cloud.Status())
	}
	if !strings.Contains(cloud.StatusDetails(), "não configurada") {
		t.Fatalf("expected clear details for unconfigured cloud api, got %s", cloud.StatusDetails())
	}

	// Gateway with empty config must report NOT_CONFIGURED
	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{}, nil)
	if gw.Status() != GateStatusNotConfigured {
		t.Fatalf("expected gateway status %s without config, got %s", GateStatusNotConfigured, gw.Status())
	}
}

func TestWhatsAppCloudWebhookHMACAndIdentity(t *testing.T) {
	adapter := NewCloudAPIAdapter(CloudAPIConfig{PhoneNumberID: "phone-001", AppSecret: "app-secret"})
	body := `{"object":"whatsapp","entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"phone-001"}}}]}]}`
	header := http.Header{"X-Hub-Signature-256": []string{signedWhatsAppPayload(body, "app-secret")}}
	if err := adapter.ValidateWebhook([]byte(body), header); err != nil {
		t.Fatalf("valid webhook rejected: %v", err)
	}
	header.Set("X-Hub-Signature-256", signedWhatsAppPayload(body, "wrong-secret"))
	if err := adapter.ValidateWebhook([]byte(body), header); err != ErrWhatsAppInvalidSignature {
		t.Fatalf("invalid signature must be rejected, got %v", err)
	}
	missingSecret := NewCloudAPIAdapter(CloudAPIConfig{PhoneNumberID: "phone-001"})
	if err := missingSecret.ValidateWebhook([]byte(body), http.Header{}); err != ErrWhatsAppWebhookNotConfigured {
		t.Fatalf("missing app secret must be NOT_CONFIGURED, got %v", err)
	}
	badObject := `{"object":"not-whatsapp","entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"phone-001"}}}]}]}`
	if err := adapter.ValidateWebhook([]byte(badObject), http.Header{"X-Hub-Signature-256": []string{signedWhatsAppPayload(badObject, "app-secret")}}); err != ErrWhatsAppInvalidPayload {
		t.Fatalf("wrong object must be rejected, got %v", err)
	}
	badPhone := `{"object":"whatsapp","entry":[{"changes":[{"value":{"metadata":{"phone_number_id":"phone-999"}}}]}]}`
	if err := adapter.ValidateWebhook([]byte(badPhone), http.Header{"X-Hub-Signature-256": []string{signedWhatsAppPayload(badPhone, "app-secret")}}); err != ErrWhatsAppInvalidPayload {
		t.Fatalf("wrong phone_number_id must be rejected, got %v", err)
	}
}

func TestWhatsAppOutboundRequiresAllowlistAndApproval(t *testing.T) {
	mock := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{Allowlist: []WhatsAppContactPolicy{
		{PhoneNumber: "5511999999999", Role: ContactRoleOwner, Allowed: true},
		{PhoneNumber: "5511888888888", Role: ContactRoleOperator, Allowed: true},
	}}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mock)
	msg := WhatsAppOutboundMessage{To: "5511888888888", Text: "mensagem controlada"}
	if _, err := gw.SendMessage(context.Background(), msg); err != ErrWhatsAppOutboundApproval {
		t.Fatalf("missing HITL approval must block, got %v", err)
	}
	msg.To = "5511777777777"
	if _, err := gw.SendMessage(context.Background(), msg); err != ErrWhatsAppUnauthorized {
		t.Fatalf("recipient outside allowlist must block, got %v", err)
	}
	msg.To = "5511888888888"
	approvalID, err := gw.RequestOutboundApproval("5511999999999", msg)
	if err != nil {
		t.Fatalf("request approval: %v", err)
	}
	if err := gw.ApproveOutbound(approvalID, "5511999999999"); err != nil {
		t.Fatalf("approve outbound: %v", err)
	}
	msg.ApprovalID = approvalID
	if _, err := gw.SendMessage(context.Background(), msg); err != nil {
		t.Fatalf("approved allowlisted outbound rejected: %v", err)
	}
}

// 2. fromMe loop filter
func TestWhatsAppGatewayFromMeLoopFilter(t *testing.T) {
	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_loop_test",
				From:      "5511999999999",
				Text:      "mensagem enviada pelo próprio bot",
				FromMe:    true, // From me!
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	results, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("payload"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != "ignored_from_me" {
		t.Fatalf("expected status 'ignored_from_me', got %s", results[0].Status)
	}

	if len(mockAdp.sentMessages) > 0 {
		t.Fatalf("bot must not reply to fromMe messages (loop protection), sent: %d", len(mockAdp.sentMessages))
	}
}

// 3. Dedupe / Idempotence
func TestWhatsAppGatewayDedupeIdempotence(t *testing.T) {
	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "unique_msg_id_12345",
				From:      "5511999999999",
				Text:      "status",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	// First delivery: should process
	res1, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("test"), nil)
	if err != nil {
		t.Fatalf("first delivery failed: %v", err)
	}
	if len(res1) != 1 || res1[0].Status != "processed" {
		t.Fatalf("first delivery expected 'processed', got %+v", res1)
	}
	if len(mockAdp.sentMessages) != 1 {
		t.Fatalf("expected 1 reply sent on first delivery, got %d", len(mockAdp.sentMessages))
	}

	// Second delivery (duplicate ID): should be ignored
	res2, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("test"), nil)
	if err != nil {
		t.Fatalf("second delivery failed: %v", err)
	}
	if len(res2) != 1 || res2[0].Status != "ignored_duplicate" {
		t.Fatalf("expected duplicate to be ignored, got status %s", res2[0].Status)
	}
	if len(mockAdp.sentMessages) != 1 {
		t.Fatalf("reply must not be resent for duplicate message, total sent: %d", len(mockAdp.sentMessages))
	}
}

// 4. Allowlist & Unauthorized Bloqueado
func TestWhatsAppGatewayAllowlistAndUnauthorizedBlocked(t *testing.T) {
	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_unauth_1",
				From:      "5511888888888", // Not in allowlist!
				Text:      "health",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	res, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("test"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}

	if res[0].Status != "unauthorized" {
		t.Fatalf("expected status 'unauthorized', got %s", res[0].Status)
	}

	// No message should be sent to unauthorized contact
	if len(mockAdp.sentMessages) > 0 {
		t.Fatalf("expected zero messages sent to unauthorized contact, got %d", len(mockAdp.sentMessages))
	}

	// Now add the number to allowlist and verify it is allowed
	gw.SetContactPolicy(WhatsAppContactPolicy{
		PhoneNumber: "5511888888888",
		Name:        "Operator",
		Role:        ContactRoleOperator,
		Allowed:     true,
	})

	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_now_auth_2",
				From:      "5511888888888",
				Text:      "health",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	resAuth, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("test"), nil)
	if err != nil {
		t.Fatalf("unexpected error after authorization: %v", err)
	}
	if len(resAuth) != 1 || resAuth[0].Status != "processed" {
		t.Fatalf("expected 'processed' after authorization, got %+v", resAuth)
	}
}

// 5. Sensitive Command Classification & Approval HITL
func TestWhatsAppGatewaySensitiveCommandRequiresApproval(t *testing.T) {
	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)

	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	// Step 1: Send sensitive command "deletar banco de dados"
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_sensitive_1",
				From:      "5511999999999",
				Text:      "deletar banco de dados de staging",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	res, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("cmd"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 || res[0].Status != "awaiting_approval" {
		t.Fatalf("expected status 'awaiting_approval', got %+v", res)
	}

	if !strings.Contains(res[0].ReplySent, "Aprovação necessária (HITL)") {
		t.Fatalf("expected approval notice in reply, got %s", res[0].ReplySent)
	}

	// Extract approval ID from reply
	// "APROVAR appr_xxxx"
	reply := res[0].ReplySent
	idx := strings.Index(reply, "APROVAR appr_")
	if idx == -1 {
		t.Fatalf("could not find approval id in reply: %s", reply)
	}
	apprID := strings.Trim(strings.Fields(reply[idx:])[1], "*_ `")

	// Step 2: Test Rejection
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_reject_decision",
				From:      "5511999999999",
				Text:      "REJEITAR " + apprID,
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	resRej, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("rej"), nil)
	if err != nil {
		t.Fatalf("rejection webhook failed: %v", err)
	}
	if !strings.Contains(resRej[0].ReplySent, "Ação rejeitada") {
		t.Fatalf("expected rejection confirmation, got %s", resRej[0].ReplySent)
	}

	// Step 3: Trigger new approval and Approve it
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_sensitive_2",
				From:      "5511999999999",
				Text:      "deploy prod release 1.0",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	resApprReq, _ := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("appr_req"), nil)
	replyAppr := resApprReq[0].ReplySent
	idx2 := strings.Index(replyAppr, "APROVAR appr_")
	apprID2 := strings.Trim(strings.Fields(replyAppr[idx2:])[1], "*_ `")

	// Approve as Owner
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_approve_decision",
				From:      "5511999999999",
				Text:      "APROVAR " + apprID2,
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	resApproved, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("ok"), nil)
	if err != nil {
		t.Fatalf("approval decision failed: %v", err)
	}
	if !strings.Contains(resApproved[0].ReplySent, "Ação aprovada e executada") {
		t.Fatalf("expected approval execution confirmation, got %s", resApproved[0].ReplySent)
	}
}

// 6. Command Bridge Cria Missão
func TestWhatsAppGatewayCommandBridgeCreatesMission(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)

	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, runtime)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	// Send "/goal auditar segurança da fábrica"
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "msg_goal_bridge_1",
				From:      "5511999999999",
				Text:      "/goal auditar segurança da fábrica",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	res, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("goal"), nil)
	if err != nil {
		t.Fatalf("webhook process failed: %v", err)
	}

	if len(res) != 1 || res[0].Status != "processed" {
		t.Fatalf("expected status 'processed', got %+v", res)
	}

	// A ponte cria a missão, mas não a executa: o objetivo veio de texto
	// externo e quem inicia é um operador autenticado.
	if !strings.Contains(res[0].ReplySent, "Missão criada via WhatsApp Bridge") {
		t.Fatalf("expected mission bridge response, got %s", res[0].ReplySent)
	}
	if !strings.Contains(res[0].ReplySent, "PENDENTE") {
		t.Fatalf("mission created from an external message must not start by itself, got %s", res[0].ReplySent)
	}

	// Verify mission was created in runtime store
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("list missions: %v", err)
	}
	if len(missions) != 1 {
		t.Fatalf("expected 1 mission created in runtime, got %d", len(missions))
	}
	if missions[0].Objective != "auditar segurança da fábrica" {
		t.Fatalf("expected objective 'auditar segurança da fábrica', got %s", missions[0].Objective)
	}
	// A missão criada por mensagem externa não pode ter começado sozinha: só um
	// operador autenticado a inicia.
	if missions[0].State == MissionRunning || missions[0].State == MissionCompleted {
		t.Fatalf("mission from an external message must remain unstarted, got state %s", missions[0].State)
	}
	// Checar o estado não basta: uma missão enfileirada permanece READY. A ponte
	// também não pode tê-la colocado na fila de execução.
	for _, status := range []QueueStatus{QueuePending, QueueRunning} {
		for _, job := range runtime.QueueJobs(status) {
			if job.MissionID == missions[0].ID {
				t.Fatalf("mission from an external message must not be enqueued, found %s job %s", status, job.ID)
			}
		}
	}
}

// 7. Intent Routing (Health, Resumo, Lembrete, Ajuda)
func TestWhatsAppGatewayIntentRouting(t *testing.T) {
	runtime, _ := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})

	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, runtime)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	tests := []struct {
		input       string
		expectedSub string
	}{
		{"ajuda", "Comandos via Celular"},
		{"health", "Saúde do Sistema"},
		{"resumo", "Resumo Executivo Diário"},
		{"lembrete testar gateway às 14h", "Lembrete registrado com sucesso"},
	}

	for _, tt := range tests {
		mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
			return []WhatsAppInboundMessage{
				{
					ID:        "msg_" + tt.input,
					From:      "5511999999999",
					Text:      tt.input,
					FromMe:    false,
					Timestamp: time.Now().Unix(),
					Backend:   WhatsAppBackendEvolution,
				},
			}, nil
		}

		res, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte(tt.input), nil)
		if err != nil {
			t.Fatalf("intent %s failed: %v", tt.input, err)
		}
		if !strings.Contains(res[0].ReplySent, tt.expectedSub) {
			t.Fatalf("intent '%s': expected reply to contain '%s', got '%s'", tt.input, tt.expectedSub, res[0].ReplySent)
		}
	}
}

// 8. Optional Capabilities Degradation (STT, TTS, Vision)
func TestWhatsAppGatewayOptionalCapabilitiesDegradation(t *testing.T) {
	mockAdp := NewMockWhatsAppAdapter(WhatsAppBackendEvolution, GateStatusPass)
	// Without media manager, STT and TTS must report NOT_CONFIGURED honestly
	gw := NewWhatsAppGateway(WhatsAppGatewayConfig{
		Allowlist: []WhatsAppContactPolicy{
			{PhoneNumber: "5511999999999", Name: "Owner", Role: ContactRoleOwner, Allowed: true},
		},
	}, nil)
	gw.SetAdapter(WhatsAppBackendEvolution, mockAdp)

	// Send Audio when STT not configured
	mockAdp.parseFunc = func(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
		return []WhatsAppInboundMessage{
			{
				ID:        "audio_msg_1",
				From:      "5511999999999",
				Type:      "audio",
				MediaURL:  "https://example.com/audio.ogg",
				FromMe:    false,
				Timestamp: time.Now().Unix(),
				Backend:   WhatsAppBackendEvolution,
			},
		}, nil
	}

	res, err := gw.ProcessWebhook(context.Background(), WhatsAppBackendEvolution, []byte("audio"), nil)
	if err != nil {
		t.Fatalf("process audio failed: %v", err)
	}

	if !strings.Contains(res[0].ReplySent, "NOT_CONFIGURED") {
		t.Fatalf("expected honest NOT_CONFIGURED for audio STT, got: %s", res[0].ReplySent)
	}
}
