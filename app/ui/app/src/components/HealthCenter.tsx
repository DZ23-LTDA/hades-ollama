import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { fetchAgentHealth } from "@/lib/agenticClient";

const labels: Record<string, string> = {
  agent: "Motor de missões",
  store: "Persistência",
  queue: "Fila de execução",
  sandbox: "Isolamento",
};

function statusLabel(status: string): string {
  if (status === "ok" || status.endsWith("configured") || status === "local") return "Operacional";
  if (status === "degraded" || status === "best-effort") return "Atenção";
  return "Indisponível";
}

export function HealthCenter() {
  const [details, setDetails] = useState<string | null>(null);
  const health = useQuery({ queryKey: ["agent-health"], queryFn: fetchAgentHealth, retry: 1, refetchInterval: 30_000 });

  if (health.isLoading) {
    return <section aria-labelledby="health-heading" className="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950"><h2 id="health-heading" className="text-sm font-semibold">Saúde do sistema</h2><p className="mt-2 text-xs text-neutral-500" role="status">Verificando os serviços…</p></section>;
  }
  if (health.isError || !health.data) {
    return <section aria-labelledby="health-heading" className="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-900/60 dark:bg-red-950/20"><h2 id="health-heading" className="text-sm font-semibold text-red-950 dark:text-red-100">Saúde do sistema</h2><p className="mt-2 text-xs text-red-800 dark:text-red-200" role="alert">Não foi possível consultar o diagnóstico do backend.</p><button type="button" className="mt-3 min-h-9 rounded-lg border border-red-300 px-3 text-xs font-medium" onClick={() => void health.refetch()}>Tentar novamente</button></section>;
  }

  const entries = Object.entries(health.data.subsystems ?? {});
  const healthStatus = health.data.status ?? "degraded";
  const checkedAt = health.data.checked_at ? new Date(health.data.checked_at).toLocaleTimeString("pt-BR") : "agora";
  return <section aria-labelledby="health-heading" className="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950">
    <div className="flex items-start justify-between gap-3"><div><h2 id="health-heading" className="text-sm font-semibold">Saúde do sistema</h2><p className="mt-1 text-xs text-neutral-500">Diagnóstico recebido do backend em {checkedAt}.</p></div><span className={`rounded-full px-2.5 py-1 text-[11px] font-semibold ${healthStatus === "ok" ? "bg-emerald-100 text-emerald-800" : "bg-amber-100 text-amber-800"}`} role="status">{healthStatus === "ok" ? "Tudo certo" : "Atenção"}</span></div>
    <div className="mt-4 grid gap-2 sm:grid-cols-2">{entries.map(([id, item]) => <div key={id} className="rounded-xl border border-neutral-100 p-3 dark:border-neutral-800"><div className="flex items-center justify-between gap-2"><span className="text-xs font-medium">{labels[id] ?? id}</span><span className="text-[11px] font-semibold text-neutral-600 dark:text-neutral-300">{statusLabel(item.status)}</span></div><div className="mt-2 flex gap-2"><button type="button" className="min-h-8 rounded-md border border-neutral-200 px-2.5 text-[11px] font-medium dark:border-neutral-700" onClick={() => setDetails(id)}>Detalhes</button><button type="button" className="min-h-8 rounded-md border border-neutral-200 px-2.5 text-[11px] font-medium dark:border-neutral-700" onClick={() => setDetails(id)}>Corrigir</button></div>{details === id && <p className="mt-2 text-[11px] leading-5 text-neutral-600 dark:text-neutral-300" role="status">{item.detail}. A ação corretiva depende da configuração exibida e não é executada automaticamente.</p>}</div>)}</div>
  </section>;
}
