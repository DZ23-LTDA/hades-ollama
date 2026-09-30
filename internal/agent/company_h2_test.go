package agent

import (
	"errors"
	"testing"
)

func TestCompanyApprovalLedgerRejectsRequesterSelfApproval(t *testing.T) {
	store, err := NewCompanyStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org-a", Name: "approval test"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.AddCampaign(company.ID, CompanyCampaign{Name: "ad", Objective: "test"})
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.PendingApproval(company.ID, "campaign", company.Campaigns[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.BindApprovalRequester(company.ID, approval.ID, "operator-a"); err != nil {
		t.Fatal(err)
	}
	current, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DecideApproval(company.ID, approval.ID, true, "self approval must fail", "operator-a", "org-a", current.Version, approval.Nonce)
	if !errors.Is(err, ErrCompanyApprovalSelf) {
		t.Fatalf("self approval error=%v, want %v", err, ErrCompanyApprovalSelf)
	}
	unchanged, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Campaigns[0].Approved || unchanged.Approvals[0].Status != CompanyApprovalPending {
		t.Fatalf("self approval mutated state: %+v", unchanged)
	}
}
