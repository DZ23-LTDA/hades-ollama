package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxImportedArchiveBytes = 1 << 30
	maxImportedFileBytes    = 256 << 20
	maxImportedFiles        = 10000
	maxImportedIndexBytes   = 64 << 20
)

var (
	ErrGitHubURLInvalid        = errors.New("GitHub repository URL must use https://github.com/owner/repository")
	ErrGitHubAuthRequired      = errors.New("private GitHub import requires server-side GitHub authentication")
	ErrImportArchiveTooLarge   = errors.New("project archive exceeds the import limit")
	ErrImportArchiveUnsafe     = errors.New("project archive contains an unsafe path or entry")
	ErrImportCodeNotExecutable = errors.New("imported code is never executed during import")
)

type ProjectImportRequest struct {
	URL       string `json:"url,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Name      string `json:"name,omitempty"`
	UploadID  string `json:"upload_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type ProjectImportResult struct {
	Project       Project `json:"project"`
	Source        string  `json:"source"`
	RepositoryURL string  `json:"repository_url,omitempty"`
	Ref           string  `json:"ref,omitempty"`
	WorktreePath  string  `json:"worktree_path"`
	Branch        string  `json:"branch"`
	IndexedFiles  int     `json:"indexed_files"`
	IndexedMemory int     `json:"indexed_memories"`
	ArchiveSHA256 string  `json:"archive_sha256"`
	State         string  `json:"state"`
	Notice        string  `json:"notice"`
}

type ProjectImporter struct {
	WorkspaceRoot string
	DataRoot      string
	Context       *ContextStore
	Ingestion     DocumentIngestor
	HTTPClient    *http.Client
	GitHubToken   func() string
}

func NewProjectImporter(workspaceRoot, dataRoot string, store *ContextStore, ingestion DocumentIngestor) *ProjectImporter {
	return &ProjectImporter{WorkspaceRoot: workspaceRoot, DataRoot: dataRoot, Context: store, Ingestion: ingestion}
}

func (i *ProjectImporter) ImportGitHub(ctx context.Context, organizationID string, request ProjectImportRequest) (ProjectImportResult, error) {
	owner, repo, err := parseGitHubRepositoryURL(request.URL)
	if err != nil {
		return ProjectImportResult{}, err
	}
	token := ""
	if i.GitHubToken != nil {
		token = strings.TrimSpace(i.GitHubToken())
	}
	client := i.HTTPClient
	if client == nil {
		client = NewSafeEgressHTTPClient(EgressOptions{Callsite: "agent.project_import.github", Timeout: 2 * time.Minute, MaxBodyBytes: maxImportedArchiveBytes})
		client.CheckRedirect = githubImportRedirectPolicy
	}
	archiveURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, url.PathEscape(strings.TrimSpace(request.Ref)))
	if strings.TrimSpace(request.Ref) == "" {
		archiveURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/HEAD", owner, repo)
	}
	archive, digest, err := i.downloadArchive(ctx, client, archiveURL, token)
	if err != nil {
		if token == "" && (errors.Is(err, errGitHubUnauthorized) || errors.Is(err, errGitHubForbidden)) {
			return ProjectImportResult{}, ErrGitHubAuthRequired
		}
		return ProjectImportResult{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = repo
	}
	result, err := i.importArchive(ctx, organizationID, "", name, bytes.NewReader(archive), int64(len(archive)), digest, "github", fmt.Sprintf("https://github.com/%s/%s", owner, repo), request.Ref)
	if err != nil {
		return ProjectImportResult{}, err
	}
	return result, nil
}

func githubImportRedirectPolicy(request *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	if request.URL.Scheme != "https" {
		return ErrEgressRedirectDisallowed
	}
	allowed := map[string]bool{"api.github.com": true, "github.com": true, "codeload.github.com": true}
	from := strings.ToLower(via[0].URL.Hostname())
	to := strings.ToLower(request.URL.Hostname())
	if !allowed[from] || !allowed[to] {
		DefaultEgressAuditStore.Record(EgressDecision{Timestamp: time.Now(), Callsite: "agent.project_import.github", Destination: request.URL.String(), Host: to, Allowed: false, Reason: "redirect target is not an approved GitHub archive host"})
		return ErrEgressRedirectDisallowed
	}
	StripSensitiveEgressHeaders(request)
	if _, err := ResolveAllPublicIPs(request.Context(), to); err != nil {
		return fmt.Errorf("GitHub redirect destination failed egress validation: %w", err)
	}
	return nil
}

func (i *ProjectImporter) ImportUpload(ctx context.Context, organizationID string, upload UploadSession, name string) (ProjectImportResult, error) {
	if upload.State != UploadCompleted || strings.TrimSpace(upload.FinalPath) == "" {
		return ProjectImportResult{}, ErrUploadIncomplete
	}
	if upload.OrganizationID != strings.TrimSpace(organizationID) {
		return ProjectImportResult{}, ErrUploadForbidden
	}
	file, err := os.Open(upload.FinalPath)
	if err != nil {
		return ProjectImportResult{}, err
	}
	defer file.Close()
	if upload.TotalSize > maxImportedArchiveBytes {
		return ProjectImportResult{}, ErrImportArchiveTooLarge
	}
	info, err := file.Stat()
	if err != nil {
		return ProjectImportResult{}, err
	}
	if info.Size() > maxImportedArchiveBytes {
		return ProjectImportResult{}, ErrImportArchiveTooLarge
	}
	digest, err := sha256File(upload.FinalPath)
	if err != nil {
		return ProjectImportResult{}, err
	}
	return i.importArchive(ctx, organizationID, upload.ProjectID, name, file, info.Size(), digest, "zip", "", "")
}

var errGitHubUnauthorized = errors.New("GitHub rejected the import credentials")
var errGitHubForbidden = errors.New("GitHub denied access to the repository")

func (i *ProjectImporter) downloadArchive(ctx context.Context, client *http.Client, rawURL, token string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ollama-full-project-import")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return nil, "", errGitHubUnauthorized
	}
	if response.StatusCode == http.StatusForbidden {
		return nil, "", errGitHubForbidden
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("GitHub archive request failed with status %d", response.StatusCode)
	}
	data, err := ReadBoundedBody(response.Body, maxImportedArchiveBytes)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(data)
	return data, hex.EncodeToString(digest[:]), nil
}

func (i *ProjectImporter) importArchive(ctx context.Context, organizationID, existingProjectID, name string, archive io.ReaderAt, archiveSize int64, digest, source, repositoryURL, ref string) (ProjectImportResult, error) {
	if i.Context == nil || strings.TrimSpace(i.WorkspaceRoot) == "" || strings.TrimSpace(i.DataRoot) == "" {
		return ProjectImportResult{}, errors.New("project importer is not configured")
	}
	importID := "imp_" + uuid.NewString()
	// Imported source and worktrees must remain under the runtime workspace. The
	// data root may intentionally be outside that boundary for persistence, but
	// ContextStore rejects project roots outside WorkspaceRoot.
	origin := filepath.Join(i.WorkspaceRoot, ".agent-imports", importID, "origin")
	if err := os.MkdirAll(origin, 0o700); err != nil {
		return ProjectImportResult{}, err
	}
	if archiveSize <= 0 || archiveSize > maxImportedArchiveBytes {
		return ProjectImportResult{}, ErrImportArchiveTooLarge
	}
	if err := extractProjectArchive(archive, archiveSize, origin); err != nil {
		_ = os.RemoveAll(filepath.Dir(origin))
		return ProjectImportResult{}, err
	}
	if err := initializeImportedRepository(ctx, origin); err != nil {
		_ = os.RemoveAll(filepath.Dir(origin))
		return ProjectImportResult{}, err
	}
	session, err := CreateGitWorktree(ctx, origin, i.WorkspaceRoot, importID, "agent/import_"+importID)
	if err != nil {
		_ = os.RemoveAll(filepath.Dir(origin))
		return ProjectImportResult{}, err
	}
	var project Project
	if strings.TrimSpace(existingProjectID) != "" {
		project, err = i.Context.GetProject(existingProjectID)
		if err == nil && !organizationOwnsRecord(project.OrganizationID, organizationID) {
			err = ErrPluginOrganizationScope
		}
		if err == nil {
			project, err = i.Context.UpdateProject(project.ID, name, session.WorktreeDir)
		}
	} else {
		project, err = i.Context.CreateProject(name, session.WorktreeDir, organizationID)
	}
	if err != nil {
		_ = RemoveGitWorktree(ctx, session)
		_ = os.RemoveAll(filepath.Dir(origin))
		return ProjectImportResult{}, err
	}
	files := importedTextFiles(session.WorktreeDir)
	memories, ingestErr := i.Ingestion.Ingest(ctx, DocumentIngestRequest{ProjectID: project.ID, Workspace: session.WorktreeDir, Paths: files, MaxBytes: maxImportedIndexBytes})
	if ingestErr != nil {
		return ProjectImportResult{}, fmt.Errorf("project imported but indexing failed: %w", ingestErr)
	}
	return ProjectImportResult{Project: project, Source: source, RepositoryURL: repositoryURL, Ref: ref, WorktreePath: session.WorktreeDir, Branch: session.BranchName, IndexedFiles: len(files), IndexedMemory: len(memories), ArchiveSHA256: digest, State: "IMPORTED_INDEXED", Notice: "Código importado apenas como contexto; nenhum arquivo foi executado. Testes/build exigem sandbox H1 e aprovação HITL."}, nil
}

func parseGitHubRepositoryURL(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", ErrGitHubURLInvalid
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || !validGitHubSegment(parts[0]) || !validGitHubSegment(strings.TrimSuffix(parts[1], ".git")) {
		return "", "", ErrGitHubURLInvalid
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}

var githubSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func validGitHubSegment(value string) bool {
	return value != "" && len(value) <= 100 && githubSegmentPattern.MatchString(value)
}

func extractProjectArchive(reader io.ReaderAt, size int64, destination string) error {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return fmt.Errorf("open project ZIP: %w", err)
	}
	if len(archive.File) > maxImportedFiles {
		return ErrImportArchiveTooLarge
	}
	var total int64
	for _, entry := range archive.File {
		if entry.Name == "" || strings.ContainsRune(entry.Name, '\x00') || filepath.IsAbs(entry.Name) {
			return ErrImportArchiveUnsafe
		}
		clean := filepath.Clean(filepath.FromSlash(entry.Name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return ErrImportArchiveUnsafe
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return ErrImportArchiveUnsafe
		}
		parts := strings.Split(filepath.ToSlash(clean), "/")
		if len(parts) > 0 && strings.HasPrefix(parts[0], "__") {
			continue
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(filepath.Join(destination, clean), 0o700); err != nil {
				return err
			}
			continue
		}
		if entry.UncompressedSize64 > maxImportedFileBytes || total+int64(entry.UncompressedSize64) > maxImportedArchiveBytes {
			return ErrImportArchiveTooLarge
		}
		total += int64(entry.UncompressedSize64)
		target := filepath.Join(destination, clean)
		if !isWithin(destination, target) {
			return ErrImportArchiveUnsafe
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, maxImportedFileBytes+1))
		closeErr := errors.Join(input.Close(), output.Close())
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if info, statErr := os.Stat(target); statErr != nil || info.Size() > maxImportedFileBytes {
			return ErrImportArchiveTooLarge
		}
	}
	return nil
}

func initializeImportedRepository(ctx context.Context, root string) error {
	commands := [][]string{{"init", root}, {"-C", root, "config", "user.email", "ollama-full@localhost"}, {"-C", root, "config", "user.name", "Ollama Full Import"}, {"-C", root, "config", "core.hooksPath", "/dev/null"}, {"-C", root, "add", "--all"}, {"-C", root, "commit", "--no-verify", "-m", "Imported project snapshot"}}
	for _, args := range commands {
		command := exec.CommandContext(ctx, "git", args...)
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("initialize imported repository: %s: %w", RedactDLP(strings.TrimSpace(string(output))), err)
		}
	}
	return nil
}

func importedTextFiles(root string) []string {
	var paths []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if info, statErr := entry.Info(); statErr == nil && info.Size() <= 4<<20 && isLikelyTextImportFile(path) {
			relative, relErr := filepath.Rel(root, path)
			if relErr == nil {
				paths = append(paths, relative)
			}
		}
		return nil
	})
	return paths
}
func isLikelyTextImportFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return true
	}
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".json", ".md", ".txt", ".yaml", ".yml", ".toml", ".css", ".html", ".htm", ".sql", ".py", ".rs", ".java", ".sh", ".xml", ".csv":
		return true
	}
	return false
}

func (r ProjectImportResult) MarshalJSON() ([]byte, error) {
	type alias ProjectImportResult
	return json.Marshal(alias(r))
}
