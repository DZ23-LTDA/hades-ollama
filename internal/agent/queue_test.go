package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNackAppliesJitteredBackoff(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}

	const base = time.Second // attempt 1 -> 1<<0 seconds
	delays := make([]time.Duration, 0, 24)
	for i := range 24 {
		if _, err := queue.Enqueue(fmt.Sprintf("mission-jitter-%d", i), 3); err != nil {
			t.Fatal(err)
		}
		claimed, ok, err := queue.Claim("worker", time.Now().UTC())
		if err != nil || !ok {
			t.Fatalf("claim: ok=%v err=%v", ok, err)
		}
		t0 := time.Now().UTC()
		retry, err := queue.Nack(claimed, errors.New("temporary"))
		if err != nil {
			t.Fatalf("nack: %v", err)
		}
		delay := retry.AvailableAt.Sub(t0)
		// Equal jitter keeps the retry delay within [base/2, base].
		if delay < base/2-50*time.Millisecond || delay > base+100*time.Millisecond {
			t.Fatalf("iteration %d: backoff %v outside jitter window [%v, %v]", i, delay, base/2, base)
		}
		delays = append(delays, delay)
	}

	// Jitter must actually vary: a fixed backoff would make every delay identical.
	allEqual := true
	for _, d := range delays[1:] {
		if d != delays[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		t.Fatalf("all %d backoffs were identical (%v); jitter not applied", len(delays), delays[0])
	}
}

func TestRunQueueHandlerRecoversPanicAsNonRetryable(t *testing.T) {
	err := runQueueHandler(context.Background(), QueueJob{ID: "job_panic_test"}, func(context.Context, QueueJob) error {
		panic("provider callback panic")
	})
	if !errors.Is(err, ErrQueueNonRetryable) || !strings.Contains(err.Error(), "queue handler panicked") {
		t.Fatalf("panic result=%v, want non-retryable recovered error", err)
	}
}

func TestJobQueueRejectsInvalidPersistedAttemptMetadata(t *testing.T) {
	root := t.TempDir()
	data, err := json.Marshal(map[string]QueueJob{"job_bad": {ID: "job_bad", MissionID: "mis_bad", Status: QueuePending, Attempts: 0, MaxAttempts: int(^uint(0) >> 1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "jobs.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewJobQueue(root); err == nil || !strings.Contains(err.Error(), "retry metadata is invalid") {
		t.Fatalf("invalid persisted attempt metadata error = %v", err)
	}
}

func TestJobQueueRetriesDeadLettersAndReplays(t *testing.T) {
	root := t.TempDir()
	queue, err := NewJobQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mission-1", 2)
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := queue.Claim("worker-1", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("claim = %+v, %v, %v", claimed, ok, err)
	}
	if _, err := queue.Nack(claimed, errors.New("temporary")); err != nil {
		t.Fatal(err)
	}
	queue.mu.Lock()
	retry := queue.jobs[job.ID]
	retry.AvailableAt = time.Now().UTC()
	queue.jobs[job.ID] = retry
	_ = queue.persistLocked(retry)
	queue.mu.Unlock()
	claimed, ok, err = queue.Claim("worker-2", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("second claim = %+v, %v, %v", claimed, ok, err)
	}
	dead, err := queue.Nack(claimed, errors.New("permanent"))
	if err != nil || dead.Status != QueueDeadLetter {
		t.Fatalf("dead = %+v, err=%v", dead, err)
	}
	replayed, err := queue.Replay(job.ID)
	if err != nil || replayed.Status != QueuePending || replayed.Attempts != 0 {
		t.Fatalf("replayed = %+v, err=%v", replayed, err)
	}
	reloaded, err := NewJobQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	if jobs := reloaded.List(QueuePending); len(jobs) != 1 {
		t.Fatalf("pending after reload = %+v", jobs)
	}
}

func TestJobQueueWorkerAcknowledgesJobs(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mission-2", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	queue.Start(ctx, "worker", func(_ context.Context, got QueueJob) error {
		if got.ID != job.ID {
			t.Errorf("got job %+v", got)
		}
		close(done)
		return nil
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not claim job")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if status := queue.List(QueueSucceeded); len(status) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker returned but job was not acknowledged: pending=%+v running=%+v succeeded=%+v", queue.List(QueuePending), queue.List(QueueRunning), queue.List(QueueSucceeded))
		}
		time.Sleep(time.Millisecond)
	}
}

func TestJobQueueRollsBackClaimWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	queue, err := NewJobQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mission-rollback", 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := queue.Claim("worker-rollback", time.Now().UTC()); err == nil || ok {
		t.Fatalf("claim = ok=%v err=%v, want persistence failure", ok, err)
	}
	queue.mu.Lock()
	pending, exists := queue.jobs[job.ID]
	queue.mu.Unlock()
	if !exists || pending.Status != QueuePending || pending.Attempts != 0 {
		t.Fatalf("claim mutation was not rolled back in memory: %+v exists=%v", pending, exists)
	}
}

func TestJobQueueRejectsAckAndNackForNonRunningJobs(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mission-state", 1)
	if err != nil {
		t.Fatal(err)
	}
	claim := QueueJob{ID: job.ID, WorkerID: "untrusted", LeaseToken: "stale"}
	if err := queue.Ack(claim); err == nil {
		t.Fatal("ack of pending job unexpectedly succeeded")
	}
	if _, err := queue.Nack(claim, errors.New("unexpected")); err == nil {
		t.Fatal("nack of pending job unexpectedly succeeded")
	}
}

func TestJobQueueEnqueueBindsOrganizationOwner(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := queue.Enqueue("mission-owner", 3)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := queue.EnqueueForOrganization("org-a", "mission-owner", 3)
	if err != nil || bound.ID != legacy.ID || bound.OrganizationID != "org-a" {
		t.Fatalf("legacy bind=%+v err=%v", bound, err)
	}
	duplicate, err := queue.EnqueueForOrganization("org-a", "mission-owner", 3)
	if err != nil || duplicate.ID != bound.ID {
		t.Fatalf("same-tenant duplicate=%+v err=%v", duplicate, err)
	}
	if _, err := queue.EnqueueForOrganization("org-b", "mission-owner", 3); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("cross-tenant duplicate error=%v, want forbidden", err)
	}
}

func TestJobQueueReplayRequiresPersistedOrganizationOwner(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.EnqueueForOrganization("org-a", "mission-replay-owner", 1)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker", time.Now().UTC())
	if err != nil || !ok || claim.ID != job.ID {
		t.Fatalf("claim=%+v ok=%v err=%v", claim, ok, err)
	}
	dead, err := queue.Nack(claim, errors.New("synthetic failure"))
	if err != nil || dead.Status != QueueDeadLetter {
		t.Fatalf("dead-letter=%+v err=%v", dead, err)
	}
	if _, err := queue.ReplayForOrganization("org-b", job.ID); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("cross-owner replay error=%v, want forbidden", err)
	}
	replayed, err := queue.ReplayForOrganization("org-a", job.ID)
	if err != nil || replayed.Status != QueuePending || replayed.OrganizationID != "org-a" {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}

	legacyQueue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := legacyQueue.Enqueue("mission-legacy-replay", 1)
	if err != nil {
		t.Fatal(err)
	}
	legacyClaim, ok, err := legacyQueue.Claim("legacy-worker", time.Now().UTC())
	if err != nil || !ok || legacyClaim.ID != legacy.ID {
		t.Fatalf("legacy claim=%+v ok=%v err=%v", legacyClaim, ok, err)
	}
	if _, err := legacyQueue.Nack(legacyClaim, errors.New("synthetic failure")); err != nil {
		t.Fatal(err)
	}
	_, err = legacyQueue.ReplayForOrganization("org-a", legacy.ID)
	if !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("ownerless replay error=%v, want forbidden", err)
	}
	legacyItems := legacyQueue.List(QueueDeadLetter)
	if len(legacyItems) != 1 || legacyItems[0].ID != legacy.ID || legacyItems[0].OrganizationID != "" {
		t.Fatalf("ownerless dead-letter changed after rejected replay: %+v", legacyItems)
	}
}

func TestJobQueueEnqueueIsIdempotentByMission(t *testing.T) {
	queue, err := NewJobQueue("")
	if err != nil {
		t.Fatal(err)
	}
	first, err := queue.Enqueue("mission-idempotent", 3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := queue.Enqueue("mission-idempotent", 3)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || len(queue.List("")) != 1 {
		t.Fatalf("duplicate mission jobs: first=%+v second=%+v jobs=%+v", first, second, queue.List(""))
	}
}
