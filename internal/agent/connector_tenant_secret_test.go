package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectorTenantTokenDoesNotFallbackToProcessEnvironment(t *testing.T) {
	t.Setenv("TENANT_SHARED_TOKEN", "org-a-secret")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer org-a-secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager := NewConnectorManager()
	manager.client = server.Client()
	config := ConnectorConfig{ID: "tenant-secret", OrganizationID: "org-a", Provider: "test", BaseURL: server.URL, TokenEnv: "TENANT_SHARED_TOKEN", Operations: []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}
	if err := manager.RegisterForOrganization("org-a", config); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.CallForOrganization(context.Background(), "org-a", config.ID, "read", http.MethodGet, "/", nil); !errors.Is(err, ErrConnectorCredentialUnavailable) {
		t.Fatalf("expected tenant-scoped credential failure, got %v", err)
	}
	if err := manager.SetOrganizationSecret("org-a", config.TokenEnv, "org-a-secret"); err != nil {
		t.Fatal(err)
	}
	if status, _, err := manager.CallForOrganization(context.Background(), "org-a", config.ID, "read", http.MethodGet, "/", nil); err != nil || status != http.StatusOK {
		t.Fatalf("tenant secret call status=%d err=%v", status, err)
	}
}
