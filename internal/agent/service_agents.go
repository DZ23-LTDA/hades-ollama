package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Agentes de atendimento e vendas (estágio 35).
//
// Sete agentes especializados reutilizáveis que **usam** o que já existe: CRM,
// caixa de entrada, registro de consentimento e DLP. O que o estágio exige e este
// código entrega:
//   - **reconhecer empresa, contato, canal e missão sem acessar não autorizado**;
//   - **políticas aplicadas mesmo trocando de modelo**;
//   - **continuidade** humano/IA com histórico e contexto.
//
// DECISÕES DELIBERADAS:
//   - **O texto NÃO é fingido como IA**: cada turno declara a `ReplySource`. Sem um
//     modelo conectado o agente usa um **modelo de resposta determinístico** e diz
//     isso; apresentar template como "resposta da IA" seria mentir para o usuário;
//   - **Política é verificada ANTES do texto**, e continua sendo verificada em
//     qualquer troca de modelo: trocar de modelo **não** pode remover consentimento
//     nem DLP;
//   - **Sem consentimento, o agente ESCALA em vez de responder**: a decisão correta
//     não é "manda mesmo assim", é passar para uma pessoa;
//   - **Ferramentas são limitadas por agente**: um agente não ganha ferramenta que não
//     está na lista dele, e ferramenta desconhecida é registrada como bloqueada;
//   - **Acesso é escopado por organização** em todas as leituras (CRM, inbox,
//     consentimento).

const (
	ServiceAgentReceptionist = "receptionist"
	ServiceAgentSales        = "sales"
	ServiceAgentSupport      = "support"
	ServiceAgentSocial       = "social"
	ServiceAgentVoice        = "voice"
	ServiceAgentCommerce     = "commerce"
	ServiceAgentSupervisor   = "supervisor"

	// Procedência do texto do turno. Nunca inferir: declarar.
	ReplySourceTemplate = "template-deterministico"
	ReplySourceModel    = "model"

	// Handoff automático quando a política impede responder.
	ServiceAgentEscalateConsent = "consentimento ausente ou revogado"
	ServiceAgentEscalateDLP     = "conteúdo bloqueado pela política de DLP"
	ServiceAgentEscalateHuman   = "o contato pediu atendimento humano"
)

var (
	ErrServiceAgentNotFound    = errors.New("service agent not found")
	ErrServiceAgentTurnInvalid = errors.New("service agent turn invalid")
)

// ServiceAgentSpec descreve um agente reutilizável.
type ServiceAgentSpec struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Objective       string   `json:"objective"`
	Channels        []string `json:"channels"`
	Tools           []string `json:"tools"`
	EscalationRules []string `json:"escalation_rules"`
	MaxTurns        int      `json:"max_turns"`
}

// ServiceAgentCatalog devolve os agentes do estágio 35.
func ServiceAgentCatalog() []ServiceAgentSpec {
	specs := []ServiceAgentSpec{
		{
			ID: ServiceAgentReceptionist, Name: "Recepcionista IA",
			Objective:       "Receber o contato, identificar a necessidade e encaminhar para a área certa",
			Channels:        []string{InboxChannelWhatsApp, InboxChannelTelegram, InboxChannelWebchat, InboxChannelPhone},
			Tools:           []string{"crm.contact.read", "inbox.conversation.read", "inbox.reply", "inbox.handoff"},
			EscalationRules: []string{ServiceAgentEscalateConsent, ServiceAgentEscalateHuman}, MaxTurns: 20,
		},
		{
			ID: ServiceAgentSales, Name: "Agente de Vendas IA",
			Objective:       "Qualificar a oportunidade, apresentar a oferta e registrar o próximo passo",
			Channels:        []string{InboxChannelWhatsApp, InboxChannelTelegram, InboxChannelWebchat},
			Tools:           []string{"crm.contact.read", "crm.opportunity.write", "catalog.read", "inbox.reply", "inbox.handoff"},
			EscalationRules: []string{ServiceAgentEscalateConsent, ServiceAgentEscalateHuman, "desconto fora da política"}, MaxTurns: 30,
		},
		{
			ID: ServiceAgentSupport, Name: "Agente de Suporte IA",
			Objective:       "Resolver dúvidas e falhas de primeiro nível com histórico do contato",
			Channels:        []string{InboxChannelWhatsApp, InboxChannelTelegram, InboxChannelEmail, InboxChannelWebchat},
			Tools:           []string{"crm.contact.read", "inbox.conversation.read", "inbox.reply", "inbox.handoff", "knowledge.search"},
			EscalationRules: []string{ServiceAgentEscalateConsent, ServiceAgentEscalateHuman, "falha grave em produção"}, MaxTurns: 40,
		},
		{
			ID: ServiceAgentSocial, Name: "Agente de Redes Sociais IA",
			Objective:       "Responder e acompanhar interações públicas com aprovação",
			Channels:        []string{InboxChannelInstagram, InboxChannelMessenger},
			Tools:           []string{"crm.contact.read", "social.draft", "inbox.reply"},
			EscalationRules: []string{ServiceAgentEscalateConsent, "crise de marca"}, MaxTurns: 20,
		},
		{
			ID: ServiceAgentVoice, Name: "Agente de Voz IA",
			Objective:       "Atender chamadas receptivas com transferência para humano quando preciso",
			Channels:        []string{InboxChannelPhone},
			Tools:           []string{"crm.contact.read", "inbox.reply", "inbox.handoff", "voice.transfer"},
			EscalationRules: []string{ServiceAgentEscalateConsent, ServiceAgentEscalateHuman, "pedido de gravação"}, MaxTurns: 25,
		},
		{
			ID: ServiceAgentCommerce, Name: "Agente de Comércio IA",
			Objective:       "Acompanhar pedido, estoque e entrega com dados reais da empresa",
			Channels:        []string{InboxChannelWhatsApp, InboxChannelTelegram, InboxChannelWebchat},
			Tools:           []string{"crm.contact.read", "commerce.order.read", "commerce.stock.read", "inbox.reply", "inbox.handoff"},
			EscalationRules: []string{ServiceAgentEscalateConsent, "pagamento contestado"}, MaxTurns: 30,
		},
		{
			ID: ServiceAgentSupervisor, Name: "Supervisor de Atendimento IA",
			Objective:       "Acompanhar filas, SLA e qualidade e escalar o que estoura o limite",
			Channels:        []string{InboxChannelWhatsApp, InboxChannelTelegram, InboxChannelEmail, InboxChannelWebchat, InboxChannelPhone},
			Tools:           []string{"inbox.metrics.read", "inbox.assign", "inbox.handoff", "company.report.read"},
			EscalationRules: []string{"SLA violado", "fila humana acima do limite"}, MaxTurns: 50,
		},
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return specs
}

// ServiceAgentSpecByID procura um agente pelo identificador.
func ServiceAgentSpecByID(id string) (ServiceAgentSpec, bool) {
	for _, spec := range ServiceAgentCatalog() {
		if spec.ID == strings.TrimSpace(id) {
			return spec, true
		}
	}
	return ServiceAgentSpec{}, false
}

// ServiceAgentContext é o contexto REAL montado para o turno — o que o modelo
// receberia. Montá-lo explicitamente torna verificável o que foi usado.
type ServiceAgentContext struct {
	OrganizationID string           `json:"organization_id"`
	AgentID        string           `json:"agent_id"`
	ConversationID string           `json:"conversation_id"`
	Channel        string           `json:"channel"`
	Contact        *CRMContact      `json:"contact,omitempty"`
	Opportunities  []CRMOpportunity `json:"opportunities,omitempty"`
	History        []InboxMessage   `json:"history,omitempty"`
	ConsentState   string           `json:"consent_state"`
	AllowedTools   []string         `json:"allowed_tools"`
	BlockedTools   []string         `json:"blocked_tools,omitempty"`
	Policies       []string         `json:"policies"`
	ModelID        string           `json:"model_id,omitempty"`
}

// ServiceAgentTurn é o resultado do turno, com a decisão e a PROCEDÊNCIA do texto.
type ServiceAgentTurn struct {
	AgentID     string              `json:"agent_id"`
	Decision    string              `json:"decision"`
	ReplySource string              `json:"reply_source"`
	Reply       string              `json:"reply,omitempty"`
	Escalated   bool                `json:"escalated"`
	Escalation  string              `json:"escalation_reason,omitempty"`
	Context     ServiceAgentContext `json:"context"`
	Message     *InboxMessage       `json:"message,omitempty"`
	DecidedAt   time.Time           `json:"decided_at"`
}

// ServiceAgentTurnRequest é a entrada do turno: quem fala, onde, o que o contato
// trouxe e qual modelo foi escolhido.
type ServiceAgentTurnRequest struct {
	OrganizationID string
	AgentID        string
	ConversationID string
	// Intent é o texto que o contato/envio trouxe (dado de terceiro).
	Intent string
	// ModelID é o modelo escolhido; trocar de modelo NÃO altera as políticas.
	ModelID string
}

// RunTurn executa o turno do agente: monta contexto, verifica política, decide e —
// se permitido — responde. Nunca finge procedência de modelo.
func (s *ServiceAgentRuntime) RunTurn(request ServiceAgentTurnRequest, now time.Time) (ServiceAgentTurn, error) {
	spec, ok := ServiceAgentSpecByID(request.AgentID)
	if !ok {
		return ServiceAgentTurn{}, fmt.Errorf("%w: %s", ErrServiceAgentNotFound, request.AgentID)
	}
	organizationID := consentScope(request.OrganizationID)
	turn := ServiceAgentTurn{AgentID: spec.ID, DecidedAt: now, ReplySource: ReplySourceTemplate}
	turn.Context = ServiceAgentContext{
		OrganizationID: organizationID,
		AgentID:        spec.ID,
		ConversationID: strings.TrimSpace(request.ConversationID),
		AllowedTools:   append([]string(nil), spec.Tools...),
		ModelID:        strings.TrimSpace(request.ModelID),
		Policies:       []string{"consentimento obrigatório no envio", "DLP antes do egress", "ferramentas limitadas por agente"},
	}
	blocked, allowed := classifyServiceAgentTools(spec.Tools, s.availableTools())
	turn.Context.BlockedTools = blocked
	turn.Context.AllowedTools = allowed

	if s.inbox == nil {
		return ServiceAgentTurn{}, fmt.Errorf("%w: caixa de entrada indisponível", ErrServiceAgentTurnInvalid)
	}
	conversation, history, err := s.inbox.Conversation(organizationID, request.ConversationID)
	if err != nil {
		return ServiceAgentTurn{}, err
	}
	turn.Context.Channel = conversation.Channel
	turn.Context.History = history
	turn.Context.Contact = s.contactFor(organizationID, conversation.Channel, conversation.ContactID)
	if turn.Context.Contact != nil {
		turn.Context.Opportunities = s.opportunitiesFor(organizationID, turn.Context.Contact.ID)
	}

	// 1. O contato pediu humano? Escalar é a resposta CORRETA, não responder.
	if asksForHuman(request.Intent) {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateHuman
		turn.Context.ConsentState = s.consentState(organizationID, conversation.Channel, conversation.ContactID, now)
		return turn, nil
	}

	// 2. Consentimento: sem ele o agente NÃO responde — escala.
	consentState := s.consentState(organizationID, conversation.Channel, conversation.ContactID, now)
	turn.Context.ConsentState = consentState
	if consentState != ConsentGranted {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateConsent
		return turn, nil
	}

	// 3. DLP: conteúdo bloqueado não sai, e o motivo é declarado. A pergunta é de
	// POLÍTICA ("este texto pode sair?"), por isso a resposta é booleana: bloquear não
	// é falha do servidor, é decisão que vira escalada.
	reply := s.composeReply(spec, turn.Context)
	if !outboundTextAllowed(reply) {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateDLP
		return turn, nil
	}

	// 4. Responder: a mensagem entra pela MESMA caixa de entrada, que revalida o
	// consentimento — a política não depende do caminho.
	_, message, err := s.inbox.AppendOutbound(organizationID, conversation.ID, "agent", spec.ID, reply, s.consent, now)
	if err != nil {
		if errors.Is(err, ErrInboxConsentRequired) {
			turn.Decision = "escalar"
			turn.Escalated = true
			turn.Escalation = ServiceAgentEscalateConsent
			return turn, nil
		}
		return ServiceAgentTurn{}, err
	}
	turn.Decision = "responder"
	turn.Reply = reply
	turn.Message = &message
	return turn, nil
}

// classifyServiceAgentTools separa as ferramentas que o agente PODE usar das que o
// runtime bloqueia (não registradas ou fora da lista do agente).
func classifyServiceAgentTools(specTools []string, available map[string]bool) ([]string, []string) {
	blocked := make([]string, 0)
	allowed := make([]string, 0, len(specTools))
	for _, tool := range specTools {
		if available[tool] {
			allowed = append(allowed, tool)
			continue
		}
		blocked = append(blocked, tool)
	}
	return blocked, allowed
}

func (s *ServiceAgentRuntime) contactFor(organizationID, channel, contactID string) *CRMContact {
	if s.crm == nil {
		return nil
	}
	for _, contact := range s.crm.ListContacts(organizationID, "") {
		if crmIdentity(contact.Phone, contact.Email) == crmIdentity(contactID, "") && crmIdentity(contactID, "") != "" {
			candidate := contact
			return &candidate
		}
		if strings.EqualFold(strings.TrimSpace(contact.Email), strings.TrimSpace(contactID)) {
			candidate := contact
			return &candidate
		}
	}
	return nil
}

func (s *ServiceAgentRuntime) opportunitiesFor(organizationID, contactID string) []CRMOpportunity {
	if s.crm == nil {
		return nil
	}
	out := make([]CRMOpportunity, 0)
	for _, opportunity := range s.crm.ListOpportunities(organizationID, "") {
		if opportunity.ContactID == contactID {
			out = append(out, opportunity)
		}
	}
	return out
}

func (s *ServiceAgentRuntime) consentState(organizationID, channel, contactID string, now time.Time) string {
	if s.consent == nil {
		return ConsentRevoked
	}
	if err := s.consent.AuthorizeContact(organizationID, channel, contactID, now); err != nil {
		switch {
		case errors.Is(err, ErrConsentRevoked):
			return ConsentRevoked
		default:
			return "ausente"
		}
	}
	return ConsentGranted
}

// composeReply monta a resposta. Sem modelo conectado, o texto é um MODELO
// determinístico — e o turno declara isso em `ReplySource`, em vez de apresentar
// template como resposta de IA.
func (s *ServiceAgentRuntime) composeReply(spec ServiceAgentSpec, context ServiceAgentContext) string {
	name := ""
	if context.Contact != nil {
		name = strings.TrimSpace(context.Contact.Name)
	}
	greeting := "Olá"
	if name != "" {
		greeting = "Olá, " + name
	}
	history := len(context.History)
	switch spec.ID {
	case ServiceAgentSales:
		if len(context.Opportunities) > 0 {
			return fmt.Sprintf("%s! Recebi seu contato sobre %q e já tenho o histórico desta conversa (%d mensagens). Vou preparar os próximos passos da proposta.",
				greeting, context.Opportunities[0].Title, history)
		}
		return fmt.Sprintf("%s! Obrigado pelo contato. Vou entender sua necessidade e montar a proposta (%d mensagens no histórico).", greeting, history)
	case ServiceAgentCommerce:
		return fmt.Sprintf("%s! Vou verificar seu pedido, estoque e entrega com os dados da empresa.", greeting)
	case ServiceAgentSupervisor:
		return "Acompanhando filas e SLA do atendimento; vou escalar o que estourar o limite."
	default:
		return fmt.Sprintf("%s! Recebi sua mensagem e vou ajudar. Este é o atendimento de %s.", greeting, spec.Objective)
	}
}

// outboundTextAllowed responde se o texto pode sair segundo a política de DLP.
func outboundTextAllowed(text string) bool {
	return validateOutboundPayloadWithLimit(map[string]any{"text": text}, telegramMaxPayloadBytes) == nil
}

// asksForHuman reconhece o pedido explícito de atendimento humano.
func asksForHuman(intent string) bool {
	normalized := strings.ToLower(strings.TrimSpace(intent))
	for _, marker := range []string{"falar com humano", "falar com uma pessoa", "atendente", "human agent", "pessoa de verdade", "supervisor", "gerente"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// ServiceAgentRuntime liga os agentes aos recursos existentes.
type ServiceAgentRuntime struct {
	crm     *CRMStore
	inbox   *InboxStore
	consent *ConsentRegistry
	// tools são as ferramentas REALMENTE registradas. Ferramenta que o agente lista e
	// o runtime não tem aparece como bloqueada, em vez de ser ignorada.
	tools map[string]bool
}

// NewServiceAgentRuntime liga os agentes ao CRM, à caixa de entrada e ao consentimento.
func NewServiceAgentRuntime(crm *CRMStore, inbox *InboxStore, consent *ConsentRegistry, tools map[string]bool) *ServiceAgentRuntime {
	if tools == nil {
		tools = map[string]bool{}
	}
	return &ServiceAgentRuntime{crm: crm, inbox: inbox, consent: consent, tools: tools}
}

func (s *ServiceAgentRuntime) availableTools() map[string]bool {
	return s.tools
}

// RegisterTool registra uma ferramenta disponível para os agentes.
func (s *ServiceAgentRuntime) RegisterTool(name string) {
	if s.tools == nil {
		s.tools = map[string]bool{}
	}
	s.tools[strings.TrimSpace(name)] = true
}

// DefaultServiceAgentTools lista as ferramentas REALMENTE implementadas hoje. As que
// os agentes declaram e não estão aqui aparecem como **bloqueadas** no contexto do
// turno — informação honesta, em vez de fingir capacidade.
func DefaultServiceAgentTools() map[string]bool {
	return map[string]bool{
		"crm.contact.read":        true,
		"crm.opportunity.write":   true,
		"inbox.conversation.read": true,
		"inbox.reply":             true,
		"inbox.handoff":           true,
		"inbox.metrics.read":      true,
		"inbox.assign":            true,
		"company.report.read":     true,
	}
}
