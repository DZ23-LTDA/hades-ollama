package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrWebhookReplay           = errors.New("webhook idempotency key was already consumed")
	ErrWebhookReplayInProgress = errors.New("webhook idempotency key is already being processed")
	ErrWebhookReplayCapacity   = errors.New("webhook idempotency store is at capacity")
)

const (
	webhookReplayRetention = 30 * 24 * time.Hour
	webhookPendingLease    = 5 * time.Minute
	webhookReplayMaxClaims = 100_000
)

type webhookReplayState string

const (
	webhookReplayPending  webhookReplayState = "pending"
	webhookReplayAccepted webhookReplayState = "accepted"
)

type webhookReplayClaim struct {
	State     webhookReplayState `json:"state"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type WebhookReplayStore struct {
	mu     sync.Mutex
	root   string
	claims map[string]webhookReplayClaim
}

func NewWebhookReplayStore(root string) (*WebhookReplayStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("webhook replay store root is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	store := &WebhookReplayStore{root: root, claims: map[string]webhookReplayClaim{}}
	if err := store.withDiskLock(func() error { return store.loadLocked() }); err != nil {
		return nil, err
	}
	return store, nil
}

// Begin atomically claims an event across processes sharing this durable root.
// Unaccepted claims have a short lease so a crashed handler can be retried;
// accepted claims are retained for a bounded replay window.
func (s *WebhookReplayStore) Begin(scheduleID, idempotencyKey string) error {
	if s == nil {
		return errors.New("webhook replay store is unavailable")
	}
	key, err := webhookReplayKey(scheduleID, idempotencyKey)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withDiskLock(func() error {
		if err := s.loadLocked(); err != nil {
			return err
		}
		now := time.Now().UTC()
		s.pruneLocked(now)
		if claim, ok := s.claims[key]; ok {
			if claim.State == webhookReplayAccepted {
				if err := s.persistLocked(); err != nil {
					return fmt.Errorf("persist webhook replay cleanup: %w", err)
				}
				return ErrWebhookReplay
			}
			if now.Sub(claim.UpdatedAt) < webhookPendingLease {
				return ErrWebhookReplayInProgress
			}
		}
		if _, exists := s.claims[key]; !exists && len(s.claims) >= webhookReplayMaxClaims {
			if err := s.persistLocked(); err != nil {
				return fmt.Errorf("persist webhook replay cleanup: %w", err)
			}
			return ErrWebhookReplayCapacity
		}
		s.claims[key] = webhookReplayClaim{State: webhookReplayPending, UpdatedAt: now}
		if err := s.persistLocked(); err != nil {
			return fmt.Errorf("persist webhook idempotency key: %w", err)
		}
		return nil
	})
}

// Complete marks a successfully created mission as accepted and protected from
// replay for the retention window.
func (s *WebhookReplayStore) Complete(scheduleID, idempotencyKey string) error {
	if s == nil {
		return errors.New("webhook replay store is unavailable")
	}
	key, err := webhookReplayKey(scheduleID, idempotencyKey)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withDiskLock(func() error {
		if err := s.loadLocked(); err != nil {
			return err
		}
		claim, ok := s.claims[key]
		if !ok {
			return errors.New("webhook idempotency claim is missing")
		}
		claim.State = webhookReplayAccepted
		claim.UpdatedAt = time.Now().UTC()
		s.claims[key] = claim
		if err := s.persistLocked(); err != nil {
			return fmt.Errorf("persist accepted webhook event: %w", err)
		}
		return nil
	})
}

// Release frees only a pending claim after a failure known to have occurred
// before mission persistence. Accepted claims cannot be rolled back.
func (s *WebhookReplayStore) Release(scheduleID, idempotencyKey string) error {
	if s == nil {
		return errors.New("webhook replay store is unavailable")
	}
	key, err := webhookReplayKey(scheduleID, idempotencyKey)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withDiskLock(func() error {
		if err := s.loadLocked(); err != nil {
			return err
		}
		if claim, ok := s.claims[key]; ok && claim.State == webhookReplayPending {
			delete(s.claims, key)
			if err := s.persistLocked(); err != nil {
				return fmt.Errorf("release webhook idempotency key: %w", err)
			}
		}
		return nil
	})
}

// Claim preserves the one-step API for existing internal callers and tests.
func (s *WebhookReplayStore) Claim(scheduleID, idempotencyKey string) error {
	if err := s.Begin(scheduleID, idempotencyKey); err != nil {
		return err
	}
	if err := s.Complete(scheduleID, idempotencyKey); err != nil {
		_ = s.Release(scheduleID, idempotencyKey)
		return err
	}
	return nil
}

// MissionID deterministically maps a schedule/event pair to an opaque, safe ID.
// This lets a retry recover the already-persisted mission after an ambiguous
// commit or a process crash without creating duplicate side effects.
func (s *WebhookReplayStore) MissionID(scheduleID, idempotencyKey string) (string, error) {
	key, err := webhookReplayKey(scheduleID, idempotencyKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(key))
	return "mis_webhook_" + hex.EncodeToString(sum[:]), nil
}

func webhookReplayKey(scheduleID, idempotencyKey string) (string, error) {
	scheduleID = strings.TrimSpace(scheduleID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if scheduleID == "" || idempotencyKey == "" || len(idempotencyKey) > 200 {
		return "", errors.New("schedule id and idempotency key are required; key must be <= 200 characters")
	}
	return scheduleID + "\x00" + idempotencyKey, nil
}

func (s *WebhookReplayStore) withDiskLock(run func() error) error {
	return withFileLock(filepath.Join(s.root, ".claims.lock"), run)
}

func (s *WebhookReplayStore) loadLocked() error {
	data, err := os.ReadFile(filepath.Join(s.root, "claims.json"))
	if errors.Is(err, os.ErrNotExist) {
		s.claims = map[string]webhookReplayClaim{}
		return nil
	}
	if err != nil {
		return err
	}
	var current map[string]webhookReplayClaim
	if err := json.Unmarshal(data, &current); err == nil && current != nil {
		s.claims = current
		return nil
	}
	// Preserve stores written by the earlier format `{key: timestamp}`.
	var legacy map[string]time.Time
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	s.claims = make(map[string]webhookReplayClaim, len(legacy))
	for key, timestamp := range legacy {
		s.claims[key] = webhookReplayClaim{State: webhookReplayAccepted, UpdatedAt: timestamp}
	}
	return nil
}

func (s *WebhookReplayStore) pruneLocked(now time.Time) {
	for key, claim := range s.claims {
		if claim.UpdatedAt.IsZero() || now.Sub(claim.UpdatedAt) > webhookReplayRetention ||
			(claim.State == webhookReplayPending && now.Sub(claim.UpdatedAt) >= webhookPendingLease) {
			delete(s.claims, key)
		}
	}
}

func (s *WebhookReplayStore) persistLocked() error {
	return writeJSONAtomic(filepath.Join(s.root, "claims.json"), s.claims)
}
