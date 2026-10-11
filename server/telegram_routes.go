package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas do canal Telegram. O comando `/telegram` estava indisponível com o motivo
// "o adaptador de Telegram não está integrado a esta base: sem rota registrada" —
// honesto, porque o canal e as ferramentas existiam mas NÃO havia rota. Aqui a rota
// passa a existir, reusando o MESMO canal e as MESMAS travas: DLP antes do egress e
// consentimento/opt-out antes de falar com terceiros. Nenhum cliente novo foi
// escrito, para não duplicar token, DLP ou consentimento.
//
// O status é honesto: sem credencial o canal aparece NOT_CONFIGURED, com o motivo
// real, em vez de "pronto para usar".
func (a *agentAPI) telegramStatus(c *gin.Context) {
	status := gin.H{
		"channel":          "telegram",
		"credential_env":   agent.TelegramBotTokenEnv,
		"configured":       false,
		"gate_status":      string(agent.GateStatusNotConfigured),
		"consent_required": true,
		"dlp_required":     true,
		"note":             "sem a credencial do bot o canal NAO esta configurado e nenhum envio e tentado",
	}
	identity, err := agent.VerifyTelegramBot(c.Request.Context())
	if err != nil {
		status["unavailable_reason"] = strings.TrimSpace(err.Error())
		c.JSON(http.StatusOK, status)
		return
	}
	status["configured"] = true
	status["gate_status"] = string(agent.GateStatusPass)
	status["bot_id"] = identity.ID
	if identity.Username != "" {
		status["bot_username"] = identity.Username
	}
	c.JSON(http.StatusOK, status)
}

// telegramSend envia uma mensagem reusando o canal. A ordem é deliberada:
// validação → credencial → DLP → CONSENTIMENTO → egress (a mesma do canal).
func (a *agentAPI) telegramSend(c *gin.Context) {
	var input struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errTelegramUnavailable)
		return
	}
	var consent *agent.ConsentRegistry
	if runtimeConsent := a.runtime.Consent(); runtimeConsent != nil {
		consent = runtimeConsent
	}
	outcome, err := agent.SendTelegramMessage(c.Request.Context(), agentOrganizationID(c), consent, input.ChatID, input.Text)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// A resposta não devolve a credencial e diz que o consentimento foi exigido.
	c.JSON(http.StatusOK, gin.H{
		"channel":          "telegram",
		"chat_id":          outcome.ChatID,
		"message_id":       outcome.MessageID,
		"duration_ms":      outcome.DurationMS,
		"consent_required": true,
		"sent_at":          time.Now().UTC(),
	})
}

// errTelegramUnavailable é o erro explícito quando o runtime não está disponível:
// melhor dizer que não dá para enviar do que devolver sucesso vazio.
var errTelegramUnavailable = agent.ErrTelegramNotConfigured
