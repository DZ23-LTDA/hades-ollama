package server

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) requireWhatsAppAdmin(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "WhatsApp administration requires organization owner or admin"})
		return false
	}
	return true
}

func (a *agentAPI) whatsappWebhookVerify(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.String(http.StatusServiceUnavailable, "whatsapp gateway not available")
		return
	}

	if challenge, ok := a.runtime.WhatsApp().VerifyWebhook(c.Request); ok {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", challenge)
		return
	}

	c.String(http.StatusBadRequest, "invalid verification request")
}

func (a *agentAPI) whatsappWebhook(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	// Bound the webhook body before buffering it: the HMAC signature is only
	// verified inside ProcessWebhook (after the full read), so without a cap a
	// multi-GB POST would be buffered into memory before being rejected. Webhook
	// payloads are small JSON; mirror the agent JSON limit.
	const maxWhatsAppWebhookBody = 4 << 20 // 4 MiB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWhatsAppWebhookBody)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "webhook body too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Detect backend or fallback to active
	backendStr := strings.TrimSpace(c.Query("backend"))
	backend := a.runtime.WhatsApp().ActiveBackend()
	if backendStr == string(agent.WhatsAppBackendCloudAPI) {
		backend = agent.WhatsAppBackendCloudAPI
	} else if backendStr == string(agent.WhatsAppBackendEvolution) {
		backend = agent.WhatsAppBackendEvolution
	}

	results, err := a.runtime.WhatsApp().ProcessWebhook(c.Request.Context(), backend, body, c.Request.Header)
	if err != nil {
		if errors.Is(err, agent.ErrWhatsAppInvalidSignature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
			return
		}
		if errors.Is(err, agent.ErrWhatsAppWebhookNotConfigured) {
			c.JSON(http.StatusFailedDependency, gin.H{"error": "whatsapp webhook is NOT_CONFIGURED"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"processed_count": len(results),
		"results":         results,
	})
}

func (a *agentAPI) whatsappStatus(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	summary := a.runtime.WhatsApp().StatusSummary()
	c.JSON(http.StatusOK, summary)
}

func (a *agentAPI) whatsappSend(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	var req agent.WhatsAppOutboundMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid send request body"})
		return
	}

	if strings.TrimSpace(req.To) == "" || strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to and text are required"})
		return
	}

	if a.runtime.WhatsApp().Status() == agent.GateStatusNotConfigured {
		c.JSON(http.StatusFailedDependency, gin.H{"error": "whatsapp adapter is NOT_CONFIGURED"})
		return
	}

	res, err := a.runtime.WhatsApp().SendMessage(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, agent.ErrWhatsAppAdapterNotConfig) {
			c.JSON(http.StatusFailedDependency, gin.H{"error": "whatsapp adapter is NOT_CONFIGURED"})
			return
		}
		if errors.Is(err, agent.ErrWhatsAppUnauthorized) || errors.Is(err, agent.ErrWhatsAppOutboundApproval) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (a *agentAPI) whatsappDLQ(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	dlq := a.runtime.WhatsApp().GetDLQ()
	c.JSON(http.StatusOK, gin.H{
		"count": len(dlq),
		"items": dlq,
	})
}

func (a *agentAPI) whatsappClearDLQ(c *gin.Context) {
	if !a.requireWhatsAppAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	a.runtime.WhatsApp().ClearDLQ()
	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}

func (a *agentAPI) whatsappAllowlist(c *gin.Context) {
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	allowlist := a.runtime.WhatsApp().GetAllowlist()
	c.JSON(http.StatusOK, gin.H{
		"count": len(allowlist),
		"items": allowlist,
	})
}

func (a *agentAPI) whatsappSetContactPolicy(c *gin.Context) {
	if !a.requireWhatsAppAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	var policy agent.WhatsAppContactPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact policy"})
		return
	}

	if strings.TrimSpace(policy.PhoneNumber) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone_number is required"})
		return
	}

	a.runtime.WhatsApp().SetContactPolicy(policy)
	c.JSON(http.StatusOK, gin.H{
		"status": "updated",
		"policy": policy,
	})
}

func (a *agentAPI) whatsappRemoveContactPolicy(c *gin.Context) {
	if !a.requireWhatsAppAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	phone := c.Param("phone")
	if strings.TrimSpace(phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}

	a.runtime.WhatsApp().RemoveContactPolicy(phone)
	c.JSON(http.StatusOK, gin.H{
		"status": "removed",
		"phone":  phone,
	})
}

func (a *agentAPI) whatsappConfig(c *gin.Context) {
	if !a.requireWhatsAppAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.WhatsApp() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "whatsapp gateway not available"})
		return
	}

	var req struct {
		ActiveBackend string `json:"active_backend,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config body"})
		return
	}

	if req.ActiveBackend == string(agent.WhatsAppBackendCloudAPI) {
		a.runtime.WhatsApp().SetActiveBackend(agent.WhatsAppBackendCloudAPI)
	} else if req.ActiveBackend == string(agent.WhatsAppBackendEvolution) {
		a.runtime.WhatsApp().SetActiveBackend(agent.WhatsAppBackendEvolution)
	}

	c.JSON(http.StatusOK, a.runtime.WhatsApp().StatusSummary())
}
