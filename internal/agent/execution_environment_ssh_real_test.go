package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// A5 PROVADO A SERIO: ligacao SSH REAL a um servidor efemero em processo, com chave de host
// propria, autenticacao por chave publica e um comando executado do outro lado.
//
// Porque isto importa: a E-1454 mediu que existe `SSHEnvironment` com `HostKeyCallback`
// obrigatoria, mas NAO provou execucao remota. Um cliente que nunca abriu ligacao nenhuma
// passa em 22 testes e continua a não ser um runtime remoto.
func servidorSSHEfemero(t *testing.T, chavePublicaCliente ssh.PublicKey) (endereco string, chaveHost ssh.PublicKey) {
	t.Helper()

	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gerar chave de host: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("signer de host: %v", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, chave ssh.PublicKey) (*ssh.Permissions, error) {
			if string(chave.Marshal()) == string(chavePublicaCliente.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, os.ErrPermission
		},
	}
	config.AddHostKey(hostSigner)

	ouvinte, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ouvinte: %v", err)
	}
	t.Cleanup(func() { _ = ouvinte.Close() })

	go func() {
		for {
			conn, errAccept := ouvinte.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, canais, pedidos, errHandshake := ssh.NewServerConn(conn, config)
				if errHandshake != nil {
					return
				}
				go ssh.DiscardRequests(pedidos)
				for canal := range canais {
					if canal.ChannelType() != "session" {
						_ = canal.Reject(ssh.UnknownChannelType, "apenas sessao")
						continue
					}
					aceite, pedidosCanal, errAceite := canal.Accept()
					if errAceite != nil {
						continue
					}
					go func() {
						defer aceite.Close()
						for pedido := range pedidosCanal {
							if pedido.Type != "exec" {
								_ = pedido.Reply(false, nil)
								continue
							}
							_ = pedido.Reply(true, nil)
							// Resposta fixa: o teste prova o CAMINHO (handshake, host key,
							// comando executado), nao um shell remoto qualquer.
							_, _ = aceite.Write([]byte("pong-remoto"))
							_, _ = aceite.SendRequest("exit-status", false, ssh.Marshal(struct{ C uint32 }{0}))
							return
						}
					}()
				}
			}()
		}
	}()

	return ouvinte.Addr().String(), hostSigner.PublicKey()
}

func escreverChavePrivadaDeTeste(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gerar chave de cliente: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal chave: %v", err)
	}
	caminho := filepath.Join(t.TempDir(), "id_cliente")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(caminho, pemBytes, 0o600); err != nil {
		t.Fatalf("escrever chave: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer de cliente: %v", err)
	}
	t.Cleanup(func() {
		// a chave publica do cliente e' derivada do signer pelo chamador
		_ = signer
	})
	// guarda a publica no ficheiro de apoio para o servidor validar
	sshPub := filepath.Join(filepath.Dir(caminho), "id_cliente.pub")
	print, err := os.OpenFile(sshPub, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("escrever pub: %v", err)
	}
	defer print.Close()
	_, _ = print.WriteString(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	return caminho
}

func chavePublicaDoFicheiro(t *testing.T, caminho string) ssh.PublicKey {
	t.Helper()
	chaveBytes, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler chave privada: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(chaveBytes)
	if err != nil {
		t.Fatalf("parse chave privada: %v", err)
	}
	return signer.PublicKey()
}

// POSITIVO: com a chave de host CORRECTA, o comando corre do outro lado.
func TestSSHEnvironmentExecutaEmServidorRealComHostKeyCorreta(t *testing.T) {
	caminhoChave := escreverChavePrivadaDeTeste(t)
	pub := chavePublicaDoFicheiro(t, caminhoChave)
	endereco, chaveHost := servidorSSHEfemero(t, pub)

	ambiente := &SSHEnvironment{
		Host:            endereco,
		User:            "tester",
		PrivateKeyPath:  caminhoChave,
		HostKeyCallback: ssh.FixedHostKey(chaveHost),
		DefaultTimeout:  10 * time.Second,
	}

	if err := ambiente.Probe(context.Background()); err != nil {
		t.Fatalf("Probe devia passar com host key correcta: %v", err)
	}

	resultado, err := ambiente.Run(context.Background(), ExecutionRequest{Command: "echo ola"})
	if err != nil {
		t.Fatalf("Run devia executar do outro lado: %v", err)
	}
	if !strings.Contains(resultado.Stdout, "pong-remoto") {
		t.Fatalf("stdout devia vir do servidor remoto, veio %q", resultado.Stdout)
	}
	if resultado.Locality != LocalitySSH {
		t.Fatalf("a localidade devia ser SSH, veio %q", resultado.Locality)
	}
}

// NEGATIVO: chave de host DIFERENTE -> o handshake tem de falhar (nao ha' InsecureIgnoreHostKey).
func TestSSHEnvironmentRecusaHostKeyDivergente(t *testing.T) {
	caminhoChave := escreverChavePrivadaDeTeste(t)
	pub := chavePublicaDoFicheiro(t, caminhoChave)
	endereco, _ := servidorSSHEfemero(t, pub)

	_, outraHostPub, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gerar outra chave: %v", err)
	}
	outroSigner, err := ssh.NewSignerFromKey(outraHostPub)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	ambiente := &SSHEnvironment{
		Host:            endereco,
		User:            "tester",
		PrivateKeyPath:  caminhoChave,
		HostKeyCallback: ssh.FixedHostKey(outroSigner.PublicKey()),
		DefaultTimeout:  10 * time.Second,
	}
	if _, err := ambiente.Run(context.Background(), ExecutionRequest{Command: "echo ola"}); err == nil {
		t.Fatal("com chave de host DIVERGENTE a execucao tem de falhar")
	}
}

// NEGATIVO: sem HostKeyCallback nao ha' ligacao nenhuma (falha fechada, antes do dial).
func TestSSHEnvironmentSemHostKeyCallbackFalhaFechado(t *testing.T) {
	caminhoChave := escreverChavePrivadaDeTeste(t)
	pub := chavePublicaDoFicheiro(t, caminhoChave)
	endereco, _ := servidorSSHEfemero(t, pub)

	ambiente := &SSHEnvironment{
		Host:           endereco,
		User:           "tester",
		PrivateKeyPath: caminhoChave,
		DefaultTimeout: 5 * time.Second,
	}
	if _, err := ambiente.Run(context.Background(), ExecutionRequest{Command: "echo ola"}); err == nil {
		t.Fatal("sem HostKeyCallback a execucao tem de ser recusada")
	}
	if err := ambiente.Probe(context.Background()); err == nil {
		t.Fatal("sem HostKeyCallback o Probe tem de recusar")
	}
}
