package server

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// defaultMaxConcurrentStreams caps how many long-lived SSE/WS connections the
// agent API serves at once when OLLAMA_MAX_STREAMS is not set.
const defaultMaxConcurrentStreams int64 = 512

// maxConcurrentStreams returns the configured cap on concurrent streams. A value
// of 0 (OLLAMA_MAX_STREAMS=0) disables the limit.
func maxConcurrentStreams() int64 {
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_MAX_STREAMS")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	return defaultMaxConcurrentStreams
}

// acquireStreamSlot reserves a slot for a long-lived SSE/WS stream (SEC-13).
// Without a bound, a client can open unlimited event streams and exhaust server
// goroutines, sockets and memory. On success it returns a release function
// (idempotent — safe to defer) and true. When the cap is already reached it
// writes a 503 and returns false, and the caller must stop.
func (a *agentAPI) acquireStreamSlot(c *gin.Context) (func(), bool) {
	limit := maxConcurrentStreams()
	n := a.activeStreams.Add(1)
	if limit > 0 && n > limit {
		a.activeStreams.Add(-1)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "too many concurrent streams; retry later"})
		return func() {}, false
	}
	var once sync.Once
	return func() { once.Do(func() { a.activeStreams.Add(-1) }) }, true
}
