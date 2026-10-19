package server

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
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
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/grok"
	"github.com/ollama/ollama/internal/multillm"
	"github.com/ollama/ollama/version"
)

// PluginTrustedKeysEnv autoriza chaves de assinatura de plugin no formato
// `key_id:base64url_ed25519_public_key`, separadas por vírgula. Sem essa
// configuração o conjunto fica VAZIO e nenhuma promoção de confiança é
// possível: falha fechada, como o resto da política de capacidades.
const PluginTrustedKeysEnv = "OLLAMA_AGENT_PLUGIN_TRUSTED_KEYS"

func pluginCapabilityPolicy() (agent.CapabilityPolicy, error) {
	policy := agent.DefaultCapabilityPolicy()
	raw := strings.TrimSpace(os.Getenv(PluginTrustedKeysEnv))
	if raw == "" {
		return policy, nil
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		keyID, encoded, ok := strings.Cut(entry, ":")
		if !ok || strings.TrimSpace(keyID) == "" {
			return policy, fmt.Errorf("%s: entry %q must be key_id:base64_key", PluginTrustedKeysEnv, entry)
		}
		key, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil || len(key) != ed25519.PublicKeySize {
			return policy, fmt.Errorf("%s: key %q is not a base64url ed25519 public key", PluginTrustedKeysEnv, keyID)
		}
		policy = policy.WithTrustedSkillKey(strings.TrimSpace(keyID), ed25519.PublicKey(key))
	}
	return policy, nil
}

// listPlugins devolve as instalações da organização autenticada, sem qualquer
// valor de credencial. Escopo concedido aparece separado do escopo PEDIDO: a
// interface precisa poder mostrar "instalado mas ainda não autorizado".
func (a *agentAPI) listPlugins(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"plugins":         a.plugins.ListForOrganization(organizationID),
	})
}

// installPlugin registra um plugin na organização autenticada. Escopo
// desconhecido da política é recusado; nada é concedido nesta chamada.
func (a *agentAPI) installPlugin(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	var manifest agent.PluginManifest
	if err := decodeJSON(c, &manifest); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	policy, err := pluginCapabilityPolicy()
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	installation, err := a.plugins.InstallForOrganization(organizationID, manifest, policy)
	if err != nil {
		writeAgentError(c, statusForPluginError(err), err)
		return
	}
	c.JSON(http.StatusCreated, installation)
}

// installPluginBundle instala o plugin a partir de um PACOTE assinado. O
// caminho só é aceito dentro do diretório permitido pelo operador
// (OLLAMA_AGENT_PLUGIN_BUNDLE_DIR); sem essa configuração a rota falha fechada,
// para que a API nunca vire leitura arbitrária de arquivo do host.
func (a *agentAPI) installPluginBundle(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	var request struct {
		Manifest   agent.PluginManifest `json:"manifest"`
		BundlePath string               `json:"bundle_path"`
		Signature  agent.SignedArtifact `json:"signature"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	bundlePath, err := agent.ResolvePluginBundlePath(strings.TrimSpace(os.Getenv(agent.PluginBundleDirEnv)), request.BundlePath)
	if err != nil {
		writeAgentError(c, http.StatusForbidden, err)
		return
	}
	policy, err := pluginCapabilityPolicy()
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	installation, err := a.plugins.InstallBundleForOrganization(organizationID, request.Manifest, bundlePath, request.Signature, policy)
	if err != nil {
		writeAgentError(c, statusForPluginError(err), err)
		return
	}
	c.JSON(http.StatusCreated, installation)
}

// rollbackPlugin desfaz a última atualização registrada do plugin.
func (a *agentAPI) rollbackPlugin(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	installation, err := a.plugins.RollbackForOrganization(organizationID, c.Param("plugin_id"))
	if err != nil {
		writeAgentError(c, statusForPluginError(err), err)
		return
	}
	c.JSON(http.StatusOK, installation)
}

// promotePlugin verifica a assinatura contra as chaves autorizadas e só então
// concede os escopos pedidos pelo manifesto.
// pluginStatuses devolve o estado canônico de cada plugin instalado (V2.1 §11.3):
// a interface só pode oferecer "pronto para usar" quando houver execução real.
func (a *agentAPI) pluginStatuses(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	policy, err := pluginCapabilityPolicy()
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	statuses, err := agent.NewPluginRuntime(a.plugins, policy).StatusForOrganization(organizationID)
	if err != nil {
		writeAgentError(c, statusForPluginError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"plugins":         statuses,
	})
}

func (a *agentAPI) promotePlugin(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	policy, err := pluginCapabilityPolicy()
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	installation, err := a.plugins.PromoteTrustedForOrganization(organizationID, c.Param("plugin_id"), policy)
	if err != nil {
		writeAgentError(c, statusForPluginError(err), err)
		return
	}
	c.JSON(http.StatusOK, installation)
}

// statusForPluginError mapeia os erros do registro de plugins para HTTP sem
// transformar recusa de segurança em 500.
func statusForPluginError(err error) int {
	switch {
	case errors.Is(err, agent.ErrUnknownCapability), errors.Is(err, agent.ErrPluginManifestInvalid):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrPluginVersionConflict):
		return http.StatusConflict
	case errors.Is(err, agent.ErrPluginNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrPluginOrganizationScope), errors.Is(err, agent.ErrPluginKeyUnauthorized),
		errors.Is(err, agent.ErrPluginSignatureReq), errors.Is(err, agent.ErrPluginSignatureInvalid),
		errors.Is(err, agent.ErrPluginBundleUnavailable):
		return http.StatusForbidden
	case errors.Is(err, agent.ErrPluginRollbackDisabled):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

type agentAPI struct {
	runtime      *agent.Runtime
	context      *agent.ContextStore
	plugins      *agent.PluginRegistry
	auth         *agent.AuthStore
	oauthClients *agent.OAuthClientStore
	authRequired bool

	// FASE 28.7: o store dos builds iniciados por `build.start`.
	//
	// E' por instancia e com lazy init: um singleton de package seria estado partilhado entre
	// testes e esconderia o facto de dois servidores nao partilharem builds. O zero value de
	// `sync.Once` funciona, por isso `&agentAPI{}` nos testes continua valido.
	builds       *agent.BuildStore
	buildsOnce   sync.Once
	push         *agent.PushService
	grok         *grok.Client
	samlMu       sync.Mutex
	samlServices map[string]*agent.SAMLService
	// chatHandlers é o caminho de inferência reaproveitado pela rota agentic de
	// chat (estágio 5): a injeção do contexto acontece antes dele, sem duplicar
	// a inferência.
	chatHandlers []gin.HandlerFunc
	// activeStreams bounds concurrent long-lived SSE/WS connections (SEC-13).
	activeStreams atomic.Int64
	// E-449 (ADV-65): o MOTOR DE AGENDAMENTO ligado à superfície. **Ate' aqui ele nao era construido fora dos testes**
	// - *o motor existia, era testado, e nao estava ligado a nada*. Nulo significa "sem superficie ligada", e as rotas
	// **falham fechado** em vez de devolver lista vazia (*que pareceria "nao ha servicos" em vez de "nao ha motor"*).
	scheduling *agent.SchedulingStore

	// FASE 30: o ledger de personas do Agent Center. Nulo significa "sem superficie ligada", e as rotas
	// falham fechado em vez de devolver lista vazia -- *que pareceria "nao ha agentes" em vez de "nao ha
	// armazenamento"*. Mesmo `storeRoot` do registro de plugins: nao se cria uma segunda raiz de dados.
	personas   *agent.PersonaLedger
	quarantine *agent.QuarantineStore
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
	if runtime != nil {
		runtime.SetAuthStore(auth)
	}
	oauthClientRoot := ""
	if runtime != nil {
		oauthClientRoot = runtime.DataRoot()
	}
	oauthClients, err := agent.NewOAuthClientStore(oauthClientRoot)
	if err != nil {
		return nil, err
	}
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
	// Registro de plugins (estágio 8): manifesto versionado, atualização
	// otimista e rollback. A confiança continua sendo decidida pelo servidor,
	// pela chave autorizada em OLLAMA_AGENT_PLUGIN_TRUSTED_KEYS.
	personaLedger, err := agent.NewPersonaLedger(storeRoot)
	if err != nil {
		return nil, err
	}
	quarantineStore, err := agent.NewQuarantineStore(storeRoot)
	if err != nil {
		return nil, err
	}
	pluginRegistry, err := agent.NewPluginRegistry(storeRoot)
	if err != nil {
		return nil, err
	}
	// E-449 (ADV-65): o motor de agendamento passa a ser CONSTRUIDO em producao. *Em E-441 a busca ampla mostrou que as
	// duas ocorrencias de `NewSchedulingStore` fora dos testes eram o **comentario** e a **definicao**.* Sob `storeRoot`,
	// como os outros stores - *um diretorio por preocupacao, e persistencia herdada.*
	return &agentAPI{runtime: runtime, context: runtime.Context(), plugins: pluginRegistry, personas: personaLedger, quarantine: quarantineStore, auth: auth, oauthClients: oauthClients, authRequired: required, push: runtime.Push(), grok: grokClient, samlServices: map[string]*agent.SAMLService{}, scheduling: agent.NewSchedulingStore(filepath.Join(storeRoot, "scheduling"))}, nil
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
	// E-667: o MESMO cliente de chat do planejador e' injetado como executor de INFERENCIA POR PASSO. *Criar um
	// segundo cliente aqui contornaria o controle de egresso (`newServerEgressClient`, auditado) e daria dois
	// caminhos para falar com modelo - a classe de defeito que o E-658 varreu do roteador.* Sem `OLLAMA_AGENT_MODEL`
	// nao ha cliente, e o kind `model.infer` **nao se registra**: o passo e' recusado no planejamento.
	var inferenceClient agent.InferenceChatClient
	if model := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MODEL")); model != "" {
		inferenceClient = api.NewClient(envconfig.ConnectableHost(), newServerEgressClient("server.agent.ollama_local", true))
		planner = &agent.OllamaPlanner{
			Client: inferenceClient,
			Model:  model,
			Price:  agent.LocalModelPrice(),
		}
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Store: store, Context: contextStore, Company: companyStore, Planner: planner, InferenceChatClient: inferenceClient, WorkspaceRoot: workspaceRoot, DataRoot: storeRoot, Connectors: connectors, MCP: mcp, RemoteMCP: remoteMCP, Media: media, RedisQueue: redisQueue, Telemetry: telemetry, Push: push, Deployments: deployments})
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
	// Chat da superfície Hades: mesmo caminho de inferência do /api/chat, com o
	// contexto canônico injetado pelo middleware (estágio 5). O /api/chat de
	// sempre permanece intocado para CLI e clientes compatíveis.
	if len(a.chatHandlers) > 0 {
		chatRoute := append([]gin.HandlerFunc{withAgentChatContext(a, a.context.Root())}, a.chatHandlers...)
		group.POST("/chat", chatRoute...)
	}
	// Comandos slash (§6/§11.1): descoberta e execução pelo registro canônico.
	group.GET("/commands", a.commandDiscovery)
	group.POST("/commands", a.runCommand)
	group.GET("/grok/status", a.grokStatus)
	group.GET("/config/safe", a.safeConfig)
	group.GET("/diagnostics", a.diagnostics)
	group.GET("/backup", a.downloadBackup)
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
	group.POST("/projects/import/attachments", a.importMissionAttachments)
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
	group.POST("/connectors/:id/oauth/start", a.connectorOAuthStart)
	group.GET("/connectors/:id/oauth/callback", a.connectorOAuthCallback)
	group.DELETE("/connectors/:id", a.removeConnector)
	group.GET("/oauth-clients", a.listOAuthClients)
	group.POST("/oauth-clients/:provider", a.saveOAuthClient)
	group.GET("/browser/environment", a.browserEnvironment)
	group.POST("/browser/environment/setup", a.browserEnvironmentSetup)
	// E-449 (ADV-65): a SUPERFICIE do motor de agendamento. *Um motor sem superficie nao e' capacidade: e' biblioteca.*
	// Para OFERECER horario usa-se `/scheduling/slots`, que chama `OpenSlots` - **nao** `Availability` (ADV-60).
	group.GET("/scheduling/services", a.schedulingServices)
	group.GET("/scheduling/slots", a.schedulingSlots)
	group.POST("/scheduling/book", a.schedulingBook)
	// E-456: a JORNADA DO CLIENTE COMPLETA. *Um cliente que agenda e nao consegue cancelar nem remarcar nao tem um
	// produto: tem uma demonstracao.* E `reschedule/slots` responde ANTES de o cliente escolher.
	group.GET("/scheduling/appointments", a.schedulingAppointments)
	group.POST("/scheduling/cancel", a.schedulingCancel)
	group.GET("/scheduling/reschedule/slots", a.schedulingRescheduleSlots)
	group.POST("/scheduling/reschedule", a.schedulingReschedule)
	// E-457: FILA DE ESPERA e NO-SHOW - *o motor tinha as quatro funcoes e nenhuma porta.*
	group.GET("/scheduling/waitlist", a.schedulingWaitlist)
	group.POST("/scheduling/waitlist", a.schedulingWaitlistAdd)
	group.POST("/scheduling/waitlist/offer", a.schedulingWaitlistOffer)
	group.POST("/scheduling/waitlist/confirm", a.schedulingWaitlistConfirm)
	group.POST("/scheduling/no-show", a.schedulingNoShow)
	// E-479: os LEMBRETES - *quem deve ser avisado, e o registo de que foi. O motor ja' tinha as duas metades.*
	group.GET("/scheduling/reminders/due", a.schedulingDueReminders)
	group.POST("/scheduling/reminders/sent", a.schedulingMarkReminderSent)
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
	// E-654: §9 — a VERDADE de capacidade por modelo, na superficie (o registo existia e era invisivel).
	group.GET("/capabilities", a.capabilities)
	group.GET("/skills", a.skills)
	group.GET("/skills/active", a.activeSkills)
	group.POST("/skills/:id/activate", a.activateSkill)
	group.POST("/skills/:id/deactivate", a.deactivateSkill)
	group.POST("/skills", a.registerSkill)
	group.POST("/skills/:id/enable", a.enableSkill)
	group.POST("/skills/:id/disable", a.disableSkill)
	// REPOSTO NO E-650. Esta rota foi APAGADA POR ENGANO no E-642: usei a linha do `DELETE` como
	// ANCORA de edicao e substitui-a pelas rotas novas SEM A REPOR. A consequencia nao foi um erro
	// de compilacao (nada a referencia) mas uma FUNCIONALIDADE A MENOS: apagar skill deixou de
	// existir, e o cliente da UI continuava a chamar a rota.
	//
	// *Quem apanhou foi o CI* (`node scripts/verify-contracts.mjs`, dentro de "Preserve Class A+
	// surfaces"), a comparar o caminho do frontend com as rotas do backend. A minha bateria local
	// de 16 verificadores NAO o via, porque nenhum deles e' esse script - a mesma classe de falha
	// do E-636: *a bateria nao continha o gate que o CI continha.* O verificador de contrato passa
	// a fazer parte dela.
	group.DELETE("/skills/:id", a.removeSkill)
	// E-966: a rota de EXECUCAO. *O lifecycle ja' existia (13 handlers); executar nao.*
	group.POST("/plugins/:plugin_id/execute", a.executePlugin)
	// E-976: a rota do FORMATO ABERTO. *O formato existia e instalava; faltava a superficie.*
	group.POST("/plugins/format", a.installHadesPlugin)
	group.GET("/plugins/format", a.hadesPluginFormat)
	// E-642: HISTORICO de versoes de skill. As rotas de skills ja' existiam; o que faltava era
	// expor o arquivo que o E-629 passou a guardar e o restauro do E-631.
	//
	// PREFIXO PROPRIO DE PROPOSITO: `GET /skills/active` usa um LITERAL na segunda posicao, e um
	// `:id` no mesmo nivel arriscaria conflito na arvore de rotas do gin (o servidor pode recusar
	// arrancar). `/skill-versions/...` nao tem irmao literal - *e' melhor um nome menos bonito do
	// que uma colisao que so' aparece ao subir o servidor.*
	group.GET("/skill-versions/:id", a.skillVersions)
	group.POST("/skill-versions/:id/restore/:version", a.restoreSkillVersion)
	// E-649: comparar duas revisoes (§9 Paperclip). `from`/`to` sao versoes; uma delas pode ser a
	// ACTIVA, que nao esta' no arquivo (so' as superadas ficam la').
	group.GET("/skill-versions/:id/compare", a.compararVersoesDeSkill)
	// E-647: TRANSPORTE do servidor MCP — a memoria do Hades exposta pelo protocolo.
	group.POST("/mcp/rpc", a.mcpRPC)
	group.GET("/companies", a.companies)
	group.POST("/companies", a.createCompany)
	group.GET("/companies/:id", a.getCompany)
	group.PATCH("/companies/:id", a.updateCompany)
	group.GET("/companies/:id/report", a.companyReport)
	group.GET("/companies/:id/ledger", a.listLedgerEntries)
	group.POST("/companies/:id/ledger", a.recordLedgerEntry)
	group.GET("/companies/:id/ledger/summary", a.ledgerSummary)
	group.POST("/companies/:id/ledger/:entry_id/reconcile", a.reconcileLedgerEntry)
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
	group.POST("/companies/:id/campaigns/:campaign_id/schedule", a.scheduleCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/metrics", a.recordCompanyCampaignMetrics)
	group.GET("/companies/:id/campaigns/report", a.companyCampaignReport)
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
	group.GET("/companies/:id/marketplace/channels", a.marketplaceChannels)
	group.POST("/companies/:id/marketplace/sync", a.marketplaceSync)
	group.GET("/companies/:id/marketplace/mirror/:channel_id/:sku", a.marketplaceMirror)
	group.GET("/companies/:id/business-status", a.businessStatus)
	group.GET("/companies/:id/business-snapshots", a.businessSnapshots)
	group.POST("/companies/:id/business-snapshots", a.recordBusinessSnapshot)
	group.GET("/companies/:id/business-export", a.businessExport)
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
	group.POST("/schedules/:id/run", a.runScheduleNow)
	group.POST("/schedules/:id/reprocess", a.reprocessScheduleNow)
	group.POST("/schedules/:id/approvals", a.decideScheduleApproval)
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
	// Governança de memória (estágio 15): exportar, apagar e aplicar retenção.
	// Todas passam por `projectForRequest`, então herdam o isolamento por
	// organização do projeto.
	group.GET("/projects/:id/memories/export", a.exportProjectMemories)
	group.GET("/projects/:id/memories/summary", a.memorySummary)
	group.DELETE("/projects/:id/memories/:memory_id", a.deleteProjectMemory)
	group.PATCH("/projects/:id/memories/:memory_id", a.updateProjectMemory)
	group.GET("/projects/:id/memories/:memory_id/history", a.memoryHistory)
	group.POST("/projects/:id/memories/prune", a.pruneProjectMemories)
	// Re-embed: o gatilho de operador para trazer ao espaço vetorial atual o conhecimento que ficou para trás
	// quando o modelo de embedding mudou.
	group.POST("/projects/:id/memories/reembed", a.reembedProjectMemories)
	// Re-embed ASSÍNCRONO: estimativa antes, job com progresso e cancelamento pelo operador.
	group.GET("/projects/:id/memories/reembed/estimate", a.reembedEstimate)
	group.POST("/projects/:id/memories/reembed/jobs", a.startReembedJob)
	group.GET("/projects/:id/memories/reembed/jobs/:job_id", a.reembedJobStatus)
	group.POST("/projects/:id/memories/reembed/jobs/:job_id/cancel", a.cancelReembedJob)
	// Política de retenção por organização (estágio 15). A aplicação é
	// EXPLÍCITA: não existe laço de fundo fingindo automação.
	group.GET("/retention", a.retentionPolicy)
	group.PUT("/retention", a.setRetentionPolicy)
	group.POST("/retention/apply", a.applyRetentionPolicy)
	group.GET("/projects/:id/ask", a.askProjectDocuments)
	group.GET("/collab/:project_id", a.collabSnapshot)
	group.GET("/collab/:project_id/stream", a.collabStream)
	group.POST("/collab/:project_id/comments", a.collabComment)
	group.POST("/collab/:project_id/presence", a.collabPresence)
	// Complemento 48 / FASE 2 — a SALA e os MEMBROS passam a ser alcançáveis.
	// *Sem estas rotas, o domínio (sala, membros humanos e agentes) existia e **ninguém** o podia usar* - e a
	// superfície de UI ficaria a chamar um caminho que não existe. Mesma forma dos vizinhos: autorização de
	// projeto primeiro (`projectForRequestID`), depois o store, e erro sempre por `writeAgentError`.
	group.GET("/collab/:project_id/members", a.collabMembers)
	group.POST("/collab/:project_id/members", a.collabJoinProject)
	group.GET("/collab/:project_id/rooms", a.collabRooms)
	group.POST("/collab/:project_id/rooms", a.collabCreateRoom)
	group.GET("/collab/:project_id/rooms/:room_id/members", a.collabRoomMembers)
	group.POST("/collab/:project_id/rooms/:room_id/members", a.collabJoinRoom)
	group.POST("/missions", a.createMission)
	group.GET("/missions", a.missions)
	// E-625: relatorio de desempenho das missoes visiveis para o chamador (Paperclip §10).
	group.GET("/performance", a.performance)
	// FASE 21 + FASE 24: a telemetria AGREGADA (latencia/p50/p95/p99, custo previsto vs
	// realizado) dos eventos REAIS. Nome distinto de `/metrics` de proposito: a rota
	// existente devolve CONTADORES (`runtime.Metrics()`), nao agregados. *Duas rotas com
	// o mesmo nome e conteudos diferentes seriam duas verdades sobre a mesma pergunta.*
	group.GET("/metrics/telemetry", a.agentTelemetry)
	// FASE 23 do adendo V3: catalogo de presets de MISSAO. Fica em `/presets` e nao em
	// `/missions/presets` de proposito: `/missions/:id` ja' existe, e um literal ao lado de um
	// parametro na mesma posicao e' ambiguidade de roteador, nao funcionalidade.
	group.GET("/presets", a.missionPresets)
	// FASE 19 + FASE 24: a decisao de roteamento com as ALTERNATIVAS. *Escolher bem quando ha'
	// um vencedor claro e' facil; escolher quando nao ha' e' que precisa de ser explicado.*
	group.GET("/routing", a.routing)
	// FASE 28.7 do adendo V3: preflight de toolchain por alvo de build. Todas LEITURA -- nenhuma
	// instala, altera ou executa trabalho de utilizador. *Um preflight que muda o sistema nao e'
	// preflight: e' instalacao disfarcada.*
	// FASE 28.7: project.detect -- a porta do ProjectDetector. Le' HADES_RAIZ_PROJECTO.
	group.POST("/project/detect", a.detectarProjecto)
	group.GET("/toolchain/targets", a.toolchainTargets)
	group.GET("/toolchain/doctor", a.toolchainDoctor)
	group.GET("/toolchain/plan/:alvo", a.toolchainPlan)
	// FASE 28.6-A / 28.7: instalador modular. Superficie de LEITURA: planeia, nao instala.
	// FASE 2: superficie de worktrees (E-1226..E-1231). So' leitura: remover exige decisao
	// explicita sobre trabalho nao integrado, nunca um efeito lateral de um GET.
	group.GET("/worktrees", a.worktreeInventario)
	group.GET("/worktrees/diagnose", a.worktreeDiagnostico)
	// FASE 2 (E-1236): a UNICA rota mutante de worktree. Aceita suspender/retomar/arquivar --
	// transicoes REVERSIVEIS. `removido` NAO e' aceite aqui: *remover exige uma decisao sobre
	// trabalho nao integrado, e essa decisao nao cabe num clique que o painel nao sabe explicar.*
	group.POST("/worktrees/state", a.worktreeMudarEstado)

	// FASE 3 (E-1240): comparacao objectiva de solucoes. POST porque o corpo pode ser grande;
	// NAO muta nada e diz isso na resposta (`somente_leitura`).
	group.POST("/solutions/compare", a.comparacaoDeSolucoes)
	// FASE 28.6/28.7: o plano de release do projection. Le' e sonda; nao instala nem constroi.
	group.GET("/build/plan", a.planoDeRelease)
	// FASE 28.7 do adendo V3: `artifact.checksum` e `artifact.verify`. A primeira MEDE, a segunda
	// COMPARA com uma referencia. Nenhuma escreve, assina ou instala.
	group.GET("/build/artifact/checksum", a.checksumDeArtefacto)
	group.GET("/build/artifact/verify", a.verificarArtefacto)
	// FASE 28.9 passo 7: `build.verify`. Verifica o LOTE de artefactos de um build contra os
	// SHA-256 que o build mediu. POST porque a lista nao cabe num URL sem limites arbitrarios.
	group.POST("/build/start", a.iniciarBuild)
	group.POST("/build/verify", a.verificarBuild)
	group.POST("/build/smoke", a.executarSmoke)
	// FASE 28.7 do adendo V3: o estado do build. O E-1306 recusou-se a inventar estas rotas
	// porque nao havia store; agora ha'. Nenhuma delas escreve, assina ou publica.
	group.GET("/build/status/:id", a.estadoDeBuild)
	group.GET("/build/logs/:id", a.saidaDoBuild)
	group.GET("/build/artifacts/:id", a.artefactosDoBuild)
	group.GET("/build/registos", a.registosDeBuild)
	group.POST("/build/cancel/:id", a.cancelarBuild)
	group.POST("/publish/rollback", a.planearRollback)
	group.POST("/publish/preflight/:canal", a.preflightDePublicacao)
	group.POST("/sbom/generate", a.gerarSBOM)
	// FASE 28.7: `sign.plan`. AVALIA se a identidade serve para o tipo pedido; nao assina, nao le' a
	// chave privada e nao valida cadeia criptografica.
	group.POST("/sign/plan", a.planearAssinatura)
	group.GET("/credentials/status", a.inventariarCredenciais)
	group.GET("/install/profiles", a.installProfiles)
	group.GET("/install/plan/:perfil", a.installPlan)
	// FASE 28.5 do adendo V3: o Signing Identity Center. Le METADADOS de um ficheiro fora do
	// repositorio (ao lado do broker de credenciais) e nunca devolve valor privado. Leitura
	// apenas: criar/importar/revogar identidade exige confirmacao humana no momento da accao.
	group.GET("/signing/identities", a.signingIdentities)
	group.GET("/signing/summary", a.signingIdentitySummary)
	// E-638: despertares coalescidos por agente, com enfileiramento do que precisa de execucao.
	group.POST("/wakeups", a.wakeups)
	group.GET("/notifications", a.notifications)
	group.GET("/notifications/stream", a.notificationStream)
	group.GET("/creations", a.creations)
	group.GET("/missions/:id", a.getMission)
	group.GET("/missions/:id/events", a.events)
	group.GET("/missions/:id/events/stream", a.eventStream)
	// Estado REAL do Companion para uma missao (complemento 48 / item 46). Reutiliza o
	// mesmo escopo das rotas acima: *nao inventa autorizacao nova.*
	group.GET("/missions/:id/companion", a.companion)
	group.GET("/missions/:id/traces", a.traces)
	group.GET("/missions/:id/artifacts/:artifact_id", a.artifact)
	// ESTAGIO 47: a superficie de VERSOES. *O dominio (historico/comparar/restaurar) existia
	// e testado em `internal/agent` e nao tinha rota nenhuma - a capacidade era inalcancavel.*
	// Leitura sem CAS; escrita (`restore`) passa pelo `missionVersionMatches`, como todas.
	group.GET("/missions/:id/artifacts/:artifact_id/versions", a.artifactVersions)
	group.POST("/missions/:id/artifacts/:artifact_id/restore", a.artifactRestore)
	group.POST("/missions/:id/run", a.runMission)
	group.POST("/missions/:id/cancel", a.cancelMission)
	group.DELETE("/missions/:id", a.deleteMission)
	group.POST("/missions/:id/approvals/:approval_id", a.decideApproval)
	group.GET("/missions/:id/worktree", a.getMissionWorktree)
	group.GET("/missions/:id/context", a.missionContext)
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
	group.GET("/supervisor/incidents", a.supervisorIncidents)
	group.GET("/supervisor/ticks", a.supervisorTicks)
	group.POST("/supervisor/incidents/:id/acknowledge", a.supervisorAcknowledgeIncident)
	group.POST("/supervisor/incidents/:id/resolve", a.supervisorResolveIncident)
	group.GET("/crm/contacts", a.listCRMContacts)
	group.POST("/crm/contacts", a.upsertCRMContact)
	group.GET("/crm/contacts/:id", a.getCRMContact)
	group.POST("/crm/contacts/:id/opportunities", a.createCRMOpportunity)
	group.POST("/crm/contacts/:id/consent", a.grantCRMContactConsent)
	group.POST("/crm/contacts/:id/consent/revoke", a.revokeCRMContactConsent)
	group.POST("/crm/opportunities/:id/stage", a.setCRMOpportunityStage)
	group.GET("/crm/pipeline", a.crmPipeline)
	group.GET("/telegram/status", a.telegramStatus)
	group.POST("/telegram/send", a.telegramSend)
	group.GET("/inbox/conversations", a.listInboxConversations)
	group.POST("/inbox/messages", a.appendInboxMessage)
	group.GET("/inbox/conversations/:id", a.getInboxConversation)
	group.POST("/inbox/conversations/:id/reply", a.replyInboxConversation)
	group.POST("/inbox/conversations/:id/assign", a.assignInboxConversation)
	group.POST("/inbox/conversations/:id/handoff", a.handoffInboxConversation)
	group.POST("/inbox/conversations/:id/close", a.closeInboxConversation)
	group.GET("/inbox/metrics", a.inboxMetrics)
	group.POST("/omnichannel/connectors", a.registerOmnichannelConnector)
	group.GET("/omnichannel/connectors", a.listOmnichannelConnectors)
	group.POST("/omnichannel/webhooks/:channel", a.ingestOmnichannelWebhook)
	group.GET("/omnichannel/events", a.listOmnichannelEvents)
	group.GET("/omnichannel/dlq", a.listOmnichannelDeadLetters)
	group.POST("/omnichannel/dlq/:id/retry", a.retryOmnichannelDeadLetter)
	group.GET("/service-agents", a.listServiceAgents)
	group.POST("/service-agents/:id/turn", a.serviceAgentTurn)
	group.GET("/service-agents/audit", a.serviceAgentAudit)
	group.GET("/service-agents/approvals", a.serviceAgentApprovals)
	group.POST("/service-agents/approvals/:id/decide", a.serviceAgentDecideApproval)
	group.PUT("/service-agents/approval-approvers", a.serviceAgentApprovalApprovers)
	group.POST("/commerce/products", a.upsertCommerceProduct)
	group.GET("/commerce/products", a.listCommerceProducts)
	group.POST("/commerce/stock", a.recordCommerceStock)
	group.GET("/commerce/stock/:sku", a.commerceStock)
	group.POST("/commerce/orders", a.createCommerceOrder)
	group.GET("/commerce/orders", a.listCommerceOrders)
	group.GET("/commerce/orders/:id", a.getCommerceOrder)
	group.POST("/commerce/orders/:id/payment", a.confirmCommercePayment)
	group.POST("/commerce/orders/:id/reconcile", a.reconcileCommerceOrder)
	group.POST("/commerce/orders/:id/fulfill", a.fulfillCommerceOrder)
	group.POST("/commerce/orders/:id/cancel", a.cancelCommerceOrder)
	group.GET("/commerce/summary", a.commerceSummary)
	group.POST("/finance/orders/:id/revenue", a.postOrderRevenue)
	group.POST("/finance/orders/:id/cost", a.postOrderCost)
	group.POST("/finance/orders/:id/reconcile", a.reconcileOrderWithStatement)
	group.GET("/finance/report", a.financeReport)
	group.POST("/business/certify", a.certifyBusiness)
	group.POST("/company/factory/launch", a.launchCompanyFactory)
	// Propostas comerciais e pos-venda (estagio 27).
	group.POST("/sales/proposals", a.createSalesProposal)
	group.GET("/sales/proposals", a.listSalesProposals)
	group.GET("/sales/proposals/:id", a.getSalesProposal)
	group.POST("/sales/proposals/:id/send", a.sendSalesProposal)
	group.POST("/sales/proposals/:id/decide", a.decideSalesProposal)
	group.POST("/sales/follow-ups", a.createSalesFollowUp)
	group.GET("/sales/follow-ups", a.listSalesFollowUps)
	group.POST("/sales/follow-ups/:id/complete", a.completeSalesFollowUp)
	group.GET("/company/factory/report/:id", a.companyFactoryReport)
	group.POST("/business/tick", a.businessSupervisorTick)
	group.GET("/business/actions", a.listBusinessActions)
	group.POST("/business/actions/:id/decide", a.decideBusinessAction)
	group.GET("/egress/logs", a.getEgressLogs)
	group.GET("/egress/status", a.getEgressStatus)
	group.GET("/plugins", a.listPlugins)
	group.GET("/plugins/status", a.pluginStatuses)
	group.POST("/plugins", a.installPlugin)
	group.POST("/plugins/:plugin_id/promote", a.promotePlugin)
	group.POST("/plugins/:plugin_id/rollback", a.rollbackPlugin)
	group.POST("/plugins/bundle", a.installPluginBundle)

	// FASE 30 -- Agent Center. O painel consome as tres no mesmo commit: *uma rota que existe e nao e'
	// consumida e' uma promessa do backend, nao uma funcionalidade do produto.*
	group.GET("/agents", a.listAgentPersonas)
	group.GET("/agents/catalog", a.listAgentCatalog)
	group.GET("/agents/evaluation", a.listAgentEvaluation)
	group.GET("/agents/teams", a.listAgentTeams)
	group.POST("/agents/backup", a.backupAgentPersonas)
	group.GET("/agents/backups", a.listAgentBackups)
	group.POST("/agents/restore", a.restoreAgentPersonas)
	group.POST("/agents/import", a.importAgentPersona)
	group.POST("/agents/quarantine", a.holdAgentPersona)
	group.GET("/agents/quarantine", a.listAgentQuarantine)
	group.POST("/agents/quarantine/release", a.releaseAgentPersona)
	group.POST("/agents/evaluation/run", a.runAgentEvaluation)
	group.GET("/agents/drift", a.agentPersonaDrift)
	group.POST("/agents/install", a.installAgentPersona)
	group.POST("/agents/remove", a.removeAgentPersona)
	group.POST("/agents/update", a.updateAgentPersona)
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
			mfaErr = a.auth.VerifyRecoveryCodeWithThrottle(user.ID, recoveryCode, throttleHostKey(c.Request.RemoteAddr), time.Now().UTC())
		} else {
			mfaErr = a.auth.VerifyMFAWithThrottle(user.ID, mfaCode, throttleHostKey(c.Request.RemoteAddr), time.Now().UTC())
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
	if c == nil || c.Request == nil || c.Request.Method == http.MethodOptions || isPublicSSORoute(c) {
		return true
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		// Cliente não-browser (CLI, SDK). O browser sempre envia Origin em
		// request cross-origin, inclusive GET, então a ausência não é bypass.
		return true
	}
	if strings.ContainsAny(origin, "\r\n") {
		return false
	}
	// A superfície agêntica executa e administra; origens opacas herdadas da
	// API de modelos (file://, app://, tauri://, vscode-*) não valem aqui.
	if !isHTTPOrigin(origin) {
		return false
	}
	for _, allowed := range envconfig.AllowedOrigins() {
		allowed = strings.TrimRight(strings.TrimSpace(allowed), "/")
		if allowed == "*" || !isHTTPOriginPattern(allowed) {
			continue
		}
		if originMatchesAllowed(origin, allowed) {
			return true
		}
	}
	return false
}

func isHTTPOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}

func isHTTPOriginPattern(allowed string) bool {
	lowered := strings.ToLower(strings.TrimSpace(allowed))
	return strings.HasPrefix(lowered, "http://") || strings.HasPrefix(lowered, "https://")
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

// devTokenSecretHeader carries the operator credential that gates dev-token
// issuance.
const devTokenSecretHeader = "X-Ollama-Agent-Dev-Secret"

func isDevTokenRequestAllowed(c *gin.Context) bool {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_DEV")), "true") {
		return false
	}
	// Credencial de operador obrigatória. As checagens de loopback abaixo são
	// defesa em profundidade, mas um proxy same-host que encaminha para
	// 127.0.0.1 sem adicionar cabeçalhos passaria por elas; o segredo fecha esse
	// caminho, pois o atacante remoto não o conhece. Sem segredo configurado, o
	// endpoint é fail-closed.
	secret := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_AUTH_DEV_SECRET"))
	if secret == "" {
		return false
	}
	presented := strings.TrimSpace(c.GetHeader(devTokenSecretHeader))
	if subtle.ConstantTimeCompare([]byte(presented), []byte(secret)) != 1 {
		return false
	}
	// Se o servidor escuta fora de loopback, há um proxy ou acesso remoto na
	// frente e o RemoteAddr loopback deixa de provar origem local. O token de
	// dev só é emitido quando o próprio listener é loopback-only.
	if !serverListensOnLoopback() {
		return false
	}
	if !isLoopbackRemoteAddr(c.Request.RemoteAddr) {
		return false
	}
	// Atrás de um reverse proxy no mesmo host, RemoteAddr é sempre loopback.
	// A presença de cabeçalhos de encaminhamento prova que o request foi
	// intermediado, então o loopback deixa de ser evidência de origem local.
	return !hasForwardedHeaders(c.Request)
}

// globalProcessScopeAllowed autoriza superfícies de processo global (sem campo
// de organização), como o buffer de egress e a config do supervisor. Um admin
// de uma organização não deve enxergar nem alterar o estado global das demais:
// o acesso exige modo local single-tenant ou um opt-in explícito do operador do
// servidor.
func (a *agentAPI) globalProcessScopeAllowed(c *gin.Context, surface string) bool {
	if !a.authRequired {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_ALLOW_GLOBAL_SCOPE")), "true") {
		return a.requireOrganizationAdmin(c, surface)
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": surface + " is process-global and not scoped to one organization; set OLLAMA_AGENT_ALLOW_GLOBAL_SCOPE=true to permit a server operator to access it"})
	return false
}

func serverListensOnLoopback() bool {
	host := envconfig.Host()
	if host == nil {
		return false
	}
	ip := net.ParseIP(host.Hostname())
	return ip != nil && ip.IsLoopback()
}

func hasForwardedHeaders(request *http.Request) bool {
	if request == nil {
		return false
	}
	for _, header := range []string{"X-Forwarded-For", "X-Real-Ip", "Forwarded", "X-Forwarded-Host", "X-Client-Ip"} {
		if strings.TrimSpace(request.Header.Get(header)) != "" {
			return true
		}
	}
	return false
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

// throttleHostKey returns the client host without the ephemeral TCP port so
// rate limits key on the IP, not IP:port. Keying on the full RemoteAddr let a
// caller reset an MFA/recovery throttle just by opening a new connection with a
// fresh source port (SEC-05).
func throttleHostKey(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = strings.Trim(remoteAddr, "[]")
	}
	if host == "" {
		return remoteAddr
	}
	return host
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

// tenantOrganizationID devolve a organização explicitamente autenticada no
// contexto da requisição. Diferente de agentOrganizationID, NÃO cai para a
// organização local: string vazia significa "chamador sem locatário", que é
// exatamente o caso em que uma superfície de escopo global precisa do opt-in
// explícito do operador (OLLAMA_AGENT_ALLOW_GLOBAL_SCOPE).
func tenantOrganizationID(c *gin.Context) string {
	if value, ok := c.Get("agent.organization"); ok {
		if organization, ok := value.(agent.Organization); ok {
			return strings.TrimSpace(organization.ID)
		}
	}
	return ""
}

func agentOrganizationID(c *gin.Context) string {
	// The organization is trusted only from the request context, which the auth
	// middleware populates from the verified bearer token (and fixes to the
	// local organization in local mode). The client-supplied
	// X-Ollama-Organization header is NEVER used to select the tenant: doing so
	// would let any caller on an auth-exempt route (e.g. an OAuth callback) pick
	// an arbitrary organization. The header is only ever cross-checked against
	// the authenticated org in authMiddleware. Absent a context org, fail closed
	// to the local organization.
	if organizationID := tenantOrganizationID(c); organizationID != "" {
		return organizationID
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
	queueHealth := "ok"
	var queueErr error
	if a.runtime != nil {
		for _, probe := range []struct {
			status agent.QueueStatus
			count  *int
		}{
			{agent.QueuePending, &pending},
			{agent.QueueRunning, &running},
			{agent.QueueDeadLetter, &dead},
		} {
			jobs, err := a.runtime.QueueJobsWithError(probe.status)
			if err != nil {
				queueErr = err
				break
			}
			*probe.count = len(jobs)
		}
	}
	status := "ok"
	if !runtimeReady {
		status = "degraded"
	}
	if queueErr != nil {
		// The queue dependency is unreachable or returned an unexpected
		// response. Report it as unavailable instead of masking the outage as
		// a healthy, empty queue.
		queueHealth = "unavailable"
		status = "degraded"
	} else if dead > 0 {
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
			// E-168/E-169: a POLÍTICA DE DURABILIDADE fica VISÍVEL. Trocar `fsync` por velocidade é uma decisão
			// sobre perder escritas numa queda abrupta — quem opera precisa ver em qual modo o sistema está e
			// quantas escritas ainda NÃO estão sincronizadas, em vez de descobrir por um arquivo perdido.
			"escrita": gin.H{
				"status": agent.WriteDurability(),
				"detail": map[string]string{
					agent.WriteDurabilityDurable: "Grava e sincroniza arquivo e diretório (sobrevive a queda abrupta)",
					agent.WriteDurabilityBatched: "Grava agora e sincroniza no máximo uma vez por janela de 2s: a perda possível fica limitada à janela",
					agent.WriteDurabilityRelaxed: "Grava sem sincronizar: mais rápido, o último lote pode ser perdido numa queda abrupta",
				}[agent.WriteDurability()],
				// A janela em aberto é medida, não prometida.
				"pendentes": agent.PendingDurableWrites(),
			},
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

// verifySecondFactorBeforeCredentialChange exige a prova de posse do SEGUNDO factor antes de uma operacao
// que o altera. Sem isto, um bearer token valido desativava o MFA e gerava codigos novos com um pedido
// vazio -- e um segundo factor que qualquer portador do primeiro pode remover nao e' um segundo factor.
func (a *agentAPI) verifySecondFactorBeforeCredentialChange(c *gin.Context, user agent.User) bool {
	if !user.MFAEnabled {
		// Sem MFA ligado nao ha' segundo factor para provar. A operacao pedida e' recusada pelo handler.
		return true
	}
	code := strings.TrimSpace(c.GetHeader("X-Ollama-MFA-Code"))
	recovery := strings.TrimSpace(c.GetHeader("X-Ollama-MFA-Recovery-Code"))
	key := throttleHostKey(c.Request.RemoteAddr)
	now := time.Now().UTC()
	var err error
	if recovery != "" {
		err = a.auth.VerifyRecoveryCodeWithThrottle(user.ID, recovery, key, now)
	} else {
		err = a.auth.VerifyMFAWithThrottle(user.ID, code, key, now)
	}
	if err == nil {
		return true
	}
	var throttle *agent.MFAThrottleError
	if errors.As(err, &throttle) {
		retryAfter := int64(throttle.RetryAfter / time.Second)
		if throttle.RetryAfter%time.Second != 0 {
			retryAfter++
		}
		c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "too many mfa attempts",
		})
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error": "changing mfa requires proof of the second factor (X-Ollama-MFA-Code)",
	})
	return false
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
	if !a.verifySecondFactorBeforeCredentialChange(c, user) {
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
	// Gerar codigos novos de um segundo factor que esta' desligado nao producao nada: recusa.
	if !user.MFAEnabled {
		writeAgentError(c, http.StatusBadRequest, errors.New("mfa is not enabled for this user"))
		return
	}
	if !a.verifySecondFactorBeforeCredentialChange(c, user) {
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

// skillVersions lista as versões ARQUIVADAS de uma skill (E-642).
//
// O escopo é aplicado DENTRO do store (E-631): a skill que não é da organização de quem pergunta
// devolve VAZIO, e não um erro — *distinguir «não existe» de «não é sua» seria uma pista sobre a
// existência dela.* E «não consegui olhar» é ERRO, não vazio: as duas coisas são diferentes.
func (a *agentAPI) skillVersions(c *gin.Context) {
	versoes, err := a.context.SkillVersionsForOrganization(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": versoes})
}

// compararVersoesDeSkill compara duas revisoes de uma skill (E-649).
//
// `from` e `to` vem por QUERY e uma delas pode ser a versao ACTIVA — o historico guarda so' as
// superadas, e o nucleo (`CompararVersoesDeSkill`, E-648) sabe resolver isso com o manifesto vivo.
//
// A comparacao NAO mostra `trusted`/`enabled` como mudanca de conteudo: sao estado derivado, e
// apresenta-los sugeriria que uma versao antiga e' mais confiavel do que e'.
func (a *agentAPI) compararVersoesDeSkill(c *gin.Context) {
	diferenca, err := a.context.CompararVersoesDeSkill(agentOrganizationID(c), c.Param("id"), c.Query("from"), c.Query("to"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"difference": diferenca})
}

// restoreSkillVersion restaura uma versão arquivada (E-642).
//
// ***RESTAURAR É RECUPERAR CONTEÚDO, NÃO CONFIANÇA***: o caminho passa pelo mesmo `registerSkill`,
// que impõe `Trusted=false` e verifica o escopo de organização (E-631). Promover a confiança
// continua a exigir `PromoteSkillTrustedForOrganization` com política.
func (a *agentAPI) restoreSkillVersion(c *gin.Context) {
	manifesto, err := a.context.RestoreSkillVersionForOrganization(agentOrganizationID(c), c.Param("id"), c.Param("version"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"skill": manifesto})
}

// mcpRPC responde a pedidos MCP (JSON-RPC 2.0) na organizacao DA SESSAO (E-647).
//
// SEM ISTO, O RESPONDEDOR NAO SERVIA NINGUEM: o E-646 implementou os metodos e a ferramenta, mas
// nenhum cliente MCP real se ligava. Este e' o transporte.
//
// O VERIFICADOR DE PROJETO USA O MESMO GUARD DO RESTO (`projectForRequestID`): um projeto que nao
// pertence a organizacao da sessao nao e' projeto - e o servidor responde o mesmo para "nao existe"
// e "nao e' seu", para nao dar pistas sobre o que existe noutra organizacao.
//
// E OS ERROS SAO DO PROTOCOLO, NAO DO HTTP: um metodo desconhecido devolve 200 com corpo de erro
// JSON-RPC. *Quem fala MCP le JSON-RPC - responder 4xx obrigaria o cliente a adivinhar.*
func (a *agentAPI) mcpRPC(c *gin.Context) {
	var pedido struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusOK, gin.H{"jsonrpc": "2.0", "id": nil, "error": gin.H{"code": -32700, "message": "parse error"}})
		return
	}
	verificarProjeto := func(projectID string) bool {
		_, err := a.projectForRequestID(c, projectID)
		return err == nil
	}
	servidor := agent.NovoServidorMCP(a.context, agentOrganizationID(c), verificarProjeto)
	resultado, erro := servidor.Responder(c.Request.Context(), pedido.Method, pedido.Params)
	if erro != nil {
		c.JSON(http.StatusOK, gin.H{"jsonrpc": "2.0", "id": pedido.ID, "error": gin.H{"code": erro.Codigo, "message": erro.Mensagem}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jsonrpc": "2.0", "id": pedido.ID, "result": resultado})
}

func (a *agentAPI) skills(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"skills": a.context.SkillsForOrganization(organizationID)})
}

// activateSkill ativa uma skill para um contexto (projeto/agente/modelo). O
// servidor recusa skill não confiável: catálogo não concede privilégio.
func (a *agentAPI) activateSkill(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var payload struct {
		ProjectID string `json:"project_id"`
		AgentID   string `json:"agent_id"`
		Model     string `json:"model"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	activation, err := a.context.ActivateSkillForOrganization(organizationID, agent.SkillActivation{
		SkillID:   c.Param("id"),
		ProjectID: payload.ProjectID,
		AgentID:   payload.AgentID,
		Model:     payload.Model,
	})
	if err != nil {
		writeAgentError(c, statusForSkillError(err), err)
		return
	}
	c.JSON(http.StatusOK, activation)
}

// deactivateSkill remove a ativação do contexto informado.
func (a *agentAPI) deactivateSkill(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var payload struct {
		ProjectID string `json:"project_id"`
		AgentID   string `json:"agent_id"`
		Model     string `json:"model"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	key := agent.SkillActivation{
		SkillID:   c.Param("id"),
		ProjectID: payload.ProjectID,
		AgentID:   payload.AgentID,
		Model:     payload.Model,
	}.Key()
	if err := a.context.DeactivateSkillForOrganization(organizationID, key); err != nil {
		writeAgentError(c, statusForSkillError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deactivated": key})
}

// activeSkills devolve a seleção real do catálogo para o contexto pedido, com o
// orçamento de tokens aplicado e o motivo de cada exclusão.
func (a *agentAPI) activeSkills(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	budget := 0
	if raw := strings.TrimSpace(c.Query("token_budget")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeAgentError(c, http.StatusBadRequest, errors.New("token_budget precisa ser um inteiro não negativo"))
			return
		}
		budget = parsed
	}
	selection, err := a.context.SelectSkillsForMission(organizationID, agent.SkillContext{
		ProjectID: c.Query("project_id"),
		AgentID:   c.Query("agent_id"),
		Model:     c.Query("model"),
	}, budget)
	if err != nil {
		writeAgentError(c, statusForSkillError(err), err)
		return
	}
	c.JSON(http.StatusOK, selection)
}

// statusForSkillError mapeia recusas do catálogo para HTTP sem transformar
// negação de segurança em 500.
func statusForSkillError(err error) int {
	switch {
	case errors.Is(err, agent.ErrSkillNotTrusted):
		return http.StatusForbidden
	case errors.Is(err, agent.ErrSkillNotFound), errors.Is(err, agent.ErrSkillActivationNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrSkillActivationInvalid), errors.Is(err, agent.ErrUnknownCapability):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (a *agentAPI) schedules(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": a.context.ListSchedulesForOrganization(organizationID)})
}

// decideScheduleApproval registra a decisão HUMANA sobre um passo approval.request.
// A decisão fica vinculada ao passo e à execução: aprovar durante uma pausa não
// autoriza a mesma etapa numa execução futura. A retomada acontece no
// reprocessamento, que reusa a mesma execução.
func (a *agentAPI) decideScheduleApproval(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent runtime is not configured"))
		return
	}
	var body struct {
		StepID  string `json:"step_id"`
		RunKey  string `json:"run_key"`
		Approve bool   `json:"approved"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.StepID) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("step_id is required"))
		return
	}
	schedule, err := a.context.GetScheduleForOrganization(c.Param("id"), organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	runKey := strings.TrimSpace(body.RunKey)
	if runKey == "" {
		// Sem chave explícita, a decisão se aplica à execução que está PARADA.
		if len(schedule.PendingApprovals) == 0 {
			writeAgentError(c, http.StatusConflict, errors.New("schedule has no pending approval"))
			return
		}
		for _, pending := range schedule.PendingApprovals {
			if pending.StepID == strings.TrimSpace(body.StepID) {
				runKey = pending.RunKey
				break
			}
		}
		if runKey == "" {
			writeAgentError(c, http.StatusConflict, errors.New("step is not waiting for approval"))
			return
		}
	}
	runtime := a.runtime.WithOrganizationContext(c.Request.Context(), organizationID)
	updated, err := runtime.DecideScheduleApproval(schedule, body.StepID, runKey, body.Approve, agentActorID(c), body.Reason)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedule": updated, "approved": body.Approve})
}

// reprocessScheduleNow reexecuta SOMENTE os passos que ficaram ignorados ou
// falharam na última execução, reusando a chave daquela execução. Os passos que já
// tiveram sucesso são preservados e não geram missão nova (idempotência).
func (a *agentAPI) reprocessScheduleNow(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent runtime is not configured"))
		return
	}
	schedule, err := a.context.GetScheduleForOrganization(c.Param("id"), organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	runtime := a.runtime.WithOrganizationContext(c.Request.Context(), organizationID)
	report, runErr := runtime.ReprocessScheduleFlow(c.Request.Context(), schedule)
	if errors.Is(runErr, agent.ErrScheduleReprocessUnavailable) {
		// Reprocessar sem execução anterior exigiria adivinhar a chave da execução,
		// e adivinhar poderia criar missão duplicada. Recusa explícita.
		writeAgentError(c, http.StatusConflict, runErr)
		return
	}
	updated := schedule
	recordWarning := ""
	if recorded, recordErr := runtime.RecordScheduleRun(schedule, report, runErr); recordErr == nil {
		updated = recorded
	} else {
		recordWarning = recordErr.Error()
	}
	payload := gin.H{
		"schedule":  updated,
		"report":    report,
		"succeeded": runErr == nil,
	}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}
	if recordWarning != "" {
		payload["record_warning"] = recordWarning
	}
	c.JSON(http.StatusOK, payload)
}

// runScheduleNow executa o GRAFO persistido do schedule, e não uma missão avulsa.
//
// O botão de execução imediata da interface criava uma missão com o objetivo do
// schedule e ignorava os passos: um fluxo com condição e várias ações era
// executado como uma única missão. Aqui a execução manual passa pelo MESMO
// caminho do despacho automático (ExecuteScheduleFlowReport) e registra o
// resultado por nó, para que a evidência não divirja entre manual e agendado.
func (a *agentAPI) runScheduleNow(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent runtime is not configured"))
		return
	}
	schedule, err := a.context.GetScheduleForOrganization(c.Param("id"), organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	runtime := a.runtime.WithOrganizationContext(c.Request.Context(), organizationID)
	report, runErr := runtime.ExecuteScheduleFlowReport(c.Request.Context(), schedule)
	updated := schedule
	recordWarning := ""
	if recorded, recordErr := runtime.RecordScheduleRun(schedule, report, runErr); recordErr == nil {
		updated = recorded
	} else {
		recordWarning = recordErr.Error()
	}
	payload := gin.H{
		"schedule":  updated,
		"report":    report,
		"succeeded": runErr == nil,
	}
	if runErr != nil {
		// O relatório vai junto: uma execução que falhou precisa chegar ao
		// operador com o passo culpado, não como erro genérico.
		payload["error"] = runErr.Error()
	}
	if recordWarning != "" {
		payload["record_warning"] = recordWarning
	}
	c.JSON(http.StatusOK, payload)
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

// collabMembers devolve os MEMBROS do projeto — humanos e agentes na MESMA lista (complemento 48, FASE 2).
func (a *agentAPI) collabMembers(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"membros": a.runtime.Collaboration().Participants(c.Param("project_id"))})
}

// collabJoinProject põe um humano OU um agente na lista de membros. A ORIGEM é o ator autenticado da requisição
// - *não se aceita do corpo: quem declara é quem está a chamar, e isso o cliente não escolhe.*
func (a *agentAPI) collabJoinProject(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		ID   string `json:"id"`
		Tipo string `json:"tipo"`
		Nome string `json:"nome"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	membro, err := a.runtime.Collaboration().JoinProject(
		c.Param("project_id"), request.ID, agent.ParticipantKind(request.Tipo), request.Nome, a.collabActor(c),
	)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, membro)
}

// collabRooms lista as salas do projeto.
func (a *agentAPI) collabRooms(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"salas": a.runtime.Collaboration().Rooms(c.Param("project_id"))})
}

// collabCreateRoom cria a sala. O criador tem de ser membro do projeto - *a regra vive no domínio, e a rota não
// a contorna: quem não é membro recebe 400 com o motivo.*
func (a *agentAPI) collabCreateRoom(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		ID   string `json:"id"`
		Nome string `json:"nome"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	sala, err := a.runtime.Collaboration().CreateRoom(c.Param("project_id"), request.ID, request.Nome, a.collabActor(c))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, sala)
}

// collabRoomMembers devolve os membros de UMA sala.
//
// SALA INEXISTENTE ⇒ **404**, e não uma lista vazia: *«não há ninguém» e «essa sala não existe» são coisas
// diferentes* - a mesma distinção que a superfície de UI testa. **Uma lista vazia só significa sala vazia.**
func (a *agentAPI) collabRoomMembers(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	projectID := c.Param("project_id")
	roomID := c.Param("room_id")
	existe := false
	for _, sala := range a.runtime.Collaboration().Rooms(projectID) {
		if sala.ID == roomID {
			existe = true
			break
		}
	}
	if !existe {
		writeAgentError(c, http.StatusNotFound, agent.ErrRoomNotFound)
		return
	}
	c.JSON(http.StatusOK, gin.H{"membros": a.runtime.Collaboration().RoomMembers(projectID, roomID)})
}

// collabJoinRoom põe um MEMBRO DO PROJETO na sala. O domínio recusa quem não é membro; a rota devolve o motivo.
func (a *agentAPI) collabJoinRoom(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	sala, err := a.runtime.Collaboration().JoinRoom(c.Param("project_id"), c.Param("room_id"), request.ID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, sala)
}

func (a *agentAPI) collabStream(c *gin.Context) {
	if _, err := a.projectForRequestID(c, c.Param("project_id")); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	release, ok := a.acquireStreamSlot(c)
	if !ok {
		return
	}
	defer release()
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

// updateProjectMemory altera o conteúdo de uma memória visível ao ator e preserva
// a revisão anterior no histórico (versionamento, estágio 15).
func (a *agentAPI) updateProjectMemory(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		Content string `json:"content"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.context.UpdateMemoryForActor(c.Request.Context(), c.Param("id"), c.Param("memory_id"), agentActorID(c), request.Content, request.Reason)
	if err != nil {
		if errors.Is(err, agent.ErrMemoryNotFound) {
			// Mesmo 404 para memória inexistente e para memória privada alheia: a
			// resposta não revela a existência de dado privado.
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// memoryHistory devolve as revisões anteriores de uma memória visível ao ator.
func (a *agentAPI) memoryHistory(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	history, err := a.context.MemoryHistoryForActor(c.Param("id"), c.Param("memory_id"), agentActorID(c))
	if err != nil {
		if errors.Is(err, agent.ErrMemoryNotFound) {
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id": c.Param("id"),
		"memory_id":  c.Param("memory_id"),
		"count":      len(history),
		"versions":   history,
	})
}

// agentMemoryScope monta o escopo de leitura a partir da requisição. O ATOR vem
// sempre da sessão; missão, agente e equipe vêm de parâmetro de consulta e apenas
// RESTRINGEM a divulgação (um valor errado esconde, nunca revela).
func agentMemoryScope(c *gin.Context) agent.MemoryScope {
	return agent.MemoryScope{
		ActorID:   agentActorID(c),
		MissionID: strings.TrimSpace(c.Query("mission_id")),
		AgentID:   strings.TrimSpace(c.Query("agent_id")),
		TeamID:    strings.TrimSpace(c.Query("team_id")),
	}
}

// memorySummary resume a memória visível ao escopo, DECLARANDO qual resumidor foi
// usado. O padrão é o digest determinístico: o texto copia trechos com referência em
// vez de gerar afirmações novas.
func (a *agentAPI) memorySummary(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	summary, err := a.context.SummarizeMemoriesInScope(c.Request.Context(), c.Param("id"), agentMemoryScope(c), c.Query("q"), agent.DefaultMemorySummarySources, 0, nil)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, summary)
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
	// O ator vem da sessão: o corpo da requisição não escolhe a autoria de uma
	// memória privada (isolamento por usuário, estágio 15).
	created, err := a.context.AddMemoryForActor(c.Request.Context(), memory, agentActorID(c))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

// reembedProjectMemories recalcula os vetores do projeto no espaço do embedder ATUAL.
//
// A rota existe porque a detecção sem re-embed só sabe DECLARAR que parte do conhecimento ficou fora do
// alcance semântico; sem um gatilho de operador, a declaração vira promessa sem remédio. O escopo é o do
// projeto (mesma checagem das outras rotas de memória) e o `limit` evita transformar manutenção em uma
// requisição que ninguém consegue terminar — o que sobrar volta em `remaining`.
func (a *agentAPI) reembedProjectMemories(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	limit := maxReembedPerRequest
	if bruto := strings.TrimSpace(c.Query("limit")); bruto != "" {
		valor, err := strconv.Atoi(bruto)
		if err != nil || valor < 0 {
			writeAgentError(c, http.StatusBadRequest, errors.New("limit precisa ser um inteiro não negativo"))
			return
		}
		if valor < limit {
			limit = valor
		}
	}
	resultado, err := a.context.ReembedMemories(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id": c.Param("id"),
		"reembedded": resultado.Reembedded,
		"failed":     resultado.Failed,
		"skipped":    resultado.Skipped,
		"remaining":  resultado.Remaining,
		// O limite vigente vai na resposta: sem ele, `remaining` seria um número sem referência.
		"limit": limit,
	})
}

// maxReembedPerRequest limita quantas memórias uma chamada de re-embed recalcula.
const maxReembedPerRequest = 200

// startReembedJob dispara o re-embed ASSÍNCRONO do projeto.
//
// A rota síncrona continua existindo para volumes pequenos; esta existe porque uma base grande não pode ser
// operada por tentativa e erro: o job devolve id e a estimativa, e o progresso é consultável.
func (a *agentAPI) startReembedJob(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	job, err := a.context.StartReembedJob(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusAccepted, job)
}

// reembedJobStatus devolve o progresso do job.
func (a *agentAPI) reembedJobStatus(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	job, err := a.context.ReembedJobStatus(c.Param("job_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// O job é do PROJETO: consultar por outro projeto não pode revelar andamento alheio.
	if job.ProjectID != c.Param("id") {
		writeAgentError(c, http.StatusNotFound, errors.New("job não pertence a este projeto"))
		return
	}
	c.JSON(http.StatusOK, job)
}

// cancelReembedJob pede o cancelamento, sem prometer que nada mais será processado.
func (a *agentAPI) cancelReembedJob(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	job, err := a.context.ReembedJobStatus(c.Param("job_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if job.ProjectID != c.Param("id") {
		writeAgentError(c, http.StatusNotFound, errors.New("job não pertence a este projeto"))
		return
	}
	// Cancelar é PEDIDO: o job verifica entre lotes e termina com o que já fez contabilizado.
	aceito := a.context.CancelReembedJob(job.ID)
	c.JSON(http.StatusOK, gin.H{"job_id": job.ID, "cancel_requested": aceito})
}

// reembedEstimate diz quanto trabalho existe ANTES de começar (a estimativa que faltava em E-159).
func (a *agentAPI) reembedEstimate(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	projectID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"project_id": projectID,
		"pendentes":  a.context.EstimateReembed(projectID),
		// O espaço de destino vai junto: "faltam 300" sem dizer PARA QUAL modelo é um número solto.
		"modelo_atual": a.context.CurrentEmbeddingModel(),
	})
}

func (a *agentAPI) searchMemories(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// O escopo vem da requisição (missão/agente/equipe) e restringe a DIVULGAÇÃO:
	// memória presa a uma missão não aparece em busca sem essa missão.
	memories, err := a.context.SearchMemoriesInScope(c.Request.Context(), c.Param("id"), agentMemoryScope(c), c.Query("q"), 20)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_id": c.Param("id"), "memories": memories})
}

// askProjectDocuments retrieves relevant excerpts from the project's indexed
// documents (RAG, G1). It returns citations and snippets only; it does not
// synthesize an answer or expose the internal prompt/context block to the UI.
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
	// O RAG também respeita o isolamento por usuário: memória privada de outro
	// ator não entra no contexto nem nas citações.
	_, citations, err := a.context.GroundedAnswerForActor(c.Request.Context(), c.Param("id"), agentActorID(c), query, askLimit, askMinScore)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id": c.Param("id"),
		"query":      query,
		"citations":  citations,
		"grounded":   len(citations) > 0,
	})
}

// exportProjectMemories devolve a memória do projeto visível ao ATOR autenticado,
// em ordem cronológica, com proveniência (kind, source, confidence, created_at) e
// SEM os vetores de embedding por padrão. `?include_embeddings=true` inclui os
// vetores.
//
// A exportação é escopada por ator de propósito: o caminho anterior exportava o
// projeto INTEIRO, então a busca escondia a memória privada de um colega e o
// export a entregava — mesmo dado, duas respostas.
func (a *agentAPI) exportProjectMemories(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	includeEmbeddings := false
	if raw := strings.TrimSpace(c.Query("include_embeddings")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			writeAgentError(c, http.StatusBadRequest, errors.New("include_embeddings deve ser booleano"))
			return
		}
		includeEmbeddings = parsed
	}
	export, err := a.context.ExportMemoriesInScope(c.Param("id"), agentMemoryScope(c), includeEmbeddings)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id":         export.ProjectID,
		"actor_id":           export.ActorID,
		"generated_at":       export.GeneratedAt,
		"count":              export.Count,
		"include_embeddings": includeEmbeddings,
		"memories":           export.Memories,
	})
}

// deleteProjectMemory apaga uma memória específica do projeto QUANDO o ator
// autenticado pode lê-la. Memória de outro projeto — ou memória privada de outro
// ator — devolve 404: a resposta não serve de oráculo sobre a existência de dado
// privado alheio.
func (a *agentAPI) deleteProjectMemory(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if err := a.context.DeleteMemoryForActor(c.Param("id"), c.Param("memory_id"), agentActorID(c)); err != nil {
		if errors.Is(err, agent.ErrMemoryNotFound) {
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     "deleted",
		"project_id": c.Param("id"),
		"memory_id":  c.Param("memory_id"),
	})
}

// retentionPolicy mostra a política declarada da organização autenticada. A
// ausência de política é resposta legítima (configured=false), não erro.
func (a *agentAPI) retentionPolicy(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	policy, err := a.context.RetentionPolicyForOrganization(organizationID)
	if err != nil {
		if errors.Is(err, agent.ErrRetentionNotConfigured) {
			c.JSON(http.StatusOK, gin.H{"organization_id": organizationID, "configured": false})
			return
		}
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"organization_id": organizationID, "configured": true, "policy": policy})
}

// setRetentionPolicy grava a política de retenção da organização autenticada. A
// organização vem da sessão, nunca do corpo: uma organização não define política
// para outra.
func (a *agentAPI) setRetentionPolicy(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	var request struct {
		MaxAgeDays            int `json:"max_age_days"`
		MaxMemoriesPerProject int `json:"max_memories_per_project"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	policy, err := a.context.SetRetentionPolicyForOrganization(agent.MemoryRetentionPolicy{
		OrganizationID:        organizationID,
		MaxAgeDays:            request.MaxAgeDays,
		MaxMemoriesPerProject: request.MaxMemoriesPerProject,
	})
	if err != nil {
		writeAgentError(c, statusForRetentionError(err), err)
		return
	}
	c.JSON(http.StatusOK, policy)
}

// applyRetentionPolicy aplica a política agora e devolve os números REAIS do
// que foi removido. Aplicar duas vezes é idempotente.
func (a *agentAPI) applyRetentionPolicy(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	result, err := a.context.ApplyRetentionForOrganization(organizationID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForRetentionError(err), err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func statusForRetentionError(err error) int {
	switch {
	case errors.Is(err, agent.ErrRetentionPolicyInvalid):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrRetentionNotConfigured):
		return http.StatusNotFound
	default:
		return statusForAgentError(err)
	}
}

// pruneProjectMemories aplica retenção por idade. O corte é obrigatório e
// limitado para que um pedido não apague o histórico inteiro por acidente.
func (a *agentAPI) pruneProjectMemories(c *gin.Context) {
	if _, err := a.projectForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var request struct {
		OlderThanDays int `json:"older_than_days"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if request.OlderThanDays < 1 || request.OlderThanDays > 3650 {
		writeAgentError(c, http.StatusBadRequest, errors.New("older_than_days deve estar entre 1 e 3650"))
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -request.OlderThanDays)
	removed, err := a.context.PruneMemories(c.Param("id"), cutoff)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id":      c.Param("id"),
		"removed":         removed,
		"older_than_days": request.OlderThanDays,
	})
}

// missoesDoChamador devolve as missoes que a ORGANIZACAO DA SESSAO pode ver (E-626).
//
// Existe porque o filtro estava DUPLICADO no `missions` e no `performance`, e duplicacao de guard
// de inquilino e' a que sai mais caro: uma mudanca num dos sitios deixa o outro a filtrar a menos
// — e uma rota que filtra a menos NAO SE NOTA, porque continua a responder 200 com dados de outro
// inquilino. Agora ha' um sitio so'. (A versao anterior reutilizava `missions[:0]` no proprio
// slice; esta aloca o resultado, para nao deixar aliasing a quem chame.)
func (a *agentAPI) missoesDoChamador(c *gin.Context) ([]agent.Mission, error) {
	missions, err := a.scopedRuntime(c).ListMissions()
	if err != nil {
		return nil, err
	}
	return missoesDaOrganizacao(missions, agentOrganizationID(c)), nil
}

// missoesDaOrganizacao e' a REGRA, separada da ligacao HTTP de proposito (E-627): assim o
// isolamento entre inquilinos e' PROVADO por um teste puro, sem harness de servidor, e nao fica
// dependente de um teste de rota que ninguem escreveu. Organizacao vazia devolve tudo — e' o
// comportamento anterior e esta' fixado por teste, para nao mudar por acidente.
func missoesDaOrganizacao(missions []agent.Mission, organizationID string) []agent.Mission {
	if organizationID == "" {
		return missions
	}
	filtered := make([]agent.Mission, 0, len(missions))
	for _, mission := range missions {
		if mission.OrganizationID == organizationID {
			filtered = append(filtered, mission)
		}
	}
	return filtered
}

func (a *agentAPI) missions(c *gin.Context) {
	missions, err := a.missoesDoChamador(c)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"missions": missions})
}

// performance agrega o desempenho das missoes QUE O CHAMADOR PODE VER (E-625).
//
// O FILTRO POR ORGANIZACAO E' O MESMO DO `missions` ACIMA, E E' OBRIGATORIO. Sem ele o relatorio
// contaria missoes de outro inquilino: uma rota que PARECE governada e filtra a menos e' pior do
// que uma rota ausente, porque a ausente sabe-se que falta (foi a licao do D-5, quando a
// identidade de missao no chat usava a funcao errada e um teste negativo passava por o recurso
// nunca funcionar).
func (a *agentAPI) performance(c *gin.Context) {
	missions, err := a.missoesDoChamador(c)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	// O relatorio DECLARA o escopo: um agregado apresentado sem escopo e' lido como global.
	c.JSON(http.StatusOK, gin.H{
		"performance": agent.AvaliarDesempenho(missions),
		"scope":       "missoes visiveis para o chamador (organizacao da sessao)",
	})
}

// wakeups calcula os despertares COALESCIDOS por agente e ENFILEIRA o que precisa de execucao
// (E-638). E' a ligacao entre a regra pura (E-633) e a fila duravel que ja' existe.
//
// O QUE VAI A' FILA: so' as missoes cujo motivo e' `missao pronta a executar`, decidido por
// `agent.MissoesAExecutar` (funcao pura, testada). Aprovacao pendente e recuperacao produzem
// despertar INFORMATIVO: *enfileirar uma missao que espera aprovacao seria executa-la sem a
// decisao que ela propria pediu.*
//
// NAO DUPLICA: `EnqueueMission` e' idempotente por missao (ha' teste no pacote do agente) - logo
// repetir o despertar nao cria job novo. E' o que torna esta rota segura de chamar periodicamente.
//
// FALHA DE UMA MISSAO NAO ESCONDE AS OUTRAS: é reportada em `failed`, e se NADA foi enfileirado a
// resposta e' 500 — *sucesso parcial declarado nao e' sucesso, e silencio nao e' desfecho.*
func (a *agentAPI) wakeups(c *gin.Context) {
	missions, err := a.missoesDoChamador(c)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	// UMA implementacao, dois chamadores (E-639): a MESMA operacao que o tique do agendador usa.
	// Duplicar aqui a decisao de enfileirar deixaria um lado a agir a menos sem se notar.
	resultado := a.scopedRuntime(c).DespertarAgentes(missions, time.Now().UTC())
	resposta := gin.H{
		"wakeups":  resultado.Despertares,
		"enqueued": resultado.Enfileiradas,
		"failed":   resultado.Falhas,
		"scope":    "missoes visiveis para o chamador (organizacao da sessao)",
	}
	if len(resultado.Enfileiradas) == 0 && len(resultado.Falhas) > 0 {
		c.JSON(http.StatusInternalServerError, resposta)
		return
	}
	c.JSON(http.StatusOK, resposta)
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
	release, ok := a.acquireStreamSlot(c)
	if !ok {
		return
	}
	defer release()
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

func validateAutoRunCapabilities(capabilities []string) error {
	for _, capability := range capabilities {
		if strings.TrimSpace(capability) != "" && strings.TrimSpace(capability) != "workspace:read" {
			return errors.New("autorun accepts only workspace:read; create the mission without autorun for elevated capabilities")
		}
	}
	return nil
}

func (a *agentAPI) createMission(c *gin.Context) {
	var request agent.CreateMissionRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if request.AutoRun {
		if err := validateAutoRunCapabilities(request.Capabilities); err != nil {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
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

// missionContext devolve o Hades Context Bootstrap canônico de uma missão
// (estágio 5): identidade e versão do servidor, escopo autorizado, estado
// declarado de cada permissão, plano registrado e memória do projeto.
//
// É somente leitura: nada aqui concede capacidade. A missão passa por
// `missionForRequest`, que já aplica o isolamento por organização, e as
// memórias vêm do `ContextStore` limitado ao projeto da missão. O documento
// nunca carrega credencial: todo texto é redigido antes de sair.
func (a *agentAPI) missionContext(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	memories := []agent.Memory{}
	if a.context != nil && strings.TrimSpace(mission.ProjectID) != "" {
		// Busca escopada por ATOR: a memória privada de outro usuário do mesmo
		// projeto não pode entrar no contexto entregue ao modelo. O bootstrap
		// também filtra (fail-closed), mas não se entrega o que não se deve ler.
		found, searchErr := a.context.SearchMemoriesForActor(c.Request.Context(), mission.ProjectID, agentActorID(c), mission.Objective, agent.MaxContextBootstrapMemories)
		if searchErr != nil {
			writeAgentError(c, statusForAgentError(searchErr), searchErr)
			return
		}
		memories = found
	}
	// HANDOFF (§4): derivado da missão para que objetivo, plano, capacidades concedidas e o
	// que EXIGE APROVAÇÃO cheguem ao contexto entregue. Derivar é mais seguro do que anexar.
	// O erro NÃO derruba o contexto: uma missão sem handoff possível continua a ter contexto
	// — entregar sem handoff é degradação declarada; falhar por causa dele seria pior.
	// O bloco que vai aqui é o REDIGIDO (`handoff_preambulo.go`), porque este documento
	// alimenta o preâmbulo que vai ao modelo (vazamento medido em mission_context_test.go:118).
	var handoff *agent.Handoff
	if derivado, handoffErr := agent.HandoffDaMissao(mission); handoffErr == nil {
		handoff = &derivado
	}
	bootstrap, err := agent.BuildContextBootstrap(agent.ContextBootstrapInput{
		Handoff:             handoff,
		Version:             version.Version,
		OrganizationID:      mission.OrganizationID,
		ProjectID:           mission.ProjectID,
		MissionID:           mission.ID,
		ActorID:             agentActorID(c),
		Workspace:           mission.Workspace,
		Objective:           mission.Objective,
		PlanTitles:          missionPlanTitles(mission.Plan),
		Memories:            memories,
		GrantedCapabilities: mission.Capabilities,
		Policy:              agent.DefaultCapabilityPolicy(),
	})
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, bootstrap)
}

func missionPlanTitles(steps []agent.Step) []string {
	titles := make([]string, 0, len(steps))
	for _, step := range steps {
		if title := strings.TrimSpace(step.Title); title != "" {
			titles = append(titles, title)
		}
	}
	return titles
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
	// Optional limit returns only the most recent N events so a long-running
	// mission's history cannot force an unbounded response. total lets the
	// client know how many exist. Without limit the behavior is unchanged.
	total := len(events)
	if limit, limitErr := strconv.Atoi(strings.TrimSpace(c.Query("limit"))); limitErr == nil && limit > 0 && limit < len(events) {
		events = events[len(events)-limit:]
	}
	c.JSON(http.StatusOK, gin.H{"mission_id": c.Param("id"), "events": events, "total": total})
}

// companion devolve o estado que o Companion PODE dizer sobre uma missao (item 46).
//
// O que isto NAO faz, de proposito:
//
//   - **Nao inventa estado.** O estado sai de `agent.DerivarEstadoCompanion` sobre os
//     eventos PERSISTIDOS da missao. Sem eventos, a derivacao responde o estado honesto
//     ("ausente") com a razao - *nao um "a trabalhar" de cortesia.*
//   - **Nao transporta detalhe de evento.** `Event.Payload` nao tem sanitizacao
//     garantida neste caminho, e o nucleo exige detalhe *ja' sanitizado a montante*;
//     por isso o detalhe vai vazio (ver `internal/agent/companion_eventos.go`).
//   - **Nao inventa autorizacao.** O escopo e' o que `missionForRequest` ja' prova; a
//     resposta DECLARA esse escopo, porque um agregado sem escopo e' lido como global.
func (a *agentAPI) companion(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	missaoID := c.Param("id")
	eventos, err := a.scopedRuntime(c).Events(missaoID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	estado, razao, err := agent.DerivarEstadoCompanion(missaoID, agent.EventosDoCompanionDaMissao(missaoID, eventos, nil))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mission_id": missaoID,
		"estado":     estado,
		"razao":      razao,
		"scope":      "missao do chamador (organizacao da sessao)",
	})
}

func (a *agentAPI) eventStream(c *gin.Context) {
	if _, err := a.missionForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	release, ok := a.acquireStreamSlot(c)
	if !ok {
		return
	}
	defer release()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Status(http.StatusNotImplemented)
		return
	}
	// Resume from the client's Last-Event-ID so a reconnect does not replay the
	// entire history. EventSource sends this header automatically with the id:
	// we emit below; other clients may send it explicitly.
	lastCount := 0
	if resume := strings.TrimSpace(c.GetHeader("Last-Event-ID")); resume != "" {
		if parsed, parseErr := strconv.Atoi(resume); parseErr == nil && parsed >= 0 {
			lastCount = parsed
		}
	}
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	const heartbeatEveryTicks = 20 // ~15s with a 750ms tick
	idleTicks := 0
	for {
		events, err := a.scopedRuntime(c).Events(c.Param("id"))
		if err != nil {
			return
		}
		// Guard against a cursor beyond the current log (e.g. a stale resume id
		// or a reset log) so the slice below never panics.
		if lastCount > len(events) {
			lastCount = 0
		}
		if len(events) > lastCount {
			payload, _ := json.Marshal(events[lastCount:])
			// The id: lets the client resume from here on reconnect.
			_, _ = fmt.Fprintf(c.Writer, "id: %d\nevent: mission\ndata: %s\n\n", len(events), payload)
			flusher.Flush()
			lastCount = len(events)
			idleTicks = 0
		} else {
			// Periodic heartbeat comment keeps the connection (and any proxy)
			// alive without emitting a data event. SSE comment lines start with
			// ':' and are ignored by clients.
			idleTicks++
			if idleTicks >= heartbeatEveryTicks {
				_, _ = fmt.Fprint(c.Writer, ": heartbeat\n\n")
				flusher.Flush()
				idleTicks = 0
			}
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
	// F1 DA AUDITORIA DE SEGURANÇA (P1): `Workspace` e `ProjectID` vêm do CORPO do
	// pedido, e ate' aqui iam direto para `PlanForOrganization` SEM verificar se
	// pertencem à organização autenticada. O guard que todos os outros caminhos usam
	// (`projectForRequestID`) estava AUSENTE precisamente aqui — e o runner do
	// orquestrador é o runtime BASE (`organizationScope == ""`), logo a checagem de
	// dono que existe no runtime também não corria. Um membro de outra organização
	// podia apontar o job a um projeto alheio e LER o workspace dele.
	// A correcção reusa o guard existente em vez de criar uma segunda regra que
	// pudesse divergir (o defeito do "irmão esquecido").
	if strings.TrimSpace(request.ProjectID) != "" {
		if _, err := a.projectForRequestID(c, request.ProjectID); err != nil {
			writeAgentError(c, statusForAgentError(err), err)
			return
		}
	}
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
	// The mission is enqueued, not yet running: the transition to RUNNING only
	// happens when a worker claims the job. Report the real mission state and
	// the queued job instead of a premature RUNNING that misleads the client.
	c.JSON(http.StatusAccepted, gin.H{"mission_id": id, "state": mission.State, "queued": true, "job": job})
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

func (a *agentAPI) deleteMission(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("mission id is required"))
		return
	}
	if err := a.scopedRuntime(c).DeleteMission(c.Request.Context(), id); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
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

// requireOrganizationAdmin exige papel owner/admin para superfícies administrativas.
// Em modo local (sem auth) o servidor já é single-tenant e loopback-only.
func (a *agentAPI) requireOrganizationAdmin(c *gin.Context, surface string) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": surface + " requires organization admin"})
		return false
	}
	return true
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
	// SEC-03: the merge must carry the diff digest the reviewer approved so the
	// server can bind the approval to the exact diff being merged. Accept it from
	// the JSON body or, as a convenience, a query parameter.
	approvedDiffSHA := strings.TrimSpace(c.Query("approved_diff_sha256"))
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		var body struct {
			ApprovedDiffSHA256 string `json:"approved_diff_sha256"`
		}
		if err := decodeJSON(c, &body); err == nil && strings.TrimSpace(body.ApprovedDiffSHA256) != "" {
			approvedDiffSHA = strings.TrimSpace(body.ApprovedDiffSHA256)
		}
	}
	commitSHA, err := a.scopedRuntime(c).MergeMissionWorktree(c.Request.Context(), id, approvedDiffSHA)
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
	// `ErrRestauroConflitante` entra aqui: a restauracao de versao de artefacto usa o CAS
	// (`expectedVersion`) e um conflito e' 409 - *nao* 500. Sem isto, o cliente nao distingue
	// «alguem escreveu entretanto» de «o servidor avariou».
	if errors.Is(err, agent.ErrApprovalVersionConflict) || errors.Is(err, agent.ErrBuilderVersionConflict) || errors.Is(err, agent.ErrDeploymentApprovalNonce) || errors.Is(err, agent.ErrDeploymentApprovalConflict) || errors.Is(err, agent.ErrDeploymentApprovalExpired) || errors.Is(err, agent.ErrReembedJobRunning) || errors.Is(err, agent.ErrRestauroConflitante) {
		return http.StatusConflict
	}
	// Job de re-embed desconhecido é 404: o recurso não existe (ou se perdeu com o reinício do processo).
	// `ErrVersaoInexistente` junta-se ao mesmo 404: *o erro e' PARTILHADO entre artefactos
	// (`ArtefactoCompararVersoes`) e skills (`CompararVersoesDeSkill`)* - mapear aqui serve as duas
	// funcionalidades; mapear na rota do artefacto deixaria as skills a devolver 500.
	if errors.Is(err, agent.ErrReembedJobNotFound) || errors.Is(err, agent.ErrVersaoInexistente) {
		return http.StatusNotFound
	}
	if errors.Is(err, agent.ErrBuilderExportExpired) {
		return http.StatusGone
	}
	if errors.Is(err, agent.ErrDeploymentApprovalNotFound) || errors.Is(err, agent.ErrDeploymentApprovalOrganization) || errors.Is(err, agent.ErrDeploymentProviderNotFound) {
		return http.StatusNotFound
	}
	// Omnichannel (estágio 34): cada recusa tem o código que o provedor precisa ver.
	// Assinatura inválida é 401 (o provedor precisa corrigir a assinatura); carimbo
	// fora da janela é 409 (a assinatura confere, mas o evento não é aceitável agora) —
	// separar os dois evita mandar o provedor depurar a coisa errada.
	if errors.Is(err, agent.ErrOmnichannelSignature) {
		return http.StatusUnauthorized
	}
	if errors.Is(err, agent.ErrOmnichannelReplay) {
		return http.StatusConflict
	}
	if errors.Is(err, agent.ErrOmnichannelRateLimited) {
		return http.StatusTooManyRequests
	}
	if errors.Is(err, agent.ErrOmnichannelNotConfigured) || errors.Is(err, agent.ErrOmnichannelDLQNotFound) {
		return http.StatusNotFound
	}
	// Comércio (estágio 24): estoque insuficiente é 409 (o pedido não cabe no saldo),
	// dado de cartão é 422 (a requisição é inaceitável por política) e conflito de
	// estado/conciliação é 409.
	if errors.Is(err, agent.ErrCommerceProductMissing) || errors.Is(err, agent.ErrCommerceOrderMissing) {
		return http.StatusNotFound
	}
	if errors.Is(err, agent.ErrCommerceCardDataRefused) {
		return http.StatusUnprocessableEntity
	}
	if errors.Is(err, agent.ErrCommerceInsufficient) || errors.Is(err, agent.ErrCommerceOrderState) || errors.Is(err, agent.ErrCommerceReconcileConflict) {
		return http.StatusConflict
	}
	// Financeiro (estágio 30): lançar receita de pedido NÃO PAGO é 409 (o estado não
	// permite), conciliar com referência divergente é 409 (conflito) e
	// organização/pedido ausente é 404.
	if errors.Is(err, agent.ErrFinanceOrderNotPayable) || errors.Is(err, agent.ErrLedgerAlreadyReconciled) {
		return http.StatusConflict
	}
	if errors.Is(err, agent.ErrFinanceOrderMissing) || errors.Is(err, agent.ErrFinanceCompanyMissing) {
		return http.StatusNotFound
	}
	// Supervisor de negócio (estágio 31): ação inexistente é 404, tipo não executável é
	// 422 (a requisição é inaceitável) e orçamento estourado é 409.
	if errors.Is(err, agent.ErrBusinessActionNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, agent.ErrBusinessActionUnknown) {
		return http.StatusUnprocessableEntity
	}
	if errors.Is(err, agent.ErrBusinessBudgetExceeded) || errors.Is(err, agent.ErrBusinessActionDecided) {
		return http.StatusConflict
	}
	// Fábrica de empresa (estágio 23): margem/custo inválidos são erro do cliente (400);
	// empresa de outra organização não é acessível (404).
	if errors.Is(err, agent.ErrCompanyPricingInvalid) || errors.Is(err, agent.ErrCompanyFactoryInvalid) {
		return http.StatusBadRequest
	}
	if errors.Is(err, agent.ErrCompanyNotFound) {
		return http.StatusNotFound
	}
	// Força de vendas (estágio 27): SKU fora do catálogo e pedido inválido são erro de
	// cliente (422/400); proposta ou acompanhamento inexistente é 404; proposta sem
	// consentimento é 403; estado que não permite a ação ou vigência vencida é 409.
	if errors.Is(err, agent.ErrProposalConsentRequired) {
		return http.StatusForbidden
	}
	if errors.Is(err, agent.ErrProposalNotFound) || errors.Is(err, agent.ErrFollowUpNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, agent.ErrProposalStateInvalid) || errors.Is(err, agent.ErrProposalExpired) || errors.Is(err, agent.ErrFollowUpStateInvalid) {
		return http.StatusConflict
	}
	if errors.Is(err, agent.ErrFollowUpInvalid) {
		return http.StatusBadRequest
	}
	if errors.Is(err, agent.ErrProposalUnknownSKU) {
		return http.StatusUnprocessableEntity
	}
	if errors.Is(err, agent.ErrProposalInvalid) {
		return http.StatusBadRequest
	}
	// Campanha em sandbox (estágio 26): agendamento e métrica inválidos são erro de cliente
	// (400); estado que não permite a ação e orçamento estourado são conflito (409).
	if errors.Is(err, agent.ErrCampaignScheduleInvalid) || errors.Is(err, agent.ErrCampaignMetricsInvalid) {
		return http.StatusBadRequest
	}
	if errors.Is(err, agent.ErrCampaignBudgetExceeded) || errors.Is(err, agent.ErrCampaignStateInvalid) {
		return http.StatusConflict
	}
	// Marketplace (estágio 25): 429 quando o teto de operações por execução é excedido (é limitação
	// de taxa, não erro do pedido), 404 para canal desconhecido e 400 para pedido inválido.
	if errors.Is(err, agent.ErrMarketplaceRateLimited) {
		return http.StatusTooManyRequests
	}
	if errors.Is(err, agent.ErrMarketplaceChannelUnknown) {
		return http.StatusNotFound
	}
	if errors.Is(err, agent.ErrMarketplaceSyncInvalid) {
		return http.StatusBadRequest
	}
	// BI / status de negócio (estágio 32): painel pedido sem organização é erro do cliente.
	if errors.Is(err, agent.ErrBusinessStatusScopeRequired) {
		return http.StatusBadRequest
	}
	// Série temporal do BI (estágio 32): escopo ausente é erro do cliente.
	if errors.Is(err, agent.ErrBusinessSnapshotScopeRequired) {
		return http.StatusBadRequest
	}
	// Exportação do BI (estágio 32): escopo ausente é erro do cliente.
	if errors.Is(err, agent.ErrBusinessExportScope) {
		return http.StatusBadRequest
	}
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

func agentTenantContextKeyVersionFromEnv() (int, error) {
	return agent.TenantContextKeyVersionFromEnv()
}

// missionPresets devolve o catalogo de presets de MISSAO (FASE 23 do adendo V3).
//
// O catalogo e' o mesmo que `agent.BuiltInMissionPresets()` serve ao `/preset`: uma unica fonte,
// dois acessos. *Uma segunda copia do catalogo para a API seria uma segunda verdade com o mesmo
// nome -- e a que ficasse para tras mentiria sem aviso.*
//
// Nao expoe nada por tenant: presets sao configuracao de PRODUTO, nao dado de utilizador. O que
// cada preset PEDE (`permissions`) e' o que o policy engine depois avalia; esta rota nao concede.
func (a *agentAPI) missionPresets(c *gin.Context) {
	presets := agent.BuiltInMissionPresets()
	c.JSON(http.StatusOK, gin.H{
		"presets": presets,
		"count":   len(presets),
	})
}

// routing expoe a decisao de roteamento de modelo com os candidatos E a fronteira de Pareto
// (FASE 19 do adendo V3). E' a superficie que faltava: a fronteira existia no motor e no
// evento interno, mas nao havia como o Mission Control mostrar ALTERNATIVAS.
//
// Parameters (todos opcionais):
//
//	capabilities= coding,vision   capacidades obrigatorias (repetivel ou separado por virgula)
//	preference=   quality         auto (vazio) | fastest | quality | cheapest | private
//	local_only=   true            RESTRICAO rigida: recusa o que nao for local
//	max_cost_cents=10             tecto de custo por 1k tokens
//	path=         /api/chat       caminho de inferencia (default /api/chat)
//
// Devolve `winner`, `candidates` (com os factors que SOMAM o score) e `pareto_frontier`.
// Sem registry de modelos devolve 503 -- *nunca 200 com lista vazia: uma lista vazia
// pareceria "nao ha' modelos" em vez de "nao ha' motor de modelos ligado".*
func (a *agentAPI) routing(c *gin.Context) {
	if a == nil || a.runtime == nil || a.runtime.Models() == nil {
		writeAgentError(c, http.StatusServiceUnavailable,
			errors.New("model registry unavailable: o runtime de agente nao tem motor de modelos ligado; "+
				"a rota nao devolve lista vazia porque uma lista vazia pareceria a resposta"))
		return
	}
	capabilities := make([]string, 0, 4)
	for _, bruto := range c.QueryArray("capabilities") {
		for _, parte := range strings.Split(bruto, ",") {
			if limpo := strings.TrimSpace(parte); limpo != "" {
				capabilities = append(capabilities, limpo)
			}
		}
	}
	preference, err := preferenciaDeRota(c.Query("preference"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	localOnly, err := boolDeQuery(c.Query("local_only"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, fmt.Errorf("local_only: %w", err))
		return
	}
	maxCost := int64(0)
	if bruto := strings.TrimSpace(c.Query("max_cost_cents")); bruto != "" {
		valor, convErr := strconv.ParseInt(bruto, 10, 64)
		if convErr != nil || valor < 0 {
			writeAgentError(c, http.StatusBadRequest, errors.New("max_cost_cents tem de ser um inteiro >= 0"))
			return
		}
		maxCost = valor
	}
	caminho := strings.TrimSpace(c.Query("path"))
	if caminho == "" {
		caminho = "/api/chat"
	}

	registro := a.runtime.Models()
	pedido := multillm.RouteRequest{
		RequiredCapabilities: capabilities,
		Path:                 caminho,
		Preference:           preference,
		LocalOnly:            localOnly,
		MaxCostCents:         maxCost,
	}
	candidatos, err := registro.RouteCandidates(pedido)
	if err != nil {
		if errors.Is(err, multillm.ErrNoRoute) {
			// A recusa e' o resultado honesto, e vem com o pedido que a produziu.
			writeAgentError(c, http.StatusUnprocessableEntity, fmt.Errorf(
				"nenhum modelo satisfaz as restricoes (capabilities=%v local_only=%v max_cost_cents=%d path=%q): %w",
				capabilities, localOnly, maxCost, caminho, err))
			return
		}
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}

	localDe := func(decisao multillm.RouteDecision) bool {
		provider, existe := registro.Provider(decisao.Model.Provider)
		return existe && provider.AllowPrivate
	}
	trilha := make([]gin.H, 0, len(candidatos))
	for indice, candidato := range candidatos {
		trilha = append(trilha, gin.H{
			"rank":              indice + 1,
			"model":             candidato.Model.ID,
			"provider":          candidato.Model.Provider,
			"score":             candidato.Score,
			"cost_cents_per_1k": candidato.Model.CostPer1KInputCents + candidato.Model.CostPer1KOutputCents,
			"quality_score":     candidato.Model.QualityScore,
			"reason":            candidato.Reason,
			// Os factors SOMAM o score. *Um score sem os componentes que o somam obriga
			// quem o le' a confiar em quem o escreveu.*
			"factors": candidato.Factors,
		})
	}
	fronteira := make([]gin.H, 0, len(candidatos))
	for _, alternativa := range multillm.ParetoResumo(candidatos, localDe) {
		fronteira = append(fronteira, gin.H{
			"model":   alternativa.ModelID,
			"score":   alternativa.Score,
			"quality": alternativa.Eixos.Quality,
			"cost":    alternativa.Eixos.Cost,
			"latency": alternativa.Eixos.Latency,
			"privacy": alternativa.Eixos.Privacy,
		})
	}

	vencedor := candidatos[0]
	c.JSON(http.StatusOK, gin.H{
		"winner": gin.H{
			"model":    vencedor.Model.ID,
			"provider": vencedor.Model.Provider,
			"score":    vencedor.Score,
			"reason":   vencedor.Reason,
		},
		"candidates": trilha,
		// `pareto_frontier` sao as alternativas NAO-DOMINADAS. Um cliente que so' leia
		// `winner` nao sabe que existia uma escolha melhor noutro eixo.
		"pareto_frontier": fronteira,
		// Declarar os eixos que NAO foram medidos e' parte do contrato: *inventar energia ou
		// VRAM quando a plataforma nao os fornece seria um numero falso num relatorio.*
		"axes_measured":     []string{"quality", "cost", "latency", "privacy"},
		"axes_not_measured": []string{"energy", "vram"},
		"request": gin.H{
			"capabilities":   capabilities,
			"preference":     string(preference),
			"local_only":     localOnly,
			"max_cost_cents": maxCost,
			"path":           caminho,
		},
	})
}

// preferenciaDeRota traduz o parametro de query. Vazio = Auto (o default do motor).
func preferenciaDeRota(bruto string) (multillm.RoutePreference, error) {
	switch strings.ToLower(strings.TrimSpace(bruto)) {
	case "":
		return multillm.RoutePreferenceAuto, nil
	case "fastest":
		return multillm.RoutePreferenceFastest, nil
	case "quality", "best_quality":
		return multillm.RoutePreferenceBestQuality, nil
	case "cheapest":
		return multillm.RoutePreferenceCheapest, nil
	case "private", "most_private":
		return multillm.RoutePreferenceMostPrivate, nil
	default:
		return "", fmt.Errorf("preference invalida %q: use auto, fastest, quality, cheapest ou private", bruto)
	}
}

// boolDeQuery aceita os valores que um cliente HTTP realmente envia; recusa o resto em vez de
// adivinhar. *Adivinhar `local_only` para "yes please" seria escolher seguranca por conta propria.*
func boolDeQuery(bruto string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(bruto)) {
	case "", "false", "0", "no":
		return false, nil
	case "true", "1", "yes":
		return true, nil
	default:
		return false, fmt.Errorf("valor booleano invalido %q", bruto)
	}
}

// agentTelemetry devolve a telemetria AGREGADA das missoes visiveis ao chamador (FASE 21).
//
// PORQUE NAO SE CHAMA `/metrics`: essa rota JA' EXISTE e devolve `runtime.Metrics()`, que sao
// CONTADORES (stepsStarted, toolCalls, retries). *Dois conteudos diferentes com o mesmo nome
// seriam duas verdades sobre a mesma pergunta* -- o consumidor pediria `/metrics` e nao
// saberia qual das duas recebeu.
//
// A colheita vem dos eventos `step.succeeded`/`step.failed`, que passam a trazer `duration_ms`
// medido em volta do `tool.Execute`. *Sem essa medicao no runtime, a agregacao seria decorativa:
// somaria zeros bonitos.*
//
// O QUE A ROTA **NAO** INVENTA: tokens, custo realizado, energia e VRAM. O `step.model.route`
// traz custo ESTIMADO por 1k tokens -- que nao e' custo realizado e nao se soma como se fosse.
// Esses eixos saem em `axes_not_measured`, e isso e' o contrato, nao uma omissao.
func (a *agentAPI) agentTelemetry(c *gin.Context) {
	missions, err := a.missoesDoChamador(c)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	runtime := a.scopedRuntime(c)
	resumo, ilegiveis := agent.MetricasDasMissoes(missions, runtime.Events)
	c.JSON(http.StatusOK, gin.H{
		"metrics": resumo,
		// A comparacao por modelo ordena por p95, nao pela media: *a cauda e' o que o
		// utilizador sente.*
		"by_model": agent.ComparacaoPorModeloAmostras(missions, runtime.Events),
		// Declarar o escopo e' parte do contrato: um agregado sem escopo le-se como global.
		"scope": "missoes visiveis para o chamador (organizacao da sessao)",
		// Uma leitura que falhou nao pode apagar as metricas das outras missoes -- mas
		// tambem nao pode desaparecer. *Uma falha silenciosa faria o relatorio parecer mais
		// completo do que e'.*
		"missions_unreadable": ilegiveis,
		"percentiles_method":  agent.MetodoDosPercentis,
	})
}
