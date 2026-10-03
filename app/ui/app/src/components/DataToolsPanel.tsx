import { useState } from "react";

import { downloadBackupArchive, fetchDiagnostics } from "@/lib/agenticClient";

export const BACKUP_ERROR_MESSAGE = "Não foi possível gerar o backup agora. Tente novamente.";
export const DIAGNOSTICS_ERROR_MESSAGE = "Não foi possível gerar o diagnóstico agora. Tente novamente.";

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

// DataToolsPanel exposes local-first data tools (A6/A7): download a backup of
// the workspace (secrets excluded) and export a secret-safe diagnostic snapshot.
export function DataToolsPanel() {
  const [busy, setBusy] = useState<null | "backup" | "diagnostics">(null);
  const [error, setError] = useState<string | null>(null);

  async function handleBackup() {
    setBusy("backup");
    setError(null);
    try {
      const blob = await downloadBackupArchive();
      triggerDownload(blob, `hades-backup-${new Date().toISOString().slice(0, 10)}.tar.gz`);
    } catch {
      setError(BACKUP_ERROR_MESSAGE);
    } finally {
      setBusy(null);
    }
  }

  async function handleDiagnostics() {
    setBusy("diagnostics");
    setError(null);
    try {
      const report = await fetchDiagnostics();
      triggerDownload(new Blob([JSON.stringify(report, null, 2)], { type: "application/json" }), "hades-diagnostico.json");
    } catch {
      setError(DIAGNOSTICS_ERROR_MESSAGE);
    } finally {
      setBusy(null);
    }
  }

  return (
    <section aria-label="Backup e diagnóstico" className="flex flex-col gap-3 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
      <div>
        <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Backup e diagnóstico</h3>
        <p className="mt-1 text-xs text-neutral-600 dark:text-neutral-300">
          Baixe uma cópia local dos seus projetos, memória e configurações (segredos não são incluídos), ou exporte um
          diagnóstico sem dados sensíveis para suporte.
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => void handleBackup()}
          disabled={busy !== null}
          className="min-h-10 rounded-lg bg-neutral-900 px-4 py-2 text-sm font-semibold text-white hover:bg-neutral-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
        >
          {busy === "backup" ? "Gerando…" : "Baixar backup"}
        </button>
        <button
          type="button"
          onClick={() => void handleDiagnostics()}
          disabled={busy !== null}
          className="min-h-10 rounded-lg border border-neutral-300 px-4 py-2 text-sm font-semibold text-neutral-800 hover:bg-neutral-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          {busy === "diagnostics" ? "Gerando…" : "Exportar diagnóstico"}
        </button>
      </div>
      {error ? (
        <p role="alert" className="text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}
    </section>
  );
}
