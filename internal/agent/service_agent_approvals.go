package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Aprovação HUMANA para ferramentas de EFEITO (estágio 35).
//
// O limite declarado era: *"a escrita segue sem aprovação humana"*. Um agente que altera CRM, caixa de
// entrada ou atribuição por conta de uma frase do cliente é exatamente o risco que o documento de missão
// proíbe ("aprovação humana é necessária conforme risco"; `REQUER_APROVACAO` é um dos estados de permissão
// declarados no contexto).
//
// Decisões:
//
//  1. **LEITURA não precisa de aprovação; EFEITO precisa.** Bloquear leitura tornaria o agente inútil sem
//     ganho de segurança.
//  2. **A aprovação é AMARRADA aos PARÂMETROS da ação** (impressão digital de ferramenta + escopo + valores):
//     aprovar "criar oportunidade de 100" não autoriza criar de 10.000.
//  3. **USO ÚNICO**: ao executar, a aprovação é CONSUMIDA. Aprovação reutilizável seria autorização
//     permanente disfarçada de aprovação pontual.
//  4. **PERSISTE por organização**: aprovação que desaparece no reinício obriga o humano a decidir de novo —
//     e quem não entende por quê aprova no automático.
//  5. **NEGADO é decidido, não ignorado**: a tentativa seguinte é BLOQUEADA com o motivo e quem recusou.

// Estados de uma aprovação.
const (
	ServiceAgentApprovalPending  = "pending"
	ServiceAgentApprovalApproved = "approved"
	ServiceAgentApprovalDenied   = "denied"
	ServiceAgentApprovalConsumed = "consumed"
)

// DefaultServiceAgentApprovalRetention é o teto de aprovações guardadas por organização.
const DefaultServiceAgentApprovalRetention = 500

// ServiceAgentApproval é um pedido de autorização para uma ferramenta de efeito.
type ServiceAgentApproval struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	AgentID        string `json:"agent_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	ToolName       string `json:"tool_name"`
	// Fingerprint amarra a aprovação a PARÂMETROS e ESCOPO: aprovar um valor não autoriza outro.
	Fingerprint string     `json:"fingerprint"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	RequestedBy string     `json:"requested_by,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	DecidedBy   string     `json:"decided_by,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
}

// ServiceAgentApprovalStore persiste as aprovações por organização.
type ServiceAgentApprovalStore struct {
	root string
	// MaxPerOrganization limita quantas aprovações cada organização guarda.
	MaxPerOrganization int
}

type serviceAgentApprovalFile struct {
	Approvals []ServiceAgentApproval `json:"approvals"`
}

// NewServiceAgentApprovalStore cria o store de aprovações.
func NewServiceAgentApprovalStore(root string) *ServiceAgentApprovalStore {
	return &ServiceAgentApprovalStore{
		root:               strings.TrimSpace(root),
		MaxPerOrganization: DefaultServiceAgentApprovalRetention,
	}
}

// Root devolve o diretório usado (vazio quando não há persistência).
func (s *ServiceAgentApprovalStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *ServiceAgentApprovalStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve as aprovações da organização.
func (s *ServiceAgentApprovalStore) Load(organizationID string) ([]ServiceAgentApproval, error) {
	path := s.path(organizationID)
	if path == "" {
		return nil, nil
	}
	var file serviceAgentApprovalFile
	err := readJSON(path, &file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		// Arquivo ilegível é REPORTADO: tratar como "sem aprovações" esconderia pedidos pendentes.
		return nil, err
	}
	return file.Approvals, nil
}

// Save grava as aprovações respeitando o teto da organização.
func (s *ServiceAgentApprovalStore) Save(organizationID string, approvals []ServiceAgentApproval) error {
	path := s.path(organizationID)
	if path == "" {
		return nil
	}
	limit := s.MaxPerOrganization
	if limit <= 0 {
		limit = DefaultServiceAgentApprovalRetention
	}
	if len(approvals) > limit {
		// As PENDENTES ficam: descartar um pedido pendente por causa de volume silenciaria uma decisão
		// que alguém precisa tomar. O corte atinge primeiro as já decididas.
		pendentes := make([]ServiceAgentApproval, 0, len(approvals))
		decididas := make([]ServiceAgentApproval, 0, len(approvals))
		for _, approval := range approvals {
			if approval.Status == ServiceAgentApprovalPending {
				pendentes = append(pendentes, approval)
			} else {
				decididas = append(decididas, approval)
			}
		}
		sobra := limit - len(pendentes)
		if sobra < 0 {
			sobra = 0
		}
		if len(decididas) > sobra {
			decididas = decididas[len(decididas)-sobra:]
		}
		approvals = append(pendentes, decididas...)
		sort.SliceStable(approvals, func(i, j int) bool { return approvals[i].RequestedAt.Before(approvals[j].RequestedAt) })
	}
	return writeJSONAtomic(path, serviceAgentApprovalFile{Approvals: approvals})
}

// Request registra um pedido pendente, REUSANDO o pedido já pendente para a mesma impressão digital.
//
// Reusar é deliberado: repetir o turno enquanto o humano decide não pode inundar a caixa de aprovações com
// o mesmo pedido — e um pedido duplicado esconde quantas decisões realmente faltam.
func (s *ServiceAgentApprovalStore) Request(organizationID string, request ServiceAgentApproval) (ServiceAgentApproval, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	for _, existing := range approvals {
		if existing.Status == ServiceAgentApprovalPending && existing.Fingerprint == request.Fingerprint {
			return existing, nil
		}
		// Pedido NEGADO para a mesma impressão digital não vira novo pedido: a decisão foi tomada.
		if existing.Status == ServiceAgentApprovalDenied && existing.Fingerprint == request.Fingerprint {
			return existing, nil
		}
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = "apr_" + uuid.NewString()[:12]
	}
	request.Status = ServiceAgentApprovalPending
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	approvals = append(approvals, request)
	if err := s.Save(organizationID, approvals); err != nil {
		return ServiceAgentApproval{}, err
	}
	return request, nil
}

// Decide registra a decisão humana. Sem ator identificado não há aprovação: não se decide no anonimato.
func (s *ServiceAgentApprovalStore) Decide(organizationID, approvalID string, approved bool, actorID, reason string, now time.Time) (ServiceAgentApproval, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ServiceAgentApproval{}, fmt.Errorf("%w: decidir aprovação exige o ator humano", ErrServiceAgentApprovalInvalid)
	}
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	for index := range approvals {
		if approvals[index].ID != approvalID {
			continue
		}
		if approvals[index].Status == ServiceAgentApprovalConsumed {
			// Decidir sobre aprovação já usada reabriria uma autorização que o sistema consumiu.
			return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação já consumida", ErrServiceAgentApprovalConflict)
		}
		decidedAt := now.UTC()
		approvals[index].DecidedAt = &decidedAt
		approvals[index].DecidedBy = actorID
		approvals[index].Reason = strings.TrimSpace(reason)
		if approved {
			approvals[index].Status = ServiceAgentApprovalApproved
		} else {
			approvals[index].Status = ServiceAgentApprovalDenied
		}
		if err := s.Save(organizationID, approvals); err != nil {
			return ServiceAgentApproval{}, err
		}
		return approvals[index], nil
	}
	return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação %q não existe nesta organização", ErrServiceAgentApprovalNotFound, approvalID)
}

// FindApproved devolve a aprovação APROVADA mais antiga para a impressão digital, se houver.
func (s *ServiceAgentApprovalStore) FindApproved(organizationID, fingerprint string) (ServiceAgentApproval, bool, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, false, err
	}
	for _, approval := range approvals {
		if approval.Status == ServiceAgentApprovalApproved && approval.Fingerprint == fingerprint {
			return approval, true, nil
		}
	}
	return ServiceAgentApproval{}, false, nil
}

// FindDenied diz se a impressão digital já foi NEGADA por um humano.
func (s *ServiceAgentApprovalStore) FindDenied(organizationID, fingerprint string) (ServiceAgentApproval, bool, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, false, err
	}
	for _, approval := range approvals {
		if approval.Status == ServiceAgentApprovalDenied && approval.Fingerprint == fingerprint {
			return approval, true, nil
		}
	}
	return ServiceAgentApproval{}, false, nil
}

// Consume marca a aprovação como USADA e devolve o registro. É o que impede reutilização.
func (s *ServiceAgentApprovalStore) Consume(organizationID, approvalID string, now time.Time) (ServiceAgentApproval, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	for index := range approvals {
		if approvals[index].ID != approvalID {
			continue
		}
		if approvals[index].Status != ServiceAgentApprovalApproved {
			// Sem esta checagem, duas execuções simultâneas consumiriam a mesma aprovação.
			return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação não está aprovada", ErrServiceAgentApprovalConflict)
		}
		consumedAt := now.UTC()
		approvals[index].Status = ServiceAgentApprovalConsumed
		approvals[index].ConsumedAt = &consumedAt
		if err := s.Save(organizationID, approvals); err != nil {
			return ServiceAgentApproval{}, err
		}
		return approvals[index], nil
	}
	return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação %q não existe nesta organização", ErrServiceAgentApprovalNotFound, approvalID)
}

// ListApprovals lista as aprovações da organização por estado (vazio lista todas), mais recentes primeiro.
func (s *ServiceAgentApprovalStore) ListApprovals(organizationID, status string) ([]ServiceAgentApproval, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	out := make([]ServiceAgentApproval, 0, len(approvals))
	for _, approval := range approvals {
		if approval.OrganizationID != organizationID {
			continue
		}
		if status != "" && approval.Status != status {
			continue
		}
		out = append(out, approval)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RequestedAt.After(out[j].RequestedAt) })
	return out, nil
}

// Erros dedicados das aprovações.
var (
	ErrServiceAgentApprovalInvalid  = errors.New("service agent approval request is invalid")
	ErrServiceAgentApprovalNotFound = errors.New("service agent approval not found")
	ErrServiceAgentApprovalConflict = errors.New("service agent approval state conflict")
)

// serviceAgentApprovalFingerprint amarra a aprovação a FERRAMENTA + ESCOPO + PARÂMETROS.
//
// Só a ferramenta não basta: "criar oportunidade" com 100 centavos e com 10.000 centavos são decisões
// diferentes, e aprovar uma não pode liberar a outra. Só o escopo também não basta — o humano aprova uma
// AÇÃO, não um agente.
func serviceAgentApprovalFingerprint(request ServiceAgentToolRequest, toolName string) string {
	partes := []string{
		strings.TrimSpace(toolName),
		"org=" + consentScope(request.OrganizationID),
		"conversation=" + strings.TrimSpace(request.ConversationID),
		"contact=" + strings.TrimSpace(request.ContactID),
		"title=" + strings.TrimSpace(request.Title),
		fmt.Sprintf("value=%d", request.ValueCents),
		"owner=" + strings.TrimSpace(request.OwnerID),
		"assignee=" + strings.TrimSpace(request.Assignee),
	}
	soma := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return hex.EncodeToString(soma[:])
}

// serviceAgentApprovalRoot devolve o diretório das aprovações dentro da raiz de dados.
func serviceAgentApprovalRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-service-approvals")
}
