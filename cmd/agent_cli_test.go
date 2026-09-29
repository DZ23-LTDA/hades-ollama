package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
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
