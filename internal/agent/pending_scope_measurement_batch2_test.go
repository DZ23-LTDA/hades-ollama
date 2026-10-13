package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// SEGUNDO LOTE DE MEDIÇÃO POR COMPORTAMENTO (E-340): crm e inbox.
//
// `consent_registry` FOI RETIRADO deste lote: a suíte mostrou que há CONTRATO TESTADO de normalizar organização
// vazia para a organização local, logo a guarda de escopo cru contradizia o contrato do repositório — *contrato
// testado ganha de preferência minha*, e o arquivo voltou ao registro da catraca com esse motivo declarado.
//
// A guarda roda ANTES de qualquer uso dos demais parâmetros, então é legítimo passar valores de relleno e exigir o
// motivo declarado (`ErrTenantScopeRequired`) — *se a recusa viesse por outro motivo, a guarda estaria passando por
// acidente, e este teste falharia*.

func TestSegundoLoteRecusaEscopoAusente(t *testing.T) {
	t.Run("crm", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "crm")
		store, err := NewCRMStore(raiz)
		if err != nil {
			t.Fatal(err)
		}
		// `GetContact` devolve (CRMContact, error): com organização vazia a recusa vem antes do uso do ID.
		if _, err := store.GetContact("", "contato-qualquer"); !errors.Is(err, ErrTenantScopeRequired) {
			t.Fatalf("crm: leitura sem organização precisa ser recusada pelo MOTIVO declarado: %v", err)
		}
		// Leitura em lista não devolve nada sem tenant (fail-closed, sem erro na assinatura).
		if contatos := store.ListContacts("", ""); len(contatos) != 0 {
			t.Fatalf("crm: lista sem organização precisa vir vazia, não compartilhada: %d", len(contatos))
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("inbox", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "inbox")
		store, err := NewInboxStore(raiz)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Assign("", "conversa-qualquer", "agente", "humano", time.Now().UTC()); !errors.Is(err, ErrTenantScopeRequired) {
			t.Fatalf("inbox: atribuição sem organização precisa ser recusada pelo motivo declarado: %v", err)
		}
		if _, err := store.Close("", "conversa-qualquer", "ator", "motivo", time.Now().UTC()); !errors.Is(err, ErrTenantScopeRequired) {
			t.Fatalf("inbox: fechamento sem organização precisa ser recusado: %v", err)
		}
		if conversas := store.ListConversations("", ""); len(conversas) != 0 {
			t.Fatalf("inbox: lista sem organização precisa vir vazia: %d", len(conversas))
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

}

// O inverso: com organização válida o caminho continua funcionando (a guarda não pode virar bloqueio geral).
func TestSegundoLoteComOrganizacaoValidaFunciona(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "crm")
	store, err := NewCRMStore(raiz)
	if err != nil {
		t.Fatal(err)
	}
	// ListContacts de organização válida não precisa explodir e o arquivo do tenant pode ser criado por escrita real.
	if contatos := store.ListContacts("org-a", ""); len(contatos) != 0 {
		t.Fatalf("organização nova começa vazia: %d", len(contatos))
	}
}
