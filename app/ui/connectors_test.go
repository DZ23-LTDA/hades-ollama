//go:build windows || darwin

package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

func TestConnectConnectorRegistersWithTokenEnv(t *testing.T) {
	var registered map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/agent/v1/connectors" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/agent/v1/connectors" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &registered)
			w.Write([]byte(`{}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN", "")
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN_FILE", "")
	t.Cleanup(func() { _ = removeTestConnectorEnv("OLLAMA_CONNECTOR_RESEND_TOKEN") })

	rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/resend/key", `{"key":"re_test_123"}`)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if registered["id"] != "resend" || registered["base_url"] != "https://api.resend.com" || registered["token_env"] != "OLLAMA_CONNECTOR_RESEND_TOKEN" {
		t.Fatalf("registered = %v", registered)
	}
	if _, leaked := registered["key"]; leaked {
		t.Fatal("the key must never be sent to the agent API")
	}
	ops, ok := registered["operations"].([]any)
	if !ok || len(ops) != 1 {
		t.Fatalf("quick-connect operations = %#v; want one read-only operation", registered["operations"])
	}
	operation := ops[0].(map[string]any)
	if operation["name"] != "read" {
		t.Fatalf("quick-connect operation = %#v; want read", operation)
	}
	if methods := operation["methods"].([]any); len(methods) != 1 || methods[0] != http.MethodGet {
		t.Fatalf("quick-connect methods = %#v; want GET only", operation["methods"])
	}
	if prefixes := operation["path_prefixes"].([]any); len(prefixes) != 1 || prefixes[0] != "/domains" {
		t.Fatalf("resend quick-connect path policy = %#v; want /domains", operation["path_prefixes"])
	}
}

func TestConnectConnectorErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bearer token is required"}`, http.StatusUnauthorized)
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("OLLAMA_CONNECTOR_GITHUB_TOKEN", "")
	t.Setenv("OLLAMA_CONNECTOR_GITHUB_TOKEN_FILE", "")
	t.Cleanup(func() { _ = removeTestConnectorEnv("OLLAMA_CONNECTOR_GITHUB_TOKEN") })

	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/gmail/key", `{"key":"x"}`); rr.Code != http.StatusNotFound {
		t.Fatalf("oauth-only connector status = %d", rr.Code)
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/algolia/key", `{"key":"x","base_url":"http://insecure"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("self-hosted without https status = %d", rr.Code)
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/github/key", `{"key":""}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("empty key status = %d", rr.Code)
	}
	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/github/key", `{"key":"ghp_x"}`); rr.Code != http.StatusForbidden {
		t.Fatalf("login-required status = %d, body %s", rr.Code, rr.Body.String())
	}
}

// removeTestConnectorEnv clears the saved key and the persisted user
// variable that secrets.Save writes, so tests leave no trace.
func removeTestConnectorEnv(envName string) error {
	return secrets.Remove(multillm.DefaultCredentialDir(), envName)
}

func TestConnectConnectorSendsHeaderAndSelfHostedURL(t *testing.T) {
	var registered map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &registered)
		w.Write([]byte(`{}`))
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	for _, env := range []string{"OLLAMA_CONNECTOR_APPWRITE_TOKEN", "OLLAMA_CONNECTOR_ALGOLIA_TOKEN"} {
		t.Setenv(env, "")
		t.Setenv(env+"_FILE", "")
		t.Cleanup(func() { _ = removeTestConnectorEnv(env) })
	}

	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/appwrite/key", `{"key":"k"}`); rr.Code != http.StatusNoContent {
		t.Fatalf("appwrite status %d: %s", rr.Code, rr.Body.String())
	}
	if registered["auth_header"] != "X-Appwrite-Key" || registered["base_url"] != "https://cloud.appwrite.io/v1" {
		t.Fatalf("appwrite registered = %v", registered)
	}

	if rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/algolia/key", `{"key":"k","base_url":"https://algolia.example.com/api/v1/"}`); rr.Code != http.StatusNoContent {
		t.Fatalf("algolia status %d: %s", rr.Code, rr.Body.String())
	}
	if registered["base_url"] != "https://algolia.example.com/api/v1" {
		t.Fatalf("algolia registered = %v", registered)
	}
}

func TestConnectConnectorForwardsAgentSessionAndOrganization(t *testing.T) {
	var got []http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Clone())
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN", "")
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN_FILE", "")
	t.Cleanup(func() { _ = removeTestConnectorEnv("OLLAMA_CONNECTOR_RESEND_TOKEN") })

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer workspace-session")
	headers.Set("X-Ollama-Organization", "org-a")
	rr := providerRequestWithHeaders(t, http.MethodPut, "/api/v1/connectors/resend/key", `{"key":"re_test_123"}`, headers)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if len(got) != 2 {
		t.Fatalf("agent API requests = %d; want preflight and registration", len(got))
	}
	for _, requestHeaders := range got {
		if requestHeaders.Get("Authorization") != "Bearer workspace-session" || requestHeaders.Get("X-Ollama-Organization") != "org-a" {
			t.Fatalf("session headers not forwarded: %#v", requestHeaders)
		}
	}
}

func TestConnectConnectorPreflightFailureDoesNotPersistCredential(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, `{"error":"bearer token is required"}`, http.StatusUnauthorized)
			return
		}
		t.Fatal("registration must not run after failed authentication preflight")
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN", "")
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN_FILE", "")
	t.Cleanup(func() { _ = removeTestConnectorEnv("OLLAMA_CONNECTOR_RESEND_TOKEN") })

	rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/resend/key", `{"key":"re_test_123"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body %s", rr.Code, rr.Body.String())
	}
	if secrets.Configured("OLLAMA_CONNECTOR_RESEND_TOKEN") {
		t.Fatal("credential was persisted despite failed auth preflight")
	}
}

func TestConnectConnectorRollsBackCredentialAfterRegistrationFailure(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN", "")
	t.Setenv("OLLAMA_CONNECTOR_RESEND_TOKEN_FILE", "")
	// The default directory on Windows is LOCALAPPDATA/Ollama DZ23/secrets.
	credentialDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Ollama DZ23", "secrets")
	oldKey := "re_existing_previous_key"
	if _, err := secrets.Save(credentialDir, "OLLAMA_CONNECTOR_RESEND_TOKEN", oldKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = removeTestConnectorEnv("OLLAMA_CONNECTOR_RESEND_TOKEN") })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, `{"error":"registration rejected"}`, http.StatusInternalServerError)
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)
	rr := providerRequest(t, http.MethodPut, "/api/v1/connectors/resend/key", `{"key":"re_new_key"}`)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body %s", rr.Code, rr.Body.String())
	}
	if got := multillm.CredentialValue("OLLAMA_CONNECTOR_RESEND_TOKEN"); got != oldKey {
		t.Fatalf("credential after failed registration = %q; want prior credential restored", got)
	}
}

func TestQuickConnectPoliciesAreNarrowAndReadOnly(t *testing.T) {
	for _, entry := range agent.ConnectorCatalog() {
		if !entry.QuickConnect {
			continue
		}
		operations := agent.QuickConnectOperations(entry.ID)
		if len(operations) == 0 {
			t.Errorf("quick-connect entry %q has no policy", entry.ID)
			continue
		}
		for _, operation := range operations {
			if operation.Name != "read" {
				t.Errorf("%s operation %q is not read-only", entry.ID, operation.Name)
			}
			for _, method := range operation.Methods {
				if method != http.MethodGet {
					t.Errorf("%s allows unexpected method %q", entry.ID, method)
				}
			}
			for _, prefix := range operation.PathPrefixes {
				if prefix == "/" || prefix == "" {
					t.Errorf("%s has broad/empty path prefix %q", entry.ID, prefix)
				}
			}
		}
	}
}
