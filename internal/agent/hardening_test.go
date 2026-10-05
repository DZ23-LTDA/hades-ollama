package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectTestRunRequiresApproval(t *testing.T) {
	descriptor := projectTestRunnerTool{}.Descriptor()
	if !descriptor.RequiresApproval {
		t.Fatal("project.test.run executa processo do workspace: precisa de approval")
	}
	if descriptor.Risk == RiskRead {
		t.Fatalf("project.test.run não é leitura, risco declarado = %q", descriptor.Risk)
	}
}

func TestRunProjectTestsRejectsSubPathEscape(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "go.mod"), []byte("module outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	relative, err := filepath.Rel(workspace, outside)
	if err != nil {
		t.Fatal(err)
	}

	for _, subPath := range []string{"../", relative, filepath.Join("..", "..")} {
		if _, err := RunProjectTests(context.Background(), workspace, TestRunOptions{SubPath: subPath}); err == nil {
			t.Fatalf("sub_path %q escapa do workspace e deveria ser recusado", subPath)
		}
	}
}

func TestNormalizeStepsIgnoresPlannerSuppliedRisk(t *testing.T) {
	steps, err := normalizeSteps([]Step{
		{Kind: "project.test.run", Risk: RiskRead},
		{Kind: "workspace.write", Risk: RiskRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.Risk == RiskRead {
			t.Fatalf("passo %q manteve o risco rebaixado pelo planner", step.Kind)
		}
		if !step.RequiresApproval {
			t.Fatalf("passo %q deveria exigir approval", step.Kind)
		}
	}
}

func TestRejectGenericInterpreterCommand(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST", "")
	for _, command := range []string{"/bin/sh", "/bin/bash", "/usr/bin/env", "/usr/bin/perl", "/usr/bin/sudo"} {
		if err := rejectGenericInterpreterCommand(command); !errors.Is(err, ErrMCPInterpreterCommand) {
			t.Fatalf("comando %q deveria ser recusado como interpretador genérico, erro=%v", command, err)
		}
	}
	if err := rejectGenericInterpreterCommand("/usr/local/bin/my-mcp-server"); err != nil {
		t.Fatalf("binário de servidor MCP não deveria ser recusado: %v", err)
	}

	t.Setenv("OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST", "/bin/sh")
	if err := rejectGenericInterpreterCommand("/bin/sh"); err != nil {
		t.Fatalf("caminho liberado explicitamente pelo operador deveria passar: %v", err)
	}
	if err := rejectGenericInterpreterCommand("/bin/bash"); !errors.Is(err, ErrMCPInterpreterCommand) {
		t.Fatal("allowlist é por caminho exato: /bin/bash não foi liberado")
	}
}

func TestMinimalChildEnvDropsSecrets(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "valor-secreto")
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "valor-secreto")
	t.Setenv("OLLAMA_AGENT_POSTGRES_RUNTIME_URL", "postgres://user:senha@host/db")
	t.Setenv("PATH", "/usr/bin:/bin")

	env := minimalChildEnv("OLLAMA_AGENT_BROWSER_ROOT=/tmp/workspace/.browser")

	for _, entry := range env {
		if strings.Contains(entry, "valor-secreto") || strings.Contains(entry, "senha@host") {
			t.Fatalf("ambiente do processo filho carrega credencial: %q", strings.SplitN(entry, "=", 2)[0])
		}
	}

	var sawPath, sawExtra bool
	for _, entry := range env {
		switch {
		case strings.HasPrefix(entry, "PATH="):
			sawPath = true
		case entry == "OLLAMA_AGENT_BROWSER_ROOT=/tmp/workspace/.browser":
			sawExtra = true
		}
	}
	if !sawPath {
		t.Fatal("PATH precisa ser repassado para o processo filho funcionar")
	}
	if !sawExtra {
		t.Fatal("a variável explícita do chamador precisa estar no ambiente")
	}
}
