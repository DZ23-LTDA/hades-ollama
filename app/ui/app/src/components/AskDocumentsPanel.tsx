import { useState, type FormEvent } from "react";

import { askProjectDocuments, type AskDocumentsResult } from "@/lib/agenticClient";

export const ASK_NO_SOURCES_MESSAGE =
  "Não encontrei essa informação nos seus documentos.";
export const ASK_ERROR_MESSAGE =
  "Não foi possível consultar os documentos agora. Tente novamente.";

// AskDocumentsPanel lets the user ask a question answered strictly from a
// project's indexed documents (RAG, G1). When the backend reports grounded=false
// it shows an honest "não encontrei" message instead of a fabricated answer, and
// always lists the cited sources.
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
    <section aria-label="Perguntar aos documentos" className="flex flex-col gap-3">
      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <label htmlFor="ask-query" className="text-sm font-medium text-neutral-800 dark:text-neutral-100">
          Pergunte aos seus documentos
        </label>
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
            {loading ? "Consultando…" : "Perguntar"}
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
            <ul aria-label="Fontes citadas" className="flex flex-col gap-1">
              {result.citations.map((citation) => (
                <li key={citation.index} className="text-neutral-700 dark:text-neutral-300">
                  <span className="font-semibold">[{citation.index}]</span> {citation.source}
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </div>
    </section>
  );
}
