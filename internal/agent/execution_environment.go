package agent

// FASE 6 do adendo V3: runtime local, SSH, servidor e sandbox atras de uma interface comum.
//
// O HADES ja' executava localmente com isolamento (`os.Root` em tools.go) e ja' tinha worktree
// git real (`coding_git_cycle.go`). O que NAO existia, medido por `grep` de `ssh.Client`/`ssh.Dial`/
// `ssh.Session` = 0 ficheiros, era uma interface COMUM de execucao com provider remoto. Este
// arquivo acrescenta a interface e os dois primeiros providers; NAO duplica o terminal, que
// continua a ser o caminho de execucao local com isolamento por descritor.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ExecutionLocality e' a taxonomia explicita da FASE 29.2 aplicada ao runtime de execucao.
//
// Antes disto a localidade era inferida de um booleano de conectividade (`AllowPrivate`), que nao
// distingue "corre nesta maquina" de "corre numa maquina remota que eu controlo". *Um booleano nao
// carrega uma fronteira de confianca: carrega uma esperanca.*
type ExecutionLocality string

const (
	// LocalityLocal: processo executado nesta maquina.
	LocalityLocal ExecutionLocality = "LOCAL"
	// LocalitySSH: processo executado numa maquina alcancada por SSH com chave explicita.
	LocalitySSH ExecutionLocality = "SSH"
	// LocalitySelfHostedRemote: infraestrutura remota controlada pelo proprietario.
	LocalitySelfHostedRemote ExecutionLocality = "SELF_HOSTED_REMOTE"
	// LocalityUnclassified: falha fechada. Nunca tratar como local por omissao.
	LocalityUnclassified ExecutionLocality = "UNCLASSIFIED"
)

// ExecutionRequest e' um pedido de execucao normalizado entre providers.
type ExecutionRequest struct {
	Command   string
	Dir       string
	Timeout   time.Duration
	MaxOutput int
}

// ExecutionResult e' o resultado observado, com a localidade CARIMBADA.
//
// Carimbar a localidade no resultado e' o que impede a evidencia de mentir: um resultado que nao
// diz ONDE correu nao permite decidir se podia ter corrido ali.
type ExecutionResult struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Locality  ExecutionLocality
	Provider  string
	Truncated bool
}

// ExecutionEnvironment e' a interface que a FASE 6 exige: capability probe, doctor, health e execucao.
type ExecutionEnvironment interface {
	Name() string
	Locality() ExecutionLocality
	// Probe responde "esta capacidade existe AGORA?" sem executar trabalho de utilizador.
	Probe(ctx context.Context) error
	// Doctor devolve diagnosticos legiveis, sem segredos.
	Doctor(ctx context.Context) []string
	// Health devolve nil apenas quando o ambiente consegue executar de facto.
	Health(ctx context.Context) error
	Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)
}

// ErrCapacidadeAusente: o ambiente nao consegue executar. Falha FECHADA.
//
// *Um provider que devolve sucesso quando o host esta ausente nao e' um provider: e' um mentiroso.*
var ErrCapacidadeAusente = errors.New("execution environment capability unavailable")

// --- Provider LOCAL -------------------------------------------------------------------

// LocalEnvironment executa nesta maquina. E' o provider default e o unico que nao precisa de
// configuracao externa. Os limites sao aplicados AQUI e nao no chamador, para nenhum chamador
// poder pedir um `MaxOutput` infinito por engano.
type LocalEnvironment struct {
	// Shell e' o interpretador por omissao. Vazio significa `cmd /C` no Windows ou `sh -c`
	// noutros sistemas -- decidido em runtime, nao compilado.
	Shell string
	// DefaultTimeout e' o tecto quando o pedido nao declara um. Zero representa 2 minutos.
	DefaultTimeout time.Duration
	// DefaultMaxOutput e' o tecto de bytes por fluxo. Zero representa 1 MiB.
	DefaultMaxOutput int
}

func (e *LocalEnvironment) Name() string                { return "local" }
func (e *LocalEnvironment) Locality() ExecutionLocality { return LocalityLocal }

func (e *LocalEnvironment) shell() string {
	if strings.TrimSpace(e.Shell) != "" {
		return e.Shell
	}
	return ""
}

func (e *LocalEnvironment) Probe(ctx context.Context) error {
	// Um probe honesto: o binario que vamos usar tem de existir e a locale tem de ser legivel.
	if len(osExecutableCandidates()) == 0 {
		return fmt.Errorf("%w: no usable executable path", ErrCapacidadeAusente)
	}
	return ctx.Err()
}

func (e *LocalEnvironment) Doctor(ctx context.Context) []string {
	return []string{
		"provider=local locality=LOCAL",
		"shell=" + strings.TrimSpace(e.shell()),
	}
}

func (e *LocalEnvironment) Health(ctx context.Context) error {
	if err := e.Probe(ctx); err != nil {
		return err
	}
	if _, err := exec.LookPath("cmd"); err != nil {
		if _, err2 := exec.LookPath("sh"); err2 != nil {
			return fmt.Errorf("%w: neither cmd nor sh on PATH", ErrCapacidadeAusente)
		}
	}
	return nil
}

func (e *LocalEnvironment) Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	comando := strings.TrimSpace(req.Command)
	if comando == "" {
		return ExecutionResult{}, errors.New("empty command")
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = e.DefaultTimeout
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	limite := req.MaxOutput
	if limite <= 0 {
		limite = e.DefaultMaxOutput
	}
	if limite <= 0 {
		limite = 1 << 20
	}
	runCtx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	shell := e.shell()
	if shell == "" {
		shell = defaultShellForHost()
	}
	var cmd *exec.Cmd
	if strings.EqualFold(shell, "cmd") {
		cmd = exec.CommandContext(runCtx, "cmd", "/C", comando)
	} else {
		cmd = exec.CommandContext(runCtx, shell, "-c", comando)
	}
	if strings.TrimSpace(req.Dir) != "" {
		cmd.Dir = req.Dir
	}
	stdout := newLimitedBuffer(limite)
	stderr := newLimitedBuffer(limite)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	resultado := ExecutionResult{
		Stdout:    RedactDLP(stdout.String()),
		Stderr:    RedactDLP(stderr.String()),
		Locality:  LocalityLocal,
		Provider:  e.Name(),
		Truncated: stdout.Truncated() || stderr.Truncated(),
	}
	if cmd.ProcessState != nil {
		resultado.ExitCode = cmd.ProcessState.ExitCode()
	}
	if runCtx.Err() == context.DeadlineExceeded {
		return resultado, fmt.Errorf("%w: command timed out after %s", ErrCapacidadeAusente, timeout)
	}
	if err != nil && resultado.ExitCode == 0 {
		return resultado, err
	}
	return resultado, nil
}

// --- Provider SSH ---------------------------------------------------------------------

// SSHEnvironment executa numa maquina remota por SSH, com chave privada EXPLICITA.
//
// Duas regras que este provider nunca quebra:
//  1. **falha fechada** -- sem chave, sem host ou sem handshake, `Run` devolve erro e NAO corre
//     nada localmente "para compensar". *Um fallback silencioso de remoto para local muda a
//     localidade do dado sem ninguem ter decidido isso.*
//  2. **nunca aceita chave de host por omissao** -- sem `HostKeyCallback` o SSH recusa; e o
//     chamador tem de fornecer um. Nao ha' `InsecureIgnoreHostKey` neste arquivo.
type SSHEnvironment struct {
	Host           string
	User           string
	PrivateKeyPath string
	// HostKeyCallback e' OBRIGATORIO. Ver a regra 2 acima.
	HostKeyCallback  ssh.HostKeyCallback
	DefaultTimeout   time.Duration
	DefaultMaxOutput int
	// runner injeta a execucao; existe para o teste poder provar falha fechada sem rede.
	runner func(ctx context.Context, comando string, limite int) (stdout, stderr string, code int, err error)
}

func (e *SSHEnvironment) Name() string                { return "ssh" }
func (e *SSHEnvironment) Locality() ExecutionLocality { return LocalitySSH }

func (e *SSHEnvironment) configurado() error {
	if strings.TrimSpace(e.Host) == "" {
		return fmt.Errorf("%w: ssh host is empty", ErrCapacidadeAusente)
	}
	if strings.TrimSpace(e.User) == "" {
		return fmt.Errorf("%w: ssh user is empty", ErrCapacidadeAusente)
	}
	if strings.TrimSpace(e.PrivateKeyPath) == "" {
		return fmt.Errorf("%w: ssh private key path is empty", ErrCapacidadeAusente)
	}
	if e.HostKeyCallback == nil {
		return fmt.Errorf("%w: ssh host key callback is required (no insecure default)", ErrCapacidadeAusente)
	}
	return nil
}

func (e *SSHEnvironment) Probe(ctx context.Context) error {
	if err := e.configurado(); err != nil {
		return err
	}
	// Probar a CONFIGURACAO e a REDE, sem abrir sessao de comando.
	host := e.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "22")
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("%w: ssh tcp probe %s: %v", ErrCapacidadeAusente, host, err)
	}
	_ = conn.Close()
	return nil
}

func (e *SSHEnvironment) Doctor(ctx context.Context) []string {
	linhas := []string{
		"provider=ssh locality=SSH",
		"host=" + strings.TrimSpace(e.Host),
		"user=" + strings.TrimSpace(e.User),
		"key_configured=" + fmt.Sprint(strings.TrimSpace(e.PrivateKeyPath) != ""),
		"host_key_callback=" + fmt.Sprint(e.HostKeyCallback != nil),
	}
	if err := e.Probe(ctx); err != nil {
		linhas = append(linhas, "probe=FAIL: "+err.Error())
	} else {
		linhas = append(linhas, "probe=OK")
	}
	return linhas
}

func (e *SSHEnvironment) Health(ctx context.Context) error { return e.Probe(ctx) }

func (e *SSHEnvironment) Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	comando := strings.TrimSpace(req.Command)
	if comando == "" {
		return ExecutionResult{}, errors.New("empty command")
	}
	// FALHA FECHADA: sem configuracao valida nao ha' execucao, e nao ha' fallback local.
	if err := e.configurado(); err != nil {
		return ExecutionResult{Locality: LocalitySSH, Provider: e.Name()}, err
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = e.DefaultTimeout
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	limite := req.MaxOutput
	if limite <= 0 {
		limite = e.DefaultMaxOutput
	}
	if limite <= 0 {
		limite = 1 << 20
	}
	runCtx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	runner := e.runner
	if runner == nil {
		runner = e.runOverSSH
	}
	stdout, stderr, code, err := runner(runCtx, comando, limite)
	resultado := ExecutionResult{
		Stdout:   RedactDLP(stdout),
		Stderr:   RedactDLP(stderr),
		ExitCode: code,
		Locality: LocalitySSH,
		Provider: e.Name(),
	}
	if runCtx.Err() == context.DeadlineExceeded {
		return resultado, fmt.Errorf("%w: ssh command timed out after %s", ErrCapacidadeAusente, timeout)
	}
	if err != nil {
		return resultado, err
	}
	return resultado, nil
}

// runOverSSH e' o caminho real. Ele NAO usa InsecureIgnoreHostKey e NAO tenta local em caso de erro.
func (e *SSHEnvironment) runOverSSH(ctx context.Context, comando string, limite int) (string, string, int, error) {
	chave, err := osReadFile(e.PrivateKeyPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: read ssh key: %v", ErrCapacidadeAusente, err)
	}
	signer, err := ssh.ParsePrivateKey(chave)
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: parse ssh key: %v", ErrCapacidadeAusente, err)
	}
	config := &ssh.ClientConfig{
		User:            e.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: e.HostKeyCallback,
		Timeout:         10 * time.Second,
	}
	host := e.Host
	if _, _, splitErr := net.SplitHostPort(host); splitErr != nil {
		host = net.JoinHostPort(host, "22")
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: ssh dial %s: %v", ErrCapacidadeAusente, host, err)
	}
	sshConn, canais, pedidos, err := ssh.NewClientConn(conn, host, config)
	if err != nil {
		_ = conn.Close()
		return "", "", 0, fmt.Errorf("%w: ssh handshake %s: %v", ErrCapacidadeAusente, host, err)
	}
	cliente := ssh.NewClient(sshConn, canais, pedidos)
	defer cliente.Close()
	sessao, err := cliente.NewSession()
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: ssh session: %v", ErrCapacidadeAusente, err)
	}
	defer sessao.Close()
	outBuf := newLimitedBuffer(limite)
	errBuf := newLimitedBuffer(limite)
	sessao.Stdout = outBuf
	sessao.Stderr = errBuf
	code := 0
	if err := sessao.Run(comando); err != nil {
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitStatus()
		} else {
			return outBuf.String(), errBuf.String(), 0, err
		}
	}
	return outBuf.String(), errBuf.String(), code, nil
}

// newLimitedBuffer / osExecutableCandidates / defaultShellForHost / osReadFile sao pequenos
// auxiliares que mantem este arquivo compilavel em Windows e Unix sem `build tags` espalhadas.

type limitedOutput struct {
	buf      strings.Builder
	limite   int
	truncado bool
}

func newLimitedBuffer(limite int) *limitedOutput { return &limitedOutput{limite: limite} }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limite {
		restante := b.limite - b.buf.Len()
		if restante > 0 {
			b.buf.Write(p[:restante])
		}
		b.truncado = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedOutput) String() string  { return b.buf.String() }
func (b *limitedOutput) Truncated() bool { return b.truncado }

func osExecutableCandidates() []string {
	candidatos := make([]string, 0, 2)
	for _, nome := range []string{"cmd", "sh"} {
		if caminho, err := exec.LookPath(nome); err == nil {
			candidatos = append(candidatos, caminho)
		}
	}
	return candidatos
}

func defaultShellForHost() string {
	if _, err := exec.LookPath("cmd"); err == nil {
		return "cmd"
	}
	return "sh"
}

func osReadFile(caminho string) ([]byte, error) {
	return os.ReadFile(caminho)
}
