package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func newPluginAPI(t *testing.T) *agentAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	registry, err := agent.NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{plugins: registry, authRequired: true}
}

func pluginRouteContext(t *testing.T, method, path, body, pluginID, organization string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if pluginID != "" {
		ctx.Params = gin.Params{{Key: "plugin_id", Value: pluginID}}
	}
	ctx.Set("agent.organization", agent.Organization{ID: organization})
	return ctx, recorder
}

func pluginBody(t *testing.T, manifest agent.PluginManifest) string {
	t.Helper()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestPluginRoutesInstallListAndKeepScopesUngranted(t *testing.T) {
	api := newPluginAPI(t)
	manifest := agent.PluginManifest{
		ID:      "estoque-widget",
		Version: "1.0.0",
		Name:    "Widget de estoque",
		Kind:    agent.PluginKindConnector,
		Scopes:  []string{"connector:external"},
	}

	installCtx, installRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, manifest), "", "org-a")
	api.installPlugin(installCtx)
	if installRecorder.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", installRecorder.Code, installRecorder.Body.String())
	}
	var installed agent.PluginInstallation
	if err := json.Unmarshal(installRecorder.Body.Bytes(), &installed); err != nil {
		t.Fatalf("decode install: %v", err)
	}
	if installed.Trusted || len(installed.GrantedScopes) != 0 {
		t.Fatalf("install must not grant anything: %+v", installed)
	}
	if len(installed.RequestedScopes) != 1 || installed.RequestedScopes[0] != "connector:external" {
		t.Fatalf("requested scopes = %+v", installed.RequestedScopes)
	}

	listCtx, listRecorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins", "", "", "org-a")
	api.listPlugins(listCtx)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status=%d", listRecorder.Code)
	}
	var listed struct {
		Plugins []agent.PluginInstallation `json:"plugins"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Plugins) != 1 || listed.Plugins[0].Manifest.Version != "1.0.0" {
		t.Fatalf("listed plugins = %+v", listed.Plugins)
	}

	// Escopo desconhecido é recusado com 400 e nada é instalado.
	bad := manifest
	bad.ID = "outro-widget"
	bad.Scopes = []string{"root:tudo"}
	badCtx, badRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, bad), "", "org-a")
	api.installPlugin(badCtx)
	if badRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown scope status=%d body=%s", badRecorder.Code, badRecorder.Body.String())
	}
}

func TestPluginRoutesRejectCrossOrganizationInstallAndMissingRollback(t *testing.T) {
	api := newPluginAPI(t)
	manifest := agent.PluginManifest{ID: "estoque-widget", Version: "1.0.0", Name: "Widget", Kind: agent.PluginKindSkill}

	firstCtx, firstRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, manifest), "", "org-a")
	api.installPlugin(firstCtx)
	if firstRecorder.Code != http.StatusCreated {
		t.Fatalf("install status=%d", firstRecorder.Code)
	}

	// Mesmo id em outra organização: 403 (isolamento), sem vazar o plugin.
	crossCtx, crossRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, manifest), "", "org-b")
	api.installPlugin(crossCtx)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-organization install status=%d body=%s", crossRecorder.Code, crossRecorder.Body.String())
	}

	// Rollback sem versão anterior: 409, não 500.
	rollbackCtx, rollbackRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/estoque-widget/rollback", "", "estoque-widget", "org-b")
	api.rollbackPlugin(rollbackCtx)
	if rollbackRecorder.Code != http.StatusNotFound {
		t.Fatalf("rollback in the wrong organization status=%d", rollbackRecorder.Code)
	}
	rollbackCtx2, rollbackRecorder2 := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/estoque-widget/rollback", "", "estoque-widget", "org-a")
	api.rollbackPlugin(rollbackCtx2)
	if rollbackRecorder2.Code != http.StatusConflict {
		t.Fatalf("rollback without history status=%d body=%s", rollbackRecorder2.Code, rollbackRecorder2.Body.String())
	}
}

func TestPluginRoutePromotionRequiresOperatorAuthorizedKey(t *testing.T) {
	api := newPluginAPI(t)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := agent.SignPluginManifest(agent.PluginManifest{
		ID:      "estoque-widget",
		Version: "1.0.0",
		Name:    "Widget",
		Kind:    agent.PluginKindConnector,
		Scopes:  []string{"connector:external"},
	}, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	installCtx, installRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, signed), "", "org-a")
	api.installPlugin(installCtx)
	if installRecorder.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", installRecorder.Code, installRecorder.Body.String())
	}

	// Sem chave autorizada no ambiente: promoção falha fechada com 403.
	t.Setenv(PluginTrustedKeysEnv, "")
	promoteCtx, promoteRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/estoque-widget/promote", "", "estoque-widget", "org-a")
	api.promotePlugin(promoteCtx)
	if promoteRecorder.Code != http.StatusForbidden {
		t.Fatalf("promotion without trust root status=%d body=%s", promoteRecorder.Code, promoteRecorder.Body.String())
	}

	// Chave autorizada pelo operador: promoção concede os escopos.
	t.Setenv(PluginTrustedKeysEnv, "publisher-1:"+base64.RawStdEncoding.EncodeToString(publicKey))
	promoteCtx2, promoteRecorder2 := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/estoque-widget/promote", "", "estoque-widget", "org-a")
	api.promotePlugin(promoteCtx2)
	if promoteRecorder2.Code != http.StatusOK {
		t.Fatalf("promotion status=%d body=%s", promoteRecorder2.Code, promoteRecorder2.Body.String())
	}
	var promoted agent.PluginInstallation
	if err := json.Unmarshal(promoteRecorder2.Body.Bytes(), &promoted); err != nil {
		t.Fatalf("decode promotion: %v", err)
	}
	if !promoted.Trusted || len(promoted.GrantedScopes) != 1 || promoted.GrantedScopes[0] != "connector:external" {
		t.Fatalf("promoted plugin = %+v", promoted)
	}

	// Configuração inválida do operador: 500 explícito, nunca promoção silenciosa.
	t.Setenv(PluginTrustedKeysEnv, "sem-separador")
	badCtx, badRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/estoque-widget/promote", "", "estoque-widget", "org-a")
	api.promotePlugin(badCtx)
	if badRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("invalid trust root status=%d body=%s", badRecorder.Code, badRecorder.Body.String())
	}
}
