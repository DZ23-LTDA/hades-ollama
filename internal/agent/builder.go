package agent

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type BuilderKind string

const (
	BuilderWebsite   BuilderKind = "website"
	BuilderApp       BuilderKind = "app"
	BuilderGame      BuilderKind = "game"
	BuilderSlides    BuilderKind = "slides"
	BuilderDashboard BuilderKind = "dashboard"
	// BuilderFullstack gera uma aplicacao de gestao REAL: login, CRUD e persistencia em disco.
	BuilderFullstack BuilderKind = "fullstack"
)

type BuilderProject struct {
	ID             string              `json:"id"`
	OrganizationID string              `json:"organization_id"`
	Name           string              `json:"name"`
	Kind           BuilderKind         `json:"kind"`
	Entry          string              `json:"entry"`
	Version        int                 `json:"version"`
	Status         string              `json:"status"`
	Root           string              `json:"root"`
	PreviewPath    string              `json:"preview_path,omitempty"`
	PublishedPath  string              `json:"published_path,omitempty"`
	ExportChecksum string              `json:"export_checksum,omitempty"`
	ExportPath     string              `json:"export_path,omitempty"`
	ExportVersion  int                 `json:"export_version,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Components     []VisualComponent   `json:"components,omitempty"`
	UndoStack      [][]VisualComponent `json:"undo_stack,omitempty"`
	RedoStack      [][]VisualComponent `json:"redo_stack,omitempty"`
}

var (
	ErrBuilderVersionConflict = errors.New("builder version conflict")
	ErrBuilderExportExpired   = errors.New("builder export is expired; export the current version again")
)

type BuilderVersionConflictError struct {
	ProjectID string
	Expected  int
	Current   int
}

func (e *BuilderVersionConflictError) Error() string {
	return fmt.Sprintf("builder version conflict for %s: expected %d, current %d", e.ProjectID, e.Expected, e.Current)
}

func (e *BuilderVersionConflictError) Unwrap() error { return ErrBuilderVersionConflict }

type VisualComponent struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Props    map[string]string `json:"props,omitempty"`
	Style    map[string]string `json:"style,omitempty"`
	Bindings map[string]string `json:"bindings,omitempty"`
	Events   map[string]string `json:"events,omitempty"`
	Children []VisualComponent `json:"children,omitempty"`
	X        int               `json:"x,omitempty"`
	Y        int               `json:"y,omitempty"`
	Width    int               `json:"width,omitempty"`
	Height   int               `json:"height,omitempty"`
}

type BuilderSpec struct {
	Name           string            `json:"name"`
	OrganizationID string            `json:"organization_id,omitempty"`
	Kind           BuilderKind       `json:"kind"`
	Entry          string            `json:"entry"`
	Files          map[string]string `json:"files"`
	Components     []VisualComponent `json:"components,omitempty"`
}

type BuilderService struct {
	mu       sync.Mutex
	root     string
	projects map[string]BuilderProject
}

func NewBuilderService(root string) (*BuilderService, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("builder root is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	service := &BuilderService{root: root, projects: map[string]BuilderProject{}}
	data, err := os.ReadFile(filepath.Join(root, "projects.json"))
	if err == nil {
		if err := json.Unmarshal(data, &service.projects); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return service, nil
}

func (b *BuilderService) Root() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.root
}

func (b *BuilderService) Create(ctx context.Context, spec BuilderSpec) (BuilderProject, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, err
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" || len(name) > 160 {
		return BuilderProject{}, errors.New("builder name is required and must be <= 160 characters")
	}
	if !validBuilderKind(spec.Kind) {
		return BuilderProject{}, errors.New("unsupported builder kind")
	}
	if len(spec.Files) == 0 && len(spec.Components) == 0 {
		spec.Files = templateFiles(spec.Kind, name)
	}
	if len(spec.Components) > 200 {
		return BuilderProject{}, errors.New("too many visual components")
	}
	if err := validateVisualComponents(spec.Components, 0); err != nil {
		return BuilderProject{}, err
	}
	if len(spec.Components) > 0 && len(spec.Files) == 0 {
		spec.Files = map[string]string{"index.html": renderVisualHTML(name, spec.Components), "visual.json": mustJSON(spec.Components)}
	}
	if len(spec.Files) > 200 {
		return BuilderProject{}, errors.New("too many builder files")
	}
	entry := strings.TrimSpace(spec.Entry)
	if entry == "" {
		entry = "index.html"
	}
	for path, contents := range spec.Files {
		if err := validateBuilderFile(path, contents); err != nil {
			return BuilderProject{}, err
		}
	}
	if _, exists := spec.Files[entry]; !exists {
		return BuilderProject{}, fmt.Errorf("builder entry %q does not exist", entry)
	}
	organizationID := strings.TrimSpace(spec.OrganizationID)
	if organizationID == "" {
		organizationID = "local"
	}
	id := "bld_" + uuid.NewString()
	now := time.Now().UTC()
	projectRoot := filepath.Join(b.root, id)
	for relative, contents := range spec.Files {
		target := filepath.Join(projectRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return BuilderProject{}, err
		}
		if err := os.WriteFile(target, []byte(contents), 0o600); err != nil {
			return BuilderProject{}, err
		}
	}
	project := BuilderProject{ID: id, OrganizationID: organizationID, Name: name, Kind: spec.Kind, Entry: filepath.ToSlash(entry), Version: 1, Status: "draft", Root: projectRoot, Components: spec.Components, CreatedAt: now, UpdatedAt: now}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.projects[id] = project
	if err := b.persistLocked(); err != nil {
		return BuilderProject{}, err
	}
	return project, nil
}

func (b *BuilderService) ApplyVisualComponents(ctx context.Context, id string, components []VisualComponent) (BuilderProject, error) {
	return b.applyVisualComponents(ctx, id, components, nil)
}

func (b *BuilderService) ApplyVisualComponentsCAS(ctx context.Context, id string, expectedVersion int, components []VisualComponent) (BuilderProject, error) {
	return b.applyVisualComponents(ctx, id, components, &expectedVersion)
}

func (b *BuilderService) applyVisualComponents(ctx context.Context, id string, components []VisualComponent, expectedVersion *int) (BuilderProject, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, err
	}
	if len(components) > 200 {
		return BuilderProject{}, errors.New("too many visual components")
	}
	if err := validateVisualComponents(components, 0); err != nil {
		return BuilderProject{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id = strings.TrimSpace(id)
	project, ok := b.projects[id]
	if !ok {
		return BuilderProject{}, os.ErrNotExist
	}
	if expectedVersion != nil && project.Version != *expectedVersion {
		return BuilderProject{}, &BuilderVersionConflictError{ProjectID: id, Expected: *expectedVersion, Current: project.Version}
	}
	indexPath := filepath.Join(project.Root, filepath.FromSlash(project.Entry))
	if err := os.WriteFile(indexPath, []byte(renderVisualHTML(project.Name, components)), 0o600); err != nil {
		return BuilderProject{}, err
	}
	if err := os.WriteFile(filepath.Join(project.Root, "visual.json"), []byte(mustJSON(components)), 0o600); err != nil {
		return BuilderProject{}, err
	}
	project.UndoStack = append(project.UndoStack, cloneComponents(project.Components))
	if len(project.UndoStack) > 50 {
		project.UndoStack = project.UndoStack[len(project.UndoStack)-50:]
	}
	project.RedoStack = nil
	project.Components = cloneComponents(components)
	project.Version++
	project.Status = "draft"
	project.UpdatedAt = time.Now().UTC()
	b.invalidateExportLocked(&project)
	b.projects[id] = project
	return project, b.persistLocked()
}

func (b *BuilderService) invalidateExportLocked(project *BuilderProject) {
	if project.ExportPath != "" {
		_ = os.Remove(project.ExportPath)
	}
	_ = os.Remove(filepath.Join(b.root, project.ID+".zip"))
	project.ExportChecksum = ""
	project.ExportPath = ""
	project.ExportVersion = 0
}

func (b *BuilderService) Undo(ctx context.Context, id string) (BuilderProject, error) {
	return b.undo(ctx, id, nil)
}

func (b *BuilderService) UndoCAS(ctx context.Context, id string, expectedVersion int) (BuilderProject, error) {
	return b.undo(ctx, id, &expectedVersion)
}

func (b *BuilderService) undo(ctx context.Context, id string, expectedVersion *int) (BuilderProject, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id = strings.TrimSpace(id)
	project, ok := b.projects[id]
	if !ok {
		return BuilderProject{}, os.ErrNotExist
	}
	if expectedVersion != nil && project.Version != *expectedVersion {
		return BuilderProject{}, &BuilderVersionConflictError{ProjectID: id, Expected: *expectedVersion, Current: project.Version}
	}
	if len(project.UndoStack) == 0 {
		return project, errors.New("builder has no undo history")
	}
	previous := cloneComponents(project.UndoStack[len(project.UndoStack)-1])
	project.UndoStack = project.UndoStack[:len(project.UndoStack)-1]
	project.RedoStack = append(project.RedoStack, cloneComponents(project.Components))
	project.Components = previous
	if err := b.writeVisualProject(project); err != nil {
		return BuilderProject{}, err
	}
	project.Version++
	project.UpdatedAt = time.Now().UTC()
	b.invalidateExportLocked(&project)
	b.projects[id] = project
	return project, b.persistLocked()
}

func (b *BuilderService) Redo(ctx context.Context, id string) (BuilderProject, error) {
	return b.redo(ctx, id, nil)
}

func (b *BuilderService) RedoCAS(ctx context.Context, id string, expectedVersion int) (BuilderProject, error) {
	return b.redo(ctx, id, &expectedVersion)
}

func (b *BuilderService) redo(ctx context.Context, id string, expectedVersion *int) (BuilderProject, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id = strings.TrimSpace(id)
	project, ok := b.projects[id]
	if !ok {
		return BuilderProject{}, os.ErrNotExist
	}
	if expectedVersion != nil && project.Version != *expectedVersion {
		return BuilderProject{}, &BuilderVersionConflictError{ProjectID: id, Expected: *expectedVersion, Current: project.Version}
	}
	if len(project.RedoStack) == 0 {
		return project, errors.New("builder has no redo history")
	}
	next := cloneComponents(project.RedoStack[len(project.RedoStack)-1])
	project.RedoStack = project.RedoStack[:len(project.RedoStack)-1]
	project.UndoStack = append(project.UndoStack, cloneComponents(project.Components))
	project.Components = next
	if err := b.writeVisualProject(project); err != nil {
		return BuilderProject{}, err
	}
	project.Version++
	project.UpdatedAt = time.Now().UTC()
	b.invalidateExportLocked(&project)
	b.projects[id] = project
	return project, b.persistLocked()
}

func (b *BuilderService) writeVisualProject(project BuilderProject) error {
	indexPath := filepath.Join(project.Root, filepath.FromSlash(project.Entry))
	if err := os.WriteFile(indexPath, []byte(renderVisualHTML(project.Name, project.Components)), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(project.Root, "visual.json"), []byte(mustJSON(project.Components)), 0o600)
}

func (b *BuilderService) Get(id string) (BuilderProject, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	project, ok := b.projects[strings.TrimSpace(id)]
	if !ok {
		return BuilderProject{}, os.ErrNotExist
	}
	return project, nil
}

var ErrBuilderForbidden = errors.New("builder project is outside the active organization")

func normalizedBuilderOrganization(id string) string {
	if strings.TrimSpace(id) == "" {
		return "local"
	}
	return strings.TrimSpace(id)
}

func (b *BuilderService) GetForOrganization(id, organizationID string) (BuilderProject, error) {
	project, err := b.Get(id)
	if err != nil {
		// NÃO INFERÊNCIA (E-192, fechamento de B-16): o scanner calibrado (E-191) apontou este método como a mesma
		// forma do B-15 — "não existe" devolvia o erro cru da busca (404) e "é de outro tenant" devolvia
		// ErrBuilderForbidden (403), e a diferença revela a existência do ID. As duas recusas passam a ser a MESMA; o
		// motivo real fica no log de quem opera.
		return BuilderProject{}, ErrBuilderForbidden
	}
	if normalizedBuilderOrganization(project.OrganizationID) != normalizedBuilderOrganization(organizationID) {
		return BuilderProject{}, ErrBuilderForbidden
	}
	return project, nil
}

func (b *BuilderService) Preview(ctx context.Context, id string) (BuilderProject, ArtifactManifest, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, ArtifactManifest{}, err
	}
	b.mu.Lock()
	project, ok := b.projects[strings.TrimSpace(id)]
	b.mu.Unlock()
	if !ok {
		return BuilderProject{}, ArtifactManifest{}, os.ErrNotExist
	}
	entryPath := filepath.Join(project.Root, filepath.FromSlash(project.Entry))
	if err := rejectSymlinkComponents(project.Root, entryPath); err != nil {
		return BuilderProject{}, ArtifactManifest{}, err
	}
	if _, err := os.Stat(entryPath); err != nil {
		return BuilderProject{}, ArtifactManifest{}, err
	}
	project.PreviewPath = entryPath
	project.Status = "preview"
	project.UpdatedAt = time.Now().UTC()
	b.mu.Lock()
	b.projects[id] = project
	err := b.persistLocked()
	b.mu.Unlock()
	if err != nil {
		return BuilderProject{}, ArtifactManifest{}, err
	}
	artifact, err := BuildArtifactManifest(project.Root, "", "", filepath.Base(entryPath), project.Entry)
	if err != nil {
		return BuilderProject{}, ArtifactManifest{}, err
	}
	return project, artifact, nil
}

func (b *BuilderService) Export(ctx context.Context, id string) (BuilderProject, string, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, "", err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id = strings.TrimSpace(id)
	project, ok := b.projects[id]
	if !ok {
		return BuilderProject{}, "", os.ErrNotExist
	}
	path := filepath.Join(b.root, id+".zip")
	tmpPath := path + ".tmp-" + uuid.NewString()
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return BuilderProject{}, "", err
	}
	defer os.Remove(tmpPath)
	archive := zip.NewWriter(file)
	err = filepath.Walk(project.Root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("builder export rejects non-regular file: %s", path)
		}
		relative, err := filepath.Rel(project.Root, path)
		if err != nil {
			return err
		}
		writer, err := archive.Create(filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		closeErr := input.Close()
		return errors.Join(copyErr, closeErr)
	})
	if closeErr := archive.Close(); err == nil {
		err = closeErr
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return BuilderProject{}, "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return BuilderProject{}, "", err
	}
	data, readErr := os.ReadFile(path)
	if readErr == nil {
		hasher := sha256.New()
		hasher.Write(data)
		project.ExportChecksum = hex.EncodeToString(hasher.Sum(nil))
		project.ExportPath = path
		project.ExportVersion = project.Version
		b.projects[id] = project
		if err := b.persistLocked(); err != nil {
			return BuilderProject{}, "", err
		}
	}
	return project, path, nil
}

func (b *BuilderService) PublishLocal(ctx context.Context, id string) (BuilderProject, string, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, "", err
	}
	project, err := b.Get(id)
	if err != nil {
		return BuilderProject{}, "", err
	}
	published := filepath.Join(b.root, "published", id, fmt.Sprintf("v%d", project.Version))
	if err := os.MkdirAll(published, 0o700); err != nil {
		return BuilderProject{}, "", err
	}
	if err := copyFiles(project.Root, published); err != nil {
		return BuilderProject{}, "", err
	}
	project.PublishedPath = published
	project.Status = "published"
	project.UpdatedAt = time.Now().UTC()
	b.mu.Lock()
	b.projects[id] = project
	err = b.persistLocked()
	b.mu.Unlock()
	return project, published, err
}

// ErrBuilderVersionNotPublished indica rollback para uma versao que nunca foi
// publicada (nao existe published/id/vN).
var ErrBuilderVersionNotPublished = errors.New("builder version was not published")

// RollbackPublish re-aponta a publicacao ativa para uma versao ja publicada
// (published/id/vN), sem reconstruir os arquivos. Org-scoped e idempotente:
// rollback para a versao atual apenas reafirma o estado publicado.
func (b *BuilderService) RollbackPublish(ctx context.Context, id, organizationID string, version int) (BuilderProject, string, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, "", err
	}
	if version < 1 {
		return BuilderProject{}, "", errors.New("builder rollback version must be >= 1")
	}
	project, err := b.GetForOrganization(id, organizationID)
	if err != nil {
		return BuilderProject{}, "", err
	}
	target := filepath.Join(b.root, "published", id, fmt.Sprintf("v%d", version))
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return BuilderProject{}, "", ErrBuilderVersionNotPublished
	}
	project.PublishedPath = target
	project.Status = "published"
	project.UpdatedAt = time.Now().UTC()
	b.mu.Lock()
	b.projects[id] = project
	err = b.persistLocked()
	b.mu.Unlock()
	return project, target, err
}

func (b *BuilderService) List() []BuilderProject {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]BuilderProject, 0, len(b.projects))
	for _, project := range b.projects {
		result = append(result, project)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.Before(result[j].UpdatedAt) })
	return result
}

func (b *BuilderService) ListForOrganization(organizationID string) []BuilderProject {
	organizationID = normalizedBuilderOrganization(organizationID)
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]BuilderProject, 0, len(b.projects))
	for _, project := range b.projects {
		if normalizedBuilderOrganization(project.OrganizationID) == organizationID {
			result = append(result, project)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.Before(result[j].UpdatedAt) })
	return result
}

func (b *BuilderService) persistLocked() error {
	return writeJSONAtomic(filepath.Join(b.root, "projects.json"), b.projects)
}

func validBuilderKind(kind BuilderKind) bool {
	switch kind {
	case BuilderWebsite, BuilderApp, BuilderGame, BuilderSlides, BuilderDashboard, BuilderFullstack:
		return true
	default:
		return false
	}
}

func validateBuilderFile(path, contents string) error {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, "../") {
		return errors.New("builder file path escapes project")
	}
	if len(contents) > 2<<20 {
		return errors.New("builder file exceeds 2 MiB")
	}
	return nil
}

func templateFiles(kind BuilderKind, name string) map[string]string {
	title := htmlEscape(name)
	switch kind {
	case BuilderFullstack:
		return map[string]string{"main.go": appFullstackMain, "index.html": appFullstackIndex, "README.md": appFullstackReadme}
	case BuilderGame:
		return map[string]string{"index.html": "<!doctype html><html><head><meta charset=\"utf-8\"><link rel=\"icon\" href=\"data:,\"><title>" + title + "</title></head><body><main id=\"game\"></main><script type=\"module\" src=\"game.ts\"></script></body></html>", "game.ts": "// Babylon.js game entrypoint. Add GameCanvas integration here.\nconst root = document.querySelector('#game'); if (root) root.textContent = 'Game preview ready';"}
	case BuilderSlides:
		return map[string]string{"index.html": "<!doctype html><html><head><meta charset=\"utf-8\"><link rel=\"icon\" href=\"data:,\"><title>" + title + " slides</title></head><body><section data-slide=\"1\"><h1>" + title + "</h1><p>Slide preview</p></section></body></html>"}
	case BuilderDashboard:
		return map[string]string{"index.html": "<!doctype html><html><head><meta charset=\"utf-8\"><link rel=\"icon\" href=\"data:,\"><title>" + title + " dashboard</title></head><body><main><h1>" + title + "</h1><section aria-label=\"Dashboard\"><article data-metric=\"one\">Metric ready</article></section></main></body></html>"}
	default:
		return map[string]string{"index.html": "<!doctype html><html><head><meta charset=\"utf-8\"><link rel=\"icon\" href=\"data:,\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" + title + "</title></head><body><main><h1>" + title + "</h1><p>Builder preview ready.</p></main></body></html>"}
	}
}

func mustJSON(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data)
}

func renderVisualHTML(name string, components []VisualComponent) string {
	data := mustJSON(components)
	return "<!doctype html><html><head><meta charset=\"utf-8\"><link rel=\"icon\" href=\"data:,\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>" + htmlEscape(name) + "</title><style>body{font-family:system-ui;margin:0;padding:24px} [data-dz23-component]{border:1px dashed #bbb;padding:12px;margin:8px;border-radius:8px}</style></head><body><main id=\"dz23-canvas\"></main><script type=\"application/json\" id=\"dz23-visual\">" + scriptEscape(data) + "</script><script>const tree=JSON.parse(document.querySelector('#dz23-visual').textContent); const render=(nodes,parent)=>nodes.forEach(n=>{const el=document.createElement('section');el.dataset.dz23Component=n.type;el.textContent=(n.props&&n.props.text)||n.type;Object.assign(el.style,{position:n.x||n.y?'absolute':'static',left:(n.x||0)+'px',top:(n.y||0)+'px',width:n.width?(n.width+'px'):'auto',height:n.height?(n.height+'px'):'auto'});parent.appendChild(el);render(n.children||[],el)});render(tree,document.querySelector('#dz23-canvas'));</script></body></html>"
}

func validateVisualComponents(components []VisualComponent, depth int) error {
	if depth > 12 {
		return errors.New("visual component tree is too deep")
	}
	for _, component := range components {
		if strings.TrimSpace(component.ID) == "" || strings.TrimSpace(component.Type) == "" {
			return errors.New("visual components require id and type")
		}
		if len(component.Props)+len(component.Style)+len(component.Bindings)+len(component.Events) > 100 {
			return fmt.Errorf("visual component %q has too many properties", component.ID)
		}
		if err := validateVisualComponents(component.Children, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func cloneComponents(input []VisualComponent) []VisualComponent {
	if input == nil {
		return nil
	}
	output := make([]VisualComponent, len(input))
	for index, component := range input {
		output[index] = component
		output[index].Props = cloneStringMap(component.Props)
		output[index].Style = cloneStringMap(component.Style)
		output[index].Bindings = cloneStringMap(component.Bindings)
		output[index].Events = cloneStringMap(component.Events)
		output[index].Children = cloneComponents(component.Children)
	}
	return output
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func htmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;").Replace(value)
}

func scriptEscape(value string) string {
	return strings.NewReplacer("&", "\\u0026", "<", "\\u003c", ">", "\\u003e").Replace(value)
}

func copyFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("builder export refuses symlink")
		}
		target := filepath.Join(dst, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

// Conteúdo do app full-stack gerado, em string INTERPRETADA: o código usa crases nas tags json.
const appFullstackMain = "package main\n\nimport (\n\t\"crypto/rand\"\n\t\"crypto/subtle\"\n\t\"encoding/hex\"\n\t\"encoding/json\"\n\t\"errors\"\n\t\"fmt\"\n\t\"net\"\n\t\"net/http\"\n\t\"os\"\n\t\"path/filepath\"\n\t\"strings\"\n\t\"sync\"\n\t\"time\"\n)\n\n// Aplicacao de gestao gerada pelo Hades: login, CRUD de itens e persistencia em disco.\n// Sem credencial padrao: sem APP_ADMIN_PASSWORD a aplicacao NAO sobe.\ntype item struct {\n\tID        string    `json:\"id\"`\n\tName      string    `json:\"name\"`\n\tQuantity  int       `json:\"quantity\"`\n\tCreatedAt time.Time `json:\"created_at\"`\n\tUpdatedAt time.Time `json:\"updated_at\"`\n}\n\ntype store struct {\n\tmu    sync.Mutex\n\tpath  string\n\tItems []item `json:\"items\"`\n}\n\nfunc openStore(path string) (*store, error) {\n\ts := &store{path: path}\n\tdados, err := os.ReadFile(path)\n\tif errors.Is(err, os.ErrNotExist) {\n\t\treturn s, nil\n\t}\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := json.Unmarshal(dados, s); err != nil {\n\t\treturn nil, err\n\t}\n\treturn s, nil\n}\n\n// save grava de forma ATOMICA (arquivo temporario + rename): uma queda no meio da escrita nao deixa o banco pela metade.\nfunc (s *store) save() error {\n\tdados, err := json.MarshalIndent(s, \"\", \"  \")\n\tif err != nil {\n\t\treturn err\n\t}\n\tif err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {\n\t\treturn err\n\t}\n\ttmp := s.path + \".tmp\"\n\tif err := os.WriteFile(tmp, dados, 0o600); err != nil {\n\t\treturn err\n\t}\n\treturn os.Rename(tmp, s.path)\n}\n\nfunc novoID() (string, error) {\n\tbruto := make([]byte, 8)\n\tif _, err := rand.Read(bruto); err != nil {\n\t\treturn \"\", err\n\t}\n\treturn hex.EncodeToString(bruto), nil\n}\n\nfunc (s *store) create(name string, quantity int) (item, error) {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tid, err := novoID()\n\tif err != nil {\n\t\treturn item{}, err\n\t}\n\tagora := time.Now().UTC()\n\tnovo := item{ID: id, Name: name, Quantity: quantity, CreatedAt: agora, UpdatedAt: agora}\n\ts.Items = append(s.Items, novo)\n\treturn novo, s.save()\n}\n\nfunc (s *store) list() []item {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tcopia := make([]item, len(s.Items))\n\tcopy(copia, s.Items)\n\treturn copia\n}\n\nfunc (s *store) update(id, name string, quantity int) (item, error) {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tfor i := range s.Items {\n\t\tif s.Items[i].ID != id {\n\t\t\tcontinue\n\t\t}\n\t\ts.Items[i].Name = name\n\t\ts.Items[i].Quantity = quantity\n\t\ts.Items[i].UpdatedAt = time.Now().UTC()\n\t\treturn s.Items[i], s.save()\n\t}\n\treturn item{}, errNaoEncontrado\n}\n\nfunc (s *store) remove(id string) error {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tfor i := range s.Items {\n\t\tif s.Items[i].ID != id {\n\t\t\tcontinue\n\t\t}\n\t\ts.Items = append(s.Items[:i], s.Items[i+1:]...)\n\t\treturn s.save()\n\t}\n\treturn errNaoEncontrado\n}\n\nvar errNaoEncontrado = errors.New(\"item nao encontrado\")\n\ntype servidor struct {\n\tstore    *store\n\tsenha    string\n\tmu       sync.Mutex\n\ttokens   map[string]time.Time\n\tvalidade time.Duration\n}\n\nfunc (s *servidor) emitirToken() (string, error) {\n\tbruto := make([]byte, 32)\n\tif _, err := rand.Read(bruto); err != nil {\n\t\treturn \"\", err\n\t}\n\ttoken := hex.EncodeToString(bruto)\n\ts.mu.Lock()\n\ts.tokens[token] = time.Now().UTC().Add(s.validade)\n\ts.mu.Unlock()\n\treturn token, nil\n}\n\nfunc (s *servidor) tokenValido(r *http.Request) bool {\n\tcabecalho := strings.TrimSpace(r.Header.Get(\"Authorization\"))\n\tif !strings.HasPrefix(cabecalho, \"Bearer \") {\n\t\treturn false\n\t}\n\ttoken := strings.TrimSpace(strings.TrimPrefix(cabecalho, \"Bearer \"))\n\ts.mu.Lock()\n\texpira, existe := s.tokens[token]\n\tif existe && time.Now().UTC().After(expira) {\n\t\tdelete(s.tokens, token)\n\t\texiste = false\n\t}\n\ts.mu.Unlock()\n\treturn existe\n}\n\nfunc escreverJSON(w http.ResponseWriter, status int, corpo any) {\n\tw.Header().Set(\"Content-Type\", \"application/json; charset=utf-8\")\n\tw.WriteHeader(status)\n\t_ = json.NewEncoder(w).Encode(corpo)\n}\n\nfunc (s *servidor) login(w http.ResponseWriter, r *http.Request) {\n\tif r.Method != http.MethodPost {\n\t\tescreverJSON(w, http.StatusMethodNotAllowed, map[string]string{\"error\": \"use POST\"})\n\t\treturn\n\t}\n\tvar pedido struct {\n\t\tPassword string `json:\"password\"`\n\t}\n\tif err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&pedido); err != nil {\n\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"corpo invalido\"})\n\t\treturn\n\t}\n\t// Comparacao em tempo constante: comparar com == vazaria o tamanho do prefixo correto.\n\tif subtle.ConstantTimeCompare([]byte(pedido.Password), []byte(s.senha)) != 1 {\n\t\tescreverJSON(w, http.StatusUnauthorized, map[string]string{\"error\": \"credencial invalida\"})\n\t\treturn\n\t}\n\ttoken, err := s.emitirToken()\n\tif err != nil {\n\t\tescreverJSON(w, http.StatusInternalServerError, map[string]string{\"error\": \"falha ao emitir token\"})\n\t\treturn\n\t}\n\tescreverJSON(w, http.StatusOK, map[string]any{\"token\": token, \"expires_in_seconds\": int(s.validade.Seconds())})\n}\n\nfunc (s *servidor) comAutenticacao(proximo http.HandlerFunc) http.HandlerFunc {\n\treturn func(w http.ResponseWriter, r *http.Request) {\n\t\tif !s.tokenValido(r) {\n\t\t\tescreverJSON(w, http.StatusUnauthorized, map[string]string{\"error\": \"token ausente ou invalido\"})\n\t\t\treturn\n\t\t}\n\t\tproximo(w, r)\n\t}\n}\n\nfunc (s *servidor) itens(w http.ResponseWriter, r *http.Request) {\n\tswitch r.Method {\n\tcase http.MethodGet:\n\t\tescreverJSON(w, http.StatusOK, map[string]any{\"items\": s.store.list()})\n\tcase http.MethodPost:\n\t\tvar pedido struct {\n\t\t\tName     string `json:\"name\"`\n\t\t\tQuantity int    `json:\"quantity\"`\n\t\t}\n\t\tif err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&pedido); err != nil {\n\t\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"corpo invalido\"})\n\t\t\treturn\n\t\t}\n\t\tif strings.TrimSpace(pedido.Name) == \"\" {\n\t\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"name e obrigatorio\"})\n\t\t\treturn\n\t\t}\n\t\tif pedido.Quantity < 0 {\n\t\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"quantity nao pode ser negativa\"})\n\t\t\treturn\n\t\t}\n\t\tcriado, err := s.store.create(strings.TrimSpace(pedido.Name), pedido.Quantity)\n\t\tif err != nil {\n\t\t\tescreverJSON(w, http.StatusInternalServerError, map[string]string{\"error\": err.Error()})\n\t\t\treturn\n\t\t}\n\t\tescreverJSON(w, http.StatusCreated, criado)\n\tdefault:\n\t\tescreverJSON(w, http.StatusMethodNotAllowed, map[string]string{\"error\": \"metodo nao suportado\"})\n\t}\n}\n\nfunc (s *servidor) itemPorID(w http.ResponseWriter, r *http.Request) {\n\tid := strings.TrimPrefix(r.URL.Path, \"/api/items/\")\n\tif strings.TrimSpace(id) == \"\" {\n\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"id ausente\"})\n\t\treturn\n\t}\n\tswitch r.Method {\n\tcase http.MethodPut:\n\t\tvar pedido struct {\n\t\t\tName     string `json:\"name\"`\n\t\t\tQuantity int    `json:\"quantity\"`\n\t\t}\n\t\tif err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&pedido); err != nil {\n\t\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"corpo invalido\"})\n\t\t\treturn\n\t\t}\n\t\tif strings.TrimSpace(pedido.Name) == \"\" || pedido.Quantity < 0 {\n\t\t\tescreverJSON(w, http.StatusBadRequest, map[string]string{\"error\": \"name e obrigatorio e quantity nao pode ser negativa\"})\n\t\t\treturn\n\t\t}\n\t\tatualizado, err := s.store.update(id, strings.TrimSpace(pedido.Name), pedido.Quantity)\n\t\tif errors.Is(err, errNaoEncontrado) {\n\t\t\tescreverJSON(w, http.StatusNotFound, map[string]string{\"error\": \"item nao encontrado\"})\n\t\t\treturn\n\t\t}\n\t\tif err != nil {\n\t\t\tescreverJSON(w, http.StatusInternalServerError, map[string]string{\"error\": err.Error()})\n\t\t\treturn\n\t\t}\n\t\tescreverJSON(w, http.StatusOK, atualizado)\n\tcase http.MethodDelete:\n\t\tif err := s.store.remove(id); errors.Is(err, errNaoEncontrado) {\n\t\t\tescreverJSON(w, http.StatusNotFound, map[string]string{\"error\": \"item nao encontrado\"})\n\t\t\treturn\n\t\t} else if err != nil {\n\t\t\tescreverJSON(w, http.StatusInternalServerError, map[string]string{\"error\": err.Error()})\n\t\t\treturn\n\t\t}\n\t\tw.WriteHeader(http.StatusNoContent)\n\tdefault:\n\t\tescreverJSON(w, http.StatusMethodNotAllowed, map[string]string{\"error\": \"metodo nao suportado\"})\n\t}\n}\n\nfunc main() {\n\tsenha := strings.TrimSpace(os.Getenv(\"APP_ADMIN_PASSWORD\"))\n\tif senha == \"\" {\n\t\tfmt.Fprintln(os.Stderr, \"APP_ADMIN_PASSWORD e obrigatorio: a aplicacao nao sobe com credencial padrao\")\n\t\tos.Exit(1)\n\t}\n\tdirDados := strings.TrimSpace(os.Getenv(\"APP_DATA_DIR\"))\n\tif dirDados == \"\" {\n\t\tdirDados = \".\"\n\t}\n\tloja, err := openStore(filepath.Join(dirDados, \"data.json\"))\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, \"falha ao abrir os dados:\", err)\n\t\tos.Exit(1)\n\t}\n\tsrv := &servidor{store: loja, senha: senha, tokens: map[string]time.Time{}, validade: 12 * time.Hour}\n\n\tmux := http.NewServeMux()\n\tmux.HandleFunc(\"/api/login\", srv.login)\n\tmux.HandleFunc(\"/api/items\", srv.comAutenticacao(srv.itens))\n\tmux.HandleFunc(\"/api/items/\", srv.comAutenticacao(srv.itemPorID))\n\tmux.HandleFunc(\"/healthz\", func(w http.ResponseWriter, _ *http.Request) {\n\t\tescreverJSON(w, http.StatusOK, map[string]string{\"status\": \"ok\"})\n\t})\n\tmux.Handle(\"/\", http.FileServer(http.Dir(\".\")))\n\n\tporta := strings.TrimSpace(os.Getenv(\"PORT\"))\n\tif porta == \"\" {\n\t\tporta = \"0\"\n\t}\n\touvinte, err := net.Listen(\"tcp\", \"127.0.0.1:\"+porta)\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, \"falha ao abrir a porta:\", err)\n\t\tos.Exit(1)\n\t}\n\t// Anunciar a porta REAL permite subir em :0 (teste automatizado) sem adivinhar.\n\tfmt.Println(\"PORT=\" + fmt.Sprint(ouvinte.Addr().(*net.TCPAddr).Port))\n\tif err := http.Serve(ouvinte, mux); err != nil {\n\t\tfmt.Fprintln(os.Stderr, \"servidor encerrado:\", err)\n\t\tos.Exit(1)\n\t}\n}\n"

const appFullstackIndex = "<!doctype html>\n<html lang=\"pt-BR\">\n<head>\n<meta charset=\"utf-8\">\n<link rel=\"icon\" href=\"data:,\">\n<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">\n<title>Gestao gerada pelo Hades</title>\n<style>\n body{font-family:system-ui;margin:0;padding:16px;background:#fafafa;color:#111}\n @media (prefers-color-scheme:dark){body{background:#111;color:#eee}}\n main{max-width:720px;margin:0 auto}\n form{display:flex;flex-wrap:wrap;gap:8px;margin:12px 0}\n label{font-size:.85rem;display:block;width:100%}\n input{font:inherit;padding:8px;border:1px solid #999;border-radius:6px;min-width:0;flex:1}\n button{font:inherit;padding:8px 14px;border-radius:6px;border:1px solid #555;background:#eee;cursor:pointer}\n table{width:100%;border-collapse:collapse;margin-top:12px}\n th,td{text-align:left;padding:6px;border-bottom:1px solid #ccc;font-size:.9rem}\n :focus-visible{outline:3px solid #2563eb;outline-offset:2px}\n</style>\n</head>\n<body>\n<main>\n  <h1>Gestao de itens</h1>\n  <p id=\"aviso\" role=\"status\" aria-live=\"polite\"></p>\n\n  <section aria-labelledby=\"titulo-login\">\n    <h2 id=\"titulo-login\">Entrar</h2>\n    <form id=\"form-login\">\n      <label for=\"senha\">Senha de administrador</label>\n      <input id=\"senha\" name=\"senha\" type=\"password\" autocomplete=\"current-password\" required>\n      <button type=\"submit\">Entrar</button>\n    </form>\n  </section>\n\n  <section id=\"area-itens\" hidden aria-labelledby=\"titulo-itens\">\n    <h2 id=\"titulo-itens\">Itens</h2>\n    <form id=\"form-item\">\n      <label for=\"nome\">Nome do item</label>\n      <input id=\"nome\" name=\"nome\" required>\n      <label for=\"quantidade\">Quantidade</label>\n      <input id=\"quantidade\" name=\"quantidade\" type=\"number\" min=\"0\" value=\"0\" required>\n      <button type=\"submit\">Adicionar</button>\n    </form>\n    <table>\n      <caption class=\"sr-only\">Itens cadastrados</caption>\n      <thead><tr><th scope=\"col\">Nome</th><th scope=\"col\">Quantidade</th><th scope=\"col\">Acoes</th></tr></thead>\n      <tbody id=\"corpo\"></tbody>\n    </table>\n    <button id=\"sair\" type=\"button\">Sair</button>\n  </section>\n</main>\n<script>\nlet token = \"\";\nconst aviso = (texto, erro) => {\n  const el = document.getElementById(\"aviso\");\n  el.textContent = texto;\n  el.style.color = erro ? \"#b91c1c\" : \"inherit\";\n};\nasync function api(caminho, opcoes = {}) {\n  const resposta = await fetch(caminho, {\n    ...opcoes,\n    headers: { \"Content-Type\": \"application/json\", ...(token ? { Authorization: \"Bearer \" + token } : {}), ...(opcoes.headers || {}) },\n  });\n  if (resposta.status === 204) return null;\n  const corpo = await resposta.json().catch(() => ({}));\n  if (!resposta.ok) throw new Error(corpo.error || (\"HTTP \" + resposta.status));\n  return corpo;\n}\nasync function carregar() {\n  const dados = await api(\"/api/items\");\n  const corpo = document.getElementById(\"corpo\");\n  corpo.textContent = \"\";\n  for (const item of dados.items || []) {\n    const linha = document.createElement(\"tr\");\n    const nome = document.createElement(\"td\");\n    nome.textContent = item.name;\n    const qtd = document.createElement(\"td\");\n    qtd.textContent = String(item.quantity);\n    const acoes = document.createElement(\"td\");\n    const remover = document.createElement(\"button\");\n    remover.type = \"button\";\n    remover.textContent = \"Remover\";\n    remover.setAttribute(\"aria-label\", \"Remover \" + item.name);\n    remover.addEventListener(\"click\", async () => { await api(\"/api/items/\" + item.id, { method: \"DELETE\" }); await carregar(); });\n    acoes.appendChild(remover);\n    linha.append(nome, qtd, acoes);\n    corpo.appendChild(linha);\n  }\n}\ndocument.getElementById(\"form-login\").addEventListener(\"submit\", async (evento) => {\n  evento.preventDefault();\n  try {\n    const dados = await api(\"/api/login\", { method: \"POST\", body: JSON.stringify({ password: document.getElementById(\"senha\").value }) });\n    token = dados.token;\n    document.getElementById(\"area-itens\").hidden = false;\n    aviso(\"Sessao iniciada.\");\n    await carregar();\n  } catch (erro) { aviso(erro.message, true); }\n});\ndocument.getElementById(\"form-item\").addEventListener(\"submit\", async (evento) => {\n  evento.preventDefault();\n  try {\n    await api(\"/api/items\", { method: \"POST\", body: JSON.stringify({ name: document.getElementById(\"nome\").value, quantity: Number(document.getElementById(\"quantidade\").value) }) });\n    document.getElementById(\"nome\").value = \"\";\n    await carregar();\n  } catch (erro) { aviso(erro.message, true); }\n});\ndocument.getElementById(\"sair\").addEventListener(\"click\", () => {\n  token = \"\";\n  document.getElementById(\"area-itens\").hidden = true;\n  aviso(\"Sessao encerrada.\");\n});\n</script>\n</body>\n</html>\n"

const appFullstackReadme = "# Aplicacao de gestao (gerada pelo Hades)\n\nAplicacao full-stack real, sem dependencias externas: login, CRUD de itens e persistencia em disco.\n\n## Como rodar\n\n```sh\nAPP_ADMIN_PASSWORD='uma-senha-forte' APP_DATA_DIR=./dados go run main.go\n```\n\n- `APP_ADMIN_PASSWORD` e **obrigatoria**: a aplicacao nao sobe com credencial padrao.\n- `PORT` vazio (ou `0`) faz a aplicacao escolher uma porta livre e anunciar `PORT=<numero>` na saida.\n- Os dados ficam em `APP_DATA_DIR/data.json`, gravados com escrita ATOMICA (temporario + rename).\n\n## API\n\n| Metodo | Rota | Descricao |\n| --- | --- | --- |\n| POST | `/api/login` | `{\"password\":\"...\"}` -> `{\"token\":\"...\"}` |\n| GET | `/api/items` | lista (exige `Authorization: Bearer <token>`) |\n| POST | `/api/items` | cria `{\"name\":\"...\",\"quantity\":N}` |\n| PUT | `/api/items/{id}` | atualiza |\n| DELETE | `/api/items/{id}` | remove |\n| GET | `/healthz` | saude |\n\n## Limites declarados\n\n- Persistencia em ARQUIVO com escrita atomica, **nao** um SGBD relacional.\n- Token em memoria: reiniciar a aplicacao invalida as sessoes (os dados permanecem).\n"
