package agent

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// ORÁCULO DE EXISTÊNCIA ENTRE TENANTS NO REFRESH DE OAUTH (E-252).
//
// E-195 unificou as recusas de tenant e E-198 fechou o teste de comparação da REVOGAÇÃO. O refresh ficou de fora, e
// a ordem das checagens em `refreshOAuthCredential` deixa um oráculo: a checagem de PROVIDER vem ANTES da de
// ORGANIZAÇÃO. Com um provider errado, um intruso distingue
//
//	"este ID existe (em algum outro tenant)"  -> "oauth credential provider mismatch"
//	"este ID não existe"                      -> "oauth credential is outside the active organization"
//
// e assim enumera credenciais de OUTRA organização. É a mesma família de defeito que E-198 corrigiu na fila (a
// checagem de organização vinha depois da de status).
//
// O teste compara as duas recusas: se voltarem a divergir, ele falha.
func TestOAuthRefreshDoesNotRevealExistenceAcrossTenants(t *testing.T) {
	// Chave de criptografia de FIXTURE DE TESTE (não é segredo real).
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "chave-de-fixture-de-teste-32-caracteres")
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	usuario, err := store.CreateUser("titular@teste.invalid", "Titular")
	if err != nil {
		t.Fatal(err)
	}
	organizacao, _, err := store.CreateOrganization("Org A", usuario)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := store.StoreOAuthCredential("teste", usuario.ID, organizacao.ID, map[string]any{
		"access_token":  "token-de-teste",
		"refresh_token": "refresh-de-teste",
		"expires_in":    3600,
	})
	if err != nil {
		t.Fatalf("criar credencial de teste: %v", err)
	}

	const intrusa = "org-intrusa"
	const inexistente = "cred_que_nao_existe"
	// Provider ERRADO de propósito: é o caminho que expõe a ordem das checagens.
	providerErrado := OAuthProvider{Name: "outro-provider"}

	_, errExistente := store.RefreshOAuthCredentialForOrganization(context.Background(), intrusa, providerErrado, credential.ID, http.DefaultClient)
	_, errInexistente := store.RefreshOAuthCredentialForOrganization(context.Background(), intrusa, providerErrado, inexistente, http.DefaultClient)

	if errExistente == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir (existente=%v, inexistente=%v)", errExistente, errInexistente)
	}
	if errExistente.Error() != errInexistente.Error() {
		t.Fatalf(
			"ORÁCULO DE EXISTÊNCIA ENTRE TENANTS no refresh de OAuth: recusa de credencial de outro tenant (%q) difere da recusa de inexistente (%q)",
			errExistente.Error(), errInexistente.Error())
	}
}

// CONTROLE POSITIVO: o dono continua vendo o próprio erro de provider (dado dele, não recusa de tenant).
func TestOwnerRefreshStillSeesOwnProviderMismatch(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "chave-de-fixture-de-teste-32-caracteres")
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	usuario, err := store.CreateUser("titular@teste.invalid", "Titular")
	if err != nil {
		t.Fatal(err)
	}
	organizacao, _, err := store.CreateOrganization("Org A", usuario)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := store.StoreOAuthCredential("teste", usuario.ID, organizacao.ID, map[string]any{
		"access_token":  "token-de-teste",
		"refresh_token": "refresh-de-teste",
		"expires_in":    3600,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RefreshOAuthCredentialForOrganization(context.Background(), organizacao.ID, OAuthProvider{Name: "outro-provider"}, credential.ID, http.DefaultClient)
	if err == nil {
		t.Fatal("o dono com provider errado precisa receber erro de provider")
	}
	if strings.Contains(err.Error(), "outside the active organization") {
		t.Fatalf("a recusa do DONO não pode ser a recusa de tenant: %v", err)
	}
	if !strings.Contains(err.Error(), "provider mismatch") {
		t.Fatalf("o dono precisa ver o erro de provider do PRÓPRIO dado: %v", err)
	}
}
