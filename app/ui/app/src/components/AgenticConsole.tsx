import { useEffect, useMemo, useRef, useState } from "react";
import { agentFetch, listProjects, type AgentProject } from "@/lib/agenticClient";
import { AgenticSplitShell } from "@/components/AgenticSplitShell";
import { getModels } from "@/api";
import type { Model } from "@/gotypes";
import { humanizeApiError } from "@/lib/userFacingError";
import { MissionDeliveryCard } from "@/components/MissionDeliveryCard";
import { missionStateLabel, eventTypeLabel } from "@/lib/labels";

type Mission = {
  id: string;
  objective: string;
  state: string;
  plan?: Array<{ id: string; title: string; kind: string; state: string; requires_approval: boolean }>;
  approvals?: Array<{ id: string; step_id: string; status: string; policy?: string; nonce?: string; reason?: string }>;
  artifacts?: Array<{ id: string; name: string; sha256: string; size: number }>;
  last_error?: string;
  model?: string;
  project_id?: string;
  workspace_isolated?: boolean;
  workspace_snapshot_id?: string;
};
type Event = { id: string; type: string; step_id?: string; created_at: string; payload?: unknown };
type Metrics = Record<string, number>;
type OrchestrationTask = { id: string; role: string; state: string; output?: string; error?: string; evidence?: Array<{ url: string; title?: string; excerpt?: string }> };
type OrchestrationJob = { id: string; objective: string; state: string; summary?: string; conflicts?: string[]; tasks: OrchestrationTask[] };
type ResearchReport = { query: string; summary: string; citations: Array<{ url: string; title?: string; excerpt?: string }>; sources: Array<{ url: string; title?: string; error?: string }> };

function approvalExplanation(approval: { step_id: string; policy?: string }): string {
  const value = `${approval.step_id} ${approval.policy ?? ""}`.toLowerCase();
  if (/send|message|whatsapp|email|publish|social/.test(value)) return "Pode enviar ou publicar conteúdo fora do Hades. Revise destinatário, texto e escopo antes de decidir.";
  if (/spend|budget|payment|purchase|ads|contract/.test(value)) return "Pode gerar gasto, compromisso financeiro ou anúncio. Confirme valor, limite e destinatário antes de decidir.";
  if (/write|delete|merge|deploy|external/.test(value)) return "Pode alterar arquivos, publicar ou modificar um serviço externo. Verifique o resumo e o projeto afetado.";
  return "Esta ação foi bloqueada preventivamente pelo servidor até uma decisão explícita. Revise o contexto antes de decidir.";
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  return agentFetch<T>(path, init);
}

export default function AgenticConsole() {
  const [objective, setObjective] = useState("");
  const [mission, setMission] = useState<Mission | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [metrics, setMetrics] = useState<Metrics>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [provider, setProvider] = useState("ollama-local");
  const [selectedModel, setSelectedModel] = useState("");
  const [availableModels, setAvailableModels] = useState<Model[]>([]);
  const [allowWorkspaceWrite, setAllowWorkspaceWrite] = useState(false);
  const [projectID, setProjectID] = useState("");
  const [isolateWorkspace, setIsolateWorkspace] = useState(false);
  const [projects, setProjects] = useState<AgentProject[]>([]);
  const [orchestrationObjective, setOrchestrationObjective] = useState("");
  const [orchestration, setOrchestration] = useState<OrchestrationJob | null>(null);
  const [researchQuery, setResearchQuery] = useState("");
  const [researchURLs, setResearchURLs] = useState("");
  const [research, setResearch] = useState<ResearchReport | null>(null);
  const [approvalReasons, setApprovalReasons] = useState<Record<string, string>>({});
  const [viewMode, setViewMode] = useState<"split" | "classic">("classic");
  const errorRef = useRef<HTMLParagraphElement>(null);

  const providerChoices = useMemo(() => {
    const choices = new Map<string, { label: string; models: string[]; available: boolean }>();
    choices.set("ollama-local", { label: "Ollama local", models: [], available: true });
    for (const model of availableModels) {
      const identifier = model.model?.trim();
      if (!identifier || identifier.startsWith("auto/") || identifier === "local/private") continue;
      const providerID = model.provider?.trim() || (identifier.includes("/") ? identifier.split("/", 1)[0] : "ollama-local");
      const choice = choices.get(providerID) ?? { label: providerID, models: [], available: false };
      if (!choice.models.includes(identifier)) choice.models.push(identifier);
      // A provider is usable once any of its models has a configured credential.
      if (model.available !== false) choice.available = true;
      choices.set(providerID, choice);
    }
    return [...choices.entries()].map(([id, choice]) => ({ id, ...choice }));
  }, [availableModels]);

  const selectedProviderChoice = providerChoices.find((choice) => choice.id === provider) ?? providerChoices[0];

  useEffect(() => {
    const current = providerChoices.find((choice) => choice.id === provider);
    if (!current) {
      setProvider("ollama-local");
      setSelectedModel("");
    } else if (current.models.length > 0 && !current.models.includes(selectedModel)) {
      setSelectedModel(current.models[0]);
    } else if (current.models.length === 0 && selectedModel) {
      setSelectedModel("");
    }
  }, [providerChoices, provider, selectedModel]);

  const load = async (missionId?: string, orchestrationId?: string) => {
    try {
      const [nextMetrics, nextMission, nextEvents, nextOrchestration] = await Promise.all([
        api<Metrics>("/api/agent/v1/metrics"),
        missionId ? api<Mission>(`/api/agent/v1/missions/${encodeURIComponent(missionId)}`) : Promise.resolve(null),
        missionId ? api<{ events: Event[] }>(`/api/agent/v1/missions/${encodeURIComponent(missionId)}/events`) : Promise.resolve({ events: [] }),
        orchestrationId ? api<OrchestrationJob>(`/api/agent/v1/orchestration/jobs/${encodeURIComponent(orchestrationId)}`) : Promise.resolve(null),
      ]);
      setMetrics(nextMetrics);
      if (nextMission) setMission(nextMission);
      setEvents(nextEvents.events ?? []);
      if (nextOrchestration) setOrchestration(nextOrchestration);
    } catch (cause) {
      setError(humanizeApiError(cause, "Não foi possível carregar o runtime agentic.").message);
    }
  };

  useEffect(() => {
    void load(mission?.id, orchestration?.id);
    const timer = window.setInterval(() => void load(mission?.id, orchestration?.id), 3000);
    return () => window.clearInterval(timer);
  }, [mission?.id, orchestration?.id]);

  useEffect(() => {
    void listProjects().then((result) => setProjects(result.projects ?? [])).catch(() => setProjects([]));
    void getModels("").then(setAvailableModels).catch(() => setAvailableModels([]));
    const searchParams = new URLSearchParams(window.location.search);
    const requestedObjective = searchParams.get("objective");
    const autoRunParam = searchParams.get("autorun") === "true";
    const requestedAllowWrite = searchParams.get("allow_write") === "true";
    const requestedProjectID = searchParams.get("project_id") || "";
    const slashCommand = searchParams.get("slash");
    const requestedProvider = searchParams.get("provider") || "ollama-local";
    const requestedModel = searchParams.get("model") || "auto/coding";
    const requestedMissionId = searchParams.get("mission_id") || searchParams.get("missionId");
    if (requestedMissionId) {
      setViewMode("split");
      void load(requestedMissionId);
    } else if (!requestedObjective) {
      void api<{ missions: Mission[] }>("/api/agent/v1/missions")
        .then((res) => {
          if (res.missions && res.missions.length > 0) {
            const latest = res.missions[res.missions.length - 1];
            setMission(latest);
            setViewMode("split");
            void load(latest.id);
          }
        })
        .catch(() => {});
    }
    if (requestedObjective) {
      setObjective(requestedObjective);
      setProjectID(requestedProjectID);
      setAllowWorkspaceWrite(requestedAllowWrite);
      setProvider(requestedProvider);
      setSelectedModel(requestedModel.startsWith("auto/") ? "" : requestedModel);
      if (!requestedAllowWrite && (autoRunParam || slashCommand === "plan")) {
        setViewMode("split");
        void (async () => {
          setBusy(true);
          try {
            const created = await api<Mission>("/api/agent/v1/missions", {
              method: "POST",
              body: JSON.stringify({
                objective: requestedObjective,
                provider: requestedProvider,
                model: requestedModel,
                project_id: requestedProjectID || undefined,
                capabilities: ["workspace:read"],
                auto_run: autoRunParam && !requestedAllowWrite,
              }),
            });
            setMission(created);
            if (slashCommand === "goal") {
              const delegated = await api<OrchestrationJob>("/api/agent/v1/orchestration/jobs", {
                method: "POST",
                body: JSON.stringify({
                  objective: requestedObjective,
                  roles: ["research", "programming", "testing", "security", "review"],
                  budget: { max_agents: 3, max_seconds: 600, max_retries: 1 },
                  auto_run: true,
                }),
              });
              setOrchestration(delegated);
            }
            await load(created.id);
          } catch (cause) {
            setError(humanizeApiError(cause, "Falha ao iniciar a missão automática.").message);
          } finally {
            setBusy(false);
          }
        })();
      }
    }
  }, []);

  useEffect(() => {
    if (error) errorRef.current?.focus();
  }, [error]);

  const pendingApprovals = useMemo(() => mission?.approvals?.filter((approval) => approval.status === "PENDING") ?? [], [mission]);

  const createMission = async (objArg?: string) => {
    // Accept the objective as an explicit argument so the footer composer does
    // not depend on the asynchronous setObjective state update (which would be
    // read stale by closure and fall into the early-return).
    const obj = (objArg ?? objective).trim();
    if (!obj) return;
    setBusy(true); setError("");
    try {
      const created = await api<Mission>("/api/agent/v1/missions", { method: "POST", body: JSON.stringify({ objective: obj, provider, model: selectedModel || undefined, project_id: projectID || undefined, isolate_workspace: isolateWorkspace && projectID ? true : undefined, capabilities: allowWorkspaceWrite ? ["workspace:read", "workspace:write"] : ["workspace:read"], auto_run: false }) });
      setMission(created); setObjective(""); await load(created.id, orchestration?.id);
    } catch (cause) { setError(humanizeApiError(cause, "Falha ao criar a missão.").message); } finally { setBusy(false); }
  };

  const decide = async (approval: { id: string; nonce?: string }, approved: boolean) => {
    if (!mission) return;
    const reason = approvalReasons[approval.id]?.trim() ?? "";
    if (!reason) {
      setError("Informe o motivo antes de decidir a aprovação.");
      return;
    }
    setBusy(true);
    try {
      await api(`/api/agent/v1/missions/${encodeURIComponent(mission.id)}/approvals/${encodeURIComponent(approval.id)}`, { method: "POST", body: JSON.stringify({ decision: approved ? "approve" : "reject", nonce: approval.nonce, reason }) });
      setApprovalReasons((current) => {
        const next = { ...current };
        delete next[approval.id];
        return next;
      });
      await load(mission.id, orchestration?.id);
    } catch (cause) { setError(humanizeApiError(cause, "Falha ao registrar a decisão.").message); } finally { setBusy(false); }
  };

  const run = async () => {
    if (!mission) return;
    setBusy(true);
    try { await api(`/api/agent/v1/missions/${encodeURIComponent(mission.id)}/run`, { method: "POST", body: "{}" }); await load(mission.id, orchestration?.id); }
    catch (cause) { setError(humanizeApiError(cause, "Falha ao iniciar a missão.").message); } finally { setBusy(false); }
  };

  const createOrchestration = async () => {
    if (!orchestrationObjective.trim()) return;
    setBusy(true); setError("");
    try {
      const created = await api<OrchestrationJob>("/api/agent/v1/orchestration/jobs", { method: "POST", body: JSON.stringify({ objective: orchestrationObjective, roles: ["research", "programming", "testing", "security", "review"], budget: { max_agents: 3, max_seconds: 600, max_retries: 1 }, auto_run: true }) });
      setOrchestration(created); setOrchestrationObjective(""); await load(mission?.id, created.id);
    } catch (cause) { setError(humanizeApiError(cause, "Falha ao iniciar a orquestração.").message); } finally { setBusy(false); }
  };

  const runResearch = async () => {
    const urls = researchURLs.split(/\r?\n/).map((url) => url.trim()).filter(Boolean);
    if (!researchQuery.trim() || urls.length === 0) return;
    setBusy(true); setError("");
    try { setResearch(await api<ResearchReport>("/api/agent/v1/research", { method: "POST", body: JSON.stringify({ query: researchQuery, urls, max_sources: 8, respect_robots: true }) })); }
    catch (cause) { setError(humanizeApiError(cause, "Falha na pesquisa profunda.").message); } finally { setBusy(false); }
  };

  return (
    <section className="flex h-full min-h-0 flex-col gap-5 overflow-y-auto p-6" aria-busy={busy} aria-label="Console agentic">
      <header><p className="text-xs font-medium uppercase tracking-[0.18em] text-neutral-500">Runtime de Agentes DZ23</p><h1 className="mt-1 text-2xl font-semibold text-neutral-900 dark:text-neutral-100">Console de Missões</h1><p className="mt-2 max-w-3xl text-sm text-neutral-600 dark:text-neutral-400">Planeje, orquestre, pesquise, aprove, execute e observe operações com a mesma trilha persistente usada pela API local.</p></header>

      <section className="rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950"><label className="text-sm font-medium text-neutral-800 dark:text-neutral-200" htmlFor="agent-objective">Nova tarefa</label><div className="mt-3 flex flex-col gap-3"><textarea id="agent-objective" value={objective} onChange={(event) => setObjective(event.target.value)} placeholder="Descreva o que você quer construir, pesquisar, revisar ou automatizar" className="min-h-20 w-full resize-y rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700" /><div className="flex flex-col gap-2 sm:flex-row"><label className="flex flex-1 flex-col gap-1 text-[11px] text-neutral-500">Motor<select aria-label="Motor" value={provider} onChange={(event) => { const nextProvider = event.target.value; setProvider(nextProvider); setSelectedModel(providerChoices.find((choice) => choice.id === nextProvider)?.models[0] ?? ""); }} className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm text-neutral-800 outline-none dark:border-neutral-700 dark:text-neutral-200">{providerChoices.map((choice) => <option key={choice.id} value={choice.id}>{choice.label}{choice.id === "ollama-local" ? " (local)" : choice.available ? " (configurado)" : " (sem chave)"}</option>)}</select></label>{selectedProviderChoice && selectedProviderChoice.models.length > 0 && <label className="flex flex-1 flex-col gap-1 text-[11px] text-neutral-500">Modelo<select aria-label="Modelo" value={selectedModel} onChange={(event) => setSelectedModel(event.target.value)} className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm text-neutral-800 outline-none dark:border-neutral-700 dark:text-neutral-200">{selectedProviderChoice.models.map((model) => <option key={model} value={model}>{model}</option>)}</select></label>}<label className="flex flex-1 flex-col gap-1 text-[11px] text-neutral-500">Projeto<select aria-label="Projeto" value={projectID} onChange={(event) => { setProjectID(event.target.value); setIsolateWorkspace(false); }} className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm text-neutral-800 outline-none dark:border-neutral-700 dark:text-neutral-200"><option value="">Sem projeto</option>{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select></label><label className="flex items-center gap-2 self-end rounded-xl border border-neutral-300 px-3 py-2 text-xs text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"><input id="isolate-workspace" type="checkbox" aria-describedby="isolate-workspace-help" checked={isolateWorkspace} disabled={!projectID || busy} onChange={(event) => setIsolateWorkspace(event.target.checked)} />Executar em cópia isolada</label><label className="flex items-center gap-2 self-end rounded-xl border border-neutral-300 px-3 py-2 text-xs text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"><input type="checkbox" checked={allowWorkspaceWrite} onChange={(event) => setAllowWorkspaceWrite(event.target.checked)} />Permitir escrita</label><button type="button" disabled={busy || !objective.trim()} onClick={() => void createMission()} className="h-10 self-end rounded-xl bg-neutral-900 px-4 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900">Criar missão</button></div><p className="text-[11px] text-neutral-500">Motor selecionado: <span className="font-mono">{provider}</span>{selectedModel ? ` · ${selectedModel}` : ""}. Providers remotos só aparecem quando o catálogo os publica; a missão falha fechado se não houver adapter/credencial. A missão começa somente com leitura; habilite “Permitir escrita” quando a tarefa precisar alterar arquivos. “Executar em cópia isolada” cria um snapshot Git por missão e não altera a origem; exige projeto associado à organização e exclui caminhos/arquivos sensíveis. A opção só fica disponível com projeto selecionado. O servidor valida grants e exige approval para efeitos de escrita.</p><p id="isolate-workspace-help" className="text-[11px] text-neutral-500">A cópia isolada é opt-in; selecione um projeto. Segredos, arquivos ignorados e symlinks não são copiados.</p></div>{error && <p ref={errorRef} id="agentic-console-error" role="alert" aria-live="assertive" aria-atomic="true" tabIndex={-1} className="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 outline-none focus:ring-2 focus:ring-red-500 dark:bg-red-950/30 dark:text-red-300">{error}</p>}</section>

      <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">{[["Criadas", metrics.missions_created], ["Concluídas", metrics.missions_completed], ["Novas tentativas", metrics.retries], ["Ferramentas", metrics.tool_calls]].map(([label, value]) => <div key={label} className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"><p className="text-xs text-neutral-500">{label}</p><p className="mt-1 text-2xl font-semibold text-neutral-900 dark:text-neutral-100">{value ?? 0}</p></div>)}</section>

      <section className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"><div className="flex items-start justify-between gap-3"><div><p className="text-xs font-medium uppercase tracking-[0.14em] text-neutral-500">Multiagente</p><h2 className="mt-1 font-medium text-neutral-900 dark:text-neutral-100">Orquestrar especialistas</h2></div>{orchestration && <span className="rounded-full bg-neutral-100 px-3 py-1 text-xs font-medium dark:bg-neutral-800">{missionStateLabel(orchestration.state)}</span>}</div><textarea value={orchestrationObjective} onChange={(event) => setOrchestrationObjective(event.target.value)} placeholder="Ex.: pesquisar concorrentes, revisar segurança e propor implementação" className="mt-3 min-h-20 w-full resize-y rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-sm outline-none dark:border-neutral-700" /><button type="button" disabled={busy || !orchestrationObjective.trim()} onClick={() => void createOrchestration()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-sm text-white disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900">Iniciar orquestração</button>{orchestration && <div className="mt-4 space-y-2">{orchestration.tasks.map((task) => <div key={task.id} className="rounded-lg border border-neutral-200 p-3 text-sm dark:border-neutral-800"><div className="flex justify-between"><span className="font-medium">{task.role}</span><span className="text-xs text-neutral-500">{missionStateLabel(task.state)}</span></div>{task.error && <p className="mt-1 text-xs text-red-600">{task.error}</p>}</div>)}{orchestration.summary && <details className="rounded-lg bg-neutral-50 p-3 text-xs dark:bg-neutral-900"><summary className="cursor-pointer font-medium">Ver síntese</summary><pre className="mt-2 whitespace-pre-wrap font-sans">{orchestration.summary}</pre></details>}</div>}</div>
        <div className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"><p className="text-xs font-medium uppercase tracking-[0.14em] text-neutral-500">Pesquisa profunda</p><h2 className="mt-1 font-medium text-neutral-900 dark:text-neutral-100">Fontes e citações</h2><input value={researchQuery} onChange={(event) => setResearchQuery(event.target.value)} placeholder="Pergunta de pesquisa" className="mt-3 h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none dark:border-neutral-700" /><textarea value={researchURLs} onChange={(event) => setResearchURLs(event.target.value)} placeholder="Uma URL HTTPS pública por linha" className="mt-2 min-h-20 w-full resize-y rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-sm outline-none dark:border-neutral-700" /><button type="button" disabled={busy || !researchQuery.trim() || !researchURLs.trim()} onClick={() => void runResearch()} className="mt-3 rounded-lg border border-neutral-300 px-3 py-2 text-sm dark:border-neutral-700">Pesquisar</button>{research && <div className="mt-4 space-y-2 text-sm"><p className="whitespace-pre-wrap text-neutral-700 dark:text-neutral-300">{research.summary}</p>{research.citations.map((citation) => <a key={citation.url} href={citation.url} target="_blank" rel="noreferrer" className="block rounded-lg border border-neutral-200 p-2 text-xs underline dark:border-neutral-800">{citation.title || citation.url}</a>)}</div>}</div>
      </section>

      {mission && (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between border-b border-neutral-200 pb-2 dark:border-neutral-800">
            <div className="flex items-center gap-1 rounded-xl bg-neutral-100 p-1 dark:bg-neutral-800">
              <button
                type="button"
                onClick={() => setViewMode("split")}
                className={`rounded-lg px-3 py-1 text-xs font-semibold transition-colors ${
                  viewMode === "split"
                    ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-700 hover:text-neutral-900 dark:text-neutral-300"
                }`}
              >
                Visão dividida
              </button>
              <button
                type="button"
                onClick={() => setViewMode("classic")}
                className={`rounded-lg px-3 py-1 text-xs font-semibold transition-colors ${
                  viewMode === "classic"
                    ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-neutral-700 hover:text-neutral-900 dark:text-neutral-300"
                }`}
              >
                Visão Clássica
              </button>
            </div>
          </div>

          {viewMode === "split" ? (
            <div className="h-[650px] min-h-[500px] overflow-hidden rounded-2xl border border-neutral-200 shadow-sm dark:border-neutral-800">
              <AgenticSplitShell
                mission={mission}
                onRefreshMission={() => load(mission.id, orchestration?.id)}
                onCreateMission={async (obj) => {
                  await createMission(obj);
                }}
                busy={busy}
              />
            </div>
          ) : (
            <section className="grid min-h-0 gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <div className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-xs text-neutral-500">{mission.id}</p>
                    <h2 className="mt-1 font-medium text-neutral-900 dark:text-neutral-100">{mission.objective}</h2>
                  </div>
                  <span className="rounded-full bg-neutral-100 px-3 py-1 text-xs font-medium dark:bg-neutral-800">{missionStateLabel(mission.state)}</span>
                </div>
                {(mission.plan?.length ?? 0) > 0 && (() => {
                  const steps = mission.plan ?? [];
                  const done = steps.filter((s) => ["DONE", "COMPLETED"].includes(s.state.toUpperCase())).length;
                  return (
                    <div className="mt-4 rounded-xl border border-neutral-200 bg-neutral-50 p-3 dark:border-neutral-800 dark:bg-neutral-900/40">
                      <div className="flex items-center justify-between text-xs">
                        <span className="font-medium text-neutral-700 dark:text-neutral-200">Plano</span>
                        <span className="text-neutral-500">Etapa {Math.min(done + 1, steps.length)} de {steps.length}</span>
                      </div>
                      <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
                        <div className="h-full rounded-full bg-violet-500 transition-all" style={{ width: `${Math.round((done / steps.length) * 100)}%` }} />
                      </div>
                      <ol className="mt-3 space-y-1.5">
                        {steps.map((step, index) => {
                          const s = step.state.toUpperCase();
                          const kind = s === "DONE" || s === "COMPLETED" ? "done" : s === "RUNNING" || s === "OBSERVING" ? "run" : "todo";
                          return (
                            <li key={step.id} className="flex items-center gap-2 text-xs">
                              <span className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[9px] font-semibold ${kind === "done" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300" : kind === "run" ? "bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-300" : "bg-neutral-200 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400"}`}>{kind === "done" ? "✓" : index + 1}</span>
                              <span className={`min-w-0 flex-1 truncate ${kind === "todo" ? "text-neutral-500" : "text-neutral-800 dark:text-neutral-200"}`}>{step.title}</span>
                              {step.requires_approval && <span className="shrink-0 rounded-full bg-amber-50 px-1.5 py-0.5 text-[9px] text-amber-700 dark:bg-amber-950/30 dark:text-amber-300">aprovação</span>}
                            </li>
                          );
                        })}
                      </ol>
                    </div>
                  );
                })()}
                <div className="mt-5 space-y-3">
                  {events.map((event) => (
                    <div key={event.id} className="flex gap-3 text-sm">
                      <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-neutral-500" />
                      <div>
                        <p className="font-medium text-neutral-800 dark:text-neutral-200">{eventTypeLabel(event.type)}</p>
                        <p className="text-xs text-neutral-500">{event.step_id ?? "mission"} · {new Date(event.created_at).toLocaleString()}</p>
                      </div>
                    </div>
                  ))}
                </div>
                <MissionDeliveryCard missionId={mission.id} state={mission.state} artifacts={mission.artifacts} />
                <div className="mt-5 flex gap-2">
                  <button type="button" disabled={busy || pendingApprovals.length > 0 || mission.state === "COMPLETED"} onClick={() => void run()} className="rounded-lg bg-neutral-900 px-3 py-2 text-sm text-white disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900">Executar</button>
                </div>
              </div>
              <div className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950">
                <h2 className="font-medium text-neutral-900 dark:text-neutral-100">Aprovações</h2>
                {pendingApprovals.length === 0 ? (
                  <p className="mt-3 text-sm text-neutral-500">Nenhuma aprovação pendente.</p>
                ) : (
                  pendingApprovals.map((approval) => {
                    const reason = approvalReasons[approval.id] ?? "";
                    const canDecide = reason.trim().length > 0;
                    return (
                      <div key={approval.id} className="mt-3 rounded-xl border border-amber-200 bg-amber-50 p-3 dark:border-amber-900 dark:bg-amber-950/30">
                        <p className="text-xs text-amber-800 dark:text-amber-200">{approval.step_id}</p>
                        <p className="mt-2 text-xs leading-5 text-amber-900 dark:text-amber-100" role="note">{approvalExplanation(approval)}</p>
                        <p className="mt-2 text-[11px] text-amber-800/80 dark:text-amber-200/80">Nada será executado enquanto a decisão não for validada pelo servidor. A aprovação não pode ser feita automaticamente pela missão.</p>
                        <label className="mt-3 block text-xs font-medium text-amber-900 dark:text-amber-100" htmlFor={`approval-reason-${approval.id}`}>Motivo da decisão</label>
                        <textarea id={`approval-reason-${approval.id}`} aria-label={`Motivo da decisão para ${approval.step_id}`} maxLength={512} value={reason} onChange={(event) => setApprovalReasons((current) => ({ ...current, [approval.id]: event.target.value }))} placeholder="Explique por que esta ação deve ser aprovada ou rejeitada" className="mt-1 min-h-16 w-full resize-y rounded-lg border border-amber-300 bg-white/70 px-2 py-2 text-xs text-neutral-900 outline-none focus:border-amber-500 dark:border-amber-800 dark:bg-neutral-950/50 dark:text-neutral-100" />
                        <div className="mt-3 flex gap-2">
                          <button type="button" disabled={busy || !canDecide} onClick={() => void decide(approval, true)} className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40">Aprovar</button>
                          <button type="button" disabled={busy || !canDecide} onClick={() => void decide(approval, false)} className="rounded-lg bg-red-600 px-3 py-2 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40">Rejeitar</button>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </section>
          )}
        </div>
      )}
    </section>
  );
}
