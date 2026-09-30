import { useEffect, useState } from "react";
import {
  ArrowDownTrayIcon,
  CodeBracketIcon,
  DocumentTextIcon,
  EyeIcon,
  ShieldCheckIcon,
} from "@heroicons/react/24/outline";

export interface ArtifactItem {
  id: string;
  name: string;
  sha256?: string;
  size?: number;
  media_type?: string;
  content?: string;
}

interface ArtifactsViewerProps {
  missionId: string;
  artifacts: ArtifactItem[];
  selectedArtifactId?: string;
  onSelectArtifact?: (id: string) => void;
}

export function ArtifactsViewer({
  missionId,
  artifacts,
  selectedArtifactId,
  onSelectArtifact,
}: ArtifactsViewerProps) {
  const [activeTab, setActiveTab] = useState<"preview" | "markdown" | "code" | "download">("preview");
  const [activeArtifactIndex, setActiveArtifactIndex] = useState(0);
  const [content, setContent] = useState<string>("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (selectedArtifactId) {
      const idx = artifacts.findIndex((a) => a.id === selectedArtifactId);
      if (idx !== -1) setActiveArtifactIndex(idx);
    }
  }, [selectedArtifactId, artifacts]);

  const currentArtifact = artifacts[activeArtifactIndex];

  useEffect(() => {
    if (!currentArtifact) {
      setContent("");
      return;
    }
    if (currentArtifact.content) {
      setContent(currentArtifact.content);
      return;
    }

    setLoading(true);
    fetch(`/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(currentArtifact.id)}`)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.text();
      })
      .then((text) => setContent(text))
      .catch(() => setContent("// Não foi possível carregar o conteúdo prévio do artefato."))
      .finally(() => setLoading(false));
  }, [missionId, currentArtifact]);

  if (artifacts.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center p-8 text-center text-neutral-400 dark:text-neutral-500">
        <DocumentTextIcon className="h-12 w-12 stroke-[1.2]" />
        <p className="mt-3 text-sm font-medium">Nenhum artefato produzido nesta missão ainda.</p>
        <p className="mt-1 text-xs">Documentos, relatórios, códigos e mídias gerados aparecerão aqui.</p>
      </div>
    );
  }

  const isHtml = currentArtifact?.name.endsWith(".html") || currentArtifact?.media_type === "text/html";
  const isMd = currentArtifact?.name.endsWith(".md") || currentArtifact?.media_type === "text/markdown";

  return (
    <div className="flex h-full flex-col bg-white dark:bg-neutral-900">
      {/* Header com seletor de artefatos */}
      <div className="flex flex-wrap items-center justify-between border-b border-neutral-200 px-4 py-2.5 dark:border-neutral-800">
        <div className="flex items-center gap-2 overflow-x-auto py-1">
          {artifacts.map((art, idx) => (
            <button
              key={art.id}
              onClick={() => {
                setActiveArtifactIndex(idx);
                onSelectArtifact?.(art.id);
              }}
              className={`rounded-lg px-2.5 py-1 text-xs font-medium transition-colors ${
                activeArtifactIndex === idx
                  ? "bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900"
                  : "bg-neutral-100 text-neutral-600 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300"
              }`}
            >
              {art.name}
            </button>
          ))}
        </div>

        {/* Abas de visualização */}
        <div className="flex items-center gap-1 rounded-lg bg-neutral-100 p-1 dark:bg-neutral-800">
          <button
            onClick={() => setActiveTab("preview")}
            className={`flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium ${
              activeTab === "preview" ? "bg-white shadow-sm dark:bg-neutral-700 text-neutral-900 dark:text-white" : "text-neutral-500 hover:text-neutral-800 dark:text-neutral-400"
            }`}
          >
            <EyeIcon className="h-3.5 w-3.5" />
            Visualização
          </button>
          <button
            onClick={() => setActiveTab("code")}
            className={`flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium ${
              activeTab === "code" ? "bg-white shadow-sm dark:bg-neutral-700 text-neutral-900 dark:text-white" : "text-neutral-500 hover:text-neutral-800 dark:text-neutral-400"
            }`}
          >
            <CodeBracketIcon className="h-3.5 w-3.5" />
            Código
          </button>
          <button
            onClick={() => setActiveTab("download")}
            className={`flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium ${
              activeTab === "download" ? "bg-white shadow-sm dark:bg-neutral-700 text-neutral-900 dark:text-white" : "text-neutral-500 hover:text-neutral-800 dark:text-neutral-400"
            }`}
          >
            <ArrowDownTrayIcon className="h-3.5 w-3.5" />
            Exportar
          </button>
        </div>
      </div>

      {/* Conteúdo da aba selecionada */}
      <div className="min-h-0 flex-1 overflow-auto">
        {loading ? (
          <div className="flex h-full items-center justify-center p-8 text-neutral-400">
            <span className="text-sm">Carregando conteúdo do artefato...</span>
          </div>
        ) : activeTab === "preview" ? (
          isHtml ? (
            <iframe
              title={currentArtifact.name}
              srcDoc={content}
              sandbox="allow-scripts"
              className="h-full w-full border-0 bg-white"
            />
          ) : isMd ? (
            <div className="prose prose-neutral dark:prose-invert max-w-none p-6 text-sm">
              <pre className="whitespace-pre-wrap font-sans text-neutral-800 dark:text-neutral-200">{content}</pre>
            </div>
          ) : (
            <div className="p-6">
              <pre className="overflow-x-auto rounded-xl bg-neutral-50 p-4 font-mono text-xs text-neutral-800 dark:bg-neutral-950 dark:text-neutral-200">
                {content}
              </pre>
            </div>
          )
        ) : activeTab === "code" ? (
          <div className="h-full overflow-auto bg-neutral-950 p-4 font-mono text-xs text-neutral-100">
            <pre className="whitespace-pre">{content}</pre>
          </div>
        ) : (
          /* Aba Download & Integridade */
          <div className="p-6 space-y-4">
            <div className="rounded-2xl border border-neutral-200 bg-neutral-50/50 p-5 dark:border-neutral-800 dark:bg-neutral-900/40">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-semibold text-neutral-900 dark:text-white text-base">{currentArtifact.name}</h3>
                  <p className="mt-1 text-xs text-neutral-500">
                    Tamanho: {currentArtifact.size ? `${(currentArtifact.size / 1024).toFixed(1)} KB` : "Automático"}
                  </p>
                </div>
                <a
                  href={`/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(currentArtifact.id)}`}
                  download={currentArtifact.name}
                  className="flex items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white shadow hover:bg-neutral-800 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-100"
                >
                  <ArrowDownTrayIcon className="h-4 w-4" />
                  Baixar Arquivo
                </a>
              </div>

              {currentArtifact.sha256 && (
                <div className="mt-4 flex items-center gap-2 rounded-xl bg-white p-3 border border-neutral-200 dark:border-neutral-800 dark:bg-neutral-950">
                  <ShieldCheckIcon className="h-5 w-5 text-emerald-600 dark:text-emerald-400 shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-[11px] font-medium text-neutral-500">Hash de Proveniência SHA-256</p>
                    <p className="font-mono text-xs text-neutral-800 dark:text-neutral-200 truncate">{currentArtifact.sha256}</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
