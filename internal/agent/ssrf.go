package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"
)

// unsafeIP reports whether an address must never be dialed by an outbound agent
// fetch (browser, research, media, connectors). Kept in one place so every
// egress path shares the same block list.
func unsafeIP(ip net.IP) bool {
	blocked, _ := ClassifyEgressIP(ip)
	return blocked
}

// resolvePublicIPs resolves host and returns its addresses, rejecting the whole
// host if any resolved address is private/loopback/link-local. Returning on the
// first unsafe address (rather than filtering it out) prevents a DNS response
// that mixes a public and a private record from being partially accepted.
func resolvePublicIPs(ctx context.Context, host string) ([]net.IP, error) {
	if literal := net.ParseIP(host); literal != nil {
		if unsafeIP(literal) {
			return nil, errors.New("host is a private or local address")
		}
		return []net.IP{literal}, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("host resolved to no addresses")
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if unsafeIP(a.IP) {
			return nil, errors.New("host resolves to a private or local address")
		}
		ips = append(ips, append(net.IP(nil), a.IP...))
	}
	return ips, nil
}

// pinnedClient returns an *http.Client that will only connect to the supplied
// URL's host by dialing the exact IPs validated at check time. This closes the
// DNS-rebinding / TOCTOU window: the address cannot re-resolve to an internal
// target between validation and the actual dial. Redirects are disabled so
// every hop is bound to the same validated host.
func pinnedClient(ctx context.Context, base *http.Client, rawURL string) (*http.Client, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return nil, errors.New("invalid URL")
	}
	approved, err := resolvePublicIPs(ctx, parsed.Hostname())
	if err != nil {
		return nil, err
	}
	var source http.RoundTripper = http.DefaultTransport
	if base != nil && base.Transport != nil {
		source = base.Transport
	}
	baseTransport, ok := source.(*http.Transport)
	if !ok {
		return nil, errors.New("transport cannot enforce destination pinning")
	}
	transport := baseTransport.Clone()
	transport.Proxy = nil
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	dial := transport.DialContext
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range approved {
			conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errors.New("no approved address could be dialed")
		}
		return nil, lastErr
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("redirects are disabled for pinned fetches")
		},
	}
	if base != nil && base.Timeout > 0 {
		client.Timeout = base.Timeout
	}
	return client, nil
}
