package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// TestPluginStatusRouteDoesNotClaimReadyWithoutExecution prova, pela rota real,
// a regra do V2.1 §11.3: catalogar não é habilitar. Um plugin instalado e
// confiável, mas nunca executado, sai como INSTALLED com ready_to_use=false.
func TestPluginStatusRouteDoesNotClaimReadyWithoutExecution(t *testing.T) {
	api := newPluginAPI(t)
	manifest := agent.PluginManifest{
		ID:         "runtime-plugin",
		Version:    "1.0.0",
		Name:       "Plugin de runtime",
		Kind:       agent.PluginKindConnector,
		Runtime:    "python",
		Entrypoint: "main.py",
		Scopes:     []string{"sandbox:execute"},
	}
	if _, err := api.plugins.InstallForOrganization("org-a", manifest, agent.DefaultCapabilityPolicy()); err != nil {
		t.Fatalf("install: %v", err)
	}

	ctx, recorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins/status", "", "", "org-a")
	api.pluginStatuses(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OrganizationID string               `json:"organization_id"`
		Plugins        []agent.PluginStatus `json:"plugins"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v (body=%s)", err, recorder.Body.String())
	}
	if payload.OrganizationID != "org-a" || len(payload.Plugins) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	status := payload.Plugins[0]
	if status.PluginID != "runtime-plugin" {
		t.Fatalf("plugin id = %q", status.PluginID)
	}
	if status.ReadyToUse {
		t.Fatal("nunca executado não pode sair como pronto para usar")
	}
	if status.State == agent.PluginStateConnected {
		t.Fatalf("CONNECTED não pode ser declarado sem verificação de vínculo: %+v", status)
	}
	if status.State != agent.PluginStateUnauthorized {
		// Instalação por manifesto nasce não confiável: a prioridade é a recusa.
		t.Fatalf("estado = %q (%v)", status.State, status.Reasons)
	}
	if len(status.Reasons) == 0 {
		t.Fatal("estado de recusa precisa explicar o motivo")
	}
}

// TestPluginStatusRouteUsesLocalScopeWithoutLeakingOtherOrganizations cobre o
// comportamento real do modo local (single-tenant com escopo fixo): sem
// organização no contexto, o status é servido para `local` — e os plugins de
// outra organização NÃO aparecem.
func TestPluginStatusRouteUsesLocalScopeWithoutLeakingOtherOrganizations(t *testing.T) {
	api := newPluginAPI(t)
	manifest := agent.PluginManifest{
		ID:         "runtime-plugin",
		Version:    "1.0.0",
		Name:       "Plugin de runtime",
		Kind:       agent.PluginKindConnector,
		Runtime:    "python",
		Entrypoint: "main.py",
		Scopes:     []string{"sandbox:execute"},
	}
	if _, err := api.plugins.InstallForOrganization("org-a", manifest, agent.DefaultCapabilityPolicy()); err != nil {
		t.Fatalf("install: %v", err)
	}
	ctx, recorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins/status", "", "", "")
	api.pluginStatuses(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OrganizationID string               `json:"organization_id"`
		Plugins        []agent.PluginStatus `json:"plugins"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if payload.OrganizationID != agent.LocalOrganizationID {
		t.Fatalf("escopo local esperado, veio %q", payload.OrganizationID)
	}
	if len(payload.Plugins) != 0 {
		t.Fatalf("plugin de outra organização não pode aparecer no escopo local: %+v", payload.Plugins)
	}
}

// TestPluginStatusRouteListsEmptyOrganization garante que uma organização sem
// plugins recebe lista vazia — e não um erro nem um plugin inventado.
func TestPluginStatusRouteListsEmptyOrganization(t *testing.T) {
	api := newPluginAPI(t)
	ctx, recorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins/status", "", "", "org-vazia")
	api.pluginStatuses(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Plugins []agent.PluginStatus `json:"plugins"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(payload.Plugins) != 0 {
		t.Fatalf("organização sem plugins deveria devolver lista vazia, veio %+v", payload.Plugins)
	}
}
