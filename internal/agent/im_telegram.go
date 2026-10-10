package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ollama/ollama/internal/multillm"
)

// Canal de mensagens instantaneas do Telegram.
//
// Os conectores genericos do Hades autenticam por cabecalho e resolvem a
// credencial pelo nome em TokenEnv, mas a Bot API do Telegram carrega o token
// no proprio caminho da URL (/bot<TOKEN>/sendMessage). Reutilizar o
// ConnectorManager exigiria liberar credencial na URL, exatamente o que a
// validacao de caminho do gerenciador recusa. Este adaptador dedicado fixa o
// host em api.telegram.org, aceita apenas a forma <id>:<segredo> do token,
// nunca devolve a credencial em erro, resultado de ferramenta ou log de
// egresso, e submete o texto a guarda de DLP antes de qualquer chamada.
const (
	// TelegramBotTokenEnv e o nome da credencial do bot, resolvida pelo
	// ambiente, por <NOME>_FILE ou pelo cofre do aplicativo desktop.
	TelegramBotTokenEnv = "TELEGRAM_BOT_TOKEN"

	telegramAPIBaseURL = "https://api.telegram.org"
	telegramAPIHost    = "api.telegram.org"
	telegramCallsite   = "im.telegram"

	telegramRequestTimeout      = 20 * time.Second
	telegramMaxTextBytes        = 4096
	telegramMaxPayloadBytes     = 1 << 20
	telegramMaxResponseBytes    = 64 << 10
	telegramMaxDescriptionRunes = 300
	telegramRedactedToken       = "bot<redigido>"
)

var (
	ErrTelegramCredentialUnavailable = errors.New("credencial do bot do Telegram indisponivel")
	ErrTelegramCredentialInvalid     = errors.New("credencial do bot do Telegram invalida")
	ErrTelegramChatInvalid           = errors.New("chat do Telegram invalido")
	ErrTelegramMessageEmpty          = errors.New("mensagem do Telegram vazia")
	ErrTelegramMessageTooLong        = errors.New("mensagem excede o limite do Telegram")
	ErrTelegramMessageBlocked        = errors.New("mensagem bloqueada pela politica de DLP")
	ErrTelegramHostNotAllowed        = errors.New("host da Bot API do Telegram nao permitido")
	ErrTelegramCallFailed            = errors.New("chamada a Bot API do Telegram falhou")
	ErrTelegramResponseInvalid       = errors.New("resposta da Bot API do Telegram invalida")
)

var (
	// telegramTokenPattern reproduz a forma emitida pelo BotFather.
	telegramTokenPattern = regexp.MustCompile(`^[0-9]{5,20}:[A-Za-z0-9_-]{20,80}$`)
	// telegramChatPattern aceita id numerico (inclusive de grupo) ou @canal.
	telegramChatPattern = regexp.MustCompile(`^(-?[0-9]{1,20}|@[A-Za-z][A-Za-z0-9_]{3,31})$`)
)

type telegramChannel struct {
	baseURL        string
	credential     func(string) string
	client         *http.Client
	organizationID string
	timeout        time.Duration
}

func newTelegramChannel() *telegramChannel {
	return &telegramChannel{
		baseURL:    telegramAPIBaseURL,
		credential: multillm.CredentialValue,
		timeout:    telegramRequestTimeout,
	}
}

type telegramSendResult struct {
	ChatID     string
	MessageID  int64
	DurationMS int64
}

type telegramBotIdentity struct {
	ID        int64
	Username  string
	FirstName string
}

// base resolve a base da Bot API. Apenas o host oficial -- mais o loopback, que
// existe somente como costura de teste do httptest -- e aceito, o que impede
// que uma base configuravel torne o canal um vetor de SSRF.
func (c *telegramChannel) base() (*url.URL, error) {
	raw := strings.TrimSpace(telegramAPIBaseURL)
	if c != nil && strings.TrimSpace(c.baseURL) != "" {
		raw = strings.TrimSpace(c.baseURL)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return nil, ErrTelegramHostNotAllowed
	}
	host := parsed.Hostname()
	switch {
	case strings.EqualFold(host, telegramAPIHost):
		if !strings.EqualFold(parsed.Scheme, "https") {
			return nil, ErrTelegramHostNotAllowed
		}
	case connectorHostIsLoopback(host):
		if !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http") {
			return nil, ErrTelegramHostNotAllowed
		}
	default:
		return nil, ErrTelegramHostNotAllowed
	}
	return parsed, nil
}

func (c *telegramChannel) token() (string, error) {
	if c == nil || c.credential == nil {
		return "", ErrTelegramCredentialUnavailable
	}
	token := strings.TrimSpace(c.credential(TelegramBotTokenEnv))
	if token == "" {
		return "", ErrTelegramCredentialUnavailable
	}
	if !telegramTokenPattern.MatchString(token) {
		return "", ErrTelegramCredentialInvalid
	}
	return token, nil
}

func (c *telegramChannel) timeoutOrDefault() time.Duration {
	if c != nil && c.timeout > 0 {
		return c.timeout
	}
	return telegramRequestTimeout
}

func (c *telegramChannel) httpClient(base *url.URL) *http.Client {
	allowLoopback := connectorHostIsLoopback(base.Hostname())
	organizationID := ""
	if c != nil {
		organizationID = c.organizationID
	}
	if c != nil && c.client != nil {
		client := *c.client
		client.Timeout = c.timeoutOrDefault()
		client.CheckRedirect = NewEgressCheckRedirect(telegramCallsite, allowLoopback)
		return &client
	}
	return NewSafeEgressHTTPClient(EgressOptions{
		Callsite:       telegramCallsite,
		OrganizationID: organizationID,
		Timeout:        c.timeoutOrDefault(),
		MaxBodyBytes:   telegramMaxResponseBytes,
		AllowLoopback:  allowLoopback,
	})
}

// endpoint monta a URL da Bot API com o token no caminho. O token ja passou por
// telegramTokenPattern, portanto nao pode injetar caminho, consulta ou host.
func (c *telegramChannel) endpoint(base *url.URL, token, method string) string {
	endpoint := *base
	endpoint.Path = strings.TrimSuffix(base.Path, "/") + "/bot" + token + "/" + method
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	endpoint.RawPath = ""
	return endpoint.String()
}

func (c *telegramChannel) auditDestination(base *url.URL, method string) string {
	return fmt.Sprintf("%s://%s%s/%s/%s", base.Scheme, base.Host, strings.TrimSuffix(base.Path, "/"), telegramRedactedToken, method)
}

func (c *telegramChannel) call(ctx context.Context, token, method string, payload any) (json.RawMessage, error) {
	base, err := c.base()
	if err != nil {
		return nil, err
	}
	var body []byte
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, ErrTelegramCallFailed
		}
		if len(body) > telegramMaxPayloadBytes {
			return nil, ErrTelegramMessageTooLong
		}
		// Guarda de ultima linha: nenhum corpo chega a rede sem passar pelo DLP.
		if err := validateOutboundPayloadWithLimit(string(body), telegramMaxPayloadBytes); err != nil {
			return nil, ErrTelegramMessageBlocked
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(base, token, method), bytes.NewReader(body))
	if err != nil {
		// O erro do construtor carrega a URL, logo o token: nunca propagar.
		return nil, ErrTelegramCallFailed
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c != nil {
		DefaultEgressAuditStore.Record(EgressDecision{
			Timestamp:      time.Now(),
			OrganizationID: c.organizationID,
			Callsite:       telegramCallsite,
			Method:         http.MethodPost,
			Destination:    c.auditDestination(base, method),
			Host:           base.Host,
			Allowed:        true,
			Reason:         fmt.Sprintf("canal de IM aprovado; dlp_findings=%d", len(ScanDLP(string(body)))),
		})
	}
	response, err := c.httpClient(base).Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTelegramCallFailed, redactTelegramError(err))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, telegramMaxResponseBytes+1))
	if err != nil {
		return nil, ErrTelegramCallFailed
	}
	if len(data) > telegramMaxResponseBytes {
		return nil, ErrTelegramResponseInvalid
	}
	var envelope struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, ErrTelegramResponseInvalid
	}
	if !envelope.OK {
		if envelope.Description == "" && envelope.ErrorCode == 0 {
			return nil, ErrTelegramResponseInvalid
		}
		return nil, fmt.Errorf("%w (http %d, codigo %d): %s", ErrTelegramCallFailed, response.StatusCode, envelope.ErrorCode, sanitizeTelegramDescription(envelope.Description))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w (http %d)", ErrTelegramCallFailed, response.StatusCode)
	}
	if len(envelope.Result) == 0 {
		return nil, ErrTelegramResponseInvalid
	}
	return envelope.Result, nil
}

func (c *telegramChannel) sendMessage(ctx context.Context, chatID, text string) (telegramSendResult, error) {
	chat, err := normalizeTelegramChatID(chatID)
	if err != nil {
		return telegramSendResult{}, err
	}
	if strings.TrimSpace(text) == "" {
		return telegramSendResult{}, ErrTelegramMessageEmpty
	}
	if len([]byte(text)) > telegramMaxTextBytes {
		return telegramSendResult{}, ErrTelegramMessageTooLong
	}
	if err := validateOutboundPayloadWithLimit(map[string]any{"text": text}, telegramMaxPayloadBytes); err != nil {
		return telegramSendResult{}, ErrTelegramMessageBlocked
	}
	token, err := c.token()
	if err != nil {
		return telegramSendResult{}, err
	}
	started := time.Now()
	result, err := c.call(ctx, token, "sendMessage", map[string]any{"chat_id": chat, "text": text})
	if err != nil {
		return telegramSendResult{}, err
	}
	var message struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	}
	if err := json.Unmarshal(result, &message); err != nil {
		return telegramSendResult{}, ErrTelegramResponseInvalid
	}
	if message.MessageID <= 0 {
		return telegramSendResult{}, ErrTelegramResponseInvalid
	}
	return telegramSendResult{ChatID: chat, MessageID: message.MessageID, DurationMS: time.Since(started).Milliseconds()}, nil
}

func (c *telegramChannel) verify(ctx context.Context) (telegramBotIdentity, error) {
	token, err := c.token()
	if err != nil {
		return telegramBotIdentity{}, err
	}
	result, err := c.call(ctx, token, "getMe", nil)
	if err != nil {
		return telegramBotIdentity{}, err
	}
	var bot struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	}
	if err := json.Unmarshal(result, &bot); err != nil {
		return telegramBotIdentity{}, ErrTelegramResponseInvalid
	}
	if bot.ID <= 0 {
		return telegramBotIdentity{}, ErrTelegramResponseInvalid
	}
	return telegramBotIdentity{
		ID:        bot.ID,
		Username:  sanitizeTelegramDescription(bot.Username),
		FirstName: sanitizeTelegramDescription(bot.FirstName),
	}, nil
}

func normalizeTelegramChatID(chatID string) (string, error) {
	chat := strings.TrimSpace(chatID)
	if chat == "" || !telegramChatPattern.MatchString(chat) {
		return "", ErrTelegramChatInvalid
	}
	return chat, nil
}

// redactTelegramError remove a credencial de qualquer erro de transporte, que
// pode citar a URL completa da Bot API.
func redactTelegramError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(RedactDLP(err.Error()))
	for _, field := range strings.Fields(message) {
		if strings.Contains(field, "/bot") {
			message = strings.ReplaceAll(message, field, telegramRedactedToken)
		}
	}
	if len(message) > telegramMaxDescriptionRunes {
		message = string([]rune(message)[:telegramMaxDescriptionRunes])
	}
	return message
}

func sanitizeTelegramDescription(raw string) string {
	trimmed := strings.TrimSpace(RedactDLP(raw))
	runes := []rune(trimmed)
	if len(runes) > telegramMaxDescriptionRunes {
		runes = runes[:telegramMaxDescriptionRunes]
	}
	trimmed = strings.TrimSpace(string(runes))
	if trimmed == "" {
		return "sem descricao"
	}
	return trimmed
}
