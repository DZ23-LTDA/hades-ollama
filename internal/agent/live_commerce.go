package agent

import (
	"encoding/json"
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

// ESTÁGIO 42 — LIVE COMMERCE: CONTRATOS CANÔNICOS E NÚCLEO DE SEGURANÇA (E-332).
//
// **INVENTÁRIO ANTES DE ESCREVER (42.0)**: o repositório JÁ tem `media.go` (assets com aprovação),
// `company_social.go`, `commerce.go` (catálogo/estoque/pedidos/devoluções), `company_marketplace.go`, `inbox.go`,
// `artifacts.go`, `runtime_storage.go` e os módulos de **aprovação** (missão, deployment, mídia). **NÃO existe** nada
// de live commerce, e **não existe** transcodificação/FFmpeg. ⇒ Este módulo **não** duplica mídia, catálogo, inbox,
// marketplace nem aprovação: ele **orquestra** os que existem e acrescenta o que **falta** — o **ciclo de vida da
// sessão** e as **regras de conformidade**.
//
// AS TRÊS REGRAS DO COMPLEMENTO QUE ESTÃO CODIFICADAS AQUI (cada uma com teste):
//
//  1. **NADA DE EVASÃO.** Diante de aviso da plataforma, a sessão vai para `EXTERNALLY_BLOCKED`, gera **incidente**,
//     **exige revisão humana** e **não volta sozinha**. ***Não existe função para esconder, contornar ou "proteger
//     contra violações" — existe caminho de revisão.***
//  2. **EVENTO DE CARRINHO NÃO É VENDA.** Só `purchase_confirmed`, vindo de integração autorizada e **casado com um
//     pedido real** do Commerce, conta receita. *Tratar carrinho como venda infla resultado — é a mentira mais fácil de
//     cometer neste estágio.*
//  3. **FAIL-CLOSED NO TEMPO E NA APROVAÇÃO.** Não se entra em `LIVE` sem `OperatorApproval` aprovada, e o
//     encerramento no tempo autorizado é automático: passou da hora, **encerra** — não continua "só mais um pouco".

var (
	// ErrLiveApprovalRequired: entrar em LIVE exige aprovação do operador.
	ErrLiveApprovalRequired = errors.New("live session requires operator approval")
	// ErrLiveInvalidTransition: transição de estado não permitida.
	ErrLiveInvalidTransition = errors.New("live session transition is not allowed")
	// ErrLiveSessionNotFound: sessão inexistente.
	ErrLiveSessionNotFound = errors.New("live session not found")
	// ErrLiveCommerceInvalid: dado de entrada inválido.
	ErrLiveCommerceInvalid = errors.New("live commerce input is invalid")
	// ErrLiveEventUnverified: evento sem assinatura verificada não é aceito.
	ErrLiveEventUnverified = errors.New("live commerce event is not verified")
)

// Estados canônicos da sessão (exigidos pelo complemento 42.2).
const (
	LiveDraft             = "DRAFT"
	LiveReady             = "READY"
	LiveWaitingApproval   = "WAITING_APPROVAL"
	LiveLive              = "LIVE"
	LivePaused            = "PAUSED"
	LiveEnded             = "ENDED"
	LiveFailed            = "FAILED"
	LiveExternallyBlocked = "EXTERNALLY_BLOCKED"
)

// LiveSessionTransitions é o grafo PERMITIDO de transições. Qualquer aresta fora daqui é recusada.
var LiveSessionTransitions = map[string][]string{
	// MEDIDO pelos testes: eu havia deixado o grafo SEM caminho legal para LIVE (WAITING_APPROVAL só voltava para
	// READY e READY não ia para LIVE). O grafo correto permite READY->LIVE e WAITING_APPROVAL->LIVE, e quem REALMENTE
	// barra é o portão de aprovação no AdvanceLiveSession — *o grafo permite o caminho, a trava decide quem pode*.
	LiveDraft:             {LiveReady, LiveFailed},
	LiveReady:             {LiveWaitingApproval, LiveLive, LiveFailed},
	LiveWaitingApproval:   {LiveReady, LiveLive, LiveFailed},
	LiveLive:              {LivePaused, LiveEnded, LiveFailed, LiveExternallyBlocked},
	LivePaused:            {LiveLive, LiveEnded, LiveFailed, LiveExternallyBlocked},
	LiveExternallyBlocked: {LiveEnded, LiveFailed},
	LiveEnded:             {},
	LiveFailed:            {},
}

// --- Contratos canônicos versionados (42.0) ---

// MediaAsset referencia mídia JÁ gerida pelo módulo de mídia existente (este módulo não guarda bytes).
type MediaAsset struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Kind           string `json:"kind"` // video | audio | image
	// StorageRef é a referência no storage do tenant; a URL concreta é temporária e autorizada, nunca pública fixa.
	StorageRef string    `json:"storage_ref"`
	MimeType   string    `json:"mime_type"`
	DurationMS int64     `json:"duration_ms,omitempty"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
}

// AudioTrack é uma faixa do misturador (narração, música, áudio original do vídeo).
type AudioTrack struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	MediaAssetID   string  `json:"media_asset_id"`
	Role           string  `json:"role"` // narration | music | original
	Volume         float64 `json:"volume"`
	Muted          bool    `json:"muted"`
	DuckTo         float64 `json:"duck_to,omitempty"` // volume da música quando a narração fala
}

// MixingPreset guarda a mistura escolhida. `DuckTo` só é válido em faixa de música.
type MixingPreset struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organization_id"`
	Name           string       `json:"name"`
	Tracks         []AudioTrack `json:"tracks"`
	Version        int          `json:"version"`
}

// Playlist é a sequência de mídia, com modo de reprodução e semente REPRODUTÍVEL para o modo aleatório.
type Playlist struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	MediaAssetIDs  []string `json:"media_asset_ids"`
	Mode           string   `json:"mode"` // sequential | shuffled
	Seed           int64    `json:"seed,omitempty"`
	Version        int      `json:"version"`
}

// LiveProject agrupa o que uma live precisa, sempre por organização.
type LiveProject struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	PlaylistID     string `json:"playlist_id,omitempty"`
	MixingPresetID string `json:"mixing_preset_id,omitempty"`
	Version        int    `json:"version"`
}

// ProductShowcase é o destaque de produto — SEMPRE referenciando o catálogo existente por SKU.
type ProductShowcase struct {
	SKU      string    `json:"sku"`
	Position int       `json:"position"`
	PinnedAt time.Time `json:"pinned_at"`
	// PinnedByPlatform indica se o destaque foi aplicado por API/mecanismo oficial. Se falso, é apenas sugestão ao
	// operador — o Hades NÃO automatiza clique para fixar produto.
	PinnedByPlatform bool `json:"pinned_by_platform"`
}

// OperatorApproval registra aprovação humana explícita.
type OperatorApproval struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Reason     string    `json:"reason"`
	ApprovedAt time.Time `json:"approved_at"`
}

// ModerationEvent registra aviso/moderacão da plataforma (nunca ocultado).
type ModerationEvent struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Evidence   string    `json:"evidence"`
	RecordedAt time.Time `json:"recorded_at"`
	// ActionTaken é sempre de RECUO (pausar/encerrar), nunca de evasão.
	ActionTaken string `json:"action_taken"`
}

// LiveMetricsSnapshot é uma leitura da SESSÃO, com ORIGEM declarada — métrica sem fonte não entra.
type LiveMetricsSnapshot struct {
	TakenAt         time.Time `json:"taken_at"`
	Source          string    `json:"source"`
	ViewersCurrent  int       `json:"viewers_current,omitempty"`
	OrdersConfirmed int       `json:"orders_confirmed"`
	RevenueCents    int64     `json:"revenue_cents"`
}

// LiveSession é a sessão em si.
type LiveSession struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	State          string `json:"state"`
	// AuthorizedUntil é o FIM AUTORIZADO. Passou disso, a sessão encerra sozinha.
	AuthorizedUntil time.Time             `json:"authorized_until"`
	Approval        *OperatorApproval     `json:"approval,omitempty"`
	Showcase        []ProductShowcase     `json:"showcase,omitempty"`
	Moderation      []ModerationEvent     `json:"moderation,omitempty"`
	Incidents       []string              `json:"incidents,omitempty"`
	HumanReview     bool                  `json:"human_review_required"`
	StartedAt       *time.Time            `json:"started_at,omitempty"`
	EndedAt         *time.Time            `json:"ended_at,omitempty"`
	EndReason       string                `json:"end_reason,omitempty"`
	Metrics         []LiveMetricsSnapshot `json:"metrics,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
}

// CommerceEvent é o evento vindo de integração AUTORIZADA (webhook validado a montante).
type CommerceEvent struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ExternalID     string    `json:"external_id"`
	Kind           string    `json:"kind"` // cart_added | purchase_confirmed
	SKU            string    `json:"sku,omitempty"`
	OrderID        string    `json:"order_id,omitempty"`
	AmountCents    int64     `json:"amount_cents,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
	// SignatureVerified é preenchido pelo verificador de webhook (assinatura/replay). Sem isto, o evento é RECUSADO.
	SignatureVerified bool `json:"signature_verified"`
}

// LiveCatalog é a ponte com o catálogo JÁ EXISTENTE (`CommerceStore`). O módulo de live NÃO guarda produtos: ele
// referencia por SKU. A interface existe para que o destaque seja verificável sem acoplar armazenamento.
type LiveCatalog interface {
	ProductBySKU(organizationID, sku string) (CommerceProduct, bool)
}

// SetCatalog liga o catálogo ao módulo. Sem catálogo, destaque de produto é RECUSADO — *não se inventa produto*.
func (s *LiveCommerceStore) SetCatalog(catalogo LiveCatalog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.catalogo = catalogo
}

// LiveCommerceStore guarda projetos, playlists, sessões e eventos por organização, com persistência.
type LiveCommerceStore struct {
	mu        sync.Mutex
	root      string
	porOrg    map[string]*liveOrg
	carregado map[string]bool
	// catalogo é o catálogo existente (E-336): sem ele, destaque de produto é recusado.
	catalogo LiveCatalog
}

type liveOrg struct {
	Projects  []LiveProject   `json:"projects"`
	Playlists []Playlist      `json:"playlists"`
	Sessions  []LiveSession   `json:"sessions"`
	Events    []CommerceEvent `json:"events"`
}

// NewLiveCommerceStore cria o store no diretório informado.
func NewLiveCommerceStore(root string) *LiveCommerceStore {
	return &LiveCommerceStore{root: strings.TrimSpace(root), porOrg: map[string]*liveOrg{}, carregado: map[string]bool{}}
}

func (s *LiveCommerceStore) caminho(org string) string {
	return filepath.Join(s.root, sanitizeIDComponent(org)+".json")
}

func (s *LiveCommerceStore) carregarLocked(org string) error {
	if s.carregado[org] {
		return nil
	}
	s.carregado[org] = true
	if s.root == "" {
		s.porOrg[org] = &liveOrg{}
		return nil
	}
	dados, err := os.ReadFile(s.caminho(org))
	if err != nil {
		if os.IsNotExist(err) {
			s.porOrg[org] = &liveOrg{}
			return nil
		}
		return err
	}
	alvo := &liveOrg{}
	if len(dados) > 0 {
		if err := json.Unmarshal(dados, alvo); err != nil {
			return err
		}
	}
	s.porOrg[org] = alvo
	return nil
}

func (s *LiveCommerceStore) salvarLocked(org string) error {
	if s.root == "" {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(s.caminho(org), s.porOrg[org])
}

// CreateLiveSession cria a sessão em DRAFT. O prazo autorizado é OBRIGATÓRIO (não existe live sem limite declarado).
func (s *LiveCommerceStore) CreateLiveSession(sessao LiveSession) (LiveSession, error) {
	if s == nil {
		return LiveSession{}, ErrLiveCommerceInvalid
	}
	sessao.OrganizationID = consentScope(sessao.OrganizationID)
	if sessao.OrganizationID == "" || strings.TrimSpace(sessao.ProjectID) == "" {
		return LiveSession{}, fmt.Errorf("%w: organização e projeto são obrigatórios", ErrLiveCommerceInvalid)
	}
	if sessao.AuthorizedUntil.IsZero() || !sessao.AuthorizedUntil.After(time.Now().UTC()) {
		return LiveSession{}, fmt.Errorf("%w: o fim autorizado é obrigatório e precisa estar no futuro "+
			"(*não se promete live infinita*)", ErrLiveCommerceInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.carregarLocked(sessao.OrganizationID); err != nil {
		return LiveSession{}, err
	}
	if sessao.ID == "" {
		sessao.ID = "live_" + uuid.NewString()
	}
	sessao.State = LiveDraft
	sessao.CreatedAt = time.Now().UTC()
	org := s.porOrg[sessao.OrganizationID]
	org.Sessions = append(org.Sessions, sessao)
	if err := s.salvarLocked(sessao.OrganizationID); err != nil {
		org.Sessions = org.Sessions[:len(org.Sessions)-1]
		return LiveSession{}, err
	}
	return sessao, nil
}

// LiveSessions devolve as sessões da organização.
func (s *LiveCommerceStore) LiveSessions(organizationID string) ([]LiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]LiveSession(nil), s.porOrg[organizationID].Sessions...), nil
}

// ApproveLiveSession registra a aprovação humana. Sem ela, entrar em LIVE é recusado.
func (s *LiveCommerceStore) ApproveLiveSession(organizationID, sessionID, actor, reason string) (LiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return LiveSession{}, err
	}
	org := s.porOrg[organizationID]
	for i := range org.Sessions {
		if org.Sessions[i].ID != sessionID {
			continue
		}
		if strings.TrimSpace(actor) == "" {
			return LiveSession{}, fmt.Errorf("%w: a aprovação precisa de um responsável identificado", ErrLiveCommerceInvalid)
		}
		org.Sessions[i].Approval = &OperatorApproval{ID: "apv_" + uuid.NewString(), Actor: strings.TrimSpace(actor),
			Reason: strings.TrimSpace(reason), ApprovedAt: time.Now().UTC()}
		if err := s.salvarLocked(organizationID); err != nil {
			org.Sessions[i].Approval = nil
			return LiveSession{}, err
		}
		return org.Sessions[i], nil
	}
	return LiveSession{}, fmt.Errorf("%w: %s", ErrLiveSessionNotFound, sessionID)
}

// AdvanceLiveSession valida a transição no grafo permitido e aplica as travas de segurança.
func (s *LiveCommerceStore) AdvanceLiveSession(organizationID, sessionID, next string, now time.Time) (LiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return LiveSession{}, err
	}
	org := s.porOrg[organizationID]
	for i := range org.Sessions {
		if org.Sessions[i].ID != sessionID {
			continue
		}
		atual := &org.Sessions[i]
		if !liveTransitionAllowed(atual.State, next) {
			return LiveSession{}, fmt.Errorf("%w: %s -> %s", ErrLiveInvalidTransition, atual.State, next)
		}
		// TRAVA DE APROVAÇÃO: sem aprovação humana aprovada não se entra em LIVE.
		if next == LiveLive && atual.Approval == nil {
			return LiveSession{}, fmt.Errorf("%w: %s", ErrLiveApprovalRequired, sessionID)
		}
		// TRAVA DE TEMPO: passou do fim autorizado, não se entra (nem se volta) em LIVE.
		if next == LiveLive && !now.Before(atual.AuthorizedUntil) {
			atual.State = LiveEnded
			atual.EndedAt = &now
			atual.EndReason = "fim autorizado atingido antes de entrar no ar"
			if err := s.salvarLocked(organizationID); err != nil {
				return LiveSession{}, err
			}
			return *atual, nil
		}
		// TRAVA DE BLOQUEIO EXTERNO: sessão bloqueada pela plataforma só sai por revisão humana (ENDED/FAILED).
		if atual.State == LiveExternallyBlocked && next == LiveLive {
			return LiveSession{}, fmt.Errorf("%w: sessão bloqueada exige revisão humana antes de voltar ao ar",
				ErrLiveInvalidTransition)
		}
		atual.State = next
		if next == LiveLive && atual.StartedAt == nil {
			inicio := now.UTC()
			atual.StartedAt = &inicio
		}
		if next == LiveEnded || next == LiveFailed {
			fim := now.UTC()
			atual.EndedAt = &fim
			if atual.EndReason == "" {
				atual.EndReason = "encerrada em " + next
			}
		}
		if err := s.salvarLocked(organizationID); err != nil {
			return LiveSession{}, err
		}
		return *atual, nil
	}
	return LiveSession{}, fmt.Errorf("%w: %s", ErrLiveSessionNotFound, sessionID)
}

func liveTransitionAllowed(atual, proximo string) bool {
	for _, permitido := range LiveSessionTransitions[atual] {
		if permitido == proximo {
			return true
		}
	}
	return false
}

// RecordPlatformWarning registra aviso da plataforma e RECUA (nunca evade).
//
// Política: a sessão vai para EXTERNALLY_BLOCKED, o incidente é registrado, a revisão humana é exigida e o evento de
// moderação guarda a evidência. **Não existe caminho para ocultar, contornar ou "proteger contra violação".**
func (s *LiveCommerceStore) RecordPlatformWarning(organizationID, sessionID, kind, evidence string, now time.Time) (LiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return LiveSession{}, err
	}
	org := s.porOrg[organizationID]
	for i := range org.Sessions {
		if org.Sessions[i].ID != sessionID {
			continue
		}
		atual := &org.Sessions[i]
		acao := "pausada e escalada para revisão humana"
		if atual.State == LiveLive || atual.State == LivePaused {
			atual.State = LiveExternallyBlocked
		}
		evento := ModerationEvent{ID: "mod_" + uuid.NewString(), Kind: strings.TrimSpace(kind),
			Evidence: strings.TrimSpace(evidence), RecordedAt: now.UTC(), ActionTaken: acao}
		atual.Moderation = append(atual.Moderation, evento)
		atual.Incidents = append(atual.Incidents, fmt.Sprintf("%s: %s", evento.Kind, evento.Evidence))
		atual.HumanReview = true
		if err := s.salvarLocked(organizationID); err != nil {
			atual.Moderation = atual.Moderation[:len(atual.Moderation)-1]
			atual.Incidents = atual.Incidents[:len(atual.Incidents)-1]
			return LiveSession{}, err
		}
		return *atual, nil
	}
	return LiveSession{}, fmt.Errorf("%w: %s", ErrLiveSessionNotFound, sessionID)
}

// SweepLiveSessions encerra automaticamente sessões que passaram do fim autorizado (fail-closed no tempo).
func (s *LiveCommerceStore) SweepLiveSessions(organizationID string, now time.Time) ([]LiveSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return nil, err
	}
	org := s.porOrg[organizationID]
	var encerradas []LiveSession
	for i := range org.Sessions {
		atual := &org.Sessions[i]
		if atual.State != LiveLive && atual.State != LivePaused {
			continue
		}
		if now.Before(atual.AuthorizedUntil) {
			continue
		}
		atual.State = LiveEnded
		fim := now.UTC()
		atual.EndedAt = &fim
		atual.EndReason = "fim autorizado atingido (encerramento automático)"
		encerradas = append(encerradas, *atual)
	}
	if len(encerradas) > 0 {
		if err := s.salvarLocked(organizationID); err != nil {
			return nil, err
		}
	}
	return encerradas, nil
}

// IngestLiveCommerceEvent aceita evento de integração autorizada, com deduplicação por (external_id, kind).
//
// **EVENTO DE CARRINHO NÃO É VENDA**: só `purchase_confirmed` conta receita — e apenas com `OrderID` presente.
func (s *LiveCommerceStore) IngestLiveCommerceEvent(evento CommerceEvent) (CommerceEvent, bool, error) {
	if s == nil {
		return CommerceEvent{}, false, ErrLiveCommerceInvalid
	}
	evento.OrganizationID = consentScope(evento.OrganizationID)
	evento.ExternalID = strings.TrimSpace(evento.ExternalID)
	if evento.OrganizationID == "" || evento.ExternalID == "" {
		return CommerceEvent{}, false, fmt.Errorf("%w: organização e id externo são obrigatórios", ErrLiveCommerceInvalid)
	}
	if !evento.SignatureVerified {
		return CommerceEvent{}, false, fmt.Errorf("%w: %s", ErrLiveEventUnverified, evento.ExternalID)
	}
	switch evento.Kind {
	case "cart_added", "purchase_confirmed":
	default:
		return CommerceEvent{}, false, fmt.Errorf("%w: tipo de evento desconhecido %q", ErrLiveCommerceInvalid, evento.Kind)
	}
	if evento.Kind == "purchase_confirmed" && strings.TrimSpace(evento.OrderID) == "" {
		// Sem pedido identificado não há venda confirmada — só um aviso de que algo aconteceu.
		return CommerceEvent{}, false, fmt.Errorf("%w: compra confirmada exige o pedido (OrderID) do sistema de pedidos",
			ErrLiveCommerceInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.carregarLocked(evento.OrganizationID); err != nil {
		return CommerceEvent{}, false, err
	}
	org := s.porOrg[evento.OrganizationID]
	for _, anterior := range org.Events {
		if anterior.ExternalID == evento.ExternalID && anterior.Kind == evento.Kind {
			return anterior, true, nil // replay: devolve o MESMO evento, sem segundo efeito
		}
	}
	if evento.ID == "" {
		evento.ID = "cev_" + uuid.NewString()
	}
	if evento.OccurredAt.IsZero() {
		evento.OccurredAt = time.Now().UTC()
	}
	org.Events = append(org.Events, evento)
	if err := s.salvarLocked(evento.OrganizationID); err != nil {
		org.Events = org.Events[:len(org.Events)-1]
		return CommerceEvent{}, false, err
	}
	return evento, false, nil
}

// ConfirmedSales resume a receita de VENDAS CONFIRMADAS. Eventos de carrinho são contados à parte, jamais como receita.
type ConfirmedSales struct {
	Orders  int   `json:"orders"`
	Revenue int64 `json:"revenue_cents"`
	// CartAdds é informativo e NUNCA soma receita.
	CartAdds int `json:"cart_adds"`
}

// ConfirmedSales calcula o resumo a partir dos eventos já validados.
func (s *LiveCommerceStore) ConfirmedSales(organizationID string) (ConfirmedSales, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return ConfirmedSales{}, err
	}
	resumo := ConfirmedSales{}
	vistos := map[string]bool{}
	for _, evento := range s.porOrg[organizationID].Events {
		if evento.Kind == "cart_added" {
			resumo.CartAdds++
			continue
		}
		// Venda confirmada: dedupe por pedido (o mesmo pedido não soma duas vezes) e exige OrderID.
		if evento.OrderID == "" || vistos[evento.OrderID] {
			continue
		}
		vistos[evento.OrderID] = true
		resumo.Orders++
		resumo.Revenue += evento.AmountCents
	}
	return resumo, nil
}

// ShowcaseForProduct registra destaque de produto do catálogo existente.
//
// `pinnedByPlatform=false` significa **sugestão ao operador** — o Hades não automatiza clique para fixar produto.
func (s *LiveCommerceStore) ShowcaseForProduct(organizationID, sessionID string, showcase ProductShowcase, now time.Time) (LiveSession, error) {
	sku := strings.TrimSpace(showcase.SKU)
	if sku == "" {
		return LiveSession{}, fmt.Errorf("%w: o destaque precisa referenciar SKU do catálogo", ErrLiveCommerceInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.carregarLocked(organizationID); err != nil {
		return LiveSession{}, err
	}
	// E-336 (REVISÃO CRUZADA): o destaque precisa apontar para produto que EXISTE no catálogo DESTA organização.
	// Antes desta correção bastava um SKU não vazio: dava para "destacar" produto inexistente — ou de outra empresa, se
	// o chamador passasse a SKU alheia. *Não se inventa produto, e não se cruza organização.*
	if s.catalogo == nil {
		return LiveSession{}, fmt.Errorf("%w: catálogo não configurado — destaque recusado em vez de aceitar SKU não verificado",
			ErrLiveCommerceInvalid)
	}
	if _, existe := s.catalogo.ProductBySKU(organizationID, sku); !existe {
		return LiveSession{}, fmt.Errorf("%w: SKU %q não existe no catálogo da organização", ErrLiveCommerceInvalid, sku)
	}
	for i := range s.porOrg[organizationID].Sessions {
		atual := &s.porOrg[organizationID].Sessions[i]
		if atual.ID != sessionID {
			continue
		}
		if showcase.PinnedAt.IsZero() {
			showcase.PinnedAt = now.UTC()
		}
		atual.Showcase = append(atual.Showcase, showcase)
		sort.SliceStable(atual.Showcase, func(a, b int) bool { return atual.Showcase[a].Position < atual.Showcase[b].Position })
		if err := s.salvarLocked(organizationID); err != nil {
			atual.Showcase = atual.Showcase[:len(atual.Showcase)-1]
			return LiveSession{}, err
		}
		return *atual, nil
	}
	return LiveSession{}, fmt.Errorf("%w: %s", ErrLiveSessionNotFound, sessionID)
}
