package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DeploymentApprovalPending  = "PENDING"
	DeploymentApprovalApproved = "APPROVED"
	DeploymentApprovalRejected = "REJECTED"
	DeploymentApprovalConsumed = "CONSUMED"
)

var (
	ErrDeploymentApprovalNotFound     = errors.New("deployment approval not found")
	ErrDeploymentApprovalNonce        = errors.New("deployment approval nonce mismatch")
	ErrDeploymentApprovalConflict     = errors.New("deployment approval is no longer pending")
	ErrDeploymentApprovalExpired      = errors.New("deployment approval expired")
	ErrDeploymentApprovalOrganization = errors.New("deployment approval is outside the active organization")
)

type DeploymentApproval struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	BuilderID      string    `json:"builder_id"`
	Provider       string    `json:"provider"`
	Target         string    `json:"target,omitempty"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	RequestedBy    string    `json:"requested_by"`
	DecidedBy      string    `json:"decided_by,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	Nonce          string    `json:"nonce"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	DecidedAt      time.Time `json:"decided_at,omitempty"`
	ConsumedAt     time.Time `json:"consumed_at,omitempty"`
}

type DeploymentApprovalStore struct {
	mu        sync.Mutex
	root      string
	approvals map[string]DeploymentApproval
}

func NewDeploymentApprovalStore(root string) (*DeploymentApprovalStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("deployment approval store root is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	store := &DeploymentApprovalStore{root: root, approvals: map[string]DeploymentApproval{}}
	data, err := os.ReadFile(filepath.Join(root, "approvals.json"))
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&store.approvals); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("deployment approval store contains trailing JSON")
		}
		return nil, err
	}
	if store.approvals == nil {
		store.approvals = map[string]DeploymentApproval{}
	}
	changed := false
	for id, approval := range store.approvals {
		if approval.ID != id {
			return nil, errors.New("deployment approval id does not match its key")
		}
		if err := validateDeploymentApprovalRecord(approval); err != nil {
			return nil, fmt.Errorf("deployment approval %q is invalid: %w", id, err)
		}
		safe := redactDeploymentApproval(approval)
		if safe.Reason != approval.Reason {
			store.approvals[id] = safe
			changed = true
		}
	}
	if changed {
		if err := store.persistLocked(); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func redactDeploymentApproval(approval DeploymentApproval) DeploymentApproval {
	approval.Reason = RedactDLP(approval.Reason)
	return approval
}

func validateDeploymentApprovalRecord(approval DeploymentApproval) error {
	if strings.TrimSpace(approval.RequestedBy) == "" {
		return errors.New("requesting actor is required")
	}
	switch approval.Status {
	case DeploymentApprovalPending:
		if strings.TrimSpace(approval.DecidedBy) != "" || !approval.DecidedAt.IsZero() {
			return errors.New("pending approval cannot contain decision metadata")
		}
	case DeploymentApprovalApproved, DeploymentApprovalRejected, DeploymentApprovalConsumed:
		if strings.TrimSpace(approval.DecidedBy) == "" || approval.DecidedAt.IsZero() {
			return errors.New("deciding actor and timestamp are required")
		}
		if approval.Status == DeploymentApprovalConsumed && approval.ConsumedAt.IsZero() {
			return errors.New("consumed approval timestamp is required")
		}
	default:
		return errors.New("approval status is invalid")
	}
	return nil
}

func (s *DeploymentApprovalStore) Request(organizationID, builderID, provider, target, manifestSHA256, actorID string) (DeploymentApproval, error) {
	if s == nil {
		return DeploymentApproval{}, errors.New("deployment approval store is unavailable")
	}
	organizationID = strings.TrimSpace(organizationID)
	builderID = strings.TrimSpace(builderID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	manifestSHA256 = strings.ToLower(strings.TrimSpace(manifestSHA256))
	actorID = strings.TrimSpace(actorID)
	if organizationID == "" || builderID == "" || provider == "" || manifestSHA256 == "" || actorID == "" {
		return DeploymentApproval{}, errors.New("organization, builder, provider, manifest hash and actor are required")
	}
	now := time.Now().UTC()
	approval := DeploymentApproval{ID: "dapr_" + uuid.NewString(), OrganizationID: organizationID, BuilderID: builderID, Provider: provider, Target: strings.TrimSpace(target), ManifestSHA256: manifestSHA256, RequestedBy: actorID, Nonce: uuid.NewString(), Status: DeploymentApprovalPending, CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute)}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[approval.ID] = approval
	if err := s.persistLocked(); err != nil {
		delete(s.approvals, approval.ID)
		return DeploymentApproval{}, err
	}
	return approval, nil
}

func (s *DeploymentApprovalStore) Decide(id, organizationID, builderID, provider, actorID, reason, nonce string, approved bool) (DeploymentApproval, error) {
	if s == nil {
		return DeploymentApproval{}, errors.New("deployment approval store is unavailable")
	}
	if len(reason) > 4096 {
		return DeploymentApproval{}, errors.New("deployment approval reason exceeds 4096 bytes")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return DeploymentApproval{}, errors.New("deployment approval actor is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, ok := s.approvals[strings.TrimSpace(id)]
	if !ok {
		return DeploymentApproval{}, ErrDeploymentApprovalNotFound
	}
	if approval.OrganizationID != strings.TrimSpace(organizationID) || approval.BuilderID != strings.TrimSpace(builderID) || approval.Provider != strings.ToLower(strings.TrimSpace(provider)) {
		return DeploymentApproval{}, ErrDeploymentApprovalOrganization
	}
	if strings.TrimSpace(nonce) == "" || nonce != approval.Nonce {
		return DeploymentApproval{}, ErrDeploymentApprovalNonce
	}
	if approval.Status != DeploymentApprovalPending {
		return DeploymentApproval{}, ErrDeploymentApprovalConflict
	}
	if err := validateDeploymentApprovalRecord(approval); err != nil {
		return DeploymentApproval{}, fmt.Errorf("deployment approval is invalid: %w", err)
	}
	if time.Now().UTC().After(approval.ExpiresAt) {
		previous := approval
		approval.Status = DeploymentApprovalRejected
		approval.DecidedBy = "system:expiry"
		approval.Reason = "approval expired"
		approval.DecidedAt = time.Now().UTC()
		s.approvals[approval.ID] = approval
		if err := s.persistLocked(); err != nil {
			s.approvals[approval.ID] = previous
			return DeploymentApproval{}, err
		}
		return DeploymentApproval{}, ErrDeploymentApprovalExpired
	}
	previous := approval
	approval.DecidedBy = actorID
	approval.Reason = RedactDLP(strings.TrimSpace(reason))
	approval.DecidedAt = time.Now().UTC()
	if approved {
		approval.Status = DeploymentApprovalApproved
	} else {
		approval.Status = DeploymentApprovalRejected
	}
	s.approvals[approval.ID] = approval
	if err := s.persistLocked(); err != nil {
		s.approvals[approval.ID] = previous
		return DeploymentApproval{}, err
	}
	return redactDeploymentApproval(approval), nil
}

func (s *DeploymentApprovalStore) Consume(id, organizationID, builderID, provider, target, manifestSHA256, nonce string) (DeploymentApproval, error) {
	if s == nil {
		return DeploymentApproval{}, errors.New("deployment approval store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, ok := s.approvals[strings.TrimSpace(id)]
	if !ok {
		return DeploymentApproval{}, ErrDeploymentApprovalNotFound
	}
	if approval.OrganizationID != strings.TrimSpace(organizationID) || approval.BuilderID != strings.TrimSpace(builderID) || approval.Provider != strings.ToLower(strings.TrimSpace(provider)) {
		return DeploymentApproval{}, ErrDeploymentApprovalOrganization
	}
	if approval.Target != strings.TrimSpace(target) {
		return DeploymentApproval{}, errors.New("deployment target does not match approval")
	}
	if approval.ManifestSHA256 != strings.ToLower(strings.TrimSpace(manifestSHA256)) {
		return DeploymentApproval{}, errors.New("deployment manifest does not match approval")
	}
	if strings.TrimSpace(nonce) == "" || nonce != approval.Nonce {
		return DeploymentApproval{}, ErrDeploymentApprovalNonce
	}
	if approval.Status != DeploymentApprovalApproved {
		return DeploymentApproval{}, ErrDeploymentApprovalConflict
	}
	if err := validateDeploymentApprovalRecord(approval); err != nil {
		return DeploymentApproval{}, fmt.Errorf("deployment approval is invalid: %w", err)
	}
	if time.Now().UTC().After(approval.ExpiresAt) {
		return DeploymentApproval{}, ErrDeploymentApprovalExpired
	}
	previous := approval
	approval.Status = DeploymentApprovalConsumed
	approval.ConsumedAt = time.Now().UTC()
	s.approvals[approval.ID] = approval
	if err := s.persistLocked(); err != nil {
		s.approvals[approval.ID] = previous
		return DeploymentApproval{}, err
	}
	return redactDeploymentApproval(approval), nil
}

func (s *DeploymentApprovalStore) ListForOrganization(organizationID string) []DeploymentApproval {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]DeploymentApproval, 0)
	for _, approval := range s.approvals {
		if approval.OrganizationID == strings.TrimSpace(organizationID) {
			result = append(result, redactDeploymentApproval(approval))
		}
	}
	return result
}

func (s *DeploymentApprovalStore) persistLocked() error {
	if s.root == "" {
		return nil
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "approvals.json"), s.approvals); err != nil {
		return fmt.Errorf("persist deployment approvals: %w", err)
	}
	return nil
}
