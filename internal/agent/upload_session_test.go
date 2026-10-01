package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func newMgr(t *testing.T, quota, maxFile int64) *UploadManager {
	t.Helper()
	m, err := NewUploadManager(t.TempDir(), quota, maxFile)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestUploadSessionResumableFinalizeAndHash(t *testing.T) {
	m := newMgr(t, 1<<20, 1<<20)
	content := []byte("abcdefghij0123456789ZZZZZ") // 25 bytes
	sum := sha256Hex(content)

	s, err := m.StartUpload("org-a", "prj", "data.bin", int64(len(content)), 10, sum)
	if err != nil || s.State != UploadReceiving {
		t.Fatalf("start: %+v err=%v", s, err)
	}
	// Append sequencial [0:10]; consulta ReceivedBytes (resume) e continua.
	part, err := m.AppendChunk("org-a", s.ID, 0, content[:10])
	if err != nil || part.ReceivedBytes != 10 {
		t.Fatalf("chunk1: %+v err=%v", part, err)
	}
	// Finalize incompleto -> erro.
	if _, err := m.FinalizeUpload("org-a", s.ID); !errors.Is(err, ErrUploadIncomplete) {
		t.Fatalf("incompleto err=%v", err)
	}
	// Retoma a partir de ReceivedBytes.
	if _, err := m.AppendChunk("org-a", s.ID, part.ReceivedBytes, content[10:]); err != nil {
		t.Fatal(err)
	}
	done, err := m.FinalizeUpload("org-a", s.ID)
	if err != nil || done.State != UploadCompleted || done.ExpectedSHA256 != sum {
		t.Fatalf("finalize: %+v err=%v", done, err)
	}
	got, err := os.ReadFile(done.FinalPath)
	if err != nil || string(got) != string(content) {
		t.Fatalf("final content mismatch: err=%v", err)
	}
}

func TestUploadSessionHashMismatch(t *testing.T) {
	m := newMgr(t, 1<<20, 1<<20)
	content := []byte("hello world")
	s, err := m.StartUpload("org-a", "p", "f.bin", int64(len(content)), 0, sha256Hex([]byte("different")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AppendChunk("org-a", s.ID, 0, content); err != nil {
		t.Fatal(err)
	}
	if _, err := m.FinalizeUpload("org-a", s.ID); !errors.Is(err, ErrUploadHashMismatch) {
		t.Fatalf("hash mismatch err=%v", err)
	}
}

func TestUploadSessionQuotaAndSizeAndFilename(t *testing.T) {
	m := newMgr(t, 100, 1000) // quota 100 bytes
	if _, err := m.StartUpload("org-a", "p", "big.bin", 200, 0, ""); !errors.Is(err, ErrUploadQuotaExceeded) {
		t.Fatalf("quota err=%v", err)
	}
	if _, err := m.StartUpload("org-a", "p", "zero.bin", 0, 0, ""); !errors.Is(err, ErrUploadSizeInvalid) {
		t.Fatalf("size err=%v", err)
	}
	if _, err := m.StartUpload("org-a", "p", "../evil", 10, 0, ""); !errors.Is(err, ErrUploadFilename) {
		t.Fatalf("filename err=%v", err)
	}
}

func TestUploadSessionInvalidChunkAndCancel(t *testing.T) {
	m := newMgr(t, 1<<20, 1<<20)
	s, err := m.StartUpload("org-a", "p", "f.bin", 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	// Chunk fora dos limites.
	if _, err := m.AppendChunk("org-a", s.ID, 5, []byte("0123456789")); !errors.Is(err, ErrUploadInvalidChunk) {
		t.Fatalf("chunk oob err=%v", err)
	}
	cancelled, err := m.CancelUpload("org-a", s.ID)
	if err != nil || cancelled.State != UploadCancelled {
		t.Fatalf("cancel: %+v err=%v", cancelled, err)
	}
	// Apos cancelar, append falha (nao receiving).
	if _, err := m.AppendChunk("org-a", s.ID, 0, []byte("x")); !errors.Is(err, ErrUploadNotReceiving) {
		t.Fatalf("append apos cancel err=%v", err)
	}
}

func TestUploadSessionRejectsCrossTenant(t *testing.T) {
	m := newMgr(t, 1<<20, 1<<20)
	s, err := m.StartUpload("org-a", "p", "f.bin", 4, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetUploadForOrganization("org-b", s.ID); !errors.Is(err, ErrUploadForbidden) {
		t.Fatalf("get cross-tenant err=%v", err)
	}
	if _, err := m.AppendChunk("org-b", s.ID, 0, []byte("data")); !errors.Is(err, ErrUploadForbidden) {
		t.Fatalf("append cross-tenant err=%v", err)
	}
	if _, err := m.CancelUpload("org-b", s.ID); !errors.Is(err, ErrUploadForbidden) {
		t.Fatalf("cancel cross-tenant err=%v", err)
	}
}

func TestUploadExpirationRemovesPartialFileAndReleasesQuota(t *testing.T) {
	m := newMgr(t, 10, 10)
	session, err := m.StartUpload("org-a", "p", "first.bin", 10, 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AppendChunk("org-a", session.ID, 0, []byte("12345")); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.sessions[session.ID].ExpiresAt = time.Now().UTC().Add(-time.Second)
	m.mu.Unlock()
	if _, err := m.GetUploadForOrganization("org-a", session.ID); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("expired upload lookup error=%v, want not found", err)
	}
	if _, err := os.Stat(session.tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired partial file remains: %v", err)
	}
	if _, err := m.StartUpload("org-a", "p", "second.bin", 10, 10, ""); err != nil {
		t.Fatalf("expired bytes still consume quota: %v", err)
	}
}

func TestUploadManagerStartupRemovesOldOwnedFiles(t *testing.T) {
	root := t.TempDir()
	const ownedID = "upl_123e4567-e89b-42d3-a456-426614174000"
	oldPart := filepath.Join(root, ownedID+".part")
	if err := os.WriteFile(oldPart, []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldFinal := filepath.Join(root, ownedID+"-report.bin")
	if err := os.WriteFile(oldFinal, []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(root, "upl_old-report.bin")
	if err := os.WriteFile(unrelated, []byte("not manager-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldPart, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldFinal, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := NewUploadManager(root, 1<<20, 1<<20); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{oldPart, oldFinal} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale upload file %s remains: %v", filepath.Base(path), err)
		}
	}
	if data, err := os.ReadFile(unrelated); err != nil || string(data) != "not manager-owned" {
		t.Fatalf("unrelated upload-like file was altered: data=%q err=%v", data, err)
	}
}
