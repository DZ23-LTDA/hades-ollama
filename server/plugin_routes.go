package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func (a *agentAPI) requirePluginAdmin(c *gin.Context) bool {
	if !a.authRequired {
		return true
	}
	value, _ := c.Get("agent.membership")
	membership, ok := value.(agent.Membership)
	if !ok || (membership.Role != agent.RoleOwner && membership.Role != agent.RoleAdmin) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "plugin lifecycle requires organization admin"})
		return false
	}
	return true
}

func pluginCatalog(c *gin.Context, a *agentAPI) {
	organizationID := ""
	if a.authRequired {
		organizationID = agentOrganizationID(c)
	}
	if !a.authRequired {
		organizationID = agent.LocalOrganizationID
	}
	connectors := a.runtime.ConnectorsForOrganization(organizationID)
	mcp := a.runtime.MCPServersForOrganization(organizationID)
	remoteMCP := a.runtime.RemoteMCPServersForOrganization(organizationID)
	skills := a.runtime.SkillsForOrganization(organizationID)
	c.JSON(http.StatusOK, gin.H{
		"connectors": connectors,
		"mcp":        mcp,
		"remote_mcp": remoteMCP,
		"skills":     skills,
	})
}

func connectorCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"connectors": agent.ConnectorCatalog()})
}

func writePluginLifecycleError(c *gin.Context, err error) {
	if errors.Is(err, agent.ErrPluginOrganizationScope) || errors.Is(err, agent.ErrPluginNotFound) {
		writeAgentError(c, http.StatusNotFound, agent.ErrPluginNotFound)
		return
	}
	writeAgentError(c, http.StatusBadRequest, err)
}

func writePluginRegistrationError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, agent.ErrPluginOrganizationScope) {
		status = http.StatusForbidden
	}
	writeAgentError(c, status, err)
}

func (a *agentAPI) enableConnector(c *gin.Context)  { a.connectorLifecycle(c, true, false) }
func (a *agentAPI) disableConnector(c *gin.Context) { a.connectorLifecycle(c, false, false) }
func (a *agentAPI) removeConnector(c *gin.Context)  { a.connectorLifecycle(c, false, true) }

func (a *agentAPI) registerConnector(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.ConnectorConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if input.CredentialConfigured {
		writeAgentError(c, http.StatusBadRequest, errors.New("credential_configured is server-derived and cannot be supplied"))
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterConnectorForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		// The unauthenticated local mode has one operator scope. The runtime
		// still validates the HTTPS endpoint, methods and environment names.
		err = a.runtime.RegisterConnector(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) connectorLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveConnectorForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveConnector(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetConnectorEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetConnectorEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableMCP(c *gin.Context)  { a.mcpLifecycle(c, true, false) }
func (a *agentAPI) disableMCP(c *gin.Context) { a.mcpLifecycle(c, false, false) }
func (a *agentAPI) removeMCP(c *gin.Context)  { a.mcpLifecycle(c, false, true) }

func (a *agentAPI) registerMCP(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.MCPServerConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterMCPForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterMCP(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) mcpLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveMCPForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveMCP(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetMCPEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetMCPEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableRemoteMCP(c *gin.Context)  { a.remoteMCPLifecycle(c, true, false) }
func (a *agentAPI) disableRemoteMCP(c *gin.Context) { a.remoteMCPLifecycle(c, false, false) }
func (a *agentAPI) removeRemoteMCP(c *gin.Context)  { a.remoteMCPLifecycle(c, false, true) }

func (a *agentAPI) registerRemoteMCP(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.RemoteMCPServerConfig
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterRemoteMCPForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterRemoteMCP(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) remoteMCPLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveRemoteMCPForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveRemoteMCP(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetRemoteMCPEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetRemoteMCPEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) enableSkill(c *gin.Context)  { a.skillLifecycle(c, true, false) }
func (a *agentAPI) disableSkill(c *gin.Context) { a.skillLifecycle(c, false, false) }
func (a *agentAPI) removeSkill(c *gin.Context)  { a.skillLifecycle(c, false, true) }
func (a *agentAPI) registerSkill(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var input agent.SkillManifest
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if input.Trusted || input.Enabled {
		writeAgentError(c, http.StatusBadRequest, errors.New("skill trusted and enabled state are server-derived"))
		return
	}
	var err error
	if a.authRequired {
		err = a.runtime.RegisterSkillForOrganization(agentOrganizationID(c), input)
	} else {
		if input.OrganizationID != "" {
			writeAgentError(c, http.StatusBadRequest, errors.New("organization_id is not accepted without authenticated organization scope"))
			return
		}
		err = a.runtime.RegisterSkill(input)
	}
	if err != nil {
		writePluginRegistrationError(c, err)
		return
	}
	pluginCatalog(c, a)
}

func (a *agentAPI) skillLifecycle(c *gin.Context, enabled, remove bool) {
	if !a.requirePluginAdmin(c) {
		return
	}
	var err error
	organizationID := agentOrganizationID(c)
	if remove {
		if a.authRequired {
			err = a.runtime.RemoveSkillForOrganization(organizationID, c.Param("id"))
		} else {
			err = a.runtime.RemoveSkill(c.Param("id"))
		}
	} else {
		if a.authRequired {
			err = a.runtime.SetSkillEnabledForOrganization(organizationID, c.Param("id"), enabled)
		} else {
			err = a.runtime.SetSkillEnabled(c.Param("id"), enabled)
		}
	}
	if err != nil {
		writePluginLifecycleError(c, err)
		return
	}
	pluginCatalog(c, a)
}

// ---------------------------------------------------------------------------
// E-966: A ROTA DE EXECUCAO — a que faltava.
//
// **O QUE ESTA ROTA E'**: a porta pela qual um plugin INSTALADO executa. *Registar,
// habilitar e remover ja' existiam* (13 handlers acima); **executar** nao tinha
// superficie, e por isso `PluginRuntime.Execute` tinha **16 chamadas, todas em
// `*_test.go`, e zero em producao** (medido no `E-959`/`E-965`).
//
// **O QUE ELA *NAO* E', e precisa ficar dito**: ela **nao** implementa governanca
// nenhuma. *A governanca ja' existe, inteira, no `PluginRuntime.Execute`* — instalacao,
// estado habilitado, confianca, verificacao de conteudo (*reverificada a cada execucao*),
// entrypoint, escopos via `CapabilityPolicy`, modo de sandbox, aviso de aprovacao, e
// workspace temporario com protecao contra travessia, link simbolico e zip bomb.
// Esta rota **so' liga** o que existia — *e o `E-449` ja' mostrou o custo de nao ligar:
// «o motor existia, era testado, e nao estava ligado a nada».*
//
// **ESPECIFICIDADE**: o pedido **nao** traz `OrganizationID` nem `ActorID`, porque um
// cliente nao escolhe o seu proprio tenant nem a sua propria identidade. A organizacao vem
// do **contexto autenticado** (`agentOrganizationID`, que recusa o header do cliente — ver
// o comentario em `agent_routes.go:1268`), e o ator vem da **membership**. O `PluginID`
// vem do **caminho**. *O corpo so' confirma o que o manifesto assinado ja' declara* — e
// `Execute` recusa qualquer confirmacao divergente em vez de a aceitar como override.
//
// ***Um motor sem superficie nao e' uma capacidade: e' uma biblioteca.***
// ---------------------------------------------------------------------------

// pluginExecutionInput e' o corpo aceito pela rota de execucao.
//
// Nao tem `OrganizationID` nem `ActorID`: *o pedido nao escolhe tenant nem identidade.*
// Nao tem `Entrypoint` nem `Runtime`: *o manifesto assinado e' a fonte, e estes campos
// so' o confirmam.*
type pluginExecutionInput struct {
	MissionID              string `json:"mission_id"`
	StepID                 string `json:"step_id"`
	EntrypointConfirmation string `json:"entrypoint_confirmation"`
	RuntimeConfirmation    string `json:"runtime_confirmation"`
	SandboxMode            string `json:"sandbox_mode"`
	ApprovalNotice         string `json:"approval_notice"`
}

// writePluginExecutionError mapeia as portas de execucao para status HTTP.
//
// *Falha fechado*: um erro de execucao **nunca** vira 200. E o `writeAgentError` aplica
// `RedactDLP` a' mensagem, para que a recusa nao vire fuga de conteudo.
func writePluginExecutionError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, agent.ErrPluginExecutionNotInstalled),
		errors.Is(err, agent.ErrPluginNotFound):
		status = http.StatusNotFound
	case errors.Is(err, agent.ErrPluginExecutionDisabled):
		status = http.StatusConflict
	case errors.Is(err, agent.ErrPluginExecutionUntrusted),
		errors.Is(err, agent.ErrPluginExecutionUnverified):
		status = http.StatusForbidden
	case errors.Is(err, agent.ErrPluginExecutionScopeDenied),
		errors.Is(err, agent.ErrPluginExecutionApproval):
		status = http.StatusForbidden
	case errors.Is(err, agent.ErrPluginExecutionSandbox):
		// O sandbox rodou e falhou: *nao e' erro do pedido.*
		status = http.StatusBadGateway
	}
	writeAgentError(c, status, err)
}

// executePlugin executa um plugin instalado pela superficie canonica.
//
// `POST /plugins/:id/execute` — *o `:id` e' o plugin; tudo o resto que o identifica
// vem do contexto.*
func (a *agentAPI) executePlugin(c *gin.Context) {
	if !a.requirePluginAdmin(c) {
		return
	}
	if a.plugins == nil {
		// FALHA FECHADO, como no `E-449`: *sem registro nao ha' superficie* — e dizer
		// isso e' melhor que devolver um codigo que pareceria "plugin nao existe".
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("registro de plugins nao esta' ligado neste runtime"))
		return
	}
	var input pluginExecutionInput
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	// Espelha a `pluginStatuses` (`agent_routes.go:163-182`): *a mesma organizacao, a mesma
	// porta de erro, e a MESMA fonte de policy.* Nao se inventa uma segunda forma de obter
	// a policy — *duas fontes para a mesma decisao e' como se comeca a divergir.*
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
	actorID := ""
	if value, ok := c.Get("agent.membership"); ok {
		if membership, ok := value.(agent.Membership); ok {
			actorID = membership.UserID
		}
	}
	request := agent.PluginExecutionRequest{
		// Tenant e ator vem do contexto autenticado, nunca do corpo.
		OrganizationID: organizationID,
		PluginID:       c.Param("plugin_id"),
		MissionID:      input.MissionID,
		StepID:         input.StepID,
		ActorID:        actorID,
		// Confirmacoes: o manifesto assinado decide; isto so' confirma.
		EntrypointConfirmation: input.EntrypointConfirmation,
		RuntimeConfirmation:    input.RuntimeConfirmation,
		SandboxMode:            input.SandboxMode,
		ApprovalNotice:         input.ApprovalNotice,
	}
	pluginRuntime := agent.NewPluginRuntime(a.plugins, policy)
	result, err := pluginRuntime.Execute(c.Request.Context(), request)
	if err != nil {
		// O registro auditavel existe mesmo na recusa: `Succeeded` distingue
		// execucao comprovada de tentativa recusada, e devolve-lo permite auditar
		// a recusa. O erro HTTP continua a ser erro.
		if result.Record.PluginID != "" {
			c.Header("X-Hades-Plugin-Executed", "false")
		}
		writePluginExecutionError(c, err)
		return
	}
	c.Header("X-Hades-Plugin-Executed", "true")
	c.JSON(http.StatusOK, gin.H{
		"record":    result.Record,
		"output":    result.Output,
		"artifacts": result.Artifacts,
	})
}
