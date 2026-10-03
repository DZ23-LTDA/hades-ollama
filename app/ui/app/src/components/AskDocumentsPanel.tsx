import { useState, type FormEvent } from "react";

import { askProjectDocuments, type AskDocumentsResult } from "@/lib/agenticClient";

export const ASK_NO_SOURCES_MESSAGE =
  "Não encontrei essa informação nos seus documentos.";
export const ASK_ERROR_MESSAGE =
  "Não foi possível consultar os documentos agora. Tente novamente.";

// This panel retrieves grounded excerpts from a project's indexed documents;
// it does not call a model to synthesize an answer from those excerpts.
export function AskDocumentsPanel({ projectId }: { projectId: string }) {
  const [query, setQuery] = useState("");
  const [result, setResult] = useState<AskDocumentsResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const q = query.trim();
    if (!q || loading) return;
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      setResult(await askProjectDocuments(projectId, q));
    } catch {
      setError(ASK_ERROR_MESSAGE);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section aria-label="Buscar nos documentos" className="flex flex-col gap-3">
      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <label htmlFor="ask-query" className="text-sm font-medium text-neutral-800 dark:text-neutral-100">
          Buscar nos documentos
        </label>
        <p className="text-xs text-neutral-500 dark:text-neutral-400">Localiza trechos relevantes e mostra as fontes; não gera uma resposta de IA.</p>
        <div className="flex flex-wrap gap-2">
          <input
            id="ask-query"
            type="text"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Ex.: Qual a política de férias?"
            className="min-h-10 flex-1 rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          />
          <button
            type="submit"
            disabled={loading || query.trim() === ""}
            className="min-h-10 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "Buscando trechos…" : "Buscar"}
          </button>
        </div>
      </form>

      <div aria-live="polite" className="flex flex-col gap-2 text-sm">
        {error ? (
          <p role="alert" className="text-red-700 dark:text-red-300">
            {error}
          </p>
        ) : null}

        {result && !result.grounded ? (
          <p className="text-neutral-700 dark:text-neutral-300">{ASK_NO_SOURCES_MESSAGE}</p>
        ) : null}

        {result && result.grounded ? (
          <div className="flex flex-col gap-2">
            <ul aria-label="Trechos e fontes encontrados" className="flex flex-col gap-3">
              {result.citations.map((citation) => (
                <li key={citation.index} className="rounded-lg border border-neutral-200 p-3 text-neutral-700 dark:border-neutral-800 dark:text-neutral-300">
                  <p><span className="font-semibold">[{citation.index}]</span> {citation.source}</p>
                  <p className="mt-1 whitespace-pre-wrap text-xs leading-5 text-neutral-600 dark:text-neutral-400">{citation.snippet?.trim() || "Fonte localizada; não há trecho disponível para exibição."}</p>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </div>
    </section>
  );
}
