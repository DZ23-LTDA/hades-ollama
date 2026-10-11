package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Estágio 32 — BI e status de negócio: indicadores REAIS, com proveniência e com o que NÃO é medido.
//
// O estágio pedia "BI / business status e métricas" e esta é a parte que pode ser entregue sem
// credencial externa: agregar o que os módulos internos JÁ registram. Três decisões deliberadas:
//
//   - **CADA NÚMERO TEM PROVENIÊNCIA**: o relatório declara de qual store cada bloco veio. Um painel
//     sem origem é um painel em que ninguém pode confiar nem auditar;
//   - **O QUE NÃO É MEDIDO É DECLARADO**: faturamento fiscal, saldo bancário, custo de tokens,
//     publicação externa e telefonia aparecem em `nao_medido` com o motivo. Um painel que mostra
//     zero onde não há medição é **pior que um painel vazio**, porque zero parece resposta;
//   - **ESCOPO POR ORGANIZAÇÃO**: toda consulta é filtrada pela organização pedida. Os módulos já
//     são escopados; o agregador não pode ser a camada que vaza dado entre organizações.

// ErrBusinessStatusScopeRequired indica que o painel foi pedido sem organização.
var ErrBusinessStatusScopeRequired = errors.New("business status requires an organization scope")

// BusinessMetricSource liga um bloco de indicadores à sua origem.
type BusinessMetricSource struct {
	Metric string `json:"metric"`
	Source string `json:"source"`
}

// BusinessCommerceStatus resume comércio.
type BusinessCommerceStatus struct {
	Products     int      `json:"products"`
	Orders       int      `json:"orders"`
	Paid         int      `json:"paid"`
	Fulfilled    int      `json:"fulfilled"`
	Cancelled    int      `json:"cancelled"`
	RevenueCents int64    `json:"revenue_cents"`
	LowStockSKUs []string `json:"low_stock_skus,omitempty"`
}

// BusinessFinanceStatus resume o livro financeiro interno.
type BusinessFinanceStatus struct {
	Entries         int   `json:"entries"`
	IncomeCents     int64 `json:"income_cents"`
	ExpenseCents    int64 `json:"expense_cents"`
	NetCents        int64 `json:"net_cents"`
	ReconciledCents int64 `json:"reconciled_cents"`
	Unreconciled    int   `json:"unreconciled_entries"`
}

// BusinessCampaignStatus resume campanhas em sandbox.
type BusinessCampaignStatus struct {
	Total             int    `json:"total"`
	Draft             int    `json:"draft"`
	Approved          int    `json:"approved"`
	Scheduled         int    `json:"scheduled"`
	Active            int    `json:"active"`
	Impressions       int64  `json:"impressions"`
	Clicks            int64  `json:"clicks"`
	Conversions       int64  `json:"conversions"`
	SpendCents        int64  `json:"spend_cents"`
	PublicacaoExterna string `json:"publicacao_externa"`
}

// BusinessSalesStatus resume a equipe comercial.
type BusinessSalesStatus struct {
	Proposals        int            `json:"proposals"`
	ProposalsByState map[string]int `json:"proposals_by_state,omitempty"`
	FollowUps        int            `json:"follow_ups"`
	FollowUpsOverdue int            `json:"follow_ups_overdue"`
}

// BusinessInboxStatus resume o atendimento.
type BusinessInboxStatus struct {
	Conversations int `json:"conversations"`
	Open          int `json:"open"`
	Closed        int `json:"closed"`
	SLAViolations int `json:"sla_violations"`
}

// BusinessStatusReport é o painel de negócio de uma organização.
type BusinessStatusReport struct {
	OrganizationID      string                 `json:"organization_id"`
	GeneratedAt         time.Time              `json:"generated_at"`
	Companies           int                    `json:"companies"`
	Commerce            BusinessCommerceStatus `json:"commerce"`
	Finance             BusinessFinanceStatus  `json:"finance"`
	Campaigns           BusinessCampaignStatus `json:"campaigns"`
	Sales               BusinessSalesStatus    `json:"sales"`
	Inbox               BusinessInboxStatus    `json:"inbox"`
	MarketplaceChannels int                    `json:"marketplace_channels_incompletos"`
	Sources             []BusinessMetricSource `json:"sources"`
	// NotMeasured declara o que este painel NÃO mede, com o motivo. Zero sem medição não é resposta.
	NotMeasured []string `json:"nao_medido"`
	Sandbox     bool     `json:"sandbox"`
}

// BusinessStatus agrega os indicadores internos da organização.
func (r *Runtime) BusinessStatus(organizationID string, now time.Time) (BusinessStatusReport, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return BusinessStatusReport{}, fmt.Errorf("%w: organização é obrigatória", ErrBusinessStatusScopeRequired)
	}
	report := BusinessStatusReport{
		OrganizationID: organizationID,
		GeneratedAt:    now.UTC(),
		Sandbox:        true,
		Sources: []BusinessMetricSource{
			{Metric: "commerce", Source: "CommerceStore (pedidos, produtos e estoque)"},
			{Metric: "finance", Source: "CompanyStore.ListLedgerEntries (livro financeiro por empresa)"},
			{Metric: "campaigns", Source: "CompanyStore.CampaignReport (campanha em sandbox)"},
			{Metric: "sales", Source: "SalesWorkforceStore (propostas e follow-ups)"},
			{Metric: "inbox", Source: "InboxStore.ListConversations (conversas e SLA)"},
			{Metric: "marketplace", Source: "MarketplaceChannels (estado declarado do canal)"},
		},
		NotMeasured: []string{
			"faturamento fiscal e obrigações tributárias: exigem integração fiscal autorizada",
			"saldo e conciliação bancária: exigem extrato ou PSP autorizado",
			"custo de tokens e de modelo: exigem provedor externo com preço declarado",
			"publicação em marketplace e redes sociais: BLOCKED_BY_EXTERNAL_DEPENDENCY",
			"telefonia e voz: exigem número e provedor",
			"preço de venda negociado: as propostas usam o preço do catálogo",
		},
	}

	// Comércio: resumo já escopado por organização.
	if commerce := r.Commerce(); commerce != nil {
		summary := commerce.Summary(organizationID)
		report.Commerce = BusinessCommerceStatus{
			Products:     summary.Products,
			Orders:       summary.Orders,
			Paid:         summary.Paid,
			Fulfilled:    summary.Fulfilled,
			Cancelled:    summary.Cancelled,
			RevenueCents: summary.RevenueCents,
			LowStockSKUs: summary.LowStockSKUs,
		}
	}

	// Empresas da organização: o livro financeiro e as campanhas vivem dentro da empresa.
	var companies []Company
	if store := r.CompanyStore(); store != nil {
		companies = store.List(organizationID)
		report.Companies = len(companies)
	}
	for _, company := range companies {
		if store := r.CompanyStore(); store != nil {
			entries, err := store.ListLedgerEntries(company.ID, time.Time{}, time.Time{})
			if err != nil {
				return BusinessStatusReport{}, err
			}
			for _, entry := range entries {
				report.Finance.Entries++
				switch entry.Kind {
				case LedgerKindIncome:
					report.Finance.IncomeCents += entry.AmountCents
				case LedgerKindExpense:
					report.Finance.ExpenseCents += entry.AmountCents
				}
				if entry.ReconciledAt != nil {
					report.Finance.ReconciledCents += entry.AmountCents
				} else {
					report.Finance.Unreconciled++
				}
			}

			campaigns, err := store.CampaignReport(company.ID)
			if err != nil {
				return BusinessStatusReport{}, err
			}
			for _, campaign := range campaigns.Campaigns {
				report.Campaigns.Total++
				switch campaign.Status {
				case "draft":
					report.Campaigns.Draft++
				case "approved":
					report.Campaigns.Approved++
				case CampaignStatusScheduled:
					report.Campaigns.Scheduled++
				case "active":
					report.Campaigns.Active++
				}
			}
			report.Campaigns.Impressions += campaigns.Total.Impressions
			report.Campaigns.Clicks += campaigns.Total.Clicks
			report.Campaigns.Conversions += campaigns.Total.Conversions
			report.Campaigns.SpendCents += campaigns.Total.SpendCents
			report.Campaigns.PublicacaoExterna = campaigns.PublicacaoExterna
		}
	}
	report.Finance.NetCents = report.Finance.IncomeCents - report.Finance.ExpenseCents
	if report.Campaigns.PublicacaoExterna == "" {
		report.Campaigns.PublicacaoExterna = CampaignExternalPublication
	}

	// Vendas: propostas e follow-ups da organização.
	if sales := r.SalesWorkforce(); sales != nil {
		states := map[string]int{}
		for _, proposal := range sales.Proposals(organizationID, "") {
			report.Sales.Proposals++
			states[proposal.Status]++
		}
		if len(states) > 0 {
			report.Sales.ProposalsByState = states
		}
		report.Sales.FollowUps = len(sales.FollowUps(organizationID, false, now))
		report.Sales.FollowUpsOverdue = len(sales.FollowUps(organizationID, true, now))
	}

	// Atendimento: conversas e violações de SLA (prazo vencido sem primeira resposta).
	if inbox := r.Inbox(); inbox != nil {
		for _, conversation := range inbox.ListConversations(organizationID, "") {
			report.Inbox.Conversations++
			switch conversation.Status {
			case "closed":
				report.Inbox.Closed++
			default:
				report.Inbox.Open++
			}
			if conversation.FirstReplyAt == nil && conversation.ReplyDueAt != nil && conversation.ReplyDueAt.Before(now) {
				report.Inbox.SLAViolations++
			}
		}
	}

	// Marketplace: quantos canais estão incompletos. Estado declarado, não chamada externa.
	for _, channel := range MarketplaceChannels(map[string]bool{}) {
		if channel.State != MarketplaceStateConnected {
			report.MarketplaceChannels++
		}
	}

	sort.Strings(report.NotMeasured)
	return report, nil
}
