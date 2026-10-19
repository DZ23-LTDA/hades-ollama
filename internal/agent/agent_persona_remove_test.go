package agent

import (
	"testing"
)

// Remove retira a entrada do ledger -- e diz se ela existia.
//
// O QUE ESTES TESTES PROTEGEM
//
//	*Um rollback que restaura metade nao e' um rollback: e' um segundo defeito.* -- e o inverso tambem:
//	uma remocao que deixa a entrada no ficheiro e' uma remocao que se desfaz ao reabrir.

func TestRemoveTiraDoLedger(t *testing.T) {
	dir := t.TempDir()
	ledger, err := NewPersonaLedger(dir)
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	entrada := PersonaLedgerEntry{
		PersonaID:   "hades-executor",
		Target:      TargetCodex,
		Destination: "destino/AGENTS.md",
		Scope:        "org-1",
		SourceHash:   "h1",
		RenderedHash: "h2",
	}
	if _, _, err := ledger.Record("org-1", entrada); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if n := len(ledger.Entries("org-1")); n != 1 {
		t.Fatalf("esperava 1 entrada, tenho %d", n)
	}

	removido, err := ledger.Remove("org-1", "hades-executor")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !removido {
		t.Fatal("Remove devia dizer que removeu")
	}
	if n := len(ledger.Entries("org-1")); n != 0 {
		t.Fatalf("esperava 0 entradas, tenho %d", n)
	}
}

// Persistencia: reabrir o ledger nao pode ressuscitar a entrada.
func TestRemovePersisteNoFicheiro(t *testing.T) {
	dir := t.TempDir()
	ledger, err := NewPersonaLedger(dir)
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	if _, _, err := ledger.Record("org-1", PersonaLedgerEntry{
		PersonaID:   "p1",
		Target:      TargetCodex,
		Destination:  "d/AGENTS.md",
		Scope:        "org-1",
		SourceHash:   "h1",
		RenderedHash: "h2",
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := ledger.Remove("org-1", "p1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	reaberto, err := NewPersonaLedger(dir)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	if err := reaberto.Load("org-1"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if n := len(reaberto.Entries("org-1")); n != 0 {
		t.Fatalf("a entrada voltou ao reabrir: %d", n)
	}
}

// Remover o que nao existe devolve falso, nao erro: nao e' falha, e' ausencia.
func TestRemoveInexistenteDevolveFalso(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	removido, err := ledger.Remove("org-1", "nao-existe")
	if err != nil {
		t.Fatalf("Remove nao devia dar erro: %v", err)
	}
	if removido {
		t.Fatal("Remove devia devolver falso para uma persona inexistente")
	}
}

// Sem organizacao nao se remove nada -- a mesma guarda que Record e Load tem.
func TestRemoveSemOrganizacaoERecusado(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	if _, err := ledger.Remove("", "p1"); err == nil {
		t.Fatal("Remove sem scope devia ser recusado")
	}
}
