package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeAgentConfig(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAgentMCPBootstrapsStrictManifest(t *testing.T) {
	// A política de MCP recusa link simbólico. Em distribuições onde /bin/echo
	// é link (Ubuntu com coreutils via cargo, por exemplo), o manifesto precisa
	// apontar para o caminho real.
	echoPath, err := exec.LookPath("echo")
	if err != nil {
		t.Skipf("echo não encontrado no PATH: %v", err)
	}
	echoPath, err = filepath.EvalSymlinks(echoPath)
	if err != nil {
		t.Skipf("não foi possível resolver o caminho real de echo: %v", err)
	}
	manifest := fmt.Sprintf(`[{"id":"local","command":%q,"args":["mcp"],"allowed_methods":["tools/list"],"environment_vars":["PATH"],"timeout_seconds":3}]`, echoPath)
	path := writeAgentConfig(t, "mcp.json", manifest)
	t.Setenv("OLLAMA_AGENT_MCP", path)
	manager, err := loadAgentMCP()
	if err != nil {
		t.Fatal(err)
	}
	if manager == nil {
		t.Fatal("expected MCP manager")
	}
	defer manager.StopAll()
	servers := manager.List()
	if len(servers) != 1 || servers[0].ID != "local" || len(servers[0].AllowedMethods) != 1 || servers[0].AllowedMethods[0] != "tools/list" {
		t.Fatalf("unexpected MCP bootstrap: %+v", servers)
	}
}

func TestLoadAgentRemoteMCPBootstrapsStrictManifest(t *testing.T) {
	path := writeAgentConfig(t, "remote-mcp.json", `[{"id":"remote","url":"https://mcp.example.test/rpc","allowed_methods":["tools/list"],"headers_env":{"Authorization":"MCP_TOKEN"}}]`)
	t.Setenv("OLLAMA_AGENT_REMOTE_MCP", path)
	manager, err := loadAgentRemoteMCP()
	if err != nil {
		t.Fatal(err)
	}
	if manager == nil {
		t.Fatal("expected Remote MCP manager")
	}
	servers := manager.List()
	if len(servers) != 1 || servers[0].ID != "remote" || servers[0].URL != "https://mcp.example.test" {
		t.Fatalf("unexpected Remote MCP bootstrap: %+v", servers)
	}
}

func TestLoadAgentMCPRejectsUnknownAndTrailingJSON(t *testing.T) {
	unknown := writeAgentConfig(t, "unknown.json", `[{"id":"local","command":"/bin/echo","allowed_methods":["tools/list"],"unexpected":true}]`)
	t.Setenv("OLLAMA_AGENT_MCP", unknown)
	if _, err := loadAgentMCP(); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}

	trailing := writeAgentConfig(t, "trailing.json", `[{"id":"local","command":"/bin/echo","allowed_methods":["tools/list"]}] {"extra":true}`)
	t.Setenv("OLLAMA_AGENT_MCP", trailing)
	if _, err := loadAgentMCP(); err == nil || !strings.Contains(err.Error(), "trailing JSON") {
		t.Fatalf("expected trailing JSON rejection, got %v", err)
	}
}

func TestLoadAgentRemoteMCPRejectsEmptyAllowlist(t *testing.T) {
	path := writeAgentConfig(t, "remote-invalid.json", `[{"id":"remote","url":"https://mcp.example.test/rpc","allowed_methods":[]}]`)
	t.Setenv("OLLAMA_AGENT_REMOTE_MCP", path)
	if _, err := loadAgentRemoteMCP(); err == nil || !strings.Contains(err.Error(), "allowed_methods") {
		t.Fatalf("expected empty allowlist rejection, got %v", err)
	}
}
