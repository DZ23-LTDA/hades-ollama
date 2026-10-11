import { useCallback, useEffect, useState } from "react";
import {
  getCompanyCampaignReport,
  getMarketplaceChannels,
  recordCompanyCampaignMetrics,
  scheduleCompanyCampaign,
  syncMarketplace,
} from "@/lib/agenticClient";
import type {
  AgentCompany,
  CompanyCampaignReport,
  MarketplaceChannel,
  MarketplaceSyncReport,
} from "@/lib/agenticClient";

const field = "h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
const button = "rounded-xl bg-neutral-950 px-3 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950";

// Agendamento e métricas de campanha + catálogo de marketplace.
//
// Duas regras de honestidade guiam esta superfície:
//
//  1. **NADA é exibido como publicado**: o servidor devolve `publicacao_externa` como
//     `BLOCKED_BY_EXTERNAL_DEPENDENCY` e este painel mostra exatamente isso, com o motivo. Um selo de
//     "campanha no ar" aqui seria mentira;
//  2. **O controle só aparece quando a AÇÃO é possível**: agendar exige aprovação e métrica exige
//     campanha que executou — as mesmas regras do servidor. Botão que sempre falha é controle
//     decorativo, o que o estágio 19 proíbe.
export function CompanyCampaignSchedulingPanel({
  company,
  onCompanyChange,
}: {
  company: AgentCompany;
  onCompanyChange: (company: AgentCompany) => void;
}) {
  const [report, setReport] = useState<CompanyCampaignReport | null>(null);
  const [channels, setChannels] = useState<MarketplaceChannel[]>([]);
  const [syncReport, setSyncReport] = useState<MarketplaceSyncReport | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [hours, setHours] = useState("2");
  const [impressions, setImpressions] = useState("100");
  const [clicks, setClicks] = useState("5");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      setReport(await getCompanyCampaignReport(company.id));
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Painel de campanhas indisponível.");
    }
    try {
      const catalog = await getMarketplaceChannels(company.id);
      setChannels(catalog.channels ?? []);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Catálogo de marketplaces indisponível.");
    }
  }, [company.id]);
  useEffect(() => {
    void refresh();
  }, [refresh]);

  const run = async (action: () => Promise<AgentCompany>, message: string) => {
    setBusy(true);
    try {
      const updated = await action();
      onCompanyChange(updated);
      setNotice(message);
      await refresh();
    } catch (error) {
      // Falha aparece como falha: nada é marcado como concluído sem confirmação do servidor.
      setNotice(error instanceof Error ? error.message : "A operação de campanha falhou.");
    } finally {
      setBusy(false);
    }
  };

  const campaigns = company.campaigns ?? [];
  const measurable = (status: string) => ["scheduled", "active", "paused"].includes(status);
  const declaredExternal = report?.publicacao_externa ?? "BLOCKED_BY_EXTERNAL_DEPENDENCY";
  const declaredReason = report?.motivo ?? "";

  return (
    <section className="mt-5 rounded-2xl border border-amber-200/80 bg-amber-50/40 p-5 dark:border-amber-900/60 dark:bg-amber-950/10">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.14em] text-amber-700 dark:text-amber-300">Campanhas e marketplaces</div>
          <h2 className="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">Agendamento, métricas medidas e sincronização em sandbox</h2>
          <p className="mt-1 max-w-3xl text-xs leading-5 text-neutral-600 dark:text-neutral-400">
            Agendar exige campanha aprovada e início no futuro; métrica exige campanha que já executou e apenas soma. A
            sincronização de marketplace roda contra um dublê local: <strong>nada é publicado em plataforma externa</strong>.
          </p>
        </div>
        <button type="button" className="rounded-xl border border-amber-300 px-3 py-2 text-xs dark:border-amber-800" onClick={() => void refresh()}>
          Atualizar painel
        </button>
      </div>

      <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-amber-300 bg-white px-3 py-2 text-xs text-amber-900 dark:border-amber-900 dark:bg-neutral-900 dark:text-amber-200">
        Publicação externa: <strong>{declaredExternal}</strong>
        {declaredReason ? ` — ${declaredReason}` : ""}
      </div>
      {notice && (
        <div role="status" aria-live="polite" className="mt-2 rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-700 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300">
          {notice}
        </div>
      )}

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">Agendar e medir campanha</h3>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <label className="text-xs text-neutral-500" htmlFor="campaign-hours">
              Início em horas
              <input id="campaign-hours" className={field} type="number" min="1" value={hours} onChange={(event) => setHours(event.target.value)} />
            </label>
            <label className="text-xs text-neutral-500" htmlFor="campaign-impressions">
              Impressões medidas
              <input id="campaign-impressions" className={field} type="number" min="0" value={impressions} onChange={(event) => setImpressions(event.target.value)} />
            </label>
            <label className="text-xs text-neutral-500" htmlFor="campaign-clicks">
              Cliques medidos
              <input id="campaign-clicks" className={field} type="number" min="0" value={clicks} onChange={(event) => setClicks(event.target.value)} />
            </label>
          </div>
          <div className="mt-3 space-y-2">
            {campaigns.length === 0 && <p className="text-xs text-neutral-500">Nenhuma campanha ainda: crie uma no painel Growth OS acima.</p>}
            {campaigns.map((campaign) => (
              <div key={campaign.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
                <div className="flex flex-wrap justify-between gap-2">
                  <span className="font-medium">{campaign.name}</span>
                  <span>{campaign.status}</span>
                </div>
                {campaign.scheduled_at && <div className="mt-1 text-neutral-500">Agendada para {new Date(campaign.scheduled_at).toLocaleString()}</div>}
                <div className="mt-1 text-neutral-500">
                  Medido: {campaign.impressions ?? 0} impressões · {campaign.clicks ?? 0} cliques · {campaign.conversions ?? 0} conversões
                </div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {/* Só aparece quando o servidor aceitaria: aprovada e ainda não agendada. */}
                  {campaign.approved && !campaign.scheduled_at && (
                    <button
                      type="button"
                      className="rounded-lg border px-2 py-1"
                      disabled={busy}
                      onClick={() => void run(() => scheduleCompanyCampaign(company.id, campaign.id, Number(hours)), "Campanha agendada no sandbox.")}
                    >
                      Agendar
                    </button>
                  )}
                  {/* Só aparece quando existe execução: medir rascunho seria inventar resultado. */}
                  {measurable(campaign.status) && (
                    <button
                      type="button"
                      className="rounded-lg border px-2 py-1"
                      disabled={busy}
                      onClick={() => void run(async () => {
                        await recordCompanyCampaignMetrics(company.id, campaign.id, {
                          impressions: Number(impressions),
                          clicks: Number(clicks),
                          conversions: 0,
                          spend_cents: 0,
                        });
                        return company;
                      }, "Métricas medidas registradas (somam ao total).")}
                    >
                      Registrar métricas
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
          {report && (
            <div className="mt-3 rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
              Total medido: {report.total.impressions} impressões · {report.total.clicks} cliques · CTR {report.total.click_rate} ·
              conversão {report.total.conversion_rate} · gasto {(report.total.spend_cents / 100).toFixed(2)}
            </div>
          )}
        </div>

        <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-sm font-semibold">Marketplaces</h3>
          <p className="mt-1 text-xs text-neutral-500">
            O catálogo informa o estado <strong>real</strong> do canal. Sem credencial configurada, o canal não está pronto — e o
            painel diz isso em vez de exibir um selo verde.
          </p>
          <div className="mt-3 space-y-2">
            {channels.length === 0 && <p className="text-xs text-neutral-500">Nenhum canal no catálogo.</p>}
            {channels.map((channel) => (
              <div key={channel.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
                <div className="flex flex-wrap justify-between gap-2">
                  <span className="font-medium">{channel.name}</span>
                  <span>{channel.state}</span>
                </div>
                <div className="mt-1 text-neutral-500">Testado ao vivo: {channel.live_tested ? "sim" : "não"}</div>
                {channel.unblock_procedure && <div className="mt-1 text-neutral-500">{channel.unblock_procedure}</div>}
              </div>
            ))}
          </div>
          <button
            type="button"
            className={`${button} mt-3`}
            disabled={busy || channels.length === 0}
            onClick={() => void (async () => {
              setBusy(true);
              try {
                const result = await syncMarketplace(company.id, {
                  channel_id: channels[0].id,
                  direction: "push",
                  items: [],
                });
                setSyncReport(result);
              } catch (error) {
                // Zero itens é recusado pelo servidor com 400: a interface mostra o motivo real.
                setNotice(error instanceof Error ? error.message : "A sincronização falhou.");
              } finally {
                setBusy(false);
              }
            })()}
          >
            Sincronizar catálogo (sandbox)
          </button>
          {syncReport && (
            <div className="mt-3 rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
              {syncReport.applied} aplicados · {syncReport.skipped} ignorados · {syncReport.conflicts} conflitos ·{" "}
              {syncReport.operations_used}/{syncReport.operation_limit} operações · {syncReport.homologacao_live}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
