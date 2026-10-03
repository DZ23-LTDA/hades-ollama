package server

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/internal/agent"
)

// TestCompanionWebSocketDropsIdleConnection verifies that a companion client
// which completes the handshake then goes silent is dropped within the idle
// window, instead of pinning a goroutine + socket forever.
func TestCompanionWebSocketDropsIdleConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_HOST", "127.0.0.1:11434")
	t.Setenv("OLLAMA_AGENT_ALLOW_INSECURE_COMPANION", "1")

	prev := companionIdleTimeout
	companionIdleTimeout = 300 * time.Millisecond
	t.Cleanup(func() { companionIdleTimeout = prev })

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	code, _, err := runtime.Devices().StartPairing("user-1", "local", time.Minute)
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	device, token, err := runtime.Devices().CompletePairing(code, "Test Device", "linux", "user-1", "local", nil)
	if err != nil {
		t.Fatalf("complete pairing: %v", err)
	}

	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}

	r := gin.New()
	r.GET("/devices/:id/connect", api.deviceConnect)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL := strings.Replace(srv.URL, "http", "ws", 1) + "/devices/" + device.ID + "/connect"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			t.Fatalf("dial: %v (status %d)", err, resp.StatusCode)
		}
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{"type": "hello", "device_id": device.ID, "token": token}); err != nil {
		t.Fatalf("send hello: %v", err)
	}

	var welcome map[string]any
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("read welcome: %v", err)
	}
	if welcome["type"] != "welcome" {
		t.Fatalf("expected welcome frame, got %v", welcome)
	}

	// Go idle: the server must close the connection within the idle window.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	start := time.Now()
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected the idle connection to be closed by the server")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("idle connection took too long to close: %v", elapsed)
	}
}
