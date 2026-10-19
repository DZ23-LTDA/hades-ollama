// Testes do backup e rollback do ledger.
//
// Discriminantes: mesmo estado -> mesmo id (sem duplicar); o restore repoe FICHEIRO e LEDGER; um backup
// incompleto e' recusado ANTES de escrever; um destino que ja' faltava nao e' inventado pelo restore.
package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const escopoTeste = "org-teste"

func ledgerComEntrada(t *testing.T) (*PersonaLedger, string) {
	t.Helper()
	raiz := t.TempDir()
	l, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	destino := filepath.Join(t.TempDir(), "agente.md")
	if err := os.WriteFile(destino, []byte("# v1 do agente\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = l.Record(escopoTeste, PersonaLedgerEntry{
		PersonaID: "p", Version: "1", Target: TargetHades, Destination: destino,
		Scope: escopoTeste, SourceHash: "aaa", RenderedHash: hashBytes([]byte("# v1 do agente\n")),
	})
	if err != nil {
		t.Fatal(err)
	}
	return l, destino
}

// *Idempotencia de verdade e' devolver o MESMO RECURSO.*
func TestBackupMesmoEstadoMesmoID(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	agora := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	b1, err := l.Backup(escopoTeste, agora)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := l.Backup(escopoTeste, agora)
	if err != nil {
		t.Fatal(err)
	}
	if b1.ID != b2.ID {
		t.Fatalf("ids diferentes para o mesmo estado: %q vs %q", b1.ID, b2.ID)
	}
	ids, err := l.Backups(escopoTeste)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("o segundo backup duplicou: %v", ids)
	}
}

func TestBackupGuardaFicheiroELedger(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Entries) != 1 {
		t.Fatalf("entradas guardadas: %d", len(bk.Entries))
	}
	if len(bk.Files) != 1 {
		t.Fatalf("ficheiros guardados: %d", len(bk.Files))
	}
	for destino, h := range bk.Files {
		if h == "" {
			t.Fatalf("hash vazio para %q", destino)
		}
	}
}

func TestBackupRecusaEscopoVazio(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	if _, err := l.Backup("  ", time.Now()); err == nil {
		t.Fatal("escopo vazio foi aceito")
	}
}

// *Um rollback que restaura metade nao e' um rollback: e' um segundo defeito.*
func TestRestoreRepoeFicheiroApagado(t *testing.T) {
	l, destino := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(destino); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(escopoTeste, bk.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatalf("o ficheiro nao voltou: %v", err)
	}
	if string(b) != "# v1 do agente\n" {
		t.Fatalf("conteudo restaurado errado: %q", string(b))
	}
}

func TestRestoreRepoeLedgerAlterado(t *testing.T) {
	l, destino := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Estado "novo": outra versao da persona, outro conteudo.
	if err := os.WriteFile(destino, []byte("# v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Record(escopoTeste, PersonaLedgerEntry{
		PersonaID: "p", Version: "2", Target: TargetHades, Destination: destino,
		Scope: escopoTeste, SourceHash: "bbb", RenderedHash: hashBytes([]byte("# v2\n")),
	}); err == nil {
		// conflito esperado: hashes diferentes para a mesma chave. Segue-se o caminho do restore.
		t.Log("record recusou (conflito) -- esperado")
	}
	if err := l.Restore(escopoTeste, bk.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	entradas := l.Entries(escopoTeste)
	if len(entradas) != 1 {
		t.Fatalf("o ledger devia ter 1 entrada, tem %d", len(entradas))
	}
	if entradas[0].Version != "1" {
		t.Fatalf("o ledger nao voltou a versao 1: %q", entradas[0].Version)
	}
}

func TestLoadBackupIDInexistente(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	if _, err := l.LoadBackup(escopoTeste, "nao-existe"); !errors.Is(err, ErrBackupNaoEncontrado) {
		t.Fatalf("esperava ErrBackupNaoEncontrado, veio %v", err)
	}
}

func TestLoadBackupRecusaManifestoCorrompido(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	dir := l.backupPath(escopoTeste, bk.ID)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{nao json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := l.LoadBackup(escopoTeste, bk.ID); !errors.Is(err, ErrBackupCorrompido) {
		t.Fatalf("esperava ErrBackupCorrompido, veio %v", err)
	}
}

// O restore tem de recusar ANTES de escrever, se o backup estiver incompleto.
func TestRestoreRecusaBackupIncompletoSemEscrever(t *testing.T) {
	l, destino := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	dir := l.backupPath(escopoTeste, bk.ID)
	for _, h := range bk.Files {
		if h == "" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, "files", h)); err != nil {
			t.Fatal(err)
		}
	}
	// altera o destino: o restore NAO pode te-lo tocado
	if err := os.WriteFile(destino, []byte("# intocado\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := l.Restore(escopoTeste, bk.ID); !errors.Is(err, ErrBackupCorrompido) {
		t.Fatalf("esperava ErrBackupCorrompido, veio %v", err)
	}
	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "# intocado\n" {
		t.Fatalf("o restore escreveu apesar de recusar: %q", string(b))
	}
}

// Um destino que ja' faltava ao fazer o backup tem hash vazio: o restore nao inventa o ficheiro.
func TestRestoreNaoInventaFicheiroQueJaFaltava(t *testing.T) {
	raiz := t.TempDir()
	l, err := NewPersonaLedger(raiz)
	if err != nil {
		t.Fatal(err)
	}
	destino := filepath.Join(t.TempDir(), "nunca-existiu.md")
	if _, _, err := l.Record(escopoTeste, PersonaLedgerEntry{
		PersonaID: "p", Version: "1", Target: TargetHades, Destination: destino,
		Scope: escopoTeste, SourceHash: "aaa", RenderedHash: "bbb",
	}); err != nil {
		t.Fatal(err)
	}
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if bk.Files[destino] != "" {
		t.Fatalf("esperava hash vazio para destino ausente, veio %q", bk.Files[destino])
	}
	if err := l.Restore(escopoTeste, bk.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := os.Stat(destino); !os.IsNotExist(err) {
		t.Fatal("o restore inventou um ficheiro que nunca existiu")
	}
}

func TestBackupsListaOrdenada(t *testing.T) {
	l, destino := ledgerComEntrada(t)
	// dois backups diferentes: muda o conteudo entre eles
	if _, err := l.Backup(escopoTeste, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destino, []byte("# outro\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Backup(escopoTeste, time.Now()); err != nil {
		t.Fatal(err)
	}
	ids, err := l.Backups(escopoTeste)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("esperava 2 backups, veio %d: %v", len(ids), ids)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Fatalf("ordem violada: %v", ids)
		}
	}
}

func TestBackupsSemNenhumDevolveVazio(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	ids, err := l.Backups(escopoTeste)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("esperava zero, veio %v", ids)
	}
}

func TestBackupIDsSaoNomesDeFicheiro(t *testing.T) {
	l, _ := ledgerComEntrada(t)
	bk, err := l.Backup(escopoTeste, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(bk.ID) != bk.ID {
		t.Fatalf("id com separador de caminho: %q", bk.ID)
	}
	if len(bk.ID) != 24 {
		t.Fatalf("id com tamanho inesperado: %d", len(bk.ID))
	}
}
