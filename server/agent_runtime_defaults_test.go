package server

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

func TestNewDefaultAgentRuntimeUsesDurableDefaults(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	for _, name := range []string{
		"OLLAMA_AGENT_ROOT",
		"OLLAMA_AGENT_STORE",
		"OLLAMA_AGENT_AUTH_STORE",
		"OLLAMA_AGENT_DATABASE_URL",
		"OLLAMA_AGENT_TENANT_CONTEXT_KEY",
		"OLLAMA_AGENT_REDIS_URL",
		"OLLAMA_AGENT_CONNECTORS",
		"OLLAMA_AGENT_MCP",
		"OLLAMA_AGENT_REMOTE_MCP",
		"OLLAMA_AGENT_DEPLOYMENTS",
		"OLLAMA_AGENT_MEDIA_BASE_URL",
		"OLLAMA_AGENT_MEDIA_API_KEY",
		"OLLAMA_AGENT_OTLP_ENDPOINT",
	} {
		t.Setenv(name, "")
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	runtime, err := newDefaultAgentRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(runtime.WorkspaceRoot(), filepath.Clean(configDir)+string(os.PathSeparator)) {
		t.Fatalf("workspace default escaped XDG config directory: %q", runtime.WorkspaceRoot())
	}
	if filepath.Base(runtime.WorkspaceRoot()) != "workspaces" {
		t.Fatalf("workspace default = %q, want workspaces directory", runtime.WorkspaceRoot())
	}
	if filepath.Base(runtime.DataRoot()) != "data" {
		t.Fatalf("data default = %q, want data directory", runtime.DataRoot())
	}
	if _, err := newAgentAPI(runtime); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(runtime.DataRoot(), "auth")); err != nil {
		t.Fatalf("durable auth store was not created: %v", err)
	}
}

func TestNewDefaultAgentRuntimeRequiresPostgresTenantKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", "postgres://invalid.invalid/unused")
	t.Setenv("OLLAMA_AGENT_REDIS_URL", "redis://invalid.invalid/unused")
	if _, err := newDefaultAgentRuntime(); err == nil || !strings.Contains(err.Error(), "OLLAMA_AGENT_TENANT_CONTEXT_KEY") {
		t.Fatalf("PostgreSQL startup error=%v, want missing tenant-key error", err)
	}
}

func TestPostgresRuntimeFailsClosedEvenWhenAuthIsEnabled(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_AUTH_REQUIRED", "true")
	if _, err := agent.NewRuntime(agent.RuntimeConfig{Store: &agent.PostgresStore{}, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()}); !errors.Is(err, agent.ErrPostgresTenantIsolationUnavailable) {
		t.Fatalf("PostgreSQL runtime construction error=%v, want tenant-isolation fail-closed error", err)
	}
}

func TestGenerateRoutesFailurePreservesPreexistingRuntime(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_REDIS_URL", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", "")
	t.Setenv("OLLAMA_AGENT_OTLP_ENDPOINT", "")
	invalidConfig := filepath.Join(t.TempDir(), "missing-config.json")
	t.Setenv("OLLAMA_DZ23_CONFIG", invalidConfig)
	runtime, err := newDefaultAgentRuntime()
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{agentRuntime: runtime, addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 11434}}
	if _, err := server.GenerateRoutes(); err == nil {
		_ = runtime.Close(context.Background())
		t.Fatal("GenerateRoutes unexpectedly accepted a missing provider configuration")
	}
	if server.agentRuntime != runtime {
		_ = runtime.Close(context.Background())
		t.Fatal("failed route regeneration cleared a Runtime owned by the existing Server")
	}
	if err := runtime.Close(context.Background()); err != nil {
		t.Fatalf("close retained Runtime: %v", err)
	}
}
