package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestSupervisorAdminRejectsNonAdminAndCrossOrg locks in SEC-09: the global
// supervisor is a deployment-wide component, so reconfiguring it or ticking it
// must reject ordinary members, and with OLLAMA_SUPERVISOR_ORG set an admin of
// any other organization must not be able to drive it.
func TestSupervisorAdminRejectsNonAdminAndCrossOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: true}

	for _, role := range []agent.Role{agent.RoleViewer, agent.RoleOperator} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("agent.membership", agent.Membership{Role: role, OrganizationID: "org-a"})
		if api.requireSupervisorAdmin(ctx) {
			t.Fatalf("role %s must not pass supervisor admin gate", role)
		}
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	if !api.requireSupervisorAdmin(ctx) {
		t.Fatal("org admin must pass when no org pin is configured")
	}

	t.Setenv("OLLAMA_SUPERVISOR_ORG", "org-owner")

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	if api.requireSupervisorAdmin(ctx) {
		t.Fatal("admin of a non-pinned org must be rejected (SEC-09 cross-tenant)")
	}

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleOwner, OrganizationID: "org-owner"})
	if !api.requireSupervisorAdmin(ctx) {
		t.Fatal("owner of the pinned org must pass")
	}
}
