package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CHAMADA ATIVA (OUTBOUND) AUTORIZADA — estágio 37 / §11.6.
//
// O estágio 37 diz: "Chamadas ativas só para finalidades autorizadas e contatos elegíveis", e nomeia o risco
// "Chamada ativa sem consentimento". O módulo de voz já tinha latência e gravação com consentimento, mas **não
// tinha nenhum conceito de chamada ATIVA** — o Hades ligando para alguém é o efeito externo de maior risco do
// estágio, e não havia portão nenhum.
//
// Este arquivo implementa o portão com três exigências INDEPENDENTES, todas com recusa EXPLÍCITA:
//
//  1. **FINALIDADE AUTORIZADA** — só uma lista declarada pode originar ligação. Finalidade desconhecida é recusada,
//     porque "o modelo pediu" não é finalidade;
//  2. **CONTATO ELEGÍVEL** — consentimento CONCEDIDO e não revogado/expirado, verificado pelo `ConsentRegistry` que
//     já existe (não se inventa segundo registro de consentimento);
//  3. **APROVAÇÃO HUMANA** — a ligação nasce PENDENTE; sem aprovação registrada ela não é executada.
//
// E um TETO por janela, porque "proteção contra abuso" está no mesmo estágio: uma enxurrada de ligações é abuso
// mesmo com consentimento. A recusa é EXPLÍCITA, nunca silenciosa.
//
// LIMITE DECLARADO: isto é a **política local** do Hades e o **registro** da autorização. A ligação REAL depende de
// provedor, número e credencial — segue `BLOCKED_BY_EXTERNAL_DEPENDENCY`. Não se finge que aqui já se telefonou.
const (
	// VoiceActiveCallLimitPerWindow é o teto de chamadas ativas autorizadas por janela, por organização.
	VoiceActiveCallLimitPerWindow = 3
	// VoiceActiveCallWindow é a janela do teto.
	VoiceActiveCallWindow = 24 * time.Hour

	// VoiceActiveCallPending é o estado de quem pediu a ligação e ainda não recebeu aprovação humana.
	VoiceActiveCallPending = "pending_approval"
	// VoiceActiveCallAuthorized é o estado da ligação autorizada (pronta para o provedor executar).
	VoiceActiveCallAuthorized = "authorized"
)

var (
	// ErrVoiceActiveCallPurposeNotAllowed recusa finalidade que não está na lista declarada.
	ErrVoiceActiveCallPurposeNotAllowed = errors.New("voice active call purpose is not allowed")
	// ErrVoiceActiveCallRequiresApproval recusa execução sem aprovação humana registrada.
	ErrVoiceActiveCallRequiresApproval = errors.New("voice active call requires human approval")
	// ErrVoiceActiveCallThrottled recusa a enxurrada de ligações, com motivo explícito.
	ErrVoiceActiveCallThrottled = errors.New("voice active calls throttled by local policy")
	// ErrVoiceActiveCallInvalid recusa pedido malformado.
	ErrVoiceActiveCallInvalid = errors.New("voice active call request is invalid")
)

// VoiceActiveCallAllowedPurposes é a lista DECLARADA de finalidades que podem originar ligação. Estar fora dela é
// recusa — o propósito não pode ser inventado pelo modelo nem pelo texto do contato.
var VoiceActiveCallAllowedPurposes = []string{
	"order_confirmation",
	"appointment_reminder",
	"support_followup",
	"delivery_notice",
}

// VoiceActiveCall é o registro auditável de um pedido de ligação e do desfecho dele.
type VoiceActiveCall struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ContactID      string     `json:"contact_id"`
	Channel        string     `json:"channel"`
	Purpose        string     `json:"purpose"`
	RequestedBy    string     `json:"requested_by,omitempty"`
	Status         string     `json:"status"`
	Reason         string     `json:"reason,omitempty"`
	ApprovedBy     string     `json:"approved_by,omitempty"`
	RequestedAt    time.Time  `json:"requested_at"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`
	// ProviderExecutedAt fica NIL de propósito: quem liga é o provedor externo. Registro honesto do que NÃO
	// aconteceu aqui em vez de um campo que finge execução.
	ProviderExecutedAt *time.Time `json:"provider_executed_at,omitempty"`
}

// VoiceActiveCallStore guarda as autorizações por organização, com escrita atômica.
type VoiceActiveCallStore struct {
	root string
}

type voiceActiveCallFile struct {
	Calls []VoiceActiveCall `json:"calls"`
}

// NewVoiceActiveCallStore cria o store no diretório informado.
func NewVoiceActiveCallStore(root string) *VoiceActiveCallStore {
	return &VoiceActiveCallStore{root: strings.TrimSpace(root)}
}

// Root devolve o diretório do store.
func (s *VoiceActiveCallStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *VoiceActiveCallStore) path(organizationID string) string {
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Calls devolve as chamadas registradas da organização (mais recentes primeiro é irrelevante: a ordem do arquivo é a
// ordem dos pedidos, e quem lê decide o que fazer com ela).
func (s *VoiceActiveCallStore) Calls(organizationID string) ([]VoiceActiveCall, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	if s == nil || s.root == "" {
		return nil, nil
	}
	dados, err := os.ReadFile(s.path(organizationID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(dados) == 0 {
		return nil, nil
	}
	var arquivo voiceActiveCallFile
	if err := json.Unmarshal(dados, &arquivo); err != nil {
		return nil, err
	}
	return arquivo.Calls, nil
}

func (s *VoiceActiveCallStore) save(organizationID string, calls []VoiceActiveCall) error {
	if s == nil || s.root == "" {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(s.path(organizationID), voiceActiveCallFile{Calls: calls})
}

// voiceActiveCallPurposeAllowed diz se a finalidade está na lista declarada.
func voiceActiveCallPurposeAllowed(purpose string) bool {
	alvo := strings.ToLower(strings.TrimSpace(purpose))
	for _, permitida := range VoiceActiveCallAllowedPurposes {
		if alvo == permitida {
			return true
		}
	}
	return false
}

// RequestActiveCall pede uma ligação ATIVA.
//
// A ordem dos portões é deliberada: **finalidade primeiro** (o que não é permitido não chega a tocar consentimento),
// depois **elegibilidade do contato** pelo registro de consentimento que já existe, depois **teto da janela**, e por
// último a **aprovação humana** — o pedido nasce PENDENTE, e o efeito externo só é liberado por `ApproveActiveCall`.
func (s *VoiceActiveCallStore) RequestActiveCall(
	organizationID string,
	request VoiceActiveCall,
	registry *ConsentRegistry,
	now time.Time,
) (VoiceActiveCall, error) {
	if s == nil {
		return VoiceActiveCall{}, ErrVoiceActiveCallInvalid
	}
	organizationID = consentScope(organizationID)
	if organizationID == "" || strings.TrimSpace(request.ContactID) == "" || strings.TrimSpace(request.Channel) == "" {
		return VoiceActiveCall{}, fmt.Errorf("%w: organização, contato e canal são obrigatórios", ErrVoiceActiveCallInvalid)
	}
	if !voiceActiveCallPurposeAllowed(request.Purpose) {
		return VoiceActiveCall{}, fmt.Errorf("%w: %q", ErrVoiceActiveCallPurposeNotAllowed, strings.TrimSpace(request.Purpose))
	}
	if registry == nil {
		// Sem registro de consentimento não há como afirmar elegibilidade: recusar é o único desfecho honesto.
		return VoiceActiveCall{}, fmt.Errorf("%w: registro de consentimento indisponível", ErrVoiceActiveCallInvalid)
	}
	if err := registry.AuthorizeContact(organizationID, request.Channel, request.ContactID, now); err != nil {
		return VoiceActiveCall{}, err
	}

	calls, err := s.Calls(organizationID)
	if err != nil {
		return VoiceActiveCall{}, err
	}
	limite := now.UTC().Add(-VoiceActiveCallWindow)
	autorizadasNaJanela := 0
	for _, anterior := range calls {
		if anterior.Status == VoiceActiveCallAuthorized && anterior.RequestedAt.After(limite) {
			autorizadasNaJanela++
		}
	}
	if autorizadasNaJanela >= VoiceActiveCallLimitPerWindow {
		return VoiceActiveCall{}, fmt.Errorf("%w: %d autorizadas na janela", ErrVoiceActiveCallThrottled, autorizadasNaJanela)
	}

	request.OrganizationID = organizationID
	request.Purpose = strings.ToLower(strings.TrimSpace(request.Purpose))
	if request.ID == "" {
		// MEDIDO durante o teste: derivar o ID de organização+timestamp fazia DOIS PEDIDOS NO MESMO SEGUNDO
		// colidirem, e a segunda aprovação encontrava a primeira já autorizada. Identidade de efeito externo não
		// pode depender do relógio: usa UUID, como o resto do pacote.
		request.ID = "vcall_" + uuid.NewString()
	}
	request.Status = VoiceActiveCallPending
	request.Reason = ""
	request.ApprovedBy = ""
	request.RequestedAt = now.UTC()
	request.DecidedAt = nil
	request.ProviderExecutedAt = nil
	calls = append(calls, request)
	if err := s.save(organizationID, calls); err != nil {
		return VoiceActiveCall{}, err
	}
	return request, nil
}

// ApproveActiveCall registra a APROVAÇÃO HUMANA e só então libera a chamada. Aprovar duas vezes não republica nada:
// a segunda tentativa é recusada, porque o efeito externo acontece uma vez.
func (s *VoiceActiveCallStore) ApproveActiveCall(organizationID, callID, approver string, now time.Time) (VoiceActiveCall, error) {
	if err := requireOrganization(organizationID); err != nil {
		return VoiceActiveCall{}, err
	}
	if s == nil {
		return VoiceActiveCall{}, ErrVoiceActiveCallInvalid
	}
	organizationID = consentScope(organizationID)
	if strings.TrimSpace(approver) == "" {
		return VoiceActiveCall{}, fmt.Errorf("%w: aprovador é obrigatório", ErrVoiceActiveCallRequiresApproval)
	}
	calls, err := s.Calls(organizationID)
	if err != nil {
		return VoiceActiveCall{}, err
	}
	for index := range calls {
		if calls[index].ID != callID {
			continue
		}
		if calls[index].Status != VoiceActiveCallPending {
			return VoiceActiveCall{}, fmt.Errorf("%w: chamada já está em %q", ErrVoiceActiveCallRequiresApproval, calls[index].Status)
		}
		instante := now.UTC()
		calls[index].Status = VoiceActiveCallAuthorized
		calls[index].ApprovedBy = strings.TrimSpace(approver)
		calls[index].DecidedAt = &instante
		if err := s.save(organizationID, calls); err != nil {
			return VoiceActiveCall{}, err
		}
		return calls[index], nil
	}
	return VoiceActiveCall{}, fmt.Errorf("%w: chamada %s não encontrada", ErrVoiceActiveCallInvalid, callID)
}
