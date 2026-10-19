// Backup e rollback do ledger de personas (FASE 30.4).
//
// PORQUE ISTO NAO E' COPIA-LOJA
//
// Um backup que so' guarda o ledger nao restaura nada: o que o utilizador perdeu foi o FICHEIRO. Um backup
// que so' guarda o ficheiro nao sabe o que repor: o que diz o que devia la' estar e' o ledger. O backup
// guarda as duas coisas e o `Restore` repoe as duas -- ou nenhuma.
//
//	*Um rollback que restaura metade nao e' um rollback: e' um segundo defeito.*
//
// # O ID DO BACKUP VEM DO CONTEUDO
//
// `backupID` e' o sha256 (truncado) do par (ledger ordenado + hashes dos ficheiros). Duas chamadas sobre o
// mesmo estado devolvem o MESMO id, e o segundo `Backup` nao duplica nada.
//
//	*Idempotencia de verdade e' devolver o MESMO RECURSO.*
//
// Nao uso `sanitizeIDComponent` para o id: o id e' hex gerado aqui, nao vem de fora. Um id que o servidor
// deriva do conteudo nao precisa de ser sanitizado -- precisa de ser derivado por inteiro do conteudo.
//
// RESTORE E' TUDO-OU-NADA
//
// O `Restore` le' e valida TUDO primeiro (backup existe, entradas legiveis, ficheiros de backup presentes)
// e so' depois escreve. Se falhar a meio, o estado fica como estava -- e o erro diz o que faltou.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrBackupNaoEncontrado cobre um id de backup que nao existe.
var ErrBackupNaoEncontrado = errors.New("agent persona backup not found")

// ErrBackupCorrompido cobre um backup ilegivel ou incompleto.
var ErrBackupCorrompido = errors.New("agent persona backup is corrupt")

// PersonaBackup e' o manifesto de um backup.
type PersonaBackup struct {
	ID        string               `json:"id"`
	Scope     string               `json:"scope"`
	CreatedAt string               `json:"created_at"`
	Entries   []PersonaLedgerEntry `json:"entries"`
	// Files: destino -> hash do conteudo guardado em `files/<hash>`.
	//
	// O nome do ficheiro de backup e' o HASH, nao o destino: dois destinos com o mesmo conteudo partilham
	// um ficheiro, e um destino com `..` nao escreve fora da pasta de backup.
	Files map[string]string `json:"files"`
}

// backupsDir e' onde os backups vivem, debaixo da raiz do ledger.
func (l *PersonaLedger) backupsDir(scope string) string {
	return filepath.Join(l.root, "personas", "backups", sanitizeIDComponent(scope))
}

// backupPath e' a pasta de um backup. O id e' hex: nao pode conter separador de caminho.
func (l *PersonaLedger) backupPath(scope, id string) string {
	return filepath.Join(l.backupsDir(scope), id)
}

// backupID deriva o id do CONTEUDO. Mesmo estado -> mesmo id.
func backupID(scope string, entradas []PersonaLedgerEntry, hashes map[string]string) string {
	var b strings.Builder
	b.WriteString("v1|")
	b.WriteString(scope)
	b.WriteString("|")
	for _, e := range entradas {
		b.WriteString(PersonaLedgerKey(e))
		b.WriteString("#")
		b.WriteString(e.SourceHash)
		b.WriteString("#")
		b.WriteString(e.RenderedHash)
		b.WriteString(";")
	}
	b.WriteString("|")
	destinos := make([]string, 0, len(hashes))
	for d := range hashes {
		destinos = append(destinos, d)
	}
	sort.Strings(destinos)
	for _, d := range destinos {
		b.WriteString(d)
		b.WriteString("=")
		b.WriteString(hashes[d])
		b.WriteString(";")
	}
	return hashBytes([]byte(b.String()))[:24]
}

// Backup guarda ledger + conteudo dos destinos de um escopo.
//
// Idempotente: chamar duas vezes sobre o mesmo estado devolve o mesmo id e NAO duplica o backup.
//
// Um destino que nao existe no disco e' registado com hash vazio: o backup tem de saber que o ficheiro ja'
// faltava, para que o restore nao invente um ficheiro que nunca existiu.
func (l *PersonaLedger) Backup(scope string, agora time.Time) (PersonaBackup, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return PersonaBackup{}, fmt.Errorf("%w: scope is required", ErrPersonaInvalid)
	}
	// A pasta de backup e' indexada pelo escopo. Ler ou escrever o backup de outra organizacao e' tao grave
	// como escrever no ledger dela: a guarda tem de estar aqui tambem.
	if err := requireOrganization(scope); err != nil {
		return PersonaBackup{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.root == "" {
		return PersonaBackup{}, fmt.Errorf("%w: ledger without a root cannot back up", ErrPersonaInvalid)
	}

	entradas := l.entriesLocked(scope)

	// Ler o conteudo dos destinos ANTES de decidir o id.
	conteudo := map[string][]byte{}
	hashes := map[string]string{}
	for _, e := range entradas {
		destino := strings.TrimSpace(e.Destination)
		if destino == "" {
			continue
		}
		b, err := os.ReadFile(destino)
		if err != nil {
			// Ja' nao estava la'. O backup regista a ausencia.
			hashes[destino] = ""
			continue
		}
		conteudo[destino] = b
		hashes[destino] = hashBytes(b)
	}

	id := backupID(scope, entradas, hashes)
	bk := PersonaBackup{
		ID:        id,
		Scope:     scope,
		CreatedAt: agora.UTC().Format(time.RFC3339),
		Entries:   entradas,
		Files:     hashes,
	}

	dir := l.backupPath(scope, id)
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err == nil {
		// Ja' existe e e' o mesmo conteudo. *Idempotencia de verdade e' devolver o MESMO RECURSO.*
		return bk, nil
	}
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		return PersonaBackup{}, err
	}
	// Os ficheiros sao gravados pelo HASH: dois destinos iguais partilham um ficheiro.
	for destino, b := range conteudo {
		h := hashes[destino]
		if err := writeFileAtomic(filepath.Join(dir, "files", h), b); err != nil {
			return PersonaBackup{}, err
		}
	}
	manifesto, err := json.MarshalIndent(bk, "", "  ")
	if err != nil {
		return PersonaBackup{}, err
	}
	if err := writeFileAtomic(filepath.Join(dir, "manifest.json"), manifesto); err != nil {
		return PersonaBackup{}, err
	}
	return bk, nil
}

// writeFileAtomic escreve e renomeia, para um crash a meio nao deixar um ficheiro pela metade.
func writeFileAtomic(caminho string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return err
	}
	tmp := caminho + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, caminho)
}

// Backups lista os ids de backup de um escopo, ordenados.
func (l *PersonaLedger) Backups(scope string) ([]string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil, fmt.Errorf("%w: scope is required", ErrPersonaInvalid)
	}
	if err := requireOrganization(scope); err != nil {
		return nil, err
	}
	if l.root == "" {
		return nil, nil
	}
	entradas, err := os.ReadDir(l.backupsDir(scope))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ids := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// LoadBackup le' e VALIDA um backup sem escrever nada.
func (l *PersonaLedger) LoadBackup(scope, id string) (PersonaBackup, error) {
	scope = strings.TrimSpace(scope)
	id = strings.TrimSpace(id)
	if scope == "" || id == "" {
		return PersonaBackup{}, fmt.Errorf("%w: scope and id are required", ErrPersonaInvalid)
	}
	if err := requireOrganization(scope); err != nil {
		return PersonaBackup{}, err
	}
	dir := l.backupPath(scope, id)
	if _, err := os.Stat(dir); err != nil {
		return PersonaBackup{}, fmt.Errorf("%w: %s", ErrBackupNaoEncontrado, id)
	}
	manifesto, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return PersonaBackup{}, fmt.Errorf("%w: %s", ErrBackupCorrompido, err.Error())
	}
	var bk PersonaBackup
	if err := json.Unmarshal(manifesto, &bk); err != nil {
		return PersonaBackup{}, fmt.Errorf("%w: %s", ErrBackupCorrompido, err.Error())
	}
	if bk.ID != id {
		return PersonaBackup{}, fmt.Errorf("%w: manifest id %q != %q", ErrBackupCorrompido, bk.ID, id)
	}
	if bk.Scope != scope {
		return PersonaBackup{}, fmt.Errorf("%w: manifest scope %q != %q", ErrBackupCorrompido, bk.Scope, scope)
	}
	// Validar os ficheiros ANTES de qualquer escrita. *Um rollback que restaura metade...*
	for destino, h := range bk.Files {
		if h == "" {
			continue // ja' faltava ao fazer o backup: nao ha' ficheiro para repor
		}
		if _, err := os.Stat(filepath.Join(dir, "files", h)); err != nil {
			return PersonaBackup{}, fmt.Errorf("%w: file for %q is missing", ErrBackupCorrompido, destino)
		}
	}
	return bk, nil
}

// Restore repoe o ledger E os ficheiros de um backup.
//
// Tudo-ou-nada: valida tudo primeiro (via `LoadBackup`), depois escreve. Se a escrita de um ficheiro falhar
// a meio, devolve o erro -- e o chamador sabe que o estado pode estar parcial. O manifesto NAO e' apagado:
// um restore malsucedido tem de poder ser tentado de novo.
func (l *PersonaLedger) Restore(scope, id string) error {
	bk, err := l.LoadBackup(scope, id)
	if err != nil {
		return err
	}
	dir := l.backupPath(bk.Scope, bk.ID)

	// Ficheiros primeiro, ledger depois: se o processo cair, o ledger ainda aponta para o estado antigo e o
	// proximo `Reconcile` ve' a diferenca. Ao contrario, o ledger apontaria para ficheiros que nao existem.
	for destino, h := range bk.Files {
		if h == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, "files", h))
		if err != nil {
			return fmt.Errorf("%w: reading backup file for %q: %s", ErrBackupCorrompido, destino, err.Error())
		}
		if err := writeFileAtomic(destino, b); err != nil {
			return fmt.Errorf("restoring %q: %w", destino, err)
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	escopo := map[string]PersonaLedgerEntry{}
	for _, e := range bk.Entries {
		escopo[PersonaLedgerKey(e)] = e
	}
	l.byScope[bk.Scope] = escopo
	return l.persistLocked(bk.Scope)
}

// entriesLocked devolve as entradas do escopo ordenadas pela chave. Chamador deve segurar o mutex (ou ser o
// proprio `Backup`, que ja' o segura).
func (l *PersonaLedger) entriesLocked(scope string) []PersonaLedgerEntry {
	escopo := l.byScope[scope]
	chaves := make([]string, 0, len(escopo))
	for chave := range escopo {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	out := make([]PersonaLedgerEntry, 0, len(chaves))
	for _, k := range chaves {
		out = append(out, escopo[k])
	}
	return out
}
