package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Runtime struct {
	store               Store
	planner             Planner
	plannerResolver     PlannerResolver
	tools               *Registry
	capabilityPolicy    CapabilityPolicy
	workspaceRoot       string
	dataRoot            string
	organizationScope   string
	context             *ContextStore
	company             *CompanyStore
	remoteMCP           *RemoteMCPManager
	metrics             *RuntimeMetrics
	connectors          *ConnectorManager
	mcp                 *MCPManager
	queue               *JobQueue
	redisQueue          *RedisQueue
	traces              *TraceStore
	telemetry           *Telemetry
	media               *MediaManager
	builder             *BuilderService
	collaboration       *CollaborationStore
	orchestrator        *AgentOrchestrator
	research            *ResearchEngine
	devices             *DeviceStore
	uploads             *UploadManager
	projectImporter     *ProjectImporter
	ingestion           DocumentIngestor
	push                *PushService
	pushOutbox          *PushOutbox
	deployments         *DeploymentManager
	deploymentApprovals *DeploymentApprovalStore
	webhookReplay       *WebhookReplayStore
	authStore           *AuthStore
	whatsapp            *WhatsAppGateway
	supervisor          *Supervisor
	mu                  *sync.Mutex
	running             map[string]bool
	activeCancels       map[string]context.CancelFunc
	// cancelPollInterval is how often a running mission re-reads the durable
	// store to detect a cancellation requested by ANOTHER instance and abort
	// the in-flight step. Zero falls back to the default.
	cancelPollInterval time.Duration
}

type RuntimeConfig struct {
	Store               Store
	Planner             Planner
	PlannerResolver     PlannerResolver
	Tools               *Registry
	WorkspaceRoot       string
	DataRoot            string
	Context             *ContextStore
	Company             *CompanyStore
	RemoteMCP           *RemoteMCPManager
	Connectors          *ConnectorManager
	MCP                 *MCPManager
	Queue               *JobQueue
	RedisQueue          *RedisQueue
	Traces              *TraceStore
	Telemetry           *Telemetry
	Media               *MediaManager
	Builder             *BuilderService
	Collaboration       *CollaborationStore
	Devices             *DeviceStore
	Push                *PushService
	PushOutbox          *PushOutbox
	Deployments         *DeploymentManager
	DeploymentApprovals *DeploymentApprovalStore
	WebhookReplay       *WebhookReplayStore
	WhatsApp            *WhatsAppGateway
	SupervisorConfig    *SupervisorConfig
}

var (
	ErrApprovalVersionConflict = errors.New("approval mission version conflict")
	ErrApprovalNonceMismatch   = errors.New("approval nonce mismatch")
	ErrApprovalReasonTooLong   = errors.New("approval reason exceeds 2048 bytes")
	ErrQueueJobForbidden       = errors.New("job is outside the active organization")
	ErrQueueNonRetryable       = errors.New("queue job must not be retried automatically")
	ErrMissionTerminal         = errors.New("mission is terminal and cannot be resumed automatically")
	ErrMissionNotRunnable      = errors.New("mission is not in a runnable state")
)

func NewRuntime(config RuntimeConfig) (*Runtime, error) {
	store := config.Store
	if store == nil {
		store = NewMemoryStore()
	}
	if usesPostgresStore(store) && !postgresTenantRuntimeReady(store) {
		return nil, ErrPostgresTenantIsolationUnavailable
	}
	if usesPostgresStore(store) && config.RedisQueue == nil {
		return nil, errors.New("PostgreSQL multi-tenant runtime requires the owner-bound Redis queue")
	}
	planner := config.Planner
	if planner == nil {
		planner = UnconfiguredPlanner{}
	}
	tools := config.Tools
	if tools == nil {
		tools = NewRegistry()
	}
	capabilityPolicy := DefaultCapabilityPolicy()
	if config.Connectors != nil {
		tools.Register(connectorTool{manager: config.Connectors})
	}
	if config.MCP != nil {
		tools.Register(mcpCallTool{manager: config.MCP})
	}
	if config.RemoteMCP != nil {
		tools.Register(remoteMCPCallTool{manager: config.RemoteMCP})
	}
	if config.Media != nil {
		tools.Register(mediaProcessTool{manager: config.Media})
	}
	for _, descriptor := range tools.Descriptors() {
		if err := capabilityPolicy.ValidateToolDescriptor(descriptor); err != nil {
			return nil, err
		}
	}
	root := config.WorkspaceRoot
	if strings.TrimSpace(root) == "" {
		root, _ = os.Getwd()
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	root, err = canonicalExistingDirectory(root)
	if err != nil {
		return nil, fmt.Errorf("canonicalize runtime workspace: %w", err)
	}
	dataRoot, err := resolveRuntimeDataRoot(root, config.DataRoot)
	if err != nil {
		return nil, err
	}
	pushOutbox := config.PushOutbox
	if pushOutbox == nil && config.Push != nil {
		pushOutbox, err = NewPushOutbox(filepath.Join(dataRoot, ".agent-push-outbox"))
		if err != nil {
			return nil, err
		}
	}
	contextStore := config.Context
	if contextStore == nil {
		contextStore, err = NewContextStore(filepath.Join(dataRoot, ".agent-context"))
		if err != nil {
			return nil, err
		}
	}
	if err := contextStore.SetWorkspaceRoot(root); err != nil {
		return nil, err
	}
	deploymentApprovals := config.DeploymentApprovals
	if deploymentApprovals == nil {
		deploymentApprovals, err = NewDeploymentApprovalStore(filepath.Join(dataRoot, ".agent-deployment-approvals"))
		if err != nil {
			return nil, err
		}
	}
	webhookReplay := config.WebhookReplay
	if webhookReplay == nil {
		webhookReplay, err = NewWebhookReplayStore(filepath.Join(dataRoot, ".agent-webhook-replay"))
		if err != nil {
			return nil, err
		}
	}
	companyStore := config.Company
	if companyStore == nil {
		companyStore, err = NewCompanyStore(filepath.Join(dataRoot, ".agent-companies"))
		if err != nil {
			return nil, err
		}
	}
	queue := config.Queue
	if queue == nil {
		queue, err = NewJobQueue(filepath.Join(dataRoot, ".agent-queue"))
		if err != nil {
			return nil, err
		}
	}
	traces := config.Traces
	if traces == nil {
		traces, err = NewTraceStore(filepath.Join(dataRoot, ".agent-traces"))
		if err != nil {
			return nil, err
		}
	}
	builder := config.Builder
	if builder == nil {
		builder, err = NewBuilderService(filepath.Join(dataRoot, ".agent-builders"))
		if err != nil {
			return nil, err
		}
	}
	collaboration := config.Collaboration
	if collaboration == nil {
		collaboration, err = NewCollaborationStore(filepath.Join(dataRoot, ".agent-collaboration"))
		if err != nil {
			return nil, err
		}
	}
	telemetry := config.Telemetry
	if telemetry == nil {
		telemetry, err = NewTelemetry(context.Background(), "")
		if err != nil {
			return nil, err
		}
	}
	runtime := &Runtime{store: store, planner: planner, plannerResolver: config.PlannerResolver, tools: tools, capabilityPolicy: capabilityPolicy, workspaceRoot: root, dataRoot: dataRoot, context: contextStore, company: companyStore, remoteMCP: config.RemoteMCP, metrics: &RuntimeMetrics{}, connectors: config.Connectors, mcp: config.MCP, queue: queue, redisQueue: config.RedisQueue, traces: traces, telemetry: telemetry, media: config.Media, builder: builder, collaboration: collaboration, push: config.Push, pushOutbox: pushOutbox, deployments: config.Deployments, deploymentApprovals: deploymentApprovals, webhookReplay: webhookReplay, mu: &sync.Mutex{}, running: make(map[string]bool), activeCancels: make(map[string]context.CancelFunc), cancelPollInterval: durableCancelPollInterval()}
	orchestrator, err := NewAgentOrchestrator(filepath.Join(dataRoot, ".agent-orchestrator"), runtime.SubagentRunner)
	if err != nil {
		return nil, err
	}
	runtime.orchestrator = orchestrator
	runtime.research = NewResearchEngine()
	devices := config.Devices
	if devices == nil {
		devices, err = NewDeviceStore(filepath.Join(dataRoot, ".agent-devices"))
		if err != nil {
			return nil, err
		}
	}
	runtime.devices = devices
	runtime.uploads, err = NewUploadManager(filepath.Join(dataRoot, ".agent-uploads"), 0, 0)
	if err != nil {
		return nil, err
	}
	runtime.ingestion = DocumentIngestor{Context: contextStore, Research: runtime.research, WorkspaceRoot: root}
	runtime.projectImporter = NewProjectImporter(root, dataRoot, contextStore, runtime.ingestion)
	if config.WhatsApp != nil {
		runtime.whatsapp = config.WhatsApp
	} else {
		runtime.whatsapp = NewWhatsAppGateway(WhatsAppGatewayConfig{}, runtime)
		// Re-apply WhatsApp credentials saved by a previous session so the
		// gateway keeps working across restarts.
		if err := runtime.whatsapp.LoadPersistedConfig(); err != nil {
			slog.Warn("failed to load persisted WhatsApp config", "error", err)
		}
	}
	supCfg := DefaultSupervisorConfig()
	if config.SupervisorConfig != nil {
		supCfg = *config.SupervisorConfig
	}
	runtime.supervisor = NewSupervisor(runtime, supCfg)
	return runtime, nil
}

// WithOrganization returns a request-scoped runtime view. Every Store receives
// application-level object scoping; PostgreSQL also receives transaction-local RLS.
func (r *Runtime) WithOrganization(organizationID string) *Runtime {
	return r.WithOrganizationContext(context.Background(), organizationID)
}

// WithOrganizationContext returns a tenant view whose Postgres operations are
// canceled with the supplied request or worker context.
func (r *Runtime) WithOrganizationContext(ctx context.Context, organizationID string) *Runtime {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	view := *r
	view.organizationScope = strings.TrimSpace(organizationID)
	if postgres := postgresStoreFromStore(r.store); postgres != nil {
		view.store = postgres.WithOrganizationContext(ctx, view.organizationScope)
	}
	// Local mode is a reserved single-user scope. Keep the underlying store
	// unwrapped so ownerless legacy local records remain readable; all runtime
	// access checks still apply organizationOwnsRecord below.
	if view.organizationScope != "" && view.organizationScope != LocalOrganizationID && view.store != nil {
		view.store = organizationScopedStore{store: view.store, organizationID: view.organizationScope}
	}
	return &view
}

func (r *Runtime) SetPlannerResolver(resolver PlannerResolver) {
	if r != nil {
		r.plannerResolver = resolver
	}
}

func (r *Runtime) Context() *ContextStore {
	return r.context
}

func (r *Runtime) Supervisor() *Supervisor {
	if r == nil {
		return nil
	}
	return r.supervisor
}

func (r *Runtime) SetSupervisor(s *Supervisor) {
	if r != nil {
		r.supervisor = s
	}
}

func (r *Runtime) WorkspaceRoot() string {
	if r == nil {
		return ""
	}
	return r.workspaceRoot
}

// OpenMissionWorkspaceRoot returns the descriptor-anchored workspace belonging
// to a persisted mission. Callers must close the returned root. The mission is
// loaded through the Runtime's organization-scoped store before its workspace
// identity/snapshot is validated.
func (r *Runtime) OpenMissionWorkspaceRoot(missionID string) (Mission, *os.Root, error) {
	if r == nil {
		return Mission{}, nil, errors.New("mission runtime is unavailable")
	}
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return Mission{}, nil, err
	}
	root, err := openMissionWorkspaceSnapshot(r.dataRoot, mission)
	if err != nil {
		return Mission{}, nil, fmt.Errorf("validate mission workspace: %w", err)
	}
	if root == nil {
		return Mission{}, nil, errors.New("pinned mission workspace is unavailable")
	}
	return mission, root, nil
}

// RequiresOrganizationAuthentication reports whether the durable store can
// contain multiple organizations and therefore cannot use the unauthenticated
// single-user API mode.
func (r *Runtime) RequiresOrganizationAuthentication() bool {
	if r == nil {
		return false
	}
	return usesPostgresStore(r.store)
}

func usesPostgresStore(store Store) bool {
	switch typed := store.(type) {
	case *PostgresStore:
		return true
	case organizationScopedStore:
		return usesPostgresStore(typed.store)
	default:
		return false
	}
}

func postgresStoreFromStore(store Store) *PostgresStore {
	switch typed := store.(type) {
	case *PostgresStore:
		return typed
	case organizationScopedStore:
		return postgresStoreFromStore(typed.store)
	default:
		return nil
	}
}

func (r *Runtime) DataRoot() string {
	if r == nil {
		return ""
	}
	return r.dataRoot
}

func (r *Runtime) WhatsApp() *WhatsAppGateway {
	if r == nil {
		return nil
	}
	return r.whatsapp
}

func (r *Runtime) SetWhatsApp(w *WhatsAppGateway) {
	if r != nil {
		r.whatsapp = w
	}
}

func (r *Runtime) CompanyStore() *CompanyStore {
	return r.company
}

func (r *Runtime) Metrics() MetricsSnapshot {
	return r.metrics.Snapshot()
}

func (r *Runtime) Connectors() []ConnectorConfig {
	if r.connectors == nil {
		return nil
	}
	return r.connectors.List()
}

func (r *Runtime) SetAuthStore(store *AuthStore) {
	if r == nil {
		return
	}
	r.authStore = store
	if r.connectors != nil {
		r.connectors.SetOAuthStore(store)
	}
}

func (r *Runtime) OAuthAccessTokenForOrganization(organizationID, provider string) (string, error) {
	if r.authStore == nil {
		return "", os.ErrNotExist
	}
	token, _, err := r.authStore.OAuthAccessTokenForOrganization(organizationID, provider)
	return token, err
}

func (r *Runtime) SetConnectorEnabled(id string, enabled bool) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.SetEnabled(id, enabled)
}

func (r *Runtime) RegisterConnector(config ConnectorConfig) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.Register(config)
}

func (r *Runtime) RemoveConnector(id string) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.Remove(id)
}

func (r *Runtime) RegisterConnectorForOrganization(organizationID string, config ConnectorConfig) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.RegisterForOrganization(organizationID, config)
}

func (r *Runtime) SetConnectorOrganizationSecret(organizationID, name, value string) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.SetOrganizationSecret(organizationID, name, value)
}

func (r *Runtime) SetMCPEnabled(id string, enabled bool) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.SetEnabled(id, enabled)
}

func (r *Runtime) RegisterMCP(config MCPServerConfig) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.Register(config)
}

func (r *Runtime) RemoveMCP(id string) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.Remove(id)
}

func (r *Runtime) SetRemoteMCPEnabled(id string, enabled bool) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.SetEnabled(id, enabled)
}

func (r *Runtime) RegisterRemoteMCP(config RemoteMCPServerConfig) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.Register(config)
}

func (r *Runtime) SetRemoteMCPOrganizationSecret(organizationID, name, value string) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.SetOrganizationSecret(organizationID, name, value)
}

func (r *Runtime) RemoveRemoteMCP(id string) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.Remove(id)
}

func (r *Runtime) BeginRemoteMCPOAuth(id, organizationID string) (string, error) {
	if r.remoteMCP == nil {
		return "", errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.BeginOAuth(id, organizationID)
}

func (r *Runtime) CompleteRemoteMCPOAuth(ctx context.Context, state, code string) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.CompleteOAuth(ctx, state, code)
}

func (r *Runtime) BeginRemoteMCPPairing(id, organizationID string) (string, error) {
	if r.remoteMCP == nil {
		return "", errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.BeginPairing(id, organizationID)
}

func (r *Runtime) CompleteRemoteMCPPairing(id, organizationID, presented, challenge string) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.CompletePairing(id, organizationID, presented, challenge)
}

func (r *Runtime) SetSkillEnabled(id string, enabled bool) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.SetSkillEnabled(id, enabled)
}

func (r *Runtime) RegisterSkill(manifest SkillManifest) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.RegisterSkill(manifest)
}

func (r *Runtime) PromoteSkillTrustedForOrganization(organizationID, id string, policy CapabilityPolicy) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.PromoteSkillTrustedForOrganization(organizationID, id, policy)
}

func (r *Runtime) RemoveSkill(id string) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.RemoveSkill(id)
}

func (r *Runtime) MCPServers() []MCPServerConfig {
	if r.mcp == nil {
		return nil
	}
	return r.mcp.List()
}

func (r *Runtime) RemoteMCPServers() []RemoteMCPServerConfig {
	if r.remoteMCP == nil {
		return nil
	}
	return r.remoteMCP.List()
}

func (r *Runtime) Skills() []SkillManifest {
	if r.context == nil {
		return nil
	}
	return r.context.Skills()
}

func (r *Runtime) Traces(traceID string) []TraceSpan {
	return r.traces.List(traceID, 500)
}

func (r *Runtime) TracesForOrganization(organizationID, traceID string) []TraceSpan {
	return r.traces.ListForOrganization(organizationID, traceID, 500)
}

func (r *Runtime) Media() *MediaManager { return r.media }

func (r *Runtime) Builder() *BuilderService { return r.builder }

// Uploads exposes the large-file upload manager (chunked/resumable, org-scoped).
func (r *Runtime) Uploads() *UploadManager { return r.uploads }

// ProjectImporter exposes the bounded, non-executing GitHub/ZIP import flow.
func (r *Runtime) ProjectImporter() *ProjectImporter { return r.projectImporter }

func (r *Runtime) Collaboration() *CollaborationStore { return r.collaboration }

func (r *Runtime) Orchestrator() *AgentOrchestrator { return r.orchestrator }

func (r *Runtime) Research() *ResearchEngine { return r.research }

func (r *Runtime) Devices() *DeviceStore { return r.devices }

func (r *Runtime) Ingestion() DocumentIngestor { return r.ingestion }

func (r *Runtime) Push() *PushService { return r.push }

func (r *Runtime) Deployments() *DeploymentManager { return r.deployments }

// Close releases background resources held by the runtime, notably the
// OpenTelemetry batch span processor goroutine. Safe on a nil runtime and to
// call more than once.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var result error
	if r.telemetry != nil {
		result = r.telemetry.Shutdown(ctx)
	}
	if postgres, ok := r.store.(*PostgresStore); ok {
		if err := postgres.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (r *Runtime) DeploymentApprovals() *DeploymentApprovalStore { return r.deploymentApprovals }

func (r *Runtime) WebhookReplay() *WebhookReplayStore { return r.webhookReplay }

func (r *Runtime) CreateMission(ctx context.Context, request CreateMissionRequest) (Mission, error) {
	objective := strings.TrimSpace(request.Objective)
	if objective == "" {
		return Mission{}, errors.New("objective is required")
	}
	if len(objective) > 8<<10 {
		return Mission{}, errors.New("objective is too long")
	}
	provider := strings.TrimSpace(request.Provider)
	if provider == "" {
		provider = "ollama-local"
	}
	projectID := strings.TrimSpace(request.ProjectID)
	organizationID := strings.TrimSpace(request.OrganizationID)
	if r.organizationScope != "" {
		if organizationID != "" && organizationID != r.organizationScope {
			return Mission{}, errors.New("mission is outside the active organization")
		}
		organizationID = r.organizationScope
		if projectID == "" && r.organizationScope != LocalOrganizationID {
			return Mission{}, errors.New("organization-scoped missions require a project owned by the active organization")
		}
	}
	var project Project
	var err error
	if projectID != "" {
		if r.context == nil {
			return Mission{}, errors.New("context store is required for project missions")
		}
		project, err = r.context.GetProject(projectID)
		if err != nil {
			return Mission{}, err
		}
		if r.organizationScope != "" && !organizationOwnsRecord(project.OrganizationID, r.organizationScope) {
			return Mission{}, errors.New("project is not owned by the active organization")
		}
		if request.IsolateWorkspace && (strings.TrimSpace(project.OrganizationID) == "" || (organizationID != "" && project.OrganizationID != organizationID)) {
			return Mission{}, errors.New("isolated workspace requires a project owned by the active organization")
		}
		if organizationID != "" && project.OrganizationID != "" && project.OrganizationID != organizationID {
			return Mission{}, errors.New("project is outside the active organization")
		}
		if project.OrganizationID != "" {
			organizationID = project.OrganizationID
		}
	}
	requestedWorkspace := strings.TrimSpace(request.Workspace)
	if projectID != "" && requestedWorkspace == "" {
		requestedWorkspace = strings.TrimSpace(project.Root)
	}
	workspace, err := r.resolveWorkspace(requestedWorkspace)
	if err != nil {
		return Mission{}, err
	}
	workspaceIdentity := ""
	if !request.IsolateWorkspace {
		workspaceIdentity, err = workspaceDirectoryIdentity(workspace)
		if err != nil {
			return Mission{}, fmt.Errorf("bind mission workspace identity: %w", err)
		}
	}
	if provider != "ollama-local" {
		outbound := map[string]any{"objective": objective}
		if err := validateOutboundPayload(outbound); err != nil {
			return Mission{}, errors.New("planner payload rejected by data-egress policy")
		}
	}
	if projectID != "" {
		projectRoot, rootErr := filepath.Abs(strings.TrimSpace(project.Root))
		if rootErr != nil {
			return Mission{}, rootErr
		}
		if strings.TrimSpace(project.Root) == "" || !isWithin(projectRoot, workspace) {
			return Mission{}, errors.New("mission workspace must be inside the selected project")
		}
		if rootErr := rejectSymlinkComponents(projectRoot, workspace); rootErr != nil {
			return Mission{}, fmt.Errorf("mission project workspace is not safe: %w", rootErr)
		}
	}
	capabilities := normalizeMissionCapabilities(request.Capabilities)
	if len(capabilities) == 0 {
		capabilities = []string{"workspace:read"}
	}
	capabilities, err = r.capabilityPolicy.ValidateMissionCapabilities(capabilities)
	if err != nil {
		return Mission{}, err
	}
	missionID := strings.TrimSpace(request.MissionID)
	if missionID == "" {
		missionID = "mis_" + uuid.NewString()
	} else if !validSnapshotID(missionID) {
		return Mission{}, errors.New("internal mission id is invalid")
	}
	workspaceSnapshotID := ""
	workspaceSnapshotSHA256 := ""
	var snapshotHandles *workspaceSnapshotHandles
	snapshotPersisted := false
	defer func() {
		if snapshotHandles != nil {
			if !snapshotPersisted {
				_ = snapshotHandles.Remove()
			}
			_ = snapshotHandles.Close()
		}
	}()
	if request.IsolateWorkspace {
		if projectID == "" || organizationID == "" {
			return Mission{}, errors.New("isolated workspace requires a project and organization")
		}
		projectRoot, rootErr := canonicalExistingDirectory(project.Root)
		if rootErr != nil {
			return Mission{}, fmt.Errorf("resolve isolated project root: %w", rootErr)
		}
		projectRootInfo, rootErr := os.Lstat(projectRoot)
		if rootErr != nil || projectRootInfo.Mode()&os.ModeSymlink != 0 || !projectRootInfo.IsDir() {
			if rootErr != nil {
				return Mission{}, fmt.Errorf("inspect isolated project root: %w", rootErr)
			}
			return Mission{}, errors.New("isolated project root must be a real directory")
		}
		snapshotSourceRoot, rootErr := resolveWorkspaceSnapshotRepositoryRoot(ctx, projectRoot)
		if rootErr != nil {
			return Mission{}, fmt.Errorf("resolve isolated project Git repository: %w", rootErr)
		}
		if !isWithin(r.workspaceRoot, snapshotSourceRoot) {
			return Mission{}, errors.New("isolated Git repository is outside the configured workspace root")
		}
		relativeWorkspace, relErr := filepath.Rel(snapshotSourceRoot, workspace)
		if relErr != nil || !filepath.IsLocal(relativeWorkspace) {
			return Mission{}, errors.New("isolated workspace must be inside the selected project")
		}
		manifest, handles, snapshotErr := createWorkspaceSnapshotWithHandles(ctx, WorkspaceSnapshotRequest{
			SourceRoot:              snapshotSourceRoot,
			ExpectedProjectRoot:     projectRoot,
			ExpectedProjectRootInfo: projectRootInfo,
			DataRoot:                filepath.Join(r.dataRoot, ".agent-workspace-snapshots"),
			MissionID:               missionID,
			OrganizationID:          organizationID,
			ProjectID:               projectID,
		})
		if snapshotErr != nil {
			return Mission{}, fmt.Errorf("create isolated workspace snapshot: %w", snapshotErr)
		}
		snapshotHandles = handles
		workspace, workspaceSnapshotSHA256, err = prepareWorkspaceSnapshotHandoff(manifest, handles, relativeWorkspace)
		if err != nil {
			return Mission{}, err
		}
		workspaceSnapshotID = manifest.SnapshotID
	}
	gitRepoRoot := ""
	gitBranch := ""
	gitWorktreePath := ""
	gitBaseCommit := ""
	gitMergeStatus := ""
	gitWorktreeActive := false

	if request.IsolateWorktree {
		worktreeTarget := workspace
		if projectID != "" && project.Root != "" {
			worktreeTarget = project.Root
		}
		dataRoot := r.dataRoot
		if dataRoot == "" {
			dataRoot = filepath.Join(os.TempDir(), "ollama-agent-data")
		}
		session, wtErr := CreateGitWorktree(ctx, worktreeTarget, dataRoot, missionID, request.WorktreeBranch)
		if wtErr != nil {
			return Mission{}, fmt.Errorf("create isolated git worktree: %w", wtErr)
		}
		workspace = session.WorktreeDir
		gitRepoRoot = session.RepoRoot
		gitBranch = session.BranchName
		gitWorktreePath = session.WorktreeDir
		gitBaseCommit = session.BaseCommit
		gitMergeStatus = "pending"
		gitWorktreeActive = true
	}
	now := time.Now().UTC()
	mission := Mission{ID: missionID, Version: 1, Objective: objective, Provider: provider, Model: strings.TrimSpace(request.Model), Workspace: workspace, WorkspaceIdentity: workspaceIdentity, WorkspaceIsolated: request.IsolateWorkspace, WorkspaceSnapshotID: workspaceSnapshotID, WorkspaceSnapshotSHA256: workspaceSnapshotSHA256, GitRepoRoot: gitRepoRoot, GitBranch: gitBranch, GitWorktreePath: gitWorktreePath, GitBaseCommit: gitBaseCommit, GitMergeStatus: gitMergeStatus, GitWorktreeActive: gitWorktreeActive, AutoRepair: request.AutoRepair, MaxRepairTries: request.MaxRepairTries, ProjectID: projectID, OrganizationID: organizationID, Capabilities: capabilities, AutoRun: request.AutoRun, State: MissionPlanning, CreatedAt: now, UpdatedAt: now}
	if err := r.store.CreateMission(mission); err != nil {
		// The store may have committed before a connection/timeout error reached
		// this caller. Preserve the snapshot; an orphan sweep can reclaim it only
		// after checking durable mission references.
		if snapshotHandles != nil {
			snapshotPersisted = true
		}
		return Mission{}, err
	}
	snapshotPersisted = true
	r.metrics.missionsCreated.Add(1)
	createdPayload := map[string]any{"objective": objective}
	if mission.WorkspaceIsolated {
		createdPayload["workspace_snapshot_id"] = mission.WorkspaceSnapshotID
		createdPayload["workspace_snapshot_sha256"] = mission.WorkspaceSnapshotSHA256
	}
	if mission.GitWorktreeActive {
		createdPayload["git_repo_root"] = mission.GitRepoRoot
		createdPayload["git_branch"] = mission.GitBranch
		createdPayload["git_worktree_path"] = mission.GitWorktreePath
		createdPayload["git_base_commit"] = mission.GitBaseCommit
	}
	if err := r.observeEvent(mission, "mission.created", "", createdPayload); err != nil {
		return Mission{}, err
	}
	planner := r.planner
	if routed, ok := r.plannerResolver.(RoutedPlannerResolver); ok && (mission.Model == "" || strings.HasPrefix(mission.Model, "auto/") || mission.Model == "auto") {
		var resolution PlannerResolution
		planner, resolution, err = routed.ResolvePlannerForMission(ctx, mission.Provider, mission.Model, mission.OrganizationID, mission.Capabilities)
		if err != nil {
			return r.failMission(mission, err)
		}
		if resolution.Provider != "" {
			mission.Provider = resolution.Provider
		}
		if resolution.Model != "" {
			mission.Model = resolution.Model
		}
		if err := r.observeEvent(mission, "router.decision", "", map[string]any{
			"provider": resolution.Provider,
			"model":    resolution.Model,
			"reason":   resolution.Reason,
			"cost_tag": resolution.CostTag,
		}); err != nil {
			return r.failMission(mission, err)
		}
	} else if provider == "ollama-local" && mission.Model != "" && r.plannerResolver != nil {
		planner, err = r.plannerResolver.ResolvePlanner(provider, mission.Model)
		if err != nil {
			return r.failMission(mission, err)
		}
		if planner == nil {
			return r.failMission(mission, errors.New("local provider returned no planner"))
		}
	} else if provider != "ollama-local" {
		if r.plannerResolver == nil {
			return r.failMission(mission, fmt.Errorf("provider %q is not configured in this runtime", provider))
		}
		planner, err = r.plannerResolver.ResolvePlanner(provider, mission.Model)
		if err != nil {
			return r.failMission(mission, err)
		}
		if planner == nil {
			return r.failMission(mission, fmt.Errorf("provider %q returned no planner", provider))
		}
	}
	if planner == nil {
		return r.failMission(mission, errors.New("planner is not configured"))
	}
	plan, err := planner.Plan(ctx, mission)
	if err != nil {
		return r.failMission(mission, err)
	}
	plan, err = normalizeSteps(plan)
	if err != nil {
		return r.failMission(mission, err)
	}
	if mission.WorkspaceIsolated {
		// project.test.run usa acesso por caminho e é bloqueado na execução de
		// missão isolada; rejeitar já no planejamento evita aprovar uma missão
		// que falharia depois de trabalho parcial (ex.: após workspace.write).
		for index := range plan {
			if plan[index].Kind == "project.test.run" {
				return r.failMission(mission, fmt.Errorf("tool %q is not supported in isolated snapshots and cannot be planned", plan[index].Kind))
			}
		}
	}
	for index := range plan {
		tool, ok := r.tools.Get(plan[index].Kind)
		if !ok {
			return r.failMission(mission, fmt.Errorf("planner returned unregistered tool %q", plan[index].Kind))
		}
		descriptor := tool.Descriptor()
		if err := r.capabilityPolicy.ValidateToolDescriptor(descriptor); err != nil {
			return r.failMission(mission, err)
		}
		plan[index].ToolDescriptorSHA256, err = toolDescriptorSHA256(descriptor)
		if err != nil {
			return r.failMission(mission, fmt.Errorf("fingerprint tool descriptor: %w", err))
		}
		if plan[index].Kind == "connector.http" {
			if strings.TrimSpace(mission.OrganizationID) == "" {
				plan[index].ToolConfigSHA256, err = r.connectors.ApprovalConfigSHA256Global(plan[index].Input)
			} else {
				plan[index].ToolConfigSHA256, err = r.connectors.ApprovalConfigSHA256(mission.OrganizationID, plan[index].Input)
			}
			if err != nil {
				return r.failMission(mission, fmt.Errorf("fingerprint connector configuration: %w", err))
			}
		}
		if plan[index].Kind == "sandbox.exec" {
			mode, modeErr := configuredSandboxMode()
			if modeErr != nil {
				return r.failMission(mission, modeErr)
			}
			if plan[index].Input == nil {
				plan[index].Input = make(map[string]any)
			}
			// Planner-provided values are untrusted. Store the authoritative
			// runtime mode in the approved payload and execute only that mode.
			plan[index].Input["sandbox_mode"] = mode
			if mode == "best-effort" {
				plan[index].Input["sandbox_gate_status"] = string(GateStatusNotConfigured)
				plan[index].Input["sandbox_approval_notice"] = sandboxBestEffortApprovalNotice
			}
		}
		plan[index].RequiresApproval = plan[index].RequiresApproval || descriptor.RequiresApproval
		if riskRank(descriptor.Risk) > riskRank(plan[index].Risk) {
			plan[index].Risk = descriptor.Risk
		}
		if plan[index].Kind == "workspace.write" {
			if !r.capabilityPolicy.Allows(descriptor, mission.Capabilities) {
				return r.failMission(mission, fmt.Errorf("%w: %s", ErrCapabilityDenied, plan[index].Kind))
			}
			plan[index].Input, err = prepareWorkspaceWriteApproval(workspace, plan[index].Input)
			if err != nil {
				return r.failMission(mission, fmt.Errorf("prepare workspace write approval: %w", err))
			}
		}
		if plan[index].Kind == "workspace.patch" {
			if !r.capabilityPolicy.Allows(descriptor, mission.Capabilities) {
				return r.failMission(mission, fmt.Errorf("%w: %s", ErrCapabilityDenied, plan[index].Kind))
			}
			plan[index].Input, err = prepareWorkspacePatchApproval(workspace, plan[index].Input)
			if err != nil {
				return r.failMission(mission, fmt.Errorf("prepare workspace patch approval: %w", err))
			}
		}
		if plan[index].Kind == "media.process" {
			if !r.capabilityPolicy.Allows(descriptor, mission.Capabilities) {
				return r.failMission(mission, fmt.Errorf("%w: %s", ErrCapabilityDenied, plan[index].Kind))
			}
			plan[index].Input, err = prepareMediaProcessApproval(workspace, plan[index].Input)
			if err != nil {
				return r.failMission(mission, fmt.Errorf("prepare media approval: %w", err))
			}
		}
	}
	mission.Plan = plan
	mission.State = MissionReady
	approvalExpiresAt := now.Add(30 * time.Minute)
	for _, step := range plan {
		if step.RequiresApproval {
			descriptor, _ := r.tools.Get(step.Kind)
			payloadHash, hashErr := approvalPayloadSHA256(step)
			if hashErr != nil {
				return r.failMission(mission, fmt.Errorf("hash approval payload: %w", hashErr))
			}
			requester := strings.TrimSpace(request.ActorID)
			mission.Approvals = append(mission.Approvals, Approval{ID: "apr_" + uuid.NewString(), MissionID: mission.ID, StepID: step.ID, OrganizationID: mission.OrganizationID, Policy: toolApprovalPolicy(descriptor.Descriptor(), step.Risk), PayloadSHA256: payloadHash, Nonce: uuid.NewString(), RequestedBy: requester, Status: ApprovalPending, ExpiresAt: &approvalExpiresAt, CreatedAt: now, UpdatedAt: now})
		}
	}
	if len(mission.Approvals) > 0 {
		mission.State = MissionAwaitingApproval
	}
	expectedVersion := mission.Version
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
		return Mission{}, err
	}
	if err := r.observeEvent(mission, "mission.planned", "", map[string]any{"steps": len(plan), "approvals": len(mission.Approvals)}); err != nil {
		return Mission{}, err
	}
	if request.AutoRun && mission.State == MissionReady {
		if _, enqueueErr := r.EnqueueMission(mission.ID); enqueueErr != nil {
			queueErr := fmt.Errorf("auto-run enqueue failed: %w", enqueueErr)
			expectedVersion := mission.Version
			mission.State = MissionFailed
			mission.LastError = RedactDLP(queueErr.Error())
			mission.Version++
			mission.UpdatedAt = time.Now().UTC()
			r.metrics.missionsFailed.Add(1)
			if saveErr := r.store.PutMissionIfVersion(mission, expectedVersion); saveErr != nil {
				return Mission{}, errors.Join(queueErr, saveErr)
			}
			if eventErr := r.event(mission, "mission.queue_failed", "", map[string]any{"error": mission.LastError}); eventErr != nil {
				return redactMissionForPersistence(mission), errors.Join(queueErr, eventErr)
			}
			return redactMissionForPersistence(mission), queueErr
		}
	}
	return redactMissionForPersistence(mission), nil
}

func normalizeMissionCapabilities(capabilities []string) []string {
	seen := make(map[string]struct{}, len(capabilities))
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		result = append(result, capability)
	}
	sort.Strings(result)
	return result
}

func (r *Runtime) GetMission(id string) (Mission, error) {
	return r.getMission(strings.TrimSpace(id))
}

func (r *Runtime) GetMissionWorktreeDiff(ctx context.Context, missionID string) (*GitMergeApproval, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return nil, err
	}
	if !mission.GitWorktreeActive || mission.GitWorktreePath == "" {
		return nil, errors.New("mission does not have an active git worktree")
	}
	session := &GitWorktreeSession{
		MissionID:    mission.ID,
		RepoRoot:     mission.GitRepoRoot,
		WorktreeDir:  mission.GitWorktreePath,
		BranchName:   mission.GitBranch,
		BaseCommit:   mission.GitBaseCommit,
		TargetBranch: "main",
	}
	return GetWorktreeDiff(ctx, session)
}

// MergeMissionWorktree merges the mission's worktree branch into origin. The
// human approval must be bound to the exact diff being merged (SEC-03):
// approvedDiffSHA is the diff_sha256 the reviewer saw (from
// GetMissionWorktreeDiff). The merge recomputes the current diff and refuses if
// it no longer matches, so an agent that mutates the worktree after approval
// cannot merge a different diff than the one that was reviewed ("approve A,
// merge B"). An empty approvedDiffSHA is rejected: a merge is never unbound.
func (r *Runtime) MergeMissionWorktree(ctx context.Context, missionID, approvedDiffSHA string) (string, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return "", err
	}
	if !mission.GitWorktreeActive || mission.GitWorktreePath == "" {
		return "", errors.New("mission does not have an active git worktree")
	}
	approvedDiffSHA = strings.ToLower(strings.TrimSpace(approvedDiffSHA))
	if approvedDiffSHA == "" {
		return "", errors.New("merge requires the approved diff digest (approved_diff_sha256) from the reviewed diff")
	}
	session := &GitWorktreeSession{
		MissionID:    mission.ID,
		RepoRoot:     mission.GitRepoRoot,
		WorktreeDir:  mission.GitWorktreePath,
		BranchName:   mission.GitBranch,
		BaseCommit:   mission.GitBaseCommit,
		TargetBranch: "main",
	}
	// Recompute the diff that is actually about to be merged and bind it to the
	// approval the caller is presenting.
	current, err := GetWorktreeDiff(ctx, session)
	if err != nil {
		return "", fmt.Errorf("inspect worktree before merge: %w", err)
	}
	if !strings.EqualFold(current.DiffSHA256, approvedDiffSHA) {
		_ = r.observeEvent(mission, "git.merge.rejected", "", map[string]any{
			"reason":       "diff_changed_since_approval",
			"approved_sha": approvedDiffSHA,
			"current_sha":  current.DiffSHA256,
		})
		return "", fmt.Errorf("the worktree changed since approval (approved %s, current %s); re-review the diff before merging", approvedDiffSHA, current.DiffSHA256)
	}
	mergeCommit, err := MergeWorktreeToOrigin(ctx, session)
	if err != nil {
		return "", err
	}
	expectedVersion := mission.Version
	mission.GitMergeStatus = "merged"
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
		// O merge no origin já ocorreu; não persistir o estado "merged" deixa a
		// missão divergir do repositório. Registre para investigação.
		slog.Error("failed to persist merged mission state", "mission", mission.ID, "error", err)
	}
	_ = r.observeEvent(mission, "git.merge.succeeded", "", map[string]any{
		"branch":       session.BranchName,
		"merge_commit": mergeCommit,
		"approved_sha": approvedDiffSHA,
	})
	return mergeCommit, nil
}

func (r *Runtime) ListMissions() ([]Mission, error) {
	missions, err := r.store.ListMissions()
	if err != nil || r.organizationScope == "" {
		return missions, err
	}
	filtered := make([]Mission, 0, len(missions))
	for _, mission := range missions {
		if organizationOwnsRecord(mission.OrganizationID, r.organizationScope) {
			filtered = append(filtered, mission)
		}
	}
	return filtered, nil
}

func (r *Runtime) getMission(id string) (Mission, error) {
	mission, err := r.store.GetMission(strings.TrimSpace(id))
	if err != nil {
		return Mission{}, err
	}
	if r.organizationScope != "" && !organizationOwnsRecord(mission.OrganizationID, r.organizationScope) {
		return Mission{}, os.ErrNotExist
	}
	return mission, nil
}

// DeleteMission permanently removes a mission (and its event history) from the
// store. A mission that is still active cannot be deleted — it must be cancelled
// first — so a running task is never yanked out from under the runtime. The
// lookup is organization-scoped, so a mission owned by another tenant is
// reported as not-found.
func (r *Runtime) DeleteMission(ctx context.Context, missionID string) error {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return err
	}
	switch mission.State {
	case "RUNNING", "OBSERVING", "RECOVERING", "PLANNING", "PENDING":
		return fmt.Errorf("a tarefa está em andamento (%s); cancele-a antes de excluir", mission.State)
	}
	if err := r.store.DeleteMission(mission.ID); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) runQueueJob(jobContext context.Context, job QueueJob) error {
	organizationID := strings.TrimSpace(job.OrganizationID)
	if r.organizationScope != "" {
		if !organizationOwnsRecord(organizationID, r.organizationScope) {
			return ErrQueueJobForbidden
		}
	}
	workerRuntime := r
	if organizationID != "" {
		workerRuntime = r.WithOrganizationContext(jobContext, organizationID)
	}
	before, loadErr := workerRuntime.GetMission(job.MissionID)
	if loadErr != nil || before.OrganizationID != organizationID {
		return ErrQueueJobForbidden
	}
	runErr := workerRuntime.Run(jobContext, job.MissionID)
	if runErr == nil {
		return nil
	}
	if before.State == MissionFailed || missionHasNonReadStep(before, workerRuntime.tools) {
		return fmt.Errorf("%w: %w", ErrQueueNonRetryable, runErr)
	}
	mission, loadErr := workerRuntime.GetMission(job.MissionID)
	if loadErr == nil && (mission.State == MissionFailed || missionHasNonReadStep(mission, workerRuntime.tools)) {
		return fmt.Errorf("%w: %w", ErrQueueNonRetryable, runErr)
	}
	return runErr
}

func (r *Runtime) Start(ctx context.Context) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	worker := r.runQueueJob
	workerID := "agent-runtime-" + uuid.NewString()
	if r.redisQueue != nil {
		if err := r.redisQueue.Start(ctx, workerID, worker); err != nil {
			r.metrics.redisQueueFailures.Add(1)
			slog.Error("agent Redis worker failed to start", "error", err)
		} else {
			queueErrors := r.redisQueue.Errors()
			go func() {
				var reportedDropped uint64
				droppedTicker := time.NewTicker(time.Second)
				defer droppedTicker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-droppedTicker.C:
						dropped := r.redisQueue.DroppedErrors()
						if dropped > reportedDropped {
							delta := dropped - reportedDropped
							r.metrics.redisQueueDroppedErrors.Add(int64(delta))
							slog.Error("agent Redis queue error channel overflowed", "dropped_errors", delta, "total_dropped_errors", dropped)
							reportedDropped = dropped
						}
					case err, ok := <-queueErrors:
						if !ok {
							return
						}
						r.metrics.redisQueueFailures.Add(1)
						slog.Error("agent Redis queue operation failed", "error", err)
					}
				}
			}()
		}
	} else {
		r.queue.Start(ctx, workerID, worker)
	}
	if r.push != nil && r.pushOutbox != nil {
		go func() {
			r.flushPushOutbox(ctx)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					r.flushPushOutbox(ctx)
				}
			}
		}()
	}
	go func() {
		recoveryInterval := 2 * time.Second
		if usesPostgresStore(r.store) {
			recoveryInterval = 15 * time.Second
		}
		if err := r.resumePending(ctx); err != nil {
			slog.Error("agent pending work recovery failed", "error", err)
		}
		ticker := time.NewTicker(recoveryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.resumePending(ctx); err != nil {
					slog.Error("agent pending work recovery failed", "error", err)
				}
			}
		}
	}()
}

func (r *Runtime) flushPushOutbox(ctx context.Context) {
	if r == nil || r.push == nil || r.pushOutbox == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, maxPushOutboxFlushDuration)
	defer cancel()
	startedAt := time.Now()
	for processed := 0; processed < maxPushOutboxBatch; processed++ { //nolint:intrange // bounded loop uses a side-effectful reflective length
		if ctx.Err() != nil || time.Since(startedAt) >= maxPushOutboxFlushDuration {
			return
		}
		item, ok, err := r.pushOutbox.ClaimDueContext(ctx, time.Now().UTC())
		if err != nil {
			r.metrics.pushOutboxFailures.Add(1)
			return
		}
		if !ok {
			return
		}
		if err := validateOutboundPayloadWithLimit(map[string]any{"title": item.Title, "body": item.Body, "data": item.Data}, maxOutboundDLPScanBytes); err != nil {
			r.metrics.pushDeliveryFailures.Add(1)
			if failErr := r.pushOutbox.FailContext(ctx, item.ID, item.LeaseToken, errOutboundPayloadBlocked); failErr != nil {
				r.metrics.pushOutboxFailures.Add(1)
			}
			continue
		}
		if ctx.Err() != nil {
			if failErr := r.pushOutbox.FailContext(ctx, item.ID, item.LeaseToken, ctx.Err()); failErr != nil {
				r.metrics.pushOutboxFailures.Add(1)
			}
			return
		}
		err = r.push.NotifyOrganization(ctx, item.OrganizationID, item.Title, item.Body, item.Data)
		if err != nil {
			r.metrics.pushDeliveryFailures.Add(1)
			if failErr := r.pushOutbox.FailContext(ctx, item.ID, item.LeaseToken, err); failErr != nil {
				r.metrics.pushOutboxFailures.Add(1)
			}
			continue
		}
		if err := r.pushOutbox.CompleteContext(ctx, item.ID, item.LeaseToken); err != nil {
			r.metrics.pushOutboxFailures.Add(1)
		}
	}
}

func (r *Runtime) resumePending(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var recoveryErrors []error
	organizationScope := strings.TrimSpace(r.organizationScope)
	// The base PostgreSQL runtime cannot query globally. Use the trusted auth
	// directory as the tenant index and recover each tenant through its scoped
	// runtime; no SQL-level global-read capability is added.
	if usesPostgresStore(r.store) && organizationScope == "" {
		if r.authStore == nil {
			return errors.New("PostgreSQL recovery requires the trusted AuthStore tenant directory")
		}
		organizations := r.authStore.Organizations()
		if len(organizations) > maxPostgresRecoveryOrganizations {
			return fmt.Errorf("PostgreSQL recovery tenant count %d exceeds limit %d", len(organizations), maxPostgresRecoveryOrganizations)
		}
		for _, organization := range organizations {
			if err := ctx.Err(); err != nil {
				return errors.Join(errors.Join(recoveryErrors...), err)
			}
			if strings.TrimSpace(organization.ID) == "" {
				continue
			}
			if err := r.WithOrganizationContext(ctx, organization.ID).resumePending(ctx); err != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("recover organization %s: %w", organization.ID, err))
			}
		}
		return errors.Join(recoveryErrors...)
	}
	for _, schedule := range r.context.ClaimDueSchedulesForOrganization(organizationScope, time.Now().UTC()) {
		if companyID := companyIDFromWorkspace(schedule.Workspace); companyID != "" && r.company != nil {
			company, err := r.company.Get(companyID)
			if err == nil && (company.Status == CompanyPaused || company.Risk.Paused) {
				continue
			}
		}
		// ExecuteScheduleFlow runs the persisted flow graph, or the legacy single
		// mission when the schedule carries no steps.
		if _, err := r.ExecuteScheduleFlow(ctx, schedule); err != nil {
			if _, recordErr := r.recordScheduleFailure(schedule); recordErr != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("record schedule %s failure: %w", schedule.ID, recordErr))
			}
			continue
		}
		if _, err := r.recordScheduleSuccess(schedule); err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("record schedule %s success: %w", schedule.ID, err))
		}
	}
	missions, err := r.ListMissions()
	if err != nil {
		return err
	}
	organizationScope = strings.TrimSpace(r.organizationScope)
	if err := sweepOrphanedWorkspaceSnapshots(r.dataRoot, organizationScope, missions, time.Now().UTC()); err != nil {
		// Snapshot housekeeping is best-effort during restart. Keep mission
		// recovery fail-closed for the individual mission, but do not prevent
		// an otherwise valid persisted mission from being re-enqueued because
		// a concurrent temporary-root cleanup interrupted the sweep.
		slog.Warn("orphaned workspace snapshot sweep deferred", "error", err)
	}
	for _, mission := range missions {
		if mission.State != MissionCompleted && mission.State != MissionCancelled && mission.State != MissionFailed {
			if err := r.validateMissionApprovalBindings(mission); err != nil {
				_, failErr := r.failMission(mission, ErrApprovalPayloadChanged)
				if failErr != nil && !errors.Is(failErr, ErrApprovalPayloadChanged) {
					recoveryErrors = append(recoveryErrors, fmt.Errorf("fail mission %s with invalid approval binding: %w", mission.ID, failErr))
				}
				continue
			}
		}
		// MissionObserving is a transient state persisted right before a tool or
		// observation runs; a crash in that window must be re-enqueued too, or the
		// mission is stranded forever. The execution entry point accepts it.
		resume := mission.State == MissionRunning || mission.State == MissionRecovering || mission.State == MissionObserving || (mission.State == MissionReady && mission.AutoRun)
		if !resume || !r.approvalsReady(mission) {
			continue
		}
		// In-flight states (Running/Observing) are persisted mid-execution and are
		// not directly enqueueable; move them to Recovering so the queue accepts the
		// resume. Recovering and Ready+AutoRun already pass EnqueueMission's guard.
		if mission.State == MissionRunning || mission.State == MissionObserving {
			expectedVersion := mission.Version
			mission.State = MissionRecovering
			mission.Version++
			mission.UpdatedAt = time.Now().UTC()
			if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("mark mission %s recovering: %w", mission.ID, err))
				continue
			}
		}
		if r.queue != nil {
			if err := r.queue.recoverStateForRestart(); err != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("recover queue state before mission %s: %w", mission.ID, err))
				continue
			}
		}
		if _, err := r.EnqueueMission(mission.ID); err != nil {
			// A crash can leave the queue root between directory discovery and
			// opening its lock (notably while macOS cleans a temporary root).
			// Retry once only for that missing-path condition; ordinary queue
			// persistence failures remain fail-closed.
			if errors.Is(err, os.ErrNotExist) && r.queue != nil {
				if recoverErr := r.queue.recoverStateForRestart(); recoverErr == nil {
					_, err = r.EnqueueMission(mission.ID)
				}
			}
			if err != nil {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("enqueue mission %s during recovery: %w", mission.ID, err))
			}
		}
	}
	return errors.Join(recoveryErrors...)
}

const maxPostgresRecoveryOrganizations = 10000

const maxScheduleFailures = 3

func (r *Runtime) recordScheduleFailure(schedule Schedule) (Schedule, error) {
	if r.context == nil {
		return Schedule{}, errors.New("context store is required for schedule failure")
	}
	schedule.FailureCount++
	schedule.LastFailureCode = "mission_creation_failed"
	if schedule.FailureCount >= maxScheduleFailures {
		schedule.Enabled = false
	} else {
		retryDelay := time.Duration(schedule.FailureCount*5) * time.Second
		if retryDelay > 5*time.Minute {
			retryDelay = 5 * time.Minute
		}
		schedule.NextRunAt = time.Now().UTC().Add(retryDelay)
	}
	return r.context.UpdateScheduleForOrganization(r.organizationScope, schedule.ID, schedule)
}

func (r *Runtime) recordScheduleSuccess(schedule Schedule) (Schedule, error) {
	if r.context == nil || (schedule.FailureCount == 0 && schedule.LastFailureCode == "") {
		return schedule, nil
	}
	schedule.FailureCount = 0
	schedule.LastFailureCode = ""
	return r.context.UpdateScheduleForOrganization(r.organizationScope, schedule.ID, schedule)
}

func companyIDFromWorkspace(workspace string) string {
	workspace = strings.TrimSpace(workspace)
	if !strings.HasPrefix(workspace, "company://") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(workspace, "company://"))
}

func (r *Runtime) EnqueueMission(missionID string) (QueueJob, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return QueueJob{}, err
	}
	workspaceRoot, err := openMissionWorkspaceSnapshot(r.dataRoot, mission)
	if err != nil {
		return QueueJob{}, fmt.Errorf("validate mission workspace snapshot: %w", err)
	}
	if workspaceRoot != nil {
		_ = workspaceRoot.Close()
	}
	if mission.State != MissionReady && mission.State != MissionRecovering {
		return QueueJob{}, ErrMissionNotRunnable
	}
	if step := interruptedNonReadStep(mission, r.tools); step != nil {
		return QueueJob{}, fmt.Errorf("%w: interrupted non-read step %s requires inspection", ErrMissionTerminal, step.ID)
	}
	if r.redisQueue != nil {
		return r.redisQueue.EnqueueForOrganization(mission.OrganizationID, mission.ID, 3)
	}
	return r.queue.EnqueueForOrganization(mission.OrganizationID, mission.ID, 3)
}

// QueueJobs returns the jobs for a status on a best-effort basis: a backing
// queue error yields an empty slice. Callers that must distinguish "no jobs"
// from "queue dependency is down" (e.g. health) use QueueJobsWithError.
func (r *Runtime) QueueJobs(status QueueStatus) []QueueJob {
	jobs, _ := r.QueueJobsWithError(status)
	return jobs
}

// QueueJobsWithError is like QueueJobs but surfaces a backing-queue failure
// instead of masking it as an empty result.
func (r *Runtime) QueueJobsWithError(status QueueStatus) ([]QueueJob, error) {
	jobs, err := r.queueJobsRaw(status)
	if err != nil {
		return nil, err
	}
	if r.organizationScope == "" {
		return jobs, nil
	}
	filtered := make([]QueueJob, 0, len(jobs))
	for _, job := range jobs {
		if mission, err := r.getMission(job.MissionID); err == nil && mission.OrganizationID == r.organizationScope && job.OrganizationID == mission.OrganizationID {
			filtered = append(filtered, job)
		}
	}
	return filtered, nil
}

func (r *Runtime) queueJobsRaw(status QueueStatus) ([]QueueJob, error) {
	if r.redisQueue != nil {
		return r.redisQueue.List(status)
	}
	return r.queue.List(status)
}

func (r *Runtime) ReplayJob(jobID string) (QueueJob, error) {
	jobID = strings.TrimSpace(jobID)
	deadJobs, err := r.queueJobsRaw(QueueDeadLetter)
	if err != nil {
		return QueueJob{}, err
	}
	for _, job := range deadJobs {
		if job.ID != jobID {
			continue
		}
		mission, err := r.getMission(job.MissionID)
		if err != nil {
			if r.organizationScope != "" {
				return QueueJob{}, ErrQueueJobForbidden
			}
			return QueueJob{}, err
		}
		if mission.State != MissionReady && mission.State != MissionRecovering {
			return QueueJob{}, ErrMissionTerminal
		}
		if step := interruptedNonReadStep(mission, r.tools); step != nil {
			return QueueJob{}, fmt.Errorf("%w: interrupted non-read step %s requires inspection", ErrMissionTerminal, step.ID)
		}
		if r.redisQueue != nil {
			return r.redisQueue.ReplayForOrganization(mission.OrganizationID, jobID)
		}
		return r.queue.ReplayForOrganization(mission.OrganizationID, jobID)
	}
	return QueueJob{}, os.ErrNotExist
}

func (r *Runtime) QueueJobsForOrganization(organizationID string, status QueueStatus) ([]QueueJob, error) {
	jobs := r.QueueJobs(status)
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		for _, job := range jobs {
			mission, err := r.getMission(job.MissionID)
			if err != nil {
				return nil, err
			}
			if mission.OrganizationID != "" || job.OrganizationID != mission.OrganizationID {
				return nil, ErrQueueJobForbidden
			}
		}
		return jobs, nil
	}
	filtered := make([]QueueJob, 0, len(jobs))
	for _, job := range jobs {
		mission, err := r.getMission(job.MissionID)
		if err != nil {
			continue
		}
		if mission.OrganizationID == organizationID && job.OrganizationID == mission.OrganizationID {
			filtered = append(filtered, job)
		}
	}
	return filtered, nil
}

func (r *Runtime) ReplayJobForOrganization(jobID, organizationID string) (QueueJob, error) {
	organizationID = strings.TrimSpace(organizationID)
	if r.organizationScope != "" && organizationID != r.organizationScope {
		return QueueJob{}, ErrQueueJobForbidden
	}
	allJobs, err := r.queueJobsRaw("")
	if err != nil {
		return QueueJob{}, err
	}
	if organizationID == "" {
		for _, job := range allJobs {
			if job.ID != strings.TrimSpace(jobID) {
				continue
			}
			mission, err := r.getMission(job.MissionID)
			if err != nil {
				return QueueJob{}, err
			}
			if mission.OrganizationID != "" {
				return QueueJob{}, ErrQueueJobForbidden
			}
			return r.ReplayJob(jobID)
		}
		return QueueJob{}, os.ErrNotExist
	}
	for _, job := range allJobs {
		if job.ID != strings.TrimSpace(jobID) {
			continue
		}
		mission, err := r.getMission(job.MissionID)
		if err != nil {
			if r.organizationScope != "" {
				return QueueJob{}, ErrQueueJobForbidden
			}
			return QueueJob{}, err
		}
		if mission.OrganizationID != organizationID {
			return QueueJob{}, ErrQueueJobForbidden
		}
		return r.ReplayJob(jobID)
	}
	return QueueJob{}, os.ErrNotExist
}

func (r *Runtime) ListTools() []ToolDescriptor {
	return r.tools.Descriptors()
}

func (r *Runtime) Run(ctx context.Context, id string) (runErr error) {
	ctx, otelSpan := r.telemetry.Start(ctx, "agent.mission.run", map[string]string{"mission.id": id})
	defer otelSpan.End()
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("mission id is required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	if r.running[id] {
		r.mu.Unlock()
		cancel()
		return nil
	}
	r.running[id] = true
	r.activeCancels[id] = cancel
	r.mu.Unlock()
	// Propagate a cancellation persisted by ANY instance to this run's context,
	// so a long-running step here is aborted even when another instance served
	// the Cancel() call and holds no in-memory cancel func for this mission.
	cancelWatchDone := make(chan struct{})
	go r.watchDurableCancellation(runCtx, id, cancel, cancelWatchDone)
	defer func() {
		close(cancelWatchDone)
		cancel()
		r.mu.Lock()
		delete(r.running, id)
		delete(r.activeCancels, id)
		r.mu.Unlock()
	}()

	mission, err := r.getMission(id)
	if err != nil {
		return err
	}
	workspaceRoot, err := openMissionWorkspaceSnapshot(r.dataRoot, mission)
	if err != nil {
		return fmt.Errorf("validate mission workspace snapshot: %w", err)
	}
	if workspaceRoot != nil {
		defer workspaceRoot.Close()
	}
	var snapshotManifest *WorkspaceSnapshotManifest
	snapshotPrefix := ""
	if mission.WorkspaceIsolated {
		for _, step := range mission.Plan {
			if step.Kind != "git.repo.inspect" {
				continue
			}
			manifest, prefix, manifestErr := readMissionWorkspaceSnapshotManifest(r.dataRoot, mission)
			if manifestErr != nil {
				_, failErr := r.failMission(mission, fmt.Errorf("read isolated Git baseline: %w", manifestErr))
				return failErr
			}
			snapshotManifest = &manifest
			snapshotPrefix = prefix
			break
		}
	}
	if mission.State == MissionFailed {
		return ErrMissionTerminal
	}
	if step := interruptedNonReadStep(mission, r.tools); step != nil {
		return r.failStep(mission, step, fmt.Errorf("%w: step %s may have partially executed; inspect workspace/provider state before creating a new mission", ErrMissionTerminal, step.ID))
	}
	missionSpan := r.traces.StartForOrganization(mission.OrganizationID, "tr_"+id, "", "mission.run", map[string]any{"mission_id": id})
	defer func() { missionSpan.End("ok", runErr) }()
	if mission.State == MissionCompleted || mission.State == MissionCancelled {
		return nil
	}
	if err := r.validateMissionApprovalBindings(mission); err != nil {
		_, failErr := r.failMission(mission, ErrApprovalPayloadChanged)
		return failErr
	}
	if mission.State == MissionAwaitingApproval && !r.approvalsReady(mission) {
		return nil
	}
	if mission.State != MissionReady && mission.State != MissionRecovering && mission.State != MissionRunning && mission.State != MissionObserving && mission.State != MissionAwaitingApproval {
		return ErrMissionNotRunnable
	}
	if !r.approvalsReady(mission) {
		expectedVersion := mission.Version
		mission.State = MissionAwaitingApproval
		mission.Version++
		mission.UpdatedAt = time.Now().UTC()
		if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
			return err
		}
		return nil
	}
	expectedVersion := mission.Version
	mission.State = MissionRunning
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
		return err
	}
	if err := r.observeEvent(mission, "mission.running", "", nil); err != nil {
		return err
	}

	for index := 0; index < len(mission.Plan); index++ {
		if r.missionCancelled(id) {
			return nil
		}
		step := &mission.Plan[index]
		if step.State == StepSucceeded {
			continue
		}
		tool, ok := r.tools.Get(step.Kind)
		if !ok {
			return r.failStep(mission, step, fmt.Errorf("tool %q is not registered", step.Kind))
		}
		if mission.WorkspaceIsolated {
			switch step.Kind {
			case "terminal.exec", "sandbox.exec", "media.process", "project.test.run":
				return r.failStep(mission, step, fmt.Errorf("tool %q uses pathname-based workspace access and is disabled for isolated snapshots", step.Kind))
			}
		}
		if err := r.capabilityPolicy.ValidateToolDescriptor(tool.Descriptor()); err != nil {
			return r.failStep(mission, step, err)
		}
		descriptorHash, descriptorErr := toolDescriptorSHA256(tool.Descriptor())
		if descriptorErr != nil || step.ToolDescriptorSHA256 == "" || descriptorHash != step.ToolDescriptorSHA256 || tool.Descriptor().RequiresApproval && !step.RequiresApproval {
			return r.failStep(mission, step, ErrApprovalPayloadChanged)
		}
		if step.RequiresApproval && step.Kind == "connector.http" {
			var configHash string
			var configErr error
			if strings.TrimSpace(mission.OrganizationID) == "" {
				configHash, configErr = r.connectors.ApprovalConfigSHA256Global(step.Input)
			} else {
				configHash, configErr = r.connectors.ApprovalConfigSHA256(mission.OrganizationID, step.Input)
			}
			if configErr != nil || configHash != step.ToolConfigSHA256 {
				return r.failStep(mission, step, ErrApprovalPayloadChanged)
			}
		}
		if !r.capabilityPolicy.Allows(tool.Descriptor(), mission.Capabilities) {
			return r.failStep(mission, step, fmt.Errorf("%w: %s", ErrCapabilityDenied, step.Kind))
		}
		if step.RequiresApproval && !r.stepApproved(mission, step.ID) {
			expectedVersion = mission.Version
			step.State = StepBlocked
			mission.State = MissionAwaitingApproval
			mission.Version++
			mission.UpdatedAt = time.Now().UTC()
			if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
				return err
			}
			if err := r.observeEvent(mission, "step.awaiting_approval", step.ID, nil); err != nil {
				return err
			}
			return nil
		}
		if approval, ok := approvalForStep(mission, step.ID); ok && approval.Status == ApprovalApproved && approval.PayloadSHA256 != "" {
			payloadHash, hashErr := approvalPayloadSHA256(*step)
			if hashErr != nil || payloadHash != approval.PayloadSHA256 {
				return r.failStep(mission, step, ErrApprovalPayloadChanged)
			}
		}
		expectedVersion = mission.Version
		step.State = StepRunning
		step.Attempts++
		r.metrics.stepsStarted.Add(1)
		r.metrics.toolCalls.Add(1)
		mission.State = MissionObserving
		mission.Version++
		mission.UpdatedAt = time.Now().UTC()
		if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
			return err
		}
		if err := r.observeEvent(mission, "step.started", step.ID, map[string]any{"tool": step.Kind, "attempt": step.Attempts}); err != nil {
			return err
		}
		toolSpan := r.traces.StartForOrganization(mission.OrganizationID, "tr_"+mission.ID, missionSpan.ID(), "tool."+step.Kind, map[string]any{"mission_id": mission.ID, "step_id": step.ID, "tool": step.Kind})
		toolContext := ToolContext{MissionID: mission.ID, StepID: step.ID, Workspace: mission.Workspace, WorkspaceRoot: workspaceRoot, OrganizationID: mission.OrganizationID, ToolConfigSHA256: step.ToolConfigSHA256}
		if step.Kind == "git.repo.inspect" && mission.WorkspaceIsolated {
			toolContext.WorkspaceSnapshotManifest = snapshotManifest
			toolContext.WorkspaceSnapshotPrefix = snapshotPrefix
			toolContext.MissionArtifacts = append([]ArtifactManifest(nil), mission.Artifacts...)
		}
		result, executeErr := tool.Execute(runCtx, toolContext, step.Input)
		toolSpan.End("ok", executeErr)
		if r.missionCancelled(id) {
			return nil
		}
		if executeErr != nil {
			mission.Artifacts = appendUniqueArtifacts(mission.Artifacts, result.Artifacts)
			if shouldRetryStep(*step, r.tools) {
				expectedVersion = mission.Version
				r.metrics.retries.Add(1)
				step.State = StepPending
				mission.State = MissionRecovering
				mission.Version++
				mission.UpdatedAt = time.Now().UTC()
				if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
					return err
				}
				if err := r.observeEvent(mission, "step.retry_scheduled", step.ID, map[string]any{"error": RedactDLP(executeErr.Error())}); err != nil {
					return err
				}
				index--
				continue
			}
			return r.failStep(mission, step, executeErr)
		}
		expectedVersion = mission.Version
		step.State = StepSucceeded
		r.metrics.stepsSucceeded.Add(1)
		step.Result = RedactValue(result.Value)
		step.Error = ""
		mission.Artifacts = appendUniqueArtifacts(mission.Artifacts, result.Artifacts)
		mission.State = MissionRunning
		mission.Version++
		mission.UpdatedAt = time.Now().UTC()
		if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
			return err
		}
		if browserMap, ok := result.Value.(map[string]any); ok {
			if shot, hasShot := browserMap["screenshot"].(string); hasShot && shot != "" {
				_ = r.observeEvent(mission, "browser.frame", step.ID, map[string]any{
					"url":        browserMap["url"],
					"title":      browserMap["title"],
					"screenshot": shot,
				})
			}
		}
		if err := r.observeEvent(mission, "step.succeeded", step.ID, map[string]any{"artifacts": len(result.Artifacts)}); err != nil {
			return err
		}
	}
	if r.missionCancelled(id) {
		return nil
	}
	completed := time.Now().UTC()
	expectedVersion = mission.Version
	mission.State = MissionCompleted
	r.metrics.missionsCompleted.Add(1)
	mission.CompletedAt = &completed
	mission.Version++
	mission.UpdatedAt = completed
	if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
		return err
	}
	if err := r.observeEvent(mission, "mission.completed", "", map[string]any{"artifacts": len(mission.Artifacts)}); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) Cancel(id string) (Mission, error) {
	id = strings.TrimSpace(id)
	mission, err := r.getMission(id)
	if err != nil {
		return Mission{}, err
	}
	if mission.State == MissionCompleted {
		return Mission{}, errors.New("completed mission cannot be cancelled")
	}
	r.mu.Lock()
	cancel := r.activeCancels[id]
	r.mu.Unlock()
	if mission.State == MissionCancelled {
		if cancel != nil {
			cancel()
		}
		return mission, nil
	}
	expectedVersion := mission.Version
	for index := range mission.Plan {
		if mission.Plan[index].State == StepPending || mission.Plan[index].State == StepRunning || mission.Plan[index].State == StepBlocked {
			mission.Plan[index].State = StepBlocked
			mission.Plan[index].Error = "mission cancelled"
		}
	}
	mission.State = MissionCancelled
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if err := r.store.PutMissionIfVersion(mission, expectedVersion); err != nil {
		return Mission{}, err
	}
	if cancel != nil {
		cancel()
	}
	if err := r.observeEvent(mission, "mission.cancelled", "", nil); err != nil {
		return mission, err
	}
	return mission, nil
}

func (r *Runtime) DecideApproval(missionID, approvalID string, approved bool, reason string) (Mission, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return Mission{}, err
	}
	return r.DecideApprovalForActor(missionID, approvalID, approved, reason, "local", mission.OrganizationID)
}

func (r *Runtime) DecideApprovalForActor(missionID, approvalID string, approved bool, reason, actorID, organizationID string) (Mission, error) {
	return r.decideApprovalForActor(missionID, approvalID, approved, reason, actorID, organizationID, 0, "", false)
}

func (r *Runtime) DecideApprovalForActorCAS(missionID, approvalID string, approved bool, reason, actorID, organizationID string, expectedVersion int64, nonce string) (Mission, error) {
	return r.decideApprovalForActor(missionID, approvalID, approved, reason, actorID, organizationID, expectedVersion, nonce, true)
}

func (r *Runtime) decideApprovalForActor(missionID, approvalID string, approved bool, reason, actorID, organizationID string, expectedVersion int64, nonce string, requireNonce bool) (Mission, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return Mission{}, err
	}
	if err := r.validateMissionApprovalBindings(mission); err != nil {
		return Mission{}, ErrApprovalPayloadChanged
	}
	if expectedVersion > 0 && mission.Version != expectedVersion {
		return Mission{}, ErrApprovalVersionConflict
	}
	expectedVersion = mission.Version
	for index := range mission.Approvals {
		if mission.Approvals[index].ID != approvalID {
			continue
		}
		if mission.Approvals[index].Status != ApprovalPending {
			continue
		}
		if mission.Approvals[index].ExpiresAt != nil && time.Now().UTC().After(*mission.Approvals[index].ExpiresAt) {
			return Mission{}, errors.New("approval has expired")
		}
		if mission.OrganizationID != "" && strings.TrimSpace(organizationID) != mission.OrganizationID {
			return Mission{}, errors.New("approval organization mismatch")
		}
		if strings.TrimSpace(actorID) == "" {
			return Mission{}, errors.New("approval actor is required")
		}
		// Separação de funções (SoD): o solicitante não pode aprovar a própria
		// decisão. EXCETO no modo local de usuário único, em que todo request é
		// atribuído ao ator sintético LocalActorID ("local"): ali existe apenas
		// uma pessoa, que É o aprovador humano, então exigir um segundo aprovador
		// tornaria qualquer missão de escrita impossível de aprovar no desktop.
		// O gate humano continua existindo (a pessoa precisa aprovar cada passo);
		// um ator real/nomeado (enterprise ou autenticado) segue sob SoD.
		requestedBy := strings.TrimSpace(mission.Approvals[index].RequestedBy)
		if approved && requestedBy != "" && requestedBy == strings.TrimSpace(actorID) && requestedBy != LocalActorID {
			return Mission{}, errors.New("approval requester cannot approve the same decision")
		}
		reason = strings.TrimSpace(reason)
		if reason == "" {
			return Mission{}, errors.New("approval reason is required")
		}
		if len([]byte(reason)) > 2048 {
			return Mission{}, ErrApprovalReasonTooLong
		}
		reason = RedactDLP(reason)
		if requireNonce && (strings.TrimSpace(nonce) == "" || nonce != mission.Approvals[index].Nonce) {
			return Mission{}, ErrApprovalNonceMismatch
		}
		if approved && mission.Approvals[index].PayloadSHA256 != "" {
			step, ok := missionStep(mission, mission.Approvals[index].StepID)
			if !ok {
				return Mission{}, ErrApprovalPayloadChanged
			}
			payloadHash, hashErr := approvalPayloadSHA256(step)
			if hashErr != nil || payloadHash != mission.Approvals[index].PayloadSHA256 {
				return Mission{}, ErrApprovalPayloadChanged
			}
		}
		mission.Approvals[index].ActorID = strings.TrimSpace(actorID)
		mission.Approvals[index].OrganizationID = mission.OrganizationID
		if approved {
			mission.Approvals[index].Status = ApprovalApproved
		} else {
			mission.Approvals[index].Status = ApprovalRejected
		}
		mission.Approvals[index].Reason = reason
		mission.Approvals[index].UpdatedAt = time.Now().UTC()
		if approved && r.approvalsReady(mission) {
			mission.State = MissionReady
		} else if !approved {
			mission.State = MissionFailed
			mission.LastError = "approval rejected"
		}
		mission.Version++
		mission.UpdatedAt = time.Now().UTC()
		saveErr := r.store.PutMissionIfVersion(mission, expectedVersion)
		if errors.Is(saveErr, ErrMissionVersionConflict) {
			saveErr = ErrApprovalVersionConflict
		}
		if saveErr != nil {
			return Mission{}, saveErr
		}
		r.metrics.approvals.Add(1)
		if err := r.observeEvent(mission, "approval.decided", mission.Approvals[index].StepID, map[string]any{"approved": approved, "reason": reason}); err != nil {
			return redactMissionForPersistence(mission), err
		}
		return redactMissionForPersistence(mission), nil
	}
	return Mission{}, errors.New("approval not found or already decided")
}

func (r *Runtime) missionCancelled(id string) bool {
	mission, err := r.getMission(strings.TrimSpace(id))
	return err == nil && mission.State == MissionCancelled
}

const defaultDurableCancelPollInterval = 2 * time.Second

// durableCancelPollInterval reads OLLAMA_AGENT_CANCEL_POLL_MS (clamped to a sane
// range) so operators can tune how quickly a running instance reacts to a
// cancellation persisted by another instance. Defaults to 2s.
func durableCancelPollInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_CANCEL_POLL_MS"))
	if raw == "" {
		return defaultDurableCancelPollInterval
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 50 {
		return defaultDurableCancelPollInterval
	}
	if ms > 60000 {
		ms = 60000
	}
	return time.Duration(ms) * time.Millisecond
}

// watchDurableCancellation bridges a durable cancellation (persisted in the
// store, possibly by a DIFFERENT instance) to the local run context. The step
// loop already re-reads the store between steps, but a long-running step on
// this instance would otherwise keep going until it finishes, because only the
// instance that served Cancel() holds the in-memory cancel func. Polling the
// durable state here lets any instance's cancel abort the in-flight step.
func (r *Runtime) watchDurableCancellation(ctx context.Context, id string, cancel context.CancelFunc, done <-chan struct{}) {
	interval := r.cancelPollInterval
	if interval <= 0 {
		interval = defaultDurableCancelPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if r.missionCancelled(id) {
				cancel()
				return
			}
		}
	}
}

func (r *Runtime) Events(id string) ([]Event, error) {
	if _, err := r.getMission(strings.TrimSpace(id)); err != nil {
		return nil, err
	}
	return r.store.ListEvents(strings.TrimSpace(id))
}

func (r *Runtime) Artifact(missionID, artifactID string) (ArtifactManifest, string, error) {
	mission, err := r.getMission(strings.TrimSpace(missionID))
	if err != nil {
		return ArtifactManifest{}, "", err
	}
	for _, artifact := range mission.Artifacts {
		if artifact.ID != artifactID {
			continue
		}
		if _, err := safeWorkspacePath(mission.Workspace, artifact.Path); err != nil {
			return ArtifactManifest{}, "", err
		}
		workspaceRoot, err := openMissionWorkspaceSnapshot(r.dataRoot, mission)
		if err != nil {
			return ArtifactManifest{}, "", fmt.Errorf("validate mission workspace snapshot: %w", err)
		}
		if workspaceRoot != nil {
			defer workspaceRoot.Close()
		}
		var snapshotPath string
		if workspaceRoot != nil {
			snapshotPath, err = createVerifiedArtifactSnapshotFromRoot(workspaceRoot, artifact.Path, artifact)
		} else {
			snapshotPath, err = createVerifiedArtifactSnapshot(mission.Workspace, artifact.Path, artifact)
		}
		if err != nil {
			return ArtifactManifest{}, "", err
		}
		return artifact, snapshotPath, nil
	}
	return ArtifactManifest{}, "", os.ErrNotExist
}

func (r *Runtime) approvalsReady(mission Mission) bool {
	if r.validateMissionApprovalBindings(mission) != nil {
		return false
	}
	for _, approval := range mission.Approvals {
		if approval.Status != ApprovalApproved {
			return false
		}
	}
	return true
}

func (r *Runtime) stepApproved(mission Mission, stepID string) bool {
	if r.validateMissionApprovalBindings(mission) != nil {
		return false
	}
	approval, ok := approvalForStep(mission, stepID)
	return ok && approval.Status == ApprovalApproved && approval.PayloadSHA256 != ""
}

func (r *Runtime) validateMissionApprovalBindings(mission Mission) error {
	if r == nil || r.tools == nil {
		return errors.New("approval tool registry is unavailable")
	}
	steps := make(map[string]Step, len(mission.Plan))
	for _, step := range mission.Plan {
		if step.ID == "" {
			return errors.New("approval step id is missing")
		}
		if _, exists := steps[step.ID]; exists {
			return errors.New("approval step id is ambiguous")
		}
		steps[step.ID] = step
	}
	approvals := make(map[string]Approval, len(mission.Approvals))
	approvalIDs := make(map[string]struct{}, len(mission.Approvals))
	for _, approval := range mission.Approvals {
		step, exists := steps[approval.StepID]
		if !exists || !step.RequiresApproval || approval.ID == "" || approval.MissionID != mission.ID || approval.OrganizationID != mission.OrganizationID || approval.PayloadSHA256 == "" || approval.Nonce == "" || approval.ExpiresAt == nil {
			return errors.New("approval record is missing or not bound to its mission")
		}
		if _, duplicate := approvals[approval.StepID]; duplicate {
			return errors.New("approval step has duplicate approval records")
		}
		if _, duplicate := approvalIDs[approval.ID]; duplicate {
			return errors.New("approval id is ambiguous")
		}
		approvalIDs[approval.ID] = struct{}{}
		if approval.Status != ApprovalPending && approval.Status != ApprovalApproved && approval.Status != ApprovalRejected {
			return errors.New("approval status is invalid")
		}
		if time.Now().UTC().After(*approval.ExpiresAt) {
			return errors.New("approval has expired")
		}
		if approval.Status != ApprovalPending && (strings.TrimSpace(approval.ActorID) == "" || strings.TrimSpace(approval.Reason) == "") {
			return errors.New("decided approval is missing actor or reason")
		}
		tool, ok := r.tools.Get(step.Kind)
		if !ok || approval.Policy == "" || approval.Policy != toolApprovalPolicy(tool.Descriptor(), step.Risk) {
			return errors.New("approval policy does not match the current tool descriptor")
		}
		payloadHash, err := approvalPayloadSHA256(step)
		if err != nil || payloadHash != approval.PayloadSHA256 {
			return ErrApprovalPayloadChanged
		}
		approvals[approval.StepID] = approval
	}
	for _, step := range steps {
		if step.RequiresApproval {
			if _, exists := approvals[step.ID]; !exists {
				return errors.New("approval-required step has no approval record")
			}
		}
	}
	return nil
}

func approvalForStep(mission Mission, stepID string) (Approval, bool) {
	for _, approval := range mission.Approvals {
		if approval.StepID == stepID {
			return approval, true
		}
	}
	return Approval{}, false
}

func missionStep(mission Mission, stepID string) (Step, bool) {
	for _, step := range mission.Plan {
		if step.ID == stepID {
			return step, true
		}
	}
	return Step{}, false
}

func (r *Runtime) failMission(mission Mission, err error) (Mission, error) {
	expectedVersion := mission.Version
	mission.State = MissionFailed
	r.metrics.missionsFailed.Add(1)
	mission.LastError = RedactDLP(err.Error())
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if saveErr := r.store.PutMissionIfVersion(mission, expectedVersion); saveErr != nil {
		return Mission{}, saveErr
	}
	if eventErr := r.observeEvent(mission, "mission.failed", "", map[string]any{"error": err.Error()}); eventErr != nil {
		return redactMissionForPersistence(mission), errors.Join(err, eventErr)
	}
	return redactMissionForPersistence(mission), err
}

func (r *Runtime) failStep(mission Mission, step *Step, err error) error {
	expectedVersion := mission.Version
	step.State = StepFailed
	r.metrics.stepsFailed.Add(1)
	step.Error = RedactDLP(err.Error())
	mission.State = MissionFailed
	mission.LastError = RedactDLP(err.Error())
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if saveErr := r.store.PutMissionIfVersion(mission, expectedVersion); saveErr != nil {
		return saveErr
	}
	if eventErr := r.observeEvent(mission, "step.failed", step.ID, map[string]any{"error": RedactDLP(err.Error()), "attempts": step.Attempts}); eventErr != nil {
		return errors.Join(err, eventErr)
	}
	return err
}

func appendUniqueArtifacts(existing, incoming []ArtifactManifest) []ArtifactManifest {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	key := func(artifact ArtifactManifest) string {
		if artifact.ID != "" {
			return artifact.ID
		}
		return artifact.MissionID + "\x00" + artifact.StepID + "\x00" + artifact.Path + "\x00" + artifact.SHA256
	}
	for _, artifact := range existing {
		seen[key(artifact)] = struct{}{}
	}
	for _, artifact := range incoming {
		artifactKey := key(artifact)
		if _, exists := seen[artifactKey]; exists {
			continue
		}
		seen[artifactKey] = struct{}{}
		existing = append(existing, artifact)
	}
	return existing
}

func shouldRetryStep(step Step, tools *Registry) bool {
	return step.Attempts < 2 && effectiveStepRisk(step, tools) == RiskRead
}

func missionHasNonReadStep(mission Mission, tools *Registry) bool {
	for _, step := range mission.Plan {
		if effectiveStepRisk(step, tools) != RiskRead {
			return true
		}
	}
	return false
}

func interruptedNonReadStep(mission Mission, tools *Registry) *Step {
	for index := range mission.Plan {
		step := &mission.Plan[index]
		if step.State == StepRunning && effectiveStepRisk(*step, tools) != RiskRead {
			return step
		}
	}
	return nil
}

func effectiveStepRisk(step Step, tools *Registry) RiskClass {
	if strings.TrimSpace(string(step.Risk)) == "" {
		if tools != nil {
			if tool, ok := tools.Get(step.Kind); ok {
				return effectiveRisk(tool.Descriptor().Risk)
			}
		}
		return RiskDestructive
	}
	risk := effectiveRisk(step.Risk)
	if tools != nil {
		if tool, ok := tools.Get(step.Kind); ok {
			toolRisk := effectiveRisk(tool.Descriptor().Risk)
			if riskRank(toolRisk) > riskRank(risk) {
				return toolRisk
			}
		}
	}
	return risk
}

func (r *Runtime) observeEvent(mission Mission, eventType, stepID string, payload any) error {
	if err := r.event(mission, eventType, stepID, payload); err != nil {
		slog.Error("agent event persistence failed", "mission_id", mission.ID, "event_type", eventType, "step_id", stepID, "error", err)
		return err
	}
	return nil
}

func (r *Runtime) event(mission Mission, eventType, stepID string, payload any) error {
	err := r.store.AppendEvent(Event{ID: "evt_" + uuid.NewString(), MissionID: mission.ID, OrganizationID: mission.OrganizationID, Type: eventType, StepID: stepID, Payload: RedactValue(payload), CreatedAt: time.Now().UTC()})
	if err != nil {
		r.metrics.eventPersistFailures.Add(1)
	}
	if r.push != nil && mission.OrganizationID != "" && (eventType == "mission.completed" || eventType == "mission.failed" || eventType == "step.awaiting_approval") {
		title := "DZ23 Agentic"
		body := "A missão " + mission.ID + " mudou de estado"
		if eventType == "mission.completed" {
			body = "A missão " + mission.ID + " foi concluída"
		}
		if r.pushOutbox == nil {
			r.metrics.pushOutboxFailures.Add(1)
			slog.Error("push notification outbox is unavailable", "mission_id", mission.ID, "event_type", eventType)
		} else if _, enqueueErr := r.pushOutbox.Enqueue(mission.OrganizationID, title, body, map[string]any{"mission_id": mission.ID, "event": eventType}); enqueueErr != nil {
			r.metrics.pushOutboxFailures.Add(1)
			slog.Error("push notification enqueue failed", "mission_id", mission.ID, "event_type", eventType, "error", enqueueErr)
		}
	}
	return err
}

func riskRank(risk RiskClass) int {
	switch effectiveRisk(risk) {
	case RiskRead:
		return 0
	case RiskWrite:
		return 1
	case RiskExternalSideEffect:
		return 2
	case RiskDestructive:
		return 3
	default:
		return 3
	}
}

func effectiveRisk(risk RiskClass) RiskClass {
	switch risk {
	case "":
		return RiskDestructive
	case RiskRead, RiskWrite, RiskExternalSideEffect, RiskDestructive:
		return risk
	default:
		return RiskDestructive
	}
}

func (r *Runtime) resolveWorkspace(requested string) (string, error) {
	root, err := canonicalExistingDirectory(r.workspaceRoot)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(requested) == "" {
		return root, nil
	}
	candidate, err := canonicalProjectRoot(root, requested)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(candidate, 0o700); err != nil {
		return "", err
	}
	return candidate, nil
}

func (r *Runtime) ConnectorsForOrganization(organizationID string) []ConnectorConfig {
	if r.connectors == nil {
		return nil
	}
	return r.connectors.ListForOrganization(organizationID)
}

func (r *Runtime) SetConnectorEnabledForOrganization(organizationID, id string, enabled bool) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.SetEnabledForOrganization(organizationID, id, enabled)
}

func (r *Runtime) RemoveConnectorForOrganization(organizationID, id string) error {
	if r.connectors == nil {
		return errors.New("connector manager is unavailable")
	}
	return r.connectors.RemoveForOrganization(organizationID, id)
}

func (r *Runtime) MCPServersForOrganization(organizationID string) []MCPServerConfig {
	if r.mcp == nil {
		return nil
	}
	return r.mcp.ListForOrganization(organizationID)
}

func (r *Runtime) SetMCPEnabledForOrganization(organizationID, id string, enabled bool) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.SetEnabledForOrganization(organizationID, id, enabled)
}

func (r *Runtime) RegisterMCPForOrganization(organizationID string, config MCPServerConfig) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.RegisterForOrganization(organizationID, config)
}

func (r *Runtime) RemoveMCPForOrganization(organizationID, id string) error {
	if r.mcp == nil {
		return errors.New("MCP manager is unavailable")
	}
	return r.mcp.RemoveForOrganization(organizationID, id)
}

func (r *Runtime) RemoteMCPServersForOrganization(organizationID string) []RemoteMCPServerConfig {
	if r.remoteMCP == nil {
		return nil
	}
	return r.remoteMCP.ListForOrganization(organizationID)
}

func (r *Runtime) SetRemoteMCPEnabledForOrganization(organizationID, id string, enabled bool) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.SetEnabledForOrganization(organizationID, id, enabled)
}

func (r *Runtime) RegisterRemoteMCPForOrganization(organizationID string, config RemoteMCPServerConfig) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.RegisterForOrganization(organizationID, config)
}

func (r *Runtime) RemoveRemoteMCPForOrganization(organizationID, id string) error {
	if r.remoteMCP == nil {
		return errors.New("remote MCP manager is unavailable")
	}
	return r.remoteMCP.RemoveForOrganization(organizationID, id)
}

func (r *Runtime) SkillsForOrganization(organizationID string) []SkillManifest {
	if r.context == nil {
		return nil
	}
	return r.context.SkillsForOrganization(organizationID)
}

func (r *Runtime) SetSkillEnabledForOrganization(organizationID, id string, enabled bool) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.SetSkillEnabledForOrganization(organizationID, id, enabled)
}

func (r *Runtime) RegisterSkillForOrganization(organizationID string, manifest SkillManifest) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.RegisterSkillForOrganization(organizationID, manifest)
}

func (r *Runtime) RemoveSkillForOrganization(organizationID, id string) error {
	if r.context == nil {
		return errors.New("context store is unavailable")
	}
	return r.context.RemoveSkillForOrganization(organizationID, id)
}
