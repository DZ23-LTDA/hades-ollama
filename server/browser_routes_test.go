package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// A non-local organization must never reach the browser operator environment
// endpoints: the operator is gated to the single-user local org.
func TestBrowserEnvironmentRejectsNonLocalOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name    string
		handler func(*agentAPI, *gin.Context)
	}{
		{"status", (*agentAPI).browserEnvironment},
		{"setup", (*agentAPI).browserEnvironmentSetup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Set("agent.organization", agent.Organization{ID: "org_tenant_xyz"})

			tc.handler(&agentAPI{}, c)

			if recorder.Code != 403 {
				t.Fatalf("status = %d, want 403 (body=%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}
