package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Caixa de entrada unificada (estágio 29). Destrava `/inbox`, `/handoff` e
// `/support`, que estavam indisponíveis por não existir inbox. Cada rota é escopada
// pela organização da sessão, e a resposta ao contato passa pelo CONSENTIMENTO.
func (a *agentAPI) inboxStore() *agent.InboxStore {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.Inbox()
}

var errInboxUnavailable = agent.ErrInboxConversationGone

func (a *agentAPI) listInboxConversations(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	conversations := store.ListConversations(agentOrganizationID(c), c.Query("status"))
	c.JSON(http.StatusOK, gin.H{"conversations": conversations, "count": len(conversations)})
}

// appendInboxMessage aceita mensagem RECEBIDA. Reentrega do provedor com o mesmo
// `external_id` devolve a mensagem original marcada como deduplicada, em vez de
// duplicar — webhook duplicado não pode virar mensagem duplicada (§11.7).
func (a *agentAPI) appendInboxMessage(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	var input struct {
		Channel     string `json:"channel"`
		ContactID   string `json:"contact_id"`
		ContactName string `json:"contact_name"`
		Body        string `json:"body"`
		ExternalID  string `json:"external_id"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	conversation, message, err := store.AppendInbound(agentOrganizationID(c), input.Channel, input.ContactID, input.ContactName, input.Body, input.ExternalID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusCreated
	if message.Deduplicated {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"conversation": conversation, "message": message, "deduplicated": message.Deduplicated})
}

func (a *agentAPI) getInboxConversation(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	conversation, messages, err := store.Conversation(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	payload := gin.H{"conversation": conversation, "messages": messages}
	// O consentimento do contato é anexado à visão para o atendente saber, sem
	// consultar outro lugar, se pode responder.
	if a.runtime != nil && a.runtime.Consent() != nil {
		if err := a.runtime.Consent().AuthorizeContact(conversation.OrganizationID, conversation.Channel, conversation.ContactID, time.Now().UTC()); err != nil {
			payload["reply_blocked"] = true
			payload["reply_blocked_reason"] = err.Error()
		} else {
			payload["reply_blocked"] = false
		}
	}
	c.JSON(http.StatusOK, payload)
}

// replyInboxConversation responde ao contato. O consentimento é checado no envio.
func (a *agentAPI) replyInboxConversation(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	var input struct {
		Body string `json:"body"`
		Kind string `json:"kind"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var consent *agent.ConsentRegistry
	if a.runtime != nil {
		consent = a.runtime.Consent()
	}
	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = "agent"
	}
	conversation, message, err := store.AppendOutbound(agentOrganizationID(c), c.Param("id"), kind, agentActorID(c), input.Body, consent, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"conversation": conversation, "message": message})
}

func (a *agentAPI) assignInboxConversation(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	var input struct {
		Assignee string `json:"assignee"`
		Kind     string `json:"kind"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	conversation, err := store.Assign(agentOrganizationID(c), c.Param("id"), input.Assignee, input.Kind, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, conversation)
}

// handoffInboxConversation é o alvo do comando `/handoff`: exige MOTIVO, move a
// conversa para atendimento humano e grava a passagem na própria conversa.
func (a *agentAPI) handoffInboxConversation(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	var input struct {
		Reason   string `json:"reason"`
		Assignee string `json:"assignee"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	conversation, message, err := store.HandoffToHuman(agentOrganizationID(c), c.Param("id"), agentActorID(c), input.Assignee, input.Reason, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation": conversation, "message": message})
}

func (a *agentAPI) closeInboxConversation(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	conversation, err := store.Close(agentOrganizationID(c), c.Param("id"), agentActorID(c), input.Reason, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, conversation)
}

// inboxMetrics devolve SLA e volume: indicadores MEDIDOS, não presumidos.
func (a *agentAPI) inboxMetrics(c *gin.Context) {
	store := a.inboxStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errInboxUnavailable)
		return
	}
	c.JSON(http.StatusOK, store.Metrics(agentOrganizationID(c), time.Now().UTC()))
}
