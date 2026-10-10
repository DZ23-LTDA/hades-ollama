package agent

// Ponte de mão dupla com as CLIs oficiais de agentes de código (Claude Code,
// Codex, Gemini e Copilot).
//
// A ponte já existente é reversa: Claude Desktop e Codex consomem os modelos do
// Hades. Este arquivo implementa o sentido inverso, exigido pelo backlog P1-4: o
// Hades inicia a CLI instalada pelo operador, em modo não interativo, sob
// aprovação explícita, com ambiente saneado, saída limitada e redigida por DLP.
//
// Garantias deliberadas:
//   - nenhuma credencial do CLI é lida, copiada ou encaminhada: variáveis com
//     cara de segredo (chaves de API, tokens, sessões) nunca entram no ambiente
//     do processo filho, mesmo que existam no ambiente do Hades;
//   - o prompt e a linha de comando passam por ScanDLP antes do disparo (fail-closed);
//   - toda saída do processo passa por RedactDLP antes de virar resultado;
//   - o processo roda com timeout, sem stdin (não interativo) e com os filhos
//     encerrados em caso de cancelamento;
//   - o diretório de trabalho é validado contra a raiz do workspace.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	// CLIPromptPlaceholder é substituído pelo prompt do usuário.
	CLIPromptPlaceholder = "{prompt}"
	// CLIModelPlaceholder é substituído pelo modelo pedido, quando houver.
	CLIModelPlaceholder = "{model}"

	defaultGovernedCLITimeout    = 120 * time.Second
	maxGovernedCLITimeout        = 900 * time.Second
	defaultGovernedCLIMaxOutput  = 64 << 10
	maxGovernedCLIMaxOutput      = 1 << 20
	maxGovernedCLIPromptBytes    = 32 << 10
	maxGovernedCLIArgCount       = 64
	maxGovernedCLIArgumentBytes  = 32 << 10
	governedCLIIsolation         = "approval-gated-non-interactive-cli"
	governedCLIResourceLimits    = "context-timeout-and-output-bounded"
	governedCLIEnvironmentPolicy = "allowlist-without-credentials"
)

var (
	// ErrUnknownGovernedCLI indica um harness não suportado.
	ErrUnknownGovernedCLI = errors.New("harness de CLI desconhecido")
	// ErrGovernedCLIBinaryNotFound indica que o binário não está no PATH.
	ErrGovernedCLIBinaryNotFound = errors.New("binario da CLI nao encontrado")
	// ErrGovernedCLIBinaryUnsupported indica um alvo que não pode ser executado direto.
	ErrGovernedCLIBinaryUnsupported = errors.New("binario da CLI nao e executavel direto")
	// ErrGovernedCLIPromptBlocked indica que a DLP bloqueou o prompt ou um argumento.
	ErrGovernedCLIPromptBlocked = errors.New("prompt bloqueado pela DLP")
	// ErrGovernedCLILimit indica tamanho acima do limite aceito.
	ErrGovernedCLILimit = errors.New("limite de tamanho excedido para a CLI")
)

// GovernedCLISpec descreve como uma CLI oficial é chamada em modo não
// interativo. Verified=false significa que a linha de comando ainda não foi
// conferido contra uma instalação real, portanto ele pode ser sobrescrito pelo
// operador nas configurações.
type GovernedCLISpec struct {
	Harness     string
	DisplayName string
	Binary      string
	InstallHint string
	Args        []string
	Verified    bool
}

var governedCLISpecs = []GovernedCLISpec{
	{
		Harness:     "claude",
		DisplayName: "Claude Code",
		Binary:      "claude",
		InstallHint: "npm install -g @anthropic-ai/claude-code",
		Args:        []string{"-p", CLIPromptPlaceholder},
	},
	{
		Harness:     "codex",
		DisplayName: "Codex CLI",
		Binary:      "codex",
		InstallHint: "npm install -g @openai/codex",
		Args:        []string{"exec", CLIPromptPlaceholder},
	},
	{
		Harness:     "gemini",
		DisplayName: "Gemini CLI",
		Binary:      "gemini",
		InstallHint: "npm install -g @google/gemini-cli",
		Args:        []string{"-p", CLIPromptPlaceholder},
	},
	{
		Harness:     "copilot",
		DisplayName: "Copilot CLI",
		Binary:      "copilot",
		InstallHint: "npm install -g @github/copilot",
		Args:        []string{"-p", CLIPromptPlaceholder},
	},
}

// GovernedCLICatalog devolve uma cópia do catálogo suportado.
func GovernedCLICatalog() []GovernedCLISpec {
	catalog := make([]GovernedCLISpec, 0, len(governedCLISpecs))
	for _, spec := range governedCLISpecs {
		clone := spec
		clone.Args = append([]string(nil), spec.Args...)
		catalog = append(catalog, clone)
	}
	return catalog
}

func lookupGovernedCLI(harness string) (GovernedCLISpec, bool) {
	normalized := strings.ToLower(strings.TrimSpace(harness))
	for _, spec := range governedCLISpecs {
		if spec.Harness == normalized {
			return spec, true
		}
	}
	return GovernedCLISpec{}, false
}

// renderGovernedCLIArgs aplica prompt e modelo à linha de comando.
func renderGovernedCLIArgs(args []string, prompt, model string) []string {
	rendered := make([]string, 0, len(args)+2)
	for _, arg := range args {
		rendered = append(rendered, strings.ReplaceAll(arg, CLIPromptPlaceholder, prompt))
	}
	if trimmed := strings.TrimSpace(model); trimmed != "" {
		rendered = append(rendered, "--model", trimmed)
	}
	return rendered
}

// governedCLIResolver localiza o binário do CLI de forma determinística.
type governedCLIResolver struct {
	lookPath func(string) (string, error)
	stat     func(string) (os.FileInfo, error)
	getenv   func(string) string
	goos     string
}

func defaultGovernedCLIResolver() governedCLIResolver {
	return governedCLIResolver{lookPath: exec.LookPath, stat: os.Stat, getenv: os.Getenv, goos: runtime.GOOS}
}

func (r governedCLIResolver) pathDirectories() []string {
	raw := strings.Split(r.getenv("PATH"), string(os.PathListSeparator))
	directories := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" || seen[entry] {
			continue
		}
		seen[entry] = true
		directories = append(directories, entry)
	}
	return directories
}

// resolve encontra o executável do CLI. No Windows a busca é própria e prefere
// .exe a .cmd/.bat, porque shims .ps1 dependem da política de execução do
// PowerShell e não podem ser disparados pelo harness.
func (r governedCLIResolver) resolve(binary string) (string, error) {
	name := strings.TrimSpace(binary)
	if name == "" {
		return "", fmt.Errorf("%w: nome vazio", ErrGovernedCLIBinaryNotFound)
	}
	if strings.ContainsAny(name, `/\`) {
		return r.validate(filepath.Clean(name))
	}
	if r.goos == "windows" {
		extensions := []string{".exe", ".cmd", ".bat", ".com"}
		for _, directory := range r.pathDirectories() {
			for _, extension := range extensions {
				candidate := filepath.Join(directory, name+extension)
				if info, err := r.stat(candidate); err == nil && info.Mode().IsRegular() {
					return r.validate(candidate)
				}
			}
		}
		return "", fmt.Errorf("%w: %s (procurei .exe/.cmd/.bat/.com no PATH)", ErrGovernedCLIBinaryNotFound, name)
	}
	if path, err := r.lookPath(name); err == nil && path != "" {
		return r.validate(path)
	}
	for _, directory := range r.pathDirectories() {
		candidate := filepath.Join(directory, name)
		if info, err := r.stat(candidate); err == nil && info.Mode().IsRegular() {
			return r.validate(candidate)
		}
	}
	return "", fmt.Errorf("%w: %s", ErrGovernedCLIBinaryNotFound, name)
}

// validate confirma que o alvo é um arquivo comum executável direto e que não é
// um interpretador genérico (mesma lista já usada para comandos MCP).
func (r governedCLIResolver) validate(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: caminho vazio", ErrGovernedCLIBinaryNotFound)
	}
	info, err := r.stat(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrGovernedCLIBinaryNotFound, path)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s nao e arquivo comum", ErrGovernedCLIBinaryUnsupported, path)
	}
	base := strings.ToLower(filepath.Base(path))
	base = strings.TrimSuffix(base, ".exe")
	if genericInterpreters[base] || versionedInterpreter.MatchString(filepath.Base(path)) {
		return "", fmt.Errorf("%w: %s e um interpretador generico, nao a CLI", ErrGovernedCLIBinaryUnsupported, filepath.Base(path))
	}
	if r.goos == "windows" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe", ".cmd", ".bat", ".com":
			return path, nil
		default:
			return "", fmt.Errorf("%w: %s (no Windows o harness executa apenas .exe/.cmd/.bat/.com; shims .ps1 dependem da politica de execucao do PowerShell)", ErrGovernedCLIBinaryUnsupported, path)
		}
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%w: %s sem permissao de execucao", ErrGovernedCLIBinaryUnsupported, path)
	}
	return path, nil
}

// Variáveis repassadas ao CLI. Tudo o que não está aqui é descartado, inclusive
// qualquer chave de API, token ou sessão que exista no ambiente do Hades.
var governedCLIEnvAllowlist = []string{
	"PATH", "PATHEXT", "ComSpec", "SystemRoot", "SystemDrive", "windir",
	"HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "APPDATA", "LOCALAPPDATA",
	"PROGRAMDATA", "PROGRAMFILES", "TEMP", "TMP", "TMPDIR",
	"LANG", "LC_ALL", "LC_CTYPE", "TERM", "SHELL", "USER", "LOGNAME",
	"NUMBER_OF_PROCESSORS", "OS", "PROCESSOR_ARCHITECTURE",
}

// Marcadores de segredo removidos mesmo que um nome entre na allowlist acima.
var governedCLISecretMarkers = []string{
	"_API_KEY", "_APIKEY", "_TOKEN", "_SECRET", "_PASSWORD", "_PASSWD", "_CREDENTIAL",
	"_SESSION_KEY", "_SESSION_ID", "_PRIVATE_KEY", "_ACCESS_KEY",
	"ANTHROPIC_", "OPENAI_", "GEMINI_", "GOOGLE_API_KEY", "COPILOT_",
	"AWS_", "GITHUB_", "GH_", "NPM_", "CLAUDE_",
}

func governedCLIKeyLooksSecret(key string) bool {
	normalized := strings.ToUpper(key)
	for _, marker := range governedCLISecretMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// governedCLIEnv monta o ambiente mínimo do processo filho, em ordem estável.
func governedCLIEnv(source func(string) string) []string {
	if source == nil {
		source = os.Getenv
	}
	env := make([]string, 0, len(governedCLIEnvAllowlist))
	for _, key := range governedCLIEnvAllowlist {
		if governedCLIKeyLooksSecret(key) {
			continue
		}
		value := source(key)
		if strings.TrimSpace(value) == "" {
			continue
		}
		env = append(env, key+"="+value)
	}
	sort.Strings(env)
	return env
}

// cliOutputBuffer limita a captura sem quebrar o filho: o excedente é
// descartado e apenas sinalizado, ao contrário de um erro de escrita que
// mataria o CLI no meio da resposta.
type cliOutputBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *cliOutputBuffer) Write(data []byte) (int, error) {
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(data), nil
	}
	if len(data) > remaining {
		b.truncated = true
		_, _ = b.buf.Write(data[:remaining])
		return len(data), nil
	}
	if _, err := b.buf.Write(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (b *cliOutputBuffer) String() string { return b.buf.String() }

// GovernedCLIRequest descreve uma execução pedida pelo operador.
type GovernedCLIRequest struct {
	Harness      string
	Prompt       string
	Model        string
	Workdir      string
	Timeout      time.Duration
	BinaryPath   string
	ArgsOverride []string
	MaxOutput    int
	Env          []string
}

// GovernedCLIResult é o retorno auditável da execução.
type GovernedCLIResult struct {
	Harness         string
	DisplayName     string
	BinaryPath      string
	ExitCode        int
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	TimedOut        bool
	DurationMS      int64
	CommandLine     string
	DLPFindings     int
	Environment     string
	Isolation       string
	ResourceLimits  string
}

func sanitizeGovernedCLIText(text string, limit int) string {
	redacted := RedactDLP(text)
	if limit > 0 && len(redacted) > limit {
		redacted = redacted[:limit]
	}
	return redacted
}

func validateGovernedCLIArguments(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: linha de comando vazia", ErrGovernedCLILimit)
	}
	if len(args) > maxGovernedCLIArgCount {
		return fmt.Errorf("%w: %d itens na linha de comando (maximo %d)", ErrGovernedCLILimit, len(args), maxGovernedCLIArgCount)
	}
	for _, arg := range args {
		if len(arg) > maxGovernedCLIArgumentBytes {
			return fmt.Errorf("%w: item com %d bytes na linha de comando (maximo %d)", ErrGovernedCLILimit, len(arg), maxGovernedCLIArgumentBytes)
		}
	}
	return nil
}

// runGovernedCLI executa a CLI sob timeout, com ambiente saneado, saída
// limitada e redação por DLP. Sai com erro apenas quando nada foi executado
// (harness desconhecido, binário inválido, DLP, limite): código de saída
// diferente de zero é resultado, não falha de infraestrutura.
func runGovernedCLI(ctx context.Context, request GovernedCLIRequest) (GovernedCLIResult, error) {
	result := GovernedCLIResult{
		ExitCode:       -1,
		Environment:    governedCLIEnvironmentPolicy,
		Isolation:      governedCLIIsolation,
		ResourceLimits: governedCLIResourceLimits,
	}
	spec, ok := lookupGovernedCLI(request.Harness)
	if !ok {
		return result, fmt.Errorf("%w: %q", ErrUnknownGovernedCLI, request.Harness)
	}
	result.Harness = spec.Harness
	result.DisplayName = spec.DisplayName

	prompt := request.Prompt
	if strings.TrimSpace(prompt) == "" {
		return result, fmt.Errorf("%w: prompt vazio", ErrGovernedCLILimit)
	}
	if len(prompt) > maxGovernedCLIPromptBytes {
		return result, fmt.Errorf("%w: prompt com %d bytes (maximo %d)", ErrGovernedCLILimit, len(prompt), maxGovernedCLIPromptBytes)
	}
	promptFindings := ScanDLP(prompt)
	if len(promptFindings) > 0 {
		return result, fmt.Errorf("%w: %s", ErrGovernedCLIPromptBlocked, dlpFindingKinds(promptFindings))
	}

	args := request.ArgsOverride
	if len(args) == 0 {
		args = spec.Args
	}
	rendered := renderGovernedCLIArgs(args, prompt, request.Model)
	if err := validateGovernedCLIArguments(rendered); err != nil {
		return result, err
	}
	for _, arg := range rendered {
		if findings := ScanDLP(arg); len(findings) > 0 {
			return result, fmt.Errorf("%w: argumento com %s", ErrGovernedCLIPromptBlocked, dlpFindingKinds(findings))
		}
	}

	resolver := defaultGovernedCLIResolver()
	var binaryPath string
	var err error
	if strings.TrimSpace(request.BinaryPath) != "" {
		binaryPath, err = resolver.validate(request.BinaryPath)
	} else {
		binaryPath, err = resolver.resolve(spec.Binary)
	}
	if err != nil {
		return result, fmt.Errorf("%s: %w (instale com: %s)", spec.DisplayName, err, spec.InstallHint)
	}
	result.BinaryPath = binaryPath
	result.CommandLine = sanitizeGovernedCLIText(strings.Join(append([]string{binaryPath}, rendered...), " "), 4096)

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultGovernedCLITimeout
	}
	if timeout > maxGovernedCLITimeout {
		timeout = maxGovernedCLITimeout
	}
	limit := request.MaxOutput
	if limit <= 0 {
		limit = defaultGovernedCLIMaxOutput
	}
	if limit > maxGovernedCLIMaxOutput {
		limit = maxGovernedCLIMaxOutput
	}

	env := request.Env
	if len(env) == 0 {
		env = governedCLIEnv(os.Getenv)
	}
	stdout := &cliOutputBuffer{limit: limit}
	stderr := &cliOutputBuffer{limit: limit}

	executionContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.Command(binaryPath, rendered...)
	command.Dir = request.Workdir
	command.Env = env
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	configureToolProcess(command)

	startedAt := time.Now()
	runErr := runToolCommand(executionContext, command)
	result.DurationMS = time.Since(startedAt).Milliseconds()
	result.Stdout = sanitizeGovernedCLIText(stdout.String(), limit)
	result.Stderr = sanitizeGovernedCLIText(stderr.String(), limit)
	result.StdoutTruncated = stdout.truncated
	result.StderrTruncated = stderr.truncated
	result.DLPFindings = len(ScanDLP(stdout.String())) + len(ScanDLP(stderr.String()))

	switch {
	case runErr == nil:
		result.ExitCode = 0
	case errors.Is(runErr, context.DeadlineExceeded), errors.Is(runErr, context.Canceled):
		result.TimedOut = true
	default:
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			result.ExitCode = exitError.ExitCode()
			break
		}
		return result, fmt.Errorf("falha ao executar %s: %w", spec.DisplayName, runErr)
	}
	return result, nil
}

func dlpFindingKinds(findings []DLPFinding) string {
	kinds := make([]string, 0, len(findings))
	seen := make(map[string]bool, len(findings))
	for _, finding := range findings {
		if finding.Kind == "" || seen[finding.Kind] {
			continue
		}
		seen[finding.Kind] = true
		kinds = append(kinds, finding.Kind)
	}
	sort.Strings(kinds)
	return "achados de DLP: " + strings.Join(kinds, ",")
}
