package agent

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	"time"
)

type DeployConfig struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id,omitempty"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	TokenEnv       string `json:"token_env,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type DeploymentRequest struct {
	Name           string
	Root           string
	Target         string
	ManifestSHA256 string
}

type DeploymentManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type DeploymentManifest struct {
	SHA256   string                    `json:"sha256"`
	Included []DeploymentManifestEntry `json:"included"`
	Excluded []DeploymentManifestEntry `json:"excluded"`
}

type DeploymentResult struct {
	Provider     string `json:"provider"`
	DeploymentID string `json:"deployment_id,omitempty"`
	URL          string `json:"url,omitempty"`
	Status       string `json:"status"`
	Files        int    `json:"files"`
}

// DeploymentError preserves the provider state that is known when a deploy
// fails after a remote side effect. Callers must not report the operation as a
// clean failure when the provider may have created a site or deployment.
type DeploymentError struct {
	Result DeploymentResult
	Err    error
}

func (e *DeploymentError) Error() string {
	if e == nil || e.Err == nil {
		return "deployment failed with unknown provider state"
	}
	return e.Err.Error()
}

func (e *DeploymentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type DeploymentManager struct {
	mu      sync.RWMutex
	configs map[string]DeployConfig
	client  *http.Client
}

var ErrDeploymentProviderNotFound = errors.New("deployment provider is not configured")

type deploymentLoopbackContextKey struct{}

func NewDeploymentManager() *DeploymentManager {
	return &DeploymentManager{configs: map[string]DeployConfig{}, client: newDeploymentHTTPClient()}
}

func newDeploymentHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = deploymentDialContext
	return &http.Client{Timeout: 120 * time.Second, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("deployment redirects are disabled") }}
}

func deploymentRequestContext(ctx context.Context, rawURL string) (context.Context, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("deployment URL is invalid")
	}
	return context.WithValue(ctx, deploymentLoopbackContextKey{}, isLoopbackHost(parsed.Hostname())), nil
}

func deploymentDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return deploymentDialContextWithResolver(ctx, network, address, net.DefaultResolver.LookupIPAddr)
}

func deploymentDialContextWithResolver(ctx context.Context, network, address string, lookup func(context.Context, string) ([]net.IPAddr, error)) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return nil, errors.New("deployment destination address is invalid")
	}
	var addresses []net.IPAddr
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		addresses = []net.IPAddr{{IP: ip}}
	} else {
		if lookup == nil {
			return nil, errors.New("deployment destination resolver is unavailable")
		}
		addresses, err = lookup(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("deployment destination lookup failed: %w", err)
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("deployment destination has no addresses")
	}
	allowLoopback, _ := ctx.Value(deploymentLoopbackContextKey{}).(bool)
	for _, address := range addresses {
		if address.IP == nil || (allowLoopback && !address.IP.IsLoopback()) {
			return nil, errors.New("deployment loopback destination resolves outside loopback")
		}
		if !allowLoopback && unsafeEgressIP(address.IP) {
			return nil, errors.New("deployment destination resolves to a private address")
		}
	}
	var lastErr error
	for _, address := range addresses {
		if network == "tcp4" && address.IP.To4() == nil {
			continue
		}
		if network == "tcp6" && address.IP.To4() != nil {
			continue
		}
		target := net.JoinHostPort(address.IP.String(), port)
		if address.Zone != "" {
			target = net.JoinHostPort(address.IP.String()+"%"+address.Zone, port)
		}
		conn, dialErr := dialer.DialContext(ctx, network, target)
		if dialErr != nil {
			lastErr = dialErr
			continue
		}
		remoteHost, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		remoteIP := net.ParseIP(strings.Trim(remoteHost, "[]"))
		if splitErr != nil || remoteIP == nil || !remoteIP.Equal(address.IP) || (allowLoopback && !remoteIP.IsLoopback()) || (!allowLoopback && unsafeEgressIP(remoteIP)) {
			_ = conn.Close()
			lastErr = errors.New("deployment connected address was not approved")
			continue
		}
		return conn, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("deployment destination has no address for requested network")
}

func deploymentPrivateIP(ip net.IP) bool {
	return unsafeEgressIP(ip)
}

func (m *DeploymentManager) Register(config DeployConfig) error {
	if m == nil {
		return errors.New("deployment manager is unavailable")
	}
	config.ID = strings.TrimSpace(config.ID)
	config.OrganizationID = strings.TrimSpace(config.OrganizationID)
	config.Provider = strings.ToLower(strings.TrimSpace(config.Provider))
	if !validDeploymentIdentifier(config.ID) || config.Provider == "" {
		return errors.New("deployment id and provider are required")
	}
	if (config.ProjectID != "" && !validDeploymentIdentifier(config.ProjectID)) || (config.AccountID != "" && !validDeploymentIdentifier(config.AccountID)) {
		return errors.New("deployment project or account id is invalid")
	}
	if config.Provider != "vercel" && config.Provider != "netlify" && config.Provider != "generic" {
		return fmt.Errorf("unsupported deployment provider %q", config.Provider)
	}
	base, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || base == nil || base.Host == "" || base.User != nil {
		return errors.New("deployment base_url must be HTTPS or loopback HTTP without userinfo")
	}
	if err := validateConfiguredEndpointURL(config.BaseURL); err != nil {
		return fmt.Errorf("deployment base_url is not safe to persist: %w", err)
	}
	loopbackHTTP := base.Scheme == "http" && isLoopbackHost(base.Hostname())
	if base.Scheme != "https" && !loopbackHTTP {
		return errors.New("deployment base_url must be HTTPS or loopback HTTP without userinfo")
	}
	if config.TokenEnv != "" && !validEnvName(config.TokenEnv) {
		return errors.New("invalid deployment token_env")
	}
	if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > 600 {
		config.TimeoutSeconds = 120
	}
	m.mu.Lock()
	m.configs[config.ID] = config
	m.mu.Unlock()
	return nil
}

func (m *DeploymentManager) List() []DeployConfig {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]DeployConfig, 0, len(m.configs))
	for _, config := range m.configs {
		config.BaseURL = providerCatalogOrigin(config.BaseURL)
		config.TokenEnv = ""
		result = append(result, config)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (m *DeploymentManager) ListForOrganization(organizationID string) []DeployConfig {
	organizationID = strings.TrimSpace(organizationID)
	result := make([]DeployConfig, 0)
	if m == nil || organizationID == "" {
		return result
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, config := range m.configs {
		if strings.TrimSpace(config.OrganizationID) != organizationID {
			continue
		}
		config.BaseURL = providerCatalogOrigin(config.BaseURL)
		config.TokenEnv = ""
		result = append(result, config)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (m *DeploymentManager) Deploy(ctx context.Context, providerID string, request DeploymentRequest) (DeploymentResult, error) {
	return m.deploy(ctx, "", providerID, request, true)
}

func (m *DeploymentManager) DeployForOrganization(ctx context.Context, organizationID, providerID string, request DeploymentRequest) (DeploymentResult, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return DeploymentResult{}, ErrDeploymentProviderNotFound
	}
	return m.deploy(ctx, organizationID, providerID, request, true)
}

func (m *DeploymentManager) deploy(ctx context.Context, organizationID, providerID string, request DeploymentRequest, scoped bool) (DeploymentResult, error) {
	if m == nil {
		return DeploymentResult{}, errors.New("deployment manager is unavailable")
	}
	m.mu.RLock()
	config, ok := m.configs[strings.TrimSpace(providerID)]
	m.mu.RUnlock()
	if !ok || scoped && strings.TrimSpace(config.OrganizationID) != organizationID {
		return DeploymentResult{}, ErrDeploymentProviderNotFound
	}
	files, manifest, err := collectDeploySnapshot(request.Root)
	if err != nil {
		return DeploymentResult{}, err
	}
	if len(files) == 0 {
		return DeploymentResult{}, errors.New("deployment workspace has no files")
	}
	if expected := strings.TrimSpace(request.ManifestSHA256); expected != "" && !strings.EqualFold(expected, manifest.SHA256) {
		return DeploymentResult{}, errors.New("deployment workspace changed after approval")
	}
	if err := validateOutboundPayload(map[string]any{"name": request.Name, "target": request.Target}); err != nil {
		return DeploymentResult{}, err
	}
	for _, file := range files {
		if err := validateOutboundPayloadWithLimit(string(file.Data), 50<<20); err != nil {
			return DeploymentResult{}, fmt.Errorf("deployment file content blocked by DLP policy")
		}
	}
	var result DeploymentResult
	switch config.Provider {
	case "vercel":
		result, err = m.deployVercel(ctx, config, request, files)
	case "netlify":
		result, err = m.deployNetlify(ctx, config, request, files)
	default:
		result, err = m.deployGeneric(ctx, config, request, files)
	}
	result = sanitizeDeploymentResult(result)
	if err != nil {
		result.Provider = config.Provider
		if result.Status == "" {
			result.Status = "unknown"
		}
		if result.Status == "unknown" && result.Files == 0 {
			result.Files = len(files)
		}
		return result, &DeploymentError{Result: result, Err: err}
	}
	result.Provider = config.Provider
	result.Files = len(files)
	return result, nil
}

type deployFile struct {
	Path string
	Data []byte
}

func collectDeployFiles(root string) ([]deployFile, error) {
	files, _, err := collectDeploySnapshot(root)
	return files, err
}

func BuildDeploymentManifest(root string) (DeploymentManifest, error) {
	_, manifest, err := collectDeploySnapshot(root)
	return manifest, err
}

func collectDeploySnapshot(root string) ([]deployFile, DeploymentManifest, error) {
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || root == "." || root == string(filepath.Separator) {
		return nil, DeploymentManifest{}, errors.New("invalid deployment root")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, DeploymentManifest{}, errors.New("deployment root is not a directory")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, DeploymentManifest{}, errors.New("deployment root must not be a symlink")
	}
	var files []deployFile
	manifest := DeploymentManifest{Included: []DeploymentManifestEntry{}, Excluded: []DeploymentManifestEntry{}}
	var total int64
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			if relative == "." && path == root {
				return nil
			}
			return errors.New("deployment path escaped root")
		}
		relative = filepath.ToSlash(relative)
		if info.IsDir() {
			if relative != "." {
				if reason := deploymentPathExclusionReason(relative); reason != "" {
					manifest.Excluded = append(manifest.Excluded, DeploymentManifestEntry{Path: relative, Reason: reason})
					return filepath.SkipDir
				}
			}
			if deploymentPathContainsPrivateDirectory(relative) {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("deployment workspace contains a non-regular file")
		}
		if reason := deploymentPathExclusionReason(relative); reason != "" {
			manifest.Excluded = append(manifest.Excluded, DeploymentManifestEntry{Path: relative, Size: info.Size(), Reason: reason})
			return nil
		}
		if len(files) >= 2000 || total+info.Size() > 50<<20 {
			return errors.New("deployment workspace exceeds file or size limit")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, deployFile{Path: relative, Data: data})
		digest := sha256.Sum256(data)
		manifest.Included = append(manifest.Included, DeploymentManifestEntry{Path: relative, Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
		total += int64(len(data))
		return nil
	})
	if err != nil {
		return nil, DeploymentManifest{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	sort.Slice(manifest.Included, func(i, j int) bool { return manifest.Included[i].Path < manifest.Included[j].Path })
	sort.Slice(manifest.Excluded, func(i, j int) bool { return manifest.Excluded[i].Path < manifest.Excluded[j].Path })
	manifestPayload, err := json.Marshal(struct {
		Included []DeploymentManifestEntry `json:"included"`
		Excluded []DeploymentManifestEntry `json:"excluded"`
	}{Included: manifest.Included, Excluded: manifest.Excluded})
	if err != nil {
		return nil, DeploymentManifest{}, err
	}
	manifestDigest := sha256.Sum256(manifestPayload)
	manifest.SHA256 = hex.EncodeToString(manifestDigest[:])
	return files, manifest, nil
}

func deploymentPathContainsPrivateDirectory(relative string) bool {
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		switch strings.ToLower(part) {
		case ".git", ".hg", ".svn", ".agent", ".ollama", ".secrets", "node_modules":
			return true
		}
	}
	return false
}

func deploymentPathExclusionReason(relative string) string {
	if deploymentPathContainsPrivateDirectory(relative) {
		return "private-directory"
	}
	base := strings.ToLower(filepath.Base(relative))
	if base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".crt") || strings.HasSuffix(base, ".p12") || strings.HasSuffix(base, ".pfx") || strings.HasSuffix(base, ".log") || strings.HasSuffix(base, ".backup") || strings.HasSuffix(base, ".bak") || strings.HasSuffix(base, ".sqlite") || strings.HasSuffix(base, ".sqlite3") || strings.HasSuffix(base, ".db") || strings.HasSuffix(base, ".dump") || strings.HasSuffix(base, ".sql") {
		return "private-extension"
	}
	for _, marker := range []string{"secret", "credential", "password", "apikey", "api_key"} {
		if strings.Contains(base, marker) {
			return "private-name-marker"
		}
	}
	if base == "token" || strings.HasPrefix(base, "token.") || strings.Contains(base, "-token.") || strings.Contains(base, "_token.") || strings.Contains(base, ".token.") {
		return "private-name-marker"
	}
	return ""
}

func (m *DeploymentManager) request(ctx context.Context, config DeployConfig, method, endpoint string, body []byte, contentType string) (map[string]any, error) {
	token := ""
	if config.TokenEnv != "" {
		value, ok := os.LookupEnv(config.TokenEnv)
		if !ok || strings.TrimSpace(value) == "" {
			return nil, errors.New("deployment provider credential is unavailable")
		}
		token = strings.TrimSpace(value)
	}
	base, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}
	relative, err := url.Parse(endpoint)
	if err != nil || relative.IsAbs() || !validConnectorPath(relative.Path) {
		return nil, errors.New("invalid deployment endpoint")
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + relative.Path
	base.RawQuery = relative.RawQuery
	requestContext, err := deploymentRequestContext(ctx, base.String())
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(requestContext, method, base.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := *m.client
	client.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("deployment provider request failed")
	}
	defer resp.Body.Close()
	const maxDeploymentResponseBytes = 4 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDeploymentResponseBytes+1))
	if err != nil {
		return nil, errors.New("deployment provider response could not be read")
	}
	if len(data) > maxDeploymentResponseBytes {
		return nil, errors.New("deployment provider response exceeded the size limit")
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("deployment provider returned HTTP status %d", resp.StatusCode)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, errors.New("deployment provider returned malformed JSON")
	}
	if payload == nil {
		return nil, errors.New("deployment provider response must be a JSON object")
	}
	return payload, nil
}

func (m *DeploymentManager) deployGeneric(ctx context.Context, config DeployConfig, request DeploymentRequest, files []deployFile) (DeploymentResult, error) {
	payload := map[string]any{"name": request.Name, "target": request.Target, "files": encodeFiles(files)}
	body, _ := json.Marshal(payload)
	response, err := m.request(ctx, config, http.MethodPost, "/deploy", body, "application/json")
	if err != nil {
		return DeploymentResult{}, err
	}
	result := resultFromPayload(response)
	if result.DeploymentID == "" && result.URL == "" {
		return DeploymentResult{}, errors.New("deployment provider response omitted deployment identity")
	}
	return result, nil
}

func (m *DeploymentManager) deployVercel(ctx context.Context, config DeployConfig, request DeploymentRequest, files []deployFile) (DeploymentResult, error) {
	payload := map[string]any{"name": request.Name, "target": request.Target, "files": encodeFiles(files)}
	if config.ProjectID != "" {
		payload["project"] = config.ProjectID
	}
	body, _ := json.Marshal(payload)
	endpoint := "/v13/deployments"
	if config.AccountID != "" {
		endpoint += "?teamId=" + url.QueryEscape(config.AccountID)
	}
	response, err := m.request(ctx, config, http.MethodPost, endpoint, body, "application/json")
	if err != nil {
		return DeploymentResult{}, err
	}
	result := resultFromPayload(response)
	if result.DeploymentID == "" && result.URL == "" {
		return DeploymentResult{}, errors.New("vercel response omitted deployment identity")
	}
	return result, nil
}

func (m *DeploymentManager) deployNetlify(ctx context.Context, config DeployConfig, request DeploymentRequest, files []deployFile) (DeploymentResult, error) {
	siteID := config.ProjectID
	var site map[string]any
	var err error
	if siteID == "" {
		createPayload, _ := json.Marshal(map[string]any{"name": request.Name})
		site, err = m.request(ctx, config, http.MethodPost, "/api/v1/sites", createPayload, "application/json")
		if err != nil {
			return DeploymentResult{}, err
		}
		siteID = firstString(site, "id", "site_id")
	}
	if siteID == "" {
		return DeploymentResult{}, errors.New("netlify response has no site id")
	}
	if !validDeploymentIdentifier(siteID) {
		return DeploymentResult{}, errors.New("netlify response has an invalid site id")
	}
	digests := map[string]string{}
	for _, file := range files {
		digest := sha1.Sum(file.Data)
		digests["/"+file.Path] = hex.EncodeToString(digest[:])
	}
	deployPayload, _ := json.Marshal(map[string]any{"files": digests})
	deploy, err := m.request(ctx, config, http.MethodPost, "/api/v1/sites/"+url.PathEscape(siteID)+"/deploys", deployPayload, "application/json")
	if err != nil {
		return DeploymentResult{DeploymentID: siteID, Status: "partial"}, fmt.Errorf("netlify deployment creation failed after site creation: %w", err)
	}
	deployID := firstString(deploy, "id", "deploy_id")
	if deployID == "" {
		return DeploymentResult{DeploymentID: siteID, Status: "partial"}, errors.New("netlify response has no deployment id after site/deploy side effects")
	}
	if !validDeploymentIdentifier(deployID) {
		return DeploymentResult{DeploymentID: siteID, Status: "partial"}, errors.New("netlify response has an invalid deployment id after site/deploy side effects")
	}
	result := resultFromPayload(deploy)
	result.Status = "partial"
	result.Files = 0
	for _, file := range files {
		endpoint := "/api/v1/deploys/" + url.PathEscape(deployID) + "/files/" + url.PathEscape(file.Path)
		if _, err := m.request(ctx, config, http.MethodPut, endpoint, file.Data, "application/octet-stream"); err != nil {
			return result, fmt.Errorf("netlify file upload failed after %d files: %w", result.Files, err)
		}
		result.Files++
	}
	result.Status = firstString(deploy, "status", "state")
	if result.Status == "" {
		result.Status = "accepted"
	}
	return result, nil
}

func encodeFiles(files []deployFile) []map[string]string {
	encoded := make([]map[string]string, 0, len(files))
	for _, file := range files {
		encoded = append(encoded, map[string]string{"file": file.Path, "data": base64.StdEncoding.EncodeToString(file.Data)})
	}
	return encoded
}

func resultFromPayload(payload map[string]any) DeploymentResult {
	return DeploymentResult{DeploymentID: firstString(payload, "id", "deployment_id", "deploy_id"), URL: firstString(payload, "url", "deploy_url", "ssl_url"), Status: firstString(payload, "status", "state")}
}

func sanitizeDeploymentResult(result DeploymentResult) DeploymentResult {
	if result.DeploymentID != "" && !validDeploymentIdentifier(result.DeploymentID) {
		result.DeploymentID = "[REDACTED]"
	} else {
		result.DeploymentID = RedactDLP(result.DeploymentID)
	}
	if len(result.Status) > 64 || strings.ContainsAny(result.Status, "\r\n\x00") {
		result.Status = "[REDACTED]"
	} else {
		result.Status = RedactDLP(result.Status)
	}
	result.URL = sanitizeProviderURL(result.URL)
	return result
}

func validDeploymentIdentifier(value string) bool {
	if value == "" || len(value) > 128 || strings.Contains(value, "..") {
		return false
	}
	for index, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || (index > 0 && (r == '-' || r == '_' || r == '.')) {
			continue
		}
		return false
	}
	return true
}

func sanitizeProviderURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Fragment != "" || endpointURLHasSensitiveMaterial(raw) {
		return "[REDACTED]"
	}
	for key := range parsed.Query() {
		key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", ""), "_", ""))
		if strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.Contains(key, "credential") || strings.Contains(key, "apikey") || strings.Contains(key, "accesskey") || strings.Contains(key, "signature") || key == "sig" {
			return "[REDACTED]"
		}
	}
	return RedactDLP(raw)
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
