// Side-by-side model comparison.
//
// The comparison surface answers the same prompt with two or more models at
// once so the user can compare latency and answer quality without leaving the
// chat. It is intentionally pure: the fan-out is described by these helpers and
// the component only wires them to `sendMessage`. Every metric here is
// measured, never estimated — we report wall-clock latency and character
// counts because the UI has no tokenizer available, so a "token" number would
// be an invention.

export const MAX_COMPARISON_MODELS = 4;
export const MIN_COMPARISON_MODELS = 2;
export const COMPARISON_MAX_PROMPT_CHARACTERS = 8000;

export type ComparisonStatus =
  | "idle"
  | "streaming"
  | "done"
  | "error"
  | "aborted";

export type ComparisonCandidate = {
  id: string;
  label: string;
  provider?: string;
  costTag?: string;
};

export type ComparisonModelLike = {
  model: string;
  provider?: string;
  cost_tag?: string;
  digest?: string;
  kind?: string;
  available?: boolean;
  capabilities?: string[];
};

export type ComparisonColumn = {
  id: string;
  label: string;
  provider?: string;
  status: ComparisonStatus;
  answer: string;
  thinking: string;
  error?: string;
  chatId?: string;
  startedAt: number;
  firstOutputMs?: number;
  finishedAt?: number;
  characters: number;
};

// Minimal structural view of the streaming events emitted by `sendMessage`.
export type ComparisonStreamEvent = {
  eventName: string;
  content?: string;
  chatId?: string;
  error?: string;
  thinking?: string;
};

/**
 * A model is runnable when it can actually answer right now: not flagged
 * unavailable, and either installed on this computer (digest), served by a
 * configured provider, or an Ollama cloud model. A model that is only a
 * download suggestion would produce a column that always fails, so it is
 * excluded from the comparison picker.
 */
export function isRunnableModel(model: ComparisonModelLike): boolean {
  if (model.available === false) return false;
  if (model.provider) return true;
  if (model.kind === "cloud") return true;
  if (model.model.endsWith("cloud")) return true;
  return Boolean(model.digest);
}

/**
 * Comparison candidates are the locally available chat models. A model that is
 * not installed/available cannot answer, so it is not offered — offering it
 * would produce a column that always errors.
 */
export function comparisonCandidates(
  models: readonly ComparisonModelLike[] | undefined,
): ComparisonCandidate[] {
  if (!models) return [];
  const seen = new Set<string>();
  const candidates: ComparisonCandidate[] = [];
  for (const model of models) {
    const id = typeof model?.model === "string" ? model.model.trim() : "";
    if (!id || seen.has(id)) continue;
    if (!isRunnableModel(model)) continue;
    const capabilities = model.capabilities;
    if (Array.isArray(capabilities) && capabilities.length > 0) {
      if (!capabilities.includes("chat")) continue;
    }
    seen.add(id);
    candidates.push({
      id,
      label: id,
      provider: model.provider,
      costTag: model.cost_tag,
    });
  }
  return candidates.sort((a, b) => a.label.localeCompare(b.label));
}

/**
 * toggleCandidate adds or removes a selection while enforcing the 1..4 bound.
 * Selecting beyond the limit is refused instead of silently evicting a choice.
 */
export function toggleCandidate(
  selected: readonly string[],
  id: string,
): string[] {
  const current = [...selected];
  const index = current.indexOf(id);
  if (index >= 0) {
    current.splice(index, 1);
    return current;
  }
  if (current.length >= MAX_COMPARISON_MODELS) return current;
  current.push(id);
  return current;
}

export function selectionIsRunnable(selected: readonly string[]): boolean {
  return selected.length >= MIN_COMPARISON_MODELS;
}

export function promptIsRunnable(
  prompt: string,
  selected: readonly string[],
): boolean {
  const trimmed = prompt.trim();
  return (
    trimmed.length > 0 &&
    trimmed.length <= COMPARISON_MAX_PROMPT_CHARACTERS &&
    selectionIsRunnable(selected)
  );
}

export function createColumn(
  candidate: ComparisonCandidate,
  startedAt: number,
): ComparisonColumn {
  return {
    id: candidate.id,
    label: candidate.label,
    provider: candidate.provider,
    status: "idle",
    answer: "",
    thinking: "",
    startedAt,
    characters: 0,
  };
}

/**
 * applyStreamEvent folds one stream event into a column. It never mutates the
 * input, and it ignores events that arrive after the column is terminal so a
 * late chunk cannot resurrect a finished or aborted answer.
 */
export function applyStreamEvent(
  column: ComparisonColumn,
  event: ComparisonStreamEvent,
  nowMs: number,
): ComparisonColumn {
  const terminal =
    column.status === "done" ||
    column.status === "error" ||
    column.status === "aborted";
  if (terminal) return column;

  switch (event.eventName) {
    case "chat_created":
      return event.chatId
        ? { ...column, chatId: event.chatId, status: "streaming" }
        : column;
    case "chat":
    case "assistant_with_tools": {
      const chunk = event.content ?? "";
      if (!chunk) return column;
      return {
        ...column,
        status: "streaming",
        answer: column.answer + chunk,
        characters: column.characters + chunk.length,
        firstOutputMs: column.firstOutputMs ?? nowMs - column.startedAt,
      };
    }
    case "thinking": {
      const chunk = event.thinking ?? event.content ?? "";
      if (!chunk) return column;
      return {
        ...column,
        status: "streaming",
        thinking: column.thinking + chunk,
        firstOutputMs: column.firstOutputMs ?? nowMs - column.startedAt,
      };
    }
    case "done":
      return { ...column, status: "done", finishedAt: nowMs };
    case "error":
      return {
        ...column,
        status: "error",
        error: event.error ?? "Falha desconhecida ao consultar o modelo.",
        finishedAt: nowMs,
      };
    default:
      return column;
  }
}

/**
 * abortColumn marks a still-running column as aborted without discarding the
 * partial answer, which is the point of running models side by side.
 */
export function abortColumn(
  column: ComparisonColumn,
  nowMs: number,
): ComparisonColumn {
  if (column.status !== "streaming" && column.status !== "idle") return column;
  return { ...column, status: "aborted", finishedAt: nowMs };
}

export function failColumn(
  column: ComparisonColumn,
  message: string,
  nowMs: number,
): ComparisonColumn {
  if (column.status === "done" || column.status === "aborted") return column;
  return { ...column, status: "error", error: message, finishedAt: nowMs };
}

export function elapsedMs(column: ComparisonColumn, nowMs: number): number {
  const end = column.finishedAt ?? nowMs;
  return Math.max(0, end - column.startedAt);
}

export type ComparisonSummary = {
  answered: number;
  failed: number;
  totalCharacters: number;
  fastestModelId?: string;
  fastestMs?: number;
};

export function summarize(
  columns: readonly ComparisonColumn[],
  nowMs: number,
): ComparisonSummary {
  let answered = 0;
  let failed = 0;
  let totalCharacters = 0;
  let fastestModelId: string | undefined;
  let fastestMs: number | undefined;

  for (const column of columns) {
    totalCharacters += column.characters;
    if (column.status === "done" || column.status === "aborted") answered += 1;
    if (column.status === "error") failed += 1;
    if (column.status !== "done") continue;
    const elapsed = elapsedMs(column, nowMs);
    if (fastestMs === undefined || elapsed < fastestMs) {
      fastestMs = elapsed;
      fastestModelId = column.id;
    }
  }

  return { answered, failed, totalCharacters, fastestModelId, fastestMs };
}

/**
 * formatMilliseconds renders a measured latency. Sub-second values keep
 * milliseconds; longer values use one decimal in seconds. Missing values are
 * rendered as an em dash instead of 0 ms, which would be a false measurement.
 */
export function formatMilliseconds(ms: number | undefined): string {
  if (ms === undefined || !Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)} ms`;
  const seconds = ms / 1000;
  return `${seconds.toFixed(seconds < 10 ? 1 : 0).replace(".", ",")} s`;
}

export function comparisonStatusLabel(status: ComparisonStatus): string {
  switch (status) {
    case "idle":
      return "Aguardando";
    case "streaming":
      return "Respondendo";
    case "done":
      return "Concluído";
    case "error":
      return "Falhou";
    case "aborted":
      return "Interrompido";
  }
}
