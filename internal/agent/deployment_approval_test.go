package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeploymentApprovalPersistsAcrossRestartAndConsumesOnce(t *testing.T) {
	root := t.TempDir()
	store, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	requested, err := store.Request("org_a", "bld_a", "self", "staging", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	if requested.Status != DeploymentApprovalPending || requested.Nonce == "" {
		t.Fatalf("requested approval = %+v", requested)
	}
	if _, err := store.Decide(requested.ID, "org_a", "bld_a", "self", "admin_a", "approved", requested.Nonce, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide(requested.ID, "org_a", "bld_a", "self", "operator_a", "replay", requested.Nonce, true); !errors.Is(err, ErrDeploymentApprovalConflict) {
		t.Fatalf("repeated decision error = %v", err)
	}

	reloaded, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.Consume(requested.ID, "org_a", "bld_a", "self", "staging", "manifest-b", requested.Nonce); err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Fatalf("manifest mismatch error = %v", err)
	}
	consumed, err := reloaded.Consume(requested.ID, "org_a", "bld_a", "self", "staging", "manifest-a", requested.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.Status != DeploymentApprovalConsumed || consumed.DecidedBy != "admin_a" {
		t.Fatalf("consumed approval = %+v", consumed)
	}
	if _, err := reloaded.Consume(requested.ID, "org_a", "bld_a", "self", "staging", "manifest-a", requested.Nonce); !errors.Is(err, ErrDeploymentApprovalConflict) {
		t.Fatalf("replay consume error = %v", err)
	}
}

func TestDeploymentApprovalRejectsWrongTenantAndNonce(t *testing.T) {
	store, err := NewDeploymentApprovalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.Request("org_a", "bld_a", "self", "production", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide(approval.ID, "org_b", "bld_a", "self", "admin_b", "approve", approval.Nonce, true); !errors.Is(err, ErrDeploymentApprovalOrganization) {
		t.Fatalf("wrong tenant decision error = %v", err)
	}
	if _, err := store.Decide(approval.ID, "org_a", "bld_a", "self", "admin_a", "approve", "wrong", true); !errors.Is(err, ErrDeploymentApprovalNonce) {
		t.Fatalf("wrong nonce decision error = %v", err)
	}
}

func TestDeploymentApprovalRequiresApproverActor(t *testing.T) {
	store, err := NewDeploymentApprovalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.Request("org_a", "bld_a", "self", "staging", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide(approval.ID, "org_a", "bld_a", "self", "  ", "approve", approval.Nonce, true); err == nil || !strings.Contains(err.Error(), "actor is required") {
		t.Fatalf("blank approver error=%v, want actor rejection", err)
	}
	listed := store.ListForOrganization("org_a")
	if len(listed) != 1 || listed[0].Status != DeploymentApprovalPending || listed[0].DecidedBy != "" {
		t.Fatalf("blank approver changed approval state: %+v", listed)
	}
}

func TestDeploymentApprovalReloadRejectsDecidedRecordWithoutActor(t *testing.T) {
	root := t.TempDir()
	store, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.Request("org_a", "bld_a", "self", "staging", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	decided, err := store.Decide(approval.ID, "org_a", "bld_a", "self", "approver_a", "approved", approval.Nonce, true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "approvals.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records map[string]DeploymentApproval
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	record := records[decided.ID]
	record.DecidedBy = "  "
	records[decided.ID] = record
	data, err = json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "approvals.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDeploymentApprovalStore(root); err == nil || !strings.Contains(err.Error(), "deciding actor") {
		t.Fatalf("tampered decided approval reload error=%v, want fail-closed actor validation", err)
	}
}

func TestDeploymentApprovalExpiryRecordsSystemActor(t *testing.T) {
	root := t.TempDir()
	store, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.Request("org_a", "bld_a", "self", "staging", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	stored := store.approvals[approval.ID]
	stored.ExpiresAt = time.Now().UTC().Add(-time.Second)
	store.approvals[approval.ID] = stored
	if _, err := store.Decide(approval.ID, "org_a", "bld_a", "self", "approver_a", "approve", approval.Nonce, true); !errors.Is(err, ErrDeploymentApprovalExpired) {
		t.Fatalf("expired decision error=%v, want expiration", err)
	}
	reloaded, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	decision := reloaded.ListForOrganization("org_a")
	if len(decision) != 1 || decision[0].Status != DeploymentApprovalRejected || decision[0].DecidedBy != "system:expiry" || decision[0].DecidedAt.IsZero() {
		t.Fatalf("expired decision record = %+v", decision)
	}
}

func TestDeploymentApprovalRedactsDecisionReasonInMemoryAndOnDisk(t *testing.T) {
	root := t.TempDir()
	store, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	requested, err := store.Request("org_a", "bld_a", "self", "staging", "manifest-a", "operator_a")
	if err != nil {
		t.Fatal(err)
	}
	secret := `{"api_key":"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn"}`
	decided, err := store.Decide(requested.ID, "org_a", "bld_a", "self", "admin_a", secret, requested.Nonce, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(decided.Reason, "ghp_") || !strings.Contains(decided.Reason, "[REDACTED]") {
		t.Fatalf("decided approval reason was not redacted: %q", decided.Reason)
	}
	data, err := os.ReadFile(filepath.Join(root, "approvals.json"))
	if err != nil || strings.Contains(string(data), "ghp_") {
		t.Fatalf("persisted approval exposed credential: err=%v", err)
	}

	var legacyRecords map[string]DeploymentApproval
	if err := json.Unmarshal(data, &legacyRecords); err != nil {
		t.Fatal(err)
	}
	legacyRecord := legacyRecords[decided.ID]
	legacyRecord.Reason = secret
	legacyRecords[decided.ID] = legacyRecord
	legacy, err := json.Marshal(legacyRecords)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "approvals.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewDeploymentApprovalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	listed := reloaded.ListForOrganization("org_a")
	if len(listed) != 1 || strings.Contains(listed[0].Reason, "ghp_") {
		t.Fatalf("legacy approval list exposed credential: %+v", listed)
	}
	migrated, err := os.ReadFile(filepath.Join(root, "approvals.json"))
	if err != nil || strings.Contains(string(migrated), "ghp_") {
		t.Fatalf("legacy approval was not scrubbed at rest: err=%v", err)
	}
}
