package agent

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

// TestStudioCanvasInteractiveWorkflow tests the full visual interactive cycle:
// create project -> apply components -> move/edit -> undo -> redo -> preview -> export with SHA-256.
func TestStudioCanvasInteractiveWorkflow(t *testing.T) {
	ctx := context.Background()
	service, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create builder service: %v", err)
	}

	// 1. Create a project
	project, err := service.Create(ctx, BuilderSpec{
		Name: "Interactive Landing Page",
		Kind: BuilderWebsite,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	if project.Version != 1 {
		t.Errorf("expected initial version 1, got %d", project.Version)
	}

	// 2. Add visual components (drag-and-drop simulation)
	initialComponents := []VisualComponent{
		{
			ID:   "hero_title",
			Type: "heading",
			Props: map[string]string{
				"text":  "Construído com Ollama Full Studio",
				"level": "h1",
			},
			X:      20,
			Y:      40,
			Width:  600,
			Height: 80,
		},
		{
			ID:   "cta_btn",
			Type: "button",
			Props: map[string]string{
				"label": "Começar Agora",
			},
			X:      20,
			Y:      140,
			Width:  160,
			Height: 44,
		},
	}

	updated, err := service.ApplyVisualComponents(ctx, project.ID, initialComponents)
	if err != nil {
		t.Fatalf("failed to apply visual components: %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("expected version 2 after apply, got %d", updated.Version)
	}
	if len(updated.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(updated.Components))
	}
	if len(updated.UndoStack) != 1 {
		t.Errorf("expected 1 item on undo stack, got %d", len(updated.UndoStack))
	}

	// 3. Move component and add another component
	movedComponents := []VisualComponent{
		{
			ID:   "hero_title",
			Type: "heading",
			Props: map[string]string{
				"text":  "Construído com Ollama Full Studio - Atualizado",
				"level": "h1",
			},
			X:      50, // moved
			Y:      60, // moved
			Width:  650,
			Height: 90,
		},
		{
			ID:   "cta_btn",
			Type: "button",
			Props: map[string]string{
				"label": "Começar Agora",
			},
			X:      50,
			Y:      170,
			Width:  160,
			Height: 44,
		},
		{
			ID:   "feature_card",
			Type: "card",
			Props: map[string]string{
				"title": "Zero-Trust Nativo",
				"body":  "Total controle local e privacidade.",
			},
			X:      50,
			Y:      240,
			Width:  300,
			Height: 180,
		},
	}

	moved, err := service.ApplyVisualComponents(ctx, project.ID, movedComponents)
	if err != nil {
		t.Fatalf("failed to apply moved components: %v", err)
	}
	if moved.Version != 3 {
		t.Errorf("expected version 3, got %d", moved.Version)
	}
	if len(moved.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(moved.Components))
	}
	if moved.Components[0].X != 50 || moved.Components[0].Y != 60 {
		t.Errorf("expected moved coordinates (50, 60), got (%d, %d)", moved.Components[0].X, moved.Components[0].Y)
	}

	// 4. Test Undo
	undone, err := service.Undo(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to undo: %v", err)
	}
	if len(undone.Components) != 2 {
		t.Fatalf("expected 2 components after undo, got %d", len(undone.Components))
	}
	if undone.Components[0].X != 20 || undone.Components[0].Y != 40 {
		t.Errorf("expected restored coordinates (20, 40), got (%d, %d)", undone.Components[0].X, undone.Components[0].Y)
	}
	if len(undone.RedoStack) != 1 {
		t.Errorf("expected 1 item on redo stack, got %d", len(undone.RedoStack))
	}

	// 5. Test Redo
	redone, err := service.Redo(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to redo: %v", err)
	}
	if len(redone.Components) != 3 {
		t.Fatalf("expected 3 components after redo, got %d", len(redone.Components))
	}
	if redone.Components[0].X != 50 || redone.Components[0].Y != 60 {
		t.Errorf("expected redone coordinates (50, 60), got (%d, %d)", redone.Components[0].X, redone.Components[0].Y)
	}

	// 6. Test Preview
	previewed, artifact, err := service.Preview(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to preview: %v", err)
	}
	if previewed.Status != "preview" {
		t.Errorf("expected preview status, got %s", previewed.Status)
	}
	if artifact.SHA256 == "" {
		t.Errorf("expected non-empty artifact SHA256")
	}

	// 7. Test Export with Traceable Checksum
	exported, zipPath, err := service.Export(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}
	if exported.ExportChecksum == "" {
		t.Fatal("expected non-empty ExportChecksum")
	}

	// Verify the archive exists and checksum matches
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("failed to read exported zip: %v", err)
	}
	h := sha256.Sum256(data)
	computedChecksum := hex.EncodeToString(h[:])
	if exported.ExportChecksum != computedChecksum {
		t.Fatalf("checksum mismatch: expected %s, got %s", computedChecksum, exported.ExportChecksum)
	}

	// Verify zip contents
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer reader.Close()
	foundIndex := false
	foundVisual := false
	for _, f := range reader.File {
		if f.Name == "index.html" {
			foundIndex = true
		}
		if f.Name == "visual.json" {
			foundVisual = true
		}
	}
	if !foundIndex || !foundVisual {
		t.Errorf("missing expected files in zip archive: index=%v, visual=%v", foundIndex, foundVisual)
	}
}
