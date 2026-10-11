package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Agentes de atendimento e vendas (estágio 35). As rotas expõem o catálogo e o turno.
// O turno SEMPRE declara a procedência do texto e o motivo de eventual escalada.
var (
	errServiceAgentsAuditUnavailable     = errors.New("trilha de auditoria durável não está configurada")
	errServiceAgentsApprovalsUnavailable = errors.New("store de aprovação humana não está configurado")
	errServiceAgentApprovalStatusInvalid = errors.New("status de aprovação inválido")
	// errServiceAgentApproversActorRequired: mudar QUEM aprova exige ator identificado — é decisão de risco.
	errServiceAgentApproversActorRequired = errors.New("definir aprovadores exige ator identificado")
)

func (a *agentAPI) serviceAgents() *agent.ServiceAgentRuntime {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.ServiceAgents()
}

func (a *agentAPI) listServiceAgents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"agents": agent.ServiceAgentCatalog(), "count": len(agent.ServiceAgentCatalog())})
}

// serviceAgentAudit expõe a TRILHA DURÁVEL de execução de ferramentas da organização da sessão.
//
// A trilha vem com o ESTADO junto: durabilidade não se presume, e falha de gravação precisa aparecer para
// quem audita — uma trilha "em memória" apresentada como durável seria pior que a ausência dela.
func (a *agentAPI) serviceAgentAudit(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	organizationID := agentOrganizationID(c)
	status, err := runtime.AuditStatus(organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !status.Persisted {
		// Sem trilha durável a leitura é recusa declarada, não lista vazia: vazio pareceria "nada aconteceu".
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsAuditUnavailable)
		return
	}
	entries, err := runtime.AuditTrail(organizationID, 0)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "count": len(entries), "storage": status})
}

// serviceAgentApprovals lista os pedidos de aprovação da organização da sessão.
//
// O filtro `status` é validado: um valor desconhecido é ERRO, não uma lista vazia que pareceria "nada
// pendente" — em aprovação, mostrar vazio quando há pendências é o pior desfecho.
func (a *agentAPI) serviceAgentApprovals(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	store := runtime.ApprovalStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsApprovalsUnavailable)
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	switch status {
	case "", agent.ServiceAgentApprovalPending, agent.ServiceAgentApprovalApproved,
		agent.ServiceAgentApprovalDenied, agent.ServiceAgentApprovalConsumed, agent.ServiceAgentApprovalExpired:
	default:
		writeAgentError(c, http.StatusBadRequest, errServiceAgentApprovalStatusInvalid)
		return
	}
	organizationID := agentOrganizationID(c)
	approvals, err := store.ListApprovals(organizationID, status, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// A FILA acompanha a listagem com idade e atraso: é o que diz a quem opera o que está esperando demais.
	// Não há push/e-mail — o que existe é uma fila que não esconde o tempo de espera.
	queue, err := store.ApprovalQueue(organizationID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"approvals": approvals,
		"count":     len(approvals),
		"queue":     queue,
		// O estado do AVISO aparece junto: se o destino do webhook estiver fora do ar, quem opera precisa
		// saber que a fila existe mas ninguém foi avisado. A falha vai por CLASSE, sem URL e sem segredo.
		"notification": gin.H{
			"configurado": store.Notifier != nil,
			"entregues":   store.NotificationsSent(),
			"ultimo_erro": store.LastNotificationError(),
		},
	})
}

// serviceAgentDecideApproval registra a DECISÃO HUMANA sobre um pedido de escrita.
//
// Decidir exige ator identificado: aprovação anônima não é aprovação. A decisão fica no store durável para
// o próximo turno poder executar — ou ser bloqueado, quando negada.
func (a *agentAPI) serviceAgentDecideApproval(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	store := runtime.ApprovalStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsApprovalsUnavailable)
		return
	}
	var input struct {
		Approved bool   `json:"approved"`
		ActorID  string `json:"actor_id"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	approvalID := strings.TrimSpace(c.Param("id"))
	if approvalID == "" {
		writeAgentError(c, http.StatusBadRequest, errServiceAgentApprovalStatusInvalid)
		return
	}
	decided, err := store.Decide(agentOrganizationID(c), approvalID, input.Approved, input.ActorID, input.Reason, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, agent.ErrServiceAgentApprovalNotFound):
			writeAgentError(c, http.StatusNotFound, err)
		case errors.Is(err, agent.ErrServiceAgentApprovalNotAllowed):
			// Ator fora da lista de aprovadores: PROIBIDO, não "pedido inválido".
			writeAgentError(c, http.StatusForbidden, err)
		case errors.Is(err, agent.ErrServiceAgentApprovalConflict), errors.Is(err, agent.ErrServiceAgentApprovalExpired):
			writeAgentError(c, http.StatusConflict, err)
		default:
			writeAgentError(c, http.StatusBadRequest, err)
		}
		return
	}
	// O ATO humano entra na trilha durável. A decisão não é desfeita se a trilha não puder ser gravada — mas
	// a resposta DECLARA a ausência do registro, porque auditoria que falta em silêncio parece existir.
	auditRecorded := true
	auditReason := ""
	if err := runtime.RecordApprovalDecision(decided, input.ActorID, input.Reason, time.Now().UTC()); err != nil {
		auditRecorded = false
		auditReason = err.Error()
	}
	c.JSON(http.StatusOK, gin.H{
		"approval":       decided,
		"audit_recorded": auditRecorded,
		"audit_reason":   auditReason,
	})
}

// serviceAgentApprovalApprovers define QUEM pode aprovar nesta organização.
//
// A configuração exige ator identificado e é uma decisão de risco em si: definir quem aprova é definir o
// alcance da autorização. Sem `actor_id`, a alteração é recusada.
func (a *agentAPI) serviceAgentApprovalApprovers(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	store := runtime.ApprovalStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsApprovalsUnavailable)
		return
	}
	var input struct {
		ActorID   string   `json:"actor_id"`
		Approvers []string `json:"approvers"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.ActorID) == "" {
		writeAgentError(c, http.StatusBadRequest, errServiceAgentApproversActorRequired)
		return
	}
	organizationID := agentOrganizationID(c)
	if err := store.SetApprovers(organizationID, input.Approvers); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	approvers, err := store.Approvers(organizationID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// A mudança de QUEM aprova entra na trilha: é ato de política, e sem registro ela seria invisível para
	// quem audita. Se a trilha não puder gravar, a resposta DECLARA isso.
	auditRecorded := true
	auditReason := ""
	if err := runtime.RecordApproversChange(organizationID, input.ActorID, approvers, time.Now().UTC()); err != nil {
		auditRecorded = false
		auditReason = err.Error()
	}
	// A resposta diz o estado REAL: lista vazia significa que a decisão não passará por checagem de papel.
	c.JSON(http.StatusOK, gin.H{
		"approvers":      approvers,
		"count":          len(approvers),
		"role_checked":   len(approvers) > 0,
		"changed_by":     strings.TrimSpace(input.ActorID),
		"audit_recorded": auditRecorded,
		"audit_reason":   auditReason,
	})
}

// serviceAgentTurn executa um turno do agente sobre uma conversa da caixa de entrada.
func (a *agentAPI) serviceAgentTurn(c *gin.Context) {
	runtime := a.serviceAgents()
	if runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errServiceAgentsUnavailable)
		return
	}
	var input struct {
		ConversationID string `json:"conversation_id"`
		Intent         string `json:"intent"`
		ModelID        string `json:"model_id"`
		// RequestedTools é o que o agente PEDE para este turno. Pedir não é poder: o runtime confere
		// declaração do agente, executor registrado e política antes de qualquer efeito.
		RequestedTools []string `json:"requested_tools,omitempty"`
		// ToolArguments alimenta as ferramentas de escrita (título, valor, responsável, motivo) e as de
		// empresa/social/voz (empresa, sessão, provedor, corpo). O ESCOPO (organização, contato, conversa) vem
		// da sessão e da conversa resolvida, NUNCA daqui.
		ToolArguments struct {
			Title      string `json:"title,omitempty"`
			ValueCents int64  `json:"value_cents,omitempty"`
			OwnerID    string `json:"owner_id,omitempty"`
			Assignee   string `json:"assignee,omitempty"`
			Reason     string `json:"reason,omitempty"`
			CompanyID  string `json:"company_id,omitempty"`
			SessionID  string `json:"session_id,omitempty"`
			Provider   string `json:"provider,omitempty"`
			Body       string `json:"body,omitempty"`
		} `json:"tool_arguments,omitempty"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	turn, err := runtime.RunTurn(agent.ServiceAgentTurnRequest{
		OrganizationID: agentOrganizationID(c),
		AgentID:        c.Param("id"),
		ConversationID: input.ConversationID,
		Intent:         input.Intent,
		ModelID:        input.ModelID,
		RequestedTools: input.RequestedTools,
		ToolArguments: agent.ServiceAgentToolRequest{
			Title:      input.ToolArguments.Title,
			ValueCents: input.ToolArguments.ValueCents,
			OwnerID:    input.ToolArguments.OwnerID,
			Assignee:   input.ToolArguments.Assignee,
			Reason:     input.ToolArguments.Reason,
			CompanyID:  input.ToolArguments.CompanyID,
			SessionID:  input.ToolArguments.SessionID,
			Provider:   input.ToolArguments.Provider,
			Body:       input.ToolArguments.Body,
		},
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	// Escalada é uma resposta legítima e vem com o motivo; responder traz a mensagem.
	status := http.StatusOK
	if turn.Escalated {
		status = http.StatusAccepted
	}
	c.JSON(status, turn)
}

var errServiceAgentsUnavailable = agent.ErrServiceAgentTurnInvalid
