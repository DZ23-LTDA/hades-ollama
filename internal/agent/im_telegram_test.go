package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testTelegramToken = "123456789:AAHteste_token_de_teste_com_tamanho_ok"
	testTelegramChat  = "123456"
	testTelegramText  = "mensagem de teste do canal"
)

type telegramRecorder struct {
	mu     sync.Mutex
	paths  []string
	bodies []map[string]any
}

func (r *telegramRecorder) snapshot() ([]string, []map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	paths := append([]string(nil), r.paths...)
	bodies := append([]map[string]any(nil), r.bodies...)
	return paths, bodies
}

func newTelegramTestServer(t *testing.T, reply http.HandlerFunc) (*httptest.Server, *telegramRecorder) {
	t.Helper()
	recorder := &telegramRecorder{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = json.Unmarshal(body, &decoded)
		recorder.mu.Lock()
		recorder.paths = append(recorder.paths, r.URL.Path)
		recorder.bodies = append(recorder.bodies, decoded)
		recorder.mu.Unlock()
		if reply != nil {
			reply(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":42,"chat":{"id":123}}}`))
	}))
	t.Cleanup(server.Close)
	return server, recorder
}

func newTelegramTestChannel(t *testing.T, server *httptest.Server, token string) *telegramChannel {
	t.Helper()
	channel := newTelegramChannel()
	channel.timeout = 5 * time.Second
	if server != nil {
		channel.baseURL = server.URL
		channel.client = server.Client()
	}
	channel.credential = func(name string) string {
		if name != TelegramBotTokenEnv {
			t.Fatalf("nome de credencial inesperado: %q", name)
		}
		return token
	}
	return channel
}

func mustParseTelegramURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url invalida %q: %v", raw, err)
	}
	return parsed
}

func TestTelegramChannelSendMessagePostsToBotEndpoint(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	channel := newTelegramTestChannel(t, server, testTelegramToken)

	result, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText)
	if err != nil {
		t.Fatalf("envio falhou: %v", err)
	}
	if result.MessageID != 42 {
		t.Fatalf("message_id = %d, quer 42", result.MessageID)
	}
	if result.ChatID != testTelegramChat {
		t.Fatalf("chat_id = %q, quer %q", result.ChatID, testTelegramChat)
	}
	paths, bodies := recorder.snapshot()
	if len(paths) != 1 {
		t.Fatalf("chamadas = %d, quer 1", len(paths))
	}
	if want := "/bot" + testTelegramToken + "/sendMessage"; paths[0] != want {
		t.Fatalf("caminho = %q, quer %q", paths[0], want)
	}
	if bodies[0]["text"] != testTelegramText {
		t.Fatalf("text = %v, quer %q", bodies[0]["text"], testTelegramText)
	}
	if bodies[0]["chat_id"] != testTelegramChat {
		t.Fatalf("chat_id = %v, quer %q", bodies[0]["chat_id"], testTelegramChat)
	}
}

func TestTelegramChannelSendMessageBlockedByDLPBeforeEgress(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	channel := newTelegramTestChannel(t, server, testTelegramToken)

	_, err := channel.sendMessage(context.Background(), testTelegramChat, "segue a chave AKIAIOSFODNN7EXAMPLE do ambiente")
	if !errors.Is(err, ErrTelegramMessageBlocked) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramMessageBlocked)
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("a guarda de DLP deixou passar %d chamada(s)", len(paths))
	}
}

func TestTelegramChannelRequiresCredential(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	channel := newTelegramTestChannel(t, server, "")
	channel.credential = func(string) string { return "  " }

	if _, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText); !errors.Is(err, ErrTelegramCredentialUnavailable) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramCredentialUnavailable)
	}
	if _, err := channel.verify(context.Background()); !errors.Is(err, ErrTelegramCredentialUnavailable) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramCredentialUnavailable)
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("houve chamada sem credencial: %v", paths)
	}
}

func TestTelegramChannelRejectsMalformedCredential(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	tokens := []string{
		"nao-e-token",
		"12345:curto",
		"123456789:AAHteste_token_de_teste_com_tamanho_ok/../../sendMessage",
		"123456789:AAH teste token com espaco no meio",
		"1234:AAHteste_token_de_teste_com_tamanho_ok",
	}
	for _, token := range tokens {
		channel := newTelegramTestChannel(t, server, token)
		if _, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText); !errors.Is(err, ErrTelegramCredentialInvalid) {
			t.Fatalf("token %q: erro = %v, quer %v", token, err, ErrTelegramCredentialInvalid)
		}
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("houve chamada com credencial invalida: %v", paths)
	}
}

func TestTelegramChannelChatAllowlist(t *testing.T) {
	accepted := []string{"123456", "-1001234567890", "@meucanal"}
	for _, chat := range accepted {
		normalized, err := normalizeTelegramChatID("  " + chat + "  ")
		if err != nil {
			t.Fatalf("chat %q rejeitado: %v", chat, err)
		}
		if normalized != chat {
			t.Fatalf("chat normalizado = %q, quer %q", normalized, chat)
		}
	}
	rejected := []string{
		"",
		"   ",
		"abc",
		"@ab",
		"@meu-canal",
		"1/2",
		"123 456",
		"+5511999999999",
		"../../etc/passwd",
		"123456789012345678901",
		"chat_id=1",
	}
	for _, chat := range rejected {
		if _, err := normalizeTelegramChatID(chat); !errors.Is(err, ErrTelegramChatInvalid) {
			t.Fatalf("chat %q aceito, quer %v", chat, ErrTelegramChatInvalid)
		}
	}
}

func TestTelegramChannelPinsAPIHost(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	hosts := []string{
		"https://api.telegram.org.evil.example",
		"http://api.telegram.org",
		"https://evil.example",
		"ftp://api.telegram.org",
	}
	for _, host := range hosts {
		channel := newTelegramTestChannel(t, server, testTelegramToken)
		channel.baseURL = host
		if _, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText); !errors.Is(err, ErrTelegramHostNotAllowed) {
			t.Fatalf("host %q: erro = %v, quer %v", host, err, ErrTelegramHostNotAllowed)
		}
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("houve chamada para host nao permitido: %v", paths)
	}
}

func TestTelegramChannelEndpointKeepsTokenInPathOnly(t *testing.T) {
	channel := newTelegramTestChannel(t, nil, testTelegramToken)
	base := mustParseTelegramURL(t, "https://api.telegram.org/?rastro=1#fragmento")
	endpoint := channel.endpoint(base, testTelegramToken, "sendMessage")
	want := "https://api.telegram.org/bot" + testTelegramToken + "/sendMessage"
	if endpoint != want {
		t.Fatalf("endpoint = %q, quer %q", endpoint, want)
	}
}

func TestTelegramAuditDestinationNeverCarriesToken(t *testing.T) {
	channel := newTelegramTestChannel(t, nil, testTelegramToken)
	destination := channel.auditDestination(mustParseTelegramURL(t, "https://api.telegram.org"), "sendMessage")
	if strings.Contains(destination, testTelegramToken) {
		t.Fatalf("destino de auditoria expoe o token: %q", destination)
	}
	if destination != "https://api.telegram.org/"+telegramRedactedToken+"/sendMessage" {
		t.Fatalf("destino = %q", destination)
	}
}

func TestTelegramChannelVerifyReturnsBotIdentity(t *testing.T) {
	server, recorder := newTelegramTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":777,"is_bot":true,"first_name":"Hades","username":"hades_bot"}}`))
	})
	channel := newTelegramTestChannel(t, server, testTelegramToken)

	identity, err := channel.verify(context.Background())
	if err != nil {
		t.Fatalf("verificacao falhou: %v", err)
	}
	if identity.ID != 777 || identity.Username != "hades_bot" || identity.FirstName != "Hades" {
		t.Fatalf("identidade = %+v", identity)
	}
	paths, bodies := recorder.snapshot()
	if want := "/bot" + testTelegramToken + "/getMe"; len(paths) != 1 || paths[0] != want {
		t.Fatalf("caminhos = %v, quer [%s]", paths, want)
	}
	if len(bodies[0]) != 0 {
		t.Fatalf("getMe enviou corpo inesperado: %v", bodies[0])
	}
}

func TestTelegramChannelErrorsNeverExposeToken(t *testing.T) {
	t.Run("erro da API", func(t *testing.T) {
		server, _ := newTelegramTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
		})
		channel := newTelegramTestChannel(t, server, testTelegramToken)
		_, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText)
		if !errors.Is(err, ErrTelegramCallFailed) {
			t.Fatalf("erro = %v, quer %v", err, ErrTelegramCallFailed)
		}
		if strings.Contains(err.Error(), testTelegramToken) {
			t.Fatalf("erro expoe o token: %v", err)
		}
		if !strings.Contains(err.Error(), "Unauthorized") {
			t.Fatalf("erro sem a descricao do provedor: %v", err)
		}
	})

	t.Run("falha de transporte", func(t *testing.T) {
		server, _ := newTelegramTestServer(t, nil)
		channel := newTelegramTestChannel(t, server, testTelegramToken)
		server.Close()
		_, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText)
		if !errors.Is(err, ErrTelegramCallFailed) {
			t.Fatalf("erro = %v, quer %v", err, ErrTelegramCallFailed)
		}
		if strings.Contains(err.Error(), testTelegramToken) {
			t.Fatalf("erro expoe o token: %v", err)
		}
		if !strings.Contains(err.Error(), telegramRedactedToken) {
			t.Fatalf("erro sem a mascara do token: %v", err)
		}
	})

	t.Run("credencial invalida", func(t *testing.T) {
		token := "123456789:AAHteste_token_de_teste_com_tamanho_ok"
		channel := newTelegramTestChannel(t, nil, token+" ")
		channel.credential = func(string) string { return token + "x/y" }
		_, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText)
		if !errors.Is(err, ErrTelegramCredentialInvalid) {
			t.Fatalf("erro = %v, quer %v", err, ErrTelegramCredentialInvalid)
		}
		if strings.Contains(err.Error(), token) {
			t.Fatalf("erro expoe o token: %v", err)
		}
	})
}

func TestTelegramChannelRejectsOversizedResponse(t *testing.T) {
	server, _ := newTelegramTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":"` + strings.Repeat("a", telegramMaxResponseBytes+1024) + `"}`))
	})
	channel := newTelegramTestChannel(t, server, testTelegramToken)

	if _, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText); !errors.Is(err, ErrTelegramResponseInvalid) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramResponseInvalid)
	}
}

func TestTelegramChannelRejectsInvalidEnvelope(t *testing.T) {
	replies := []string{
		`nao-e-json`,
		`{"ok":true}`,
		`{"result":{"message_id":42,"chat":{"id":123}}}`,
		`{"ok":true,"result":{"message_id":0,"chat":{"id":123}}}`,
		`{"ok":true,"result":"texto"}`,
	}
	for index, reply := range replies {
		server, _ := newTelegramTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(reply))
		})
		channel := newTelegramTestChannel(t, server, testTelegramToken)
		if _, err := channel.sendMessage(context.Background(), testTelegramChat, testTelegramText); !errors.Is(err, ErrTelegramResponseInvalid) {
			t.Fatalf("resposta %d (%s): erro = %v, quer %v", index, reply, err, ErrTelegramResponseInvalid)
		}
	}
}

func TestTelegramChannelRejectsMessageOutsideLimits(t *testing.T) {
	server, recorder := newTelegramTestServer(t, nil)
	channel := newTelegramTestChannel(t, server, testTelegramToken)

	if _, err := channel.sendMessage(context.Background(), testTelegramChat, "   "); !errors.Is(err, ErrTelegramMessageEmpty) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramMessageEmpty)
	}
	if _, err := channel.sendMessage(context.Background(), testTelegramChat, strings.Repeat("a", telegramMaxTextBytes+1)); !errors.Is(err, ErrTelegramMessageTooLong) {
		t.Fatalf("erro = %v, quer %v", err, ErrTelegramMessageTooLong)
	}
	paths, _ := recorder.snapshot()
	if len(paths) != 0 {
		t.Fatalf("houve chamada com mensagem invalida: %v", paths)
	}
}

func TestTelegramChannelDescriptionSanitizer(t *testing.T) {
	if got := sanitizeTelegramDescription("   "); got != "sem descricao" {
		t.Fatalf("descricao vazia = %q", got)
	}
	long := strings.Repeat("á", telegramMaxDescriptionRunes+50)
	got := sanitizeTelegramDescription(long)
	if len([]rune(got)) != telegramMaxDescriptionRunes {
		t.Fatalf("descricao truncada com %d runas, quer %d", len([]rune(got)), telegramMaxDescriptionRunes)
	}
	if redactTelegramError(nil) != "" {
		t.Fatalf("erro nulo virou texto")
	}
	scrubbed := redactTelegramError(fmt.Errorf(`Post "https://api.telegram.org/bot%s/sendMessage": dial tcp: connect: connection refused`, testTelegramToken))
	if strings.Contains(scrubbed, testTelegramToken) {
		t.Fatalf("mascara nao removeu o token: %q", scrubbed)
	}
	if !strings.Contains(scrubbed, telegramRedactedToken) {
		t.Fatalf("mascara ausente: %q", scrubbed)
	}
}

func TestTelegramToolDescriptorsAreDenyByDefault(t *testing.T) {
	send := imTelegramSendTool{}.Descriptor()
	if send.Name != "im.telegram.send" || send.Version != "1" {
		t.Fatalf("descritor de envio = %+v", send)
	}
	if send.Risk != RiskExternalSideEffect || !send.RequiresApproval {
		t.Fatalf("envio sem aprovacao obrigatoria: %+v", send)
	}
	if len(send.Scopes) != 1 || send.Scopes[0] != "connector:external" {
		t.Fatalf("escopos de envio = %v", send.Scopes)
	}
	if capabilityAllowed(send, nil) {
		t.Fatalf("envio permitido sem concessao de escopo")
	}
	if !capabilityAllowed(send, []string{"connector:external"}) {
		t.Fatalf("envio negado com o escopo declarado")
	}
	if capabilityAllowed(send, []string{"workspace:read"}) {
		t.Fatalf("envio permitido por um escopo alheio")
	}

	verify := imTelegramVerifyTool{}.Descriptor()
	if verify.Name != "im.telegram.verify" || verify.Version != "1" {
		t.Fatalf("descritor de verificacao = %+v", verify)
	}
	if verify.Risk != RiskRead || verify.RequiresApproval {
		t.Fatalf("verificacao exige aprovacao indevida: %+v", verify)
	}
	if capabilityAllowed(verify, nil) {
		t.Fatalf("verificacao permitida sem concessao de escopo")
	}
	if !capabilityAllowed(verify, []string{"connector:external"}) {
		t.Fatalf("verificacao negada com o escopo declarado")
	}
}

func TestNewRegistryExposesTelegramChannelTools(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"im.telegram.send", "im.telegram.verify"} {
		tool, ok := registry.Get(name)
		if !ok || tool == nil {
			t.Fatalf("ferramenta %q ausente do registro", name)
		}
	}
}

func TestTelegramToolExecuteValidatesInputBeforeEgress(t *testing.T) {
	tool := imTelegramSendTool{}
	cases := []struct {
		name  string
		input map[string]any
		want  error
	}{
		{"chat invalido", map[string]any{"chat_id": "../../etc/passwd", "text": testTelegramText}, ErrTelegramChatInvalid},
		{"chat ausente", map[string]any{"text": testTelegramText}, ErrTelegramChatInvalid},
		{"texto vazio", map[string]any{"chat_id": testTelegramChat}, ErrTelegramMessageEmpty},
		{"texto longo", map[string]any{"chat_id": testTelegramChat, "text": strings.Repeat("a", telegramMaxTextBytes+1)}, ErrTelegramMessageTooLong},
		{"texto sensivel", map[string]any{"chat_id": testTelegramChat, "text": "chave sk-abcdefghijklmnopqrstuvwxyz0123456789 vazada"}, ErrTelegramMessageBlocked},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := tool.Execute(context.Background(), ToolContext{}, testCase.input); !errors.Is(err, testCase.want) {
				t.Fatalf("erro = %v, quer %v", err, testCase.want)
			}
		})
	}
}
