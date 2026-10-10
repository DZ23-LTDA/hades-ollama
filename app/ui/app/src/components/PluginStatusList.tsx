import {
  isReadyToUse,
  pluginExecutionSummary,
  pluginStateDescription,
  pluginStateLabel,
  pluginStateTone,
  type AgentPluginStatus,
} from "@/lib/pluginStatus";

export type PluginStatusListProps = {
  statuses: AgentPluginStatus[];
  loading?: boolean;
  error?: string | null;
};

const toneClasses: Record<string, string> = {
  green: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300",
  amber: "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300",
  red: "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300",
  neutral: "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300",
};

/**
 * Lista o estado REAL de cada plugin instalado.
 *
 * Regra de interface (§11.3): nunca dizer que algo está pronto porque aparece no
 * catálogo. O selo "Pronto para usar" só existe quando o servidor devolve
 * `ready_to_use` — o que, por sua vez, exige execução real comprovada.
 */
export function PluginStatusList({ statuses, loading = false, error = null }: PluginStatusListProps) {
  if (loading) {
    return (
      <p className="text-xs text-neutral-500 dark:text-neutral-400" role="status">
        Carregando o estado real dos plugins…
      </p>
    );
  }
  if (error) {
    return (
      <p className="text-xs text-red-600 dark:text-red-400" role="alert">
        {error}
      </p>
    );
  }
  if (!statuses.length) {
    return (
      <p className="text-xs text-neutral-500 dark:text-neutral-400">
        Nenhum plugin instalado nesta organização. Instalar um pacote assinado habilita o estado real aqui.
      </p>
    );
  }

  return (
    <ul className="space-y-3" aria-label="Estado real dos plugins instalados">
      {statuses.map((status) => {
        const tone = toneClasses[pluginStateTone(status.state)] ?? toneClasses.neutral;
        const ready = isReadyToUse(status);
        return (
          <li
            key={`${status.plugin_id}@${status.version}`}
            className="rounded-xl border border-neutral-200/80 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-900"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-neutral-900 dark:text-white">
                  {status.plugin_id} <span className="text-xs text-neutral-400">v{status.version}</span>
                </p>
                <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                  {status.kind} · {pluginStateDescription(status.state)}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <span className={`rounded-full px-2 py-1 text-[10px] font-medium ${tone}`}>
                  {pluginStateLabel(status.state)}
                </span>
                {ready ? (
                  <span className="rounded-full bg-emerald-600 px-2 py-1 text-[10px] font-medium text-white">
                    Pronto para usar
                  </span>
                ) : null}
              </div>
            </div>
            <p className="mt-2 text-[11px] text-neutral-500 dark:text-neutral-400">{pluginExecutionSummary(status)}</p>
            {status.reasons?.length ? (
              <ul className="mt-2 space-y-1 text-[11px] text-amber-700 dark:text-amber-300">
                {status.reasons.map((reason) => (
                  <li key={reason}>• {reason}</li>
                ))}
              </ul>
            ) : null}
            <p className="mt-2 text-[10px] text-neutral-400">
              Capacidades: {status.capabilities?.length ? status.capabilities.join(", ") : "nenhuma"}
            </p>
          </li>
        );
      })}
    </ul>
  );
}
