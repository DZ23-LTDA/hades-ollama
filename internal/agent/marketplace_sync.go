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

// Estágio 25 — MARKETPLACE: contrato de sincronização com duble de sandbox.
//
// O estágio pedia "conectores oficiais operando em sandbox com evidências" e a homologação
// live declarada bloqueada por fornecedor. O que existe aqui é **exatamente isso, e nada além**:
// um **contrato** de sincronização (produtos, preços, estoque e pedidos) exercitado contra um
// **duble de sandbox** em processo. Nenhuma chamada a Mercado Livre, Amazon, Shopee ou qualquer
// plataforma acontece — e o relatório **declara** isso em cada execução.
//
// Decisões deliberadas:
//
//   - **CATALOGAR NÃO É HABILITAR** (§11.3): o canal expõe estado real. Sem credencial
//     configurada ele é `CONFIG_REQUIRED`; com credencial mas sem teste, `UNAUTHORIZED`; e o
//     `CONNECTED` só vale para o **duble de sandbox**, porque não houve handshake real;
//   - **LIMITE DE OPERAÇÕES POR EXECUÇÃO**: um conector que "sincroniza tudo" sem teto é a forma
//     mais rápida de violar rate limit de plataforma. O teto é explícito e a violação é recusada
//     **antes** de aplicar qualquer efeito;
//   - **IDEMPOTÊNCIA POR VERSÃO**: reexecutar a mesma sincronização **não** reaplica o que já foi
//     aplicado; o item aparece como `ignorado`, não como sucesso novo;
//   - **CONFLITO NÃO É SOBRESCRITA SILENCIOSA**: quando o estoque remoto diverge do local, o item
//     é registrado como **conflito** e **não** é aplicado. Sobrescrever esconderia divergência de
//     estoque — exatamente o erro que causa venda sem produto;
//   - **SEM PUBLICAÇÃO EXTERNA**: o relatório traz `HomologacaoLive` como
//     `BLOCKED_BY_EXTERNAL_DEPENDENCY` com o motivo e o procedimento de desbloqueio.

const (
	// MarketplaceStateConfigRequired indica canal sem credencial configurada.
	MarketplaceStateConfigRequired = "CONFIG_REQUIRED"
	// MarketplaceStateUnauthorized indica credencial presente sem teste aprovado.
	MarketplaceStateUnauthorized = "UNAUTHORIZED"
	// MarketplaceStateConnected indica canal operante — no caso, contra o duble de sandbox.
	MarketplaceStateConnected = "CONNECTED"
	// MarketplaceStateUnsupported indica canal fora do escopo.
	MarketplaceStateUnsupported = "UNSUPPORTED"

	// MarketplaceLiveStatus é o estado declarado da homologação live.
	MarketplaceLiveStatus = "BLOCKED_BY_EXTERNAL_DEPENDENCY"
	// MarketplaceLiveReason explica o desbloqueio da homologação live.
	MarketplaceLiveReason = "sincronizar com marketplace real exige conta autorizada, aplicação registrada e " +
		"scopes válidos da plataforma; sem credencial, a sincronização roda apenas contra o duble de sandbox"

	// MarketplaceMaxOperationsPerRun é o teto de operações por execução.
	MarketplaceMaxOperationsPerRun = 50

	// Entidades suportadas pelo contrato de sincronização.
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

// MarketplaceChannelSpec descreve um canal no catálogo, com estado REAL e capacidades.
type MarketplaceChannelSpec struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// CredentialEnv é o NOME da variável de ambiente que carrega a credencial. O valor nunca é
	// lido nem exibido.
	CredentialEnv string   `json:"credential_env"`
	State         string   `json:"state"`
	Capabilities  []string `json:"capabilities"`
	// LiveTested é false enquanto não houve handshake real: é o que separa o catálogo da operação.
	LiveTested bool `json:"live_tested"`
	// Unblock descreve o que falta para habilitar de verdade.
	Unblock string `json:"unblock_procedure,omitempty"`
}

// MarketplaceChannels devolve o catálogo com o estado declarado por canal.
//
// `configuredEnv` informa quais variáveis de credencial existem no ambiente do operador — apenas
// os NOMES são considerados, nunca os valores.
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
			spec.Unblock = "definir a variável " + spec.CredentialEnv + " e executar um teste de conexão real"
			continue
		}
		// Credencial presente NÃO é operação: sem handshake real o estado é UNAUTHORIZED.
		spec.State = MarketplaceStateUnauthorized
		spec.Unblock = "executar teste de conexão real e registrar a evidência de handshake"
	}
	return specs
}

// MarketplaceSyncItem é um item do espelho local que será enviado ao canal.
type MarketplaceSyncItem struct {
	Entity string `json:"entity"`
	SKU    string `json:"sku"`
	// Version permite idempotência: a mesma versão não é aplicada duas vezes.
	Version int64 `json:"version"`
	// Quantity é usado para estoque; PriceCents para preço.
	Quantity   int64 `json:"quantity,omitempty"`
	PriceCents int64 `json:"price_cents,omitempty"`
}

// MarketplaceSyncRequest é o pedido de sincronização.
type MarketplaceSyncRequest struct {
	OrganizationID string
	ChannelID      string
	Direction      string
	Items          []MarketplaceSyncItem
}

// MarketplaceSyncOutcome é o resultado por item.
type MarketplaceSyncOutcome struct {
	Entity  string `json:"entity"`
	SKU     string `json:"sku"`
	Version int64  `json:"version"`
	Result  string `json:"result"`
	Detail  string `json:"detail,omitempty"`
}

const (
	// Resultados possíveis por item.
	MarketplaceResultApplied  = "aplicado"
	MarketplaceResultSkipped  = "ignorado"
	MarketplaceResultConflict = "conflito"
)

// MarketplaceSyncReport é o relatório da execução.
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

// marketplaceRemote é o DUBLE de sandbox: guarda o espelho remoto em memória.
type marketplaceRemote struct {
	stock map[string]int64
}

// MarketplaceSyncStore executa a sincronização contra o duble e registra a idempotência.
type MarketplaceSyncStore struct {
	mu         sync.Mutex
	remote     *marketplaceRemote
	applied    map[string]bool
	commerce   *CommerceStore
	root       string
	loadedOrgs map[string]bool
}

// NewMarketplaceSyncStore cria o executor de sincronização.
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

// Sync executa a sincronização contra o DUBLE de sandbox.
//
// O teto de operações é verificado ANTES de qualquer efeito: um conector que aplica tudo e só
// depois reclama já teria violado o limite da plataforma.
func (s *MarketplaceSyncStore) Sync(request MarketplaceSyncRequest, now time.Time) (MarketplaceSyncReport, error) {
	organizationID := strings.TrimSpace(request.OrganizationID)
	channelID := strings.TrimSpace(request.ChannelID)
	if organizationID == "" || channelID == "" {
		return MarketplaceSyncReport{}, fmt.Errorf("%w: organização e canal são obrigatórios", ErrMarketplaceSyncInvalid)
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
		return MarketplaceSyncReport{}, fmt.Errorf("%w: direção %q desconhecida", ErrMarketplaceSyncInvalid, request.Direction)
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
		// IDEMPOTÊNCIA: a mesma versão não é aplicada duas vezes.
		if s.applied[key] {
			report.Skipped++
			report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
				Entity: entity, SKU: sku, Version: item.Version,
				Result: MarketplaceResultSkipped, Detail: "versão já aplicada",
			})
			continue
		}
		if entity == MarketplaceEntityStock {
			local := int64(0)
			if s.commerce != nil {
				local = s.commerce.Stock(organizationID, sku).Available
			}
			remote, seen := s.remote.stock[channelID+"|"+sku]
			// CONFLITO: o remoto já tem uma quantidade diferente da que queremos publicar, e ela
			// não veio deste espelho. Aplicar por cima esconderia divergência de estoque.
			if seen && remote != local && remote != item.Quantity {
				report.Conflicts++
				report.Outcomes = append(report.Outcomes, MarketplaceSyncOutcome{
					Entity: entity, SKU: sku, Version: item.Version,
					Result: MarketplaceResultConflict,
					Detail: fmt.Sprintf("remoto=%d local=%d desejado=%d: divergência não é sobrescrita", remote, local, item.Quantity),
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

// RemoteStock devolve o espelho remoto do duble, para inspeção e verificação em teste.
func (s *MarketplaceSyncStore) RemoteStock(channelID, sku string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.remote.stock[channelID+"|"+sku]
	return value, ok
}
