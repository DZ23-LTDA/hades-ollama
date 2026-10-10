package agent

import (
	"context"
	"strings"
)

// Ferramentas do canal de IM do Telegram.
//
// O escopo reutilizado e "connector:external", ja presente na politica padrao:
// o canal fala com um provedor externo exatamente como os demais conectores, e
// ampliar a politica exigiria uma nova concessao para o mesmo risco. O envio
// continua sujeito a aprovacao humana explicita (RequiresApproval) e a
// concessao de escopo por missao, portanto um agente sem "connector:external"
// nao consegue disparar mensagem alguma.
type imTelegramSendTool struct{}

func (imTelegramSendTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:             "im.telegram.send",
		Version:          "1",
		Description:      "Enviar mensagem de texto em um chat do Telegram pelo bot configurado, com aprovacao humana e bloqueio de conteudo sensivel",
		Risk:             RiskExternalSideEffect,
		Scopes:           []string{"connector:external"},
		RequiresApproval: true,
	}
}

func (imTelegramSendTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	chatID := strings.TrimSpace(stringInput(input, "chat_id", ""))
	text := stringInput(input, "text", "")
	channel := newTelegramChannel()
	channel.organizationID = strings.TrimSpace(toolContext.OrganizationID)
	result, err := channel.sendMessage(ctx, chatID, text)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Value: map[string]any{
		"provider":       "telegram",
		"channel":        "telegram",
		"host":           telegramAPIHost,
		"credential_env": TelegramBotTokenEnv,
		"chat_id":        result.ChatID,
		"message_id":     result.MessageID,
		"text_bytes":     len([]byte(text)),
		"duration_ms":    result.DurationMS,
		"approval":       "humana obrigatoria antes do envio",
	}}, nil
}

type imTelegramVerifyTool struct{}

func (imTelegramVerifyTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:        "im.telegram.verify",
		Version:     "1",
		Description: "Confirmar a credencial do bot do Telegram e devolver a identidade do bot, sem revelar o token",
		Risk:        RiskRead,
		Scopes:      []string{"connector:external"},
	}
}

func (imTelegramVerifyTool) Execute(ctx context.Context, toolContext ToolContext, _ map[string]any) (ToolResult, error) {
	channel := newTelegramChannel()
	channel.organizationID = strings.TrimSpace(toolContext.OrganizationID)
	identity, err := channel.verify(ctx)
	if err != nil {
		return ToolResult{}, err
	}
	value := map[string]any{
		"provider":       "telegram",
		"channel":        "telegram",
		"host":           telegramAPIHost,
		"credential_env": TelegramBotTokenEnv,
		"configured":     true,
		"bot_id":         identity.ID,
	}
	if identity.Username != "" {
		value["bot_username"] = identity.Username
	}
	if identity.FirstName != "" {
		value["bot_name"] = identity.FirstName
	}
	return ToolResult{Value: value}, nil
}

var (
	_ Tool = imTelegramSendTool{}
	_ Tool = imTelegramVerifyTool{}
)
