package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	// ToolCalls é a trilha de execução: o que foi tentado, o que rodou e por que o resto não rodou.
	ToolCalls []ServiceAgentToolCall `json:"tool_calls,omitempty"`
	// AuditError declara que a trilha DURÁVEL não pôde ser gravada. A ausência de auditoria precisa ser
	// visível: perder registro em silêncio é o pior desfecho para quem investiga um incidente.
	AuditError string    `json:"audit_error,omitempty"`
	DecidedAt  time.Time `json:"decided_at"`
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
	// RequestedTools são as ferramentas que o agente PEDE para este turno. Pedir não é poder: o runtime
	// confere declaração, executor e política. Sem pedido explícito, nenhuma ferramenta de ESCRITA ou
	// SAÍDA roda — efeito secundário não acontece por acidente.
	RequestedTools []string
	// ToolArguments alimenta as ferramentas de escrita (título, valor, responsável…).
	ToolArguments ServiceAgentToolRequest
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

	// 1. Consentimento é calculado UMA vez e cedo: ele governa tanto a escalada quanto as ferramentas de
	// saída. Calculá-lo antes das decisões evita que o mesmo estado seja lido com regras diferentes.
	consentState := s.consentState(organizationID, conversation.Channel, conversation.ContactID, now)
	turn.Context.ConsentState = consentState

	// 2. FERRAMENTAS PEDIDAS: pedir não é poder. O runtime confere declaração, executor e política, e
	// registra TUDO na trilha do turno — inclusive o que não rodou e por quê. Roda antes das decisões
	// para que transferência e atribuição funcionem também quando o turno termina em escalada.
	// Contato e canal vêm da CONVERSA resolvida: é o escopo da sessão, não o que o pedido mandou.
	turn.ToolCalls = s.runRequestedTools(request, conversation.ContactID, conversation.Channel, consentState == ConsentGranted, now)
	// A trilha DURÁVEL é gravada AQUI, num único ponto: qualquer caminho de saída do turno (escalada,
	// resposta, falha) já passou por este registro — auditoria não pode depender de qual caminho seguiu.
	turn.AuditError = s.recordAudit(organizationID, spec.ID, conversation.ID, request.ModelID, turn.ToolCalls, now)

	// 3. O contato pediu humano? Escalar é a resposta CORRETA, não responder.
	if asksForHuman(request.Intent) {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateHuman
		return turn, nil
	}

	// 4. Consentimento: sem ele o agente NÃO responde — escala.
	if consentState != ConsentGranted {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateConsent
		return turn, nil
	}

	// 5. DLP: conteúdo bloqueado não sai, e o motivo é declarado. A pergunta é de
	// POLÍTICA ("este texto pode sair?"), por isso a resposta é booleana: bloquear não
	// é falha do servidor, é decisão que vira escalada.
	reply := s.composeReply(spec, turn.Context)
	if !outboundTextAllowed(reply) {
		turn.Decision = "escalar"
		turn.Escalated = true
		turn.Escalation = ServiceAgentEscalateDLP
		return turn, nil
	}

	// 6. Responder: a mensagem entra pela MESMA caixa de entrada, que revalida o
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

// runRequestedTools executa as ferramentas PEDIDAS, registrando cada tentativa.
//
// Decisões:
//   - **O ESCOPO VEM DA SESSÃO**: organização, agente, conversa, contato e canal são copiados do turno.
//     `ToolArguments` só fornece dados de domínio (título, valor, responsável, motivo) — assim o pedido
//     não consegue escolher outra organização;
//   - **`inbox.reply` é recusado aqui**, com motivo: a resposta do turno já sai pelo caminho único de
//     resposta, e permitir a ferramenta criaria uma SEGUNDA mensagem para o mesmo turno;
//   - **pedido repetido não duplica execução**: executar o efeito duas vezes por engano é pior que
//     ignorar a repetição.
func (s *ServiceAgentRuntime) runRequestedTools(request ServiceAgentTurnRequest, contactID, channel string, consentGranted bool, now time.Time) []ServiceAgentToolCall {
	if len(request.RequestedTools) == 0 {
		return nil
	}
	spec, ok := ServiceAgentSpecByID(request.AgentID)
	if !ok {
		return nil
	}
	organizationID := consentScope(request.OrganizationID)
	seen := map[string]bool{}
	calls := make([]ServiceAgentToolCall, 0, len(request.RequestedTools))
	for _, name := range request.RequestedTools {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if name == "inbox.reply" {
			calls = append(calls, ServiceAgentToolCall{
				Name:       name,
				Effect:     ServiceAgentEffectOutbound,
				Status:     ServiceAgentToolBlocked,
				Reason:     "a resposta do turno já é enviada pelo caminho único de resposta",
				ExecutedAt: now,
			})
			continue
		}
		arguments := request.ToolArguments
		// Escopo da sessão sobrepõe o que veio no pedido — organização, agente, conversa, CONTATO e canal
		// são do turno resolvido, nunca do corpo do pedido.
		arguments.OrganizationID = organizationID
		arguments.AgentID = spec.ID
		arguments.ConversationID = strings.TrimSpace(request.ConversationID)
		arguments.ContactID = contactID
		arguments.Channel = channel
		arguments.Intent = request.Intent
		arguments.Now = now
		// `declared` é nil DE PROPÓSITO: quem declara ferramenta é a ESPECIFICAÇÃO do agente, não o pedido.
		// Passar `request.RequestedTools` aqui tornaria o pedido a própria autorização — e aí pedir seria
		// poder, exatamente o que esta função existe para impedir.
		calls = append(calls, s.RunTool(spec, nil, consentGranted, name, arguments))
	}
	return calls
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

// SetAuditStore liga a trilha DURÁVEL. Sem store, a trilha continua vindo na resposta do turno (por turno)
// e `AuditError` não é preenchido — durabilidade não se presume.
func (s *ServiceAgentRuntime) SetAuditStore(store *ServiceAgentAuditStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = store
	s.auditError = ""
}

// recordAudit grava as chamadas do turno na trilha durável e devolve o erro, quando houver.
func (s *ServiceAgentRuntime) recordAudit(organizationID, agentID, conversationID, modelID string, calls []ServiceAgentToolCall, now time.Time) string {
	if len(calls) == 0 {
		return ""
	}
	s.mu.Lock()
	store := s.audit
	s.mu.Unlock()
	if store == nil {
		return ""
	}
	entries := make([]ServiceAgentAuditEntry, 0, len(calls))
	for _, call := range calls {
		entries = append(entries, ServiceAgentAuditEntry{
			ID:             "aud_" + uuid.NewString()[:12],
			OrganizationID: organizationID,
			AgentID:        agentID,
			ConversationID: conversationID,
			ToolName:       call.Name,
			Effect:         call.Effect,
			Status:         call.Status,
			Summary:        call.Summary,
			Reason:         call.Reason,
			ModelID:        modelID,
			DecidedAt:      call.ExecutedAt,
			RecordedAt:     now,
		})
	}
	if _, err := store.Append(organizationID, entries); err != nil {
		s.mu.Lock()
		s.auditError = err.Error()
		s.mu.Unlock()
		return err.Error()
	}
	return ""
}

// AuditTrail devolve a trilha durável da organização, mais recentes primeiro, com limite.
//
// Lê do DISCO a cada chamada: a trilha vem de fora do processo, e uma leitura que dependesse de cache em
// memória poderia mostrar menos do que foi registrado — em auditoria, mostrar menos é pior que demorar.
func (s *ServiceAgentRuntime) AuditTrail(organizationID string, limit int) ([]ServiceAgentAuditEntry, error) {
	organizationID = consentScope(organizationID)
	s.mu.Lock()
	store := s.audit
	s.mu.Unlock()
	if store == nil {
		return nil, fmt.Errorf("%w: trilha durável não configurada", ErrServiceAgentTurnInvalid)
	}
	entries, err := store.Load(organizationID)
	if err != nil {
		return nil, err
	}
	// Isolamento também na leitura: arquivo de outra organização não contamina este escopo.
	own := make([]ServiceAgentAuditEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.OrganizationID == organizationID {
			own = append(own, entry)
		}
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].RecordedAt.After(own[j].RecordedAt) })
	if limit > 0 && len(own) > limit {
		own = own[:limit]
	}
	return own, nil
}

// AuditStatus é o estado real da trilha durável de uma organização.
type ServiceAgentAuditStatus struct {
	OrganizationID string `json:"organization_id"`
	// Persisted diz se há store configurado — durabilidade não se presume.
	Persisted bool `json:"persisted"`
	// Storage é o diretório usado quando há persistência.
	Storage string `json:"storage,omitempty"`
	// Entries é quantos registros a organização tem agora.
	Entries int `json:"entries"`
	// PerOrganization é o teto aplicado a ESTA organização.
	PerOrganization int `json:"per_organization"`
	// Error é a última falha de persistência, quando houve.
	Error string `json:"error,omitempty"`
	// Verification é o resultado da CADEIA de hashes: adulteração aparece aqui, em vez de passar silenciosa.
	Verification *ServiceAgentAuditVerification `json:"verification,omitempty"`
}

// AuditStatus reporta o estado da trilha durável.
func (s *ServiceAgentRuntime) AuditStatus(organizationID string) (ServiceAgentAuditStatus, error) {
	organizationID = consentScope(organizationID)
	s.mu.Lock()
	store := s.audit
	errMessage := s.auditError
	s.mu.Unlock()
	status := ServiceAgentAuditStatus{OrganizationID: organizationID, Error: errMessage}
	if store == nil {
		return status, nil
	}
	entries, err := store.Load(organizationID)
	if err != nil {
		// Arquivo ilegível NÃO vira "sem registros": o erro sobe para quem audita, porque uma trilha
		// corrompida apresentada como vazia apagaria a evidência de que existiam execuções.
		return ServiceAgentAuditStatus{}, err
	}
	status.Persisted = true
	status.Storage = store.Root()
	status.PerOrganization = store.MaxPerOrganization
	for _, entry := range entries {
		if entry.OrganizationID == organizationID {
			status.Entries++
		}
	}
	// A verificação acompanha o estado: quem audita precisa saber se a trilha é ÍNTEGRA, não só se existe.
	verification, err := store.VerifyTrail(organizationID)
	if err != nil {
		return ServiceAgentAuditStatus{}, err
	}
	status.Verification = &verification
	return status, nil
}

// ServiceAgentRuntime liga os agentes aos recursos existentes.
type ServiceAgentRuntime struct {
	mu      sync.Mutex
	crm     *CRMStore
	inbox   *InboxStore
	consent *ConsentRegistry
	// audit é a trilha durável de execução de ferramentas; nil mantém a trilha apenas por turno.
	audit *ServiceAgentAuditStore
	// auditError guarda a última falha de gravação da trilha, para ser REPORTADA nunca escondida.
	auditError string
	// executors são as ferramentas REALMENTE executáveis. A disponibilidade é DERIVADA daqui: um mapa
	// digitado à mão não pode afirmar capacidade que ninguém implementou.
	executors map[string]ServiceAgentTool
	// declared guarda o que foi declarado pelo chamador, para o relatório de indisponíveis.
	declared map[string]bool
}

// NewServiceAgentRuntime liga os agentes ao CRM, à caixa de entrada e ao consentimento.
//
// `declaredTools` é o que os agentes PODEM declarar; a disponibilidade real vem dos executores
// registrados. Ferramenta declarada e sem executor aparece como **indisponível, com motivo**.
func NewServiceAgentRuntime(crm *CRMStore, inbox *InboxStore, consent *ConsentRegistry, declaredTools map[string]bool) *ServiceAgentRuntime {
	if declaredTools == nil {
		declaredTools = map[string]bool{}
	}
	runtime := &ServiceAgentRuntime{
		crm:       crm,
		inbox:     inbox,
		consent:   consent,
		declared:  declaredTools,
		executors: serviceAgentTools(crm, inbox, consent),
	}
	// Os executores registrados por padrão também passam a constar como declarados: sem isso, uma
	// ferramenta executável ficaria invisível para os agentes.
	for name := range runtime.executors {
		runtime.declared[name] = true
	}
	return runtime
}

// availableTools devolve a INTERSECÇÃO entre o que foi declarado e o que tem executor.
func (s *ServiceAgentRuntime) availableTools() map[string]bool {
	out := map[string]bool{}
	for name := range s.executors {
		if s.declared[name] {
			out[name] = true
		}
	}
	return out
}

// RegisterTool declara uma ferramenta como permitida aos agentes.
//
// Atenção: declarar NÃO a torna disponível — quem faz isso é `RegisterExecutor`. A separação existe para
// que "o agente pode pedir" e "o runtime consegue executar" sejam coisas distintas e verificáveis.
func (s *ServiceAgentRuntime) RegisterTool(name string) {
	if s.declared == nil {
		s.declared = map[string]bool{}
	}
	s.declared[strings.TrimSpace(name)] = true
}

// RegisterExecutor registra um executor de verdade, tornando a ferramenta disponível.
func (s *ServiceAgentRuntime) RegisterExecutor(tool ServiceAgentTool) {
	if tool == nil {
		return
	}
	if s.executors == nil {
		s.executors = map[string]ServiceAgentTool{}
	}
	if s.declared == nil {
		s.declared = map[string]bool{}
	}
	name := strings.TrimSpace(tool.Name())
	s.executors[name] = tool
	s.declared[name] = true
}

// SetCommerce liga o motor de comércio e passa a registrar as ferramentas dele.
//
// Sem esta ligação, `catalog.read`, `commerce.stock.read` e `commerce.order.read` aparecem como
// INDISPONÍVEIS — a disponibilidade continua sendo consequência do registro, não de uma lista digitada.
func (s *ServiceAgentRuntime) SetCommerce(commerce *CommerceStore) {
	for _, tool := range serviceAgentCommerceTools(commerce) {
		s.RegisterExecutor(tool)
	}
}

// SetCompanies liga o cadastro de empresas e registra `company.report.read`.
func (s *ServiceAgentRuntime) SetCompanies(companies *CompanyStore) {
	for _, tool := range serviceAgentCompanyTools(companies) {
		s.RegisterExecutor(tool)
	}
}

// ToolExecutors lista as ferramentas com executor registrado (a disponibilidade real).
func (s *ServiceAgentRuntime) ToolExecutors() []string {
	out := make([]string, 0, len(s.executors))
	for name := range s.executors {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// UnavailableTools lista o que os agentes declaram e o runtime NÃO consegue executar.
func (s *ServiceAgentRuntime) UnavailableTools() []string {
	out := make([]string, 0)
	for name := range s.declared {
		if _, ok := s.executors[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// DefaultServiceAgentTools lista as ferramentas que os agentes PODEM declarar por catálogo.
//
// A versão anterior deste comentário dizia que eram as "REALMENTE implementadas hoje" — o que era falso
// para `commerce.order.read`, `commerce.stock.read` e `company.report.read`: eram booleanos sem executor.
// Agora a disponibilidade é derivada dos executores registrados e o que não tem executor aparece como
// **indisponível**, com motivo (`UnavailableTools`).
func DefaultServiceAgentTools() map[string]bool {
	return map[string]bool{
		"crm.contact.read":        true,
		"crm.opportunity.write":   true,
		"inbox.conversation.read": true,
		"inbox.reply":             true,
		"inbox.handoff":           true,
		"inbox.metrics.read":      true,
		"inbox.assign":            true,
		// SEM executor neste runtime: dependem dos módulos de comércio e de empresa, que não são
		// entregues a este serviço. Ficam declaradas de propósito, para aparecerem como
		// INDISPONÍVEIS no turno — visível é melhor que silencioso.
		"company.report.read": true,
		"commerce.order.read": true,
		"commerce.stock.read": true,
	}
}
