package agent

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestCompanyStoreRollsBackMutationsWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	store, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{
		OrganizationID: "org_atomic",
		Name:           "Original",
		Budget:         CompanyBudget{Currency: "BRL", MonthlyLimitCents: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	store.root = unwritableStoreRoot(t, root, "companies")
	if _, err := store.Update(company.ID, CompanyUpdate{Name: "Changed"}); err == nil {
		t.Fatal("company update unexpectedly succeeded")
	}
	if got, err := store.Get(company.ID); err != nil || got.Name != "Original" {
		t.Fatalf("company remained changed after failed update: got=%+v err=%v", got, err)
	}
	if _, err := store.AddRoadmap(company.ID, CompanyRoadmapItem{Title: "must rollback"}); err == nil {
		t.Fatal("roadmap write unexpectedly succeeded")
	}
	if got, err := store.Get(company.ID); err != nil || len(got.Roadmap) != 0 {
		t.Fatalf("roadmap remained after failed mutation: got=%+v err=%v", got, err)
	}
	if _, err := store.RecordSpend(company.ID, "infrastructure", 101, true); err == nil || !errors.Is(err, ErrCompanyBudgetExceeded) {
		t.Fatalf("budget error=%v, want ErrCompanyBudgetExceeded plus persistence error", err)
	}
	if got, err := store.Get(company.ID); err != nil || got.Status != CompanyActive || got.Budget.SpentCents != 0 || got.Risk.Paused {
		t.Fatalf("company pause remained after failed persistence: got=%+v err=%v", got, err)
	}
	reloaded, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := reloaded.Get(company.ID); err != nil || got.Name != "Original" || len(got.Roadmap) != 0 || got.Budget.SpentCents != 0 {
		t.Fatalf("company disk diverged after failed mutations: got=%+v err=%v", got, err)
	}
}
