package agent

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestCompanyStoreLifecycleAndPersistence(t *testing.T) {
	root := t.TempDir()
	store, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{
		OrganizationID: "org_local",
		Name:           "Classe A+ Labs",
		Mission:        "Construir produtos agentic úteis",
		Positioning:    "Local-first e seguro",
		BusinessModel:  "SaaS",
		Budget:         CompanyBudget{Currency: "BRL", MonthlyLimitCents: 10000, ApprovalThresholdCents: 2000, RequireApprovalForAds: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if company.ID == "" || len(company.Departments) != 7 || company.Status != CompanyActive {
		t.Fatalf("unexpected company defaults: %+v", company)
	}
	if _, err := store.AddRoadmap(company.ID, CompanyRoadmapItem{Title: "Validar MVP", OwnerDepartment: "product"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddGoal(company.ID, CompanyGoal{Title: "Primeiros clientes", Metric: "clientes_pagos", Target: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddBacklog(company.ID, CompanyBacklogItem{Title: "Landing page", Priority: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCycle(company.ID, CompanyCycle{Name: "Ciclo diário", Objective: "Revisar o roadmap e escolher a próxima tarefa", Frequency: "daily", IntervalSeconds: 24 * 60 * 60}); err != nil {
		t.Fatal(err)
	}
	report, err := store.Report(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if report.OpenBacklog != 1 || report.GoalsOnTrack != 1 || report.EnabledCycles != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if _, err := store.RecordSpend(company.ID, "ads", 100, false); !errors.Is(err, ErrCompanyApprovalRequired) {
		t.Fatalf("expected approval error, got %v", err)
	}
	if _, err := store.RecordSpend(company.ID, "ads", 100, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordSpend(company.ID, "infrastructure", 10001, true); !errors.Is(err, ErrCompanyBudgetExceeded) {
		t.Fatalf("expected budget error, got %v", err)
	}
	paused, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Status != CompanyPaused || !paused.Risk.Paused {
		t.Fatalf("budget did not pause company: %+v", paused)
	}

	reloaded, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reloaded.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Name != company.Name || len(persisted.Roadmap) != 1 || persisted.Budget.SpentCents != 100 {
		t.Fatalf("persistence lost company state: %+v", persisted)
	}
}

func TestCompanyAnomalyAndWorkspaceGuard(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_local", Name: "Guardrails"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAnomaly(company.ID, "high", "taxa de erro acima do limite"); err != nil {
		t.Fatal(err)
	}
	paused, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Status != CompanyPaused || paused.Risk.PauseReason == "" {
		t.Fatalf("high anomaly did not pause company: %+v", paused)
	}
	if got := companyIDFromWorkspace("company://" + company.ID); got != company.ID {
		t.Fatalf("company workspace parser = %q", got)
	}
	if got := companyIDFromWorkspace("/tmp/workspace"); got != "" {
		t.Fatalf("normal workspace parser = %q", got)
	}
}

func TestCompanyCreateResetsServerManagedFields(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.Create(Company{
		ID:             "attacker-controlled",
		OrganizationID: "org-a",
		Name:           "Allowlisted",
		Status:         CompanyPaused,
		Roadmap:        []CompanyRoadmapItem{{ID: "forged"}},
		Budget:         CompanyBudget{SpentCents: 9999},
		Risk:           CompanyRisk{Paused: true, PauseReason: "forged"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "attacker-controlled" || created.Status != CompanyActive || len(created.Roadmap) != 0 || created.Budget.SpentCents != 0 || created.Risk.Paused {
		t.Fatalf("server-managed fields were accepted: %+v", created)
	}
}

func TestCompanyAgentSpendFailsAtomicallyAtBudgetAndPause(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org-a", Name: "Budgeted"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.mutate(company.ID, func(current *Company) error {
		current.Agents[0].BudgetCents = 100
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAgentSpend(company.ID, "ceo", 80); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAgentSpend(company.ID, "ceo", 30); !errors.Is(err, ErrCompanyAgentBudgetExceeded) {
		t.Fatalf("expected atomic agent budget error, got %v", err)
	}
	current, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Agents[0].SpentCents != 80 || current.Agents[0].Status == "paused" {
		t.Fatalf("agent spend mutated after rejected over-budget request: %+v", current.Agents[0])
	}
	if _, err := store.PauseAgent(company.ID, "ceo", "manual test pause"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAgentSpend(company.ID, "ceo", 1); !errors.Is(err, ErrCompanyPaused) {
		t.Fatalf("expected paused agent rejection, got %v", err)
	}
}
