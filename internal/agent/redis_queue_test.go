package agent

import (
	"context"
	"strings"
	"testing"
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
