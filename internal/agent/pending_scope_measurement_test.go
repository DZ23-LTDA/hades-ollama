package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// MEDIÇÃO POR COMPORTAMENTO DAS PENDÊNCIAS DA CATRACA (E-339).
//
// A catraca de E-338 declarou 16 arquivos como `NAO_MEDIDO`. Aqui a lista **encolhe de verdade**: os sete
// armazenamentos com assinatura uniforme (`New*Store(root)` + `Load(org)` + `Save(org, payload)`) são atacados com
// organização **vazia** e precisam devolver **erro** — e deixar o diretório **sem** arquivo compartilhado.
//
// ***Declarar a dívida foi o passo anterior; este passo é o pagamento.***

func medirArmazenamento(t *testing.T, raiz string, carregar func(org string) error, salvar func(org string) error) {
	t.Helper()
	if err := carregar(""); err == nil {
		t.Fatal("leitura sem organização precisa ser recusada (ler arquivo compartilhado é vazamento entre tenants)")
	}
	if err := salvar(""); err == nil {
		t.Fatal("gravação sem organização precisa ser recusada (gravar em arquivo compartilhado mistura tenants)")
	}
	assertSemArquivoCompartilhado(t, raiz)
	// E com organização válida funciona: a guarda não pode virar bloqueio geral.
	if err := salvar("org-a"); err != nil {
		t.Fatalf("organização válida precisa funcionar: %v", err)
	}
	if err := carregar("org-a"); err != nil {
		t.Fatalf("leitura com organização válida precisa funcionar: %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, sanitizeIDComponent("org-a")+".json")); err != nil {
		t.Fatalf("cada organização precisa ter o próprio arquivo: %v", err)
	}
}

func TestPendenciasDaCatracaMedidasPorComportamento(t *testing.T) {
	t.Run("service_agent_audit", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "audit")
		store := NewServiceAgentAuditStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("service_agent_approvals", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "apv")
		store := NewServiceAgentApprovalStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("omnichannel_dlq", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "dlq")
		store := NewOmnichannelDLQStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("omnichannel_dedupe", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "dedupe")
		store := NewOmnichannelDedupeStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("omnichannel_connectors", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "conn")
		store := NewOmnichannelConnectorStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("omnichannel_ordering_store", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "ord")
		store := NewOmnichannelOrderingStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})

	t.Run("omnichannel_rate_limit", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "rl")
		store := NewOmnichannelRateLimitStore(raiz)
		medirArmazenamento(t, raiz,
			func(org string) error { _, err := store.Load(org); return err },
			func(org string) error { return store.Save(org, nil) })
	})
}

// A recusa precisa ser pelo MOTIVO CERTO (escopo ausente), não por qualquer erro — senão a guarda passaria por acidente.
func TestRecusaDeEscopoUsaOMotivoDeclarado(t *testing.T) {
	store := NewOmnichannelDLQStore(filepath.Join(t.TempDir(), "dlq"))
	if _, err := store.Load(""); !errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("a leitura sem organização precisa falhar por escopo ausente: %v", err)
	}
	if err := store.Save("", nil); !errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("a gravação sem organização precisa falhar por escopo ausente: %v", err)
	}
}
