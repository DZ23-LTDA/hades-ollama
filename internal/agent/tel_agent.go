package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxTelAgentMessageBytes        = 2048
	maxTelAgentTitleBytes          = 256
	maxTelAgentDetailBytes         = 2048
	maxTelAgentHistory             = 100
	maxTelAgentIdempotencyKeyBytes = 128
)

type TelAgentRequest struct {
	Message          string `json:"message"`
	Operation        string `json:"operation"`
	Title            string `json:"title,omitempty"`
	Description      string `json:"description,omitempty"`
	Priority         int    `json:"priority,omitempty"`
	DailyBudgetCents int64  `json:"daily_budget_cents,omitempty"`
	IdempotencyKey   string `json:"-"`
}

type TelAgentExchange struct {
	ID                string    `json:"id"`
	Channel           string    `json:"channel"`
	ActorID           string    `json:"actor_id"`
	OrganizationID    string    `json:"organization_id"`
	Message           string    `json:"message"`
	Operation         string    `json:"operation"`
	Status            string    `json:"status"`
	Reply             string    `json:"reply"`
	CreatedResourceID string    `json:"created_resource_id,omitempty"`
	ApprovalRequired  bool      `json:"approval_required"`
	CreatedAt         time.Time `json:"created_at"`
}

type TelAgentResult struct {
	Exchange TelAgentExchange `json:"exchange"`
	Report   *CompanyReport   `json:"report,omitempty"`
}

var (
	ErrTelAgentOrganizationMismatch  = errors.New("tel-agent organization mismatch")
	ErrTelAgentActorRequired         = errors.New("tel-agent actor is required")
	ErrTelAgentMessageRequired       = errors.New("tel-agent message is required")
	ErrTelAgentMessageTooLong        = errors.New("tel-agent message exceeds 2048 bytes")
	ErrTelAgentUnsupportedOperation  = errors.New("tel-agent operation is not allowlisted")
	ErrTelAgentIdempotencyKeyTooLong = errors.New("tel-agent idempotency key exceeds 128 bytes")
)

func validateTelAgentText(value, field string, maxBytes int) (string, error) {
	value = strings.TrimSpace(RedactDLP(value))
	if value == "" {
		return "", fmt.Errorf("tel-agent %s is required", field)
	}
	if len([]byte(value)) > maxBytes {
		return "", fmt.Errorf("tel-agent %s exceeds %d bytes", field, maxBytes)
	}
	return value, nil
}

func companyReportSnapshot(company Company) CompanyReport {
	report := CompanyReport{Company: company}
	for _, item := range company.Backlog {
		if item.Status == "done" || item.Status == "completed" {
			report.CompletedBacklog++
		} else {
			report.OpenBacklog++
		}
	}
	for _, goal := range company.Goals {
		if goal.Status == "at_risk" || goal.Status == "blocked" {
			report.GoalsAtRisk++
		} else {
			report.GoalsOnTrack++
		}
	}
	for _, cycle := range company.Cycles {
		if cycle.Enabled {
			report.EnabledCycles++
		}
	}
	if company.Budget.MonthlyLimitCents > 0 {
		report.BudgetUtilizationPct = float64(company.Budget.SpentCents) / float64(company.Budget.MonthlyLimitCents) * 100
	}
	return report
}

func (s *CompanyStore) ExecuteTelAgent(id, organizationID, actorID string, request TelAgentRequest) (Company, TelAgentResult, error) {
	organizationID = strings.TrimSpace(organizationID)
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return Company{}, TelAgentResult{}, ErrTelAgentActorRequired
	}
	message, err := validateTelAgentText(request.Message, "message", maxTelAgentMessageBytes)
	if err != nil {
		if strings.Contains(err.Error(), "required") {
			return Company{}, TelAgentResult{}, ErrTelAgentMessageRequired
		}
		return Company{}, TelAgentResult{}, ErrTelAgentMessageTooLong
	}
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
	if operation != "report.read" && operation != "backlog.create" && operation != "campaign.draft" {
		return Company{}, TelAgentResult{}, ErrTelAgentUnsupportedOperation
	}
	title := strings.TrimSpace(RedactDLP(request.Title))
	description := strings.TrimSpace(RedactDLP(request.Description))
	if len([]byte(title)) > maxTelAgentTitleBytes || len([]byte(description)) > maxTelAgentDetailBytes {
		return Company{}, TelAgentResult{}, errors.New("tel-agent operation details exceed the allowed size")
	}
	if operation != "report.read" && title == "" {
		return Company{}, TelAgentResult{}, errors.New("tel-agent title is required for this operation")
	}
	if request.Priority < 0 || request.Priority > 1000 || request.DailyBudgetCents < 0 {
		return Company{}, TelAgentResult{}, errors.New("tel-agent operation values are outside the allowed range")
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if len([]byte(idempotencyKey)) > maxTelAgentIdempotencyKeyBytes {
		return Company{}, TelAgentResult{}, ErrTelAgentIdempotencyKeyTooLong
	}
	fingerprintInput := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", operation, message, title, request.Priority, request.DailyBudgetCents)
	fingerprint := companyIdempotencyDigest("company.tel_agent.input", fingerprintInput)
	idempotencyDigest := ""
	if idempotencyKey != "" {
		idempotencyDigest = companyIdempotencyDigest("company.tel_agent", idempotencyKey)
	}

	var result TelAgentResult
	updated, err := s.mutate(id, func(company *Company) error {
		if strings.TrimSpace(company.OrganizationID) != organizationID {
			return ErrTelAgentOrganizationMismatch
		}
		if err := checkCompanyIdempotency(company, "company.tel_agent", idempotencyKey, fingerprint); err != nil {
			if !errors.Is(err, ErrCompanyIdempotentReplay) || idempotencyDigest == "" {
				return err
			}
			var resultID string
			for index := len(company.Idempotency) - 1; index >= 0; index-- {
				record := company.Idempotency[index]
				if record.Operation == "company.tel_agent" && record.Digest == idempotencyDigest {
					resultID = record.ResultID
					break
				}
			}
			for index := len(company.TelAgentHistory) - 1; index >= 0; index-- {
				previous := company.TelAgentHistory[index]
				if resultID == "" || previous.ID != resultID {
					continue
				}
				result.Exchange = previous
				if operation == "report.read" {
					report := companyReportSnapshot(*company)
					result.Report = &report
				}
				return nil
			}
			return err
		}
		now := time.Now().UTC()
		exchange := TelAgentExchange{
			ID:             "telx_" + uuid.NewString(),
			Channel:        "tel-agent.text",
			ActorID:        actorID,
			OrganizationID: company.OrganizationID,
			Message:        message,
			Operation:      operation,
			Status:         "completed",
			CreatedAt:      now,
		}
		switch operation {
		case "report.read":
			report := companyReportSnapshot(*company)
			result.Report = &report
			exchange.Reply = fmt.Sprintf("Empresa %s: %d itens abertos, %d ciclos ativos e budget em %.1f%%.", company.Name, report.OpenBacklog, report.EnabledCycles, report.BudgetUtilizationPct)
		case "backlog.create":
			item := CompanyBacklogItem{ID: "task_" + uuid.NewString(), Title: title, Description: description, Priority: request.Priority, Status: "planned", Source: "tel-agent", CreatedAt: now, UpdatedAt: now}
			if item.Priority <= 0 {
				item.Priority = 50
			}
			company.Backlog = append(company.Backlog, item)
			sort.SliceStable(company.Backlog, func(i, j int) bool { return company.Backlog[i].Priority < company.Backlog[j].Priority })
			exchange.CreatedResourceID = item.ID
			exchange.Reply = fmt.Sprintf("Backlog criado: %s (prioridade %d).", item.Title, item.Priority)
		case "campaign.draft":
			campaign := CompanyCampaign{ID: "cmp_" + uuid.NewString(), Name: title, Channel: "content", Objective: description, Mode: "sandbox", Status: "draft", DailyBudgetCents: request.DailyBudgetCents, ApprovalRequired: true, CreatedAt: now, UpdatedAt: now}
			if campaign.Objective == "" {
				campaign.Objective = message
			}
			company.Campaigns = append(company.Campaigns, campaign)
			queueCompanyApproval(company, "campaign", campaign.ID, "campaign:external", now)
			exchange.CreatedResourceID = campaign.ID
			exchange.ApprovalRequired = true
			exchange.Status = "approval_pending"
			exchange.Reply = fmt.Sprintf("Rascunho de campanha criado em sandbox: %s. Approval obrigatório antes de qualquer ação externa.", campaign.Name)
		}
		company.TelAgentHistory = append(company.TelAgentHistory, exchange)
		rememberCompanyIdempotency(company, "company.tel_agent", idempotencyKey, fingerprint)
		if idempotencyDigest != "" {
			for index := len(company.Idempotency) - 1; index >= 0; index-- {
				if company.Idempotency[index].Operation == "company.tel_agent" && company.Idempotency[index].Digest == idempotencyDigest {
					company.Idempotency[index].ResultID = exchange.ID
					break
				}
			}
		}
		if len(company.TelAgentHistory) > maxTelAgentHistory {
			company.TelAgentHistory = append([]TelAgentExchange(nil), company.TelAgentHistory[len(company.TelAgentHistory)-maxTelAgentHistory:]...)
		}
		result.Exchange = exchange
		return nil
	})
	if err != nil {
		return Company{}, TelAgentResult{}, err
	}
	return updated, result, nil
}

func (s *CompanyStore) TelAgentHistoryForOrganization(id, organizationID string, limit int) ([]TelAgentExchange, error) {
	// NÃO INFERÊNCIA (E-189, achado B-15): "recurso de outro tenant" e "recurso inexistente" precisam devolver a
	// MESMA negação. Antes, o primeiro dava `ErrTelAgentOrganizationMismatch` e o segundo o erro de "não
	// encontrado" — a diferença é um ORÁCULO DE ENUMERAÇÃO: quem não é dono distingue, pela resposta, se o ID
	// existe em algum lugar, sem acessar dado nenhum. A razão real continua disponível no LOG do servidor (onde só o
	// operador lê), não na resposta ao tenant.
	company, err := s.Get(id)
	if err != nil {
		return nil, ErrTelAgentOrganizationMismatch
	}
	if strings.TrimSpace(company.OrganizationID) != strings.TrimSpace(organizationID) {
		return nil, ErrTelAgentOrganizationMismatch
	}
	if limit <= 0 || limit > maxTelAgentHistory {
		limit = maxTelAgentHistory
	}
	if len(company.TelAgentHistory) > limit {
		return append([]TelAgentExchange(nil), company.TelAgentHistory[len(company.TelAgentHistory)-limit:]...), nil
	}
	return append([]TelAgentExchange(nil), company.TelAgentHistory...), nil
}
