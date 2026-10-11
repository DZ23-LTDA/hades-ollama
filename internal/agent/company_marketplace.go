package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Estágio 26 — operação de campanha em SANDBOX: aprovação, agendamento e métricas.
//
// O que faltava: a campanha já nascia com `Mode = "sandbox"`, exigia aprovação e recusava
// lançamento fora do sandbox, e os campos de métrica (impressões, cliques, conversões, gasto)
// existiam — mas **não havia agendamento nem forma de registrar métrica**, então o painel
// mostrava zeros permanentemente e "métricas" era só intenção.
//
// Decisões deliberadas:
//
//   - **MÉTRICA SÓ SOMA**: os contadores são incrementais e valores negativos são recusados.
//     Permitir subtração apagaria histórico de desempenho, que é justamente o dado que se quer
//     preservar;
//   - **MÉTRICA EXIGE CAMPANHA QUE RODOU**: registrar desempenho de campanha em `draft` ou
//     apenas aprovada seria inventar resultado de algo que não executou. Só estados de execução
//     em sandbox aceitam medição;
//   - **ORÇAMENTO É LIMITE, NÃO SUGESTÃO**: o incremento de gasto não pode passar do orçamento
//     diário configurado. Sem isso, "orçamento" não limitaria nada;
//   - **NUNCA PUBLICA**: nada aqui chama API de plataforma. O resultado traz
//     `PublicadoExternamente: false` e o relatório declara a publicação externa como
//     `BLOCKED_BY_EXTERNAL_DEPENDENCY`, com o motivo — para ninguém confundir simulação com
//     campanha no ar;
//   - **CTR SEM DIVISÃO POR ZERO**: com zero impressões a taxa é 0, não `NaN`.

const (
	// CampaignStatusScheduled é o estado após o agendamento aprovado.
	CampaignStatusScheduled = "scheduled"
	// CampaignExternalPublication declara a natureza bloqueada da publicação real.
	CampaignExternalPublication = "BLOCKED_BY_EXTERNAL_DEPENDENCY"
	// CampaignExternalReason explica o desbloqueio da publicação externa.
	CampaignExternalReason = "publicar campanha exige conta autorizada e scopes válidos da plataforma; " +
		"sem credencial, a operação fica em sandbox e a homologação live é declarada bloqueada"
)

var (
	ErrCampaignScheduleInvalid = errors.New("campaign schedule invalid")
	ErrCampaignMetricsInvalid  = errors.New("campaign metrics invalid")
	ErrCampaignBudgetExceeded  = errors.New("campaign daily budget exceeded")
	ErrCampaignStateInvalid    = errors.New("campaign state does not allow this action")
)

// CampaignMetricsInput é o incremento medido de uma campanha em sandbox.
type CampaignMetricsInput struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	Conversions int64 `json:"conversions"`
	SpendCents  int64 `json:"spend_cents"`
}

// CampaignMetrics é o resultado do registro, com os totais acumulados.
type CampaignMetrics struct {
	CampaignID     string `json:"campaign_id"`
	Status         string `json:"status"`
	Mode           string `json:"mode"`
	Impressions    int64  `json:"impressions"`
	Clicks         int64  `json:"clicks"`
	Conversions    int64  `json:"conversions"`
	SpendCents     int64  `json:"spend_cents"`
	ClickRate      string `json:"click_rate"`
	ConversionRate string `json:"conversion_rate"`
	// PublicadoExternamente é SEMPRE false: esta camada não publica.
	PublicadoExternamente bool   `json:"publicado_externamente"`
	PublicacaoExterna     string `json:"publicacao_externa"`
}

// CampaignReport é o painel de campanhas da empresa.
type CampaignReport struct {
	CompanyID string            `json:"company_id"`
	Campaigns []CompanyCampaign `json:"campaigns"`
	Total     CampaignMetrics   `json:"total"`
	// Sandbox declara que tudo aqui é simulação, não campanha no ar.
	Sandbox           bool   `json:"sandbox"`
	PublicacaoExterna string `json:"publicacao_externa"`
	Motivo            string `json:"motivo"`
}

// ScheduleCampaign agenda uma campanha APROVADA para começar no futuro.
func (s *CompanyStore) ScheduleCampaign(id, campaignID string, startAt time.Time, now time.Time) (Company, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return Company{}, fmt.Errorf("%w: campanha é obrigatória", ErrCampaignScheduleInvalid)
	}
	if !startAt.After(now) {
		// Agendar no passado significaria "já rodou", e nada rodou.
		return Company{}, fmt.Errorf("%w: o início precisa estar no futuro", ErrCampaignScheduleInvalid)
	}
	return s.mutate(id, func(company *Company) error {
		for index := range company.Campaigns {
			campaign := &company.Campaigns[index]
			if campaign.ID != campaignID {
				continue
			}
			// AGENDAR EXIGE APROVAÇÃO: sem isso o agendamento viraria um caminho paralelo para
			// colocar campanha em execução sem passar pela aprovação humana.
			if !campaign.Approved {
				return fmt.Errorf("%w: campanha precisa de aprovação antes do agendamento", ErrCampaignStateInvalid)
			}
			if campaign.Mode != "sandbox" {
				return ErrCompanyGrowthExternalUnavailable
			}
			if campaign.Status != "approved" && campaign.Status != CampaignStatusScheduled {
				return fmt.Errorf("%w: campanha em %q não pode ser agendada", ErrCampaignStateInvalid, campaign.Status)
			}
			moment := startAt.UTC()
			campaign.ScheduledAt = &moment
			campaign.Status = CampaignStatusScheduled
			campaign.UpdatedAt = now.UTC()
			return nil
		}
		return ErrCompanyCampaignNotFound
	})
}

// RecordCampaignMetrics registra desempenho medido em sandbox.
//
// Métrica só SOMA: valores negativos são recusados, e o gasto não pode passar do orçamento
// diário configurado — orçamento que não limita nada não é orçamento.
func (s *CompanyStore) RecordCampaignMetrics(id, campaignID string, input CampaignMetricsInput, now time.Time) (CampaignMetrics, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignMetrics{}, fmt.Errorf("%w: campanha é obrigatória", ErrCampaignMetricsInvalid)
	}
	if input.Impressions < 0 || input.Clicks < 0 || input.Conversions < 0 || input.SpendCents < 0 {
		return CampaignMetrics{}, fmt.Errorf("%w: contadores não podem ser negativos", ErrCampaignMetricsInvalid)
	}
	result := CampaignMetrics{
		CampaignID:            campaignID,
		PublicadoExternamente: false,
		PublicacaoExterna:     CampaignExternalPublication,
	}
	_, err := s.mutate(id, func(company *Company) error {
		for index := range company.Campaigns {
			campaign := &company.Campaigns[index]
			if campaign.ID != campaignID {
				continue
			}
			switch campaign.Status {
			case CampaignStatusScheduled, "active", "paused":
				// Estados de execução em sandbox: medir faz sentido.
			default:
				return fmt.Errorf("%w: campanha em %q não executou, então não há desempenho a medir", ErrCampaignStateInvalid, campaign.Status)
			}
			if campaign.Mode != "sandbox" {
				return ErrCompanyGrowthExternalUnavailable
			}
			if campaign.DailyBudgetCents > 0 && input.SpendCents > campaign.DailyBudgetCents {
				return fmt.Errorf("%w: incremento de %d centavos passa do orçamento diário de %d",
					ErrCampaignBudgetExceeded, input.SpendCents, campaign.DailyBudgetCents)
			}
			campaign.Impressions += input.Impressions
			campaign.Clicks += input.Clicks
			campaign.Conversions += input.Conversions
			campaign.SpendCents += input.SpendCents
			campaign.UpdatedAt = now.UTC()
			result.Status = campaign.Status
			result.Mode = campaign.Mode
			result.Impressions = campaign.Impressions
			result.Clicks = campaign.Clicks
			result.Conversions = campaign.Conversions
			result.SpendCents = campaign.SpendCents
			result.ClickRate = campaignRate(campaign.Clicks, campaign.Impressions)
			result.ConversionRate = campaignRate(campaign.Conversions, campaign.Clicks)
			return nil
		}
		return ErrCompanyCampaignNotFound
	})
	if err != nil {
		return CampaignMetrics{}, err
	}
	return result, nil
}

// campaignRate devolve a razão formatada, tratando base zero sem NaN.
func campaignRate(numerator, denominator int64) string {
	if denominator <= 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", float64(numerator)*100/float64(denominator))
}

// CampaignReport monta o painel de campanhas da empresa e DECLARA que nada foi publicado.
func (s *CompanyStore) CampaignReport(id string) (CampaignReport, error) {
	company, err := s.Get(id)
	if err != nil {
		return CampaignReport{}, err
	}
	report := CampaignReport{
		CompanyID:         company.ID,
		Campaigns:         company.Campaigns,
		Sandbox:           true,
		PublicacaoExterna: CampaignExternalPublication,
		Motivo:            CampaignExternalReason,
	}
	for _, campaign := range company.Campaigns {
		report.Total.Impressions += campaign.Impressions
		report.Total.Clicks += campaign.Clicks
		report.Total.Conversions += campaign.Conversions
		report.Total.SpendCents += campaign.SpendCents
	}
	report.Total.CampaignID = company.ID
	report.Total.Mode = "sandbox"
	report.Total.PublicadoExternamente = false
	report.Total.PublicacaoExterna = CampaignExternalPublication
	report.Total.ClickRate = campaignRate(report.Total.Clicks, report.Total.Impressions)
	report.Total.ConversionRate = campaignRate(report.Total.Conversions, report.Total.Clicks)
	return report, nil
}
