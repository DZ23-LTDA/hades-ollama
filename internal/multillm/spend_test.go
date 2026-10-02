package multillm

import (
	"path/filepath"
	"testing"
	"time"
)

func TestModelCostCents(t *testing.T) {
	m := Model{CostPer1KInputCents: 30, CostPer1KOutputCents: 60}
	if got := m.CostCents(2000, 1000); got != 120 { // 2*30 + 1*60
		t.Fatalf("CostCents = %d, want 120", got)
	}
	if got := m.CostCents(0, 0); got != 0 {
		t.Fatalf("CostCents(0,0) = %d, want 0", got)
	}
}

func TestSpendLedgerEnforcesDailyAndMonthlyCaps(t *testing.T) {
	ledger, err := NewSpendLedger("", 100 /*daily*/, 250 /*monthly*/)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

	// Under the daily cap: authorized and recorded.
	if err := ledger.Authorize("org", 60, now); err != nil {
		t.Fatalf("authorize 60: %v", err)
	}
	if err := ledger.Record("org", "gpt-x", 60, now); err != nil {
		t.Fatal(err)
	}
	// Another 60 would be 120 > 100 daily cap -> blocked.
	if err := ledger.Authorize("org", 60, now); err != ErrSpendCapExceeded {
		t.Fatalf("over-daily authorize err = %v, want ErrSpendCapExceeded", err)
	}
	// A different organization is unaffected by org's spend.
	if err := ledger.Authorize("other", 90, now); err != nil {
		t.Fatalf("other org authorize: %v", err)
	}

	// Next day resets the daily window; monthly still accumulates.
	tomorrow := now.Add(24 * time.Hour)
	if got := ledger.SpentTodayCents("org", tomorrow); got != 0 {
		t.Fatalf("spent tomorrow = %d, want 0 (daily reset)", got)
	}
	if got := ledger.SpentThisMonthCents("org", tomorrow); got != 60 {
		t.Fatalf("spent this month = %d, want 60", got)
	}
	// Spend up to the monthly cap over subsequent days, then block.
	if err := ledger.Record("org", "gpt-x", 90, tomorrow); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Record("org", "gpt-x", 90, tomorrow.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Month total now 240; +20 = 260 > 250 monthly cap -> blocked.
	if err := ledger.Authorize("org", 20, tomorrow.Add(48*time.Hour)); err != ErrSpendCapExceeded {
		t.Fatalf("over-monthly authorize err = %v, want ErrSpendCapExceeded", err)
	}
}

func TestSpendLedgerPersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spend", "ledger.json")
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	l1, err := NewSpendLedger(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := l1.Record("org", "m", 42, now); err != nil {
		t.Fatal(err)
	}
	l2, err := NewSpendLedger(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := l2.SpentTodayCents("org", now); got != 42 {
		t.Fatalf("reloaded spend = %d, want 42", got)
	}
}
