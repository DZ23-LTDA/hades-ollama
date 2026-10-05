package server

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/internal/agent"
)

// companionIdleTimeout limita quanto tempo uma conexão companion pode ficar
// sem enviar frame algum antes de ser encerrada.
const companionIdleTimeout = 2 * time.Minute

var companionUpgrader = websocket.Upgrader{ReadBufferSize: 16 << 10, WriteBufferSize: 16 << 10, CheckOrigin: companionOriginAllowed}

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
	if err := connection.ReadJSON(&hello); err != nil || hello.Type != "hello" {
		_ = connection.WriteJSON(gin.H{"type": "error", "error": "invalid companion handshake"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hello.DeviceID), []byte(c.Param("id"))) != 1 {
		_ = connection.WriteJSON(gin.H{"type": "error", "error": "invalid companion handshake"})
		return
	}
	device, err := a.runtime.Devices().Heartbeat(hello.DeviceID, hello.Token, hello.Capabilities)
	if err != nil {
		_ = connection.WriteJSON(gin.H{"type": "error", "error": "device authentication failed"})
		return
	}
	// Sem deadline, uma conexão ociosa (ou um peer que sumiu sem FIN) segura a
	// goroutine e o descriptor indefinidamente. O prazo é renovado a cada frame
	// recebido e a cada pong do peer.
	_ = connection.SetReadDeadline(time.Now().Add(companionIdleTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(companionIdleTimeout))
	})
	_ = connection.WriteJSON(agent.CompanionWelcome{Type: "welcome", DeviceID: device.ID, Protocol: "dz23-companion.v1", ServerNow: time.Now().UTC()})
	for {
		var frame agent.CompanionFrame
		if err := connection.ReadJSON(&frame); err != nil {
			return
		}
		_ = connection.SetReadDeadline(time.Now().Add(companionIdleTimeout))
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
	if request.TLS != nil {
		return true
	}
	// X-Forwarded-Proto é escrito pelo próprio cliente quando não há proxy na
	// frente. Só vale como prova de TLS se o operador declarar explicitamente
	// que existe um proxy confiável terminando TLS antes deste servidor.
	if os.Getenv("OLLAMA_AGENT_TRUSTED_TLS_PROXY") != "1" {
		return false
	}
	return strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")
}
