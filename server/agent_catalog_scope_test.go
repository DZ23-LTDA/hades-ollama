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
	executable := filepath.Join(t.TempDir(), "mcp-server")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
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
	workspace := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Connectors: connectors, MCP: mcp, RemoteMCP: remote, WorkspaceRoot: workspace})
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
}
