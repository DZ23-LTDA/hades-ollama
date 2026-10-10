package agent

import (
	"context"
	"os"
	"strings"
	"testing"
)

// oauthLiveEnv descreve o contrato de ambiente do teste live de OAuth do
// aceite P1-5. Todos os campos são obrigatórios: nenhuma metade de credencial
// produz um teste parcial que falharia por configuração incompleta.
type oauthLiveEnv struct {
	ProviderName string
	AuthorizeURL string
	TokenURL     string
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// oauthLiveEnvFromEnvironment lê o contrato opt-in. O primeiro retorno é falso
// quando o portão OLLAMA_TEST_OAUTH não está ativo ou quando falta qualquer
// credencial, e nesse caso o motivo explica o que ficou faltando.
func oauthLiveEnvFromEnvironment() (oauthLiveEnv, string) {
	if strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH")) != "1" {
		return oauthLiveEnv{}, "teste live de OAuth desativado: exporte OLLAMA_TEST_OAUTH=1 para habilitá-lo"
	}
	env := oauthLiveEnv{
		ProviderName: strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_PROVIDER")),
		AuthorizeURL: strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_AUTHORIZE_URL")),
		TokenURL:     strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_TOKEN_URL")),
		ClientID:     strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_CLIENT_SECRET")),
		RefreshToken: strings.TrimSpace(os.Getenv("OLLAMA_TEST_OAUTH_REFRESH_TOKEN")),
	}
	var missing []string
	for _, field := range []struct {
		name  string
		value string
	}{
		{"OLLAMA_TEST_OAUTH_PROVIDER", env.ProviderName},
		{"OLLAMA_TEST_OAUTH_AUTHORIZE_URL", env.AuthorizeURL},
		{"OLLAMA_TEST_OAUTH_TOKEN_URL", env.TokenURL},
		{"OLLAMA_TEST_OAUTH_CLIENT_ID", env.ClientID},
		{"OLLAMA_TEST_OAUTH_CLIENT_SECRET", env.ClientSecret},
		{"OLLAMA_TEST_OAUTH_REFRESH_TOKEN", env.RefreshToken},
	} {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return oauthLiveEnv{}, "teste live de OAuth sem credencial completa; faltando: " + strings.Join(missing, ", ")
	}
	return env, ""
}

func TestOAuthLiveEnvironmentRequiresEveryCredential(t *testing.T) {
	for _, name := range []string{
		"OLLAMA_TEST_OAUTH",
		"OLLAMA_TEST_OAUTH_PROVIDER",
		"OLLAMA_TEST_OAUTH_AUTHORIZE_URL",
		"OLLAMA_TEST_OAUTH_TOKEN_URL",
		"OLLAMA_TEST_OAUTH_CLIENT_ID",
		"OLLAMA_TEST_OAUTH_CLIENT_SECRET",
		"OLLAMA_TEST_OAUTH_REFRESH_TOKEN",
	} {
		t.Setenv(name, "")
	}
	if _, reason := oauthLiveEnvFromEnvironment(); reason == "" {
		t.Fatal("o teste live precisa ficar desativado quando o portão OLLAMA_TEST_OAUTH está ausente")
	}
	t.Setenv("OLLAMA_TEST_OAUTH", "1")
	if _, reason := oauthLiveEnvFromEnvironment(); reason == "" {
		t.Fatal("o portão ligado sem credenciais precisa continuar desativado, e não falhar")
	}
	t.Setenv("OLLAMA_TEST_OAUTH_PROVIDER", "github")
	t.Setenv("OLLAMA_TEST_OAUTH_AUTHORIZE_URL", "https://idp.example.test/authorize")
	t.Setenv("OLLAMA_TEST_OAUTH_TOKEN_URL", "https://idp.example.test/token")
	t.Setenv("OLLAMA_TEST_OAUTH_CLIENT_ID", "fixture-client")
	t.Setenv("OLLAMA_TEST_OAUTH_CLIENT_SECRET", "fixture-secret")
	if _, reason := oauthLiveEnvFromEnvironment(); reason == "" {
		t.Fatal("sem o token de atualização a configuração live precisa permanecer incompleta")
	}
	t.Setenv("OLLAMA_TEST_OAUTH_REFRESH_TOKEN", "fixture-refresh")
	env, reason := oauthLiveEnvFromEnvironment()
	if reason != "" {
		t.Fatalf("configuração live completa foi recusada: %s", reason)
	}
	if env.ProviderName != "github" || env.TokenURL == "" || env.RefreshToken == "" {
		t.Fatalf("configuração live lida de forma incorreta: %+v", env)
	}
}

// TestOAuthLiveRefreshAgainstRealProvider executa a renovação real de token
// contra o provedor configurado pelo operador. Ele é pulado, nunca falho,
// quando o portão ou qualquer credencial está ausente.
func TestOAuthLiveRefreshAgainstRealProvider(t *testing.T) {
	env, reason := oauthLiveEnvFromEnvironment()
	if reason != "" {
		t.Skip(reason)
	}
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "oauth-live-test-key-long-enough")
	provider := OAuthProvider{
		Name:         env.ProviderName,
		AuthorizeURL: env.AuthorizeURL,
		TokenURL:     env.TokenURL,
		ClientIDEnv:  "OLLAMA_TEST_OAUTH_CLIENT_ID",
		SecretEnv:    "OLLAMA_TEST_OAUTH_CLIENT_SECRET",
	}
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.CreateUser("oauth-live@example.com", "OAuth Live")
	if err != nil {
		t.Fatal(err)
	}
	organization, _, err := store.CreateOrganization("OAuth Live", owner)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := store.StoreOAuthCredential(provider.Name, owner.ID, organization.ID, map[string]any{
		"access_token":  "live-access-placeholder",
		"refresh_token": env.RefreshToken,
		"expires_in":    float64(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := store.RefreshOAuthCredentialForOrganization(context.Background(), organization.ID, provider, credential.ID, nil)
	if err != nil {
		t.Fatalf("renovação live falhou: %v", err)
	}
	if rotated.AccessTokenCiphertext == credential.AccessTokenCiphertext {
		t.Fatal("a renovação live não trocou o token de acesso")
	}
	access, _, err := store.OAuthAccessTokenForOrganization(organization.ID, provider.Name)
	if err != nil || access == "" {
		t.Fatalf("token de acesso live não ficou disponível: err=%v", err)
	}
	if access == "live-access-placeholder" {
		t.Fatal("o token de acesso continuou sendo o marcador local")
	}
}
