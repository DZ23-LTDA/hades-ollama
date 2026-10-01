package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ocrInputLimit int64 = 25 << 20

// OCRLocal uses a pinned mission workspace for all filesystem access. Image
// bytes are read once through the root descriptor and supplied on stdin, so the
// subprocess never reopens a caller-controlled pathname.
func OCRLocal(ctx context.Context, workspace, inputPath, language string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(inputPath) == "" {
		return MediaResult{}, errors.New("ocr input is required")
	}
	if len(pinnedRoots) == 0 || pinnedRoots[0] == nil {
		return MediaResult{}, errors.New("pinned media workspace is required")
	}
	executable, err := trustedToolExecutable("tesseract")
	if err != nil {
		return MediaResult{}, errors.New("tesseract is not installed in a trusted system directory")
	}
	_, input, err := readMediaInputBoundedFromRoot(ctx, workspace, pinnedRoots[0], inputPath, ocrInputLimit)
	if err != nil {
		return MediaResult{}, err
	}
	language = strings.TrimSpace(language)
	if language == "" {
		language = "eng"
	}
	if len(language) > 64 || strings.ContainsAny(language, " ;|&\n\r\t") {
		return MediaResult{}, errors.New("invalid OCR language")
	}
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	command := exec.CommandContext(ctx, executable, "stdin", "stdout", "-l", language)
	command.Stdin = bytes.NewReader(input)
	command.Env = []string{"PATH=" + safeToolPath(), "LANG=C.UTF-8", "LC_ALL=C.UTF-8"}
	var output bytes.Buffer
	stderr := &limitedWriter{writer: &bytes.Buffer{}, limit: 1 << 20}
	command.Stdout = &limitedWriter{writer: &output, limit: 8 << 20}
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return MediaResult{}, errors.New("tesseract execution failed")
		}
		return MediaResult{}, fmt.Errorf("tesseract execution failed: %w", err)
	}
	text := RedactDLP(strings.TrimSpace(output.String()))
	if text == "" {
		return MediaResult{}, errors.New("OCR returned no text")
	}
	relativePath := filepath.ToSlash(filepath.Join(".agent-media", "ocr-result.txt"))
	path, err := writeMediaFile(workspace, relativePath, []byte(text+"\n"), 8<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, relativePath, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: path, MediaType: "text/plain", Text: text, Artifact: artifact}, nil
}

type limitedWriter struct {
	writer *bytes.Buffer
	limit  int
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.writer.Len() {
		return 0, errors.New("process output exceeds limit")
	}
	return w.writer.Write(data)
}
