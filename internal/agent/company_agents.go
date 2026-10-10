package agent

import (
	"errors"
	"strings"
	"time"
)

type CompanyAgent struct {
	ID               string             `json:"id"`
	DepartmentID     string             `json:"department_id"`
	Name             string             `json:"name"`
	Objective        string             `json:"objective"`
	BudgetCents      int64              `json:"budget_cents"`
	SpentCents       int64              `json:"spent_cents"`
	AllowedTools     []string           `json:"allowed_tools,omitempty"`
	MemoryScope      string             `json:"memory_scope"`
	Metrics          map[string]float64 `json:"metrics,omitempty"`
	SLA              string             `json:"sla,omitempty"`
	SupervisorID     string             `json:"supervisor_id,omitempty"`
	Autonomy         string             `json:"autonomy"`
	PauseCondition   string             `json:"pause_condition,omitempty"`
	ApprovalRequired []string           `json:"approval_required,omitempty"`
	Status           string             `json:"status"`
	PausedReason     string             `json:"paused_reason,omitempty"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

var (
	ErrCompanyAgentNotFound       = errors.New("company agent not found")
	ErrCompanyAgentBudgetExceeded = errors.New("company agent budget limit exceeded")
)

func defaultCompanyAgents() []CompanyAgent {
	return []CompanyAgent{
		{ID: "ceo", DepartmentID: "ceo", Name: "CEO Agent", Objective: "Priorizar estratégia e trade-offs", AllowedTools: []string{"workspace:read", "workspace:write", "company:read"}, MemoryScope: "company", SLA: "weekly", Autonomy: "assistido", PauseCondition: "budget_or_anomaly", ApprovalRequired: []string{"external_message", "spend"}, Status: "active"},
		{ID: "product", DepartmentID: "product", Name: "Product Agent", Objective: "Descobrir necessidades e manter roadmap", AllowedTools: []string{"workspace:read", "workspace:write", "company:read"}, MemoryScope: "department:product", SLA: "weekly", Autonomy: "assistido", Status: "active"},
		{ID: "engineering", DepartmentID: "engineering", Name: "Engineering Agent", Objective: "Construir, testar e operar software", AllowedTools: []string{"workspace:read", "workspace:write", "sandbox:execute"}, MemoryScope: "department:engineering", SLA: "daily", Autonomy: "assistido", ApprovalRequired: []string{"deploy", "external_connector"}, Status: "active"},
		{ID: "marketing", DepartmentID: "marketing", Name: "Marketing Agent", Objective: "Criar conteúdo e aquisição mensurável", AllowedTools: []string{"workspace:read", "workspace:write", "company:read"}, MemoryScope: "department:marketing", SLA: "daily", Autonomy: "assistido", PauseCondition: "anomaly", ApprovalRequired: []string{"ad", "publish"}, Status: "active"},
		{ID: "sales", DepartmentID: "sales", Name: "Sales Agent", Objective: "Qualificar leads e manter pipeline", AllowedTools: []string{"workspace:read", "company:read"}, MemoryScope: "department:sales", SLA: "daily", Autonomy: "assistido", ApprovalRequired: []string{"external_message"}, Status: "active"},
		{ID: "support", DepartmentID: "support", Name: "Support Agent", Objective: "Responder clientes e manter conhecimento", AllowedTools: []string{"workspace:read", "company:read"}, MemoryScope: "department:support", SLA: "4h", Autonomy: "assistido", Status: "active"},
		{ID: "finance", DepartmentID: "operations", Name: "Finance Agent", Objective: "Controlar orçamento, margem e reconciliação", AllowedTools: []string{"workspace:read", "company:read"}, MemoryScope: "department:operations", SLA: "daily", Autonomy: "assistido", PauseCondition: "budget_or_anomaly", ApprovalRequired: []string{"spend", "contract"}, Status: "active"},
	}
}

func (s *CompanyStore) ListAgents(id string) ([]CompanyAgent, error) {
	company, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return append([]CompanyAgent(nil), company.Agents...), nil
}

func (s *CompanyStore) PauseAgent(id, agentID, reason string) (Company, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Company{}, errors.New("agent pause reason is required")
	}
	return s.mutate(id, func(company *Company) error {
		for index := range company.Agents {
			if company.Agents[index].ID == agentID {
				company.Agents[index].Status = "paused"
				company.Agents[index].PausedReason = reason
				company.Agents[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyAgentNotFound
	})
}

func (s *CompanyStore) ResumeAgent(id, agentID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Agents {
			if company.Agents[index].ID == agentID {
				company.Agents[index].Status = "active"
				company.Agents[index].PausedReason = ""
				company.Agents[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyAgentNotFound
	})
}

func (s *CompanyStore) RecordAgentSpend(id, agentID string, amountCents int64) (Company, error) {
	if amountCents <= 0 {
		return Company{}, errors.New("agent spend must be positive")
	}
	return s.mutate(id, func(company *Company) error {
		if company.Status == CompanyPaused || company.Risk.Paused {
			return ErrCompanyPaused
		}
		for index := range company.Agents {
			agent := &company.Agents[index]
			if agent.ID == agentID {
				if agent.Status == "paused" {
					return ErrCompanyPaused
				}
				if agent.BudgetCents > 0 && (agent.SpentCents > agent.BudgetCents || amountCents > agent.BudgetCents-agent.SpentCents) {
					return ErrCompanyAgentBudgetExceeded
				}
				agent.SpentCents += amountCents
				agent.UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyAgentNotFound
	})
}
