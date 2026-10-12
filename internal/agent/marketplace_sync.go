package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// EstÃ¡gio 25 â€” MARKETPLACE: contrato de sincronizaÃ§Ã£o com duble de sandbox.
//
// O estÃ¡gio pedia "conectores oficiais operando em sandbox com evidÃªncias" e a homologaÃ§Ã£o
// live declarada bloqueada por fornecedor. O que existe aqui Ã© **exatamente isso, e nada alÃ©m**:
// um **contrato** de sincronizaÃ§Ã£o (produtos, preÃ§os, estoque e pedidos) exercitado contra um
// **duble de sandbox** em processo. Nenhuma chamada a Mercado Livre, Amazon, Shopee ou qualquer
// plataforma acontece â€” e o relatÃ³rio **declara** isso em cada execuÃ§Ã£o.
//
// DecisÃµes deliberadas:
//
//   - **CATALOGAR NÃƒO Ã‰ HABILITAR** (Â§11.3): o canal expÃµe estado real. Sem credencial
//     configurada ele Ã© `CONFIG_REQUIRED`; com credencial mas sem teste, `UNAUTHORIZED`; e o
//     `CONNECTED` sÃ³ vale para o **duble de sandbox**, porque nÃ£o houve handshake real;
//   - **LIMITE DE OPERAÃ‡Ã•ES POR EXECUÃ‡ÃƒO**: um conector que "sincroniza tudo" sem teto Ã© a forma
//     mais rÃ¡pida de violar rate limit de plataforma. O teto Ã© explÃ­cito e a violaÃ§Ã£o Ã© recusada
//     **antes** de aplicar qualquer efeito;
//   - **IDEMPOTÃŠNCIA POR VERSÃƒO**: reexecutar a mesma sincronizaÃ§Ã£o **nÃ£o** reaplica o que jÃ¡ foi
//     aplicado; o item aparece como `ignorado`, nÃ£o como sucesso novo;
//   - **CONFLITO NÃƒO Ã‰ SOBRESCRITA SILENCIOSA**: quando o estoque remoto diverge do local, o item
//     Ã© registrado como **conflito** e **nÃ£o** Ã© aplicado. Sobrescrever esconderia divergÃªncia de
//     estoque â€” exatamente o erro que causa venda sem produto;
//   - **SEM PUBLICAÃ‡ÃƒO EXTERNA**: o relatÃ³rio traz `HomologacaoLive` como
//     `BLOCKED_BY_EXTERNAL_DEPENDENCY` com o motivo e o procedimento de desbloqueio.

const (
	// MarketplaceStateConfigRequired indica canal sem credencial configurada.
	MarketplaceStateConfigRequired = "CONFIG_REQUIRED"
	// MarketplaceStateUnauthorized indica credencial presente sem teste aprovado.
	MarketplaceStateUnauthorized = "UNAUTHORIZED"
	// MarketplaceStateConnected indica canal operante â€” no caso, contra o duble de sandbox.
	MarketplaceStateConnected = "CONNECTED"
	// MarketplaceStateUnsupported indica canal fora do escopo.
	MarketplaceStateUnsupported = "UNSUPPORTED"

	// MarketplaceLiveStatus Ã© o estado declarado da homologaÃ§Ã£o live.
	MarketplaceLiveStatus = "BLOCKED_BY_EXTERNAL_DEPENDENCY"
	// MarketplaceLiveReason explica o desbloqueio da homologaÃ§Ã£o live.
	MarketplaceLiveReason = "sincronizar com marketplace real exige conta autorizada, aplicaÃ§Ã£o registrada e " +
		"scopes vÃ¡lidos da plataforma; sem credencial, a sincronizaÃ§Ã£o roda apenas contra o duble de sandbox"

	// MarketplaceMaxOperationsPerRun Ã© o teto de operaÃ§Ãµes por execuÃ§Ã£o.
	MarketplaceMaxOperationsPerRun = 50

	// Entidades suportadas pelo contrato de sincronizaÃ§Ã£o.
	MarketplaceEntityProducts = "produtos"
	MarketplaceEntityPrices   = "precos"
	MarketplaceEntityStock    = "estoque"
	MarketplaceEntityOrders   = "pedidos"
)

var (
	ErrMarketplaceChannelUnknown = errors.New("marketplace channel unknown")
	ErrMarketplaceRateLimited    = errors.New("marketplace sync exceeds the per-run operation limit")
	ErrMarketplaceSyncInvalid    = errors.New("marketplace sync request invalid")
)

// MarketplaceChannelSpec descreve um canal no catÃ¡logo, com estado REAL e capacidades.
type MarketplaceChannelSpec struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// CredentialEnv Ã© o NOME da variÃ¡vel de ambiente que carrega a credencial. O valor nunca Ã©
	// lido nem exibido.
	CredentialEnv string   `json:"credential_env"`
	State         string   `json:"state"`
	Capabilities  []string `json:"capabilities"`
	// LiveTested Ã© false enquanto nÃ£o houve handshake real: Ã© o que separa o catÃ¡logo da operaÃ§Ã£o.
	LiveTested bool `json:"live_tested"`
	// Unblock descreve o que falta para habilitar de verdade.
	Unblock string `json:"unblock_procedure,omitempty"`
}

// MarketplaceChannels devolve o catÃ¡logo com o estado declarado por canal.
//
// `configuredEnv` informa quais variÃ¡veis de credencial existem no ambiente do operador â€” apenas
// os NOMES sÃ£o considerados, nunca os valores.
func MarketplaceChannels(configuredEnv map[string]bool) []MarketplaceChannelSpec {
	specs := []MarketplaceChannelSpec{
		{
			ID: "mercado_livre", Name: "Mercado Livre", CredentialEnv: "HADES_MARKETPLACE_MERCADO_LIVRE_TOKEN",
			Capabilities: []string{MarketplaceEntityProducts, MarketplaceEntityPrices, MarketplaceEntityStock, MarketplaceEntityOrders},
		},
		{
			ID: "amazon", Name: "Amazon", CredentialEnv: "HADES_MARKETPLACE_AMAZON_TOKEN",
			Capabilities: []string{MarketplaceEntityProducts, MarketplaceEntityPrices, MarketplaceEntityStock},
		},
		{
			ID: "shopee", Name: "Shopee", CredentialEnv: "HADES_MARKETPLACE_SHOPEE_TOKEN",
			Capabilities: []string{MarketplaceEntityProducts, MarketplaceEntityPrices, MarketplaceEntityStock},
		},
	}
	for index := range specs {
		spec := &specs[index]
		if !configuredEnv[spec.CredentialEnv] {
			spec.State = MarketplaceStateConfigRequired
			spec.Unblock = "definir a variÃ¡vel " + spec.CredentialEnv + " e executar um teste de conexÃ£o real"
			continue
		}
		// Credencial presente NÃƒO Ã© operaÃ§Ã£o: sem handshake real o estado Ã© UNAUTHORIZED.
		spec.State = MarketplaceStateUnauthorized
		spec.Unblock = "executar teste de conexÃ£o real e registrar a evidÃªncia de handshake"
	}
	return specs
}

// MarketplaceSyncItem Ã© um item do espelho local que serÃ¡ enviado ao canal.
type MarketplaceSyncItem struct {
	Entity string `json:"entity"`
	SKU    string `json:"sku"`
	// ExternalID Ã© o identificador do item NA PLATAFORMA (ex.: id do pedido no canal). Ã‰ a chave de
	// idempotÃªncia do pull de pedidos: o mesmo pedido nÃ£o pode entrar duas vezes.
	ExternalID string `json:"external_id,omitempty"`
	// Version permite idempotÃªncia: a mesma versÃ£o nÃ£o Ã© aplicada duas vezes.
	Version int64 `json:"version"`
	// Quantity Ã© usado para estoque; PriceCents para preÃ§o.
	Quantity   int64 `json:"quantity,omitempty"`
	PriceCents int64 `json:"price_cents,omitempty"`
}

// MarketplaceSyncRequest Ã© o pedido de sincronizaÃ§Ã£o.
type MarketplaceSyncRequest struct {
	OrganizationID string
	ChannelID      string
	Direction      string
	Items          []MarketplaceSyncItem
}

// MarketplaceSyncOutcome Ã© o resultado por item.
type MarketplaceSyncOutcome struct {
	Entity  string `json:"entity"`
	SKU     string `json:"sku"`
	Version int64  `json:"version"`
	Result  string `json:"result"`
	Detail  string `json:"detail,omitempty"`
}

const (
	// Resultados possÃ­veis por item.
	MarketplaceResultApplied  = "aplicado"
	MarketplaceResultSkipped  = "ignorado"
	MarketplaceResultConflict = "conflito"
)

// MarketplaceSyncReport Ã© o relatÃ³rio da execuÃ§Ã£o.
type MarketplaceSyncReport struct {
	ChannelID      string                   `json:"channel_id"`
	Direction      string                   `json:"direction"`
	Sandbox        bool                     `json:"sandbox"`
	LiveStatus     string                   `json:"homologacao_live"`
	LiveReason     string                   `json:"motivo"`
	Applied        int                      `json:"applied"`
	Skipped        int                      `json:"skipped"`
	Conflicts      int                      `json:"conflicts"`
	Outcomes       []MarketplaceSyncOutcome `json:"outcomes"`
	OperationsUsed int                      `json:"operations_used"`
	OperationLimit int                      `json:"operation_limit"`
}

// marketplaceRemote Ã© o DUBLE de sandbox: guarda o espelho remoto em memÃ³ria.
type marketplaceRemote struct {
	stock map[string]int64
}

// MarketplaceSyncStore executa a sincronizaÃ§Ã£o contra o duble e registra a idempotÃªncia.
type MarketplaceSyncStore struct {
	mu         sync.Mutex
	remote     *marketplaceRemote
	applied    map[string]bool
	commerce   *CommerceStore
	root       string
	loadedOrgs map[string]bool
}

// NewMarketplaceSyncStore cria o executor de sincronizaÃ§Ã£o.
func NewMarketplaceSyncStore(root string, commerce *CommerceStore) *MarketplaceSyncStore {
	return &MarketplaceSyncStore{
		remote:     &marketplaceRemote{stock: map[string]int64{}},
		applied:    map[string]bool{},
		commerce:   commerce,
		root:       strings.TrimSpace(root),
		loadedOrgs: map[string]bool{},
	}
}

type marketplaceSyncFile struct {
	Applied []string `json:"applied"`
}

func (s *MarketplaceSyncStore) appliedPath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, "marketplace", sanitizeIDComponent(organizationID)+".json")
}

func (s *MarketplaceSyncStore) loadLocked(organizationID string) {
	if s.loadedOrgs[organizationID] {
		return
	}
	s.loadedOrgs[organizationID] = true
	path := s.appliedPath(organizationID)
	if path == "" {
		return
	}
	var file marketplaceSyncFile
	if err := readJSON(path, &file); err != nil {
		return
	}
	for _, key := range file.Applied {
		s.applied[key] = true
	}
}

func (s *MarketplaceSyncStore) persistLocked(organizationID string) error {
	path := s.appliedPath(organizationID)
	if path == "" {
		return nil
	}
	keys := make([]string, 0, len(s.applied))
	for key := range s.applied {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return writeJSONAtomic(path, marketplaceSyncFile{Applied: keys})
}

// Sync executa a sincronizaÃ§Ã£o contra o DUBLE de sandbox.
//
// O teto de operaÃ§Ãµes Ã© verificado ANTES de qualquer efeito: um conector que aplica tudo e sÃ³
// depois reclama jÃ¡ teria violado o limite da plataforma.
func (s *MarketplaceSyncStore) Sync(request MarketplaceSyncRequest, now time.Time) (MarketplaceSyncReport, error) {
	organizationID := strings.TrimSpace(request.OrganizationID)
	channelID := strings.TrimSpace(request.ChannelID)
	if organizationID == "" || channelID == "" {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: organizaÃ§Ã£o e canal sÃ£o obrigatÃ³rios", ErrMarketplaceSyncInvalid)
	}
	if len(request.Items) == 0 {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: nada a sincronizar", ErrMarketplaceSyncInvalid)
	}
	specs := MarketplaceChannels(map[string]bool{})
	known := false
	for _, spec := range specs {
		if spec.ID == channelID {
			known = true
		}
	}
	if !known {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: %s", ErrMarketplaceChannelUnknown, channelID)
	}
	if len(request.Items) > MarketplaceMaxOperationsPerRun {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: %d itens para o teto de %d",
			ErrMarketplaceRateLimited, len(request.Items), MarketplaceMaxOperationsPerRun)
	}
	direction := strings.ToLower(strings.TrimSpace(request.Direction))
	if direction == "" {
		direction = "push"
	}
	if direction != "push" && direction != "pull" {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: direÃ§Ã£o %q desconhecida", ErrMarketplaceSyncInvalid, request.Direction)
	}

	report := MarketplaceSyncReport{
		ChannelID:      channelID,
		Direction:      direction,
		Sandbox:        true,
		LiveStatus:     MarketplaceLiveStatus,
		LiveReason:     MarketplaceLiveReason,
		OperationLimit: MarketplaceMaxOperationsPerRun,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked(organizationID)

	for _, item := range request.Items {
		entity := strings.ToLower(strings.TrimSpace(item.Entity))
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || entity == "" {
			return MarketplaceSyncReport{}, fmt.Errorf("%w: item precisa de entidade e SKU", ErrMarketplaceSyncInvalid)
		}
		key := fmt.Sprintf("%s|%s|%s|%d", channelID, entity, sku, item.Version)
		// IDEMPOTÃŠNCIA: a mesma versÃ£o nÃ£o Ã© aplicada duas vezes.
		if s.applied[key] {
			report.Skipped++
			report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
				Entity: entity, SKU: sku, Version: item.Version,
				Result: MarketplaceResultSkipped, Detail: "versÃ£o jÃ¡ aplicada",
			})
			continue
		}
		if entity == MarketplaceEntityOrders {
			if direction != "pull" {
				return MarketplaceSyncReport{}, fmt.Errorf("%w: pedido de canal entra por pull, nÃ£o por push", ErrMarketplaceSyncInvalid)
			}
			externalID := strings.TrimSpace(item.ExternalID)
			if externalID == "" {
				return MarketplaceSyncReport{}, fmt.Errorf("%w: pedido do canal precisa do id externo", ErrMarketplaceSyncInvalid)
			}
			if s.commerce == nil {
				return MarketplaceSyncReport{}, fmt.Errorf("%w: sem motor de comÃ©rcio nÃ£o hÃ¡ como registrar o pedido", ErrMarketplaceSyncInvalid)
			}
			// IDEMPOTÃŠNCIA POR ID EXTERNO (nÃ£o por versÃ£o): o mesmo pedido do canal entra UMA vez.
			orderKey := fmt.Sprintf("%s|%s|%s", channelID, entity, externalID)
			if s.applied[orderKey] {
				report.Skipped++
				report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
					Entity: entity, SKU: sku, Version: item.Version,
					Result: MarketplaceResultSkipped, Detail: "pedido jÃ¡ importado do canal",
				})
				continue
			}
			pedido, err := s.commerce.CreateOrder(CommerceOrderRequest{
				OrganizationID: organizationID,
				Items:          []CommerceOrderItem{{SKU: sku, Quantity: item.Quantity}},
				Currency:       "BRL",
				IdempotencyKey: fmt.Sprintf("canal:%s:%s", channelID, externalID),
			}, now)
			if err != nil {
				return MarketplaceSyncReport{}, fmt.Errorf("importar pedido %s do canal %s: %w", externalID, channelID, err)
			}
			s.applied[orderKey] = true
			report.Applied++
			report.OperationsUsed++
			report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
				Entity: entity, SKU: sku, Version: item.Version,
				Result: MarketplaceResultApplied, Detail: "pedido local " + pedido.ID,
			})
			continue
		}
		if entity == MarketplaceEntityStock {
			local := int64(0)
			if s.commerce != nil {
				local = s.commerce.Stock(organizationID, sku).Available
			}
			remote, seen := s.remote.stock[channelID+"|"+sku]
			// CONFLITO: o remoto jÃ¡ tem uma quantidade diferente da que queremos publicar, e ela
			// nÃ£o veio deste espelho. Aplicar por cima esconderia divergÃªncia de estoque.
			if seen && remote != local && remote != item.Quantity {
				report.Conflicts++
				report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
					Entity: entity, SKU: sku, Version: item.Version,
					Result: MarketplaceResultConflict,
					Detail: fmt.Sprintf("remoto=%d local=%d desejado=%d: divergÃªncia nÃ£o Ã© sobrescrita", remote, local, item.Quantity),
				})
				continue
			}
			s.remote.stock[channelID+"|"+sku] = item.Quantity
		}
		s.applied[key] = true
		report.Applied++
		report.OperationsUsed++
		report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
			Entity: entity, SKU: sku, Version: item.Version,
			Result: MarketplaceResultApplied,
		})
	}
	if err := s.persistLocked(organizationID); err != nil {
		return MarketplaceSyncReport{}, err
	}
	return report, nil
}

// RemoteStock devolve o espelho remoto do duble, para inspeÃ§Ã£o e verificaÃ§Ã£o em teste.
func (s *MarketplaceSyncStore) RemoteStock(channelID, sku string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.remote.stock[channelID+"|"+sku]
	return value, ok
}
