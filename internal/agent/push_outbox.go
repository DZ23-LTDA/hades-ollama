package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PushOutboxItem struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Title          string         `json:"title"`
	Body           string         `json:"body"`
	Data           map[string]any `json:"data,omitempty"`
	Attempts       int            `json:"attempts"`
	NextAttemptAt  time.Time      `json:"next_attempt_at"`
	LeaseUntil     *time.Time     `json:"lease_until,omitempty"`
	LeaseToken     string         `json:"lease_token,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
}

var (
	errPushOutboxLeaseLost = errors.New("push outbox lease is no longer current")
	errPushOutboxQuota     = errors.New("push outbox capacity limit reached")
)

const (
	maxPushOutboxAttempts       = 8
	maxPushOutboxItems          = 256
	maxPushOutboxItemsPerTenant = 64
	maxPushOutboxBatch          = 32
	maxPushOutboxFlushDuration  = 30 * time.Second
	maxPushOutboxItemBytes      = 64 << 10
	maxPushOutboxFileBytes      = 20 << 20
	pushOutboxTerminalRetention = 7 * 24 * time.Hour
)

type PushOutbox struct {
	mu    sync.Mutex
	path  string
	items map[string]PushOutboxItem
}

func NewPushOutbox(root string) (*PushOutbox, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("push outbox root is required")
	}
	var err error
	root, err = canonicalPushPersistentRoot(root)
	if err != nil {
		return nil, err
	}
	outbox := &PushOutbox{path: filepath.Join(root, "outbox.json"), items: map[string]PushOutboxItem{}}
	if err := withFileLock(outbox.lockPath(), func() error {
		migrated, err := outbox.refreshLocked()
		if err != nil {
			return err
		}
		if migrated {
			return outbox.persistLocked()
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return outbox, nil
}

func (o *PushOutbox) lockPath() string {
	return filepath.Join(filepath.Dir(o.path), ".outbox.lock")
}

// withFileStateLocked serializes persistent operations across processes and
// reloads the complete snapshot before applying each read or mutation.
// It acquires the process-local state mutex only after the file lock succeeds,
// so context-aware waiters never block cleanup while holding o.mu.
func (o *PushOutbox) withFileStateLocked(persist bool, run func() error) error {
	return o.withFileStateLockedContext(context.Background(), persist, run)
}

func (o *PushOutbox) withFileStateLockedContext(ctx context.Context, persist bool, run func() error) error {
	if ctx == nil {
		return errors.New("push outbox context is required")
	}
	return withFileLockContext(ctx, o.lockPath(), func() error {
		// Acquire the process-local state mutex only after the cross-process
		// lock. A blocked file-lock waiter can then never strand cancellation
		// cleanup behind o.mu.
		o.mu.Lock()
		defer o.mu.Unlock()
		previous := clonePushOutboxItems(o.items)
		migrated, err := o.refreshLocked()
		if err != nil {
			// Do not retain potentially sensitive in-memory contents after a
			// failed validation of the on-disk snapshot.
			o.items = map[string]PushOutboxItem{}
			return err
		}
		if run != nil {
			if err := run(); err != nil {
				o.items = previous
				return err
			}
		}
		if !persist && !migrated {
			return nil
		}
		if err := o.persistLocked(); err != nil {
			o.items = previous
			return err
		}
		return nil
	})
}

func (o *PushOutbox) Enqueue(organizationID, title, body string, data map[string]any) (PushOutboxItem, error) {
	if o == nil {
		return PushOutboxItem{}, errors.New("push outbox is unavailable")
	}
	organizationID = strings.TrimSpace(organizationID)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if organizationID == "" || title == "" || body == "" {
		return PushOutboxItem{}, errors.New("push outbox organization, title and body are required")
	}
	clonedData, err := clonePushDataChecked(data)
	if err != nil {
		return PushOutboxItem{}, errOutboundPayloadBlocked
	}
	normalized, err := normalizedOutboundPayload(map[string]any{"title": title, "body": body, "data": clonedData}, maxOutboundDLPScanBytes)
	if err != nil {
		return PushOutboxItem{}, err
	}
	payload := normalized.(map[string]any)
	clonedData, _ = payload["data"].(map[string]any)
	item := PushOutboxItem{
		ID:             "out_" + uuid.NewString(),
		OrganizationID: organizationID,
		Title:          title,
		Body:           body,
		Data:           clonedData,
		NextAttemptAt:  time.Now().UTC(),
	}
	if err := validatePushOutboxItemSize(item); err != nil {
		return PushOutboxItem{}, err
	}
	if err := o.withFileStateLocked(true, func() error {
		prunePushOutboxTerminals(o.items, time.Now().UTC())
		if len(o.items) >= maxPushOutboxItems {
			return errPushOutboxQuota
		}
		tenantItems := 0
		for _, existing := range o.items {
			if existing.OrganizationID == organizationID {
				tenantItems++
			}
		}
		if tenantItems >= maxPushOutboxItemsPerTenant {
			return errPushOutboxQuota
		}
		o.items[item.ID] = item
		return nil
	}); err != nil {
		return PushOutboxItem{}, err
	}
	return clonePushOutboxItem(item), nil
}

func (o *PushOutbox) ClaimDue(now time.Time) (PushOutboxItem, bool, error) {
	return o.ClaimDueContext(context.Background(), now)
}

func (o *PushOutbox) ClaimDueContext(ctx context.Context, now time.Time) (PushOutboxItem, bool, error) {
	if o == nil {
		return PushOutboxItem{}, false, errors.New("push outbox is unavailable")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var claimed PushOutboxItem
	found := false
	err := o.withFileStateLockedContext(ctx, true, func() error {
		prunePushOutboxTerminals(o.items, now)
		for id, item := range o.items {
			if item.NextAttemptAt.After(now) || (item.LeaseUntil != nil && item.LeaseUntil.After(now)) {
				continue
			}
			if item.Attempts >= maxPushOutboxAttempts {
				// A terminal record remains available for inspection but must not
				// starve other due notifications in this shared outbox.
				continue
			}
			leaseUntil := now.Add(time.Minute)
			item.Attempts++
			item.LeaseUntil = &leaseUntil
			item.LeaseToken = uuid.NewString()
			o.items[id] = item
			claimed = clonePushOutboxItem(item)
			found = true
			return nil
		}
		return nil
	})
	if err != nil || !found {
		return PushOutboxItem{}, false, err
	}
	return claimed, true, nil
}

func (o *PushOutbox) Complete(id, leaseToken string) error {
	return o.CompleteContext(context.Background(), id, leaseToken)
}

func (o *PushOutbox) CompleteContext(ctx context.Context, id, leaseToken string) error {
	if o == nil {
		return errors.New("push outbox is unavailable")
	}
	id = strings.TrimSpace(id)
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return o.withFileStateLockedContext(cleanupCtx, true, func() error {
		item, ok := o.items[id]
		if !ok {
			return errors.New("push outbox item not found")
		}
		if leaseToken == "" || item.LeaseToken != leaseToken || item.LeaseUntil == nil || !item.LeaseUntil.After(time.Now().UTC()) {
			return errPushOutboxLeaseLost
		}
		delete(o.items, id)
		return nil
	})
}

func (o *PushOutbox) Fail(id, leaseToken string, cause error) error {
	return o.FailContext(context.Background(), id, leaseToken, cause)
}

func (o *PushOutbox) FailContext(ctx context.Context, id, leaseToken string, cause error) error {
	// Lease cleanup must outlive the delivery context: after ClaimDue persists a
	// lease, the caller may cancel that context before it can clear the lease.
	// Keep cleanup bounded so shutdown cannot wait indefinitely on a lock.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return o.failAtContext(cleanupCtx, id, leaseToken, cause, time.Now().UTC())
}

// failAt keeps deterministic clock control private to the package so external
// callers cannot pass a stale timestamp to bypass lease expiry fencing.
func (o *PushOutbox) failAt(id, leaseToken string, cause error, now time.Time) error {
	return o.failAtContext(context.Background(), id, leaseToken, cause, now)
}

func (o *PushOutbox) failAtContext(ctx context.Context, id, leaseToken string, cause error, now time.Time) error {
	if o == nil {
		return errors.New("push outbox is unavailable")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	id = strings.TrimSpace(id)
	return o.withFileStateLockedContext(ctx, true, func() error {
		item, ok := o.items[id]
		if !ok {
			return errors.New("push outbox item not found")
		}
		if leaseToken == "" || item.LeaseToken != leaseToken || item.LeaseUntil == nil || !item.LeaseUntil.After(now) {
			return errPushOutboxLeaseLost
		}
		if item.Attempts < 0 || item.Attempts > maxPushOutboxAttempts {
			return errors.New("push outbox retry metadata is invalid")
		}
		item.LeaseUntil = nil
		item.LeaseToken = ""
		item.LastError = RedactDLP(limitError(errorString(cause), 800))
		backoff := time.Duration(1<<minInt(item.Attempts, 6)) * time.Second
		item.NextAttemptAt = now.Add(backoff)
		o.items[id] = item
		return nil
	})
}

func (o *PushOutbox) List() []PushOutboxItem {
	if o == nil {
		return nil
	}
	var items []PushOutboxItem
	if err := o.withFileStateLocked(false, func() error {
		items = make([]PushOutboxItem, 0, len(o.items))
		for _, item := range o.items {
			items = append(items, clonePushOutboxItem(item))
		}
		return nil
	}); err != nil {
		return nil
	}
	return items
}

func (o *PushOutbox) persistLocked() error {
	if err := validatePushOutboxCapacity(o.items); err != nil {
		return err
	}
	stored := make(map[string]PushOutboxItem, len(o.items))
	for id, item := range o.items {
		if id == "" || item.ID != id {
			return errors.New("push outbox item id is inconsistent")
		}
		normalized, err := normalizePushOutboxItem(item)
		if err != nil {
			return err
		}
		stored[id] = normalized
	}
	encoded, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if len(encoded)+1 > maxPushOutboxFileBytes {
		return errPushOutboxQuota
	}
	return writeJSONAtomic(o.path, stored)
}

// refreshLocked validates the entire snapshot before replacing memory. A
// sensitive legacy record therefore causes startup or the current operation
// to fail closed; the original file is left intact for operator recovery.
func (o *PushOutbox) refreshLocked() (bool, error) {
	var items map[string]PushOutboxItem
	if err := readPushOutboxJSON(o.path, &items); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			o.items = map[string]PushOutboxItem{}
			return false, nil
		}
		return false, err
	}
	if items == nil {
		items = map[string]PushOutboxItem{}
	}
	normalizedItems := make(map[string]PushOutboxItem, len(items))
	migrated := false
	for id, item := range items {
		if strings.TrimSpace(id) == "" || item.ID != id {
			return false, fmt.Errorf("push outbox item %s has mismatched id", id)
		}
		normalized, err := normalizePushOutboxItem(item)
		if err != nil {
			return false, fmt.Errorf("validate push outbox item %s: %w", id, err)
		}
		if !reflect.DeepEqual(item, normalized) {
			migrated = true
		}
		normalizedItems[id] = normalized
	}
	if err := validatePushOutboxCapacity(normalizedItems); err != nil {
		return false, err
	}
	o.items = normalizedItems
	return migrated, nil
}

func readPushOutboxJSON(path string, target any) error {
	before, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return errors.New("push outbox snapshot is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	after, err := os.Lstat(path)
	if err != nil || !after.Mode().IsRegular() || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, opened) || !os.SameFile(opened, after) {
		return errors.New("push outbox snapshot changed during open")
	}
	if opened.Size() > maxPushOutboxFileBytes {
		return errPushOutboxQuota
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPushOutboxFileBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxPushOutboxFileBytes {
		return errPushOutboxQuota
	}
	return json.Unmarshal(data, target)
}

func normalizePushOutboxItem(item PushOutboxItem) (PushOutboxItem, error) {
	if strings.TrimSpace(item.OrganizationID) == "" || len(item.OrganizationID) > 256 {
		return PushOutboxItem{}, errors.New("push outbox organization identity is invalid")
	}
	if item.Attempts < 0 || item.Attempts > maxPushOutboxAttempts {
		return PushOutboxItem{}, errors.New("push outbox retry metadata is invalid")
	}
	_, err := normalizedOutboundPayload(map[string]any{
		"title":      item.Title,
		"body":       item.Body,
		"data":       item.Data,
		"last_error": item.LastError,
	}, maxOutboundDLPScanBytes)
	if err != nil {
		return PushOutboxItem{}, err
	}
	if err := validatePushOutboxItemSize(item); err != nil {
		return PushOutboxItem{}, err
	}
	return item, nil
}

func validatePushOutboxItemSize(item PushOutboxItem) error {
	encoded, err := json.Marshal(item)
	if err != nil {
		return errOutboundPayloadBlocked
	}
	if len(encoded) > maxPushOutboxItemBytes {
		return errOutboundPayloadLimit
	}
	return nil
}

func validatePushOutboxCapacity(items map[string]PushOutboxItem) error {
	if len(items) > maxPushOutboxItems {
		return errPushOutboxQuota
	}
	counts := make(map[string]int)
	for _, item := range items {
		counts[item.OrganizationID]++
		if counts[item.OrganizationID] > maxPushOutboxItemsPerTenant {
			return errPushOutboxQuota
		}
	}
	return nil
}

func prunePushOutboxTerminals(items map[string]PushOutboxItem, now time.Time) {
	for id, item := range items {
		if item.LeaseUntil != nil && item.LeaseUntil.After(now) {
			continue
		}
		if item.Attempts >= maxPushOutboxAttempts && !item.NextAttemptAt.Add(pushOutboxTerminalRetention).After(now) {
			delete(items, id)
		}
	}
}

func clonePushOutboxItem(item PushOutboxItem) PushOutboxItem {
	item.Data = clonePushData(item.Data)
	item.LeaseUntil = clonePushTime(item.LeaseUntil)
	return item
}

func clonePushOutboxItems(items map[string]PushOutboxItem) map[string]PushOutboxItem {
	cloned := make(map[string]PushOutboxItem, len(items))
	for id, item := range items {
		cloned[id] = clonePushOutboxItem(item)
	}
	return cloned
}

func clonePushData(data map[string]any) map[string]any {
	cloned, _ := clonePushDataChecked(data)
	return cloned
}

func clonePushDataChecked(data map[string]any) (map[string]any, error) {
	if data == nil {
		return nil, nil
	}
	budget := outboundShapeBudget{nodes: maxOutboundDLPNodes, bytes: maxOutboundDLPScanBytes, maxString: maxOutboundDLPScanBytes}
	if err := validateOutboundShape(reflect.ValueOf(data), 0, &budget); err != nil {
		return nil, errOutboundPayloadBlocked
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func errorString(err error) string {
	if err == nil {
		return "unknown push delivery error"
	}
	return fmt.Sprintf("%v", err)
}
