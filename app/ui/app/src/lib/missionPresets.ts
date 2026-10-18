/**
 * Presets de MISSÃO (FASE 23 do adendo V3).
 *
 * A fonte de verdade é o backend: `GET /api/agent/v1/presets` serve
 * `agent.BuiltInMissionPresets()`. Este ficheiro NÃO tem catálogo próprio de propósito --
 * *uma segunda lista com o mesmo nome e conteúdo diferente mente em silêncio*.
 *
 * A distinção que interessa ao produto: um preset de MOTOR (`internal/multillm/presets.go`)
 * escolhe onde a inferência corre; um preset de MISSÃO escolhe o que a missão faz, quanto
 * pode gastar, que permissões pede e quando se considera concluída.
 */

/** Espelha `agent.MissionPreset` do backend. Campos opcionais no JSON ficam opcionais aqui. */
export type MissionPreset = {
  id: string;
  name: string;
  description: string;
  goal: string;
  agents?: string[];
  models_pref?: string[];
  tools?: string[];
  skills?: string[];
  mcps?: string[];
  memory?: string[];
  environment?: string;
  permissions?: string[];
  approvals?: string[];
  budget_seconds?: number;
  budget_tokens?: number;
  metrics?: string[];
  tests?: string[];
  completion_criteria?: string[];
  schedule?: string;
};

/** O que o `/preset` precisa de saber para se apresentar honestamente. */
export type PresetSummary = {
  /** Rótulo curto de orçamento, ou `null` quando o preset não declara orçamento. */
  budget: string | null;
  /** Quantas permissões o preset PEDE (pedir não é receber: o policy engine decide). */
  permissionCount: number;
  /** Quantos critérios de conclusão o preset declara. */
  criteriaCount: number;
  /** `true` quando o preset declara um agendamento (briefing diário, monitor). */
  scheduled: boolean;
};

const SCHEDULE_LABEL: Record<string, string> = {
  daily: "diário",
  weekly: "semanal",
  hourly: "de hora a hora",
  once: "uma vez",
};

function formatSeconds(seconds: number): string {
  if (seconds <= 0) return "sem limite de tempo";
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} min`;
  const hours = Math.round(minutes / 60);
  return `${hours} h`;
}

function formatTokens(tokens: number): string {
  if (tokens <= 0) return "";
  if (tokens < 1000) return `${tokens} tokens`;
  const thousands = Math.round(tokens / 1000);
  return `${thousands}k tokens`;
}

/**
 * Resume um preset para apresentação. Não inventa valores: um preset que não declara
 * orçamento devolve `budget: null`, e a UI diz «sem orçamento declarado» em vez de «0».
 * *Mostrar zero onde não há número é uma mentira de formatação.*
 */
export function summarizePreset(preset: MissionPreset): PresetSummary {
  const parts: string[] = [];
  const seconds = preset.budget_seconds ?? 0;
  const tokens = preset.budget_tokens ?? 0;
  if (seconds > 0) parts.push(formatSeconds(seconds));
  const tokensLabel = formatTokens(tokens);
  if (tokensLabel) parts.push(tokensLabel);
  return {
    budget: parts.length > 0 ? parts.join(" · ") : null,
    permissionCount: preset.permissions?.length ?? 0,
    criteriaCount: preset.completion_criteria?.length ?? 0,
    scheduled: (preset.schedule ?? "").trim() !== "",
  };
}

/** Texto do agendamento em pt-BR, ou `null` quando o preset não é agendado. */
export function describeSchedule(preset: MissionPreset): string | null {
  const schedule = (preset.schedule ?? "").trim();
  if (!schedule) return null;
  return SCHEDULE_LABEL[schedule] ?? schedule;
}

/**
 * Valida a resposta do backend antes de a UI confiar nela. O servidor é a autoridade, mas
 * a UI não pode desenhar um preset sem `id` ou sem critério de conclusão: *um preset sem
 * critério de conclusão é um preset que nunca acaba.*
 */
export function parseMissionPresets(payload: unknown): MissionPreset[] {
  if (!Array.isArray(payload)) return [];
  const out: MissionPreset[] = [];
  for (const raw of payload) {
    if (!raw || typeof raw !== "object") continue;
    const candidate = raw as Partial<MissionPreset>;
    if (typeof candidate.id !== "string" || candidate.id.trim() === "") continue;
    if (!Array.isArray(candidate.completion_criteria) || candidate.completion_criteria.length === 0) continue;
    out.push({
      ...candidate,
      id: candidate.id,
      name: candidate.name ?? candidate.id,
      description: candidate.description ?? "",
      goal: candidate.goal ?? "",
    });
  }
  return out;
}