package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOpenRedisQueueUsesTLSForRediss(t *testing.T) {
	_, err := OpenRedisQueue(context.Background(), "rediss://127.0.0.1:1/0", "test")
	if err == nil || strings.Contains(err.Error(), "TLS-configured Redis adapter") {
		t.Fatalf("rediss error = %v", err)
	}
}

func TestOpenRedisQueueRejectsPlaintextRemoteBeforeDial(t *testing.T) {
	for _, rawURL := range []string{"redis://example.invalid:6379/0", "redis://localhost:6379/0"} {
		if _, err := OpenRedisQueue(context.Background(), rawURL, "test"); err == nil || !strings.Contains(err.Error(), "literal loopback IP") {
			t.Fatalf("plaintext remote URL %q error = %v", rawURL, err)
		}
	}
}

func TestOpenRedisQueueRejectsGlobPrefixBeforeDial(t *testing.T) {
	_, err := OpenRedisQueue(context.Background(), "redis://127.0.0.1:1/0", "tenant:*:queue")
	if err == nil || !strings.Contains(err.Error(), "prefix") {
		t.Fatalf("invalid prefix error = %v", err)
	}
}

func TestRedisClaimAndReclaimUseAtomicLeaseScripts(t *testing.T) {
	for name, script := range map[string]string{"claim": redisClaimScript, "reclaim": redisReclaimScript} {
		if !strings.Contains(script, "redis.call") || !strings.Contains(script, "job.status") {
			t.Fatalf("%s script is not a Redis state transition", name)
		}
	}
	if redisLeaseDuration <= 0 {
		t.Fatal("redis lease duration must be positive")
	}
}

func TestRedisQueueClaimRetryDelayIsBoundedAndJittered(t *testing.T) {
	for _, failures := range []int{1, 2, 5, 7, 20} {
		delay := redisQueueClaimRetryDelay(failures)
		if delay <= 0 || delay >= 30*time.Second {
			t.Fatalf("claim retry delay for %d failures = %s, want (0, 30s)", failures, delay)
		}
	}
	if got := redisQueueClaimRetryDelay(0); got <= 0 || got >= time.Second {
		t.Fatalf("first claim retry delay = %s, want a bounded initial retry", got)
	}
}

func TestRedisQueueFailuresAreExportedInMetrics(t *testing.T) {
	metrics := &RuntimeMetrics{}
	metrics.redisQueueFailures.Add(2)
	snapshot := metrics.Snapshot()
	if snapshot.RedisQueueFailures != 2 {
		t.Fatalf("redis queue failures = %d, want 2", snapshot.RedisQueueFailures)
	}
	if !strings.Contains(snapshot.Prometheus(), "ollama_agent_redis_queue_failures 2") {
		t.Fatal("Prometheus output omitted the Redis queue failure counter")
	}
}

func TestRedisQueueDroppedErrorsRemainObservable(t *testing.T) {
	queue := &RedisQueue{errors: make(chan error, 1)}
	queue.reportError(errors.New("first"))
	queue.reportError(errors.New("overflow"))
	if got := queue.DroppedErrors(); got != 1 {
		t.Fatalf("dropped queue errors = %d, want 1", got)
	}
	if got := len(queue.Errors()); got != 1 {
		t.Fatalf("queued error count = %d, want 1", got)
	}
	metrics := &RuntimeMetrics{}
	metrics.redisQueueDroppedErrors.Add(int64(queue.DroppedErrors()))
	snapshot := metrics.Snapshot()
	if snapshot.RedisQueueDroppedErrors != 1 || !strings.Contains(snapshot.Prometheus(), "ollama_agent_redis_queue_dropped_errors 1") {
		t.Fatalf("dropped error metric missing: %+v", snapshot)
	}
}
