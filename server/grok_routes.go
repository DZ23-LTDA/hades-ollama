package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/grok"
)

func (a *agentAPI) grokStatus(c *gin.Context) {
	if a.grok == nil {
		c.JSON(http.StatusOK, grok.ProviderStatus{Provider: "xai/grok", State: grok.StatusCataloged, CheckedAt: time.Now().UTC()})
		return
	}
	status := a.grok.Probe(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (a *agentAPI) grokResponses(c *gin.Context) {
	if a.grok == nil || strings.TrimSpace(a.grok.APIKey) == "" {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("xAI/Grok credential is not configured"))
		return
	}
	var request grok.ResponseRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if request.Stream {
		writeAgentError(c, http.StatusNotImplemented, errors.New("Grok streaming is not exposed by this route yet"))
		return
	}
	if err := agent.ValidateOutboundPayload(request); err != nil {
		writeAgentError(c, http.StatusUnprocessableEntity, errors.New("Grok payload rejected by data-egress policy"))
		return
	}
	response, err := a.grok.Responses(c.Request.Context(), request)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, agent.RedactValue(response))
}
