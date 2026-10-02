package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func writeExecutableFixture(path string) error {
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		return err
	}
	return nil
}

func mcpSkillRegistrationContext(t *testing.T, body string, organizationID string, role agent.Role) (*agentAPI, *gin.Context, *httptest.ResponseRecorder, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	command := filepath.Join(root, "mcp-server")
	// Windows determines "executable" by extension (.exe/.com), not a mode bit,
	// so the fixture needs an executable extension there to be accepted (E9).
	if runtime.GOOS == "windows" {
		command += ".exe"
	}
	if err := writeExecutableFixture(command); err != nil {
		t.Fatal(err)
	}
	contextStore, err := agent.NewContextStore(filepath.Join(root, "context"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: t.TempDir(),
		DataRoot:      t.TempDir(),
		Context:       contextStore,
		MCP:           agent.NewMCPManager(),
		RemoteMCP:     agent.NewRemoteMCPManager(),
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/plugins", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	ctx.Set("agent.membership", agent.Membership{Role: role})
	return &agentAPI{runtime: runtime, context: contextStore, authRequired: true}, ctx, recorder, command
}

func TestRegisterMCPRequiresAdminAndBindsOrganization(t *testing.T) {
	api, ctx, recorder, command := mcpSkillRegistrationContext(t, `{}`, "org-a", agent.RoleViewer)
	api.registerMCP(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("member status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	// Recreate the request after the authorization-only assertion.
	body := fmt.Sprintf(`{"id":"stdio-tools","command":%q,"allowed_methods":["tools/list"]}`, command)
	api, ctx, recorder, _ = mcpSkillRegistrationContext(t, body, "org-a", agent.RoleAdmin)
	api.registerMCP(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := api.runtime.MCPServersForOrganization("org-a"); len(got) != 1 || got[0].OrganizationID != "org-a" {
		t.Fatalf("org-a MCP servers=%+v", got)
	}
	if got := api.runtime.MCPServersForOrganization("org-b"); len(got) != 0 {
		t.Fatalf("cross-tenant MCP visible=%+v", got)
	}
}

func TestRegisterRemoteMCPRejectsRawSecretAndCrossTenant(t *testing.T) {
	crossTenant := `{"id":"remote-tools","organization_id":"org-b","url":"https://example.com/mcp","allowed_methods":["tools/list"]}`
	api, ctx, recorder, _ := mcpSkillRegistrationContext(t, crossTenant, "org-a", agent.RoleOwner)
	api.registerRemoteMCP(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	rawSecret := `{"id":"remote-tools","url":"https://example.com/mcp","token":"do-not-accept","allowed_methods":["tools/list"]}`
	api, ctx, recorder, _ = mcpSkillRegistrationContext(t, rawSecret, "org-a", agent.RoleOwner)
	api.registerRemoteMCP(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("raw-secret status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterSkillRejectsCallerTrustAndBindsOrganization(t *testing.T) {
	trusted := `{"id":"unsafe-skill","version":"1.0.0","trusted":true}`
	api, ctx, recorder, _ := mcpSkillRegistrationContext(t, trusted, "org-a", agent.RoleOwner)
	api.registerSkill(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("trusted status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	valid := `{"id":"research-skill","version":"1.0.0","description":"safe manifest","scopes":["research:read"],"tools":["search"]}`
	api, ctx, recorder, _ = mcpSkillRegistrationContext(t, valid, "org-a", agent.RoleOwner)
	api.registerSkill(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("valid status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := api.runtime.SkillsForOrganization("org-a"); len(got) != 1 || got[0].Trusted || !got[0].Enabled {
		t.Fatalf("skill state=%+v", got)
	}
}
