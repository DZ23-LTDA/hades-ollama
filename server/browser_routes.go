package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// browserEnvironment reports whether the Playwright-backed browser operator can
// run on this machine, with pt-BR guidance for any missing dependency. Local
// organization only — the operator itself is gated to the single-user local
// org until a strict OS/network sandbox exists.
func (a *agentAPI) browserEnvironment(c *gin.Context) {
	if a.organizationID(c) != agent.LocalOrganizationID {
		writeAgentError(c, http.StatusForbidden, errors.New("o operador de navegador está disponível apenas na organização local"))
		return
	}
	c.JSON(http.StatusOK, agent.BrowserEnvironment(c.Request.Context()))
}

// browserEnvironmentSetup installs the missing browser-operator dependencies
// (Playwright + its Chromium) on explicit user request. Local organization only.
func (a *agentAPI) browserEnvironmentSetup(c *gin.Context) {
	if a.organizationID(c) != agent.LocalOrganizationID {
		writeAgentError(c, http.StatusForbidden, errors.New("o operador de navegador está disponível apenas na organização local"))
		return
	}
	result := agent.SetupBrowserEnvironment(c.Request.Context())
	status := http.StatusOK
	if !result.Status.Ready {
		// The setup ran but the environment is still not ready; surface it as a
		// 207-style partial outcome the UI can show without treating it as a hard
		// server error.
		status = http.StatusAccepted
	}
	c.JSON(status, result)
}
