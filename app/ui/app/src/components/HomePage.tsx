import { useMemo, useRef, useState, type ChangeEvent, type KeyboardEvent } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowRightIcon,
  BoltIcon,
  BookOpenIcon,
  ChatBubbleLeftRightIcon,
  FolderIcon,
  PaperAirplaneIcon,
  PaperClipIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { Model } from "@/gotypes";
import Logo from "@/components/Logo";
import { FileUpload } from "@/components/FileUpload";
import { FirstModelCard } from "@/components/FirstModelCard";
import { ModelPicker } from "@/components/ModelPicker";
import { AGENT_ATTACHMENT_EXTENSIONS, processFiles } from "@/utils/fileValidation";
import { SlashCommandMenu } from "@/components/SlashCommandMenu";
import { ImportProjectDialog } from "@/components/ImportProjectDialog";
import { importMissionAttachments, type AgentProject } from "@/lib/agenticClient";
import {
  filterSlashCommands,
  parseSlashCommand,
  moveSlashCommandIndex,
  slashCommandQuery,
  slashCommandURL,
  type SlashCommand,
} from "@/lib/slashCommands";

const ACTIONS: {
  to: string;
  eyebrow: string;
  title: string;
  description: string;
  Icon: typeof BoltIcon;
}[] = [
  { to: "/agentic", eyebrow: "Compilar >", title: "Sites, aplicativos e jogos", description: "Planeje, execute em split-screen, aprove etapas e gere código com artifacts.", Icon: BoltIcon },
  { to: "/library", eyebrow: "Criar >", title: "Slides, imagens e vídeos", description: "Documentos executivos, apresentações e mídias geradas pelas missões.", Icon: BookOpenIcon },
  { to: "/projects", eyebrow: "Workspace local", title: "Começar a partir de arquivo local", description: "Abrir e orquestrar arquivos e projetos locais com contexto persistente.", Icon: FolderIcon },
  { to: "/c/new", eyebrow: "Chat e Modelos", title: "Conversa com modelo", description: "Chat local-first com modelos do Ollama e provedores configurados.", Icon: ChatBubbleLeftRightIcon },
];

function commandObjective(command: SlashCommand, objective: string): string {
  if (command.id === "test") return `Rodar o runner de testes do projeto atual e registrar o resultado. Contexto: ${objective}`;
  if (command.id === "review") return `Revisar o código ou artefato atual, apontar riscos e recomendações acionáveis. Contexto: ${objective}`;
  return objective;
}

export function HomePage() {
  const [objective, setObjective] = useState("");
  const [selectionMode, setSelectionMode] = useState<"auto" | "manual">("auto");
  const [manualModel, setManualModel] = useState<Model | null>(null);
  const [attachments, setAttachments] = useState<Array<{ filename: string; data: Uint8Array; type?: string }>>([]);
  const [attachmentErrors, setAttachmentErrors] = useState<string[]>([]);
  const [startError, setStartError] = useState("");
  const [starting, setStarting] = useState(false);
  const [allowWorkspaceWrite, setAllowWorkspaceWrite] = useState(false);
  const [allowRemoteAttachmentSend, setAllowRemoteAttachmentSend] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<AgentProject | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [activeCommand, setActiveCommand] = useState(0);
  const query = slashCommandQuery(objective);
  const commands = useMemo(() => filterSlashCommands(query ?? ""), [query]);

  const selectCommand = (command: SlashCommand) => {
    setObjective(`${command.label} `);
    setActiveCommand(0);
  };

  const remoteModelWithAttachments = attachments.length > 0 && selectionMode === "manual" && manualModel?.kind === "remote";

  const start = async () => {
    if (starting || !objective.trim()) return;
    if (remoteModelWithAttachments && !allowRemoteAttachmentSend) {
      setStartError("Confirme que entende que o provider remoto selecionado pode receber o pedido e o contexto dos anexos.");
      return;
    }
    const parsed = parseSlashCommand(objective);
    const url = parsed ? slashCommandURL(objective) : null;
    const value = objective.trim();
    const targetObjective = parsed && url ? commandObjective(parsed.command, parsed.objective) : value;
    setStarting(true);
    setStartError("");
    try {
      let importedProjectID = selectedProject?.id ?? "";
      if (attachments.length > 0) {
        const imported = await importMissionAttachments(attachments, `Anexos: ${targetObjective.slice(0, 96)}`);
        importedProjectID = imported.project?.id ?? "";
        if (!importedProjectID) throw new Error("O servidor não confirmou a criação do projeto dos anexos.");
      }

      const params = parsed && url ? new URLSearchParams(url.split("?")[1]) : new URLSearchParams();
      params.set("objective", targetObjective);
      params.set("mode", selectionMode);
      params.set("model", selectionMode === "manual" && manualModel ? manualModel.model : "auto/coding");
      params.set("provider", selectionMode === "manual" && manualModel ? manualModel.provider || "ollama-local" : "ollama-local");
      if (importedProjectID) params.set("project_id", importedProjectID);
      if (allowWorkspaceWrite) {
        params.set("allow_write", "true");
        params.delete("autorun");
      } else if (!(parsed && url)) {
        params.set("autorun", "true");
      }
      window.location.assign(`/agentic?${params.toString()}`);
    } catch (cause) {
      setStartError(cause instanceof Error ? cause.message : "Falha ao enviar ou indexar os anexos.");
    } finally {
      setStarting(false);
    }
  };

  const addFiles = (files: Array<{ filename: string; data: Uint8Array; type?: string }>) => {
    const accepted: typeof files = [];
    const errors: string[] = [];
    let totalBytes = attachments.reduce((total, file) => total + file.data.byteLength, 0);
    for (const file of files) {
      if (attachments.length + accepted.length >= 10) {
        errors.push(`${file.filename}: limite de 10 arquivos por tarefa`);
      } else if (totalBytes + file.data.byteLength > 32 * 1024 * 1024) {
        errors.push(`${file.filename}: limite total de 32 MB por tarefa`);
      } else {
        accepted.push(file);
        totalBytes += file.data.byteLength;
      }
    }
    if (accepted.length > 0) {
      setSelectedProject(null);
      setAttachments((current) => [...current, ...accepted]);
    }
    if (errors.length > 0) setAttachmentErrors((current) => [...current, ...errors]);
  };

  const handleFilesAdded = (files: Array<{ filename: string; data: Uint8Array; type?: string }>, errors: Array<{ filename: string; error: string }> = []) => {
    addFiles(files);
    if (errors.length > 0) setAttachmentErrors((current) => [...current, ...errors.map((item) => `${item.filename}: ${item.error}`)]);
  };

  const handleFileInput = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    if (files.length === 0) return;
    const result = await processFiles(files, { maxFileSize: 10, allowedExtensions: AGENT_ATTACHMENT_EXTENSIONS, hasVisionCapability: false });
    handleFilesAdded(result.validFiles, result.errors);
    event.target.value = "";
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (query !== null && commands.length > 0) {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setActiveCommand((index) => moveSlashCommandIndex(index, "next", commands.length));
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setActiveCommand((index) => moveSlashCommandIndex(index, "previous", commands.length));
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setObjective("");
        return;
      }
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        if (!objective.trimEnd().includes(" ")) {
          selectCommand(commands[activeCommand] ?? commands[0]);
        } else {
          start();
        }
        return;
      }
    }
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      start();
      return;
    }
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      start();
    }
  };

  return (
    <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-950">
      <div className="mx-auto w-full max-w-5xl px-6 pb-16 pt-14 lg:px-10">
        <header className="flex flex-col items-center text-center"><Logo size={56} containerClassName="mb-5" /><h1 className="page-title sm:text-4xl">Hades</h1><p className="page-description max-w-xl">Um runtime local-first: converse, compile, pesquise, automatize e acompanhe missões com aprovações e dados no seu ambiente.</p></header>
        <div className="mt-9"><FirstModelCard /></div>
        <section className="mx-auto mt-9 w-full max-w-3xl">
          <label htmlFor="home-objective" className="sr-only">Descreva uma tarefa</label>
          <div className="relative">
            <div className="absolute inset-x-0 bottom-full mb-2"><SlashCommandMenu query={query} activeIndex={activeCommand} onActiveIndexChange={setActiveCommand} onSelect={selectCommand} /></div>
            <FileUpload onFilesAdded={handleFilesAdded} allowedExtensions={AGENT_ATTACHMENT_EXTENSIONS} maxFileSize={10}>
            <div className="rounded-2xl border border-neutral-200 bg-white p-3 shadow-sm focus-within:border-neutral-400 dark:border-neutral-800 dark:bg-neutral-900">
              {attachments.length > 0 && <div className="mb-2 flex flex-wrap gap-1.5" aria-label="Arquivos anexados">{attachments.map((file, index) => <span key={`${file.filename}-${index}`} className="inline-flex items-center gap-1 rounded-lg bg-neutral-100 px-2 py-1 text-[11px] text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">{file.filename}<button type="button" aria-label={`Remover ${file.filename}`} onClick={() => setAttachments((current) => current.filter((_, fileIndex) => fileIndex !== index))}><XMarkIcon className="h-3 w-3" /></button></span>)}</div>}
              <div className="flex items-end gap-2">
              <textarea id="home-objective" aria-label="Objetivo da nova tarefa" value={objective} onChange={(event) => { setObjective(event.target.value); setActiveCommand(0); }} onKeyDown={handleKeyDown} rows={2} placeholder="Atribua uma tarefa ou digite / para mais opções..." className="min-h-11 w-full resize-none bg-transparent px-2 py-1.5 text-sm outline-none placeholder:text-neutral-400" />
              <button type="button" aria-label="Anexar arquivo" title="Anexar arquivo" onClick={() => fileInputRef.current?.click()} className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-neutral-200 text-neutral-600 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"><PaperClipIcon className="h-4 w-4" /></button>
              <button type="button" onClick={() => void start()} disabled={!objective.trim() || starting || (remoteModelWithAttachments && !allowRemoteAttachmentSend)} aria-busy={starting || undefined} aria-label="Iniciar no Console agentic" className="inline-flex h-10 shrink-0 items-center gap-2 rounded-xl bg-neutral-950 px-4 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40 dark:bg-white dark:text-neutral-950"><PaperAirplaneIcon className="h-4 w-4" />{starting ? "Enviando…" : allowWorkspaceWrite ? "Continuar para revisar" : "Iniciar"}</button>
              </div>
              <input ref={fileInputRef} type="file" multiple accept={AGENT_ATTACHMENT_EXTENSIONS.map((extension) => `.${extension}`).join(",")} className="hidden" onChange={(event) => void handleFileInput(event)} aria-label="Escolher arquivos para anexar" />
              {attachmentErrors.length > 0 && <p role="alert" className="mt-2 text-xs text-rose-600 dark:text-rose-400">{attachmentErrors.join(" • ")}</p>}
              {startError && <p role="alert" className="mt-2 text-xs text-rose-600 dark:text-rose-400">{startError}</p>}
              <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-neutral-100 pt-3 dark:border-neutral-800"><span className="text-[11px] font-medium text-neutral-500">IA da tarefa</span><button type="button" aria-pressed={selectionMode === "auto"} onClick={() => setSelectionMode("auto")} className={`rounded-lg px-2.5 py-1.5 text-[11px] font-medium ${selectionMode === "auto" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"}`}>Automático · grátis-primeiro</button><button type="button" aria-pressed={selectionMode === "manual"} onClick={() => setSelectionMode("manual")} className={`rounded-lg px-2.5 py-1.5 text-[11px] font-medium ${selectionMode === "manual" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"}`}>Manual</button>{selectionMode === "manual" && <ModelPicker chatId="new" selectedModelOverride={manualModel} onModelSelectModel={setManualModel} selectableOnly buttonLabel={manualModel ? `${manualModel.model} · ${manualModel.cost_tag || (manualModel.kind === "remote" ? "pago" : manualModel.kind === "cli_subscription" ? "0-assinatura" : "0-local")}` : "Escolher modelo PASS"} />}</div>
              <div className="mt-3 flex flex-col gap-2 border-t border-neutral-100 pt-3 text-xs text-neutral-600 dark:border-neutral-800 dark:text-neutral-300">
                <label className="flex items-start gap-2"><input type="checkbox" checked={allowWorkspaceWrite} onChange={(event) => setAllowWorkspaceWrite(event.target.checked)} className="mt-0.5 rounded" /><span>Permitir alterações nos arquivos do projeto. Se marcar, a tarefa não executará automaticamente: você revisará e iniciará a missão no console.</span></label>
                {remoteModelWithAttachments && <label className="flex items-start gap-2"><input type="checkbox" checked={allowRemoteAttachmentSend} onChange={(event) => setAllowRemoteAttachmentSend(event.target.checked)} className="mt-0.5 rounded" /><span>Entendo que o provider remoto selecionado pode receber meu pedido e trechos dos anexos para gerar a resposta.</span></label>}
                <p className="text-[11px] text-neutral-400">Anexos são enviados ao Hades para indexação local (até 10 arquivos, 10 MB cada e 32 MB no total). Imagens ainda não são indexadas. O modo automático recebe somente leitura; escrita exige consentimento e revisão.</p>
              </div>
            </div>
            </FileUpload>
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-2 px-1" aria-label="Formas de começar">
            <span className="text-[11px] font-medium text-neutral-400">Começar por:</span>
            <button type="button" onClick={() => fileInputRef.current?.click()} className="rounded-lg border border-neutral-200/80 bg-white px-2.5 py-1.5 text-[11px] font-medium text-neutral-600 hover:border-neutral-300 hover:bg-neutral-100 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800">Anexar arquivo</button>
            <button type="button" id="import-project-open" onClick={() => setImportOpen(true)} className="rounded-lg border border-violet-200 bg-violet-50 px-2.5 py-1.5 text-[11px] font-medium text-violet-700 hover:border-violet-300 hover:bg-violet-100 dark:border-violet-900/60 dark:bg-violet-950/20 dark:text-violet-300 dark:hover:bg-violet-950/40">Importar projeto</button>
          </div>
          {selectedProject && <p role="status" className="mt-2 px-1 text-[11px] text-violet-700 dark:text-violet-300">Projeto selecionado: {selectedProject.name} <button type="button" aria-label="Remover projeto selecionado" onClick={() => setSelectedProject(null)} className="underline">Remover</button></p>}
          <p className="mt-2 px-1 text-[11px] text-neutral-400">Comandos reais: /goal delega, /plan apenas planeja, /test executa testes e /review revisa. Ctrl+Enter para iniciar.</p>
          <div className="mt-3 flex flex-wrap items-center gap-2 px-1"><span className="text-[11px] font-medium text-neutral-400">Ações rápidas:</span>{[{ label: "Criar slides", prompt: "Criar apresentação profissional de slides sobre inovação em IA" }, { label: "Criar site", prompt: "Criar uma landing page moderna responsiva com Tailwind e React" }, { label: "Pesquisa profunda", prompt: "Realizar pesquisa aprofundada de mercado com síntese e fontes citadas" }, { label: "Analisar código", prompt: "Inspecionar o repositório, auditar segurança e listar recomendações" }].map(({ label, prompt }) => <button key={label} type="button" onClick={() => setObjective(prompt)} className="rounded-lg border border-neutral-200/80 bg-white px-2.5 py-1 text-[11px] font-medium text-neutral-600 transition hover:border-neutral-300 hover:bg-neutral-100 hover:text-neutral-900 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800 dark:hover:text-white">{label}</button>)}</div>
        </section>
        <ImportProjectDialog open={importOpen} onClose={() => setImportOpen(false)} onImported={(project) => { setSelectedProject(project); setImportOpen(false); }} />
        <section className="mt-10 grid gap-4 sm:grid-cols-2">{ACTIONS.map(({ to, eyebrow, title, description, Icon }) => <Link key={to} to={to} className="group flex items-start gap-4 rounded-2xl border border-neutral-200/80 bg-white p-5 transition hover:border-neutral-300 hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"><div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"><Icon className="h-5 w-5" /></div><div className="min-w-0 flex-1"><p className="text-[11px] font-medium uppercase tracking-[0.12em] text-violet-600 dark:text-violet-300">{eyebrow}</p><h2 className="mt-1 flex items-center gap-1 font-medium text-neutral-900 dark:text-white">{title}<ArrowRightIcon className="h-4 w-4 -translate-x-1 opacity-0 transition group-hover:translate-x-0 group-hover:opacity-100" /></h2><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{description}</p></div></Link>)}</section>
      </div>
    </div>
  );
}
