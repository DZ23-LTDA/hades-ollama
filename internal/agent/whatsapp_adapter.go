package agent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WhatsAppBackendType identifies the underlying WhatsApp provider.
type WhatsAppBackendType string

const (
	WhatsAppBackendEvolution WhatsAppBackendType = "evolution_api"
	WhatsAppBackendCloudAPI  WhatsAppBackendType = "cloud_api"
)

// WhatsAppInboundMessage represents a parsed incoming WhatsApp message.
type WhatsAppInboundMessage struct {
	ID         string              `json:"id"`
	From       string              `json:"from"` // E.164 digits without + (e.g. 5511999999999)
	To         string              `json:"to"`
	SenderName string              `json:"sender_name,omitempty"`
	Text       string              `json:"text"`
	Type       string              `json:"type"` // "text", "audio", "image", "document", etc.
	MediaURL   string              `json:"media_url,omitempty"`
	MediaBytes []byte              `json:"-"`
	FromMe     bool                `json:"from_me"`
	Timestamp  int64               `json:"timestamp"`
	Backend    WhatsAppBackendType `json:"backend"`
	RawPayload map[string]any      `json:"raw_payload,omitempty"`
}

// WhatsAppOutboundMessage represents a message to be sent via WhatsApp.
type WhatsAppOutboundMessage struct {
	To         string `json:"to"`
	Text       string `json:"text,omitempty"`
	Type       string `json:"type,omitempty"` // "text", "audio", "image"
	MediaURL   string `json:"media_url,omitempty"`
	MediaBytes []byte `json:"-"`
	FileName   string `json:"file_name,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}

// WhatsAppSendResult is the outcome of an outbound send operation.
type WhatsAppSendResult struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// WhatsAppAdapter is the unified interface for WhatsApp backends.
type WhatsAppAdapter interface {
	Backend() WhatsAppBackendType
	Status() GateStatus
	StatusDetails() string
	VerifyWebhook(req *http.Request) ([]byte, bool)
	ParseWebhook(body []byte, header http.Header) ([]WhatsAppInboundMessage, error)
	SendMessage(ctx context.Context, msg WhatsAppOutboundMessage) (WhatsAppSendResult, error)
}

// webhookAuthenticator authenticates the raw provider payload before parsing.
type webhookAuthenticator interface {
	ValidateWebhook(body []byte, header http.Header) error
}

var (
	ErrWhatsAppWebhookNotConfigured = errors.New("whatsapp webhook is NOT_CONFIGURED: app secret is required")
	ErrWhatsAppInvalidSignature     = errors.New("whatsapp webhook signature is invalid")
	ErrWhatsAppInvalidPayload       = errors.New("whatsapp webhook identity is invalid")
)

// -------------------------------------------------------------------------
// Evolution API Backend
// -------------------------------------------------------------------------

// EvolutionConfig holds credentials and endpoint for Evolution API.
type EvolutionConfig struct {
	BaseURL       string `json:"base_url"`
	APIKey        string `json:"api_key"`
	Instance      string `json:"instance"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// EvolutionAdapter implements WhatsAppAdapter for Evolution API.
type EvolutionAdapter struct {
	config     EvolutionConfig
	httpClient *http.Client
}

// NewEvolutionAdapter creates a new Evolution API adapter.
func NewEvolutionAdapter(cfg EvolutionConfig) *EvolutionAdapter {
	return &EvolutionAdapter{
		config: cfg,
		httpClient: NewSafeEgressHTTPClient(EgressOptions{
			Callsite:      "whatsapp_evolution",
			Timeout:       10 * time.Second,
			AllowLoopback: true,
		}),
	}
}

func (e *EvolutionAdapter) Backend() WhatsAppBackendType {
	return WhatsAppBackendEvolution
}

func (e *EvolutionAdapter) Status() GateStatus {
	if strings.TrimSpace(e.config.BaseURL) == "" || strings.TrimSpace(e.config.APIKey) == "" || strings.TrimSpace(e.config.Instance) == "" || strings.TrimSpace(e.config.WebhookSecret) == "" {
		return GateStatusNotConfigured
	}
	return GateStatusPass
}

func (e *EvolutionAdapter) StatusDetails() string {
	if e.Status() == GateStatusNotConfigured {
		return "Evolution API não configurada (requer base_url, api_key, instance e webhook_secret)"
	}
	return fmt.Sprintf("Evolution API ativa (instância: %s)", e.config.Instance)
}

func (e *EvolutionAdapter) VerifyWebhook(req *http.Request) ([]byte, bool) {
	// Evolution has no Meta-style challenge. Unsigned payloads are never accepted.
	return nil, false
}

func (e *EvolutionAdapter) ValidateWebhook(body []byte, header http.Header) error {
	return validateWhatsAppHMAC(body, header.Get("X-Hub-Signature-256"), e.config.WebhookSecret)
}

func (e *EvolutionAdapter) ParseWebhook(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("empty webhook payload")
	}

	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("unmarshal evolution webhook: %w", err)
	}

	// Evolution API structure:
	// { "event": "messages.upsert", "data": { "key": { "remoteJid": "5511999999999@s.whatsapp.net", "fromMe": false, "id": "..." }, ... } }
	data, ok := root["data"].(map[string]any)
	if !ok {
		return nil, errors.New("missing data object in evolution payload")
	}

	key, ok := data["key"].(map[string]any)
	if !ok {
		return nil, errors.New("missing key object in evolution payload")
	}

	msgID, _ := key["id"].(string)
	remoteJid, _ := key["remoteJid"].(string)
	fromMe, _ := key["fromMe"].(bool)

	// Clean phone number from remoteJid (e.g. "5511999999999@s.whatsapp.net" -> "5511999999999")
	cleanFrom := cleanWhatsAppNumber(remoteJid)

	senderName, _ := data["pushName"].(string)

	// Extract message content
	msgType := "text"
	text := ""
	mediaURL := ""

	if msgObj, ok := data["message"].(map[string]any); ok {
		if conv, ok := msgObj["conversation"].(string); ok && conv != "" {
			text = conv
		} else if ext, ok := msgObj["extendedTextMessage"].(map[string]any); ok {
			if extText, ok := ext["text"].(string); ok {
				text = extText
			}
		} else if audio, ok := msgObj["audioMessage"].(map[string]any); ok {
			msgType = "audio"
			if u, ok := audio["url"].(string); ok {
				mediaURL = u
			}
		} else if img, ok := msgObj["imageMessage"].(map[string]any); ok {
			msgType = "image"
			if u, ok := img["url"].(string); ok {
				mediaURL = u
			}
			if cap, ok := img["caption"].(string); ok {
				text = cap
			}
		}
	}

	var ts int64
	if tsFloat, ok := data["messageTimestamp"].(float64); ok {
		ts = int64(tsFloat)
	} else {
		ts = time.Now().Unix()
	}

	inbound := WhatsAppInboundMessage{
		ID:         msgID,
		From:       cleanFrom,
		To:         e.config.Instance,
		SenderName: senderName,
		Text:       strings.TrimSpace(text),
		Type:       msgType,
		MediaURL:   mediaURL,
		FromMe:     fromMe,
		Timestamp:  ts,
		Backend:    WhatsAppBackendEvolution,
		RawPayload: root,
	}

	return []WhatsAppInboundMessage{inbound}, nil
}

func (e *EvolutionAdapter) SendMessage(ctx context.Context, msg WhatsAppOutboundMessage) (WhatsAppSendResult, error) {
	if e.Status() == GateStatusNotConfigured {
		return WhatsAppSendResult{}, errors.New("evolution api adapter is NOT_CONFIGURED")
	}

	endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(e.config.BaseURL, "/"), e.config.Instance)
	payload := map[string]any{
		"number": cleanWhatsAppNumber(msg.To),
		"text":   msg.Text,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return WhatsAppSendResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return WhatsAppSendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", e.config.APIKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return WhatsAppSendResult{}, fmt.Errorf("evolution send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := ReadBoundedBody(resp.Body, 64<<10)
		return WhatsAppSendResult{}, fmt.Errorf("evolution send failed HTTP %d: %s", resp.StatusCode, RedactDLP(strings.TrimSpace(string(respBody))))
	}

	var resMap map[string]any
	// Bound the success body too (the error path already does): the client has no
	// enforced body limit, so a huge 200 from a user-configured provider could OOM.
	raw, _ := ReadBoundedBody(resp.Body, 1<<20)
	_ = json.Unmarshal(raw, &resMap)

	id := "evo_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if key, ok := resMap["key"].(map[string]any); ok {
		if kID, ok := key["id"].(string); ok && kID != "" {
			id = kID
		}
	}

	return WhatsAppSendResult{
		MessageID: id,
		Status:    "sent",
	}, nil
}

// -------------------------------------------------------------------------
// WhatsApp Business Cloud API Backend (Meta)
// -------------------------------------------------------------------------

// CloudAPIConfig holds credentials and endpoint for Meta's WhatsApp Cloud API.
type CloudAPIConfig struct {
	PhoneNumberID string `json:"phone_number_id"`
	AccessToken   string `json:"access_token"`
	VerifyToken   string `json:"verify_token"`
	AppSecret     string `json:"app_secret,omitempty"`
}

// CloudAPIAdapter implements WhatsAppAdapter for Meta WhatsApp Business Cloud API.
type CloudAPIAdapter struct {
	config     CloudAPIConfig
	httpClient *http.Client
}

// NewCloudAPIAdapter creates a new WhatsApp Cloud API adapter.
func NewCloudAPIAdapter(cfg CloudAPIConfig) *CloudAPIAdapter {
	return &CloudAPIAdapter{
		config: cfg,
		httpClient: NewSafeEgressHTTPClient(EgressOptions{
			Callsite:      "whatsapp_cloud_api",
			Timeout:       10 * time.Second,
			AllowLoopback: false,
		}),
	}
}

func (c *CloudAPIAdapter) Backend() WhatsAppBackendType {
	return WhatsAppBackendCloudAPI
}

func (c *CloudAPIAdapter) Status() GateStatus {
	if strings.TrimSpace(c.config.PhoneNumberID) == "" || strings.TrimSpace(c.config.AccessToken) == "" || strings.TrimSpace(c.config.AppSecret) == "" {
		return GateStatusNotConfigured
	}
	return GateStatusPass
}

func (c *CloudAPIAdapter) StatusDetails() string {
	if c.Status() == GateStatusNotConfigured {
		return "WhatsApp Cloud API não configurada (requer phone_number_id, access_token e app_secret)"
	}
	return fmt.Sprintf("WhatsApp Cloud API ativa (Phone ID: %s)", c.config.PhoneNumberID)
}

func (c *CloudAPIAdapter) VerifyWebhook(req *http.Request) ([]byte, bool) {
	mode := req.URL.Query().Get("hub.mode")
	token := req.URL.Query().Get("hub.verify_token")
	challenge := req.URL.Query().Get("hub.challenge")

	if strings.TrimSpace(c.config.VerifyToken) != "" && mode == "subscribe" && token != "" && token == c.config.VerifyToken {
		return []byte(challenge), true
	}
	return nil, false
}

func (c *CloudAPIAdapter) ValidateWebhook(body []byte, header http.Header) error {
	if err := validateWhatsAppHMAC(body, header.Get("X-Hub-Signature-256"), c.config.AppSecret); err != nil {
		return err
	}

	var root struct {
		Object string `json:"object"`
		Entry  []struct {
			Changes []struct {
				Value struct {
					Metadata struct {
						PhoneNumberID string `json:"phone_number_id"`
					} `json:"metadata"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("%w: invalid JSON", ErrWhatsAppInvalidPayload)
	}
	if root.Object != "whatsapp" || strings.TrimSpace(c.config.PhoneNumberID) == "" || len(root.Entry) == 0 {
		return ErrWhatsAppInvalidPayload
	}
	for _, entry := range root.Entry {
		if len(entry.Changes) == 0 {
			return ErrWhatsAppInvalidPayload
		}
		for _, change := range entry.Changes {
			if change.Value.Metadata.PhoneNumberID == "" || change.Value.Metadata.PhoneNumberID != c.config.PhoneNumberID {
				return ErrWhatsAppInvalidPayload
			}
		}
	}
	return nil
}

func validateWhatsAppHMAC(body []byte, signature, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ErrWhatsAppWebhookNotConfigured
	}
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "sha256=") {
		return ErrWhatsAppInvalidSignature
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return ErrWhatsAppInvalidSignature
	}
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write(body)
	if !hmac.Equal(provided, digest.Sum(nil)) {
		return ErrWhatsAppInvalidSignature
	}
	return nil
}

func (c *CloudAPIAdapter) ParseWebhook(body []byte, header http.Header) ([]WhatsAppInboundMessage, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("empty webhook payload")
	}

	var root struct {
		Object string `json:"object"`
		Entry  []struct {
			ID      string `json:"id"`
			Changes []struct {
				Value struct {
					MessagingProduct string `json:"messaging_product"`
					Metadata         struct {
						DisplayPhoneNumber string `json:"display_phone_number"`
						PhoneNumberID      string `json:"phone_number_id"`
					} `json:"metadata"`
					Contacts []struct {
						Profile struct {
							Name string `json:"name"`
						} `json:"profile"`
						WaID string `json:"wa_id"`
					} `json:"contacts"`
					Messages []struct {
						From      string `json:"from"`
						ID        string `json:"id"`
						Timestamp string `json:"timestamp"`
						Type      string `json:"type"`
						Text      struct {
							Body string `json:"body"`
						} `json:"text"`
						Audio struct {
							ID       string `json:"id"`
							MimeType string `json:"mime_type"`
						} `json:"audio"`
						Image struct {
							ID       string `json:"id"`
							Caption  string `json:"caption"`
							MimeType string `json:"mime_type"`
						} `json:"image"`
					} `json:"messages"`
				} `json:"value"`
				Field string `json:"field"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("unmarshal cloud api webhook: %w", err)
	}

	var messages []WhatsAppInboundMessage

	for _, entry := range root.Entry {
		for _, change := range entry.Changes {
			contactMap := make(map[string]string)
			for _, contact := range change.Value.Contacts {
				contactMap[contact.WaID] = contact.Profile.Name
			}

			for _, m := range change.Value.Messages {
				cleanFrom := cleanWhatsAppNumber(m.From)
				senderName := contactMap[m.From]

				text := m.Text.Body
				mediaURL := ""
				if m.Type == "audio" {
					mediaURL = m.Audio.ID
				} else if m.Type == "image" {
					mediaURL = m.Image.ID
					if text == "" {
						text = m.Image.Caption
					}
				}

				var ts int64
				fmt.Sscanf(m.Timestamp, "%d", &ts)
				if ts == 0 {
					ts = time.Now().Unix()
				}

				messages = append(messages, WhatsAppInboundMessage{
					ID:         m.ID,
					From:       cleanFrom,
					To:         change.Value.Metadata.PhoneNumberID,
					SenderName: senderName,
					Text:       strings.TrimSpace(text),
					Type:       m.Type,
					MediaURL:   mediaURL,
					FromMe:     false,
					Timestamp:  ts,
					Backend:    WhatsAppBackendCloudAPI,
				})
			}
		}
	}

	return messages, nil
}

func (c *CloudAPIAdapter) SendMessage(ctx context.Context, msg WhatsAppOutboundMessage) (WhatsAppSendResult, error) {
	if c.Status() == GateStatusNotConfigured {
		return WhatsAppSendResult{}, errors.New("cloud api adapter is NOT_CONFIGURED")
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", c.config.PhoneNumberID)
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                cleanWhatsAppNumber(msg.To),
		"type":              "text",
		"text": map[string]string{
			"body": msg.Text,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return WhatsAppSendResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return WhatsAppSendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return WhatsAppSendResult{}, fmt.Errorf("cloud api send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := ReadBoundedBody(resp.Body, 64<<10)
		return WhatsAppSendResult{}, fmt.Errorf("cloud api send failed HTTP %d: %s", resp.StatusCode, RedactDLP(strings.TrimSpace(string(respBody))))
	}

	var resMap struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	// Bound the success body too (the error path already does): a huge 200 from a
	// user-configured Cloud API endpoint could otherwise OOM the process.
	raw, _ := ReadBoundedBody(resp.Body, 1<<20)
	_ = json.Unmarshal(raw, &resMap)

	id := "wamid_" + fmt.Sprintf("%d", time.Now().UnixNano())
	if len(resMap.Messages) > 0 && resMap.Messages[0].ID != "" {
		id = resMap.Messages[0].ID
	}

	return WhatsAppSendResult{
		MessageID: id,
		Status:    "sent",
	}, nil
}

// cleanWhatsAppNumber strips non-digits and domain suffixes from phone numbers.
func cleanWhatsAppNumber(input string) string {
	input = strings.TrimSpace(input)
	if idx := strings.Index(input, "@"); idx != -1 {
		input = input[:idx]
	}
	var out strings.Builder
	for _, r := range input {
		if r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}
