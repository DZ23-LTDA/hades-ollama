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

func connectorRegistrationContext(t *testing.T, body string, organizationID string, role agent.Role) (*agentAPI, *gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: t.TempDir(),
		DataRoot:      t.TempDir(),
		Connectors:    agent.NewConnectorManager(),
	})
	if err != nil {
		t.Fatal(err)
	}
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/connectors", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	ctx.Set("agent.membership", agent.Membership{Role: role})
	return &agentAPI{runtime: runtime, context: contextStore, authRequired: true}, ctx, recorder
}

func TestRegisterConnectorRequiresAdminAndBindsOrganization(t *testing.T) {
	body := `{"id":"github","provider":"GitHub","base_url":"https://api.github.example","token_env":"OLLAMA_GITHUB_TOKEN","operations":[{"name":"profile","methods":["GET"],"path_prefixes":["/user"]}]}`
	api, memberContext, memberRecorder := connectorRegistrationContext(t, body, "org-a", agent.RoleViewer)
	api.registerConnector(memberContext)
	if memberRecorder.Code != http.StatusForbidden {
		t.Fatalf("member status=%d body=%s", memberRecorder.Code, memberRecorder.Body.String())
	}

	api, adminContext, adminRecorder := connectorRegistrationContext(t, body, "org-a", agent.RoleAdmin)
	api.registerConnector(adminContext)
	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("admin status=%d body=%s", adminRecorder.Code, adminRecorder.Body.String())
	}
	if got := api.runtime.ConnectorsForOrganization("org-a"); len(got) != 1 || got[0].OrganizationID != "org-a" {
		t.Fatalf("org-a connectors=%+v", got)
	}
	if got := api.runtime.ConnectorsForOrganization("org-b"); len(got) != 0 {
		t.Fatalf("cross-tenant connector visible=%+v", got)
	}
}

func TestRegisterConnectorRejectsCrossTenantAndRawSecretFields(t *testing.T) {
	crossTenant := `{"id":"github","organization_id":"org-b","provider":"GitHub","base_url":"https://api.github.example","operations":[{"name":"profile","methods":["GET"],"path_prefixes":["/user"]}]}`
	api, ctx, recorder := connectorRegistrationContext(t, crossTenant, "org-a", agent.RoleOwner)
	api.registerConnector(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	rawSecret := `{"id":"github","provider":"GitHub","base_url":"https://api.github.example","token":"do-not-accept","operations":[{"name":"profile","methods":["GET"],"path_prefixes":["/user"]}]}`
	api, ctx, recorder = connectorRegistrationContext(t, rawSecret, "org-a", agent.RoleOwner)
	api.registerConnector(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("raw-secret status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := api.runtime.ConnectorsForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("raw-secret request registered connector=%+v", got)
	}
}

func TestRegisterConnectorDoesNotExposeTokenEnvironmentInResponse(t *testing.T) {
	body := `{"id":"github","provider":"GitHub","base_url":"https://api.github.example","token_env":"OLLAMA_GITHUB_TOKEN","operations":[{"name":"profile","methods":["GET"],"path_prefixes":["/user"]}]}`
	api, ctx, recorder := connectorRegistrationContext(t, body, "org-a", agent.RoleOwner)
	api.registerConnector(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(response)
	if bytes.Contains(encoded, []byte("OLLAMA_GITHUB_TOKEN")) {
		t.Fatalf("response leaked token environment name: %s", encoded)
	}
}

func TestConnectorLifecycleHidesForeignIDsAsNotFound(t *testing.T) {
	api, _, _ := connectorRegistrationContext(t, "", "org-a", agent.RoleOwner)
	if err := api.runtime.RegisterConnectorForOrganization("org-a", agent.ConnectorConfig{ID: "tenant-a", Provider: "test", BaseURL: "https://api.example.test", Operations: []agent.ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}}); err != nil {
		t.Fatal(err)
	}
	invoke := func(id string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/agent/v1/plugins/connectors/"+id, nil)
		ctx.Params = gin.Params{{Key: "id", Value: id}}
		ctx.Set("agent.organization", agent.Organization{ID: "org-b"})
		ctx.Set("agent.membership", agent.Membership{OrganizationID: "org-b", Role: agent.RoleOwner})
		api.removeConnector(ctx)
		return recorder
	}
	foreign := invoke("tenant-a")
	unknown := invoke("does-not-exist")
	if foreign.Code != http.StatusNotFound || unknown.Code != http.StatusNotFound || foreign.Body.String() != unknown.Body.String() {
		t.Fatalf("foreign/unknown connector oracle: foreign=%d %s unknown=%d %s", foreign.Code, foreign.Body.String(), unknown.Code, unknown.Body.String())
	}
}
