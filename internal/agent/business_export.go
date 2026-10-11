package agent

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Estágio 32 — EXPORTAÇÃO do painel de negócio.
//
// Um arquivo exportado circula fora do produto: é aberto numa planilha, mandado por e-mail, colado num
// relatório. Por isso as decisões aqui são conservadoras:
//
//  1. **ESQUEMA ÚNICO E PARSEÁVEL**: `secao,chave,valor,observacao` — uma linha por dado. Nada de
//     cabeçalhos diferentes por bloco, que quebram qualquer leitor de CSV. A série vira uma linha por
//     (janela, métrica);
//  2. **AS LACUNAS VÃO NO ARQUIVO**: as linhas `nao_medido` acompanham o export. Fora do produto, a
//     lista que ficava na tela some — e um número exportado sem as lacunas **parece completo**, que é
//     exatamente o engano que o painel evita;
//  3. **A ORIGEM VAI EM CADA INDICADOR**: o arquivo diz de qual store cada número veio, para poder ser
//     conferido longe da interface;
//  4. **ESCAPING FEITO PELA BIBLIOTECA**: `encoding/csv` cuida de vírgula, aspas e quebra de linha.
//     Montar CSV por concatenação de strings é a forma clássica de corromper um relatório.

// BusinessExportRow é uma linha do export.
type BusinessExportRow struct {
	Section string
	Key     string
	Value   string
	Note    string
}

var ErrBusinessExportScope = errors.New("business export requires an organization scope")

// BusinessExportCSV gera o CSV do painel e da série da organização.
func (r *Runtime) BusinessExportCSV(organizationID string, now time.Time) (string, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return "", fmt.Errorf("%w: organização é obrigatória", ErrBusinessExportScope)
	}
	report, err := r.BusinessStatus(organizationID, now)
	if err != nil {
		return "", err
	}
	trend, err := r.BusinessTrend(organizationID, time.Time{})
	if err != nil {
		return "", err
	}

	rows := []BusinessExportRow{
		{Section: "meta", Key: "organizacao", Value: organizationID, Note: "escopo do export"},
		{Section: "meta", Key: "gerado_em", Value: now.UTC().Format(time.RFC3339), Note: "instante da consulta"},
		{Section: "meta", Key: "sandbox", Value: strconv.FormatBool(report.Sandbox), Note: "nada aqui é produção"},
		// INDICADORES com a origem declarada.
		{Section: "indicador", Key: "empresas", Value: strconv.Itoa(report.Companies), Note: "CompanyStore"},
		{Section: "indicador", Key: "produtos", Value: strconv.Itoa(report.Commerce.Products), Note: "CommerceStore"},
		{Section: "indicador", Key: "pedidos", Value: strconv.Itoa(report.Commerce.Orders), Note: "CommerceStore"},
		{Section: "indicador", Key: "pedidos_pagos", Value: strconv.Itoa(report.Commerce.Paid), Note: "CommerceStore"},
		{Section: "indicador", Key: "pedidos_cancelados", Value: strconv.Itoa(report.Commerce.Cancelled), Note: "CommerceStore"},
		{Section: "indicador", Key: "receita_cents", Value: strconv.FormatInt(report.Commerce.RevenueCents, 10), Note: "CommerceStore"},
		{Section: "indicador", Key: "estoque_baixo", Value: strconv.Itoa(len(report.Commerce.LowStockSKUs)), Note: strings.Join(report.Commerce.LowStockSKUs, "|")},
		{Section: "indicador", Key: "lancamentos", Value: strconv.Itoa(report.Finance.Entries), Note: "CompanyStore.ListLedgerEntries"},
		{Section: "indicador", Key: "receita_financeira_cents", Value: strconv.FormatInt(report.Finance.IncomeCents, 10), Note: "CompanyStore.ListLedgerEntries"},
		{Section: "indicador", Key: "despesa_financeira_cents", Value: strconv.FormatInt(report.Finance.ExpenseCents, 10), Note: "CompanyStore.ListLedgerEntries"},
		{Section: "indicador", Key: "liquido_cents", Value: strconv.FormatInt(report.Finance.NetCents, 10), Note: "receita menos despesa"},
		{Section: "indicador", Key: "conciliado_cents", Value: strconv.FormatInt(report.Finance.ReconciledCents, 10), Note: "CompanyStore.ListLedgerEntries"},
		{Section: "indicador", Key: "lancamentos_em_aberto", Value: strconv.Itoa(report.Finance.Unreconciled), Note: "CompanyStore.ListLedgerEntries"},
		{Section: "indicador", Key: "campanhas", Value: strconv.Itoa(report.Campaigns.Total), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "campanhas_agendadas", Value: strconv.Itoa(report.Campaigns.Scheduled), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "campanhas_ativas", Value: strconv.Itoa(report.Campaigns.Active), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "impressoes", Value: strconv.FormatInt(report.Campaigns.Impressions, 10), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "cliques", Value: strconv.FormatInt(report.Campaigns.Clicks, 10), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "conversoes", Value: strconv.FormatInt(report.Campaigns.Conversions, 10), Note: "CompanyStore.CampaignReport"},
		{Section: "indicador", Key: "propostas", Value: strconv.Itoa(report.Sales.Proposals), Note: "SalesWorkforceStore"},
		{Section: "indicador", Key: "follow_ups", Value: strconv.Itoa(report.Sales.FollowUps), Note: "SalesWorkforceStore"},
		{Section: "indicador", Key: "follow_ups_vencidos", Value: strconv.Itoa(report.Sales.FollowUpsOverdue), Note: "SalesWorkforceStore"},
		{Section: "indicador", Key: "conversas", Value: strconv.Itoa(report.Inbox.Conversations), Note: "InboxStore"},
		{Section: "indicador", Key: "conversas_abertas", Value: strconv.Itoa(report.Inbox.Open), Note: "InboxStore"},
		{Section: "indicador", Key: "violacoes_sla", Value: strconv.Itoa(report.Inbox.SLAViolations), Note: "InboxStore"},
		{Section: "indicador", Key: "canais_marketplace_incompletos", Value: strconv.Itoa(report.MarketplaceChannels), Note: "MarketplaceChannels"},
		{Section: "indicador", Key: "publicacao_externa", Value: report.Campaigns.PublicacaoExterna, Note: "nada foi publicado em plataforma externa"},
	}
	// AS LACUNAS VÃO NO ARQUIVO: fora do produto, quem lê o CSV não vê a tela.
	for _, gap := range report.NotMeasured {
		parts := strings.SplitN(gap, ":", 2)
		key := strings.TrimSpace(parts[0])
		note := ""
		if len(parts) == 2 {
			note = strings.TrimSpace(parts[1])
		}
		rows = append(rows, BusinessExportRow{Section: "nao_medido", Key: key, Value: "", Note: note})
	}
	// SÉRIE: uma linha por (janela, métrica).
	rows = append(rows, BusinessExportRow{
		Section: "serie", Key: "resumo", Value: strconv.Itoa(trend.Snapshots),
		Note: fmt.Sprintf("retencao_maxima=%d; fora_do_recorte=%d; sem_base=%t; motivo=%s",
			trend.Retention, trend.Discarded, trend.SemBase, trend.Motivo),
	})
	for _, snapshot := range trend.History {
		metrics := map[string]int64{
			"pedidos":                  int64(snapshot.Report.Commerce.Orders),
			"receita_cents":            snapshot.Report.Commerce.RevenueCents,
			"receita_financeira_cents": snapshot.Report.Finance.IncomeCents,
			"despesa_financeira_cents": snapshot.Report.Finance.ExpenseCents,
			"liquido_cents":            snapshot.Report.Finance.NetCents,
			"campanhas_ativas":         int64(snapshot.Report.Campaigns.Active),
			"impressoes":               snapshot.Report.Campaigns.Impressions,
			"propostas":                int64(snapshot.Report.Sales.Proposals),
			"follow_ups_vencidos":      int64(snapshot.Report.Sales.FollowUpsOverdue),
			"conversas":                int64(snapshot.Report.Inbox.Conversations),
			"violacoes_sla":            int64(snapshot.Report.Inbox.SLAViolations),
		}
		metricsOrder := []string{
			"pedidos", "receita_cents", "receita_financeira_cents", "despesa_financeira_cents",
			"liquido_cents", "campanhas_ativas", "impressoes", "propostas", "follow_ups_vencidos",
			"conversas", "violacoes_sla",
		}
		for _, metric := range metricsOrder {
			rows = append(rows, BusinessExportRow{
				Section: "serie",
				Key:     snapshot.Window + "." + metric,
				Value:   strconv.FormatInt(metrics[metric], 10),
				Note:    "amostra_em=" + snapshot.TakenAt.UTC().Format(time.RFC3339),
			})
		}
	}
	if !trend.SemBase {
		for _, delta := range trend.Deltas {
			rows = append(rows, BusinessExportRow{
				Section: "tendencia", Key: delta.Metric,
				Value: strconv.FormatInt(delta.Delta, 10),
				Note:  fmt.Sprintf("de=%d; para=%d; intervalo=%s", delta.From, delta.To, trend.Interval),
			})
		}
	} else {
		rows = append(rows, BusinessExportRow{
			Section: "tendencia", Key: "sem_base", Value: "",
			Note: trend.Motivo,
		})
	}

	var builder strings.Builder
	if err := csvEncodeRows(&builder, rows); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// csvEncodeRows escreve o esquema uniforme `secao,chave,valor,observacao`.
//
// Fica separado de `BusinessExportCSV` para poder ser testado com valores HOSTIS (vírgula, aspas,
// quebra de linha): montar CSV por concatenação é a forma clássica de corromper um relatório, e essa
// garantia precisa de teste próprio.
func csvEncodeRows(builder *strings.Builder, rows []BusinessExportRow) error {
	writer := csv.NewWriter(builder)
	// A biblioteca cuida de vírgula, aspas e quebra de linha.
	if err := writer.Write([]string{"secao", "chave", "valor", "observacao"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Section, row.Key, row.Value, row.Note}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
