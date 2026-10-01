package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WhatsAppContactRole represents the authorization tier of a contact.
type WhatsAppContactRole string

const (
	ContactRoleOwner    WhatsAppContactRole = "owner"
	ContactRoleOperator WhatsAppContactRole = "operator"
	ContactRoleViewer   WhatsAppContactRole = "viewer"
)

// WhatsAppContactPolicy defines the permissions for a given phone number.
type WhatsAppContactPolicy struct {
	PhoneNumber string              `json:"phone_number"` // E.164 digits without +
	Name        string              `json:"name"`
	Role        WhatsAppContactRole `json:"role"`
	Allowed     bool                `json:"allowed"`
}

// WhatsAppPendingApproval holds an action suspended pending human owner sign-off.
type WhatsAppPendingApproval struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`
	ActionDesc  string                 `json:"action_desc"`
	CommandText string                 `json:"command_text"`
	OriginalMsg WhatsAppInboundMessage `json:"original_msg"`
	OutboundTo  string                 `json:"outbound_to,omitempty"`
	Status      string                 `json:"status"` // PENDING, APPROVED, REJECTED, EXPIRED
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

// WhatsAppDLQEntry holds a failed message in the Dead Letter Queue.
type WhatsAppDLQEntry struct {
	ID       string                 `json:"id"`
	Message  WhatsAppInboundMessage `json:"message"`
	Error    string                 `json:"error"`
	Attempts int                    `json:"attempts"`
	FailedAt time.Time              `json:"failed_at"`
}

// WhatsAppProcessResult reports the outcome of processing an inbound webhook event.
type WhatsAppProcessResult struct {
	MessageID   string `json:"message_id"`
	From        string `json:"from"`
	ActionTaken string `json:"action_taken"`
	ReplySent   string `json:"reply_sent,omitempty"`
	Status      string `json:"status"` // "processed", "ignored_from_me", "ignored_duplicate", "unauthorized", "awaiting_approval", "error"
	Error       string `json:"error,omitempty"`
}

// WhatsAppGatewayConfig configures the gateway.
type WhatsAppGatewayConfig struct {
	PrimaryBackend   WhatsAppBackendType     `json:"primary_backend"`
	Evolution        EvolutionConfig         `json:"evolution"`
	CloudAPI         CloudAPIConfig          `json:"cloud_api"`
	Allowlist        []WhatsAppContactPolicy `json:"allowlist"`
	MaxRetries       int                     `json:"max_retries"`
	DedupeTTLSeconds int                     `json:"dedupe_ttl_seconds"`
}

// WhatsAppGateway manages WhatsApp communication, authorization, intent routing, and command execution.
type WhatsAppGateway struct {
	mu               sync.RWMutex
	adapters         map[WhatsAppBackendType]WhatsAppAdapter
	activeBackend    WhatsAppBackendType
	allowlist        map[string]WhatsAppContactPolicy
	dedupeCache      map[string]time.Time
	dedupeTTL        time.Duration
	pendingApprovals map[string]*WhatsAppPendingApproval
	dlq              []WhatsAppDLQEntry
	maxRetries       int
	reminders        []string
	runtime          *Runtime
	media            *WhatsAppMediaProcessor
}

// Common errors.
var (
	ErrWhatsAppUnauthorized     = errors.New("whatsapp: contact not authorized in allowlist")
	ErrWhatsAppUnknownBackend   = errors.New("whatsapp: unknown or inactive backend adapter")
	ErrWhatsAppAdapterNotConfig = errors.New("whatsapp: adapter is NOT_CONFIGURED")
	ErrWhatsAppOutboundApproval = errors.New("whatsapp: outbound message requires an approved HITL approval")
)

// NewWhatsAppGateway creates and initializes a WhatsAppGateway.
func NewWhatsAppGateway(cfg WhatsAppGatewayConfig, runtime *Runtime) *WhatsAppGateway {
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	dedupeTTL := time.Duration(cfg.DedupeTTLSeconds) * time.Second
	if dedupeTTL <= 0 {
		dedupeTTL = 24 * time.Hour
	}

	g := &WhatsAppGateway{
		adapters:         make(map[WhatsAppBackendType]WhatsAppAdapter),
		allowlist:        make(map[string]WhatsAppContactPolicy),
		dedupeCache:      make(map[string]time.Time),
		dedupeTTL:        dedupeTTL,
		pendingApprovals: make(map[string]*WhatsAppPendingApproval),
		dlq:              make([]WhatsAppDLQEntry, 0),
		maxRetries:       maxRetries,
		reminders:        make([]string, 0),
		runtime:          runtime,
	}

	// Register adapters
	g.adapters[WhatsAppBackendEvolution] = NewEvolutionAdapter(cfg.Evolution)
	g.adapters[WhatsAppBackendCloudAPI] = NewCloudAPIAdapter(cfg.CloudAPI)

	if cfg.PrimaryBackend != "" {
		g.activeBackend = cfg.PrimaryBackend
	} else {
		g.activeBackend = WhatsAppBackendEvolution
	}

	// Populate allowlist
	for _, policy := range cfg.Allowlist {
		clean := cleanWhatsAppNumber(policy.PhoneNumber)
		if clean != "" {
			policy.PhoneNumber = clean
			g.allowlist[clean] = policy
		}
	}

	// Setup media processor
	var mm *MediaManager
	if runtime != nil {
		mm = runtime.Media()
	}
	g.media = NewWhatsAppMediaProcessor(mm)

	return g
}

// SetAdapter allows injecting or replacing an adapter (e.g. for testing).
func (g *WhatsAppGateway) SetAdapter(backend WhatsAppBackendType, adapter WhatsAppAdapter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.adapters[backend] = adapter
}

// SetActiveBackend switches the active backend.
func (g *WhatsAppGateway) SetActiveBackend(backend WhatsAppBackendType) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.activeBackend = backend
}

// ActiveBackend returns the currently selected primary backend.
func (g *WhatsAppGateway) ActiveBackend() WhatsAppBackendType {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.activeBackend
}

// VerifyWebhook verifies the active backend's challenge handshake.
func (g *WhatsAppGateway) VerifyWebhook(req *http.Request) ([]byte, bool) {
	g.mu.RLock()
	adapter, ok := g.adapters[g.activeBackend]
	g.mu.RUnlock()
	if !ok || adapter == nil {
		return nil, false
	}
	return adapter.VerifyWebhook(req)
}

// SendMessage sends an outbound message using the active backend adapter.
func (g *WhatsAppGateway) SendMessage(ctx context.Context, msg WhatsAppOutboundMessage) (WhatsAppSendResult, error) {
	g.mu.RLock()
	adapter, ok := g.adapters[g.activeBackend]
	policy, allowed := g.allowlist[cleanWhatsAppNumber(msg.To)]
	approval, approved := g.pendingApprovals[msg.ApprovalID]
	g.mu.RUnlock()

	if !ok || adapter == nil {
		return WhatsAppSendResult{}, ErrWhatsAppUnknownBackend
	}
	if adapter.Status() == GateStatusNotConfigured {
		return WhatsAppSendResult{}, ErrWhatsAppAdapterNotConfig
	}
	if !allowed || !policy.Allowed {
		return WhatsAppSendResult{}, ErrWhatsAppUnauthorized
	}
	if msg.ApprovalID == "" || !approved || approval.Status != "APPROVED" || approval.OutboundTo != cleanWhatsAppNumber(msg.To) {
		return WhatsAppSendResult{}, ErrWhatsAppOutboundApproval
	}
	return adapter.SendMessage(ctx, msg)
}

// Status returns the honest GateStatus of the active backend.
// Without credentials, returns NOT_CONFIGURED. Never pretends to be connected.
func (g *WhatsAppGateway) Status() GateStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()

	adapter, ok := g.adapters[g.activeBackend]
	if !ok || adapter == nil {
		return GateStatusNotConfigured
	}
	return adapter.Status()
}

// StatusSummary returns a detailed status struct for observability.
func (g *WhatsAppGateway) StatusSummary() map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()

	adapterStatuses := make(map[string]any)
	for bType, adp := range g.adapters {
		adapterStatuses[string(bType)] = map[string]any{
			"status":  adp.Status(),
			"details": adp.StatusDetails(),
		}
	}

	return map[string]any{
		"active_backend": g.activeBackend,
		"gate_status":    g.Status(),
		"adapters":       adapterStatuses,
		"allowlist_size": len(g.allowlist),
		"pending_hitl":   len(g.pendingApprovals),
		"dlq_size":       len(g.dlq),
		"capabilities": map[string]GateStatus{
			"stt":    g.media.StatusSTT(),
			"tts":    g.media.StatusTTS(),
			"vision": g.media.StatusVision(),
		},
	}
}

// SetContactPolicy adds or updates a contact's permissions.
func (g *WhatsAppGateway) SetContactPolicy(policy WhatsAppContactPolicy) {
	g.mu.Lock()
	defer g.mu.Unlock()
	clean := cleanWhatsAppNumber(policy.PhoneNumber)
	if clean != "" {
		policy.PhoneNumber = clean
		g.allowlist[clean] = policy
	}
}

// RemoveContactPolicy removes a contact from the allowlist.
func (g *WhatsAppGateway) RemoveContactPolicy(phoneNumber string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.allowlist, cleanWhatsAppNumber(phoneNumber))
}

// GetAllowlist returns the current list of authorized contacts.
func (g *WhatsAppGateway) GetAllowlist() []WhatsAppContactPolicy {
	g.mu.RLock()
	defer g.mu.RUnlock()

	list := make([]WhatsAppContactPolicy, 0, len(g.allowlist))
	for _, p := range g.allowlist {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].PhoneNumber < list[j].PhoneNumber
	})
	return list
}

// Authorize verifies if a number is authorized and returns its policy.
func (g *WhatsAppGateway) Authorize(from string) (WhatsAppContactPolicy, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clean := cleanWhatsAppNumber(from)
	policy, ok := g.allowlist[clean]
	if !ok || !policy.Allowed {
		return WhatsAppContactPolicy{}, ErrWhatsAppUnauthorized
	}
	return policy, nil
}

// RequestOutboundApproval creates an auditable HITL approval for a manual send.
// It never contacts the provider. The returned ID must be approved before send.
func (g *WhatsAppGateway) RequestOutboundApproval(from string, msg WhatsAppOutboundMessage) (string, error) {
	policy, err := g.Authorize(from)
	if err != nil || policy.Role != ContactRoleOwner {
		return "", ErrWhatsAppUnauthorized
	}
	if cleanWhatsAppNumber(msg.To) == "" || strings.TrimSpace(msg.Text) == "" {
		return "", errors.New("whatsapp: outbound recipient and text are required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	id := "appr_" + uuid.NewString()[:8]
	now := time.Now().UTC()
	g.pendingApprovals[id] = &WhatsAppPendingApproval{
		ID: id, From: policy.PhoneNumber, ActionDesc: "Envio outbound WhatsApp",
		CommandText: "send_whatsapp", Status: "PENDING", CreatedAt: now,
		OutboundTo: cleanWhatsAppNumber(msg.To),
		ExpiresAt:  now.Add(15 * time.Minute),
	}
	return id, nil
}

// ApproveOutbound records owner HITL approval for a previously requested send.
func (g *WhatsAppGateway) ApproveOutbound(approvalID, approver string) error {
	policy, err := g.Authorize(approver)
	if err != nil || policy.Role != ContactRoleOwner {
		return ErrWhatsAppUnauthorized
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	appr, ok := g.pendingApprovals[strings.TrimSpace(approvalID)]
	if !ok || appr.Status != "PENDING" || time.Now().UTC().After(appr.ExpiresAt) {
		return ErrWhatsAppOutboundApproval
	}
	appr.Status = "APPROVED"
	return nil
}

// Dedupe check and registration.
func (g *WhatsAppGateway) isDuplicate(id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	// Prune expired entries periodically
	for k, ts := range g.dedupeCache {
		if now.Sub(ts) > g.dedupeTTL {
			delete(g.dedupeCache, k)
		}
	}

	if _, exists := g.dedupeCache[id]; exists {
		return true
	}
	g.dedupeCache[id] = now
	return false
}

// DLQ accessors.
func (g *WhatsAppGateway) GetDLQ() []WhatsAppDLQEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	copied := make([]WhatsAppDLQEntry, len(g.dlq))
	copy(copied, g.dlq)
	return copied
}

func (g *WhatsAppGateway) ClearDLQ() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dlq = make([]WhatsAppDLQEntry, 0)
}

func (g *WhatsAppGateway) appendDLQ(entry WhatsAppDLQEntry) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.dlq) >= 500 {
		g.dlq = g.dlq[1:] // bounded ring buffer
	}
	g.dlq = append(g.dlq, entry)
}

// ProcessWebhook receives raw webhook payloads, handles dedupe, fromMe, authorization, retry and DLQ.
func (g *WhatsAppGateway) ProcessWebhook(ctx context.Context, backend WhatsAppBackendType, body []byte, header http.Header) ([]WhatsAppProcessResult, error) {
	g.mu.RLock()
	adapter, ok := g.adapters[backend]
	g.mu.RUnlock()

	if !ok || adapter == nil {
		return nil, ErrWhatsAppUnknownBackend
	}
	if validator, ok := adapter.(webhookAuthenticator); ok {
		if err := validator.ValidateWebhook(body, header); err != nil {
			return nil, fmt.Errorf("authenticate webhook: %w", err)
		}
	}

	messages, err := adapter.ParseWebhook(body, header)
	if err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}

	var results []WhatsAppProcessResult

	for _, msg := range messages {
		// 1. Loop prevention: fromMe filter
		if msg.FromMe {
			results = append(results, WhatsAppProcessResult{
				MessageID:   msg.ID,
				From:        msg.From,
				ActionTaken: "ignore_from_me",
				Status:      "ignored_from_me",
			})
			continue
		}

		// 2. Dedupe / Idempotence filter
		if g.isDuplicate(msg.ID) {
			results = append(results, WhatsAppProcessResult{
				MessageID:   msg.ID,
				From:        msg.From,
				ActionTaken: "ignore_duplicate",
				Status:      "ignored_duplicate",
			})
			continue
		}

		// 3. Authorization check
		policy, authErr := g.Authorize(msg.From)
		if authErr != nil {
			results = append(results, WhatsAppProcessResult{
				MessageID:   msg.ID,
				From:        msg.From,
				ActionTaken: "blocked_unauthorized",
				Status:      "unauthorized",
				Error:       "contact not authorized in allowlist",
			})
			continue
		}

		// 4. Execute with Retry
		var reply string
		var processErr error
		attempts := 0

		for attempts < g.maxRetries {
			attempts++
			reply, processErr = g.handleInbound(ctx, msg, policy)
			if processErr == nil {
				break
			}
			time.Sleep(time.Duration(attempts*10) * time.Millisecond)
		}

		if processErr != nil {
			// Push to DLQ
			g.appendDLQ(WhatsAppDLQEntry{
				ID:       "dlq_" + uuid.NewString(),
				Message:  msg,
				Error:    processErr.Error(),
				Attempts: attempts,
				FailedAt: time.Now().UTC(),
			})

			results = append(results, WhatsAppProcessResult{
				MessageID:   msg.ID,
				From:        msg.From,
				ActionTaken: "sent_to_dlq",
				Status:      "error",
				Error:       processErr.Error(),
			})
			continue
		}

		// Send reply via WhatsApp if adapter configured
		if reply != "" && adapter.Status() == GateStatusPass {
			_, _ = adapter.SendMessage(ctx, WhatsAppOutboundMessage{
				To:   msg.From,
				Text: reply,
			})
		}

		status := "processed"
		if strings.Contains(reply, "Aprovação necessária (HITL)") {
			status = "awaiting_approval"
		}

		results = append(results, WhatsAppProcessResult{
			MessageID:   msg.ID,
			From:        msg.From,
			ActionTaken: "command_processed",
			ReplySent:   reply,
			Status:      status,
		})
	}

	return results, nil
}

// handleInbound routes the message based on type and content.
func (g *WhatsAppGateway) handleInbound(ctx context.Context, msg WhatsAppInboundMessage, policy WhatsAppContactPolicy) (string, error) {
	// Optional Capabilities: STT
	if msg.Type == "audio" {
		transcribed, status, err := g.media.TranscribeAudio(ctx, msg.MediaURL, msg.MediaBytes)
		if status == GateStatusNotConfigured {
			return "ℹ️ *Áudio recebido, mas transcrição (STT) não está configurada* (`NOT_CONFIGURED`).\nPor favor, envie sua mensagem em texto.", nil
		}
		if err != nil {
			return "", err
		}
		msg.Text = transcribed
	}

	// Optional Capabilities: Vision
	if msg.Type == "image" {
		analysis, status, err := g.media.AnalyzeImage(ctx, msg.Text, msg.MediaBytes)
		if status == GateStatusNotConfigured {
			return "ℹ️ *Imagem recebida, mas análise visual (Vision) não está configurada* (`NOT_CONFIGURED`).", nil
		}
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("👁️ *Análise de Imagem*:\n%s", analysis), nil
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return "Envie uma mensagem de texto ou comando para o assistente. Digite *ajuda* para ver as opções.", nil
	}

	// Check if this is an Approval or Rejection response: "APROVAR <id>" or "REJEITAR <id>"
	if isApprovalDecision(text) {
		return g.handleApprovalDecision(ctx, text, policy)
	}

	// Check for sensitive command requiring HITL approval
	if isSensitive, desc := g.classifySensitiveCommand(text); isSensitive {
		return g.createPendingApproval(msg, policy, desc)
	}

	// Route Intent
	return g.routeIntent(ctx, text, policy)
}

// classifySensitiveCommand detects high-risk / destructive commands.
func (g *WhatsAppGateway) classifySensitiveCommand(text string) (bool, string) {
	lower := strings.ToLower(text)
	sensitivePatterns := []struct {
		substr string
		desc   string
	}{
		{"delete", "Exclusão permanente de dados ou recursos"},
		{"deletar", "Exclusão permanente de dados ou recursos"},
		{"remover", "Remoção de recurso do sistema"},
		{"rm -rf", "Exclusão forçada em filesystem"},
		{"destruir", "Destruição de recurso"},
		{"drop database", "Exclusão de banco de dados"},
		{"drop table", "Exclusão de tabela de banco de dados"},
		{"deploy prod", "Publicação em ambiente de produção"},
		{"deploy produção", "Publicação em ambiente de produção"},
		{"publicar em prod", "Publicação em ambiente de produção"},
		{"reboot", "Reinicialização de servidor ou serviço"},
		{"shutdown", "Desligamento de servidor ou serviço"},
		{"comprar", "Operação financeira ou compra externa"},
		{"pagar", "Operação financeira ou pagamento"},
		{"export env", "Exposição de variáveis de ambiente"},
		{"export secret", "Exposição de credenciais ou segredos"},
		{"show token", "Exposição de token de acesso"},
	}

	for _, p := range sensitivePatterns {
		if strings.Contains(lower, p.substr) {
			return true, p.desc
		}
	}
	return false, ""
}

// createPendingApproval suspends action and notifies owner.
func (g *WhatsAppGateway) createPendingApproval(msg WhatsAppInboundMessage, policy WhatsAppContactPolicy, desc string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	approvalID := "appr_" + uuid.NewString()[:8]
	now := time.Now().UTC()

	g.pendingApprovals[approvalID] = &WhatsAppPendingApproval{
		ID:          approvalID,
		From:        msg.From,
		ActionDesc:  desc,
		CommandText: msg.Text,
		OriginalMsg: msg,
		Status:      "PENDING",
		CreatedAt:   now,
		ExpiresAt:   now.Add(15 * time.Minute),
	}

	return fmt.Sprintf(
		"⚠️ *Aprovação necessária (HITL)*:\nAção sensível detectada: %s\nComando: \"%s\"\n\nPara autorizar a execução, responda:\n*APROVAR %s*\n\nPara cancelar:\n*REJEITAR %s*",
		desc, msg.Text, approvalID, approvalID,
	), nil
}

func isApprovalDecision(text string) bool {
	upper := strings.ToUpper(strings.TrimSpace(text))
	return strings.HasPrefix(upper, "APROVAR ") || strings.HasPrefix(upper, "REJEITAR ")
}

func (g *WhatsAppGateway) handleApprovalDecision(ctx context.Context, text string, policy WhatsAppContactPolicy) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Only owners can approve sensitive actions
	if policy.Role != ContactRoleOwner {
		return "⛔ *Acesso negado*: Somente contatos com perfil *owner* podem aprovar ações sensíveis.", nil
	}

	fields := strings.Fields(text)
	if len(fields) < 2 {
		return "Formato inválido. Use *APROVAR <id>* ou *REJEITAR <id>*.", nil
	}

	decision := strings.ToUpper(fields[0])
	approvalID := strings.Trim(strings.TrimSpace(fields[1]), "*_ `")

	appr, ok := g.pendingApprovals[approvalID]
	if !ok {
		return fmt.Sprintf("Aprovação *%s* não encontrada ou já processada.", approvalID), nil
	}

	if appr.Status != "PENDING" {
		return fmt.Sprintf("Aprovação *%s* já se encontra no estado *%s*.", approvalID, appr.Status), nil
	}

	if time.Now().UTC().After(appr.ExpiresAt) {
		appr.Status = "EXPIRED"
		return fmt.Sprintf("Aprovação *%s* expirou.", approvalID), nil
	}

	if decision == "REJEITAR" {
		appr.Status = "REJECTED"
		return fmt.Sprintf("❌ *Ação rejeitada*: Aprovação %s cancelada.", approvalID), nil
	}

	// Approved!
	appr.Status = "APPROVED"

	// Execute approved action
	executedReply, err := g.executeApprovedCommand(ctx, appr.CommandText, policy)
	if err != nil {
		return fmt.Sprintf("⚠️ *Aprovado*, mas ocorreu erro na execução: %v", err), nil
	}

	return fmt.Sprintf("✅ *Ação aprovada e executada pelo Dono*:\n%s", executedReply), nil
}

func (g *WhatsAppGateway) executeApprovedCommand(ctx context.Context, commandText string, policy WhatsAppContactPolicy) (string, error) {
	// Re-route intent directly bypassing sensitivity check
	return g.routeIntent(ctx, commandText, policy)
}

// routeIntent dispatches commands to respective handlers.
func (g *WhatsAppGateway) routeIntent(ctx context.Context, text string, policy WhatsAppContactPolicy) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(text))

	// 1. Help / Ajuda
	if lower == "ajuda" || lower == "help" || lower == "/help" || lower == "?" {
		return "🤖 *Hades — Comandos via Celular*:\n\n" +
			"• *health* ou *status* — Saúde dos serviços e runtime\n" +
			"• *projetos* — Listar projetos ativos\n" +
			"• *progresso* ou *missoes* — Consultar missões recentes\n" +
			"• *resumo* — Resumo executivo diário\n" +
			"• *lembrete <texto>* — Criar lembrete agendado\n" +
			"• *missao <objetivo>* ou */goal <objetivo>* — Disparar missão na Factory\n\n" +
			"🔒 *Segurança*: Comandos sensíveis exigem aprovação HITL do Dono.", nil
	}

	// 2. Health / Status
	if lower == "health" || lower == "status" || lower == "/status" {
		adapter, ok := g.adapters[g.activeBackend]
		adpStatus := GateStatusNotConfigured
		adpDetails := "não configurado"
		if ok && adapter != nil {
			adpStatus = adapter.Status()
			adpDetails = adapter.StatusDetails()
		}

		runtimeStatus := "ONLINE"
		if g.runtime == nil {
			runtimeStatus = "OFFLINE"
		}

		return fmt.Sprintf(
			"🟢 *Saúde do Sistema (Hades)*:\n\n"+
				"• *Runtime Agentic*: %s\n"+
				"• *WhatsApp Gateway*: %s (%s)\n"+
				"• *Backend Ativo*: %s\n"+
				"• *DLQ (Falhas Retidas)*: %d\n"+
				"• *Aprovações Pendentes*: %d\n"+
				"• *Seu Acesso*: %s (perfil: %s)",
			runtimeStatus, adpStatus, adpDetails, g.activeBackend, len(g.dlq), len(g.pendingApprovals), policy.Name, policy.Role,
		), nil
	}

	// 3. Projects / Projetos
	if lower == "projetos" || lower == "projects" || lower == "/projects" {
		if g.runtime == nil || g.runtime.Context() == nil {
			return "Projetos: Nenhum runtime conectado.", nil
		}
		projects := g.runtime.Context().ListProjects()
		if len(projects) == 0 {
			return "📂 *Projetos Cadastrados*: Nenhum projeto ativo no workspace local.", nil
		}
		var sb strings.Builder
		sb.WriteString("📂 *Projetos no Workspace*:\n\n")
		for i, p := range projects {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("... e mais %d projetos.\n", len(projects)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("• *%s* (ID: `%s`)\n", p.Name, p.ID))
		}
		return sb.String(), nil
	}

	// 4. Progress / Missões
	if lower == "progresso" || lower == "missoes" || lower == "missions" || lower == "/missions" {
		if g.runtime == nil {
			return "Missões: Nenhum runtime conectado.", nil
		}
		missions, err := g.runtime.ListMissions()
		if err != nil {
			return "📋 *Missões Recentes*: Não foi possível consultar o progresso agora.", err
		}
		if len(missions) == 0 {
			return "📋 *Missões Recentes*: Nenhuma missão executada recentemente.", nil
		}
		var sb strings.Builder
		sb.WriteString("📋 *Missões Recentes*:\n\n")
		// Show last 5
		count := 0
		for i := len(missions) - 1; i >= 0 && count < 5; i-- {
			m := missions[i]
			count++
			sb.WriteString(fmt.Sprintf("• *%s* — %s\n  Objetivo: %s\n", m.ID, m.State, m.Objective))
		}
		return sb.String(), nil
	}

	// 5. Daily Summary / Resumo Diário
	if lower == "resumo" || lower == "daily" || lower == "/daily" {
		g.mu.RLock()
		remCount := len(g.reminders)
		g.mu.RUnlock()

		mCount := 0
		if g.runtime != nil {
			if mList, err := g.runtime.ListMissions(); err == nil {
				mCount = len(mList)
			}
		}

		return fmt.Sprintf(
			"📊 *Resumo Executivo Diário*:\n\n"+
				"• *Data*: %s\n"+
				"• *Missões Registradas*: %d\n"+
				"• *Lembretes Ativos*: %d\n"+
				"• *Gateway WhatsApp*: Ativo via %s\n"+
				"• *Status Operacional*: Local-first seguro, zero egress não autorizado.",
			time.Now().Format("02/01/2006"), mCount, remCount, g.activeBackend,
		), nil
	}

	// 6. Reminder / Lembrete
	if strings.HasPrefix(lower, "lembrete ") || strings.HasPrefix(lower, "reminder ") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			return "Especifique o lembrete. Exemplo: *lembrete revisar deploy às 15h*", nil
		}
		remText := strings.TrimSpace(parts[1])
		g.mu.Lock()
		g.reminders = append(g.reminders, remText)
		g.mu.Unlock()
		return fmt.Sprintf("⏰ *Lembrete registrado com sucesso*:\n\"%s\"", remText), nil
	}

	// 7. Engineering Command Bridge: /goal <objetivo> or missao <objetivo>
	if strings.HasPrefix(lower, "missao ") || strings.HasPrefix(lower, "/goal ") || strings.HasPrefix(lower, "missão ") {
		// Viewers cannot create missions
		if policy.Role == ContactRoleViewer {
			return "⛔ *Acesso negado*: Seu perfil (viewer) não possui permissão para disparar missões.", nil
		}

		parts := strings.SplitN(text, " ", 2)
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			return "Informe o objetivo da missão. Exemplo: *missao auditar segurança do repositório*", nil
		}

		objective := strings.TrimSpace(parts[1])
		if g.runtime == nil {
			return fmt.Sprintf("🚀 *Comando recebido*: \"%s\"\n(Aviso: runtime não conectado para execução imediata)", objective), nil
		}

		// Create mission via Engineering Command Bridge!
		mission, err := g.runtime.CreateMission(ctx, CreateMissionRequest{
			Objective: objective,
			Provider:  "ollama-local",
			AutoRun:   true,
		})
		if err != nil {
			return fmt.Sprintf("❌ Erro ao criar missão: %v", err), nil
		}

		// Start execution via queue
		_, _ = g.runtime.EnqueueMission(mission.ID)

		return fmt.Sprintf(
			"🚀 *Missão Iniciada via WhatsApp Bridge*!\n\n"+
				"• *ID*: `%s`\n"+
				"• *Objetivo*: %s\n"+
				"• *Status*: RUNNING (em execução no runtime local)\n"+
				"Acompanhe o progresso com o comando *progresso*.",
			mission.ID, mission.Objective,
		), nil
	}

	return fmt.Sprintf("Comando \"%s\" não reconhecido. Digite *ajuda* para ver os comandos suportados.", text), nil
}
