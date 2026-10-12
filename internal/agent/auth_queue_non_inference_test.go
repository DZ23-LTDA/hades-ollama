package agent

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

// NÃO INFERÊNCIA EM AUTENTICAÇÃO E FILA (E-198).
//
// E-195 unificou as recusas de tenant em 6 métodos e **declarou um residual honesto**: `auth` e `queue` ficaram com a
// correção coberta apenas pelas suítes, **sem teste de comparação dedicado**. Residual declarado que não é fechado
// vira dívida esquecida. Este arquivo fecha os dois:
//
//   - `AuthStore.RevokeOAuthCredentialForOrganization`: "credencial não existe" devolvia `os.ErrNotExist` e "credencial
//     de outro tenant" devolvia `oauth credential is outside the active organization`;
//   - `JobQueue.ReplayForOrganization`: "job não existe" devolvia `os.ErrNotExist` e "job de outro tenant" devolvia
//     `ErrQueueJobForbidden` — e, pior, a checagem de organização vinha DEPOIS da checagem de status.
//
// O teste compara as duas recusas: se voltarem a divergir, ele falha.

func TestOAuthRevocationDoesNotRevealExistence(t *testing.T) {
	// Chave de criptografia de FIXTURE DE TESTE (não é segredo real): a store exige uma configurada para guardar
	// credencial. Sem ela, `StoreOAuthCredential` recusa por configuração — outra falha, não o oráculo sob teste.
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "chave-de-fixture-de-teste-32-caracteres")
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A credencial precisa de um usuário com vínculo real na organização: a store recusa (corretamente) credencial de
	// quem não é membro — e essa recusa é outra coisa, não o oráculo que estamos medindo.
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
	if credential.ID == "" {
		t.Fatal("a credencial de teste precisa ter ID")
	}

	const intrusa = "org-intrusa"
	const inexistente = "cred_que_nao_existe"
	provider := OAuthProvider{Name: "teste"}

	errOutro := store.RevokeOAuthCredentialForOrganization(context.Background(), intrusa, credential.ID, provider, http.DefaultClient)
	errInexistente := store.RevokeOAuthCredentialForOrganization(context.Background(), intrusa, inexistente, provider, http.DefaultClient)

	if errOutro == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir (outro=%v, inexistente=%v)", errOutro, errInexistente)
	}
	if errOutro.Error() != errInexistente.Error() {
		t.Fatalf(
			"ORÁCULO DE ENUMERAÇÃO na revogação de OAuth: recusa de outro tenant (%q) difere da recusa de inexistente (%q)",
			errOutro.Error(), errInexistente.Error())
	}
}

// CONTROLE POSITIVO: o dono continua revogando a própria credencial.
func TestOwnerStillRevokesOwnCredential(t *testing.T) {
	// Chave de criptografia de FIXTURE DE TESTE (não é segredo real): a store exige uma configurada para guardar
	// credencial. Sem ela, `StoreOAuthCredential` recusa por configuração — outra falha, não o oráculo sob teste.
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "chave-de-fixture-de-teste-32-caracteres")
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A credencial precisa de um usuário com vínculo real na organização: a store recusa (corretamente) credencial de
	// quem não é membro — e essa recusa é outra coisa, não o oráculo que estamos medindo.
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
	if err := store.RevokeOAuthCredentialForOrganization(context.Background(), organizacao.ID, credential.ID, OAuthProvider{Name: "teste"}, http.DefaultClient); err != nil {
		t.Fatalf("o dono precisa revogar a própria credencial: %v", err)
	}
}

// jobEmDLQ leva um job da organização até a DLQ pelo caminho público (MaxAttempts=1 e uma falha).
func jobEmDLQ(t *testing.T, queue *JobQueue, organizationID string) QueueJob {
	t.Helper()
	if _, err := queue.EnqueueForOrganization(organizationID, "mission-1", 1); err != nil {
		t.Fatalf("enfileirar job: %v", err)
	}
	claim, ok, err := queue.Claim("worker-1", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("reivindicar job: ok=%v err=%v", ok, err)
	}
	nacked, err := queue.Nack(claim, errors.New("falha definitiva"))
	if err != nil {
		t.Fatalf("levar job a DLQ: %v", err)
	}
	if nacked.Status != QueueDeadLetter {
		t.Fatalf("o job de teste precisa terminar em DLQ, ficou %q", nacked.Status)
	}
	return nacked
}

func TestQueueReplayDoesNotRevealExistence(t *testing.T) {
	queue, err := NewJobQueue(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	claim := jobEmDLQ(t, queue, "org-a")

	const intrusa = "org-intrusa"
	const inexistente = "job_que_nao_existe"

	_, errOutro := queue.ReplayForOrganization(intrusa, claim.ID)
	_, errInexistente := queue.ReplayForOrganization(intrusa, inexistente)

	if errOutro == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir (outro=%v, inexistente=%v)", errOutro, errInexistente)
	}
	if errOutro.Error() != errInexistente.Error() {
		t.Fatalf(
			"ORÁCULO DE ENUMERAÇÃO no replay de fila: recusa de outro tenant (%q) difere da recusa de inexistente (%q)",
			errOutro.Error(), errInexistente.Error())
	}
}

// CONTROLE POSITIVO: o dono continua reprocessando o próprio DLQ.
func TestOwnerStillReplaysOwnDeadLetter(t *testing.T) {
	queue, err := NewJobQueue(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	claim := jobEmDLQ(t, queue, "org-a")
	if _, err := queue.ReplayForOrganization("org-a", claim.ID); err != nil {
		t.Fatalf("o dono precisa reprocessar o próprio DLQ: %v", err)
	}
}
