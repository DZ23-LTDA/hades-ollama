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
)

// Registro de consentimento e opt-out (estágio 27 — AI Sales Workforce).
//
// O estágio exige **consentimento, opt-out, limites e proteção de dados**, e a
// diretriz do projeto proíbe contato sem consentimento. Antes disto o produto tinha
// **allowlist** e **aprovação por mensagem** no WhatsApp, mas o canal de Telegram
// **não checava autorização alguma de contato** — e não havia registro persistente de
// consentimento nem opt-out em lugar nenhum.
//
// DECISÕES DELIBERADAS:
//   - **FALHA FECHADA**: contato **desconhecido NÃO é contatável**. O padrão do
//     sistema é "não", e o consentimento precisa ser registrado explicitamente.
//     Assumir consentimento por omissão é exatamente a violação que se quer evitar;
//   - **Opt-out é decisão final até nova manifestação explícita**: revogar grava motivo
//     e responsável. Um pedido de descadastro não pode ser revertido por um fluxo
//     automático;
//   - **Prazo de validated**: consentimento com `ExpiresAt` vencido **volta a ser
//     ausência de consentimento** — consentimento velho não vale para sempre;
//   - **Trilha de auditoria**: quem concedeu, quem revogou, quando e por quê. Sem
//     isso não há como provar conformidade nem responder a um titular;
//   - **Escopo por organização e por canal**: consentir no WhatsApp **não** autoriza
//     Telegram nem outro tenant. O silêncio entre canais era uma brecha real.

const (
	// ConsentChannelWhatsApp e ConsentChannelTelegram identificam o canal.
	ConsentChannelWhatsApp = "whatsapp"
	ConsentChannelTelegram = "telegram"

	// Estados possíveis do consentimento.
	ConsentGranted = "granted"
	ConsentRevoked = "revoked"
)

var (
	ErrConsentMissing      = errors.New("contact has no recorded consent")
	ErrConsentRevoked      = errors.New("contact has opted out")
	ErrConsentExpired      = errors.New("contact consent expired")
	ErrConsentInvalid      = errors.New("consent record invalid")
	ErrConsentNotFound     = errors.New("consent record not found")
	ErrConsentReasonNeeded = errors.New("revoking consent requires a reason")
)

// ConsentRecord é o estado de consentimento de um contato num canal.
type ConsentRecord struct {
	OrganizationID string     `json:"organization_id"`
	Channel        string     `json:"channel"`
	ContactID      string     `json:"contact_id"`
	State          string     `json:"state"`
	Source         string     `json:"source,omitempty"`
	Evidence       string     `json:"evidence,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	GrantedBy      string     `json:"granted_by,omitempty"`
	RevokedBy      string     `json:"revoked_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// ConsentRegistry guarda o consentimento por organização.
type ConsentRegistry struct {
	mu      sync.RWMutex
	root    string
	records map[string]ConsentRecord
}

// NewConsentRegistry carrega o registro do diretório informado.
func NewConsentRegistry(root string) (*ConsentRegistry, error) {
	registry := &ConsentRegistry{root: strings.TrimSpace(root), records: map[string]ConsentRecord{}}
	if registry.root == "" {
		return registry, nil
	}
	if err := os.MkdirAll(registry.root, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(registry.root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var stored []ConsentRecord
		if err := readJSON(filepath.Join(registry.root, entry.Name()), &stored); err != nil {
			return nil, fmt.Errorf("load consents from %s: %w", entry.Name(), err)
		}
		for _, record := range stored {
			registry.records[consentKey(record.OrganizationID, record.Channel, record.ContactID)] = record
		}
	}
	return registry, nil
}

func consentKey(organizationID, channel, contactID string) string {
	return strings.Join([]string{
		consentScope(organizationID),
		strings.ToLower(strings.TrimSpace(channel)),
		strings.TrimSpace(contactID),
	}, "|")
}

// consentScope normaliza o escopo de organização: vazio significa a organização
// LOCAL, a mesma convenção usada no HTTP e no supervisor. Sem esta normalização, um
// consentimento gravado sob uma chave nunca casaria com a checagem feita sob outra —
// e o resultado seria o pior dos dois mundos: envio impossível de autorizar, ou
// autorização que não vale.
func consentScope(organizationID string) string {
	if trimmed := strings.TrimSpace(organizationID); trimmed != "" {
		return trimmed
	}
	return LocalOrganizationID
}

func (r *ConsentRegistry) organizationPath(organizationID string) string {
	if r.root == "" {
		return ""
	}
	return filepath.Join(r.root, sanitizeIDComponent(organizationID)+".json")
}

func (r *ConsentRegistry) persistOrganizationLocked(organizationID string) error {
	path := r.organizationPath(organizationID)
	if path == "" {
		return nil
	}
	records := make([]ConsentRecord, 0)
	for _, record := range r.records {
		if record.OrganizationID == organizationID {
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].ContactID != records[j].ContactID {
			return records[i].ContactID < records[j].ContactID
		}
		return records[i].Channel < records[j].Channel
	})
	return writeJSONAtomic(path, records)
}

// GrantConsent registra consentimento explícito.
func (r *ConsentRegistry) GrantConsent(organizationID, channel, contactID, source, evidence, actor string, expiresAt *time.Time, now time.Time) (ConsentRecord, error) {
	organizationID = consentScope(organizationID)
	channel = strings.ToLower(strings.TrimSpace(channel))
	contactID = strings.TrimSpace(contactID)
	if organizationID == "" || channel == "" || contactID == "" {
		return ConsentRecord{}, fmt.Errorf("%w: organization, channel and contact are required", ErrConsentInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := consentKey(organizationID, channel, contactID)
	record := r.records[key]
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	record = ConsentRecord{
		OrganizationID: organizationID,
		Channel:        channel,
		ContactID:      contactID,
		State:          ConsentGranted,
		Source:         strings.TrimSpace(source),
		Evidence:       strings.TrimSpace(evidence),
		GrantedBy:      strings.TrimSpace(actor),
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		ExpiresAt:      expiresAt,
	}
	r.records[key] = record
	if err := r.persistOrganizationLocked(organizationID); err != nil {
		delete(r.records, key)
		return ConsentRecord{}, err
	}
	return record, nil
}

// RevokeConsent registra o opt-out. Motivo é obrigatório: um descadastro sem
// justificativa não é auditável.
func (r *ConsentRegistry) RevokeConsent(organizationID, channel, contactID, reason, actor string, now time.Time) (ConsentRecord, error) {
	organizationID = consentScope(organizationID)
	channel = strings.ToLower(strings.TrimSpace(channel))
	contactID = strings.TrimSpace(contactID)
	reason = strings.TrimSpace(reason)
	if organizationID == "" || channel == "" || contactID == "" {
		return ConsentRecord{}, fmt.Errorf("%w: organization, channel and contact are required", ErrConsentInvalid)
	}
	if reason == "" {
		return ConsentRecord{}, ErrConsentReasonNeeded
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := consentKey(organizationID, channel, contactID)
	record := r.records[key]
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	// O opt-out PRESERVA a origem do consentimento original: apagar a história
	// dificultaria provar quando e como o contato consentiu antes de sair.
	record.State = ConsentRevoked
	record.Reason = reason
	record.RevokedBy = strings.TrimSpace(actor)
	record.UpdatedAt = now
	record.ExpiresAt = nil
	if record.CreatedAt.IsZero() {
		record.CreatedAt = createdAt
	}
	record.OrganizationID = organizationID
	record.Channel = channel
	record.ContactID = contactID
	r.records[key] = record
	if err := r.persistOrganizationLocked(organizationID); err != nil {
		return ConsentRecord{}, err
	}
	return record, nil
}

// ConsentFor devolve o registro de consentimento, se existir.
func (r *ConsentRegistry) ConsentFor(organizationID, channel, contactID string) (ConsentRecord, bool) {
	organizationID = consentScope(organizationID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[consentKey(organizationID, channel, contactID)]
	return record, ok
}

// ListConsents lista os consentimentos de uma organização.
func (r *ConsentRegistry) ListConsents(organizationID string) []ConsentRecord {
	organizationID = strings.TrimSpace(organizationID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ConsentRecord, 0)
	for _, record := range r.records {
		if organizationID != "" && record.OrganizationID != organizationID {
			continue
		}
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

// AuthorizeContact decide se o contato PODE ser procurado neste canal.
// É o ÚNICO ponto de decisão dos caminhos de envio: um teste garante que os canais
// usam esta função, para o controle não valer só num caminho.
//
// Devolve o sentinela PURO (sem embrulho): os caminhos de canal do repositório
// comparam erros por identidade (`==`), e um erro embrulhado faria o bloqueio parecer
// "outro erro" para quem chama — inclusive para os testes de segurança.
func (r *ConsentRegistry) AuthorizeContact(organizationID, channel, contactID string, now time.Time) error {
	if r == nil {
		return ErrConsentMissing
	}
	organizationID = consentScope(organizationID)
	record, ok := r.ConsentFor(organizationID, channel, contactID)
	if !ok {
		return ErrConsentMissing
	}
	if record.State == ConsentRevoked {
		return ErrConsentRevoked
	}
	if record.State != ConsentGranted {
		return ErrConsentMissing
	}
	if record.ExpiresAt != nil && !record.ExpiresAt.After(now) {
		return ErrConsentExpired
	}
	return nil
}

// authorizedContact é a chamada que TODOS os caminhos de envio fazem. Registro
// ausente NEGA o contato: caminho não instrumentado não pode virar caminho livre.
func authorizedContact(registry *ConsentRegistry, organizationID, channel, contactID string, now time.Time) error {
	return registry.AuthorizeContact(organizationID, channel, contactID, now)
}
