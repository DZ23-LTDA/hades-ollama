package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Local-first backup and restore of the agent data root. Secrets (encrypted
// credentials, recovery codes, push tokens) are deliberately excluded so a
// backup can be moved between machines without carrying key material; they are
// handled separately. Integrity is verified by a SHA-256 digest of the archive.

const (
	maxBackupBytes     = 2 << 30   // 2 GiB aggregate
	maxBackupFileBytes = 256 << 20 // 256 MiB per entry
)

var (
	// ErrBackupUnsafePath is returned when an archive entry would escape the
	// restore destination (path traversal) or is otherwise unsafe.
	ErrBackupUnsafePath = errors.New("backup entry path is unsafe")
	// ErrBackupTooLarge is returned when a restore would exceed the size caps.
	ErrBackupTooLarge = errors.New("backup exceeds size limit")
	// ErrBackupChecksumMismatch is returned when the archive digest does not
	// match the expected value supplied to RestoreData.
	ErrBackupChecksumMismatch = errors.New("backup checksum mismatch")
)

// backupExcludedDirs are top-level subdirectories of the data root that are
// never captured in a backup because they hold secret material.
var backupExcludedDirs = map[string]bool{
	"auth": true,
	"push": true,
}

// BackupManifest describes a produced backup archive.
type BackupManifest struct {
	CreatedAt    time.Time `json:"created_at"`
	Files        int       `json:"files"`
	Bytes        int64     `json:"bytes"`
	SHA256       string    `json:"sha256"`
	ExcludedDirs []string  `json:"excluded_dirs"`
}

// BackupData writes a gzip-compressed tar of the agent data root to w, skipping
// secret-bearing directories and symlinks, and returns a manifest whose SHA256
// is the digest of the bytes written to w.
func BackupData(dataRoot string, w io.Writer) (BackupManifest, error) {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return BackupManifest{}, errors.New("data root is required")
	}
	manifest := BackupManifest{CreatedAt: time.Now().UTC()}
	excluded := make([]string, 0, len(backupExcludedDirs))
	for name := range backupExcludedDirs {
		excluded = append(excluded, name)
	}
	sort.Strings(excluded)
	manifest.ExcludedDirs = excluded

	hasher := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(w, hasher))
	tw := tar.NewWriter(gz)

	walkErr := filepath.WalkDir(dataRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(dataRoot, path)
		if rerr != nil {
			return rerr
		}
		if rel == "." {
			return nil
		}
		top := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		if backupExcludedDirs[top] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil // never follow or archive symlinks
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil // skip devices, sockets, pipes
		}
		hdr, herr := tar.FileInfoHeader(info, "")
		if herr != nil {
			return herr
		}
		hdr.Name = filepath.ToSlash(rel)
		if d.IsDir() {
			hdr.Name += "/"
		}
		if werr := tw.WriteHeader(hdr); werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		f, oerr := os.Open(path)
		if oerr != nil {
			return oerr
		}
		n, cerr := io.Copy(tw, f)
		_ = f.Close()
		if cerr != nil {
			return cerr
		}
		manifest.Files++
		manifest.Bytes += n
		return nil
	})
	if walkErr != nil {
		return BackupManifest{}, walkErr
	}
	if err := tw.Close(); err != nil {
		return BackupManifest{}, err
	}
	if err := gz.Close(); err != nil {
		return BackupManifest{}, err
	}
	manifest.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	return manifest, nil
}

// RestoreData verifies the archive's SHA-256 against expectedSHA (when it is
// non-empty) and extracts it into destination, rejecting path traversal,
// symlinks and oversized content. The archive is fully read and verified before
// any file is written.
func RestoreData(r io.Reader, destination, expectedSHA string) error {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return errors.New("destination is required")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxBackupBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxBackupBytes {
		return ErrBackupTooLarge
	}
	if want := strings.ToLower(strings.TrimSpace(expectedSHA)); want != "" {
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if !hmac.Equal([]byte(got), []byte(want)) {
			return ErrBackupChecksumMismatch
		}
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(filepath.FromSlash(hdr.Name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return ErrBackupUnsafePath
		}
		target := filepath.Join(destination, clean)
		if !isWithin(destination, target) {
			return ErrBackupUnsafePath
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, oerr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if oerr != nil {
				return oerr
			}
			n, cerr := io.Copy(out, io.LimitReader(tr, maxBackupFileBytes+1))
			closeErr := out.Close()
			if cerr != nil {
				return cerr
			}
			if closeErr != nil {
				return closeErr
			}
			if n > maxBackupFileBytes {
				return ErrBackupTooLarge
			}
			total += n
			if total > maxBackupBytes {
				return ErrBackupTooLarge
			}
		default:
			// Skip symlinks, hardlinks, devices: never materialize them.
		}
	}
	return nil
}
