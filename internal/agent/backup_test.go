package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeBackupFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBackupDataRoundTripExcludesSecrets(t *testing.T) {
	src := t.TempDir()
	writeBackupFixture(t, src, "missions/m1.json", `{"id":"m1"}`)
	writeBackupFixture(t, src, "context/notes/a.txt", "hello world")
	writeBackupFixture(t, src, "companies/c1.json", `{"id":"c1"}`)
	// Secret-bearing dirs that must NOT be captured.
	writeBackupFixture(t, src, "auth/credentials.json", "SECRET-CREDENTIAL")
	writeBackupFixture(t, src, "push/tokens.json", "SECRET-TOKEN")

	var buf bytes.Buffer
	manifest, err := BackupData(src, &buf)
	if err != nil {
		t.Fatalf("BackupData: %v", err)
	}
	if manifest.SHA256 == "" {
		t.Fatal("manifest missing SHA256")
	}
	if manifest.Files != 3 {
		t.Fatalf("manifest.Files = %d, want 3 (secrets excluded)", manifest.Files)
	}
	// The secret contents must not appear anywhere in the archive bytes.
	if bytes.Contains(buf.Bytes(), []byte("SECRET-CREDENTIAL")) || bytes.Contains(buf.Bytes(), []byte("SECRET-TOKEN")) {
		t.Fatal("backup archive leaked secret content")
	}

	dst := t.TempDir()
	if err := RestoreData(bytes.NewReader(buf.Bytes()), dst, manifest.SHA256); err != nil {
		t.Fatalf("RestoreData: %v", err)
	}

	// Restored files must match the originals...
	for _, rel := range []string{"missions/m1.json", "context/notes/a.txt", "companies/c1.json"} {
		got, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("restored %s: %v", rel, err)
		}
		want, _ := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		if !bytes.Equal(got, want) {
			t.Fatalf("restored %s mismatch: got %q want %q", rel, got, want)
		}
	}
	// ...and the excluded secret dirs must be absent.
	for _, rel := range []string{"auth", "push"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); !os.IsNotExist(err) {
			t.Fatalf("restore recreated excluded dir %q (err=%v)", rel, err)
		}
	}
}

func TestRestoreDataRejectsChecksumMismatch(t *testing.T) {
	src := t.TempDir()
	writeBackupFixture(t, src, "context/a.txt", "data")
	var buf bytes.Buffer
	if _, err := BackupData(src, &buf); err != nil {
		t.Fatal(err)
	}
	err := RestoreData(bytes.NewReader(buf.Bytes()), t.TempDir(), "deadbeef")
	if err != ErrBackupChecksumMismatch {
		t.Fatalf("RestoreData with wrong checksum: err = %v, want ErrBackupChecksumMismatch", err)
	}
}
