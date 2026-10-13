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

// CRM de contatos, leads e oportunidades (estágios 27 e 35; comando `/crm`).
//
// O comando `/crm` estava **registrado como indisponível** — e era honesto, porque
// não existia CRM. A diretriz exige que todo comando do catálogo mapeie para rota
// real, então esta é a implementação que falta, não uma lista decorativa.
//
// DECISÕES DELIBERADAS:
//   - **Deduplicação por identidade normalizada** (telefone só dígitos ou e-mail em
//     minúsculas): o mesmo cliente entrando por canais diferentes não pode virar
//     três contatos, porque isso quebra histórico, consentimento e cobrança;
//   - **Contato SEM telefone e SEM e-mail não é aceito**: um contato sem identidade
//     não pode ser reconhecido nem respeitar opt-out;
//   - **Consentimento NÃO é duplicado aqui**: o estado vem do `ConsentRegistry`, que
//     é a fonte única do opt-out. Guardar um segundo campo criaria duas verdades —
//     exatamente o defeito de irmão esquecido que já apareceu várias vezes;
//   - **Fase terminal é terminal**: `won`/`lost` não voltam atrás; reabrir exige um
//     registro novo, e a data do fechamento fica gravada;
//   - **Escopo por organização** em todas as leituras e escritas.

const (
	CRMStageNew       = "new"
	CRMStageQualified = "qualified"
	CRMStageProposal  = "proposal"
	CRMStageWon       = "won"
	CRMStageLost      = "lost"
)

var (
	ErrCRMContactInvalid   = errors.New("crm contact invalid")
	ErrCRMContactNotFound  = errors.New("crm contact not found")
	ErrCRMOpportunityFound = errors.New("crm opportunity not found")
	ErrCRMStageInvalid     = errors.New("crm opportunity stage invalid")
	ErrCRMStageTerminal    = errors.New("crm opportunity is already closed; reopening would erase the outcome")
)

// CRMContact é uma pessoa ou empresa com quem a operação fala.
type CRMContact struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone,omitempty"`
	Email          string    `json:"email,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	OwnerID        string    `json:"owner_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CRMOpportunity é uma negociação com valor e fase.
type CRMOpportunity struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ContactID      string     `json:"contact_id"`
	Title          string     `json:"title"`
	ValueCents     int64      `json:"value_cents"`
	Currency       string     `json:"currency"`
	Stage          string     `json:"stage"`
	OwnerID        string     `json:"owner_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	StageChangedAt time.Time  `json:"stage_changed_at"`
	StageChangedBy string     `json:"stage_changed_by,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

// CRMPipelineStage é o indicador por fase.
type CRMPipelineStage struct {
	Stage      string `json:"stage"`
	Count      int    `json:"count"`
	TotalCents int64  `json:"total_cents"`
}

// CRMPipeline é o resumo do funil.
type CRMPipeline struct {
	OrganizationID string             `json:"organization_id"`
	Currency       string             `json:"currency"`
	Stages         []CRMPipelineStage `json:"stages"`
	OpenCount      int                `json:"open_count"`
	OpenValueCents int64              `json:"open_value_cents"`
	WonValueCents  int64              `json:"won_value_cents"`
	LostCount      int                `json:"lost_count"`
}

// CRMContactView junta o contato ao estado de consentimento por canal. O estado é
// LIDO do registro de consentimento, nunca copiado para o contato.
type CRMContactView struct {
	Contact CRMContact             `json:"contact"`
	Consent []ConsentRecord        `json:"consent"`
	Blocked map[string]bool        `json:"blocked_channels"`
	Extra   map[string]interface{} `json:"-"`
}

// CRMStore guarda contatos e oportunidades por organização.
type CRMStore struct {
	mu        sync.RWMutex
	root      string
	contacts  map[string]CRMContact
	stageData map[string]CRMOpportunity
}

type crmFile struct {
	Contacts      []CRMContact     `json:"contacts"`
	Opportunities []CRMOpportunity `json:"opportunities"`
}

// NewCRMStore carrega o CRM do diretório informado.
func NewCRMStore(root string) (*CRMStore, error) {
	store := &CRMStore{
		root:      strings.TrimSpace(root),
		contacts:  map[string]CRMContact{},
		stageData: map[string]CRMOpportunity{},
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
		var stored crmFile
		if err := readJSON(filepath.Join(store.root, entry.Name()), &stored); err != nil {
			return nil, fmt.Errorf("load crm from %s: %w", entry.Name(), err)
		}
		for _, contact := range stored.Contacts {
			store.contacts[contact.ID] = contact
		}
		for _, opportunity := range stored.Opportunities {
			store.stageData[opportunity.ID] = opportunity
		}
	}
	return store, nil
}

func (s *CRMStore) organizationPath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

func (s *CRMStore) persistOrganizationLocked(organizationID string) error {
	path := s.organizationPath(organizationID)
	if path == "" {
		return nil
	}
	payload := crmFile{Contacts: []CRMContact{}, Opportunities: []CRMOpportunity{}}
	for _, contact := range s.contacts {
		if contact.OrganizationID == organizationID {
			payload.Contacts = append(payload.Contacts, contact)
		}
	}
	for _, opportunity := range s.stageData {
		if opportunity.OrganizationID == organizationID {
			payload.Opportunities = append(payload.Opportunities, opportunity)
		}
	}
	sort.SliceStable(payload.Contacts, func(i, j int) bool { return payload.Contacts[i].ID < payload.Contacts[j].ID })
	sort.SliceStable(payload.Opportunities, func(i, j int) bool { return payload.Opportunities[i].ID < payload.Opportunities[j].ID })
	return writeJSONAtomic(path, payload)
}

// crmIdentity normaliza a identidade de deduplicação: só dígitos do telefone, ou
// e-mail em minúsculas. Sem identidade não há contato.
//
// O nono dígito do celular brasileiro é normalizado: `55` + DDD + 8 dígitos
// começando por 6-9 é o MESMO número que o formato com o nono dígito. Bases legadas
// guardam as duas formas, e sem essa regra o mesmo cliente vira dois contatos — o que
// quebra histórico, consentimento e cobrança. Números FIXOS (iniciando por 2-5) não
// são tocados, porque aí a regra não vale.
func crmIdentity(phone, email string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if len(digits) >= 8 {
		return "phone:" + normalizeBrazilianMobile(digits)
	}
	if normalized := strings.ToLower(strings.TrimSpace(email)); normalized != "" {
		return "email:" + normalized
	}
	return ""
}

// normalizeBrazilianMobile acrescenta o nono dígito quando o número é claramente um
// celular brasileiro escrito sem ele.
func normalizeBrazilianMobile(digits string) string {
	if len(digits) != 12 || !strings.HasPrefix(digits, "55") {
		return digits
	}
	switch digits[4] {
	case '6', '7', '8', '9':
		return digits[:4] + "9" + digits[4:]
	default:
		return digits
	}
}

func normalizeCRMStage(stage string) string {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case CRMStageNew, "novo", "nova":
		return CRMStageNew
	case CRMStageQualified, "qualificado", "qualificada":
		return CRMStageQualified
	case CRMStageProposal, "proposta":
		return CRMStageProposal
	case CRMStageWon, "ganho", "ganha":
		return CRMStageWon
	case CRMStageLost, "perdido", "perdida":
		return CRMStageLost
	default:
		return ""
	}
}

func crmStageTerminal(stage string) bool {
	return stage == CRMStageWon || stage == CRMStageLost
}

// UpsertContact cria ou atualiza o contato. A identidade normalizada evita duplicar
// o mesmo cliente; `created=false` significa que o contato existente foi atualizado.
func (s *CRMStore) UpsertContact(contact CRMContact, now time.Time) (CRMContact, bool, error) {
	contact.OrganizationID = consentScope(contact.OrganizationID)
	contact.Name = strings.TrimSpace(contact.Name)
	contact.Phone = strings.TrimSpace(contact.Phone)
	contact.Email = strings.TrimSpace(contact.Email)
	if contact.Name == "" {
		return CRMContact{}, false, fmt.Errorf("%w: name is required", ErrCRMContactInvalid)
	}
	identity := crmIdentity(contact.Phone, contact.Email)
	if identity == "" {
		return CRMContact{}, false, fmt.Errorf("%w: a phone number or an e-mail is required to identify the contact", ErrCRMContactInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, existing := range s.contacts {
		if existing.OrganizationID != contact.OrganizationID {
			continue
		}
		if crmIdentity(existing.Phone, existing.Email) != identity {
			continue
		}
		// Atualiza o existente preservando criação e histórico de oportunidade.
		existing.Name = contact.Name
		existing.Phone = contact.Phone
		existing.Email = contact.Email
		existing.Tags = append([]string(nil), contact.Tags...)
		if contact.Notes != "" {
			existing.Notes = contact.Notes
		}
		if contact.OwnerID != "" {
			existing.OwnerID = contact.OwnerID
		}
		existing.UpdatedAt = now
		s.contacts[id] = existing
		if err := s.persistOrganizationLocked(contact.OrganizationID); err != nil {
			return CRMContact{}, false, err
		}
		return existing, false, nil
	}
	contact.ID = "ct_" + uuid.NewString()[:12]
	contact.CreatedAt = now
	contact.UpdatedAt = now
	if contact.Tags == nil {
		contact.Tags = []string{}
	}
	s.contacts[contact.ID] = contact
	if err := s.persistOrganizationLocked(contact.OrganizationID); err != nil {
		delete(s.contacts, contact.ID)
		return CRMContact{}, false, err
	}
	return contact, true, nil
}

// ListContacts lista (ou busca) contatos da organização.
func (s *CRMStore) ListContacts(organizationID, query string) []CRMContact {
	if err := requireOrganization(organizationID); err != nil {
		return nil
	}
	organizationID = consentScope(organizationID)
	query = strings.ToLower(strings.TrimSpace(query))
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CRMContact, 0)
	for _, contact := range s.contacts {
		if contact.OrganizationID != organizationID {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(contact.Name + " " + contact.Phone + " " + contact.Email + " " + strings.Join(contact.Tags, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		out = append(out, contact)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// GetContact busca pelo identificador dentro da organização.
func (s *CRMStore) GetContact(organizationID, contactID string) (CRMContact, error) {
	if err := requireOrganization(organizationID); err != nil {
		return CRMContact{}, err
	}
	organizationID = consentScope(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	contact, ok := s.contacts[strings.TrimSpace(contactID)]
	if !ok || contact.OrganizationID != organizationID {
		return CRMContact{}, fmt.Errorf("%w: %s", ErrCRMContactNotFound, contactID)
	}
	return contact, nil
}

// CreateOpportunity abre uma oportunidade para um contato da MESMA organização.
func (s *CRMStore) CreateOpportunity(organizationID, contactID, title string, valueCents int64, ownerID string, now time.Time) (CRMOpportunity, error) {
	if err := requireOrganization(organizationID); err != nil {
		return CRMOpportunity{}, err
	}
	organizationID = consentScope(organizationID)
	title = strings.TrimSpace(title)
	if title == "" {
		return CRMOpportunity{}, fmt.Errorf("%w: title is required", ErrCRMContactInvalid)
	}
	if valueCents < 0 {
		return CRMOpportunity{}, fmt.Errorf("%w: value cannot be negative", ErrCRMContactInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	contact, ok := s.contacts[strings.TrimSpace(contactID)]
	if !ok || contact.OrganizationID != organizationID {
		return CRMOpportunity{}, fmt.Errorf("%w: %s", ErrCRMContactNotFound, contactID)
	}
	opportunity := CRMOpportunity{
		ID:             "op_" + uuid.NewString()[:12],
		OrganizationID: organizationID,
		ContactID:      contact.ID,
		Title:          title,
		ValueCents:     valueCents,
		Currency:       LedgerDefaultCurrency,
		Stage:          CRMStageNew,
		OwnerID:        strings.TrimSpace(ownerID),
		CreatedAt:      now,
		UpdatedAt:      now,
		StageChangedAt: now,
	}
	s.stageData[opportunity.ID] = opportunity
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		delete(s.stageData, opportunity.ID)
		return CRMOpportunity{}, err
	}
	return opportunity, nil
}

// SetOpportunityStage muda a fase. Fase terminal NÃO volta atrás.
func (s *CRMStore) SetOpportunityStage(organizationID, opportunityID, stage, actor string, now time.Time) (CRMOpportunity, error) {
	if err := requireOrganization(organizationID); err != nil {
		return CRMOpportunity{}, err
	}
	organizationID = consentScope(organizationID)
	normalized := normalizeCRMStage(stage)
	if normalized == "" {
		return CRMOpportunity{}, fmt.Errorf("%w: %q", ErrCRMStageInvalid, stage)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	opportunity, ok := s.stageData[strings.TrimSpace(opportunityID)]
	if !ok || opportunity.OrganizationID != organizationID {
		return CRMOpportunity{}, fmt.Errorf("%w: %s", ErrCRMOpportunityFound, opportunityID)
	}
	if crmStageTerminal(opportunity.Stage) && opportunity.Stage != normalized {
		return CRMOpportunity{}, fmt.Errorf("%w: %s -> %s", ErrCRMStageTerminal, opportunity.Stage, normalized)
	}
	previous := opportunity.Stage
	opportunity.Stage = normalized
	opportunity.UpdatedAt = now
	opportunity.StageChangedAt = now
	opportunity.StageChangedBy = strings.TrimSpace(actor)
	if crmStageTerminal(normalized) {
		closed := now
		opportunity.ClosedAt = &closed
	} else {
		opportunity.ClosedAt = nil
	}
	s.stageData[opportunity.ID] = opportunity
	if err := s.persistOrganizationLocked(organizationID); err != nil {
		opportunity.Stage = previous
		s.stageData[opportunity.ID] = opportunity
		return CRMOpportunity{}, err
	}
	return opportunity, nil
}

// ListOpportunities lista oportunidades, opcionalmente por fase.
func (s *CRMStore) ListOpportunities(organizationID, stage string) []CRMOpportunity {
	organizationID = consentScope(organizationID)
	filter := normalizeCRMStage(stage)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CRMOpportunity, 0)
	for _, opportunity := range s.stageData {
		if opportunity.OrganizationID != organizationID {
			continue
		}
		if filter != "" && opportunity.Stage != filter {
			continue
		}
		out = append(out, opportunity)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// PipelineSummary calcula os indicadores do funil.
func (s *CRMStore) PipelineSummary(organizationID string) CRMPipeline {
	organizationID = consentScope(organizationID)
	opportunities := s.ListOpportunities(organizationID, "")
	pipeline := CRMPipeline{OrganizationID: organizationID, Currency: LedgerDefaultCurrency}
	order := []string{CRMStageNew, CRMStageQualified, CRMStageProposal, CRMStageWon, CRMStageLost}
	totals := map[string]*CRMPipelineStage{}
	for _, stage := range order {
		totals[stage] = &CRMPipelineStage{Stage: stage}
	}
	for _, opportunity := range opportunities {
		bucket, ok := totals[opportunity.Stage]
		if !ok {
			bucket = &CRMPipelineStage{Stage: opportunity.Stage}
			totals[opportunity.Stage] = bucket
		}
		bucket.Count++
		bucket.TotalCents += opportunity.ValueCents
		pipeline.Currency = opportunity.Currency
		switch opportunity.Stage {
		case CRMStageWon:
			pipeline.WonValueCents += opportunity.ValueCents
		case CRMStageLost:
			pipeline.LostCount++
		default:
			pipeline.OpenCount++
			pipeline.OpenValueCents += opportunity.ValueCents
		}
	}
	for _, stage := range order {
		pipeline.Stages = append(pipeline.Stages, *totals[stage])
	}
	return pipeline
}

// ContactView junta o contato ao consentimento REAL, lido do registro próprio. O
// CRM não guarda cópia do opt-out: duas verdades sobre consentimento seriam pior do
// que nenhuma.
func (s *CRMStore) ContactView(organizationID, contactID string, registry *ConsentRegistry) (CRMContactView, error) {
	if err := requireOrganization(organizationID); err != nil {
		return CRMContactView{}, err
	}
	contact, err := s.GetContact(organizationID, contactID)
	if err != nil {
		return CRMContactView{}, err
	}
	view := CRMContactView{Contact: contact, Consent: []ConsentRecord{}, Blocked: map[string]bool{}}
	if registry == nil {
		return view, nil
	}
	for _, record := range registry.ListConsents(contact.OrganizationID) {
		if strings.TrimSpace(record.ContactID) == "" {
			continue
		}
		if !crmConsentMatchesContact(contact, record) {
			continue
		}
		view.Consent = append(view.Consent, record)
		if err := registry.AuthorizeContact(contact.OrganizationID, record.Channel, record.ContactID, time.Now().UTC()); err != nil {
			view.Blocked[record.Channel] = true
		}
	}
	sort.SliceStable(view.Consent, func(i, j int) bool { return view.Consent[i].Channel < view.Consent[j].Channel })
	return view, nil
}

// crmConsentMatchesContact liga o contato do CRM ao contato do registro de
// consentimento pela identidade normalizada (telefone só dígitos).
func crmConsentMatchesContact(contact CRMContact, record ConsentRecord) bool {
	recordDigits := crmIdentity(record.ContactID, "")
	contactDigits := crmIdentity(contact.Phone, "")
	if recordDigits != "" && contactDigits != "" {
		return recordDigits == contactDigits
	}
	if strings.EqualFold(strings.TrimSpace(record.ContactID), strings.TrimSpace(contact.Email)) {
		return true
	}
	return strings.TrimSpace(record.ContactID) == strings.TrimSpace(contact.ID)
}
