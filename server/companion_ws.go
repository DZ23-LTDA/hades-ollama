package server

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/internal/agent"
)

var companionUpgrader = websocket.Upgrader{ReadBufferSize: 16 << 10, WriteBufferSize: 16 << 10, CheckOrigin: companionOriginAllowed}

// defaultCompanionIdleTimeout bounds how long the post-handshake read loop waits
// for the next client frame. Without it, an authenticated client that completes
// the handshake then goes silent pins a goroutine + socket forever, so opening
// many idle connections exhausts goroutines/FDs. A healthy companion sends
// heartbeats well within this window, which refreshes the deadline.
const defaultCompanionIdleTimeout = 2 * time.Minute

// companionIdleTimeoutNanos overrides the idle timeout when > 0 (tests only). It
// is atomic because the per-connection handler goroutine reads it concurrently
// with any test override.
var companionIdleTimeoutNanos atomic.Int64

func companionIdleTimeout() time.Duration {
	if n := companionIdleTimeoutNanos.Load(); n > 0 {
		return time.Duration(n)
	}
	return defaultCompanionIdleTimeout
}

func companionOriginAllowed(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	for _, allowed := range strings.Split(os.Getenv("OLLAMA_AGENT_COMPANION_ORIGINS"), ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return strings.EqualFold(origin, "http://"+request.Host) || strings.EqualFold(origin, "https://"+request.Host)
}

func (a *agentAPI) deviceConnect(c *gin.Context) {
	if !companionSecureRequest(c.Request) && os.Getenv("OLLAMA_AGENT_ALLOW_INSECURE_COMPANION") != "1" {
		c.AbortWithStatusJSON(http.StatusUpgradeRequired, gin.H{"error": "companion transport requires TLS; set explicit local development override to allow insecure mode"})
		return
	}
	if os.Getenv("OLLAMA_AGENT_REQUIRE_MTLS") == "1" && (c.Request.TLS == nil || len(c.Request.TLS.PeerCertificates) == 0) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "mTLS client certificate is required"})
		return
	}
	connection, err := companionUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(1 << 20)
	_ = connection.SetReadDeadline(time.Now().Add(30 * time.Second))
	var hello agent.CompanionFrame
	if err := connection.ReadJSON(&hello); err != nil || hello.Type != "hello" || hello.DeviceID != c.Param("id") {
		_ = connection.WriteJSON(gin.H{"type": "error", "error": "invalid companion handshake"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hello.DeviceID), []byte(c.Param("id"))) != 1 {
		return
	}
	device, err := a.runtime.Devices().Heartbeat(hello.DeviceID, hello.Token, hello.Capabilities)
	if err != nil {
		_ = connection.WriteJSON(gin.H{"type": "error", "error": "device authentication failed"})
		return
	}
	_ = connection.WriteJSON(agent.CompanionWelcome{Type: "welcome", DeviceID: device.ID, Protocol: "dz23-companion.v1", ServerNow: time.Now().UTC()})
	for {
		// Rolling idle deadline: a silent connection is dropped instead of
		// blocking a goroutine forever. Each received frame refreshes it.
		_ = connection.SetReadDeadline(time.Now().Add(companionIdleTimeout()))
		var frame agent.CompanionFrame
		if err := connection.ReadJSON(&frame); err != nil {
			return
		}
		switch frame.Type {
		case "heartbeat":
			if _, err := a.runtime.Devices().Heartbeat(device.ID, hello.Token, frame.Capabilities); err != nil {
				return
			}
			_ = connection.WriteJSON(gin.H{"type": "heartbeat_ack", "request_id": frame.RequestID, "created_at": time.Now().UTC()})
		case "ping":
			_ = connection.WriteJSON(gin.H{"type": "pong", "request_id": frame.RequestID, "created_at": time.Now().UTC()})
		default:
			_ = connection.WriteJSON(gin.H{"type": "ack", "request_id": frame.RequestID, "accepted": false, "reason": "frame type is not enabled"})
		}
	}
}

func companionSecureRequest(request *http.Request) bool {
	return request.TLS != nil || strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}
