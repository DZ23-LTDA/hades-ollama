package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMCPManagerCallsAllowlistedMethod(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "echo", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	result, err := manager.CallGlobal(context.Background(), "echo", "echo", map[string]any{"value": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result, &payload); err != nil || payload["value"] != "hello" {
		t.Fatalf("result=%s err=%v", result, err)
	}
	if _, err := manager.CallGlobal(context.Background(), "echo", "tools/list", nil); err == nil || !strings.Contains(err.Error(), "not allowlisted") {
		t.Fatalf("unexpected allowlist result: %v", err)
	}
}

func TestMCPManagerCallRejectsCrossOrganizationServer(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "org-b", OrganizationID: "org_b", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	if _, err := manager.CallForOrganization(context.Background(), "org_a", "org-b", "echo", map[string]any{"value": "blocked"}); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-organization error = %v", err)
	}
}

func TestMCPManagerCallRejectsOwnerlessServerForTenant(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "global", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	if _, err := manager.CallForOrganization(context.Background(), "org_a", "global", "echo", map[string]any{"value": "blocked"}); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("tenant call to ownerless MCP server error = %v", err)
	}
	if _, err := manager.Call(context.Background(), "global", "echo", nil); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("implicit unscoped MCP call error = %v", err)
	}
}

func TestMCPNotificationsDoNotBreakResponseCorrelation(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "notify", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"notify"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	result, err := manager.CallGlobal(context.Background(), "notify", "notify", map[string]any{"value": "after-notification"})
	if err != nil || !strings.Contains(string(result), "after-notification") {
		t.Fatalf("notification result=%s err=%v", result, err)
	}
}

func TestMCPRedactsSuccessfulResultAndSuppressesProviderErrors(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "sensitive", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"secret", "error-secret", "missing"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 5}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()

	result, err := manager.CallGlobal(context.Background(), "sensitive", "secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"provider-secret-value", "signature-secret-value"} {
		if strings.Contains(string(result), secret) {
			t.Fatalf("MCP result disclosed provider secret %q", secret)
		}
	}
	if !strings.Contains(string(result), "[REDACTED]") {
		t.Fatalf("MCP result was not visibly redacted: %s", result)
	}

	if _, err := manager.CallGlobal(context.Background(), "sensitive", "error-secret", nil); err == nil || strings.Contains(err.Error(), "provider-error-secret") {
		t.Fatalf("provider error leaked its message: %v", err)
	}
	if _, err := manager.CallGlobal(context.Background(), "sensitive", "missing", nil); err == nil || !strings.Contains(err.Error(), "omitted result") {
		t.Fatalf("missing MCP result was accepted: %v", err)
	}
}

func TestMCPOrganizationCatalogHidesGlobalAndForeignServers(t *testing.T) {
	manager := NewMCPManager()
	for _, config := range []MCPServerConfig{
		{ID: "global", Command: os.Args[0], AllowedMethods: []string{"ping"}, EnvironmentVars: []string{"GLOBAL_SECRET_ENV"}},
		{ID: "org-a", OrganizationID: "org_a", Command: os.Args[0], AllowedMethods: []string{"ping"}, EnvironmentVars: []string{"ORG_A_SECRET_ENV"}},
		{ID: "org-b", OrganizationID: "org_b", Command: os.Args[0], AllowedMethods: []string{"ping"}, EnvironmentVars: []string{"ORG_B_SECRET_ENV"}},
	} {
		if err := manager.Register(config); err != nil {
			t.Fatal(err)
		}
	}
	listed := manager.ListForOrganization("org_a")
	if len(listed) != 1 || listed[0].ID != "org-a" {
		t.Fatalf("org_a MCP catalog = %+v, want only its owned server", listed)
	}
	if len(listed[0].EnvironmentVars) != 0 {
		t.Fatalf("tenant MCP catalog exposed environment mappings: %+v", listed[0].EnvironmentVars)
	}
	if got := manager.ListForOrganization(""); len(got) != 0 {
		t.Fatalf("empty-scope MCP catalog = %+v, want no entries", got)
	}
	manager.StopAll()
}

func TestMCPRejectsLiteralCredentialArgumentsAndHidesArgsFromTenantCatalog(t *testing.T) {
	manager := NewMCPManager()
	for _, args := range [][]string{{"--api-key=literal-secret"}, {"--access-token", "literal-secret"}, {"https://provider.test/path?X-Amz-Signature=literal-secret"}} {
		if err := manager.Register(MCPServerConfig{ID: fmt.Sprintf("unsafe-%d", len(args)), Command: os.Args[0], Args: args, AllowedMethods: []string{"ping"}}); err == nil {
			t.Fatalf("accepted credential-shaped MCP argv: %#v", args)
		}
	}
	if err := manager.RegisterForOrganization("org_a", MCPServerConfig{ID: "safe", Command: os.Args[0], Args: []string{"--mode=read-only"}, AllowedMethods: []string{"ping"}, EnvironmentVars: []string{"PROVIDER_TOKEN"}}); err != nil {
		t.Fatal(err)
	}
	listed := manager.ListForOrganization("org_a")
	if len(listed) != 1 || len(listed[0].Args) != 0 || len(listed[0].EnvironmentVars) != 0 {
		t.Fatalf("tenant MCP catalog disclosed execution configuration: %+v", listed)
	}
	manager.StopAll()
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER_PROCESS") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			ID     int64          `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":0,\"error\":{\"message\":%q}}\n", err.Error())
			continue
		}
		if request.Method == "sleep" {
			time.Sleep(10 * time.Second)
			continue
		}
		if request.Method == "error-secret" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"error\":{\"message\":\"provider-error-secret\"}}\n", request.ID)
			continue
		}
		if request.Method == "missing" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d}\n", request.ID)
			continue
		}
		if request.Method == "secret" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"api_key\":\"provider-secret-value\",\"nested\":{\"providerSignature\":\"signature-secret-value\"}}}\n", request.ID)
			continue
		}
		if request.Method == "notify" {
			fmt.Printf("{\"jsonrpc\":\"2.0\",\"method\":\"progress\",\"params\":{\"status\":\"working\"}}\n")
		}
		value := request.Params["value"]
		fmt.Printf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":{\"value\":%q}}\n", request.ID, value)
	}
	os.Exit(0)
}

func TestMCPRequiresAllowlist(t *testing.T) {
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "empty", Command: os.Args[0]}); err == nil {
		t.Fatal("expected empty MCP allowlist rejection")
	}
}

func TestMCPRequiresAbsoluteExecutableAndUsesPrivateWorkspace(t *testing.T) {
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "relative", Command: "sh", AllowedMethods: []string{"echo"}}); err == nil {
		t.Fatal("expected absolute command rejection")
	}
	link := t.TempDir() + "/mcp-link"
	if err := os.Symlink(os.Args[0], link); err == nil {
		if err := manager.Register(MCPServerConfig{ID: "symlink", Command: link, AllowedMethods: []string{"echo"}}); err == nil {
			t.Fatal("expected symlink executable rejection")
		}
	}
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	if err := manager.Register(MCPServerConfig{ID: "workspace", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}}); err != nil {
		t.Fatal(err)
	}
	config := manager.List()[0]
	if config.WorkingDirectory == "" {
		t.Fatal("expected explicit MCP working directory")
	}
	if _, err := os.Stat(config.WorkingDirectory); err != nil {
		t.Fatalf("working directory missing: %v", err)
	}
	manager.StopAll()
	if _, err := os.Stat(config.WorkingDirectory); !os.IsNotExist(err) {
		t.Fatalf("temporary MCP workspace was not removed: %v", err)
	}
}

func TestMCPPayloadLimitAndCancellationRestart(t *testing.T) {
	t.Setenv("GO_WANT_MCP_HELPER_PROCESS", "1")
	manager := NewMCPManager()
	if err := manager.Register(MCPServerConfig{ID: "limits", Command: os.Args[0], Args: []string{"-test.run=TestMCPHelperProcess"}, AllowedMethods: []string{"echo", "sleep"}, EnvironmentVars: []string{"GO_WANT_MCP_HELPER_PROCESS"}, TimeoutSeconds: 1}); err != nil {
		t.Fatal(err)
	}
	defer manager.StopAll()
	if _, err := manager.CallGlobal(context.Background(), "limits", "echo", map[string]any{"value": strings.Repeat("x", mcpMaxMessageBytes)}); err == nil || !strings.Contains(err.Error(), "payload limit") {
		t.Fatalf("expected request payload limit, got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := manager.CallGlobal(ctx, "limits", "sleep", nil); err == nil {
		t.Fatal("expected cancelled MCP call")
	}
	result, err := manager.CallGlobal(context.Background(), "limits", "echo", map[string]any{"value": "restarted"})
	if err != nil || !strings.Contains(string(result), "restarted") {
		t.Fatalf("restart result=%s err=%v", result, err)
	}
}
