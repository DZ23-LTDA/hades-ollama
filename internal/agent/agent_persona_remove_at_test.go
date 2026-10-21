package agent

import (
	"path/filepath"
	"testing"
)

// TestRemoveAtTiraSoAUmaInstalacao e' o teste que o defeito de ronda 193 tornou necessario.
//
// # O DEFEITO
//
// O painel chamava `removerAgente(destino)` -- isto e', passava o CAMINHO onde a funcao espera o
// `persona_id`. O servidor respondia 404 ("persona nao registada"), o painel caia no `erro`, e o bloco
// `agent-center-removido` nunca aparecia. O botao existia, estava ligado a rota certa e nunca funcionou.
//
// *Dois argumentos do mesmo tipo nao sao dois argumentos: sao um que o compilador nao distingue.*
//
// So' apareceu num browser real. O teste do handler passava porque chamava a funcao com o argumento certo.
//
// # O QUE ESTE TESTE PROTEGE
//
// Que remover uma instalacao NAO remova as outras da mesma persona. O `Remove` apaga todas -- e num painel
// onde o utilizador aponta um destino, isso apagaria o que ele nao apontou.
func TestRemoveAtTiraSoAUmaInstalacao(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	scope := "org-a"

	entradas := []PersonaLedgerEntry{
		{PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"), Destination: "destino/a", Scope: scope, SourceHash: "s1", RenderedHash: "r1"},
		{PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"), Destination: "destino/b", Scope: scope, SourceHash: "s1", RenderedHash: "r1"},
		{PersonaID: "outra", Version: "1", Target: TargetFormat("hades"), Destination: "destino/c", Scope: scope, SourceHash: "s2", RenderedHash: "r2"},
	}
	for _, e := range entradas {
		if _, _, err := ledger.Record(scope, e); err != nil {
			t.Fatalf("Record(%s): %v", e.Destination, err)
		}
	}
	if n := len(ledger.Entries(scope)); n != 3 {
		t.Fatalf("esperava 3 entradas, tenho %d", n)
	}

	removido, err := ledger.RemoveAt(scope, "revisor", TargetFormat("hades"), "destino/a")
	if err != nil {
		t.Fatalf("RemoveAt: %v", err)
	}
	if !removido {
		t.Fatal("RemoveAt devolveu false para uma entrada que existe")
	}

	// O discriminante: a OUTRA instalacao da mesma persona continua la'.
	restantes := ledger.Entries(scope)
	if len(restantes) != 2 {
		t.Fatalf("esperava 2 entradas depois de remover uma, tenho %d: %+v", len(restantes), restantes)
	}
	for _, e := range restantes {
		if e.PersonaID == "revisor" && e.Destination == "destino/a" {
			t.Error("a entrada removida voltou")
		}
	}
	// E o ficheiro tem de reflectir isso: reabrir nao pode ressuscitar a entrada.
	reaberto, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	if err := reaberto.Load(scope); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if n := len(reaberto.Entries(scope)); n != 2 {
		t.Fatalf("ao reabrir esperava 2, tenho %d", n)
	}
	if _, err := filepath.Abs(raiz); err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
}

// TestRemoveAtComDestinoErradoNaoRemove nada. E' o caso do painel que mandava o caminho no lugar do id:
// tem de devolver `false`, nao "removi alguma coisa".
func TestRemoveAtComDestinoErradoNaoRemove(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	if _, _, err := ledger.Record("org-a", PersonaLedgerEntry{
		PersonaID: "revisor", Version: "1", Target: TargetFormat("hades"),
		Destination: "destino/a", Scope: "org-a", SourceHash: "s1", RenderedHash: "r1",
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// O id no lugar do destino -- exactamente o que o painel fazia.
	removido, err := ledger.RemoveAt("org-a", "revisor", TargetFormat("hades"), "revisor")
	if err != nil {
		t.Fatalf("RemoveAt: %v", err)
	}
	if removido {
		t.Fatal("RemoveAt removeu com o destino errado")
	}
	if n := len(ledger.Entries("org-a")); n != 1 {
		t.Fatalf("a entrada devia continuar, tenho %d", n)
	}
}

// TestRemoveAtSemOrganizacaoERecusado: sem escopo nao se remove nada.
func TestRemoveAtSemOrganizacaoERecusado(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatalf("NewPersonaLedger: %v", err)
	}
	if _, err := ledger.RemoveAt("", "revisor", TargetFormat("hades"), "destino/a"); err == nil {
		t.Fatal("RemoveAt sem escopo devia falhar")
	}
}
