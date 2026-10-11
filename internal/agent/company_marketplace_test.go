package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Estágio 26 — operação de campanha em sandbox: agendamento e métricas.
//
// Propriedades travadas:
//  1. AGENDAR exige APROVAÇÃO (não existe caminho paralelo para execução) e início no FUTURO;
//  2. MÉTRICA só em campanha que EXECUTOU (draft/apenas aprovada é recusado);
//  3. métrica só SOMA: valor negativo é recusado;
//  4. ORÇAMENTO limita: incremento de gasto acima do diário é recusado;
//  5. o relatório DECLARA que nada foi publicado externamente (`BLOCKED_BY_EXTERNAL_DEPENDENCY`);
//  6. CTR com zero impressões é 0, não NaN.
func marketplaceCompany(t *testing.T) (*CompanyStore, Company, time.Time) {
	t.Helper()
	root := shortTestRoot(t)
	store, err := NewCompanyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	company, err := store.Create(Company{
		OrganizationID: LocalOrganizationID, Name: "Loja Sandbox", Mission: "vender",
		Currency: "BRL",
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, company, now
}

func addCampaign(t *testing.T, store *CompanyStore, company Company, budget int64) Company {
	t.Helper()
	updated, err := store.AddCampaign(company.ID, CompanyCampaign{
		Name: "Campanha de inverno", Channel: "content", Objective: "gerar demanda",
		DailyBudgetCents: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func TestScheduleCampaignRequiresApprovalAndFutureStart(t *testing.T) {
	store, company, now := marketplaceCompany(t)
	updated := addCampaign(t, store, company, 10000)
	campaign := updated.Campaigns[0]
	if campaign.Mode != "sandbox" {
		t.Fatalf("campanha precisa nascer em sandbox: %q", campaign.Mode)
	}

	// SEM aprovação: agendar é RECUSADO.
	if _, err := store.ScheduleCampaign(company.ID, campaign.ID, now.Add(time.Hour), now); !errors.Is(err, ErrCampaignStateInvalid) {
		t.Fatalf("agendar sem aprovação precisa falhar, veio %v", err)
	}
	// Início no PASSADO é recusado.
	if _, err := store.ScheduleCampaign(company.ID, campaign.ID, now.Add(-time.Hour), now); !errors.Is(err, ErrCampaignScheduleInvalid) {
		t.Fatalf("início no passado precisa falhar, veio %v", err)
	}

	// Com aprovação: agenda e o estado muda, com a data registrada.
	if _, err := store.ApproveCampaign(company.ID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	scheduledAt := now.Add(2 * time.Hour)
	updated, err := store.ScheduleCampaign(company.ID, campaign.ID, scheduledAt, now)
	if err != nil {
		t.Fatal(err)
	}
	got := updated.Campaigns[0]
	if got.Status != CampaignStatusScheduled || got.ScheduledAt == nil {
		t.Fatalf("campanha precisa ficar agendada com data: %+v", got)
	}
	if !got.ScheduledAt.Equal(scheduledAt.UTC()) {
		t.Fatalf("data de agendamento incorreta: %s", got.ScheduledAt)
	}
	// Campanha inexistente é erro declarado.
	if _, err := store.ScheduleCampaign(company.ID, "cmp_inexistente", scheduledAt, now); !errors.Is(err, ErrCompanyCampaignNotFound) {
		t.Fatalf("campanha inexistente precisa falhar, veio %v", err)
	}
}

func TestCampaignMetricsOnlyAfterExecutionAndOnlyAdds(t *testing.T) {
	store, company, now := marketplaceCompany(t)
	updated := addCampaign(t, store, company, 5000)
	campaign := updated.Campaigns[0]

	// MÉTRICA EM RASCUNHO é recusada: nada executou.
	if _, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{Impressions: 10}, now); !errors.Is(err, ErrCampaignStateInvalid) {
		t.Fatalf("medir rascunho precisa falhar, veio %v", err)
	}
	if _, err := store.ApproveCampaign(company.ID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	// Apenas aprovada, ainda sem execução: também recusada.
	if _, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{Impressions: 10}, now); !errors.Is(err, ErrCampaignStateInvalid) {
		t.Fatalf("medir campanha apenas aprovada precisa falhar, veio %v", err)
	}
	if _, err := store.ScheduleCampaign(company.ID, campaign.ID, now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}

	// NEGATIVO é recusado: métrica só soma.
	if _, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{Clicks: -1}, now); !errors.Is(err, ErrCampaignMetricsInvalid) {
		t.Fatalf("contador negativo precisa falhar, veio %v", err)
	}
	// Incremento de gasto acima do orçamento diário é recusado.
	if _, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{SpendCents: 5001}, now); !errors.Is(err, ErrCampaignBudgetExceeded) {
		t.Fatalf("estourar o orçamento precisa falhar, veio %v", err)
	}

	// Registro válido: acumula e calcula as taxas.
	first, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{
		Impressions: 1000, Clicks: 50, Conversions: 5, SpendCents: 3000,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Impressions != 1000 || first.Clicks != 50 || first.Conversions != 5 || first.SpendCents != 3000 {
		t.Fatalf("totais incorretos: %+v", first)
	}
	if first.ClickRate != "5.00%" || first.ConversionRate != "10.00%" {
		t.Fatalf("taxas incorretas: %q / %q", first.ClickRate, first.ConversionRate)
	}
	// A camada NÃO publica: isso precisa estar explícito no resultado.
	if first.PublicadoExternamente || first.PublicacaoExterna != CampaignExternalPublication {
		t.Fatalf("o registro não pode declarar publicação externa: %+v", first)
	}

	// Segundo registro SOMA ao anterior (não substitui).
	second, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{
		Impressions: 500, Clicks: 25, Conversions: 2, SpendCents: 2000,
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.Impressions != 1500 || second.Clicks != 75 || second.SpendCents != 5000 {
		t.Fatalf("métrica precisa SOMAR: %+v", second)
	}

	// Campanha inexistente é erro declarado.
	if _, err := store.RecordCampaignMetrics(company.ID, "cmp_inexistente", CampaignMetricsInput{Impressions: 1}, now); !errors.Is(err, ErrCompanyCampaignNotFound) {
		t.Fatalf("campanha inexistente precisa falhar, veio %v", err)
	}
}

func TestCampaignReportDeclaresSandboxAndNoExternalPublication(t *testing.T) {
	store, company, now := marketplaceCompany(t)
	// Relatório SEM campanha: CTR é 0, nunca NaN.
	empty, err := store.CampaignReport(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Total.ClickRate != "0.00%" || strings.Contains(strings.ToLower(empty.Total.ClickRate), "nan") {
		t.Fatalf("sem impressões a taxa precisa ser 0: %q", empty.Total.ClickRate)
	}
	if !empty.Sandbox || empty.PublicacaoExterna != CampaignExternalPublication || empty.Motivo == "" {
		t.Fatalf("o relatório precisa declarar sandbox e publicação externa bloqueada com motivo: %+v", empty)
	}

	updated := addCampaign(t, store, company, 10000)
	campaign := updated.Campaigns[0]
	if _, err := store.ApproveCampaign(company.ID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ScheduleCampaign(company.ID, campaign.ID, now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordCampaignMetrics(company.ID, campaign.ID, CampaignMetricsInput{
		Impressions: 200, Clicks: 20, Conversions: 4, SpendCents: 1500,
	}, now); err != nil {
		t.Fatal(err)
	}
	report, err := store.CampaignReport(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Campaigns) != 1 {
		t.Fatalf("o relatório precisa trazer as campanhas: %+v", report.Campaigns)
	}
	if report.Total.Impressions != 200 || report.Total.Clicks != 20 || report.Total.ConversionRate != "20.00%" {
		t.Fatalf("totais do painel incorretos: %+v", report.Total)
	}
	if report.PublicacaoExterna != CampaignExternalPublication || report.Total.PublicadoExternamente {
		t.Fatalf("o painel não pode sugerir publicação real: %+v", report)
	}
	// Empresa inexistente é erro declarado.
	if _, err := store.CampaignReport("cmp_empresa_inexistente"); err == nil {
		t.Fatal("empresa inexistente precisa falhar no relatório")
	}
}
