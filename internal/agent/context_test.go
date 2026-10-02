package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testEmbedder map[string][]float32

func (e testEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return e[text], nil
}

func TestSemanticMemorySearchRanksByCosineSimilarity(t *testing.T) {
	store, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	store.SetEmbedder(testEmbedder{
		"alpha": {1, 0},
		"beta":  {0.8, 0.2},
		"query": {1, 0},
	})
	if _, err := store.AddMemoryContext(context.Background(), Memory{ProjectID: "project", Kind: "note", Content: "alpha", Confidence: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryContext(context.Background(), Memory{ProjectID: "project", Kind: "note", Content: "beta", Confidence: 1}); err != nil {
		t.Fatal(err)
	}
	memories, err := store.SearchMemoriesContext(context.Background(), "project", "query", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(memories) != 2 || memories[0].Content != "alpha" {
		t.Fatalf("memories = %+v", memories)
	}
}

func TestRetrieveRelevantGatesByScoreAndKeepsProvenance(t *testing.T) {
	store, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	store.SetEmbedder(testEmbedder{
		"documento sobre gatos":  {1, 0},
		"documento sobre carros": {0, 1}, // orthogonal to the "gatos" query
		"gatos":                  {1, 0}, // query vector (already lowercase)
		"avioes":                 {0, -1},
	})
	cat := Memory{ProjectID: "project", Kind: "doc", Content: "documento sobre gatos", Source: "gatos.txt#1", Confidence: 1}
	car := Memory{ProjectID: "project", Kind: "doc", Content: "documento sobre carros", Source: "carros.txt#1", Confidence: 1}
	if _, err := store.AddMemoryContext(context.Background(), cat); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryContext(context.Background(), car); err != nil {
		t.Fatal(err)
	}

	// A relevant query returns only the matching source, with its provenance.
	hits, err := store.RetrieveRelevant(context.Background(), "project", "gatos", 10, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("relevant query returned %d hits, want 1: %+v", len(hits), hits)
	}
	if hits[0].Memory.Source != "gatos.txt#1" {
		t.Fatalf("missing provenance for citation: %+v", hits[0].Memory)
	}
	if hits[0].Score < 0.9 {
		t.Fatalf("relevant score = %v, want ~1.0", hits[0].Score)
	}

	// A query with no relevant source returns nothing, so the answer layer can
	// honestly say it does not know instead of citing a weak match.
	none, err := store.RetrieveRelevant(context.Background(), "project", "avioes", 10, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("irrelevant query returned %d hits, want 0: %+v", len(none), none)
	}
}

func TestProjectAndScheduleCRUDPersistsAndDeletes(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("Workspace", "", "org_test")
	if err != nil {
		t.Fatal(err)
	}
	if got := store.ListProjects(); len(got) != 1 || got[0].OrganizationID != "org_test" {
		t.Fatalf("projects = %+v", got)
	}
	updatedProject, err := store.UpdateProject(project.ID, "Workspace atualizado", "")
	if err != nil || updatedProject.Name != "Workspace atualizado" {
		t.Fatalf("updated project = %+v, err = %v", updatedProject, err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "verificar", IntervalSeconds: 60, OrganizationID: "org_test"})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.ListSchedulesForOrganization("org_test"); len(got) != 1 || got[0].ID != schedule.ID {
		t.Fatalf("schedules = %+v", got)
	}
	if _, err := store.UpdateSchedule(schedule.ID, Schedule{Objective: "verificar atualizado", IntervalSeconds: 120, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteSchedule(schedule.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProject(project.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.ListProjects()) != 0 || len(store.ListSchedules()) != 0 {
		t.Fatalf("store was not deleted: projects=%v schedules=%v", store.ListProjects(), store.ListSchedules())
	}
}

func TestProjectRootCreateAndUpdateStayInsideRuntimeWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	store, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetWorkspaceRoot(workspace); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(workspace, "project")
	if err := os.MkdirAll(inside, 0o700); err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("Workspace", inside)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject("Outside", outside); err == nil || !strings.Contains(err.Error(), "outside the runtime workspace") {
		t.Fatalf("outside create error = %v", err)
	}
	if _, err := store.UpdateProject(project.ID, "Outside", outside); err == nil || !strings.Contains(err.Error(), "outside the runtime workspace") {
		t.Fatalf("outside update error = %v", err)
	}
	current, err := store.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Root != project.Root {
		t.Fatalf("project root changed after rejected update: %q", current.Root)
	}
}
