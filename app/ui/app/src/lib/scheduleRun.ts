/**
 * Relatório de execução de automação na interface (estágio 13).
 *
 * Regra: o servidor decide o que rodou, o que foi ignorado e o que falhou; a tela
 * apenas mostra — e NUNCA resume como sucesso limpo uma execução que teve nó
 * ignorado ou falho.
 */

export type ScheduleStepStatus = "succeeded" | "skipped" | "failed";

export type AgentScheduleStepOutcome = {
  id: string;
  kind: string;
  status: ScheduleStepStatus;
  reason?: string;
  /** Tentativas consumidas pelo passo (retry por nó); 0/ausente = não se aplica. */
  attempts?: number;
};

/** Entrada da DLQ por nó: passo que esgotou as tentativas. */
export type AgentScheduleDeadLetter = {
  step_id: string;
  kind: string;
  reason: string;
  attempts?: number;
  objective?: string;
  failed_at: string;
};

/** Rótulo curto de uma entrada da DLQ, com o motivo de verdade do servidor. */
export function scheduleDeadLetterLabel(entry: AgentScheduleDeadLetter): string {
  const attempts = entry.attempts && entry.attempts > 1 ? ` após ${entry.attempts} tentativas` : "";
  return `${entry.step_id}${attempts} — ${entry.reason}`;
}

export type AgentScheduleRunReport = {
  finished_at: string;
  missions: number;
  steps?: AgentScheduleStepOutcome[] | null;
  run_key?: string;
  dead_letters?: AgentScheduleDeadLetter[] | null;
  flow_version?: number;
};

/** Versão do fluxo executada, quando o servidor informa. */
export function scheduleFlowVersionLabel(report: AgentScheduleRunReport | null | undefined): string {
  const version = report?.flow_version ?? 0;
  return version > 0 ? `fluxo v${version}` : "";
}

/** Entradas da DLQ do relatório, sempre em lista (o servidor pode omitir). */
export function scheduleDeadLetters(report: AgentScheduleRunReport | null | undefined): AgentScheduleDeadLetter[] {
  const entries = report?.dead_letters;
  return Array.isArray(entries) ? entries.filter((entry) => entry && entry.step_id) : [];
}

export type ScheduleRunVerdict = "success" | "partial" | "failed" | "unknown";

export function scheduleStepStatusLabel(status: string): string {
  switch (status) {
    case "succeeded":
      return "executado";
    case "skipped":
      return "ignorado";
    case "failed":
      return "falhou";
    default:
      return "desconhecido";
  }
}

export function scheduleStepStatusTone(status: string): string {
  switch (status) {
    case "succeeded":
      return "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300";
    case "skipped":
      return "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300";
    case "failed":
      return "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300";
    default:
      return "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300";
  }
}

/**
 * Veredito da execução. Uma execução com passo ignorado é PARCIAL, não sucesso:
 * é exatamente o que o estágio 13 proíbe esconder.
 */
export function scheduleRunVerdict(run: AgentScheduleRunReport | null | undefined): ScheduleRunVerdict {
  if (!run) return "unknown";
  const steps = run.steps ?? [];
  if (steps.some((step) => step.status === "failed")) return "failed";
  if (steps.some((step) => step.status === "skipped")) return "partial";
  return "success";
}

export function scheduleRunVerdictLabel(verdict: ScheduleRunVerdict): string {
  switch (verdict) {
    case "success":
      return "Sucesso";
    case "partial":
      return "Parcial — houve passo ignorado";
    case "failed":
      return "Falhou";
    default:
      return "Sem execução registrada";
  }
}

export function scheduleRunVerdictTone(verdict: ScheduleRunVerdict): string {
  switch (verdict) {
    case "success":
      return "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300";
    case "partial":
      return "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300";
    case "failed":
      return "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300";
    default:
      return "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300";
  }
}

/** Resumo honesto: conta o que rodou, o que foi ignorado e quanto criou. */
export function scheduleRunSummary(run: AgentScheduleRunReport | null | undefined): string {
  if (!run) return "Nenhuma execução registrada para esta rotina.";
  const steps = run.steps ?? [];
  const succeeded = steps.filter((step) => step.status === "succeeded").length;
  const skipped = steps.filter((step) => step.status === "skipped").length;
  const failed = steps.filter((step) => step.status === "failed").length;
  const parts = [`${succeeded} executado(s)`];
  if (skipped > 0) parts.push(`${skipped} ignorado(s)`);
  if (failed > 0) parts.push(`${failed} com falha`);
  const missions = run.missions === 1 ? "1 missão criada" : `${run.missions} missões criadas`;
  return `${parts.join(" · ")} · ${missions}`;
}

/** Passos que merecem atenção do operador, com o motivo declarado pelo servidor. */
export function scheduleRunConcerns(run: AgentScheduleRunReport | null | undefined): AgentScheduleStepOutcome[] {
  if (!run) return [];
  return (run.steps ?? []).filter((step) => step.status === "skipped" || step.status === "failed");
}

/**
 * Tentativas consumidas pelo passo. Retorna vazio quando não houve repetição
 * (ou quando o passo não tenta de novo): só mostramos o que é informação.
 */
export function scheduleStepAttemptsLabel(step: AgentScheduleStepOutcome): string {
  const attempts = step.attempts ?? 0;
  if (attempts <= 1) return "";
  return `${attempts} tentativas`;
}

/** Nome legível do tipo de passo, na mesma linguagem do editor de fluxo. */
export function scheduleStepKindLabel(kind: string): string {
  switch (kind) {
    case "trigger.interval":
      return "gatilho por tempo";
    case "trigger.webhook":
      return "gatilho por webhook";
    case "action.mission":
      return "ação: criar missão";
    case "condition.if":
      return "condição";
    default:
      return kind || "passo";
  }
}
