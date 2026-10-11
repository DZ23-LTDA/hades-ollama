package egresspolicy

import (
	"net"
	"net/netip"
	"strings"
	"testing"
)

// Achado corrigido: prefixos de transição encapsulam um IPv4 arbitrário. A
// política bloqueava 169.254.169.254, 127.0.0.1, 10.0.0.0/8 e ::ffff:169.254.169.254,
// mas PERMITIA os mesmos destinos escritos como literal NAT64 (64:ff9b::/96) ou
// 6to4 (2002::/16) — ou seja, exatamente o atalho que a política existe para
// fechar. Estes testes travam a correção e o limite dela (destino público
// encapsulado continua permitido, porque NAT64 via DNS64 é tráfego legítimo).
func TestClassifyIPBlocksIPv4EmbeddedInTransitionPrefixes(t *testing.T) {
	cases := []struct {
		name   string
		addr   string
		reason string
	}{
		{"NAT64 para metadata de nuvem", "64:ff9b::a9fe:a9fe", "restricted or non-global"},
		{"NAT64 para loopback", "64:ff9b::7f00:1", "restricted or non-global"},
		{"NAT64 para rede privada", "64:ff9b::a00:1", "private network address"},
		{"NAT64 local-use para metadata", "64:ff9b:1::a9fe:a9fe", "restricted or non-global"},
		{"6to4 para metadata de nuvem", "2002:a9fe:a9fe::1", "restricted or non-global"},
		{"6to4 para loopback", "2002:7f00:1::1", "restricted or non-global"},
		{"6to4 para rede privada", "2002:c0a8:101::1", "private network address"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			blocked, reason := ClassifyIP(net.ParseIP(testCase.addr))
			if !blocked {
				t.Fatalf("%s deveria ser bloqueado", testCase.addr)
			}
			if !strings.Contains(reason, "embedded in transition prefix") {
				t.Fatalf("motivo deveria apontar o IPv4 embutido, veio %q", reason)
			}
			if !strings.Contains(reason, testCase.reason) {
				t.Fatalf("motivo deveria conter %q, veio %q", testCase.reason, reason)
			}
		})
	}
}

func TestClassifyIPAllowsPublicIPv4EmbeddedInTransitionPrefixes(t *testing.T) {
	// NAT64/6to4 legítimos carregam um destino público: não podem ser bloqueados
	// junto, senão a correção quebraria egress real em rede IPv6-only com DNS64.
	for _, address := range []string{
		"64:ff9b::808:808", // NAT64 -> 8.8.8.8
		"64:ff9b::101:101", // NAT64 -> 1.1.1.1
		"2002:808:808::1",  // 6to4 -> 8.8.8.8
		"2002:101:101::1",  // 6to4 -> 1.1.1.1
	} {
		if blocked, reason := ClassifyIP(net.ParseIP(address)); blocked {
			t.Fatalf("%s encapsula destino público e não deveria ser bloqueado (%s)", address, reason)
		}
	}
}

func TestEmbeddedIPv4ExtractsOnlyKnownTransitionPrefixes(t *testing.T) {
	cases := []struct {
		addr     string
		expected string
		ok       bool
	}{
		{"64:ff9b::a9fe:a9fe", "169.254.169.254", true},
		{"64:ff9b:1::7f00:1", "127.0.0.1", true},
		{"2002:7f00:1::1", "127.0.0.1", true},
		{"2606:4700::1", "", false},           // IPv6 público comum
		{"64:ff9b:2::1", "", false},           // NAT64 fora do prefixo conhecido
		{"64:ff9b::1:0:0:1", "", false},       // não é well-known (bytes do meio ≠ 0)
		{"2001:db8::1", "", false},            // documentação, não é transição
		{"::ffff:169.254.169.254", "", false}, // IPv4-mapped: resolvido por Unmap
		{"8.8.8.8", "", false},                // IPv4 nativo
	}
	for _, testCase := range cases {
		parsed, err := netip.ParseAddr(testCase.addr)
		if err != nil {
			t.Fatalf("endereço inválido %s: %v", testCase.addr, err)
		}
		if parsed.Is4In6() {
			parsed = parsed.Unmap()
		}
		embedded, ok := EmbeddedIPv4(parsed)
		if ok != testCase.ok {
			t.Fatalf("%s: ok=%v, esperado %v", testCase.addr, ok, testCase.ok)
		}
		if ok && embedded.String() != testCase.expected {
			t.Fatalf("%s: extraiu %s, esperado %s", testCase.addr, embedded, testCase.expected)
		}
	}
}
