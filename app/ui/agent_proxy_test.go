//go:build windows || darwin

package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentAPIIsProxiedToOllamaServer(t *testing.T) {
	var gotMethod, gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"version":"test"}`)
			return
		}
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"projects":[]}`)
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)

	handler := (&Server{Token: "secret"}).Handler()

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/agent/v1/projects", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: "token", Value: "secret"})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body = %s", method, rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("%s content-type = %q, want JSON from the agent API (not the SPA shell)", method, ct)
		}
		if gotMethod != method || gotPath != "/api/agent/v1/projects" {
			t.Fatalf("upstream saw %s %s, want %s /api/agent/v1/projects", gotMethod, gotPath, method)
		}
	}
}
