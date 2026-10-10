package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOAuthClientStoreRoundTripPersistsWithPrivateMode(t *testing.T) {
	root := t.TempDir()
	store, err := NewOAuthClientStore(root)
	if err != nil {
		t.Fatalf("criar store falhou: %v", err)
	}
	if err := store.Save("GitHub", " client-fixture ", " secret-fixture "); err != nil {
		t.Fatalf("Save falhou: %v", err)
	}
	clientID, clientSecret, ok := store.Get("github")
	if !ok || clientID != "client-fixture" || clientSecret != "secret-fixture" {
		t.Fatalf("Get devolveu ok=%v client_id=%q; o provedor deveria ser normalizado e os valores aparados", ok, clientID)
	}
	path := filepath.Join(root, "oauth-clients.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("o arquivo de credenciais não foi persistido: %v", err)
	}
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("o arquivo de credenciais deveria ser 0600, veio %o", mode)
		}
	}
	reloaded, err := NewOAuthClientStore(root)
	if err != nil {
		t.Fatalf("recarregar store falhou: %v", err)
	}
	if !reloaded.Configured("GITHUB") {
		t.Fatal("a credencial persistida não sobreviveu ao reload")
	}
	if got := reloaded.Providers(); len(got) != 1 || got[0] != "github" {
		t.Fatalf("Providers após reload = %v, esperado [github]", got)
	}
}

func TestOAuthClientStoreRejectsIncompleteAndMultilineValues(t *testing.T) {
	store, err := NewOAuthClientStore("")
	if err != nil {
		t.Fatalf("criar store em memória falhou: %v", err)
	}
	cases := []struct {
		name                       string
		provider, clientID, secret string
	}{
		{"provedor vazio", "   ", "client-fixture", "secret-fixture"},
		{"client_id vazio", "github", "  ", "secret-fixture"},
		{"client_secret vazio", "github", "client-fixture", ""},
		{"client_id com quebra de linha", "github", "line\r\nbreak", "secret-fixture"},
		{"client_secret com nulo", "github", "client-fixture", "nul\x00byte"},
	}
	for _, tc := range cases {
		if err := store.Save(tc.provider, tc.clientID, tc.secret); err == nil {
			t.Fatalf("%s: Save deveria falhar", tc.name)
		}
	}
	if len(store.Providers()) != 0 {
		t.Fatalf("nenhuma credencial inválida deveria ter sido gravada, veio %v", store.Providers())
	}
}

func TestOAuthClientStoreIsNilSafeAndInMemoryWhenRootIsEmpty(t *testing.T) {
	var nilStore *OAuthClientStore
	if err := nilStore.Save("github", "client-fixture", "secret-fixture"); err == nil {
		t.Fatal("Save em store nulo deveria falhar")
	}
	if _, _, ok := nilStore.Get("github"); ok {
		t.Fatal("Get em store nulo deveria devolver ok=false")
	}
	if nilStore.Configured("github") {
		t.Fatal("Configured em store nulo deveria ser falso")
	}
	if got := nilStore.Providers(); got != nil {
		t.Fatalf("Providers em store nulo deveria ser nil, veio %v", got)
	}

	store, err := NewOAuthClientStore("")
	if err != nil {
		t.Fatalf("criar store em memória falhou: %v", err)
	}
	if store.path != "" {
		t.Fatalf("root vazio deveria manter o store só em memória, path=%q", store.path)
	}
	if err := store.Save("github", "client-fixture", "secret-fixture"); err != nil {
		t.Fatalf("Save em memória falhou: %v", err)
	}
	if !store.Configured("github") {
		t.Fatal("o store em memória deveria servir a credencial recém-gravada")
	}
}

func TestOAuthClientStoreListsSortedProvidersWithoutSecrets(t *testing.T) {
	store, err := NewOAuthClientStore("")
	if err != nil {
		t.Fatalf("criar store em memória falhou: %v", err)
	}
	for _, provider := range []string{"Zoho", "github", "Notion"} {
		if err := store.Save(provider, "client-"+provider, "secret-"+provider); err != nil {
			t.Fatalf("Save(%s) falhou: %v", provider, err)
		}
	}
	got := store.Providers()
	want := []string{"github", "notion", "zoho"}
	if len(got) != len(want) {
		t.Fatalf("Providers = %v, esperado %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Providers = %v, esperado %v", got, want)
		}
	}
	for _, name := range got {
		if name == "client-"+name || name == "secret-"+name {
			t.Fatalf("Providers vazou material de credencial: %q", name)
		}
	}
}

func TestOAuthClientStoreIgnoresPartialEntriesAndReadsBlankFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "oauth-clients.json")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("preparar arquivo em branco falhou: %v", err)
	}
	store, err := NewOAuthClientStore(root)
	if err != nil {
		t.Fatalf("arquivo em branco deveria ser aceito: %v", err)
	}
	if len(store.Providers()) != 0 {
		t.Fatalf("arquivo em branco não deveria produzir provedores: %v", store.Providers())
	}

	// Uma entrada parcial (sem secret) não pode ser considerada configurada nem
	// aparecer na listagem, mesmo vindo do arquivo.
	partial, err := json.Marshal(map[string]oauthClientEntry{"github": {ClientID: "client-fixture"}})
	if err != nil {
		t.Fatalf("montar fixture falhou: %v", err)
	}
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatalf("preparar entrada parcial falhou: %v", err)
	}
	partialStore, err := NewOAuthClientStore(root)
	if err != nil {
		t.Fatalf("entrada parcial deveria carregar sem erro: %v", err)
	}
	if partialStore.Configured("github") {
		t.Fatal("entrada parcial não deveria contar como configurada")
	}
	if _, _, ok := partialStore.Get("github"); ok {
		t.Fatal("Get não deveria devolver uma entrada parcial")
	}
	if got := partialStore.Providers(); len(got) != 0 {
		t.Fatalf("entrada parcial não deveria ser listada: %v", got)
	}

	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("preparar JSON inválido falhou: %v", err)
	}
	if _, err := NewOAuthClientStore(root); err == nil {
		t.Fatal("JSON inválido deveria falhar alto em vez de silenciar")
	}
}

func TestOAuthClientStoreRollsBackEntryWhenPersistFails(t *testing.T) {
	root := t.TempDir()
	// Um arquivo comum no lugar do diretório faz o persist falhar de forma
	// portátil, exercitando o caminho de rollback do Save.
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("preparar bloqueio falhou: %v", err)
	}
	store := &OAuthClientStore{
		entries: map[string]oauthClientEntry{"github": {ClientID: "client-fixture", ClientSecret: "secret-fixture"}},
		path:    filepath.Join(blocked, "oauth-clients.json"),
	}
	if err := store.Save("gitlab", "client-gitlab", "secret-gitlab"); err == nil {
		t.Fatal("Save deveria falhar quando o persist falha")
	}
	if _, _, ok := store.Get("gitlab"); ok {
		t.Fatal("a entrada nova deveria ser revertida quando o persist falha")
	}
	if _, secret, ok := store.Get("github"); !ok || secret != "secret-fixture" {
		t.Fatal("a credencial anterior deveria permanecer intacta após a falha")
	}
}
