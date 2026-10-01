package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunAgentRequestUsesEnvironmentBearerToken(t *testing.T) {
	const token = "cli-test-token-not-a-real-secret"
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("OLLAMA_HOST", server.URL)
	t.Setenv("OLLAMA_AGENT_TOKEN", token)
	if err := runAgentRequest(context.Background(), http.MethodGet, "/api/agent/v1/tools", nil); err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer "+token {
		t.Fatalf("Authorization header = %q, want bearer token", gotAuthorization)
	}
}

func TestRunAgentRequestOmitsBearerTokenWhenUnset(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("OLLAMA_HOST", server.URL)
	t.Setenv("OLLAMA_AGENT_TOKEN", " \t ")
	if err := runAgentRequest(context.Background(), http.MethodGet, "/api/agent/v1/tools", nil); err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "" {
		t.Fatalf("unexpected Authorization header: %q", gotAuthorization)
	}
}

func runAgentMigrationCommand() error {
	command := agentCommand()
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs([]string{"migrate-postgres"})
	return command.Execute()
}

func TestAgentMigrationCommandFailsClosedOnMissingKey(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", "")
	err := runAgentMigrationCommand()
	if err == nil || !strings.Contains(err.Error(), "OLLAMA_AGENT_TENANT_CONTEXT_KEY") {
		t.Fatalf("migration command error=%v, want missing tenant key validation", err)
	}
}

func TestAgentMigrationCommandFailsClosedOnMissingMigratorDSN(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", strings.Repeat("ab", 32))
	err := runAgentMigrationCommand()
	if err == nil || !strings.Contains(err.Error(), "postgres migrator DSN is required") {
		t.Fatalf("migration command error=%v, want missing migrator DSN validation", err)
	}
}

func TestAgentMigrationCommandIsRegistered(t *testing.T) {
	command, _, err := agentCommand().Find([]string{"migrate-postgres"})
	if err != nil || command == nil || command.Use != "migrate-postgres" {
		t.Fatalf("migrate-postgres command lookup = %v, %v", command, err)
	}
}

func TestAgentMigrationCommandRejectsInvalidConfiguration(t *testing.T) {
	t.Run("short HMAC key", func(t *testing.T) {
		t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "postgres://ollama_agent_migrator@example.invalid/ollama_agent")
		t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", strings.Repeat("ab", 31))
		if err := runAgentMigrationCommand(); err == nil || !strings.Contains(err.Error(), "at least 64 hexadecimal characters") {
			t.Fatalf("migration command error=%v, want short-key refusal", err)
		}
	})
	t.Run("invalid hexadecimal HMAC key", func(t *testing.T) {
		t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "postgres://ollama_agent_migrator@example.invalid/ollama_agent")
		t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", strings.Repeat("zz", 32))
		if err := runAgentMigrationCommand(); err == nil || !strings.Contains(err.Error(), "at least 64 hexadecimal characters") {
			t.Fatalf("migration command error=%v, want invalid-key refusal", err)
		}
	})
	t.Run("invalid DSN", func(t *testing.T) {
		t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "not-a-postgres-dsn")
		t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", strings.Repeat("ab", 32))
		if err := runAgentMigrationCommand(); err == nil {
			t.Fatal("migration command unexpectedly accepted an invalid DSN")
		}
	})
}

func runAgentKeyRotationCommand() error {
	command := agentCommand()
	command.SilenceErrors = true
	command.SilenceUsage = true
	command.SetArgs([]string{"rotate-postgres-key"})
	return command.Execute()
}

func TestAgentKeyRotationCommandIsRegistered(t *testing.T) {
	command, _, err := agentCommand().Find([]string{"rotate-postgres-key"})
	if err != nil || command == nil || command.Use != "rotate-postgres-key" {
		t.Fatalf("rotate-postgres-key command lookup = %v, %v", command, err)
	}
}

func TestAgentKeyRotationCommandFailsClosedOnInvalidConfiguration(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT", "")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION", "")
	if err := runAgentKeyRotationCommand(); err == nil || !strings.Contains(err.Error(), "OLLAMA_AGENT_TENANT_CONTEXT_KEY") {
		t.Fatalf("rotation command error=%v, want missing current key refusal", err)
	}
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY", strings.Repeat("ab", 32))
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION", "1")
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT", strings.Repeat("cd", 32))
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION", strconv.Itoa(1))
	t.Setenv("OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL", "postgres://admin@127.0.0.1/ollama_agent")
	t.Setenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL", "postgres://migrator@127.0.0.1/ollama_agent")
	if err := runAgentKeyRotationCommand(); err == nil || !strings.Contains(err.Error(), "strictly greater") {
		t.Fatalf("rotation command error=%v, want non-monotonic version refusal", err)
	}
	t.Setenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION", "2")
	t.Setenv("OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL", "")
	if err := runAgentKeyRotationCommand(); err == nil || !strings.Contains(err.Error(), "postgres admin and migrator DSNs are required") {
		t.Fatalf("rotation command error=%v, want missing admin DSN refusal", err)
	}
}

func TestPostgresKeyRotationGuideMatchesImplementedCommand(t *testing.T) {
	guide, err := os.ReadFile(filepath.Join("..", "docs", "CLASS_A_PLUS_GUIDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := strings.ToLower(string(guide))
	for _, required := range []string{"ollama agent rotate-postgres-key", "ollama_agent_tenant_context_key_version", "versão monotônica", "versão superior"} {
		if !strings.Contains(content, required) {
			t.Errorf("operator guide is missing required rotation guidance %q", required)
		}
	}
	for _, stale := range []string{"rotação da chave hmac não é suportada", "rotação não é suportada hoje", "não há comando nem procedimento validado"} {
		if strings.Contains(content, stale) {
			t.Errorf("operator guide still contains obsolete rotation statement %q", stale)
		}
	}
}
