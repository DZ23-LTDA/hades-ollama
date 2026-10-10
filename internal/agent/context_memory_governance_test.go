package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newGovernanceStore(t *testing.T) *ContextStore {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func seedGovernanceMemories(t *testing.T, store *ContextStore) {
	t.Helper()
	older := time.Now().UTC().Add(-60 * 24 * time.Hour)
	recent := time.Now().UTC().Add(-1 * time.Hour)
	for _, memory := range []Memory{
		{ID: "mem-old", ProjectID: "proj-1", Kind: "fact", Content: "memória antiga", Source: "import", Confidence: 0.5, CreatedAt: older},
		{ID: "mem-new", ProjectID: "proj-1", Kind: "fact", Content: "memória recente", Source: "chat", Confidence: 0.9, CreatedAt: recent},
		{ID: "mem-outro", ProjectID: "proj-2", Kind: "fact", Content: "memória de outro projeto", CreatedAt: recent},
	} {
		if _, err := store.AddMemory(memory); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContextStoreExportMemoriesKeepsProvenanceAndIsolatesProjects(t *testing.T) {
	store := newGovernanceStore(t)
	seedGovernanceMemories(t, store)
	if _, err := store.AddMemory(Memory{ID: "mem-embed", ProjectID: "proj-1", Kind: "fact", Content: "com vetor", Embedding: []float32{0.1, 0.2}, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	exported := store.ExportMemories("proj-1", false)
	if len(exported) != 3 {
		t.Fatalf("expected 3 memories of proj-1, got %d", len(exported))
	}
	for _, memory := range exported {
		if memory.ProjectID != "proj-1" {
			t.Fatalf("export leaked another project: %+v", memory)
		}
		if len(memory.Embedding) != 0 {
			t.Fatalf("embeddings must be omitted by default: %+v", memory)
		}
	}
	// Proveniência sobrevive ao export (conferida nas memórias que a declaram).
	if exported[0].Source != "import" || exported[0].Confidence != 0.5 {
		t.Fatalf("provenance must survive the export: %+v", exported[0])
	}
	if exported[1].Source != "chat" || exported[1].Confidence != 0.9 {
		t.Fatalf("provenance must survive the export: %+v", exported[1])
	}
	if exported[0].ID != "mem-old" || exported[len(exported)-1].ID != "mem-embed" {
		t.Fatalf("export must be chronological, got %+v", exported)
	}

	withEmbeddings := store.ExportMemories("proj-1", true)
	if len(withEmbeddings) != 3 {
		t.Fatalf("expected 3 memories with embeddings, got %d", len(withEmbeddings))
	}
	foundVector := false
	for _, memory := range withEmbeddings {
		if len(memory.Embedding) > 0 {
			foundVector = true
		}
	}
	if !foundVector {
		t.Fatal("include_embeddings=true must carry the vectors")
	}
}

func TestContextStoreRemoveMemoryPersistsAndRejectsUnknown(t *testing.T) {
	store := newGovernanceStore(t)
	seedGovernanceMemories(t, store)

	if err := store.RemoveMemory("proj-1", "mem-old"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	remaining := store.ExportMemories("proj-1", false)
	if len(remaining) != 1 || remaining[0].ID != "mem-new" {
		t.Fatalf("remaining = %+v", remaining)
	}
	if err := store.RemoveMemory("proj-1", "mem-old"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("removing twice must report not found, got %v", err)
	}
	// Escopo: a memória do outro projeto continua intacta.
	if err := store.RemoveMemory("proj-1", "mem-outro"); !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("cross-project removal must not find the memory, got %v", err)
	}
	if got := len(store.ExportMemories("proj-2", false)); got != 1 {
		t.Fatalf("other project memory count = %d", got)
	}
	// Persistência: remover e reler do disco mantém o estado.
	reopened, err := NewContextStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.ExportMemories("proj-1", false); len(got) != 1 || got[0].ID != "mem-new" {
		t.Fatalf("reloaded memories = %+v", got)
	}
}

func TestContextStoreRemoveMemoryRollsBackWhenPersistenceFails(t *testing.T) {
	store := newGovernanceStore(t)
	seedGovernanceMemories(t, store)

	// Substitui o diretório de memórias por um ARQUIVO: a próxima gravação falha
	// e o estado em memória precisa voltar ao que era.
	memoriesDir := filepath.Join(store.root, "memories")
	if err := os.RemoveAll(memoriesDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memoriesDir, []byte("bloqueio"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := store.RemoveMemory("proj-1", "mem-old"); err == nil {
		t.Fatal("expected a persistence failure")
	}
	restored := store.ExportMemories("proj-1", false)
	if len(restored) != 2 {
		t.Fatalf("rollback must restore both memories, got %+v", restored)
	}
	found := false
	for _, memory := range restored {
		if memory.ID == "mem-old" {
			found = true
		}
	}
	if !found {
		t.Fatal("rolled back state lost the memory that failed to be removed")
	}
}

func TestContextStorePruneMemoriesKeepsRecentAndUndated(t *testing.T) {
	store := newGovernanceStore(t)
	seedGovernanceMemories(t, store)
	if _, err := store.AddMemory(Memory{ID: "mem-sem-data", ProjectID: "proj-1", Kind: "fact", Content: "sem data"}); err != nil {
		t.Fatal(err)
	}
	// AddMemory preenche CreatedAt quando zerado: recria sem data direto no mapa
	// para exercitar a regra de preservação.
	store.mu.Lock()
	for index, memory := range store.memories["proj-1"] {
		if memory.ID == "mem-sem-data" {
			memory.CreatedAt = time.Time{}
			store.memories["proj-1"][index] = memory
		}
	}
	store.mu.Unlock()

	removed, err := store.PruneMemories("proj-1", time.Now().UTC().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected exactly the old memory to be pruned, got %d", removed)
	}
	kept := store.ExportMemories("proj-1", false)
	if len(kept) != 2 {
		t.Fatalf("kept = %+v", kept)
	}
	for _, memory := range kept {
		if memory.ID == "mem-old" {
			t.Fatal("old memory survived the retention window")
		}
	}
	if got := len(store.ExportMemories("proj-2", false)); got != 1 {
		t.Fatalf("retention must not touch other projects, proj-2 = %d", got)
	}
	if _, err := store.PruneMemories("proj-1", time.Time{}); err == nil {
		t.Fatal("retention without a cutoff must be rejected")
	}
	if _, err := store.PruneMemories("", time.Now().UTC()); err == nil {
		t.Fatal("retention without a project must be rejected")
	}
}
