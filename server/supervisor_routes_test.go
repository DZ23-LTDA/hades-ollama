package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func setupSupervisorTestServer(t *testing.T) (*gin.Engine, *agent.Runtime, *agent.Supervisor) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	sup := runtime.Supervisor()
	if sup == nil {
		t.Fatal("expected supervisor to be initialized")
	}

	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}

	group := r.Group("/api/agent/v1")
	group.GET("/supervisor/status", api.supervisorStatus)
	group.POST("/supervisor/config", api.supervisorConfig)
	group.POST("/supervisor/tick", api.supervisorTick)

	return r, runtime, sup
}

func TestSupervisorRoutesStatus(t *testing.T) {
	srv, _, _ := setupSupervisorTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/supervisor/status", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if res["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", res["enabled"])
	}
}

func TestSupervisorRoutesConfig(t *testing.T) {
	srv, _, sup := setupSupervisorTestServer(t)

	cfg := sup.Config()
	cfg.Enabled = false

	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/supervisor/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	if sup.Status().Enabled != false {
		t.Fatal("expected supervisor to be disabled after config update")
	}
}

func TestSupervisorRoutesTick(t *testing.T) {
	srv, _, _ := setupSupervisorTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/supervisor/tick", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if _, ok := res["result"]; !ok {
		t.Fatal("expected result object in tick response")
	}
}
