package attachments

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type zipFile struct {
	name string
	body []byte
	mode fs.FileMode
}

func writeZip(t *testing.T, files []zipFile) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range files {
		header := &zip.FileHeader{Name: f.name, Method: zip.Deflate}
		if f.mode != 0 {
			header.SetMode(f.mode)
		}
		out, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write(f.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "in.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

var textExts = []string{"go", "md", "txt"}

func TestReadZipTextKeepsTextAndSkipsTheRest(t *testing.T) {
	path := writeZip(t, []zipFile{
		{name: "src/main.go", body: []byte("package main")},
		{name: "README.md", body: []byte("# hi")},
		{name: "logo.png", body: []byte{0x89, 'P', 'N', 'G'}},
		{name: "../escape.txt", body: []byte("x")},
		{name: "/abs.txt", body: []byte("x")},
		{name: "link.txt", body: []byte("target"), mode: fs.ModeSymlink | 0o777},
		{name: "inner.zip", body: []byte("PK")},
	})

	report, err := ReadZipText(path, textExts, DefaultZipLimits)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range report.Entries {
		names = append(names, e.Name)
	}
	if got := strings.Join(names, ","); got != "src/main.go,README.md" {
		t.Fatalf("entries = %s", got)
	}
	if len(report.Skipped) != 5 {
		t.Fatalf("skipped = %v, want 5", report.Skipped)
	}
}

func TestReadZipTextEnforcesLimits(t *testing.T) {
	big := bytes.Repeat([]byte("a"), 64<<10)

	t.Run("per file size is skipped", func(t *testing.T) {
		path := writeZip(t, []zipFile{{name: "big.txt", body: big}, {name: "ok.txt", body: []byte("ok")}})
		limits := DefaultZipLimits
		limits.MaxFileBytes = 1 << 10
		limits.MaxRatio = 1 << 20
		report, err := ReadZipText(path, textExts, limits)
		if err != nil || len(report.Entries) != 1 || report.Entries[0].Name != "ok.txt" {
			t.Fatalf("report=%+v err=%v", report, err)
		}
	})

	t.Run("compression ratio aborts", func(t *testing.T) {
		path := writeZip(t, []zipFile{{name: "bomb.txt", body: big}})
		limits := DefaultZipLimits
		limits.MaxRatio = 10
		if _, err := ReadZipText(path, textExts, limits); !errors.Is(err, ErrZipLimit) {
			t.Fatalf("err = %v, want ErrZipLimit", err)
		}
	})

	t.Run("entry count aborts", func(t *testing.T) {
		path := writeZip(t, []zipFile{{name: "a.txt", body: []byte("a")}, {name: "b.txt", body: []byte("b")}})
		limits := DefaultZipLimits
		limits.MaxEntries = 1
		if _, err := ReadZipText(path, textExts, limits); !errors.Is(err, ErrZipLimit) {
			t.Fatalf("err = %v, want ErrZipLimit", err)
		}
	})

	t.Run("total size aborts", func(t *testing.T) {
		path := writeZip(t, []zipFile{{name: "a.txt", body: []byte("aaaa")}, {name: "b.txt", body: []byte("bbbb")}})
		limits := DefaultZipLimits
		limits.MaxTotalBytes = 6
		if _, err := ReadZipText(path, textExts, limits); !errors.Is(err, ErrZipLimit) {
			t.Fatalf("err = %v, want ErrZipLimit", err)
		}
	})
}

func TestReadZipTextRejectsNonZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.zip")
	if err := os.WriteFile(path, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadZipText(path, textExts, DefaultZipLimits); err == nil {
		t.Fatal("expected error for non-zip input")
	}
}
