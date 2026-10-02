package multillm

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/internal/egresspolicy"
)

var (
	ErrProviderBlockedPrivateIP   = errors.New("provider destination resolves to private or restricted IP address")
	ErrProviderBlockedMetadata    = errors.New("provider destination resolves to cloud metadata service (169.254.169.254)")
	ErrProviderBlockedDNSRebind   = errors.New("provider DNS resolution contains restricted addresses (DNS rebinding attempt)")
	ErrProviderRedirectDisallowed = errors.New("provider redirect to unapproved host is blocked")
)

// ClassifyProviderEgressIP determines whether an IP is blocked for outbound multillm traffic.
func ClassifyProviderEgressIP(ip net.IP) (blocked bool, reason string) {
	return egresspolicy.ClassifyIP(ip)
}

// ProviderEgressDecision records an audited decision for multi-provider egress.
type ProviderEgressDecision struct {
	Timestamp   time.Time `json:"timestamp"`
	Provider    string    `json:"provider"`
	Destination string    `json:"destination"`
	Host        string    `json:"host"`
	ResolvedIPs []string  `json:"resolved_ips"`
	Allowed     bool      `json:"allowed"`
	Reason      string    `json:"reason"`
}

// ProviderEgressAuditStore tracks provider egress decisions.
type ProviderEgressAuditStore struct {
	mu      sync.RWMutex
	entries []ProviderEgressDecision
	maxSize int
}

func NewProviderEgressAuditStore(maxSize int) *ProviderEgressAuditStore {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &ProviderEgressAuditStore{
		entries: make([]ProviderEgressDecision, 0, maxSize),
		maxSize: maxSize,
	}
}

func (s *ProviderEgressAuditStore) Record(decision ProviderEgressDecision) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if decision.Timestamp.IsZero() {
		decision.Timestamp = time.Now()
	}
	if len(s.entries) >= s.maxSize {
		s.entries = append(s.entries[1:], decision)
	} else {
		s.entries = append(s.entries, decision)
	}
}

func (s *ProviderEgressAuditStore) List(limit int) []ProviderEgressDecision {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.entries) {
		limit = len(s.entries)
	}
	result := make([]ProviderEgressDecision, limit)
	copy(result, s.entries[len(s.entries)-limit:])
	return result
}

var DefaultProviderEgressAuditor = NewProviderEgressAuditStore(500)

// ResolveProviderPublicIPs resolves all IPs for a provider host, verifying none are restricted.
func ResolveProviderPublicIPs(ctx context.Context, host string) ([]net.IP, error) {
	return ResolveProviderPublicIPsWithResolver(ctx, host, func(c context.Context, h string) ([]net.IP, error) {
		return net.DefaultResolver.LookupIP(c, "ip", h)
	})
}

// ResolveProviderPublicIPsWithResolver allows custom DNS resolver injection for tests.
func ResolveProviderPublicIPsWithResolver(ctx context.Context, host string, lookup func(context.Context, string) ([]net.IP, error)) ([]net.IP, error) {
	host = strings.Trim(host, "[]")
	if literal := net.ParseIP(host); literal != nil {
		if blocked, reason := ClassifyProviderEgressIP(literal); blocked {
			return nil, fmt.Errorf("%w: %s (%s)", ErrProviderBlockedPrivateIP, literal, reason)
		}
		return []net.IP{literal}, nil
	}

	if lookup == nil {
		return nil, errors.New("resolver is unavailable")
	}

	addrs, err := lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, errors.New("host resolved to zero addresses")
	}

	approved := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if blocked, reason := ClassifyProviderEgressIP(addr); blocked {
			return nil, fmt.Errorf("%w: host %q resolved to %s (%s)", ErrProviderBlockedDNSRebind, host, addr, reason)
		}
		approved = append(approved, append(net.IP(nil), addr...))
	}
	return approved, nil
}

// StripProviderCredentials removes Authorization and API key headers on redirect.
func StripProviderCredentials(req *http.Request) {
	if req == nil || req.Header == nil {
		return
	}
	req.Header.Del("Authorization")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("X-Api-Key")
	req.Header.Del("Api-Key")
	req.Header.Del("Cookie")
}

// CheckProviderRedirectPolicy blocks cross-host redirects and HTTPS-to-HTTP downgrades.
func CheckProviderRedirectPolicy(approvedHost string) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		originHost := via[0].URL.Hostname()
		targetHost := req.URL.Hostname()

		StripProviderCredentials(req)

		if !strings.EqualFold(originHost, targetHost) {
			DefaultProviderEgressAuditor.Record(ProviderEgressDecision{
				Timestamp:   time.Now(),
				Destination: req.URL.String(),
				Host:        targetHost,
				Allowed:     false,
				Reason:      fmt.Sprintf("cross-host redirect from %q to %q blocked", originHost, targetHost),
			})
			return fmt.Errorf("%w: cross-host redirect from %s to %s", ErrProviderRedirectDisallowed, originHost, targetHost)
		}

		if via[0].URL.Scheme == "https" && req.URL.Scheme == "http" {
			DefaultProviderEgressAuditor.Record(ProviderEgressDecision{
				Timestamp:   time.Now(),
				Destination: req.URL.String(),
				Host:        targetHost,
				Allowed:     false,
				Reason:      "HTTPS to HTTP downgrade redirect blocked",
			})
			return errors.New("provider HTTPS to HTTP downgrade redirect is blocked")
		}

		return nil
	}
}

// ValidateProviderEgressURL verifies a provider base URL before registration or egress.
func ValidateProviderEgressURL(rawURL string, allowPrivate bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return errors.New("invalid provider URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("invalid provider URL scheme: only http and https are allowed")
	}
	if allowPrivate {
		return nil
	}
	ips, err := ResolveProviderPublicIPs(context.Background(), parsed.Hostname())
	if err != nil {
		DefaultProviderEgressAuditor.Record(ProviderEgressDecision{
			Timestamp:   time.Now(),
			Destination: rawURL,
			Host:        parsed.Hostname(),
			Allowed:     false,
			Reason:      err.Error(),
		})
		return err
	}
	ipStrs := make([]string, len(ips))
	for i, ip := range ips {
		ipStrs[i] = ip.String()
	}
	DefaultProviderEgressAuditor.Record(ProviderEgressDecision{
		Timestamp:   time.Now(),
		Destination: rawURL,
		Host:        parsed.Hostname(),
		ResolvedIPs: ipStrs,
		Allowed:     true,
		Reason:      "approved provider public destination",
	})
	return nil
}

// NewProviderSafeHTTPClient returns a hardened HTTP client for provider communications.
func NewProviderSafeHTTPClient(timeout time.Duration, allowPrivate bool) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		host = strings.Trim(host, "[]")
		var ips []net.IP
		if literal := net.ParseIP(host); literal != nil {
			ips = []net.IP{literal}
		} else {
			addrs, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			ips = addrs
		}
		for _, ip := range ips {
			if !allowPrivate {
				if blocked, reason := ClassifyProviderEgressIP(ip); blocked {
					return nil, fmt.Errorf("%w: %s (%s)", ErrProviderBlockedPrivateIP, ip.String(), reason)
				}
			}
		}
		var lastErr error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}

	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: CheckProviderRedirectPolicy(""),
	}
}
