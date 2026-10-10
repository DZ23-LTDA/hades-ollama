package agent

// Adaptador de ferramenta para a ponte com as CLIs oficiais de agentes de
// código. O processo só é disparado quando a missão recebeu o escopo
// harness:cli: o runtime confere o descritor antes de executar
// (runtime.go:1801) e devolve ErrCapabilityDenied sem iniciar nada.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// ScopeHarnessCLI autoriza iniciar uma CLI oficial de agente de código. É um
// escopo novo, e não uma reutilização de terminal:allowlisted, para não ampliar
// em silêncio uma permissão já concedida.
const ScopeHarnessCLI = "harness:cli"

type cliHarnessTool struct{}

func (cliHarnessTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:             "harness.cli.exec",
		Version:          "1",
		Description:      "Inicia uma CLI oficial de agente de código (Claude Code, Codex, Gemini ou Copilot) em modo não interativo, sob aprovação explícita, com ambiente sem credenciais, saída limitada e redação por DLP",
		Risk:             RiskWrite,
		Scopes:           []string{ScopeHarnessCLI},
		RequiresApproval: true,
	}
}

func (cliHarnessTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	harness := strings.ToLower(strings.TrimSpace(stringInput(input, "harness", "")))
	spec, ok := lookupGovernedCLI(harness)
	if !ok {
		return ToolResult{}, fmt.Errorf("%w: %q (suportados: %s)", ErrUnknownGovernedCLI, harness, strings.Join(governedCLIHarnessNames(), ", "))
	}
	prompt := stringInput(input, "prompt", "")
	if strings.TrimSpace(prompt) == "" {
		return ToolResult{}, fmt.Errorf("%w: informe o campo prompt", ErrGovernedCLILimit)
	}
	workdir, err := resolveGovernedCLIWorkdir(toolContext.Workspace, stringInput(input, "cwd", "."))
	if err != nil {
		return ToolResult{}, err
	}

	timeout := time.Duration(intInput(input, "timeout_seconds", int(defaultGovernedCLITimeout/time.Second))) * time.Second
	result, err := runGovernedCLI(ctx, GovernedCLIRequest{
		Harness: spec.Harness,
		Prompt:  prompt,
		Model:   stringInput(input, "model", ""),
		Workdir: workdir,
		Timeout: timeout,
	})
	if err != nil {
		return ToolResult{}, err
	}

	return ToolResult{Value: map[string]any{
		"harness":             result.Harness,
		"display_name":        result.DisplayName,
		"binary_path":         result.BinaryPath,
		"command":             result.CommandLine,
		"exit_code":           result.ExitCode,
		"stdout":              result.Stdout,
		"stderr":              result.Stderr,
		"stdout_truncated":    result.StdoutTruncated,
		"stderr_truncated":    result.StderrTruncated,
		"timed_out":           result.TimedOut,
		"duration_ms":         result.DurationMS,
		"dlp_findings":        result.DLPFindings,
		"args_verified":       spec.Verified,
		"environment_policy":  result.Environment,
		"execution_isolation": result.Isolation,
		"resource_limits":     result.ResourceLimits,
	}}, nil
}

func governedCLIHarnessNames() []string {
	names := make([]string, 0, len(governedCLISpecs))
	for _, spec := range governedCLISpecs {
		names = append(names, spec.Harness)
	}
	return names
}

// resolveGovernedCLIWorkdir mantém o diretório de trabalho dentro da raiz do
// workspace da missão.
func resolveGovernedCLIWorkdir(workspace, relative string) (string, error) {
	requested := strings.TrimSpace(relative)
	if requested == "" {
		requested = "."
	}
	workdir, err := safeWorkspacePath(workspace, requested)
	if err != nil {
		return "", fmt.Errorf("diretorio de trabalho invalido: %w", err)
	}
	info, err := os.Stat(workdir)
	if err != nil {
		return "", fmt.Errorf("diretorio de trabalho %q nao existe", requested)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("diretorio de trabalho %q nao e um diretorio", requested)
	}
	return workdir, nil
}
