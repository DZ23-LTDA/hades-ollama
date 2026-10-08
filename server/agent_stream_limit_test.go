package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestAcquireStreamSlotBoundsConcurrency locks in SEC-13: concurrent long-lived
// SSE/WS streams are capped, excess connections get a 503, and releasing a slot
// frees capacity for the next client.
func TestAcquireStreamSlotBoundsConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_MAX_STREAMS", "2")
	api := &agentAPI{}

	ctx1, _ := gin.CreateTestContext(httptest.NewRecorder())
	r1, ok := api.acquireStreamSlot(ctx1)
	if !ok {
		t.Fatal("first stream slot should be granted")
	}
	ctx2, _ := gin.CreateTestContext(httptest.NewRecorder())
	r2, ok := api.acquireStreamSlot(ctx2)
	if !ok {
		t.Fatal("second stream slot should be granted")
	}

	rec3 := httptest.NewRecorder()
	ctx3, _ := gin.CreateTestContext(rec3)
	if _, ok := api.acquireStreamSlot(ctx3); ok {
		t.Fatal("third stream slot must be rejected over the cap")
	}
	if rec3.Code != 503 {
		t.Fatalf("over-cap acquire should write 503, got %d", rec3.Code)
	}

	// Releasing one slot frees capacity again.
	r1()
	ctx4, _ := gin.CreateTestContext(httptest.NewRecorder())
	r4, ok := api.acquireStreamSlot(ctx4)
	if !ok {
		t.Fatal("slot should be available after a release")
	}

	// Releases are idempotent (safe to defer twice).
	r1()
	r2()
	r4()
}

// TestMaxConcurrentStreamsUnlimited confirms OLLAMA_MAX_STREAMS=0 disables the cap.
func TestMaxConcurrentStreamsUnlimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_MAX_STREAMS", "0")
	api := &agentAPI{}
	for i := 0; i < 1000; i++ {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		if _, ok := api.acquireStreamSlot(ctx); !ok {
			t.Fatalf("stream %d rejected although the cap is disabled", i)
		}
	}
}
