import { useCallback, useMemo, useRef, useState } from "react";

import { sendMessage } from "@/api";
import { Model } from "@/gotypes";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { useModels } from "@/hooks/useModels";
import {
  COMPARISON_MAX_PROMPT_CHARACTERS,
  MAX_COMPARISON_MODELS,
  abortColumn,
  applyStreamEvent,
  comparisonCandidates,
  comparisonStatusLabel,
  createColumn,
  elapsedMs,
  failColumn,
  formatMilliseconds,
  promptIsRunnable,
  summarize,
  toggleCandidate,
  type ComparisonCandidate,
  type ComparisonColumn,
  type ComparisonSummary,
} from "@/lib/modelComparison";

// Side-by-side model comparison: one question, N models, N answers.
//
// Every column is a real temporary chat on the same backend route the chat uses
// (`POST /api/v1/chat/{id}` with `temporary: true`), so the question never lands
// in the conversation history. The numbers shown are wall-clock latency and
// character counts — both observable here; token counts are not, so none are
// invented.
//
// The panel is exported on its own so it can be tested without the navigation
// shell; the page below only adds the sidebar layout.
export function ModelComparisonPanel() {
  const { data: models = [] } = useModels();
  const candidates = useMemo(() => comparisonCandidates(models), [models]);
  const byId = useMemo(
    () => new Map(candidates.map((candidate) => [candidate.id, candidate])),
    [candidates],
  );

  const [prompt, setPrompt] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [columns, setColumns] = useState<ComparisonColumn[]>([]);
  const [running, setRunning] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [clock, setClock] = useState(() => Date.now());
  const controllersRef = useRef<Array<AbortController | undefined>>([]);

  const canRun =
    !running &&
    promptIsRunnable(prompt, selected) &&
    selected.every((id) => byId.has(id));

  const runColumn = useCallback(
    async (
      candidate: ComparisonCandidate,
      position: number,
      message: string,
    ) => {
      const controller = new AbortController();
      controllersRef.current[position] = controller;
      const model = new Model({
        model: candidate.id,
        provider: candidate.provider,
      });
      try {
        const stream = sendMessage(
          "new",
          message,
          model,
          undefined,
          controller.signal,
          undefined,
          false,
          false,
          false,
          undefined,
          true,
        );
        for await (const event of stream) {
          const now = Date.now();
          setClock(now);
          setColumns((previous) =>
            previous.map((column, index) =>
              index === position ? applyStreamEvent(column, event, now) : column,
            ),
          );
        }
        const now = Date.now();
        setClock(now);
        setColumns((previous) =>
          previous.map((column, index) => {
            if (index !== position) return column;
            // A stream that ends without a terminal event is only "done" when
            // nobody asked it to stop: an interrupted stream keeps its partial
            // answer and is reported as interrupted.
            if (controller.signal.aborted) return abortColumn(column, now);
            if (column.status !== "streaming") return column;
            return applyStreamEvent(column, { eventName: "done" }, now);
          }),
        );
      } catch (error) {
        const now = Date.now();
        setClock(now);
        const aborted = controller.signal.aborted;
        const failure =
          error instanceof Error && error.message
            ? error.message
            : "Falha ao consultar este modelo.";
        setColumns((previous) =>
          previous.map((column, index) => {
            if (index !== position) return column;
            return aborted
              ? abortColumn(column, now)
              : failColumn(column, failure, now);
          }),
        );
      }
    },
    [],
  );

  const start = useCallback(async () => {
    if (!promptIsRunnable(prompt, selected)) return;
    const chosen = selected
      .map((id) => byId.get(id))
      .filter((candidate): candidate is ComparisonCandidate =>
        Boolean(candidate),
      );
    if (chosen.length < 2) {
      setNotice(
        "Escolha pelo menos dois modelos disponíveis neste computador para comparar.",
      );
      return;
    }
    const message = prompt.trim();
    const startedAt = Date.now();
    setNotice(null);
    setRunning(true);
    controllersRef.current = [];
    setColumns(chosen.map((candidate) => createColumn(candidate, startedAt)));
    setClock(startedAt);
    try {
      await Promise.all(
        chosen.map((candidate, position) =>
          runColumn(candidate, position, message),
        ),
      );
    } finally {
      controllersRef.current = [];
      setRunning(false);
      setClock(Date.now());
    }
  }, [byId, prompt, runColumn, selected]);

  const stop = useCallback(() => {
    for (const controller of controllersRef.current) {
      controller?.abort();
    }
  }, []);

  const clear = useCallback(() => {
    setColumns([]);
    setNotice(null);
    setClock(Date.now());
  }, []);

  const summary = summarize(columns, clock);

  return (
    <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
      <div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12">
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            Comparar modelos
          </h2>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            Envie a mesma pergunta para até {MAX_COMPARISON_MODELS} modelos ao
            mesmo tempo e leia as respostas lado a lado. Cada coluna roda em uma
            conversa temporária: nada aqui entra no seu histórico.
          </p>

          <form
            aria-label="Nova comparação"
            onSubmit={(event) => {
              event.preventDefault();
              void start();
            }}
            className="mt-8 flex flex-col gap-4 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
          >
            <div className="flex flex-col gap-2">
              <label
                htmlFor="comparison-prompt"
                className="text-sm font-medium text-neutral-800 dark:text-neutral-100"
              >
                Pergunta para comparar
              </label>
              <textarea
                id="comparison-prompt"
                aria-label="Pergunta para comparar"
                rows={4}
                maxLength={COMPARISON_MAX_PROMPT_CHARACTERS}
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                placeholder="Ex.: explique o que é uma fila de mensagens mortas"
                className="w-full resize-y rounded-xl border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 outline-none placeholder:text-neutral-400 focus:border-violet-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
              />
              <span className="text-[11px] text-neutral-500 dark:text-neutral-400">
                {prompt.trim().length}/{COMPARISON_MAX_PROMPT_CHARACTERS}{" "}
                caracteres
              </span>
            </div>

            <fieldset className="flex flex-col gap-2">
              <legend className="text-sm font-medium text-neutral-800 dark:text-neutral-100">
                Modelos ({selected.length}/{MAX_COMPARISON_MODELS})
              </legend>
              {candidates.length === 0 ? (
                <p className="text-sm text-neutral-500 dark:text-neutral-400">
                  Nenhum modelo executável encontrado. Baixe um modelo em
                  Ajustes → Modelos ou configure um provedor para comparar.
                </p>
              ) : (
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                  {candidates.map((candidate) => {
                    const checked = selected.includes(candidate.id);
                    const blocked =
                      !checked && selected.length >= MAX_COMPARISON_MODELS;
                    return (
                      <label
                        key={candidate.id}
                        className="flex min-w-0 items-center gap-2 rounded-xl border border-neutral-200 px-3 py-2 text-sm text-neutral-800 dark:border-neutral-700 dark:text-neutral-100"
                      >
                        <input
                          type="checkbox"
                          aria-label={`Comparar com ${candidate.label}`}
                          checked={checked}
                          disabled={running || blocked}
                          onChange={() =>
                            setSelected((previous) =>
                              toggleCandidate(previous, candidate.id),
                            )
                          }
                          className="h-4 w-4 shrink-0"
                        />
                        <span className="min-w-0 flex-1 truncate">
                          {candidate.label}
                        </span>
                        {candidate.provider ? (
                          <span className="shrink-0 rounded-full bg-violet-100 px-2 py-0.5 text-[11px] font-medium text-violet-700 dark:bg-violet-900/40 dark:text-violet-300">
                            {candidate.provider}
                          </span>
                        ) : null}
                        {candidate.costTag ? (
                          <span className="shrink-0 text-[11px] text-neutral-500 dark:text-neutral-400">
                            {candidate.costTag}
                          </span>
                        ) : null}
                      </label>
                    );
                  })}
                </div>
              )}
            </fieldset>

            <div className="flex flex-wrap items-center gap-2">
              <button
                type="submit"
                disabled={!canRun}
                className="inline-flex items-center justify-center gap-2 rounded-xl bg-neutral-950 px-4 py-2.5 text-sm font-medium text-white shadow-sm disabled:opacity-50 dark:bg-white dark:text-neutral-950"
              >
                {running ? "Comparando…" : "Comparar modelos"}
              </button>
              <button
                type="button"
                onClick={stop}
                disabled={!running}
                className="rounded-xl border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-700 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200"
              >
                Interromper
              </button>
              <button
                type="button"
                onClick={clear}
                disabled={columns.length === 0}
                className="rounded-xl border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-700 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200"
              >
                Limpar
              </button>
            </div>
          </form>

          {notice ? (
            <div
              role="status"
              aria-live="polite"
              className="mt-5 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs leading-5 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200"
            >
              {notice}
            </div>
          ) : null}

          {columns.length > 0 ? (
            <section aria-label="Respostas comparadas" className="mt-8">
              <p
                role="status"
                aria-live="polite"
                className="text-xs text-neutral-500 dark:text-neutral-400"
              >
                {summaryLine(summary, columns, running)}
              </p>
              <div className="mt-4 grid gap-4 lg:grid-cols-2">
                {columns.map((column) => (
                  <article
                    key={column.id}
                    aria-label={`Resposta de ${column.label}`}
                    className="flex min-w-0 flex-col gap-3 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
                  >
                    <header className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <h3 className="truncate text-sm font-semibold text-neutral-900 dark:text-white">
                          {column.label}
                        </h3>
                        <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                          {comparisonStatusLabel(column.status)}
                          {column.provider ? ` · ${column.provider}` : ""}
                        </p>
                      </div>
                      <span className="shrink-0 rounded-full bg-neutral-100 px-2.5 py-1 text-[11px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300">
                        {formatMilliseconds(elapsedMs(column, clock))}
                      </span>
                    </header>

                    <div className="whitespace-pre-wrap text-sm leading-6 text-neutral-800 dark:text-neutral-100">
                      {column.answer ||
                        (column.status === "streaming"
                          ? "Aguardando a primeira resposta…"
                          : "")}
                    </div>

                    {column.error ? (
                      <p
                        role="alert"
                        className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-xs leading-5 text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300"
                      >
                        {column.error}
                      </p>
                    ) : null}

                    <dl className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-neutral-500 dark:text-neutral-400">
                      <div className="flex gap-1">
                        <dt>Primeiro trecho:</dt>
                        <dd className="font-medium text-neutral-700 dark:text-neutral-200">
                          {formatMilliseconds(column.firstOutputMs)}
                        </dd>
                      </div>
                      <div className="flex gap-1">
                        <dt>Caracteres:</dt>
                        <dd className="font-medium text-neutral-700 dark:text-neutral-200">
                          {column.characters}
                        </dd>
                      </div>
                      <div className="flex gap-1">
                        <dt>Total:</dt>
                        <dd className="font-medium text-neutral-700 dark:text-neutral-200">
                          {formatMilliseconds(elapsedMs(column, clock))}
                        </dd>
                      </div>
                    </dl>
                  </article>
                ))}
              </div>
              <p className="mt-4 text-[11px] leading-5 text-neutral-500 dark:text-neutral-400">
                Latência medida aqui é tempo de parede (primeiro trecho e total)
                e contagem de caracteres — os dois observáveis por este caminho.
                Contagem de tokens não é exibida porque este fluxo não expõe um
                tokenizer confiável.
              </p>
            </section>
          ) : null}
        </div>
      </div>
  );
}

export function ModelComparisonPage() {
  return (
    <SidebarLayout
      title="Comparar modelos"
      sidebar={<AppSidebar current="compare" />}
    >
      <ModelComparisonPanel />
    </SidebarLayout>
  );
}

function summaryLine(
  summary: ComparisonSummary,
  columns: ComparisonColumn[],
  running: boolean,
): string {
  const fastest = columns.find(
    (column) => column.id === summary.fastestModelId,
  );
  if (running) {
    return `Consultando ${columns.length} modelos… ${summary.answered} concluído(s).`;
  }
  if (summary.answered === 0) {
    return "Nenhum modelo concluiu a resposta.";
  }
  if (fastest && summary.fastestMs !== undefined) {
    return `${summary.answered} de ${columns.length} responderam. Mais rápido: ${fastest.label} em ${formatMilliseconds(summary.fastestMs)}.`;
  }
  return `${summary.answered} de ${columns.length} responderam.`;
}
