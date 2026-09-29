package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ContextStore struct {
	mu            sync.RWMutex
	root          string
	workspaceRoot string
	projects      map[string]Project
	memories      map[string][]Memory
	skills        map[string]SkillManifest
	skillPaths    map[string]string
	schedules     map[string]Schedule
	embedder      Embedder
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// LocalOrganizationID is reserved for the unauthenticated single-user API.
// Ownerless records are accepted for backwards compatibility with local data,
// but tenant-owned records must never be visible through the local scope.
const LocalOrganizationID = "local"

func scheduleOwnedByOrganization(owner, organizationID string) bool {
	owner = strings.TrimSpace(owner)
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return owner == ""
	}
	if organizationID == LocalOrganizationID {
		return owner == "" || owner == LocalOrganizationID
	}
	return owner == organizationID
}

func NewContextStore(root string) (*ContextStore, error) {
	if strings.TrimSpace(root) == "" {
		return &ContextStore{projects: map[string]Project{}, memories: map[string][]Memory{}, skills: map[string]SkillManifest{}, skillPaths: map[string]string{}, schedules: map[string]Schedule{}}, nil
	}
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "memories"), 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "schedules"), 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o700); err != nil {
		return nil, err
	}
	store := &ContextStore{root: root, projects: map[string]Project{}, memories: map[string][]Memory{}, skills: map[string]SkillManifest{}, skillPaths: map[string]string{}, schedules: map[string]Schedule{}}
	projectEntries, err := os.ReadDir(filepath.Join(root, "projects"))
	if err != nil {
		return nil, err
	}
	for _, entry := range projectEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var project Project
		if err := readJSON(filepath.Join(root, "projects", entry.Name()), &project); err != nil {
			return nil, err
		}
		store.projects[project.ID] = project
	}
	memoryEntries, err := os.ReadDir(filepath.Join(root, "memories"))
	if err != nil {
		return nil, err
	}
	for _, entry := range memoryEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var memories []Memory
		if err := readJSON(filepath.Join(root, "memories", entry.Name()), &memories); err != nil {
			return nil, err
		}
		projectID := strings.TrimSuffix(entry.Name(), ".json")
		store.memories[projectID] = memories
	}
	scheduleEntries, err := os.ReadDir(filepath.Join(root, "schedules"))
	if err != nil {
		return nil, err
	}
	for _, entry := range scheduleEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var schedule Schedule
		if err := readJSON(filepath.Join(root, "schedules", entry.Name()), &schedule); err != nil {
			return nil, err
		}
		store.schedules[schedule.ID] = schedule
	}
	skillEntries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return nil, err
	}
	for _, entry := range skillEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var manifest SkillManifest
		if err := readJSON(filepath.Join(root, "skills", entry.Name()), &manifest); err != nil {
			return nil, err
		}
		if err := validateSkillManifest(&manifest); err != nil {
			return nil, fmt.Errorf("skill %s: %w", entry.Name(), err)
		}
		manifest.Trusted = false
		store.skills[manifest.ID] = manifest
		store.skillPaths[manifest.ID] = filepath.Join(root, "skills", entry.Name())
	}
	return store, nil
}

func (s *ContextStore) SetWorkspaceRoot(root string) error {
	if s == nil {
		return errors.New("context store is required")
	}
	canonical, err := canonicalExistingDirectory(root)
	if err != nil {
		return fmt.Errorf("runtime workspace is invalid: %w", err)
	}
	s.mu.Lock()
	s.workspaceRoot = canonical
	s.mu.Unlock()
	return nil
}

func (s *ContextStore) CreateProject(name, root string, organizationIDs ...string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, errors.New("project name is required")
	}
	if len(name) > 200 {
		return Project{}, errors.New("project name is too long")
	}
	now := time.Now().UTC()
	organizationID := ""
	if len(organizationIDs) > 0 {
		organizationID = strings.TrimSpace(organizationIDs[0])
	}
	project := Project{ID: "prj_" + uuid.NewString(), Name: name, Root: root, OrganizationID: organizationID, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	canonicalRoot, err := canonicalProjectRoot(s.workspaceRoot, root)
	if err != nil {
		return Project{}, err
	}
	project.Root = canonicalRoot
	s.projects[project.ID] = project
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "projects", project.ID+".json"), project); err != nil {
			delete(s.projects, project.ID)
			return Project{}, err
		}
	}
	return project, nil
}

func (s *ContextStore) ListProjects() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Project, 0, len(s.projects))
	for _, project := range s.projects {
		result = append(result, project)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *ContextStore) GetProject(id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[strings.TrimSpace(id)]
	if !ok {
		return Project{}, os.ErrNotExist
	}
	return project, nil
}

func (s *ContextStore) UpdateProject(id, name, root string) (Project, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" {
		return Project{}, errors.New("project id is required")
	}
	if name == "" {
		return Project{}, errors.New("project name is required")
	}
	if len(name) > 200 {
		return Project{}, errors.New("project name is too long")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[id]
	if !ok {
		return Project{}, os.ErrNotExist
	}
	canonicalRoot, err := canonicalProjectRoot(s.workspaceRoot, root)
	if err != nil {
		return Project{}, err
	}
	previous := project
	project.Name = name
	project.Root = canonicalRoot
	project.UpdatedAt = time.Now().UTC()
	s.projects[id] = project
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "projects", id+".json"), project); err != nil {
			s.projects[id] = previous
			return Project{}, err
		}
	}
	return project, nil
}

func (s *ContextStore) DeleteProject(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("project id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return os.ErrNotExist
	}
	delete(s.projects, id)
	delete(s.memories, id)
	if s.root != "" {
		if err := os.Remove(filepath.Join(s.root, "projects", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Remove(filepath.Join(s.root, "memories", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *ContextStore) SetEmbedder(embedder Embedder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.embedder = embedder
}

func (s *ContextStore) AddMemory(memory Memory) (Memory, error) {
	return s.AddMemoryContext(context.Background(), memory)
}

func (s *ContextStore) AddMemoryContext(ctx context.Context, memory Memory) (Memory, error) {
	memory.Content = strings.TrimSpace(memory.Content)
	if memory.Content == "" {
		return Memory{}, errors.New("memory content is required")
	}
	if len(memory.Content) > 64<<10 {
		return Memory{}, errors.New("memory content is too long")
	}
	if memory.ID == "" {
		memory.ID = "mem_" + uuid.NewString()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now().UTC()
	}
	if memory.Confidence < 0 || memory.Confidence > 1 {
		return Memory{}, errors.New("memory confidence must be between 0 and 1")
	}
	s.mu.RLock()
	embedder := s.embedder
	s.mu.RUnlock()
	if embedder != nil && len(memory.Embedding) == 0 {
		embedding, err := embedder.Embed(ctx, memory.Content)
		if err != nil {
			return Memory{}, fmt.Errorf("embed memory: %w", err)
		}
		if len(embedding) == 0 || len(embedding) > 16384 {
			return Memory{}, errors.New("embedder returned an invalid vector")
		}
		memory.Embedding = embedding
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]Memory(nil), s.memories[memory.ProjectID]...)
	s.memories[memory.ProjectID] = append(s.memories[memory.ProjectID], memory)
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "memories", memory.ProjectID+".json"), s.memories[memory.ProjectID]); err != nil {
			if previous == nil {
				delete(s.memories, memory.ProjectID)
			} else {
				s.memories[memory.ProjectID] = previous
			}
			return Memory{}, err
		}
	}
	return memory, nil
}

func (s *ContextStore) SearchMemories(projectID, query string, limit int) []Memory {
	result, _ := s.SearchMemoriesContext(context.Background(), projectID, query, limit)
	return result
}

func (s *ContextStore) SearchMemoriesContext(ctx context.Context, projectID, query string, limit int) ([]Memory, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	s.mu.RLock()
	embedder := s.embedder
	memories := append([]Memory(nil), s.memories[projectID]...)
	s.mu.RUnlock()
	var queryVector []float32
	var err error
	if embedder != nil && query != "" {
		queryVector, err = embedder.Embed(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("embed query: %w", err)
		}
	}
	type scoredMemory struct {
		memory Memory
		score  float64
	}
	scored := make([]scoredMemory, 0, len(memories))
	for _, memory := range memories {
		var score float64
		if len(queryVector) > 0 && len(memory.Embedding) > 0 {
			score = cosineSimilarity(queryVector, memory.Embedding)
		} else if query == "" || strings.Contains(strings.ToLower(memory.Content), query) || strings.Contains(strings.ToLower(memory.Kind), query) {
			score = 1
		} else {
			continue
		}
		scored = append(scored, scoredMemory{memory: memory, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].memory.CreatedAt.After(scored[j].memory.CreatedAt)
		}
		return scored[i].score > scored[j].score
	})
	matches := make([]Memory, 0, len(scored))
	for _, item := range scored {
		matches = append(matches, item.memory)
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func cosineSimilarity(a, b []float32) float64 {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	var dot, normA, normB float64
	for i := range length {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		normA += x * x
		normB += y * y
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *ContextStore) LoadSkills(dir string, _ bool) error {
	return s.LoadSkillsForOrganization(dir, "")
}

func (s *ContextStore) LoadSkillsForOrganization(dir, organizationID string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	loaded := make(map[string]SkillManifest)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		var manifest SkillManifest
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&manifest); err != nil {
			return fmt.Errorf("skill %s: %w", entry.Name(), err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return fmt.Errorf("skill %s contains trailing JSON", entry.Name())
			}
			return fmt.Errorf("skill %s trailing JSON: %w", entry.Name(), err)
		}
		if strings.TrimSpace(manifest.ID) == "" || strings.TrimSpace(manifest.Version) == "" {
			return fmt.Errorf("skill %s has no id or version", entry.Name())
		}
		// A skill manifest is untrusted until a future signed-attestation path
		// verifies its source, digest and owner. Never accept trust from JSON or
		// from a caller-controlled boolean.
		manifest.OrganizationID = strings.TrimSpace(organizationID)
		manifest.Trusted = false
		manifest.Enabled = true
		loaded[manifest.ID] = manifest
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, manifest := range loaded {
		s.skills[id] = manifest
	}
	return nil
}

func validateSkillManifest(manifest *SkillManifest) error {
	manifest.ID = strings.TrimSpace(manifest.ID)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.OrganizationID = strings.TrimSpace(manifest.OrganizationID)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if manifest.ID == "" || manifest.Version == "" {
		return errors.New("skill id and version are required")
	}
	if len(manifest.ID) > 120 || strings.ContainsAny(manifest.ID, "/\\\x00\r\n") {
		return errors.New("skill id is invalid")
	}
	if len(manifest.Version) > 64 || len(manifest.Description) > 4000 {
		return errors.New("skill manifest field is too long")
	}
	return nil
}

func (s *ContextStore) RegisterSkill(manifest SkillManifest) error {
	return s.registerSkill(manifest, "")
}

func (s *ContextStore) RegisterSkillForOrganization(organizationID string, manifest SkillManifest) error {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return errors.New("skill organization scope is required")
	}
	if supplied := strings.TrimSpace(manifest.OrganizationID); supplied != "" && supplied != organizationID {
		return ErrPluginOrganizationScope
	}
	manifest.OrganizationID = organizationID
	return s.registerSkill(manifest, organizationID)
}

func (s *ContextStore) registerSkill(manifest SkillManifest, organizationID string) error {
	manifest.Trusted = false
	manifest.Enabled = true
	if err := validateSkillManifest(&manifest); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if organizationID != "" {
		if existing, ok := s.skills[manifest.ID]; ok && !pluginOwnedByOrganization(existing.OrganizationID, organizationID) {
			return ErrPluginOrganizationScope
		}
	}
	if s.skillPaths == nil {
		s.skillPaths = map[string]string{}
	}
	path := s.skillPaths[manifest.ID]
	if path == "" && s.root != "" {
		path = filepath.Join(s.root, "skills", manifest.ID+".json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
	}
	previous, existed := s.skills[manifest.ID]
	previousPath := s.skillPaths[manifest.ID]
	s.skills[manifest.ID] = manifest
	if path != "" {
		s.skillPaths[manifest.ID] = path
		if err := writeJSONAtomic(path, manifest); err != nil {
			if existed {
				s.skills[manifest.ID] = previous
			} else {
				delete(s.skills, manifest.ID)
			}
			if previousPath != "" {
				s.skillPaths[manifest.ID] = previousPath
			} else {
				delete(s.skillPaths, manifest.ID)
			}
			return fmt.Errorf("persist skill manifest: %w", err)
		}
	}
	return nil
}

func (s *ContextStore) persistSkillLocked(id string) error {
	path := s.skillPaths[id]
	if path == "" {
		return nil
	}
	manifest := s.skills[id]
	manifest.Trusted = false
	return writeJSONAtomic(path, manifest)
}

func (s *ContextStore) Skills() []SkillManifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]SkillManifest, 0, len(s.skills))
	for _, skill := range s.skills {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *ContextStore) SkillsForOrganization(organizationID string) []SkillManifest {
	organizationID = strings.TrimSpace(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]SkillManifest, 0)
	for _, skill := range s.skills {
		if skill.OrganizationID != "" && !pluginOwnedByOrganization(skill.OrganizationID, organizationID) {
			continue
		}
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *ContextStore) SetSkillEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	skill, ok := s.skills[id]
	if !ok {
		return fmt.Errorf("skill %q is not registered", id)
	}
	skill.Enabled = enabled
	s.skills[id] = skill
	if err := s.persistSkillLocked(id); err != nil {
		skill.Enabled = !skill.Enabled
		s.skills[id] = skill
		return fmt.Errorf("persist skill manifest: %w", err)
	}
	return nil
}

func (s *ContextStore) SetSkillEnabledForOrganization(organizationID, id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	skill, ok := s.skills[id]
	if !ok {
		return fmt.Errorf("skill %q is not registered", id)
	}
	if !pluginOwnedByOrganization(skill.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	skill.Enabled = enabled
	s.skills[id] = skill
	if err := s.persistSkillLocked(id); err != nil {
		skill.Enabled = !skill.Enabled
		s.skills[id] = skill
		return fmt.Errorf("persist skill manifest: %w", err)
	}
	return nil
}

func (s *ContextStore) RemoveSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	skill, ok := s.skills[id]
	if !ok {
		return fmt.Errorf("skill %q is not registered", id)
	}
	delete(s.skills, id)
	path := s.skillPaths[id]
	delete(s.skillPaths, id)
	if path != "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.skills[id] = skill
			s.skillPaths[id] = path
			return fmt.Errorf("remove skill manifest: %w", err)
		}
	}
	return nil
}

func (s *ContextStore) RemoveSkillForOrganization(organizationID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	skill, ok := s.skills[id]
	if !ok {
		return fmt.Errorf("skill %q is not registered", id)
	}
	if !pluginOwnedByOrganization(skill.OrganizationID, strings.TrimSpace(organizationID)) {
		return ErrPluginOrganizationScope
	}
	delete(s.skills, id)
	path := s.skillPaths[id]
	delete(s.skillPaths, id)
	if path != "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.skills[id] = skill
			s.skillPaths[id] = path
			return fmt.Errorf("remove skill manifest: %w", err)
		}
	}
	return nil
}

func (s *ContextStore) CreateSchedule(schedule Schedule) (Schedule, error) {
	if strings.TrimSpace(schedule.Objective) == "" {
		return Schedule{}, errors.New("schedule objective is required")
	}
	if schedule.IntervalSeconds < 1 || schedule.IntervalSeconds > 31*24*60*60 {
		return Schedule{}, errors.New("schedule interval must be between 1 second and 31 days")
	}
	schedule.ID = strings.TrimSpace(schedule.ID)
	if schedule.ID == "" {
		schedule.ID = "sch_" + uuid.NewString()
	}
	now := time.Now().UTC()
	if schedule.NextRunAt.IsZero() || schedule.NextRunAt.Before(now) {
		schedule.NextRunAt = now.Add(time.Duration(schedule.IntervalSeconds) * time.Second)
	}
	schedule.Enabled = true
	schedule.CreatedAt = now
	schedule.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.schedules[schedule.ID]; exists {
		return Schedule{}, errors.New("schedule ID already exists")
	}
	s.schedules[schedule.ID] = schedule
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "schedules", schedule.ID+".json"), schedule); err != nil {
			delete(s.schedules, schedule.ID)
			return Schedule{}, err
		}
	}
	return schedule, nil
}

func (s *ContextStore) GetSchedule(id string) (Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	schedule, ok := s.schedules[strings.TrimSpace(id)]
	if !ok {
		return Schedule{}, os.ErrNotExist
	}
	return schedule, nil
}

func (s *ContextStore) ListSchedules() []Schedule {
	return s.ListSchedulesForOrganization(LocalOrganizationID)
}

func (s *ContextStore) GetScheduleForOrganization(id, organizationID string) (Schedule, error) {
	if s == nil {
		return Schedule{}, errors.New("context store is required")
	}
	id = strings.TrimSpace(id)
	s.mu.RLock()
	defer s.mu.RUnlock()
	schedule, ok := s.schedules[id]
	if !ok {
		return Schedule{}, os.ErrNotExist
	}
	if !scheduleOwnedByOrganization(schedule.OrganizationID, organizationID) {
		return Schedule{}, ErrPluginOrganizationScope
	}
	return schedule, nil
}

func (s *ContextStore) ListSchedulesForOrganization(organizationID string) []Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Schedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		if !scheduleOwnedByOrganization(schedule.OrganizationID, organizationID) {
			continue
		}
		result = append(result, schedule)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].NextRunAt.Before(result[j].NextRunAt) })
	return result
}

func (s *ContextStore) UpdateSchedule(id string, schedule Schedule) (Schedule, error) {
	return s.updateScheduleForOrganization("", id, schedule, false)
}

// UpdateScheduleForOrganization validates ownership while holding the store
// lock, so a claimed schedule cannot be updated through a different tenant
// scope between the claim and the retry/finalization update.
func (s *ContextStore) UpdateScheduleForOrganization(organizationID, id string, schedule Schedule) (Schedule, error) {
	return s.updateScheduleForOrganization(organizationID, id, schedule, true)
}

func (s *ContextStore) updateScheduleForOrganization(organizationID, id string, schedule Schedule, enforceOwnership bool) (Schedule, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Schedule{}, errors.New("schedule id is required")
	}
	if strings.TrimSpace(schedule.Objective) == "" {
		return Schedule{}, errors.New("schedule objective is required")
	}
	if schedule.IntervalSeconds < 1 || schedule.IntervalSeconds > 31*24*60*60 {
		return Schedule{}, errors.New("schedule interval must be between 1 second and 31 days")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.schedules[id]
	if !ok {
		return Schedule{}, os.ErrNotExist
	}
	if enforceOwnership && !scheduleOwnedByOrganization(current.OrganizationID, organizationID) {
		return Schedule{}, ErrPluginOrganizationScope
	}
	previous := current
	schedule.ID = id
	schedule.OrganizationID = current.OrganizationID
	schedule.CreatedAt = current.CreatedAt
	schedule.UpdatedAt = time.Now().UTC()
	if schedule.NextRunAt.IsZero() {
		schedule.NextRunAt = time.Now().UTC().Add(time.Duration(schedule.IntervalSeconds) * time.Second)
	}
	s.schedules[id] = schedule
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "schedules", id+".json"), schedule); err != nil {
			s.schedules[id] = previous
			return Schedule{}, err
		}
	}
	return schedule, nil
}

func (s *ContextStore) DeleteSchedule(id string) error {
	return s.deleteScheduleForOrganization("", id, false)
}

func (s *ContextStore) DeleteScheduleForOrganization(organizationID, id string) error {
	return s.deleteScheduleForOrganization(organizationID, id, true)
}

func (s *ContextStore) deleteScheduleForOrganization(organizationID, id string, enforceOwnership bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("schedule id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	schedule, ok := s.schedules[id]
	if !ok {
		return os.ErrNotExist
	}
	if enforceOwnership && !scheduleOwnedByOrganization(schedule.OrganizationID, organizationID) {
		return ErrPluginOrganizationScope
	}
	delete(s.schedules, id)
	if s.root != "" {
		if err := os.Remove(filepath.Join(s.root, "schedules", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.schedules[id] = schedule
			return err
		}
	}
	return nil
}

func (s *ContextStore) ClaimDueSchedules(now time.Time) []Schedule {
	return s.ClaimDueSchedulesForOrganization(LocalOrganizationID, now)
}

// ClaimDueSchedulesForOrganization atomically advances only schedules visible
// to the requested scope. An empty/local scope includes ownerless legacy local
// records and the reserved local namespace, never tenant-owned schedules.
func (s *ContextStore) ClaimDueSchedulesForOrganization(organizationID string, now time.Time) []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	var due []Schedule
	for id, schedule := range s.schedules {
		if !scheduleOwnedByOrganization(schedule.OrganizationID, organizationID) || !schedule.Enabled || schedule.NextRunAt.After(now) {
			continue
		}
		previous := schedule
		last := now
		schedule.LastRunAt = &last
		schedule.NextRunAt = now.Add(time.Duration(schedule.IntervalSeconds) * time.Second)
		schedule.UpdatedAt = now
		if s.root != "" {
			if err := writeJSONAtomic(filepath.Join(s.root, "schedules", id+".json"), schedule); err != nil {
				s.schedules[id] = previous
				continue
			}
		}
		s.schedules[id] = schedule
		due = append(due, schedule)
	}
	return due
}
