package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollaborationPersistsCommentsAndPresence(t *testing.T) {
	root := t.TempDir()
	store, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComment("project-1", "user-a", "revisar o preview"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetPresence("project-1", "user-a", "online"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := reloaded.Snapshot("project-1")
	if len(snapshot.Comments) != 1 || len(snapshot.Presence) != 1 || snapshot.Comments[0].Body != "revisar o preview" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestCollaborationRejectsCorruptLedgers(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "comments.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCollaborationStore(root); err == nil {
		t.Fatal("corrupt collaboration ledger unexpectedly loaded")
	}
}

func TestCollaborationRollsBackMemoryOnPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	store, err := NewCollaborationStore(root)
	if err != nil {
		t.Fatal(err)
	}
	// Falha de persistência de forma PORTÁTIL: em vez de um diretório ausente (que a escrita
	// atômica agora cria), um ARQUIVO ocupa o lugar da raiz, e `MkdirAll` falha em Linux,
	// macOS e Windows.
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("arquivo, não diretório"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddComment("project-1", "user-a", "must not remain"); err == nil {
		t.Fatal("comment unexpectedly persisted without a writable root")
	}
	if got := store.Comments("project-1"); len(got) != 0 {
		t.Fatalf("comment remained in memory after failure: %+v", got)
	}
	if _, err := store.SetPresence("project-1", "user-a", "online"); err == nil {
		t.Fatal("presence unexpectedly persisted without a writable root")
	}
	if got := store.Presence("project-1"); len(got) != 0 {
		t.Fatalf("presence remained in memory after failure: %+v", got)
	}
}
