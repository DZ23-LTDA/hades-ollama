import { useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowRightIcon,
  BoltIcon,
  BookOpenIcon,
  ChatBubbleLeftRightIcon,
  FolderIcon,
  PaperAirplaneIcon,
} from "@heroicons/react/24/outline";
import Logo from "@/components/Logo";

// Cards de início: destinos REAIS do Ollama Full (sem inventar capacidade).
const ACTIONS: {
  to: string;
  eyebrow: string;
  title: string;
  description: string;
  Icon: typeof BoltIcon;
}[] = [
  {
    to: "/agentic",
    eyebrow: "Compilar e executar",
    title: "Missão agentic",
    description: "Planeje, orquestre, pesquise, aprove e execute com trilha e artifacts.",
    Icon: BoltIcon,
  },
  {
    to: "/c/new",
    eyebrow: "Conversar",
    title: "Nova conversa",
    description: "Chat local com modelos do Ollama e provedores configurados.",
    Icon: ChatBubbleLeftRightIcon,
  },
  {
    to: "/projects",
    eyebrow: "Contexto persistente",
    title: "Projetos",
    description: "Organize memória, fontes, tarefas e workspaces isolados.",
    Icon: FolderIcon,
  },
  {
    to: "/library",
    eyebrow: "Artifacts e arquivos",
    title: "Biblioteca",
    description: "Documentos, sites, dashboards e mídias produzidos pelas missões.",
    Icon: BookOpenIcon,
  },
];

export function HomePage() {
  const [objective, setObjective] = useState("");

  const start = () => {
    const value = objective.trim();
    if (!value) return;
    // O Console agentic lê ?objective da URL ao montar.
    window.location.assign(`/agentic?objective=${encodeURIComponent(value)}`);
  };

  return (
    <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-950">
      <div className="mx-auto w-full max-w-5xl px-6 pb-16 pt-14 lg:px-10">
        <header className="flex flex-col items-center text-center">
          <Logo size={56} containerClassName="mb-5" />
          <h1 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white sm:text-4xl">
            Ollama Full
          </h1>
          <p className="mt-3 max-w-xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            Um runtime local-first: converse, compile, pesquise, automatize e
            acompanhe missões com aprovações e dados no seu ambiente.
          </p>
        </header>

        <section className="mx-auto mt-9 w-full max-w-3xl">
          <label htmlFor="home-objective" className="sr-only">
            Descreva uma tarefa
          </label>
          <div className="flex items-end gap-2 rounded-2xl border border-neutral-200 bg-white p-3 shadow-sm focus-within:border-neutral-400 dark:border-neutral-800 dark:bg-neutral-900">
            <textarea
              id="home-objective"
              value={objective}
              onChange={(event) => setObjective(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
                  event.preventDefault();
                  start();
                }
              }}
              rows={2}
              placeholder="Descreva uma tarefa (ex.: revisar o README e listar pendências)…"
              className="min-h-11 w-full resize-none bg-transparent px-2 py-1.5 text-sm outline-none placeholder:text-neutral-400"
            />
            <button
              type="button"
              onClick={start}
              disabled={!objective.trim()}
              aria-label="Iniciar no Console agentic"
              className="inline-flex h-10 shrink-0 items-center gap-2 rounded-xl bg-neutral-950 px-4 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40 dark:bg-white dark:text-neutral-950"
            >
              <PaperAirplaneIcon className="h-4 w-4" />
              Iniciar
            </button>
          </div>
          <p className="mt-2 px-1 text-[11px] text-neutral-400">
            A tarefa abre no Console agentic começando em modo leitura; escrita e
            ações externas exigem aprovação. Ctrl+Enter para iniciar.
          </p>
        </section>

        <section className="mt-10 grid gap-4 sm:grid-cols-2">
          {ACTIONS.map(({ to, eyebrow, title, description, Icon }) => (
            <Link
              key={to}
              to={to}
              className="group flex items-start gap-4 rounded-2xl border border-neutral-200/80 bg-white p-5 transition hover:border-neutral-300 hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"
            >
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">
                <Icon className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[11px] font-medium uppercase tracking-[0.12em] text-violet-600 dark:text-violet-300">
                  {eyebrow}
                </p>
                <h2 className="mt-1 flex items-center gap-1 font-medium text-neutral-900 dark:text-white">
                  {title}
                  <ArrowRightIcon className="h-4 w-4 -translate-x-1 opacity-0 transition group-hover:translate-x-0 group-hover:opacity-100" />
                </h2>
                <p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">
                  {description}
                </p>
              </div>
            </Link>
          ))}
        </section>
      </div>
    </div>
  );
}
