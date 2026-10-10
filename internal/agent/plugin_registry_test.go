package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func pluginFixtureManifest() PluginManifest {
	return PluginManifest{
		ID:          "estoque-widget",
		Version:     "1.0.0",
		Name:        "Widget de estoque",
		Kind:        PluginKindConnector,
		Description: "Consulta de estoque por conector",
		Scopes:      []string{"connector:external", "workspace:read"},
		License:     "MIT",
	}
}

func namedPluginFixture(id string) PluginManifest {
	manifest := pluginFixtureManifest()
	manifest.ID = id
	return manifest
}

func newPluginRegistryFixture(t *testing.T) *PluginRegistry {
	t.Helper()
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestPluginInstallStoresManifestWithoutGrantingScopes(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	policy := DefaultCapabilityPolicy()

	installation, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installation.Trusted {
		t.Fatal("a fresh install must not be trusted")
	}
	if len(installation.GrantedScopes) != 0 {
		t.Fatalf("catalog is not permission: granted = %+v", installation.GrantedScopes)
	}
	if len(installation.RequestedScopes) != 2 || installation.RequestedScopes[0] != "connector:external" {
		t.Fatalf("requested scopes = %+v", installation.RequestedScopes)
	}
	if installation.Digest == "" {
		t.Fatal("installation must carry a digest")
	}
	if !installation.Enabled {
		t.Fatal("a fresh install is enabled")
	}
	listed := registry.ListForOrganization("org-a")
	if len(listed) != 1 || listed[0].Manifest.ID != "estoque-widget" {
		t.Fatalf("list = %+v", listed)
	}
	// O get devolve cópia: mutar o retorno não muda o registro.
	listed[0].Manifest.Scopes[0] = "hack"
	stored, err := registry.GetForOrganization("org-a", "estoque-widget")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Manifest.Scopes[0] == "hack" {
		t.Fatal("registry leaked internal state through the copy")
	}
}

func TestPluginInstallRejectsInvalidManifestAndUnknownScope(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	policy := DefaultCapabilityPolicy()

	cases := map[string]PluginManifest{
		"sem id":            {Version: "1", Name: "x", Kind: PluginKindSkill},
		"id com separador":  {ID: "a/b", Version: "1", Name: "x", Kind: PluginKindSkill},
		"sem versao":        {ID: "p", Name: "x", Kind: PluginKindSkill},
		"versao com espaco": {ID: "p", Version: "1 0", Name: "x", Kind: PluginKindSkill},
		"tipo invalido":     {ID: "p", Version: "1", Name: "x", Kind: "wasm"},
		"auto dependencia":  {ID: "p", Version: "1", Name: "x", Kind: PluginKindSkill, Dependencies: []string{"p"}},
		"escopo inventado":  {ID: "p", Version: "1", Name: "x", Kind: PluginKindSkill, Scopes: []string{"root:tudo"}},
	}
	for name, manifest := range cases {
		if _, err := registry.InstallForOrganization("org-a", manifest, policy); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
	if _, err := registry.InstallForOrganization("", pluginFixtureManifest(), policy); err == nil {
		t.Fatal("organization is required")
	}
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatalf("valid manifest must install: %v", err)
	}
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); !errors.Is(err, ErrPluginVersionConflict) {
		t.Fatalf("duplicate install error = %v", err)
	}
}

func TestPluginRegistryIsolatesOrganizations(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	policy := DefaultCapabilityPolicy()

	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.InstallForOrganization("org-b", pluginFixtureManifest(), policy); !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("cross-organization install error = %v", err)
	}
	if got := registry.ListForOrganization("org-b"); len(got) != 0 {
		t.Fatalf("org-b must stay empty: %+v", got)
	}
	if _, err := registry.GetForOrganization("org-b", "estoque-widget"); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("get in the wrong organization = %v", err)
	}
	if err := registry.SetEnabledForOrganization("org-b", "estoque-widget", false); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("disable in the wrong organization = %v", err)
	}
	if err := registry.RemoveForOrganization("org-b", "estoque-widget"); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("remove in the wrong organization = %v", err)
	}
}

func TestPluginUpdateRequiresExpectedVersionAndKeepsHistory(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	policy := DefaultCapabilityPolicy()
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatal(err)
	}

	next := pluginFixtureManifest()
	next.Version = "1.1.0"
	if _, err := registry.UpdateForOrganization("org-a", "estoque-widget", "9.9.9", next, policy); !errors.Is(err, ErrPluginVersionConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	updated, err := registry.UpdateForOrganization("org-a", "estoque-widget", "1.0.0", next, policy)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Manifest.Version != "1.1.0" || len(updated.History) != 1 || updated.History[0].Version != "1.0.0" {
		t.Fatalf("update state = %+v", updated)
	}
	if updated.Trusted || len(updated.GrantedScopes) != 0 {
		t.Fatal("a new version must be promoted again")
	}
	if _, err := registry.UpdateForOrganization("org-a", "estoque-widget", "1.1.0", next, policy); !errors.Is(err, ErrPluginVersionConflict) {
		t.Fatalf("reinstalling the same version error = %v", err)
	}
	if _, err := registry.UpdateForOrganization("org-a", "estoque-widget", "1.1.0", PluginManifest{ID: "outro", Version: "2.0.0", Name: "x", Kind: PluginKindSkill}, policy); !errors.Is(err, ErrPluginManifestInvalid) {
		t.Fatalf("id mismatch error = %v", err)
	}
}

func TestPluginRollbackRestoresPreviousVersionOnce(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	policy := DefaultCapabilityPolicy()
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatal(err)
	}
	next := pluginFixtureManifest()
	next.Version = "1.1.0"
	if _, err := registry.UpdateForOrganization("org-a", "estoque-widget", "1.0.0", next, policy); err != nil {
		t.Fatal(err)
	}

	rolledBack, err := registry.RollbackForOrganization("org-a", "estoque-widget")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if rolledBack.Manifest.Version != "1.0.0" || len(rolledBack.History) != 0 {
		t.Fatalf("rollback state = %+v", rolledBack)
	}
	if _, err := registry.RollbackForOrganization("org-a", "estoque-widget"); !errors.Is(err, ErrPluginRollbackDisabled) {
		t.Fatalf("second rollback error = %v", err)
	}
}

func TestPluginPromotionRequiresAuthorizedSignatureAndGrantsScopes(t *testing.T) {
	registry := newPluginRegistryFixture(t)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("plugin-key-1", publicKey)

	// Instalar sem assinatura e tentar promover: recusado.
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PromoteTrustedForOrganization("org-a", "estoque-widget", policy); !errors.Is(err, ErrPluginSignatureReq) {
		t.Fatalf("unsigned promotion error = %v", err)
	}

	// Assinado por chave NÃO autorizada: recusado sem conceder nada.
	_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signedByOther, err := SignPluginManifest(namedPluginFixture("estoque-widget-b"), otherPrivate, "outra-chave")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.InstallForOrganization("org-b", signedByOther, policy); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PromoteTrustedForOrganization("org-b", "estoque-widget-b", policy); !errors.Is(err, ErrPluginKeyUnauthorized) {
		t.Fatalf("unauthorized key error = %v", err)
	}
	if after, err := registry.GetForOrganization("org-b", "estoque-widget-b"); err != nil || after.Trusted || len(after.GrantedScopes) != 0 {
		t.Fatalf("unauthorized promotion must not grant anything: %+v err=%v", after, err)
	}

	// Assinado corretamente: promoção concede exatamente os escopos pedidos.
	signed, err := SignPluginManifest(namedPluginFixture("estoque-widget-c"), privateKey, "plugin-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.InstallForOrganization("org-c", signed, policy); err != nil {
		t.Fatal(err)
	}
	promoted, err := registry.PromoteTrustedForOrganization("org-c", "estoque-widget-c", policy)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if !promoted.Trusted {
		t.Fatal("signed plugin must become trusted")
	}
	if len(promoted.GrantedScopes) != 2 || promoted.GrantedScopes[0] != "connector:external" || promoted.GrantedScopes[1] != "workspace:read" {
		t.Fatalf("granted scopes = %+v", promoted.GrantedScopes)
	}

	// Manifesto adulterado depois de assinado: a verificação falha.
	tampered := signed
	tampered.Name = "outro nome"
	if err := VerifyPluginManifestSignature(tampered, publicKey); !errors.Is(err, ErrPluginSignatureInvalid) {
		t.Fatalf("tampered manifest error = %v", err)
	}
}

func TestPluginRegistryPersistsAndRollsBackOnPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	registry, err := NewPluginRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy()
	if _, err := registry.InstallForOrganization("org-a", pluginFixtureManifest(), policy); err != nil {
		t.Fatal(err)
	}

	// Estado persistido: um registro novo lê o disco.
	reopened, err := NewPluginRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.LoadOrganization("org-a"); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := reopened.ListForOrganization("org-a"); len(got) != 1 || got[0].Manifest.Version != "1.0.0" {
		t.Fatalf("reloaded state = %+v", got)
	}

	// Falha de persistência precisa devolver erro sem deixar estado inventado.
	pluginsDir := filepath.Join(root, "plugins")
	if err := os.RemoveAll(pluginsDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginsDir, []byte("bloqueio"), 0o600); err != nil {
		t.Fatal(err)
	}
	next := pluginFixtureManifest()
	next.Version = "2.0.0"
	if _, err := registry.UpdateForOrganization("org-a", "estoque-widget", "1.0.0", next, policy); err == nil {
		t.Fatal("expected a persistence failure")
	}
	after, err := registry.GetForOrganization("org-a", "estoque-widget")
	if err != nil {
		t.Fatal(err)
	}
	if after.Manifest.Version != "1.0.0" || len(after.History) != 0 || after.Trusted {
		t.Fatalf("failed update must leave the previous state intact: %+v", after)
	}
	if err := registry.SetEnabledForOrganization("org-a", "estoque-widget", false); err == nil {
		t.Fatal("expected a persistence failure while disabling")
	}
	if got, err := registry.GetForOrganization("org-a", "estoque-widget"); err != nil || !got.Enabled {
		t.Fatalf("failed disable must keep the plugin enabled: %+v err=%v", got, err)
	}
}
