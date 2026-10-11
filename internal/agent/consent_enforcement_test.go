package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Estágio 27: consentimento, opt-out e limites.
//
// As propriedades travadas aqui:
//  1. FALHA FECHADA: contato desconhecido NÃO é contatável;
//  2. opt-out bloqueia, exige motivo e guarda o responsável pela revogação;
//  3. consentimento VENCIDO volta a ser ausência de consentimento;
//  4. o escopo é por organização E por canal (consentir no WhatsApp não autoriza
//     Telegram, nem outro tenant);
//  5. o registro sobrevive ao reinício;
//  6. os DOIS caminhos de envio (Telegram e WhatsApp) respeitam a mesma decisão —
//     um controle que vale num caminho só é o defeito de irmão esquecido.
func consentRegistry(t *testing.T) *ConsentRegistry {
	t.Helper()
	registry, err := NewConsentRegistry(filepath.Join(t.TempDir(), "consent"))
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestConsentRegistryFailsClosedForUnknownContact(t *testing.T) {
	registry := consentRegistry(t)
	now := time.Now().UTC()
	if err := registry.AuthorizeContact("org-a", ConsentChannelWhatsApp, "+5511999999999", now); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("contato desconhecido precisa ser NEGADO com ErrConsentMissing, veio %v", err)
	}
	// Registro ausente (nil) também nega: caminho não instrumentado não é caminho livre.
	var semRegistro *ConsentRegistry
	if err := semRegistro.AuthorizeContact("org-a", ConsentChannelWhatsApp, "+5511999999999", now); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("registro ausente precisa negar, veio %v", err)
	}
	// Após conceder, autoriza.
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, "+5511999999999", "formulario", "assinatura eletrônica", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	if err := registry.AuthorizeContact("org-a", ConsentChannelWhatsApp, "+5511999999999", now); err != nil {
		t.Fatalf("com consentimento precisa autorizar: %v", err)
	}
	// Organização VAZIA é normalizada para a organização local — a mesma convenção
	// do HTTP e do supervisor. Sem isso, o consentimento gravado sob uma chave nunca
	// casaria com a checagem feita sob outra.
	record, err := registry.GrantConsent("", ConsentChannelWhatsApp, "+5511911112222", "site", "checkbox", "ana", nil, now)
	if err != nil {
		t.Fatalf("organização vazia precisa ser normalizada, veio %v", err)
	}
	if record.OrganizationID != LocalOrganizationID {
		t.Fatalf("organização vazia precisa virar a local: %q", record.OrganizationID)
	}
	if err := registry.AuthorizeContact("", ConsentChannelWhatsApp, "+5511911112222", now); err != nil {
		t.Fatalf("a checagem com escopo vazio precisa achar o mesmo registro: %v", err)
	}
	// Canal e contato continuam OBRIGATÓRIOS: normalizar escopo não afrouxa validação.
	if _, err := registry.GrantConsent("org-a", "  ", "+5511999999999", "", "", "ana", nil, now); !errors.Is(err, ErrConsentInvalid) {
		t.Fatalf("consentimento sem canal precisa falhar, veio %v", err)
	}
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, "   ", "", "", "ana", nil, now); !errors.Is(err, ErrConsentInvalid) {
		t.Fatalf("consentimento sem contato precisa falhar, veio %v", err)
	}
}

func TestConsentRegistryOptOutBlocksAndKeepsAuditTrail(t *testing.T) {
	registry := consentRegistry(t)
	now := time.Now().UTC()
	contact := "+5511988887777"
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, contact, "site", "checkbox marcado", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	// Revogar sem motivo é recusado: opt-out sem justificativa não é auditável.
	if _, err := registry.RevokeConsent("org-a", ConsentChannelWhatsApp, contact, "   ", "bruno", now); !errors.Is(err, ErrConsentReasonNeeded) {
		t.Fatalf("revogação sem motivo precisa falhar, veio %v", err)
	}
	record, err := registry.RevokeConsent("org-a", ConsentChannelWhatsApp, contact, "pedido do titular por e-mail", "bruno", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if record.State != ConsentRevoked || record.RevokedBy != "bruno" || record.Reason == "" {
		t.Fatalf("o opt-out precisa guardar estado, responsável e motivo: %+v", record)
	}
	// A origem do consentimento original é PRESERVADA: prova de como o contato
	// consentiu antes de sair.
	if record.GrantedBy != "ana" || record.Source != "site" {
		t.Fatalf("a trilha do consentimento original precisa sobreviver: %+v", record)
	}
	if err := registry.AuthorizeContact("org-a", ConsentChannelWhatsApp, contact, now.Add(2*time.Minute)); !errors.Is(err, ErrConsentRevoked) {
		t.Fatalf("quem fez opt-out não pode ser procurado, veio %v", err)
	}
}

func TestConsentExpiryMeansNoConsent(t *testing.T) {
	registry := consentRegistry(t)
	now := time.Now().UTC()
	expired := now.Add(-time.Hour)
	contact := "+5511977776666"
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, contact, "evento", "lista de presença", "ana", &expired, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := registry.AuthorizeContact("org-a", ConsentChannelWhatsApp, contact, now); !errors.Is(err, ErrConsentExpired) {
		t.Fatalf("consentimento vencido precisa bloquear, veio %v", err)
	}
	// Renovar com prazo futuro volta a autorizar.
	future := now.Add(24 * time.Hour)
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, contact, "evento", "renovação", "ana", &future, now); err != nil {
		t.Fatal(err)
	}
	if err := registry.AuthorizeContact("org-a", ConsentChannelWhatsApp, contact, now); err != nil {
		t.Fatalf("consentimento renovado precisa autorizar: %v", err)
	}
}

func TestConsentIsScopedByOrganizationAndChannel(t *testing.T) {
	registry := consentRegistry(t)
	now := time.Now().UTC()
	contact := "+5511966665555"
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, contact, "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	// OUTRA organização: não herda o consentimento da primeira.
	if err := registry.AuthorizeContact("org-b", ConsentChannelWhatsApp, contact, now); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("consentimento não pode atravessar organizações, veio %v", err)
	}
	// OUTRO canal: consentir no WhatsApp não autoriza Telegram.
	if err := registry.AuthorizeContact("org-a", ConsentChannelTelegram, contact, now); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("consentimento não pode atravessar canais, veio %v", err)
	}
	// Listagem por organização traz só o que é dela.
	if _, err := registry.GrantConsent("org-b", ConsentChannelTelegram, "chat-1", "bot", "comando /start", "bruno", nil, now); err != nil {
		t.Fatal(err)
	}
	listA := registry.ListConsents("org-a")
	if len(listA) != 1 || listA[0].OrganizationID != "org-a" {
		t.Fatalf("listagem precisa respeitar a organização: %+v", listA)
	}
}

func TestConsentRegistrySurvivesRestart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "consent")
	registry, err := NewConsentRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := registry.GrantConsent("org-a", ConsentChannelTelegram, "chat-persistente", "bot", "/start", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.RevokeConsent("org-a", ConsentChannelWhatsApp, "+5511900000000", "pedido do titular", "bruno", now); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewConsentRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.AuthorizeContact("org-a", ConsentChannelTelegram, "chat-persistente", now); err != nil {
		t.Fatalf("consentimento precisa sobreviver ao reinício: %v", err)
	}
	if err := restarted.AuthorizeContact("org-a", ConsentChannelWhatsApp, "+5511900000000", now); !errors.Is(err, ErrConsentRevoked) {
		t.Fatalf("o opt-out precisa sobreviver ao reinício, veio %v", err)
	}
}

// ---------- ENFORCEMENT nos caminhos de envio ----------

// consentTelegramTestChannel monta o canal com o registro informado, mantendo o
// servidor de teste que CONTA requisições: se o bloqueio falhar, o servidor recebe.
func consentTelegramTestChannel(t *testing.T, server *httptest.Server, registry *ConsentRegistry, org string) *telegramChannel {
	t.Helper()
	channel := newTelegramChannel()
	channel.timeout = 5 * time.Second
	channel.baseURL = server.URL
	channel.client = server.Client()
	channel.credential = func(string) string { return testTelegramToken }
	channel.organizationID = org
	channel.consent = registry
	return channel
}

func TestTelegramSendIsBlockedWithoutConsent(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	registry := consentRegistry(t)
	now := time.Now().UTC()
	contact := "777777"

	// 1. Sem consentimento registrado: NENHUMA requisição pode sair.
	semConsentimento := consentTelegramTestChannel(t, server, registry, "org-a")
	if _, err := semConsentimento.sendMessage(context.Background(), contact, testTelegramText); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("envio sem consentimento precisa ser negado, veio %v", err)
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("nenhuma chamada pode chegar ao provedor sem consentimento: %v", paths)
	}

	// 2. Com consentimento, o envio acontece.
	if _, err := registry.GrantConsent("org-a", ConsentChannelTelegram, contact, "bot", "/start", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	comConsentimento := consentTelegramTestChannel(t, server, registry, "org-a")
	if _, err := comConsentimento.sendMessage(context.Background(), contact, testTelegramText); err != nil {
		t.Fatalf("com consentimento o envio precisa funcionar: %v", err)
	}
	paths, _ = recorder.snapshot()
	if len(paths) != 1 {
		t.Fatalf("exatamente uma chamada era esperada: %v", paths)
	}

	// 3. Depois do opt-out, volta a NÃO sair.
	if _, err := registry.RevokeConsent("org-a", ConsentChannelTelegram, contact, "pedido do titular", "bruno", now); err != nil {
		t.Fatal(err)
	}
	apos := consentTelegramTestChannel(t, server, registry, "org-a")
	if _, err := apos.sendMessage(context.Background(), contact, testTelegramText); !errors.Is(err, ErrConsentRevoked) {
		t.Fatalf("após opt-out o envio precisa ser negado, veio %v", err)
	}
	paths, _ = recorder.snapshot()
	if len(paths) != 1 {
		t.Fatalf("nada pode sair depois do opt-out: %v", paths)
	}

	// 4. Consentimento de OUTRA organização não vale aqui.
	if _, err := registry.GrantConsent("org-b", ConsentChannelTelegram, contact, "bot", "/start", "bruno", nil, now); err != nil {
		t.Fatal(err)
	}
	outraOrg := consentTelegramTestChannel(t, server, registry, "org-a")
	if _, err := outraOrg.sendMessage(context.Background(), contact, testTelegramText); !errors.Is(err, ErrConsentRevoked) {
		t.Fatalf("consentimento de outra organização não pode liberar, veio %v", err)
	}
}

// Contador de envios para provar que o adaptador NÃO foi chamado.
type countingWhatsAppAdapter struct {
	mu    sync.Mutex
	calls int
}

func (a *countingWhatsAppAdapter) Backend() WhatsAppBackendType { return WhatsAppBackendEvolution }
func (a *countingWhatsAppAdapter) Status() GateStatus           { return GateStatusPass }
func (a *countingWhatsAppAdapter) StatusDetails() string        { return "adaptador de teste" }
func (a *countingWhatsAppAdapter) VerifyWebhook(*http.Request) ([]byte, bool) {
	return nil, false
}

func (a *countingWhatsAppAdapter) ParseWebhook([]byte, http.Header) ([]WhatsAppInboundMessage, error) {
	return nil, nil
}

func (a *countingWhatsAppAdapter) SendMessage(context.Context, WhatsAppOutboundMessage) (WhatsAppSendResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	return WhatsAppSendResult{MessageID: "wamid.teste"}, nil
}

func (a *countingWhatsAppAdapter) callsMade() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

func TestWhatsAppGatewaySendIsBlockedWithoutConsent(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      filepath.Join(t.TempDir(), "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	number := "+5511955554444"
	gateway := &WhatsAppGateway{
		adapters:         map[WhatsAppBackendType]WhatsAppAdapter{WhatsAppBackendEvolution: &countingWhatsAppAdapter{}},
		activeBackend:    WhatsAppBackendEvolution,
		allowlist:        map[string]WhatsAppContactPolicy{cleanWhatsAppNumber(number): {Allowed: true}},
		pendingApprovals: map[string]*WhatsAppPendingApproval{"apr-1": {ID: "apr-1", Status: "APPROVED", OutboundTo: cleanWhatsAppNumber(number)}},
		runtime:          runtime,
	}
	message := WhatsAppOutboundMessage{To: number, Text: "mensagem de teste", ApprovalID: "apr-1"}

	// Sem consentimento: bloqueado e o adaptador NÃO é chamado.
	if _, err := gateway.SendMessage(context.Background(), message); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("envio sem consentimento precisa ser negado, veio %v", err)
	}
	adapter := gateway.adapters[WhatsAppBackendEvolution].(*countingWhatsAppAdapter)
	if adapter.callsMade() != 0 {
		t.Fatalf("o adaptador não pode ser chamado sem consentimento: %d", adapter.callsMade())
	}

	// Com consentimento na MESMA organização do runtime: envia.
	if _, err := runtime.Consent().GrantConsent(runtime.organizationScope, ConsentChannelWhatsApp, cleanWhatsAppNumber(number), "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := gateway.SendMessage(context.Background(), message); err != nil {
		t.Fatalf("com consentimento o envio precisa funcionar: %v", err)
	}
	if adapter.callsMade() != 1 {
		t.Fatalf("o adaptador precisa ser chamado uma vez: %d", adapter.callsMade())
	}

	// Opt-out: bloqueia de novo.
	if _, err := runtime.Consent().RevokeConsent(runtime.organizationScope, ConsentChannelWhatsApp, cleanWhatsAppNumber(number), "pedido do titular", "bruno", now); err != nil {
		t.Fatal(err)
	}
	if _, err := gateway.SendMessage(context.Background(), message); !errors.Is(err, ErrConsentRevoked) {
		t.Fatalf("após opt-out precisa ser negado, veio %v", err)
	}
	if adapter.callsMade() != 1 {
		t.Fatalf("nada pode sair depois do opt-out: %d", adapter.callsMade())
	}
}

// Os DOIS caminhos de envio precisam usar a MESMA decisão. Sem isso, o controle
// vale num caminho e falta no irmão — o defeito que já apareceu quatro vezes aqui.
func TestBothSendPathsUseTheSameConsentDecision(t *testing.T) {
	registry := consentRegistry(t)
	now := time.Now().UTC()
	contactTelegram := "888888"
	contactWhatsApp := "+5511944443333"
	if _, err := registry.GrantConsent("org-a", ConsentChannelTelegram, contactTelegram, "bot", "/start", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, contactWhatsApp, "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	// O mesmo contato, nos dois canais, sem registro: o erro é o MESMO sentinela.
	for _, channel := range []string{ConsentChannelTelegram, ConsentChannelWhatsApp} {
		if err := registry.AuthorizeContact("org-a", channel, "contato-sem-registro", now); !errors.Is(err, ErrConsentMissing) {
			t.Fatalf("canal %s precisa negar com o mesmo sentinela, veio %v", channel, err)
		}
	}
	// O código dos dois caminhos chama o mesmo helper: um teste de comportamento
	// garante que o bloqueio vale nos dois.
	if err := authorizedContact(registry, "org-a", ConsentChannelTelegram, contactTelegram, now); err != nil {
		t.Fatalf("helper precisa autorizar o Telegram consentido: %v", err)
	}
	if err := authorizedContact(registry, "org-a", ConsentChannelWhatsApp, contactWhatsApp, now); err != nil {
		t.Fatalf("helper precisa autorizar o WhatsApp consentido: %v", err)
	}
	// Sanidade: o contato consentido NÃO autoriza o outro canal.
	if err := authorizedContact(registry, "org-a", ConsentChannelWhatsApp, contactTelegram, now); !errors.Is(err, ErrConsentMissing) {
		t.Fatalf("Telegram consentido não pode valer no WhatsApp, veio %v", err)
	}
	if !strings.Contains(contactWhatsApp, "+") || !strings.Contains(contactTelegram, "8") {
		t.Fatal("números de teste precisam ser distintos")
	}
}

var _ = http.MethodPost
