import { useState, useMemo, useEffect } from "react";
import { Link } from "@tanstack/react-router";
import { type AgentArtifact, agentFetch, agentFetchBlob } from "@/lib/agenticClient";
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

// CreationPreview carrega a prévia através do cliente autenticado. A URL do
// artefato é protegida por Bearer, então um <iframe src=previewUrl> cru daria
// 401; buscamos o blob e renderizamos via blob URL dentro de um sandbox restrito.
function CreationPreview({ previewUrl, name }: { previewUrl: string; name: string }) {
  const [src, setSrc] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let objectURL: string | null = null;
    setFailed(false);
    setSrc(null);
    void agentFetchBlob(previewUrl)
      .then((blob) => {
        if (cancelled) return;
        objectURL = URL.createObjectURL(blob);
        setSrc(objectURL);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [previewUrl]);

  if (failed) {
    return (
      <div className="flex h-full w-full items-center justify-center px-3 text-center text-[11px] text-neutral-500 dark:text-neutral-400">
        Prévia autenticada indisponível
      </div>
    );
  }

  return (
    <iframe
      src={src ?? undefined}
      title={name}
      sandbox="allow-scripts"
      className="pointer-events-none h-full w-full border-0 opacity-90 transition-opacity group-hover:opacity-100"
    />
  );
}

export function CreationsPage() {
  const [creationsList, setCreationsList] = useState<CreationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState("NOT_EXECUTED");
  const [statusReason, setStatusReason] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<CreationCategory>("all");
  const [openView, setOpenView] = useState<{ url: string; name: string } | null>(null);

  useEffect(() => {
    if (!openView) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenView(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [openView]);

  useEffect(() => {
    let active = true;
    // Requisição autenticada: no modo exposto /creations exige Bearer. Em 401 o
    // agentFetch lança, e tratamos como falha de carregamento (não como
    // "publicação indisponível", que é uma condição diferente).
    agentFetch<{ creations?: CreationItem[]; status?: string; reason?: string }>("/api/agent/v1/creations")
      .then((data) => {
        if (!active) return;
        setCreationsList(Array.isArray(data.creations) ? data.creations : []);
        setStatus(data.status || "NOT_EXECUTED");
        setStatusReason(data.reason || "");
        setLoading(false);
      })
      .catch((err) => {
        if (!active) return;
        setLoadError(err instanceof Error ? err.message : "Não foi possível carregar as criações.");
        setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  // "Abrir": o artefato é protegido por Bearer, então buscamos o blob autenticado
  // e o exibimos num modal in-app (<iframe src=blob>). Evita window.open(blob:…),
  // que o webview do app desktop não consegue abrir (cai em "Not Found").
  const handleOpen = async (previewUrl: string, name: string) => {
    setActionError(null);
    try {
      const blob = await agentFetchBlob(previewUrl);
      const objectURL = URL.createObjectURL(blob);
      setOpenView((prev) => {
        if (prev) URL.revokeObjectURL(prev.url);
        return { url: objectURL, name };
      });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Falha ao abrir a criação.");
    }
  };

  const closeView = () => {
    setOpenView((prev) => {
      if (prev) URL.revokeObjectURL(prev.url);
      return null;
    });
  };

  // "Baixar código": download autenticado via blob em vez de <a href download>.
  const handleDownloadCode = async (previewUrl: string, name: string) => {
    setActionError(null);
    try {
      const blob = await agentFetchBlob(previewUrl);
      const objectURL = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = objectURL;
      link.download = name || "criacao";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.setTimeout(() => URL.revokeObjectURL(objectURL), 0);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Falha ao baixar o código.");
    }
  };

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
              Artefatos publicáveis produzidos pelas missões. O runtime informa honestamente quando a publicação ainda não está disponível.
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
              className="inline-flex shrink-0 items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <SparklesIcon className="h-4 w-4" />
              Abrir console de missões
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

        {actionError && (
          <div role="alert" aria-live="assertive" className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
            {actionError}
          </div>
        )}

        {/* Lista de Criações */}
        {loading ? (
          <div className="py-16 text-center text-sm text-neutral-500">
            Carregando criações...
          </div>
        ) : filteredCreations.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-neutral-300 p-16 text-center dark:border-neutral-700">
            <CodeBracketSquareIcon className="mx-auto h-12 w-12 text-neutral-300 dark:text-neutral-600" />
            <h3 className="mt-4 text-base font-semibold text-neutral-900 dark:text-white">
              {loadError ? "Não foi possível carregar as criações" : "Nenhuma criação publicável"}
            </h3>
            <p className="mx-auto mt-2 max-w-md text-sm text-neutral-500 dark:text-neutral-400">
              {loadError
                ? loadError
                : creationsList.length > 0
                  ? "Nenhum artefato corresponde aos filtros atuais."
                  : status === "NOT_EXECUTED" && statusReason
                    ? statusReason
                    : "Nenhuma criação ainda."}
            </p>
            <Link
              to="/agentic"
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-5 py-2.5 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <SparklesIcon className="h-4 w-4" />
              Abrir console de missões
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
                  <CreationPreview previewUrl={item.previewUrl} name={item.name} />
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
                      <button
                        type="button"
                        onClick={() => void handleOpen(item.previewUrl, item.name)}
                        className="inline-flex items-center gap-1 rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
                        title="Abrir em Tela Cheia"
                      >
                        <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" />
                        Abrir
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleDownloadCode(item.previewUrl, item.name)}
                        className="rounded-lg p-1 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
                        title="Baixar código"
                      >
                        <ArrowDownTrayIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Abrir em tela cheia: modal in-app com iframe autenticado (funciona no app e no navegador) */}
        {openView && (
          <div
            role="dialog"
            aria-modal="true"
            aria-label={`Criação: ${openView.name}`}
            className="fixed inset-0 z-50 flex flex-col bg-black/70 p-4"
            onClick={closeView}
          >
            <div
              className="mx-auto flex h-full w-full max-w-5xl flex-col overflow-hidden rounded-2xl bg-white shadow-xl dark:bg-neutral-900"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-2 dark:border-neutral-800">
                <span className="truncate text-sm font-semibold text-neutral-900 dark:text-white">{openView.name}</span>
                <button
                  type="button"
                  onClick={closeView}
                  aria-label="Fechar"
                  className="rounded-lg px-3 py-1 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  Fechar ✕
                </button>
              </div>
              <iframe
                title={openView.name}
                src={openView.url}
                sandbox="allow-scripts"
                className="h-full w-full flex-1 border-0 bg-white"
              />
            </div>
          </div>
        )}
      </div>
    </SidebarLayout>
  );
}
