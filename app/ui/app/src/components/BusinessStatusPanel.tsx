import { useCallback, useEffect, useState } from "react";
import { useT } from "@/i18n";
import { downloadBusinessExport, getBusinessStatus, getBusinessTrend, recordBusinessSnapshot } from "@/lib/agenticClient";
import { triggerDownload } from "@/lib/download";
import { BusinessTrendChart, TREND_METRICS } from "@/components/BusinessTrendChart";
import type { BusinessStatusReport, BusinessTrend } from "@/lib/agenticClient";

export const BUSINESS_EXPORT_FILENAME = "hades-business-status.csv";

// Painel de BI / status de negócio (estágio 32).
//
// É uma superfície de LEITURA, e as regras que a guiam vêm do próprio desenho do servidor:
//
//  1. **NADA É INVENTADO**: os números vêm do relatório; quando não há dado, a tela DIZ que não há
//     dado em vez de exibir zero com aparência de resposta;
//  2. **O QUE NÃO É MEDIDO APARECE NA TELA**: a lista `nao_medido` é renderizada inteira, com o
//     motivo. Esconder essa lista faria o painel parecer completo — que é a mentira mais fácil de
//     cometer num dashboard;
//  3. **A PROVENIÊNCIA APARECE**: cada bloco diz de qual store veio, para o número poder ser
//     conferido por quem desconfia dele (e deveria desconfiar);
//  4. **A SÉRIE NÃO VIRA DELTA ZERO**: quando não há duas amostras, a tela mostra o MOTIVO em vez de
//     uma variação de zero — "não mudou" e "não há com o que comparar" são coisas diferentes;
//  5. **REGISTRAR AMOSTRA DIZ A VERDADE**: o servidor é idempotente por janela de hora, então a tela
//     informa "já havia amostra nesta hora" quando `created=false`, em vez de comemorar um dado novo
//     que não existe.
export function BusinessStatusPanel({ companyId }: { companyId: string }) {
  const { t } = useT();
  const [report, setReport] = useState<BusinessStatusReport | null>(null);
  const [trend, setTrend] = useState<BusinessTrend | null>(null);
  const [metricId, setMetricId] = useState<string>(TREND_METRICS[0].id);
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      setReport(await getBusinessStatus(companyId));
      setNotice(null);
    } catch (error) {
      // Falha aparece como falha e o relatório anterior NÃO é mantido como se fosse atual.
      setReport(null);
      setNotice(error instanceof Error ? error.message : t("businessStatus.errors.panel"));
    }
    try {
      setTrend(await getBusinessTrend(companyId));
    } catch (error) {
      setTrend(null);
      setNotice(error instanceof Error ? error.message : t("businessStatus.errors.trend"));
    } finally {
      setLoading(false);
    }
  }, [companyId, t]);
  useEffect(() => {
    void refresh();
  }, [refresh]);

  const record = useCallback(async () => {
    setLoading(true);
    try {
      const record_ = await recordBusinessSnapshot(companyId);
      setTrend(record_.trend);
      setNotice(
        record_.created
          ? t("businessStatus.record.created")
          : t("businessStatus.record.existing"),
      );
    } catch (error) {
      setNotice(error instanceof Error ? error.message : t("businessStatus.errors.record"));
    } finally {
      setLoading(false);
    }
  }, [companyId, t]);

  const money = (cents: number) => (cents / 100).toFixed(2);  const hasData = report
    ? report.commerce.orders > 0 ||
      report.finance.entries > 0 ||
      report.campaigns.total > 0 ||
      report.sales.proposals > 0 ||
      report.inbox.conversations > 0
    : false;
  const indicators: Array<[string, string | number]> = report
    ? [
        ["businessStatus.indicators.companies", report.companies],
        ["businessStatus.indicators.orders", report.commerce.orders],
        ["businessStatus.indicators.paid", report.commerce.paid],
        ["businessStatus.indicators.revenue", money(report.commerce.revenue_cents)],
        ["businessStatus.indicators.financeIncome", money(report.finance.income_cents)],
        ["businessStatus.indicators.financeExpense", money(report.finance.expense_cents)],
        ["businessStatus.indicators.net", money(report.finance.net_cents)],
        ["businessStatus.indicators.reconciled", money(report.finance.reconciled_cents)],
        ["businessStatus.indicators.campaigns", report.campaigns.total],
        ["businessStatus.indicators.scheduled", report.campaigns.scheduled],
        ["businessStatus.indicators.impressions", report.campaigns.impressions],
        ["businessStatus.indicators.clicks", report.campaigns.clicks],
        ["businessStatus.indicators.proposals", report.sales.proposals],
        ["businessStatus.indicators.followUpsOverdue", report.sales.follow_ups_overdue],
        ["businessStatus.indicators.conversations", report.inbox.conversations],
        ["businessStatus.indicators.slaViolations", report.inbox.sla_violations],
        ["businessStatus.indicators.incompleteChannels", report.marketplace_channels_incompletos],
      ]
    : [];

  // Métrica do gráfico: escolhida por quem usa, com o rótulo em pt-BR; o padrão é a receita de pedidos.
  const selectedMetric = TREND_METRICS.find((metric) => metric.id === metricId) ?? TREND_METRICS[0];

  // Exportar é uma ação de LEITURA que sai do produto: o arquivo é aberto numa planilha, longe da
  // tela. Por isso o aviso diz explicitamente que ele leva as lacunas e a origem de cada número — e
  // a falha aparece como falha, nunca como "exportado" silencioso.
  const exportCSV = useCallback(async () => {
    setLoading(true);
    try {
      const blob = await downloadBusinessExport(companyId);
      triggerDownload(blob, BUSINESS_EXPORT_FILENAME);
      setNotice(t("businessStatus.export.done", { file: BUSINESS_EXPORT_FILENAME }));
    } catch (error) {
      setNotice(error instanceof Error ? error.message : t("businessStatus.errors.export"));
    } finally {
      setLoading(false);
    }
  }, [companyId, t]);

  // Plural por duas chaves (One/Many): nunca concatenar "s".
  const samplesLabel = trend
    ? t(
        trend.snapshots === 1
          ? "businessStatus.trend.samplesOne"
          : "businessStatus.trend.samplesMany",
        { count: trend.snapshots, max: trend.retencao_maxima },
      )
    : "";
  const discardedLabel =
    trend && trend.amostras_descartadas > 0
      ? t(
          trend.amostras_descartadas === 1
            ? "businessStatus.trend.discardedOne"
            : "businessStatus.trend.discardedMany",
          { count: trend.amostras_descartadas },
        )
      : "";

  return (
    <section className="mt-5 rounded-2xl border border-sky-200/80 bg-sky-50/40 p-5 dark:border-sky-900/60 dark:bg-sky-950/10">      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.14em] text-sky-700 dark:text-sky-300">{t("businessStatus.header.eyebrow")}</div>
          <h2 className="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">{t("businessStatus.header.title")}</h2>
          <p className="mt-1 max-w-3xl text-xs leading-5 text-neutral-600 dark:text-neutral-400">
            {t("businessStatus.header.leadBefore")} <strong>{t("businessStatus.header.leadStrong")}</strong>{" "}
            {t("businessStatus.header.leadAfter")}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="rounded-xl border border-sky-300 px-3 py-2 text-xs dark:border-sky-800"
            disabled={loading}
            aria-describedby="business-export-note"
            onClick={() => void exportCSV()}
          >
            {t("businessStatus.actions.export")}
          </button>
          <button
            type="button"
            className="rounded-xl border border-sky-300 px-3 py-2 text-xs dark:border-sky-800"
            disabled={loading}
            onClick={() => void refresh()}
          >
            {loading
              ? t("businessStatus.actions.refreshing")
              : t("businessStatus.actions.refresh")}
          </button>
        </div>
      </div>
      <p id="business-export-note" className="mt-2 text-[11px] leading-4 text-neutral-500 dark:text-neutral-400">
        {t("businessStatus.exportNote.before")} (<code>secao,chave,valor,observacao</code>){" "}
        {t("businessStatus.exportNote.afterSchema")} <strong>{t("businessStatus.exportNote.gaps")}</strong>{" "}
        {t("businessStatus.exportNote.conjunction")}{" "}
        <strong>{t("businessStatus.exportNote.origin")}</strong>
        {t("businessStatus.exportNote.after")}
      </p>

      {notice && (
        <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-amber-300 bg-white px-3 py-2 text-xs text-amber-900 dark:border-amber-900 dark:bg-neutral-900 dark:text-amber-200">
          {notice}
        </div>
      )}

      {report && !hasData && (
        <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-700 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300">
          {t("businessStatus.empty.text")}
        </div>
      )}

      {report && hasData && (
        <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {indicators.map(([label, value]) => (
            <div key={label} className="rounded-xl border border-sky-200 bg-white p-3 dark:border-sky-900/60 dark:bg-neutral-900">
              <div className="text-[10px] uppercase tracking-wider text-neutral-400">{t(label)}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900 dark:text-white">{value}</div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-4 rounded-xl border border-sky-200 bg-white p-4 dark:border-sky-900/60 dark:bg-neutral-900">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold">{t("businessStatus.trend.title")}</h3>
          <button
            type="button"
            className="rounded-xl border border-sky-300 px-3 py-1.5 text-xs dark:border-sky-800"
            disabled={loading}
            onClick={() => void record()}
          >
            {t("businessStatus.trend.record")}
          </button>
        </div>
        {!trend && <p className="mt-2 text-xs text-neutral-500">{t("businessStatus.trend.unavailable")}</p>}
        {trend && (
          <>
            <p className="mt-2 text-xs text-neutral-600 dark:text-neutral-400">
              {samplesLabel}
              {trend.amostras_descartadas > 0 ? ` · ${discardedLabel}` : ""}
            </p>
            {trend.sem_base ? (
              // SEM DUAS AMOSTRAS NÃO HÁ TENDÊNCIA: o motivo aparece em vez de uma variação de zero.
              <p role="status" aria-live="polite" className="mt-2 rounded-lg border border-neutral-200 px-3 py-2 text-xs text-neutral-600 dark:border-neutral-800 dark:text-neutral-400">
                {t("businessStatus.trend.noBase", { reason: trend.motivo ?? "" })}
              </p>
            ) : (
              <>
                <p className="mt-1 text-xs text-neutral-500">{t("businessStatus.trend.interval", { interval: trend.intervalo ?? "" })}</p>
                <ul className="mt-2 grid gap-1 text-xs text-neutral-700 sm:grid-cols-2 dark:text-neutral-300">
                  {(trend.deltas ?? []).map((delta) => (
                    <li key={delta.metric}>
                      {delta.metric}: {delta.delta > 0 ? `+${delta.delta}` : delta.delta} ({delta.from} → {delta.to})
                    </li>
                  ))}
                </ul>
              </>
            )}
            {trend.history.length > 0 && (
              <div className="mt-3 rounded-lg border border-neutral-200 p-3 dark:border-neutral-800">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h4 className="text-xs font-semibold">{t("businessStatus.chart.title")}</h4>
                  <label className="text-[11px] text-neutral-500 dark:text-neutral-400">
                    {t("businessStatus.chart.metricLabel")}
                    <select
                      className="ml-2 rounded-lg border border-neutral-300 bg-white px-2 py-1 text-xs dark:border-neutral-700 dark:bg-neutral-900"
                      value={metricId}
                      onChange={(event) => setMetricId(event.target.value)}
                    >
                      {TREND_METRICS.map((metric) => (
                        <option key={metric.id} value={metric.id}>
                          {metric.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
                {/* Curva em SVG sem dependência nova, com a tabela equivalente sempre visível. */}
                <BusinessTrendChart history={trend.history} metric={selectedMetric} />
              </div>
            )}
          </>
        )}
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">{t("businessStatus.naoMedido.title")}</h3>
          <ul className="mt-2 space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
            {(report?.nao_medido ?? []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">{t("businessStatus.sources.title")}</h3>
          <ul className="mt-2 space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
            {(report?.sources ?? []).map((source) => (
              <li key={source.metric}>
                <strong>{source.metric}</strong>: {source.source}
              </li>
            ))}
          </ul>
          {report?.campaigns.publicacao_externa && (
            <p className="mt-2 text-xs text-neutral-500">
              {t("businessStatus.sources.external")} <strong>{report.campaigns.publicacao_externa}</strong>
            </p>
          )}
        </div>
      </div>
    </section>
  );
}
