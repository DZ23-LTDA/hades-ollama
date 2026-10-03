package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCompanionSecureRequestForwardedProtoSpoofing proves SEC-11: the
// X-Forwarded-Proto header is only honored from a trusted proxy. A direct,
// non-TLS client cannot spoof it to bypass the TLS requirement.
func TestCompanionSecureRequestForwardedProtoSpoofing(t *testing.T) {
	newReq := func(remoteAddr, forwardedProto string, tlsState *tls.ConnectionState) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/devices/abc/connect", nil)
		req.RemoteAddr = remoteAddr
		if forwardedProto != "" {
			req.Header.Set("X-Forwarded-Proto", forwardedProto)
		}
		req.TLS = tlsState
		return req
	}

	// Untrusted remote spoofing X-Forwarded-Proto: must be treated as insecure.
	if companionSecureRequest(newReq("203.0.113.9:5555", "https", nil)) {
		t.Fatal("X-Forwarded-Proto from an untrusted remote must not be trusted")
	}

	// Loopback proxy with X-Forwarded-Proto=https: accepted.
	if !companionSecureRequest(newReq("127.0.0.1:5555", "https", nil)) {
		t.Fatal("X-Forwarded-Proto=https from loopback proxy should be accepted")
	}
	if !companionSecureRequest(newReq("[::1]:5555", "https", nil)) {
		t.Fatal("X-Forwarded-Proto=https from IPv6 loopback proxy should be accepted")
	}

	// Direct TLS connection: always secure regardless of headers.
	if !companionSecureRequest(newReq("203.0.113.9:5555", "", &tls.ConnectionState{})) {
		t.Fatal("a direct TLS request must be treated as secure")
	}

	// No TLS and no forwarded header from loopback: still insecure.
	if companionSecureRequest(newReq("127.0.0.1:5555", "", nil)) {
		t.Fatal("loopback without TLS or X-Forwarded-Proto=https must be insecure")
	}
}

// TestCompanionSecureRequestTrustedProxyAllowlist proves that an explicit
// allowlist entry (OLLAMA_TRUSTED_PROXIES) enables X-Forwarded-Proto trust for a
// non-loopback proxy, by exact IP and by CIDR range.
func TestCompanionSecureRequestTrustedProxyAllowlist(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/devices/abc/connect", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	req.Header.Set("X-Forwarded-Proto", "https")

	// Without allowlist, not trusted.
	if companionSecureRequest(req) {
		t.Fatal("remote must not be trusted before being allowlisted")
	}

	t.Setenv("OLLAMA_TRUSTED_PROXIES", "198.51.100.1, 203.0.113.9")
	if !companionSecureRequest(req) {
		t.Fatal("exact allowlisted proxy IP should be trusted")
	}

	t.Setenv("OLLAMA_TRUSTED_PROXIES", "203.0.113.0/24")
	if !companionSecureRequest(req) {
		t.Fatal("proxy within an allowlisted CIDR should be trusted")
	}

	// A remote outside the allowlisted CIDR stays untrusted.
	other := httptest.NewRequest(http.MethodGet, "/devices/abc/connect", nil)
	other.RemoteAddr = "198.51.100.42:5555"
	other.Header.Set("X-Forwarded-Proto", "https")
	if companionSecureRequest(other) {
		t.Fatal("remote outside the allowlisted CIDR must not be trusted")
	}
}
