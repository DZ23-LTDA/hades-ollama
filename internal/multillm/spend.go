package multillm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cost control for the router: track accumulated spend per organization and
// refuse calls that would exceed a configured daily or monthly cap, so a
// provider never silently bills past the user's budget (G3 / A9). There is no
// silent fallback to a paid provider — exceeding the cap is an explicit error.

// ErrSpendCapExceeded is returned by Authorize when a call would push spend over
// the configured daily or monthly limit.
var ErrSpendCapExceeded = errors.New("spend cap exceeded")

// CostCents returns the cost in cents for a call to this model with the given
// input/output token counts, using the model's per-1K pricing.
func (m Model) CostCents(inputTokens, outputTokens int64) int64 {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	return (inputTokens*m.CostPer1KInputCents)/1000 + (outputTokens*m.CostPer1KOutputCents)/1000
}

type spendRecord struct {
	At             time.Time `json:"at"`
	OrganizationID string    `json:"organization_id,omitempty"`
	Model          string    `json:"model,omitempty"`
	Cents          int64     `json:"cents"`
}

// SpendLedger accumulates per-organization spend and enforces caps. A cap of 0
// means "no limit". It is safe for concurrent use.
type SpendLedger struct {
	mu              sync.Mutex
	path            string
	dailyCapCents   int64
	monthlyCapCents int64
	records         []spendRecord
}

// NewSpendLedger creates a ledger, loading prior records from path when it
// exists. An empty path keeps the ledger in memory only.
func NewSpendLedger(path string, dailyCapCents, monthlyCapCents int64) (*SpendLedger, error) {
	l := &SpendLedger{path: path, dailyCapCents: dailyCapCents, monthlyCapCents: monthlyCapCents}
	if path == "" {
		return l, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return l, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &l.records); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func sameDay(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func sameMonth(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	ay, am, _ := a.Date()
	by, bm, _ := b.Date()
	return ay == by && am == bm
}

func (l *SpendLedger) spentLocked(org string, now time.Time, within func(a, b time.Time) bool) int64 {
	var total int64
	for _, r := range l.records {
		if r.OrganizationID != org {
			continue
		}
		if within(r.At, now) {
			total += r.Cents
		}
	}
	return total
}

// SpentTodayCents returns the organization's spend for the UTC day of now.
func (l *SpendLedger) SpentTodayCents(org string, now time.Time) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.spentLocked(org, now, sameDay)
}

// SpentThisMonthCents returns the organization's spend for the UTC month of now.
func (l *SpendLedger) SpentThisMonthCents(org string, now time.Time) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.spentLocked(org, now, sameMonth)
}

// Authorize reports whether adding addCents of spend for org at now stays within
// both caps. It does not record anything; call Record after a successful call.
func (l *SpendLedger) Authorize(org string, addCents int64, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.dailyCapCents > 0 && l.spentLocked(org, now, sameDay)+addCents > l.dailyCapCents {
		return ErrSpendCapExceeded
	}
	if l.monthlyCapCents > 0 && l.spentLocked(org, now, sameMonth)+addCents > l.monthlyCapCents {
		return ErrSpendCapExceeded
	}
	return nil
}

// Record appends spend for a completed call and persists the ledger when a path
// is configured.
func (l *SpendLedger) Record(org, model string, cents int64, now time.Time) error {
	if cents <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, spendRecord{At: now.UTC(), OrganizationID: org, Model: model, Cents: cents})
	return l.persistLocked()
}

func (l *SpendLedger) persistLocked() error {
	if l.path == "" {
		return nil
	}
	data, err := json.Marshal(l.records)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}
