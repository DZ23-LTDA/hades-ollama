package agent

import (
	"testing"
	"time"
)

const secTestVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"

// TestOAuthStateSweepRemovesExpired locks in SEC-06: starting a new OAuth flow
// sweeps expired/consumed states so the persisted store cannot grow without
// bound.
func TestOAuthStateSweepRemovesExpired(t *testing.T) {
	store, err := NewAuthStore("")
	if err != nil {
		t.Fatal(err)
	}
	const uri = "https://app.example.test/oauth/callback"
	if _, _, err := store.CreateOAuthState("github", uri, secTestVerifier, "", time.Minute); err != nil {
		t.Fatal(err)
	}
	// Force the existing state to be expired.
	store.mu.Lock()
	for h, s := range store.oauthStates {
		s.ExpiresAt = time.Now().UTC().Add(-time.Hour)
		store.oauthStates[h] = s
	}
	before := len(store.oauthStates)
	store.mu.Unlock()
	if before == 0 {
		t.Fatal("expected a pending state before sweep")
	}

	// Creating a new state must sweep the expired one first.
	if _, _, err := store.CreateOAuthState("github", uri, secTestVerifier, "", time.Minute); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	after := len(store.oauthStates)
	store.mu.Unlock()
	if after != 1 {
		t.Errorf("after sweep+create, expected only the new state (1), got %d", after)
	}
}

// TestProvisionOAuthUserBindsSubjectNotEmail locks in SEC-04: a federated
// account is keyed by (provider, subject), never by email alone. The same
// (provider, subject) always resolves to the same local user, and a different
// sign-in that merely asserts the same email must NOT silently take over the
// existing account.
func TestProvisionOAuthUserBindsSubjectNotEmail(t *testing.T) {
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// A stable subject is mandatory.
	if _, _, _, err := store.ProvisionOAuthUser(map[string]any{"email": "a@example.test"}, "github"); err == nil {
		t.Fatal("provisioning without a subject must be rejected (SEC-04)")
	}

	first, _, _, err := store.ProvisionOAuthUser(map[string]any{"sub": "idp-subject-1", "email": "a@example.test", "name": "A"}, "github")
	if err != nil {
		t.Fatal(err)
	}

	// Same provider+subject (even with a changed email) → same local user.
	again, _, _, err := store.ProvisionOAuthUser(map[string]any{"sub": "idp-subject-1", "email": "renamed@example.test"}, "github")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Fatalf("same (provider, subject) must resolve to the same user: %s vs %s", again.ID, first.ID)
	}

	// A different sign-in asserting the same original email must be refused,
	// not merged into the existing account (account-takeover vector).
	if _, _, _, err := store.ProvisionOAuthUser(map[string]any{"sub": "attacker-subject", "email": "a@example.test"}, "google"); err == nil {
		t.Fatal("a different identity claiming an existing email must be rejected (SEC-04)")
	}

	// The subject can also arrive nested in the OIDC id_token claims.
	nested, _, _, err := store.ProvisionOAuthUser(map[string]any{"id_token_claims": map[string]any{"sub": "oidc-subject-2", "email": "b@example.test"}}, "okta")
	if err != nil {
		t.Fatal(err)
	}
	if nested.ID == "" || nested.ID == first.ID {
		t.Fatalf("nested-claim provisioning produced an unexpected user: %+v", nested)
	}
}
