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

// Estágio 32 — SÉRIE TEMPORAL do painel de negócio.
//
// O BI entregue nas rodadas anteriores era uma **fotografia**: cada chamada calculava o agora, sem
// memória. Sem série não há tendência, e sem tendência um painel não responde a pergunta que importa
// ("está melhorando?"). Este arquivo guarda **amostras datadas** — e quatro decisões deliberadas:
//
//  1. **UMA AMOSTRA POR JANELA**: registrar duas vezes na mesma hora devolve a amostra existente e
//     `Created=false`. Sem isso, chamar o endpoint em laço criaria uma "tendência" feita de repetições
//     do mesmo instante — ruído com aparência de série;
//  2. **RETENÇÃO COM LIMITE DECLARADO**: o histórico tem teto, e o que é descartado é **relatado**
//     (`Discarded`). Um histórico que finge ser infinito é uma promessa que o disco não cumpre;
//  3. **SEM BASE DE COMPARAÇÃO É DECLARADO**: com menos de duas amostras, a tendência traz
//     `SemBase=true` e **nenhum número**. Delta zero significaria "não mudou" quando na verdade
//     significa "não há com o que comparar" — a diferença importa;
//  4. **O INTERVALO É EXPLÍCITO**: a tendência informa de quando até quando comparou, para o número
//     não ser lido como se fosse de outro período.

// BusinessSnapshot é uma amostra datada do painel de uma organização.
type BusinessSnapshot struct {
	ID             string               `json:"id"`
	OrganizationID string               `json:"organization_id"`
	TakenAt        time.Time            `json:"taken_at"`
	Window         string               `json:"window"`
	Report         BusinessStatusReport `json:"report"`
}

// BusinessTrendDelta é a variação de um indicador entre a primeira e a última amostra do período.
type BusinessTrendDelta struct {
	Metric string `json:"metric"`
	From   int64  `json:"from"`
	To     int64  `json:"to"`
	Delta  int64  `json:"delta"`
}

// BusinessTrend é a leitura da série no período pedido.
type BusinessTrend struct {
	OrganizationID string     `json:"organization_id"`
	Snapshots      int        `json:"snapshots"`
	First          *time.Time `json:"first,omitempty"`
	Last           *time.Time `json:"last,omitempty"`
	Interval       string     `json:"intervalo,omitempty"`
	// SemBase é declarado quando não há duas amostras: a ausência de base NÃO é delta zero.
	SemBase bool                 `json:"sem_base"`
	Motivo  string               `json:"motivo,omitempty"`
	Deltas  []BusinessTrendDelta `json:"deltas,omitempty"`
	History []BusinessSnapshot   `json:"history"`
	// Discarded informa quantas amostras a retenção descartou nesta leitura.
	Discarded int `json:"amostras_descartadas"`
	Retention int `json:"retencao_maxima"`
}

// BusinessSnapshotStore persiste as amostras por organização.
type BusinessSnapshotStore struct {
	mu     sync.Mutex
	root   string
	loaded map[string]bool
	// MaxSnapshots é o teto de retenção por organização.
	MaxSnapshots int
}

// DefaultBusinessSnapshotRetention é o teto padrão de amostras por organização.
const DefaultBusinessSnapshotRetention = 96

// SnapshotWindow devolve a janela de deduplicação (uma hora) de um instante.
func SnapshotWindow(now time.Time) string {
	return now.UTC().Format("2006-01-02T15")
}

var ErrBusinessSnapshotScopeRequired = errors.New("business snapshot requires an organization scope")

type businessSnapshotFile struct {
	Snapshots []BusinessSnapshot `json:"snapshots"`
}

// NewBusinessSnapshotStore cria o store de amostras.
func NewBusinessSnapshotStore(root string) *BusinessSnapshotStore {
	return &BusinessSnapshotStore{
		root:         strings.TrimSpace(root),
		loaded:       map[string]bool{},
		MaxSnapshots: DefaultBusinessSnapshotRetention,
	}
}

func (s *BusinessSnapshotStore) snapshotPath(organizationID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

func (s *BusinessSnapshotStore) load(organizationID string) []BusinessSnapshot {
	path := s.snapshotPath(organizationID)
	if path == "" {
		return nil
	}
	var file businessSnapshotFile
	if err := readJSON(path, &file); err != nil {
		return nil
	}
	sort.Slice(file.Snapshots, func(i, j int) bool { return file.Snapshots[i].TakenAt.Before(file.Snapshots[j].TakenAt) })
	return file.Snapshots
}

func (s *BusinessSnapshotStore) persist(organizationID string, snapshots []BusinessSnapshot) error {
	path := s.snapshotPath(organizationID)
	if path == "" {
		return nil
	}
	// A escrita atômica já cria o diretório (correção de raiz de E-127).
	return writeJSONAtomic(path, businessSnapshotFile{Snapshots: snapshots})
}

// Record grava (ou reaproveita) a amostra da janela atual.
//
// Devolve `created=false` quando a janela já tinha amostra: a idempotência por janela é o que impede
// a série de virar repetição do mesmo instante.
func (s *BusinessSnapshotStore) Record(report BusinessStatusReport, now time.Time) (BusinessSnapshot, bool, error) {
	organizationID := strings.TrimSpace(report.OrganizationID)
	if organizationID == "" {
		return BusinessSnapshot{}, false, fmt.Errorf("%w: organização é obrigatória", ErrBusinessSnapshotScopeRequired)
	}
	window := SnapshotWindow(now)

	s.mu.Lock()
	defer s.mu.Unlock()
	snapshots := s.load(organizationID)
	for _, existing := range snapshots {
		if existing.Window == window {
			return existing, false, nil
		}
	}
	snapshot := BusinessSnapshot{
		ID:             "bsn_" + window,
		OrganizationID: organizationID,
		TakenAt:        now.UTC(),
		Window:         window,
		Report:         report,
	}
	snapshots = append(snapshots, snapshot)
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].TakenAt.Before(snapshots[j].TakenAt) })
	// RETENÇÃO: o teto é aplicado aqui e o descarte é informado na leitura da série.
	if limit := s.MaxSnapshots; limit > 0 && len(snapshots) > limit {
		snapshots = snapshots[len(snapshots)-limit:]
	}
	if err := s.persist(organizationID, snapshots); err != nil {
		return BusinessSnapshot{}, false, err
	}
	return snapshot, true, nil
}

// Trend lê a série da organização no período pedido (from inclusivo; zero significa tudo).
func (s *BusinessSnapshotStore) Trend(organizationID string, from time.Time) (BusinessTrend, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return BusinessTrend{}, fmt.Errorf("%w: organização é obrigatória", ErrBusinessSnapshotScopeRequired)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.load(organizationID)
	selected := make([]BusinessSnapshot, 0, len(all))
	for _, snapshot := range all {
		if !from.IsZero() && snapshot.TakenAt.Before(from) {
			continue
		}
		selected = append(selected, snapshot)
	}
	trend := BusinessTrend{
		OrganizationID: organizationID,
		Snapshots:      len(selected),
		History:        selected,
		Retention:      s.MaxSnapshots,
		Discarded:      len(all) - len(selected),
	}
	if len(selected) == 0 {
		trend.SemBase = true
		trend.Motivo = "nenhuma amostra registrada ainda: registre uma amostra para começar a série"
		return trend, nil
	}
	first := selected[0].TakenAt
	last := selected[len(selected)-1].TakenAt
	trend.First = &first
	trend.Last = &last
	trend.Interval = first.Format(time.RFC3339) + " → " + last.Format(time.RFC3339)
	if len(selected) < 2 {
		trend.SemBase = true
		trend.Motivo = "apenas uma amostra no período: sem duas medições não há tendência (delta zero significaria “não mudou”, que é outra coisa)"
		return trend, nil
	}
	oldest := selected[0].Report
	newest := selected[len(selected)-1].Report
	trend.Deltas = []BusinessTrendDelta{
		{Metric: "pedidos", From: int64(oldest.Commerce.Orders), To: int64(newest.Commerce.Orders)},
		{Metric: "receita_cents", From: oldest.Commerce.RevenueCents, To: newest.Commerce.RevenueCents},
		{Metric: "receita_financeira_cents", From: oldest.Finance.IncomeCents, To: newest.Finance.IncomeCents},
		{Metric: "despesa_financeira_cents", From: oldest.Finance.ExpenseCents, To: newest.Finance.ExpenseCents},
		{Metric: "liquido_cents", From: oldest.Finance.NetCents, To: newest.Finance.NetCents},
		{Metric: "campanhas_ativas", From: int64(oldest.Campaigns.Active), To: int64(newest.Campaigns.Active)},
		{Metric: "impressoes", From: oldest.Campaigns.Impressions, To: newest.Campaigns.Impressions},
		{Metric: "propostas", From: int64(oldest.Sales.Proposals), To: int64(newest.Sales.Proposals)},
		{Metric: "follow_ups_vencidos", From: int64(oldest.Sales.FollowUpsOverdue), To: int64(newest.Sales.FollowUpsOverdue)},
		{Metric: "conversas", From: int64(oldest.Inbox.Conversations), To: int64(newest.Inbox.Conversations)},
		{Metric: "violacoes_sla", From: int64(oldest.Inbox.SLAViolations), To: int64(newest.Inbox.SLAViolations)},
	}
	for index := range trend.Deltas {
		trend.Deltas[index].Delta = trend.Deltas[index].To - trend.Deltas[index].From
	}
	return trend, nil
}

// BusinessSnapshot registrar a amostra do agora e devolve a série atualizada.
func (r *Runtime) BusinessSnapshot(organizationID string, now time.Time) (BusinessSnapshot, bool, BusinessTrend, error) {
	if r.businessSnapshots == nil {
		return BusinessSnapshot{}, false, BusinessTrend{}, fmt.Errorf("%w: store de amostras indisponível", ErrBusinessSnapshotScopeRequired)
	}
	report, err := r.BusinessStatus(organizationID, now)
	if err != nil {
		return BusinessSnapshot{}, false, BusinessTrend{}, err
	}
	snapshot, created, err := r.businessSnapshots.Record(report, now)
	if err != nil {
		return BusinessSnapshot{}, false, BusinessTrend{}, err
	}
	trend, err := r.businessSnapshots.Trend(organizationID, time.Time{})
	if err != nil {
		return BusinessSnapshot{}, false, BusinessTrend{}, err
	}
	return snapshot, created, trend, nil
}

// BusinessTrend lê a série sem registrar nada.
func (r *Runtime) BusinessTrend(organizationID string, from time.Time) (BusinessTrend, error) {
	if r.businessSnapshots == nil {
		return BusinessTrend{}, fmt.Errorf("%w: store de amostras indisponível", ErrBusinessSnapshotScopeRequired)
	}
	return r.businessSnapshots.Trend(organizationID, from)
}

// Diretório padrão das amostras, dentro do data root do runtime.
func businessSnapshotRoot(dataRoot string) string {
	return filepath.Join(dataRoot, ".agent-business-snapshots")
}
