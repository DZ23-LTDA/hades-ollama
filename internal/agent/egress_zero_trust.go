package agent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrEgressBlockedPrivateIP        = errors.New("egress destination resolves to a private or restricted IP address")
	ErrEgressBlockedMetadata         = errors.New("egress destination resolves to cloud metadata service (169.254.169.254)")
	ErrEgressBlockedLoopback         = errors.New("egress destination resolves to loopback address")
	ErrEgressBlockedLinkLocal        = errors.New("egress destination resolves to link-local address")
	ErrEgressBlockedDNSRebind        = errors.New("egress destination DNS response contains restricted addresses (DNS rebinding attempt)")
	ErrEgressRedirectDisallowed      = errors.New("egress redirect to unapproved host is blocked")
	ErrEgressCredentialLeakPrevented = errors.New("egress credential leak to untrusted host was prevented")
	ErrEgressPayloadExceedsLimit     = errors.New("egress response payload exceeds maximum allowed size")
)

// ClassifyEgressIP rigorously inspects an IP address and determines whether it
// is safe for outbound communication. Blocks loopback, private RFC 1918, RFC 4193,
// link-local (RFC 3927), cloud metadata (169.254.169.254), Carrier-Grade NAT (RFC 6598),
// multicast, broadcast and unspecified addresses.
// Also safely unwraps IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1).
func ClassifyEgressIP(ip net.IP) (blocked bool, reason string) {
	if ip == nil {
		return true, "nil IP address"
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	if !unsafeEgressIP(ip) {
		return false, ""
	}

	// Determine specific human-readable reason for audit log
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true, "cloud metadata service address (169.254.169.254)"
	}
	if ip.IsLoopback() {
		return true, "loopback address"
	}
	if ip.IsPrivate() {
		return true, "private network address"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true, "link-local address"
	}
	if ip.IsUnspecified() {
		return true, "unspecified (0.0.0.0) address"
	}
	if ip.IsMulticast() {
		return true, "multicast address"
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && (v4[1]&0xc0) == 64 {
			return true, "carrier-grade NAT address (100.64.0.0/10)"
		}
		if v4[0] == 255 && v4[1] == 255 && v4[2] == 255 && v4[3] == 255 {
			return true, "broadcast address"
		}
	}
	return true, "restricted or non-global IP address"
}

// ResolveAllPublicIPs resolves all IP addresses for a host, rejecting the entire host
// if ANY resolved address is blocked (preventing DNS rebinding / mixed-record attacks).
func ResolveAllPublicIPs(ctx context.Context, host string) ([]net.IP, error) {
	return ResolveAllPublicIPsWithResolver(ctx, host, func(ctx context.Context, host string) ([]net.IP, error) {
		return net.DefaultResolver.LookupIP(ctx, "ip", host)
	})
}

// ResolveAllPublicIPsWithResolver allows injecting a custom resolver for testing.
func ResolveAllPublicIPsWithResolver(ctx context.Context, host string, lookup func(context.Context, string) ([]net.IP, error)) ([]net.IP, error) {
	host = strings.Trim(host, "[]")
	if literal := net.ParseIP(host); literal != nil {
		if blocked, reason := ClassifyEgressIP(literal); blocked {
			return nil, fmt.Errorf("%w: %s (%s)", ErrEgressBlockedPrivateIP, literal, reason)
		}
		return []net.IP{literal}, nil
	}

	if lookup == nil {
		return nil, errors.New("egress destination resolver is unavailable")
	}

	addrs, err := lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("egress destination lookup failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, errors.New("egress destination resolved to zero addresses")
	}

	approved := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if blocked, reason := ClassifyEgressIP(addr); blocked {
			return nil, fmt.Errorf("%w: host %q resolved to %s (%s)", ErrEgressBlockedDNSRebind, host, addr, reason)
		}
		approved = append(approved, append(net.IP(nil), addr...))
	}
	return approved, nil
}

// EgressDecision represents an audited egress access decision.
type EgressDecision struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Callsite    string    `json:"callsite"`
	Method      string    `json:"method"`
	Destination string    `json:"destination"`
	Host        string    `json:"host"`
	ResolvedIPs []string  `json:"resolved_ips"`
	Allowed     bool      `json:"allowed"`
	Reason      string    `json:"reason"`
	LatencyMS   int64     `json:"latency_ms"`
}

// EgressAuditStore is a thread-safe ring buffer for egress decisions.
type EgressAuditStore struct {
	mu      sync.RWMutex
	entries []EgressDecision
	maxSize int
}

// NewEgressAuditStore creates an audit store with a maximum capacity.
func NewEgressAuditStore(maxSize int) *EgressAuditStore {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &EgressAuditStore{
		entries: make([]EgressDecision, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record appends a decision to the audit log.
func (s *EgressAuditStore) Record(decision EgressDecision) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if decision.Timestamp.IsZero() {
		decision.Timestamp = time.Now()
	}
	if decision.ID == "" {
		decision.ID = fmt.Sprintf("egr-%d-%d", decision.Timestamp.UnixNano(), len(s.entries)+1)
	}
	if len(s.entries) >= s.maxSize {
		s.entries = append(s.entries[1:], decision)
	} else {
		s.entries = append(s.entries, decision)
	}
}

// List returns the latest n decisions.
func (s *EgressAuditStore) List(limit int) []EgressDecision {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.entries) {
		limit = len(s.entries)
	}
	result := make([]EgressDecision, limit)
	copy(result, s.entries[len(s.entries)-limit:])
	return result
}

// Filter returns decisions matching a specific callsite.
func (s *EgressAuditStore) Filter(callsite string, limit int) []EgressDecision {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var filtered []EgressDecision
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].Callsite == callsite {
			filtered = append(filtered, s.entries[i])
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}
	return filtered
}

// Clear resets the audit buffer.
func (s *EgressAuditStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
}

// DefaultEgressAuditStore is the shared global auditor.
var DefaultEgressAuditStore = NewEgressAuditStore(1000)

// EgressOptions configures egress behavior.
type EgressOptions struct {
	Callsite      string
	Timeout       time.Duration
	AllowLoopback bool
	MaxBodyBytes  int64
	Lookup        func(context.Context, string) ([]net.IP, error)
}

// NewEgressTransport returns an *http.Transport enforcing zero-trust egress:
// - no environmental proxy
// - no TLS hook bypass
// - DNS resolution and IP pinning
// - peer address verification
// - audit logging of all decisions
func NewEgressTransport(opts EgressOptions) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		start := time.Now()
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			DefaultEgressAuditStore.Record(EgressDecision{
				Timestamp:   start,
				Callsite:    opts.Callsite,
				Destination: address,
				Allowed:     false,
				Reason:      "invalid destination address: " + err.Error(),
			})
			return nil, err
		}
		host = strings.Trim(host, "[]")

		var ips []net.IP
		if literal := net.ParseIP(host); literal != nil {
			ips = []net.IP{literal}
		} else {
			lookup := opts.Lookup
			if lookup == nil {
				lookup = func(c context.Context, h string) ([]net.IP, error) {
					return net.DefaultResolver.LookupIP(c, "ip", h)
				}
			}
			addrs, err := lookup(ctx, host)
			if err != nil {
				DefaultEgressAuditStore.Record(EgressDecision{
					Timestamp:   start,
					Callsite:    opts.Callsite,
					Destination: address,
					Host:        host,
					Allowed:     false,
					Reason:      "DNS lookup failed: " + err.Error(),
				})
				return nil, err
			}
			ips = addrs
		}

		if len(ips) == 0 {
			DefaultEgressAuditStore.Record(EgressDecision{
				Timestamp:   start,
				Callsite:    opts.Callsite,
				Destination: address,
				Host:        host,
				Allowed:     false,
				Reason:      "destination resolved to zero addresses",
			})
			return nil, errors.New("egress destination has no addresses")
		}

		ipStrings := make([]string, len(ips))
		for i, ip := range ips {
			ipStrings[i] = ip.String()
			blocked, reason := ClassifyEgressIP(ip)
			if blocked {
				if opts.AllowLoopback && ip.IsLoopback() {
					continue
				}
				DefaultEgressAuditStore.Record(EgressDecision{
					Timestamp:   start,
					Callsite:    opts.Callsite,
					Destination: address,
					Host:        host,
					ResolvedIPs: ipStrings,
					Allowed:     false,
					Reason:      fmt.Sprintf("blocked restricted IP %s (%s)", ip.String(), reason),
				})
				return nil, fmt.Errorf("%w: %s (%s)", ErrEgressBlockedPrivateIP, ip.String(), reason)
			}
		}

		var lastErr error
		for _, ip := range ips {
			target := net.JoinHostPort(ip.String(), port)
			conn, err := dialer.DialContext(ctx, network, target)
			if err == nil {
				// Verify the peer address connected matches an approved IP
				if remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
					if blocked, reason := ClassifyEgressIP(remoteAddr.IP); blocked && !(opts.AllowLoopback && remoteAddr.IP.IsLoopback()) {
						conn.Close()
						DefaultEgressAuditStore.Record(EgressDecision{
							Timestamp:   start,
							Callsite:    opts.Callsite,
							Destination: address,
							Host:        host,
							ResolvedIPs: ipStrings,
							Allowed:     false,
							Reason:      fmt.Sprintf("peer connection hijacked to restricted IP %s (%s)", remoteAddr.IP.String(), reason),
						})
						return nil, fmt.Errorf("%w: connected peer %s is restricted", ErrEgressBlockedPrivateIP, remoteAddr.IP.String())
					}
				}
				DefaultEgressAuditStore.Record(EgressDecision{
					Timestamp:   start,
					Callsite:    opts.Callsite,
					Destination: address,
					Host:        host,
					ResolvedIPs: ipStrings,
					Allowed:     true,
					Reason:      "approved public destination",
					LatencyMS:   time.Since(start).Milliseconds(),
				})
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	return transport
}

// StripSensitiveEgressHeaders removes sensitive credentials from an outbound request.
func StripSensitiveEgressHeaders(req *http.Request) {
	if req == nil || req.Header == nil {
		return
	}
	sensitiveHeaders := []string{
		"Authorization",
		"Proxy-Authorization",
		"Cookie",
		"Set-Cookie",
		"X-Api-Key",
		"X-Auth-Token",
		"X-Session-Token",
		"Private-Token",
		"Apikey",
	}
	for _, h := range sensitiveHeaders {
		req.Header.Del(h)
	}
}

// NewEgressCheckRedirect returns a CheckRedirect function enforcing:
// 1. Host pinning: redirect target host must match initial request host.
// 2. HTTPS-to-HTTP downgrade prevention: redirects from HTTPS to HTTP are rejected.
// 3. Credential isolation: on any redirect, sensitive headers are stripped before dispatch.
// 4. IP inspection of redirect target host.
func NewEgressCheckRedirect(callsite string, allowLoopback bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		originHost := via[0].URL.Hostname()
		targetHost := req.URL.Hostname()

		// Credential isolation: strip sensitive headers on redirect attempt
		StripSensitiveEgressHeaders(req)

		// 1. Host Pinning: reject cross-host redirects
		if !strings.EqualFold(originHost, targetHost) {
			DefaultEgressAuditStore.Record(EgressDecision{
				Timestamp:   time.Now(),
				Callsite:    callsite,
				Method:      req.Method,
				Destination: req.URL.String(),
				Host:        targetHost,
				Allowed:     false,
				Reason:      fmt.Sprintf("cross-host redirect from %q to unapproved host %q blocked", originHost, targetHost),
			})
			return fmt.Errorf("%w: redirect from %s to %s", ErrEgressRedirectDisallowed, originHost, targetHost)
		}

		// 2. HTTPS to HTTP downgrade prevention
		if via[0].URL.Scheme == "https" && req.URL.Scheme == "http" {
			DefaultEgressAuditStore.Record(EgressDecision{
				Timestamp:   time.Now(),
				Callsite:    callsite,
				Method:      req.Method,
				Destination: req.URL.String(),
				Host:        targetHost,
				Allowed:     false,
				Reason:      "HTTPS to HTTP downgrade redirect blocked",
			})
			return errors.New("egress HTTPS to HTTP downgrade redirect is blocked")
		}

		// 3. Verify redirect destination IP is not restricted
		if _, err := ResolveAllPublicIPs(req.Context(), targetHost); err != nil {
			if !(allowLoopback && strings.EqualFold(targetHost, "localhost")) {
				DefaultEgressAuditStore.Record(EgressDecision{
					Timestamp:   time.Now(),
					Callsite:    callsite,
					Method:      req.Method,
					Destination: req.URL.String(),
					Host:        targetHost,
					Allowed:     false,
					Reason:      fmt.Sprintf("redirect target host %q resolves to restricted IP: %v", targetHost, err),
				})
				return fmt.Errorf("redirect target resolves to restricted IP: %w", err)
			}
		}

		return nil
	}
}

// NewSafeEgressHTTPClient constructs a fully hardened http.Client with zero-trust egress.
func NewSafeEgressHTTPClient(opts EgressOptions) *http.Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := NewEgressTransport(opts)
	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: NewEgressCheckRedirect(opts.Callsite, opts.AllowLoopback),
	}
}

// ReadBoundedBody safely reads response bytes up to maxBytes, returning ErrEgressPayloadExceedsLimit
// if the stream exceeds the limit, preventing memory exhaustion attacks.
func ReadBoundedBody(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 2 << 20 // 2MB default
	}
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: read exceeded limit of %d bytes", ErrEgressPayloadExceedsLimit, maxBytes)
	}
	return data, nil
}
