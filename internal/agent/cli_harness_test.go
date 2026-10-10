package agent

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// governedCLIHelperRunFlag marca o processo auxiliar que se comporta como uma
// CLI falsa. O binário de teste é reexecutado com este -test.run exato.
const governedCLIHelperRunFlag = "-test.run=^TestGovernedCLIHelperProcess$"

// TestGovernedCLIHelperProcess é o processo auxiliar usado como CLI falsa pelos
// testes da ponte: ele só age quando reexecutado com o -test.run exato acima e,
// na suíte normal, apenas se declara ignorado.
func TestGovernedCLIHelperProcess(t *testing.T) {
	args := helperCLIArgs()
	if args == nil {
		t.Skip("processo auxiliar: reexecutado como CLI falsa pelos testes da ponte")
	}
	switch {
	case helperCLIHasArg(args, "--emit-big"):
		_, _ = os.Stdout.WriteString(strings.Repeat("A", 200_000))
		helperCLIExitOK()
	case helperCLIHasArg(args, "--emit-secret"):
		_, _ = os.Stdout.WriteString(`{"api_key": "sk-live-nao-pode-vazar"}` + " https://user:sk-nao-pode-vazar@example.com/repo.git\n")
		helperCLIExitOK()
	case helperCLIHasArg(args, "--sleep"):
		time.Sleep(30 * time.Second)
	case helperCLIHasArg(args, "--exit"):
		os.Exit(7)
	case helperCLIHasArg(args, "--env-probe"):
		_, _ = fmt.Fprintf(os.Stdout, "probe_segredos=%d\nprobe_path_len=%d\n", helperCLISecretCount(), len(os.Getenv("PATH")))
		helperCLIExitOK()
	default:
		_, _ = fmt.Fprintf(os.Stdout, "HARNESS_OK|%s", args[len(args)-1])
		helperCLIExitOK()
	}
}

// helperCLIExitOK encerra o processo auxiliar com sucesso sem passar pelo
// encerramento do framework de testes, que acrescentaria um "PASS" à saída e
// tornaria a captura acoplada a um detalhe interno do Go.
func helperCLIExitOK() {
	os.Exit(0)
}

// helperCLISecretCount conta quantas variáveis de credencial chegaram ao
// processo auxiliar sem imprimir nomes que a própria DLP redigiria.
func helperCLISecretCount() int {
	present := 0
	for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "CLAUDE_SESSION_KEY"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			present++
		}
	}
	return present
}

func helperCLIArgs() []string {
	requested := false
	for _, arg := range os.Args {
		if arg == governedCLIHelperRunFlag {
			requested = true
			break
		}
	}
	if !requested {
		return nil
	}
	args := os.Args[1:]
	for index, arg := range args {
		if arg == "--" {
			return args[index+1:]
		}
	}
	return args
}

func helperCLIHasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

// governedCLIFakeCLIRequest monta uma execução cujo binário é o próprio binário
// de teste, com o prompt renderizado como último argumento.
func governedCLIFakeCLIRequest(t *testing.T, extra string) GovernedCLIRequest {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	args := []string{governedCLIHelperRunFlag, "--"}
	if extra != "" {
		args = append(args, extra)
	}
	args = append(args, CLIPromptPlaceholder)
	return GovernedCLIRequest{
		Harness:      "claude",
		Prompt:       "diagnostique o repositorio",
		Workdir:      t.TempDir(),
		BinaryPath:   binary,
		ArgsOverride: args,
	}
}

func TestGovernedCLIRunsFakeCLIEndToEnd(t *testing.T) {
	result, err := runGovernedCLI(t.Context(), governedCLIFakeCLIRequest(t, ""))
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code=%d, want 0", result.ExitCode)
	}
	if result.Stdout != "HARNESS_OK|diagnostique o repositorio" {
		t.Fatalf("stdout=%q", result.Stdout)
	}
	if result.Harness != "claude" || result.DisplayName != "Claude Code" {
		t.Fatalf("harness=%q display=%q", result.Harness, result.DisplayName)
	}
	if result.BinaryPath == "" || result.CommandLine == "" {
		t.Fatalf("binary_path=%q command=%q", result.BinaryPath, result.CommandLine)
	}
	if strings.Contains(result.CommandLine, CLIPromptPlaceholder) {
		t.Fatalf("command line manteve o marcador: %q", result.CommandLine)
	}
	if result.Environment != governedCLIEnvironmentPolicy || result.Isolation != governedCLIIsolation {
		t.Fatalf("environment=%q isolation=%q", result.Environment, result.Isolation)
	}
	if result.DLPFindings != 0 || result.StdoutTruncated || result.TimedOut {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestGovernedCLIEnvironmentDropsCredentials(t *testing.T) {
	const anthropic = "sk-ant-nao-pode-vazar"
	const openai = "sk-openai-nao-pode-vazar"
	const session = "sessao-nao-pode-vazar"
	t.Setenv("ANTHROPIC_API_KEY", anthropic)
	t.Setenv("OPENAI_API_KEY", openai)
	t.Setenv("CLAUDE_SESSION_KEY", session)
	if os.Getenv("ANTHROPIC_API_KEY") != anthropic {
		t.Fatal("ambiente do teste não recebeu a chave de controle")
	}
	result, err := runGovernedCLI(t.Context(), governedCLIFakeCLIRequest(t, "--env-probe"))
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	for _, want := range []string{
		"probe_segredos=0",
		"probe_path_len=",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("stdout=%q sem %q", result.Stdout, want)
		}
	}
	if strings.Contains(result.Stdout, "probe_path_len=0") {
		t.Fatalf("PATH não foi repassado ao processo filho: %q", result.Stdout)
	}
	for _, secret := range []string{anthropic, openai, session} {
		if strings.Contains(result.Stdout, secret) || strings.Contains(result.CommandLine, secret) {
			t.Fatal("segredo do ambiente apareceu no resultado")
		}
	}
}

func TestGovernedCLIRedactsSecretOutput(t *testing.T) {
	result, err := runGovernedCLI(t.Context(), governedCLIFakeCLIRequest(t, "--emit-secret"))
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	if strings.Contains(result.Stdout, "nao-pode-vazar") {
		t.Fatalf("saida do CLI nao foi redigida: %q", result.Stdout)
	}
	if result.DLPFindings == 0 {
		t.Fatalf("dlp_findings=0 para uma saida com credenciais: %q", result.Stdout)
	}
}

func TestGovernedCLITruncatesOutput(t *testing.T) {
	request := governedCLIFakeCLIRequest(t, "--emit-big")
	request.MaxOutput = 4096
	result, err := runGovernedCLI(t.Context(), request)
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	if !result.StdoutTruncated {
		t.Fatal("saida acima do limite não foi sinalizada como truncada")
	}
	if len(result.Stdout) > 4096 {
		t.Fatalf("stdout com %d bytes acima do limite", len(result.Stdout))
	}
}

func TestGovernedCLIKillsProcessOnTimeout(t *testing.T) {
	request := governedCLIFakeCLIRequest(t, "--sleep")
	request.Timeout = time.Second
	startedAt := time.Now()
	result, err := runGovernedCLI(t.Context(), request)
	elapsed := time.Since(startedAt)
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	if !result.TimedOut {
		t.Fatal("timeout não foi sinalizado")
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit code=%d, want -1", result.ExitCode)
	}
	if elapsed > 15*time.Second {
		t.Fatalf("timeout de 1s levou %s", elapsed)
	}
}

func TestGovernedCLIReturnsExitCode(t *testing.T) {
	result, err := runGovernedCLI(t.Context(), governedCLIFakeCLIRequest(t, "--exit"))
	if err != nil {
		t.Fatalf("runGovernedCLI: %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code=%d, want 7", result.ExitCode)
	}
	if result.TimedOut {
		t.Fatal("saida com código 7 não é timeout")
	}
}

func TestGovernedCLIBlocksPromptWithCredentials(t *testing.T) {
	const secret = `{"api_key": "sk-live-nao-pode-vazar"}`
	request := governedCLIFakeCLIRequest(t, "")
	request.Prompt = secret
	_, err := runGovernedCLI(t.Context(), request)
	if !errors.Is(err, ErrGovernedCLIPromptBlocked) {
		t.Fatalf("err=%v, want ErrGovernedCLIPromptBlocked", err)
	}
	if strings.Contains(err.Error(), "sk-live") {
		t.Fatal("a mensagem de erro expôs o segredo")
	}
}

func TestGovernedCLIResolverRejectsPowerShellShims(t *testing.T) {
	directory := t.TempDir()
	shim := filepath.Join(directory, "claude.ps1")
	if err := os.WriteFile(shim, []byte("Write-Host oi"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	resolver := governedCLIResolver{
		lookPath: func(string) (string, error) { return "", exec.ErrNotFound },
		stat:     os.Stat,
		getenv:   func(string) string { return directory },
		goos:     runtime.GOOS,
	}
	if _, err := resolver.resolve("claude"); !errors.Is(err, ErrGovernedCLIBinaryNotFound) {
		t.Fatalf("shim .ps1 foi aceito como CLI: err=%v", err)
	}
	if _, err := resolver.validate(shim); !errors.Is(err, ErrGovernedCLIBinaryUnsupported) {
		t.Fatalf("validate(shim)=%v, want ErrGovernedCLIBinaryUnsupported", err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if _, err := resolver.validate(binary); err != nil {
		t.Fatalf("binário de teste real foi rejeitado: %v", err)
	}

	command := filepath.Join(directory, "claude.cmd")
	if err := os.WriteFile(command, []byte("@echo off\r\necho oi\r\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	resolver.goos = "windows"
	resolved, err := resolver.resolve("claude")
	if err != nil {
		t.Fatalf("resolve(.cmd): %v", err)
	}
	if resolved != command {
		t.Fatalf("resolved=%q, want %q (o shim .ps1 nunca pode ser escolhido)", resolved, command)
	}
	plain := filepath.Join(directory, "codex")
	if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := resolver.validate(plain); !errors.Is(err, ErrGovernedCLIBinaryUnsupported) {
		t.Fatalf("arquivo sem extensão aceito no Windows: %v", err)
	}
}

func TestGovernedCLICatalogRendersModelAndRejectsUnknown(t *testing.T) {
	spec, ok := lookupGovernedCLI("Claude")
	if !ok || spec.Harness != "claude" {
		t.Fatalf("lookupGovernedCLI(Claude)=%+v ok=%v", spec, ok)
	}
	rendered := renderGovernedCLIArgs(spec.Args, "resuma o repositorio", "claude-sonnet-4")
	want := []string{"-p", "resuma o repositorio", "--model", "claude-sonnet-4"}
	if len(rendered) != len(want) {
		t.Fatalf("rendered=%v, want %v", rendered, want)
	}
	for index, value := range want {
		if rendered[index] != value {
			t.Fatalf("rendered=%v, want %v", rendered, want)
		}
	}
	if _, ok := lookupGovernedCLI("manus"); ok {
		t.Fatal("harness desconhecido foi aceito")
	}
	if err := validateGovernedCLIArguments(nil); !errors.Is(err, ErrGovernedCLILimit) {
		t.Fatalf("linha de comando vazia: err=%v", err)
	}
	if err := validateGovernedCLIArguments(make([]string, maxGovernedCLIArgCount+1)); !errors.Is(err, ErrGovernedCLILimit) {
		t.Fatalf("excesso de itens: err=%v", err)
	}
	oversized := []string{strings.Repeat("y", maxGovernedCLIArgumentBytes+1)}
	if err := validateGovernedCLIArguments(oversized); !errors.Is(err, ErrGovernedCLILimit) {
		t.Fatalf("argumento gigante: err=%v", err)
	}
	longPrompt := GovernedCLIRequest{Harness: "claude", Prompt: strings.Repeat("x", maxGovernedCLIPromptBytes+1)}
	if _, err := runGovernedCLI(t.Context(), longPrompt); !errors.Is(err, ErrGovernedCLILimit) {
		t.Fatalf("prompt gigante: err=%v", err)
	}
}

func TestGovernedCLIToolIsDenyByDefault(t *testing.T) {
	tool, ok := NewRegistry().Get("harness.cli.exec")
	if !ok {
		t.Fatal("harness.cli.exec não está registrado em NewRegistry")
	}
	descriptor := tool.Descriptor()
	if descriptor.Risk != RiskWrite || !descriptor.RequiresApproval {
		t.Fatalf("descritor sem risco de escrita e aprovação: %+v", descriptor)
	}
	policy := DefaultCapabilityPolicy()
	if policy.Allows(descriptor, nil) {
		t.Fatal("ferramenta autorizada sem nenhuma concessão")
	}
	if policy.Allows(descriptor, []string{"workspace:write", "terminal:allowlisted", "sandbox:execute"}) {
		t.Fatal("escopos não relacionados autorizaram a CLI")
	}
	if !policy.Allows(descriptor, []string{ScopeHarnessCLI}) {
		t.Fatal("harness:cli deveria autorizar a ferramenta")
	}
}

func TestGovernedCLIToolRejectsBadInputs(t *testing.T) {
	workspace := t.TempDir()
	ctx := t.Context()
	cases := []struct {
		name  string
		input map[string]any
		want  error
	}{
		{name: "harness desconhecido", input: map[string]any{"harness": "manus", "prompt": "oi"}, want: ErrUnknownGovernedCLI},
		{name: "prompt vazio", input: map[string]any{"harness": "claude"}, want: ErrGovernedCLILimit},
		{name: "prompt com credencial", input: map[string]any{"harness": "claude", "prompt": `{"api_key": "sk-live-nao-pode-vazar"}`}, want: ErrGovernedCLIPromptBlocked},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := (cliHarnessTool{}).Execute(ctx, ToolContext{Workspace: workspace}, testCase.input)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("err=%v, want %v", err, testCase.want)
			}
		})
	}
	for _, escape := range []string{"../fora", "nao-existe"} {
		_, err := (cliHarnessTool{}).Execute(ctx, ToolContext{Workspace: workspace}, map[string]any{"harness": "claude", "prompt": "oi", "cwd": escape})
		if err == nil {
			t.Fatalf("cwd %q deveria falhar", escape)
		}
	}
}
