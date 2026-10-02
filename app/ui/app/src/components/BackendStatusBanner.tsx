import { useQuery } from "@tanstack/react-query";
import { fetchHealth } from "@/api";

export const BACKEND_OFFLINE_MESSAGE =
  "O backend local está offline. Suas alterações não foram enviadas.";

export function BackendStatusBanner() {
  const health = useQuery({
    queryKey: ["backend-health"],
    queryFn: fetchHealth,
    refetchInterval: 15_000,
    retry: false,
    staleTime: 5_000,
  });

  if (health.isPending || health.data !== false) return null;

  return (
    <div
      role="alert"
      aria-live="assertive"
      className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-amber-300 bg-amber-50 px-4 py-2.5 text-sm text-amber-950 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100"
    >
      <p>{BACKEND_OFFLINE_MESSAGE}</p>
      <button
        type="button"
        onClick={() => void health.refetch()}
        className="min-h-9 rounded-lg border border-amber-700 px-3 py-1.5 text-xs font-semibold hover:bg-amber-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-800 dark:border-amber-300 dark:hover:bg-amber-900/50"
      >
        Tentar novamente
      </button>
    </div>
  );
}
