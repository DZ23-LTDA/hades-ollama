import {
  scheduleRunConcerns,
  scheduleRunSummary,
  scheduleRunVerdict,
  scheduleRunVerdictLabel,
  scheduleRunVerdictTone,
  scheduleStepKindLabel,
  scheduleStepStatusLabel,
  scheduleStepStatusTone,
  type AgentScheduleRunReport,
} from "@/lib/scheduleRun";

export type ScheduleRunReportCardProps = {
  run: AgentScheduleRunReport | null | undefined;
  /** Código de falha registrado pela última execução, quando houver. */
  lastFailureCode?: string | null;
  /** Data da última execução, quando o servidor já registrou. */
  lastRunAt?: string | null;
};

/**
 * Mostra o resultado POR NÓ da última execução de uma rotina.
 *
 * O veredito nunca esconde passo ignorado: uma execução parcial aparece como
 * "Parcial — houve passo ignorado", com o motivo declarado pelo servidor.
 */
export function ScheduleRunReportCard({ run, lastFailureCode, lastRunAt }: ScheduleRunReportCardProps) {
  const verdict = scheduleRunVerdict(run);
  const concerns = scheduleRunConcerns(run);

  return (
    <div className="mt-2 space-y-2" aria-label="Última execução da rotina">
      <div className="flex flex-wrap items-center gap-2">
        <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${scheduleRunVerdictTone(verdict)}`}>
          {scheduleRunVerdictLabel(verdict)}
        </span>
        <span className="text-[10px] text-neutral-500 dark:text-neutral-400">{scheduleRunSummary(run)}</span>
        {lastRunAt ? (
          <span className="text-[10px] text-neutral-400">
            última execução: {new Date(lastRunAt).toLocaleString("pt-BR")}
          </span>
        ) : null}
      </div>

      {lastFailureCode ? (
        <p className="text-[10px] text-red-600 dark:text-red-400">código da falha: {lastFailureCode}</p>
      ) : null}

      {concerns.length > 0 ? (
        <ul className="space-y-1" aria-label="Passos que exigem atenção">
          {concerns.map((step) => (
            <li key={step.id} className="text-[10px] text-neutral-600 dark:text-neutral-300">
              <span className={`mr-1 rounded px-1.5 py-0.5 font-medium ${scheduleStepStatusTone(step.status)}`}>
                {scheduleStepStatusLabel(step.status)}
              </span>
              {step.id} <span className="text-neutral-400">({scheduleStepKindLabel(step.kind)})</span>
              {step.reason ? <span className="text-amber-700 dark:text-amber-300"> — {step.reason}</span> : null}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
