package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func bundleFixture(t *testing.T, payload string) (string, SignedArtifact, ed25519.PublicKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := t.TempDir()
	path := filepath.Join(bundleDir, "plugin.zip")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	signed, err := SignArtifact(path, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	return path, signed, publicKey
}

func TestVerifyPluginBundleRequiresAuthorizedKeyAndIntactFile(t *testing.T) {
	path, signed, publicKey := bundleFixture(t, "conteudo do plugin")
	policy := DefaultCapabilityPolicy()

	// Sem chave autorizada: recusado.
	if _, err := VerifyPluginBundle(path, signed, policy); !errors.Is(err, ErrPluginKeyUnauthorized) {
		t.Fatalf("unauthorized key error = %v", err)
	}
	// Chave autorizada: digest devolvido.
	trusted := policy.WithTrustedSkillKey("publisher-1", publicKey)
	digest, err := VerifyPluginBundle(path, signed, trusted)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if digest != signed.SHA256 {
		t.Fatalf("digest = %q, want %q", digest, signed.SHA256)
	}
	// Arquivo alterado depois de assinado: recusado.
	if err := os.WriteFile(path, []byte("conteudo adulterado"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPluginBundle(path, signed, trusted); !errors.Is(err, ErrPluginSignatureInvalid) {
		t.Fatalf("tampered bundle error = %v", err)
	}
	// Sem assinatura declarada: recusado com erro explícito.
	if _, err := VerifyPluginBundle(path, SignedArtifact{}, trusted); !errors.Is(err, ErrPluginSignatureReq) {
		t.Fatalf("missing signature error = %v", err)
	}
	// Caminho inexistente: indisponível, não "válido".
	if _, err := VerifyPluginBundle(filepath.Join(t.TempDir(), "nao-existe.zip"), signed, trusted); !errors.Is(err, ErrPluginBundleUnavailable) {
		t.Fatalf("missing file error = %v", err)
	}
}

func TestResolvePluginBundlePathStaysInsideAllowedDirectory(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "plugin.zip")
	if err := os.WriteFile(inside, []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolvePluginBundlePath(root, inside)
	if err != nil {
		t.Fatalf("inside path must resolve: %v", err)
	}
	if resolved == "" {
		t.Fatal("resolved path is empty")
	}
	outside := filepath.Join(t.TempDir(), "alheio.zip")
	if err := os.WriteFile(outside, []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolvePluginBundlePath(root, outside); !errors.Is(err, ErrPluginBundleUnavailable) {
		t.Fatalf("outside path must be refused, got %v", err)
	}
	if _, err := ResolvePluginBundlePath(root, filepath.Join(root, "..", "escape.zip")); !errors.Is(err, ErrPluginBundleUnavailable) {
		t.Fatalf("relative escape must be refused, got %v", err)
	}
	// Sem diretório permitido configurado: falha fechada.
	if _, err := ResolvePluginBundlePath("", inside); !errors.Is(err, ErrPluginBundleUnavailable) {
		t.Fatalf("missing allowlist must fail closed, got %v", err)
	}
}

func TestPluginInstallBundleVerifiesAndStoresContent(t *testing.T) {
	path, signed, publicKey := bundleFixture(t, "conteudo verificado")
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("publisher-1", publicKey)

	installation, err := registry.InstallBundleForOrganization("org-a", namedPluginFixture("bundle-plugin"), path, signed, policy)
	if err != nil {
		t.Fatalf("install bundle: %v", err)
	}
	if !installation.ContentVerified || installation.ContentSHA256 != signed.SHA256 {
		t.Fatalf("content metadata = %+v", installation)
	}
	if installation.ContentPath == "" {
		t.Fatal("installed bundle must record where the content lives")
	}
	if _, err := os.Stat(installation.ContentPath); err != nil {
		t.Fatalf("stored content missing: %v", err)
	}
	// Digest recomputado confere.
	if err := registry.VerifyInstalledContent("org-a", "bundle-plugin"); err != nil {
		t.Fatalf("verify installed content: %v", err)
	}
	// Adulterar o pacote guardado é detectado.
	if err := os.WriteFile(installation.ContentPath, []byte("adulterado"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := registry.VerifyInstalledContent("org-a", "bundle-plugin"); !errors.Is(err, ErrPluginSignatureInvalid) {
		t.Fatalf("post-install tamper error = %v", err)
	}
}

func TestPluginInstallBundleRejectsTamperedAndRollsBackOnCopyFailure(t *testing.T) {
	path, signed, publicKey := bundleFixture(t, "conteudo bom")
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("publisher-1", publicKey)

	// Assinatura inválida para o arquivo atual: nada é instalado.
	if err := os.WriteFile(path, []byte("outro conteudo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.InstallBundleForOrganization("org-a", namedPluginFixture("bundle-plugin"), path, signed, policy); !errors.Is(err, ErrPluginSignatureInvalid) {
		t.Fatalf("tampered install error = %v", err)
	}
	if got := registry.ListForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("refused install must not register the plugin: %+v", got)
	}

	// Conteúdo válido, mas o diretório de conteúdo é um ARQUIVO: a cópia falha e
	// a instalação precisa ser revertida.
	path2, signed2, publicKey2 := bundleFixture(t, "conteudo valido")
	registry2, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy2 := DefaultCapabilityPolicy().WithTrustedSkillKey("publisher-1", publicKey2)
	contentRoot := filepath.Join(registry2.root, "plugins", "content")
	if err := os.MkdirAll(filepath.Dir(contentRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contentRoot, []byte("bloqueio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := registry2.InstallBundleForOrganization("org-a", namedPluginFixture("bundle-plugin"), path2, signed2, policy2); err == nil {
		t.Fatal("expected a content copy failure")
	}
	if got := registry2.ListForOrganization("org-a"); len(got) != 0 {
		t.Fatalf("failed content copy must roll back the installation: %+v", got)
	}
}
