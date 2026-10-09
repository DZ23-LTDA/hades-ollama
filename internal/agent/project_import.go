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
	maxImportedFileBytes  = 256 << 20
	maxImportedFiles      = 10000
	maxImportedIndexBytes = 64 << 20
	// maxIndexedImportFileBytes is the per-file ceiling for indexing an imported
	// text file; larger files are kept in the worktree but skipped by the
	// indexer and reported in the per-file manifest.
	maxIndexedImportFileBytes = 4 << 20
)

// maxImportedArchiveBytes is the aggregate on-disk budget for an imported
// archive. It is a var (not a const) only so tests can lower it to exercise the
// real-bytes aggregate enforcement without materializing a gigabyte on disk.
var maxImportedArchiveBytes int64 = 1 << 30

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
	Project       Project             `json:"project"`
	Source        string              `json:"source"`
	RepositoryURL string              `json:"repository_url,omitempty"`
	Ref           string              `json:"ref,omitempty"`
	WorktreePath  string              `json:"worktree_path"`
	Branch        string              `json:"branch"`
	IndexedFiles  int                 `json:"indexed_files"`
	IndexedMemory int                 `json:"indexed_memories"`
	IgnoredFiles  int                 `json:"ignored_files"`
	Files         []ProjectImportFile `json:"files"`
	ArchiveSHA256 string              `json:"archive_sha256"`
	State         string              `json:"state"`
	Notice        string              `json:"notice"`
}

// ProjectImportFile is the per-file status of an import so the user can see
// which files were indexed and, for the rest, why they were skipped — instead
// of silently getting empty answers about a document that was never indexed.
type ProjectImportFile struct {
	Path      string `json:"path"`
	Indexed   bool   `json:"indexed"`
	Reason    string `json:"reason,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
}

type ProjectImporter struct {
	WorkspaceRoot              string
	DataRoot                   string
	Context                    *ContextStore
	Ingestion                  DocumentIngestor
	HTTPClient                 *http.Client
	GitHubToken                func() string
	GitHubTokenForOrganization func(string) string
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
	if i.GitHubTokenForOrganization != nil {
		token = strings.TrimSpace(i.GitHubTokenForOrganization(strings.TrimSpace(organizationID)))
	} else if i.GitHubToken != nil && strings.TrimSpace(organizationID) == LocalOrganizationID {
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

var (
	errGitHubUnauthorized = errors.New("GitHub rejected the import credentials")
	errGitHubForbidden    = errors.New("GitHub denied access to the repository")
)

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
	manifest := importedFileManifest(session.WorktreeDir)
	files := make([]string, 0, len(manifest))
	ignored := 0
	for _, file := range manifest {
		if file.Indexed {
			files = append(files, filepath.FromSlash(file.Path))
		} else {
			ignored++
		}
	}
	memories, ingestErr := i.Ingestion.Ingest(ctx, DocumentIngestRequest{ProjectID: project.ID, Workspace: session.WorktreeDir, Paths: files, MaxBytes: maxImportedIndexBytes})
	if ingestErr != nil {
		return ProjectImportResult{}, fmt.Errorf("project imported but indexing failed: %w", ingestErr)
	}
	return ProjectImportResult{Project: project, Source: source, RepositoryURL: repositoryURL, Ref: ref, WorktreePath: session.WorktreeDir, Branch: session.BranchName, IndexedFiles: len(files), IndexedMemory: len(memories), IgnoredFiles: ignored, Files: manifest, ArchiveSHA256: digest, State: "IMPORTED_INDEXED", Notice: "Código importado apenas como contexto; nenhum arquivo foi executado. Testes/build exigem sandbox H1 e aprovação HITL."}, nil
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
		// Never import Git's control plane. .git/config can enable filters,
		// includes, alternates or core.worktree outside the extracted tree.
		if strings.Split(filepath.ToSlash(clean), "/")[0] == ".git" {
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
		// ZIP header sizes are attacker-controlled; use them only as a cheap early
		// reject, never as the real budget. The authoritative accounting happens
		// after the copy, against the actual bytes written to disk.
		if entry.UncompressedSize64 > maxImportedFileBytes || total+int64(entry.UncompressedSize64) > maxImportedArchiveBytes {
			return ErrImportArchiveTooLarge
		}
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
		info, statErr := os.Stat(target)
		if statErr != nil || info.Size() > maxImportedFileBytes {
			return ErrImportArchiveTooLarge
		}
		// Enforce the aggregate budget against real bytes written, so an archive
		// that lies about per-entry sizes in its header cannot exhaust the disk.
		total += info.Size()
		if total > maxImportedArchiveBytes {
			return ErrImportArchiveTooLarge
		}
	}
	return nil
}

func initializeImportedRepository(ctx context.Context, root string) error {
	commands := [][]string{{"init", root}, {"-C", root, "config", "user.email", "hades@localhost"}, {"-C", root, "config", "user.name", "Hades Import"}, {"-C", root, "config", "core.hooksPath", ""}, {"-C", root, "config", "core.worktree", root}, {"-C", root, "add", "--all"}, {"-C", root, "commit", "--no-verify", "-m", "Imported project snapshot"}}
	for _, args := range commands {
		command := exec.CommandContext(ctx, "git", args...)
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("initialize imported repository: %s: %w", RedactDLP(strings.TrimSpace(string(output))), err)
		}
	}
	check := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--is-inside-work-tree")
	check.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=", "GIT_TERMINAL_PROMPT=0")
	output, err := check.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		return ErrImportArchiveUnsafe
	}

	// `-C root` is already the directory we created and initialized. The empty
	// show-prefix proves Git considers it the repository root without relying on
	// platform-specific drive-letter or slash formatting.
	prefix := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-prefix")
	prefix.Env = check.Env
	prefixOutput, err := prefix.Output()
	if err != nil || strings.TrimSpace(string(prefixOutput)) != "" {
		return ErrImportArchiveUnsafe
	}
	return nil
}

func importedTextFiles(root string) []string {
	manifest := importedFileManifest(root)
	paths := make([]string, 0, len(manifest))
	for _, file := range manifest {
		if file.Indexed {
			paths = append(paths, filepath.FromSlash(file.Path))
		}
	}
	return paths
}

// importedFileManifest walks the imported worktree and classifies every regular
// file as indexed or skipped (with a human-readable reason). The .git control
// plane is never listed. Paths use forward slashes for a stable JSON contract.
func importedFileManifest(root string) []ProjectImportFile {
	var manifest []ProjectImportFile
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry == nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr == nil {
			file := ProjectImportFile{Path: filepath.ToSlash(relative)}
			info, statErr := entry.Info()
			switch {
			case statErr != nil:
				file.Reason = "Não foi possível ler os metadados do arquivo."
			case !isLikelyTextImportFile(path):
				file.SizeBytes = info.Size()
				file.Reason = "Formato não indexável (sem adaptador de leitura para este tipo)."
			case info.Size() > maxIndexedImportFileBytes:
				file.SizeBytes = info.Size()
				file.Reason = "Arquivo acima de 4 MB; não indexado."
			default:
				file.SizeBytes = info.Size()
				file.Indexed = true
			}
			manifest = append(manifest, file)
			return nil
		}
		// filepath.Rel only fails for a path on another volume. Keep the entry
		// visible in the manifest instead of silently dropping it or aborting
		// the whole import walk.
		manifest = append(manifest, ProjectImportFile{
			Path:   entry.Name(),
			Reason: "Não foi possível obter o caminho relativo do arquivo no projeto.",
		})
		return nil
	})
	return manifest
}

// SupportedProjectDocumentFilename reports whether the document ingestion
// pipeline can index a standalone project attachment. Images are intentionally
// excluded until a vision/OCR adapter is configured.
func SupportedProjectDocumentFilename(name string) bool {
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".docx", ".xlsx", ".txt", ".md", ".csv", ".json", ".xml", ".html", ".htm", ".css", ".sql",
		".js", ".jsx", ".ts", ".tsx", ".py", ".java", ".cpp", ".c", ".cc", ".h", ".cs",
		".php", ".rb", ".go", ".rs", ".swift", ".kt", ".scala", ".sh", ".bat", ".yaml",
		".yml", ".toml", ".ini", ".cfg", ".conf", ".log":
		return true
	default:
		return false
	}
}

func isLikelyTextImportFile(path string) bool {
	if filepath.Ext(path) == "" {
		return true
	}
	return SupportedProjectDocumentFilename(filepath.Base(path))
}

func (r ProjectImportResult) MarshalJSON() ([]byte, error) {
	type alias ProjectImportResult
	return json.Marshal(alias(r))
}
