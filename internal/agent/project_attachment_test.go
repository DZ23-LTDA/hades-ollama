package agent

import "testing"

func TestSupportedProjectDocumentFilename(t *testing.T) {
	for _, name := range []string{"notes.txt", "handbook.pdf", "contract.docx", "budget.xlsx", "src/main.go"} {
		if name == "src/main.go" {
			if SupportedProjectDocumentFilename(name) {
				t.Errorf("path-bearing filename accepted: %q", name)
			}
			continue
		}
		if !SupportedProjectDocumentFilename(name) {
			t.Errorf("supported document rejected: %q", name)
		}
	}
	for _, name := range []string{"photo.png", "archive.zip", "notes.rtf", "../escape.txt", "folder\\escape.txt", ""} {
		if SupportedProjectDocumentFilename(name) {
			t.Errorf("unsupported or unsafe filename accepted: %q", name)
		}
	}
}
