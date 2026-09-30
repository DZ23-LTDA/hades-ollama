package agent

import (
	"errors"
	"fmt"
	"strings"
)

// GateStatus defines the honest evaluation state for any capability or gate.
type GateStatus string

const (
	GateStatusPass            GateStatus = "PASS"
	GateStatusFail            GateStatus = "FAIL"
	GateStatusNotExecuted     GateStatus = "NOT_EXECUTED"
	GateStatusNotPresent      GateStatus = "NOT_PRESENT"
	GateStatusNotConfigured   GateStatus = "NOT_CONFIGURED"
	GateStatusBlockedExternal GateStatus = "BLOCKED_EXTERNAL"
	GateStatusUnknown         GateStatus = "UNKNOWN"
)

// ValidGateStatuses contains all allowed honest statuses.
var ValidGateStatuses = map[GateStatus]bool{
	GateStatusPass:            true,
	GateStatusFail:            true,
	GateStatusNotExecuted:     true,
	GateStatusNotPresent:      true,
	GateStatusNotConfigured:   true,
	GateStatusBlockedExternal: true,
	GateStatusUnknown:         true,
}

// CapabilityGate holds the honest state of an individual capability or release gate.
type CapabilityGate struct {
	ID          string     `json:"id"`
	Domain      string     `json:"domain"`
	Capability  string     `json:"capability"`
	Status      GateStatus `json:"status"`
	Executed    bool       `json:"executed"`
	EvidenceRef string     `json:"evidence_ref,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

// EvaluateGate enforces the fundamental hard rule:
// A gate that was NOT executed is NOT_EXECUTED and can NEVER be PASS.
// Any attempt to mark PASS without verified execution or without an evidence reference is an error.
func EvaluateGate(id, domain, capability string, executed bool, passed bool, hasConfig bool, externalBlocked bool, evidenceRef, notes string) (CapabilityGate, error) {
	gate := CapabilityGate{
		ID:          strings.TrimSpace(id),
		Domain:      strings.TrimSpace(domain),
		Capability:  strings.TrimSpace(capability),
		Executed:    executed,
		EvidenceRef: strings.TrimSpace(evidenceRef),
		Notes:       strings.TrimSpace(notes),
	}

	if externalBlocked {
		gate.Status = GateStatusBlockedExternal
		return gate, nil
	}
	if !hasConfig {
		gate.Status = GateStatusNotConfigured
		return gate, nil
	}
	if !executed {
		gate.Status = GateStatusNotExecuted
		return gate, nil
	}

	if passed {
		if gate.EvidenceRef == "" {
			return gate, fmt.Errorf("gate %q: cannot mark PASS without explicit evidence reference", id)
		}
		gate.Status = GateStatusPass
	} else {
		gate.Status = GateStatusFail
	}
	return gate, nil
}

// ValidateGateStatus verifies that a status string belongs to the honest enum.
func ValidateGateStatus(status GateStatus) error {
	if !ValidGateStatuses[status] {
		return fmt.Errorf("invalid gate status %q: must be one of PASS, FAIL, NOT_EXECUTED, NOT_PRESENT, NOT_CONFIGURED, BLOCKED_EXTERNAL, UNKNOWN", status)
	}
	return nil
}

// GateMatrix represents a collection of capability gates with aggregate health checks.
type GateMatrix struct {
	Gates []CapabilityGate `json:"gates"`
}

// NewGateMatrix creates a new gate matrix.
func NewGateMatrix() *GateMatrix {
	return &GateMatrix{Gates: make([]CapabilityGate, 0)}
}

// Add appends a evaluated gate to the matrix.
func (m *GateMatrix) Add(g CapabilityGate) error {
	if err := ValidateGateStatus(g.Status); err != nil {
		return err
	}
	// Hard check: never allow fake PASS
	if g.Status == GateStatusPass && (!g.Executed || g.EvidenceRef == "") {
		return errors.New("integrity violation: gate marked PASS without execution or evidence")
	}
	m.Gates = append(m.Gates, g)
	return nil
}

// Summary returns the count of each status in the matrix.
func (m *GateMatrix) Summary() map[GateStatus]int {
	summary := make(map[GateStatus]int)
	for _, g := range m.Gates {
		summary[g.Status]++
	}
	return summary
}
