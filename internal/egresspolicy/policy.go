// Package egresspolicy contains the single conservative IP policy used by every outbound adapter.
package egresspolicy

import (
	"net"
	"net/netip"
)

// ClassifyIP reports whether an address is unsafe for outbound communication.
// The policy intentionally blocks non-global, special-use, documentation and
// benchmarking ranges because DNS-pinned outbound clients must fail closed.
func ClassifyIP(ip net.IP) (blocked bool, reason string) {
	if ip == nil {
		return true, "nil IP address"
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	address, ok := netip.AddrFromSlice(ip)
	if !ok || !address.IsGlobalUnicast() {
		return true, "restricted or non-global IP address"
	}
	address = address.Unmap()
	if address.IsPrivate() || address.IsLoopback() || address.IsUnspecified() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() {
		return true, classifyStandard(address)
	}
	// Prefixos de transição encapsulam um IPv4 arbitrário. Sem isto, um literal
	// como 64:ff9b::a9fe:a9fe (NAT64) ou 2002:a9fe:a9fe::1 (6to4) atravessa a
	// política e alcança metadata de nuvem / loopback / rede privada — o atalho
	// exato que esta política existe para fechar. O IP embutido passa pela MESMA
	// classificação, então NAT64 legítimo (DNS64 que aponta para host público)
	// segue permitido e apenas os destinos internos são recusados.
	if embedded, ok := EmbeddedIPv4(address); ok {
		if embeddedBlocked, embeddedReason := ClassifyIP(embedded); embeddedBlocked {
			return true, "IPv4 embedded in transition prefix: " + embeddedReason
		}
	}

	blockedPrefixes := [...]netip.Prefix{
		netip.MustParsePrefix("100.64.0.0/10"), // RFC 6598 CGNAT
		netip.MustParsePrefix("198.18.0.0/15"), // RFC 2544 benchmarking
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"), // TEST-NET-1
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
		netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("2001:0000::/32"),
		netip.MustParsePrefix("2001:0002::/48"),
		netip.MustParsePrefix("2001:0010::/28"),
		netip.MustParsePrefix("2001:0020::/28"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("3fff::/20"),
		netip.MustParsePrefix("5f00::/16"),
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return true, "special-use, documentation or benchmarking IP range"
		}
	}
	return false, ""
}

// EmbeddedIPv4 extrai o IPv4 transportado por prefixos de transição conhecidos:
// 6to4 (2002::/16, RFC 3056), NAT64 well-known (64:ff9b::/96, RFC 6052) e NAT64
// local-use (64:ff9b:1::/48, RFC 8215). Devolve ok=false para qualquer outro
// endereço, incluindo IPv4 nativo e IPv4-mapped (esse último já é resolvido por
// netip.Addr.Unmap).
func EmbeddedIPv4(address netip.Addr) (net.IP, bool) {
	if !address.Is6() || address.Is4In6() {
		return nil, false
	}
	raw := address.As16()
	// 6to4: os 32 bits seguintes ao prefixo carregam o IPv4 de destino.
	if raw[0] == 0x20 && raw[1] == 0x02 {
		return net.IPv4(raw[2], raw[3], raw[4], raw[5]), true
	}
	// NAT64 well-known 64:ff9b::/96 e local-use 64:ff9b:1::/48.
	if raw[0] == 0x00 && raw[1] == 0x64 && raw[2] == 0xff && raw[3] == 0x9b {
		localUse := raw[4] == 0x00 && raw[5] == 0x01
		wellKnown := true
		for index := 4; index < 12; index++ {
			if raw[index] != 0x00 {
				wellKnown = false
				break
			}
		}
		if wellKnown || localUse {
			return net.IPv4(raw[12], raw[13], raw[14], raw[15]), true
		}
	}
	return nil, false
}

func classifyStandard(address netip.Addr) string {
	switch {
	case address.IsLoopback():
		return "loopback address"
	case address.IsPrivate():
		return "private network address"
	case address.IsLinkLocalUnicast(), address.IsLinkLocalMulticast():
		return "link-local address"
	case address.IsUnspecified():
		return "unspecified address"
	case address.IsMulticast():
		return "multicast address"
	default:
		return "restricted or non-global IP address"
	}
}
