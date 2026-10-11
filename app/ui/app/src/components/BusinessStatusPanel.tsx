import { useCallback, useEffect, useState } from "react";
import { getBusinessStatus } from "@/lib/agenticClient";
import type { BusinessStatusReport } from "@/lib/agenticClient";

// Painel de BI / status de negócio (estágio 32).
//
// É uma superfície de LEITURA, e as três regras que a guiam vêm do próprio desenho do servidor:
//
//  1. **NADA É INVENTADO**: os números vêm do relatório; quando não há dado, a tela DIZ que não há
//     dado em vez de exibir zero com aparência de resposta;
//  2. **O QUE NÃO É MEDIDO APARECE NA TELA**: a lista `nao_medido` é renderizada inteira, com o
//     motivo. Esconder essa lista faria o painel parecer completo — que é a mentira mais fácil de
//     cometer num dashboard;
//  3. **A PROVENIÊNCIA APARECE**: cada bloco diz de qual store veio, para o número poder ser
//     conferido por quem desconfia dele (e deveria desconfiar).
//
// O único controle é "Atualizar painel", que realmente refaz a consulta. Não há botão decorativo.
export function BusinessStatusPanel({ companyId }: { companyId: string }) {
  const [report, setReport] = useState<BusinessStatusReport | null>(null);
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
      setNotice(error instanceof Error ? error.message : "Painel de negócio indisponível.");
    } finally {
      setLoading(false);
    }
  }, [companyId]);
  useEffect(() => {
    void refresh();
  }, [refresh]);

  const money = (cents: number) => (cents / 100).toFixed(2);
  const hasData = report
    ? report.commerce.orders > 0 ||
      report.finance.entries > 0 ||
      report.campaigns.total > 0 ||
      report.sales.proposals > 0 ||
      report.inbox.conversations > 0
    : false;
  const indicators: Array<[string, string | number]> = report
    ? [
        ["Empresas", report.companies],
        ["Pedidos", report.commerce.orders],
        ["Pagos", report.commerce.paid],
        ["Receita", money(report.commerce.revenue_cents)],
        ["Receita financeira", money(report.finance.income_cents)],
        ["Despesa financeira", money(report.finance.expense_cents)],
        ["Líquido", money(report.finance.net_cents)],
        ["Conciliado", money(report.finance.reconciled_cents)],
        ["Campanhas", report.campaigns.total],
        ["Agendadas", report.campaigns.scheduled],
        ["Impressões", report.campaigns.impressions],
        ["Cliques", report.campaigns.clicks],
        ["Propostas", report.sales.proposals],
        ["Follow-ups vencidos", report.sales.follow_ups_overdue],
        ["Conversas", report.inbox.conversations],
        ["Violações de SLA", report.inbox.sla_violations],
        ["Canais incompletos", report.marketplace_channels_incompletos],
      ]
    : [];

  return (
    <section className="mt-5 rounded-2xl border border-sky-200/80 bg-sky-50/40 p-5 dark:border-sky-900/60 dark:bg-sky-950/10">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.14em] text-sky-700 dark:text-sky-300">BI e status de negócio</div>
          <h2 className="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">Indicadores reais, com origem e lacunas declaradas</h2>
          <p className="mt-1 max-w-3xl text-xs leading-5 text-neutral-600 dark:text-neutral-400">
            Cada número vem de um módulo interno e diz de onde veio. O que este painel <strong>não mede</strong> aparece
            logo abaixo, com o motivo: zero sem medição não é resposta.
          </p>
        </div>
        <button
          type="button"
          className="rounded-xl border border-sky-300 px-3 py-2 text-xs dark:border-sky-800"
          disabled={loading}
          onClick={() => void refresh()}
        >
          {loading ? "Atualizando…" : "Atualizar painel"}
        </button>
      </div>

      {notice && (
        <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-amber-300 bg-white px-3 py-2 text-xs text-amber-900 dark:border-amber-900 dark:bg-neutral-900 dark:text-amber-200">
          {notice}
        </div>
      )}

      {report && !hasData && (
        <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-700 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300">
          Sem dados ainda nesta empresa: os indicadores aparecem quando houver pedido, lançamento, campanha,
          proposta ou conversa registrados.
        </div>
      )}

      {report && hasData && (
        <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {indicators.map(([label, value]) => (
            <div key={label} className="rounded-xl border border-sky-200 bg-white p-3 dark:border-sky-900/60 dark:bg-neutral-900">
              <div className="text-[10px] uppercase tracking-wider text-neutral-400">{label}</div>
              <div className="mt-1 text-lg font-semibold text-neutral-900 dark:text-white">{value}</div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">O que este painel NÃO mede</h3>
          <ul className="mt-2 space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
            {(report?.nao_medido ?? []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </div>
        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">Origem de cada número</h3>
          <ul className="mt-2 space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
            {(report?.sources ?? []).map((source) => (
              <li key={source.metric}>
                <strong>{source.metric}</strong>: {source.source}
              </li>
            ))}
          </ul>
          {report?.campaigns.publicacao_externa && (
            <p className="mt-2 text-xs text-neutral-500">
              Publicação externa: <strong>{report.campaigns.publicacao_externa}</strong>
            </p>
          )}
        </div>
      </div>
    </section>
  );
}
