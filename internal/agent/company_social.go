package agent

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CompanySocialAccount struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"`
	Name            string    `json:"name"`
	ExternalAccount string    `json:"external_account,omitempty"`
	Status          string    `json:"status"`
	OAuthRequired   bool      `json:"oauth_required"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompanySocialDraft struct {
	ID               string     `json:"id"`
	Provider         string     `json:"provider"`
	AccountID        string     `json:"account_id"`
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	MediaURLs        []string   `json:"media_urls,omitempty"`
	Status           string     `json:"status"`
	ApprovalRequired bool       `json:"approval_required"`
	Approved         bool       `json:"approved"`
	Mode             string     `json:"mode"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CompanySocialMetric struct {
	ID          string    `json:"id"`
	DraftID     string    `json:"draft_id"`
	Provider    string    `json:"provider"`
	Impressions int64     `json:"impressions"`
	Clicks      int64     `json:"clicks"`
	Conversions int64     `json:"conversions"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type CompanySocialReport struct {
	Company           Company `json:"company"`
	ConnectedAccounts int     `json:"connected_accounts"`
	PendingOAuth      int     `json:"pending_oauth"`
	Drafts            int     `json:"drafts"`
	ApprovedDrafts    int     `json:"approved_drafts"`
	PublishedSandbox  int     `json:"published_sandbox"`
	Impressions       int64   `json:"impressions"`
	Clicks            int64   `json:"clicks"`
	Conversions       int64   `json:"conversions"`
}

var (
	ErrCompanySocialNotFound            = errors.New("company social resource not found")
	ErrCompanySocialProviderUnsupported = errors.New("social provider is not supported")
	ErrCompanySocialOAuthRequired       = errors.New("social account requires OAuth authorization")
	ErrCompanySocialExternalUnavailable = errors.New("external social publishing is not configured")
)

var supportedSocialProviders = map[string]struct{}{
	"x": {}, "instagram": {}, "facebook": {}, "youtube": {}, "tiktok": {}, "linkedin": {}, "pinterest": {}, "telegram": {}, "discord": {}, "whatsapp": {},
}

// PUBLICAÇÃO SOCIAL: EFEITO UMA VEZ E SEM ENXURRADA (E-302) — estágio 36.
//
// Duas guardas que faltavam, ambas sobre o MESMO risco: publicar o que não devia, ou publicar de novo.
//
//  1. **EFEITO UMA VEZ**: republicar um rascunho já publicado é postar DUAS vezes na rede social. O §11.7 exige que
//     "o mesmo efeito externo não ocorra duas vezes" — o segundo `PublishSocialDraft` é recusado;
//  2. **SEM ENXURRADA**: publicar sem teto é o risco "spam" que o estágio 36 nomeia. O teto é por **janela** e a
//     recusa é **explícita** (`ErrCompanySocialPublishThrottled`) — nunca silenciosa.
//
// LIMITE DECLARADO: isto é **política local do Hades**. **Não** é o rate limit da API da rede social, que só se mede
// com conta real e continua `BLOCKED_BY_EXTERNAL_DEPENDENCY`. Não se finge que é o mesmo.
const (
	// CompanySocialPublishLimitPerWindow é o teto de publicações por janela (por empresa).
	CompanySocialPublishLimitPerWindow = 5
	// CompanySocialPublishWindow é a janela do teto.
	CompanySocialPublishWindow = 10 * time.Minute
)

var (
	// ErrCompanySocialAlreadyPublished recusa o SEGUNDO publish do mesmo rascunho (efeito externo uma vez).
	ErrCompanySocialAlreadyPublished = errors.New("social draft already published")
	// ErrCompanySocialPublishThrottled recusa a enxurrada, com motivo explícito em vez de silêncio.
	ErrCompanySocialPublishThrottled = errors.New("social publish throttled by local policy")
	// ErrCompanySocialMetricNotPublished recusa métrica de conteúdo que NUNCA foi publicado: número sem causa
	// é métrica inventada (E-307).
	ErrCompanySocialMetricNotPublished = errors.New("social metric requires published content")
)

func validateSocialProvider(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := supportedSocialProviders[value]; !ok {
		return ""
	}
	return value
}

func (s *CompanyStore) AddSocialAccount(id string, account CompanySocialAccount) (Company, error) {
	account.Provider = validateSocialProvider(account.Provider)
	account.Name = strings.TrimSpace(account.Name)
	if account.Provider == "" {
		return Company{}, ErrCompanySocialProviderUnsupported
	}
	if account.Name == "" {
		return Company{}, errors.New("social account name is required")
	}
	account.ID = "soc_" + uuid.NewString()
	account.Status = "pending_oauth"
	account.OAuthRequired = true
	account.CreatedAt = time.Now().UTC()
	account.UpdatedAt = account.CreatedAt
	return s.mutate(id, func(company *Company) error {
		company.SocialAccounts = append(company.SocialAccounts, account)
		return nil
	})
}

func (s *CompanyStore) CreateSocialDraft(id string, draft CompanySocialDraft) (Company, error) {
	draft.Provider = validateSocialProvider(draft.Provider)
	draft.Title = strings.TrimSpace(draft.Title)
	draft.Body = strings.TrimSpace(draft.Body)
	draft.Mode = strings.ToLower(strings.TrimSpace(draft.Mode))
	if draft.Provider == "" {
		return Company{}, ErrCompanySocialProviderUnsupported
	}
	if draft.Title == "" || draft.Body == "" {
		return Company{}, errors.New("social draft title and body are required")
	}
	if draft.Mode == "" {
		draft.Mode = "sandbox"
	}
	if draft.Mode != "sandbox" && draft.Mode != "external" {
		return Company{}, errors.New("social draft mode must be sandbox or external")
	}
	draft.ID = "sdraft_" + uuid.NewString()
	draft.Status = "draft"
	draft.ApprovalRequired = true
	draft.Approved = false
	draft.CreatedAt = time.Now().UTC()
	draft.UpdatedAt = draft.CreatedAt
	return s.mutate(id, func(company *Company) error {
		if draft.AccountID != "" {
			found := false
			for _, account := range company.SocialAccounts {
				if account.ID == draft.AccountID && account.Provider == draft.Provider {
					found = true
					if account.Status != "connected" {
						return ErrCompanySocialOAuthRequired
					}
				}
			}
			if !found {
				return ErrCompanySocialNotFound
			}
		}
		company.SocialDrafts = append(company.SocialDrafts, draft)
		queueCompanyApproval(company, "social_draft", draft.ID, "social:publish", draft.CreatedAt)
		return nil
	})
}

func (s *CompanyStore) ApproveSocialDraft(id, draftID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.SocialDrafts {
			if company.SocialDrafts[index].ID == draftID {
				company.SocialDrafts[index].Approved = true
				company.SocialDrafts[index].Status = "approved"
				company.SocialDrafts[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanySocialNotFound
	})
}

func (s *CompanyStore) PublishSocialDraft(id, draftID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.SocialDrafts {
			draft := &company.SocialDrafts[index]
			if draft.ID != draftID {
				continue
			}
			if !draft.Approved {
				return ErrCompanyApprovalRequiredForExternal
			}
			// EFEITO UMA VEZ (E-302): republicar o mesmo rascunho postaria de novo na rede social.
			if draft.PublishedAt != nil || strings.HasPrefix(strings.ToLower(draft.Status), "published") {
				return ErrCompanySocialAlreadyPublished
			}
			// SEM ENXURRADA (E-302): conta as publicações DENTRO da janela; passou do teto, recusa com motivo.
			limite := time.Now().UTC().Add(-CompanySocialPublishWindow)
			publicadosNaJanela := 0
			for _, outro := range company.SocialDrafts {
				if outro.PublishedAt != nil && outro.PublishedAt.After(limite) {
					publicadosNaJanela++
				}
			}
			if publicadosNaJanela >= CompanySocialPublishLimitPerWindow {
				return ErrCompanySocialPublishThrottled
			}
			if draft.Mode != "sandbox" {
				return ErrCompanySocialExternalUnavailable
			}
			now := time.Now().UTC()
			draft.Status = "published_sandbox"
			draft.PublishedAt = &now
			draft.UpdatedAt = now
			return nil
		}
		return ErrCompanySocialNotFound
	})
}

func (s *CompanyStore) RecordSocialMetric(id string, metric CompanySocialMetric) (Company, error) {
	return s.RecordSocialMetricWithIdempotency(id, metric, "")
}

func (s *CompanyStore) RecordSocialMetricWithIdempotency(id string, metric CompanySocialMetric, idempotencyKey string) (Company, error) {
	if metric.DraftID == "" || metric.Impressions < 0 || metric.Clicks < 0 || metric.Conversions < 0 {
		return Company{}, errors.New("social metric draft and non-negative counters are required")
	}
	metric.Provider = validateSocialProvider(metric.Provider)
	if metric.Provider == "" {
		return Company{}, ErrCompanySocialProviderUnsupported
	}
	fingerprint := metric.DraftID + ":" + metric.Provider + ":" + strconv.FormatInt(metric.Impressions, 10) + ":" + strconv.FormatInt(metric.Clicks, 10) + ":" + strconv.FormatInt(metric.Conversions, 10)
	metric.ID = "smetric_" + uuid.NewString()
	metric.RecordedAt = time.Now().UTC()
	return s.mutate(id, func(company *Company) error {
		if err := checkCompanyIdempotency(company, "company.social.metric", idempotencyKey, fingerprint); err != nil {
			return err
		}
		found := false
		for _, draft := range company.SocialDrafts {
			if draft.ID != metric.DraftID {
				continue
			}
			// MÉTRICA SÓ EXISTE PARA CONTEÚDO PUBLICADO (E-307). MEDIDO: antes aceitava métrica de rascunho nunca
			// publicado, e o relatório somava impressões/cliques/conversões sem causa. Número sem causa é métrica
			// inventada — e "não inventar métricas" é exigência explícita da missão.
			if draft.PublishedAt == nil && !strings.HasPrefix(strings.ToLower(draft.Status), "published") {
				return fmt.Errorf("%w: %s", ErrCompanySocialMetricNotPublished, draft.ID)
			}
			found = true
			break
		}
		if !found {
			return ErrCompanySocialNotFound
		}
		company.SocialMetrics = append(company.SocialMetrics, metric)
		rememberCompanyIdempotency(company, "company.social.metric", idempotencyKey, fingerprint)
		return nil
	})
}

func (s *CompanyStore) SocialReport(id string) (CompanySocialReport, error) {
	company, err := s.Get(id)
	if err != nil {
		return CompanySocialReport{}, err
	}
	report := CompanySocialReport{Company: company, Drafts: len(company.SocialDrafts)}
	for _, account := range company.SocialAccounts {
		if account.Status == "connected" {
			report.ConnectedAccounts++
		} else if account.Status == "pending_oauth" {
			report.PendingOAuth++
		}
	}
	for _, draft := range company.SocialDrafts {
		if draft.Approved {
			report.ApprovedDrafts++
		}
		if draft.Status == "published_sandbox" {
			report.PublishedSandbox++
		}
	}
	for _, metric := range company.SocialMetrics {
		report.Impressions += metric.Impressions
		report.Clicks += metric.Clicks
		report.Conversions += metric.Conversions
	}
	return report, nil
}
