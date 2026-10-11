package agent

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CompanyCampaign struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Channel          string    `json:"channel"`
	Objective        string    `json:"objective"`
	Mode             string    `json:"mode"`
	Status           string    `json:"status"`
	DailyBudgetCents int64     `json:"daily_budget_cents"`
	ApprovalRequired bool      `json:"approval_required"`
	Approved         bool      `json:"approved"`
	Content          []string  `json:"content,omitempty"`
	Impressions      int64     `json:"impressions"`
	Clicks           int64     `json:"clicks"`
	Conversions      int64     `json:"conversions"`
	SpendCents       int64     `json:"spend_cents"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	// ScheduledAt é preenchido pelo agendamento aprovado (estágio 26).
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

type CompanyAffiliateProgram struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Network          string    `json:"network"`
	Mode             string    `json:"mode"`
	Status           string    `json:"status"`
	CommissionBps    int64     `json:"commission_bps"`
	ApprovalRequired bool      `json:"approval_required"`
	Approved         bool      `json:"approved"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CompanyAffiliateLink struct {
	ID           string    `json:"id"`
	ProgramID    string    `json:"program_id"`
	ProductID    string    `json:"product_id,omitempty"`
	Destination  string    `json:"destination"`
	Mode         string    `json:"mode"`
	Clicks       int64     `json:"clicks"`
	Conversions  int64     `json:"conversions"`
	RevenueCents int64     `json:"revenue_cents"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CompanyProduct struct {
	ID         string    `json:"id"`
	SKU        string    `json:"sku"`
	Name       string    `json:"name"`
	Supplier   string    `json:"supplier"`
	CostCents  int64     `json:"cost_cents"`
	PriceCents int64     `json:"price_cents"`
	Inventory  int64     `json:"inventory"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CompanyOrder struct {
	ID             string    `json:"id"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	ProductID      string    `json:"product_id"`
	CustomerRef    string    `json:"customer_ref"`
	Quantity       int64     `json:"quantity"`
	TotalCents     int64     `json:"total_cents"`
	Mode           string    `json:"mode"`
	Status         string    `json:"status"`
	Approved       bool      `json:"approved"`
	TrackingCode   string    `json:"tracking_code,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CompanyGrowthReport struct {
	Company              Company `json:"company"`
	SandboxOnly          bool    `json:"sandbox_only"`
	CampaignsTotal       int     `json:"campaigns_total"`
	CampaignsActive      int     `json:"campaigns_active"`
	AffiliatePrograms    int     `json:"affiliate_programs"`
	AffiliateConversions int64   `json:"affiliate_conversions"`
	Products             int     `json:"products"`
	PendingOrders        int     `json:"pending_orders"`
	FulfilledOrders      int     `json:"fulfilled_orders"`
	RevenueCents         int64   `json:"revenue_cents"`
}

var (
	ErrCompanyCampaignNotFound            = errors.New("company campaign not found")
	ErrCompanyAffiliateNotFound           = errors.New("company affiliate resource not found")
	ErrCompanyProductNotFound             = errors.New("company product not found")
	ErrCompanyOrderNotFound               = errors.New("company order not found")
	ErrCompanyApprovalRequiredForExternal = errors.New("approval is required before external campaign or order action")
	ErrCompanyGrowthExternalUnavailable   = errors.New("external growth action is not configured")
	ErrCompanyInvalidExternalURL          = errors.New("external destination must use HTTPS")
)

func normalizeCompanyGrowthModes(company *Company) {
	for index := range company.Campaigns {
		if strings.TrimSpace(company.Campaigns[index].Mode) == "" {
			company.Campaigns[index].Mode = "sandbox"
		}
	}
	for index := range company.AffiliatePrograms {
		if strings.TrimSpace(company.AffiliatePrograms[index].Mode) == "" {
			company.AffiliatePrograms[index].Mode = "sandbox"
		}
	}
	for index := range company.AffiliateLinks {
		if strings.TrimSpace(company.AffiliateLinks[index].Mode) == "" {
			company.AffiliateLinks[index].Mode = "sandbox"
		}
	}
	for index := range company.Orders {
		if strings.TrimSpace(company.Orders[index].Mode) == "" {
			company.Orders[index].Mode = "sandbox"
		}
	}
}

func validateGrowthURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ErrCompanyInvalidExternalURL
	}
	return nil
}

func (s *CompanyStore) AddCampaign(id string, campaign CompanyCampaign) (Company, error) {
	campaign.Name = strings.TrimSpace(campaign.Name)
	campaign.Channel = strings.ToLower(strings.TrimSpace(campaign.Channel))
	campaign.Objective = strings.TrimSpace(campaign.Objective)
	if campaign.Name == "" || campaign.Objective == "" {
		return Company{}, errors.New("campaign name and objective are required")
	}
	if campaign.Channel == "" {
		campaign.Channel = "content"
	}
	campaign.Mode = "sandbox"
	if campaign.DailyBudgetCents < 0 {
		return Company{}, errors.New("campaign budget cannot be negative")
	}
	campaign.ID = "cmp_" + uuid.NewString()
	campaign.Status = "draft"
	campaign.ApprovalRequired = true
	campaign.Approved = false
	campaign.CreatedAt = time.Now().UTC()
	campaign.UpdatedAt = campaign.CreatedAt
	return s.mutate(id, func(company *Company) error {
		company.Campaigns = append(company.Campaigns, campaign)
		queueCompanyApproval(company, "campaign", campaign.ID, "campaign:external", campaign.CreatedAt)
		return nil
	})
}

func (s *CompanyStore) ApproveCampaign(id, campaignID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Campaigns {
			if company.Campaigns[index].ID == campaignID {
				company.Campaigns[index].Approved = true
				company.Campaigns[index].Status = "approved"
				company.Campaigns[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyCampaignNotFound
	})
}

func (s *CompanyStore) LaunchCampaign(id, campaignID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		if company.Status == CompanyPaused || company.Risk.Paused {
			return ErrCompanyPaused
		}
		for index := range company.Campaigns {
			if company.Campaigns[index].ID == campaignID {
				if company.Campaigns[index].Mode != "sandbox" {
					return ErrCompanyGrowthExternalUnavailable
				}
				if !company.Campaigns[index].Approved {
					return ErrCompanyApprovalRequiredForExternal
				}
				company.Campaigns[index].Status = "active"
				company.Campaigns[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyCampaignNotFound
	})
}

func (s *CompanyStore) PauseCampaign(id, campaignID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Campaigns {
			if company.Campaigns[index].ID == campaignID {
				company.Campaigns[index].Status = "paused"
				company.Campaigns[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyCampaignNotFound
	})
}

func (s *CompanyStore) AddAffiliateProgram(id string, program CompanyAffiliateProgram) (Company, error) {
	program.Name = strings.TrimSpace(program.Name)
	program.Network = strings.TrimSpace(program.Network)
	if program.Name == "" || program.Network == "" {
		return Company{}, errors.New("affiliate program name and network are required")
	}
	if program.CommissionBps < 0 || program.CommissionBps > 10000 {
		return Company{}, errors.New("affiliate commission must be between 0 and 10000 basis points")
	}
	program.ID = "aff_" + uuid.NewString()
	program.Mode = "sandbox"
	program.Status = "pending"
	program.ApprovalRequired = true
	program.Approved = false
	program.CreatedAt = time.Now().UTC()
	program.UpdatedAt = program.CreatedAt
	return s.mutate(id, func(company *Company) error {
		company.AffiliatePrograms = append(company.AffiliatePrograms, program)
		queueCompanyApproval(company, "affiliate_program", program.ID, "affiliate:external", program.CreatedAt)
		return nil
	})
}

func (s *CompanyStore) ApproveAffiliateProgram(id, programID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.AffiliatePrograms {
			if company.AffiliatePrograms[index].ID == programID {
				company.AffiliatePrograms[index].Approved = true
				company.AffiliatePrograms[index].Status = "active"
				company.AffiliatePrograms[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyAffiliateNotFound
	})
}

func (s *CompanyStore) AddAffiliateLink(id string, link CompanyAffiliateLink) (Company, error) {
	if strings.TrimSpace(link.ProgramID) == "" || validateGrowthURL(link.Destination) != nil {
		return Company{}, ErrCompanyInvalidExternalURL
	}
	link.Destination = strings.TrimSpace(link.Destination)
	link.Mode = "sandbox"
	link.ID = "alink_" + uuid.NewString()
	link.CreatedAt = time.Now().UTC()
	link.UpdatedAt = link.CreatedAt
	return s.mutate(id, func(company *Company) error {
		for _, program := range company.AffiliatePrograms {
			if program.ID == link.ProgramID {
				if program.Mode != "sandbox" {
					return ErrCompanyGrowthExternalUnavailable
				}
				if !program.Approved {
					return ErrCompanyApprovalRequiredForExternal
				}
				company.AffiliateLinks = append(company.AffiliateLinks, link)
				return nil
			}
		}
		return ErrCompanyAffiliateNotFound
	})
}

func (s *CompanyStore) RecordAffiliateConversion(id, linkID string, revenueCents int64) (Company, error) {
	return s.RecordAffiliateConversionWithIdempotency(id, linkID, revenueCents, "")
}

func (s *CompanyStore) RecordAffiliateConversionWithIdempotency(id, linkID string, revenueCents int64, idempotencyKey string) (Company, error) {
	if revenueCents < 0 {
		return Company{}, errors.New("affiliate revenue cannot be negative")
	}
	fingerprint := strings.TrimSpace(linkID) + ":" + strconv.FormatInt(revenueCents, 10)
	return s.mutate(id, func(company *Company) error {
		if err := checkCompanyIdempotency(company, "company.affiliate.conversion", idempotencyKey, fingerprint); err != nil {
			return err
		}
		for index := range company.AffiliateLinks {
			if company.AffiliateLinks[index].ID == linkID {
				company.AffiliateLinks[index].Conversions++
				company.AffiliateLinks[index].RevenueCents += revenueCents
				company.AffiliateLinks[index].UpdatedAt = time.Now().UTC()
				rememberCompanyIdempotency(company, "company.affiliate.conversion", idempotencyKey, fingerprint)
				return nil
			}
		}
		return ErrCompanyAffiliateNotFound
	})
}

func (s *CompanyStore) AddProduct(id string, product CompanyProduct) (Company, error) {
	product.SKU = strings.TrimSpace(product.SKU)
	product.Name = strings.TrimSpace(product.Name)
	product.Supplier = strings.TrimSpace(product.Supplier)
	if product.SKU == "" || product.Name == "" || product.Supplier == "" {
		return Company{}, errors.New("product sku, name and supplier are required")
	}
	if product.CostCents < 0 || product.PriceCents <= 0 || product.Inventory < 0 {
		return Company{}, errors.New("product cost, price or inventory is invalid")
	}
	return s.mutate(id, func(company *Company) error {
		for _, existing := range company.Products {
			if strings.EqualFold(existing.SKU, product.SKU) {
				return errors.New("product sku already exists")
			}
		}
		product.ID = "prod_" + uuid.NewString()
		product.Status = "active"
		product.CreatedAt = time.Now().UTC()
		product.UpdatedAt = product.CreatedAt
		company.Products = append(company.Products, product)
		return nil
	})
}

func (s *CompanyStore) CreateOrder(id string, order CompanyOrder) (Company, error) {
	order.CustomerRef = strings.TrimSpace(order.CustomerRef)
	if order.ProductID == "" || order.CustomerRef == "" || order.Quantity <= 0 {
		return Company{}, errors.New("order product, customer and positive quantity are required")
	}
	return s.mutate(id, func(company *Company) error {
		if order.IdempotencyKey != "" {
			for _, existing := range company.Orders {
				if existing.IdempotencyKey == order.IdempotencyKey {
					return nil
				}
			}
		}
		for _, product := range company.Products {
			if product.ID == order.ProductID {
				if product.Status != "active" || product.Inventory < order.Quantity {
					return errors.New("product is unavailable for requested quantity")
				}
				if product.PriceCents > 0 && order.Quantity > int64(^uint64(0)>>1)/product.PriceCents {
					return errors.New("order total exceeds supported amount")
				}
				order.ID = "ord_" + uuid.NewString()
				order.TotalCents = product.PriceCents * order.Quantity
				order.Mode = "sandbox"
				order.Status = "pending_approval"
				order.Approved = false
				order.CreatedAt = time.Now().UTC()
				order.UpdatedAt = order.CreatedAt
				company.Orders = append(company.Orders, order)
				queueCompanyApproval(company, "order", order.ID, "order:external", order.CreatedAt)
				return nil
			}
		}
		return ErrCompanyProductNotFound
	})
}

func (s *CompanyStore) ApproveOrder(id, orderID string) (Company, error) {
	return s.mutate(id, func(company *Company) error {
		for index := range company.Orders {
			if company.Orders[index].ID == orderID {
				company.Orders[index].Approved = true
				company.Orders[index].Status = "approved"
				company.Orders[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return ErrCompanyOrderNotFound
	})
}

func (s *CompanyStore) FulfillOrder(id, orderID, trackingCode string) (Company, error) {
	trackingCode = strings.TrimSpace(trackingCode)
	if trackingCode == "" {
		return Company{}, errors.New("tracking code is required for sandbox fulfillment")
	}
	return s.mutate(id, func(company *Company) error {
		for orderIndex := range company.Orders {
			order := &company.Orders[orderIndex]
			if order.ID == orderID {
				if order.Mode != "sandbox" {
					return ErrCompanyGrowthExternalUnavailable
				}
				if order.Status == "fulfilled" {
					return nil
				}
				if !order.Approved {
					return ErrCompanyApprovalRequiredForExternal
				}
				for productIndex := range company.Products {
					product := &company.Products[productIndex]
					if product.ID == order.ProductID {
						if product.Inventory < order.Quantity {
							return errors.New("product inventory is insufficient")
						}
						product.Inventory -= order.Quantity
						product.UpdatedAt = time.Now().UTC()
						order.Status = "fulfilled"
						order.TrackingCode = trackingCode
						order.UpdatedAt = time.Now().UTC()
						return nil
					}
				}
				return ErrCompanyProductNotFound
			}
		}
		return ErrCompanyOrderNotFound
	})
}

func (s *CompanyStore) GrowthReport(id string) (CompanyGrowthReport, error) {
	company, err := s.Get(id)
	if err != nil {
		return CompanyGrowthReport{}, err
	}
	report := CompanyGrowthReport{Company: company, SandboxOnly: true, CampaignsTotal: len(company.Campaigns), AffiliatePrograms: len(company.AffiliatePrograms), Products: len(company.Products)}
	for _, campaign := range company.Campaigns {
		if campaign.Status == "active" {
			report.CampaignsActive++
		}
	}
	for _, link := range company.AffiliateLinks {
		report.AffiliateConversions += link.Conversions
		report.RevenueCents += link.RevenueCents
	}
	for _, order := range company.Orders {
		if order.Status == "fulfilled" {
			report.FulfilledOrders++
			report.RevenueCents += order.TotalCents
		} else {
			report.PendingOrders++
		}
	}
	return report, nil
}

func (r CompanyGrowthReport) SortForStableJSON() CompanyGrowthReport {
	sort.SliceStable(r.Company.Campaigns, func(i, j int) bool { return r.Company.Campaigns[i].UpdatedAt.Before(r.Company.Campaigns[j].UpdatedAt) })
	return r
}
