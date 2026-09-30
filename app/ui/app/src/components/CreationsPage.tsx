import { useState, useMemo, useEffect } from "react";
import { Link } from "@tanstack/react-router";
import { type AgentArtifact } from "@/lib/agenticClient";
import { API_BASE } from "@/lib/config";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import {
  SparklesIcon,
  GlobeAltIcon,
  RocketLaunchIcon,
  DevicePhoneMobileIcon,
  MagnifyingGlassIcon,
  ArrowTopRightOnSquareIcon,
  ArrowDownTrayIcon,
  EyeIcon,
  CodeBracketSquareIcon,
  PencilSquareIcon,
} from "@heroicons/react/24/outline";

type CreationCategory = "all" | "sites" | "games" | "mobile";

interface CreationItem {
  id: string;
  name: string;
  category: CreationCategory;
  categoryLabel: string;
  missionId: string;
  missionObjective: string;
  createdAt: string;
  previewUrl: string;
  artifact: AgentArtifact;
}

export function CreationsPage() {
  const [creationsList, setCreationsList] = useState<CreationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<CreationCategory>("all");

  useEffect(() => {
    let active = true;
    fetch(`${API_BASE}/api/agent/v1/creations`)
      .then((res) => (res.ok ? res.json() : { creations: [] }))
      .then((data: { creations?: CreationItem[] }) => {
        if (active) {
          setCreationsList(data.creations || []);
          setLoading(false);
        }
      })
      .catch(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const filteredCreations = useMemo(() => {
    return creationsList.filter((item) => {
      const matchesCat = activeCategory === "all" || item.category === activeCategory;
      const matchesSearch =
        search === "" ||
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.missionObjective.toLowerCase().includes(search.toLowerCase());
      return matchesCat && matchesSearch;
    });
  }, [creationsList, activeCategory, search]);

  const categories: Array<{ id: CreationCategory; label: string; icon: React.ReactNode }> = [
    { id: "all", label: "Todos", icon: <SparklesIcon className="h-4 w-4" /> },
    { id: "sites", label: "Sites", icon: <GlobeAltIcon className="h-4 w-4" /> },
    { id: "games", label: "Jogos", icon: <RocketLaunchIcon className="h-4 w-4" /> },
    { id: "mobile", label: "Aplicativos móveis", icon: <DevicePhoneMobileIcon className="h-4 w-4" /> },
  ];

  return (
    <SidebarLayout title="Criações" sidebar={<AppSidebar current="creations" />}>
      <div className="mx-auto max-w-6xl space-y-6 px-4 py-8 lg:px-8">
        {/* Cabeçalho */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
              Criações
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              Sites, jogos e aplicações construídos pelas suas missões no Ollama Full.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="relative w-full sm:w-64">
              <MagnifyingGlassIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Buscar criações..."
              className="w-full rounded-xl border border-neutral-300 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 outline-none transition-all placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white dark:focus:border-white dark:focus:ring-white"
            />
          </div>
          <Link
            to="/studio"
            className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-neutral-200 bg-white px-3.5 py-2 text-sm font-medium text-neutral-800 transition hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-750"
          >
            <PencilSquareIcon className="h-4 w-4" />
            Abrir Studio
          </Link>
          <Link
            to="/agentic"
              search={{ auto_run: true, prompt: "Construir um aplicativo web interativo completo com interface moderna" }}
              className="inline-flex shrink-0 items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <SparklesIcon className="h-4 w-4" />
              Construir agora
            </Link>
          </div>
        </div>

        {/* Abas */}
        <div className="border-b border-neutral-200 dark:border-neutral-800">
          <nav className="flex space-x-4 overflow-x-auto pb-px">
            {categories.map((cat) => {
              const active = activeCategory === cat.id;
              return (
                <button
                  key={cat.id}
                  onClick={() => setActiveCategory(cat.id)}
                  className={`inline-flex items-center gap-2 whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium transition-colors ${
                    active
                      ? "border-neutral-900 text-neutral-900 dark:border-white dark:text-white"
                      : "border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200"
                  }`}
                >
                  {cat.icon}
                  {cat.label}
                </button>
              );
            })}
          </nav>
        </div>

        {/* Lista de Criações */}
        {loading ? (
          <div className="py-16 text-center text-sm text-neutral-500">
            Carregando criações...
          </div>
        ) : filteredCreations.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-neutral-300 p-16 text-center dark:border-neutral-700">
            <CodeBracketSquareIcon className="mx-auto h-12 w-12 text-neutral-300 dark:text-neutral-600" />
            <h3 className="mt-4 text-base font-semibold text-neutral-900 dark:text-white">
              Ainda sem criações
            </h3>
            <p className="mx-auto mt-2 max-w-md text-sm text-neutral-500 dark:text-neutral-400">
              Transforme ideias em sites, protótipos e jogos executáveis no navegador.
            </p>
            <Link
              to="/agentic"
              search={{ auto_run: true, prompt: "Construir uma aplicação interativa com HTML5 e visual moderno" }}
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-5 py-2.5 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <SparklesIcon className="h-4 w-4" />
              Construir agora
            </Link>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredCreations.map((item) => (
              <div
                key={item.id}
                className="group flex flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white transition-all hover:shadow-lg dark:border-neutral-800 dark:bg-neutral-900"
              >
                {/* Preview iframe compacto */}
                <div className="relative aspect-video w-full overflow-hidden bg-neutral-100 dark:bg-neutral-800">
                  <iframe
                    src={item.previewUrl}
                    title={item.name}
                    sandbox="allow-scripts"
                    className="pointer-events-none h-full w-full border-0 opacity-90 transition-opacity group-hover:opacity-100"
                  />
                  <div className="absolute inset-0 bg-transparent" />
                  <span className="absolute left-3 top-3 rounded-full bg-black/70 px-2.5 py-1 text-[11px] font-medium text-white backdrop-blur-sm">
                    {item.categoryLabel}
                  </span>
                </div>

                {/* Conteúdo */}
                <div className="flex flex-1 flex-col justify-between p-5">
                  <div>
                    <h3 className="truncate text-base font-semibold text-neutral-900 dark:text-white">
                      {item.name}
                    </h3>
                    <p className="mt-1 line-clamp-2 text-xs text-neutral-500 dark:text-neutral-400">
                      {item.missionObjective}
                    </p>
                  </div>

                  <div className="mt-5 flex items-center justify-between border-t border-neutral-100 pt-4 dark:border-neutral-800">
                    <span className="text-[11px] text-neutral-400">
                      {new Date(item.createdAt).toLocaleDateString()}
                    </span>
                    <div className="flex items-center gap-2">
                      <Link
                        to="/agentic"
                        search={{ mission_id: item.missionId }}
                        className="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
                        title="Ver no Canvas"
                      >
                        <EyeIcon className="h-3.5 w-3.5" />
                        Canvas
                      </Link>
                      <a
                        href={item.previewUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
                        title="Abrir em Tela Cheia"
                      >
                        <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" />
                        Abrir
                      </a>
                      <a
                        href={item.previewUrl}
                        download={item.name}
                        className="rounded-lg p-1 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
                        title="Baixar código"
                      >
                        <ArrowDownTrayIcon className="h-4 w-4" />
                      </a>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </SidebarLayout>
  );
}
