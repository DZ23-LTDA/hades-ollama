package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// bundleRouteRequest monta o contexto gin da rota de pacote de plugin.
func bundleRouteRequest(t *testing.T, api *agentAPI, bundlePath string, manifest agent.PluginManifest, signed agent.SignedArtifact, organization string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"manifest":    manifest,
		"bundle_path": bundlePath,
		"signature":   signed,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, recorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/bundle", string(body), "", organization)
	api.installPluginBundle(ctx)
	return recorder.Code, recorder.Body.String()
}

func TestPluginBundleRouteInstallsSignedPackage(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := t.TempDir()
	bundlePath := filepath.Join(bundleDir, "widget.zip")
	if err := os.WriteFile(bundlePath, []byte("pacote do plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	signed, err := agent.SignArtifact(bundlePath, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	api := newPluginAPI(t)
	manifest := agent.PluginManifest{
		ID:      "estoque-widget",
		Version: "1.0.0",
		Name:    "Widget de estoque",
		Kind:    agent.PluginKindConnector,
		Scopes:  []string{"connector:external"},
	}

	// Sem diretório permitido configurado: falha fechada (403), nada instalado.
	t.Setenv(agent.PluginBundleDirEnv, "")
	t.Setenv(PluginTrustedKeysEnv, "publisher-1:"+base64.RawStdEncoding.EncodeToString(publicKey))
	status, body := bundleRouteRequest(t, api, bundlePath, manifest, signed, "org-a")
	if status != http.StatusForbidden {
		t.Fatalf("missing allowlist status=%d body=%s", status, body)
	}
	if got := api.plugins.ListForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("failed install must not register anything: %+v", got)
	}

	// Caminho fora do diretório permitido: 403.
	outside := filepath.Join(t.TempDir(), "alheio.zip")
	if err := os.WriteFile(outside, []byte("pacote do plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(agent.PluginBundleDirEnv, bundleDir)
	status, body = bundleRouteRequest(t, api, outside, manifest, signed, "org-a")
	if status != http.StatusForbidden {
		t.Fatalf("outside path status=%d body=%s", status, body)
	}

	// Caminho permitido e assinatura válida: 201 com conteúdo verificado.
	status, body = bundleRouteRequest(t, api, bundlePath, manifest, signed, "org-a")
	if status != http.StatusCreated {
		t.Fatalf("bundle install status=%d body=%s", status, body)
	}
	var installation agent.PluginInstallation
	if err := json.Unmarshal([]byte(body), &installation); err != nil {
		t.Fatalf("decode installation: %v body=%s", err, body)
	}
	if !installation.ContentVerified || installation.ContentSHA256 != signed.SHA256 {
		t.Fatalf("installation = %+v", installation)
	}
	if installation.Trusted || len(installation.GrantedScopes) != 0 {
		t.Fatalf("installing a package must not grant scopes: %+v", installation)
	}
	if err := api.plugins.VerifyInstalledContent("org-a", "estoque-widget"); err != nil {
		t.Fatalf("installed content must verify: %v", err)
	}
}

func TestPluginBundleRouteRejectsTamperedPackageAndUnknownKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := t.TempDir()
	bundlePath := filepath.Join(bundleDir, "widget.zip")
	if err := os.WriteFile(bundlePath, []byte("pacote original"), 0o600); err != nil {
		t.Fatal(err)
	}
	signed, err := agent.SignArtifact(bundlePath, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	manifest := agent.PluginManifest{ID: "estoque-widget", Version: "1.0.0", Name: "Widget", Kind: agent.PluginKindSkill}
	t.Setenv(agent.PluginBundleDirEnv, bundleDir)

	// Pacote adulterado depois da assinatura: 403, nada instalado.
	api := newPluginAPI(t)
	t.Setenv(PluginTrustedKeysEnv, "publisher-1:"+base64.RawStdEncoding.EncodeToString(publicKey))
	if err := os.WriteFile(bundlePath, []byte("pacote adulterado"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, body := bundleRouteRequest(t, api, bundlePath, manifest, signed, "org-a")
	if status != http.StatusForbidden {
		t.Fatalf("tampered package status=%d body=%s", status, body)
	}
	if got := api.plugins.ListForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("tampered package must not install: %+v", got)
	}

	// Chave do pacote não autorizada pelo operador: 403.
	foreignDir := t.TempDir()
	foreignPath := filepath.Join(foreignDir, "widget.zip")
	if err := os.WriteFile(foreignPath, []byte("pacote de outro publicador"), 0o600); err != nil {
		t.Fatal(err)
	}
	foreignSigned, err := agent.SignArtifact(foreignPath, otherPrivate, "outro-publicador")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(agent.PluginBundleDirEnv, foreignDir)
	status, body = bundleRouteRequest(t, api, foreignPath, manifest, foreignSigned, "org-a")
	if status != http.StatusForbidden {
		t.Fatalf("unauthorized publisher status=%d body=%s", status, body)
	}
}
