import { useCallback, useEffect, useState, useTransition } from "react";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { missionStateLabel } from "@/lib/labels";
import { generateStudioHTML } from "@/lib/studioHtml";
import { reorderById } from "@/lib/studioReorder";
import {
  type BuilderProject,
  type VisualComponent,
  listBuilders,
  getBuilder,
  createBuilder,
  updateBuilderVisual,
  undoBuilder,
  redoBuilder,
  previewBuilder,
  exportBuilder,
  deployBuilder,
  agentFetchBlob,
} from "@/lib/agenticClient";
import {
  SparklesIcon,
  ArrowUturnLeftIcon,
  ArrowUturnRightIcon,
  ArrowDownTrayIcon,
  RocketLaunchIcon,
  EyeIcon,
  PencilSquareIcon,
  TrashIcon,
  PlusIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  GlobeAltIcon,
  PresentationChartBarIcon,
  PuzzlePieceIcon,
  DevicePhoneMobileIcon,
  ChartBarIcon,
} from "@heroicons/react/24/outline";

const COMPONENT_TEMPLATES: Array<{
  type: string;
  label: string;
  icon: string;
  defaultProps: Record<string, string>;
  defaultStyle: Record<string, string>;
  width: number;
  height: number;
}> = [
  {
    type: "heading",
    label: "Título",
    icon: "H",
    defaultProps: { text: "Novo Título de Impacto", level: "h1" },
    defaultStyle: { color: "#111827", fontSize: "2rem", fontWeight: "bold" },
    width: 600,
    height: 60,
  },
  {
    type: "paragraph",
    label: "Parágrafo",
    icon: "P",
    defaultProps: { text: "Descreva sua ideia, solução ou produto com detalhes claros e objetivos." },
    defaultStyle: { color: "#4B5563", fontSize: "1rem" },
    width: 600,
    height: 50,
  },
  {
    type: "button",
    label: "Botão de Ação",
    icon: "B",
    defaultProps: { label: "Clique Aqui", action: "submit" },
    defaultStyle: { backgroundColor: "#111827", color: "#FFFFFF", borderRadius: "8px" },
    width: 160,
    height: 44,
  },
  {
    type: "card",
    label: "Card / Recurso",
    icon: "C",
    defaultProps: { title: "Destaque do Produto", body: "Explicação dos benefícios chave da solução construída." },
    defaultStyle: { backgroundColor: "#F9FAFB", borderColor: "#E5E7EB", borderRadius: "12px" },
    width: 320,
    height: 160,
  },
  {
    type: "metric",
    label: "Métrica / KPI",
    icon: "M",
    defaultProps: { label: "Taxa de Sucesso", value: "99.8%", change: "+4.2%" },
    defaultStyle: { backgroundColor: "#F3F4F6", borderRadius: "8px" },
    width: 200,
    height: 90,
  },
  {
    type: "navbar",
    label: "Barra de Navegação",
    icon: "N",
    defaultProps: { brand: "Ollama Studio", links: "Início, Recursos, Preços, Contato" },
    defaultStyle: { backgroundColor: "#FFFFFF", borderBottom: "1px solid #E5E7EB" },
    width: 800,
    height: 60,
  },
  {
    type: "hero",
    label: "Hero / Destaque",
    icon: "★",
    defaultProps: { title: "Sua ideia, pronta em minutos", subtitle: "Monte, publique e evolua — tudo local.", cta: "Começar agora" },
    defaultStyle: {},
    width: 800,
    height: 220,
  },
  {
    type: "image",
    label: "Imagem",
    icon: "I",
    defaultProps: { src: "", alt: "Descrição da imagem" },
    defaultStyle: {},
    width: 400,
    height: 240,
  },
  {
    type: "input",
    label: "Campo de Formulário",
    icon: "F",
    defaultProps: { label: "Seu e-mail", placeholder: "voce@exemplo.com" },
    defaultStyle: {},
    width: 320,
    height: 70,
  },
  {
    type: "link",
    label: "Link",
    icon: "L",
    defaultProps: { text: "Saiba mais", href: "https://" },
    defaultStyle: {},
    width: 160,
    height: 32,
  },
  {
    type: "list",
    label: "Lista",
    icon: "≡",
    defaultProps: { items: "Primeiro item, Segundo item, Terceiro item" },
    defaultStyle: {},
    width: 400,
    height: 120,
  },
  {
    type: "divider",
    label: "Divisor",
    icon: "—",
    defaultProps: {},
    defaultStyle: {},
    width: 600,
    height: 20,
  },
];

// Provedores de publicação: cada um exige a credencial correta no ambiente do
// servidor (inclusive SSH). O slug é enviado ao Deploy Adapter real do backend.
const DEPLOY_PROVIDERS: Array<{ label: string; slug: string; credential: string }> = [
  { label: "Vercel", slug: "vercel", credential: "VERCEL_TOKEN" },
  { label: "Cloudflare Pages", slug: "cloudflare", credential: "CLOUDFLARE_API_TOKEN e CLOUDFLARE_ACCOUNT_ID" },
  { label: "Netlify", slug: "netlify", credential: "NETLIFY_AUTH_TOKEN" },
  { label: "SSH / Servidor Próprio", slug: "ssh", credential: "chave SSH e host (SSH_PRIVATE_KEY, SSH_HOST, SSH_USER)" },
];

export function StudioCanvasPage() {
  const [project, setProject] = useState<BuilderProject | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"canvas" | "preview">("canvas");
  const [selectedCompId, setSelectedCompId] = useState<string | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);
  const [notice, setNotice] = useState<{ type: "success" | "error" | "info"; message: string; checksum?: string } | null>(null);
  const [deployModalOpen, setDeployModalOpen] = useState(false);
  const [deployStatus, setDeployStatus] = useState<{ status: string; message: string } | null>(null);
  const [previewSrc, setPreviewSrc] = useState<string | null>(null);
  const [, startTransition] = useTransition();

  // Initialize or fetch the active builder project from the real backend
  const loadProject = useCallback(async () => {
    try {
      setLoading(true);
      const res = await listBuilders();
      if (res.projects && res.projects.length > 0) {
        // Pick the most recent project or existing
        const latest = res.projects[res.projects.length - 1];
        const detail = await getBuilder(latest.id);
        setProject(detail);
      } else {
        // Create initial starter project via real backend API
        const created = await createBuilder({
          name: "Hades Studio App",
          kind: "website",
          components: [
            {
              id: "comp_hero",
              type: "heading",
              props: { text: "Bem-vindo ao Studio do Hades", level: "h1" },
              x: 40,
              y: 40,
              width: 700,
              height: 70,
            },
            {
              id: "comp_lead",
              type: "paragraph",
              props: { text: "Editor visual interativo de ponta a ponta com preview ao vivo, undo/redo e export rastreável." },
              x: 40,
              y: 120,
              width: 700,
              height: 50,
            },
            {
              id: "comp_cta",
              type: "button",
              props: { label: "Experimentar Agora" },
              x: 40,
              y: 190,
              width: 180,
              height: 48,
            },
          ],
        });
        setProject(created);
      }
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Erro ao carregar projeto: ${String(err)}` });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProject();
  }, [loadProject]);

  // A cross-origin iframe cannot attach an Authorization header itself. Fetch
  // the backend preview through the authenticated client and render only the
  // resulting blob inside a restricted sandbox.
  useEffect(() => {
    if (activeTab !== "preview" || !project) {
      setPreviewSrc(null);
      return;
    }
    let objectURL: string | null = null;
    let cancelled = false;
    void agentFetchBlob(`/api/agent/v1/builders/${encodeURIComponent(project.id)}/preview/index.html`)
      .then((blob) => {
        if (cancelled) return;
        objectURL = URL.createObjectURL(blob);
        setPreviewSrc(objectURL);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setPreviewSrc(null);
          setNotice({ type: "error", message: `Erro ao carregar preview autenticado: ${String(err)}` });
        }
      });
    return () => {
      cancelled = true;
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [activeTab, project]);

  // Synchronize visual components with the real backend
  const syncComponents = async (newComponents: VisualComponent[]) => {
    if (!project) return;
    try {
      const updated = await updateBuilderVisual(project.id, newComponents, project.version);
      setProject(updated);
      setNotice({ type: "info", message: `Canvas atualizado (v${updated.version})` });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Erro ao salvar canvas: ${String(err)}` });
    }
  };

  // Add component from palette
  const handleAddComponent = (tmpl: (typeof COMPONENT_TEMPLATES)[0]) => {
    if (!project) return;
    const current = project.components || [];
    const newId = `comp_${Date.now()}`;
    const newComponent: VisualComponent = {
      id: newId,
      type: tmpl.type,
      props: { ...tmpl.defaultProps },
      style: { ...tmpl.defaultStyle },
      x: 40,
      y: 40 + current.length * 60,
      width: tmpl.width,
      height: tmpl.height,
    };
    const nextComponents = [...current, newComponent];
    startTransition(() => {
      syncComponents(nextComponents);
      setSelectedCompId(newId);
    });
  };

  // Move component up/down in the stack
  // Drag-and-drop reordering on the canvas: the reorder itself is a pure,
  // unit-tested function; here we only track the dragged id and persist.
  const handleReorderDrop = (targetId: string) => {
    if (!project?.components || !dragId || dragId === targetId) {
      setDragId(null);
      return;
    }
    const next = reorderById(project.components, dragId, targetId);
    setDragId(null);
    syncComponents(next);
  };

  const handleMoveComponent = (id: string, direction: "up" | "down") => {
    if (!project || !project.components) return;
    const comps = [...project.components];
    const index = comps.findIndex((c) => c.id === id);
    if (index === -1) return;
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= comps.length) return;

    const temp = comps[index];
    comps[index] = comps[targetIndex];
    comps[targetIndex] = temp;
    syncComponents(comps);
  };

  // Remove component
  const handleDeleteComponent = (id: string) => {
    if (!project || !project.components) return;
    const next = project.components.filter((c) => c.id !== id);
    syncComponents(next);
    if (selectedCompId === id) setSelectedCompId(null);
  };

  // Update selected component props
  const handleUpdateProps = (id: string, propKey: string, value: string) => {
    if (!project || !project.components) return;
    const next = project.components.map((c) => {
      if (c.id === id) {
        return {
          ...c,
          props: { ...c.props, [propKey]: value },
        };
      }
      return c;
    });
    syncComponents(next);
  };

  // Undo via real backend
  // Instant client-side preview: render the project to standalone HTML and open
  // it in a new tab via a blob URL — no backend build needed, works offline.
  const handleInstantPreview = () => {
    if (!project) return;
    try {
      const html = generateStudioHTML(project);
      const blob = new Blob([html], { type: "text/html" });
      const url = URL.createObjectURL(blob);
      const tab = window.open(url, "_blank", "noopener,noreferrer");
      if (!tab) {
        setNotice({ type: "error", message: "O navegador bloqueou a prévia. Permita pop-ups para visualizar." });
      }
      // Revoke after a delay so the new tab has time to load the document.
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Não foi possível gerar a prévia: ${String(err)}` });
    }
  };

  // Download the project as a standalone .html file (offline export).
  const handleDownloadHTML = () => {
    if (!project) return;
    try {
      const html = generateStudioHTML(project);
      const blob = new Blob([html], { type: "text/html" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      const safeName = (project.name || "meu-app").replace(/[^a-zA-Z0-9-_]+/g, "-").toLowerCase();
      a.href = url;
      a.download = `${safeName || "meu-app"}.html`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 10_000);
      setNotice({ type: "info", message: "HTML do app baixado." });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Não foi possível baixar o HTML: ${String(err)}` });
    }
  };

  const handleUndo = async () => {
    if (!project) return;
    try {
      const undone = await undoBuilder(project.id, project.version);
      setProject(undone);
      setNotice({ type: "info", message: `Undo executado (v${undone.version})` });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Nada para desfazer: ${String(err)}` });
    }
  };

  // Redo via real backend
  const handleRedo = async () => {
    if (!project) return;
    try {
      const redone = await redoBuilder(project.id, project.version);
      setProject(redone);
      setNotice({ type: "info", message: `Redo executado (v${redone.version})` });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Nada para refazer: ${String(err)}` });
    }
  };

  // Preview via real backend
  const handlePreview = async () => {
    if (!project) return;
    try {
      const res = await previewBuilder(project.id);
      setProject(res.project);
      setActiveTab("preview");
      setNotice({ type: "success", message: "Preview ao vivo gerado no backend com sucesso!" });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Erro ao gerar preview: ${String(err)}` });
    }
  };

  // Traceable Export (ZIP) with SHA-256 Checksum
  const handleExport = async () => {
    if (!project) return;
    try {
      const res = await exportBuilder(project.id);
      setProject(res.project);
      const checksum = res.checksum || res.sha256;
      setNotice({
        type: "success",
        message: `Projeto exportado com sucesso (SHA-256 verificado)!`,
        checksum: checksum,
      });

      // Trigger an authenticated download; an anchor alone cannot carry Bearer.
      const archive = await agentFetchBlob(res.download_url);
      const archiveURL = URL.createObjectURL(archive);
      const link = document.createElement("a");
      link.href = archiveURL;
      link.download = `${project.name.replace(/\s+/g, "_")}.zip`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.setTimeout(() => URL.revokeObjectURL(archiveURL), 0);
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Erro ao exportar: ${String(err)}` });
    }
  };

  // Criação a partir de modelo: confirma antes de substituir o projeto aberto e
  // trata erros (os botões antigos trocavam o projeto sem aviso e sem .catch).
  const handleCreateTemplate = async (name: string, kind: BuilderProject["kind"]) => {
    if (project && !window.confirm(`Criar "${name}"? O projeto atualmente aberto no editor será substituído por um novo projeto em branco.`)) {
      return;
    }
    try {
      const created = await createBuilder({ name, kind });
      setProject(created);
      setSelectedCompId(null);
      setNotice({ type: "success", message: `Novo projeto "${name}" criado.` });
    } catch (err: unknown) {
      setNotice({ type: "error", message: `Erro ao criar projeto a partir do modelo: ${String(err)}` });
    }
  };

  // Publicação real: chama o Deploy Adapter do backend e mostra o resultado
  // honesto — sucesso com URL, estado em processamento, ou o motivo real (ex.:
  // credencial ausente), sempre citando a credencial correta do provedor.
  const handleDeployAttempt = async (provider: (typeof DEPLOY_PROVIDERS)[number]) => {
    if (!project) return;
    setDeployStatus({ status: "Publicando…", message: `Enviando o projeto para ${provider.label} pelo Deploy Adapter…` });
    try {
      const result = await deployBuilder(project.id, provider.slug, {});
      const deployment = (result as { deployment?: { status?: string; url?: string }; status?: string; url?: string }).deployment ?? result;
      const url = deployment?.url;
      const rawStatus = deployment?.status ?? "";
      const humanStatus = missionStateLabel(rawStatus);
      if (url) {
        setDeployStatus({ status: humanStatus || "Publicado", message: `Publicado em ${provider.label}. URL: ${url}` });
      } else {
        setDeployStatus({
          status: humanStatus || "Em processamento",
          message: `${provider.label}: solicitação de publicação aceita pelo runtime${humanStatus ? ` (estado: ${humanStatus})` : ""}. Acompanhe a conclusão e a URL final.`,
        });
      }
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      setDeployStatus({
        status: "Bloqueado (dependência externa)",
        message: `Não foi possível publicar em ${provider.label}: ${reason}. Configure a credencial no ambiente do servidor: ${provider.credential}. O sistema não falsifica a publicação.`,
      });
    }
  };

  const selectedComponent = project?.components?.find((c) => c.id === selectedCompId);

  return (
    <SidebarLayout title="Studio" sidebar={<AppSidebar current="studio" />}>
      <div className="flex h-[calc(100vh-4rem)] flex-col bg-neutral-50 dark:bg-neutral-950">
        {/* Top Navbar */}
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-neutral-200 bg-white px-4 dark:border-neutral-800 dark:bg-neutral-900 lg:px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-neutral-900 text-white dark:bg-white dark:text-neutral-900">
              <SparklesIcon className="h-5 w-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-sm font-semibold text-neutral-900 dark:text-white sm:text-base">
                  {project?.name || "Editor do Studio"}
                </h1>
                {project && (
                  <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-mono text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                    v{project.version}
                  </span>
                )}
                {project?.kind && (
                  <span className="hidden rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300 sm:inline-block">
                    {project.kind}
                  </span>
                )}
              </div>
              <p className="text-xs text-neutral-400">Monte um site ou app adicionando componentes com um clique e reordenando a lista; veja a prévia e exporte em ZIP.</p>
            </div>
          </div>

          <div className="flex items-center gap-1.5 sm:gap-2">
            {/* Undo / Redo */}
            <div className="flex items-center rounded-lg border border-neutral-200 bg-neutral-50 p-0.5 dark:border-neutral-800 dark:bg-neutral-800/60">
              <button
                type="button"
                onClick={handleUndo}
                disabled={!project || !project.undo_stack || project.undo_stack.length === 0}
                className="rounded p-1.5 text-neutral-600 hover:bg-white hover:text-neutral-900 disabled:opacity-30 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-white"
                title="Desfazer (Undo)"
              >
                <ArrowUturnLeftIcon className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={handleRedo}
                disabled={!project || !project.redo_stack || project.redo_stack.length === 0}
                className="rounded p-1.5 text-neutral-600 hover:bg-white hover:text-neutral-900 disabled:opacity-30 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-white"
                title="Refazer (Redo)"
              >
                <ArrowUturnRightIcon className="h-4 w-4" />
              </button>
            </div>

            {/* Mode Toggle: Canvas vs Preview */}
            <div className="flex items-center rounded-lg border border-neutral-200 bg-neutral-50 p-0.5 dark:border-neutral-800 dark:bg-neutral-800/60">
              <button
                type="button"
                onClick={() => setActiveTab("canvas")}
                className={`flex items-center gap-1 rounded px-2.5 py-1 text-xs font-medium ${
                  activeTab === "canvas"
                    ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
                }`}
              >
                <PencilSquareIcon className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Editor</span>
              </button>
              <button
                type="button"
                onClick={handlePreview}
                className={`flex items-center gap-1 rounded px-2.5 py-1 text-xs font-medium ${
                  activeTab === "preview"
                    ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
                }`}
              >
                <EyeIcon className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Prévia</span>
              </button>
            </div>

            {/* Traceable Export (ZIP) */}
            <button
              type="button"
              onClick={handleExport}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-700 shadow-sm hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-750"
              title="Exportar ZIP com checksum SHA-256"
            >
              <ArrowDownTrayIcon className="h-4 w-4" />
              <span className="hidden md:inline">Exportar ZIP</span>
            </button>

            {/* Download the app as standalone HTML */}
            <button
              type="button"
              onClick={handleDownloadHTML}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              title="Baixar o app como um arquivo HTML standalone"
            >
              <ArrowDownTrayIcon className="h-4 w-4" />
              <span className="hidden md:inline">Baixar HTML</span>
            </button>

            {/* Instant client-side preview */}
            <button
              type="button"
              onClick={handleInstantPreview}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              title="Ver o app agora em uma nova aba (prévia instantânea, sem servidor)"
            >
              <EyeIcon className="h-4 w-4" />
              <span className="hidden md:inline">Ver agora</span>
            </button>

            {/* Deploy Adapter Button */}
            <button
              type="button"
              onClick={() => setDeployModalOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-neutral-800 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-100"
            >
              <RocketLaunchIcon className="h-4 w-4" />
              <span className="hidden sm:inline">Publicar</span>
            </button>
          </div>
        </header>

        {/* Notice Bar */}
        {notice && (
          <div
            className={`flex items-center justify-between px-4 py-2 text-xs ${
              notice.type === "success"
                ? "bg-emerald-50 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
                : notice.type === "error"
                ? "bg-red-50 text-red-800 dark:bg-red-950/40 dark:text-red-300"
                : "bg-blue-50 text-blue-800 dark:bg-blue-950/40 dark:text-blue-300"
            }`}
          >
            <div className="flex items-center gap-2">
              {notice.type === "success" ? (
                <CheckCircleIcon className="h-4 w-4 shrink-0 text-emerald-600" />
              ) : notice.type === "error" ? (
                <ExclamationTriangleIcon className="h-4 w-4 shrink-0 text-red-600" />
              ) : (
                <SparklesIcon className="h-4 w-4 shrink-0 text-blue-600" />
              )}
              <span>{notice.message}</span>
              {notice.checksum && (
                <span className="rounded bg-black/10 px-1.5 py-0.5 font-mono text-[10px] dark:bg-white/10">
                  SHA-256: {notice.checksum.slice(0, 16)}...{notice.checksum.slice(-8)}
                </span>
              )}
            </div>
            <button
              type="button"
              onClick={() => setNotice(null)}
              className="text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200"
            >
              &times;
            </button>
          </div>
        )}

        {/* Main Work Area */}
        <div className="flex flex-1 overflow-hidden">
          {/* Left Palette (Components) */}
          <aside className="hidden md:block w-56 shrink-0 border-r border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 overflow-y-auto">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
              Componentes
            </h2>
            <div className="mt-3 space-y-2">
              {COMPONENT_TEMPLATES.map((tmpl) => (
                <button
                  key={tmpl.type}
                  type="button"
                  onClick={() => handleAddComponent(tmpl)}
                  className="flex w-full items-center gap-2.5 rounded-xl border border-neutral-200/80 bg-neutral-50/60 p-2.5 text-left text-xs font-medium text-neutral-800 transition hover:border-neutral-300 hover:bg-white hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-800/40 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-neutral-200 font-bold text-neutral-700 dark:bg-neutral-700 dark:text-neutral-200">
                    {tmpl.icon}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-semibold">{tmpl.label}</div>
                    <div className="text-[10px] text-neutral-400">Clique para adicionar</div>
                  </div>
                  <PlusIcon className="h-4 w-4 shrink-0 text-neutral-400" />
                </button>
              ))}
            </div>

            <div className="mt-6 border-t border-neutral-100 pt-4 dark:border-neutral-800">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
                Modelos de Projeto
              </h3>
              <div className="mt-2 space-y-1 text-xs">
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("Novo Site Web", "website")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <GlobeAltIcon className="h-4 w-4 text-emerald-500" />
                  Site / Landing Page
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("Novo Dashboard", "dashboard")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <ChartBarIcon className="h-4 w-4 text-blue-500" />
                  Dashboard Analítico
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("Nova Apresentação", "slides")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <PresentationChartBarIcon className="h-4 w-4 text-violet-500" />
                  Slides / Pitch Deck
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("Novo Jogo Web", "game")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <PuzzlePieceIcon className="h-4 w-4 text-amber-500" />
                  Jogo 2D Interativo
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("Novo App Móvel", "app")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <DevicePhoneMobileIcon className="h-4 w-4 text-rose-500" />
                  Aplicação Web App
                </button>
              </div>
            </div>
          </aside>

          {/* Center Workspace (Canvas or Preview) */}
          <main className="flex-1 overflow-auto p-4 lg:p-6">
            {/* Mobile Component Strip */}
            <div className="md:hidden flex items-center gap-1.5 overflow-x-auto pb-3 mb-3 border-b border-neutral-200 dark:border-neutral-800">
              <span className="text-[10px] uppercase font-bold text-neutral-400 shrink-0">Adicionar:</span>
              {COMPONENT_TEMPLATES.map((tmpl) => (
                <button
                  key={tmpl.type}
                  type="button"
                  onClick={() => handleAddComponent(tmpl)}
                  className="flex items-center gap-1 shrink-0 rounded-lg border border-neutral-200 bg-white px-2 py-1 text-xs font-medium text-neutral-800 shadow-xs dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                >
                  <span className="text-neutral-400">+</span>
                  <span>{tmpl.label}</span>
                </button>
              ))}
            </div>
            {activeTab === "canvas" ? (
              <div className="mx-auto max-w-4xl">
                <div className="mb-3 flex items-center justify-between">
                  <div className="text-xs font-medium text-neutral-500">
                    Área do Canvas ({project?.components?.length || 0} componentes)
                  </div>
                  <div className="text-xs text-neutral-400">
                    Clique em um componente para editar suas propriedades
                  </div>
                </div>

                <div className="min-h-[500px] rounded-2xl border border-dashed border-neutral-300 bg-white p-6 shadow-sm dark:border-neutral-800 dark:bg-neutral-900/60">
                  {loading ? (
                    <div className="flex flex-col items-center justify-center py-20 text-center text-neutral-400">
                      <SparklesIcon className="h-8 w-8 animate-pulse text-neutral-400" />
                      <p className="mt-2 text-xs">Carregando canvas do Studio...</p>
                    </div>
                  ) : (!project?.components || project.components.length === 0) ? (
                    <div className="flex flex-col items-center justify-center py-20 text-center text-neutral-400">
                      <SparklesIcon className="h-10 w-10 text-neutral-300 dark:text-neutral-600" />
                      <p className="mt-3 text-sm font-medium text-neutral-600 dark:text-neutral-300">
                        O canvas está vazio
                      </p>
                      <p className="mt-1 text-xs text-neutral-400">
                        Clique em um componente na barra lateral esquerda para começar a montar.
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      {project.components.map((comp, idx) => {
                        const isSelected = comp.id === selectedCompId;
                        return (
                          <div
                            key={comp.id}
                            onClick={() => setSelectedCompId(comp.id)}
                            draggable
                            onDragStart={(e) => {
                              setDragId(comp.id);
                              e.dataTransfer.effectAllowed = "move";
                            }}
                            onDragOver={(e) => {
                              e.preventDefault();
                              e.dataTransfer.dropEffect = "move";
                            }}
                            onDrop={(e) => {
                              e.preventDefault();
                              handleReorderDrop(comp.id);
                            }}
                            onDragEnd={() => setDragId(null)}
                            title="Arraste para reordenar"
                            className={`group relative rounded-xl border p-4 transition cursor-pointer ${dragId === comp.id ? "opacity-50 ring-2 ring-violet-400" : ""} ${
                              isSelected
                                ? "border-neutral-900 bg-neutral-50/80 shadow-md ring-2 ring-neutral-900/10 dark:border-white dark:bg-neutral-800/80 dark:ring-white/10"
                                : "border-neutral-200/80 bg-white hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900"
                            }`}
                          >
                            {/* Badges / Controls */}
                            <div className="mb-2 flex items-center justify-between text-xs">
                              <span className="rounded bg-neutral-100 px-2 py-0.5 font-mono text-[10px] uppercase text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                                #{idx + 1} {comp.type}
                              </span>
                              <div className="flex items-center gap-1 opacity-0 transition group-hover:opacity-100">
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleMoveComponent(comp.id, "up");
                                  }}
                                  disabled={idx === 0}
                                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 disabled:opacity-20 dark:hover:bg-neutral-800"
                                  title="Mover para cima"
                                >
                                  <ArrowUpIcon className="h-3.5 w-3.5" />
                                </button>
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleMoveComponent(comp.id, "down");
                                  }}
                                  disabled={idx === (project?.components?.length ?? 0) - 1}
                                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 disabled:opacity-20 dark:hover:bg-neutral-800"
                                  title="Mover para baixo"
                                >
                                  <ArrowDownIcon className="h-3.5 w-3.5" />
                                </button>
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleDeleteComponent(comp.id);
                                  }}
                                  className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30"
                                  title="Excluir"
                                >
                                  <TrashIcon className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </div>

                            {/* Render Component Content */}
                            {comp.type === "heading" && (
                              <h2 className="text-xl font-bold text-neutral-900 dark:text-white">
                                {comp.props?.text || "Título"}
                              </h2>
                            )}
                            {comp.type === "paragraph" && (
                              <p className="text-sm text-neutral-600 dark:text-neutral-300">
                                {comp.props?.text || "Texto do parágrafo"}
                              </p>
                            )}
                            {comp.type === "button" && (
                              <button
                                type="button"
                                className="inline-flex rounded-lg bg-neutral-900 px-4 py-2 text-xs font-semibold text-white shadow-sm dark:bg-white dark:text-neutral-900"
                              >
                                {comp.props?.label || "Botão"}
                              </button>
                            )}
                            {comp.type === "card" && (
                              <div className="rounded-lg border border-neutral-200 bg-neutral-50/50 p-3 dark:border-neutral-800 dark:bg-neutral-800/40">
                                <h3 className="font-semibold text-neutral-900 dark:text-white">
                                  {comp.props?.title || "Título do Card"}
                                </h3>
                                <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                                  {comp.props?.body || "Corpo do card explicativo."}
                                </p>
                              </div>
                            )}
                            {comp.type === "metric" && (
                              <div className="rounded-lg bg-neutral-100 p-3 dark:bg-neutral-800">
                                <div className="text-[11px] uppercase tracking-wider text-neutral-400">
                                  {comp.props?.label || "Métrica"}
                                </div>
                                <div className="mt-1 text-2xl font-bold text-neutral-900 dark:text-white">
                                  {comp.props?.value || "0"}
                                </div>
                              </div>
                            )}
                            {comp.type === "navbar" && (
                              <div className="flex items-center justify-between border-b border-neutral-200 pb-2 dark:border-neutral-800">
                                <div className="font-bold text-neutral-900 dark:text-white">
                                  {comp.props?.brand || "Brand"}
                                </div>
                                <div className="flex gap-3 text-xs text-neutral-500">
                                  {(comp.props?.links || "Link 1, Link 2").split(",").map((l, i) => (
                                    <span key={i}>{l.trim()}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {comp.type === "hero" && (
                              <div className="rounded-xl bg-gradient-to-br from-neutral-900 to-neutral-700 p-6 text-center text-white dark:from-neutral-100 dark:to-neutral-300 dark:text-neutral-900">
                                <h1 className="text-2xl font-bold">{comp.props?.title || "Título do Hero"}</h1>
                                <p className="mt-2 text-sm opacity-80">{comp.props?.subtitle || "Subtítulo explicativo"}</p>
                                <span className="mt-4 inline-flex rounded-lg bg-white px-4 py-2 text-xs font-semibold text-neutral-900 dark:bg-neutral-900 dark:text-white">{comp.props?.cta || "Ação"}</span>
                              </div>
                            )}
                            {comp.type === "image" && (
                              comp.props?.src ? (
                                <img src={comp.props.src} alt={comp.props?.alt || ""} className="max-w-full rounded-lg" />
                              ) : (
                                <div className="flex h-24 items-center justify-center rounded-lg border border-dashed border-neutral-300 text-xs text-neutral-400 dark:border-neutral-700">Imagem (defina a URL nas propriedades)</div>
                              )
                            )}
                            {comp.type === "input" && (
                              <label className="block text-xs">
                                <span className="text-neutral-700 dark:text-neutral-300">{comp.props?.label || "Rótulo"}</span>
                                <input type="text" disabled placeholder={comp.props?.placeholder || ""} className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900" />
                              </label>
                            )}
                            {comp.type === "link" && (
                              <span className="text-sm font-medium text-blue-600 underline dark:text-blue-400">{comp.props?.text || "Link"}</span>
                            )}
                            {comp.type === "list" && (
                              <ul className="list-disc space-y-1 pl-5 text-sm text-neutral-700 dark:text-neutral-300">
                                {(comp.props?.items || "Item 1, Item 2").split(",").map((it, i) => (
                                  <li key={i}>{it.trim()}</li>
                                ))}
                              </ul>
                            )}
                            {comp.type === "divider" && (
                              <hr className="border-neutral-200 dark:border-neutral-800" />
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
            ) : (
              /* Live Preview Mode (Real iframe to backend preview) */
              <div className="flex h-full flex-col">
                <div className="mb-2 flex items-center justify-between text-xs text-neutral-500">
                  <span>Prévia autenticada: carregada com a sua sessão; a URL interna é protegida e não é compartilhável.</span>
                  <a
                    href={previewSrc ?? undefined}
                    target="_blank"
                    rel="noreferrer"
                    className={`font-medium text-blue-600 hover:underline dark:text-blue-400 ${previewSrc ? "" : "pointer-events-none opacity-50"}`}
                  >
                    Abrir em nova aba &rarr;
                  </a>
                </div>
                <div className="flex-1 overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
                  <iframe
                    src={previewSrc ?? undefined}
                    title="Prévia ao vivo"
                    sandbox="allow-scripts"
                    className="h-full w-full border-0 bg-white"
                  />
                </div>
              </div>
            )}
          </main>

          {/* Right Inspector (Properties of selected component) */}
          {selectedComponent && activeTab === "canvas" && (
            <aside className="w-72 shrink-0 border-l border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 overflow-y-auto">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
                  Propriedades
                </h3>
                <button
                  type="button"
                  onClick={() => setSelectedCompId(null)}
                  className="text-neutral-400 hover:text-neutral-600"
                >
                  &times;
                </button>
              </div>

              <div className="mt-4 space-y-3 text-xs">
                <div>
                  <label className="block text-neutral-500">Tipo</label>
                  <input
                    type="text"
                    disabled
                    value={selectedComponent.type}
                    className="mt-1 w-full rounded-lg border border-neutral-200 bg-neutral-100 px-2.5 py-1.5 font-mono text-neutral-600 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-400"
                  />
                </div>

                {selectedComponent.props?.text !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Texto</label>
                    <textarea
                      rows={3}
                      value={selectedComponent.props.text}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "text", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 p-2 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.title !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Título</label>
                    <input
                      type="text"
                      value={selectedComponent.props.title}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "title", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.body !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Corpo</label>
                    <textarea
                      rows={3}
                      value={selectedComponent.props.body}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "body", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 p-2 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.label !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Rótulo</label>
                    <input
                      type="text"
                      value={selectedComponent.props.label}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "label", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.value !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Valor</label>
                    <input
                      type="text"
                      value={selectedComponent.props.value}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "value", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.brand !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Marca</label>
                    <input
                      type="text"
                      value={selectedComponent.props.brand}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "brand", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.links !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Links (separados por vírgula)</label>
                    <textarea
                      rows={2}
                      value={selectedComponent.props.links}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "links", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 p-2 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.change !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Variação</label>
                    <input
                      type="text"
                      value={selectedComponent.props.change}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "change", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                {selectedComponent.props?.action !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">Ação</label>
                    <input
                      type="text"
                      value={selectedComponent.props.action}
                      onChange={(e) => handleUpdateProps(selectedComponent.id, "action", e.target.value)}
                      className="mt-1 w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-neutral-900 outline-none focus:border-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                    />
                  </div>
                )}

                <div className="pt-4 border-t border-neutral-100 dark:border-neutral-800 flex gap-2">
                  <button
                    type="button"
                    onClick={() => handleDeleteComponent(selectedComponent.id)}
                    className="flex-1 rounded-lg border border-red-200 bg-red-50 py-1.5 font-medium text-red-700 hover:bg-red-100 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300"
                  >
                    Excluir
                  </button>
                </div>
              </div>
            </aside>
          )}
        </div>

        {/* Deploy Adapter Modal */}
        {deployModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
            <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl dark:bg-neutral-900">
              <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                Publicar projeto
              </h3>
              <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                Selecione o provedor de nuvem para hospedar seu projeto. Sem credenciais ativas, o sistema mantém status honesto.
              </p>

              <div className="mt-4 space-y-2">
                {DEPLOY_PROVIDERS.map((prov) => (
                  <button
                    key={prov.slug}
                    type="button"
                    onClick={() => void handleDeployAttempt(prov)}
                    className="flex w-full items-center justify-between rounded-xl border border-neutral-200 p-3 text-left text-xs font-medium text-neutral-800 transition hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-800"
                  >
                    <span>{prov.label}</span>
                    <span className="rounded bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
                      Requer credenciais
                    </span>
                  </button>
                ))}
              </div>

              {deployStatus && (
                <div className="mt-4 rounded-xl bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
                  <div className="font-semibold">{deployStatus.status}</div>
                  <div className="mt-1">{deployStatus.message}</div>
                </div>
              )}

              <div className="mt-6 flex justify-end">
                <button
                  type="button"
                  onClick={() => {
                    setDeployModalOpen(false);
                    setDeployStatus(null);
                  }}
                  className="rounded-lg border border-neutral-300 px-4 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  Fechar
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </SidebarLayout>
  );
}
