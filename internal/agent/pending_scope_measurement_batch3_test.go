package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// TERCEIRO LOTE DE MEDIÇÃO POR COMPORTAMENTO (E-341): voice, company_factory, company_ledger, business_snapshots.
//
// A guarda roda ANTES de qualquer uso dos outros parâmetros, então medir com os métodos que recebem SÓ a organização já
// prova o gatilho. Métodos com mais parâmetros ficam para a fatia seguinte — *declarado, não presumido*.

func TestTerceiroLoteRecusaEscopoAusente(t *testing.T) {
	t.Run("voice_active_call", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "voice")
		store := NewVoiceActiveCallStore(raiz)
		if _, err := store.Calls(""); !errors.Is(err, ErrTenantScopeRequired) {
			t.Fatalf("voice: chamadas sem organização precisam ser recusadas pelo motivo declarado: %v", err)
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("company_ledger", func(t *testing.T) {
		// MEDIDO: `ListLedgerEntries` vive no `CompanyStore`, nao num store de ledger proprio.
		raiz := filepath.Join(t.TempDir(), "ledger")
		store, err := NewCompanyStore(raiz)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.ListLedgerEntries("", time.Time{}, time.Now().UTC()); !errors.Is(err, ErrTenantScopeRequired) {
			t.Fatalf("ledger: listagem sem organização precisa ser recusada: %v", err)
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("sales_workforce", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "sales")
		// `Proposals` não devolve erro: sem tenant a lista precisa vir VAZIA (fail-closed).
		store := NewSalesWorkforceStore(raiz, nil, nil)
		if propostas := store.Proposals("", ""); len(propostas) != 0 {
			t.Fatalf("sales: propostas sem organização precisam vir vazias: %d", len(propostas))
		}
		assertSemArquivoCompartilhado(t, raiz)
	})
}
