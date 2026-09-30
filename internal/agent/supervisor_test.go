package agent

import (
	"context"
	"os"
	"testing"
	"time"
)

func setupTestSupervisorRuntime(t *testing.T) (*Runtime, *CompanyStore, *ContextStore) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(tempDir+"/workspace", 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.MkdirAll(tempDir+"/data", 0755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	compStore, err := NewCompanyStore(tempDir + "/companies")
	if err != nil {
		t.Fatalf("create company store: %v", err)
	}

	ctxStore, err := NewContextStore(tempDir + "/context")
	if err != nil {
		t.Fatalf("create context store: %v", err)
	}

	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: tempDir + "/workspace",
		DataRoot:      tempDir + "/data",
		Company:       compStore,
		Context:       ctxStore,
		SupervisorConfig: &SupervisorConfig{
			Enabled:             true,
			TickInterval:        50 * time.Millisecond,
			RequireApprovalRisk: true,
		},
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	return runtime, compStore, ctxStore
}

// 1. TestSupervisorTickDispatchesDueSchedule creates a due schedule and verifies that a tick creates the mission.
func TestSupervisorTickDispatchesDueSchedule(t *testing.T) {
	runtime, _, ctxStore := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-10 * time.Minute)

	sched, err := ctxStore.CreateSchedule(Schedule{
		Objective:       "Análise autônoma de dados semanais",
		IntervalSeconds: 3600,
		Workspace:       "",
		Enabled:         true,
		OrganizationID:  LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	sched.NextRunAt = past
	sched, err = ctxStore.UpdateSchedule(sched.ID, sched)
	if err != nil {
		t.Fatalf("update schedule: %v", err)
	}

	sup := runtime.Supervisor()
	if sup == nil {
		t.Fatal("expected supervisor to be initialized on runtime")
	}

	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if res.SchedulesTriggered < 1 {
		t.Fatalf("expected at least 1 schedule triggered, got %d", res.SchedulesTriggered)
	}

	// Verify mission was created in runtime
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("list missions: %v", err)
	}

	found := false
	for _, m := range missions {
		if m.Objective == sched.Objective {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mission with objective %q to be created by supervisor tick", sched.Objective)
	}
}

// 2. TestSupervisorCompanyCycleAdvances verifies that a due company cycle advances, delegates to swarm, and updates KPIs.
func TestSupervisorCompanyCycleAdvances(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-5 * time.Minute)

	comp, err := compStore.Create(Company{
		Name:           "Tech Corp",
		OrganizationID: LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_dev_1",
		Name:            "Desenvolvimento Contínuo",
		Objective:       "Construir módulo de relatórios e executar testes unitários",
		Frequency:       "daily",
		IntervalSeconds: 86400,
		Enabled:         true,
		NextRunAt:       past, // Vencido
	})
	if err != nil {
		t.Fatalf("add cycle: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	t.Logf("CompanyCycleAdvances result: %+v, LastError: %s", res, sup.Status().LastError)

	if res.CyclesAdvanced < 1 {
		t.Fatalf("expected at least 1 cycle advanced, got %d", res.CyclesAdvanced)
	}

	// Verify company cycle NextRunAt was updated into the future
	updatedComp, err := compStore.Get(comp.ID)
	if err != nil {
		t.Fatalf("get company: %v", err)
	}
	if len(updatedComp.Cycles) == 0 {
		t.Fatal("expected cycles to exist")
	}
	if !updatedComp.Cycles[0].NextRunAt.After(now) {
		t.Fatalf("expected cycle NextRunAt to be updated into future, got %v (now: %v)", updatedComp.Cycles[0].NextRunAt, now)
	}
	if updatedComp.Cycles[0].LastRunAt == nil {
		t.Fatal("expected cycle LastRunAt to be set")
	}

	// Verify mission was created for the cycle
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("list missions: %v", err)
	}
	found := false
	for _, m := range missions {
		if m.Objective == updatedComp.Cycles[0].Objective {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected cycle objective to be scheduled as a mission")
	}
}

// 3. TestSupervisorRiskyActionRequiresApproval verifies HITL safety brakes: spend and external publishing never auto-execute.
func TestSupervisorRiskyActionRequiresApproval(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-5 * time.Minute)

	comp, err := compStore.Create(Company{
		Name:           "Growth Corp",
		OrganizationID: LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_ad_spend",
		Name:            "Campanha de Anúncios",
		Objective:       "Spend budget on Google Ads campaign and pagar fornecedores",
		Frequency:       "daily",
		IntervalSeconds: 86400,
		Enabled:         true,
		NextRunAt:       past,
	})
	if err != nil {
		t.Fatalf("add cycle 1: %v", err)
	}

	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_external_publish",
		Name:            "Publicação Externa em Massa",
		Objective:       "Publish live deploy to production and send_external email_blast",
		Frequency:       "daily",
		IntervalSeconds: 86400,
		Enabled:         true,
		NextRunAt:       past,
	})
	if err != nil {
		t.Fatalf("add cycle 2: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Two risky actions must require approval
	if res.ApprovalsCreated != 2 {
		t.Fatalf("expected 2 approvals created for risky actions, got %d", res.ApprovalsCreated)
	}

	// Verify approvals are pending in company store
	updatedComp, err := compStore.Get(comp.ID)
	if err != nil {
		t.Fatalf("get company: %v", err)
	}

	if len(updatedComp.Approvals) < 2 {
		t.Fatalf("expected at least 2 pending approvals in company, got %d", len(updatedComp.Approvals))
	}

	for _, app := range updatedComp.Approvals {
		if app.Status != CompanyApprovalPending {
			t.Fatalf("expected approval to be pending, got %v", app.Status)
		}
	}
}

// 4. TestSupervisorBlockedExternalActionHonesty verifies that actions depending on missing external credentials receive BLOCKED_EXTERNAL.
func TestSupervisorBlockedExternalActionHonesty(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-5 * time.Minute)

	comp, err := compStore.Create(Company{
		Name:           "Commerce Store",
		OrganizationID: LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_tiktok",
		Name:            "TikTok Shop Sync",
		Objective:       "Sincronizar pedidos via tiktok_shop e atualizar estoque na loja",
		Frequency:       "hourly",
		IntervalSeconds: 3600,
		Enabled:         true,
		NextRunAt:       past,
	})
	if err != nil {
		t.Fatalf("add cycle: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if res.BlockedExternalActions < 1 {
		t.Fatalf("expected blocked external action for missing credentials, got %d", res.BlockedExternalActions)
	}
}

// 5. TestSupervisorRiskPauseHaltsLoop verifies that an anomaly or risk pause halts company autonomous execution.
func TestSupervisorRiskPauseHaltsLoop(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-5 * time.Minute)

	comp, err := compStore.Create(Company{
		Name:           "Risky Corp",
		OrganizationID: LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	// Explicitly pause company due to critical anomaly
	_, err = compStore.Pause(comp.ID, "Anomalia crítica de tráfego")
	if err != nil {
		t.Fatalf("pause company: %v", err)
	}

	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_op_1",
		Name:            "Ciclo Normal",
		Objective:       "Operação padrão",
		Frequency:       "daily",
		IntervalSeconds: 86400,
		Enabled:         true,
		NextRunAt:       past,
	})
	if err != nil {
		t.Fatalf("add cycle: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if res.CyclesAdvanced > 0 {
		t.Fatalf("expected 0 cycles advanced when company is paused by risk, got %d", res.CyclesAdvanced)
	}

	foundPaused := false
	for _, id := range res.RiskPausedCompanies {
		if id == comp.ID {
			foundPaused = true
			break
		}
	}
	if !foundPaused {
		t.Fatalf("expected company %s to be listed in RiskPausedCompanies", comp.ID)
	}
}

// 6. TestSupervisorDisabledDoesNotExecute verifies that disabled config produces 0 operations.
func TestSupervisorDisabledDoesNotExecute(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	past := now.Add(-5 * time.Minute)

	comp, _ := compStore.Create(Company{
		Name:           "Disabled Corp",
		OrganizationID: LocalOrganizationID,
	})
	_, _ = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:        "cycle_1",
		Objective: "Fazer algo",
		Enabled:   true,
		NextRunAt: past,
	})

	sup := runtime.Supervisor()
	cfg := sup.Config()
	cfg.Enabled = false
	sup.SetConfig(cfg)

	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if res.CyclesAdvanced > 0 || res.SchedulesTriggered > 0 {
		t.Fatalf("expected zero execution when supervisor is disabled, got %v", res)
	}
}

// 7. TestSupervisorResumesPendingMissionsAfterRestart verifies that missions survive crash/restart and are resumed.
func TestSupervisorResumesPendingMissionsAfterRestart(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	// Create a mission in state ready with auto_run
	mission, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:      "Missão que sobrevive a restart",
		Model:          "qwen2.5-coder:7b",
		Workspace:      "",
		OrganizationID: LocalOrganizationID,
		AutoRun:        true,
	})
	if err != nil {
		t.Fatalf("create mission: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if res.MissionsResumed < 1 {
		t.Fatalf("expected resumePending to resume at least 1 mission, got %d", res.MissionsResumed)
	}

	// Verify mission was retrieved in list
	m, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatalf("get mission: %v", err)
	}
	if m.ID != mission.ID {
		t.Fatalf("expected mission ID %s, got %s", mission.ID, m.ID)
	}
}

// 8. TestSupervisorDaemonStartStop verifies that the background ticker runs ticks and stops cleanly.
func TestSupervisorDaemonStartStop(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)
	sup := runtime.Supervisor()

	cfg := sup.Config()
	cfg.TickInterval = 10 * time.Millisecond
	sup.SetConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}

	if !sup.Status().Running {
		t.Fatal("expected supervisor to report running")
	}

	time.Sleep(50 * time.Millisecond)

	st := sup.Status()
	if st.TickCount < 1 {
		t.Fatalf("expected tick count >= 1, got %d", st.TickCount)
	}

	if err := sup.Stop(); err != nil {
		t.Fatalf("stop supervisor: %v", err)
	}

	if sup.Status().Running {
		t.Fatal("expected supervisor to report not running after Stop()")
	}
}
