package server

import (
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

// TestSpendLedgerEnforcesPaidModelCap proves the router honors a per-organization
// spend cap: once an org's recorded spend would exceed the cap, paid models are
// refused (so the resolver falls back to a local/free planner), while free
// models and other organizations are unaffected, and no ledger means no limit.
func TestSpendLedgerEnforcesPaidModelCap(t *testing.T) {
	ledger, err := multillm.NewSpendLedger("", 15, 0) // 15 cents/day, no monthly cap
	if err != nil {
		t.Fatal(err)
	}
	resolver := multiProviderPlannerResolver{spend: ledger}
	// CostCents(1000,1000) = input + output cents = 10c per routing.
	paid := multillm.Model{ID: "openai/gpt", Provider: "openai", CostPer1KInputCents: 5, CostPer1KOutputCents: 5}
	free := multillm.Model{ID: "ollama/local", Provider: "ollama-local", CostTag: "0-local"}

	if !resolver.spendAllows("org-a", paid) {
		t.Fatal("first paid routing (10c) should be within the 15c cap")
	}
	if resolver.spendAllows("org-a", paid) {
		t.Fatal("second paid routing (total 20c) must exceed the 15c daily cap")
	}
	if !resolver.spendAllows("org-a", free) {
		t.Fatal("a free/local model must never be blocked by the spend cap")
	}
	if !resolver.spendAllows("org-b", paid) {
		t.Fatal("a different organization has its own budget and must not be blocked")
	}

	noLedger := multiProviderPlannerResolver{}
	if !noLedger.spendAllows("org-a", paid) {
		t.Fatal("without a configured ledger, routing must never be blocked")
	}
}
