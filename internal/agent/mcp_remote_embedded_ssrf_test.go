package agent

import (
	"context"
	"net"
	"strings"
	"testing"
)

// Evidência de que a correção do IPv4 embutido alcança a DEFESA REAL, e não
// apenas o helper puro: o dial do remote MCP é quem decide se o socket é aberto.
// Um destino que se disfarça de IPv6 (NAT64/6to4) apontando para metadata de
// nuvem, loopback ou rede privada precisa ser recusado antes de qualquer conexão.
func TestRemoteMCPDialRejectsMetadataHiddenInTransitionPrefix(t *testing.T) {
	cases := []struct {
		name     string
		resolved string
	}{
		{"NAT64 para metadata de nuvem", "64:ff9b::a9fe:a9fe"},
		{"NAT64 local-use para metadata", "64:ff9b:1::a9fe:a9fe"},
		{"NAT64 para loopback", "64:ff9b::7f00:1"},
		{"6to4 para metadata de nuvem", "2002:a9fe:a9fe::1"},
		{"6to4 para rede privada", "2002:c0a8:101::1"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved := net.ParseIP(testCase.resolved)
			if resolved == nil {
				t.Fatalf("endereço de teste inválido: %s", testCase.resolved)
			}
			lookup := func(context.Context, string) ([]net.IP, error) {
				return []net.IP{resolved}, nil
			}
			conn, err := remoteMCPDialContextWithResolver(context.Background(), "tcp", "metadata.internal:443", lookup)
			if err == nil {
				_ = conn.Close()
				t.Fatalf("dial para %s deveria ser recusado", testCase.resolved)
			}
			// A recusa precisa vir da POLÍTICA, não de uma falha de rede: é isso
			// que prova que a classificação de IPv4 embutido está no caminho.
			if !strings.Contains(err.Error(), "private address") {
				t.Fatalf("recusa deveria vir da política de egresso, veio: %v", err)
			}
		})
	}
}

// O outro lado da correção: um NAT64/6to4 legítimo (destino público) continua
// podendo ser discado. Sem este teste, "bloquear o prefixo inteiro" passaria como
// correção e quebraria egress real em rede IPv6-only com DNS64.
func TestRemoteMCPDialPermitsPublicAddressInsideTransitionPrefix(t *testing.T) {
	// Escuta em loopback apenas para ter um alvo local controlado; o dial recebe
	// um IP APROVADO (público) e o teste verifica que a decisão de política é
	// "permitido" — a conexão em si falha em seguida, o que não é o objeto aqui.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	lookup := func(context.Context, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("64:ff9b::808:808")}, nil
	}
	// O endereço aprovado é público, então o dialer prossegue para conectar no IP
	// aprovado (8.8.8.8, inalcançável no teste) e devolve erro de REDE, não de
	// política. É essa distinção que prova a ausência de over-blocking.
	conn, err := remoteMCPDialContextWithResolver(context.Background(), "tcp", net.JoinHostPort("public.example", port), lookup)
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Skip("conexão de rede inesperadamente bem-sucedida neste ambiente")
	}
	if strings.Contains(err.Error(), "embedded in transition prefix") {
		t.Fatalf("destino público dentro do prefixo foi bloqueado por política: %v", err)
	}
}
