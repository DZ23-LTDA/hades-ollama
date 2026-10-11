package server

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// errSupervisorUnavailable é o erro explícito quando o runtime não tem supervisor:
// melhor dizer que não está disponível do que devolver lista vazia como se não
// houvesse incidente.
var errSupervisorUnavailable = errors.New("supervisor não está disponível no runtime")

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

// supervisorIncidents lista o registro de incidente com estado e recuperação.
// A guarda é a MESMA do supervisor (SEC-09): incidente operacional é assunto de
// administrador da organização, não de membro comum.
func (a *agentAPI) supervisorIncidents(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusOK, gin.H{"incidents": []agent.SupervisorIncident{}, "count": 0})
		return
	}
	incidents := a.runtime.Supervisor().Incidents()
	c.JSON(http.StatusOK, gin.H{"incidents": incidents, "count": len(incidents)})
}

// supervisorAcknowledgeIncident registra CIÊNCIA humana sem declarar resolução.
func (a *agentAPI) supervisorAcknowledgeIncident(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSupervisorUnavailable)
		return
	}
	updated, err := a.runtime.Supervisor().AcknowledgeIncident(c.Param("id"), agentActorID(c), time.Now().UTC())
	if err != nil {
		if errors.Is(err, agent.ErrSupervisorIncidentNotFound) {
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// supervisorResolveIncident fecha o incidente por decisão humana, com motivo.
func (a *agentAPI) supervisorResolveIncident(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSupervisorUnavailable)
		return
	}
	var request struct {
		Resolution string `json:"resolution"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.runtime.Supervisor().ResolveIncident(c.Param("id"), agentActorID(c), request.Resolution, time.Now().UTC())
	if err != nil {
		if errors.Is(err, agent.ErrSupervisorIncidentNotFound) {
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// supervisorTicks devolve o diário dos últimos ticks (evidência "logs de
// supervisão"), incluindo os erros de cada tick.
func (a *agentAPI) supervisorTicks(c *gin.Context) {
	if !a.requireSupervisorAdmin(c) {
		return
	}
	if a.runtime == nil || a.runtime.Supervisor() == nil {
		c.JSON(http.StatusOK, gin.H{"ticks": []agent.SupervisorTickRecord{}, "count": 0})
		return
	}
	journal := a.runtime.Supervisor().TickJournal()
	c.JSON(http.StatusOK, gin.H{"ticks": journal, "count": len(journal)})
}

// supervisorStatus mostra o estado do supervisor (SEC-09).
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
