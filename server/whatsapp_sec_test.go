package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestWhatsAppAdminRejectsNonAdminAndCrossOrg locks in SEC-08: the global
// WhatsApp gateway is deployment-wide, so its sensitive handlers must reject
// ordinary members, and when OLLAMA_WHATSAPP_ORG pins an owning organization,
// an admin of any other organization must not be able to administer it.
func TestWhatsAppAdminRejectsNonAdminAndCrossOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: true}

	// Ordinary members are always rejected, regardless of the org pin.
	for _, role := range []agent.Role{agent.RoleViewer, agent.RoleOperator} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("agent.membership", agent.Membership{Role: role, OrganizationID: "org-a"})
		if api.requireWhatsAppAdmin(ctx) {
			t.Fatalf("role %s must not pass WhatsApp admin gate", role)
		}
	}

	// Without a pin, any org owner/admin passes (single-tenant/local default).
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	if !api.requireWhatsAppAdmin(ctx) {
		t.Fatal("org admin must pass when no org pin is configured")
	}

	// With a pin, only owners/admins of the pinned org pass.
	t.Setenv("OLLAMA_WHATSAPP_ORG", "org-owner")

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleAdmin, OrganizationID: "org-a"})
	if api.requireWhatsAppAdmin(ctx) {
		t.Fatal("admin of a non-pinned org must be rejected (SEC-08 cross-tenant)")
	}

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("agent.membership", agent.Membership{Role: agent.RoleOwner, OrganizationID: "org-owner"})
	if !api.requireWhatsAppAdmin(ctx) {
		t.Fatal("owner of the pinned org must pass")
	}
}
