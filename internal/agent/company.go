package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CompanyStatus string

const (
	CompanyActive  CompanyStatus = "active"
	CompanyPaused  CompanyStatus = "paused"
	CompanyArchive CompanyStatus = "archived"
)

type CompanyDepartment struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Mandate          string   `json:"mandate"`
	Autonomy         string   `json:"autonomy"`
	ApprovalRequired []string `json:"approval_required,omitempty"`
}

type CompanyChannel struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	RequiresApproval bool   `json:"requires_approval"`
}

type CompanyRoadmapItem struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description,omitempty"`
	OwnerDepartment string     `json:"owner_department,omitempty"`
	Priority        int        `json:"priority"`
	Status          string     `json:"status"`
	DueAt           *time.Time `json:"due_at,omitempty"`
	Dependencies    []string   `json:"dependencies,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CompanyGoal struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Metric          string    `json:"metric"`
	Target          float64   `json:"target"`
	Current         float64   `json:"current"`
	Unit            string    `json:"unit,omitempty"`
	Period          string    `json:"period"`
	OwnerDepartment string    `json:"owner_department,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompanyBacklogItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	OwnerDepartment string    `json:"owner_department,omitempty"`
	Priority        int       `json:"priority"`
	Status          string    `json:"status"`
	Source          string    `json:"source,omitempty"`
	EstimatedHours  float64   `json:"estimated_hours,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompanyBudget struct {
	Currency                string `json:"currency"`
	MonthlyLimitCents       int64  `json:"monthly_limit_cents"`
	SpentCents              int64  `json:"spent_cents"`
	ApprovalThresholdCents  int64  `json:"approval_threshold_cents"`
	RequireApprovalForAds   bool   `json:"require_approval_for_ads"`
	RequireApprovalForSales bool   `json:"require_approval_for_sales"`
}

type CompanyRisk struct {
	Paused        bool      `json:"paused"`
	PauseReason   string    `json:"pause_reason,omitempty"`
	AnomalyCount  int       `json:"anomaly_count"`
	LastAnomaly   string    `json:"last_anomaly,omitempty"`
	LastAnomalyAt time.Time `json:"last_anomaly_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CompanyApprovalStatus string

const (
	CompanyApprovalPending  CompanyApprovalStatus = "pending"
	CompanyApprovalApproved CompanyApprovalStatus = "approved"
	CompanyApprovalRejected CompanyApprovalStatus = "rejected"
)

type CompanyApproval struct {
	ID             string                `json:"id"`
	CompanyID      string                `json:"company_id"`
	OrganizationID string                `json:"organization_id"`
	ResourceType   string                `json:"resource_type"`
	ResourceID     string                `json:"resource_id"`
	Policy         string                `json:"policy"`
	Category       string                `json:"category,omitempty"`
	AmountCents    int64                 `json:"amount_cents,omitempty"`
	Nonce          string                `json:"nonce"`
	ActorID        string                `json:"actor_id,omitempty"`
	Status         CompanyApprovalStatus `json:"status"`
	Reason         string                `json:"reason,omitempty"`
	ExpiresAt      *time.Time            `json:"expires_at,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type CompanyIdempotencyRecord struct {
	Digest      string    `json:"digest"`
	Operation   string    `json:"operation"`
	Fingerprint string    `json:"fingerprint"`
	ResultID    string    `json:"result_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CompanyCycle struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Objective       string     `json:"objective"`
	Frequency       string     `json:"frequency"`
	IntervalSeconds int64      `json:"interval_seconds"`
	ScheduleID      string     `json:"schedule_id,omitempty"`
	Enabled         bool       `json:"enabled"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	PausedReason    string     `json:"paused_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Company struct {
	ID                string                     `json:"id"`
	Version           int64                      `json:"version"`
	OrganizationID    string                     `json:"organization_id"`
	Name              string                     `json:"name"`
	Mission           string                     `json:"mission,omitempty"`
	Positioning       string                     `json:"positioning,omitempty"`
	BusinessModel     string                     `json:"business_model,omitempty"`
	TargetAudience    string                     `json:"target_audience,omitempty"`
	Offer             string                     `json:"offer,omitempty"`
	Website           string                     `json:"website,omitempty"`
	Currency          string                     `json:"currency"`
	Status            CompanyStatus              `json:"status"`
	Departments       []CompanyDepartment        `json:"departments"`
	Agents            []CompanyAgent             `json:"agents,omitempty"`
	Channels          []CompanyChannel           `json:"channels,omitempty"`
	Roadmap           []CompanyRoadmapItem       `json:"roadmap,omitempty"`
	Goals             []CompanyGoal              `json:"goals,omitempty"`
	Backlog           []CompanyBacklogItem       `json:"backlog,omitempty"`
	Cycles            []CompanyCycle             `json:"cycles,omitempty"`
	Campaigns         []CompanyCampaign          `json:"campaigns,omitempty"`
	AffiliatePrograms []CompanyAffiliateProgram  `json:"affiliate_programs,omitempty"`
	AffiliateLinks    []CompanyAffiliateLink     `json:"affiliate_links,omitempty"`
	Products          []CompanyProduct           `json:"products,omitempty"`
	Orders            []CompanyOrder             `json:"orders,omitempty"`
	SocialAccounts    []CompanySocialAccount     `json:"social_accounts,omitempty"`
	SocialDrafts      []CompanySocialDraft       `json:"social_drafts,omitempty"`
	SocialMetrics     []CompanySocialMetric      `json:"social_metrics,omitempty"`
	TelAgentHistory   []TelAgentExchange         `json:"tel_agent_history,omitempty"`
	TelAgentSessions  []TelAgentSession          `json:"tel_agent_sessions,omitempty"`
	Budget            CompanyBudget              `json:"budget"`
	Risk              CompanyRisk                `json:"risk"`
	Approvals         []CompanyApproval          `json:"approvals,omitempty"`
	Idempotency       []CompanyIdempotencyRecord `json:"idempotency,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

type CompanyReport struct {
	Company              Company `json:"company"`
	OpenBacklog          int     `json:"open_backlog"`
	CompletedBacklog     int     `json:"completed_backlog"`
	GoalsOnTrack         int     `json:"goals_on_track"`
	GoalsAtRisk          int     `json:"goals_at_risk"`
	EnabledCycles        int     `json:"enabled_cycles"`
	BudgetUtilizationPct float64 `json:"budget_utilization_pct"`
}

type CompanyUpdate struct {
	Name           string `json:"name"`
	Mission        string `json:"mission"`
	Positioning    string `json:"positioning"`
	BusinessModel  string `json:"business_model"`
	TargetAudience string `json:"target_audience"`
	Offer          string `json:"offer"`
	Website        string `json:"website"`
	Currency       string `json:"currency"`
}

type CompanyCreateBudgetRequest struct {
	Currency                string `json:"currency"`
	MonthlyLimitCents       int64  `json:"monthly_limit_cents"`
	ApprovalThresholdCents  int64  `json:"approval_threshold_cents"`
	RequireApprovalForAds   bool   `json:"require_approval_for_ads"`
	RequireApprovalForSales bool   `json:"require_approval_for_sales"`
}

type CompanyCreateRequest struct {
	Name           string                     `json:"name"`
	Mission        string                     `json:"mission"`
	Positioning    string                     `json:"positioning"`
	BusinessModel  string                     `json:"business_model"`
	TargetAudience string                     `json:"target_audience"`
	Offer          string                     `json:"offer"`
	Website        string                     `json:"website"`
	Currency       string                     `json:"currency"`
	Budget         CompanyCreateBudgetRequest `json:"budget"`
}

var (
	ErrCompanyNotFound                   = errors.New("company not found")
	ErrCompanyBudgetExceeded             = errors.New("company budget limit exceeded; company paused")
	ErrCompanyApprovalRequired           = errors.New("approval is required for this company action")
	ErrCompanyPaused                     = errors.New("company is paused")
	ErrCompanyApprovalConflict           = errors.New("company approval version conflict")
	ErrCompanyApprovalNonce              = errors.New("company approval nonce mismatch")
	ErrCompanyApprovalNotFound           = errors.New("company approval not found or already decided")
	ErrCompanySpendApprovalPending       = errors.New("company spend approval is pending")
	ErrCompanyIdempotentReplay           = errors.New("company idempotent replay")
	ErrCompanyIdempotencyConflict        = errors.New("company idempotency key was reused with different input")
	ErrCompanyCycleIdempotencyKeyTooLong = errors.New("company cycle idempotency key exceeds 128 bytes")
	ErrCompanyCycleReplayMissing         = errors.New("company cycle idempotent replay has no persisted cycle")
)

type CompanyStore struct {
	mu        sync.RWMutex
	root      string
	companies map[string]Company
}

func NewCompanyStore(root string) (*CompanyStore, error) {
	store := &CompanyStore{root: strings.TrimSpace(root), companies: map[string]Company{}}
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
		var company Company
		if err := readJSON(filepath.Join(store.root, entry.Name()), &company); err != nil {
			return nil, err
		}
		if company.ID != "" {
			normalizeCompanyGrowthModes(&company)
			store.companies[company.ID] = company
		}
	}
	return store, nil
}

func defaultCompanyDepartments() []CompanyDepartment {
	departments := []struct{ id, name, mandate string }{
		{"ceo", "CEO / Estratégia", "Priorizar a empresa, decidir trade-offs e manter a visão."},
		{"product", "Produto", "Descobrir necessidades, roadmap, experiência e proposta de valor."},
		{"engineering", "Engenharia", "Construir, testar, publicar e operar produtos digitais."},
		{"marketing", "Marketing", "Pesquisa, posicionamento, conteúdo, campanhas e aquisição."},
		{"sales", "Vendas", "Qualificar leads, conduzir pipeline e fechar oportunidades com aprovação."},
		{"support", "Suporte", "Responder clientes, manter base de conhecimento e identificar riscos."}, //nolint:misspell // Portuguese product copy.
		{"operations", "Operações", "Métricas, orçamento, processos, fornecedores e continuidade."},     //nolint:misspell // Portuguese product copy.
	}
	result := make([]CompanyDepartment, 0, len(departments))
	for _, item := range departments {
		result = append(result, CompanyDepartment{ID: item.id, Name: item.name, Mandate: item.mandate, Autonomy: "assistido", ApprovalRequired: []string{"gasto", "anuncio", "contrato", "mensagem_externa"}})
	}
	return result
}

func (s *CompanyStore) persistLocked(company Company) error {
	if s.root == "" {
		return nil
	}
	return writeJSONAtomic(filepath.Join(s.root, company.ID+".json"), company)
}

func cloneCompany(company Company) Company {
	data, err := json.Marshal(company)
	if err != nil {
		return company
	}
	var clone Company
	if err := json.Unmarshal(data, &clone); err != nil {
		return company
	}
	return clone
}

func (s *CompanyStore) Create(company Company) (Company, error) {
	company.ID = ""
	company.Status = ""
	company.Departments = nil
	company.Agents = nil
	company.Channels = nil
	company.Roadmap = nil
	company.Goals = nil
	company.Backlog = nil
	company.Cycles = nil
	company.Campaigns = nil
	company.AffiliatePrograms = nil
	company.AffiliateLinks = nil
	company.Products = nil
	company.Orders = nil
	company.SocialAccounts = nil
	company.SocialDrafts = nil
	company.SocialMetrics = nil
	company.TelAgentHistory = nil
	company.Approvals = nil
	company.Idempotency = nil
	company.Version = 1
	company.Budget.SpentCents = 0
	company.Risk = CompanyRisk{}
	company.CreatedAt = time.Time{}
	company.UpdatedAt = time.Time{}
	company.Name = strings.TrimSpace(company.Name)
	if company.Name == "" {
		return Company{}, errors.New("company name is required")
	}
	if len(company.Name) > 200 {
		return Company{}, errors.New("company name is too long")
	}
	if company.OrganizationID == "" {
		return Company{}, errors.New("company organization is required")
	}
	if company.Budget.MonthlyLimitCents < 0 || company.Budget.ApprovalThresholdCents < 0 {
		return Company{}, errors.New("company budget values cannot be negative")
	}
	if company.ID == "" {
		company.ID = "co_" + uuid.NewString()
	}
	if company.Currency == "" {
		company.Currency = "USD"
	}
	if company.Budget.Currency == "" {
		company.Budget.Currency = company.Currency
	}
	if company.Status == "" {
		company.Status = CompanyActive
	}
	if len(company.Departments) == 0 {
		company.Departments = defaultCompanyDepartments()
	}
	if len(company.Agents) == 0 {
		company.Agents = defaultCompanyAgents()
	}
	now := time.Now().UTC()
	company.CreatedAt = now
	company.UpdatedAt = now
	company.Risk.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.companies[company.ID]; exists {
		return Company{}, errors.New("company id already exists")
	}
	s.companies[company.ID] = company
	if err := s.persistLocked(company); err != nil {
		delete(s.companies, company.ID)
		return Company{}, err
	}
	return company, nil
}

func (s *CompanyStore) CreateRequest(request CompanyCreateRequest, organizationID string) (Company, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return Company{}, errors.New("company organization is required")
	}
	return s.Create(Company{
		OrganizationID: organizationID,
		Name:           request.Name,
		Mission:        request.Mission,
		Positioning:    request.Positioning,
		BusinessModel:  request.BusinessModel,
		TargetAudience: request.TargetAudience,
		Offer:          request.Offer,
		Website:        request.Website,
		Currency:       request.Currency,
		Budget: CompanyBudget{
			Currency:                request.Budget.Currency,
			MonthlyLimitCents:       request.Budget.MonthlyLimitCents,
			ApprovalThresholdCents:  request.Budget.ApprovalThresholdCents,
			RequireApprovalForAds:   request.Budget.RequireApprovalForAds,
			RequireApprovalForSales: request.Budget.RequireApprovalForSales,
		},
	})
}

func queueCompanyApproval(company *Company, resourceType, resourceID, policy string, now time.Time) {
	expiresAt := now.Add(30 * time.Minute)
	company.Approvals = append(company.Approvals, CompanyApproval{
		ID: "capr_" + uuid.NewString(), CompanyID: company.ID, OrganizationID: company.OrganizationID,
		ResourceType: resourceType, ResourceID: resourceID, Policy: policy, Nonce: uuid.NewString(),
		Status: CompanyApprovalPending, ExpiresAt: &expiresAt, CreatedAt: now, UpdatedAt: now,
	})
}

func companyIdempotencyDigest(operation, key string) string {
	sum := sha256.Sum256([]byte(operation + "\x00" + key))
	return hex.EncodeToString(sum[:])
}

func checkCompanyIdempotency(company *Company, operation, key, fingerprint string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	digest := companyIdempotencyDigest(operation, key)
	for _, record := range company.Idempotency {
		if record.Operation != operation || record.Digest != digest {
			continue
		}
		if record.Fingerprint != fingerprint {
			return ErrCompanyIdempotencyConflict
		}
		return ErrCompanyIdempotentReplay
	}
	return nil
}

func rememberCompanyIdempotency(company *Company, operation, key, fingerprint string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	company.Idempotency = append(company.Idempotency, CompanyIdempotencyRecord{Digest: companyIdempotencyDigest(operation, key), Operation: operation, Fingerprint: fingerprint, CreatedAt: time.Now().UTC()})
	if len(company.Idempotency) > 1024 {
		company.Idempotency = append([]CompanyIdempotencyRecord(nil), company.Idempotency[len(company.Idempotency)-1024:]...)
	}
}

func (s *CompanyStore) RecordSpendRequest(id, category string, amountCents int64) (Company, error) {
	return s.RecordSpendRequestWithIdempotency(id, category, amountCents, "")
}

func (s *CompanyStore) RecordSpendRequestWithIdempotency(id, category string, amountCents int64, idempotencyKey string) (Company, error) {
	if amountCents <= 0 {
		return Company{}, errors.New("spend amount must be positive")
	}
	category = strings.TrimSpace(category)
	fingerprint := category + ":" + strconv.FormatInt(amountCents, 10)
	return s.mutate(id, func(company *Company) error {
		if err := checkCompanyIdempotency(company, "company.spend.request", idempotencyKey, fingerprint); err != nil {
			return err
		}
		if company.Status == CompanyPaused || company.Risk.Paused {
			return ErrCompanyPaused
		}
		requiresApproval := (company.Budget.ApprovalThresholdCents > 0 && amountCents >= company.Budget.ApprovalThresholdCents) || category == "ads" || category == "contract"
		if !requiresApproval {
			if company.Budget.MonthlyLimitCents > 0 && company.Budget.SpentCents+amountCents > company.Budget.MonthlyLimitCents {
				company.Status = CompanyPaused
				company.Risk.Paused = true
				company.Risk.PauseReason = "limite de orçamento excedido"
				company.Risk.UpdatedAt = time.Now().UTC()
				return ErrCompanyBudgetExceeded
			}
			company.Budget.SpentCents += amountCents
			rememberCompanyIdempotency(company, "company.spend.request", idempotencyKey, fingerprint)
			return nil
		}
		for _, approval := range company.Approvals {
			if approval.ResourceType == "spend" && approval.Status == CompanyApprovalPending && approval.Category == category && approval.AmountCents == amountCents {
				return ErrCompanySpendApprovalPending
			}
		}
		resourceID := "spend_" + uuid.NewString()
		now := time.Now().UTC()
		queueCompanyApproval(company, "spend", resourceID, "spend:"+category, now)
		approval := &company.Approvals[len(company.Approvals)-1]
		approval.Category = category
		approval.AmountCents = amountCents
		rememberCompanyIdempotency(company, "company.spend.request", idempotencyKey, fingerprint)
		return ErrCompanySpendApprovalPending
	})
}

func (s *CompanyStore) PendingApproval(id, resourceType, resourceID string) (CompanyApproval, error) {
	company, err := s.Get(id)
	if err != nil {
		return CompanyApproval{}, err
	}
	for _, approval := range company.Approvals {
		if approval.ResourceType == resourceType && approval.ResourceID == resourceID && approval.Status == CompanyApprovalPending {
			return approval, nil
		}
	}
	return CompanyApproval{}, ErrCompanyApprovalNotFound
}

func markCompanyApproval(company *Company, resourceType, resourceID string, approved bool, actorID, reason string) bool {
	for index := range company.Approvals {
		approval := &company.Approvals[index]
		if approval.ResourceType != resourceType || approval.ResourceID != resourceID || approval.Status != CompanyApprovalPending {
			continue
		}
		approval.ActorID = strings.TrimSpace(actorID)
		approval.Reason = strings.TrimSpace(reason)
		if approved {
			approval.Status = CompanyApprovalApproved
		} else {
			approval.Status = CompanyApprovalRejected
		}
		approval.UpdatedAt = time.Now().UTC()
		return true
	}
	return false
}

func applyCompanyApproval(company *Company, resourceType, resourceID string, approved bool) error {
	if !approved {
		return nil
	}
	switch resourceType {
	case "campaign":
		for index := range company.Campaigns {
			if company.Campaigns[index].ID == resourceID {
				company.Campaigns[index].Approved = true
				company.Campaigns[index].Status = "approved"
				company.Campaigns[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
	case "affiliate_program":
		for index := range company.AffiliatePrograms {
			if company.AffiliatePrograms[index].ID == resourceID {
				company.AffiliatePrograms[index].Approved = true
				company.AffiliatePrograms[index].Status = "active"
				company.AffiliatePrograms[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
	case "order":
		for index := range company.Orders {
			if company.Orders[index].ID == resourceID {
				company.Orders[index].Approved = true
				company.Orders[index].Status = "approved"
				company.Orders[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
	case "social_draft":
		for index := range company.SocialDrafts {
			if company.SocialDrafts[index].ID == resourceID {
				company.SocialDrafts[index].Approved = true
				company.SocialDrafts[index].Status = "approved"
				company.SocialDrafts[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
	case "spend":
		for _, approval := range company.Approvals {
			if approval.ResourceType != resourceType || approval.ResourceID != resourceID {
				continue
			}
			if approval.AmountCents <= 0 {
				return ErrCompanyApprovalRequired
			}
			if company.Status == CompanyPaused || company.Risk.Paused {
				return ErrCompanyPaused
			}
			if company.Budget.MonthlyLimitCents > 0 && company.Budget.SpentCents+approval.AmountCents > company.Budget.MonthlyLimitCents {
				company.Status = CompanyPaused
				company.Risk.Paused = true
				company.Risk.PauseReason = "limite de orçamento excedido"
				company.Risk.UpdatedAt = time.Now().UTC()
				return ErrCompanyBudgetExceeded
			}
			company.Budget.SpentCents += approval.AmountCents
			return nil
		}
	}
	return ErrCompanyApprovalNotFound
}

func (s *CompanyStore) DecideApproval(id, approvalID string, approved bool, reason, actorID, organizationID string, expectedVersion int64, nonce string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		if expectedVersion > 0 && company.Version != expectedVersion {
			return ErrCompanyApprovalConflict
		}
		if strings.TrimSpace(actorID) == "" || strings.TrimSpace(reason) == "" {
			return ErrCompanyApprovalRequired
		}
		if strings.TrimSpace(organizationID) == "" || company.OrganizationID != strings.TrimSpace(organizationID) {
			return ErrCompanyApprovalRequired
		}
		for index := range company.Approvals {
			approval := &company.Approvals[index]
			if approval.ID != strings.TrimSpace(approvalID) || approval.Status != CompanyApprovalPending {
				continue
			}
			if approval.ExpiresAt != nil && time.Now().UTC().After(*approval.ExpiresAt) {
				return ErrCompanyApprovalRequired
			}
			if strings.TrimSpace(nonce) == "" || strings.TrimSpace(nonce) != approval.Nonce {
				return ErrCompanyApprovalNonce
			}
			if err := applyCompanyApproval(company, approval.ResourceType, approval.ResourceID, approved); err != nil {
				return err
			}
			if !markCompanyApproval(company, approval.ResourceType, approval.ResourceID, approved, actorID, reason) {
				return ErrCompanyApprovalNotFound
			}
			return nil
		}
		return ErrCompanyApprovalNotFound
	})
}

func (s *CompanyStore) List(organizationID string) []Company {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Company, 0, len(s.companies))
	for _, company := range s.companies {
		if organizationID != "" && company.OrganizationID != organizationID {
			continue
		}
		result = append(result, company)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}

func (s *CompanyStore) Get(id string) (Company, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	company, ok := s.companies[strings.TrimSpace(id)]
	if !ok {
		return Company{}, ErrCompanyNotFound
	}
	return company, nil
}

func (s *CompanyStore) Update(id string, update CompanyUpdate) (Company, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	company, ok := s.companies[id]
	if !ok {
		return Company{}, ErrCompanyNotFound
	}
	previous := cloneCompany(company)
	if value := strings.TrimSpace(update.Name); value != "" {
		company.Name = value
	}
	if update.Mission != "" {
		company.Mission = strings.TrimSpace(update.Mission)
	}
	if update.Positioning != "" {
		company.Positioning = strings.TrimSpace(update.Positioning)
	}
	if update.BusinessModel != "" {
		company.BusinessModel = strings.TrimSpace(update.BusinessModel)
	}
	if update.TargetAudience != "" {
		company.TargetAudience = strings.TrimSpace(update.TargetAudience)
	}
	if update.Offer != "" {
		company.Offer = strings.TrimSpace(update.Offer)
	}
	if update.Website != "" {
		company.Website = strings.TrimSpace(update.Website)
	}
	if update.Currency != "" {
		company.Currency = strings.TrimSpace(update.Currency)
		company.Budget.Currency = company.Currency
	}
	if company.Version <= 0 {
		company.Version = 1
	}
	company.Version++
	company.UpdatedAt = time.Now().UTC()
	s.companies[id] = company
	if err := s.persistLocked(company); err != nil {
		s.companies[id] = previous
		return Company{}, err
	}
	return company, nil
}

func (s *CompanyStore) AddRoadmap(id string, item CompanyRoadmapItem) (Company, error) {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		return Company{}, errors.New("roadmap title is required")
	}
	if item.Priority <= 0 {
		item.Priority = 50
	}
	if item.Status == "" {
		item.Status = "planned"
	}
	if item.ID == "" {
		item.ID = "road_" + uuid.NewString()
	}
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return s.mutate(id, func(company *Company) error { company.Roadmap = append(company.Roadmap, item); return nil })
}

func (s *CompanyStore) AddGoal(id string, goal CompanyGoal) (Company, error) {
	goal.Title = strings.TrimSpace(goal.Title)
	goal.Metric = strings.TrimSpace(goal.Metric)
	if goal.Title == "" || goal.Metric == "" {
		return Company{}, errors.New("goal title and metric are required")
	}
	if goal.Status == "" {
		goal.Status = "on_track"
	}
	if goal.Period == "" {
		goal.Period = "monthly"
	}
	if goal.ID == "" {
		goal.ID = "goal_" + uuid.NewString()
	}
	now := time.Now().UTC()
	goal.CreatedAt = now
	goal.UpdatedAt = now
	return s.mutate(id, func(company *Company) error { company.Goals = append(company.Goals, goal); return nil })
}

func (s *CompanyStore) AddBacklog(id string, item CompanyBacklogItem) (Company, error) {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		return Company{}, errors.New("backlog title is required")
	}
	if item.Priority <= 0 {
		item.Priority = 50
	}
	if item.Status == "" {
		item.Status = "planned"
	}
	if item.ID == "" {
		item.ID = "task_" + uuid.NewString()
	}
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	return s.mutate(id, func(company *Company) error {
		company.Backlog = append(company.Backlog, item)
		sort.SliceStable(company.Backlog, func(i, j int) bool { return company.Backlog[i].Priority < company.Backlog[j].Priority })
		return nil
	})
}

func (s *CompanyStore) AddCycle(id string, cycle CompanyCycle) (Company, error) {
	company, _, _, err := s.AddCycleWithIdempotency(id, cycle, "")
	return company, err
}

func (s *CompanyStore) AddCycleWithIdempotency(id string, cycle CompanyCycle, idempotencyKey string) (Company, CompanyCycle, bool, error) {
	cycle.Name = strings.TrimSpace(cycle.Name)
	cycle.Objective = strings.TrimSpace(cycle.Objective)
	if cycle.Name == "" || cycle.Objective == "" {
		return Company{}, CompanyCycle{}, false, errors.New("cycle name and objective are required")
	}
	if cycle.IntervalSeconds < 1 || cycle.IntervalSeconds > 31*24*60*60 {
		return Company{}, CompanyCycle{}, false, errors.New("cycle interval must be between 1 second and 31 days")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if len([]byte(idempotencyKey)) > 128 {
		return Company{}, CompanyCycle{}, false, ErrCompanyCycleIdempotencyKeyTooLong
	}
	if cycle.ID == "" {
		cycle.ID = "cyc_" + uuid.NewString()
	}
	cycle.Enabled = true
	now := time.Now().UTC()
	cycle.CreatedAt = now
	cycle.UpdatedAt = now
	if cycle.NextRunAt.IsZero() {
		cycle.NextRunAt = now.Add(time.Duration(cycle.IntervalSeconds) * time.Second)
	}
	fingerprintInput := cycle.Name + "\x00" + cycle.Objective + "\x00" + cycle.Frequency + "\x00" + strconv.FormatInt(cycle.IntervalSeconds, 10)
	fingerprint := companyIdempotencyDigest("company.cycle.input", fingerprintInput)
	replayed := false
	var stored CompanyCycle
	updated, err := s.mutate(id, func(company *Company) error {
		if err := checkCompanyIdempotency(company, "company.cycle", idempotencyKey, fingerprint); err != nil {
			if !errors.Is(err, ErrCompanyIdempotentReplay) {
				return err
			}
			var resultID string
			for index := len(company.Idempotency) - 1; index >= 0; index-- {
				record := company.Idempotency[index]
				if record.Operation == "company.cycle" && record.Digest == companyIdempotencyDigest("company.cycle", idempotencyKey) {
					resultID = record.ResultID
					break
				}
			}
			for _, existing := range company.Cycles {
				if existing.ID == resultID {
					stored = existing
					replayed = true
					return nil
				}
			}
			return ErrCompanyCycleReplayMissing
		}
		company.Cycles = append(company.Cycles, cycle)
		if idempotencyKey != "" {
			rememberCompanyIdempotency(company, "company.cycle", idempotencyKey, fingerprint)
			for index := len(company.Idempotency) - 1; index >= 0; index-- {
				if company.Idempotency[index].Operation == "company.cycle" && company.Idempotency[index].Digest == companyIdempotencyDigest("company.cycle", idempotencyKey) {
					company.Idempotency[index].ResultID = cycle.ID
					break
				}
			}
		}
		stored = cycle
		return nil
	})
	return updated, stored, replayed, err
}

func (s *CompanyStore) RemoveCycle(id, cycleID string) (Company, error) {
	cycleID = strings.TrimSpace(cycleID)
	if cycleID == "" {
		return Company{}, errors.New("cycle id is required")
	}
	return s.mutate(id, func(company *Company) error {
		found := false
		cycles := make([]CompanyCycle, 0, len(company.Cycles))
		for _, cycle := range company.Cycles {
			if cycle.ID == cycleID {
				found = true
				continue
			}
			cycles = append(cycles, cycle)
		}
		if !found {
			return errors.New("company cycle not found")
		}
		company.Cycles = cycles
		filtered := company.Idempotency[:0]
		for _, record := range company.Idempotency {
			if record.Operation == "company.cycle" && record.ResultID == cycleID {
				continue
			}
			filtered = append(filtered, record)
		}
		company.Idempotency = filtered
		return nil
	})
}

func (s *CompanyStore) SetCycleSchedule(id, cycleID, scheduleID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Cycles {
			if company.Cycles[index].ID == cycleID {
				company.Cycles[index].ScheduleID = scheduleID
				company.Cycles[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return errors.New("company cycle not found")
	})
}

func (s *CompanyStore) AddApproval(id string, approval CompanyApproval) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		if approval.ID == "" {
			approval.ID = "appr_" + uuid.NewString()[:8]
		}
		if approval.CreatedAt.IsZero() {
			approval.CreatedAt = time.Now().UTC()
		}
		approval.UpdatedAt = approval.CreatedAt
		if approval.Status == "" {
			approval.Status = CompanyApprovalPending
		}
		approval.CompanyID = company.ID
		approval.OrganizationID = company.OrganizationID
		company.Approvals = append(company.Approvals, approval)
		return nil
	})
}

func (s *CompanyStore) UpdateCycleRun(id, cycleID string, lastRunAt time.Time, nextRunAt time.Time) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Cycles {
			if company.Cycles[index].ID == cycleID {
				company.Cycles[index].LastRunAt = &lastRunAt
				company.Cycles[index].NextRunAt = nextRunAt
				company.Cycles[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return errors.New("company cycle not found")
	})
}

func (s *CompanyStore) Pause(id, reason string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		company.Status = CompanyPaused
		company.Risk.Paused = true
		company.Risk.PauseReason = strings.TrimSpace(reason)
		company.Risk.UpdatedAt = time.Now().UTC()
		return nil
	})
}

func (s *CompanyStore) Resume(id string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		company.Status = CompanyActive
		company.Risk.Paused = false
		company.Risk.PauseReason = ""
		company.Risk.UpdatedAt = time.Now().UTC()
		return nil
	})
}

func (s *CompanyStore) RecordAnomaly(id, severity, reason string) (Company, error) {
	severity = strings.ToLower(strings.TrimSpace(severity))
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Company{}, errors.New("anomaly reason is required")
	}
	return s.mutate(id, func(company *Company) error {
		company.Risk.AnomalyCount++
		company.Risk.LastAnomaly = severity + ": " + reason
		company.Risk.LastAnomalyAt = time.Now().UTC()
		if severity == "critical" || severity == "high" || company.Risk.AnomalyCount >= 3 {
			company.Status = CompanyPaused
			company.Risk.Paused = true
			company.Risk.PauseReason = "anomalia detectada: " + reason
		}
		company.Risk.UpdatedAt = time.Now().UTC()
		return nil
	})
}

func (s *CompanyStore) RecordSpend(id, category string, amountCents int64, approved bool) (Company, error) {
	if amountCents <= 0 {
		return Company{}, errors.New("spend amount must be positive")
	}
	category = strings.TrimSpace(category)
	return s.mutate(id, func(company *Company) error {
		if company.Status == CompanyPaused || company.Risk.Paused {
			return ErrCompanyPaused
		}
		if (company.Budget.ApprovalThresholdCents > 0 && amountCents >= company.Budget.ApprovalThresholdCents) || category == "ads" || category == "contract" {
			if !approved {
				return ErrCompanyApprovalRequired
			}
		}
		if company.Budget.MonthlyLimitCents > 0 && company.Budget.SpentCents+amountCents > company.Budget.MonthlyLimitCents {
			company.Status = CompanyPaused
			company.Risk.Paused = true
			company.Risk.PauseReason = "limite de orçamento excedido"
			company.Risk.UpdatedAt = time.Now().UTC()
			return ErrCompanyBudgetExceeded
		}
		company.Budget.SpentCents += amountCents
		return nil
	})
}

func (s *CompanyStore) Report(id string) (CompanyReport, error) {
	company, err := s.Get(id)
	if err != nil {
		return CompanyReport{}, err
	}
	return companyReportSnapshot(company), nil
}

func (s *CompanyStore) mutate(id string, fn func(*Company) error) (Company, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	company, ok := s.companies[strings.TrimSpace(id)]
	if !ok {
		return Company{}, ErrCompanyNotFound
	}
	previous := cloneCompany(company)
	if err := fn(&company); err != nil {
		if errors.Is(err, ErrCompanyBudgetExceeded) || errors.Is(err, ErrCompanySpendApprovalPending) {
			if company.Version <= 0 {
				company.Version = 1
			}
			company.Version++
			company.UpdatedAt = time.Now().UTC()
			s.companies[company.ID] = company
			if persistErr := s.persistLocked(company); persistErr != nil {
				s.companies[company.ID] = previous
				return previous, errors.Join(err, persistErr)
			}
			return company, err
		}
		return previous, err
	}
	if company.Version <= 0 {
		company.Version = 1
	}
	company.Version++
	company.UpdatedAt = time.Now().UTC()
	s.companies[company.ID] = company
	if err := s.persistLocked(company); err != nil {
		s.companies[company.ID] = previous
		return Company{}, err
	}
	return company, nil
}

func (c Company) MarshalJSON() ([]byte, error) {
	type alias Company
	return json.Marshal(alias(c))
}
