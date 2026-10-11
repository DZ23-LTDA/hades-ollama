package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Estágio 32 — BI/status de negócio. Propriedades travadas:
//
//  1. os números vêm dos stores REAIS (nada é inventado nem estimado);
//  2. cada bloco declara sua PROVENIÊNCIA;
//  3. o que NÃO é medido é DECLARADO com motivo — zero sem medição não é resposta;
//  4. ISOLAMENTO: o relatório de uma organização não contém dados de outra;
//  5. organização ausente é erro declarado.
//
// businessFixtureClock é o relógio FIXO dos fixtures de negócio do pacote.
//
// Existe por causa de um defeito real: os fixtures usavam `time.Now().UTC()` e um teste que somava
// `time.Minute` podia CRUZAR A VIRADA DE HORA — como a janela da série é de hora, a asserção de
// idempotência falhava SOZINHA (o CI ficou vermelho às 13:59:57 UTC). Com um instante fixo, os testes
// deixam de depender de onde o ponteiro estava: o determinismo passa a ser **estrutural**.
var businessFixtureClock = time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)

func businessStatusFixture(t *testing.T) (*Runtime, time.Time) {
	t.Helper()
	runtime := marketplaceRuntime(t)
	now := businessFixtureClock
	organization := LocalOrganizationID

	company, err := runtime.CompanyStore().Create(Company{
		OrganizationID: organization, Name: "Loja de BI", Currency: "BRL",
		Budget: CompanyBudget{Currency: "BRL", MonthlyLimitCents: 1_000_00},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "BI-1", Name: "Produto de BI", PriceCents: 2500, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	// Estoque de 9 unidades, consumido por um pedido pago e por um cancelado (que devolve a reserva).
	if _, err := runtime.Commerce().RecordStock(organization, "BI-1", 9, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	// Um segundo SKU fica com disponível ZERO (o limiar de estoque baixo é `disponível <= 0`), para
	// o indicador ser exercitado de verdade em vez de assumido.
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "BI-LOW", Name: "Produto esgotado", PriceCents: 1000, Currency: "BRL", Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(organization, "BI-LOW", 1, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	drained, err := runtime.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: organization, IdempotencyKey: "bi-low",
		Items: []CommerceOrderItem{{SKU: "BI-LOW", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().ConfirmPayment(organization, drained.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "bi-low-pay"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	// Um pedido pago (receita) e um pedido cancelado.
	order, err := runtime.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: organization, IdempotencyKey: "bi-1",
		Items: []CommerceOrderItem{{SKU: "BI-1", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().ConfirmPayment(organization, order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "bi-pay"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	cancelled, err := runtime.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: organization, IdempotencyKey: "bi-2",
		Items: []CommerceOrderItem{{SKU: "BI-1", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().Cancel(organization, cancelled.ID, "cliente desistiu", now); err != nil {
		t.Fatal(err)
	}
	// Livro financeiro: uma receita conciliada e uma despesa em aberto.
	if _, _, err := runtime.CompanyStore().RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindIncome, AmountCents: 2500, Currency: "BRL",
		Description: "venda sandbox", OccurredAt: now,
	}, now); err != nil {
		t.Fatal(err)
	}
	income, err := runtime.CompanyStore().ListLedgerEntries(company.ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.CompanyStore().ReconcileLedgerEntry(company.ID, income[0].ID, "extrato-sandbox", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.CompanyStore().RecordLedgerEntry(LedgerEntry{
		CompanyID: company.ID, Kind: LedgerKindExpense, AmountCents: 1000, Currency: "BRL",
		Description: "insumo sandbox", OccurredAt: now,
	}, now); err != nil {
		t.Fatal(err)
	}
	// Campanha aprovada, agendada e medida.
	campaign, err := runtime.CompanyStore().AddCampaign(company.ID, CompanyCampaign{
		Name: "Campanha de BI", Channel: "social", Objective: "gerar demanda", DailyBudgetCents: 10_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	campaignID := campaign.Campaigns[0].ID
	if _, err := runtime.CompanyStore().ApproveCampaign(company.ID, campaignID); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CompanyStore().ScheduleCampaign(company.ID, campaignID, now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CompanyStore().RecordCampaignMetrics(company.ID, campaignID, CampaignMetricsInput{
		Impressions: 300, Clicks: 30, Conversions: 3, SpendCents: 1500,
	}, now); err != nil {
		t.Fatal(err)
	}
	// Vendas: uma proposta e um follow-up vencido.
	if _, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "ct_bi", Currency: "BRL", Validity: time.Hour,
		Items: []SalesProposalItemRequest{{SKU: "BI-1", Quantity: 1}},
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().ScheduleFollowUp(organization, "ct_bi", "ord_bi", "retomar", time.Hour, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Atendimento: uma conversa aberta com SLA vencido (janela curta).
	runtime.Inbox().SetFirstResponseSLA(organization, time.Minute)
	if _, _, err := runtime.Inbox().AppendInbound(organization, "whatsapp", "ct_sla", "Cliente", "preciso de ajuda", "ext-1", now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	return runtime, now
}

func TestBusinessStatusAggregatesRealStores(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	report, err := runtime.BusinessStatus(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}

	if report.Companies != 1 {
		t.Fatalf("empresas: %d, esperado 1", report.Companies)
	}
	commerce := report.Commerce
	if commerce.Products != 2 || commerce.Orders != 3 || commerce.Paid != 2 || commerce.Cancelled != 1 {
		t.Fatalf("comércio incorreto: %+v", commerce)
	}
	if commerce.RevenueCents != 3500 {
		t.Fatalf("receita precisa somar os pedidos pagos: %d", commerce.RevenueCents)
	}
	if len(commerce.LowStockSKUs) != 1 || commerce.LowStockSKUs[0] != "BI-LOW" {
		t.Fatalf("estoque baixo precisa apontar o SKU zerado: %+v", commerce.LowStockSKUs)
	}

	finance := report.Finance
	if finance.Entries != 2 || finance.IncomeCents != 2500 || finance.ExpenseCents != 1000 || finance.NetCents != 1500 {
		t.Fatalf("financeiro incorreto: %+v", finance)
	}
	if finance.ReconciledCents != 2500 || finance.Unreconciled != 1 {
		t.Fatalf("conciliação incorreta: %+v", finance)
	}

	campaigns := report.Campaigns
	if campaigns.Total != 1 || campaigns.Scheduled != 1 || campaigns.Active != 0 {
		t.Fatalf("campanhas incorretas: %+v", campaigns)
	}
	if campaigns.Impressions != 300 || campaigns.Clicks != 30 || campaigns.Conversions != 3 || campaigns.SpendCents != 1500 {
		t.Fatalf("métricas de campanha incorretas: %+v", campaigns)
	}
	if campaigns.PublicacaoExterna != CampaignExternalPublication {
		t.Fatalf("campanha precisa declarar publicação externa bloqueada: %q", campaigns.PublicacaoExterna)
	}

	sales := report.Sales
	if sales.Proposals != 1 || sales.FollowUps != 1 || sales.FollowUpsOverdue != 1 {
		t.Fatalf("vendas incorretas: %+v", sales)
	}
	if sales.ProposalsByState == nil {
		t.Fatalf("propostas precisam declarar o estado: %+v", sales)
	}

	inbox := report.Inbox
	if inbox.Conversations != 1 || inbox.Open != 1 || inbox.SLAViolations != 1 {
		t.Fatalf("atendimento incorreto: %+v", inbox)
	}

	// PROVENIÊNCIA: cada bloco diz de onde veio.
	if len(report.Sources) < 5 {
		t.Fatalf("o relatório precisa declarar a origem de cada bloco: %+v", report.Sources)
	}
	metrics := map[string]bool{}
	for _, source := range report.Sources {
		if source.Metric == "" || source.Source == "" {
			t.Fatalf("proveniência incompleta: %+v", source)
		}
		metrics[source.Metric] = true
	}
	for _, expected := range []string{"commerce", "finance", "campaigns", "sales", "inbox"} {
		if !metrics[expected] {
			t.Fatalf("bloco %q sem proveniência declarada", expected)
		}
	}
	if !report.Sandbox {
		t.Fatal("o painel precisa declarar que opera em sandbox")
	}
}

func TestBusinessStatusDeclaresWhatIsNotMeasured(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	report, err := runtime.BusinessStatus(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.ToLower(strings.Join(report.NotMeasured, " | "))
	for _, esperado := range []string{"fiscal", "bancária", "tokens", "marketplace", "telefonia"} {
		if !strings.Contains(joined, esperado) {
			t.Fatalf("o relatório precisa declarar que NÃO mede %q: %s", esperado, joined)
		}
	}
	// O que não é medido vem com MOTIVO, não como item solto.
	for _, item := range report.NotMeasured {
		if !strings.Contains(item, ":") {
			t.Fatalf("item não medido precisa do motivo: %q", item)
		}
	}
	if report.MarketplaceChannels == 0 {
		t.Fatal("canais de marketplace incompletos precisam ser declarados")
	}
}

func TestBusinessStatusIsScopedByOrganization(t *testing.T) {
	runtime, now := businessStatusFixture(t)
	// A organização "org_b" não tem nada: NENHUM número da org_a pode aparecer.
	other, err := runtime.BusinessStatus("org_b", now)
	if err != nil {
		t.Fatal(err)
	}
	if other.Companies != 0 || other.Commerce.Products != 0 || other.Commerce.Orders != 0 || other.Commerce.RevenueCents != 0 {
		t.Fatalf("comércio vazou entre organizações: %+v", other.Commerce)
	}
	if other.Finance.Entries != 0 || other.Finance.IncomeCents != 0 || other.Finance.ExpenseCents != 0 {
		t.Fatalf("financeiro vazou entre organizações: %+v", other.Finance)
	}
	if other.Campaigns.Total != 0 || other.Campaigns.Impressions != 0 {
		t.Fatalf("campanhas vazaram entre organizações: %+v", other.Campaigns)
	}
	if other.Sales.Proposals != 0 || other.Sales.FollowUps != 0 {
		t.Fatalf("vendas vazaram entre organizações: %+v", other.Sales)
	}
	if other.Inbox.Conversations != 0 {
		t.Fatalf("atendimento vazou entre organizações: %+v", other.Inbox)
	}

	// E a organização original continua com os seus números (o agregador não "roubou" dado).
	own, err := runtime.BusinessStatus(LocalOrganizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	if own.Commerce.RevenueCents != 3500 || own.Inbox.Conversations != 1 {
		t.Fatalf("os números da própria organização mudaram: %+v", own)
	}

	// Organização ausente é erro declarado, não relatório vazio.
	if _, err := runtime.BusinessStatus("   ", now); err == nil {
		t.Fatal("organização ausente precisa ser erro declarado")
	}
}

// O agregador não pode criar dado: sem nada cadastrado, o relatório é zero E continua declarando o
// que não mede. Um painel vazio precisa continuar dizendo o que ele não cobre.
// Os fixtures de negócio NÃO podem voltar a usar o relógio de parede.
//
// A guarda não fixa a data (isso deixaria o teste rígido a mudanças legítimas): ela verifica a
// PROPRIEDADE que importa — o instante do fixture **não** pode ser o instante atual. Se alguém voltar a
// escrever `time.Now()`, este teste falha aqui, e não numa terça-feira às 13:59 no CI.
func TestBusinessFixtureClockIsNotTheWallClock(t *testing.T) {
	_, now := businessStatusFixture(t)
	if delta := time.Since(now); delta < time.Minute && delta > -time.Minute {
		t.Fatalf("o fixture de negócio está usando o relógio de parede (%s): "+
			"isso reintroduz o flake da virada de hora; use businessFixtureClock", now)
	}
	// E o mesmo instante precisa valer para todos os usos dentro de uma execução.
	_, again := businessStatusFixture(t)
	if !again.Equal(now) {
		t.Fatalf("o instante do fixture precisa ser estável: %s vs %s", now, again)
	}
}

func TestBusinessStatusOnEmptyOrganizationStillDeclaresGaps(t *testing.T) {
	runtime := marketplaceRuntime(t)
	report, err := runtime.BusinessStatus(LocalOrganizationID, businessFixtureClock)
	if err != nil {
		t.Fatal(err)
	}
	if report.Companies != 0 || report.Commerce.Orders != 0 {
		t.Fatalf("organização vazia precisa zerar os indicadores: %+v", report)
	}
	if len(report.NotMeasured) == 0 || len(report.Sources) == 0 {
		t.Fatal("mesmo vazio, o relatório precisa declarar origem e lacunas")
	}
	if !errors.Is(err, error(nil)) && err != nil {
		t.Fatal(err)
	}
}
