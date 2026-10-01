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
  type SupervisorStatus,
} from "@/lib/agenticClient";

export function CompanySupervisorPanel() {
  const [status, setStatus] = useState<SupervisorStatus | null>(null);
  const [, setLoading] = useState(true);
  const [ticking, setTicking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const st = await getSupervisorStatus();
      setStatus(st);
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
              Executa rotinas, avança ciclos de negócio e monitora agendamentos com freios HITL de segurança.
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
          <span className="text-[10px] uppercase tracking-wider text-neutral-400">Freios HITL (Aprovação)</span>
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
              ? `Último ciclo: ${new Date(status.last_tick_at).toLocaleTimeString()}`
              : "Aguardando primeiro ciclo"}
          </span>
        </div>
      </div>
    </section>
  );
}
