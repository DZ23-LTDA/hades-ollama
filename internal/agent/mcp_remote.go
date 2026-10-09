package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var errRemoteMCPIDMismatch = errors.New("remote MCP response id does not match request")

var lookupRemoteMCPIPs = func(ctx context.Context, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

type RemoteMCPServerConfig struct {
	ID              string                `json:"id"`
	OrganizationID  string                `json:"organization_id,omitempty"`
	URL             string                `json:"url"`
	TokenEnv        string                `json:"token_env,omitempty"`
	HeadersEnv      map[string]string     `json:"headers_env,omitempty"`
	AllowedMethods  []string              `json:"allowed_methods,omitempty"`
	TimeoutSeconds  int                   `json:"timeout_seconds,omitempty"`
	Disabled        bool                  `json:"disabled,omitempty"`
	OAuth           *RemoteMCPOAuthConfig `json:"oauth,omitempty"`
	PairingTokenEnv string                `json:"pairing_token_env,omitempty"`
}

type RemoteMCPOAuthConfig struct {
	AuthorizationURL string   `json:"authorization_url"`
	TokenURL         string   `json:"token_url"`
	ClientIDEnv      string   `json:"client_id_env"`
	ClientSecretEnv  string   `json:"client_secret_env,omitempty"`
	RedirectURI      string   `json:"redirect_uri"`
	Scopes           []string `json:"scopes,omitempty"`
}

type remoteMCPOAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type remoteMCPAuthState struct {
	ServerID       string
	OrganizationID string
	Verifier       string
	State          string
	ExpiresAt      time.Time
}

type remoteMCPSession struct {
	ID             string
	ServerID       string
	OrganizationID string
	ExpiresAt      time.Time
	LastEventID    string
}

type remoteMCPPairing struct {
	Challenge string
	ExpiresAt time.Time
}

var (
	ErrRemoteMCPNotConfigured       = errors.New("remote MCP is NOT_CONFIGURED")
	ErrRemoteMCPSessionInvalid      = errors.New("remote MCP session is invalid or expired")
	ErrRemoteMCPPairingUnauthorized = errors.New("remote MCP pairing is unauthorized")
	ErrRemoteMCPOAuthStateInvalid   = errors.New("remote MCP OAuth state is invalid or expired")
)

type RemoteMCPManager struct {
	mu                  sync.RWMutex
	client              *http.Client
	servers             map[string]RemoteMCPServerConfig
	nextID              int64
	persistPath         string
	oauthTokens         map[string]remoteMCPOAuthToken
	oauthStates         map[string]remoteMCPAuthState
	sessions            map[string]remoteMCPSession
	pairings            map[string]remoteMCPPairing
	organizationSecrets map[string]map[string]string
}

func NewRemoteMCPManager() *RemoteMCPManager {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = remoteMCPDialContext
	return &RemoteMCPManager{
		client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) == 0 {
					return nil
				}
				previous := via[len(via)-1].URL
				if !sameRemoteMCPOrigin(previous, req.URL) {
					return errors.New("remote MCP redirect changes origin")
				}
				if !remoteMCPURLAllowed(req.URL) {
					return errors.New("remote MCP redirect target is not allowed")
				}
				return nil
			},
		},
		servers:             map[string]RemoteMCPServerConfig{},
		oauthTokens:         map[string]remoteMCPOAuthToken{},
		oauthStates:         map[string]remoteMCPAuthState{},
		sessions:            map[string]remoteMCPSession{},
		pairings:            make(map[string]remoteMCPPairing),
		organizationSecrets: make(map[string]map[string]string),
	}
}

// SetOrganizationSecret binds MCP token/header material to one tenant.
func (m *RemoteMCPManager) SetOrganizationSecret(organizationID, name, value string) error {
	organizationID = strings.TrimSpace(organizationID)
	name = normalizeSecretName(name)
	if organizationID == "" || name == "" || strings.TrimSpace(value) == "" {
		return errors.New("organization, secret name and value are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.organizationSecrets[organizationID] == nil {
		m.organizationSecrets[organizationID] = make(map[string]string)
	}
	m.organizationSecrets[organizationID][name] = value
	return nil
}

func (m *RemoteMCPManager) organizationSecret(organizationID, name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.organizationSecrets[strings.TrimSpace(organizationID)][normalizeSecretName(name)]
}

func (m *RemoteMCPManager) resolveSecret(organizationID, name string, global bool) (string, bool) {
	if global {
		return os.LookupEnv(name)
	}
	value := m.organizationSecret(organizationID, name)
	return value, strings.TrimSpace(value) != ""
}

func NewPersistentRemoteMCPManager(manifestPath string) (*RemoteMCPManager, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	manager := NewRemoteMCPManager()
	if manifestPath == "" {
		return manager, nil
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		var configs []RemoteMCPServerConfig
		if err := decoder.Decode(&configs); err != nil {
			return nil, fmt.Errorf("decode remote MCP manifest: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return nil, errors.New("remote MCP manifest contains trailing JSON")
			}
			return nil, fmt.Errorf("decode remote MCP manifest trailing data: %w", err)
		}
		for _, config := range configs {
			if err := manager.Register(config); err != nil {
				return nil, fmt.Errorf("load remote MCP server %q: %w", config.ID, err)
			}
		}
	}
	manager.persistPath = manifestPath
	return manager, nil
}

func (m *RemoteMCPManager) Register(config RemoteMCPServerConfig) error {
	return m.register(config, "")
}

func (m *RemoteMCPManager) RegisterForOrganization(organizationID string, config RemoteMCPServerConfig) error {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return errors.New("remote MCP organization scope is required")
	}
	if supplied := strings.TrimSpace(config.OrganizationID); supplied != "" && supplied != organizationID {
		return ErrPluginOrganizationScope
	}
	config.OrganizationID = organizationID
	return m.register(config, organizationID)
}

func (m *RemoteMCPManager) register(config RemoteMCPServerConfig, organizationID string) error {
	config.ID = strings.TrimSpace(config.ID)
	config.OrganizationID = strings.TrimSpace(config.OrganizationID)
	config.URL = strings.TrimSpace(config.URL)
	if config.ID == "" || config.URL == "" {
		return errors.New("remote MCP id and url are required")
	}
	parsed, err := url.Parse(config.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("remote MCP url must be an absolute URL without credentials or fragment")
	}
	if err := validateConfiguredEndpointURL(config.URL); err != nil {
		return fmt.Errorf("remote MCP url is not safe to persist: %w", err)
	}
	if !remoteMCPURLAllowed(parsed) {
		return errors.New("remote MCP requires HTTPS outside loopback")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && unsafeEgressIP(ip) && !ip.IsLoopback() {
		return errors.New("remote MCP destination cannot be a private address")
	}
	if config.TokenEnv != "" && !validEnvName(config.TokenEnv) {
		return errors.New("remote MCP token_env is invalid")
	}
	if config.PairingTokenEnv != "" && !validEnvName(config.PairingTokenEnv) {
		return errors.New("remote MCP pairing_token_env is invalid")
	}
	if config.OAuth != nil {
		oauth := *config.OAuth
		if err := validateConfiguredEndpointURL(oauth.AuthorizationURL); err != nil {
			return fmt.Errorf("remote MCP OAuth authorization URL is unsafe: %w", err)
		}
		if err := validateConfiguredEndpointURL(oauth.TokenURL); err != nil {
			return fmt.Errorf("remote MCP OAuth token URL is unsafe: %w", err)
		}
		authURL, authErr := url.Parse(oauth.AuthorizationURL)
		tokenURL, tokenErr := url.Parse(oauth.TokenURL)
		if authErr != nil || tokenErr != nil || !remoteMCPURLAllowed(authURL) || !remoteMCPURLAllowed(tokenURL) {
			return errors.New("remote MCP OAuth endpoints require HTTPS outside loopback")
		}
		if !validEnvName(oauth.ClientIDEnv) || (oauth.ClientSecretEnv != "" && !validEnvName(oauth.ClientSecretEnv)) || strings.TrimSpace(oauth.RedirectURI) == "" {
			return errors.New("remote MCP OAuth client environment and redirect URI are required")
		}
		config.OAuth = &oauth
	}
	for header, envName := range config.HeadersEnv {
		if !validRemoteMCPHeaderName(header) || !validEnvName(envName) {
			return errors.New("remote MCP header name is invalid")
		}
	}
	if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > 300 {
		config.TimeoutSeconds = 30
	}
	allowed := make([]string, 0, len(config.AllowedMethods))
	seenMethods := make(map[string]struct{}, len(config.AllowedMethods))
	for _, method := range config.AllowedMethods {
		method = strings.TrimSpace(method)
		if method == "" {
			continue
		}
		if _, exists := seenMethods[method]; exists {
			continue
		}
		seenMethods[method] = struct{}{}
		allowed = append(allowed, method)
	}
	if len(allowed) == 0 {
		return errors.New("remote MCP allowed_methods must contain at least one non-empty method")
	}
	config.AllowedMethods = allowed
	config.HeadersEnv = mapsClone(config.HeadersEnv)
	m.mu.Lock()
	defer m.mu.Unlock()
	if organizationID != "" {
		if existing, ok := m.servers[config.ID]; ok && !pluginOwnedByOrganization(existing.OrganizationID, organizationID) {
			return ErrPluginOrganizationScope
		}
	}
	previous, existed := m.servers[config.ID]
	m.servers[config.ID] = config
	if err := m.persistLocked(); err != nil {
		if existed {
			m.servers[config.ID] = previous
		} else {
			delete(m.servers, config.ID)
		}
		return fmt.Errorf("persist remote MCP manifest: %w", err)
	}
	return nil
}

func (m *RemoteMCPManager) persistLocked() error {
	if strings.TrimSpace(m.persistPath) == "" {
		return nil
	}
	configs := make([]RemoteMCPServerConfig, 0, len(m.servers))
	for _, config := range m.servers {
		config.HeadersEnv = mapsClone(config.HeadersEnv)
		config.AllowedMethods = append([]string(nil), config.AllowedMethods...)
		configs = append(configs, config)
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	if err := os.MkdirAll(filepath.Dir(m.persistPath), 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(m.persistPath, configs)
}

func remoteMCPLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type remoteMCPLoopbackContextKey struct{}

type remoteMCPApprovedIPsContextKey struct{}

func remoteMCPDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return remoteMCPDialContextWithResolver(ctx, network, address, lookupRemoteMCPIPs)
}

func remoteMCPDialContextWithResolver(ctx context.Context, network, address string, lookup func(context.Context, string) ([]net.IP, error)) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("remote MCP destination is invalid")
	}
	if approved, pinned := ctx.Value(remoteMCPApprovedIPsContextKey{}).([]net.IP); pinned {
		if len(approved) == 0 {
			return nil, errors.New("remote MCP destination has no approved addresses")
		}
		var lastErr error
		for _, ip := range approved {
			if ip == nil || (!remoteMCPLoopbackContext(ctx) && unsafeEgressIP(ip)) {
				continue
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr != nil {
				lastErr = dialErr
				continue
			}
			remote, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
			connected := net.ParseIP(strings.Trim(remote, "[]"))
			if splitErr != nil || connected == nil || !remoteMCPContainsIP(approved, connected) {
				_ = conn.Close()
				lastErr = errors.New("remote MCP connected address was not approved")
				continue
			}
			return conn, nil
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, errors.New("remote MCP could not connect to an approved address")
	}
	if lookup == nil {
		return nil, errors.New("remote MCP destination resolver is unavailable")
	}
	addresses, err := lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, errors.New("remote MCP destination has no addresses")
	}
	allowLoopback := remoteMCPLoopbackContext(ctx)
	for _, ip := range addresses {
		if ip == nil || (allowLoopback && !ip.IsLoopback()) {
			return nil, errors.New("remote MCP loopback destination resolves outside loopback")
		}
		if !allowLoopback && unsafeEgressIP(ip) {
			return nil, errors.New("remote MCP destination resolves to a private address")
		}
	}
	var lastErr error
	for _, ip := range addresses {
		if network == "tcp4" && ip.To4() == nil || network == "tcp6" && ip.To4() != nil {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr != nil {
			lastErr = dialErr
			continue
		}
		remote, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		connected := net.ParseIP(strings.Trim(remote, "[]"))
		if splitErr != nil || connected == nil || !connected.Equal(ip) || (allowLoopback && !connected.IsLoopback()) || (!allowLoopback && unsafeEgressIP(connected)) {
			_ = conn.Close()
			lastErr = errors.New("remote MCP connected address was not approved")
			continue
		}
		return conn, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("remote MCP has no address for requested network")
}

func remoteMCPLoopbackContext(ctx context.Context) bool {
	value, _ := ctx.Value(remoteMCPLoopbackContextKey{}).(bool)
	return value
}

func remoteMCPPrivateIP(ip net.IP) bool { return unsafeEgressIP(ip) } //nolint:unused // compatibility/security surface retained for future adapter wiring

func remoteMCPContainsIP(values []net.IP, wanted net.IP) bool {
	for _, value := range values {
		if value != nil && value.Equal(wanted) {
			return true
		}
	}
	return false
}

func resolveRemoteMCPDestination(ctx context.Context, parsed *url.URL) ([]net.IP, error) {
	if parsed == nil {
		return nil, errors.New("remote MCP URL is required")
	}
	if remoteMCPLoopback(parsed.Hostname()) {
		return nil, nil
	}
	addresses, err := lookupRemoteMCPIPs(ctx, parsed.Hostname())
	if err != nil {
		return nil, fmt.Errorf("resolve remote MCP host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("remote MCP host resolved to no addresses")
	}
	approved := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if unsafeEgressIP(address) {
			return nil, errors.New("remote MCP host resolved to a private or link-local address")
		}
		approved = append(approved, append(net.IP(nil), address...))
	}
	return approved, nil
}

func remoteMCPURLAllowed(parsed *url.URL) bool {
	if parsed == nil || parsed.User != nil || parsed.Fragment != "" || parsed.Hostname() == "" {
		return false
	}
	return parsed.Scheme == "https" || remoteMCPLoopback(parsed.Hostname())
}

func sameRemoteMCPOrigin(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func (m *RemoteMCPManager) List() []RemoteMCPServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]RemoteMCPServerConfig, 0, len(m.servers))
	for _, config := range m.servers {
		config.URL = providerCatalogOrigin(config.URL)
		config.AllowedMethods = append([]string(nil), config.AllowedMethods...)
		config.HeadersEnv = mapsClone(config.HeadersEnv)
		result = append(result, config)
	}
	return result
}

func (m *RemoteMCPManager) ListForOrganization(organizationID string) []RemoteMCPServerConfig {
	organizationID = strings.TrimSpace(organizationID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]RemoteMCPServerConfig, 0)
	for _, config := range m.servers {
		if !pluginAccessibleByOrganization(config.OrganizationID, organizationID) {
			continue
		}
		copy := config
		copy.URL = providerCatalogOrigin(config.URL)
		copy.TokenEnv = ""
		copy.HeadersEnv = nil
		copy.AllowedMethods = append([]string(nil), config.AllowedMethods...)
		result = append(result, copy)
	}
	return result
}

func mapsClone(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (m *RemoteMCPManager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	config, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("remote MCP server %q is not registered", id)
	}
	if !pluginGlobal(config.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	config.Disabled = !enabled
	m.servers[id] = config
	if err := m.persistLocked(); err != nil {
		config.Disabled = !config.Disabled
		m.servers[id] = config
		return fmt.Errorf("persist remote MCP manifest: %w", err)
	}
	return nil
}

func (m *RemoteMCPManager) SetEnabledForOrganization(organizationID, id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	config, ok := m.servers[id]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(config.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	config.Disabled = !enabled
	m.servers[id] = config
	if err := m.persistLocked(); err != nil {
		config.Disabled = !config.Disabled
		m.servers[id] = config
		return fmt.Errorf("persist remote MCP manifest: %w", err)
	}
	return nil
}

func (m *RemoteMCPManager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	config, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("remote MCP server %q is not registered", id)
	}
	if !pluginGlobal(config.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	delete(m.servers, id)
	if err := m.persistLocked(); err != nil {
		m.servers[id] = config
		return fmt.Errorf("persist remote MCP manifest: %w", err)
	}
	return nil
}

func (m *RemoteMCPManager) RemoveForOrganization(organizationID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	config, ok := m.servers[id]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(config.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	delete(m.servers, id)
	if err := m.persistLocked(); err != nil {
		m.servers[id] = config
		return fmt.Errorf("persist remote MCP manifest: %w", err)
	}
	return nil
}

func (m *RemoteMCPManager) Call(ctx context.Context, serverID, method string, params any) (json.RawMessage, error) {
	return nil, ErrPluginOrganizationScope
}

// CallGlobal is reserved for explicitly configured ownerless servers used by
// the trusted single-user/local runtime. Organization-scoped requests must use
// CallForOrganization and can access only organization-owned servers.
func (m *RemoteMCPManager) CallGlobal(ctx context.Context, serverID, method string, params any) (json.RawMessage, error) {
	return m.call(ctx, "", serverID, method, params, true)
}

func (m *RemoteMCPManager) CallForOrganization(ctx context.Context, organizationID, serverID, method string, params any) (json.RawMessage, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, ErrPluginOrganizationScope
	}
	return m.call(ctx, organizationID, serverID, method, params, false)
}

func (m *RemoteMCPManager) call(ctx context.Context, organizationID, serverID, method string, params any, global bool) (json.RawMessage, error) {
	m.mu.RLock()
	config, ok := m.servers[strings.TrimSpace(serverID)]
	client := m.client
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("remote MCP server %q is not registered", serverID)
	}
	if global {
		if strings.TrimSpace(config.OrganizationID) != "" {
			return nil, ErrPluginOrganizationScope
		}
	} else if !pluginOwnedByOrganization(config.OrganizationID, organizationID) {
		return nil, ErrPluginOrganizationScope
	}
	if config.Disabled {
		return nil, errors.New("remote MCP server is disabled")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("remote MCP method is required")
	}
	if len(config.AllowedMethods) > 0 && !remoteMCPContains(config.AllowedMethods, method) {
		return nil, fmt.Errorf("remote MCP method %q is not allowlisted", method)
	}
	if err := validateOutboundPayload(map[string]any{"server_id": serverID, "method": method, "params": params}); err != nil {
		return nil, err
	}
	headerValues := make(map[string]string, len(config.HeadersEnv))
	for header, envName := range config.HeadersEnv {
		value, ok := m.resolveSecret(organizationID, envName, global)
		if !ok || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("remote MCP header credential %q is unavailable", header)
		}
		headerValues[header] = strings.TrimSpace(value)
	}
	requestID := atomic.AddInt64(&m.nextID, 1)
	requestBody, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": requestID, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}
	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	token, tokenErr := m.accessToken(requestContext, serverID, config, organizationID, global)
	if tokenErr != nil {
		return nil, tokenErr
	}
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}
	requestContext = context.WithValue(requestContext, remoteMCPLoopbackContextKey{}, remoteMCPLoopback(parsedURL.Hostname()))
	if !remoteMCPLoopback(parsedURL.Hostname()) {
		approved, resolveErr := resolveRemoteMCPDestination(requestContext, parsedURL)
		if resolveErr != nil {
			return nil, resolveErr
		}
		requestContext = context.WithValue(requestContext, remoteMCPApprovedIPsContextKey{}, approved)
	}
	req, err := http.NewRequestWithContext(requestContext, http.MethodPost, config.URL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if session, hasSession := m.latestSession(serverID, organizationID); hasSession {
		req.Header.Set("Mcp-Session-Id", session.ID)
		if session.LastEventID != "" {
			req.Header.Set("Last-Event-ID", session.LastEventID)
		}
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for header, value := range headerValues {
		req.Header.Set(header, value)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("remote MCP provider request failed")
	}
	defer response.Body.Close()
	if sessionID := response.Header.Get("Mcp-Session-Id"); sessionID != "" {
		m.rememberSession(serverID, organizationID, sessionID, time.Now().UTC().Add(30*time.Minute))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("remote MCP provider returned HTTP status %d", response.StatusCode)
	}
	stream := strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")
	payload, err := readRemoteMCPResponse(response.Body, stream, requestID)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func readRemoteMCPResponse(body io.Reader, stream bool, requestID int64) (json.RawMessage, error) {
	const maxPayload = 4 << 20
	if !stream {
		payload, err := io.ReadAll(io.LimitReader(body, maxPayload+1))
		if err != nil {
			return nil, err
		}
		if len(payload) > maxPayload {
			return nil, errors.New("remote MCP response exceeded limit")
		}
		return decodeRemoteMCPEnvelope(payload, requestID)
	}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var data strings.Builder
	totalBytes := 0
	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		totalBytes += len(lineBytes) + 1 // include the line terminator consumed by Scanner
		if totalBytes > maxPayload {
			return nil, errors.New("remote MCP SSE response exceeded limit")
		}
		line := string(lineBytes)
		if line == "" {
			if data.Len() == 0 {
				continue
			}
			payload, err := decodeRemoteMCPEnvelope([]byte(data.String()), requestID)
			if err == nil {
				return payload, nil
			}
			if !errors.Is(err, errRemoteMCPIDMismatch) {
				return nil, err
			}
			data.Reset()
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if value == "[DONE]" {
				continue
			}
			data.WriteString(value)
		}
		if data.Len() > maxPayload {
			return nil, errors.New("remote MCP SSE response exceeded limit")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if data.Len() > 0 {
		return decodeRemoteMCPEnvelope([]byte(data.String()), requestID)
	}
	return nil, errors.New("remote MCP SSE response has no correlated result")
}

func decodeRemoteMCPEnvelope(payload []byte, requestID int64) (json.RawMessage, error) {
	var envelope struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *mcpError       `json:"error,omitempty"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode remote MCP response: %w", err)
	}
	if !remoteMCPResponseMatches(envelope.ID, requestID) {
		return nil, errRemoteMCPIDMismatch
	}
	if envelope.Error != nil {
		return nil, errors.New("remote MCP provider returned an error")
	}
	if len(envelope.Result) == 0 {
		return nil, errors.New("remote MCP response has no result")
	}
	return sanitizeProviderJSON(envelope.Result)
}

func remoteMCPResponseMatches(raw json.RawMessage, expected int64) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var number int64
	if json.Unmarshal(raw, &number) == nil {
		return number == expected
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text == fmt.Sprintf("%d", expected)
	}
	return false
}

func remoteMCPContains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validRemoteMCPHeaderName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if char <= 32 || char >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", char) {
			return false
		}
	}
	switch strings.ToLower(value) {
	case "host", "content-length", "transfer-encoding", "connection", "proxy-connection", "proxy-authorization", "upgrade", "te", "trailer":
		return false
	default:
		return true
	}
}

type remoteMCPCallTool struct{ manager *RemoteMCPManager }

func (t remoteMCPCallTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "mcp.remote.call", Version: "1", Description: "Chamar método allowlisted de um Remote MCP Streamable HTTP", Risk: RiskExternalSideEffect, RequiresApproval: true, Scopes: []string{"mcp:remote:call"}}
}

func (t remoteMCPCallTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if t.manager == nil {
		return ToolResult{}, errors.New("remote MCP manager is unavailable")
	}
	var result json.RawMessage
	var err error
	if strings.TrimSpace(toolContext.OrganizationID) == "" {
		result, err = t.manager.CallGlobal(ctx, stringInput(input, "server_id", ""), stringInput(input, "method", ""), input["params"])
	} else {
		result, err = t.manager.CallForOrganization(ctx, toolContext.OrganizationID, stringInput(input, "server_id", ""), stringInput(input, "method", ""), input["params"])
	}
	if err != nil {
		return ToolResult{}, err
	}
	var value any
	if err := json.Unmarshal(result, &value); err != nil {
		return ToolResult{Value: string(result)}, nil //nolint:nilerr // raw MCP payloads may be valid string results.
	}
	return ToolResult{Value: value}, nil
}
