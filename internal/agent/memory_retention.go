package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Política de retenção de memória por organização (estágio 15).
//
// A fatia anterior entregou exclusão, exportação e retenção por chamada
// explícita. Faltava a POLÍTICA: um limite declarado pelo operador e aplicado de
// forma idempotente, para que "retenção indefinida" deixe de ser o padrão por
// omissão.
//
// Decisão de honestidade: a aplicação é EXPLÍCITA (`ApplyRetentionForOrganization`).
// Não existe laço de fundo fingindo automação: o ambiente atual não garante um
// executor persistente, e prometer expurgo automático sem ele seria mentira.
// Quem quiser periodicidade agenda a chamada (o runtime já tem schedules).

const (
	// minRetentionDays e maxRetentionDays limitam a janela declarada.
	minRetentionDays = 1
	maxRetentionDays = 3650
	// maxRetentionMemories limita o teto por projeto.
	maxRetentionMemories = 10000
)

var (
	ErrRetentionPolicyInvalid = errors.New("retention policy is invalid")
	ErrRetentionNotConfigured = errors.New("retention policy is not configured for this organization")
)

// MemoryRetentionPolicy é a política declarada para uma organização.
type MemoryRetentionPolicy struct {
	OrganizationID        string    `json:"organization_id"`
	MaxAgeDays            int       `json:"max_age_days"`
	MaxMemoriesPerProject int       `json:"max_memories_per_project"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// RetentionResult resume o que a aplicação fez — sempre com números reais.
type RetentionResult struct {
	OrganizationID  string    `json:"organization_id"`
	ProjectsScanned int       `json:"projects_scanned"`
	RemovedByAge    int       `json:"removed_by_age"`
	RemovedByCount  int       `json:"removed_by_count"`
	AppliedAt       time.Time `json:"applied_at"`
}

func validateRetentionPolicy(policy MemoryRetentionPolicy) (MemoryRetentionPolicy, error) {
	policy.OrganizationID = strings.TrimSpace(policy.OrganizationID)
	if policy.OrganizationID == "" || strings.ContainsAny(policy.OrganizationID, "/\\:") {
		return MemoryRetentionPolicy{}, fmt.Errorf("%w: organization is required", ErrRetentionPolicyInvalid)
	}
	if policy.MaxAgeDays < minRetentionDays || policy.MaxAgeDays > maxRetentionDays {
		return MemoryRetentionPolicy{}, fmt.Errorf("%w: max_age_days must be between %d and %d", ErrRetentionPolicyInvalid, minRetentionDays, maxRetentionDays)
	}
	if policy.MaxMemoriesPerProject < 0 || policy.MaxMemoriesPerProject > maxRetentionMemories {
		return MemoryRetentionPolicy{}, fmt.Errorf("%w: max_memories_per_project must be between 0 and %d", ErrRetentionPolicyInvalid, maxRetentionMemories)
	}
	return policy, nil
}

func (s *ContextStore) retentionPath(organizationID string) string {
	return filepath.Join(s.root, "retention", retentionFileComponent(organizationID)+".json")
}

// retentionFileComponent deixa o identificador da organização seguro como nome
// de arquivo.
func retentionFileComponent(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(strings.TrimSpace(value))
}

// SetRetentionPolicyForOrganization grava a política com persistência atômica e
// rollback do estado em memória quando a gravação falha.
func (s *ContextStore) SetRetentionPolicyForOrganization(policy MemoryRetentionPolicy) (MemoryRetentionPolicy, error) {
	validated, err := validateRetentionPolicy(policy)
	if err != nil {
		return MemoryRetentionPolicy{}, err
	}
	validated.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retention == nil {
		s.retention = map[string]MemoryRetentionPolicy{}
	}
	previous, existed := s.retention[validated.OrganizationID]
	s.retention[validated.OrganizationID] = validated
	if s.root != "" {
		if err := writeJSONAtomic(s.retentionPath(validated.OrganizationID), validated); err != nil {
			if existed {
				s.retention[validated.OrganizationID] = previous
			} else {
				delete(s.retention, validated.OrganizationID)
			}
			return MemoryRetentionPolicy{}, err
		}
	}
	return validated, nil
}

// RetentionPolicyForOrganization devolve a política declarada, se houver. A
// ausência é explícita: nenhuma retenção é aplicada por omissão.
func (s *ContextStore) RetentionPolicyForOrganization(organizationID string) (MemoryRetentionPolicy, error) {
	organizationID = strings.TrimSpace(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.retention[organizationID]
	if !ok {
		return MemoryRetentionPolicy{}, ErrRetentionNotConfigured
	}
	return policy, nil
}

// ApplyRetentionForOrganization aplica a política aos projetos DA organização:
// remove o que passou da idade máxima e corta o excedente mantendo o mais novo.
// É idempotente: repetir a chamada não remove nada a mais.
func (s *ContextStore) ApplyRetentionForOrganization(organizationID string, now time.Time) (RetentionResult, error) {
	organizationID = strings.TrimSpace(organizationID)
	policy, err := s.RetentionPolicyForOrganization(organizationID)
	if err != nil {
		return RetentionResult{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := now.AddDate(0, 0, -policy.MaxAgeDays)
	result := RetentionResult{OrganizationID: organizationID, AppliedAt: now}
	for _, project := range s.ListProjects() {
		if strings.TrimSpace(project.OrganizationID) != organizationID {
			continue
		}
		result.ProjectsScanned++
		removedByAge, err := s.PruneMemories(project.ID, cutoff)
		if err != nil {
			return result, err
		}
		result.RemovedByAge += removedByAge
		if policy.MaxMemoriesPerProject <= 0 {
			continue
		}
		trimmed, err := s.trimMemoriesToLimit(project.ID, policy.MaxMemoriesPerProject, now)
		if err != nil {
			return result, err
		}
		result.RemovedByCount += trimmed
	}
	return result, nil
}

// trimMemoriesToLimit mantém as N memórias mais recentes do projeto. Memória sem
// data é preservada (não se apaga o que não se sabe quando foi criado) e por
// isso pode fazer o total passar do teto — o resultado informa o que saiu.
func (s *ContextStore) trimMemoriesToLimit(projectID string, limit int, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	memories := s.memories[projectID]
	if len(memories) <= limit {
		return 0, nil
	}
	indexes := make([]int, 0, len(memories))
	for index, memory := range memories {
		if memory.CreatedAt.IsZero() {
			continue
		}
		indexes = append(indexes, index)
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		return memories[indexes[i]].CreatedAt.After(memories[indexes[j]].CreatedAt)
	})
	drop := map[int]struct{}{}
	for position, index := range indexes {
		if position < limit {
			continue
		}
		drop[index] = struct{}{}
	}
	if len(drop) == 0 {
		return 0, nil
	}
	kept := make([]Memory, 0, len(memories)-len(drop))
	for index, memory := range memories {
		if _, ok := drop[index]; ok {
			continue
		}
		kept = append(kept, memory)
	}
	previous := append([]Memory(nil), memories...)
	if len(kept) == 0 {
		delete(s.memories, projectID)
	} else {
		s.memories[projectID] = kept
	}
	if s.root != "" {
		if err := writeJSONAtomic(filepath.Join(s.root, "memories", projectID+".json"), kept); err != nil {
			if previous == nil {
				delete(s.memories, projectID)
			} else {
				s.memories[projectID] = previous
			}
			return 0, err
		}
	}
	return len(drop), nil
}
