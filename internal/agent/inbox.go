package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Caixa de entrada unificada (estágio 29) — destrava `/inbox`, `/handoff` e `/support`.
//
// Os três comandos estavam **indisponíveis** com o motivo *"a caixa de entrada
// unificada ainda não existe nesta versão"*. Este é o núcleo que faltava.
//
// DECISÕES DELIBERADAS:
//   - **Uma conversa por contato e canal**: mensagem nova do mesmo contato no mesmo
//     canal **reaproveita** a conversa aberta. Criar uma conversa por mensagem
//     destruiria histórico e contexto — o oposto do que o estágio pede;
//   - **Deduplicação de entrada por `ExternalID`**: webhook reentregue **não** pode
//     duplicar mensagem (§11.7). A checagem e a inserção ficam sob o mesmo lock,
//     então vale também para entregas concorrentes;
//   - **SLA com prazo calculado, não presumido**: a primeira resposta do agente para
//     o relógio; sem ela, a conversa aparece como **violada** depois do prazo. Não
//     existe "SLA medido" por declaração;
//   - **Handoff é estado, não texto**: transferir registra quem transferiu, para quem
//     e por quê, e move a conversa para `pending_human` — a IA para de responder;
//   - **Consentimento é consultado no ENVIO**, não copiado: responder para um contato
//     com opt-out é recusado, e a decisão vem do `ConsentRegistry`.

const (
	// Canais aceitos na caixa de entrada.
	InboxChannelWhatsApp  = "whatsapp"
	InboxChannelTelegram  = "telegram"
	InboxChannelEmail     = "email"
	InboxChannelWebchat   = "webchat"
	InboxChannelPhone     = "phone"
	InboxChannelInstagram = "instagram"
	InboxChannelMessenger = "messenger"

	// Status da conversa.
	InboxStatusOpen         = "open"
	InboxStatusPendingHuman = "pending_human"
	InboxStatusClosed       = "closed"

	// Direção da mensagem.
	InboxDirectionInbound  = "inbound"
	InboxDirectionOutbound = "outbound"

	// DefaultInboxFirstResponseSLA é o prazo padrão da primeira resposta.
	DefaultInboxFirstResponseSLA = 15 * time.Minute
	// DefaultInboxResolutionSLA e' a janela de RESOLUCAO quando a organizacao nao define uma (E-266).
	DefaultInboxResolutionSLA = 24 * time.Hour
)

var (
	ErrInboxInvalid          = errors.New("inbox entry invalid")
	ErrInboxConversationGone = errors.New("inbox conversation not found")
	ErrInboxConsentRequired  = errors.New("cannot reply: contact consent is missing or revoked")
	ErrInboxConversationDone = errors.New("inbox conversation is closed")
)

// InboxConversation é uma conversa com um contato num canal.
type InboxConversation struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Channel        string     `json:"channel"`
	ContactID      string     `json:"contact_id"`
	ContactName    string     `json:"contact_name,omitempty"`
	Status         string     `json:"status"`
	AssignedTo     string     `json:"assigned_to,omitempty"`
	AssignedKind   string     `json:"assigned_kind,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	Priority       string     `json:"priority,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FirstInboundAt time.Time  `json:"first_inbound_at,omitempty"`
	FirstReplyAt   *time.Time `json:"first_reply_at,omitempty"`
	ReplyDueAt     *time.Time `json:"reply_due_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	MessageCount   int        `json:"message_count"`
	// LastMessage é uma prévia curta para a lista não precisar carregar tudo.
	LastMessage string `json:"last_message,omitempty"`
}

// InboxMessage é uma mensagem da conversa.
type InboxMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	OrganizationID string `json:"organization_id"`
	Direction      string `json:"direction"`
	AuthorType     string `json:"author_type"`
	AuthorID       string `json:"author_id,omitempty"`
	Body           string `json:"body"`
	// ExternalID é o identificador do provedor: é a chave de deduplicação.
	ExternalID string    `json:"external_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	// Deduplicated marca a mensagem devolvida quando a entrega se repetiu.
	Deduplicated bool `json:"deduplicated,omitempty"`
}

// InboxMetrics são os indicadores da operação.
type InboxMetrics struct {
	OrganizationID       string  `json:"organization_id"`
	TotalConversations   int     `json:"total_conversations"`
	OpenConversations    int     `json:"open_conversations"`
	PendingHuman         int     `json:"pending_human"`
	ClosedConversations  int     `json:"closed_conversations"`
	SLABreached          int     `json:"sla_breached"`
	AwaitingFirstReply   int     `json:"awaiting_first_reply"`
	FirstResponseAvgSecs float64 `json:"first_response_avg_seconds"`
	// RESOLUCAO (E-266): o SLA de primeira resposta nao diz se o cliente foi ATENDIDO ate o fim.
	ResolutionSLABreached int     `json:"resolution_sla_breached"`
	ResolutionOverdue     int     `json:"resolution_overdue"`
	ResolutionAvgSecs     float64 `json:"resolution_avg_seconds"`
	MessagesInbound       int     `json:"messages_inbound"`
	MessagesOutbound      int     `json:"messages_outbound"`
}

type inboxFile struct {
	Conversations []InboxConversation `json:"conversations"`
	Messages      []InboxMessage      `json:"messages"`
}

// InboxStore guarda conversas e mensagens por organização.
type InboxStore struct {
	mu            sync.RWMutex
	root          string
	conversations map[string]InboxConversation
	messages      map[string][]InboxMessage
	// externalIndex evita duplicar mensagem já recebida do provedor.
	externalIndex map[string]string
	firstSLAByOrg map[string]time.Duration
	// resolutionSLAByOrg guarda a janela de RESOLUCAO por organizacao (E-266).
	resolutionSLAByOrg map[string]time.Duration
	// knowledge guarda a base de conhecimento carregada (E-265).
	knowledge map[string]InboxKnowledgeArticle
}

// NewInboxStore carrega a caixa de entrada do diretório informado.
func NewInboxStore(root string) (*InboxStore, error) {
	store := &InboxStore{
		root:               strings.TrimSpace(root),
		conversations:      map[string]InboxConversation{},
		messages:           map[string][]InboxMessage{},
		externalIndex:      map[string]string{},
		firstSLAByOrg:      map[string]time.Duration{},
		resolutionSLAByOrg: map[string]time.Duration{},
	}
	if store.root == "" {
		return store, nil
	}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var stored inboxFile
		if err := readJSON(filepath.Join(store.root, entry.Name()), &stored); err != nil {
			return nil, fmt.Errorf("load inbox from %s: %w", entry.Name(), err)
		}
		for _, conversation := range stored.Conversations {
			store.conversations[conversation.ID] = conversation
			for _, message := range stored.Messages {
				if message.ConversationID != conversation.ID {
					continue
				}
				store.messages[conversation.ID] = append(store.messages[conversation.ID], message)
				if key := inboxExternalKey(conversation.OrganizationID, message.ExternalID); key != "" {
					store.externalIndex[key] = message.ID
				}
			}
			sort.SliceStable(store.messages[conversation.ID], func(i, j int) bool {
				return store.messages[conversation.ID][i].CreatedAt.Before(store.messages[conversation.ID][j].CreatedAt)
			})
		}
	}
	return store, nil
}

func inboxExternalKey(organizationID, externalID string) string {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return ""
	}
	return consentScope(organizationID) + "|" + externalID
}

func normalizeInboxChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case InboxChannelWhatsApp, "whats", "zap":
		return InboxChannelWhatsApp
	case InboxChannelTelegram:
		return InboxChannelTelegram
	case InboxChannelEmail, "e-mail":
		return InboxChannelEmail
	case InboxChannelWebchat, "chat", "site":
		return InboxChannelWebchat
	case InboxChannelPhone, "telefone", "voz":
		return InboxChannelPhone
	case InboxChannelInstagram:
		return InboxChannelInstagram
	case InboxChannelMessenger:
		return InboxChannelMessenger
	default:
		return ""
	}
}

func (s *InboxStore) organizationPath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

func (s *InboxStore) persistOrganizationLocked(organizationID string) error {
	path := s.organizationPath(organizationID)
	if path == "" {
		return nil
	}
	payload := inboxFile{Conversations: []InboxConversation{}, Messages: []InboxMessage{}}
	for _, conversation := range s.conversations {
		if conversation.OrganizationID != organizationID {
			continue
		}
		payload.Conversations = append(payload.Conversations, conversation)
		payload.Messages = append(payload.Messages, s.messages[conversation.ID]...)
	}
	sort.SliceStable(payload.Conversations, func(i, j int) bool { return payload.Conversations[i].ID < payload.Conversations[j].ID })
	return writeJSONAtomic(path, payload)
}

// SetFirstResponseSLA ajusta o prazo de primeira resposta (0 usa o padrão).
func (s *InboxStore) SetFirstResponseSLA(organizationID string, window time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if window <= 0 {
		delete(s.firstSLAByOrg, consentScope(organizationID))
		return
	}
	s.firstSLAByOrg[consentScope(organizationID)] = window
}

// SetResolutionSLA define a janela de RESOLUCAO da organizacao. Zero remove a janela e volta ao padrao DECLARADO.
func (s *InboxStore) SetResolutionSLA(organizationID string, window time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resolutionSLAByOrg == nil {
		s.resolutionSLAByOrg = map[string]time.Duration{}
	}
	if window <= 0 {
		delete(s.resolutionSLAByOrg, consentScope(organizationID))
		return
	}
	s.resolutionSLAByOrg[consentScope(organizationID)] = window
}

func (s *InboxStore) resolutionSLA(organizationID string) time.Duration {
	if window, ok := s.resolutionSLAByOrg[consentScope(organizationID)]; ok && window > 0 {
		return window
	}
	return DefaultInboxResolutionSLA
}

func (s *InboxStore) firstResponseSLA(organizationID string) time.Duration {
	if window, ok := s.firstSLAByOrg[consentScope(organizationID)]; ok && window > 0 {
		return window
	}
	return DefaultInboxFirstResponseSLA
}

// openConversationLocked reaproveita a conversa aberta do contato no canal, ou cria
// uma nova. Uma conversa por contato e canal é o que preserva histórico e contexto.
func (s *InboxStore) openConversationLocked(organizationID, channel, contactID, contactName string, now time.Time) InboxConversation {
	for id, conversation := range s.conversations {
		if conversation.OrganizationID != organizationID || conversation.Channel != channel || conversation.ContactID != contactID {
			continue
		}
		if conversation.Status == InboxStatusClosed {
			continue
		}
		conversation.UpdatedAt = now
		if contactName != "" {
			conversation.ContactName = contactName
		}
		s.conversations[id] = conversation
		return conversation
	}
	due := now.Add(s.firstResponseSLA(organizationID))
	conversation := InboxConversation{
		ID:             "cv_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		Channel:        channel,
		ContactID:      contactID,
		ContactName:    contactName,
		Status:         InboxStatusOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
		FirstInboundAt: now,
		ReplyDueAt:     &due,
	}
	s.conversations[conversation.ID] = conversation
	return conversation
}

// AppendInbound registra mensagem recebida. Reentrega do provedor com o MESMO
// `ExternalID` devolve a mensagem original com `Deduplicated=true` em vez de criar
// outra — webhook duplicado não pode virar mensagem duplicada.
func (s *InboxStore) AppendInbound(organizationID, channel, contactID, contactName, body, externalID string, now time.Time) (InboxConversation, InboxMessage, error) {
	if err := requireOrganization(organizationID); err != nil {
		return InboxConversation{}, InboxMessage{}, err
	}
	organizationID = consentScope(organizationID)
	channel = normalizeInboxChannel(channel)
	contactID = strings.TrimSpace(contactID)
	body = strings.TrimSpace(body)
	if channel == "" || contactID == "" || body == "" {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: channel, contact and body are required", ErrInboxInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if key := inboxExternalKey(organizationID, externalID); key != "" {
		if messageID, seen := s.externalIndex[key]; seen {
			for _, conversation := range s.conversations {
				for _, message := range s.messages[conversation.ID] {
					if message.ID != messageID {
						continue
					}
					message.Deduplicated = true
					return conversation, message, nil
				}
			}
		}
	}
	conversation := s.openConversationLocked(organizationID, channel, contactID, contactName, now)
	message := InboxMessage{
		ID:             "msg_" + uuid.NewString()[:12],
		ConversationID: conversation.ID,
		OrganizationID: organizationID,
		Direction:      InboxDirectionInbound,
		AuthorType:     "contact",
		AuthorID:       contactID,
		Body:           body,
		ExternalID:     strings.TrimSpace(externalID),
		CreatedAt:      now,
	}
	s.messages[conversation.ID] = append(s.messages[conversation.ID], message)
	conversation.MessageCount = len(s.messages[conversation.ID])
	conversation.LastMessage = truncateInboxPreview(body)
	conversation.UpdatedAt = now
	if conversation.FirstInboundAt.IsZero() {
		conversation.FirstInboundAt = now
	}
	s.conversations[conversation.ID] = conversation
	if key := inboxExternalKey(organizationID, message.ExternalID); key != "" {
		s.externalIndex[key] = message.ID
	}
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		// Sem gravação a mensagem não existe: desfaz para não mentir em memória.
		s.messages[conversation.ID] = s.messages[conversation.ID][:len(s.messages[conversation.ID])-1]
		return InboxConversation{}, InboxMessage{}, err
	}
	return conversation, message, nil
}

// AppendOutbound registra resposta do agente ou de uma pessoa. O envio passa pelo
// CONSENTIMENTO: responder quem pediu para não ser procurado é recusado.
func (s *InboxStore) AppendOutbound(organizationID, conversationID, authorType, authorID, body string, consent *ConsentRegistry, now time.Time) (InboxConversation, InboxMessage, error) {
	if err := requireOrganization(organizationID); err != nil {
		return InboxConversation{}, InboxMessage{}, err
	}
	organizationID = consentScope(organizationID)
	body = strings.TrimSpace(body)
	if body == "" {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: body is required", ErrInboxInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	conversation, ok := s.conversations[strings.TrimSpace(conversationID)]
	if !ok || conversation.OrganizationID != organizationID {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: %s", ErrInboxConversationGone, conversationID)
	}
	if conversation.Status == InboxStatusClosed {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: %s", ErrInboxConversationDone, conversationID)
	}
	if consent != nil {
		if err := consent.AuthorizeContact(organizationID, conversation.Channel, conversation.ContactID, now); err != nil {
			return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: %v", ErrInboxConsentRequired, err)
		}
	}
	message := InboxMessage{
		ID:             "msg_" + uuid.NewString()[:12],
		ConversationID: conversation.ID,
		OrganizationID: organizationID,
		Direction:      InboxDirectionOutbound,
		AuthorType:     strings.TrimSpace(authorType),
		AuthorID:       strings.TrimSpace(authorID),
		Body:           body,
		CreatedAt:      now,
	}
	s.messages[conversation.ID] = append(s.messages[conversation.ID], message)
	conversation.MessageCount = len(s.messages[conversation.ID])
	conversation.LastMessage = truncateInboxPreview(body)
	conversation.UpdatedAt = now
	if conversation.FirstReplyAt == nil {
		replyAt := now
		conversation.FirstReplyAt = &replyAt
	}
	s.conversations[conversation.ID] = conversation
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		s.messages[conversation.ID] = s.messages[conversation.ID][:len(s.messages[conversation.ID])-1]
		return InboxConversation{}, InboxMessage{}, err
	}
	return conversation, message, nil
}

func truncateInboxPreview(body string) string {
	const limit = 160
	body = strings.Join(strings.Fields(body), " ")
	if len(body) <= limit {
		return body
	}
	return body[:limit] + "…"
}

// ListConversations lista as conversas, opcionalmente por status.
func (s *InboxStore) ListConversations(organizationID, status string) []InboxConversation {
	if err := requireOrganization(organizationID); err != nil {
		return nil
	}
	organizationID = consentScope(organizationID)
	status = strings.ToLower(strings.TrimSpace(status))
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]InboxConversation, 0)
	for _, conversation := range s.conversations {
		if conversation.OrganizationID != organizationID {
			continue
		}
		if status != "" && conversation.Status != status {
			continue
		}
		out = append(out, conversation)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// Conversation carrega a conversa com as mensagens.
func (s *InboxStore) Conversation(organizationID, conversationID string) (InboxConversation, []InboxMessage, error) {
	if err := requireOrganization(organizationID); err != nil {
		return InboxConversation{}, nil, err
	}
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	conversation, ok := s.conversations[strings.TrimSpace(conversationID)]
	if !ok || conversation.OrganizationID != organizationID {
		return InboxConversation{}, nil, fmt.Errorf("%w: %s", ErrInboxConversationGone, conversationID)
	}
	return conversation, append([]InboxMessage(nil), s.messages[conversation.ID]...), nil
}

// Assign define o responsável pela conversa.
func (s *InboxStore) Assign(organizationID, conversationID, assignee, kind string, now time.Time) (InboxConversation, error) {
	if err := requireOrganization(organizationID); err != nil {
		return InboxConversation{}, err
	}
	organizationID = consentScope(organizationID)
	assignee = strings.TrimSpace(assignee)
	if assignee == "" {
		return InboxConversation{}, fmt.Errorf("%w: assignee is required", ErrInboxInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	conversation, ok := s.conversations[strings.TrimSpace(conversationID)]
	if !ok || conversation.OrganizationID != organizationID {
		return InboxConversation{}, fmt.Errorf("%w: %s", ErrInboxConversationGone, conversationID)
	}
	conversation.AssignedTo = assignee
	conversation.AssignedKind = strings.TrimSpace(kind)
	conversation.UpdatedAt = now
	s.conversations[conversation.ID] = conversation
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		return InboxConversation{}, err
	}
	return conversation, nil
}

// HandoffToHuman move a conversa para atendimento humano e registra a passagem como
// MENSAGEM do sistema: a trilha fica na própria conversa.
func (s *InboxStore) HandoffToHuman(organizationID, conversationID, actorID, assignee, reason string, now time.Time) (InboxConversation, InboxMessage, error) {
	organizationID = consentScope(organizationID)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: handoff requires a reason", ErrInboxInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	conversation, ok := s.conversations[strings.TrimSpace(conversationID)]
	if !ok || conversation.OrganizationID != organizationID {
		return InboxConversation{}, InboxMessage{}, fmt.Errorf("%w: %s", ErrInboxConversationGone, conversationID)
	}
	conversation.Status = InboxStatusPendingHuman
	conversation.UpdatedAt = now
	if target := strings.TrimSpace(assignee); target != "" {
		conversation.AssignedTo = target
		conversation.AssignedKind = "human"
	}
	s.conversations[conversation.ID] = conversation
	message := InboxMessage{
		ID:             "msg_" + uuid.NewString()[:12],
		ConversationID: conversation.ID,
		OrganizationID: organizationID,
		Direction:      InboxDirectionOutbound,
		AuthorType:     "system",
		AuthorID:       strings.TrimSpace(actorID),
		Body:           "transferido para atendimento humano: " + reason,
		CreatedAt:      now,
	}
	s.messages[conversation.ID] = append(s.messages[conversation.ID], message)
	conversation.MessageCount = len(s.messages[conversation.ID])
	s.conversations[conversation.ID] = conversation
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		return InboxConversation{}, InboxMessage{}, err
	}
	return conversation, message, nil
}

// Close encerra a conversa e registra quem encerrou.
func (s *InboxStore) Close(organizationID, conversationID, actorID, reason string, now time.Time) (InboxConversation, error) {
	if err := requireOrganization(organizationID); err != nil {
		return InboxConversation{}, err
	}
	organizationID = consentScope(organizationID)
	s.mu.Lock()
	defer s.mu.Unlock()
	conversation, ok := s.conversations[strings.TrimSpace(conversationID)]
	if !ok || conversation.OrganizationID != organizationID {
		return InboxConversation{}, fmt.Errorf("%w: %s", ErrInboxConversationGone, conversationID)
	}
	closed := now
	conversation.Status = InboxStatusClosed
	conversation.ClosedAt = &closed
	conversation.UpdatedAt = now
	s.conversations[conversation.ID] = conversation
	if strings.TrimSpace(reason) != "" || strings.TrimSpace(actorID) != "" {
		s.messages[conversation.ID] = append(s.messages[conversation.ID], InboxMessage{
			ID:             "msg_" + uuid.NewString()[:12],
			ConversationID: conversation.ID,
			OrganizationID: organizationID,
			Direction:      InboxDirectionOutbound,
			AuthorType:     "system",
			AuthorID:       strings.TrimSpace(actorID),
			Body:           "conversa encerrada: " + strings.TrimSpace(reason),
			CreatedAt:      now,
		})
		conversation.MessageCount = len(s.messages[conversation.ID])
		s.conversations[conversation.ID] = conversation
	}
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		return InboxConversation{}, err
	}
	return conversation, nil
}

// Metrics calcula SLA e volume. Violação de SLA é MEDIDA contra o relógio da
// primeira resposta, não presumida.
func (s *InboxStore) Metrics(organizationID string, now time.Time) InboxMetrics {
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	metrics := InboxMetrics{OrganizationID: organizationID}
	var firstResponseTotal time.Duration
	var firstResponseCount int
	var resolutionTotal time.Duration
	var resolutionCount int
	for _, conversation := range s.conversations {
		if conversation.OrganizationID != organizationID {
			continue
		}
		metrics.TotalConversations++
		for _, message := range s.messages[conversation.ID] {
			if message.Direction == InboxDirectionInbound {
				metrics.MessagesInbound++
			} else {
				metrics.MessagesOutbound++
			}
		}
		switch conversation.Status {
		case InboxStatusClosed:
			metrics.ClosedConversations++
		case InboxStatusPendingHuman:
			metrics.PendingHuman++
		default:
			metrics.OpenConversations++
		}
		// RESOLUCAO (E-266): calculada ANTES do desvio de primeira resposta - uma conversa fechada sem primeira
		// resposta registrada continua contando para a resolucao.
		if !conversation.FirstInboundAt.IsZero() {
			janela := s.resolutionSLA(organizationID)
			if conversation.ClosedAt != nil {
				decorrido := conversation.ClosedAt.Sub(conversation.FirstInboundAt)
				resolutionTotal += decorrido
				resolutionCount++
				if decorrido > janela {
					metrics.ResolutionSLABreached++
				}
			} else if now.Sub(conversation.FirstInboundAt) > janela {
				// Ainda aberta e a janela JA venceu: e' atraso, nao "em andamento".
				metrics.ResolutionOverdue++
			}
		}
		if conversation.FirstReplyAt == nil {
			if !conversation.FirstInboundAt.IsZero() {
				metrics.AwaitingFirstReply++
			}
			if conversation.ReplyDueAt != nil && conversation.ReplyDueAt.Before(now) {
				metrics.SLABreached++
			}
			continue
		}
		if !conversation.FirstInboundAt.IsZero() {
			firstResponseTotal += conversation.FirstReplyAt.Sub(conversation.FirstInboundAt)
			firstResponseCount++
		}
	}
	if firstResponseCount > 0 {
		metrics.FirstResponseAvgSecs = firstResponseTotal.Seconds() / float64(firstResponseCount)
	}
	if resolutionCount > 0 {
		metrics.ResolutionAvgSecs = resolutionTotal.Seconds() / float64(resolutionCount)
	}
	return metrics
}

// ---------------------------------------------------------------------------------------------------------------
// BASE DE CONHECIMENTO DO INBOX (E-265)
// ---------------------------------------------------------------------------------------------------------------
//
// O que isto E: artigos por organizacao, busca DETERMINISTICA por palavras, e uma SUGESTAO de resposta a partir da
// ultima mensagem recebida. O que isto NAO E: envio automatico. Sugerir e leitura pura - quem responde e um humano
// ou um agente sob politica. A busca nao finge ser semantica: e palavra-chave, e o nome diz isso.

var (
	// ErrInboxKnowledgeInvalid cobre artigo sem organizacao/titulo/conteudo.
	ErrInboxKnowledgeInvalid = errors.New("inbox knowledge article invalid")
	// ErrInboxKnowledgeMissing cobre artigo inexistente NA organizacao do chamador.
	ErrInboxKnowledgeMissing = errors.New("inbox knowledge article not found")
)

// InboxKnowledgeArticle e' um artigo da base de conhecimento de uma organizacao.
type InboxKnowledgeArticle struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Tags           []string  `json:"tags,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// InboxKnowledgeHit e' um artigo encontrado, com a pontuacao da busca por palavra-chave.
type InboxKnowledgeHit struct {
	Article InboxKnowledgeArticle `json:"article"`
	Score   int                   `json:"score"`
}

// InboxSuggestion e' uma SUGESTAO de resposta. Found=false quando nada casa - nao se inventa resposta.
type InboxSuggestion struct {
	Found      bool   `json:"found"`
	ArticleID  string `json:"article_id,omitempty"`
	Title      string `json:"title,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Score      int    `json:"score,omitempty"`
}

type inboxKnowledgeFile struct {
	Articles []InboxKnowledgeArticle `json:"articles"`
}

func (s *InboxStore) knowledgePath(organizationID string) string {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return ""
	}
	seguro := strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(consentScope(organizationID))
	if seguro == "" {
		return ""
	}
	return filepath.Join(s.root, "inbox_knowledge_"+seguro+".json")
}

func (s *InboxStore) loadKnowledge(organizationID string) ([]InboxKnowledgeArticle, error) {
	caminho := s.knowledgePath(organizationID)
	if caminho == "" {
		return nil, nil
	}
	stored := inboxKnowledgeFile{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("ler base de conhecimento: %w", err)
	}
	return stored.Articles, nil
}

// UpsertKnowledgeArticle cria (ID vazio) ou atualiza (mesmo ID) um artigo. Devolve o artigo gravado.
func (s *InboxStore) UpsertKnowledgeArticle(article InboxKnowledgeArticle, now time.Time) (InboxKnowledgeArticle, error) {
	organizationID := consentScope(article.OrganizationID)
	if organizationID == "" || strings.TrimSpace(article.Title) == "" || strings.TrimSpace(article.Content) == "" {
		return InboxKnowledgeArticle{}, fmt.Errorf("%w: organizacao=%q titulo=%q", ErrInboxKnowledgeInvalid, organizationID, article.Title)
	}
	artigos, err := s.loadKnowledge(organizationID)
	if err != nil {
		return InboxKnowledgeArticle{}, err
	}
	gravado := article
	gravado.OrganizationID = organizationID
	gravado.Title = strings.TrimSpace(article.Title)
	gravado.Content = strings.TrimSpace(article.Content)
	gravado.UpdatedAt = now.UTC()
	tags := make([]string, 0, len(article.Tags))
	for _, tag := range article.Tags {
		if limpa := strings.TrimSpace(tag); limpa != "" {
			tags = append(tags, limpa)
		}
	}
	gravado.Tags = tags
	substituido := false
	if strings.TrimSpace(gravado.ID) == "" {
		gravado.ID = "kb_" + uuid.NewString()
	} else {
		for i := range artigos {
			if artigos[i].ID == gravado.ID {
				artigos[i] = gravado
				substituido = true
				break
			}
		}
	}
	if !substituido {
		artigos = append(artigos, gravado)
	}
	caminho := s.knowledgePath(organizationID)
	if caminho == "" {
		// Modo em memoria: o artigo vale nesta execucao. Declarado, nao fingido.
		return gravado, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.knowledge == nil {
		s.knowledge = map[string]InboxKnowledgeArticle{}
	}
	s.knowledge[organizationID+"|"+gravado.ID] = gravado
	if err := writeJSONAtomic(caminho, inboxKnowledgeFile{Articles: artigos}); err != nil {
		delete(s.knowledge, organizationID+"|"+gravado.ID)
		return InboxKnowledgeArticle{}, err
	}
	return gravado, nil
}

// KnowledgeArticles lista os artigos da organizacao (copia).
func (s *InboxStore) KnowledgeArticles(organizationID string) ([]InboxKnowledgeArticle, error) {
	artigos, err := s.loadKnowledge(organizationID)
	if err != nil {
		return nil, err
	}
	resultado := append([]InboxKnowledgeArticle(nil), artigos...)
	sort.Slice(resultado, func(i, j int) bool {
		if resultado[i].Title == resultado[j].Title {
			return resultado[i].ID < resultado[j].ID
		}
		return resultado[i].Title < resultado[j].Title
	})
	return resultado, nil
}

// DeleteKnowledgeArticle remove um artigo. Artigo de OUTRA organizacao responde como inexistente.
func (s *InboxStore) DeleteKnowledgeArticle(organizationID, articleID string) error {
	organizationID = consentScope(organizationID)
	artigos, err := s.loadKnowledge(organizationID)
	if err != nil {
		return err
	}
	restantes := make([]InboxKnowledgeArticle, 0, len(artigos))
	removido := false
	for _, artigo := range artigos {
		if artigo.ID == strings.TrimSpace(articleID) {
			removido = true
			continue
		}
		restantes = append(restantes, artigo)
	}
	if !removido {
		return fmt.Errorf("%w: artigo %s", ErrInboxKnowledgeMissing, strings.TrimSpace(articleID))
	}
	caminho := s.knowledgePath(organizationID)
	if caminho == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.knowledge, organizationID+"|"+strings.TrimSpace(articleID))
	return writeJSONAtomic(caminho, inboxKnowledgeFile{Articles: restantes})
}

// palavras separa o texto em palavras minusculas (busca por PALAVRA-CHAVE, nao semantica).
func palavras(texto string) []string {
	campos := strings.FieldsFunc(strings.ToLower(texto), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	unicas := make([]string, 0, len(campos))
	vistas := map[string]bool{}
	for _, campo := range campos {
		if len(campo) < 3 || vistas[campo] {
			continue
		}
		vistas[campo] = true
		unicas = append(unicas, campo)
	}
	return unicas
}

// SearchKnowledge busca por palavras. Titulo pesa 3, tag pesa 2, conteudo pesa 1.
func (s *InboxStore) SearchKnowledge(organizationID, consulta string, limite int) ([]InboxKnowledgeHit, error) {
	organizacao := consentScope(organizationID)
	termos := palavras(consulta)
	if len(termos) == 0 {
		return nil, nil
	}
	artigos, err := s.loadKnowledge(organizacao)
	if err != nil {
		return nil, err
	}
	buscaveis := make([]string, len(artigos))
	for i, artigo := range artigos {
		buscaveis[i] = strings.ToLower(artigo.Title + " " + strings.Join(artigo.Tags, " ") + " " + artigo.Content)
	}
	hits := make([]InboxKnowledgeHit, 0, len(artigos))
	for i, artigo := range artigos {
		titulo := strings.ToLower(artigo.Title)
		conteudo := strings.ToLower(artigo.Content)
		tags := strings.ToLower(strings.Join(artigo.Tags, " "))
		pontos := 0
		for _, termo := range termos {
			if strings.Contains(titulo, termo) {
				pontos += 3
			}
			if strings.Contains(tags, termo) {
				pontos += 2
			}
			if strings.Contains(conteudo, termo) {
				pontos++
			}
		}
		_ = buscaveis[i]
		if pontos > 0 {
			hits = append(hits, InboxKnowledgeHit{Article: artigo, Score: pontos})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Article.Title < hits[j].Article.Title
		}
		return hits[i].Score > hits[j].Score
	})
	if limite > 0 && len(hits) > limite {
		hits = hits[:limite]
	}
	return hits, nil
}

// SuggestReply monta uma SUGESTAO a partir da ultima mensagem recebida. NAO envia, NAO grava, NAO altera conversa:
// sugerir e LEITURA PURA. Sem correspondencia, Found=false - nao se inventa resposta.
func (s *InboxStore) SuggestReply(organizationID, ultimaMensagem string, limite int) (InboxSuggestion, error) {
	hits, err := s.SearchKnowledge(organizationID, ultimaMensagem, limite)
	if err != nil {
		return InboxSuggestion{}, err
	}
	if len(hits) == 0 {
		return InboxSuggestion{Found: false}, nil
	}
	melhor := hits[0]
	return InboxSuggestion{
		Found:      true,
		ArticleID:  melhor.Article.ID,
		Title:      melhor.Article.Title,
		Suggestion: melhor.Article.Content,
		Score:      melhor.Score,
	}, nil
}
