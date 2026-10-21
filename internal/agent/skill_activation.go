package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Catálogo e ativação de skills — quinta fatia do estágio 8.
//
// O manifesto da skill já era versionado e assinado; o que faltava era o que o
// estágio pede: CATEGORIA, ATIVAÇÃO POR PROJETO/AGENTE/MODELO, PROVENIÊNCIA e
// SELEÇÃO DINÂMICA POR MISSÃO COM ORÇAMENTO DE TOKENS.
//
// Duas regras de segurança sustentam tudo aqui:
//  1. skill NÃO confiável não pode ser ativada (`ErrSkillNotTrusted`) — selo do
//     usuário não substitui assinatura verificada pelo servidor;
//  2. a seleção é DETERMINÍSTICA e nunca inclui skill desabilitada; o que não
//     couber no orçamento é excluído COM MOTIVO, para não parecer que entrou.
var (
	ErrSkillActivationInvalid  = errors.New("skill activation is invalid")
	ErrSkillActivationNotFound = errors.New("skill activation was not found")
	ErrSkillNotTrusted         = errors.New("skill is not trusted by the server")
	ErrSkillNotFound           = errors.New("skill was not found for the organization")
)

// SkillActivation diz onde uma skill vale. Campo vazio é curinga: uma ativação
// sem projeto/agente/modelo vale para qualquer contexto da organização.
type SkillActivation struct {
	SkillID        string    `json:"skill_id"`
	OrganizationID string    `json:"organization_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	AgentID        string    `json:"agent_id,omitempty"`
	Model          string    `json:"model,omitempty"`
	ActivatedAt    time.Time `json:"activated_at"`
}

// Key identifica a ativação de forma estável (usada para desativar).
func (a SkillActivation) Key() string {
	return strings.Join([]string{
		strings.TrimSpace(a.SkillID),
		strings.TrimSpace(a.ProjectID),
		strings.TrimSpace(a.AgentID),
		strings.TrimSpace(a.Model),
	}, "|")
}

// SkillContext é o contexto da missão/agente que consulta o catálogo.
type SkillContext struct {
	ProjectID string
	AgentID   string
	Model     string
}

func (c SkillContext) matches(activation SkillActivation) bool {
	if project := strings.TrimSpace(activation.ProjectID); project != "" && project != strings.TrimSpace(c.ProjectID) {
		return false
	}
	if agent := strings.TrimSpace(activation.AgentID); agent != "" && agent != strings.TrimSpace(c.AgentID) {
		return false
	}
	if model := strings.TrimSpace(activation.Model); model != "" && !strings.EqualFold(model, strings.TrimSpace(c.Model)) {
		return false
	}
	return true
}

// ActiveSkill é uma skill ativa com a proveniência da ativação.
type ActiveSkill struct {
	ID          string          `json:"id"`
	Version     string          `json:"version"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	License     string          `json:"license,omitempty"`
	Homepage    string          `json:"homepage,omitempty"`
	Scopes      []string        `json:"scopes,omitempty"`
	Tools       []string        `json:"tools,omitempty"`
	TokenBudget int             `json:"token_budget,omitempty"`
	Activation  SkillActivation `json:"activation"`
}

// SkillSelection é o resultado da seleção por missão, com o motivo de cada
// exclusão. Nada é omitido em silêncio.
type SkillSelection struct {
	Selected      []ActiveSkill `json:"selected"`
	Skipped       []SkillSkip   `json:"skipped,omitempty"`
	UsedTokens    int           `json:"used_tokens"`
	TokenBudget   int           `json:"token_budget"`
	BudgetDefined bool          `json:"budget_defined"`
}

// MarshalJSON garante o contrato na SAIDA, nao em cada construtor: `selected` sai sempre como
// lista, mesmo vazia. Um slice nil serializa como `null` e o painel faz `.map()` sobre ele --
// foi o `TypeError: Cannot read properties of null (reading 'map')` que quebrou a pagina /skills
// em 2026-10-21. Garantir no tipo cobre todos os caminhos de producao, incluindo os que ainda
// nao existem.
func (s SkillSelection) MarshalJSON() ([]byte, error) {
	type semMetodos SkillSelection
	copia := semMetodos(s)
	if copia.Selected == nil {
		copia.Selected = []ActiveSkill{}
	}
	return json.Marshal(copia)
}

// SkillSkip registra por que uma skill ativa ficou de fora.
type SkillSkip struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ActivateSkillForOrganization ativa uma skill já registrada e confiável.
func (s *ContextStore) ActivateSkillForOrganization(organizationID string, activation SkillActivation) (SkillActivation, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return SkillActivation{}, fmt.Errorf("%w: organization is required", ErrSkillActivationInvalid)
	}
	activation.SkillID = strings.TrimSpace(activation.SkillID)
	if activation.SkillID == "" {
		return SkillActivation{}, fmt.Errorf("%w: skill is required", ErrSkillActivationInvalid)
	}
	activation.ProjectID = strings.TrimSpace(activation.ProjectID)
	activation.AgentID = strings.TrimSpace(activation.AgentID)
	activation.Model = strings.TrimSpace(activation.Model)

	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, err := s.skillLocked(organizationID, activation.SkillID)
	if err != nil {
		return SkillActivation{}, err
	}
	// Skill não confiável não recebe ativação: não há caminho de privilégio por
	// catálogo (V2.1 §11.3 / §5 estágio 8).
	if !manifest.Trusted {
		return SkillActivation{}, fmt.Errorf("%w: %s", ErrSkillNotTrusted, activation.SkillID)
	}
	activation.OrganizationID = organizationID
	activation.ActivatedAt = time.Now().UTC()
	if s.activations == nil {
		s.activations = map[string]map[string]SkillActivation{}
	}
	if s.activations[organizationID] == nil {
		s.activations[organizationID] = map[string]SkillActivation{}
	}
	s.activations[organizationID][activation.Key()] = activation
	if err := s.persistActivationsLocked(organizationID); err != nil {
		delete(s.activations[organizationID], activation.Key())
		return SkillActivation{}, err
	}
	return activation, nil
}

// DeactivateSkillForOrganization remove uma ativação pelo identificador estável.
func (s *ContextStore) DeactivateSkillForOrganization(organizationID, key string) error {
	organizationID = strings.TrimSpace(organizationID)
	key = strings.TrimSpace(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	byKey, ok := s.activations[organizationID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSkillActivationNotFound, key)
	}
	previous, ok := byKey[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSkillActivationNotFound, key)
	}
	delete(byKey, key)
	if err := s.persistActivationsLocked(organizationID); err != nil {
		byKey[key] = previous
		return err
	}
	return nil
}

// SkillActivationsForOrganization devolve as ativações persistidas.
func (s *ContextStore) SkillActivationsForOrganization(organizationID string) []SkillActivation {
	organizationID = strings.TrimSpace(organizationID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]SkillActivation, 0, len(s.activations[organizationID]))
	for _, activation := range s.activations[organizationID] {
		result = append(result, activation)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key() < result[j].Key() })
	return result
}

// SelectSkillsForMission monta o catálogo da missão: apenas skills ATIVAS para o
// contexto, CONFIÁVEIS e HABILITADAS, em ordem determinística, respeitando o
// orçamento de tokens e registrando o motivo de cada exclusão.
func (s *ContextStore) SelectSkillsForMission(organizationID string, context SkillContext, tokenBudget int) (SkillSelection, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return SkillSelection{}, fmt.Errorf("%w: organization is required", ErrSkillActivationInvalid)
	}
	// `Selected` nasce VAZIO, nao nil: slice nil serializa como `null` em JSON e o cliente
	// faz `.map()` sobre ele (era o `TypeError: Cannot read properties of null (reading 'map')`
	// na pagina /skills). Um contrato com listas devolve listas, mesmo vazias.
	selection := SkillSelection{Selected: []ActiveSkill{}, TokenBudget: tokenBudget, BudgetDefined: tokenBudget > 0}

	type candidate struct {
		manifest   SkillManifest
		activation SkillActivation
	}
	candidates := []candidate{}
	s.mu.RLock()
	for _, activation := range s.activations[organizationID] {
		if !context.matches(activation) {
			continue
		}
		manifest, err := s.skillLocked(organizationID, activation.SkillID)
		if err != nil {
			s.mu.RUnlock()
			return SkillSelection{}, err
		}
		candidates = append(candidates, candidate{manifest: manifest, activation: activation})
	}
	s.mu.RUnlock()

	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i].manifest, candidates[j].manifest
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		return left.ID < right.ID
	})

	for _, item := range candidates {
		manifest := item.manifest
		switch {
		case !manifest.Trusted:
			selection.Skipped = append(selection.Skipped, SkillSkip{ID: manifest.ID, Reason: "skill não é confiável"})
			continue
		case !manifest.Enabled:
			selection.Skipped = append(selection.Skipped, SkillSkip{ID: manifest.ID, Reason: "skill desabilitada"})
			continue
		}
		if selection.BudgetDefined && manifest.TokenBudget > 0 && selection.UsedTokens+manifest.TokenBudget > tokenBudget {
			selection.Skipped = append(selection.Skipped, SkillSkip{
				ID:     manifest.ID,
				Reason: fmt.Sprintf("orçamento insuficiente (%d tokens restantes, custo %d)", tokenBudget-selection.UsedTokens, manifest.TokenBudget),
			})
			continue
		}
		if selection.BudgetDefined {
			selection.UsedTokens += manifest.TokenBudget
		}
		selection.Selected = append(selection.Selected, ActiveSkill{
			ID:          manifest.ID,
			Version:     manifest.Version,
			Description: manifest.Description,
			Category:    manifest.Category,
			License:     manifest.License,
			Homepage:    manifest.Homepage,
			Scopes:      append([]string(nil), manifest.Scopes...),
			Tools:       append([]string(nil), manifest.Tools...),
			TokenBudget: manifest.TokenBudget,
			Activation:  item.activation,
		})
	}
	return selection, nil
}

func (s *ContextStore) skillLocked(organizationID, id string) (SkillManifest, error) {
	for key, manifest := range s.skills {
		if !strings.EqualFold(strings.TrimSpace(manifest.ID), id) {
			continue
		}
		if manifest.OrganizationID != "" && manifest.OrganizationID != organizationID {
			continue
		}
		_ = key
		return manifest, nil
	}
	return SkillManifest{}, fmt.Errorf("%w: skill %s", ErrSkillNotFound, id)
}

// ---------- persistência ----------

func (s *ContextStore) persistActivationsLocked(organizationID string) error {
	if s.root == "" {
		return nil
	}
	dir := filepath.Join(s.root, "skill-activations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	byKey := s.activations[organizationID]
	payload := make([]SkillActivation, 0, len(byKey))
	for _, activation := range byKey {
		payload = append(payload, activation)
	}
	sort.Slice(payload, func(i, j int) bool { return payload[i].Key() < payload[j].Key() })
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	target := filepath.Join(dir, organizationID+".json")
	temp := target + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, target)
}

func (s *ContextStore) loadSkillActivations(root string) error {
	dir := filepath.Join(root, "skill-activations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		organizationID := strings.TrimSuffix(entry.Name(), ".json")
		var payload []SkillActivation
		if err := readJSON(filepath.Join(dir, entry.Name()), &payload); err != nil {
			return fmt.Errorf("skill activation %s: %w", entry.Name(), err)
		}
		for _, activation := range payload {
			activation.OrganizationID = organizationID
			if s.activations == nil {
				s.activations = map[string]map[string]SkillActivation{}
			}
			if s.activations[organizationID] == nil {
				s.activations[organizationID] = map[string]SkillActivation{}
			}
			s.activations[organizationID][activation.Key()] = activation
		}
	}
	return nil
}
