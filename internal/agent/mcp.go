package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MCPServerConfig struct {
	ID               string   `json:"id"`
	OrganizationID   string   `json:"organization_id,omitempty"`
	Command          string   `json:"command"`
	Args             []string `json:"args,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty"`
	AllowedMethods   []string `json:"allowed_methods,omitempty"`
	// EnvironmentVars is an explicit allowlist; listed values are intentionally shared with this configured process.
	EnvironmentVars []string `json:"environment_vars,omitempty"`
	TimeoutSeconds  int      `json:"timeout_seconds,omitempty"`
	Disabled        bool     `json:"disabled,omitempty"`
}

// ErrMCPStrictSandboxNotConfigured is returned instead of starting a
// tenant-scoped local process when MCP stdio lacks a complete namespace,
// network and seccomp launcher. A plain host process is not a tenant boundary.
var ErrMCPStrictSandboxNotConfigured = errors.New("MCP local execution is BLOCKED_EXTERNAL: strict sandbox is not configured")

type MCPManager struct {
	mu          sync.RWMutex
	servers     map[string]*MCPServer
	persistPath string
}

func NewMCPManager() *MCPManager {
	return &MCPManager{servers: make(map[string]*MCPServer)}
}

func NewPersistentMCPManager(manifestPath string) (*MCPManager, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	manager := NewMCPManager()
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
		var configs []MCPServerConfig
		if err := decoder.Decode(&configs); err != nil {
			return nil, fmt.Errorf("decode MCP manifest: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return nil, errors.New("MCP manifest contains trailing JSON")
			}
			return nil, fmt.Errorf("decode MCP manifest trailing data: %w", err)
		}
		for _, config := range configs {
			if err := manager.Register(config); err != nil {
				return nil, fmt.Errorf("load MCP server %q: %w", config.ID, err)
			}
		}
	}
	manager.persistPath = manifestPath
	return manager, nil
}

func (m *MCPManager) Register(config MCPServerConfig) error {
	return m.register(config, "")
}

func (m *MCPManager) RegisterForOrganization(organizationID string, config MCPServerConfig) error {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return errors.New("MCP organization scope is required")
	}
	if supplied := strings.TrimSpace(config.OrganizationID); supplied != "" && supplied != organizationID {
		return ErrPluginOrganizationScope
	}
	config.OrganizationID = organizationID
	return m.register(config, organizationID)
}

func (m *MCPManager) register(config MCPServerConfig, organizationID string) error {
	config.OrganizationID = strings.TrimSpace(config.OrganizationID)
	if strings.TrimSpace(config.ID) == "" || strings.TrimSpace(config.Command) == "" {
		return errors.New("MCP server id and command are required")
	}
	if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > 300 {
		config.TimeoutSeconds = 30
	}
	config.Command = strings.TrimSpace(config.Command)
	if !filepath.IsAbs(config.Command) {
		return errors.New("MCP command must be an absolute executable path")
	}
	commandInfo, err := os.Lstat(config.Command)
	if err != nil || commandInfo.Mode()&os.ModeSymlink != 0 || commandInfo.IsDir() || !isExecutableMode(config.Command, commandInfo) {
		return errors.New("MCP command must be an executable regular file")
	}
	if len(config.Args) > 64 {
		return errors.New("MCP args limit exceeded")
	}
	for index, arg := range config.Args {
		if strings.IndexByte(arg, 0) >= 0 || len(arg) > 4096 {
			return errors.New("MCP argument is invalid or too long")
		}
		if mcpArgumentContainsCredential(arg) || strings.HasPrefix(strings.TrimSpace(arg), "-") && index+1 < len(config.Args) && mcpArgumentKeyIsCredential(arg) {
			return errors.New("MCP arguments must not contain literal credentials; configure an environment variable instead")
		}
	}
	config.Args = append([]string(nil), config.Args...)
	if len(config.AllowedMethods) == 0 {
		return errors.New("MCP allowed_methods must contain at least one method")
	}
	allowed := make(map[string]bool, len(config.AllowedMethods))
	for _, method := range config.AllowedMethods {
		method = strings.TrimSpace(method)
		if method == "" {
			return errors.New("MCP allowed_methods cannot contain empty methods")
		}
		allowed[method] = true
	}
	normalizedEnvironmentVars := make([]string, 0, len(config.EnvironmentVars))
	if len(config.EnvironmentVars) > 32 {
		return errors.New("MCP environment variable allowlist limit exceeded")
	}
	for _, name := range config.EnvironmentVars {
		name = strings.TrimSpace(name)
		if !validEnvName(name) {
			return fmt.Errorf("invalid MCP environment variable %q", name)
		}
		normalizedEnvironmentVars = append(normalizedEnvironmentVars, name)
	}
	config.EnvironmentVars = normalizedEnvironmentVars
	workingDirectory, err := prepareMCPWorkingDirectory(config.ID, config.WorkingDirectory)
	if err != nil {
		return err
	}
	config.WorkingDirectory = workingDirectory.path
	m.mu.Lock()
	defer m.mu.Unlock()
	if organizationID != "" {
		if existing := m.servers[config.ID]; existing != nil && !pluginOwnedByOrganization(existing.config.OrganizationID, organizationID) {
			return ErrPluginOrganizationScope
		}
	}
	old := m.servers[config.ID]
	if old != nil {
		_ = old.Stop()
	}
	m.servers[config.ID] = &MCPServer{config: config, allowedMethods: allowed, cleanupDirectory: workingDirectory.cleanup}
	if err := m.persistLocked(); err != nil {
		delete(m.servers, config.ID)
		if old != nil {
			m.servers[config.ID] = old
		}
		return fmt.Errorf("persist MCP manifest: %w", err)
	}
	return nil
}

func (m *MCPManager) persistLocked() error {
	if strings.TrimSpace(m.persistPath) == "" {
		return nil
	}
	configs := make([]MCPServerConfig, 0, len(m.servers))
	for _, server := range m.servers {
		config := server.config
		if server.cleanupDirectory {
			config.WorkingDirectory = ""
		}
		config.Args = append([]string(nil), config.Args...)
		config.EnvironmentVars = append([]string(nil), config.EnvironmentVars...)
		configs = append(configs, config)
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	if err := os.MkdirAll(filepath.Dir(m.persistPath), 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(m.persistPath, configs)
}

func (m *MCPManager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, server := range m.servers {
		_ = server.Stop()
	}
}

func (m *MCPManager) List() []MCPServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]MCPServerConfig, 0, len(m.servers))
	for _, server := range m.servers {
		config := server.config
		config.EnvironmentVars = append([]string(nil), config.EnvironmentVars...)
		result = append(result, config)
	}
	return result
}

func (m *MCPManager) ListForOrganization(organizationID string) []MCPServerConfig {
	organizationID = strings.TrimSpace(organizationID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]MCPServerConfig, 0)
	for _, server := range m.servers {
		if !pluginAccessibleByOrganization(server.config.OrganizationID, organizationID) {
			continue
		}
		config := server.config
		config.Args = nil
		config.EnvironmentVars = nil
		result = append(result, config)
	}
	return result
}

func (m *MCPManager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	server, ok := m.servers[strings.TrimSpace(id)]
	if !ok {
		return fmt.Errorf("MCP server %q is not registered", id)
	}
	if !pluginGlobal(server.config.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	server.config.Disabled = !enabled
	if !enabled {
		_ = server.Stop()
	}
	if err := m.persistLocked(); err != nil {
		server.config.Disabled = !server.config.Disabled
		return fmt.Errorf("persist MCP manifest: %w", err)
	}
	return nil
}

func (m *MCPManager) SetEnabledForOrganization(organizationID, id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	server, ok := m.servers[strings.TrimSpace(id)]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(server.config.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	server.config.Disabled = !enabled
	if !enabled {
		_ = server.Stop()
	}
	if err := m.persistLocked(); err != nil {
		server.config.Disabled = !server.config.Disabled
		return fmt.Errorf("persist MCP manifest: %w", err)
	}
	return nil
}

func (m *MCPManager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	server, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("MCP server %q is not registered", id)
	}
	if !pluginGlobal(server.config.OrganizationID) {
		return ErrPluginOrganizationScope
	}
	_ = server.Stop()
	delete(m.servers, id)
	if err := m.persistLocked(); err != nil {
		m.servers[id] = server
		return fmt.Errorf("persist MCP manifest: %w", err)
	}
	return nil
}

func (m *MCPManager) RemoveForOrganization(organizationID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	server, ok := m.servers[id]
	if !ok {
		return ErrPluginNotFound
	}
	if !pluginOwnedByOrganization(server.config.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	_ = server.Stop()
	delete(m.servers, id)
	if err := m.persistLocked(); err != nil {
		m.servers[id] = server
		return fmt.Errorf("persist MCP manifest: %w", err)
	}
	return nil
}

func (m *MCPManager) Call(ctx context.Context, serverID, method string, params any) (json.RawMessage, error) {
	return nil, ErrPluginOrganizationScope
}

func (m *MCPManager) CallForOrganization(ctx context.Context, organizationID, serverID, method string, params any) (json.RawMessage, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, ErrPluginOrganizationScope
	}
	m.mu.RLock()
	server := m.servers[serverID]
	m.mu.RUnlock()
	if server == nil {
		return nil, fmt.Errorf("MCP server %q is not registered", serverID)
	}
	server.mu.Lock()
	serverOrganizationID := server.config.OrganizationID
	server.mu.Unlock()
	if !pluginOwnedByOrganization(serverOrganizationID, organizationID) {
		return nil, ErrPluginOrganizationScope
	}
	// Do not silently downgrade a tenant call to an unsandboxed host process.
	// CallGlobal remains available only for explicitly trusted single-user/local
	// execution until MCP has a launcher equivalent to sandbox.exec.
	return nil, ErrMCPStrictSandboxNotConfigured
}

// CallGlobal is reserved for explicitly trusted single-user/local execution.
// Tenant requests must supply their organization through CallForOrganization.
func (m *MCPManager) CallGlobal(ctx context.Context, serverID, method string, params any) (json.RawMessage, error) {
	return m.callWithScope(ctx, "", serverID, method, params, true)
}

func (m *MCPManager) callWithScope(ctx context.Context, organizationID, serverID, method string, params any, global bool) (json.RawMessage, error) {
	m.mu.RLock()
	server := m.servers[serverID]
	m.mu.RUnlock()
	if server == nil {
		return nil, fmt.Errorf("MCP server %q is not registered", serverID)
	}
	server.mu.Lock()
	disabled := server.config.Disabled
	serverOrganizationID := server.config.OrganizationID
	server.mu.Unlock()
	if global && !pluginGlobal(serverOrganizationID) || !global && !pluginOwnedByOrganization(serverOrganizationID, organizationID) {
		return nil, ErrPluginOrganizationScope
	}
	if disabled {
		return nil, errors.New("MCP server is disabled")
	}
	return server.Call(ctx, method, params)
}

type MCPServer struct {
	mu               sync.Mutex
	config           MCPServerConfig
	allowedMethods   map[string]bool
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	stdout           *bufio.Reader
	stderr           *mcpStderrBuffer
	cancel           context.CancelFunc
	nextID           int64
	cleanupDirectory bool
}

const (
	mcpMaxMessageBytes = 4 << 20
	mcpMaxStderrBytes  = 64 << 10
)

type mcpStderrBuffer struct {
	mu sync.Mutex
	bytes.Buffer
	limit int
}

func (b *mcpStderrBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit > b.Len() {
		remaining := b.limit - b.Len()
		if len(data) > remaining {
			_, _ = b.Buffer.Write(data[:remaining])
		} else {
			_, _ = b.Buffer.Write(data)
		}
	}
	return len(data), nil
}

func (b *mcpStderrBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

func (b *mcpStderrBuffer) ReadFrom(reader io.Reader) (int64, error) {
	buffer := make([]byte, 32<<10)
	var total int64
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			_, _ = b.Write(buffer[:read])
			total += int64(read)
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func (s *MCPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked()
}

func (s *MCPServer) startLocked() error {
	if s.cmd != nil {
		return nil
	}
	if s.cleanupDirectory {
		if err := os.MkdirAll(s.config.WorkingDirectory, 0o700); err != nil {
			return fmt.Errorf("recreate MCP working directory: %w", err)
		}
	}
	processContext, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processContext, s.config.Command, s.config.Args...)
	cmd.Dir = s.config.WorkingDirectory
	configureMCPProcess(cmd)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/tmp"}
	for _, name := range s.config.EnvironmentVars {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, "=\x00\r\n") {
			cancel()
			return fmt.Errorf("invalid MCP environment variable %q", name)
		}
		if value, ok := os.LookupEnv(name); ok {
			if len(value) > 32<<10 {
				cancel()
				return errors.New("MCP environment variable value exceeds limit")
			}
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		cancel()
		return err
	}
	stderr := &mcpStderrBuffer{limit: mcpMaxStderrBytes}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		cancel()
		return err
	}
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReaderSize(stdout, 64<<10)
	s.stderr = stderr
	s.cancel = cancel
	return nil
}

func (s *MCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

func (s *MCPServer) stopLocked() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	stderr := ""
	if s.stderr != nil {
		stderr = strings.TrimSpace(RedactDLP(s.stderr.String()))
	}
	var err error
	if s.cmd != nil {
		_ = terminateMCPProcess(s.cmd)
		err = s.cmd.Wait()
	}
	if s.cleanupDirectory && s.config.WorkingDirectory != "" {
		_ = os.RemoveAll(s.config.WorkingDirectory)
	}
	s.cmd, s.stdin, s.stdout, s.stderr, s.cancel = nil, nil, nil, nil, nil
	if err != nil && stderr != "" {
		return fmt.Errorf("%w: %s", err, stderr)
	}
	return err
}

func (s *MCPServer) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("MCP method is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.allowedMethods) == 0 || !s.allowedMethods[method] {
		return nil, fmt.Errorf("MCP method %q is not allowlisted", method)
	}
	if err := validateOutboundPayloadWithLimit(params, mcpMaxMessageBytes); err != nil {
		return nil, err
	}
	s.nextID++
	requestID := s.nextID
	request := map[string]any{"jsonrpc": "2.0", "id": requestID, "method": method, "params": params}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(data) > mcpMaxMessageBytes {
		return nil, errors.New("MCP request payload limit exceeded")
	}
	if err := s.startLocked(); err != nil {
		return nil, err
	}
	if _, err := s.stdin.Write(append(data, '\n')); err != nil {
		_ = s.stopLocked()
		return nil, err
	}
	resultChannel := make(chan mcpResponse, 1)
	// Capture the reader now: stopLocked (on timeout or cancellation) clears
	// s.stdout, and reading the field from the goroutine raced with it and
	// could dereference nil or read a restarted process's output.
	stdout := s.stdout
	go func() {
		for {
			line, err := readMCPMessage(stdout)
			if err != nil {
				resultChannel <- mcpResponse{err: err}
				return
			}
			var response mcpResponse
			if err := json.Unmarshal(line, &response); err != nil {
				resultChannel <- mcpResponse{err: err}
				return
			}
			if response.ID == nil {
				// JSON-RPC notifications do not carry an id and are not the
				// response to the request currently being served.
				continue
			}
			resultChannel <- response
			return
		}
	}()
	timeout := time.Duration(s.config.TimeoutSeconds) * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}
	select {
	case <-ctx.Done():
		_ = s.stopLocked()
		return nil, ctx.Err()
	case <-time.After(timeout):
		_ = s.stopLocked()
		return nil, errors.New("MCP call timed out")
	case response := <-resultChannel:
		if response.err != nil {
			_ = s.stopLocked()
			return nil, response.err
		}
		if response.ID == nil || *response.ID != requestID {
			_ = s.stopLocked()
			return nil, errors.New("MCP response id does not match request")
		}
		if response.Error != nil {
			return nil, errors.New("MCP server returned an error")
		}
		if len(bytes.TrimSpace(response.Result)) == 0 {
			return nil, errors.New("MCP response omitted result")
		}
		result, err := sanitizeProviderJSON(response.Result)
		if err != nil {
			return nil, errors.New("MCP server returned an invalid result")
		}
		return result, nil
	}
}

func readMCPMessage(reader *bufio.Reader) ([]byte, error) {
	message := make([]byte, 0, 4096)
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(message)+len(fragment) > mcpMaxMessageBytes {
			return nil, errors.New("MCP response payload limit exceeded")
		}
		message = append(message, fragment...)
		if err == nil {
			return bytes.TrimSpace(message), nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return nil, err
	}
}

type mcpResponse struct {
	ID     *int64          `json:"id,omitempty"`
	Result json.RawMessage `json:"result"`
	Error  *mcpError       `json:"error,omitempty"`
	err    error
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpCallTool struct{ manager *MCPManager }

func (t mcpCallTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "mcp.call", Version: "1", Description: "Chamar método allowlisted de servidor MCP stdio", Risk: RiskExternalSideEffect, RequiresApproval: true, Scopes: []string{"mcp:call"}}
}

func (t mcpCallTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if t.manager == nil {
		return ToolResult{}, errors.New("MCP manager is unavailable")
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

func mcpArgumentContainsCredential(arg string) bool {
	if len(ScanDLP(arg)) != 0 {
		return true
	}
	if separator := strings.IndexByte(arg, '='); separator >= 0 && mcpArgumentKeyIsCredential(arg[:separator]) {
		return true
	}
	return false
}

func mcpArgumentKeyIsCredential(key string) bool {
	key = strings.TrimLeft(strings.TrimSpace(key), "-")
	return sensitiveDLPKey(key)
}
