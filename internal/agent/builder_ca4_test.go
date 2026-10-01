package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestBuilderCASRejectsStaleConcurrentWriter(t *testing.T) {
	service, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.Create(context.Background(), BuilderSpec{
		Name: "Concurrent Studio", Kind: BuilderWebsite,
		Components: []VisualComponent{{ID: "initial", Type: "text"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{"writer-a", "writer-b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, callErr := service.ApplyVisualComponentsCAS(context.Background(), project.ID, project.Version, []VisualComponent{{ID: id, Type: "card"}})
			results <- callErr
		}(id)
	}
	wg.Wait()
	close(results)

	var successes, conflicts int
	for callErr := range results {
		switch {
		case callErr == nil:
			successes++
		case errors.Is(callErr, ErrBuilderVersionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent writer error: %v", callErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want exactly one of each", successes, conflicts)
	}
	final, err := service.Get(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Version != project.Version+1 || len(final.Components) != 1 {
		t.Fatalf("final project corrupted: %+v", final)
	}
}

func TestBuilderExportIsInvalidatedAfterVersionedEdit(t *testing.T) {
	root := t.TempDir()
	service, err := NewBuilderService(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.Create(context.Background(), BuilderSpec{
		Name: "Expiring Export", Kind: BuilderWebsite,
		Components: []VisualComponent{{ID: "v1", Type: "text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	exported, archivePath, err := service.Export(context.Background(), project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exported.ExportVersion != exported.Version || exported.ExportChecksum == "" {
		t.Fatalf("export not bound to version: %+v", exported)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatal(err)
	}

	updated, err := service.ApplyVisualComponentsCAS(context.Background(), project.ID, exported.Version, []VisualComponent{{ID: "v2", Type: "button"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExportVersion != 0 || updated.ExportChecksum != "" || updated.ExportPath != "" {
		t.Fatalf("stale export metadata survived edit: %+v", updated)
	}
	if _, err := os.Stat(filepath.Join(root, project.ID+".zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale archive still exists, stat err=%v", err)
	}
	if _, _, err := service.Export(context.Background(), project.ID); err != nil {
		t.Fatal(err)
	}
	current, err := service.Get(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ExportVersion != current.Version || current.ExportChecksum == "" {
		t.Fatalf("current export not restored: %+v", current)
	}
}
