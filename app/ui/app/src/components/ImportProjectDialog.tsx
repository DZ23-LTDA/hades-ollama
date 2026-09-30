import { useEffect, useRef, useState } from "react";
import {
  createProject,
  finalizeProjectUpload,
  importGitHubProject,
  importZIPProject,
  startProjectUpload,
  uploadProjectChunk,
} from "@/lib/agenticClient";

type ImportTab = "github" | "zip";

export function ImportProjectDialog({
  open,
  onClose,
  onImported,
}: {
  open: boolean;
  onClose: () => void;
  onImported?: () => void;
}) {
  const [tab, setTab] = useState<ImportTab>("github");
  const [url, setURL] = useState("");
  const [ref, setRef] = useState("");
  const [githubName, setGithubName] = useState("");
  const [zipName, setZipName] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) return;
    returnFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setStatus(null);
    window.setTimeout(() => closeButtonRef.current?.focus(), 0);
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !pending) {
        onClose();
        window.setTimeout(() => returnFocusRef.current?.focus(), 0);
        return;
      }
      if (event.key !== "Tab" || pending || !dialogRef.current) return;
      const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])',
      ));
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open, onClose, pending]);

  if (!open) return null;

  const importGitHub = async () => {
    if (!url.trim() || pending) return;
    setPending(true);
    setStatus(null);
    try {
      const result = await importGitHubProject({ url: url.trim(), ref: ref.trim() || undefined, name: githubName.trim() || undefined });
      setStatus(`Importado e indexado: ${result.indexed_files} arquivos e ${result.indexed_memories} trechos. Worktree ${result.branch}.`);
      setURL("");
      setRef("");
      setGithubName("");
      onImported?.();
    } catch (cause) {
      setStatus(cause instanceof Error ? cause.message : "Não foi possível importar o repositório.");
    } finally {
      setPending(false);
    }
  };

  const importZIP = async () => {
    if (!file || !zipName.trim() || pending) {
      setStatus("Informe o nome do projeto e selecione um arquivo ZIP.");
      return;
    }
    setPending(true);
    setStatus(null);
    try {
      const project = await createProject(zipName.trim());
      const chunkSize = 8 * 1024 * 1024;
      const upload = await startProjectUpload({ project_id: project.id, filename: file.name, total_size: file.size, chunk_size: chunkSize });
      for (let offset = 0; offset < file.size; offset += chunkSize) {
        const chunk = await file.slice(offset, Math.min(offset + chunkSize, file.size)).arrayBuffer();
        await uploadProjectChunk(upload.id, offset, chunk);
        setStatus(`Enviando ZIP: ${Math.min(file.size, offset + chunk.byteLength)} de ${file.size} bytes…`);
      }
      await finalizeProjectUpload(upload.id);
      const result = await importZIPProject({ project_id: project.id, upload_id: upload.id, name: project.name });
      setStatus(`ZIP importado e indexado: ${result.indexed_files} arquivos e ${result.indexed_memories} trechos.`);
      setFile(null);
      setZipName("");
      onImported?.();
    } catch (cause) {
      setStatus(cause instanceof Error ? cause.message : "Não foi possível importar o ZIP.");
    } finally {
      setPending(false);
    }
  };

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="import-project-title" ref={dialogRef} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onMouseDown={(event) => { if (event.target === event.currentTarget && !pending) onClose(); }}>
      <div className="w-full max-w-xl rounded-2xl border border-neutral-200 bg-white p-5 shadow-2xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 id="import-project-title" className="text-lg font-semibold text-neutral-950 dark:text-white">Importar projeto</h2>
            <p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">A importação cria um branch/worktree dedicado e apenas lê arquivos para contexto. Código importado não é executado automaticamente.</p>
          </div>
          <button ref={closeButtonRef} type="button" aria-label="Fechar importação" onClick={onClose} disabled={pending} className="rounded-lg px-2 py-1 text-neutral-500 hover:bg-neutral-100 disabled:opacity-40 dark:hover:bg-neutral-800">Fechar</button>
        </div>
        <div role="tablist" aria-label="Fonte do projeto" className="mt-5 flex gap-2">
          <button type="button" role="tab" aria-selected={tab === "github"} onClick={() => setTab("github")} className={`rounded-xl px-3 py-2 text-xs ${tab === "github" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 dark:border-neutral-700"}`}>URL do GitHub</button>
          <button type="button" role="tab" aria-selected={tab === "zip"} onClick={() => setTab("zip")} className={`rounded-xl px-3 py-2 text-xs ${tab === "zip" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 dark:border-neutral-700"}`}>Anexo ZIP</button>
        </div>
        {tab === "github" ? (
          <div className="mt-5 space-y-3">
            <label className="block text-xs text-neutral-600 dark:text-neutral-300">URL pública ou privada<input value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://github.com/owner/repository" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /></label>
            <label className="block text-xs text-neutral-600 dark:text-neutral-300">Ref opcional<input value={ref} onChange={(event) => setRef(event.target.value)} placeholder="main ou tag" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /></label>
            <label className="block text-xs text-neutral-600 dark:text-neutral-300">Nome do projeto<input value={githubName} onChange={(event) => setGithubName(event.target.value)} placeholder="Usa o nome do repositório se vazio" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /></label>
            <p className="text-[11px] text-neutral-500">Repositórios privados exigem autenticação GitHub configurada no servidor; sem ela o estado é NOT_CONFIGURED.</p>
            <button type="button" onClick={() => void importGitHub()} disabled={!url.trim() || pending} className="rounded-xl bg-violet-700 px-4 py-2.5 text-xs font-medium text-white disabled:opacity-40">{pending ? "Importando…" : "Importar do GitHub"}</button>
          </div>
        ) : (
          <div className="mt-5 space-y-3">
            <label className="block text-xs text-neutral-600 dark:text-neutral-300">Nome do novo projeto<input value={zipName} onChange={(event) => setZipName(event.target.value)} placeholder="Nome do projeto" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /></label>
            <label className="block text-xs text-neutral-600 dark:text-neutral-300">Arquivo ZIP<input type="file" accept=".zip,application/zip" onChange={(event) => setFile(event.target.files?.[0] ?? null)} className="mt-1 block w-full text-sm" /></label>
            <p className="text-[11px] text-neutral-500">O arquivo é enviado em chunks de 8 MiB. Limites e caminhos são verificados no servidor antes da indexação.</p>
            <button type="button" onClick={() => void importZIP()} disabled={!file || !zipName.trim() || pending} className="rounded-xl bg-violet-700 px-4 py-2.5 text-xs font-medium text-white disabled:opacity-40">{pending ? "Enviando e indexando…" : "Importar ZIP"}</button>
          </div>
        )}
        {status && <p role="status" aria-live="polite" className="mt-4 rounded-xl border border-violet-200 bg-violet-50 px-3 py-2 text-xs text-violet-800 dark:border-violet-900 dark:bg-violet-950/20 dark:text-violet-200">{status}</p>}
      </div>
    </div>
  );
}
