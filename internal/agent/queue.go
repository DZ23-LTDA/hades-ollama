package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type QueueStatus string

const (
	QueuePending    QueueStatus = "pending"
	QueueRunning    QueueStatus = "running"
	QueueSucceeded  QueueStatus = "succeeded"
	QueueFailed     QueueStatus = "failed"
	QueueDeadLetter QueueStatus = "dead_letter"
)

var ErrQueueLeaseLost = errors.New("queue lease is no longer owned by this worker")

const queueHandlerDrainTimeout = 30 * time.Second

type QueueJob struct {
	ID                   string      `json:"id"`
	MissionID            string      `json:"mission_id"`
	OrganizationID       string      `json:"organization_id,omitempty"`
	Status               QueueStatus `json:"status"`
	Attempts             int         `json:"attempts"`
	MaxAttempts          int         `json:"max_attempts"`
	WorkerID             string      `json:"worker_id,omitempty"`
	LeaseToken           string      `json:"lease_token,omitempty"`
	LeaseUntil           time.Time   `json:"lease_until,omitempty"`
	LeaseUntilUnixMilli  int64       `json:"lease_until_ms,omitempty"`
	AvailableAt          time.Time   `json:"available_at"`
	AvailableAtUnixMilli int64       `json:"available_at_ms,omitempty"`
	LockedAt             *time.Time  `json:"locked_at,omitempty"`
	LockedAtUnixMilli    int64       `json:"locked_at_ms,omitempty"`
	LastError            string      `json:"last_error,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
	CreatedAtUnixMilli   int64       `json:"created_at_ms,omitempty"`
	UpdatedAt            time.Time   `json:"updated_at"`
	UpdatedAtUnixMilli   int64       `json:"updated_at_ms,omitempty"`
}

type JobQueue struct {
	mu            sync.Mutex
	root          string
	jobs          map[string]QueueJob
	notify        chan struct{}
	started       bool
	leaseDuration time.Duration
}

const (
	queueLeaseDuration = 15 * time.Minute
	maxQueueSnapshot   = 20 << 20
	maxQueueJobRecords = 100_000
)

func setQueueLeaseUntil(job *QueueJob, until time.Time) {
	job.LeaseUntil = until
	if until.IsZero() {
		job.LeaseUntilUnixMilli = 0
		return
	}
	job.LeaseUntilUnixMilli = until.UnixMilli()
}

func setQueueAvailableAt(job *QueueJob, at time.Time) {
	job.AvailableAt = at
	if at.IsZero() {
		job.AvailableAtUnixMilli = 0
		return
	}
	job.AvailableAtUnixMilli = at.UnixMilli()
}

func setQueueLockedAt(job *QueueJob, at time.Time) {
	if at.IsZero() {
		job.LockedAt = nil
		job.LockedAtUnixMilli = 0
		return
	}
	locked := at
	job.LockedAt = &locked
	job.LockedAtUnixMilli = at.UnixMilli()
}

func setQueueUpdatedAt(job *QueueJob, at time.Time) {
	job.UpdatedAt = at
	if at.IsZero() {
		job.UpdatedAtUnixMilli = 0
		return
	}
	job.UpdatedAtUnixMilli = at.UnixMilli()
}

func setQueueCreatedAt(job *QueueJob, at time.Time) {
	job.CreatedAt = at
	if at.IsZero() {
		job.CreatedAtUnixMilli = 0
		return
	}
	job.CreatedAtUnixMilli = at.UnixMilli()
}

func NewJobQueue(root string) (*JobQueue, error) {
	queue := &JobQueue{root: strings.TrimSpace(root), jobs: map[string]QueueJob{}, notify: make(chan struct{}, 1), leaseDuration: queueLeaseDuration}
	if queue.root == "" {
		return queue, nil
	}
	if err := os.MkdirAll(queue.root, 0o700); err != nil {
		return nil, err
	}
	if err := withFileLock(queue.lockPath(), func() error {
		return queue.refreshLocked()
	}); err != nil {
		return nil, err
	}
	return queue, nil
}

func (q *JobQueue) lockPath() string {
	return filepath.Join(q.root, ".jobs.lock")
}

// withFileStateLocked serializes persistent queue operations across processes
// and refreshes the complete snapshot before applying an operation.
func (q *JobQueue) withFileStateLocked(run func() error) error {
	if q.root == "" {
		return run()
	}
	// A temporary root can be replaced while a macOS race suite is draining
	// background work. Recreate only this queue-owned directory before opening
	// its lock; never recreate or follow a caller-owned parent.
	if err := os.MkdirAll(q.root, 0o700); err != nil {
		return err
	}
	if info, err := os.Lstat(q.root); err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("queue state root must be a real directory")
	}
	return withFileLock(q.lockPath(), func() error {
		if err := q.refreshLocked(); err != nil {
			return err
		}
		return run()
	})
}

func (q *JobQueue) refreshLocked() error {
	if q.root == "" {
		return nil
	}
	var jobs map[string]QueueJob
	info, err := os.Stat(filepath.Join(q.root, "jobs.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && info.Size() > maxQueueSnapshot {
		return errors.New("queue snapshot exceeds size limit")
	}
	if err := readJSON(filepath.Join(q.root, "jobs.json"), &jobs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			q.jobs = make(map[string]QueueJob)
			return nil
		}
		return err
	}
	if jobs == nil {
		jobs = make(map[string]QueueJob)
	}
	if len(jobs) > maxQueueJobRecords {
		return errors.New("queue snapshot exceeds record limit")
	}
	for id, job := range jobs {
		if err := validateQueueJobRecord(id, job); err != nil {
			return fmt.Errorf("invalid persisted queue job %q: %w", id, err)
		}
	}
	q.jobs = jobs
	return nil
}

func (q *JobQueue) Enqueue(missionID string, maxAttempts int) (QueueJob, error) {
	return q.EnqueueForOrganization("", missionID, maxAttempts)
}

func (q *JobQueue) EnqueueForOrganization(organizationID, missionID string, maxAttempts int) (QueueJob, error) {
	organizationID = strings.TrimSpace(organizationID)
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return QueueJob{}, errors.New("mission id is required")
	}
	if len(organizationID) > 256 {
		return QueueJob{}, errors.New("organization id is invalid")
	}
	if maxAttempts <= 0 || maxAttempts > 20 {
		maxAttempts = 3
	}
	now := time.Now().UTC()
	job := QueueJob{ID: "job_" + uuid.NewString(), MissionID: missionID, OrganizationID: organizationID, Status: QueuePending, MaxAttempts: maxAttempts}
	setQueueCreatedAt(&job, now)
	setQueueAvailableAt(&job, now)
	setQueueUpdatedAt(&job, now)
	q.mu.Lock()
	defer q.mu.Unlock()
	var result QueueJob
	err := q.withFileStateLocked(func() error {
		for id, existing := range q.jobs {
			if existing.MissionID == missionID && (existing.Status == QueuePending || existing.Status == QueueRunning) {
				if existing.OrganizationID != organizationID {
					if existing.OrganizationID != "" || organizationID == "" {
						return ErrQueueJobForbidden
					}
					existing.OrganizationID = organizationID
					setQueueUpdatedAt(&existing, now)
					if err := q.persistLocked(existing); err != nil {
						return err
					}
					q.jobs[id] = existing
				}
				result = existing
				return nil
			}
		}
		if err := q.persistLocked(job); err != nil {
			return err
		}
		result = job
		return nil
	})
	if err != nil {
		return QueueJob{}, err
	}
	q.signal()
	return result, nil
}

func (q *JobQueue) Claim(workerID string, now time.Time) (QueueJob, bool, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return QueueJob{}, false, errors.New("worker id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	var candidate QueueJob
	found := false
	err := q.withFileStateLocked(func() error {
		for _, job := range q.jobs {
			if job.Status == QueueRunning && !job.LeaseUntil.IsZero() && !job.LeaseUntil.After(now) {
				if job.Attempts >= job.MaxAttempts {
					job.Status = QueueDeadLetter
					job.WorkerID = ""
					job.LeaseToken = ""
					setQueueLockedAt(&job, time.Time{})
					setQueueLeaseUntil(&job, time.Time{})
					job.LastError = "worker lease expired after max attempts"
					setQueueUpdatedAt(&job, now)
					if err := q.persistLocked(job); err != nil {
						return err
					}
					continue
				}
				job.Status = QueuePending
				job.WorkerID = ""
				job.LeaseToken = ""
				setQueueLockedAt(&job, time.Time{})
				setQueueLeaseUntil(&job, time.Time{})
				setQueueAvailableAt(&job, now)
				setQueueUpdatedAt(&job, now)
				if err := q.persistLocked(job); err != nil {
					return err
				}
			}
			if job.Status != QueuePending || job.AvailableAt.After(now) {
				continue
			}
			if !found || job.AvailableAt.Before(candidate.AvailableAt) || (job.AvailableAt.Equal(candidate.AvailableAt) && job.CreatedAt.Before(candidate.CreatedAt)) {
				candidate, found = job, true
			}
		}
		if !found {
			return nil
		}
		candidate.Status = QueueRunning
		candidate.Attempts++
		candidate.WorkerID = workerID
		candidate.LeaseToken = uuid.NewString()
		locked := now
		setQueueLockedAt(&candidate, locked)
		leaseDuration := q.leaseDuration
		if leaseDuration <= 0 {
			leaseDuration = queueLeaseDuration
		}
		setQueueLeaseUntil(&candidate, now.Add(leaseDuration))
		setQueueUpdatedAt(&candidate, now)
		return q.persistLocked(candidate)
	})
	if err != nil {
		return QueueJob{}, false, err
	}
	return candidate, found, nil
}

func (q *JobQueue) Heartbeat(claim QueueJob, now time.Time) error {
	if claim.ID == "" || claim.WorkerID == "" || claim.LeaseToken == "" {
		return ErrQueueLeaseLost
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.withFileStateLocked(func() error {
		job, ok := q.jobs[claim.ID]
		if !ok || job.Status != QueueRunning || job.WorkerID != claim.WorkerID || job.LeaseToken != claim.LeaseToken {
			return ErrQueueLeaseLost
		}
		if job.LeaseUntil.IsZero() || !job.LeaseUntil.After(now) {
			return ErrQueueLeaseLost
		}
		leaseDuration := q.leaseDuration
		if leaseDuration <= 0 {
			leaseDuration = queueLeaseDuration
		}
		setQueueLeaseUntil(&job, now.Add(leaseDuration))
		setQueueUpdatedAt(&job, now)
		return q.persistLocked(job)
	})
}

func (q *JobQueue) Ack(claim QueueJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now().UTC()
	return q.withFileStateLocked(func() error {
		job, ok := q.jobs[claim.ID]
		if !ok {
			return os.ErrNotExist
		}
		if job.Status != QueueRunning || job.WorkerID != claim.WorkerID || job.LeaseToken == "" || job.LeaseToken != claim.LeaseToken {
			return ErrQueueLeaseLost
		}
		if job.LeaseUntil.IsZero() || !job.LeaseUntil.After(now) {
			return ErrQueueLeaseLost
		}
		job.Status = QueueSucceeded
		job.WorkerID = ""
		job.LeaseToken = ""
		setQueueLockedAt(&job, time.Time{})
		setQueueLeaseUntil(&job, time.Time{})
		setQueueUpdatedAt(&job, now)
		return q.persistLocked(job)
	})
}

func (q *JobQueue) Nack(claim QueueJob, runErr error) (QueueJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now().UTC()
	var result QueueJob
	err := q.withFileStateLocked(func() error {
		job, ok := q.jobs[claim.ID]
		if !ok {
			return os.ErrNotExist
		}
		if job.Status != QueueRunning || job.WorkerID != claim.WorkerID || job.LeaseToken == "" || job.LeaseToken != claim.LeaseToken {
			return ErrQueueLeaseLost
		}
		if job.LeaseUntil.IsZero() || !job.LeaseUntil.After(now) {
			return ErrQueueLeaseLost
		}
		if runErr != nil {
			job.LastError = limitError(runErr.Error(), 2000)
		}
		job.WorkerID = ""
		job.LeaseToken = ""
		setQueueLockedAt(&job, time.Time{})
		setQueueLeaseUntil(&job, time.Time{})
		setQueueUpdatedAt(&job, now)
		if errors.Is(runErr, ErrQueueNonRetryable) {
			job.Status = QueueFailed
			result = job
			return q.persistLocked(job)
		}
		if job.Attempts >= job.MaxAttempts {
			job.Status = QueueDeadLetter
			result = job
			return q.persistLocked(job)
		}
		job.Status = QueuePending
		backoff := time.Duration(1<<(job.Attempts-1)) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		setQueueAvailableAt(&job, now.Add(backoff))
		result = job
		return q.persistLocked(job)
	})
	return result, err
}

func (q *JobQueue) Replay(jobID string) (QueueJob, error) {
	return q.ReplayForOrganization("", jobID)
}

func (q *JobQueue) ReplayForOrganization(organizationID, jobID string) (QueueJob, error) {
	organizationID = strings.TrimSpace(organizationID)
	q.mu.Lock()
	defer q.mu.Unlock()
	var result QueueJob
	err := q.withFileStateLocked(func() error {
		job, ok := q.jobs[jobID]
		if !ok {
			return os.ErrNotExist
		}
		if job.Status != QueueDeadLetter {
			return fmt.Errorf("job %s is not replayable", jobID)
		}
		if job.OrganizationID != organizationID {
			return ErrQueueJobForbidden
		}
		job.Status = QueuePending
		job.Attempts = 0
		job.LastError = ""
		job.WorkerID = ""
		job.LeaseToken = ""
		setQueueLockedAt(&job, time.Time{})
		setQueueLeaseUntil(&job, time.Time{})
		setQueueAvailableAt(&job, time.Now().UTC())
		setQueueUpdatedAt(&job, time.Now().UTC())
		if err := q.persistLocked(job); err != nil {
			return err
		}
		result = job
		return nil
	})
	if err != nil {
		return QueueJob{}, err
	}
	q.signal()
	return result, nil
}

func (q *JobQueue) List(status QueueStatus) []QueueJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	var result []QueueJob
	if err := q.withFileStateLocked(func() error {
		result = make([]QueueJob, 0, len(q.jobs))
		for _, job := range q.jobs {
			if status == "" || job.Status == status {
				result = append(result, job)
			}
		}
		sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
		return nil
	}); err != nil {
		slog.Error("agent queue state refresh failed", "error", err)
		return nil
	}
	return result
}

func (q *JobQueue) Start(ctx context.Context, workerID string, handler func(context.Context, QueueJob) error) error {
	if q == nil {
		return errors.New("queue is unavailable")
	}
	if strings.TrimSpace(workerID) == "" {
		return errors.New("worker id is required")
	}
	if handler == nil {
		return errors.New("queue handler is required")
	}
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return nil
	}
	q.started = true
	q.mu.Unlock()
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			job, ok, err := q.Claim(workerID, time.Now().UTC())
			if err == nil && ok {
				runErr := queueHandlerResult(ctx, q.runWithHeartbeat(ctx, job, handler))
				if runErr != nil {
					if _, nackErr := q.Nack(job, runErr); nackErr != nil {
						slog.Error("agent queue NACK failed", "job_id", job.ID, "error", nackErr)
					}
				} else {
					if ackErr := q.Ack(job); ackErr != nil {
						slog.Error("agent queue ACK failed", "job_id", job.ID, "error", ackErr)
					}
				}
				continue
			}
			if ctx.Err() != nil {
				return
			}
			timer := time.NewTimer(500 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-q.notify:
				timer.Stop()
			case <-timer.C:
			}
		}
	}()
	return nil
}

func (q *JobQueue) runWithHeartbeat(ctx context.Context, claim QueueJob, handler func(context.Context, QueueJob) error) error {
	if handler == nil {
		return errors.New("queue handler is required")
	}
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runQueueHandler(handlerCtx, claim, handler) }()
	leaseDuration := q.leaseDuration
	if leaseDuration <= 0 {
		leaseDuration = queueLeaseDuration
	}
	interval := leaseDuration / 3
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return queueHandlerResult(ctx, err)
		case <-ctx.Done():
			return drainQueueHandler(done, cancel, func() error { return q.Heartbeat(claim, time.Now().UTC()) }, interval, ctx.Err())
		case <-ticker.C:
			if err := q.Heartbeat(claim, time.Now().UTC()); err != nil {
				return drainQueueHandler(done, cancel, func() error { return q.Heartbeat(claim, time.Now().UTC()) }, interval, fmt.Errorf("%w: renew local queue lease: %v", ErrQueueLeaseLost, err))
			}
		}
	}
}

func runQueueHandler(ctx context.Context, claim QueueJob, handler func(context.Context, QueueJob) error) (result error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = errors.Join(ErrQueueNonRetryable, fmt.Errorf("queue handler panicked for job %s: %v", claim.ID, recovered))
		}
	}()
	return handler(ctx, claim)
}

// drainQueueHandler does not let a worker release or retry a job while its handler
// may still be producing side effects. It keeps renewing an owned lease while
// cancellation propagates and returns a terminal error if the handler won't stop.
func drainQueueHandler(done <-chan error, cancel context.CancelFunc, heartbeat func() error, interval time.Duration, cause error) error {
	cancel()
	if heartbeat != nil {
		if err := heartbeat(); err != nil && !errors.Is(cause, ErrQueueLeaseLost) {
			cause = errors.Join(cause, fmt.Errorf("lease heartbeat on handler drain: %w", err))
		}
	}
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(queueHandlerDrainTimeout)
	defer timer.Stop()
	for {
		select {
		case <-done:
			return errors.Join(ErrQueueNonRetryable, cause)
		case <-ticker.C:
			if heartbeat != nil {
				if err := heartbeat(); err != nil {
					if !errors.Is(cause, ErrQueueLeaseLost) {
						cause = errors.Join(cause, fmt.Errorf("lease heartbeat while draining handler: %w", err))
					}
				}
			}
		case <-timer.C:
			return errors.Join(ErrQueueNonRetryable, cause, fmt.Errorf("queue handler did not stop within %s", queueHandlerDrainTimeout))
		}
	}
}

func queueHandlerResult(ctx context.Context, result error) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(ErrQueueNonRetryable, ctx.Err(), result)
	}
	return result
}

func (q *JobQueue) persistLocked(job QueueJob) error {
	previous, existed := q.jobs[job.ID]
	q.jobs[job.ID] = job
	if q.root == "" {
		return nil
	}
	rollback := func() {
		if existed {
			q.jobs[job.ID] = previous
		} else {
			delete(q.jobs, job.ID)
		}
	}
	if len(q.jobs) > maxQueueJobRecords {
		rollback()
		return errors.New("queue snapshot exceeds record limit")
	}
	for id, item := range q.jobs {
		if err := validateQueueJobRecord(id, item); err != nil {
			rollback()
			return fmt.Errorf("invalid queue job %q: %w", id, err)
		}
	}
	encoded, err := json.MarshalIndent(q.jobs, "", "  ")
	if err != nil {
		rollback()
		return err
	}
	if len(encoded)+1 > maxQueueSnapshot {
		rollback()
		return errors.New("queue snapshot exceeds size limit")
	}
	if err := writeJSONAtomic(filepath.Join(q.root, "jobs.json"), q.jobs); err != nil {
		rollback()
		return err
	}
	return nil
}

func validateQueueJobRecord(id string, job QueueJob) error {
	if strings.TrimSpace(id) == "" || job.ID != id || strings.TrimSpace(job.MissionID) == "" || len(job.OrganizationID) > 256 {
		return errors.New("queue identity fields are invalid")
	}
	if job.Attempts < 0 || job.MaxAttempts < 1 || job.MaxAttempts > 20 || job.Attempts > job.MaxAttempts {
		return errors.New("queue retry metadata is invalid")
	}
	switch job.Status {
	case QueuePending, QueueRunning, QueueSucceeded, QueueFailed, QueueDeadLetter:
	default:
		return errors.New("queue status is invalid")
	}
	if job.Status == QueuePending && job.Attempts >= job.MaxAttempts {
		return errors.New("pending queue job has exhausted its attempts")
	}
	if job.Status == QueueRunning && (job.Attempts < 1 || strings.TrimSpace(job.WorkerID) == "" || strings.TrimSpace(job.LeaseToken) == "" || job.LeaseUntil.IsZero()) {
		return errors.New("running queue job has invalid lease metadata")
	}
	return nil
}

func (q *JobQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func limitError(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
