package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/grok"
)

type agentAPI struct {
	runtime      *agent.Runtime
	context      *agent.ContextStore
	auth         *agent.AuthStore
	authRequired bool
	push         *agent.PushService
	grok         *grok.Client
	samlMu       sync.Mutex
	samlServices map[string]*agent.SAMLService
}

var errAgentForbidden = errors.New("object is outside the active organization")

func newAgentAPI(runtime *agent.Runtime) (*agentAPI, error) {
	storeRoot := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_STORE"))
	if storeRoot == "" && runtime != nil {
		storeRoot = filepath.Join(runtime.DataRoot(), "auth")
	}
	auth, err := agent.NewAuthStore(storeRoot)
	if err != nil {
		return nil, err
	}
	runtime.SetAuthStore(auth)
	required, err := agentAuthRequired()
	if err != nil {
		return nil, err
	}
	grokClient, err := newAgentGrokClient()
	if err != nil {
		return nil, err
	}
	if runtime != nil && runtime.RequiresOrganizationAuthentication() {
		required = true
	}
	return &agentAPI{runtime: runtime, context: runtime.Context(), auth: auth, authRequired: required, push: runtime.Push(), grok: grokClient, samlServices: map[string]*agent.SAMLService{}}, nil
}

func newAgentGrokClient() (*grok.Client, error) {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_GROK_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}
	model := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_GROK_MODEL"))
	if model == "" {
		model = "grok-4"
	}
	client, err := grok.NewClient(baseURL, os.Getenv("XAI_API_KEY"), model)
	if err != nil {
		return nil, err
	}
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_GROK_MODELS")); raw != "" {
		models := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\t' || r == ' ' })
		if err := client.SetAllowedModels(models...); err != nil {
			return nil, fmt.Errorf("OLLAMA_AGENT_GROK_MODELS: %w", err)
		}
	}
	return client, nil
}

func agentAuthRequired() (bool, error) {
	configured := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_REQUIRED"))
	loopback := agentHostIsLoopback()
	if configured == "" {
		return !loopback, nil
	}
	required, err := strconv.ParseBool(configured)
	if err != nil {
		return false, fmt.Errorf("OLLAMA_AGENT_AUTH_REQUIRED must be true or false: %w", err)
	}
	if !required && !loopback {
		return true, nil
	}
	return required, nil
}

func agentHostIsLoopback() bool {
	host := strings.TrimSpace(envconfig.Host().Hostname())
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func newDefaultAgentRuntime() (*agent.Runtime, error) {
	workspaceRoot := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_ROOT"))
	if workspaceRoot == "" {
		var err error
		workspaceRoot, err = agent.DefaultRuntimeWorkspaceRoot()
		if err != nil {
			return nil, err
		}
	}
	storeRoot := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_STORE"))
	if storeRoot == "" {
		var err error
		storeRoot, err = agent.DefaultRuntimeDataRoot()
		if err != nil {
			return nil, err
		}
	}
	var store agent.Store
	var postgresPool *agent.PostgresStore
	runtimeReady := false
	defer func() {
		if !runtimeReady && postgresPool != nil {
			_ = postgresPool.Close()
		}
	}()
	if databaseURL := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_DATABASE_URL")); databaseURL != "" {
		if strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REDIS_URL")) == "" {
			return nil, errors.New("PostgreSQL multi-tenant runtime requires OLLAMA_AGENT_REDIS_URL for durable owner-bound work")
		}
		keyHex := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY"))
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) < 32 {
			return nil, errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY must be at least 64 hexadecimal characters")
		}
		version, err := agentTenantContextKeyVersionFromEnv()
		if err != nil {
			return nil, err
		}
		postgres, err := agent.OpenPostgresRuntimeStoreVersioned(context.Background(), databaseURL, key, version)
		if err != nil {
			return nil, err
		}
		postgresPool = postgres
		store = postgres
	} else {
		local, err := agent.NewJSONStore(storeRoot)
		if err != nil {
			return nil, err
		}
		store = local
	}
	contextStore, err := agent.NewContextStore(filepath.Join(storeRoot, "context"))
	if err != nil {
		return nil, err
	}
	companyStore, err := agent.NewCompanyStore(filepath.Join(storeRoot, "companies"))
	if err != nil {
		return nil, err
	}
	if embedModel := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_EMBED_MODEL")); embedModel != "" {
		contextStore.SetEmbedder(agent.OllamaEmbedder{Client: api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.agent.ollama_local", true)), Model: embedModel})
	}
	connectors, err := loadAgentConnectors(storeRoot)
	if err != nil {
		return nil, err
	}
	mcp, err := loadAgentMCP(storeRoot)
	if err != nil {
		return nil, err
	}
	remoteMCP, err := loadAgentRemoteMCP(storeRoot)
	if err != nil {
		return nil, err
	}
	media, err := loadAgentMedia()
	if err != nil {
		return nil, err
	}
	deployments, err := loadAgentDeployments()
	if err != nil {
		return nil, err
	}
	push, err := agent.NewPushService(filepath.Join(storeRoot, "push"), os.Getenv("OLLAMA_AGENT_PUSH_ENDPOINT"))
	if err != nil {
		return nil, fmt.Errorf("initialize push service: %w", err)
	}
	telemetry, err := agent.NewTelemetry(context.Background(), os.Getenv("OLLAMA_AGENT_OTLP_ENDPOINT"))
	if err != nil {
		return nil, fmt.Errorf("initialize agent OpenTelemetry: %w", err)
	}
	var redisQueue *agent.RedisQueue
	if redisURL := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REDIS_URL")); redisURL != "" {
		redisQueue, err = agent.OpenRedisQueue(context.Background(), redisURL, os.Getenv("OLLAMA_AGENT_REDIS_PREFIX"))
		if err != nil {
			return nil, fmt.Errorf("open agent Redis queue: %w", err)
		}
	}
	var planner agent.Planner = agent.RulePlanner{}
	if model := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MODEL")); model != "" {
		planner = agent.OllamaPlanner{
			Client: api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.agent.ollama_local", true)),
			Model:  model,
		}
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: store, Context: contextStore, Company: companyStore, Planner: planner, WorkspaceRoot: workspaceRoot, DataRoot: storeRoot, Connectors: connectors, MCP: mcp, RemoteMCP: remoteMCP, Media: media, RedisQueue: redisQueue, Telemetry: telemetry, Push: push, Deployments: deployments})
	if err != nil {
		return nil, err
	}
	runtimeReady = true
	return runtime, nil
}

func loadAgentConnectors(storeRoot string) (*agent.ConnectorManager, error) {
	configPath := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_CONNECTORS"))
	if configPath == "" {
		return agent.NewPersistentConnectorManager(filepath.Join(storeRoot, "connectors.json"))
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var configs []agent.ConnectorConfig
	if err := decodeAgentConfigJSON(data, &configs); err != nil {
		return nil, err
	}
	manager := agent.NewConnectorManager()
	for _, config := range configs {
		if err := manager.Register(config); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func loadAgentDeployments() (*agent.DeploymentManager, error) {
	configPath := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_DEPLOYMENTS"))
	if configPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var configs []agent.DeployConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}
	manager := agent.NewDeploymentManager()
	for _, config := range configs {
		if err := manager.Register(config); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func loadAgentMedia() (*agent.MediaManager, error) {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MEDIA_BASE_URL"))
	if baseURL == "" && strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MEDIA_LOCAL")), "true") {
		baseURL = "http://127.0.0.1:11434/v1"
	}
	if baseURL == "" {
		return nil, nil
	}
	return agent.NewMediaManager(agent.MediaProvider{
		Name:               "configured",
		BaseURL:            baseURL,
		APIKey:             os.Getenv("OLLAMA_AGENT_MEDIA_API_KEY"),
		ImageModel:         os.Getenv("OLLAMA_AGENT_MEDIA_IMAGE_MODEL"),
		VideoModel:         os.Getenv("OLLAMA_AGENT_MEDIA_VIDEO_MODEL"),
		SpeechModel:        os.Getenv("OLLAMA_AGENT_MEDIA_SPEECH_MODEL"),
		TranscriptionModel: os.Getenv("OLLAMA_AGENT_MEDIA_TRANSCRIPTION_MODEL"),
	})
}

func loadAgentMCP(storeRoots ...string) (*agent.MCPManager, error) {
	storeRoot := ""
	if len(storeRoots) > 0 {
		storeRoot = strings.TrimSpace(storeRoots[0])
	}
	configPath := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MCP"))
	if configPath == "" {
		if storeRoot == "" {
			return nil, nil
		}
		return agent.NewPersistentMCPManager(filepath.Join(storeRoot, "mcp.json"))
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var configs []agent.MCPServerConfig
	if err := decodeAgentConfigJSON(data, &configs); err != nil {
		return nil, err
	}
	manager := agent.NewMCPManager()
	for _, config := range configs {
		if err := manager.Register(config); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func loadAgentRemoteMCP(storeRoots ...string) (*agent.RemoteMCPManager, error) {
	storeRoot := ""
	if len(storeRoots) > 0 {
		storeRoot = strings.TrimSpace(storeRoots[0])
	}
	configPath := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REMOTE_MCP"))
	if configPath == "" {
		if storeRoot == "" {
			return nil, nil
		}
		return agent.NewPersistentRemoteMCPManager(filepath.Join(storeRoot, "remote-mcp.json"))
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var configs []agent.RemoteMCPServerConfig
	if err := decodeAgentConfigJSON(data, &configs); err != nil {
		return nil, err
	}
	manager := agent.NewRemoteMCPManager()
	for _, config := range configs {
		if err := manager.Register(config); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func decodeAgentConfigJSON(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("agent config contains trailing JSON")
		}
		return err
	}
	return nil
}

func (a *agentAPI) register(r *gin.Engine) {
	group := r.Group("/api/agent/v1")
	group.Use(a.authMiddleware)
	group.GET("/health", a.health)
	group.GET("/grok/status", a.grokStatus)
	group.GET("/config/safe", a.safeConfig)
	group.GET("/auth/session", a.authSession)
	group.POST("/auth/logout", a.authLogout)
	group.POST("/auth/dev/token", a.devToken)
	group.POST("/auth/mfa/enable", a.enableMFA)
	group.POST("/auth/mfa/disable", a.disableMFA)
	group.POST("/auth/mfa/recovery/generate", a.generateRecoveryCodes)
	group.GET("/auth/oauth/:provider/start", a.oauthStart)
	group.GET("/auth/oauth/:provider/callback", a.oauthCallback)
	group.POST("/auth/oauth/:provider/refresh", a.oauthRefresh)
	group.POST("/auth/oauth/:provider/revoke", a.oauthRevoke)
	group.GET("/auth/saml/:provider/start", a.samlStart)
	group.GET("/auth/saml/:provider/metadata", a.samlMetadata)
	group.POST("/auth/saml/:provider/acs", a.samlACS)
	group.POST("/notifications/register", a.registerPush)
	group.GET("/traces", a.allTraces)
	group.POST("/uploads", a.startUpload)
	group.GET("/uploads/:id", a.getUpload)
	group.PUT("/uploads/:id/chunk", a.uploadChunk)
	group.POST("/uploads/:id/finalize", a.finalizeUpload)
	group.DELETE("/uploads/:id", a.cancelUpload)
	group.POST("/media/ocr", a.mediaOCR)
	group.POST("/media/tone", a.mediaTone)
	group.POST("/orchestration/jobs", a.createOrchestration)
	group.GET("/orchestration/jobs/:id", a.getOrchestration)
	group.POST("/orchestration/jobs/:id/run", a.runOrchestration)
	group.POST("/orchestration/jobs/:id/cancel", a.cancelOrchestration)
	group.POST("/research", a.research)
	group.GET("/devices", a.devices)
	group.POST("/devices/pair/start", a.startDevicePairing)
	group.POST("/devices/pair/complete", a.completeDevicePairing)
	group.POST("/devices/:id/heartbeat", a.deviceHeartbeat)
	group.POST("/devices/:id/revoke", a.revokeDevice)
	group.GET("/devices/:id/connect", a.deviceConnect)
	group.GET("/metrics", a.metrics)
	group.POST("/projects/import/github", a.importGitHubProject)
	group.POST("/projects/import/zip", a.importZIPProject)
	group.POST("/projects/:id/ingest", a.ingestProject)
	group.GET("/builders", a.builders)
	group.GET("/builders/:id", a.getBuilder)
	group.POST("/builders", a.createBuilder)
	group.POST("/builders/:id/visual", a.updateBuilderVisual)
	group.POST("/builders/:id/undo", a.undoBuilder)
	group.POST("/builders/:id/redo", a.redoBuilder)
	group.POST("/builders/:id/preview", a.previewBuilder)
	group.POST("/builders/:id/export", a.exportBuilder)
	group.GET("/builders/:id/download", a.downloadBuilder)
	group.POST("/builders/:id/export/:format", a.exportProfessionalBuilder)
	group.POST("/builders/:id/publish", a.publishBuilder)
	group.GET("/deployments", a.deployments)
	group.POST("/builders/:id/deploy/:provider", a.deployBuilder)
	group.POST("/builders/:id/deploy/:provider/approval", a.requestDeploymentApproval)
	group.POST("/builders/:id/deploy/:provider/approval/:approval_id", a.decideDeploymentApproval)
	group.GET("/builders/:id/preview/*path", a.builderPreviewFile)
	group.GET("/metrics/prometheus", a.prometheus)
	group.GET("/tools", a.tools)
	group.GET("/connectors", a.connectors)
	group.GET("/connector-catalog", connectorCatalog)
	group.POST("/connectors", a.registerConnector)
	group.POST("/connectors/:id/enable", a.enableConnector)
	group.POST("/connectors/:id/disable", a.disableConnector)
	group.DELETE("/connectors/:id", a.removeConnector)
	group.GET("/mcp", a.mcp)
	group.POST("/mcp", a.registerMCP)
	group.POST("/mcp/:id/enable", a.enableMCP)
	group.POST("/mcp/:id/disable", a.disableMCP)
	group.DELETE("/mcp/:id", a.removeMCP)
	group.POST("/remote-mcp/:id/enable", a.enableRemoteMCP)
	group.POST("/remote-mcp/:id/disable", a.disableRemoteMCP)
	group.DELETE("/remote-mcp/:id", a.removeRemoteMCP)
	group.POST("/remote-mcp", a.registerRemoteMCP)
	group.GET("/jobs", a.jobs)
	group.POST("/jobs/:id/replay", a.replayJob)
	group.GET("/skills", a.skills)
	group.POST("/skills", a.registerSkill)
	group.POST("/skills/:id/enable", a.enableSkill)
	group.POST("/skills/:id/disable", a.disableSkill)
	group.DELETE("/skills/:id", a.removeSkill)
	group.GET("/companies", a.companies)
	group.POST("/companies", a.createCompany)
	group.GET("/companies/:id", a.getCompany)
	group.PATCH("/companies/:id", a.updateCompany)
	group.GET("/companies/:id/report", a.companyReport)
	group.POST("/companies/:id/tel-agent", a.companyTelAgent)
	group.GET("/companies/:id/tel-agent/history", a.companyTelAgentHistory)
	group.GET("/companies/:id/agents", a.companyAgents)
	group.GET("/companies/:id/growth/report", a.companyGrowthReport)
	group.GET("/companies/:id/social/report", a.companySocialReport)
	group.POST("/companies/:id/roadmap", a.addCompanyRoadmap)
	group.POST("/companies/:id/goals", a.addCompanyGoal)
	group.POST("/companies/:id/backlog", a.addCompanyBacklog)
	group.POST("/companies/:id/cycles", a.addCompanyCycle)
	group.POST("/companies/:id/campaigns", a.addCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/approve", a.approveCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/launch", a.launchCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/pause", a.pauseCompanyCampaign)
	group.POST("/companies/:id/affiliate-programs", a.addCompanyAffiliateProgram)
	group.POST("/companies/:id/affiliate-programs/:program_id/approve", a.approveCompanyAffiliateProgram)
	group.POST("/companies/:id/affiliate-links", a.addCompanyAffiliateLink)
	group.POST("/companies/:id/affiliate-links/:link_id/conversion", a.recordCompanyAffiliateConversion)
	group.POST("/companies/:id/products", a.addCompanyProduct)
	group.POST("/companies/:id/orders", a.createCompanyOrder)
	group.POST("/companies/:id/orders/:order_id/approve", a.approveCompanyOrder)
	group.POST("/companies/:id/orders/:order_id/fulfill", a.fulfillCompanyOrder)
	group.POST("/companies/:id/social/accounts", a.addCompanySocialAccount)
	group.POST("/companies/:id/social/drafts", a.createCompanySocialDraft)
	group.POST("/companies/:id/social/drafts/:draft_id/approve", a.approveCompanySocialDraft)
	group.POST("/companies/:id/social/drafts/:draft_id/publish", a.publishCompanySocialDraft)
	group.POST("/companies/:id/social/metrics", a.recordCompanySocialMetric)
	group.POST("/companies/:id/pause", a.pauseCompany)
	group.POST("/companies/:id/resume", a.resumeCompany)
	group.POST("/companies/:id/anomalies", a.recordCompanyAnomaly)
	group.POST("/companies/:id/spend", a.recordCompanySpend)
	group.POST("/companies/:id/approvals/:approval_id/decide", a.decideCompanyApprovalByID)
	group.POST("/companies/:id/agents/:agent_id/pause", a.pauseCompanyAgent)
	group.POST("/companies/:id/agents/:agent_id/resume", a.resumeCompanyAgent)
	group.POST("/companies/:id/agents/:agent_id/spend", a.recordCompanyAgentSpend)
	group.GET("/schedules", a.schedules)
	group.POST("/schedules", a.createSchedule)
	group.PATCH("/schedules/:id", a.updateSchedule)
	group.DELETE("/schedules/:id", a.deleteSchedule)
	group.POST("/webhooks/:schedule_id", a.webhook)
	group.GET("/projects", a.projects)
	group.POST("/projects", a.createProject)
	group.GET("/projects/:id", a.getProject)
	group.PATCH("/projects/:id", a.updateProject)
	group.DELETE("/projects/:id", a.deleteProject)
	group.POST("/projects/:id/memories", a.addMemory)
	group.GET("/projects/:id/memories", a.searchMemories)
	group.GET("/projects/:id/ask", a.askProjectDocuments)
	group.GET("/collab/:project_id", a.collabSnapshot)
	group.GET("/collab/:project_id/stream", a.collabStream)
	group.POST("/collab/:project_id/comments", a.collabComment)
	group.POST("/collab/:project_id/presence", a.collabPresence)
	group.POST("/missions", a.createMission)
	group.GET("/missions", a.missions)
	group.GET("/notifications", a.notifications)
	group.GET("/notifications/stream", a.notificationStream)
	group.GET("/creations", a.creations)
	group.GET("/missions/:id", a.getMission)
	group.GET("/missions/:id/events", a.events)
	group.GET("/missions/:id/events/stream", a.eventStream)
	group.GET("/missions/:id/traces", a.traces)
	group.GET("/missions/:id/artifacts/:artifact_id", a.artifact)
	group.POST("/missions/:id/run", a.runMission)
	group.POST("/missions/:id/cancel", a.cancelMission)
	group.POST("/missions/:id/approvals/:approval_id", a.decideApproval)
	group.GET("/missions/:id/worktree", a.getMissionWorktree)
	group.POST("/missions/:id/merge", a.mergeMissionWorktree)
	group.POST("/whatsapp/webhook", a.whatsappWebhook)
	group.GET("/whatsapp/webhook", a.whatsappWebhookVerify)
	group.GET("/whatsapp/status", a.whatsappStatus)
	group.POST("/whatsapp/send", a.whatsappSend)
	group.GET("/whatsapp/dlq", a.whatsappDLQ)
	group.DELETE("/whatsapp/dlq", a.whatsappClearDLQ)
	group.GET("/whatsapp/allowlist", a.whatsappAllowlist)
	group.POST("/whatsapp/allowlist", a.whatsappSetContactPolicy)
	group.DELETE("/whatsapp/allowlist/:phone", a.whatsappRemoveContactPolicy)
	group.POST("/whatsapp/config", a.whatsappConfig)
	group.GET("/supervisor/status", a.supervisorStatus)
	group.POST("/supervisor/config", a.supervisorConfig)
	group.POST("/supervisor/tick", a.supervisorTick)
	group.GET("/egress/logs", a.getEgressLogs)
	group.GET("/egress/status", a.getEgressStatus)
}

func (a *agentAPI) authMiddleware(c *gin.Context) {
	if !a.authRequired {
		if !agentOriginAllowed(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "request origin is not allowed"})
			return
		}
		// Local/dev mode is deliberately single-tenant. Do not honor an
		// organization header or pre-populated context supplied by the client.
		// Every downstream handler and scoped store receives this fixed scope.
		c.Set("agent.organization", agent.Organization{ID: agent.LocalOrganizationID})
		c.Set("agent.local_mode", true)
		c.Next()
		return
	}
	if !agentOriginAllowed(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "request origin is not allowed"})
		return
	}
	if strings.HasSuffix(c.Request.URL.Path, "/auth/dev/token") && isDevTokenRequestAllowed(c) {
		c.Next()
		return
	}
	if isPublicSSORoute(c) && strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_SSO_PUBLIC")), "true") {
		c.Next()
		return
	}
	if isCompanionConnectRoute(c.FullPath()) {
		c.Next()
		return
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bearer token is required"})
		return
	}
	user, organization, membership, err := a.auth.Authenticate(strings.TrimSpace(header[len("Bearer "):]))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	if requested := strings.TrimSpace(c.GetHeader("X-Ollama-Organization")); requested != "" && requested != organization.ID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "organization scope mismatch"})
		return
	}
	if user.MFAEnabled {
		mfaCode := strings.TrimSpace(c.GetHeader("X-Ollama-MFA-Code"))
		recoveryCode := strings.TrimSpace(c.GetHeader("X-Ollama-MFA-Recovery-Code"))
		var mfaErr error
		if recoveryCode != "" {
			mfaErr = a.auth.VerifyRecoveryCodeWithThrottle(user.ID, recoveryCode, c.Request.RemoteAddr, time.Now().UTC())
		} else {
			mfaErr = a.auth.VerifyMFAWithThrottle(user.ID, mfaCode, c.Request.RemoteAddr, time.Now().UTC())
		}
		if mfaErr != nil {
			var throttle *agent.MFAThrottleError
			if errors.As(mfaErr, &throttle) {
				retryAfter := int64(throttle.RetryAfter / time.Second)
				if throttle.RetryAfter%time.Second != 0 {
					retryAfter++
				}
				if retryAfter < 1 {
					retryAfter = 1
				}
				c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "mfa verification temporarily locked"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "mfa verification required"})
			return
		}
	}
	action := "read"
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		action = "execute"
	}
	if _, err := a.auth.Authorize(user.ID, organization.ID, action); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": agent.RedactDLP(err.Error())})
		return
	}
	c.Set("agent.user", user)
	c.Set("agent.organization", organization)
	c.Set("agent.membership", membership)
	c.Next()
}

func agentOriginAllowed(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions || isPublicSSORoute(c) {
		return true
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		return true
	}
	if strings.ContainsAny(origin, "\r\n") {
		return false
	}
	for _, allowed := range envconfig.AllowedOrigins() {
		allowed = strings.TrimRight(strings.TrimSpace(allowed), "/")
		if allowed == "*" {
			continue
		}
		if originMatchesAllowed(origin, allowed) {
			return true
		}
	}
	return false
}

func originMatchesAllowed(origin, allowed string) bool {
	if strings.HasSuffix(allowed, ":*") {
		allowedURL, err := url.Parse(strings.TrimSuffix(allowed, ":*"))
		originURL, originErr := url.Parse(origin)
		if err != nil || originErr != nil || allowedURL.Scheme == "" || allowedURL.Hostname() == "" || originURL.Scheme == "" || originURL.Hostname() == "" || allowedURL.User != nil || originURL.User != nil || allowedURL.Path != "" || originURL.Path != "" || originURL.RawQuery != "" || originURL.Fragment != "" {
			return false
		}
		return strings.EqualFold(originURL.Scheme, allowedURL.Scheme) && strings.EqualFold(originURL.Hostname(), allowedURL.Hostname()) && originURL.Port() != ""
	}
	if strings.HasSuffix(allowed, "://*") {
		return strings.HasPrefix(origin, strings.TrimSuffix(allowed, "*"))
	}
	return origin == allowed
}

func isCompanionConnectRoute(fullPath string) bool {
	return strings.TrimSpace(fullPath) == "/api/agent/v1/devices/:id/connect"
}

func isPublicSSORoute(c *gin.Context) bool {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	for _, prefix := range []string{"/api/agent/v1/auth/oauth/", "/api/agent/v1/auth/saml/"} {
		if strings.HasPrefix(path, prefix) && (strings.HasSuffix(path, "/start") || strings.HasSuffix(path, "/callback") || strings.HasSuffix(path, "/metadata") || strings.HasSuffix(path, "/acs")) {
			return true
		}
	}
	return false
}

func isDevTokenRequestAllowed(c *gin.Context) bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_DEV")), "true") && isLoopbackRemoteAddr(c.Request.RemoteAddr)
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = strings.Trim(remoteAddr, "[]")
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (a *agentAPI) scopedRuntime(c *gin.Context) *agent.Runtime {
	if value, ok := c.Get("agent.organization"); ok {
		if organization, ok := value.(agent.Organization); ok {
			return a.runtime.WithOrganizationContext(c.Request.Context(), organization.ID)
		}
	}
	if a != nil && !a.authRequired && a.runtime != nil {
		return a.runtime.WithOrganizationContext(c.Request.Context(), agent.LocalOrganizationID)
	}
	return a.runtime
}

func (a *agentAPI) organizationID(c *gin.Context) string {
	if organizationID := strings.TrimSpace(agentOrganizationID(c)); organizationID != "" {
		return organizationID
	}
	if a != nil && !a.authRequired {
		return agent.LocalOrganizationID
	}
	return ""
}

func agentOrganizationID(c *gin.Context) string {
	if value, ok := c.Get("agent.organization"); ok {
		if organization, ok := value.(agent.Organization); ok {
			return organization.ID
		}
	}
	if c != nil && c.Request != nil {
		if value := strings.TrimSpace(c.GetHeader("X-Ollama-Organization")); value != "" {
			return value
		}
	}
	return agent.LocalOrganizationID
}

func contextRecordOwnedByOrganization(owner, organizationID string) bool {
	owner = strings.TrimSpace(owner)
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return false
	}
	if organizationID == agent.LocalOrganizationID {
		return owner == "" || owner == agent.LocalOrganizationID
	}
	return owner == organizationID
}

func (a *agentAPI) requireContextOrganization(c *gin.Context) (string, bool) {
	organizationID := strings.TrimSpace(a.organizationID(c))
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return "", false
	}
	return organizationID, true
}

func agentActorID(c *gin.Context) string {
	if value, ok := c.Get("agent.user"); ok {
		if user, ok := value.(agent.User); ok && strings.TrimSpace(user.ID) != "" {
			return user.ID
		}
	}
	if value := strings.TrimSpace(c.GetHeader("X-Ollama-User")); value != "" {
		return value
	}
	return "local"
}

func (a *agentAPI) missionForRequest(c *gin.Context) (agent.Mission, error) {
	return a.missionByID(c, c.Param("id"))
}

func (a *agentAPI) missionByID(c *gin.Context, id string) (agent.Mission, error) {
	mission, err := a.scopedRuntime(c).GetMission(strings.TrimSpace(id))
	if err != nil {
		return agent.Mission{}, err
	}
	if !a.authRequired {
		return mission, nil
	}
	value, _ := c.Get("agent.organization")
	organization, organizationOK := value.(agent.Organization)
	if !organizationOK || organization.ID == "" || mission.OrganizationID == "" || mission.OrganizationID != organization.ID {
		return agent.Mission{}, os.ErrNotExist
	}
	return mission, nil
}

func missionVersionMatches(c *gin.Context, mission agent.Mission) bool {
	value := strings.TrimSpace(c.GetHeader("If-Match"))
	if value == "" {
		return true
	}
	value = strings.Trim(value, "\"")
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version != mission.Version {
		c.JSON(http.StatusConflict, gin.H{"error": "mission version conflict", "current_version": mission.Version, "mission_id": mission.ID})
		return false
	}
	return true
}

func (a *agentAPI) health(c *gin.Context) {
	runtimeReady := a.runtime != nil
	storeStatus := "local"
	if strings.TrimSpace(os.Getenv("OLLAMA_AGENT_DATABASE_URL")) != "" {
		storeStatus = "postgres-configured"
	}
	queueStatus := "local"
	if strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REDIS_URL")) != "" {
		queueStatus = "redis-configured"
	}
	sandboxStatus := strings.ToLower(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_MODE")))
	if sandboxStatus == "" {
		sandboxStatus = "best-effort"
	}
	// Real queue depth instead of a static label: surface how much work is
	// pending/running and whether anything has landed in the dead-letter queue.
	pending, running, dead := 0, 0, 0
	if a.runtime != nil {
		pending = len(a.runtime.QueueJobs(agent.QueuePending))
		running = len(a.runtime.QueueJobs(agent.QueueRunning))
		dead = len(a.runtime.QueueJobs(agent.QueueDeadLetter))
	}
	status := "ok"
	if !runtimeReady {
		status = "degraded"
	}
	queueHealth := "ok"
	if dead > 0 {
		// Dead-lettered jobs are a real, actionable problem; reflect it.
		queueHealth = "degraded"
		if status == "ok" {
			status = "degraded"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     status,
		"runtime":    "agent-v1",
		"checked_at": time.Now().UTC(),
		"subsystems": gin.H{
			"agent": gin.H{"status": map[bool]string{true: "ok", false: "degraded"}[runtimeReady], "detail": "Motor de missões"},
			"store": gin.H{"status": storeStatus, "detail": "Persistência de missões e configurações"},
			"queue": gin.H{
				"status":      queueStatus,
				"health":      queueHealth,
				"detail":      "Fila de execução",
				"pending":     pending,
				"running":     running,
				"dead_letter": dead,
			},
			"sandbox": gin.H{"status": sandboxStatus, "detail": "Execução isolada"},
		},
	})
}

func (a *agentAPI) safeConfig(c *gin.Context) {
	store := "local"
	if strings.TrimSpace(os.Getenv("OLLAMA_AGENT_DATABASE_URL")) != "" {
		store = "postgres"
	}
	queue := "local"
	if strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REDIS_URL")) != "" {
		queue = "redis"
	}
	sandboxMode := strings.ToLower(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_MODE")))
	if sandboxMode == "" {
		sandboxMode = "best-effort"
	}
	connectorsConfigured := envConfigured("OLLAMA_AGENT_CONNECTORS")
	mcpConfigured := envConfigured("OLLAMA_AGENT_MCP") || envConfigured("OLLAMA_AGENT_REMOTE_MCP")
	mediaConfigured := envConfigured("OLLAMA_AGENT_MEDIA_BASE_URL")
	deploymentsConfigured := envConfigured("OLLAMA_AGENT_DEPLOYMENTS")
	if a.runtime != nil {
		connectorsConfigured = connectorsConfigured || len(a.runtime.Connectors()) > 0
		mcpConfigured = mcpConfigured || len(a.runtime.MCPServers()) > 0 || len(a.runtime.RemoteMCPServers()) > 0
		mediaConfigured = mediaConfigured || a.runtime.Media() != nil
		deploymentsConfigured = deploymentsConfigured || (a.runtime.Deployments() != nil && len(a.runtime.Deployments().List()) > 0)
	}
	if a.authRequired {
		deploymentsConfigured = a.runtime != nil && a.runtime.Deployments() != nil && len(a.runtime.Deployments().ListForOrganization(companyOrganizationID(c))) > 0
	}
	c.JSON(http.StatusOK, gin.H{
		"runtime":                          "agent-v1",
		"store":                            store,
		"queue":                            queue,
		"auth_required":                    a.authRequired,
		"approval_gated_tools":             true,
		"workspace_isolation":              true,
		"planner_model_configured":         envConfigured("OLLAMA_AGENT_MODEL"),
		"embedding_configured":             envConfigured("OLLAMA_AGENT_EMBED_MODEL"),
		"connectors_configured":            connectorsConfigured,
		"mcp_configured":                   mcpConfigured,
		"media_configured":                 mediaConfigured,
		"deployments_configured":           deploymentsConfigured,
		"otlp_configured":                  envConfigured("OLLAMA_AGENT_OTLP_ENDPOINT"),
		"push_configured":                  envConfigured("OLLAMA_AGENT_PUSH_ENDPOINT"),
		"sandbox_mode":                     sandboxMode,
		"sandbox_strict_cgroup_configured": sandboxMode == "strict" && envConfigured("OLLAMA_AGENT_SANDBOX_CGROUP_ROOT"),
	})
}

func envConfigured(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) != "" || strings.TrimSpace(os.Getenv(name+"_FILE")) != ""
}

func (a *agentAPI) authSession(c *gin.Context) {
	if !a.authRequired {
		c.JSON(http.StatusOK, gin.H{"authenticated": false, "mode": "local"})
		return
	}
	user, _ := c.Get("agent.user")
	if value, ok := user.(agent.User); ok {
		user = value.Public()
	}
	organization, _ := c.Get("agent.organization")
	membership, _ := c.Get("agent.membership")
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "user": user, "organization": organization, "membership": membership})
}

func (a *agentAPI) authLogout(c *gin.Context) {
	if !a.authRequired {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bearer token is required"})
		return
	}
	if err := a.auth.RevokeToken(strings.TrimSpace(header[len("Bearer "):])); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (a *agentAPI) enableMFA(c *gin.Context) {
	value, ok := c.Get("agent.user")
	user, userOK := value.(agent.User)
	if !ok || !userOK {
		writeAgentError(c, http.StatusUnauthorized, errors.New("authenticated user is required"))
		return
	}
	var request struct {
		Secret string `json:"secret"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.auth.EnableMFA(user.ID, request.Secret)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": updated.MFAEnabled, "issuer": "Ollama DZ23 Agentic", "account": updated.Email})
}

func (a *agentAPI) disableMFA(c *gin.Context) {
	value, ok := c.Get("agent.user")
	user, userOK := value.(agent.User)
	if !ok || !userOK {
		writeAgentError(c, http.StatusUnauthorized, errors.New("authenticated user is required"))
		return
	}
	updated, err := a.auth.DisableMFA(user.ID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": updated.MFAEnabled})
}

func (a *agentAPI) generateRecoveryCodes(c *gin.Context) {
	value, ok := c.Get("agent.user")
	user, userOK := value.(agent.User)
	if !ok || !userOK {
		writeAgentError(c, http.StatusUnauthorized, errors.New("authenticated user is required"))
		return
	}
	updated, codes, err := a.auth.GenerateRecoveryCodes(user.ID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": updated.Public(), "recovery_codes": codes, "warning": "store these codes securely; they are shown only once"})
}

func (a *agentAPI) registerPush(c *gin.Context) {
	if a.push == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "push service is not configured"})
		return
	}
	var request struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	user, _ := c.Get("agent.user")
	organization, _ := c.Get("agent.organization")
	userValue, userOK := user.(agent.User)
	organizationValue, organizationOK := organization.(agent.Organization)
	if !userOK || !organizationOK {
		writeAgentError(c, http.StatusUnauthorized, errors.New("authenticated user and organization are required"))
		return
	}
	subscription, err := a.push.Register(request.Token, request.Platform, userValue.ID, organizationValue.ID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	subscription.Token = "redacted"
	c.JSON(http.StatusCreated, subscription)
}

func (a *agentAPI) devToken(c *gin.Context) {
	if !isDevTokenRequestAllowed(c) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	var input struct {
		Email        string `json:"email"`
		Name         string `json:"name"`
		Organization string `json:"organization"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	user, err := a.auth.CreateUser(input.Email, input.Name)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	organization, _, err := a.auth.CreateOrganization(input.Organization, user)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	raw, token, err := a.auth.IssueToken(user.ID, organization.ID, 24*time.Hour)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"access_token": raw, "token": token, "user": user.Public(), "organization": organization})
}

func oauthProviderFromEnv(name string) agent.OAuthProvider {
	key := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(name)))
	prefix := "OLLAMA_AGENT_OAUTH_" + key
	redirects := make([]string, 0)
	for _, value := range strings.FieldsFunc(os.Getenv(prefix+"_REDIRECT_URIS"), func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if value = strings.TrimSpace(value); value != "" {
			redirects = append(redirects, value)
		}
	}
	return agent.OAuthProvider{Name: name, AuthorizeURL: os.Getenv(prefix + "_AUTHORIZE_URL"), TokenURL: os.Getenv(prefix + "_TOKEN_URL"), RevocationURL: os.Getenv(prefix + "_REVOCATION_URL"), UserInfoURL: os.Getenv(prefix + "_USERINFO_URL"), IssuerURL: os.Getenv(prefix + "_ISSUER_URL"), Audience: os.Getenv(prefix + "_AUDIENCE"), ClientIDEnv: os.Getenv(prefix + "_CLIENT_ID_ENV"), SecretEnv: os.Getenv(prefix + "_SECRET_ENV"), RedirectURIs: redirects, AllowLoopbackRedirect: strings.EqualFold(os.Getenv(prefix+"_ALLOW_LOOPBACK_REDIRECT"), "true")}
}

func prepareOIDCProvider(ctx context.Context, provider agent.OAuthProvider) (agent.OAuthProvider, error) {
	if strings.TrimSpace(provider.IssuerURL) == "" {
		return provider, provider.Validate()
	}
	discovery, err := provider.Discover(ctx, newServerEgressClient("server.agent.provider_discovery", false))
	if err != nil {
		return agent.OAuthProvider{}, err
	}
	if provider.AuthorizeURL == "" {
		provider.AuthorizeURL = discovery.AuthorizationEndpoint
	}
	if provider.TokenURL == "" {
		provider.TokenURL = discovery.TokenEndpoint
	}
	if provider.UserInfoURL == "" {
		provider.UserInfoURL = discovery.UserInfoEndpoint
	}
	return provider, provider.Validate()
}

func samlProviderFromEnv(name string) agent.SAMLProviderConfig {
	key := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(name)))
	prefix := "OLLAMA_AGENT_SAML_" + key
	return agent.SAMLProviderConfig{
		Name:               name,
		EntityID:           os.Getenv(prefix + "_ENTITY_ID"),
		IDPMetadataURL:     os.Getenv(prefix + "_IDP_METADATA_URL"),
		MetadataURL:        os.Getenv(prefix + "_METADATA_URL"),
		ACSURL:             os.Getenv(prefix + "_ACS_URL"),
		SPPrivateKeyFile:   os.Getenv(prefix + "_SP_PRIVATE_KEY_FILE"),
		SPCertificateFile:  os.Getenv(prefix + "_SP_CERTIFICATE_FILE"),
		DefaultRedirectURI: os.Getenv(prefix + "_DEFAULT_REDIRECT_URI"),
		AllowIDPInitiated:  strings.EqualFold(os.Getenv(prefix+"_ALLOW_IDP_INITIATED"), "true"),
	}
}

func loadSAMLService(ctx context.Context, name string) (*agent.SAMLService, error) {
	config := samlProviderFromEnv(name)
	return agent.NewSAMLService(ctx, config, newServerEgressClient("server.agent.saml", false))
}

func (a *agentAPI) samlService(ctx context.Context, name string) (*agent.SAMLService, error) {
	a.samlMu.Lock()
	defer a.samlMu.Unlock()
	if service, ok := a.samlServices[name]; ok {
		return service, nil
	}
	service, err := loadSAMLService(ctx, name)
	if err != nil {
		return nil, err
	}
	a.samlServices[name] = service
	return service, nil
}

func (a *agentAPI) oauthStart(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(c.Query("redirect_uri"))
	verifier := strings.TrimSpace(c.Query("code_verifier"))
	if redirectErr != nil || verifier == "" {
		if redirectErr == nil {
			redirectErr = errors.New("redirect_uri and PKCE code_verifier are required")
		}
		writeAgentError(c, http.StatusBadRequest, redirectErr)
		return
	}
	userID := ""
	if value, ok := c.Get("agent.user"); ok {
		if user, ok := value.(agent.User); ok {
			userID = user.ID
		}
	}
	nonce := fmt.Sprintf("%x", sha256.Sum256([]byte(provider.Name+"|"+redirectURI+"|"+verifier+"|"+time.Now().UTC().String())))
	state, _, err := a.auth.CreateOAuthStateWithNonce(provider.Name, redirectURI, verifier, nonce, userID, 5*time.Minute)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	scopes := strings.Fields(c.Query("scope"))
	if len(scopes) == 0 {
		scopes = []string{"openid", "email"}
	}
	authorizationURL, err := provider.AuthorizationURLWithNonce(state, nonce, scopes)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": provider.Name, "authorization_url": authorizationURL, "state": state, "expires_in": 300})
}

func (a *agentAPI) oauthCallback(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(c.Query("redirect_uri"))
	code := strings.TrimSpace(c.Query("code"))
	stateValue := strings.TrimSpace(c.Query("state"))
	if redirectErr != nil || code == "" || stateValue == "" {
		if redirectErr == nil {
			redirectErr = errors.New("code, state and redirect_uri are required")
		}
		writeAgentError(c, http.StatusBadRequest, redirectErr)
		return
	}
	state, err := a.auth.ConsumeOAuthState(stateValue, provider.Name, redirectURI)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	payload, err := provider.ExchangeCode(c.Request.Context(), newServerEgressClient("server.agent.oauth.exchange", false), code, redirectURI, state.CodeVerifier)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	if provider.IssuerURL != "" {
		idToken, _ := payload["id_token"].(string)
		if idToken == "" {
			writeAgentError(c, http.StatusBadGateway, errors.New("oidc token response has no id_token"))
			return
		}
		claims, validationErr := provider.ValidateIDToken(c.Request.Context(), newServerEgressClient("server.agent.oauth.id_token", false), idToken, state.Nonce)
		if validationErr != nil {
			writeAgentError(c, http.StatusBadGateway, validationErr)
			return
		}
		payload["id_token_claims"] = claims
	}
	userID := state.UserID
	if userID == "" && (provider.UserInfoURL != "" || provider.IssuerURL != "") {
		accessToken, _ := payload["access_token"].(string)
		userinfo, userinfoErr := provider.FetchUserInfo(c.Request.Context(), newServerEgressClient("server.agent.oauth.userinfo", false), accessToken)
		if userinfoErr != nil {
			writeAgentError(c, http.StatusBadGateway, userinfoErr)
			return
		}
		payload["userinfo"] = userinfo
	}
	if userID == "" {
		user, _, _, provisionErr := a.auth.ProvisionOAuthUser(payload, provider.Name)
		if provisionErr != nil {
			writeAgentError(c, http.StatusBadRequest, provisionErr)
			return
		}
		userID = user.ID
	}
	organization, _, err := a.auth.FirstOrganization(userID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, errors.New("OAuth user has no organization"))
		return
	}
	credential, err := a.auth.StoreOAuthCredential(provider.Name, userID, organization.ID, payload)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	localToken, session, err := a.auth.IssueToken(userID, organization.ID, 24*time.Hour)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": localToken, "token": session, "credential_id": credential.ID, "provider": provider.Name, "organization": organization, "expires_at": credential.ExpiresAt})
}

func oauthCredentialPublic(credential agent.OAuthCredential) gin.H {
	return gin.H{
		"credential_id": credential.ID,
		"provider":      credential.Provider,
		"expires_at":    credential.ExpiresAt,
		"updated_at":    credential.UpdatedAt,
		"revoked_at":    credential.RevokedAt,
	}
}

func (a *agentAPI) oauthRefresh(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var request struct {
		CredentialID string `json:"credential_id"`
	}
	if err := decodeJSON(c, &request); err != nil || strings.TrimSpace(request.CredentialID) == "" {
		if err == nil {
			err = errors.New("credential_id is required")
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	credential, err := a.auth.RefreshOAuthCredentialForOrganization(c.Request.Context(), agentOrganizationID(c), provider, request.CredentialID, newServerEgressClient("server.agent.oauth.refresh", false))
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, agent.ErrOAuthCredentialConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, agent.ErrOAuthCredentialRevoked) || strings.Contains(err.Error(), "outside the active organization") {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, oauthCredentialPublic(credential))
}

func (a *agentAPI) oauthRevoke(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var request struct {
		CredentialID string `json:"credential_id"`
	}
	if err := decodeJSON(c, &request); err != nil || strings.TrimSpace(request.CredentialID) == "" {
		if err == nil {
			err = errors.New("credential_id is required")
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if err := a.auth.RevokeOAuthCredentialForOrganization(c.Request.Context(), agentOrganizationID(c), request.CredentialID, provider, newServerEgressClient("server.agent.oauth.revoke", false)); err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, agent.ErrOAuthCredentialConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, agent.ErrOAuthCredentialRevoked) || strings.Contains(err.Error(), "outside the active organization") {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *agentAPI) samlStart(c *gin.Context) {
	service, err := a.samlService(c.Request.Context(), c.Param("provider"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirect, relay, err := service.Start(c.Query("redirect_uri"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": service.Provider.Name, "authorization_url": redirect, "relay_state": relay, "expires_in": 300})
}

func (a *agentAPI) samlMetadata(c *gin.Context) {
	service, err := a.samlService(c.Request.Context(), c.Param("provider"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	service.Metadata(c.Writer, c.Request)
}

func (a *agentAPI) samlACS(c *gin.Context) {
	service, err := a.samlService(c.Request.Context(), c.Param("provider"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	claims, err := service.ParseACS(c.Request)
	if err != nil {
		writeAgentError(c, http.StatusUnauthorized, err)
		return
	}
	user, organization, _, err := a.auth.ProvisionOAuthUser(claims, service.Provider.Name)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	localToken, session, err := a.auth.IssueToken(user.ID, organization.ID, 24*time.Hour)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": localToken, "token": session, "provider": service.Provider.Name, "organization": organization, "user": user.Public()})
}

func (a *agentAPI) metrics(c *gin.Context) {
	if a.authRequired {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, a.runtime.Metrics())
}

func (a *agentAPI) prometheus(c *gin.Context) {
	if a.authRequired {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(a.runtime.Metrics().Prometheus()))
}

func (a *agentAPI) connectors(c *gin.Context) {
	organizationID := agentOrganizationID(c)
	if !a.authRequired && organizationID == "" {
		organizationID = agent.LocalOrganizationID
	}
	c.JSON(http.StatusOK, gin.H{"connectors": a.runtime.ConnectorsForOrganization(organizationID)})
}

func (a *agentAPI) mcp(c *gin.Context) {
	organizationID := agentOrganizationID(c)
	if !a.authRequired && organizationID == "" {
		organizationID = agent.LocalOrganizationID
	}
	c.JSON(http.StatusOK, gin.H{"servers": a.runtime.MCPServersForOrganization(organizationID), "remote_servers": a.runtime.RemoteMCPServersForOrganization(organizationID)})
}

func (a *agentAPI) jobs(c *gin.Context) {
	jobs, err := a.scopedRuntime(c).QueueJobsForOrganization(a.organizationID(c), agent.QueueStatus(c.Query("status")))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (a *agentAPI) replayJob(c *gin.Context) {
	job, err := a.scopedRuntime(c).ReplayJobForOrganization(c.Param("id"), a.organizationID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, job)
}

func (a *agentAPI) tools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tools": a.runtime.ListTools()})
}

func (a *agentAPI) skills(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"skills": a.context.SkillsForOrganization(organizationID)})
}

func (a *agentAPI) schedules(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": a.context.ListSchedulesForOrganization(organizationID)})
}

func (a *agentAPI) createSchedule(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var schedule agent.Schedule
	if err := decodeJSON(c, &schedule); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	schedule.OrganizationID = organizationID
	created, err := a.context.CreateSchedule(schedule)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (a *agentAPI) updateSchedule(c *gin.Context) {
	var schedule agent.Schedule
	if err := decodeJSON(c, &schedule); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	updated, err := a.context.UpdateScheduleForOrganization(organizationID, c.Param("id"), schedule)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (a *agentAPI) deleteSchedule(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	if err := a.context.DeleteScheduleForOrganization(organizationID, c.Param("id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *agentAPI) webhook(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	schedule, err := a.context.GetScheduleForOrganization(c.Param("schedule_id"), organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	secret := strings.TrimSpace(schedule.WebhookSecretEnv)
	if secret == "" || !verifyAgentWebhook(os.Getenv(secret), c.GetHeader("X-Ollama-Agent-Secret")) {
		writeAgentError(c, http.StatusUnauthorized, errors.New("webhook secret is invalid"))
		return
	}
	scheduleOrganization := strings.TrimSpace(schedule.OrganizationID)
	if scheduleOrganization == "" {
		scheduleOrganization = "local"
	}
	if scheduleOrganization != companyOrganizationID(c) {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(c.GetHeader("X-Ollama-Webhook-ID"))
	}
	if idempotencyKey == "" || len(idempotencyKey) > 200 {
		writeAgentError(c, http.StatusBadRequest, errors.New("Idempotency-Key is required and must be <= 200 characters"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	var payload any
	err = decoder.Decode(&payload)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeAgentError(c, http.StatusBadRequest, errors.New("webhook body must contain exactly one JSON value"))
		return
	}
	if err := agent.ValidateOutboundPayload(map[string]any{"objective": schedule.Objective, "payload": payload}); err != nil {
		writeAgentError(c, http.StatusUnprocessableEntity, errors.New("webhook payload rejected by data-egress policy"))
		return
	}
	replay := a.runtime.WebhookReplay()
	if replay == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("webhook replay store is unavailable"))
		return
	}
	missionID, err := replay.MissionID(schedule.ID, idempotencyKey)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	runtime := a.scopedRuntime(c)
	if err := replay.Begin(schedule.ID, idempotencyKey); err != nil {
		if errors.Is(err, agent.ErrWebhookReplay) || errors.Is(err, agent.ErrWebhookReplayInProgress) {
			if existing, getErr := runtime.GetMission(missionID); getErr == nil {
				c.JSON(http.StatusAccepted, existing)
				return
			}
			if errors.Is(err, agent.ErrWebhookReplayInProgress) {
				writeAgentError(c, http.StatusConflict, err)
			} else {
				writeAgentError(c, http.StatusConflict, agent.ErrWebhookReplay)
			}
		} else {
			writeAgentError(c, http.StatusServiceUnavailable, errors.New("webhook replay store is unavailable"))
		}
		return
	}
	objective, err := json.Marshal(payload)
	if err != nil {
		_ = replay.Release(schedule.ID, idempotencyKey)
		writeAgentError(c, http.StatusBadRequest, errors.New("webhook JSON payload is invalid"))
		return
	}
	mission, err := runtime.CreateMission(c.Request.Context(), agent.CreateMissionRequest{MissionID: missionID, Objective: schedule.Objective + "\nWebhook payload:\n" + string(objective), Model: schedule.Model, Workspace: schedule.Workspace, ProjectID: schedule.ProjectID, OrganizationID: schedule.OrganizationID, AutoRun: true})
	if err != nil {
		if existing, getErr := runtime.GetMission(missionID); getErr == nil {
			if completeErr := replay.Complete(schedule.ID, idempotencyKey); completeErr != nil {
				writeAgentError(c, http.StatusServiceUnavailable, errors.New("webhook mission was accepted but replay state could not be finalized"))
				return
			}
			c.JSON(http.StatusAccepted, existing)
			return
		}
		if releaseErr := replay.Release(schedule.ID, idempotencyKey); releaseErr != nil {
			writeAgentError(c, http.StatusServiceUnavailable, errors.New("webhook mission creation failed and replay claim could not be released"))
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if err := replay.Complete(schedule.ID, idempotencyKey); err != nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("webhook mission was accepted but replay state could not be finalized"))
		return
	}
	c.JSON(http.StatusAccepted, mission)
}

func verifyAgentWebhook(expected, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	expectedSum := sha256.Sum256([]byte(expected))
	providedSum := sha256.Sum256([]byte(provided))
	return hmac.Equal(expectedSum[:], providedSum[:])
}

func (a *agentAPI) projects(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	projects := a.context.ListProjects()
	if organizationID != "" {
		filtered := projects[:0]
		for _, project := range projects {
			if contextRecordOwnedByOrganization(project.OrganizationID, organizationID) {
				filtered = append(filtered, project)
			}
		}
		projects = filtered
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (a *agentAPI) createProject(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var request struct {
		Name string `json:"name"`
		Root string `json:"root,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	project, err := a.context.CreateProject(request.Name, request.Root, organizationID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, project)
}

func (a *agentAPI) getProject(c *gin.Context) {
	project, err := a.projectForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (a *agentAPI) updateProject(c *gin.Context) {
	current, err := a.projectForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Name string `json:"name"`
		Root string `json:"root,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.context.UpdateProject(current.ID, request.Name, request.Root)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (a *agentAPI) deleteProject(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if err := a.context.DeleteProject(c.Param("id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *agentAPI) projectForRequest(c *gin.Context) (agent.Project, error) {
	return a.projectForRequestID(c, c.Param("id"))
}

func (a *agentAPI) projectForRequestID(c *gin.Context, projectID string) (agent.Project, error) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		return agent.Project{}, errAgentForbidden
	}
	project, err := a.context.GetProject(strings.TrimSpace(projectID))
	if err != nil {
		return agent.Project{}, err
	}
	if !contextRecordOwnedByOrganization(project.OrganizationID, organizationID) {
		return agent.Project{}, errAgentForbidden
	}
	return project, nil
}

func (a *agentAPI) collabActor(c *gin.Context) string {
	if user, ok := c.Get("agent.user"); ok {
		if actor, ok := user.(agent.User); ok && actor.ID != "" {
			return actor.ID
		}
	}
	if value := strings.TrimSpace(c.GetHeader("X-Ollama-User")); value != "" {
		return value
	}
	return "local"
}

func (a *agentAPI) collabSnapshot(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, a.runtime.Collaboration().Snapshot(c.Param("project_id")))
}

func (a *agentAPI) collabComment(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Body string `json:"body"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	comment, err := a.runtime.Collaboration().AddComment(c.Param("project_id"), a.collabActor(c), request.Body)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (a *agentAPI) collabPresence(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	presence, err := a.runtime.Collaboration().SetPresence(c.Param("project_id"), a.collabActor(c), request.Status)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, presence)
}

func (a *agentAPI) collabStream(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusNotImplemented)
		return
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		payload, _ := json.Marshal(a.runtime.Collaboration().Snapshot(c.Param("project_id")))
		_, _ = fmt.Fprintf(c.Writer, "event: collaboration\ndata: %s\n\n", payload)
		flusher.Flush()
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *agentAPI) addMemory(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var memory agent.Memory
	if err := decodeJSON(c, &memory); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	memory.ProjectID = c.Param("id")
	created, err := a.context.AddMemoryContext(c.Request.Context(), memory)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (a *agentAPI) searchMemories(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	memories, err := a.context.SearchMemoriesContext(c.Request.Context(), c.Param("id"), c.Query("q"), 20)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_id": c.Param("id"), "memories": memories})
}

// askProjectDocuments answers a question strictly from the project's indexed
// documents (RAG, G1): it retrieves the most relevant sources above a relevance
// threshold and returns a cited grounded-context block plus the citations. When
// nothing relevant is indexed it returns grounded=false and a context that tells
// the model to say it did not find the information, so the UI never fabricates.
func (a *agentAPI) askProjectDocuments(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("q (consulta) é obrigatório"))
		return
	}
	const (
		askLimit    = 8
		askMinScore = 0.2 // relevance gate: unrelated queries return no sources
	)
	prompt, citations, err := a.context.GroundedAnswerContext(c.Request.Context(), c.Param("id"), query, askLimit, askMinScore)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id": c.Param("id"),
		"query":      query,
		"context":    prompt,
		"citations":  citations,
		"grounded":   len(citations) > 0,
	})
}

func (a *agentAPI) missions(c *gin.Context) {
	missions, err := a.scopedRuntime(c).ListMissions()
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	if organizationID := agentOrganizationID(c); organizationID != "" {
		filtered := missions[:0]
		for _, mission := range missions {
			if mission.OrganizationID == organizationID {
				filtered = append(filtered, mission)
			}
		}
		missions = filtered
	}
	c.JSON(http.StatusOK, gin.H{"missions": missions})
}

func (a *agentAPI) notifications(c *gin.Context) {
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeAgentError(c, http.StatusBadRequest, errors.New("notification limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}
	notifications, err := a.scopedRuntime(c).ListNotifications(limit)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (a *agentAPI) notificationStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusNotImplemented)
		return
	}
	lastID := ""
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		notifications, err := a.scopedRuntime(c).ListNotifications(20)
		if err != nil {
			return
		}
		currentID := ""
		if len(notifications) > 0 {
			currentID = notifications[0].ID
		}
		if currentID != lastID {
			payload, _ := json.Marshal(notifications)
			_, _ = fmt.Fprintf(c.Writer, "event: notifications\ndata: %s\n\n", payload)
			flusher.Flush()
			lastID = currentID
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *agentAPI) creations(c *gin.Context) {
	missions, err := a.scopedRuntime(c).ListMissions()
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	creations := make([]gin.H, 0)
	for _, mission := range missions {
		for _, artifact := range mission.Artifacts {
			if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.MissionID) != mission.ID {
				continue
			}
			category, label := creationCategory(artifact)
			creations = append(creations, gin.H{
				"id":               artifact.ID,
				"name":             artifact.Name,
				"category":         category,
				"categoryLabel":    label,
				"missionId":        mission.ID,
				"missionObjective": mission.Objective,
				"createdAt":        artifact.CreatedAt,
				"previewUrl":       "/api/agent/v1/missions/" + url.PathEscape(mission.ID) + "/artifacts/" + url.PathEscape(artifact.ID),
				"artifact":         artifact,
			})
		}
	}
	sort.SliceStable(creations, func(i, j int) bool {
		left, _ := creations[i]["createdAt"].(time.Time)
		right, _ := creations[j]["createdAt"].(time.Time)
		return left.After(right)
	})
	c.JSON(http.StatusOK, gin.H{"creations": creations, "status": "PASS", "reason": "Artefatos reais das missões"})
}

func creationCategory(artifact agent.ArtifactManifest) (string, string) {
	value := strings.ToLower(artifact.Name + " " + artifact.MediaType)
	switch {
	case strings.Contains(value, "game"), strings.Contains(value, "jogo"):
		return "games", "Jogos"
	case strings.Contains(value, "mobile"), strings.Contains(value, "android"), strings.Contains(value, "ios"):
		return "mobile", "Aplicativos móveis"
	default:
		return "sites", "Sites"
	}
}

func (a *agentAPI) createMission(c *gin.Context) {
	var request agent.CreateMissionRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	request.ActorID = agentActorID(c)
	if value, ok := c.Get("agent.organization"); ok {
		if organization, ok := value.(agent.Organization); ok {
			request.OrganizationID = organization.ID
		}
	} else if !a.authRequired {
		if supplied := strings.TrimSpace(request.OrganizationID); supplied != "" && supplied != agent.LocalOrganizationID {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		request.OrganizationID = agent.LocalOrganizationID
	}
	if strings.TrimSpace(request.ProjectID) != "" {
		project, err := a.projectForRequestID(c, request.ProjectID)
		if err != nil {
			writeAgentError(c, statusForAgentError(err), err)
			return
		}
		if strings.TrimSpace(request.Workspace) == "" {
			request.Workspace = project.Root
		} else if filepath.Clean(request.Workspace) != filepath.Clean(project.Root) {
			writeAgentError(c, http.StatusBadRequest, errors.New("mission workspace must match the selected project"))
			return
		}
	}
	mission, err := a.scopedRuntime(c).CreateMission(c.Request.Context(), request)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, mission)
}

func (a *agentAPI) getMission(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, mission)
}

func (a *agentAPI) events(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	events, err := a.scopedRuntime(c).Events(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mission_id": c.Param("id"), "events": events})
}

func (a *agentAPI) eventStream(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusNotImplemented)
		return
	}
	lastCount := 0
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		events, err := a.scopedRuntime(c).Events(c.Param("id"))
		if err != nil {
			return
		}
		if len(events) > lastCount {
			payload, _ := json.Marshal(events[lastCount:])
			_, _ = fmt.Fprintf(c.Writer, "event: mission\ndata: %s\n\n", payload)
			flusher.Flush()
			lastCount = len(events)
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *agentAPI) traces(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mission_id": c.Param("id"), "spans": a.runtime.TracesForOrganization(agentOrganizationID(c), "tr_"+c.Param("id"))})
}

func (a *agentAPI) allTraces(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"spans": a.runtime.TracesForOrganization(agentOrganizationID(c), c.Query("trace_id"))})
}

func (a *agentAPI) createOrchestration(c *gin.Context) {
	var request struct {
		Objective string            `json:"objective"`
		Workspace string            `json:"workspace,omitempty"`
		ProjectID string            `json:"project_id,omitempty"`
		Roles     []agent.AgentRole `json:"roles,omitempty"`
		Budget    agent.AgentBudget `json:"budget,omitempty"`
		AutoRun   bool              `json:"auto_run,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	organizationID := agentOrganizationID(c)
	job, err := a.runtime.Orchestrator().PlanForOrganization(organizationID, request.Objective, request.Workspace, request.ProjectID, request.Roles, request.Budget)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if request.AutoRun {
		go func(id, organizationID string) {
			if _, err := a.runtime.Orchestrator().RunForOrganization(context.Background(), id, organizationID); err != nil {
				slog.Error("agent orchestration autorun failed", "job_id", id, "organization_id", organizationID, "error", err)
			}
		}(job.ID, organizationID)
		c.JSON(http.StatusAccepted, job)
		return
	}
	c.JSON(http.StatusCreated, job)
}

func (a *agentAPI) getOrchestration(c *gin.Context) {
	job, err := a.runtime.Orchestrator().GetForOrganization(c.Param("id"), agentOrganizationID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (a *agentAPI) runOrchestration(c *gin.Context) {
	organizationID := agentOrganizationID(c)
	job, err := a.runtime.Orchestrator().GetForOrganization(c.Param("id"), organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	go func(id, organizationID string) {
		if _, err := a.runtime.Orchestrator().RunForOrganization(context.Background(), id, organizationID); err != nil {
			slog.Error("agent orchestration run failed", "job_id", id, "organization_id", organizationID, "error", err)
		}
	}(job.ID, organizationID)
	c.JSON(http.StatusAccepted, gin.H{"id": job.ID, "state": agent.OrchestrationRunning})
}

func (a *agentAPI) cancelOrchestration(c *gin.Context) {
	job, err := a.runtime.Orchestrator().CancelForOrganization(c.Param("id"), agentOrganizationID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (a *agentAPI) research(c *gin.Context) {
	var request agent.ResearchRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	report, err := a.runtime.Research().Research(c.Request.Context(), request)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	c.JSON(http.StatusOK, report)
}

func (a *agentAPI) actorIdentity(c *gin.Context) (string, string) {
	return agentActorID(c), agentOrganizationID(c)
}

func (a *agentAPI) devices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"devices": a.runtime.Devices().ListForOrganization(agentOrganizationID(c))})
}

func (a *agentAPI) startDevicePairing(c *gin.Context) {
	userID, organizationID := a.actorIdentity(c)
	code, pairing, err := a.runtime.Devices().StartPairing(userID, organizationID, 5*time.Minute)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"pairing_code": code, "expires_at": pairing.ExpiresAt})
}

func (a *agentAPI) completeDevicePairing(c *gin.Context) {
	var request struct {
		Code         string                   `json:"pairing_code"`
		Name         string                   `json:"name"`
		Platform     string                   `json:"platform"`
		Capabilities []agent.DeviceCapability `json:"capabilities"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	userID, organizationID := a.actorIdentity(c)
	device, token, err := a.runtime.Devices().CompletePairing(request.Code, request.Name, request.Platform, userID, organizationID, request.Capabilities)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"device": device, "device_token": token})
}

func (a *agentAPI) deviceHeartbeat(c *gin.Context) {
	var request struct {
		Capabilities []agent.DeviceCapability `json:"capabilities,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Device-Token"))
	}
	device, err := a.runtime.Devices().HeartbeatForOrganization(c.Param("id"), token, request.Capabilities, agentOrganizationID(c))
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, agent.ErrDeviceForbidden) {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, device)
}

func (a *agentAPI) revokeDevice(c *gin.Context) {
	device, err := a.runtime.Devices().RevokeForOrganization(c.Param("id"), agentOrganizationID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, device)
}

func (a *agentAPI) ingestProject(c *gin.Context) {
	var request agent.DocumentIngestRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	request.ProjectID = c.Param("id")
	project, err := a.projectForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	request.Workspace = project.Root
	memories, err := a.runtime.Ingestion().Ingest(c.Request.Context(), request)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project_id": request.ProjectID, "memories": memories, "count": len(memories)})
}

func (a *agentAPI) mediaWorkspace(c *gin.Context, missionID string) (agent.Mission, string, *os.Root, error) {
	mission, err := a.missionByID(c, missionID)
	if err != nil {
		return agent.Mission{}, "", nil, err
	}
	mission, root, err := a.runtime.OpenMissionWorkspaceRoot(mission.ID)
	if err != nil {
		return agent.Mission{}, "", nil, err
	}
	if strings.TrimSpace(mission.Workspace) == "" {
		_ = root.Close()
		return agent.Mission{}, "", nil, errors.New("mission workspace is required")
	}
	return mission, mission.Workspace, root, nil
}

func (a *agentAPI) mediaImage(c *gin.Context) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	var request struct {
		MissionID string `json:"mission_id"`
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime.Media() == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "media provider is not configured"})
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	result, err := a.runtime.Media().GenerateImage(c.Request.Context(), workspace, request.Prompt, request.Model, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaVideo(c *gin.Context) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	var request struct {
		MissionID string `json:"mission_id"`
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime.Media() == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "media provider is not configured"})
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	result, err := a.runtime.Media().GenerateVideo(c.Request.Context(), workspace, request.Prompt, request.Model, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaSpeech(c *gin.Context) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	var request struct {
		MissionID string `json:"mission_id"`
		Text      string `json:"text"`
		Voice     string `json:"voice"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime.Media() == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "media provider is not configured"})
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	result, err := a.runtime.Media().GenerateSpeech(c.Request.Context(), workspace, request.Text, request.Voice, request.Model, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaTranscribe(c *gin.Context) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	var request struct {
		MissionID string `json:"mission_id"`
		InputPath string `json:"input_path"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime.Media() == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "media provider is not configured"})
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	inputPath, err := containedPath(workspace, request.InputPath)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	result, err := a.runtime.Media().Transcribe(c.Request.Context(), workspace, inputPath, request.Model, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaVision(c *gin.Context) { //nolint:unused // compatibility/security surface retained for future adapter wiring
	var request struct {
		MissionID string `json:"mission_id"`
		InputPath string `json:"input_path"`
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if a.runtime.Media() == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "media provider is not configured"})
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	inputPath, err := containedPath(workspace, request.InputPath)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	result, err := a.runtime.Media().AnalyzeImage(c.Request.Context(), workspace, inputPath, request.Prompt, request.Model, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaOCR(c *gin.Context) {
	var request struct {
		MissionID string `json:"mission_id"`
		InputPath string `json:"input_path"`
		Language  string `json:"language"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	inputPath, err := containedPath(workspace, request.InputPath)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	result, err := agent.OCRLocal(c.Request.Context(), workspace, inputPath, request.Language, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) mediaTone(c *gin.Context) {
	var request struct {
		MissionID  string  `json:"mission_id"`
		Frequency  float64 `json:"frequency"`
		DurationMS int     `json:"duration_ms"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	mission, workspace, workspaceRoot, err := a.mediaWorkspace(c, request.MissionID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer workspaceRoot.Close()
	result, err := agent.GenerateTone(workspace, request.Frequency, time.Duration(request.DurationMS)*time.Millisecond, workspaceRoot)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	result.Artifact.MissionID = mission.ID
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) builders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"projects": a.runtime.Builder().ListForOrganization(companyOrganizationID(c))})
}

func (a *agentAPI) createBuilder(c *gin.Context) {
	var spec agent.BuilderSpec
	if err := decodeJSON(c, &spec); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	spec.OrganizationID = companyOrganizationID(c)
	project, err := a.runtime.Builder().Create(c.Request.Context(), spec)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, project)
}

func (a *agentAPI) previewBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	project, artifact, err := a.runtime.Builder().Preview(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project, "artifact": artifact})
}

func (a *agentAPI) updateBuilderVisual(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Components      []agent.VisualComponent `json:"components"`
		ExpectedVersion *int                    `json:"expected_version,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var project agent.BuilderProject
	var err error
	if request.ExpectedVersion != nil {
		project, err = a.runtime.Builder().ApplyVisualComponentsCAS(c.Request.Context(), c.Param("id"), *request.ExpectedVersion, request.Components)
	} else {
		project, err = a.runtime.Builder().ApplyVisualComponents(c.Request.Context(), c.Param("id"), request.Components)
	}
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (a *agentAPI) undoBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		ExpectedVersion *int `json:"expected_version,omitempty"`
	}
	if c.Request.Body != nil && c.Request.Body != http.NoBody {
		if err := decodeJSON(c, &request); err != nil && !errors.Is(err, io.EOF) {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
	}
	var project agent.BuilderProject
	var err error
	if request.ExpectedVersion != nil {
		project, err = a.runtime.Builder().UndoCAS(c.Request.Context(), c.Param("id"), *request.ExpectedVersion)
	} else {
		project, err = a.runtime.Builder().Undo(c.Request.Context(), c.Param("id"))
	}
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (a *agentAPI) redoBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		ExpectedVersion *int `json:"expected_version,omitempty"`
	}
	if c.Request.Body != nil && c.Request.Body != http.NoBody {
		if err := decodeJSON(c, &request); err != nil && !errors.Is(err, io.EOF) {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
	}
	var project agent.BuilderProject
	var err error
	if request.ExpectedVersion != nil {
		project, err = a.runtime.Builder().RedoCAS(c.Request.Context(), c.Param("id"), *request.ExpectedVersion)
	} else {
		project, err = a.runtime.Builder().Redo(c.Request.Context(), c.Param("id"))
	}
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (a *agentAPI) exportBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	project, archivePath, err := a.runtime.Builder().Export(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	checksum := project.ExportChecksum
	if checksum == "" {
		if data, err := os.ReadFile(archivePath); err == nil {
			h := sha256.Sum256(data)
			checksum = hex.EncodeToString(h[:])
		}
	}
	c.JSON(http.StatusAccepted, gin.H{
		"project":      project,
		"archive_path": archivePath,
		"checksum":     checksum,
		"sha256":       checksum,
		"download_url": "/api/agent/v1/builders/" + project.ID + "/download",
	})
}

func (a *agentAPI) getBuilder(c *gin.Context) {
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (a *agentAPI) downloadBuilder(c *gin.Context) {
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if project.ExportVersion == 0 || project.ExportVersion != project.Version || project.ExportChecksum == "" {
		writeAgentError(c, statusForAgentError(agent.ErrBuilderExportExpired), agent.ErrBuilderExportExpired)
		return
	}
	archivePath := filepath.Join(a.runtime.Builder().Root(), project.ID+".zip")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		writeAgentError(c, http.StatusNotFound, errors.New("project archive has not been exported yet"))
		return
	}
	h := sha256.Sum256(data)
	checksum := hex.EncodeToString(h[:])

	filename := strings.ReplaceAll(project.Name, " ", "_") + ".zip"
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/zip")
	c.Header("X-Checksum-SHA256", checksum)
	c.Data(http.StatusOK, "application/zip", data)
}

func (a *agentAPI) exportProfessionalBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	project, outputPath, err := a.runtime.Builder().ExportProfessional(c.Request.Context(), c.Param("id"), c.Param("format"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"project": project, "format": c.Param("format"), "output_path": outputPath})
}

func (a *agentAPI) publishBuilder(c *gin.Context) {
	if _, err := a.builderForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	project, publishedPath, err := a.runtime.Builder().PublishLocal(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"project": project, "published_path": publishedPath})
}

func (a *agentAPI) deployments(c *gin.Context) {
	manager := a.runtime.Deployments()
	if manager == nil {
		c.JSON(http.StatusOK, gin.H{"providers": []agent.DeployConfig{}})
		return
	}
	organizationID := agent.LocalOrganizationID
	if a.authRequired {
		organizationID = companyOrganizationID(c)
	}
	providers := manager.ListForOrganization(organizationID)
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (a *agentAPI) deployBuilder(c *gin.Context) {
	var request struct {
		Target     string `json:"target,omitempty"`
		ApprovalID string `json:"approval_id"`
		Nonce      string `json:"nonce"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	manager := a.runtime.Deployments()
	if manager == nil {
		writeAgentError(c, http.StatusNotImplemented, errors.New("no deployment providers are configured"))
		return
	}
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	manifest, err := agent.BuildDeploymentManifest(project.Root)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	approval, err := a.runtime.DeploymentApprovals().Consume(request.ApprovalID, companyOrganizationID(c), project.ID, c.Param("provider"), request.Target, manifest.SHA256, request.Nonce)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	deploymentRequest := agent.DeploymentRequest{Name: project.Name, Root: project.Root, Target: request.Target, ManifestSHA256: manifest.SHA256}
	var result agent.DeploymentResult
	if a.authRequired {
		result, err = manager.DeployForOrganization(c.Request.Context(), companyOrganizationID(c), c.Param("provider"), deploymentRequest)
	} else {
		result, err = manager.Deploy(c.Request.Context(), c.Param("provider"), deploymentRequest)
	}
	if err != nil {
		var deploymentErr *agent.DeploymentError
		if errors.As(err, &deploymentErr) && (result.Status == "partial" || result.Status == "unknown") {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":      agent.RedactDLP(err.Error()),
				"project":    project,
				"approval":   approval,
				"deployment": result,
			})
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"project": project, "approval": approval, "deployment": result})
}

func (a *agentAPI) requestDeploymentApproval(c *gin.Context) {
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	manager := a.runtime.Deployments()
	if manager == nil {
		writeAgentError(c, http.StatusNotImplemented, errors.New("no deployment providers are configured"))
		return
	}
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	configured := false
	organizationID := agent.LocalOrganizationID
	if a.authRequired {
		organizationID = companyOrganizationID(c)
	}
	providers := manager.ListForOrganization(organizationID)
	for _, config := range providers {
		if config.ID == provider {
			configured = true
			break
		}
	}
	if !configured {
		writeAgentError(c, http.StatusNotFound, errors.New("deployment provider is not configured"))
		return
	}
	var request struct {
		Target string `json:"target,omitempty"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	manifest, err := agent.BuildDeploymentManifest(project.Root)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	approval, err := a.runtime.DeploymentApprovals().Request(organizationID, project.ID, provider, request.Target, manifest.SHA256, agentActorID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"approval": approval, "manifest": manifest})
}

func (a *agentAPI) decideDeploymentApproval(c *gin.Context) {
	if !a.requireApprovalApprover(c) {
		return
	}
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason,omitempty"`
		Nonce    string `json:"nonce"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approved, err := approvalDecision(request.Decision)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approval, err := a.runtime.DeploymentApprovals().Decide(c.Param("approval_id"), companyOrganizationID(c), project.ID, c.Param("provider"), agentActorID(c), request.Reason, request.Nonce, approved)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, approval)
}

func (a *agentAPI) builderPreviewFile(c *gin.Context) {
	project, err := a.builderForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	relative := strings.TrimPrefix(c.Param("path"), "/")
	if relative == "" {
		relative = project.Entry
	}
	path, err := containedPath(project.Root, relative)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeAgentError(c, http.StatusNotFound, errors.New("preview file not found"))
		return
	}
	c.File(path)
}

func (a *agentAPI) builderForRequest(c *gin.Context) (agent.BuilderProject, error) {
	return a.runtime.Builder().GetForOrganization(c.Param("id"), companyOrganizationID(c))
}

func containedPath(root, requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return "", errors.New("input_path is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootAbs, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("input_path escapes mission workspace")
	}
	realRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	realCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	realRelative, err := filepath.Rel(realRoot, realCandidate)
	if err != nil || realRelative == ".." || strings.HasPrefix(realRelative, ".."+string(filepath.Separator)) || filepath.IsAbs(realRelative) {
		return "", errors.New("input_path resolves outside mission workspace")
	}
	return candidate, nil
}

func (a *agentAPI) artifact(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	manifest, path, err := a.scopedRuntime(c).Artifact(c.Param("id"), c.Param("artifact_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	defer os.Remove(path)
	defer agent.CloseArtifactSnapshot(path)
	c.Header("X-Artifact-SHA256", manifest.SHA256)
	c.FileAttachment(path, manifest.Name)
}

func (a *agentAPI) runMission(c *gin.Context) {
	id := c.Param("id")
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	job, err := a.scopedRuntime(c).EnqueueMission(id)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"mission_id": id, "state": agent.MissionRunning, "job": job})
}

func (a *agentAPI) cancelMission(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	mission, err = a.scopedRuntime(c).Cancel(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, mission)
}

func (a *agentAPI) decideApproval(c *gin.Context) {
	var request struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason,omitempty"`
		Nonce    string `json:"nonce"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approved, err := approvalDecision(request.Decision)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if !a.requireApprovalApprover(c) {
		return
	}
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	mission, err = a.scopedRuntime(c).DecideApprovalForActorCAS(c.Param("id"), c.Param("approval_id"), approved, request.Reason, agentActorID(c), agentOrganizationID(c), mission.Version, request.Nonce)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, mission)
}

func (a *agentAPI) requireApprovalApprover(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "approval requires organization owner or admin"})
		return false
	}
	return true
}

func (a *agentAPI) getMissionWorktree(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("mission id is required"))
		return
	}
	diff, err := a.scopedRuntime(c).GetMissionWorktreeDiff(c.Request.Context(), id)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, diff)
}

func (a *agentAPI) mergeMissionWorktree(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("mission id is required"))
		return
	}
	commitSHA, err := a.scopedRuntime(c).MergeMissionWorktree(c.Request.Context(), id)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":       "merged",
		"merge_commit": commitSHA,
	})
}

func decodeJSON(c *gin.Context, value any) error {
	if c.Request.Body == nil {
		return errors.New("request body is required")
	}
	const maxAgentJSONBody = 4 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAgentJSONBody)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body contains trailing JSON")
		}
		return err
	}
	return nil
}

func writeAgentError(c *gin.Context, status int, err error) {
	if errors.Is(err, agent.ErrDeploymentApprovalOrganization) {
		err = agent.ErrDeploymentApprovalNotFound
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "request failed"
	}
	c.JSON(status, gin.H{"error": agent.RedactDLP(message)})
}

func statusForAgentError(err error) int {
	if status := companyErrorStatus(err); status != 0 {
		return status
	}
	if errors.Is(err, errAgentForbidden) || errors.Is(err, agent.ErrPluginOrganizationScope) || errors.Is(err, agent.ErrBuilderForbidden) || errors.Is(err, agent.ErrOrchestrationForbidden) || errors.Is(err, agent.ErrDeviceForbidden) || errors.Is(err, agent.ErrQueueJobForbidden) {
		return http.StatusForbidden
	}
	if errors.Is(err, agent.ErrApprovalVersionConflict) || errors.Is(err, agent.ErrBuilderVersionConflict) || errors.Is(err, agent.ErrDeploymentApprovalNonce) || errors.Is(err, agent.ErrDeploymentApprovalConflict) || errors.Is(err, agent.ErrDeploymentApprovalExpired) {
		return http.StatusConflict
	}
	if errors.Is(err, agent.ErrBuilderExportExpired) {
		return http.StatusGone
	}
	if errors.Is(err, agent.ErrDeploymentApprovalNotFound) || errors.Is(err, agent.ErrDeploymentApprovalOrganization) || errors.Is(err, agent.ErrDeploymentProviderNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

func agentTenantContextKeyVersionFromEnv() (int, error) {
	raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION"))
	if raw == "" {
		return 1, nil
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version < 1 || version > 999999999 {
		return 0, errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY_VERSION must be an integer between 1 and 999999999")
	}
	return version, nil
}
