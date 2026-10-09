package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// requireSupervisorAdmin gates the deployment-wide supervisor (SEC-09). The
// supervisor is a single global runtime component, so reconfiguring it or
// triggering a tick is a deployment-level action: it must never be reachable by
// an ordinary member, and in a multi-tenant deployment it must not let one
// organization drive another's supervisor. OLLAMA_SUPERVISOR_ORG pins the owning
// organization; when unset the check is role-based (single-tenant/local default).
func (a *agentAPI) requireSupervisorAdmin(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "supervisor administration requires organization owner or admin"})
		return false
	}
	if pinned := strings.TrimSpace(os.Getenv("OLLAMA_SUPERVISOR_ORG")); pinned != "" {
		if strings.TrimSpace(membership.OrganizationID) != pinned {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "supervisor is owned by a different organization"})
			return false
		}
	}
	return true
}

func (a *agentAPI) supervisorStatus(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"running": false,
			"error":   "supervisor not initialized in runtime",
		})
		return
	}
	st := a.runtime.Supervisor().Status()
	c.JSON(http.StatusOK, st)
}

func (a *agentAPI) supervisorConfig(c *gin.Context) {
	// SEC-09: configuring the supervisor is an administrative action, restricted
	// to the organization owner/admin (same gate as supervisorStatus).
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "supervisor not initialized in runtime"})
		return
	}

	var req agent.SupervisorConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supervisor config: " + err.Error()})
		return
	}

	a.runtime.Supervisor().SetConfig(req)
	st := a.runtime.Supervisor().Status()
	c.JSON(http.StatusOK, st)
}

func (a *agentAPI) supervisorTick(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "supervisor not initialized in runtime"})
		return
	}

	now := time.Now().UTC()
	res, err := a.runtime.Supervisor().Tick(c.Request.Context(), now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "result": res})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": res,
		"status": a.runtime.Supervisor().Status(),
	})
}
