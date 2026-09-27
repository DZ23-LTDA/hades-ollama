package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) requirePluginAdmin(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "plugin lifecycle requires organization admin"})
		return false
	}
	return true
}

func pluginCatalog(c *gin.Context, a *agentAPI) {
	organizationID := ""
	if a.authRequired {
		organizationID = agentOrganizationID(c)
	}
	if !a.authRequired {
		organizationID = agent.LocalOrganizationID
	}
	connectors := a.runtime.ConnectorsForOrganization(organizationID)
	mcp := a.runtime.MCPServersForOrganization(organizationID)
	remoteMCP := a.runtime.RemoteMCPServersForOrganization(organizationID)
	skills := a.runtime.SkillsForOrganization(organizationID)
	c.JSON(http.StatusOK, gin.H{
		"connectors": connectors,
		"mcp":        mcp,
		"remote_mcp": remoteMCP,
		"skills":     skills,
	})
}

func connectorCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"connectors": agent.ConnectorCatalog()})
}

func writePluginLifecycleError(c *gin.Context, err error) {
	if errors.Is(err, agent.ErrPluginOrganizationScope) || errors.Is(err, agent.ErrPluginNotFound) {
		writeAgentError(c, http.StatusNotFound, agent.ErrPluginNotFound)
		return
	}
	writeAgentError(c, http.StatusBadRequest, err)
}

func writePluginRegistrationError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, agent.ErrPluginOrganizationScope) {
		status = http.StatusForbidden
	}
	writeAgentError(c, status, err)
}

func (a *agentAPI) enableConnector(c *gin.Context)  { a.connectorLifecycle(c, true, false) }
func (a *agentAPI) disableConnector(c *gin.Context) { a.connectorLifecycle(c, false, false) }
func (a *agentAPI) removeConnector(c *gin.Context)  { a.connectorLifecycle(c, false, true) }

func (a *agentAPI) registerConnector(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.ConnectorConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if input.CredentialConfigured {
		writeAgentError(c, http.StatusBadRequest, errors.New("credential_configured is server-derived and cannot be supplied"))
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterConnectorForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		// The unauthenticated local mode has one operator scope. The runtime
		// still validates the HTTPS endpoint, methods and environment names.
		err = a.runtime.RegisterConnector(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) connectorLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveConnectorForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveConnector(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetConnectorEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetConnectorEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableMCP(c *gin.Context)  { a.mcpLifecycle(c, true, false) }
func (a *agentAPI) disableMCP(c *gin.Context) { a.mcpLifecycle(c, false, false) }
func (a *agentAPI) removeMCP(c *gin.Context)  { a.mcpLifecycle(c, false, true) }

func (a *agentAPI) registerMCP(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.MCPServerConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterMCPForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterMCP(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) mcpLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveMCPForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveMCP(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetMCPEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetMCPEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableRemoteMCP(c *gin.Context)  { a.remoteMCPLifecycle(c, true, false) }
func (a *agentAPI) disableRemoteMCP(c *gin.Context) { a.remoteMCPLifecycle(c, false, false) }
func (a *agentAPI) removeRemoteMCP(c *gin.Context)  { a.remoteMCPLifecycle(c, false, true) }

func (a *agentAPI) registerRemoteMCP(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.RemoteMCPServerConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterRemoteMCPForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterRemoteMCP(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) remoteMCPLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveRemoteMCPForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveRemoteMCP(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetRemoteMCPEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetRemoteMCPEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableSkill(c *gin.Context)  { a.skillLifecycle(c, true, false) }
func (a *agentAPI) disableSkill(c *gin.Context) { a.skillLifecycle(c, false, false) }
func (a *agentAPI) removeSkill(c *gin.Context)  { a.skillLifecycle(c, false, true) }

func (a *agentAPI) registerSkill(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.SkillManifest
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if input.Trusted || input.Enabled {
		writeAgentError(c, http.StatusBadRequest, errors.New("skill trusted and enabled state are server-derived"))
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterSkillForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterSkill(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) skillLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveSkillForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveSkill(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetSkillEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetSkillEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}
