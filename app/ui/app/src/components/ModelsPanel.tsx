import { useState } from "react";

import { useModels } from "@/hooks/useModels";
import { useModelPull } from "@/hooks/useModelPull";
import { useSettings } from "@/hooks/useSettings";
import Downloading from "@/components/Downloading";
import { RECOMMENDED_FIRST_MODEL } from "@/lib/firstModel";

export const MODELS_PANEL_ERROR_MESSAGE =
  "Não foi possível baixar o modelo agora. Verifique sua conexão e tente novamente.";

// ModelsPanel is the local-model management surface in Settings: see the models
// installed on this computer, pick the active one, and download new ones with
// live progress — no terminal required. Bigger/featured models can be typed in
// directly; the recommended small model is prefilled.
export function ModelsPanel() {
  const { data: models = [] } = useModels();
  const { settings, setSettings } = useSettings();
  const { pulling, progress, status, error, pull, cancel } = useModelPull(
    MODELS_PANEL_ERROR_MESSAGE,
  );

  const [newModel, setNewModel] = useState("");
  const name = (newModel || RECOMMENDED_FIRST_MODEL).trim();

  const downloaded = models.filter((m) => !!m.digest && !m.isCloud());
  const active = settings.selectedModel;

  const download = async () => {
    if (!name) return;
    const ok = await pull(name);
    if (ok) {
      setSettings({ SelectedModel: name });
      setNewModel("");
    }
  };

  return (
    <section
      aria-label="Modelos"
      className="flex flex-col gap-3 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div>
        <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
          Modelos locais
        </h3>
        <p className="mt-1 text-xs text-neutral-600 dark:text-neutral-300">
          Gerencie os modelos instalados no seu computador e baixe novos. O
          modelo ativo é usado no chat e nas missões.
        </p>
      </div>

      {downloaded.length === 0 ? (
        <p className="text-sm text-neutral-500 dark:text-neutral-400">
          Nenhum modelo instalado ainda. Baixe um abaixo para começar.
        </p>
      ) : (
        <ul className="flex flex-col divide-y divide-neutral-100 dark:divide-neutral-800">
          {downloaded.map((model) => (
            <li
              key={model.model}
              className="flex items-center justify-between gap-3 py-2"
            >
              <span className="min-w-0 truncate text-sm text-neutral-800 dark:text-neutral-100">
                {model.model}
              </span>
              {active === model.model ? (
                <span className="shrink-0 rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                  Ativo
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => void setSettings({ SelectedModel: model.model })}
                  className="shrink-0 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  Usar
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      <div className="mt-1 border-t border-neutral-100 pt-3 dark:border-neutral-800">
        <label
          htmlFor="new-model"
          className="block text-xs font-medium text-neutral-700 dark:text-neutral-300"
        >
          Baixar um novo modelo
        </label>
        {!pulling ? (
          <div className="mt-1 flex flex-col gap-2 sm:flex-row">
            <input
              id="new-model"
              value={newModel || RECOMMENDED_FIRST_MODEL}
              onChange={(event) => setNewModel(event.target.value)}
              placeholder={RECOMMENDED_FIRST_MODEL}
              className="min-w-0 flex-1 rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200 dark:border-neutral-700 dark:bg-neutral-900"
            />
            <button
              type="button"
              onClick={() => void download()}
              disabled={!name}
              className="min-h-10 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-semibold text-white hover:bg-neutral-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              Baixar modelo
            </button>
          </div>
        ) : (
          <div className="mt-2">
            <Downloading
              completed={progress.completed}
              total={progress.total}
              label={`Baixando ${name}`}
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
      </div>

      {status && !error && (
        <p
          className="text-xs text-emerald-700 dark:text-emerald-400"
          role="status"
          aria-live="polite"
        >
          {status}
        </p>
      )}
      {error && (
        <p className="text-sm text-red-700 dark:text-red-300" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
