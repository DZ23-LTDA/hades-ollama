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
	// ScheduleRunErrors expõe falhas de execução de schedule no próprio tick.
	// Antes, uma falha de fluxo neste caminho não era nem registrada nem
	// reportada: desaparecia.
	ScheduleRunErrors []string `json:"schedule_run_errors,omitempty"`
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
	// Registro de incidente (estágio 31): estado, tentativas e recuperação
	// observada. Mutex próprio para não acoplar ao lock do ciclo de vida.
	incidentsMu sync.Mutex
	incidents   []SupervisorIncident
	// Diário de ticks (evidência "logs de supervisão"): contadores por tick,
	// limitado e persistido, incluindo os erros de cada tick.
	tickJournal []SupervisorTickRecord
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
	supervisor := &Supervisor{
		runtime: runtime,
		config:  config,
		status: SupervisorStatus{
			Enabled:  config.Enabled,
			WorkerID: config.WorkerID,
		},
	}
	// O registro precisa sobreviver ao reinício: ocorrência que desaparece na
	// reinicialização não serve para operação.
	_ = supervisor.loadIncidents()
	_ = supervisor.loadTickJournal()
	return supervisor
}

// companyCycleRisk classifies the action a Company OS cycle intends to take.
// Explicit risk is authoritative; recognized sensitive intents are conservative
// by default and must pass the approval ledger before a mission is created.
func companyCycleRisk(cycle CompanyCycle) RiskClass {
	if cycle.Risk != "" {
		return effectiveRisk(cycle.Risk)
	}
	intent := companyCycleIntentTokens(cycle)
	if containsAnyCompanyIntent(intent, "spend", "budget", "buy", "purchase", "transfer", "transferir", "acquire", "adquirir", "comprar", "compras", "pagar", "pagamento", "investir", "contratar", "anuncios", "anúncios", "ads") {
		return RiskDestructive
	}
	if containsAnyCompanyIntent(intent, "publish", "post", "deploy", "publicar", "postar", "lancar", "lançar", "lancar", "lançar", "send_external", "email_blast", "disparar", "enviar") {
		return RiskExternalSideEffect
	}
	return RiskRead
}

func companyCycleIntentTokens(cycle CompanyCycle) map[string]struct{} {
	combined := strings.ToLower(strings.TrimSpace(cycle.Name + " " + cycle.Objective))
	for _, separator := range []string{"-", "_", "/", ":", ",", ".", "(", ")", "[", "]"} {
		combined = strings.ReplaceAll(combined, separator, " ")
	}
	// Explicit tokens avoid substring matches such as "adsorption" or
	// "deployment-notes" while retaining conservative coverage for commands.
	result := make(map[string]struct{})
	for _, token := range strings.Fields(combined) {
		result[token] = struct{}{}
	}
	return result
}

func containsAnyCompanyIntent(tokens map[string]struct{}, values ...string) bool {
	for _, value := range values {
		if _, ok := tokens[value]; ok {
			return true
		}
	}
	return false
}

// externalIntegrationKeys are connector identifiers whose cycles require live
// credentials and therefore must be blocked until the integration is actually
// connected — an approval must never imply an unavailable integration is ready.
var externalIntegrationKeys = []string{
	"tiktok_shop",
	"meta_ads",
	"shopify_sync",
	"whatsapp_live",
}

// cycleRequestsExternalIntegration reports whether a cycle intends to drive an
// external connector. The structured Integrations field is authoritative; when
// it is empty (cycles persisted before the field existed) it falls back to a
// conservative scan of the free-text name/objective.
func cycleRequestsExternalIntegration(cycle CompanyCycle) bool {
	if len(cycle.Integrations) > 0 {
		for _, declared := range cycle.Integrations {
			normalized := strings.ToLower(strings.TrimSpace(declared))
			for _, key := range externalIntegrationKeys {
				if normalized == key {
					return true
				}
			}
		}
		// A cycle that declared its integrations explicitly and named none of
		// the external connectors is trusted: do not second-guess it with a
		// fragile free-text scan.
		return false
	}
	combined := strings.ToLower(strings.TrimSpace(cycle.Name + " " + cycle.Objective))
	for _, key := range externalIntegrationKeys {
		if strings.Contains(combined, key) {
			return true
		}
	}
	return false
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

			// Execute the persisted flow graph of the schedule, or the legacy
			// single mission when the schedule carries no steps. Both dispatch
			// sites share ExecuteScheduleFlowReport so semantics cannot diverge,
			// e ambos registram o resultado: antes o supervisor não gravava nada
			// e uma falha de fluxo desaparecia sem rastro.
			report, err := s.runtime.ExecuteScheduleFlowReport(ctx, sched)
			if _, recordErr := s.runtime.recordScheduleRun(sched, report, err); recordErr != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "record "+sched.ID+": "+recordErr.Error())
			}
			if err != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "run "+sched.ID+": "+err.Error())
				if incidentErr := s.recordIncident(SupervisorIncidentScheduleFailure, sched.ID, err.Error(), now); incidentErr != nil {
					result.ScheduleRunErrors = append(result.ScheduleRunErrors, "incidente "+sched.ID+": "+incidentErr.Error())
				}
				continue
			}
			// SUCESSO OBSERVADO: se havia incidente aberto deste schedule, a
			// recuperação aconteceu de fato e é registrada com data.
			if _, resolveErr := s.resolveIncident(SupervisorIncidentScheduleFailure, sched.ID, "execução concluída com sucesso em tick posterior", now); resolveErr != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "resolver incidente "+sched.ID+": "+resolveErr.Error())
			}
			result.SchedulesTriggered += report.Missions
			if skipped := report.SkippedSteps(); len(skipped) > 0 {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors,
					"run "+sched.ID+": passos ignorados: "+strings.Join(skipped, ", "))
				if incidentErr := s.recordIncident(SupervisorIncidentScheduleFailure, sched.ID,
					"passos ignorados: "+strings.Join(skipped, ", "), now); incidentErr != nil {
					result.ScheduleRunErrors = append(result.ScheduleRunErrors, "incidente "+sched.ID+": "+incidentErr.Error())
				}
			}
		}
	}

	// 2. Resumable & Long-running: Recover and resume pending missions and leases
	if s.runtime != nil {
		resumeErr := s.runtime.resumePending(ctx)
		if resumeErr != nil {
			// Recovery is idempotent. Retry once when a temporary filesystem
			// race interrupts the first scan.
			resumeErr = s.runtime.resumePending(ctx)
		}
		if resumeErr == nil {
			result.MissionsResumed++
			if _, resolveErr := s.resolveIncident(SupervisorIncidentRecoveryFailure, "missions",
				"retomada de missões pendentes voltou a funcionar", now); resolveErr != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "resolver incidente de retomada: "+resolveErr.Error())
			}
		} else {
			// Falha PERSISTENTE de retomada era engolida em silêncio: o operador
			// não tinha como saber que missões pendentes pararam de ser retomadas.
			result.ScheduleRunErrors = append(result.ScheduleRunErrors, "resume: "+resumeErr.Error())
			if incidentErr := s.recordIncident(SupervisorIncidentRecoveryFailure, "missions", resumeErr.Error(), now); incidentErr != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "incidente de retomada: "+incidentErr.Error())
			}
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
				// Pausa por risco não é detalhe de log: é incidente que exige
				// ciência humana, porque o trabalho autônomo da empresa parou.
				if incidentErr := s.recordIncident(SupervisorIncidentRiskPause, company.ID,
					"empresa pausada por risco; o trabalho autônomo está parado", now); incidentErr != nil {
					result.ScheduleRunErrors = append(result.ScheduleRunErrors, "incidente "+company.ID+": "+incidentErr.Error())
				}
				continue
			}
			// Empresa que voltou a operar: a recuperação é observada e registrada.
			if _, resolveErr := s.resolveIncident(SupervisorIncidentRiskPause, company.ID,
				"empresa voltou a operar sem pausa por risco", now); resolveErr != nil {
				result.ScheduleRunErrors = append(result.ScheduleRunErrors, "resolver incidente "+company.ID+": "+resolveErr.Error())
			}

			for _, cycle := range company.Cycles {
				if !cycle.Enabled || cycle.NextRunAt.After(now) {
					continue
				}

				// FREIOS HITL OBRIGATÓRIOS: Check for risky actions before executing
				cycleRisk := companyCycleRisk(cycle)

				// Missing credentials are blocked before approval: an approval
				// must never imply that an unavailable integration is connected.
				if cycleRequestsExternalIntegration(cycle) {
					result.BlockedExternalActions++
					nextRun := now.Add(time.Hour)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)
					continue
				}

				// Approval is mandatory for every non-read risk. The config flag
				// cannot disable this safety boundary.
				if cycleRisk == RiskDestructive {
					approval := CompanyApproval{
						ID:             "appr_" + uuid.NewString()[:8],
						CompanyID:      company.ID,
						OrganizationID: company.OrganizationID,
						ResourceType:   "cycle_action",
						ResourceID:     cycle.ID,
						Policy:         "budget_spend",
						Nonce:          uuid.NewString(),
						Status:         CompanyApprovalPending,
						Reason:         fmt.Sprintf("Ação de gasto financeiro detectada pelo Supervisor (%s). Requer aprovação HITL prévia.", cycle.Name),
						CreatedAt:      now,
						UpdatedAt:      now,
					}
					expiresAt := now.Add(15 * time.Minute)
					approval.ExpiresAt = &expiresAt
					_, _ = s.runtime.company.AddApproval(company.ID, approval)
					result.ApprovalsCreated++
					nextRun := now.Add(time.Hour)
					_, _ = s.runtime.company.UpdateCycleRun(company.ID, cycle.ID, now, nextRun)
					continue
				}

				if cycleRisk == RiskExternalSideEffect {
					approval := CompanyApproval{
						ID:             "appr_" + uuid.NewString()[:8],
						CompanyID:      company.ID,
						OrganizationID: company.OrganizationID,
						ResourceType:   "cycle_action",
						ResourceID:     cycle.ID,
						Policy:         "external_publish",
						Nonce:          uuid.NewString(),
						Status:         CompanyApprovalPending,
						Reason:         fmt.Sprintf("Ação de publicação/envio externo detectada pelo Supervisor (%s). Requer aprovação HITL prévia.", cycle.Name),
						CreatedAt:      now,
						UpdatedAt:      now,
					}
					expiresAt := now.Add(15 * time.Minute)
					approval.ExpiresAt = &expiresAt
					_, _ = s.runtime.company.AddApproval(company.ID, approval)
					result.ApprovalsCreated++
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

	// Diário do tick, incluindo os erros: é a evidência de "logs de supervisão".
	s.recordTick(result)

	return result, nil
}
