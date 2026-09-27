package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UploadState e o estado de uma sessao de upload grande (chunked/resumable).
type UploadState string

const (
	UploadReceiving UploadState = "receiving"
	UploadCompleted UploadState = "completed"
	UploadCancelled UploadState = "cancelled"
)

// UploadSession rastreia um upload grande por tenant.
type UploadSession struct {
	ID             string      `json:"id"`
	ProjectID      string      `json:"project_id"`
	OrganizationID string      `json:"organization_id"`
	Filename       string      `json:"filename"`
	TotalSize      int64       `json:"total_size"`
	ChunkSize      int64       `json:"chunk_size"`
	ReceivedBytes  int64       `json:"received_bytes"`
	ReceivedChunks int         `json:"received_chunks"`
	ExpectedSHA256 string      `json:"expected_sha256,omitempty"`
	State          UploadState `json:"state"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	ExpiresAt      time.Time   `json:"expires_at"`
	FinalPath      string      `json:"-"`
	tempPath       string
}

var (
	ErrUploadNotFound      = errors.New("upload session not found")
	ErrUploadForbidden     = errors.New("upload session is outside the active organization")
	ErrUploadNotReceiving  = errors.New("upload session is not receiving")
	ErrUploadInvalidChunk  = errors.New("upload chunk is invalid or out of bounds")
	ErrUploadSizeInvalid   = errors.New("upload total size is invalid")
	ErrUploadQuotaExceeded = errors.New("upload exceeds organization quota")
	ErrUploadIncomplete    = errors.New("upload is not complete")
	ErrUploadHashMismatch  = errors.New("upload sha256 does not match")
	ErrUploadFilename      = errors.New("upload filename is invalid")
)

// UploadManager guarda sessoes de upload em memoria com temp files em disco.
type UploadManager struct {
	mu       sync.Mutex
	root     string
	quota    int64
	maxSize  int64
	ttl      time.Duration
	sessions map[string]*UploadSession
}

var uploadOwnedFilenamePattern = regexp.MustCompile(`^upl_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?:\.part|-.+)$`)

// NewUploadManager cria o gerenciador. quotaBytes limita o total por organizacao.
func NewUploadManager(root string, quotaBytes, maxFileBytes int64) (*UploadManager, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("upload root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if quotaBytes <= 0 {
		quotaBytes = 5 << 30 // 5 GiB
	}
	if maxFileBytes <= 0 {
		maxFileBytes = 1 << 30 // 1 GiB
	}
	manager := &UploadManager{
		root:     root,
		quota:    quotaBytes,
		maxSize:  maxFileBytes,
		ttl:      24 * time.Hour,
		sessions: map[string]*UploadSession{},
	}
	if err := manager.cleanupExpiredFiles(time.Now().UTC()); err != nil {
		return nil, err
	}
	return manager, nil
}

func sanitizeUploadFilename(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	// Rejeita qualquer componente de caminho ou traversal no filename.
	if trimmed == "" || trimmed == "." || trimmed == ".." ||
		strings.ContainsAny(trimmed, "/\\") || strings.Contains(trimmed, "..") {
		return "", ErrUploadFilename
	}
	base := filepath.Base(trimmed)
	if base != trimmed {
		return "", ErrUploadFilename
	}
	return base, nil
}

// usedBytesLocked soma o uso ativo/completo de uma organizacao.
func (m *UploadManager) usedBytesLocked(organizationID string) int64 {
	var used int64
	for _, s := range m.sessions {
		if s.OrganizationID == organizationID && s.State != UploadCancelled {
			used += s.ReceivedBytes
		}
	}
	return used
}

func (m *UploadManager) removeSessionFilesLocked(session *UploadSession) error {
	for _, path := range []string{session.tempPath, session.FinalPath} {
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		if filepath.Dir(path) != m.root {
			return errors.New("upload path escaped its storage root")
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (m *UploadManager) sweepExpiredLocked(now time.Time) error {
	for id, session := range m.sessions {
		if session.ExpiresAt.IsZero() || now.Before(session.ExpiresAt) {
			continue
		}
		if err := m.removeSessionFilesLocked(session); err != nil {
			return fmt.Errorf("remove expired upload %s: %w", id, err)
		}
		delete(m.sessions, id)
	}
	return nil
}

// cleanupExpiredFiles removes only upload-owned regular files older than the
// session TTL. Session metadata is in memory, so they cannot resume after a
// restart; recent files are left untouched for another live process.
func (m *UploadManager) cleanupExpiredFiles(now time.Time) error {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return err
	}
	cutoff := now.Add(-m.ttl)
	for _, entry := range entries {
		name := entry.Name()
		if !uploadOwnedFilenamePattern.MatchString(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(m.root, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// StartUpload cria uma sessao (receiving) validando tamanho, quota e filename.
func (m *UploadManager) StartUpload(organizationID, projectID, filename string, totalSize, chunkSize int64, expectedSHA256 string) (UploadSession, error) {
	organizationID = strings.TrimSpace(organizationID)
	name, err := sanitizeUploadFilename(filename)
	if err != nil {
		return UploadSession{}, err
	}
	if totalSize <= 0 || totalSize > m.maxSize {
		return UploadSession{}, ErrUploadSizeInvalid
	}
	if chunkSize <= 0 || chunkSize > totalSize {
		chunkSize = totalSize
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.sweepExpiredLocked(time.Now().UTC()); err != nil {
		return UploadSession{}, err
	}
	if m.usedBytesLocked(organizationID)+totalSize > m.quota {
		return UploadSession{}, ErrUploadQuotaExceeded
	}
	id := "upl_" + uuid.NewString()
	temp := filepath.Join(m.root, id+".part")
	f, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return UploadSession{}, err
	}
	_ = f.Close()
	now := time.Now().UTC()
	session := &UploadSession{
		ID:             id,
		ProjectID:      strings.TrimSpace(projectID),
		OrganizationID: organizationID,
		Filename:       name,
		TotalSize:      totalSize,
		ChunkSize:      chunkSize,
		ExpectedSHA256: strings.ToLower(strings.TrimSpace(expectedSHA256)),
		State:          UploadReceiving,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(m.ttl),
		tempPath:       temp,
	}
	m.sessions[id] = session
	return *session, nil
}

func (m *UploadManager) getLocked(organizationID, uploadID string) (*UploadSession, error) {
	s, ok := m.sessions[strings.TrimSpace(uploadID)]
	if !ok {
		return nil, ErrUploadNotFound
	}
	if s.OrganizationID != strings.TrimSpace(organizationID) {
		return nil, ErrUploadForbidden
	}
	if !s.ExpiresAt.IsZero() && !time.Now().UTC().Before(s.ExpiresAt) {
		if err := m.removeSessionFilesLocked(s); err != nil {
			return nil, err
		}
		delete(m.sessions, strings.TrimSpace(uploadID))
		return nil, ErrUploadNotFound
	}
	return s, nil
}

// AppendChunk escreve data no offset (resumable), com limites e quota.
func (m *UploadManager) AppendChunk(organizationID, uploadID string, offset int64, data []byte) (UploadSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.getLocked(organizationID, uploadID)
	if err != nil {
		return UploadSession{}, err
	}
	if s.State != UploadReceiving {
		return UploadSession{}, ErrUploadNotReceiving
	}
	// Append sequencial (resumable): o cliente retoma consultando ReceivedBytes.
	if offset != s.ReceivedBytes || int64(len(data)) == 0 || offset+int64(len(data)) > s.TotalSize {
		return UploadSession{}, ErrUploadInvalidChunk
	}
	if m.usedBytesLocked(s.OrganizationID)+int64(len(data)) > m.quota {
		return UploadSession{}, ErrUploadQuotaExceeded
	}
	f, err := os.OpenFile(s.tempPath, os.O_WRONLY, 0o600)
	if err != nil {
		return UploadSession{}, err
	}
	_, wErr := f.WriteAt(data, offset)
	cErr := f.Close()
	if wErr != nil {
		return UploadSession{}, wErr
	}
	if cErr != nil {
		return UploadSession{}, cErr
	}
	end := offset + int64(len(data))
	if end > s.ReceivedBytes {
		s.ReceivedBytes = end
	}
	s.ReceivedChunks++
	s.UpdatedAt = time.Now().UTC()
	return *s, nil
}

// CancelUpload remove o temp e marca cancelled.
func (m *UploadManager) CancelUpload(organizationID, uploadID string) (UploadSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.getLocked(organizationID, uploadID)
	if err != nil {
		return UploadSession{}, err
	}
	_ = os.Remove(s.tempPath)
	s.State = UploadCancelled
	s.UpdatedAt = time.Now().UTC()
	return *s, nil
}

// FinalizeUpload valida completude + sha256 e faz rename atomico para final.
func (m *UploadManager) FinalizeUpload(organizationID, uploadID string) (UploadSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.getLocked(organizationID, uploadID)
	if err != nil {
		return UploadSession{}, err
	}
	if s.State != UploadReceiving {
		return UploadSession{}, ErrUploadNotReceiving
	}
	if s.ReceivedBytes != s.TotalSize {
		return UploadSession{}, ErrUploadIncomplete
	}
	sum, err := sha256File(s.tempPath)
	if err != nil {
		return UploadSession{}, err
	}
	if s.ExpectedSHA256 != "" && sum != s.ExpectedSHA256 {
		return UploadSession{}, ErrUploadHashMismatch
	}
	final := filepath.Join(m.root, s.ID+"-"+s.Filename)
	if err := os.Rename(s.tempPath, final); err != nil {
		return UploadSession{}, err
	}
	s.State = UploadCompleted
	s.FinalPath = final
	s.ExpectedSHA256 = sum
	s.UpdatedAt = time.Now().UTC()
	return *s, nil
}

// GetUploadForOrganization le uma sessao (org-scoped).
func (m *UploadManager) GetUploadForOrganization(organizationID, uploadID string) (UploadSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.getLocked(organizationID, uploadID)
	if err != nil {
		return UploadSession{}, err
	}
	return *s, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
