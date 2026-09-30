package server

import (
	"net/http"
	"time"

	"github.com/ollama/ollama/internal/agent"
)

// newServerEgressClient is the only production HTTP client factory for server
// initiated requests. Loopback is opt-in for the local Ollama/remote-model
// compatibility paths; all other destinations are public-IP-only.
func newServerEgressClient(callsite string, allowLoopback bool) *http.Client {
	return agent.NewSafeEgressHTTPClient(agent.EgressOptions{
		Callsite:      callsite,
		AllowLoopback: allowLoopback,
		Timeout:       30 * time.Second,
		MaxBodyBytes:  20 << 20,
	})
}
