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

	"github.com/google/uuid"
)

// Estágio 27 — AI SALES WORKFORCE: propostas comerciais e pós-venda.
//
// O que faltava: os agentes de atendimento e vendas existiam, com CRM e catálogo, mas **não
// havia proposta comercial** (o documento que o vendedor manda com itens e preço) nem
// **pós-venda** (o acompanhamento depois da entrega). Um agente de vendas sem proposta não
// vende; um sem pós-venda não descobre que o cliente ficou insatisfeito.
//
// Decisões deliberadas:
//
//   - **PREÇO VEM DO CATÁLOGO, NUNCA DO PEDIDO**: a proposta recebe apenas SKU + quantidade e o
//     total é **recalculado no servidor** com o preço do catálogo. Aceitar preço enviado pelo
//     cliente permitiria "proposta" com valor inventado (e desconto forjado por quem chama a
//     API). SKU fora do catálogo é **erro**, não linha com preço zero;
//   - **ENVIAR EXIGE CONSENTIMENTO**: proposta é mensagem para terceiro. O envio passa pelo
//     `ConsentRegistry`, então quem pediu opt-out **não recebe** proposta — a mesma política do
//     resto do sistema, sem caminho alternativo;
//   - **VIGÊNCIA É RESPEITADA**: aceitar proposta vencida é recusado (`ErrProposalExpired`), em
//     vez de aceitar silenciosamente um preço que já não vale;
//   - **DECISÃO É TERMINAL**: aceita ou recusada não volta a rascunho; reabrir apagaria o
//     resultado (mesma disciplina do pipeline do CRM);
//   - **PÓS-VENDA É MEDIDO**: o acompanhamento tem prazo e o atraso é **contável**, como o SLA da
//     caixa de entrada. Um "pós-venda" que ninguém consegue medir não é operação, é intenção.

const (
	// SalesProposalDraft e seguintes são os estados da proposta.
	SalesProposalDraft    = "rascunho"
	SalesProposalSent     = "enviada"
	SalesProposalAccepted = "aceita"
	SalesProposalRejected = "recusada"

	// DefaultProposalValidity é o prazo padrão de vigência da proposta.
	DefaultProposalValidity = 7 * 24 * time.Hour
	// DefaultPostSaleFollowUp é o prazo padrão do acompanhamento de pós-venda.
	DefaultPostSaleFollowUp = 72 * time.Hour

	// SalesFollowUpScheduled, SalesFollowUpDone e SalesFollowUpSkipped são os estados.
	SalesFollowUpScheduled = "agendado"
	SalesFollowUpDone      = "concluido"
	SalesFollowUpSkipped   = "dispensado"
)

var (
	ErrProposalInvalid = errors.New("sales proposal invalid")
	// PRECIFICACAO (E-269): percentual fora de faixa, frete negativo e desconto acima do teto autorizado.
	ErrProposalPricingInvalid   = errors.New("proposal pricing invalid")
	ErrProposalDiscountExceeded = errors.New("proposal discount exceeds the authorized ceiling")
	ErrProposalUnknownSKU       = errors.New("sales proposal references unknown sku")
	ErrProposalNotFound         = errors.New("sales proposal not found")
	ErrProposalExpired          = errors.New("sales proposal expired")
	ErrProposalConsentRequired  = errors.New("sales proposal requires contact consent")
	ErrProposalStateInvalid     = errors.New("sales proposal state does not allow this action")
	ErrFollowUpInvalid          = errors.New("post-sale follow-up invalid")
	// ErrFollowUpStateInvalid é separado do erro de ENTRADA: motivo/escopo ausentes são
	// erro do cliente (400); concluir algo já concluído é conflito de estado (409).
	ErrFollowUpStateInvalid = errors.New("post-sale follow-up state does not allow this action")
	ErrFollowUpNotFound     = errors.New("post-sale follow-up not found")
)

// SalesProposalRequest é o pedido de proposta. O cliente informa SKU e quantidade; o PREÇO
// vem do catálogo, e por isso não existe campo de preço aqui.
type SalesProposalRequest struct {
	OrganizationID string
	ContactID      string
	OpportunityID  string
	Currency       string
	Notes          string
	Validity       time.Duration
	Items          []SalesProposalItemRequest
	// PRECIFICACAO (E-269). Percentuais em BASIS POINTS (1500 = 15,00%): dinheiro em inteiro, sem float.
	DiscountBasisPoints int64 `json:"discount_basis_points,omitempty"`
	TaxBasisPoints      int64 `json:"tax_basis_points,omitempty"`
	ShippingCents       int64 `json:"shipping_cents,omitempty"`
}

// SalesProposalItemRequest é uma linha pedida.
type SalesProposalItemRequest struct {
	SKU      string
	Quantity int64
}

// SalesProposalItem é uma linha da proposta. O preço unitário é SEMPRE o do catálogo.
type SalesProposalItem struct {
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

// SalesProposal é a proposta comercial.
type SalesProposal struct {
	ID             string              `json:"id"`
	OrganizationID string              `json:"organization_id"`
	ContactID      string              `json:"contact_id"`
	OpportunityID  string              `json:"opportunity_id,omitempty"`
	Currency       string              `json:"currency"`
	Items          []SalesProposalItem `json:"items"`
	// PRECIFICACAO (E-269): a conta fica ABERTA na proposta. Um total unico esconderia quanto foi
	// desconto, quanto foi imposto e quanto foi frete.
	SubtotalCents       int64 `json:"subtotal_cents"`
	DiscountBasisPoints int64 `json:"discount_basis_points"`
	DiscountCents       int64 `json:"discount_cents"`
	TaxBasisPoints      int64 `json:"tax_basis_points"`
	TaxCents            int64 `json:"tax_cents"`
	ShippingCents       int64 `json:"shipping_cents"`
	// ShippingDeclared diz se o frete foi INFORMADO. Zero nao informado nao e zero cobrado.
	ShippingDeclared bool       `json:"shipping_declared"`
	TotalCents       int64      `json:"total_cents"`
	Status           string     `json:"status"`
	ValidUntil       time.Time  `json:"valid_until"`
	Notes            string     `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	SentAt           *time.Time `json:"sent_at,omitempty"`
	DecidedAt        *time.Time `json:"decided_at,omitempty"`
	DecidedBy        string     `json:"decided_by,omitempty"`
}

// SalesFollowUp é o acompanhamento de pós-venda.
type SalesFollowUp struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ContactID      string     `json:"contact_id"`
	OrderID        string     `json:"order_id,omitempty"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	DueAt          time.Time  `json:"due_at"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CompletedBy    string     `json:"completed_by,omitempty"`
}

// SalesWorkforceStore guarda propostas e pós-venda por organização.
type SalesWorkforceStore struct {
	mu         sync.RWMutex
	root       string
	proposals  map[string][]SalesProposal
	followUps  map[string][]SalesFollowUp
	commerce   *CommerceStore
	consent    *ConsentRegistry
	loadedOrgs map[string]bool
}

// NewSalesWorkforceStore cria o armazenamento. `root` vazio mantém tudo só em memória, o que é
// útil em teste e explícito em produção.
func NewSalesWorkforceStore(root string, commerce *CommerceStore, consent *ConsentRegistry) *SalesWorkforceStore {
	return &SalesWorkforceStore{
		root:       strings.TrimSpace(root),
		proposals:  map[string][]SalesProposal{},
		followUps:  map[string][]SalesFollowUp{},
		commerce:   commerce,
		consent:    consent,
		loadedOrgs: map[string]bool{},
	}
}

// ---------------------------------------------------------------------------------------------------------------
// POLITICA DE DESCONTO (E-269)
// ---------------------------------------------------------------------------------------------------------------

// proposalPolicyFile guarda o teto de desconto autorizado da organizacao.
type proposalPolicyFile struct {
	MaxDiscountBasisPoints int64 `json:"max_discount_basis_points"`
}

// SetDiscountPolicy define o teto de desconto autorizado (basis points). Zero REMOVE a politica e volta ao padrao
// SEGURO: sem politica, desconto nenhum.
func (s *SalesWorkforceStore) SetDiscountPolicy(organizationID string, maxDiscountBasisPoints int64) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return fmt.Errorf("%w: organizacao e obrigatoria", ErrProposalPricingInvalid)
	}
	if maxDiscountBasisPoints < 0 || maxDiscountBasisPoints > 10000 {
		return fmt.Errorf("%w: teto %d fora de 0..10000", ErrProposalPricingInvalid, maxDiscountBasisPoints)
	}
	caminho := s.policyPath(organizationID)
	if caminho == "" {
		// Modo em memoria: o teto vale nesta execucao. Declarado, nao fingido.
		return nil
	}
	return writeJSONAtomic(caminho, proposalPolicyFile{MaxDiscountBasisPoints: maxDiscountBasisPoints})
}

// discountCeiling devolve o teto autorizado. SEM POLITICA O TETO E ZERO: vendedor nao da desconto sem autorizacao.
func (s *SalesWorkforceStore) discountCeiling(organizationID string) (int64, error) {
	caminho := s.policyPath(organizationID)
	if caminho == "" {
		return 0, nil
	}
	stored := proposalPolicyFile{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("ler politica de desconto: %w", err)
	}
	return stored.MaxDiscountBasisPoints, nil
}

func (s *SalesWorkforceStore) policyPath(organizationID string) string {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return ""
	}
	return filepath.Join(s.root, "sales", "policy_"+sanitizeIDComponent(organizationID)+".json")
}

type salesWorkforceFile struct {
	Proposals []SalesProposal `json:"proposals"`
	FollowUps []SalesFollowUp `json:"follow_ups"`
}

func (s *SalesWorkforceStore) organizationPath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, "sales", sanitizeIDComponent(organizationID)+".json")
}

func (s *SalesWorkforceStore) loadOrganizationLocked(organizationID string) {
	if s.loadedOrgs[organizationID] {
		return
	}
	s.loadedOrgs[organizationID] = true
	path := s.organizationPath(organizationID)
	if path == "" {
		return
	}
	var file salesWorkforceFile
	if err := readJSON(path, &file); err != nil {
		return
	}
	s.proposals[organizationID] = file.Proposals
	s.followUps[organizationID] = file.FollowUps
}

func (s *SalesWorkforceStore) persistOrganizationLocked(organizationID string) error {
	path := s.organizationPath(organizationID)
	if path == "" {
		return nil
	}
	file := salesWorkforceFile{Proposals: s.proposals[organizationID], FollowUps: s.followUps[organizationID]}
	return writeJSONAtomic(path, file)
}

// CreateProposal monta a proposta com os preços do CATÁLOGO.
//
// O cliente informa apenas SKU e quantidade: preço enviado pelo cliente é ignorado, porque
// aceitá-lo permitiria proposta com valor inventado.
func (s *SalesWorkforceStore) CreateProposal(request SalesProposalRequest, now time.Time) (SalesProposal, error) {
	organizationID := strings.TrimSpace(request.OrganizationID)
	contactID := strings.TrimSpace(request.ContactID)
	if organizationID == "" || contactID == "" {
		return SalesProposal{}, fmt.Errorf("%w: organização e contato são obrigatórios", ErrProposalInvalid)
	}
	if len(request.Items) == 0 {
		return SalesProposal{}, fmt.Errorf("%w: a proposta precisa de pelo menos um item", ErrProposalInvalid)
	}
	catalog := map[string]CommerceProduct{}
	for _, product := range s.commerce.Products(organizationID) {
		catalog[product.SKU] = product
	}
	currency := strings.TrimSpace(request.Currency)
	proposal := SalesProposal{
		ID:             "prop_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		ContactID:      contactID,
		OpportunityID:  strings.TrimSpace(request.OpportunityID),
		Status:         SalesProposalDraft,
		Notes:          strings.TrimSpace(request.Notes),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	validity := request.Validity
	if validity <= 0 {
		validity = DefaultProposalValidity
	}
	proposal.ValidUntil = now.Add(validity)

	for _, item := range request.Items {
		sku := strings.TrimSpace(item.SKU)
		product, ok := catalog[sku]
		if !ok {
			return SalesProposal{}, fmt.Errorf("%w: %s", ErrProposalUnknownSKU, sku)
		}
		if item.Quantity <= 0 {
			return SalesProposal{}, fmt.Errorf("%w: quantidade do SKU %s precisa ser positiva", ErrProposalInvalid, sku)
		}
		if currency == "" {
			currency = product.Currency
		}
		// Preço unitário SEMPRE do catálogo; o total é recalculado aqui.
		line := SalesProposalItem{
			SKU: sku, Name: product.Name, Quantity: item.Quantity,
			UnitPriceCents: product.PriceCents, TotalCents: product.PriceCents * item.Quantity,
		}
		proposal.Items = append(proposal.Items, line)
		proposal.SubtotalCents += line.TotalCents
	}
	if currency == "" {
		currency = "BRL"
	}
	proposal.Currency = currency

	// PRECIFICACAO (E-269). Validacoes ANTES de gravar: nada e persistido com preco invalido.
	if request.DiscountBasisPoints < 0 || request.DiscountBasisPoints > 10000 {
		return SalesProposal{}, fmt.Errorf("%w: desconto %d fora de 0..10000", ErrProposalPricingInvalid, request.DiscountBasisPoints)
	}
	if request.TaxBasisPoints < 0 || request.TaxBasisPoints > 10000 {
		return SalesProposal{}, fmt.Errorf("%w: imposto %d fora de 0..10000", ErrProposalPricingInvalid, request.TaxBasisPoints)
	}
	if request.ShippingCents < 0 {
		return SalesProposal{}, fmt.Errorf("%w: frete negativo (%d)", ErrProposalPricingInvalid, request.ShippingCents)
	}
	teto, err := s.discountCeiling(organizationID)
	if err != nil {
		return SalesProposal{}, err
	}
	if request.DiscountBasisPoints > teto {
		return SalesProposal{}, fmt.Errorf("%w: pedido %d bp, autorizado %d bp", ErrProposalDiscountExceeded, request.DiscountBasisPoints, teto)
	}
	proposal.DiscountBasisPoints = request.DiscountBasisPoints
	// Arredondamento PARA BAIXO: nunca cobrar a mais do cliente nem registrar desconto nao concedido.
	proposal.DiscountCents = proposal.SubtotalCents * request.DiscountBasisPoints / 10000
	base := proposal.SubtotalCents - proposal.DiscountCents
	proposal.TaxBasisPoints = request.TaxBasisPoints
	// Imposto sobre a BASE COM DESCONTO: tributar o valor cheio cobraria imposto de dinheiro que nao entrou.
	proposal.TaxCents = base * request.TaxBasisPoints / 10000
	proposal.ShippingCents = request.ShippingCents
	proposal.ShippingDeclared = request.ShippingCents > 0
	proposal.TotalCents = base + proposal.TaxCents + proposal.ShippingCents

	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	s.proposals[organizationID] = append(s.proposals[organizationID], proposal)
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		// Persistência falhou: desfaz o estado em memória para não relatar sucesso que não existe.
		s.proposals[organizationID] = s.proposals[organizationID][:len(s.proposals[organizationID])-1]
		return SalesProposal{}, err
	}
	return proposal, nil
}

// SendProposal marca o envio, exigindo CONSENTIMENTO do contato no canal informado.
func (s *SalesWorkforceStore) SendProposal(organizationID, proposalID, channel, actor string, now time.Time) (SalesProposal, error) {
	if err := requireOrganization(organizationID); err != nil {
		return SalesProposal{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	index := s.proposalIndexLocked(organizationID, proposalID)
	if index < 0 {
		return SalesProposal{}, fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}
	proposal := s.proposals[organizationID][index]
	if proposal.Status != SalesProposalDraft {
		return SalesProposal{}, fmt.Errorf("%w: proposta em %q não pode ser enviada", ErrProposalStateInvalid, proposal.Status)
	}
	// Proposta é mensagem para terceiro: sem consentimento, NÃO sai.
	if s.consent != nil {
		if err := s.consent.AuthorizeContact(organizationID, channel, proposal.ContactID, now); err != nil {
			return SalesProposal{}, fmt.Errorf("%w: %v", ErrProposalConsentRequired, err)
		}
	}
	proposal.Status = SalesProposalSent
	proposal.SentAt = &now
	proposal.UpdatedAt = now
	s.proposals[organizationID][index] = proposal
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		proposal.Status = SalesProposalDraft
		proposal.SentAt = nil
		s.proposals[organizationID][index] = proposal
		return SalesProposal{}, err
	}
	return proposal, nil
}

// DecideProposal registra a decisão do cliente. Decisão é TERMINAL.
func (s *SalesWorkforceStore) DecideProposal(organizationID, proposalID, decision, actor string, now time.Time) (SalesProposal, error) {
	if err := requireOrganization(organizationID); err != nil {
		return SalesProposal{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	index := s.proposalIndexLocked(organizationID, proposalID)
	if index < 0 {
		return SalesProposal{}, fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}
	proposal := s.proposals[organizationID][index]
	if proposal.Status != SalesProposalSent {
		return SalesProposal{}, fmt.Errorf("%w: proposta em %q não pode ser decidida", ErrProposalStateInvalid, proposal.Status)
	}
	// VIGÊNCIA: aceitar proposta vencida seria aceitar um preço que já não vale.
	if now.After(proposal.ValidUntil) {
		return SalesProposal{}, fmt.Errorf("%w: a vigência terminou em %s", ErrProposalExpired, proposal.ValidUntil.Format(time.RFC3339))
	}
	normalized := strings.ToLower(strings.TrimSpace(decision))
	switch normalized {
	case "aceitar", "aceita", "accept", "accepted":
		proposal.Status = SalesProposalAccepted
	case "recusar", "recusada", "reject", "rejected":
		proposal.Status = SalesProposalRejected
	default:
		return SalesProposal{}, fmt.Errorf("%w: decisão %q desconhecida (use aceitar ou recusar)", ErrProposalInvalid, decision)
	}
	proposal.DecidedAt = &now
	proposal.DecidedBy = strings.TrimSpace(actor)
	proposal.UpdatedAt = now
	s.proposals[organizationID][index] = proposal
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		return SalesProposal{}, err
	}
	return proposal, nil
}

// Proposals lista as propostas da organização, opcionalmente filtrando por estado.
func (s *SalesWorkforceStore) Proposals(organizationID, status string) []SalesProposal {
	if err := requireOrganization(organizationID); err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	filter := strings.ToLower(strings.TrimSpace(status))
	result := make([]SalesProposal, 0, len(s.proposals[organizationID]))
	for _, proposal := range s.proposals[organizationID] {
		if filter != "" && proposal.Status != filter {
			continue
		}
		result = append(result, proposal)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result
}

// Proposal devolve uma proposta específica, escopada por organização.
func (s *SalesWorkforceStore) Proposal(organizationID, proposalID string) (SalesProposal, error) {
	if err := requireOrganization(organizationID); err != nil {
		return SalesProposal{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	index := s.proposalIndexLocked(organizationID, proposalID)
	if index < 0 {
		return SalesProposal{}, fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}
	return s.proposals[organizationID][index], nil
}

func (s *SalesWorkforceStore) proposalIndexLocked(organizationID, proposalID string) int {
	for index, proposal := range s.proposals[organizationID] {
		if proposal.ID == proposalID {
			return index
		}
	}
	return -1
}

// ScheduleFollowUp cria o acompanhamento de pós-venda, com prazo.
func (s *SalesWorkforceStore) ScheduleFollowUp(organizationID, contactID, orderID, reason string, due time.Duration, now time.Time) (SalesFollowUp, error) {
	if err := requireOrganization(organizationID); err != nil {
		return SalesFollowUp{}, err
	}
	organizationID = strings.TrimSpace(organizationID)
	contactID = strings.TrimSpace(contactID)
	if organizationID == "" || contactID == "" {
		return SalesFollowUp{}, fmt.Errorf("%w: organização e contato são obrigatórios", ErrFollowUpInvalid)
	}
	if due <= 0 {
		due = DefaultPostSaleFollowUp
	}
	if strings.TrimSpace(reason) == "" {
		return SalesFollowUp{}, fmt.Errorf("%w: o motivo do acompanhamento é obrigatório", ErrFollowUpInvalid)
	}
	followUp := SalesFollowUp{
		ID:             "psf_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		ContactID:      contactID,
		OrderID:        strings.TrimSpace(orderID),
		Reason:         strings.TrimSpace(reason),
		Status:         SalesFollowUpScheduled,
		DueAt:          now.Add(due),
		CreatedAt:      now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	// Idempotência por pedido: agendar duas vezes o MESMO pedido não cria dois acompanhamentos.
	if followUp.OrderID != "" {
		for _, existing := range s.followUps[organizationID] {
			if existing.OrderID == followUp.OrderID && existing.Status == SalesFollowUpScheduled {
				return existing, nil
			}
		}
	}
	s.followUps[organizationID] = append(s.followUps[organizationID], followUp)
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		s.followUps[organizationID] = s.followUps[organizationID][:len(s.followUps[organizationID])-1]
		return SalesFollowUp{}, err
	}
	return followUp, nil
}

// CompleteFollowUp fecha o acompanhamento.
func (s *SalesWorkforceStore) CompleteFollowUp(organizationID, followUpID, actor, outcome string, now time.Time) (SalesFollowUp, error) {
	if err := requireOrganization(organizationID); err != nil {
		return SalesFollowUp{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	for index, followUp := range s.followUps[organizationID] {
		if followUp.ID != followUpID {
			continue
		}
		if followUp.Status != SalesFollowUpScheduled {
			return SalesFollowUp{}, fmt.Errorf("%w: acompanhamento em %q não pode ser concluído", ErrFollowUpStateInvalid, followUp.Status)
		}
		status := SalesFollowUpDone
		if strings.EqualFold(strings.TrimSpace(outcome), "dispensar") {
			status = SalesFollowUpSkipped
		}
		followUp.Status = status
		followUp.CompletedAt = &now
		followUp.CompletedBy = strings.TrimSpace(actor)
		s.followUps[organizationID][index] = followUp
		if err := s.persistOrganizationLocked(organizationID); err != nil {
			return SalesFollowUp{}, err
		}
		return followUp, nil
	}
	return SalesFollowUp{}, fmt.Errorf("%w: %s", ErrFollowUpNotFound, followUpID)
}

// FollowUps lista os acompanhamentos, opcionalmente só os atrasados.
func (s *SalesWorkforceStore) FollowUps(organizationID string, overdueOnly bool, now time.Time) []SalesFollowUp {
	if err := requireOrganization(organizationID); err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadOrganizationLocked(organizationID)
	result := make([]SalesFollowUp, 0, len(s.followUps[organizationID]))
	for _, followUp := range s.followUps[organizationID] {
		if overdueOnly && !(followUp.Status == SalesFollowUpScheduled && now.After(followUp.DueAt)) {
			continue
		}
		result = append(result, followUp)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].DueAt.Before(result[j].DueAt) })
	return result
}
