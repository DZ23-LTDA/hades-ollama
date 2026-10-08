import { useCallback, useEffect, useState } from "react";
import {
  CheckCircleIcon,
  XCircleIcon,
  ArrowPathIcon,
  GlobeAltIcon,
} from "@heroicons/react/24/outline";
import {
  getBrowserEnvironment,
  setupBrowserEnvironment,
  type BrowserEnvironmentStatus,
  type BrowserSetupResult,
} from "@/lib/agenticClient";
import { humanizeApiError } from "@/lib/userFacingError";

function StatusRow({ ok, label }: { ok: boolean; label: string }) {
  return (
    <div className="flex items-center gap-2 text-sm">
      {ok ? (
        <CheckCircleIcon className="h-4 w-4 text-emerald-500" />
      ) : (
        <XCircleIcon className="h-4 w-4 text-neutral-400" />
      )}
      <span className={ok ? "text-neutral-800 dark:text-neutral-100" : "text-neutral-500"}>{label}</span>
    </div>
  );
}

export function BrowserOperatorPanel() {
  const [status, setStatus] = useState<BrowserEnvironmentStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [settingUp, setSettingUp] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [steps, setSteps] = useState<BrowserSetupResult["steps"]>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setStatus(await getBrowserEnvironment());
    } catch (e) {
      setError(humanizeApiError(e, "Não foi possível verificar o operador de navegador.").message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSetup = useCallback(async () => {
    setSettingUp(true);
    setError(null);
    setSteps(null);
    try {
      const result = await setupBrowserEnvironment();
      setSteps(result.steps ?? []);
      setStatus(result.status);
    } catch (e) {
      setError(humanizeApiError(e, "Não foi possível preparar o navegador.").message);
    } finally {
      setSettingUp(false);
    }
  }, []);

  return (
    <section
      className="rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"
      data-testid="browser-operator-panel"
    >
      <div className="mb-3 flex items-center gap-2">
        <GlobeAltIcon className="h-5 w-5 text-neutral-500" />
        <h3 className="text-sm font-bold text-neutral-900 dark:text-white">Operador de navegador</h3>
      </div>
      <p className="mb-4 text-xs text-neutral-500 dark:text-neutral-400">
        Permite que o agente navegue, clique e preencha páginas de verdade (via Playwright/Chromium),
        sob aprovação e com destinos de rede validados. Verifique se o ambiente está pronto.
      </p>

      {loading ? (
        <p className="text-sm text-neutral-500">Verificando ambiente…</p>
      ) : status ? (
        <>
          {status.ready ? (
            <div className="flex items-center gap-2 rounded-xl bg-emerald-50 p-3 text-sm font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300" data-testid="browser-ready">
              <CheckCircleIcon className="h-5 w-5 text-emerald-500" />
              Operador de navegador pronto para usar.
            </div>
          ) : (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <StatusRow ok={status.python_ok} label={`Python${status.python_path ? ` (${status.python_path})` : ""}`} />
                <StatusRow
                  ok={status.playwright_ok}
                  label={`Playwright${status.playwright_version ? ` ${status.playwright_version}` : ""}`}
                />
                <StatusRow ok={status.chromium_ok} label="Navegador Chromium do Playwright" />
              </div>

              {status.guidance && status.guidance.length > 0 && (
                <ul className="list-disc space-y-1 rounded-xl bg-amber-50 p-3 pl-7 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-300" data-testid="browser-guidance">
                  {status.guidance.map((g, i) => (
                    <li key={i}>{g}</li>
                  ))}
                </ul>
              )}

              {status.can_auto_setup && (
                <button
                  type="button"
                  onClick={handleSetup}
                  disabled={settingUp}
                  className="inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-neutral-900"
                >
                  {settingUp && <ArrowPathIcon className="h-4 w-4 animate-spin" />}
                  {settingUp ? "Preparando… (pode levar alguns minutos)" : "Preparar navegador automaticamente"}
                </button>
              )}
            </div>
          )}

          {steps && steps.length > 0 && (
            <ul className="mt-3 space-y-1 text-xs" data-testid="browser-setup-steps">
              {steps.map((s, i) => (
                <li key={i} className="flex items-center gap-2">
                  {s.ok ? (
                    <CheckCircleIcon className="h-3.5 w-3.5 text-emerald-500" />
                  ) : (
                    <XCircleIcon className="h-3.5 w-3.5 text-red-500" />
                  )}
                  <span className={s.ok ? "text-neutral-600 dark:text-neutral-300" : "text-red-600 dark:text-red-400"}>
                    {s.description}
                  </span>
                </li>
              ))}
            </ul>
          )}

          <button
            type="button"
            onClick={() => void load()}
            className="mt-3 text-xs font-medium text-neutral-500 hover:text-neutral-900 dark:hover:text-white"
          >
            Verificar novamente
          </button>
        </>
      ) : null}

      {error && (
        <p role="alert" className="mt-3 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </section>
  );
}
