import { useCallback, useEffect, useState } from "react";
import {
  ArrowPathIcon,
  CheckCircleIcon,
  PauseCircleIcon,
  PlayCircleIcon,
  ShieldCheckIcon,
  CpuChipIcon,
  ClockIcon,
} from "@heroicons/react/24/outline";
import {
  getSupervisorStatus,
  setSupervisorConfig,
  triggerSupervisorTick,
  getSupervisorIncidents,
  getSupervisorTicks,
  type SupervisorStatus,
  type SupervisorIncident,
  type SupervisorTickRecord,
} from "@/lib/agenticClient";

export function CompanySupervisorPanel() {
  const [status, setStatus] = useState<SupervisorStatus | null>(null);
  const [, setLoading] = useState(true);
  const [ticking, setTicking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [incidentes, setIncidentes] = useState<SupervisorIncident[]>([]);
  const [ticks, setTicks] = useState<SupervisorTickRecord[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const st = await getSupervisorStatus();
      setStatus(st);
      // Incidentes e ticks sao pedidos separados: a falha deles nao apaga o estado.
      try {
        const ri = await getSupervisorIncidents();
        setIncidentes(ri.incidents ?? []);
      } catch {
        setIncidentes([]);
      }
      try {
        const rt = await getSupervisorTicks();
        setTicks(rt.ticks ?? []);
      } catch {
        setTicks([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleToggleEnabled = async () => {
    if (!status) return;
    try {
      const updated = await setSupervisorConfig({ enabled: !status.enabled });
      setStatus(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleTriggerTick = async () => {
    setTicking(true);
    setError(null);
    try {
      const res = await triggerSupervisorTick();
      setStatus(res.status);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setTicking(false);
    }
  };

  const isEnabled = status?.enabled ?? false;

  return (
    <section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-50 text-purple-600 dark:bg-purple-950/50">
            <CpuChipIcon className="h-6 w-6" />
          </div>
          <div>
            <h2 className="text-sm font-bold text-neutral-900 dark:text-white">
              Supervisor Autônomo — Agente Sempre-Ligado
            </h2>
            <p className="text-xs text-neutral-500">
              Executa rotinas, avança ciclos de negócio e monitora agendamentos com freios de aprovação humana.
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold ${
              isEnabled
                ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300"
                : "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
            }`}
          >
            {isEnabled ? (
              <>
                <CheckCircleIcon className="h-4 w-4" />
                Sempre-Ligado (Ativo)
              </>
            ) : (
              <>
                <PauseCircleIcon className="h-4 w-4" />
                Pausado
              </>
            )}
          </span>

          <button
            type="button"
            onClick={() => void handleToggleEnabled()}
            className="inline-flex items-center gap-1.5 rounded-xl border border-neutral-200 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
          >
            {isEnabled ? (
              <>
                <PauseCircleIcon className="h-4 w-4" />
                Pausar
              </>
            ) : (
              <>
                <PlayCircleIcon className="h-4 w-4" />
                Ativar
              </>
            )}
          </button>

          <button
            type="button"
            onClick={() => void handleTriggerTick()}
            disabled={ticking}
            className="inline-flex items-center gap-1.5 rounded-xl bg-neutral-900 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-neutral-800 disabled:opacity-40 dark:bg-white dark:text-neutral-900"
          >
            <ArrowPathIcon className={`h-3.5 w-3.5 ${ticking ? "animate-spin" : ""}`} />
            Ciclo Manual (Tick)
          </button>
        </div>
      </div>

      {error && (
        <div className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
          {error}
        </div>
      )}

      {/* Metrics Grid */}
      <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-5">
        <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800">
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Ciclos Avançados</span>
          <p className="mt-1 text-lg font-bold text-neutral-900 dark:text-white">
            {status?.company_cycles_advanced ?? 0}
          </p>
        </div>

        <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800">
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Schedules Disparados</span>
          <p className="mt-1 text-lg font-bold text-neutral-900 dark:text-white">
            {status?.schedules_triggered ?? 0}
          </p>
        </div>

        <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800">
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Missões Retomadas</span>
          <p className="mt-1 text-lg font-bold text-neutral-900 dark:text-white">
            {status?.pending_missions_resumed ?? 0}
          </p>
        </div>

        <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800">
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Freios de aprovação humana</span>
          <p className="mt-1 text-lg font-bold text-amber-600 dark:text-amber-400">
            {status?.pending_approvals_count ?? 0}
          </p>
        </div>

        <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800">
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Ações Bloqueadas Ext.</span>
          <p className="mt-1 text-lg font-bold text-neutral-500">
            {status?.blocked_external_actions ?? 0}
          </p>
        </div>
      </div>

      {/* Safety & Monitoring Footer */}
      <div className="mt-4 flex flex-wrap items-center justify-between gap-2 border-t border-neutral-100 pt-3 text-[11px] text-neutral-500 dark:border-neutral-800">
        <div className="flex items-center gap-1.5">
          <ShieldCheckIcon className="h-4 w-4 text-emerald-600" />
          <span>Freios de segurança ativos: gastos e publicações externas exigem aprovação humana.</span>
        </div>
        <div className="flex items-center gap-1.5">
          <ClockIcon className="h-4 w-4 text-neutral-400" />
          <span>
            {status?.last_tick_at
              ? `Último ciclo: ${new Date(status.last_tick_at).toLocaleTimeString("pt-BR")}`
              : "Aguardando primeiro ciclo"}
          </span>
        </div>
      </div>
      <div className="mt-4 border-t border-neutral-800 pt-3">
        <h4 className="mb-2 text-xs font-semibold text-neutral-300">
          Incidentes ({incidentes.length})
        </h4>

        {incidentes.length === 0 ? (
          <p data-testid="supervisor-sem-incidentes" className="text-[11px] text-neutral-500">
            Lista vazia. Isto não distingue «não há incidentes» de «não há supervisor»: sem
            runtime o servidor devolve a mesma lista vazia.
          </p>
        ) : (
          <ul data-testid="supervisor-incidentes" className="space-y-1">
            {incidentes.map((i) => (
              <li
                key={i.id}
                data-testid={`supervisor-incidente-${i.id}`}
                className="rounded border border-neutral-800 px-2 py-1 text-[11px]"
              >
                <span className="font-semibold text-neutral-200">{i.kind}</span>{" "}
                <span className="text-neutral-400">[{i.state}]</span>{" "}
                <span className="text-neutral-300">{i.subject}</span>
                <span
                  data-testid={`supervisor-ciencia-${i.id}`}
                  className="block text-neutral-500"
                >
                  {i.acknowledged_by
                    ? `reconhecido por ${i.acknowledged_by} — reconhecer NÃO é resolver`
                    : "sem reconhecimento humano"}
                </span>
                <span className="block text-neutral-500">
                  {i.recovery_attempts} tentativa(s) de recuperação
                  {i.resolution ? ` · resolução: ${i.resolution}` : " · sem resolução registada"}
                </span>
              </li>
            ))}
          </ul>
        )}

        <h4 className="mb-2 mt-3 text-xs font-semibold text-neutral-300">
          Ciclos recentes ({ticks.length})
        </h4>

        {ticks.length === 0 ? (
          <p data-testid="supervisor-sem-ticks" className="text-[11px] text-neutral-500">
            Nenhum ciclo no diário.
          </p>
        ) : (
          <ul data-testid="supervisor-ticks" className="space-y-0.5 text-[11px] text-neutral-400">
            {ticks.slice(0, 8).map((t) => (
              <li key={t.tick_number}>
                #{t.tick_number} · {t.missions_resumed} missão(ões) retomada(s) ·{" "}
                {t.approvals_created} aprovação(ões) criada(s) ·{" "}
                {t.blocked_external_actions} ação(ões) externa(s) bloqueada(s) ·{" "}
                {t.risk_paused_companies} empresa(s) pausada(s) por risco ·{" "}
                {t.error_count} erro(s)
              </li>
            ))}
          </ul>
        )}
      </div>

    </section>
  );
}
