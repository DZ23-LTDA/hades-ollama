package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestAgentCatalogRoutesFilterPrivateResourcesByOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	connectors := agent.NewConnectorManager()
	if err := connectors.Register(agent.ConnectorConfig{
		ID:             "private-connector",
		OrganizationID: "org_b",
		Provider:       "private",
		BaseURL:        "https://example.test/private-path",
		Operations:     []agent.ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := connectors.Register(agent.ConnectorConfig{ID: "global-connector", Provider: "global", BaseURL: "https://global.example.test/api", Operations: []agent.ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	mcp := agent.NewMCPManager()
	// The running test binary is a real absolute executable on every OS, so the
	// MCP command validation (absolute path + executable regular file) passes on
	// Windows too, where an extensionless shell script would be rejected.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := mcp.Register(agent.MCPServerConfig{ID: "private-mcp", OrganizationID: "org_b", Command: executable, AllowedMethods: []string{"tools/list"}}); err != nil {
		t.Fatal(err)
	}
	remote := agent.NewRemoteMCPManager()
	if err := remote.Register(agent.RemoteMCPServerConfig{ID: "private-remote", OrganizationID: "org_b", URL: "https://example.test/mcp/path-secret", TokenEnv: "MCP_TOKEN_ENV_SECRET", HeadersEnv: map[string]string{"Authorization": "MCP_HEADER_ENV_SECRET"}, AllowedMethods: []string{"tools/list"}}); err != nil {
		t.Fatal(err)
	}
	if err := remote.Register(agent.RemoteMCPServerConfig{ID: "global-remote", URL: "https://global.example.test/mcp", AllowedMethods: []string{"tools/list"}}); err != nil {
		t.Fatal(err)
	}
	deployments := agent.NewDeploymentManager()
	for _, config := range []agent.DeployConfig{
		{ID: "private-deployment", OrganizationID: "org_b", Provider: "generic", BaseURL: "https://deploy.example.test/tenant", TokenEnv: "TENANT_DEPLOY_TOKEN", ProjectID: "tenant-project", AccountID: "tenant-account"},
		{ID: "local-deployment", OrganizationID: agent.LocalOrganizationID, Provider: "generic", BaseURL: "https://deploy.example.test/local"},
	} {
		if err := deployments.Register(config); err != nil {
			t.Fatal(err)
		}
	}
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Connectors: connectors, MCP: mcp, RemoteMCP: remote, Deployments: deployments, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "private.json"), []byte(`{"id":"private-skill","version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := contextStore.LoadSkillsForOrganization(skillDir, "org_b"); err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore}

	assertCatalog := func(t *testing.T, organizationID string, wantPrivate bool) {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Set("agent.organization", agent.Organization{ID: organizationID})
		api.connectors(ctx)
		var connectorPayload struct {
			Connectors []agent.ConnectorConfig `json:"connectors"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &connectorPayload); err != nil {
			t.Fatal(err)
		}
		if (len(connectorPayload.Connectors) == 1) != wantPrivate {
			t.Fatalf("connectors for %s = %+v", organizationID, connectorPayload.Connectors)
		}
		if wantPrivate {
			connector := connectorPayload.Connectors[0]
			if connector.BaseURL != "https://example.test" || connector.TokenEnv != "" {
				t.Fatalf("tenant catalog exposed private connector config: %+v", connector)
			}
			for _, secret := range []string{"private-path"} {
				if strings.Contains(recorder.Body.String(), secret) {
					t.Fatalf("connector catalog exposed %q: %s", secret, recorder.Body.String())
				}
			}
		}

		recorder = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(recorder)
		ctx.Set("agent.organization", agent.Organization{ID: organizationID})
		api.mcp(ctx)
		var mcpPayload struct {
			Servers       []agent.MCPServerConfig       `json:"servers"`
			RemoteServers []agent.RemoteMCPServerConfig `json:"remote_servers"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &mcpPayload); err != nil {
			t.Fatal(err)
		}
		if (len(mcpPayload.Servers) == 1) != wantPrivate || (len(mcpPayload.RemoteServers) == 1) != wantPrivate {
			t.Fatalf("MCP catalog for %s = %+v", organizationID, mcpPayload)
		}
		if wantPrivate {
			remoteServer := mcpPayload.RemoteServers[0]
			if remoteServer.URL != "https://example.test" || remoteServer.TokenEnv != "" || len(remoteServer.HeadersEnv) != 0 {
				t.Fatalf("tenant catalog exposed private remote config: %+v", remoteServer)
			}
			for _, secret := range []string{"path-secret", "MCP_TOKEN_ENV_SECRET", "MCP_HEADER_ENV_SECRET"} {
				if strings.Contains(recorder.Body.String(), secret) {
					t.Fatalf("tenant catalog exposed %q: %s", secret, recorder.Body.String())
				}
			}
		}

		recorder = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(recorder)
		ctx.Set("agent.organization", agent.Organization{ID: organizationID})
		api.skills(ctx)
		var skillPayload struct {
			Skills []agent.SkillManifest `json:"skills"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &skillPayload); err != nil {
			t.Fatal(err)
		}
		if (len(skillPayload.Skills) == 1) != wantPrivate {
			t.Fatalf("skills for %s = %+v", organizationID, skillPayload.Skills)
		}
	}

	assertCatalog(t, "org_a", false)
	assertCatalog(t, "org_b", true)

	// The unauthenticated local API is not a global-tenant bypass. It may expose
	// ownerless local plugins, but never organization-owned resources.
	api.authRequired = false
	for name, handler := range map[string]func(*gin.Context){"connectors": api.connectors, "mcp": api.mcp} {
		t.Run("unauthenticated local "+name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			handler(ctx)
			for _, secret := range []string{"private-connector", "private-mcp", "private-remote", "MCP_TOKEN_ENV_SECRET", "MCP_HEADER_ENV_SECRET", "path-secret"} {
				if strings.Contains(recorder.Body.String(), secret) {
					t.Fatalf("local %s catalog exposed tenant resource %q: %s", name, secret, recorder.Body.String())
				}
			}
		})
	}
	t.Run("unauthenticated local deployments", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		api.deployments(ctx)
		var payload struct {
			Providers []agent.DeployConfig `json:"providers"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Providers) != 1 || payload.Providers[0].ID != "local-deployment" {
			t.Fatalf("unauthenticated deployment catalog = %+v", payload.Providers)
		}
		for _, secret := range []string{"private-deployment", "TENANT_DEPLOY_TOKEN", "tenant-project", "tenant-account"} {
			if strings.Contains(recorder.Body.String(), secret) {
				t.Fatalf("unauthenticated deployment catalog exposed %q: %s", secret, recorder.Body.String())
			}
		}
	})
}
