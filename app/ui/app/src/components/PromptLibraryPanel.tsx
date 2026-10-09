import { useCallback, useEffect, useMemo, useState } from "react";
import {
  MAX_PROMPT_BODY_CHARACTERS,
  MAX_PROMPT_TITLE_CHARACTERS,
  addPrompt,
  createPrompt,
  deletePrompt,
  duplicatePrompt,
  extractPromptVariables,
  filterPrompts,
  importLibrary,
  libraryFolders,
  libraryTags,
  loadPromptLibrary,
  promptValidationError,
  recordPromptUse,
  renderPrompt,
  savePromptLibrary,
  serializeLibrary,
  togglePromptFavorite,
  updatePrompt,
  type PromptLibrary as PromptLibraryType,
  type PromptRecord,
  type PromptStorage,
} from "@/lib/promptLibrary";

type Draft = {
  id: string | null;
  title: string;
  body: string;
  folder: string;
  tags: string;
};

const EMPTY_DRAFT: Draft = { id: null, title: "", body: "", folder: "", tags: "" };

type Feedback = { kind: "ok" | "error"; message: string } | null;

function draftFromRecord(record: PromptRecord): Draft {
  return {
    id: record.id,
    title: record.title,
    body: record.body,
    folder: record.folder,
    tags: record.tags.join(", "),
  };
}

function parseTags(raw: string): string[] {
  return raw
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function downloadLibrary(library: PromptLibraryType): void {
  try {
    const blob = new Blob([serializeLibrary(library)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "hades-prompt-library.json";
    link.click();
    URL.revokeObjectURL(url);
  } catch {
    // Sem DOM/Blob disponível a exportação é apenas ignorada.
  }
}

const inputClass =
  "w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-800 outline-none transition focus:border-neutral-400 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-100 dark:focus:border-neutral-600";
const labelClass = "block text-xs font-medium text-neutral-500 dark:text-neutral-400";
const buttonClass =
  "rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-600 transition hover:border-neutral-400 hover:text-neutral-900 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:border-neutral-600 dark:hover:text-white";
const primaryButtonClass =
  "rounded-lg border border-violet-200 bg-violet-50 px-3 py-1.5 text-xs font-medium text-violet-700 transition hover:border-violet-300 hover:bg-violet-100 disabled:opacity-50 dark:border-violet-900/60 dark:bg-violet-950/20 dark:text-violet-300 dark:hover:bg-violet-950/40";

/**
 * Biblioteca de prompts reutilizáveis. O painel não renderiza shell nem
 * sidebar, para poder ser testado isoladamente e embutido em qualquer tela.
 * Todo estado é local e persistido em `localStorage`.
 */
export function PromptLibraryPanel({
  onUse,
  storage,
  className = "",
}: {
  onUse?: (text: string, prompt: PromptRecord) => void;
  /** Armazenamento injetável; sem ele, usa `localStorage` quando existir. */
  storage?: PromptStorage;
  className?: string;
}) {
  const [library, setLibrary] = useState<PromptLibraryType>(() => loadPromptLibrary(storage));
  const [query, setQuery] = useState("");
  const [favoritesOnly, setFavoritesOnly] = useState(false);
  const [folderFilter, setFolderFilter] = useState("");
  const [tagFilter, setTagFilter] = useState("");
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [feedback, setFeedback] = useState<Feedback>(null);
  const [useTarget, setUseTarget] = useState<string | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [importText, setImportText] = useState("");
  const [importOpen, setImportOpen] = useState(false);

  const folders = useMemo(() => libraryFolders(library), [library]);
  const tags = useMemo(() => libraryTags(library), [library]);
  const visible = useMemo(
    () =>
      filterPrompts(library.prompts, {
        query,
        favoritesOnly,
        folder: folderFilter || undefined,
        tag: tagFilter || undefined,
      }),
    [library, query, favoritesOnly, folderFilter, tagFilter],
  );

  const persist = useCallback(
    (next: PromptLibraryType) => {
      setLibrary(next);
      savePromptLibrary(next, storage);
    },
    [storage],
  );

  useEffect(() => {
    if (useTarget && !library.prompts.some((prompt) => prompt.id === useTarget)) {
      setUseTarget(null);
    }
  }, [library, useTarget]);

  const activePrompt = useTarget
    ? library.prompts.find((prompt) => prompt.id === useTarget) ?? null
    : null;
  const activeVariables = activePrompt ? extractPromptVariables(activePrompt.body) : [];
  const activeRender = activePrompt ? renderPrompt(activePrompt.body, values) : null;

  const handleSubmit = useCallback(() => {
    const input = {
      title: draft.title,
      body: draft.body,
      folder: draft.folder,
      tags: parseTags(draft.tags),
    };
    const error = promptValidationError(input);
    if (error) {
      setFeedback({ kind: "error", message: error });
      return;
    }
    setFeedback(null);
    if (draft.id) {
      persist(updatePrompt(library, draft.id, input, Date.now()));
      setFeedback({ kind: "ok", message: "Prompt atualizado." });
    } else {
      const record = createPrompt(input, Date.now());
      persist(addPrompt(library, record));
      setFeedback({ kind: "ok", message: "Prompt salvo na biblioteca." });
    }
    setDraft(EMPTY_DRAFT);
  }, [draft, library, persist]);

  const handleUse = useCallback(
    (prompt: PromptRecord) => {
      const variables = extractPromptVariables(prompt.body);
      if (variables.length > 0 && useTarget !== prompt.id) {
        setUseTarget(prompt.id);
        setValues({});
        setFeedback(null);
        return;
      }
      const result = renderPrompt(prompt.body, values);
      if (result.missing.length > 0) {
        setFeedback({
          kind: "error",
          message: `Preencha: ${result.missing.join(", ")}`,
        });
        return;
      }
      if (onUse) onUse(result.text, prompt);
      else {
        try {
          void navigator.clipboard?.writeText(result.text);
        } catch {
          // Sem área de transferência o texto permanece visível na lista.
        }
      }
      persist(recordPromptUse(library, prompt.id, Date.now()));
      setUseTarget(null);
      setValues({});
      setFeedback({
        kind: "ok",
        message: onUse
          ? "Prompt enviado para o campo de mensagem."
          : "Prompt copiado para a área de transferência.",
      });
    },
    [library, onUse, persist, useTarget, values],
  );

  const handleImport = useCallback(() => {
    const result = importLibrary(library, importText, Date.now());
    if (result.error) {
      setFeedback({ kind: "error", message: result.error });
      return;
    }
    persist(result.library);
    setImportText("");
    setImportOpen(false);
    setFeedback({ kind: "ok", message: `${result.imported} prompt(s) importado(s).` });
  }, [importText, library, persist]);

  return (
    <section
      aria-label="Biblioteca de prompts"
      className={`rounded-2xl border border-neutral-200 bg-white p-4 text-left dark:border-neutral-800 dark:bg-neutral-950 ${className}`}
    >
      <header className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-neutral-800 dark:text-neutral-200">
            Biblioteca de prompts
          </h2>
          <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
            {library.prompts.length} prompt(s) salvos neste navegador.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button type="button" className={buttonClass} onClick={() => downloadLibrary(library)}>
            Exportar
          </button>
          <button
            type="button"
            className={buttonClass}
            aria-expanded={importOpen}
            onClick={() => setImportOpen((open) => !open)}
          >
            Importar
          </button>
        </div>
      </header>

      {importOpen && (
        <div className="mt-3 rounded-xl border border-neutral-200 p-3 dark:border-neutral-800">
          <label className={labelClass} htmlFor="prompt-import">
            Cole o JSON exportado
          </label>
          <textarea
            id="prompt-import"
            className={`${inputClass} mt-1 h-24 font-mono text-xs`}
            value={importText}
            onChange={(event) => setImportText(event.target.value)}
          />
          <button
            type="button"
            className={`${primaryButtonClass} mt-2`}
            disabled={!importText.trim()}
            onClick={handleImport}
          >
            Confirmar importação
          </button>
        </div>
      )}

      {feedback && (
        <p
          role={feedback.kind === "error" ? "alert" : "status"}
          className={`mt-3 text-xs ${
            feedback.kind === "error"
              ? "text-red-600 dark:text-red-400"
              : "text-emerald-600 dark:text-emerald-400"
          }`}
        >
          {feedback.message}
        </p>
      )}

      <form
        className="mt-4 grid gap-3 md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          handleSubmit();
        }}
      >
        <div className="md:col-span-2">
          <label className={labelClass} htmlFor="prompt-title">
            Título do prompt
          </label>
          <input
            id="prompt-title"
            className={`${inputClass} mt-1`}
            maxLength={MAX_PROMPT_TITLE_CHARACTERS}
            value={draft.title}
            onChange={(event) => setDraft({ ...draft, title: event.target.value })}
          />
        </div>
        <div className="md:col-span-2">
          <label className={labelClass} htmlFor="prompt-body">
            Conteúdo do prompt (use {"{{variavel}}"} para campos variáveis)
          </label>
          <textarea
            id="prompt-body"
            className={`${inputClass} mt-1 h-24`}
            maxLength={MAX_PROMPT_BODY_CHARACTERS}
            value={draft.body}
            onChange={(event) => setDraft({ ...draft, body: event.target.value })}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="prompt-folder">
            Pasta do prompt
          </label>
          <input
            id="prompt-folder"
            className={`${inputClass} mt-1`}
            value={draft.folder}
            onChange={(event) => setDraft({ ...draft, folder: event.target.value })}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="prompt-tags">
            Etiquetas do prompt (separadas por vírgula)
          </label>
          <input
            id="prompt-tags"
            className={`${inputClass} mt-1`}
            value={draft.tags}
            onChange={(event) => setDraft({ ...draft, tags: event.target.value })}
          />
        </div>
        <div className="flex gap-2 md:col-span-2">
          <button type="submit" className={primaryButtonClass}>
            {draft.id ? "Atualizar prompt" : "Salvar prompt"}
          </button>
          {draft.id && (
            <button type="button" className={buttonClass} onClick={() => setDraft(EMPTY_DRAFT)}>
              Cancelar edição
            </button>
          )}
        </div>
      </form>

      <div className="mt-4 flex flex-wrap items-end gap-3">
        <div className="min-w-48 flex-1">
          <label className={labelClass} htmlFor="prompt-library-search">
            Buscar prompts
          </label>
          <input
            id="prompt-library-search"
            className={`${inputClass} mt-1`}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="prompt-library-folder">
            Filtrar por pasta
          </label>
          <select
            id="prompt-library-folder"
            className={`${inputClass} mt-1`}
            value={folderFilter}
            onChange={(event) => setFolderFilter(event.target.value)}
          >
            <option value="">Todas as pastas</option>
            {folders.map((folder) => (
              <option key={folder} value={folder}>
                {folder}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="prompt-library-tag">
            Filtrar por etiqueta
          </label>
          <select
            id="prompt-library-tag"
            className={`${inputClass} mt-1`}
            value={tagFilter}
            onChange={(event) => setTagFilter(event.target.value)}
          >
            <option value="">Todas as etiquetas</option>
            {tags.map((tag) => (
              <option key={tag} value={tag}>
                {tag}
              </option>
            ))}
          </select>
        </div>
        <label className="flex items-center gap-2 text-xs text-neutral-500 dark:text-neutral-400">
          <input
            type="checkbox"
            checked={favoritesOnly}
            onChange={(event) => setFavoritesOnly(event.target.checked)}
          />
          Somente favoritos
        </label>
      </div>

      <ul className="mt-4 space-y-2" aria-label="Prompts salvos">
        {visible.length === 0 && (
          <li className="rounded-xl border border-dashed border-neutral-200 p-4 text-xs text-neutral-500 dark:border-neutral-800 dark:text-neutral-400">
            Nenhum prompt encontrado com os filtros atuais.
          </li>
        )}
        {visible.map((prompt) => (
          <li
            key={prompt.id}
            data-prompt-id={prompt.id}
            className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="min-w-48 flex-1">
                <p className="text-sm font-medium text-neutral-800 dark:text-neutral-100">
                  {prompt.title}
                  {prompt.builtin && (
                    <span className="ml-2 rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] uppercase tracking-wide text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
                      Inicial
                    </span>
                  )}
                </p>
                <p className="mt-1 whitespace-pre-wrap text-xs text-neutral-500 dark:text-neutral-400">
                  {prompt.body}
                </p>
                <p className="mt-2 text-[11px] text-neutral-400">
                  {prompt.folder || "Sem pasta"}
                  {prompt.tags.length > 0 ? ` · ${prompt.tags.join(", ")}` : ""} · usos: {prompt.uses}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  className={primaryButtonClass}
                  aria-label={`Usar o prompt ${prompt.title}`}
                  onClick={() => handleUse(prompt)}
                >
                  Usar
                </button>
                <button
                  type="button"
                  className={buttonClass}
                  aria-pressed={prompt.favorite}
                  aria-label={`Alternar favorito de ${prompt.title}`}
                  onClick={() => persist(togglePromptFavorite(library, prompt.id))}
                >
                  {prompt.favorite ? "★" : "☆"}
                </button>
                <button
                  type="button"
                  className={buttonClass}
                  aria-label={`Editar ${prompt.title}`}
                  onClick={() => {
                    setDraft(draftFromRecord(prompt));
                    setFeedback(null);
                  }}
                >
                  Editar
                </button>
                <button
                  type="button"
                  className={buttonClass}
                  aria-label={`Duplicar ${prompt.title}`}
                  onClick={() => persist(duplicatePrompt(library, prompt.id, Date.now()))}
                >
                  Duplicar
                </button>
                <button
                  type="button"
                  className={buttonClass}
                  aria-label={`Excluir ${prompt.title}`}
                  onClick={() => {
                    persist(deletePrompt(library, prompt.id));
                    setFeedback({ kind: "ok", message: "Prompt excluído." });
                  }}
                >
                  Excluir
                </button>
              </div>
            </div>

            {activePrompt?.id === prompt.id && activeVariables.length > 0 && (
              <div className="mt-3 rounded-lg border border-violet-200 bg-violet-50/40 p-3 dark:border-violet-900/60 dark:bg-violet-950/10">
                <div className="grid gap-2 md:grid-cols-2">
                  {activeVariables.map((name) => (
                    <div key={name}>
                      <label className={labelClass} htmlFor={`prompt-value-${prompt.id}-${name}`}>
                        {`Valor de ${name}`}
                      </label>
                      <input
                        id={`prompt-value-${prompt.id}-${name}`}
                        className={`${inputClass} mt-1`}
                        value={values[name] ?? ""}
                        onChange={(event) =>
                          setValues({ ...values, [name]: event.target.value })
                        }
                      />
                    </div>
                  ))}
                </div>
                <p className="mt-2 whitespace-pre-wrap text-xs text-neutral-500 dark:text-neutral-400">
                  {activeRender?.text}
                </p>
                <button
                  type="button"
                  className={`${primaryButtonClass} mt-2`}
                  onClick={() => handleUse(prompt)}
                >
                  Gerar e usar
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

export default PromptLibraryPanel;
