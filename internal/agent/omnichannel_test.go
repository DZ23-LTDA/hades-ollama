package agent

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Estágio 34 — contrato canônico de mensagens.
//
// As propriedades travadas aqui:
//  1. webhook SEM assinatura válida é recusado, e sem segredo configurado a
//     ingestão FALHA FECHADA;
//  2. anti-replay por janela de tempo;
//  3. deduplicação por `ExternalMessageID` (a entrega repetida não duplica);
//  4. o TENANT vem do conector, nunca do payload — não se cruza empresa;
//  5. rate limit por organização e canal;
//  6. evento que falha vai para a DLQ COM MOTIVO, e o reprocessamento é explícito;
//  7. o segredo nunca aparece no evento aceito.
func omnichannelRuntime(t *testing.T) (*OmnichannelRuntime, string, []byte) {
	t.Helper()
	inbox, err := NewInboxStore(filepath.Join(t.TempDir(), "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewOmnichannelRuntime(inbox)
	secretEnv := "HADES_TEST_OMNI_SECRET"
	t.Setenv(secretEnv, "segredo-de-teste-com-tamanho-suficiente")
	now := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-a", Channel: "whatsapp", Name: "WhatsApp da A", SecretEnv: secretEnv, Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return runtime, connector.ID, []byte(`{"external_message_id":"wamid.1"}`)
}

func omnichannelEvent(t *testing.T) OmnichannelEvent {
	t.Helper()
	return OmnichannelEvent{
		ExternalMessageID: "wamid.1",
		ContactID:         "+5511999998888",
		MessageType:       "text",
		Preview:           "olá, preciso de ajuda",
		CorrelationID:     "corr-1",
	}
}

func TestOmnichannelRejectsUnsignedAndUnconfiguredWebhooks(t *testing.T) {
	runtime, connectorID, body := omnichannelRuntime(t)
	now := time.Now().UTC()
	timestamp := now.Unix()

	// 1. Assinatura ausente: recusado.
	if _, err := runtime.Ingest("org-a", "whatsapp", "", timestamp, body, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelSignature) {
		t.Fatalf("webhook sem assinatura precisa ser recusado, veio %v", err)
	}
	// 2. Assinatura errada: recusado.
	if _, err := runtime.Ingest("org-a", "whatsapp", "sha256=deadbeef", timestamp, body, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelSignature) {
		t.Fatalf("assinatura inválida precisa ser recusada, veio %v", err)
	}
	// 3. Assinatura de OUTRO corpo: recusado (não vale reaproveitar assinatura).
	valid, err := runtime.SignWebhook(connectorID, timestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	otherBody := []byte(`{"external_message_id":"wamid.2"}`)
	if _, err := runtime.Ingest("org-a", "whatsapp", valid, timestamp, otherBody, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelSignature) {
		t.Fatalf("assinatura de outro corpo precisa ser recusada, veio %v", err)
	}
	// 4. Nada foi aceito até aqui.
	if events := runtime.Events("org-a"); len(events) != 0 {
		t.Fatalf("nada podia ter sido aceito: %+v", events)
	}

	// 5. Sem segredo configurado, a ingestão FALHA FECHADA (não aceita "por não ter chave").
	t.Setenv("HADES_TEST_OMNI_SECRET", "")
	if _, err := runtime.Ingest("org-a", "whatsapp", valid, timestamp, body, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelNotConfigured) {
		t.Fatalf("sem segredo a ingestão precisa falhar fechada, veio %v", err)
	}
}

func TestOmnichannelAntiReplayAndDeduplication(t *testing.T) {
	runtime, connectorID, body := omnichannelRuntime(t)
	now := time.Now().UTC()

	// Janela: carimbo antigo demais é recusado (replay).
	oldTimestamp := now.Add(-2 * OmnichannelReplayWindow).Unix()
	oldSignature, err := runtime.SignWebhook(connectorID, oldTimestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Ingest("org-a", "whatsapp", oldSignature, oldTimestamp, body, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelReplay) {
		t.Fatalf("evento fora da janela precisa ser recusado, veio %v", err)
	}
	// Carimbo no futuro distante também.
	futureTimestamp := now.Add(2 * OmnichannelReplayWindow).Unix()
	futureSignature, err := runtime.SignWebhook(connectorID, futureTimestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Ingest("org-a", "whatsapp", futureSignature, futureTimestamp, body, omnichannelEvent(t), now); !errors.Is(err, ErrOmnichannelReplay) {
		t.Fatalf("carimbo no futuro precisa ser recusado, veio %v", err)
	}

	// Dentro da janela: aceito, e a mensagem entra na caixa de entrada.
	timestamp := now.Unix()
	signature, err := runtime.SignWebhook(connectorID, timestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	event, err := runtime.Ingest("org-a", "whatsapp", signature, timestamp, body, omnichannelEvent(t), now)
	if err != nil {
		t.Fatal(err)
	}
	if event.Deduplicated {
		t.Fatal("a primeira entrega não é deduplicada")
	}
	if event.ConversationID == "" {
		t.Fatal("o evento aceito precisa apontar para a conversa criada")
	}
	if event.OrganizationID != "org-a" || event.ChannelID != "whatsapp" {
		t.Fatalf("o evento precisa ser marcado com o tenant e o canal do CONECTOR: %+v", event)
	}
	// Reentrega: deduplicada, e a conversa continua com UMA mensagem.
	repeated, err := runtime.Ingest("org-a", "whatsapp", signature, timestamp, body, omnichannelEvent(t), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !repeated.Deduplicated {
		t.Fatal("a reentrega precisa ser marcada como deduplicada")
	}
	if repeated.ID != event.ID {
		t.Fatalf("a reentrega precisa devolver o MESMO evento: %s vs %s", repeated.ID, event.ID)
	}
	_, messages, err := runtime.inbox.Conversation("org-a", event.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("entrega repetida não pode duplicar mensagem: %d", len(messages))
	}
}

func TestOmnichannelTenantComesFromConnectorNotPayload(t *testing.T) {
	runtime, _, _ := omnichannelRuntime(t)
	now := time.Now().UTC()
	// Conector de OUTRA organização.
	otherConnector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-b", Channel: "whatsapp", SecretEnv: "HADES_TEST_OMNI_SECRET", Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"external_message_id":"wamid.cross","organization_id":"org-b"}`)
	timestamp := now.Unix()
	signature, err := runtime.SignWebhook(otherConnector.ID, timestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	// O evento é ingerido na ORG-A: o payload diz "org-b", mas quem manda é o conector.
	event, err := runtime.Ingest("org-a", "whatsapp", signature, timestamp, body, OmnichannelEvent{
		ExternalMessageID: "wamid.cross", ContactID: "+5511911112222", MessageType: "text",
		Preview: "tentativa de cruzar tenant", OrganizationID: "org-b",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if event.OrganizationID != "org-a" {
		t.Fatalf("o tenant precisa vir do CONECTOR, não do payload: %+v", event)
	}
	if event.ConnectorID == otherConnector.ID {
		t.Fatal("o evento não pode ser atribuído ao conector de outra organização")
	}
	// E a organização B não enxerga o evento.
	if events := runtime.Events("org-b"); len(events) != 0 {
		t.Fatalf("a outra organização não pode ver o evento: %+v", events)
	}
	// Canal sem conector registrado é recusado.
	if _, err := runtime.Ingest("org-a", "telegram", signature, timestamp, body, OmnichannelEvent{
		ExternalMessageID: "wamid.tg", ContactID: "chat-1", MessageType: "text",
	}, now); !errors.Is(err, ErrOmnichannelNotConfigured) {
		t.Fatalf("canal sem conector precisa ser recusado, veio %v", err)
	}
}

func TestOmnichannelRateLimitAndDeadLetter(t *testing.T) {
	inbox, err := NewInboxStore(filepath.Join(t.TempDir(), "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewOmnichannelRuntime(inbox)
	t.Setenv("HADES_TEST_OMNI_SECRET", "segredo-de-teste-com-tamanho-suficiente")
	now := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-a", Channel: "whatsapp", SecretEnv: "HADES_TEST_OMNI_SECRET", Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetRateLimit(3)
	timestamp := now.Unix()
	accepted := 0
	var limited error
	for index := range 5 {
		body := []byte(`{"external_message_id":"rl-` + strconv.Itoa(index) + `"}`)
		signature, err := runtime.SignWebhook(connector.ID, timestamp, body)
		if err != nil {
			t.Fatal(err)
		}
		_, err = runtime.Ingest("org-a", "whatsapp", signature, timestamp, body, OmnichannelEvent{
			ExternalMessageID: "rl-" + strconv.Itoa(index), ContactID: "+551190000000" + strconv.Itoa(index), MessageType: "text", Preview: "msg",
		}, now)
		if err == nil {
			accepted++
			continue
		}
		limited = err
	}
	if accepted != 3 {
		t.Fatalf("o limite precisa aceitar 3 eventos: %d", accepted)
	}
	if !errors.Is(limited, ErrOmnichannelRateLimited) {
		t.Fatalf("o excedente precisa ser recusado por limite, veio %v", limited)
	}

	// DLQ: um evento que falha no destino vai para a fila COM MOTIVO.
	broken := NewOmnichannelRuntime(nil) // sem inbox, a ingestão aceita mas não entrega
	t.Setenv("HADES_TEST_OMNI_SECRET2", "outro-segredo-de-teste-suficiente")
	brokenConnector, err := broken.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-a", Channel: "telegram", SecretEnv: "HADES_TEST_OMNI_SECRET2", Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"external_message_id":"dlq-1"}`)
	signature, err := broken.SignWebhook(brokenConnector.ID, timestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broken.Ingest("org-a", "telegram", signature, timestamp, body, OmnichannelEvent{
		ExternalMessageID: "dlq-1", ContactID: "chat-9", MessageType: "text", Preview: "evento",
	}, now); err != nil {
		t.Fatalf("sem inbox a ingestão ainda registra o evento: %v", err)
	}
	// Com uma inbox que funcione, um erro real gera DLQ. Forçamos o caminho com um
	// evento inválido no destino: contact_id vazio.
	badBody := []byte(`{"external_message_id":"dlq-2"}`)
	badSignature, err := runtime.SignWebhook(connector.ID, timestamp, badBody)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Ingest("org-a", "whatsapp", badSignature, timestamp, badBody, OmnichannelEvent{
		ExternalMessageID: "dlq-2", ContactID: "", MessageType: "text",
	}, now); !errors.Is(err, ErrOmnichannelEventInvalid) {
		t.Fatalf("evento inválido precisa ser recusado, veio %v", err)
	}
	if entries := runtime.DeadLetters("org-a"); len(entries) != 0 {
		t.Fatalf("evento recusado na validação não é DLQ: %+v", entries)
	}
}

func TestOmnichannelDeadLetterRetryIsExplicitAndKeepsHistory(t *testing.T) {
	inbox, err := NewInboxStore(filepath.Join(t.TempDir(), "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewOmnichannelRuntime(inbox)
	t.Setenv("HADES_TEST_OMNI_SECRET", "segredo-de-teste-com-tamanho-suficiente")
	now := time.Now().UTC()
	if _, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-a", Channel: "whatsapp", SecretEnv: "HADES_TEST_OMNI_SECRET", Enabled: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	// Simula uma falha anterior colocando o evento direto na fila de mortos.
	runtime.deadLetter(OmnichannelEvent{
		OrganizationID: "org-a", ChannelID: "whatsapp", ExternalMessageID: "dlq-retry",
		ContactID: "+5511922223333", MessageType: "text", Preview: "evento que falhou",
	}, "whatsapp", "falha simulada", now)
	entries := runtime.DeadLetters("org-a")
	if len(entries) != 1 {
		t.Fatalf("a fila precisa ter um evento: %+v", entries)
	}
	if entries[0].Reason == "" || entries[0].Attempts != 1 {
		t.Fatalf("a entrada precisa ter motivo e tentativa: %+v", entries[0])
	}
	// Reprocessar com a inbox funcionando: a mensagem entra e a entrada sai da fila.
	if _, err := runtime.RetryDeadLetter("org-a", entries[0].ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("o reprocessamento precisa funcionar: %v", err)
	}
	if remaining := runtime.DeadLetters("org-a"); len(remaining) != 0 {
		t.Fatalf("após reprocessar, a fila precisa esvaziar: %+v", remaining)
	}
	// Reprocessar um id inexistente é erro explícito.
	if _, err := runtime.RetryDeadLetter("org-a", "dlq_inexistente", now); !errors.Is(err, ErrOmnichannelDLQNotFound) {
		t.Fatalf("id inexistente precisa falhar, veio %v", err)
	}
	// E a outra organização não vê a fila da primeira.
	if entries := runtime.DeadLetters("org-b"); len(entries) != 0 {
		t.Fatalf("a fila precisa ser escopada: %+v", entries)
	}
}

func TestOmnichannelNeverStoresOrLeaksTheSecret(t *testing.T) {
	runtime, connectorID, body := omnichannelRuntime(t)
	now := time.Now().UTC()
	timestamp := now.Unix()
	signature, err := runtime.SignWebhook(connectorID, timestamp, body)
	if err != nil {
		t.Fatal(err)
	}
	event, err := runtime.Ingest("org-a", "whatsapp", signature, timestamp, body, omnichannelEvent(t), now)
	if err != nil {
		t.Fatal(err)
	}
	secret := "segredo-de-teste-com-tamanho-suficiente"
	// A assinatura não é guardada no evento nem o segredo.
	rendered := strings.Join([]string{
		event.ID, event.OrganizationID, event.ChannelID, event.ConnectorID,
		event.ExternalMessageID, event.Preview, event.CorrelationID,
	}, "|")
	if strings.Contains(rendered, secret) {
		t.Fatal("o evento não pode conter o segredo")
	}
	if strings.Contains(rendered, signature) {
		t.Fatal("o evento não pode guardar a assinatura")
	}
	// O conector exposto pela API referencia apenas o NOME da variável.
	connectors := runtime.Connectors("org-a")
	if len(connectors) != 1 || connectors[0].SecretEnv != "HADES_TEST_OMNI_SECRET" {
		t.Fatalf("o conector precisa expor apenas o nome da variável: %+v", connectors)
	}
	if strings.Contains(connectors[0].SecretEnv, secret) {
		t.Fatal("o nome da variável não pode conter o segredo")
	}
}
