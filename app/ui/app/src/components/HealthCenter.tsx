import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { fetchAgentHealth } from "@/lib/agenticClient";
import { useT, type Translator } from "@/i18n";

// As tabelas guardam CHAVES, nao texto: `t()` e' chamado no render, para que a troca de
// idioma repinte. (Mesma licao do commit 9a19452f6: um modulo de dados que avalia t() no
// import congela o idioma do arranque.)
const LABEL_KEYS: Record<string, string> = {
  agent: "health.subsystem.agent",
  store: "health.subsystem.store",
  queue: "health.subsystem.queue",
  sandbox: "health.subsystem.sandbox",
};

function statusLabel(status: string, t: Translator): string {
  if (status === "ok" || status.endsWith("configured") || status === "local") return t("health.status.ok");
  if (status === "degraded" || status === "best-effort") return t("health.status.attention");
  return t("health.status.unavailable");
}

export function HealthCenter() {
  const { t, language } = useT();
  const [details, setDetails] = useState<string | null>(null);
  const health = useQuery({ queryKey: ["agent-health"], queryFn: fetchAgentHealth, retry: 1, refetchInterval: 30_000 });

  if (health.isLoading) {
    return <section aria-labelledby="health-heading" className="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950"><h2 id="health-heading" className="text-sm font-semibold">{t("health.title")}</h2><p className="mt-2 text-xs text-neutral-500" role="status">{t("health.loading")}</p></section>;
  }
  if (health.isError || !health.data) {
    return <section aria-labelledby="health-heading" className="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-900/60 dark:bg-red-950/20"><h2 id="health-heading" className="text-sm font-semibold text-red-950 dark:text-red-100">{t("health.title")}</h2><p className="mt-2 text-xs text-red-800 dark:text-red-200" role="alert">{t("health.error")}</p><button type="button" className="mt-3 min-h-9 rounded-lg border border-red-300 px-3 text-xs font-medium" onClick={() => void health.refetch()}>{t("health.retry")}</button></section>;
  }

  const entries = Object.entries(health.data.subsystems ?? {});
  const healthStatus = health.data.status ?? "degraded";
  // `toLocaleTimeString(language)`: a hora acompanha o idioma activo (en -> 12-hour AM/PM).
  const checkedAt = health.data.checked_at ? new Date(health.data.checked_at).toLocaleTimeString(language) : t("health.now");
  return <section aria-labelledby="health-heading" className="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950">
    <div className="flex items-start justify-between gap-3"><div><h2 id="health-heading" className="text-sm font-semibold">{t("health.title")}</h2><p className="mt-1 text-xs text-neutral-500">{t("health.checkedAt", { at: checkedAt })}</p></div><span className={`rounded-full px-2.5 py-1 text-[11px] font-semibold ${healthStatus === "ok" ? "bg-emerald-100 text-emerald-800" : "bg-amber-100 text-amber-800"}`} role="status">{healthStatus === "ok" ? t("health.status.ok") : t("health.status.attention")}</span></div>
    <div className="mt-4 grid gap-2 sm:grid-cols-2">{entries.map(([id, item]) => <div key={id} className="rounded-xl border border-neutral-100 p-3 dark:border-neutral-800"><div className="flex items-center justify-between gap-2"><span className="text-xs font-medium">{LABEL_KEYS[id] ? t(LABEL_KEYS[id]) : id}</span><span className="text-[11px] font-semibold text-neutral-600 dark:text-neutral-300">{statusLabel(item.status, t)}</span></div><div className="mt-2 flex gap-2"><button type="button" className="min-h-8 rounded-md border border-neutral-200 px-2.5 text-[11px] font-medium dark:border-neutral-700" onClick={() => setDetails((current) => (current === id ? null : id))} aria-expanded={details === id}>{t("health.details")}</button></div>{details === id && <p className="mt-2 text-[11px] leading-5 text-neutral-600 dark:text-neutral-300" role="status">{item.detail}{t("health.detailSuffix")}</p>}</div>)}</div>
  </section>;
}
