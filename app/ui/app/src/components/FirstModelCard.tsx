import { useMemo, useRef, useState } from "react";
import { ArrowDownTrayIcon } from "@heroicons/react/24/outline";

import { pullModel } from "@/api";
import { useModels, useRefetchModels } from "@/hooks/useModels";
import { useFeaturedModels } from "@/hooks/useFeaturedModels";
import { useSettings } from "@/hooks/useSettings";
import Downloading from "@/components/Downloading";
import { FIRST_MODEL_ERROR_MESSAGE, pickFirstModel } from "@/lib/firstModel";

// FirstModelCard is the zero-model first-run surface: when the local backend has
// no downloaded model yet, it offers the recommended first model and downloads it
// with live progress using the existing pull pipeline, then selects it so the
// user can start immediately — no terminal required.
export function FirstModelCard() {
  const { data: models = [], isLoading } = useModels();
  const { data: recommendations } = useFeaturedModels();
  const { setSettings } = useSettings();
  const refetchModels = useRefetchModels();

  const recommended = useMemo(
    () => pickFirstModel(recommendations),
    [recommendations],
  );
  const [modelName, setModelName] = useState("");
  const effectiveName = (modelName || recommended).trim();

  const [pulling, setPulling] = useState(false);
  const [progress, setProgress] = useState<{ completed: number; total: number }>({
    completed: 0,
    total: 0,
  });
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [skipped, setSkipped] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const hasLocalModel = models.some((m) => !!m.digest);
  // Only surface the card once we actually know there are no local models — never
  // while still loading (avoids a flash of "no models" during startup).
  if (isLoading || hasLocalModel || skipped) return null;

  const download = async () => {
    if (!effectiveName || pulling) return;
    const controller = new AbortController();
    abortRef.current = controller;
    setPulling(true);
    setError(null);
    setStatus("Preparando download…");
    setProgress({ completed: 0, total: 0 });
    try {
      for await (const event of pullModel(effectiveName, controller.signal)) {
        setProgress({
          completed: event.completed ?? 0,
          total: event.total ?? 0,
        });
        if (event.status) setStatus(event.status);
      }
      setStatus(`Modelo ${effectiveName} pronto.`);
      setSettings({ SelectedModel: effectiveName });
      await refetchModels();
    } catch (err) {
      if (controller.signal.aborted) {
        setStatus(null);
        return;
      }
      setError(
        err instanceof Error && err.message
          ? `${FIRST_MODEL_ERROR_MESSAGE} (${err.message})`
          : FIRST_MODEL_ERROR_MESSAGE,
      );
      setStatus(null);
    } finally {
      setPulling(false);
      abortRef.current = null;
    }
  };

  const cancel = () => abortRef.current?.abort();

  return (
    <section
      aria-label="Primeiro modelo"
      className="mx-auto mb-6 w-full max-w-3xl rounded-2xl border border-violet-200 bg-violet-50 p-5 dark:border-violet-900/60 dark:bg-violet-950/20"
    >
      <div className="flex items-start gap-4">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300">
          <ArrowDownTrayIcon className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <h2 className="font-medium text-neutral-900 dark:text-white">
            Baixe seu primeiro modelo para começar
          </h2>
          <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-300">
            Você ainda não tem nenhum modelo local instalado. Baixe o modelo
            recomendado e já pode conversar e rodar missões — tudo no seu
            computador, sem terminal.
          </p>

          {!pulling && (
            <div className="mt-4 flex flex-col gap-2 sm:flex-row sm:items-end">
              <div className="flex-1">
                <label
                  htmlFor="first-model"
                  className="block text-xs font-medium text-neutral-700 dark:text-neutral-300"
                >
                  Modelo local
                </label>
                <input
                  id="first-model"
                  value={modelName || recommended}
                  onChange={(event) => setModelName(event.target.value)}
                  className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200 dark:border-neutral-700 dark:bg-neutral-900"
                  placeholder={recommended}
                />
              </div>
              <button
                type="button"
                onClick={() => void download()}
                disabled={!effectiveName}
                className="min-h-10 rounded-xl bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Baixar e começar
              </button>
              <button
                type="button"
                onClick={() => setSkipped(true)}
                className="min-h-10 rounded-xl px-3 py-2 text-sm font-medium text-neutral-600 hover:bg-violet-100 dark:text-neutral-300 dark:hover:bg-violet-950/40"
              >
                Pular por enquanto
              </button>
            </div>
          )}

          {pulling && (
            <div className="mt-4">
              <Downloading
                completed={progress.completed}
                total={progress.total}
                label={`Baixando ${effectiveName}`}
              />
              <button
                type="button"
                onClick={cancel}
                className="mt-1 ml-6 text-xs font-medium text-neutral-500 underline hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200"
              >
                Cancelar
              </button>
            </div>
          )}

          {status && !error && (
            <p
              className="mt-3 text-xs text-emerald-700 dark:text-emerald-400"
              role="status"
              aria-live="polite"
            >
              {status}
            </p>
          )}
          {error && (
            <p
              className="mt-3 text-sm text-red-700 dark:text-red-300"
              role="alert"
            >
              {error}
            </p>
          )}
        </div>
      </div>
    </section>
  );
}
