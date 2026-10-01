package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/internal/multillm"
)

type ConnectorConfig struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id,omitempty"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	TokenEnv       string `json:"token_env,omitempty"`
	// AuthHeader and AuthScheme describe how the token is sent. The default is
	// "Authorization: Bearer <token>"; AuthScheme "raw" sends the bare token.
	AuthHeader           string               `json:"auth_header,omitempty"`
	AuthScheme           string               `json:"auth_scheme,omitempty"`
	OAuthProvider        string               `json:"oauth_provider,omitempty"`
	AllowedOrigins       []string             `json:"allowed_origins,omitempty"`
	Operations           []ConnectorOperation `json:"operations"`
	TimeoutSeconds       int                  `json:"timeout_seconds,omitempty"`
	Disabled             bool                 `json:"disabled,omitempty"`
	CredentialConfigured bool                 `json:"credential_configured,omitempty"`
}

type ConnectorOperation struct {
	Name         string   `json:"name"`
	Methods      []string `json:"methods"`
	PathPrefixes []string `json:"path_prefixes"`
}

type ConnectorManager struct {
	mu          sync.RWMutex
	connectors  map[string]ConnectorConfig
	client      *http.Client
	auth        *AuthStore
	persistPath string
}

var (
	errConnectorRedirectDisabled = errors.New("connector redirects are disabled")
	errConnectorResponseTooLarge = errors.New("connector response payload exceeds limit")
)

var (
	ErrConnectorDisabled              = errors.New("connector is disabled")
	ErrConnectorCredentialUnavailable = errors.New("connector credential is unavailable")
	ErrPluginNotFound                 = errors.New("plugin not found")
)

type connectorLoopbackContextKey struct{}

func NewConnectorManager() *ConnectorManager {
	return &ConnectorManager{
		connectors: make(map[string]ConnectorConfig),
		client: NewSafeEgressHTTPClient(EgressOptions{
			Callsite:     "connectors",
			Timeout:      30 * time.Second,
			MaxBodyBytes: 2 << 20,
		}),
	}
}

// NewPersistentConnectorManager loads a connector manifest that contains only
// configuration references (for example environment variable names), never
// credential values. Mutations are written back with mode 0600 and an atomic
// rename so a restart preserves the configured lifecycle state.
func NewPersistentConnectorManager(manifestPath string) (*ConnectorManager, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	manager := NewConnectorManager()
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
		var configs []ConnectorConfig
		if err := decoder.Decode(&configs); err != nil {
			return nil, fmt.Errorf("decode connector manifest: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return nil, errors.New("connector manifest contains trailing JSON")
			}
			return nil, fmt.Errorf("decode connector manifest trailing data: %w", err)
		}
		for _, config := range configs {
			if err := validateConnectorConfig(&config); err != nil {
				return nil, fmt.Errorf("load connector %q: %w", config.ID, err)
			}
			manager.connectors[config.ID] = cloneConnectorConfig(config)
		}
	}
	manager.persistPath = manifestPath
	return manager, nil
}

func (m *ConnectorManager) SetOAuthStore(store *AuthStore) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.auth = store
	m.mu.Unlock()
}

func (m *ConnectorManager) Register(config ConnectorConfig) error {
	if err := validateConnectorConfig(&config); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registerValidatedLocked(config)
}

func (m *ConnectorManager) registerValidatedLocked(config ConnectorConfig) error {
	previous, existed := m.connectors[config.ID]
	m.connectors[config.ID] = cloneConnectorConfig(config)
	if err := m.persistLocked(); err != nil {
		if existed {
			m.connectors[config.ID] = previous
		} else {
			delete(m.connectors, config.ID)
		}
		return fmt.Errorf("persist connector manifest: %w", err)
	}
	return nil
}

// RegisterForOrganization ignores any caller-supplied organization and binds
// the connector to the authenticated organization. A non-empty conflicting
// organization is rejected instead of silently moving tenant-owned state.
func (m *ConnectorManager) RegisterForOrganization(organizationID string, config ConnectorConfig) error {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return errors.New("connector organization scope is required")
	}
	if supplied := strings.TrimSpace(config.OrganizationID); supplied != "" && supplied != organizationID {
		return ErrPluginOrganizationScope
	}
	config.OrganizationID = organizationID
	if err := validateConnectorConfig(&config); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.connectors[config.ID]; ok && !pluginOwnedByOrganization(existing.OrganizationID, organizationID) {
		return ErrPluginOrganizationScope
	}
	return m.registerValidatedLocked(config)
}

func validateConnectorConfig(config *ConnectorConfig) error {
	config.ID = strings.TrimSpace(config.ID)
	config.OrganizationID = strings.TrimSpace(config.OrganizationID)
	config.Provider = strings.TrimSpace(config.Provider)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	config.TokenEnv = strings.TrimSpace(config.TokenEnv)
	config.OAuthProvider = strings.TrimSpace(config.OAuthProvider)
	config.CredentialConfigured = false
	if config.ID == "" || config.Provider == "" {
		return errors.New("connector id and provider are required")
	}
	if err := validateConnectorURL(config.BaseURL, false); err != nil {
		return fmt.Errorf("connector base_url %w", err)
	}
	for i := range config.AllowedOrigins {
		config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		if err := validateConnectorURL(config.AllowedOrigins[i], true); err != nil {
			return fmt.Errorf("connector allowed_origins[%d] %w", i, err)
		}
	}
	if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > 120 {
		config.TimeoutSeconds = 30
	}
	if config.TokenEnv != "" && !validEnvName(config.TokenEnv) {
		return errors.New("invalid connector token_env")
	}
	config.AuthHeader = strings.TrimSpace(config.AuthHeader)
	config.AuthScheme = strings.TrimSpace(config.AuthScheme)
	if config.AuthHeader != "" && !validAuthHeader(config.AuthHeader) {
		return errors.New("invalid connector auth_header")
	}
	if config.AuthScheme != "" && !validAuthScheme(config.AuthScheme) {
		return errors.New("invalid connector auth_scheme")
	}
	for i := range config.Operations {
		operation := &config.Operations[i]
		operation.Name = strings.TrimSpace(operation.Name)
		if operation.Name == "" || len(operation.Methods) == 0 || len(operation.PathPrefixes) == 0 {
			return errors.New("connector operations require name, methods and path_prefixes")
		}
		for j := range operation.Methods {
			operation.Methods[j] = strings.ToUpper(strings.TrimSpace(operation.Methods[j]))
			if operation.Methods[j] == "" {
				return errors.New("connector operation methods must be non-empty")
			}
		}
		for j := range operation.PathPrefixes {
			operation.PathPrefixes[j] = strings.TrimSpace(operation.PathPrefixes[j])
			if !validConnectorPath(operation.PathPrefixes[j]) {
				return fmt.Errorf("invalid connector path prefix %q", operation.PathPrefixes[j])
			}
		}
	}
	return nil
}

func validateConnectorURL(raw string, originOnly bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\x00\r\n") {
		return errors.New("must be a valid HTTPS URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("must be an HTTPS URL without credentials, query, or fragment")
	}
	if originOnly && parsed.Path != "" && parsed.Path != "/" {
		return errors.New("must be an origin without a path")
	}
	for decoded := raw; ; {
		if len(ScanDLP(decoded)) > 0 || strings.Contains(strings.ToLower(decoded), "signature=") || strings.Contains(strings.ToLower(decoded), "x-amz-signature") || strings.Contains(strings.ToLower(decoded), "sig=") {
			return errors.New("must not contain credential-shaped URL data")
		}
		next, decodeErr := url.PathUnescape(decoded)
		if decodeErr != nil {
			return errors.New("must contain valid URL escaping")
		}
		if next == decoded {
			break
		}
		decoded = next
	}
	return nil
}

func cloneConnectorConfig(config ConnectorConfig) ConnectorConfig {
	copy := config
	copy.AllowedOrigins = append([]string(nil), config.AllowedOrigins...)
	copy.Operations = make([]ConnectorOperation, len(config.Operations))
	for i, operation := range config.Operations {
		copy.Operations[i] = ConnectorOperation{
			Name:         operation.Name,
			Methods:      append([]string(nil), operation.Methods...),
			PathPrefixes: append([]string(nil), operation.PathPrefixes...),
		}
	}
	return copy
}

func (m *ConnectorManager) persistLocked() error {
	if strings.TrimSpace(m.persistPath) == "" {
		return nil
	}
	configs := make([]ConnectorConfig, 0, len(m.connectors))
	for _, config := range m.connectors {
		config.CredentialConfigured = false
		configs = append(configs, cloneConnectorConfig(config))
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(m.persistPath), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(m.persistPath), ".connectors-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, m.persistPath)
}

func (m *ConnectorManager) List() []ConnectorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ConnectorConfig, 0, len(m.connectors))
	for _, connector := range m.connectors {
		copy := cloneConnectorConfig(connector)
		copy.BaseURL = providerCatalogOrigin(copy.BaseURL)
		copy.TokenEnv = ""
		copy.CredentialConfigured = m.credentialConfigured(connector, "")
		result = append(result, copy)
	}
	return result
}

func (m *ConnectorManager) ListForOrganization(organizationID string) []ConnectorConfig {
	organizationID = strings.TrimSpace(organizationID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ConnectorConfig, 0)
	for _, connector := range m.connectors {
		if !pluginAccessibleByOrganization(connector.OrganizationID, organizationID) {
			continue
		}
		copy := cloneConnectorConfig(connector)
		copy.BaseURL = providerCatalogOrigin(copy.BaseURL)
		copy.TokenEnv = ""
		copy.CredentialConfigured = m.credentialConfigured(connector, organizationID)
		result = append(result, copy)
	}
	return result
}

func (m *ConnectorManager) credentialConfigured(config ConnectorConfig, organizationID string) bool {
	if config.TokenEnv != "" && strings.TrimSpace(multillm.CredentialValue(config.TokenEnv)) != "" {
		return true
	}
	return config.OAuthProvider != "" && m.auth != nil && m.auth.HasOAuthCredentialForOrganization(organizationID, config.OAuthProvider)
}

func (m *ConnectorManager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	connector, ok := m.connectors[id]
	if !ok {
		return fmt.Errorf("connector %q is not registered", id)
	}
	if !pluginGlobal(connector.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	connector.Disabled = !enabled
	m.connectors[id] = connector
	if err := m.persistLocked(); err != nil {
		connector.Disabled = !connector.Disabled
		m.connectors[id] = connector
		return fmt.Errorf("persist connector manifest: %w", err)
	}
	return nil
}

func (m *ConnectorManager) SetEnabledForOrganization(organizationID, id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	connector, ok := m.connectors[strings.TrimSpace(id)]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(connector.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	connector.Disabled = !enabled
	m.connectors[connector.ID] = connector
	if err := m.persistLocked(); err != nil {
		connector.Disabled = !connector.Disabled
		m.connectors[connector.ID] = connector
		return fmt.Errorf("persist connector manifest: %w", err)
	}
	return nil
}

func (m *ConnectorManager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	connector, ok := m.connectors[id]
	if !ok {
		return fmt.Errorf("connector %q is not registered", id)
	}
	if !pluginGlobal(connector.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	delete(m.connectors, id)
	if err := m.persistLocked(); err != nil {
		m.connectors[id] = connector
		return fmt.Errorf("persist connector manifest: %w", err)
	}
	return nil
}

func (m *ConnectorManager) RemoveForOrganization(organizationID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	connector, ok := m.connectors[id]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(connector.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	delete(m.connectors, id)
	if err := m.persistLocked(); err != nil {
		m.connectors[id] = connector
		return fmt.Errorf("persist connector manifest: %w", err)
	}
	return nil
}

func (m *ConnectorManager) Call(ctx context.Context, connectorID, operationName, method, requestPath string, body []byte) (int, string, error) {
	return 0, "", ErrPluginOrganizationScope
}

// CallGlobal is reserved for explicitly configured ownerless connectors used
// by the trusted single-user/local runtime. Tenant requests must use
// CallForOrganization and can access only connectors owned by that tenant.
func (m *ConnectorManager) CallGlobal(ctx context.Context, connectorID, operationName, method, requestPath string, body []byte) (int, string, error) {
	m.mu.RLock()
	config, ok := m.connectors[strings.TrimSpace(connectorID)]
	auth := m.auth
	m.mu.RUnlock()
	if !ok {
		return 0, "", fmt.Errorf("connector %q is not registered", connectorID)
	}
	if !pluginGlobal(config.OrganizationID) {
		return 0, "", ErrPluginOrganizationScope
	}
	if config.Disabled {
		return 0, "", ErrConnectorDisabled
	}
	config = cloneConnectorConfig(config)
	return m.call(ctx, "", config, connectorID, operationName, method, requestPath, body, auth, true)
}

func (m *ConnectorManager) CallForOrganization(ctx context.Context, organizationID, connectorID, operationName, method, requestPath string, body []byte) (int, string, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return 0, "", errors.New("connector organization scope is required")
	}
	m.mu.RLock()
	config, ok := m.connectors[strings.TrimSpace(connectorID)]
	auth := m.auth
	m.mu.RUnlock()
	if !ok {
		return 0, "", fmt.Errorf("connector %q is not registered", connectorID)
	}
	if !pluginAccessibleByOrganization(config.OrganizationID, organizationID) {
		return 0, "", ErrPluginOrganizationScope
	}
	if config.Disabled {
		return 0, "", ErrConnectorDisabled
	}
	config = cloneConnectorConfig(config)
	return m.call(ctx, organizationID, config, connectorID, operationName, method, requestPath, body, auth, false)
}

func (m *ConnectorManager) CallForOrganizationWithApproval(ctx context.Context, organizationID string, input map[string]any, body []byte, expectedSHA256 string) (int, string, error) {
	return m.callWithApproval(ctx, strings.TrimSpace(organizationID), input, body, expectedSHA256, false)
}

// CallGlobalWithApproval is the approval-bound global counterpart reserved for
// trusted local-mode execution.
func (m *ConnectorManager) CallGlobalWithApproval(ctx context.Context, input map[string]any, body []byte, expectedSHA256 string) (int, string, error) {
	return m.callWithApproval(ctx, "", input, body, expectedSHA256, true)
}

func (m *ConnectorManager) callWithApproval(ctx context.Context, organizationID string, input map[string]any, body []byte, expectedSHA256 string, global bool) (int, string, error) {
	connectorID := strings.TrimSpace(stringInput(input, "connector_id", ""))
	operationName := strings.TrimSpace(stringInput(input, "operation", ""))
	method := strings.ToUpper(strings.TrimSpace(stringInput(input, "method", "GET")))
	requestPath := strings.TrimSpace(stringInput(input, "path", "/"))
	if (!global && organizationID == "") || connectorID == "" || strings.TrimSpace(expectedSHA256) == "" {
		return 0, "", errors.New("connector approval binding is required")
	}
	m.mu.RLock()
	config, ok := m.connectors[connectorID]
	auth := m.auth
	if ok {
		config = cloneConnectorConfig(config)
	}
	configured := ok && m.credentialConfigured(config, organizationID)
	m.mu.RUnlock()
	if !ok || (global && !pluginGlobal(config.OrganizationID)) || (!global && !pluginOwnedByOrganization(config.OrganizationID, organizationID)) {
		return 0, "", ErrPluginOrganizationScope
	}
	if config.Disabled {
		return 0, "", ErrConnectorDisabled
	}
	if _, err := validateConnectorRequestPath(requestPath); err != nil {
		return 0, "", err
	}
	operation, allowed := findConnectorOperation(config.Operations, operationName, method, requestPath)
	if !allowed {
		return 0, "", fmt.Errorf("connector operation %q is not allowlisted", operationName)
	}
	digest, err := connectorApprovalConfigHash(config, operation, configured)
	if err != nil {
		return 0, "", err
	}
	if digest != expectedSHA256 {
		return 0, "", ErrApprovalPayloadChanged
	}
	return m.call(ctx, organizationID, config, connectorID, operationName, method, requestPath, body, auth, global)
}

func (m *ConnectorManager) call(ctx context.Context, organizationID string, config ConnectorConfig, connectorID, operationName, method, requestPath string, body []byte, auth *AuthStore, global bool) (int, string, error) {
	if (global && !pluginGlobal(config.OrganizationID)) || (!global && !pluginOwnedByOrganization(config.OrganizationID, organizationID)) {
		return 0, "", ErrPluginOrganizationScope
	}
	if config.Disabled {
		return 0, "", ErrConnectorDisabled
	}
	if _, err := validateConnectorRequestPath(requestPath); err != nil {
		return 0, "", err
	}
	operation, allowed := findConnectorOperation(config.Operations, operationName, method, requestPath)
	if !allowed {
		return 0, "", fmt.Errorf("connector operation %q is not allowlisted", operationName)
	}
	_ = operation
	base, _ := url.Parse(config.BaseURL)
	if len(config.AllowedOrigins) > 0 {
		origin := base.Scheme + "://" + base.Host
		permitted := false
		for _, allowed := range config.AllowedOrigins {
			if strings.EqualFold(strings.TrimRight(strings.TrimSpace(allowed), "/"), origin) {
				permitted = true
				break
			}
		}
		if !permitted {
			return 0, "", fmt.Errorf("connector origin %q is not in allowed_origins", origin)
		}
	}
	relative, err := validateConnectorRequestPath(requestPath)
	if err != nil {
		return 0, "", err
	}
	if len(body) > 1<<20 {
		return 0, "", errors.New("connector request payload exceeds limit")
	}
	if err := validateOutboundPayloadWithLimit(map[string]any{"connector_id": connectorID, "operation": operationName, "method": method, "path": requestPath, "body": string(body)}, 1<<20); err != nil {
		return 0, "", err
	}
	base.Path = path.Join(strings.TrimSuffix(base.Path, "/"), relative.Path)
	base.RawQuery = relative.RawQuery
	requestContext := context.WithValue(ctx, connectorLoopbackContextKey{}, connectorHostIsLoopback(base.Hostname()))
	request, err := http.NewRequestWithContext(requestContext, strings.ToUpper(method), base.String(), bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	token := ""
	if auth != nil && strings.TrimSpace(config.OAuthProvider) != "" {
		var err error
		token, _, err = auth.OAuthAccessTokenForOrganization(organizationID, config.OAuthProvider)
		if err != nil {
			return 0, "", fmt.Errorf("resolve OAuth credential for connector %q: %w", connectorID, err)
		}
	}
	if token == "" && config.TokenEnv != "" {
		token = multillm.CredentialValue(config.TokenEnv)
	}
	if strings.TrimSpace(token) == "" && (config.TokenEnv != "" || config.OAuthProvider != "") {
		return 0, "", fmt.Errorf("%w for connector %q", ErrConnectorCredentialUnavailable, connectorID)
	}
	if token != "" {
		header, value := connectorAuth(config, token)
		request.Header.Set(header, value)
	}
	// Never place request bodies or credentials in the audit trail. The
	// payload validator above rejects credential-shaped data; this second
	// defense records only the redaction result and request metadata.
	findings := ScanDLP(string(body))
	DefaultEgressAuditStore.Record(EgressDecision{
		Timestamp:   time.Now(),
		Callsite:    "connectors",
		Method:      request.Method,
		Destination: request.URL.Redacted(),
		Host:        request.URL.Hostname(),
		Allowed:     true,
		Reason:      fmt.Sprintf("connector request approved; dlp_findings=%d", len(findings)),
	})
	client := connectorClientForRequest(m.client, time.Duration(config.TimeoutSeconds)*time.Second)
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(err, errConnectorRedirectDisabled) || errors.Is(err, ErrEgressRedirectDisallowed) || strings.Contains(strings.ToLower(err.Error()), "redirect") {
			return 0, "", fmt.Errorf("%w: %v", errConnectorRedirectDisabled, err)
		}
		return 0, "", errors.New("connector provider request failed")
	}
	defer response.Body.Close()
	data, err := readLimitedConnectorBody(response.Body, 2<<20)
	if err != nil {
		if errors.Is(err, errConnectorResponseTooLarge) {
			return response.StatusCode, "", errConnectorResponseTooLarge
		}
		return response.StatusCode, "", errors.New("connector provider response could not be read")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, "", fmt.Errorf("connector provider returned HTTP status %d", response.StatusCode)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return response.StatusCode, "", nil
	}
	clean, err := sanitizeProviderJSON(json.RawMessage(data))
	if err != nil {
		return response.StatusCode, "", errors.New("connector provider returned an invalid successful JSON response")
	}
	return response.StatusCode, string(clean), nil
}

func connectorClientForRequest(base *http.Client, timeout time.Duration) *http.Client {
	if base == nil {
		client := NewConnectorManager().client
		client.Timeout = timeout
		return client
	}
	client := *base
	client.Timeout = timeout
	// The manager's production client is already built by
	// NewSafeEgressHTTPClient. Keep its pinned transport intact; replacing it
	// with a generic DialContext would discard peer verification and audit logs.
	client.CheckRedirect = NewEgressCheckRedirect("connectors", false)
	return &client
}

func readLimitedConnectorBody(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errConnectorResponseTooLarge
	}
	return data, nil
}

func findConnectorOperation(operations []ConnectorOperation, name, method, requestPath string) (ConnectorOperation, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	parsedPath, err := validateConnectorRequestPath(requestPath)
	if err != nil {
		return ConnectorOperation{}, false
	}
	matchPath := parsedPath.Path
	for _, operation := range operations {
		if operation.Name != name {
			continue
		}
		methodOK := false
		for _, allowedMethod := range operation.Methods {
			if strings.ToUpper(allowedMethod) == method {
				methodOK = true
				break
			}
		}
		if !methodOK {
			return operation, false
		}
		for _, prefix := range operation.PathPrefixes {
			if connectorPathMatches(matchPath, prefix) {
				return operation, true
			}
		}
	}
	return ConnectorOperation{}, false
}

func connectorPathMatches(requestPath, prefix string) bool {
	requestPath = strings.TrimSpace(requestPath)
	prefix = strings.TrimSpace(prefix)
	if prefix == "/" {
		return strings.HasPrefix(requestPath, "/")
	}
	prefix = strings.TrimSuffix(prefix, "/")
	return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
}

func connectorHostIsLoopback(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func connectorPrivateIP(ip net.IP) bool {
	return unsafeEgressIP(ip)
}

func connectorDialContext(ctx context.Context, network, address string) (net.Conn, error) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	return connectorDialContextWithResolver(ctx, network, address, func(ctx context.Context, host string) ([]net.IP, error) {
		return net.DefaultResolver.LookupIP(ctx, "ip", host)
	})
}

func connectorDialContextWithResolver(ctx context.Context, network, address string, lookup func(context.Context, string) ([]net.IP, error)) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return nil, errors.New("connector destination is invalid")
	}
	allowedLoopback, _ := ctx.Value(connectorLoopbackContextKey{}).(bool)
	host = strings.Trim(host, "[]")
	addresses := make([]net.IP, 0, 1)
	if ip := net.ParseIP(host); ip != nil {
		addresses = append(addresses, ip)
	} else {
		if lookup == nil {
			return nil, errors.New("connector destination resolver is unavailable")
		}
		addresses, err = lookup(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("connector destination lookup failed: %w", err)
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("connector destination has no addresses")
	}
	for _, ip := range addresses {
		if ip == nil || (allowedLoopback && !ip.IsLoopback()) {
			return nil, errors.New("connector loopback destination resolves outside loopback")
		}
		if !allowedLoopback && connectorPrivateIP(ip) {
			return nil, errors.New("connector destination resolves to a private address")
		}
	}
	var lastErr error
	for _, ip := range addresses {
		if network == "tcp4" && ip.To4() == nil {
			continue
		}
		if network == "tcp6" && ip.To4() != nil {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr != nil {
			lastErr = dialErr
			continue
		}
		remote, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		connected := net.ParseIP(strings.Trim(remote, "[]"))
		if splitErr != nil || connected == nil || !connected.Equal(ip) {
			_ = conn.Close()
			lastErr = errors.New("connector connected address was not approved")
			continue
		}
		if (allowedLoopback && !connected.IsLoopback()) || (!allowedLoopback && connectorPrivateIP(connected)) {
			_ = conn.Close()
			lastErr = errors.New("connector destination connected to a private address")
			continue
		}
		return conn, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("connector destination has no address for requested network")
}

// connectorAuth returns the header and value carrying the token.
func connectorAuth(config ConnectorConfig, token string) (string, string) {
	header := config.AuthHeader
	if header == "" {
		header = "Authorization"
	}
	switch scheme := config.AuthScheme; scheme {
	case "raw":
		return header, token
	case "":
		if header == "Authorization" {
			return header, "Bearer " + token
		}
		return header, token
	default:
		return header, scheme + " " + token
	}
}

// validAuthHeader accepts HTTP header names used for API keys and rejects
// headers that could change routing or framing.
func validAuthHeader(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	switch strings.ToLower(value) {
	case "host", "content-length", "transfer-encoding", "connection", "cookie", "content-type", "accept":
		return false
	}
	return true
}

func validAuthScheme(value string) bool {
	if value == "raw" {
		return true
	}
	if len(value) == 0 || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func validConnectorPath(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#\x00\r\n") {
		return false
	}
	decoded := value
	for range 8 {
		next, err := url.PathUnescape(decoded)
		if err != nil {
			return false
		}
		if next == decoded {
			break
		}
		decoded = next
	}
	if next, err := url.PathUnescape(decoded); err != nil || next != decoded {
		return false
	}
	return !strings.Contains(decoded, "..") && !strings.ContainsAny(decoded, "\\?#\x00\r\n")
}

func validateConnectorRequestPath(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" || !validConnectorPath(parsed.Path) {
		return nil, errors.New("invalid connector request path")
	}
	if parsed.RawQuery == "" {
		return parsed, nil
	}
	decodedQuery := decodeURLComponentFully(parsed.RawQuery)
	// A semicolon is not a portable query separator: some clients reject it,
	// while others treat it as an additional parameter delimiter. Reject it
	// after bounded decoding so an encoded or nested form cannot hide a
	// credential alias from the provider-request guard.
	if strings.Contains(decodedQuery, ";") {
		return nil, errors.New("connector request query contains credentials")
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return nil, errors.New("invalid connector request query")
	}
	for key, items := range values {
		decodedKey := decodeURLComponentFully(key)
		if sensitiveDLPKey(decodedKey) || len(ScanDLP(decodedKey)) > 0 {
			return nil, errors.New("connector request query contains credentials")
		}
		for _, item := range items {
			decoded := decodeURLComponentFully(item)
			if sensitiveDLPKey(decodedKey) || len(ScanDLP(decoded)) > 0 || endpointURLHasSensitiveMaterial(decoded) {
				return nil, errors.New("connector request query contains credentials")
			}
		}
	}
	return parsed, nil
}

func validEnvName(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if !(char == '_' || char >= 'A' && char <= 'Z' || index > 0 && char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

type connectorTool struct{ manager *ConnectorManager }

func (t connectorTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "connector.http", Version: "1", Description: "Chamar operação allowlisted de um conector externo", Risk: RiskExternalSideEffect, RequiresApproval: true, Scopes: []string{"connector:external"}}
}

func (t connectorTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if t.manager == nil {
		return ToolResult{}, errors.New("connector manager is unavailable")
	}
	body := []byte(stringInput(input, "body", ""))
	if len(body) > 1<<20 {
		return ToolResult{}, errors.New("connector body limit exceeded")
	}
	var status int
	var response string
	var err error
	if strings.TrimSpace(toolContext.OrganizationID) == "" {
		status, response, err = t.manager.CallGlobalWithApproval(ctx, input, body, toolContext.ToolConfigSHA256)
	} else {
		status, response, err = t.manager.CallForOrganizationWithApproval(ctx, toolContext.OrganizationID, input, body, toolContext.ToolConfigSHA256)
	}
	if err != nil {
		return ToolResult{}, err
	}
	var value any
	if json.Unmarshal([]byte(response), &value) != nil {
		value = response
	}
	return ToolResult{Value: map[string]any{"status": status, "response": value}}, nil
}

// ApprovalConfigSHA256 fingerprints the effective non-secret connector policy
// and destination used by one approved operation. Runtime rechecks this value
// immediately before executing a connector step.
func connectorApprovalConfigHash(config ConnectorConfig, operation ConnectorOperation, configured bool) (string, error) {
	origins := append([]string(nil), config.AllowedOrigins...)
	sort.Strings(origins)
	methods := append([]string(nil), operation.Methods...)
	sort.Strings(methods)
	prefixes := append([]string(nil), operation.PathPrefixes...)
	sort.Strings(prefixes)
	material := struct {
		ID, OrganizationID, Provider, BaseURL, OAuthProvider, TokenEnv string
		AllowedOrigins, Methods, PathPrefixes                          []string
		TimeoutSeconds                                                 int
		CredentialConfigured                                           bool
	}{config.ID, config.OrganizationID, config.Provider, config.BaseURL, config.OAuthProvider, config.TokenEnv, origins, methods, prefixes, config.TimeoutSeconds, configured}
	encoded, err := json.Marshal(material)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func (m *ConnectorManager) ApprovalConfigSHA256(organizationID string, input map[string]any) (string, error) {
	return m.approvalConfigSHA256(strings.TrimSpace(organizationID), input, false)
}

// ApprovalConfigSHA256Global is reserved for trusted local-mode execution.
func (m *ConnectorManager) ApprovalConfigSHA256Global(input map[string]any) (string, error) {
	return m.approvalConfigSHA256("", input, true)
}

func (m *ConnectorManager) approvalConfigSHA256(organizationID string, input map[string]any, global bool) (string, error) {
	if m == nil {
		return "", errors.New("connector manager is unavailable")
	}
	connectorID := strings.TrimSpace(stringInput(input, "connector_id", ""))
	operationName := strings.TrimSpace(stringInput(input, "operation", ""))
	method := strings.ToUpper(strings.TrimSpace(stringInput(input, "method", "GET")))
	requestPath := strings.TrimSpace(stringInput(input, "path", "/"))
	if (!global && organizationID == "") || connectorID == "" {
		return "", errors.New("connector organization and ID are required")
	}
	m.mu.RLock()
	config, ok := m.connectors[connectorID]
	configured := ok && m.credentialConfigured(config, organizationID)
	m.mu.RUnlock()
	if !ok || (global && !pluginGlobal(config.OrganizationID)) || (!global && !pluginOwnedByOrganization(config.OrganizationID, organizationID)) {
		return "", ErrPluginOrganizationScope
	}
	if config.Disabled {
		return "", ErrConnectorDisabled
	}
	if _, err := validateConnectorRequestPath(requestPath); err != nil {
		return "", err
	}
	operation, allowed := findConnectorOperation(config.Operations, operationName, method, requestPath)
	if !allowed {
		return "", fmt.Errorf("connector operation %q is not allowlisted", operationName)
	}
	return connectorApprovalConfigHash(config, operation, configured)
}
