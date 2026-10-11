package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas do Omnichannel Runtime (estágio 34). O webhook é a ÚNICA porta de entrada de
// evento externo e recusa tudo que não for autenticado:
// conector (tenant) → assinatura → janela de replay → rate limit → dedupe → inbox.
const omnichannelMaxBodyBytes = 256 * 1024

func (a *agentAPI) omnichannel() *agent.OmnichannelRuntime {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.Omnichannel()
}

func (a *agentAPI) registerOmnichannelConnector(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	var input struct {
		Channel   string `json:"channel"`
		Name      string `json:"name"`
		SecretEnv string `json:"secret_env"`
		Enabled   *bool  `json:"enabled"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	connector, err := runtime.RegisterConnector(agent.OmnichannelConnector{
		OrganizationID: agentOrganizationID(c),
		Channel:        input.Channel,
		Name:           input.Name,
		SecretEnv:      input.SecretEnv,
		Enabled:        enabled,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// A resposta traz apenas o NOME da variável de ambiente; o valor nunca sai daqui.
	c.JSON(http.StatusCreated, connector)
}

func (a *agentAPI) listOmnichannelConnectors(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	connectors := runtime.Connectors(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"connectors": connectors, "count": len(connectors)})
}

// ingestOmnichannelWebhook recebe o evento do provedor. Códigos deliberados:
// 401 assinatura ausente/ inválida, 409 fora da janela (replay), 429 limite excedido,
// 404 canal sem conector, 400 evento inválido, 202 aceito, 200 reentrega deduplicada.
func (a *agentAPI) ingestOmnichannelWebhook(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, omnichannelMaxBodyBytes))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(c.GetHeader(agent.OmnichannelTimestampHeader)), 10, 64)
	if err != nil {
		// Sem carimbo não há como defender contra replay: recusa explícita.
		writeAgentError(c, http.StatusUnauthorized, agent.ErrOmnichannelReplay)
		return
	}
	var input struct {
		ExternalMessageID string            `json:"external_message_id"`
		ContactID         string            `json:"contact_id"`
		MessageType       string            `json:"message_type"`
		Preview           string            `json:"preview"`
		Sender            string            `json:"sender"`
		Recipient         string            `json:"recipient"`
		ContentReference  string            `json:"content_reference"`
		Attachments       []string          `json:"attachments"`
		CorrelationID     string            `json:"correlation_id"`
		IdempotencyKey    string            `json:"idempotency_key"`
		AgentAssignment   string            `json:"agent_assignment"`
		ConsentState      string            `json:"consent_state"`
		AuditMetadata     map[string]string `json:"audit_metadata"`
	}
	if err := decodeJSONBytes(body, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	event, err := runtime.Ingest(
		agentOrganizationID(c),
		c.Param("channel"),
		c.GetHeader(agent.OmnichannelSignatureHeader),
		timestamp,
		body,
		agent.OmnichannelEvent{
			ExternalMessageID: input.ExternalMessageID,
			ContactID:         input.ContactID,
			MessageType:       input.MessageType,
			// A prévia é truncada pelo próprio runtime; o corpo inteiro não é guardado.
			Preview:          input.Preview,
			Sender:           input.Sender,
			Recipient:        input.Recipient,
			ContentReference: input.ContentReference,
			Attachments:      input.Attachments,
			CorrelationID:    input.CorrelationID,
			IdempotencyKey:   input.IdempotencyKey,
			AgentAssignment:  input.AgentAssignment,
			ConsentState:     input.ConsentState,
			AuditMetadata:    input.AuditMetadata,
		},
		time.Now().UTC(),
	)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusAccepted
	if event.Deduplicated {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"event": event, "deduplicated": event.Deduplicated})
}

func (a *agentAPI) listOmnichannelEvents(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	events := runtime.Events(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"events": events, "count": len(events)})
}

func (a *agentAPI) listOmnichannelDeadLetters(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	entries := runtime.DeadLetters(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"dead_letters": entries, "count": len(entries)})
}

// retryOmnichannelDeadLetter reprocessa explicitamente um evento da fila de mortos.
func (a *agentAPI) retryOmnichannelDeadLetter(c *gin.Context) {
	runtime := a.omnichannel()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, agent.ErrOmnichannelNotConfigured)
		return
	}
	entry, err := runtime.RetryDeadLetter(agentOrganizationID(c), c.Param("id"), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"retried": entry.ID, "event": entry.Event})
}

// decodeJSONBytes decodifica a partir dos bytes já lidos. O webhook lê o corpo bruto
// primeiro porque a ASSINATURA é calculada sobre ele: decodificar antes perderia a
// garantia de que a assinatura corresponde exatamente ao conteúdo recebido.
func decodeJSONBytes(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}
