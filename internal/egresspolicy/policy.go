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
