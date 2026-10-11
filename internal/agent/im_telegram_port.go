package agent

import (
	"context"
	"errors"
	"strings"
)

// Porta exportada do canal Telegram para as rotas HTTP.
//
// O canal e as ferramentas já existiam, mas **não havia rota**: o comando
// `/telegram` estava marcado indisponível exatamente por isso. Em vez de
// reimplementar um cliente na camada HTTP (o que duplicaria token, DLP e
// consentimento), esta porta expõe o **mesmo** canal com os tipos internos
// convertidos para tipos públicos. Assim as travas continuam num só lugar.

// ErrTelegramNotConfigured indica credencial ausente: o canal não está pronto e
// nenhum envio é tentado.
var ErrTelegramNotConfigured = ErrTelegramCredentialUnavailable

// TelegramSendOutcome é o resultado público de um envio.
type TelegramSendOutcome struct {
	ChatID     string
	MessageID  int64
	DurationMS int64
}

// TelegramBotIdentity é a identidade pública do bot (sem o token).
type TelegramBotIdentity struct {
	ID        int64
	Username  string
	FirstName string
}

// SendTelegramMessage envia pelo canal oficial aplicando, na ordem: validação do
// chat e do texto, credencial, DLP, **consentimento** e só então o egress.
func SendTelegramMessage(ctx context.Context, organizationID string, consent *ConsentRegistry, chatID, text string) (TelegramSendOutcome, error) {
	channel := newTelegramChannel()
	channel.organizationID = strings.TrimSpace(organizationID)
	channel.consent = consent
	result, err := channel.sendMessage(ctx, chatID, text)
	if err != nil {
		return TelegramSendOutcome{}, err
	}
	// Conversão direta: os campos são idênticos, e copiar campo a campo criaria uma
	// segunda definição que pode divergir da primeira.
	return TelegramSendOutcome(result), nil
}

// VerifyTelegramBot confirma a credencial e devolve a identidade do bot sem
// revelar o token.
func VerifyTelegramBot(ctx context.Context) (TelegramBotIdentity, error) {
	channel := newTelegramChannel()
	identity, err := channel.verify(ctx)
	if err != nil {
		return TelegramBotIdentity{}, err
	}
	return TelegramBotIdentity(identity), nil
}

// TelegramErrorIsNotConfigured informa se o erro é ausência de credencial, para a
// rota distinguir "não configurado" de "falhou ao enviar".
func TelegramErrorIsNotConfigured(err error) bool {
	return errors.Is(err, ErrTelegramCredentialUnavailable) || errors.Is(err, ErrTelegramCredentialInvalid)
}
