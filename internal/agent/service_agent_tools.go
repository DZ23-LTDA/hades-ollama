package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Execução REAL de ferramentas dos agentes de atendimento (estágio 35).
//
// O estado do projeto declarava: *"ferramentas apenas DECLARADAS (a execução real depende dos módulos de
// comércio, voz e conhecimento)"*. E havia uma consequência pior que a declaração: `DefaultServiceAgentTools`
// listava 11 ferramentas como **disponíveis** — e disponibilidade ali era um **booleano escrito à mão**.
// Ferramenta "disponível" sem executor é a forma mais fácil de apresentar capacidade que não existe.
//
// Este arquivo troca o booleano por **executor de verdade**:
//
//  1. **DISPONÍVEL = TEM EXECUTOR**. A disponibilidade passa a ser derivada do registro, não de uma lista
//     digitada. Se ninguém implementou, a ferramenta aparece como **indisponível, com motivo**;
//  2. **POLÍTICA ANTES DE EFEITO**: ferramenta de SAÍDA (outbound) exige consentimento concedido, e
//     ferramenta que o agente não declara é **bloqueada** mesmo que exista executor — privilégio não sobe
//     porque a ferramenta existe;
//  3. **NADA DE SUCESSO FINGIDO**: falha do executor é registrada como falha, com o erro real;
//  4. **ESCOPO SEMPRE DA SESSÃO**: toda execução usa a organização do turno, nunca um campo do pedido.

// Efeitos possíveis de uma ferramenta. O efeito decide a política aplicada.
const (
	ServiceAgentEffectRead     = "read"
	ServiceAgentEffectWrite    = "write"
	ServiceAgentEffectOutbound = "outbound"
)

// Estados de uma chamada de ferramenta no turno.
const (
	// ServiceAgentToolExecuted: executou e devolveu resultado real.
	ServiceAgentToolExecuted = "executed"
	// ServiceAgentToolBlocked: o agente não declara a ferramenta (privilégio) ou a política barrou.
	ServiceAgentToolBlocked = "blocked"
	// ServiceAgentToolUnavailable: declarada, mas NINGUÉM implementou — motivo explícito.
	ServiceAgentToolUnavailable = "unavailable"
	// ServiceAgentToolFailed: o executor falhou; o erro real é preservado.
	ServiceAgentToolFailed = "failed"
	// ServiceAgentToolApprovalRequired: a escrita NÃO rodou — aguarda decisão humana. É distinto de
	// `blocked` (privilégio/política): aqui o caminho existe e o que falta é autorização.
	ServiceAgentToolApprovalRequired = "approval_required"
)

// ServiceAgentToolRequest é o insumo de uma execução.
type ServiceAgentToolRequest struct {
	OrganizationID string
	AgentID        string
	ConversationID string
	ContactID      string
	Channel        string
	// Intent é o texto do contato (dado de terceiro, tratado como não confiável).
	Intent string
	// Title/ValueCents/OwnerID/Assignee/Reason alimentam as ferramentas de escrita.
	Title      string
	ValueCents int64
	OwnerID    string
	Assignee   string
	Reason     string
	Now        time.Time
}

// ServiceAgentToolCall é o REGISTRO de uma chamada no turno: o que foi tentado, o que aconteceu e por quê.
type ServiceAgentToolCall struct {
	Name       string    `json:"name"`
	Effect     string    `json:"effect"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
	// ApprovalID identifica a aprovação humana que autorizou a escrita — ou a que ainda falta.
	ApprovalID string `json:"approval_id,omitempty"`
}

// ServiceAgentTool é uma ferramenta executável de verdade.
type ServiceAgentTool interface {
	Name() string
	Effect() string
	Execute(request ServiceAgentToolRequest) (string, error)
}

// serviceAgentToolFunc permite declarar um executor por função, mantendo o nome e o efeito explícitos.
type serviceAgentToolFunc struct {
	name   string
	effect string
	run    func(request ServiceAgentToolRequest) (string, error)
}

func (t serviceAgentToolFunc) Name() string   { return t.name }
func (t serviceAgentToolFunc) Effect() string { return t.effect }
func (t serviceAgentToolFunc) Execute(request ServiceAgentToolRequest) (string, error) {
	return t.run(request)
}

// serviceAgentTools constrói os executores REAIS ligados aos stores já existentes.
//
// Cada um devolve um resumo com dado do store — não texto inventado. Os que exigem módulos que este
// runtime não recebe (`commerce.*`, `company.report.read`) **não** aparecem aqui de propósito: os três
// passam a ser reportados como indisponíveis, que é a verdade.
func serviceAgentTools(crm *CRMStore, inbox *InboxStore, consent *ConsentRegistry) map[string]ServiceAgentTool {
	tools := map[string]ServiceAgentTool{}
	add := func(name, effect string, run func(request ServiceAgentToolRequest) (string, error)) {
		tools[name] = serviceAgentToolFunc{name: name, effect: effect, run: run}
	}

	add("crm.contact.read", ServiceAgentEffectRead, func(request ServiceAgentToolRequest) (string, error) {
		contact, err := resolveServiceAgentContact(crm, request.OrganizationID, request.ContactID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("contato %s (%s), telefone %s", contact.Name, contact.ID, contact.Phone), nil
	})

	add("crm.opportunity.write", ServiceAgentEffectWrite, func(request ServiceAgentToolRequest) (string, error) {
		if crm == nil {
			return "", fmt.Errorf("%w: CRM indisponível", ErrServiceAgentTurnInvalid)
		}
		title := strings.TrimSpace(request.Title)
		if title == "" {
			return "", fmt.Errorf("%w: title é obrigatório para crm.opportunity.write", ErrServiceAgentTurnInvalid)
		}
		contact, err := resolveServiceAgentContact(crm, request.OrganizationID, request.ContactID)
		if err != nil {
			return "", err
		}
		opportunity, err := crm.CreateOpportunity(request.OrganizationID, contact.ID, title, request.ValueCents, request.OwnerID, request.Now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("oportunidade %s criada (%s, %d centavos)", opportunity.ID, opportunity.Stage, opportunity.ValueCents), nil
	})

	add("inbox.conversation.read", ServiceAgentEffectRead, func(request ServiceAgentToolRequest) (string, error) {
		if inbox == nil {
			return "", fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
		}
		conversation, history, err := inbox.Conversation(request.OrganizationID, request.ConversationID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("conversa %s (%s) com %d mensagem(ns)", conversation.ID, conversation.Status, len(history)), nil
	})

	add("inbox.reply", ServiceAgentEffectOutbound, func(request ServiceAgentToolRequest) (string, error) {
		if inbox == nil {
			return "", fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
		}
		body := strings.TrimSpace(request.Intent)
		if body == "" {
			return "", fmt.Errorf("%w: texto da resposta é obrigatório para inbox.reply", ErrServiceAgentTurnInvalid)
		}
		_, message, err := inbox.AppendOutbound(request.OrganizationID, request.ConversationID, "agent", request.AgentID, body, consent, request.Now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("resposta %s registrada na conversa", message.ID), nil
	})

	add("inbox.handoff", ServiceAgentEffectWrite, func(request ServiceAgentToolRequest) (string, error) {
		if inbox == nil {
			return "", fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
		}
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			reason = "transferência solicitada pelo agente"
		}
		assignee := strings.TrimSpace(request.Assignee)
		if assignee == "" {
			assignee = "humano"
		}
		_, message, err := inbox.HandoffToHuman(request.OrganizationID, request.ConversationID, request.AgentID, assignee, reason, request.Now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("transferência %s registrada para %s", message.ID, assignee), nil
	})

	add("inbox.assign", ServiceAgentEffectWrite, func(request ServiceAgentToolRequest) (string, error) {
		if inbox == nil {
			return "", fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
		}
		assignee := strings.TrimSpace(request.Assignee)
		if assignee == "" {
			return "", fmt.Errorf("%w: assignee é obrigatório para inbox.assign", ErrServiceAgentTurnInvalid)
		}
		conversation, err := inbox.Assign(request.OrganizationID, request.ConversationID, assignee, "agent", request.Now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("conversa %s atribuída a %s", conversation.ID, assignee), nil
	})

	add("inbox.metrics.read", ServiceAgentEffectRead, func(request ServiceAgentToolRequest) (string, error) {
		if inbox == nil {
			return "", fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
		}
		metrics := inbox.Metrics(request.OrganizationID, request.Now)
		return fmt.Sprintf("métricas da caixa: %d conversa(s), %d aberta(s), %d violação(ões) de SLA",
			metrics.TotalConversations, metrics.OpenConversations, metrics.SLABreached), nil
	})

	return tools
}

// RunTool executa UMA ferramenta sob as regras do turno.
//
// A ordem é deliberada e cada passo tem motivo:
//  1. o agente declara a ferramenta? (senão: bloqueada — privilégio não sobe por a ferramenta existir);
//  2. existe executor registrado? (senão: indisponível — e a resposta é esta, não um sucesso inventado);
//  3. a política permite o efeito? (saída exige consentimento concedido);
//  4. EFEITO exige APROVAÇÃO HUMANA: sem aprovação válida, o pedido é registrado e NADA é executado;
//  5. executa; falha vira FALHA com o erro real.
func (s *ServiceAgentRuntime) RunTool(spec ServiceAgentSpec, declared []string, effectOutboundAllowed bool, toolName string, request ServiceAgentToolRequest) ServiceAgentToolCall {
	name := strings.TrimSpace(toolName)
	call := ServiceAgentToolCall{Name: name, ExecutedAt: request.Now}

	if !containsToolName(spec.Tools, name) && !containsToolName(declared, name) {
		call.Status = ServiceAgentToolBlocked
		call.Reason = "o agente não declara esta ferramenta"
		return call
	}
	tool, ok := s.executors[name]
	if !ok {
		call.Status = ServiceAgentToolUnavailable
		call.Reason = "nenhum executor registrado para esta ferramenta neste runtime"
		return call
	}
	call.Effect = tool.Effect()
	if tool.Effect() == ServiceAgentEffectOutbound && !effectOutboundAllowed {
		call.Status = ServiceAgentToolBlocked
		call.Reason = "ferramenta de saída exige consentimento concedido"
		return call
	}
	// Passo 4: ESCRITA não acontece por conta de uma frase do cliente.
	if tool.Effect() == ServiceAgentEffectWrite {
		return s.authorizeWrite(call, request, name)
	}
	summary, err := tool.Execute(request)
	if err != nil {
		call.Status = ServiceAgentToolFailed
		// O erro REAL é preservado: quem for investigar precisa do motivo, não de "erro genérico".
		call.Reason = err.Error()
		return call
	}
	call.Status = ServiceAgentToolExecuted
	call.Summary = summary
	return call
}

// authorizeWrite aplica a aprovação humana às ferramentas de efeito.
//
// Sem store de aprovações configurado, a escrita é RECUSADA — não liberada. O padrão seguro é negar: um
// runtime que não sabe pedir aprovação não pode executar efeito.
func (s *ServiceAgentRuntime) authorizeWrite(call ServiceAgentToolCall, request ServiceAgentToolRequest, toolName string) ServiceAgentToolCall {
	s.mu.Lock()
	store := s.approvals
	s.mu.Unlock()
	if store == nil {
		call.Status = ServiceAgentToolBlocked
		call.Reason = "escrita exige aprovação humana e este runtime não tem store de aprovações"
		return call
	}
	scope := consentScope(request.OrganizationID)
	fingerprint := serviceAgentApprovalFingerprint(request, toolName)

	// Já negado por um humano? Então a resposta é a recusa, com quem negou — e não um novo pedido.
	if denied, found, err := store.FindDenied(scope, fingerprint); err == nil && found {
		call.Status = ServiceAgentToolBlocked
		call.Reason = fmt.Sprintf("aprovação %s NEGADA por %s: %s", denied.ID, denied.DecidedBy, denied.Reason)
		return call
	}
	approved, found, err := store.FindApproved(scope, fingerprint)
	if err != nil {
		call.Status = ServiceAgentToolFailed
		call.Reason = err.Error()
		return call
	}
	if !found {
		pending, err := store.Request(scope, ServiceAgentApproval{
			OrganizationID: scope,
			AgentID:        request.AgentID,
			ConversationID: request.ConversationID,
			ToolName:       toolName,
			Fingerprint:    fingerprint,
			RequestedAt:    request.Now.UTC(),
			RequestedBy:    "agente:" + request.AgentID,
		})
		if err != nil {
			call.Status = ServiceAgentToolFailed
			call.Reason = err.Error()
			return call
		}
		call.Status = ServiceAgentToolApprovalRequired
		call.ApprovalID = pending.ID
		call.Reason = fmt.Sprintf("escrita exige aprovação humana: pedido %s aguardando decisão", pending.ID)
		return call
	}
	// Uso ÚNICO: consumir ANTES de executar. Consumir depois deixaria a janela em que duas execuções usam
	// a mesma aprovação — e uma autorização pontual viraria permanente por acidente.
	if _, err := store.Consume(scope, approved.ID, request.Now); err != nil {
		call.Status = ServiceAgentToolFailed
		call.Reason = err.Error()
		return call
	}
	call.ApprovalID = approved.ID
	summary, err := s.executors[toolName].Execute(request)
	if err != nil {
		call.Status = ServiceAgentToolFailed
		call.Reason = err.Error()
		return call
	}
	call.Status = ServiceAgentToolExecuted
	call.Summary = summary
	return call
}

// serviceAgentCommerceTools são as ferramentas de comércio — e elas só EXISTEM quando o store é ligado.
//
// Antes, `commerce.order.read` e `commerce.stock.read` constavam como disponíveis num mapa digitado à mão,
// sem executor algum. Aqui a disponibilidade volta a ser consequência: sem store ligado não há executor, e a
// ferramenta aparece como indisponível — que é a verdade.
func serviceAgentCommerceTools(commerce *CommerceStore) map[string]ServiceAgentTool {
	tools := map[string]ServiceAgentTool{}
	if commerce == nil {
		return tools
	}
	add := func(name string, run func(request ServiceAgentToolRequest) (string, error)) {
		tools[name] = serviceAgentToolFunc{name: name, effect: ServiceAgentEffectRead, run: run}
	}

	add("catalog.read", func(request ServiceAgentToolRequest) (string, error) {
		products := commerce.Products(request.OrganizationID)
		if len(products) == 0 {
			// Honestidade: organização sem catálogo DIZ que não tem, em vez de inventar oferta.
			return "nenhum produto cadastrado para esta organização", nil
		}
		parts := make([]string, 0, 3)
		for _, product := range products {
			if len(parts) == 3 {
				break
			}
			parts = append(parts, fmt.Sprintf("%s (%s) por %d centavos", product.Name, product.SKU, product.PriceCents))
		}
		return fmt.Sprintf("%d produto(s) ativos: %s", len(products), strings.Join(parts, "; ")), nil
	})

	add("commerce.stock.read", func(request ServiceAgentToolRequest) (string, error) {
		summary := commerce.Summary(request.OrganizationID)
		low := "nenhum"
		if len(summary.LowStockSKUs) > 0 {
			low = strings.Join(summary.LowStockSKUs, ", ")
		}
		return fmt.Sprintf("catálogo com %d produto(s); estoque baixo: %s", summary.Products, low), nil
	})

	add("commerce.order.read", func(request ServiceAgentToolRequest) (string, error) {
		summary := commerce.Summary(request.OrganizationID)
		return fmt.Sprintf("%d pedido(s): %d pendente(s), %d pago(s), %d entregue(s), %d cancelado(s); receita %d centavos",
			summary.Orders, summary.Pending, summary.Paid, summary.Fulfilled, summary.Cancelled, summary.RevenueCents), nil
	})

	return tools
}

// serviceAgentCompanyTools são as ferramentas de empresa — também condicionadas ao store ligado.
func serviceAgentCompanyTools(companies *CompanyStore) map[string]ServiceAgentTool {
	tools := map[string]ServiceAgentTool{}
	if companies == nil {
		return tools
	}
	tools["company.report.read"] = serviceAgentToolFunc{
		name:   "company.report.read",
		effect: ServiceAgentEffectRead,
		run: func(request ServiceAgentToolRequest) (string, error) {
			list := companies.List(request.OrganizationID)
			if len(list) == 0 {
				return "nenhuma empresa cadastrada para esta organização", nil
			}
			names := make([]string, 0, 3)
			for _, company := range list {
				if len(names) == 3 {
					break
				}
				names = append(names, fmt.Sprintf("%s (%s)", company.Name, company.Currency))
			}
			return fmt.Sprintf("%d empresa(s): %s", len(list), strings.Join(names, "; ")), nil
		},
	}
	return tools
}

// resolveServiceAgentContact acha o contato a partir do que a CONVERSA carrega.
//
// A conversa guarda o TELEFONE (identidade), não o ID interno do CRM — então procurar só por ID falharia
// em todo atendimento real. A resolução repete a mesma regra usada para montar o contexto do turno, para
// que ferramenta e contexto não discordem sobre quem é o contato.
func resolveServiceAgentContact(crm *CRMStore, organizationID, contactID string) (*CRMContact, error) {
	if crm == nil {
		return nil, fmt.Errorf("%w: CRM indisponível", ErrServiceAgentTurnInvalid)
	}
	contactID = strings.TrimSpace(contactID)
	if contactID == "" {
		return nil, fmt.Errorf("%w: contato é obrigatório para esta ferramenta", ErrServiceAgentTurnInvalid)
	}
	if contact, err := crm.GetContact(organizationID, contactID); err == nil {
		return &contact, nil
	}
	for _, contact := range crm.ListContacts(organizationID, "") {
		candidate := contact
		if crmIdentity(candidate.Phone, candidate.Email) == crmIdentity(contactID, "") && crmIdentity(contactID, "") != "" {
			return &candidate, nil
		}
		if strings.EqualFold(strings.TrimSpace(candidate.Email), contactID) {
			return &candidate, nil
		}
	}
	return nil, fmt.Errorf("%w: contato %s não encontrado para esta organização", ErrServiceAgentTurnInvalid, contactID)
}

func containsToolName(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// ErrServiceAgentToolNotFound marca ferramenta pedida que não existe no catálogo declarado.
var ErrServiceAgentToolNotFound = errors.New("service agent tool not found")
