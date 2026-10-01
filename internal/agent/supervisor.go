package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSupervisorStopped   = errors.New("supervisor stopped")
	ErrSupervisorDisabled  = errors.New("supervisor disabled by configuration")
	ErrSupervisorRiskPause = errors.New("company risk paused: autonomous cycle halted")
)

type SupervisorConfig struct {
	Enabled               bool          `json:"enabled"`
	TickInterval          time.Duration `json:"tick_interval"`
	WorkerID              string        `json:"worker_id"`
	LeaseDuration         time.Duration `json:"lease_duration"`
	AutoDelegateSwarm     bool          `json:"auto_delegate_swarm"`
	RequireApprovalRisk   bool          `json:"require_approval_risk"`
	MaxConcurrentMissions int           `json:"max_concurrent_missions"`
}

func DefaultSupervisorConfig() SupervisorConfig {
	return SupervisorConfig{
		Enabled:             true,
		TickInterval:        5 * time.Second,
		WorkerID:            "supervisor-" + uuid.NewString()[:8],
		LeaseDuration:       30 * time.Second,
		AutoDelegateSwarm:   true,
		RequireApprovalRisk: true,
	}
}

type SupervisorStatus struct {
	Enabled                bool       `json:"enabled"`
	Running                bool       `json:"running"`
	WorkerID               string     `json:"worker_id"`
	LastTickAt             *time.Time `json:"last_tick_at,omitempty"`
	TickCount              int64      `json:"tick_count"`
	PendingMissionsResumed int        `json:"pending_missions_resumed"`
	SchedulesTriggered     int        `json:"schedules_triggered"`
	CompanyCyclesAdvanced  int        `json:"company_cycles_advanced"`
	RiskPausedCompanies    []string   `json:"risk_paused_companies"`
	BlockedExternalActions int        `json:"blocked_external_actions"`
	PendingApprovalsCount  int        `json:"pending_approvals_count"`
	LastError              string     `json:"last_error,omitempty"`
}

type SupervisorTickResult struct {
	TickNumber             int64     `json:"tick_number"`
	ExecutedAt             time.Time `json:"executed_at"`
	MissionsResumed        int       `json:"missions_resumed"`
	SchedulesTriggered     int       `json:"schedules_triggered"`
	CyclesAdvanced         int       `json:"cycles_advanced"`
	RiskPausedCompanies    []string  `json:"risk_paused_companies,omitempty"`
	BlockedExternalActions int       `json:"blocked_external_actions"`
	ApprovalsCreated       int       `json:"approvals_created"`
}

// Supervisor implements the always-on autonomous loop for the Agent and Company OS.
type Supervisor struct {
	runtime *Runtime
	config  SupervisorConfig
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	status  SupervisorStatus
}

func NewSupervisor(runtime *Runtime, config SupervisorConfig) *Supervisor {
	if config.TickInterval <= 0 {
		config.TickInterval = 5 * time.Second
	}
	if config.WorkerID == "" {
		config.WorkerID = "supervisor-" + uuid.NewString()[:8]
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = 30 * time.Second
	}
	return &Supervisor{
		runtime: runtime,
		config:  config,
		status: SupervisorStatus{
			Enabled:  config.Enabled,
			WorkerID: config.WorkerID,
		},
	}
}

func (s *Supervisor) Config() SupervisorConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *Supervisor) SetConfig(config SupervisorConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
	s.status.Enabled = config.Enabled
	s.status.WorkerID = config.WorkerID
}

func (s *Supervisor) Status() SupervisorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.status
	st.Running = s.running
	st.Enabled = s.config.Enabled
	return st
}

// Start launches the continuous autonomous supervisor daemon loop in the background.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.runLoop(ctx)
	return nil
}

// Stop gracefully terminates the supervisor loop.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	close(s.stopCh)
	done := s.doneCh
	s.mu.Unlock()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	return nil
}

func (s *Supervisor) runLoop(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.running = false
		if s.doneCh != nil {
			close(s.doneCh)
		}
		s.mu.Unlock()
	}()

	interval := s.config.TickInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial immediate tick
	_, _ = s.Tick(ctx, time.Now().UTC())

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			_, _ = s.Tick(ctx, t.UTC())
		}
	}
}

// Tick executes a single discrete supervisor cycle over schedules, missions, and Company OS.
func (s *Supervisor) Tick(ctx context.Context, now time.Time) (SupervisorTickResult, error) {
	s.mu.Lock()
	if !s.config.Enabled {
		s.mu.Unlock()
		return SupervisorTickResult{ExecutedAt: now}, nil
	}
	s.status.TickCount++
	tickNum := s.status.TickCount
	s.mu.Unlock()

	result := SupervisorTickResult{
		TickNumber: tickNum,
		ExecutedAt: now,
	}

	// 1. Process due schedules (Schedules vencidos)
	if s.runtime != nil && s.runtime.context != nil {
		orgScope := strings.TrimSpace(s.runtime.organizationScope)
		if orgScope == "" {
			orgScope = LocalOrganizationID
		}
		claimed := s.runtime.context.ClaimDueSchedulesForOrganization(orgScope, now)
		for _, sched := range claimed {
			// If associated with a company workspace, verify company risk status
			if companyID := companyIDFromWorkspace(sched.Workspace); companyID != "" && s.runtime.company != nil {
				company, err := s.runtime.company.Get(companyID)
				if err == nil && (company.Status == CompanyPaused || company.Risk.Paused) {
					result.RiskPausedCompanies = append(result.RiskPausedCompanies, companyID)
					continue // Risk pause halts autonomous schedule execution
				}
			}

			// Create mission from due schedule
			_, err := s.runtime.CreateMission(ctx, CreateMissionRequest{
				Objective:      sched.Objective,
				Model:          sched.Model,
				Workspace:      sched.Workspace,
				ProjectID:      sched.ProjectID,
				OrganizationID: sched.OrganizationID,
				AutoRun:        true,
			})
			if err == nil {
				result.SchedulesTriggered++
			}
		}
	}

	// 2. Resumable & Long-running: Recover and resume pending missions and leases
	if s.runtime != nil {
		if err := s.runtime.resumePending(ctx); err == nil {
			result.MissionsResumed++
		} else if retryErr := s.runtime.resumePending(ctx); retryErr == nil {
			// Recovery is idempotent. Retry once when a temporary filesystem
			// race interrupts the first scan, while persistent errors remain
			// observable through the existing recovery path.
			result.MissionsResumed++
		}
	}

	// 3. Advance Company OS cycles: roadmap -> tasks -> swarm delegation -> KPIs
	if s.runtime != nil && s.runtime.company != nil {
		orgScope := strings.TrimSpace(s.runtime.organizationScope)
		if orgScope == "" {
			orgScope = LocalOrganizationID
		}
		companies := s.runtime.company.List(orgScope)
		for _, company := range companies {
			// Check company risk / pause state (Pausa por risco para o loop)
			if company.Status == CompanyPaused || company.Risk.Paused {
				result.RiskPausedCompanies = append(result.RiskPausedCompanies, company.ID)
				continue
			}

			for _, cycle := range company.Cycles {
				if !cycle.Enabled || cycle.NextRunAt.After(now) {
					continue
				}

				// FREIOS HITL OBRIGATÓRIOS: Check for risky actions before executing
				objLower := strings.ToLower(cycle.Objective)
				nameLower := strings.ToLower(cycle.Name)
				combinedText := objLower + " " + nameLower

				// Risk Condition 1: Financial spend / budget allocation
				isSpendRisk := strings.Contains(combinedText, "spend") ||
					strings.Contains(combinedText, "budget") ||
					strings.Contains(combinedText, "buy") ||
					strings.Contains(combinedText, "comprar") ||
					strings.Contains(combinedText, "pagar") ||
					strings.Contains(combinedText, "investir") ||
					strings.Contains(combinedText, "contratar") ||
					strings.Contains(combinedText, "anúncios") ||
					strings.Contains(combinedText, "ads")

				if isSpendRisk && s.config.RequireApprovalRisk {
					approval := CompanyApproval{
						ID:             "appr_" + uuid.NewString()[:8],
						CompanyID:      company.ID,
						OrganizationID: company.OrganizationID,
						ResourceType:   "cycle_action",
						ResourceID:     cycle.ID,
						Policy:         "budget_spend",
						Status:         CompanyApprovalPending,
						Reason:         fmt.Sprintf("Ação de gasto financeiro detectada pelo Supervisor (%s). Requer aprovação HITL prévia.", cycle.Name),
						CreatedAt:      now,
						UpdatedAt:      now,
					}
					_, _ = s.runtime.company.AddApproval(company.ID, approval)
					result.ApprovalsCreated++

					// Postpone next cycle check so it does not loop infinitely
					nextRun := now.Add(time.Hour)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)
					continue
				}

				// Risk Condition 2: External publishing / message blasting
				isPublishRisk := strings.Contains(combinedText, "publish") ||
					strings.Contains(combinedText, "deploy") ||
					strings.Contains(combinedText, "publicar") ||
					strings.Contains(combinedText, "send_external") ||
					strings.Contains(combinedText, "email_blast") ||
					strings.Contains(combinedText, "disparar")

				if isPublishRisk && s.config.RequireApprovalRisk {
					approval := CompanyApproval{
						ID:             "appr_" + uuid.NewString()[:8],
						CompanyID:      company.ID,
						OrganizationID: company.OrganizationID,
						ResourceType:   "cycle_action",
						ResourceID:     cycle.ID,
						Policy:         "external_publish",
						Status:         CompanyApprovalPending,
						Reason:         fmt.Sprintf("Ação de publicação/envio externo detectada pelo Supervisor (%s). Requer aprovação HITL prévia.", cycle.Name),
						CreatedAt:      now,
						UpdatedAt:      now,
					}
					_, _ = s.runtime.company.AddApproval(company.ID, approval)
					result.ApprovalsCreated++

					nextRun := now.Add(time.Hour)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)
					continue
				}

				// Honestidade: Missing external credentials -> BLOCKED_EXTERNAL
				isExternalIntegration := strings.Contains(combinedText, "tiktok_shop") ||
					strings.Contains(combinedText, "meta_ads") ||
					strings.Contains(combinedText, "shopify_sync") ||
					strings.Contains(combinedText, "whatsapp_live")

				if isExternalIntegration {
					// External credentials missing: mark honest status BLOCKED_EXTERNAL
					result.BlockedExternalActions++
					nextRun := now.Add(time.Hour)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)
					continue
				}

				// Standard autonomous operational cycle: advance roadmap -> tasks -> swarm delegation -> mission
				workspace := ""
				missionObj := cycle.Objective
				if missionObj == "" {
					missionObj = fmt.Sprintf("Executar ciclo da empresa %s: %s", company.Name, cycle.Name)
				}

				// Use automatic routing (A2) preferring free models
				model := ""
				if s.runtime != nil {
					if _, ok := s.runtime.plannerResolver.(RoutedPlannerResolver); ok {
						model = "auto/coding"
					}
				}
				mission, err := s.runtime.CreateMission(ctx, CreateMissionRequest{
					Objective:      missionObj,
					Model:          model,
					Workspace:      workspace,
					OrganizationID: company.OrganizationID,
					Capabilities:   []string{"workspace:read", "workspace:write"},
					AutoRun:        true,
				})
				if err == nil {
					_, _ = s.runtime.EnqueueMission(mission.ID)

					// Advance cycle timestamp
					intervalSec := cycle.IntervalSeconds
					if intervalSec <= 0 {
						intervalSec = 86400 // default 24h
					}
					nextRun := now.Add(time.Duration(intervalSec) * time.Second)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)

					// Update KPI & Company report
					_, _ = s.runtime.company.Report(company.ID)
					result.CyclesAdvanced++
				} else {
					s.mu.Lock()
					s.status.LastError = fmt.Sprintf("create mission for cycle %s: %v", cycle.ID, err)
					s.mu.Unlock()
				}
			}
		}
	}

	// Update status
	s.mu.Lock()
	s.status.LastTickAt = &now
	s.status.PendingMissionsResumed += result.MissionsResumed
	s.status.SchedulesTriggered += result.SchedulesTriggered
	s.status.CompanyCyclesAdvanced += result.CyclesAdvanced
	s.status.RiskPausedCompanies = result.RiskPausedCompanies
	s.status.BlockedExternalActions += result.BlockedExternalActions
	s.status.PendingApprovalsCount += result.ApprovalsCreated
	s.mu.Unlock()

	return result, nil
}
