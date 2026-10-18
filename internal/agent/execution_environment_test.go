package agent

// FASE 6 do adendo V3: runtime local, SSH e falha fechada.
//
// O teste central e' TestSSHSemConfiguracaoFalhaFechadaSemCorrerLocalmente: ele FALHA se o provider
// SSH passar a "compensar" a ausencia de host correndo o comando nesta maquina. Esse fallback
// silencioso mudaria a localidade do dado sem ninguem ter decidido isso.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// TestLocalExecutaDeFacto prova que o provider local nao e' decorativo.
func TestLocalExecutaDeFacto(t *testing.T) {
	env := &LocalEnvironment{}
	if err := env.Health(context.Background()); err != nil {
		t.Fatalf("local health: %v", err)
	}
	if env.Locality() != LocalityLocal {
		t.Fatalf("localidade errada: %s", env.Locality())
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	res, err := env.Run(ctx, ExecutionRequest{Command: "echo hades-probe", Timeout: 20 * time.Second})
	if err != nil {
		t.Fatalf("local run: %v", err)
	}
	if !strings.Contains(res.Stdout, "hades-probe") {
		t.Fatalf("stdout sem marcador: %q", res.Stdout)
	}
	if res.Locality != LocalityLocal || res.Provider != "local" {
		t.Fatalf("resultado sem carimbo de localidade: %+v", res)
	}
}

// TestLocalTruncaSaidaAoLimite prova que o limite de output e' aplicado e DECLARADO.
func TestLocalTruncaSaidaAoLimite(t *testing.T) {
	env := &LocalEnvironment{}
	res, err := env.Run(context.Background(), ExecutionRequest{
		Command:   "echo 0123456789012345678901234567890123456789",
		MaxOutput: 5,
		Timeout:   20 * time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Stdout) > 5 {
		t.Fatalf("limite nao aplicado: %d bytes", len(res.Stdout))
	}
	if !res.Truncated {
		t.Fatalf("truncagem nao declarada no resultado")
	}
}

// TestSSHSemConfiguracaoFalhaFechadaSemCorrerLocalmente e' O teste discriminante da FASE 6.
//
// Ele falha de tres maneiras se alguem introduzir um fallback silencioso para local:
//  1. o erro deixa de ser ErrCapacidadeAusente;
//  2. o stdout passa a conter a saida do comando (logo correu);
//  3. o runner injetado passa a ser chamado (contador > 0).
func TestSSHSemConfiguracaoFalhaFechadaSemCorrerLocalmente(t *testing.T) {
	chamadas := 0
	env := &SSHEnvironment{
		// host, user, chave e callback todos vazios de proposito.
		runner: func(ctx context.Context, comando string, limite int) (string, string, int, error) {
			chamadas++
			return "CORREU-NA-MAQUINA-ERRADA", "", 0, nil
		},
	}
	res, err := env.Run(context.Background(), ExecutionRequest{Command: "echo x"})
	if !errors.Is(err, ErrCapacidadeAusente) {
		t.Fatalf("esperado ErrCapacidadeAusente, obtido %v", err)
	}
	if chamadas != 0 {
		t.Fatalf("runner foi chamado %d vez(es) apesar de nao estar configurado", chamadas)
	}
	if strings.Contains(res.Stdout, "CORREU") {
		t.Fatalf("provider SSH executou localmente: %q", res.Stdout)
	}
	if res.Locality != LocalitySSH {
		t.Fatalf("resultado nao carimba a localidade SSH: %s", res.Locality)
	}
}

// TestSSHExigeHostKeyCallback prova que nao existe InsecureIgnoreHostKey implicito no PRODUTO.
// O teste fornece um callback explicito, como qualquer chamador teria de fazer.
func TestSSHExigeHostKeyCallback(t *testing.T) {
	env := &SSHEnvironment{Host: "127.0.0.1:1", User: "u", PrivateKeyPath: "k"}
	err := env.Probe(context.Background())
	if !errors.Is(err, ErrCapacidadeAusente) {
		t.Fatalf("probe sem host key callback devia falhar fechado, obtido %v", err)
	}
	if !strings.Contains(err.Error(), "host key callback") {
		t.Fatalf("erro devia nomear o callback em falta: %v", err)
	}
	// Com callback explicito a configuracao passa a ser considerada; a falha passa a ser de REDE
	// (porta 1 fechada), nao de configuracao. Ambos sao falha FECHADA.
	env.HostKeyCallback = ssh.InsecureIgnoreHostKey() // apenas no teste: o produto exige um real
	if err := env.Probe(context.Background()); err == nil {
		t.Fatalf("probe a porta fechada devia falhar")
	}
}

// TestSSHComConfiguracaoExecutaPeloRunner prova o caminho feliz sem rede real.
func TestSSHComConfiguracaoExecutaPeloRunner(t *testing.T) {
	chamadas := 0
	env := &SSHEnvironment{
		Host: "h", User: "u", PrivateKeyPath: "k",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // apenas no teste
		runner: func(ctx context.Context, comando string, limite int) (string, string, int, error) {
			chamadas++
			return "remoto-ok", "", 0, nil
		},
	}
	res, err := env.Run(context.Background(), ExecutionRequest{Command: "echo x", Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("run configurado: %v", err)
	}
	if chamadas != 1 {
		t.Fatalf("runner chamado %d vezes, esperado 1", chamadas)
	}
	if res.Stdout != "remoto-ok" || res.Locality != LocalitySSH || res.Provider != "ssh" {
		t.Fatalf("resultado SSH errado: %+v", res)
	}
	// Doctor nao pode revelar o CONTEUDO da chave -- so o facto de estar configurada.
	for _, linha := range env.Doctor(context.Background()) {
		if strings.Contains(linha, "BEGIN") || strings.Contains(linha, "PRIVATE") {
			t.Fatalf("doctor vaza material de chave: %q", linha)
		}
	}
}
