package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func entradaDeTeste(destino string) PersonaLedgerEntry {
	return PersonaLedgerEntry{
		PersonaID:    "revisor-de-codigo",
		Version:      "1.2.0",
		Target:       TargetHades,
		Destination:  destino,
		Scope:        "project",
		SourceHash:   "aaaa1111",
		RenderedHash: "bbbb2222",
	}
}

func TestLedgerPersisteECarrega(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	entrada, criada, err := ledger.Record("project", entradaDeTeste("p/.hades/agents/a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !criada {
		t.Fatal("a primeira gravacao devia ser criada")
	}
	if entrada.PersonaID != "revisor-de-codigo" {
		t.Fatalf("entrada devolvida errada: %+v", entrada)
	}

	// Reabrir: um ledger que nao sobrevive ao reinicio nao e' um ledger.
	outro, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if err := outro.Load("project"); err != nil {
		t.Fatal(err)
	}
	entradas := outro.Entries("project")
	if len(entradas) != 1 {
		t.Fatalf("esperava 1 entrada, obtive %d", len(entradas))
	}
	if entradas[0].RenderedHash != "bbbb2222" {
		t.Fatalf("hash nao sobreviveu: %q", entradas[0].RenderedHash)
	}
}

// *Um retry do mesmo pedido nao duplica: devolve a entrada existente e diz que nao criou nada.*
func TestRecordEhIdempotenteNoRetry(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primeira, criada, err := ledger.Record("project", entradaDeTeste("p/a.md"))
	if err != nil || !criada {
		t.Fatalf("primeira: criada=%v err=%v", criada, err)
	}
	// Mesmo pedido outra vez -- identico.
	segunda, criadaOutra, err := ledger.Record("project", entradaDeTeste("p/a.md"))
	if err != nil {
		t.Fatalf("retry identico nao devia falhar: %v", err)
	}
	if criadaOutra {
		t.Fatal("o retry identico duplicou a entrada")
	}
	if segunda.RenderedHash != primeira.RenderedHash {
		t.Fatal("o retry devolveu outra entrada")
	}
	if n := len(ledger.Entries("project")); n != 1 {
		t.Fatalf("esperava 1 entrada apos retry, obtive %d", n)
	}
}

// *Duas escritas no mesmo sitio com conteudos diferentes nao sao um retry: sao um conflito.*
func TestRecordRecusaDuasVerdadesParaAMesmaChave(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ledger.Record("project", entradaDeTeste("p/a.md")); err != nil {
		t.Fatal(err)
	}
	conflito := entradaDeTeste("p/a.md")
	conflito.RenderedHash = "cccc3333"
	_, _, err = ledger.Record("project", conflito)
	if err == nil {
		t.Fatal("esperava conflito")
	}
	if !errors.Is(err, ErrPluginVersionConflict) {
		t.Fatalf("erro nao e' de conflito: %v", err)
	}
	// E o ledger continua com UMA entrada, a primeira.
	entradas := ledger.Entries("project")
	if len(entradas) != 1 || entradas[0].RenderedHash != "bbbb2222" {
		t.Fatalf("o conflito alterou o ledger: %+v", entradas)
	}
}

func TestRecordExigeHashes(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	semHash := entradaDeTeste("p/a.md")
	semHash.RenderedHash = ""
	if _, _, err := ledger.Record("project", semHash); err == nil {
		t.Fatal("registo sem hash devia ser recusado")
	}
	if n := len(ledger.Entries("project")); n != 0 {
		t.Fatalf("o registo recusado ficou no ledger (%d entradas)", n)
	}
}

func TestLedgerRecusaFicheiroAdulterado(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ledger.Record("project", entradaDeTeste("p/a.md")); err != nil {
		t.Fatal(err)
	}
	caminho := ledger.scopePath("project")

	// JSON ilegivel.
	if err := os.WriteFile(caminho, []byte("{isto nao e json"), 0o600); err != nil {
		t.Fatal(err)
	}
	fresco, _ := NewPersonaLedger(raiz)
	if err := fresco.Load("project"); !errors.Is(err, ErrPersonaLedgerCorrupt) {
		t.Fatalf("JSON ilegivel: esperava corrupcao, obtive %v", err)
	}

	// Entrada sem hashes: *um registo sem hashes nao se carrega como se medisse alguma coisa.*
	semHash := `[{"persona_id":"x","target":"hades","destination":"p/a.md"}]`
	if err := os.WriteFile(caminho, []byte(semHash), 0o600); err != nil {
		t.Fatal(err)
	}
	fresco, _ = NewPersonaLedger(raiz)
	if err := fresco.Load("project"); !errors.Is(err, ErrPersonaLedgerCorrupt) {
		t.Fatalf("entrada sem hash: esperava corrupcao, obtive %v", err)
	}
}

// *Duas gravacoes do mesmo ledger tem de produzir os mesmos bytes, senao o ficheiro muda sem que nada tenha
// mudado.*
func TestPersistenciaEhDeterminista(t *testing.T) {
	raizA := t.TempDir()
	raizB := t.TempDir()
	registar := func(raiz string) string {
		ledger, err := NewPersonaLedger(raiz)
		if err != nil {
			t.Fatal(err)
		}
		// Gravadas por ordem DIFERENTE nos dois ledgers.
		for _, destino := range []string{"p/z.md", "p/a.md", "p/m.md"} {
			if _, _, err := ledger.Record("project", entradaDeTeste(destino)); err != nil {
				t.Fatal(err)
			}
		}
		return ledger.scopePath("project")
	}
	caminhoA := registar(raizA)
	registarInvertido := func(raiz string) string {
		ledger, err := NewPersonaLedger(raiz)
		if err != nil {
			t.Fatal(err)
		}
		for _, destino := range []string{"p/m.md", "p/a.md", "p/z.md"} {
			if _, _, err := ledger.Record("project", entradaDeTeste(destino)); err != nil {
				t.Fatal(err)
			}
		}
		return ledger.scopePath("project")
	}
	caminhoB := registarInvertido(raizB)

	bytesA, err := os.ReadFile(caminhoA)
	if err != nil {
		t.Fatal(err)
	}
	bytesB, err := os.ReadFile(caminhoB)
	if err != nil {
		t.Fatal(err)
	}
	if string(bytesA) != string(bytesB) {
		t.Fatalf("ordem de gravacao mudou os bytes do ledger:\nA=%s\nB=%s", bytesA, bytesB)
	}
}

func TestReconcileContraODisco(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := personaDeTeste()
	rendido, err := RenderHades(p)
	if err != nil {
		t.Fatal(err)
	}
	hashOrigem, err := p.ContentHash()
	if err != nil {
		t.Fatal(err)
	}

	// Tres instalacoes: intacta, editada a mao, apagada.
	criar := func(destino, rendered string) {
		e := entradaDeTeste(destino)
		e.SourceHash = hashOrigem
		e.RenderedHash = rendered
		if _, _, err := ledger.Record("project", e); err != nil {
			t.Fatal(err)
		}
	}
	criar("p/intacta.md", hashBytes(rendido))
	criar("p/editada.md", "hash-do-original")
	criar("p/apagada.md", "hash-do-original")

	estadoDisco := map[string][]byte{
		"p/intacta.md": rendido,
		// *Editado fora do HADES: actualizar por cima apagaria trabalho de uma pessoa.*
		"p/editada.md": []byte("alguem mudou isto a mao"),
		// "p/apagada.md" ausente
	}
	itens := ledger.Reconcile("project", estadoDisco, func(PersonaLedgerEntry) []byte { return rendido })

	porDestino := map[string]Reconciliation{}
	for _, item := range itens {
		porDestino[item.Entry.Destination] = item
	}
	if got := porDestino["p/intacta.md"].State; got != DriftCurrent {
		t.Fatalf("intacta: %s", got)
	}
	if got := porDestino["p/editada.md"].State; got != DriftLocallyModified {
		t.Fatalf("editada: %s", got)
	}
	if got := porDestino["p/apagada.md"].State; got != DriftRemoved {
		t.Fatalf("apagada: %s", got)
	}
	// E a accao de cada uma e' a segura.
	if !strings.Contains(porDestino["p/editada.md"].Action, "Preserve") {
		t.Fatalf("editada nao manda preservar: %q", porDestino["p/editada.md"].Action)
	}
}

func TestReconcileDetectaOrigemQueMudou(t *testing.T) {
	ledger, err := NewPersonaLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ledger.Record("project", PersonaLedgerEntry{
		PersonaID: "x", Target: TargetHades, Destination: "p/a.md",
		SourceHash: "velho", RenderedHash: hashBytes([]byte("antigo")),
	})
	itens := ledger.Reconcile("project",
		map[string][]byte{"p/a.md": []byte("antigo")},
		func(PersonaLedgerEntry) []byte { return []byte("novo") })
	if itens[0].State != DriftOutdated {
		t.Fatalf("origem mudou: estado=%s", itens[0].State)
	}
}

// *Um resumo que omite o estado com zero faz o ecra dizer "tudo actual" quando nao ha' nada registado.*
func TestDriftSummaryTemTodasAsChaves(t *testing.T) {
	resumo := DriftSummary(nil)
	for _, estado := range []DriftState{DriftCurrent, DriftOutdated, DriftLocallyModified, DriftRemoved, DriftForeign, DriftConflicted, DriftBroken} {
		if _, existe := resumo[estado]; !existe {
			t.Fatalf("resumo vazio sem a chave %s", estado)
		}
	}
	if len(resumo) != 7 {
		t.Fatalf("esperava 7 chaves, obtive %d", len(resumo))
	}
	resumo = DriftSummary([]Reconciliation{{State: DriftCurrent}, {State: DriftCurrent}, {State: DriftRemoved}})
	if resumo[DriftCurrent] != 2 || resumo[DriftRemoved] != 1 {
		t.Fatalf("contagem errada: %+v", resumo)
	}
}

func TestLedgerEscoposSaoIsolados(t *testing.T) {
	raiz := t.TempDir()
	ledger, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ledger.Record("project", entradaDeTeste("p/a.md")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ledger.Record("organization", entradaDeTeste("o/a.md")); err != nil {
		t.Fatal(err)
	}
	if n := len(ledger.Entries("project")); n != 1 {
		t.Fatalf("project devia ter 1, tem %d", n)
	}
	if n := len(ledger.Entries("organization")); n != 1 {
		t.Fatalf("organization devia ter 1, tem %d", n)
	}
	// E os ficheiros sao distintos.
	if ledger.scopePath("project") == ledger.scopePath("organization") {
		t.Fatal("dois escopos no mesmo ficheiro: um apaga o outro")
	}
	if _, err := os.Stat(filepath.Join(raiz, "personas")); err != nil {
		t.Fatalf("directorio do ledger nao criado: %v", err)
	}
}
