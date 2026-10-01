// Package attachments turns user-selected archives into chat attachments.
package attachments

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// ZipLimits bounds how much of an archive is read into memory.
type ZipLimits struct {
	MaxEntries    int   // text files returned
	MaxFileBytes  int64 // per extracted file
	MaxTotalBytes int64 // across extracted files
	MaxRatio      int64 // uncompressed/compressed, zip-bomb guard
}

// DefaultZipLimits keeps an attachment batch small enough for a chat prompt.
var DefaultZipLimits = ZipLimits{MaxEntries: 200, MaxFileBytes: 2 << 20, MaxTotalBytes: 20 << 20, MaxRatio: 100}

// ZipEntry is one text file read from an archive.
type ZipEntry struct {
	Name string // slash-separated path inside the archive
	Data []byte
}

// ZipReport describes what was read and what was skipped.
type ZipReport struct {
	Entries []ZipEntry
	Skipped []string
}

var ErrZipLimit = errors.New("zip exceeds attachment limits")

// ReadZipText reads files whose extension is in allowed (lowercase, no dot)
// from the archive at zipPath. Nothing is written to disk, so path traversal
// cannot escape, but unsafe names are still skipped. Directories, symlinks,
// nested archives and non-allowed extensions are skipped and reported.
func ReadZipText(zipPath string, allowed []string, limits ZipLimits) (ZipReport, error) {
	var report ZipReport
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return report, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	allow := make(map[string]bool, len(allowed))
	for _, ext := range allowed {
		allow[strings.ToLower(ext)] = true
	}

	var total int64
	for _, file := range reader.File {
		name := file.Name
		mode := file.Mode()
		if file.FileInfo().IsDir() {
			continue
		}
		clean := path.Clean(strings.ReplaceAll(name, `\`, "/"))
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(clean), "."))
		switch {
		case !mode.IsRegular():
			report.Skipped = append(report.Skipped, name+" (não é arquivo regular)")
			continue
		case strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":"):
			report.Skipped = append(report.Skipped, name+" (caminho inseguro)")
			continue
		case !allow[ext]:
			report.Skipped = append(report.Skipped, name+" (tipo não suportado)")
			continue
		case Noisy(clean):
			report.Skipped = append(report.Skipped, name+" (dependências ou build)")
			continue
		}
		if len(report.Entries) >= limits.MaxEntries {
			return report, fmt.Errorf("%w: mais de %d arquivos", ErrZipLimit, limits.MaxEntries)
		}
		if file.UncompressedSize64 > uint64(limits.MaxFileBytes) {
			report.Skipped = append(report.Skipped, name+" (arquivo grande demais)")
			continue
		}
		if file.CompressedSize64 > 0 && int64(file.UncompressedSize64)/int64(file.CompressedSize64) > limits.MaxRatio {
			return report, fmt.Errorf("%w: taxa de compressão suspeita em %s", ErrZipLimit, name)
		}

		rc, err := file.Open()
		if err != nil {
			report.Skipped = append(report.Skipped, name+" (ilegível)")
			continue
		}
		// Never trust the header sizes: read at most one byte past the limit.
		data, err := io.ReadAll(io.LimitReader(rc, limits.MaxFileBytes+1))
		rc.Close()
		if err != nil {
			report.Skipped = append(report.Skipped, name+" (ilegível)")
			continue
		}
		if int64(len(data)) > limits.MaxFileBytes {
			report.Skipped = append(report.Skipped, name+" (arquivo grande demais)")
			continue
		}
		total += int64(len(data))
		if total > limits.MaxTotalBytes {
			return report, fmt.Errorf("%w: conteúdo total acima de %d MB", ErrZipLimit, limits.MaxTotalBytes>>20)
		}
		report.Entries = append(report.Entries, ZipEntry{Name: clean, Data: data})
	}
	return report, nil
}
