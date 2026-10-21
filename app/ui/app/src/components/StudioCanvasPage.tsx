import { useCallback, useEffect, useRef, useState, useTransition } from "react";
import { AppSidebar } from "@/components/AppSidebar";
import { LiveBuildPanel } from "@/components/LiveBuildPage";
import { SidebarLayout } from "@/components/layout/layout";
import { missionStateLabel } from "@/lib/labels";
import { generateStudioHTML } from "@/lib/studioHtml";
import { reorderById } from "@/lib/studioReorder";
import { buildGenerationPrompt, parseGeneratedComponents, pickLocalFirstModel, PRIMARY_TEXT_PROP } from "@/lib/studioAI";
import { confirmDialog } from "@/lib/confirmDialog";
import { API_BASE } from "@/lib/config";
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
  BoltIcon,
} from "@heroicons/react/24/outline";
// Relatório de publicação local (E-281/E-282): o que a cópia ENTREGOU de fato.
import { PublishReportNotice } from "./PublishReportNotice";
import { useT } from "@/i18n";

// Os modelos guardam CHAVES de i18n, não texto: o rótulo é resolvido no render e
// o conteúdo inicial no momento de inserir, para acompanhar o idioma activo.
const COMPONENT_TEMPLATES: Array<{
  type: string;
  labelKey: string;
  icon: string;
  defaultProps: Record<string, string>;
  textProps: Record<string, string>;
  defaultStyle: Record<string, string>;
  width: number;
  height: number;
}> = [
  {
    type: "heading",
    labelKey: "studio.template.heading.label",
    icon: "H",
    defaultProps: { level: "h1" },
    textProps: { text: "studio.template.heading.text" },
    defaultStyle: { color: "#111827", fontSize: "2rem", fontWeight: "bold" },
    width: 600,
    height: 60,
  },
  {
    type: "paragraph",
    labelKey: "studio.template.paragraph.label",
    icon: "P",
    defaultProps: {},
    textProps: { text: "studio.template.paragraph.text" },
    defaultStyle: { color: "#4B5563", fontSize: "1rem" },
    width: 600,
    height: 50,
  },
  {
    type: "button",
    labelKey: "studio.template.button.label",
    icon: "B",
    defaultProps: { action: "submit" },
    textProps: { label: "studio.template.button.text" },
    defaultStyle: { backgroundColor: "#111827", color: "#FFFFFF", borderRadius: "8px" },
    width: 160,
    height: 44,
  },
  {
    type: "card",
    labelKey: "studio.template.card.label",
    icon: "C",
    defaultProps: {},
    textProps: {
      title: "studio.template.card.title",
      body: "studio.template.card.body",
    },
    defaultStyle: { backgroundColor: "#F9FAFB", borderColor: "#E5E7EB", borderRadius: "12px" },
    width: 320,
    height: 160,
  },
  {
    type: "metric",
    labelKey: "studio.template.metric.label",
    icon: "M",
    defaultProps: { value: "99.8%", change: "+4.2%" },
    textProps: { label: "studio.template.metric.sampleLabel" },
    defaultStyle: { backgroundColor: "#F3F4F6", borderRadius: "8px" },
    width: 200,
    height: 90,
  },
  {
    type: "navbar",
    labelKey: "studio.template.navbar.label",
    icon: "N",
    defaultProps: { brand: "Ollama Studio" },
    textProps: { links: "studio.template.navbar.links" },
    defaultStyle: { backgroundColor: "#FFFFFF", borderBottom: "1px solid #E5E7EB" },
    width: 800,
    height: 60,
  },
  {
    type: "hero",
    labelKey: "studio.template.hero.label",
    icon: "★",
    defaultProps: {},
    textProps: {
      title: "studio.template.hero.title",
      subtitle: "studio.template.hero.subtitle",
      cta: "studio.template.hero.cta",
    },
    defaultStyle: {},
    width: 800,
    height: 220,
  },
  {
    type: "image",
    labelKey: "studio.template.image.label",
    icon: "I",
    defaultProps: { src: "" },
    textProps: { alt: "studio.template.image.alt" },
    defaultStyle: {},
    width: 400,
    height: 240,
  },
  {
    type: "input",
    labelKey: "studio.template.input.label",
    icon: "F",
    defaultProps: {},
    textProps: {
      label: "studio.template.input.sampleLabel",
      placeholder: "studio.template.input.placeholder",
    },
    defaultStyle: {},
    width: 320,
    height: 70,
  },
  {
    type: "link",
    labelKey: "studio.template.link.label",
    icon: "L",
    defaultProps: { href: "https://" },
    textProps: { text: "studio.template.link.text" },
    defaultStyle: {},
    width: 160,
    height: 32,
  },
  {
    type: "list",
    labelKey: "studio.template.list.label",
    icon: "≡",
    defaultProps: {},
    textProps: { items: "studio.template.list.items" },
    defaultStyle: {},
    width: 400,
    height: 120,
  },
  {
    type: "divider",
    labelKey: "studio.template.divider.label",
    icon: "—",
    defaultProps: {},
    textProps: {},
    defaultStyle: {},
    width: 600,
    height: 20,
  },
  {
    type: "pricing",
    labelKey: "studio.template.pricing.label",
    icon: "$",
    defaultProps: {},
    textProps: {
      plan: "studio.template.pricing.plan",
      price: "studio.template.pricing.price",
      features: "studio.template.pricing.features",
      cta: "studio.template.pricing.cta",
    },
    defaultStyle: { backgroundColor: "#F9FAFB", borderColor: "#E5E7EB", borderRadius: "12px" },
    width: 280,
    height: 260,
  },
  {
    type: "testimonial",
    labelKey: "studio.template.testimonial.label",
    icon: "“",
    defaultProps: {},
    textProps: {
      quote: "studio.template.testimonial.quote",
      author: "studio.template.testimonial.author",
    },
    defaultStyle: {},
    width: 420,
    height: 110,
  },
  {
    type: "faq",
    labelKey: "studio.template.faq.label",
    icon: "?",
    defaultProps: {},
    textProps: {
      question: "studio.template.faq.question",
      answer: "studio.template.faq.answer",
    },
    defaultStyle: { borderColor: "#E5E7EB", borderRadius: "8px" },
    width: 480,
    height: 90,
  },
  {
    type: "footer",
    labelKey: "studio.template.footer.label",
    icon: "_",
    defaultProps: {},
    textProps: {
      text: "studio.template.footer.text",
      links: "studio.template.footer.links",
    },
    defaultStyle: {},
    width: 600,
    height: 60,
  },
];

// Provedores de publicação: cada um exige a credencial correta no ambiente do
// servidor (inclusive SSH). O slug é enviado ao Deploy Adapter real do backend.
const DEPLOY_PROVIDERS: Array<{ labelKey: string; slug: string; credentialKey: string }> = [
  { labelKey: "studio.deploy.provider.vercel", slug: "vercel", credentialKey: "studio.deploy.credential.vercel" },
  { labelKey: "studio.deploy.provider.cloudflare", slug: "cloudflare", credentialKey: "studio.deploy.credential.cloudflare" },
  { labelKey: "studio.deploy.provider.netlify", slug: "netlify", credentialKey: "studio.deploy.credential.netlify" },
  { labelKey: "studio.deploy.provider.ssh", slug: "ssh", credentialKey: "studio.deploy.credential.ssh" },
];

/** Resolve as chaves de texto do modelo no idioma activo, no momento de inserir. */
function translateProps(
  keys: Record<string, string>,
  translate: (key: string) => string,
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(keys).map(([prop, key]) => [prop, translate(key)]),
  );
}

export function StudioCanvasPage() {
  const { t } = useT();
  const [project, setProject] = useState<BuilderProject | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"canvas" | "preview" | "live">("canvas");
  const [selectedCompId, setSelectedCompId] = useState<string | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);
  const [aiPrompt, setAiPrompt] = useState("");
  const [aiGenerating, setAiGenerating] = useState(false);
  const [aiReplace, setAiReplace] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [instantPreviewHtml, setInstantPreviewHtml] = useState<string | null>(null);

  useEffect(() => {
    if (instantPreviewHtml === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setInstantPreviewHtml(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [instantPreviewHtml]);
  const [notice, setNotice] = useState<{ type: "success" | "error" | "info"; message: string; checksum?: string } | null>(null);
  const [deployModalOpen, setDeployModalOpen] = useState(false);
  const [deployStatus, setDeployStatus] = useState<{ status: string; message: string } | null>(null);
  const [previewSrc, setPreviewSrc] = useState<string | null>(null);
  // The authenticated preview is a blob: URL; the webview swallows
  // target="_blank"/window.open for it, so "open in a new tab" is shown as an
  // in-app expanded overlay instead.
  const [previewExpanded, setPreviewExpanded] = useState(false);
  useEffect(() => {
    if (!previewExpanded) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setPreviewExpanded(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [previewExpanded]);
  useEffect(() => {
    if (!previewSrc) setPreviewExpanded(false);
  }, [previewSrc]);
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
              props: { text: t("studio.seed.hero"), level: "h1" },
              x: 40,
              y: 40,
              width: 700,
              height: 70,
            },
            {
              id: "comp_lead",
              type: "paragraph",
              props: { text: t("studio.seed.lead") },
              x: 40,
              y: 120,
              width: 700,
              height: 50,
            },
            {
              id: "comp_cta",
              type: "button",
              props: { label: t("studio.seed.cta") },
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
      setNotice({ type: "error", message: t("studio.notice.loadError", { error: String(err) }) });
    } finally {
      setLoading(false);
    }
  }, [t]);

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
          setNotice({ type: "error", message: t("studio.notice.previewAuthError", { error: String(err) }) });
        }
      });
    return () => {
      cancelled = true;
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [activeTab, project, t]);

  // Synchronize visual components with the real backend
  const syncComponents = async (newComponents: VisualComponent[]) => {
    if (!project) return;
    try {
      const updated = await updateBuilderVisual(project.id, newComponents, project.version);
      setProject(updated);
      setNotice({ type: "info", message: t("studio.notice.canvasUpdated", { version: updated.version }) });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.canvasSaveError", { error: String(err) }) });
    }
  };

  // Debounced prop saves. Editing an inspector field must not POST (and bump
  // project.version) on every keystroke — that floods the backend and, because
  // each save carries the optimistic-concurrency version, causes spurious
  // version-conflict errors mid-typing. We keep a ref to the latest
  // syncComponents and flush the pending edit after a short idle, on switching
  // components, and on unmount.
  const syncRef = useRef(syncComponents);
  syncRef.current = syncComponents;
  const propsSaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingComponentsRef = useRef<VisualComponent[] | null>(null);
  const flushPendingProps = useCallback(() => {
    if (propsSaveTimerRef.current) {
      clearTimeout(propsSaveTimerRef.current);
      propsSaveTimerRef.current = null;
    }
    const pending = pendingComponentsRef.current;
    pendingComponentsRef.current = null;
    if (pending) syncRef.current(pending);
  }, []);

  // Generate components from a natural-language description using a local model.
  // The model is asked for strict JSON; parseGeneratedComponents never trusts the
  // raw output — only known, sanitized component types reach the canvas.
  const handleGenerateAI = async () => {
    if (!project || !aiPrompt.trim() || aiGenerating) return;
    setAiGenerating(true);
    setNotice({ type: "info", message: t("studio.notice.generating") });
    try {
      const tags = await fetch(`${API_BASE}/api/tags`).then((r) => r.json());
      const names: string[] = (tags?.models ?? [])
        .map((m: { model?: string; name?: string }) => m.model || m.name)
        .filter((n: unknown): n is string => typeof n === "string" && n !== "");
      // Local-first: prefer a model that runs on the user's machine over a cloud one.
      const model = pickLocalFirstModel(names);
      if (!model) {
        setNotice({ type: "error", message: t("studio.notice.noModel") });
        return;
      }
      const res = await fetch(`${API_BASE}/api/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ model, prompt: buildGenerationPrompt(aiPrompt), stream: false, format: "json" }),
      }).then((r) => r.json());
      const generated = parseGeneratedComponents(String(res?.response ?? ""));
      if (generated.length === 0) {
        setNotice({ type: "error", message: t("studio.notice.invalidComponents") });
        return;
      }
      const base = aiReplace ? [] : project.components || [];
      const positioned: VisualComponent[] = generated.map((c, i) => ({
        ...c,
        x: 40,
        y: 40 + (base.length + i) * 60,
        width: 600,
        height: 80,
      }));
      await syncComponents([...base, ...positioned]);
      setNotice({
        type: "success",
        message: t(
          aiReplace ? "studio.notice.generatedFresh" : "studio.notice.generatedAdded",
          { count: generated.length, model },
        ),
      });
      setAiPrompt("");
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.generateError", { error: String(err) }) });
    } finally {
      setAiGenerating(false);
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
      props: { ...tmpl.defaultProps, ...translateProps(tmpl.textProps, t) },
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
    // Optimistic local update keeps the inspector responsive; the backend save
    // is debounced (see flushPendingProps).
    setProject((prev) => (prev ? { ...prev, components: next } : prev));
    pendingComponentsRef.current = next;
    if (propsSaveTimerRef.current) clearTimeout(propsSaveTimerRef.current);
    propsSaveTimerRef.current = setTimeout(() => {
      propsSaveTimerRef.current = null;
      const pending = pendingComponentsRef.current;
      pendingComponentsRef.current = null;
      if (pending) syncRef.current(pending);
    }, 500);
  };

  // Flush a pending debounced prop save when switching away from a component or
  // unmounting, so the last edit is never lost.
  useEffect(() => {
    return () => flushPendingProps();
  }, [selectedCompId, flushPendingProps]);

  // Undo via real backend
  // Instant client-side preview: render the project to standalone HTML and show
  // it in an in-app modal via <iframe srcDoc>. This works both in a normal
  // browser and inside the desktop app's webview — unlike window.open(blob:…),
  // which the webview cannot navigate to (it 404s against the app server).
  const handleInstantPreview = () => {
    if (!project) return;
    try {
      setInstantPreviewHtml(generateStudioHTML(project));
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.previewError", { error: String(err) }) });
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
      setNotice({ type: "info", message: t("studio.notice.downloaded") });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.downloadError", { error: String(err) }) });
    }
  };

  const handleUndo = async () => {
    if (!project) return;
    try {
      const undone = await undoBuilder(project.id, project.version);
      setProject(undone);
      setNotice({ type: "info", message: t("studio.notice.undo", { version: undone.version }) });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.undoError", { error: String(err) }) });
    }
  };

  // Redo via real backend
  const handleRedo = async () => {
    if (!project) return;
    try {
      const redone = await redoBuilder(project.id, project.version);
      setProject(redone);
      setNotice({ type: "info", message: t("studio.notice.redo", { version: redone.version }) });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.redoError", { error: String(err) }) });
    }
  };

  // Preview via real backend
  const handlePreview = async () => {
    if (!project) return;
    try {
      const res = await previewBuilder(project.id);
      setProject(res.project);
      setActiveTab("preview");
      setNotice({ type: "success", message: t("studio.notice.previewGenerated") });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.notice.previewGenerateError", { error: String(err) }) });
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
        message: t("studio.notice.exported"),
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
      setNotice({ type: "error", message: t("studio.notice.exportError", { error: String(err) }) });
    }
  };

  // Criação a partir de modelo: confirma antes de substituir o projeto aberto e
  // trata erros (os botões antigos trocavam o projeto sem aviso e sem .catch).
  const handleCreateTemplate = async (nameKey: string, kind: BuilderProject["kind"]) => {
    const name = t(nameKey);
    if (project && !(await confirmDialog(t("studio.create.confirm", { name }), { confirmLabel: t("studio.create.confirmLabel") }))) {
      return;
    }
    try {
      const created = await createBuilder({ name, kind });
      setProject(created);
      setSelectedCompId(null);
      setNotice({ type: "success", message: t("studio.create.success", { name }) });
    } catch (err: unknown) {
      setNotice({ type: "error", message: t("studio.create.error", { error: String(err) }) });
    }
  };

  // Publicação real: chama o Deploy Adapter do backend e mostra o resultado
  // honesto — sucesso com URL, estado em processamento, ou o motivo real (ex.:
  // credencial ausente), sempre citando a credencial correta do provedor.
  const handleDeployAttempt = async (provider: (typeof DEPLOY_PROVIDERS)[number]) => {
    if (!project) return;
    const providerLabel = t(provider.labelKey);
    setDeployStatus({ status: t("studio.deploy.publishing"), message: t("studio.deploy.sending", { provider: providerLabel }) });
    try {
      const result = await deployBuilder(project.id, provider.slug, {});
      const deployment = (result as { deployment?: { status?: string; url?: string }; status?: string; url?: string }).deployment ?? result;
      const url = deployment?.url;
      const rawStatus = deployment?.status ?? "";
      const humanStatus = missionStateLabel(rawStatus);
      if (url) {
        setDeployStatus({
          status: humanStatus || t("studio.deploy.published"),
          message: t("studio.deploy.publishedWithUrl", { provider: providerLabel, url }),
        });
      } else {
        setDeployStatus({
          status: humanStatus || t("studio.deploy.processing"),
          message: humanStatus
            ? t("studio.deploy.acceptedWithStatus", { provider: providerLabel, status: humanStatus })
            : t("studio.deploy.accepted", { provider: providerLabel }),
        });
      }
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      setDeployStatus({
        status: t("studio.deploy.blocked"),
        message: t("studio.deploy.failed", {
          provider: providerLabel,
          reason,
          credential: t(provider.credentialKey),
        }),
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
                  {project?.name || t("studio.header.title")}
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
              <p className="text-xs text-neutral-400">{t("studio.header.subtitle")}</p>
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
                title={t("studio.header.undo")}
              >
                <ArrowUturnLeftIcon className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={handleRedo}
                disabled={!project || !project.redo_stack || project.redo_stack.length === 0}
                className="rounded p-1.5 text-neutral-600 hover:bg-white hover:text-neutral-900 disabled:opacity-30 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-white"
                title={t("studio.header.redo")}
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
                <span className="hidden sm:inline">{t("studio.header.tabEditor")}</span>
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
                <span className="hidden sm:inline">{t("studio.header.tabPreview")}</span>
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("live")}
                className={`flex items-center gap-1 rounded px-2.5 py-1 text-xs font-medium ${
                  activeTab === "live"
                    ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
                }`}
              >
                <BoltIcon className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">{t("liveBuild.title")}</span>
              </button>
            </div>

            {/* Traceable Export (ZIP) */}
            <button
              type="button"
              onClick={handleExport}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-700 shadow-sm hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-750"
              title={t("studio.header.exportTitle")}
            >
              <ArrowDownTrayIcon className="h-4 w-4" />
              <span className="hidden md:inline">{t("studio.header.export")}</span>
            </button>

            {/* Download the app as standalone HTML */}
            <button
              type="button"
              onClick={handleDownloadHTML}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              title={t("studio.header.downloadHtmlTitle")}
            >
              <ArrowDownTrayIcon className="h-4 w-4" />
              <span className="hidden md:inline">{t("studio.header.downloadHtml")}</span>
            </button>

            {/* Instant client-side preview */}
            <button
              type="button"
              onClick={handleInstantPreview}
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              title={t("studio.header.instantPreviewTitle")}
            >
              <EyeIcon className="h-4 w-4" />
              <span className="hidden md:inline">{t("studio.header.instantPreview")}</span>
            </button>

            {/* Deploy Adapter Button */}
            <button
              type="button"
              onClick={() => setDeployModalOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-neutral-800 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-100"
            >
              <RocketLaunchIcon className="h-4 w-4" />
              <span className="hidden sm:inline">{t("studio.header.publish")}</span>
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
            {/* Geração por IA local (descreva o app e o modelo monta os componentes) */}
            <div className="mb-4 rounded-xl border border-blue-200 bg-blue-50 p-2.5 dark:border-blue-900/60 dark:bg-blue-950/20">
              <label className="text-xs font-semibold uppercase tracking-wider text-blue-700 dark:text-blue-300">
                {t("studio.palette.ai.title")}
              </label>
              <textarea
                value={aiPrompt}
                onChange={(e) => setAiPrompt(e.target.value)}
                placeholder={t("studio.palette.ai.placeholder")}
                rows={2}
                className="mt-1.5 w-full resize-none rounded-lg border border-neutral-300 bg-white px-2 py-1.5 text-xs dark:border-neutral-700 dark:bg-neutral-800"
              />
              <label className="mt-1.5 flex items-center gap-1.5 text-[11px] text-neutral-600 dark:text-neutral-300">
                <input
                  type="checkbox"
                  checked={aiReplace}
                  onChange={(e) => setAiReplace(e.target.checked)}
                  className="h-3 w-3"
                />
                {t("studio.palette.ai.replace")}
              </label>
              <button
                type="button"
                onClick={() => void handleGenerateAI()}
                disabled={aiGenerating || !aiPrompt.trim()}
                className="mt-1.5 w-full rounded-lg bg-blue-600 px-2 py-1.5 text-xs font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {aiGenerating ? t("studio.palette.ai.generating") : t("studio.palette.ai.generate")}
              </button>
            </div>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
              {t("studio.palette.components")}
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
                    <div className="truncate font-semibold">{t(tmpl.labelKey)}</div>
                    <div className="text-[10px] text-neutral-400">{t("studio.palette.addHint")}</div>
                  </div>
                  <PlusIcon className="h-4 w-4 shrink-0 text-neutral-400" />
                </button>
              ))}
            </div>

            <div className="mt-6 border-t border-neutral-100 pt-4 dark:border-neutral-800">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
                {t("studio.palette.templates")}
              </h3>
              <div className="mt-2 space-y-1 text-xs">
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("studio.templateName.site", "website")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <GlobeAltIcon className="h-4 w-4 text-emerald-500" />
                  {t("studio.palette.template.site")}
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("studio.templateName.dashboard", "dashboard")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <ChartBarIcon className="h-4 w-4 text-blue-500" />
                  {t("studio.palette.template.dashboard")}
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("studio.templateName.slides", "slides")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <PresentationChartBarIcon className="h-4 w-4 text-violet-500" />
                  {t("studio.palette.template.slides")}
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("studio.templateName.game", "game")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <PuzzlePieceIcon className="h-4 w-4 text-amber-500" />
                  {t("studio.palette.template.game")}
                </button>
                <button
                  type="button"
                  onClick={() => void handleCreateTemplate("studio.templateName.app", "app")}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <DevicePhoneMobileIcon className="h-4 w-4 text-rose-500" />
                  {t("studio.palette.template.app")}
                </button>
              </div>
            </div>
          </aside>

          {/* Center Workspace (Canvas or Preview) */}
          <main className="flex-1 overflow-auto p-4 lg:p-6">
            {/* Mobile Component Strip */}
            <div className="md:hidden flex items-center gap-1.5 overflow-x-auto pb-3 mb-3 border-b border-neutral-200 dark:border-neutral-800">
              <span className="text-[10px] uppercase font-bold text-neutral-400 shrink-0">{t("studio.palette.mobileAdd")}</span>
              {COMPONENT_TEMPLATES.map((tmpl) => (
                <button
                  key={tmpl.type}
                  type="button"
                  onClick={() => handleAddComponent(tmpl)}
                  className="flex items-center gap-1 shrink-0 rounded-lg border border-neutral-200 bg-white px-2 py-1 text-xs font-medium text-neutral-800 shadow-xs dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                >
                  <span className="text-neutral-400">+</span>
                  <span>{t(tmpl.labelKey)}</span>
                </button>
              ))}
            </div>
            {activeTab === "live" ? (
          <LiveBuildPanel />
        ) : activeTab === "canvas" ? (
              <div className="mx-auto max-w-4xl">
                <div className="mb-3 flex items-center justify-between">
                  <div className="text-xs font-medium text-neutral-500">
                    {t("studio.canvas.area", { count: project?.components?.length || 0 })}
                  </div>
                  <div className="text-xs text-neutral-400">
                    {t("studio.canvas.hint")}
                  </div>
                </div>

                <div className="min-h-[500px] rounded-2xl border border-dashed border-neutral-300 bg-white p-6 shadow-sm dark:border-neutral-800 dark:bg-neutral-900/60">
                  {loading ? (
                    <div className="flex flex-col items-center justify-center py-20 text-center text-neutral-400">
                      <SparklesIcon className="h-8 w-8 animate-pulse text-neutral-400" />
                      <p className="mt-2 text-xs">{t("studio.canvas.loading")}</p>
                    </div>
                  ) : (!project?.components || project.components.length === 0) ? (
                    <div className="flex flex-col items-center justify-center py-20 text-center text-neutral-400">
                      <SparklesIcon className="h-10 w-10 text-neutral-300 dark:text-neutral-600" />
                      <p className="mt-3 text-sm font-medium text-neutral-600 dark:text-neutral-300">
                        {t("studio.canvas.empty")}
                      </p>
                      <p className="mt-1 text-xs text-neutral-400">
                        {t("studio.canvas.emptyHint")}
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
                            onDoubleClick={(e) => {
                              e.stopPropagation();
                              if (PRIMARY_TEXT_PROP[comp.type]) {
                                setSelectedCompId(comp.id);
                                setEditingId(comp.id);
                              }
                            }}
                            draggable={editingId !== comp.id}
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
                            title={t("studio.canvas.dragHint")}
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
                                  title={t("studio.canvas.moveUp")}
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
                                  title={t("studio.canvas.moveDown")}
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
                                  title={t("studio.canvas.delete")}
                                >
                                  <TrashIcon className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </div>

                            {/* Inline editing: double-click a component to edit
                                its main text right on the canvas. */}
                            {editingId === comp.id && PRIMARY_TEXT_PROP[comp.type] && (
                              <input
                                autoFocus
                                data-testid="inline-edit-input"
                                defaultValue={comp.props?.[PRIMARY_TEXT_PROP[comp.type]] ?? ""}
                                onClick={(e) => e.stopPropagation()}
                                onBlur={(e) => {
                                  handleUpdateProps(comp.id, PRIMARY_TEXT_PROP[comp.type], e.target.value);
                                  setEditingId(null);
                                }}
                                onKeyDown={(e) => {
                                  if (e.key === "Enter") {
                                    handleUpdateProps(comp.id, PRIMARY_TEXT_PROP[comp.type], (e.target as HTMLInputElement).value);
                                    setEditingId(null);
                                  } else if (e.key === "Escape") {
                                    setEditingId(null);
                                  }
                                }}
                                className="mb-2 w-full rounded-lg border border-blue-400 bg-white px-2 py-1 text-sm text-neutral-900 outline-none ring-2 ring-blue-200 dark:bg-neutral-800 dark:text-white dark:ring-blue-900/40"
                              />
                            )}

                            {/* Render Component Content */}
                            {comp.type === "heading" && (
                              <h2 className="text-xl font-bold text-neutral-900 dark:text-white">
                                {comp.props?.text || t("studio.template.heading.label")}
                              </h2>
                            )}
                            {comp.type === "paragraph" && (
                              <p className="text-sm text-neutral-600 dark:text-neutral-300">
                                {comp.props?.text || t("studio.render.paragraphFallback")}
                              </p>
                            )}
                            {comp.type === "button" && (
                              <button
                                type="button"
                                className="inline-flex rounded-lg bg-neutral-900 px-4 py-2 text-xs font-semibold text-white shadow-sm dark:bg-white dark:text-neutral-900"
                              >
                                {comp.props?.label || t("studio.render.buttonFallback")}
                              </button>
                            )}
                            {comp.type === "card" && (
                              <div className="rounded-lg border border-neutral-200 bg-neutral-50/50 p-3 dark:border-neutral-800 dark:bg-neutral-800/40">
                                <h3 className="font-semibold text-neutral-900 dark:text-white">
                                  {comp.props?.title || t("studio.render.cardTitleFallback")}
                                </h3>
                                <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                                  {comp.props?.body || t("studio.render.cardBodyFallback")}
                                </p>
                              </div>
                            )}
                            {comp.type === "metric" && (
                              <div className="rounded-lg bg-neutral-100 p-3 dark:bg-neutral-800">
                                <div className="text-[11px] uppercase tracking-wider text-neutral-400">
                                  {comp.props?.label || t("studio.render.metricFallback")}
                                </div>
                                <div className="mt-1 text-2xl font-bold text-neutral-900 dark:text-white">
                                  {comp.props?.value || "0"}
                                </div>
                              </div>
                            )}
                            {comp.type === "navbar" && (
                              <div className="flex items-center justify-between border-b border-neutral-200 pb-2 dark:border-neutral-800">
                                <div className="font-bold text-neutral-900 dark:text-white">
                                  {comp.props?.brand || t("studio.render.navbarBrandFallback")}
                                </div>
                                <div className="flex gap-3 text-xs text-neutral-500">
                                  {(comp.props?.links || t("studio.render.navbarLinksFallback")).split(",").map((l, i) => (
                                    <span key={i}>{l.trim()}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {comp.type === "hero" && (
                              <div className="rounded-xl bg-gradient-to-br from-neutral-900 to-neutral-700 p-6 text-center text-white dark:from-neutral-100 dark:to-neutral-300 dark:text-neutral-900">
                                <h1 className="text-2xl font-bold">{comp.props?.title || t("studio.render.heroTitleFallback")}</h1>
                                <p className="mt-2 text-sm opacity-80">{comp.props?.subtitle || t("studio.render.heroSubtitleFallback")}</p>
                                <span className="mt-4 inline-flex rounded-lg bg-white px-4 py-2 text-xs font-semibold text-neutral-900 dark:bg-neutral-900 dark:text-white">{comp.props?.cta || t("studio.render.heroCtaFallback")}</span>
                              </div>
                            )}
                            {comp.type === "image" && (
                              comp.props?.src ? (
                                <img src={comp.props.src} alt={comp.props?.alt || ""} className="max-w-full rounded-lg" />
                              ) : (
                                <div className="flex h-24 items-center justify-center rounded-lg border border-dashed border-neutral-300 text-xs text-neutral-400 dark:border-neutral-700">{t("studio.render.imageEmpty")}</div>
                              )
                            )}
                            {comp.type === "input" && (
                              <label className="block text-xs">
                                <span className="text-neutral-700 dark:text-neutral-300">{comp.props?.label || t("studio.render.inputLabelFallback")}</span>
                                <input type="text" disabled placeholder={comp.props?.placeholder || ""} className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900" />
                              </label>
                            )}
                            {comp.type === "link" && (
                              <span className="text-sm font-medium text-blue-600 underline dark:text-blue-400">{comp.props?.text || t("studio.template.link.label")}</span>
                            )}
                            {comp.type === "list" && (
                              <ul className="list-disc space-y-1 pl-5 text-sm text-neutral-700 dark:text-neutral-300">
                                {(comp.props?.items || t("studio.render.listFallback")).split(",").map((it, i) => (
                                  <li key={i}>{it.trim()}</li>
                                ))}
                              </ul>
                            )}
                            {comp.type === "divider" && (
                              <hr className="border-neutral-200 dark:border-neutral-800" />
                            )}
                            {comp.type === "pricing" && (
                              <div className="rounded-xl border border-neutral-200 bg-neutral-50 p-4 text-center dark:border-neutral-700 dark:bg-neutral-900">
                                <div className="text-sm font-bold text-neutral-900 dark:text-white">{comp.props?.plan || t("studio.template.pricing.plan")}</div>
                                <div className="my-1 text-2xl font-extrabold text-neutral-900 dark:text-white">{comp.props?.price || t("studio.template.pricing.price")}</div>
                                <ul className="space-y-1 text-xs text-neutral-600 dark:text-neutral-400">
                                  {(comp.props?.features || t("studio.render.pricingFeaturesFallback")).split(",").map((f, i) => (
                                    <li key={i}>{f.trim()}</li>
                                  ))}
                                </ul>
                                <span className="mt-3 inline-flex rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-semibold text-white dark:bg-white dark:text-neutral-900">{comp.props?.cta || t("studio.template.pricing.cta")}</span>
                              </div>
                            )}
                            {comp.type === "testimonial" && (
                              <figure className="m-0 border-l-[3px] border-neutral-900 pl-4 dark:border-neutral-100">
                                <blockquote className="m-0 text-sm italic text-neutral-800 dark:text-neutral-100">{comp.props?.quote || t("studio.template.testimonial.quote")}</blockquote>
                                <figcaption className="mt-2 text-xs font-semibold text-neutral-500">{comp.props?.author || t("studio.template.testimonial.author")}</figcaption>
                              </figure>
                            )}
                            {comp.type === "faq" && (
                              <details className="rounded-lg border border-neutral-200 p-3 dark:border-neutral-700">
                                <summary className="cursor-pointer text-sm font-semibold text-neutral-800 dark:text-neutral-100">{comp.props?.question || t("studio.template.faq.question")}</summary>
                                <p className="mt-2 text-xs text-neutral-600 dark:text-neutral-400">{comp.props?.answer || t("studio.template.faq.answer")}</p>
                              </details>
                            )}
                            {comp.type === "footer" && (
                              <div className="flex items-center justify-between border-t border-neutral-200 pt-3 text-xs text-neutral-500 dark:border-neutral-800">
                                <span>{comp.props?.text || t("studio.template.footer.text")}</span>
                                <div className="flex gap-3">
                                  {(comp.props?.links || t("studio.render.footerLinksFallback")).split(",").map((l, i) => (
                                    <span key={i}>{l.trim()}</span>
                                  ))}
                                </div>
                              </div>
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
                  <span>{t("studio.preview.authNote")}</span>
                  <button
                    type="button"
                    disabled={!previewSrc}
                    onClick={() => setPreviewExpanded(true)}
                    className={`font-medium text-blue-600 hover:underline dark:text-blue-400 ${previewSrc ? "" : "pointer-events-none opacity-50"}`}
                  >
                    {t("studio.preview.expand")} &rarr;
                  </button>
                </div>
                <div className="flex-1 overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
                  <iframe
                    src={previewSrc ?? undefined}
                    title={t("studio.preview.frameTitle")}
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
                  {t("studio.inspector.title")}
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
                  <label className="block text-neutral-500">{t("studio.inspector.type")}</label>
                  <input
                    type="text"
                    disabled
                    value={selectedComponent.type}
                    className="mt-1 w-full rounded-lg border border-neutral-200 bg-neutral-100 px-2.5 py-1.5 font-mono text-neutral-600 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-400"
                  />
                </div>

                {selectedComponent.props?.text !== undefined && (
                  <div>
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.text")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.titleField")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.body")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.label")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.value")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.brand")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.links")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.change")}</label>
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
                    <label className="block text-neutral-700 dark:text-neutral-300">{t("studio.inspector.action")}</label>
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
                    {t("studio.canvas.delete")}
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
                {t("studio.deploy.title")}
              </h3>
              <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                {t("studio.deploy.description")}
              </p>

              <div className="mt-4 space-y-2">
                {DEPLOY_PROVIDERS.map((prov) => (
                  <button
                    key={prov.slug}
                    type="button"
                    onClick={() => void handleDeployAttempt(prov)}
                    className="flex w-full items-center justify-between rounded-xl border border-neutral-200 p-3 text-left text-xs font-medium text-neutral-800 transition hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-800"
                  >
                    <span>{t(prov.labelKey)}</span>
                    <span className="rounded bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
                      {t("studio.deploy.requiresCredentials")}
                    </span>
                  </button>
                ))}
              </div>

              {/* RELATÓRIO DE PUBLICAÇÃO (E-282): o que a publicação local ENTREGOU — nunca um selo verde quando a
          aplicação não é executável. */}
      <PublishReportNotice report={project?.publish_report} />
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
                  {t("studio.deploy.close")}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Prévia instantânea em modal (iframe srcDoc) — funciona no app e no navegador */}
        {instantPreviewHtml !== null && (
          <div
            role="dialog"
            aria-modal="true"
            aria-label={t("studio.preview.dialogLabel")}
            className="fixed inset-0 z-50 flex flex-col bg-black/70 p-4"
            onClick={() => setInstantPreviewHtml(null)}
          >
            <div
              className="mx-auto flex h-full w-full max-w-5xl flex-col overflow-hidden rounded-2xl bg-white shadow-xl dark:bg-neutral-900"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-2 dark:border-neutral-800">
                <span className="text-sm font-semibold text-neutral-900 dark:text-white">{t("studio.preview.dialogTitle", { name: project?.name || "app" })}</span>
                <button
                  type="button"
                  onClick={() => setInstantPreviewHtml(null)}
                  aria-label={t("studio.preview.closeLabel")}
                  className="rounded-lg px-3 py-1 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  {t("studio.preview.close")}
                </button>
              </div>
              <iframe
                title={t("studio.preview.dialogLabel")}
                srcDoc={instantPreviewHtml}
                sandbox=""
                className="h-full w-full flex-1 border-0 bg-white"
              />
            </div>
          </div>
        )}
        {previewExpanded && previewSrc && (
          <div
            role="dialog"
            aria-modal="true"
            aria-label={t("studio.preview.expandedDialogLabel")}
            className="fixed inset-0 z-50 flex flex-col bg-black/70 p-4"
            onClick={() => setPreviewExpanded(false)}
          >
            <div
              className="mx-auto flex h-full w-full max-w-5xl flex-col overflow-hidden rounded-2xl bg-white shadow-xl dark:bg-neutral-900"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-2 dark:border-neutral-800">
                <span className="text-sm font-semibold text-neutral-900 dark:text-white">{t("studio.preview.expandedDialogTitle", { name: project?.name || "app" })}</span>
                <button
                  type="button"
                  onClick={() => setPreviewExpanded(false)}
                  aria-label={t("studio.preview.closeLabel")}
                  className="rounded-lg px-3 py-1 text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  {t("studio.preview.close")}
                </button>
              </div>
              <iframe
                title={t("studio.preview.expandedFrameTitle")}
                src={previewSrc}
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
