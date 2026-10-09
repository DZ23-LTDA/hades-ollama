package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// getEgressLogs returns the audited egress decisions from the unified zero-trust policy.
func (a *agentAPI) getEgressLogs(c *gin.Context) {
	// O buffer de egress é global ao processo. O resultado é sempre filtrado
	// pela organização do chamador (ListForOrganization/FilterForOrganization),
	// então um chamador com locatário autenticado só enxerga os próprios
	// registros. Quem não tem locatário no contexto (queda fail-closed para a
	// organização local) é o único que poderia pedir a visão de processo
	// inteiro, e por isso precisa do opt-in do operador.
	if tenantOrganizationID(c) == "" && !a.globalProcessScopeAllowed(c, "egress audit log") {
		return
	}
	organizationID := a.organizationID(c)
	limitStr := c.Query("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	callsite := c.Query("callsite")
	var entries []agent.EgressDecision
	if callsite != "" {
		entries = agent.DefaultEgressAuditStore.FilterForOrganization(organizationID, callsite, limit)
	} else {
		entries = agent.DefaultEgressAuditStore.ListForOrganization(organizationID, limit)
	}
	c.JSON(http.StatusOK, gin.H{
		"count":   len(entries),
		"entries": entries,
	})
}

// getEgressStatus returns summary metrics and current state of the zero-trust egress policy.
func (a *agentAPI) getEgressStatus(c *gin.Context) {
	// SEC-10: the status counters are scoped to the caller's organization. The
	// global view is intentionally not exposed here; a process-wide audit would
	// leak other tenants' callsites.
	all := agent.DefaultEgressAuditStore.ListForOrganization(a.organizationID(c), 1000)
	allowedCount := 0
	blockedCount := 0
	callsites := make(map[string]int)

	for _, entry := range all {
		if entry.Allowed {
			allowedCount++
		} else {
			blockedCount++
		}
		if entry.Callsite != "" {
			callsites[entry.Callsite]++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"policy":          "zero-trust-unified",
		"status":          "PASS",
		"total_decisions": len(all),
		"allowed_count":   allowedCount,
		"blocked_count":   blockedCount,
		"callsites":       callsites,
		"enforcements": []string{
			"dns_pinning_peer_verification",
			"private_ip_metadata_blocking",
			"dns_rebinding_mixed_record_rejection",
			"credential_isolation_on_redirect",
			"https_downgrade_prevention",
			"response_payload_bounding",
		},
	})
}
