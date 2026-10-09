import { useState, useMemo, useEffect } from "react";
import { Link } from "@tanstack/react-router";
import { type AgentMission, type AgentArtifact, listMissions, agentFetchBlob } from "@/lib/agenticClient";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import {
  DocumentTextIcon,
  PresentationChartBarIcon,
  TableCellsIcon,
  PhotoIcon,
  VideoCameraIcon,
  SpeakerWaveIcon,
  CodeBracketIcon,
  MagnifyingGlassIcon,
  ArrowDownTrayIcon,
  EyeIcon,
  Squares2X2Icon,
  ListBulletIcon,
  FolderOpenIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";

type MediaCategory = "all" | "text" | "slides" | "sheets" | "images" | "videos" | "audio" | "other";

interface ArtifactWithMission extends AgentArtifact {
  missionId: string;
  missionObjective: string;
  missionCreatedAt: string;
}

export function LibraryPage() {
  const [missions, setMissions] = useState<AgentMission[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<MediaCategory>("all");
  const [viewMode, setViewMode] = useState<"list" | "grid">("list");
  const [downloadError, setDownloadError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    // Authenticated request: no modo exposto a biblioteca exige Bearer, então
    // usamos o cliente autenticado (agentFetch) em vez de fetch cru que daria 401.
    listMissions()
      .then((data) => {
        if (active) {
          setMissions(data.missions || []);
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

  // Download autenticado: o endpoint de artefatos é protegido por Bearer, então
  // um <a href download> daria 401. Buscamos o blob com agentFetchBlob e salvamos.
  const handleDownload = async (missionId: string, artifactId: string, name: string) => {
    setDownloadError(null);
    try {
      const blob = await agentFetchBlob(
        `/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(artifactId)}`,
      );
      const objectURL = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = objectURL;
      link.download = name;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.setTimeout(() => URL.revokeObjectURL(objectURL), 0);
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : "Falha ao baixar o arquivo.");
    }
  };

  // Coletar todos os artefatos de todas as missões
  const allArtifacts: ArtifactWithMission[] = useMemo(() => {
    const list: ArtifactWithMission[] = [];
    for (const m of missions) {
      if (m.artifacts && m.artifacts.length > 0) {
        for (const art of m.artifacts) {
          list.push({
            ...art,
            missionId: m.id,
            missionObjective: m.objective,
            missionCreatedAt: m.created_at,
          });
        }
      }
    }
    return list.sort((a, b) => new Date(b.missionCreatedAt).getTime() - new Date(a.missionCreatedAt).getTime());
  }, [missions]);

  // Função para categorizar arquivo por extensão/tipo
  const getCategory = (filename: string): MediaCategory => {
    const ext = filename.split(".").pop()?.toLowerCase() || "";
    if (["pdf", "md", "txt", "docx", "doc", "rtf"].includes(ext)) return "text";
    if (["pptx", "ppt", "slides"].includes(ext) || filename.includes("slide") || filename.includes("apresentacao")) return "slides";
    if (["csv", "xlsx", "xls", "tsv", "parquet"].includes(ext)) return "sheets";
    if (["png", "jpg", "jpeg", "svg", "webp", "gif", "ico"].includes(ext)) return "images";
    if (["mp4", "webm", "mov", "avi", "mkv"].includes(ext)) return "videos";
    if (["mp3", "wav", "ogg", "flac", "m4a"].includes(ext)) return "audio";
    return "other";
  };

  // Filtragem combinada
  const filteredArtifacts = useMemo(() => {
    return allArtifacts.filter((item) => {
      const cat = getCategory(item.name);
      const matchesCategory = activeCategory === "all" || cat === activeCategory;
      const matchesSearch =
        search === "" ||
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.missionObjective.toLowerCase().includes(search.toLowerCase());
      return matchesCategory && matchesSearch;
    });
  }, [allArtifacts, activeCategory, search]);

  // Renderizar ícone adequado por tipo
  const renderIcon = (filename: string) => {
    const cat = getCategory(filename);
    switch (cat) {
      case "text":
        return <DocumentTextIcon className="h-6 w-6 text-blue-500" />;
      case "slides":
        return <PresentationChartBarIcon className="h-6 w-6 text-amber-500" />;
      case "sheets":
        return <TableCellsIcon className="h-6 w-6 text-emerald-500" />;
      case "images":
        return <PhotoIcon className="h-6 w-6 text-purple-500" />;
      case "videos":
        return <VideoCameraIcon className="h-6 w-6 text-rose-500" />;
      case "audio":
        return <SpeakerWaveIcon className="h-6 w-6 text-indigo-500" />;
      default:
        return <CodeBracketIcon className="h-6 w-6 text-neutral-500" />;
    }
  };

  const categories: Array<{ id: MediaCategory; label: string }> = [
    { id: "all", label: "Todos" },
    { id: "text", label: "Texto e PDF" },
    { id: "slides", label: "Slides" },
    { id: "sheets", label: "Planilhas" },
    { id: "images", label: "Imagens" },
    { id: "videos", label: "Vídeos" },
    { id: "audio", label: "Áudio" },
    { id: "other", label: "Outros" },
  ];

  return (
    <SidebarLayout title="Biblioteca" sidebar={<AppSidebar current="library" />}>
      <div className="mx-auto max-w-6xl space-y-6 px-4 py-8 lg:px-8">
        {/* Cabeçalho da página */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
              Biblioteca
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              Documentos, relatórios, apresentações e criações geradas por suas missões.
            </p>
          </div>
          {/* Barra de busca e controle de visualização */}
          <div className="flex items-center gap-3">
            <div className="relative flex-1 sm:w-64">
              <MagnifyingGlassIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Buscar arquivos..."
                className="w-full rounded-xl border border-neutral-300 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 outline-none transition-all placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white dark:focus:border-white dark:focus:ring-white"
              />
            </div>
            <div className="flex items-center rounded-xl border border-neutral-200 bg-white p-1 dark:border-neutral-800 dark:bg-neutral-800">
              <button
                onClick={() => setViewMode("list")}
                title="Visualização em Lista"
                className={`rounded-lg p-1.5 transition-colors ${
                  viewMode === "list"
                    ? "bg-neutral-100 text-neutral-900 dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-400 hover:text-neutral-900 dark:hover:text-white"
                }`}
              >
                <ListBulletIcon className="h-4 w-4" />
              </button>
              <button
                onClick={() => setViewMode("grid")}
                title="Visualização em Grade"
                className={`rounded-lg p-1.5 transition-colors ${
                  viewMode === "grid"
                    ? "bg-neutral-100 text-neutral-900 dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-400 hover:text-neutral-900 dark:hover:text-white"
                }`}
              >
                <Squares2X2Icon className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Abas de Categorias */}
        <div className="border-b border-neutral-200 dark:border-neutral-800">
          <nav aria-label="Filtros da biblioteca" className="flex space-x-2 overflow-x-auto pb-px">
            {categories.map((cat) => {
              const active = activeCategory === cat.id;
              return (
                <button
                  key={cat.id}
                  onClick={() => setActiveCategory(cat.id)}
                  className={`whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium transition-colors ${
                    active
                      ? "border-neutral-900 text-neutral-900 dark:border-white dark:text-white"
                      : "border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200"
                  }`}
                >
                  {cat.label}
                </button>
              );
            })}
          </nav>
        </div>

        {downloadError && (
          <div role="alert" aria-live="assertive" className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
            {downloadError}
          </div>
        )}

        {/* Conteúdo da Biblioteca */}
        {loading ? (
          <div className="py-16 text-center text-sm text-neutral-500">
            Carregando arquivos da biblioteca...
          </div>
        ) : filteredArtifacts.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-neutral-300 p-12 text-center dark:border-neutral-700">
            <FolderOpenIcon className="mx-auto h-12 w-12 text-neutral-300 dark:text-neutral-600" />
            <h3 className="mt-4 text-base font-semibold text-neutral-900 dark:text-white">
              Nenhum arquivo encontrado
            </h3>
            <p className="mx-auto mt-2 max-w-sm text-sm text-neutral-500 dark:text-neutral-400">
              {search
                ? "Nenhum documento coincide com sua busca."
                : "Seus artefatos e arquivos produzidos pelas missões aparecerão organizados aqui."}
            </p>
            <Link
              to="/agentic"
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <SparklesIcon className="h-4 w-4" />
              Iniciar nova missão
            </Link>
          </div>
        ) : viewMode === "list" ? (
          /* Modo Lista */
          <div className="divide-y divide-neutral-200 overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:divide-neutral-800 dark:border-neutral-800 dark:bg-neutral-900">
            {filteredArtifacts.map((art) => (
              <div
                key={`${art.missionId}-${art.id}`}
                className="flex items-center justify-between gap-4 p-4 transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 dark:bg-neutral-800">
                    {renderIcon(art.name)}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-neutral-900 dark:text-white">
                      {art.name}
                    </p>
                    <p className="truncate text-xs text-neutral-500 dark:text-neutral-400">
                      {art.missionObjective} • {new Date(art.missionCreatedAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span className="hidden text-xs text-neutral-400 sm:inline">
                    {(art.size / 1024).toFixed(1)} KB
                  </span>
                  <Link
                    to="/agentic"
                    search={{ mission_id: art.missionId }}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
                  >
                    <EyeIcon className="h-3.5 w-3.5" />
                    Ver no Canvas
                  </Link>
                  <button
                    type="button"
                    onClick={() => void handleDownload(art.missionId, art.id, art.name)}
                    className="inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
                  >
                    <ArrowDownTrayIcon className="h-3.5 w-3.5" />
                    Baixar
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          /* Modo Grade */
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filteredArtifacts.map((art) => (
              <div
                key={`${art.missionId}-${art.id}`}
                className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-5 transition-shadow hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900"
              >
                <div>
                  <div className="flex items-center justify-between">
                    <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-100 dark:bg-neutral-800">
                      {renderIcon(art.name)}
                    </div>
                    <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                      {(art.size / 1024).toFixed(1)} KB
                    </span>
                  </div>
                  <h4 className="mt-4 truncate text-sm font-semibold text-neutral-900 dark:text-white" title={art.name}>
                    {art.name}
                  </h4>
                  <p className="mt-1 line-clamp-2 text-xs text-neutral-500 dark:text-neutral-400">
                    {art.missionObjective}
                  </p>
                </div>
                <div className="mt-5 flex items-center justify-between border-t border-neutral-100 pt-3 dark:border-neutral-800">
                  <span className="text-[11px] text-neutral-400">
                    {new Date(art.missionCreatedAt).toLocaleDateString()}
                  </span>
                  <div className="flex gap-2">
                    <Link
                      to="/agentic"
                      search={{ mission_id: art.missionId }}
                      className="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
                      title="Ver no Canvas"
                    >
                      <EyeIcon className="h-4 w-4" />
                    </Link>
                    <button
                      type="button"
                      onClick={() => void handleDownload(art.missionId, art.id, art.name)}
                      className="rounded-lg p-1.5 text-neutral-900 hover:bg-neutral-100 dark:text-white dark:hover:bg-neutral-800"
                      title="Baixar artefato"
                    >
                      <ArrowDownTrayIcon className="h-4 w-4" />
                    </button>
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
