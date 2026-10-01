import type { ErrorComponentProps } from "@tanstack/react-router";
import { useState } from "react";

// Route-level fallback so a failing page never traps the user: the default
// TanStack component only offered "Show Error" with no way back.
export function RouteErrorFallback({ error, reset }: ErrorComponentProps) {
  const [showDetails, setShowDetails] = useState(false);
  const message = error instanceof Error ? error.message : String(error);

  const goBack = () => {
    reset();
    if (window.history.length > 1) window.history.back();
    else window.location.assign("/");
  };

  return (
    <main
      role="alert"
      className="flex min-h-full flex-1 flex-col items-center justify-center gap-4 bg-white p-8 text-center text-neutral-900 dark:bg-neutral-900 dark:text-neutral-100"
    >
      <h1 className="text-xl font-semibold">Esta página encontrou um erro</h1>
      <p className="max-w-md text-sm text-neutral-500 dark:text-neutral-400">
        O restante do app continua funcionando. Volte, vá para o início ou tente
        carregar a página de novo.
      </p>
      <div className="flex flex-wrap justify-center gap-2">
        <button
          type="button"
          onClick={goBack}
          className="rounded-xl bg-neutral-900 px-4 py-2 text-sm text-white dark:bg-white dark:text-neutral-900"
        >
          Voltar
        </button>
        <button
          type="button"
          onClick={() => {
            reset();
            window.location.assign("/");
          }}
          className="rounded-xl border border-neutral-300 px-4 py-2 text-sm dark:border-neutral-700"
        >
          Ir para o início
        </button>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="rounded-xl border border-neutral-300 px-4 py-2 text-sm dark:border-neutral-700"
        >
          Recarregar
        </button>
      </div>
      <button
        type="button"
        onClick={() => setShowDetails((value) => !value)}
        className="text-xs text-neutral-500 underline"
      >
        {showDetails ? "Ocultar detalhes" : "Mostrar detalhes"}
      </button>
      {showDetails && (
        <pre className="max-w-2xl overflow-auto whitespace-pre-wrap rounded-xl bg-neutral-100 p-3 text-left text-xs dark:bg-neutral-800">
          {message}
        </pre>
      )}
    </main>
  );
}
