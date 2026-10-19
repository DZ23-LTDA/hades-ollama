package agent

import "testing"

// TestLedgerIsolaOrganizacoes acrescenta o que `TestLedgerEscoposSaoIsolados` (que JA' EXISTIA) nao media.
//
// CORRECCAO A MIM MESMO
//
// Comecei este ficheiro a dizer «a propriedade que nenhum teste cobria». Era falso: ao correr a suite,
// `TestLedgerEscoposSaoIsolados` apareceu e passou. O que faltava nao era o isolamento de LEITURA -- era o
// de ESCRITA por caminho cruzado.
//
// O QUE E' NOVO AQUI
//
//   1. `RemoveAt` pedido por B a apontar ao DESTINO de A: tem de devolver `false` e nao remover nada.
//      Um isolamento de leitura com escrita cruzada aberta nao e' isolamento.
//   2. `Entries("")` com escopo vazio: o zero value de uma string nao pode significar «todos».
//      *O zero value de um bool e' uma afirmacao silenciosa.*
//
// O E-1387 fechou o ciclo no browser, mas todos os passes correram com UMA organizacao (`local`).
// *Um teste de isolamento com uma so' organizacao prova que uma organizacao se ve a si mesma. Nao prova
// isolamento: prova identidade.*
//
// O QUE SE PROVA, E NA ORDEM QUE IMPORTA
//
//  1. escrita em A nao aparece em B;
//  2. leitura de B nao ve' A;
//  3. remocao pedida por B nao remove A -- nem remove nada;
//  4. o ficheiro de A continua intacto depois da tentativa de B;
//  5. e A continua a ver o que tinha.
func TestLedgerIsolaOrganizacoes(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}

	const orgA, orgB = "org-a", "org-b"

	// Uma entrada em A. A mesma persona e o mesmo destino em B seria uma colisao plausivel.
	if _, _, err := ledger.Record(orgA, PersonaLedgerEntry{
		PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"),
		Destination: "segredo-da-a", Scope: orgA, SourceHash: "sa", RenderedHash: "ra",
	}); err != nil {
		t.Fatalf("Record em A: %v", err)
	}
	if _, _, err := ledger.Record(orgB, PersonaLedgerEntry{
		PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"),
		Destination: "segredo-da-b", Scope: orgB, SourceHash: "sb", RenderedHash: "rb",
	}); err != nil {
		t.Fatalf("Record em B: %v", err)
	}

	// 1+2. Cada uma ve' so' a sua. E o destino da outra NAO aparece.
	a := ledger.Entries(orgA)
	b := ledger.Entries(orgB)
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("esperava 1 e 1 entradas, tenho %d e %d", len(a), len(b))
	}
	if a[0].Destination != "segredo-da-a" {
		t.Errorf("A ve' %q", a[0].Destination)
	}
	if b[0].Destination != "segredo-da-b" {
		t.Errorf("B ve' %q", b[0].Destination)
	}

	// 3. Remocao pedida por B, apontando ao destino de A. Tem de devolver false e nao remover nada.
	removido, err := ledger.RemoveAt(orgB, "revisor", TargetFormat("hades"), "segredo-da-a")
	if err != nil {
		t.Fatalf("RemoveAt cross-org devia ser inofensivo, nao erro: %v", err)
	}
	if removido {
		t.Fatal("B removeu uma entrada de A")
	}

	// 4+5. A continua com a sua entrada.
	if n := len(ledger.Entries(orgA)); n != 1 {
		t.Fatalf("A perdeu a entrada: tem %d", n)
	}
	if n := len(ledger.Entries(orgB)); n != 1 {
		t.Fatalf("B perdeu a sua entrada: tem %d", n)
	}

	// E reabrir do disco nao mistura os ficheiros.
	reaberto, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	if err := reaberto.Load(orgA); err != nil {
		t.Fatalf("Load A: %v", err)
	}
	if err := reaberto.Load(orgB); err != nil {
		t.Fatalf("Load B: %v", err)
	}
	for _, e := range reaberto.Entries(orgA) {
		if e.Scope != orgA {
			t.Errorf("entrada com scope %q apareceu em A", e.Scope)
		}
	}
	for _, e := range reaberto.Entries(orgB) {
		if e.Scope != orgB {
			t.Errorf("entrada com scope %q apareceu em B", e.Scope)
		}
	}
}

// TestLedgerEscopoVazioNaoLeNada: um pedido sem organizacao nao pode virar "todas".
//
// O zero value de uma string e' "" -- e *o zero value de um bool e' uma afirmacao silenciosa*. Se
// `Entries("")` devolvesse as entradas de qualquer scope, quem esquecesse o tenant veria tudo.
func TestLedgerEscopoVazioNaoLeNada(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	if _, _, err := ledger.Record("org-a", PersonaLedgerEntry{
		PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"),
		Destination: "destino", Scope: "org-a", SourceHash: "s", RenderedHash: "r",
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if n := len(ledger.Entries("")); n != 0 {
		t.Fatalf("Entries(\"\") devolveu %d entradas: um escopo vazio nao e' o escopo de todos", n)
	}
}
