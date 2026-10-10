package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newRetentionStore(t *testing.T) (*ContextStore, Project) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto retenção", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project
}

func seedRetentionMemories(t *testing.T, store *ContextStore, projectID, prefix string, count int, age time.Duration) {
	t.Helper()
	for index := range count {
		if _, err := store.AddMemory(Memory{
			ID:        prefix + "_" + time.Unix(int64(index), 0).UTC().Format("150405"),
			ProjectID: projectID,
			Kind:      "fact",
			Content:   "memória de retenção",
			CreatedAt: time.Now().UTC().Add(-age - time.Duration(index)*time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRetentionPolicyValidatesAndPersists(t *testing.T) {
	store, _ := newRetentionStore(t)
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 0}); !errors.Is(err, ErrRetentionPolicyInvalid) {
		t.Fatalf("zero days must be rejected, got %v", err)
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 4000}); !errors.Is(err, ErrRetentionPolicyInvalid) {
		t.Fatalf("out-of-range days must be rejected, got %v", err)
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "", MaxAgeDays: 30}); !errors.Is(err, ErrRetentionPolicyInvalid) {
		t.Fatalf("organization is required, got %v", err)
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 30, MaxMemoriesPerProject: 20000}); !errors.Is(err, ErrRetentionPolicyInvalid) {
		t.Fatalf("count ceiling must be validated, got %v", err)
	}
	// Ausência de política é explícita.
	if _, err := store.RetentionPolicyForOrganization("org-a"); !errors.Is(err, ErrRetentionNotConfigured) {
		t.Fatalf("missing policy error = %v", err)
	}

	saved, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 30, MaxMemoriesPerProject: 2})
	if err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if saved.UpdatedAt.IsZero() {
		t.Fatal("policy must record when it was updated")
	}
	// Persistência: reabrir o store lê a política do disco.
	reopened, err := NewContextStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.RetentionPolicyForOrganization("org-a")
	if err != nil || loaded.MaxAgeDays != 30 || loaded.MaxMemoriesPerProject != 2 {
		t.Fatalf("reloaded policy = %+v err=%v", loaded, err)
	}
	// Isolamento: outra organização não enxerga essa política.
	if _, err := reopened.RetentionPolicyForOrganization("org-b"); !errors.Is(err, ErrRetentionNotConfigured) {
		t.Fatalf("org-b must not see org-a policy, got %v", err)
	}
}

func TestApplyRetentionRemovesAgedAndTrimsByCount(t *testing.T) {
	store, project := newRetentionStore(t)
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 30, MaxMemoriesPerProject: 2}); err != nil {
		t.Fatal(err)
	}
	// 3 memórias fora da janela e 4 recentes: ficam apenas as 2 mais novas.
	seedRetentionMemories(t, store, project.ID, "velha", 3, 60*24*time.Hour)
	seedRetentionMemories(t, store, project.ID, "nova", 4, time.Minute)
	if got := len(store.ExportMemories(project.ID, false)); got != 7 {
		t.Fatalf("setup memories = %d", got)
	}

	result, err := store.ApplyRetentionForOrganization("org-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.ProjectsScanned != 1 {
		t.Fatalf("projects scanned = %d", result.ProjectsScanned)
	}
	if result.RemovedByAge != 3 {
		t.Fatalf("removed by age = %d, want 3", result.RemovedByAge)
	}
	if result.RemovedByCount != 2 {
		t.Fatalf("removed by count = %d, want 2", result.RemovedByCount)
	}
	remaining := store.ExportMemories(project.ID, false)
	if len(remaining) != 2 {
		t.Fatalf("remaining = %d, want 2", len(remaining))
	}
	// As duas mantidas são as MAIS RECENTES do conjunto recente (as "velhas"
	// saíram todas por idade e as excedentes por contagem).
	kept := map[string]bool{}
	for _, memory := range remaining {
		kept[memory.ID] = true
	}
	if !kept["nova_000000"] || !kept["nova_000001"] {
		t.Fatalf("kept memories must be the newest: %+v", remaining)
	}

	// Idempotência: reaplicar não remove nada.
	again, err := store.ApplyRetentionForOrganization("org-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("reapply: %v", err)
	}
	if again.RemovedByAge != 0 || again.RemovedByCount != 0 {
		t.Fatalf("reapply removed something: %+v", again)
	}
	if len(store.ExportMemories(project.ID, false)) != 2 {
		t.Fatal("reapply changed the stored memories")
	}
}

func TestApplyRetentionPreservesUndatedMemoriesAndOtherOrganizations(t *testing.T) {
	store, project := newRetentionStore(t)
	otherProject, err := store.CreateProject("projeto alheio", t.TempDir(), "org-b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 1, MaxMemoriesPerProject: 1}); err != nil {
		t.Fatal(err)
	}
	seedRetentionMemories(t, store, project.ID, "antiga", 2, 30*24*time.Hour)
	if _, err := store.AddMemory(Memory{ID: "mem_sem_data", ProjectID: project.ID, Kind: "fact", Content: "sem data"}); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	for index, memory := range store.memories[project.ID] {
		if memory.ID == "mem_sem_data" {
			memory.CreatedAt = time.Time{}
			store.memories[project.ID][index] = memory
		}
	}
	store.mu.Unlock()
	seedRetentionMemories(t, store, otherProject.ID, "alheia", 3, 90*24*time.Hour)

	result, err := store.ApplyRetentionForOrganization("org-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.ProjectsScanned != 1 {
		t.Fatalf("retention must only scan the organization's projects: %+v", result)
	}
	kept := store.ExportMemories(project.ID, false)
	if len(kept) != 1 || kept[0].ID != "mem_sem_data" {
		t.Fatalf("undated memory must be preserved and the rest removed: %+v", kept)
	}
	if got := len(store.ExportMemories(otherProject.ID, false)); got != 3 {
		t.Fatalf("other organization must be untouched, got %d", got)
	}
}

func TestApplyRetentionWithoutPolicyFailsExplicitly(t *testing.T) {
	store, project := newRetentionStore(t)
	seedRetentionMemories(t, store, project.ID, "antiga", 2, 90*24*time.Hour)
	if _, err := store.ApplyRetentionForOrganization("org-a", time.Now().UTC()); !errors.Is(err, ErrRetentionNotConfigured) {
		t.Fatalf("apply without policy error = %v", err)
	}
	if got := len(store.ExportMemories(project.ID, false)); got != 2 {
		t.Fatalf("nothing may be removed without a policy, got %d", got)
	}
}

func TestRetentionPolicyPersistFailureRollsBack(t *testing.T) {
	store, _ := newRetentionStore(t)
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 30}); err != nil {
		t.Fatal(err)
	}
	// Substitui o diretório de retenção por um arquivo: a gravação falha.
	retentionDir := filepath.Join(store.root, "retention")
	if err := os.RemoveAll(retentionDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retentionDir, []byte("bloqueio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 45}); err == nil {
		t.Fatal("expected a persistence failure")
	}
	kept, err := store.RetentionPolicyForOrganization("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if kept.MaxAgeDays != 30 {
		t.Fatalf("failed write must keep the previous policy, got %+v", kept)
	}
	// Organização nova sem política anterior não pode ficar "meio gravada".
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-nova", MaxAgeDays: 10}); err == nil {
		t.Fatal("expected a persistence failure for the new organization")
	}
	if _, err := store.RetentionPolicyForOrganization("org-nova"); !errors.Is(err, ErrRetentionNotConfigured) {
		t.Fatalf("failed write must not register a policy, got %v", err)
	}
}
