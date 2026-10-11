package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Omnichannel Runtime (estágio 34): contrato canônico de mensagens.
//
// O estágio exige **validação de webhooks, replay protection, deduplicação,
// idempotência, DLQ, rate limiting, logs sanitizados** e — regra dura — **nunca
// cruzar empresas**. Este núcleo é o que liga canais reais à caixa de entrada já
// existente, sem duplicar o caminho de mensagem.
//
// DECISÕES DELIBERADAS:
//   - **O TENANT VEM DO CONECTOR AUTENTICADO, nunca do payload**: um campo
//     `tenant_id` no corpo é dado de terceiro e não pode escolher onde a mensagem
//     entra. É o que impede cruzamento entre empresas;
//   - **Assinatura HMAC-SHA256, comparação em tempo constante**, com o segredo vindo
//     do ambiente. **Sem segredo configurado o webhook é RECUSADO** (falha fechada):
//     aceitar não assinado "porque não há chave" transformaria o endpoint em porta
//     aberta;
//   - **Anti-replay por janela de tempo**: evento fora da janela é recusado, e o
//     mesmo `ExternalMessageID` reentregue é **deduplicado** — as duas proteções
//     existem porque uma não cobre a outra;
//   - **DLQ de verdade**: evento que falha no processamento vai para a fila de mortos
//     **com o motivo**, e o reprocessamento é explícito — nunca descarte silencioso;
//   - **Rate limit por organização e canal**: limita abuso de um conector sem
//     penalizar os outros;
//   - **Log sanitizado**: o evento guarda uma **prévia curta** da mensagem e NUNCA o
//     segredo nem a assinatura.

const (
	// OmnichannelSignatureHeader e OmnichannelTimestampHeader são os cabeçalhos do contrato.
	OmnichannelSignatureHeader = "X-Hades-Signature"
	OmnichannelTimestampHeader = "X-Hades-Timestamp"

	// OmnichannelReplayWindow é a janela aceita para o carimbo de tempo.
	OmnichannelReplayWindow = 5 * time.Minute

	// OmnichannelDefaultRateLimit é o teto de eventos por janela, por canal e organização.
	OmnichannelDefaultRateLimit = 120
	// OmnichannelRateWindow é a janela do limitador.
	OmnichannelRateWindow = time.Minute

	// OmnichannelMaxDLQ limita a fila de mortos.
	OmnichannelMaxDLQ = 200
)

var (
	ErrOmnichannelNotConfigured = errors.New("omnichannel connector is not configured")
	ErrOmnichannelSignature     = errors.New("webhook signature is missing or invalid")
	ErrOmnichannelReplay        = errors.New("webhook timestamp is outside the accepted window")
	ErrOmnichannelRateLimited   = errors.New("omnichannel rate limit exceeded for this channel")
	ErrOmnichannelEventInvalid  = errors.New("omnichannel event invalid")
	ErrOmnichannelDLQNotFound   = errors.New("omnichannel dead letter not found")
)

// OmnichannelConnector liga um canal a uma organização. É a fonte confiável do
// tenant em toda ingestão.
type OmnichannelConnector struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Channel        string `json:"channel"`
	Name           string `json:"name,omitempty"`
	// SecretEnv é o NOME da variável de ambiente com o segredo. O valor nunca é
	// guardado nem devolvido pela API.
	SecretEnv string    `json:"secret_env"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// OmnichannelEvent é o contrato canônico de mensagem.
type OmnichannelEvent struct {
	ID string `json:"id"`
	// OrganizationID vem do CONECTOR, não do payload.
	OrganizationID    string            `json:"organization_id"`
	ChannelID         string            `json:"channel_id"`
	ConnectorID       string            `json:"connector_id"`
	ExternalMessageID string            `json:"external_message_id"`
	ConversationID    string            `json:"conversation_id,omitempty"`
	ContactID         string            `json:"contact_id"`
	Sender            string            `json:"sender,omitempty"`
	Recipient         string            `json:"recipient,omitempty"`
	MessageType       string            `json:"message_type"`
	ContentReference  string            `json:"content_reference,omitempty"`
	Attachments       []string          `json:"attachments,omitempty"`
	Timestamp         time.Time         `json:"timestamp"`
	DeliveryState     string            `json:"delivery_state"`
	CorrelationID     string            `json:"correlation_id,omitempty"`
	IdempotencyKey    string            `json:"idempotency_key,omitempty"`
	AgentAssignment   string            `json:"agent_assignment,omitempty"`
	ConsentState      string            `json:"consent_state,omitempty"`
	AuditMetadata     map[string]string `json:"audit_metadata,omitempty"`
	// Preview é uma prévia CURTA e sanitizada do conteúdo — nunca o corpo inteiro
	// nem segredo algum.
	Preview string `json:"preview,omitempty"`
	// Deduplicated marca a reentrega já conhecida.
	Deduplicated bool      `json:"deduplicated,omitempty"`
	IngestedAt   time.Time `json:"ingested_at"`
}

// OmnichannelDeadLetter é o evento que falhou, com o motivo.
type OmnichannelDeadLetter struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"organization_id"`
	Channel        string           `json:"channel"`
	Event          OmnichannelEvent `json:"event"`
	Reason         string           `json:"reason"`
	Attempts       int              `json:"attempts"`
	CreatedAt      time.Time        `json:"created_at"`
	LastAttemptAt  time.Time        `json:"last_attempt_at"`
}

type omnichannelBucket struct {
	windowStart time.Time
	count       int
}

// OmnichannelRuntime processa eventos de qualquer canal sob o mesmo contrato.
type OmnichannelRuntime struct {
	mu          sync.Mutex
	connectors  map[string]OmnichannelConnector
	eventsByKey map[string]OmnichannelEvent
	buckets     map[string]omnichannelBucket
	deadLetters []OmnichannelDeadLetter
	rateLimit   int
	// inbox é o ÚNICO destino das mensagens: reusar evita um segundo caminho de
	// mensagem que poderia divergir do primeiro.
	inbox *InboxStore
	// now permite testar a janela de replay sem esperar.
	now func() time.Time
}

// NewOmnichannelRuntime cria o runtime ligado à caixa de entrada existente.
func NewOmnichannelRuntime(inbox *InboxStore) *OmnichannelRuntime {
	return &OmnichannelRuntime{
		connectors:  map[string]OmnichannelConnector{},
		eventsByKey: map[string]OmnichannelEvent{},
		buckets:     map[string]omnichannelBucket{},
		rateLimit:   OmnichannelDefaultRateLimit,
		inbox:       inbox,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// SetRateLimit ajusta o teto por janela (0 mantém o padrão).
func (r *OmnichannelRuntime) SetRateLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit > 0 {
		r.rateLimit = limit
	}
}

// RegisterConnector registra (ou atualiza) um conector. O segredo é referenciado
// pelo NOME da variável de ambiente.
func (r *OmnichannelRuntime) RegisterConnector(connector OmnichannelConnector, now time.Time) (OmnichannelConnector, error) {
	connector.OrganizationID = consentScope(connector.OrganizationID)
	channel := normalizeInboxChannel(connector.Channel)
	if channel == "" {
		return OmnichannelConnector{}, fmt.Errorf("%w: channel is required", ErrOmnichannelEventInvalid)
	}
	connector.Channel = channel
	if strings.TrimSpace(connector.SecretEnv) == "" {
		return OmnichannelConnector{}, fmt.Errorf("%w: secret_env is required (sem segredo o webhook é recusado)", ErrOmnichannelEventInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if connector.ID == "" {
		connector.ID = "cn_" + uuid.NewString()[:12]
	}
	connector.CreatedAt = now
	r.connectors[connector.ID] = connector
	return connector, nil
}

// Connectors lista os conectores de uma organização (sem segredo).
func (r *OmnichannelRuntime) Connectors(organizationID string) []OmnichannelConnector {
	organizationID = consentScope(organizationID)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]OmnichannelConnector, 0)
	for _, connector := range r.connectors {
		if connector.OrganizationID != organizationID {
			continue
		}
		out = append(out, connector)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Channel < out[j].Channel })
	return out
}

// connectorFor resolve o conector de uma organização num canal. É a fonte do tenant.
func (r *OmnichannelRuntime) connectorFor(organizationID, channel string) (OmnichannelConnector, error) {
	organizationID = consentScope(organizationID)
	channel = normalizeInboxChannel(channel)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, connector := range r.connectors {
		if connector.OrganizationID == organizationID && connector.Channel == channel && connector.Enabled {
			return connector, nil
		}
	}
	return OmnichannelConnector{}, fmt.Errorf("%w: %s/%s", ErrOmnichannelNotConfigured, organizationID, channel)
}

// signatureFor calcula a assinatura HMAC-SHA256 do corpo e do carimbo.
func omnichannelSignature(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// SignWebhook produz os cabeçalhos de assinatura — usado pelo produtor do evento
// (teste ou conector de saída). O segredo nunca é registrado.
func (r *OmnichannelRuntime) SignWebhook(connectorID string, timestamp int64, body []byte) (string, error) {
	r.mu.Lock()
	connector, ok := r.connectors[connectorID]
	r.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrOmnichannelNotConfigured, connectorID)
	}
	secret := strings.TrimSpace(os.Getenv(connector.SecretEnv))
	if secret == "" {
		return "", fmt.Errorf("%w: secret %s is not set", ErrOmnichannelNotConfigured, connector.SecretEnv)
	}
	return omnichannelSignature(secret, timestamp, body), nil
}

// rateLimitedLocked aplica o limitador por organização e canal.
func (r *OmnichannelRuntime) rateLimitedLocked(organizationID, channel string, now time.Time) bool {
	key := organizationID + "|" + channel
	bucket, ok := r.buckets[key]
	if !ok || now.Sub(bucket.windowStart) >= OmnichannelRateWindow {
		r.buckets[key] = omnichannelBucket{windowStart: now, count: 1}
		return false
	}
	if bucket.count >= r.rateLimit {
		return true
	}
	bucket.count++
	r.buckets[key] = bucket
	return false
}

// Ingest é a ÚNICA porta de entrada de eventos externos. A ordem é deliberada:
// conector (tenant) → assinatura → janela de replay → rate limit → dedupe → inbox.
func (r *OmnichannelRuntime) Ingest(organizationID, channel string, signature string, timestamp int64, body []byte, event OmnichannelEvent, now time.Time) (OmnichannelEvent, error) {
	connector, err := r.connectorFor(organizationID, channel)
	if err != nil {
		return OmnichannelEvent{}, err
	}
	secret := strings.TrimSpace(os.Getenv(connector.SecretEnv))
	if secret == "" {
		// FALHA FECHADA: sem segredo não há como verificar, então não se aceita.
		return OmnichannelEvent{}, fmt.Errorf("%w: secret %s is not set", ErrOmnichannelNotConfigured, connector.SecretEnv)
	}
	expected := omnichannelSignature(secret, timestamp, body)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return OmnichannelEvent{}, ErrOmnichannelSignature
	}
	eventTime := time.Unix(timestamp, 0).UTC()
	if now.Sub(eventTime) > OmnichannelReplayWindow || eventTime.Sub(now) > OmnichannelReplayWindow {
		return OmnichannelEvent{}, ErrOmnichannelReplay
	}
	if strings.TrimSpace(event.ExternalMessageID) == "" {
		return OmnichannelEvent{}, fmt.Errorf("%w: external_message_id is required for deduplication", ErrOmnichannelEventInvalid)
	}
	if strings.TrimSpace(event.ContactID) == "" || strings.TrimSpace(event.MessageType) == "" {
		return OmnichannelEvent{}, fmt.Errorf("%w: contact_id and message_type are required", ErrOmnichannelEventInvalid)
	}

	r.mu.Lock()
	if r.rateLimitedLocked(connector.OrganizationID, connector.Channel, now) {
		r.mu.Unlock()
		return OmnichannelEvent{}, ErrOmnichannelRateLimited
	}
	// O TENANT é o do conector: o payload não escolhe organização.
	event.OrganizationID = connector.OrganizationID
	event.ChannelID = connector.Channel
	event.ConnectorID = connector.ID
	dedupeKey := connector.OrganizationID + "|" + connector.Channel + "|" + strings.TrimSpace(event.ExternalMessageID)
	if existing, seen := r.eventsByKey[dedupeKey]; seen {
		existing.Deduplicated = true
		r.mu.Unlock()
		return existing, nil
	}
	event.IngestedAt = now
	if event.Timestamp.IsZero() {
		event.Timestamp = eventTime
	}
	if event.DeliveryState == "" {
		event.DeliveryState = "received"
	}
	if event.ID == "" {
		event.ID = "ev_" + uuid.NewString()[:12]
	}
	r.mu.Unlock()

	// Mensagem entra pela caixa de entrada JÁ existente — sem segundo caminho.
	if r.inbox != nil {
		conversation, message, err := r.inbox.AppendInbound(event.OrganizationID, event.ChannelID, event.ContactID, "", event.Preview, event.ExternalMessageID, now)
		if err != nil {
			r.deadLetter(event, connector.Channel, fmt.Sprintf("inbox: %v", err), now)
			return OmnichannelEvent{}, err
		}
		event.ConversationID = conversation.ID
		if message.Deduplicated {
			event.Deduplicated = true
		}
	}

	r.mu.Lock()
	r.eventsByKey[dedupeKey] = event
	r.mu.Unlock()
	return event, nil
}

func (r *OmnichannelRuntime) deadLetter(event OmnichannelEvent, channel, reason string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deadLetters = append(r.deadLetters, OmnichannelDeadLetter{
		ID:             "dlq_" + uuid.NewString()[:12],
		OrganizationID: event.OrganizationID,
		Channel:        channel,
		Event:          event,
		Reason:         reason,
		Attempts:       1,
		CreatedAt:      now,
		LastAttemptAt:  now,
	})
	if len(r.deadLetters) > OmnichannelMaxDLQ {
		r.deadLetters = r.deadLetters[len(r.deadLetters)-OmnichannelMaxDLQ:]
	}
}

// DeadLetters lista a fila de mortos de uma organização.
func (r *OmnichannelRuntime) DeadLetters(organizationID string) []OmnichannelDeadLetter {
	organizationID = consentScope(organizationID)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]OmnichannelDeadLetter, 0)
	for _, entry := range r.deadLetters {
		if entry.OrganizationID != organizationID {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// RetryDeadLetter reprocessa um evento da fila de mortos. Sucesso remove a entrada;
// falha incrementa a tentativa e ATUALIZA o motivo, sem apagar o histórico.
func (r *OmnichannelRuntime) RetryDeadLetter(organizationID, id string, now time.Time) (OmnichannelDeadLetter, error) {
	organizationID = consentScope(organizationID)
	r.mu.Lock()
	index := -1
	for position, entry := range r.deadLetters {
		if entry.ID == id && entry.OrganizationID == organizationID {
			index = position
			break
		}
	}
	if index < 0 {
		r.mu.Unlock()
		return OmnichannelDeadLetter{}, fmt.Errorf("%w: %s", ErrOmnichannelDLQNotFound, id)
	}
	entry := r.deadLetters[index]
	r.mu.Unlock()

	if r.inbox == nil {
		r.bumpDeadLetter(index, "inbox indisponível", now)
		return entry, fmt.Errorf("%w: inbox indisponível", ErrOmnichannelNotConfigured)
	}
	_, _, err := r.inbox.AppendInbound(entry.Event.OrganizationID, entry.Channel, entry.Event.ContactID, "", entry.Event.Preview, entry.Event.ExternalMessageID, now)
	if err != nil {
		r.bumpDeadLetter(index, err.Error(), now)
		return entry, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if index < len(r.deadLetters) {
		r.deadLetters = append(r.deadLetters[:index], r.deadLetters[index+1:]...)
	}
	return entry, nil
}

func (r *OmnichannelRuntime) bumpDeadLetter(index int, reason string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index < len(r.deadLetters) {
		r.deadLetters[index].Attempts++
		r.deadLetters[index].Reason = reason
		r.deadLetters[index].LastAttemptAt = now
	}
}

// Events lista os eventos aceitos de uma organização (sem segredo e com prévia curta).
func (r *OmnichannelRuntime) Events(organizationID string) []OmnichannelEvent {
	organizationID = consentScope(organizationID)
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]OmnichannelEvent, 0)
	for _, event := range r.eventsByKey {
		if event.OrganizationID != organizationID {
			continue
		}
		out = append(out, event)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].IngestedAt.After(out[j].IngestedAt) })
	return out
}
