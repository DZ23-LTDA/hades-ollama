import { useState, useMemo, useRef, useEffect } from "react";
import {
  ArrowPathIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationCircleIcon,
  GlobeAltIcon,
  PaperAirplaneIcon,
  PlayIcon,
  StopIcon,
  CommandLineIcon,
  DocumentDuplicateIcon,
  DocumentTextIcon,
  WindowIcon,
} from "@heroicons/react/24/outline";
import { useMissionEvents } from "@/hooks/useMissionEvents";
import { ArtifactsViewer } from "@/components/ArtifactsViewer";
import { agentFetch } from "@/lib/agenticClient";

export interface MissionStep {
  id: string;
  title: string;
  kind: string;
  state: string;
  requires_approval: boolean;
  result?: unknown;
  error?: string;
}

export interface MissionApproval {
  id: string;
  step_id: string;
  status: string;
  nonce?: string;
  policy?: string;
  reason?: string;
}

export interface MissionData {
  id: string;
  objective: string;
  state: string;
  plan?: MissionStep[];
  approvals?: MissionApproval[];
  artifacts?: Array<{ id: string; name: string; sha256: string; size: number }>;
  last_error?: string;
}

interface AgenticSplitShellProps {
  mission: MissionData | null;
  onRefreshMission?: () => Promise<void>;
  onCreateMission?: (objective: string) => Promise<void>;
  busy?: boolean;
}

export function AgenticSplitShell({
  mission,
  onRefreshMission,
  onCreateMission,
  busy = false,
}: AgenticSplitShellProps) {
  const [activeCanvasTab, setActiveCanvasTab] = useState<"browser" | "artifacts" | "code">("browser");
  const [approvalReasons, setApprovalReasons] = useState<Record<string, string>>({});
  const [chatInput, setChatInput] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const { events, browserFrame, isLive } = useMissionEvents(mission?.id);
  const timelineEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    timelineEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events, mission?.plan]);

  // Se houver novo frame do navegador, focar automaticamente na aba do navegador
  useEffect(() => {
    if (browserFrame?.screenshot) {
      setActiveCanvasTab("browser");
    }
  }, [browserFrame]);

  // Se houver artefatos novos gerados, podemos alternar se não estiver no browser ativo
  useEffect(() => {
    if ((mission?.artifacts?.length ?? 0) > 0 && !browserFrame?.screenshot) {
      setActiveCanvasTab("artifacts");
    }
  }, [mission?.artifacts?.length, browserFrame?.screenshot]);

  const pendingApprovals = useMemo(
    () => mission?.approvals?.filter((a) => a.status === "PENDING") ?? [],
    [mission?.approvals]
  );

  const handleRun = async () => {
    if (!mission) return;
    setActionLoading(true);
    setActionError(null);
    try {
      await agentFetch(`/api/agent/v1/missions/${encodeURIComponent(mission.id)}/run`, {
        method: "POST",
        body: "{}",
      });
      await onRefreshMission?.();
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Falha ao iniciar missão");
    } finally {
      setActionLoading(false);
    }
  };

  const handleCancel = async () => {
    if (!mission) return;
    setActionLoading(true);
    try {
      await agentFetch(`/api/agent/v1/missions/${encodeURIComponent(mission.id)}/cancel`, {
        method: "POST",
        body: "{}",
      });
      await onRefreshMission?.();
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Falha ao cancelar missão");
    } finally {
      setActionLoading(false);
    }
  };

  const handleDecideApproval = async (approval: MissionApproval, approved: boolean) => {
    if (!mission) return;
    const reason = approvalReasons[approval.id]?.trim();
    if (!reason) {
      setActionError("Informe uma justificativa para a decisão.");
      return;
    }
    setActionLoading(true);
    setActionError(null);
    try {
      await agentFetch(
        `/api/agent/v1/missions/${encodeURIComponent(mission.id)}/approvals/${encodeURIComponent(approval.id)}`,
        {
          method: "POST",
          body: JSON.stringify({ decision: approved ? "approve" : "reject", nonce: approval.nonce, reason }),
        }
      );
      setApprovalReasons((prev) => {
        const next = { ...prev };
        delete next[approval.id];
        return next;
      });
      await onRefreshMission?.();
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Falha ao registrar aprovação");
    } finally {
      setActionLoading(false);
    }
  };

  const handleSendPrompt = async () => {
    const text = chatInput.trim();
    if (!text) return;
    setChatInput("");
    if (onCreateMission) {
      await onCreateMission(text);
    }
  };

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-neutral-100 dark:bg-neutral-950 lg:flex-row">
      {/* PAINEL ESQUERDO: Conversa, Raciocínio (Thought Stream) e Aprovações */}
      <section className="flex flex-1 flex-col border-b border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900 lg:w-1/2 lg:border-b-0 lg:border-r">
        {/* Cabeçalho da Missão */}
        <header className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-neutral-800">
          <div className="min-w-0 flex-1 pr-3">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs text-neutral-400 dark:text-neutral-500">
                {mission?.id || "Nenhuma missão ativa"}
              </span>
              {isLive && (
                <span className="flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  LIVE SSE
                </span>
              )}
            </div>
            <h2 className="mt-0.5 truncate text-sm font-semibold text-neutral-900 dark:text-neutral-100">
              {mission?.objective || "Pronto para planejar e executar"}
            </h2>
          </div>

          <div className="flex items-center gap-2">
            {mission?.state && (
              <span
                className={`rounded-full px-2.5 py-1 text-xs font-semibold uppercase tracking-wider ${
                  mission.state === "COMPLETED"
                    ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300"
                    : mission.state === "RUNNING"
                    ? "bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-300 animate-pulse"
                    : mission.state === "AWAITING_APPROVAL"
                    ? "bg-amber-100 text-amber-800 dark:bg-amber-950/50 dark:text-amber-300"
                    : mission.state === "FAILED"
                    ? "bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-300"
                    : "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300"
                }`}
              >
                {mission.state}
              </span>
            )}

            {mission && mission.state !== "COMPLETED" && mission.state !== "RUNNING" && (
              <button
                type="button"
                disabled={actionLoading || busy || pendingApprovals.length > 0}
                onClick={handleRun}
                className="flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900"
              >
                <PlayIcon className="h-3.5 w-3.5" />
                Executar
              </button>
            )}

            {mission && mission.state === "RUNNING" && (
              <button
                type="button"
                disabled={actionLoading}
                onClick={handleCancel}
                className="flex items-center gap-1.5 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700"
              >
                <StopIcon className="h-3.5 w-3.5" />
                Interromper
              </button>
            )}
          </div>
        </header>

        {actionError && (
          <div className="bg-red-50 px-4 py-2 text-xs text-red-700 dark:bg-red-950/30 dark:text-red-300 border-b border-red-200 dark:border-red-900">
            {actionError}
          </div>
        )}

        {/* Thought Stream & Timeline dos Passos */}
        <div className="min-h-0 flex-1 overflow-y-auto p-5 space-y-4">
          {!mission && (
            <div className="flex h-full flex-col items-center justify-center text-center text-neutral-400">
              <CommandLineIcon className="h-10 w-10 stroke-[1.2]" />
              <p className="mt-2 text-sm font-medium">Inicie uma tarefa abaixo para começar a execução.</p>
            </div>
          )}

          {/* Plano de Passos Interativo */}
          {mission?.plan && mission.plan.length > 0 && (
            <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4 dark:border-neutral-800 dark:bg-neutral-950/50">
              <h3 className="text-xs font-bold uppercase tracking-wider text-neutral-500 mb-3">
                Plano de Execução ({mission.plan.filter((s) => s.state.toUpperCase() === "DONE" || s.state.toUpperCase() === "COMPLETED").length}/{mission.plan.length})
              </h3>
              <div className="space-y-3">
                {mission.plan.map((step, idx) => {
                  const state = step.state.toUpperCase();
                  const isDone = state === "DONE" || state === "COMPLETED" || state === "SUCCEEDED";
                  const isRunning = state === "RUNNING" || state === "OBSERVING";
                  const isFailed = state === "FAILED";
                  const approval = mission.approvals?.find((a) => a.step_id === step.id && a.status === "PENDING");

                  return (
                    <div
                      key={step.id}
                      className={`rounded-xl border p-3.5 transition-colors ${
                        isRunning
                          ? "border-violet-300 bg-violet-50/40 dark:border-violet-800 dark:bg-violet-950/20"
                          : approval
                          ? "border-amber-300 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/30"
                          : isDone
                          ? "border-neutral-200 bg-white dark:border-neutral-800/80 dark:bg-neutral-900"
                          : "border-neutral-200/60 bg-white/40 dark:border-neutral-800/40 dark:bg-neutral-900/30"
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <span className="mt-0.5 shrink-0">
                          {isDone ? (
                            <CheckCircleIcon className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                          ) : isRunning ? (
                            <ArrowPathIcon className="h-5 w-5 text-violet-600 dark:text-violet-400 animate-spin" />
                          ) : isFailed ? (
                            <ExclamationCircleIcon className="h-5 w-5 text-red-600" />
                          ) : approval ? (
                            <ClockIcon className="h-5 w-5 text-amber-600 dark:text-amber-400 animate-pulse" />
                          ) : (
                            <div className="flex h-5 w-5 items-center justify-center rounded-full bg-neutral-200 text-[10px] font-bold text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                              {idx + 1}
                            </div>
                          )}
                        </span>

                        <div className="min-w-0 flex-1">
                          <div className="flex items-center justify-between gap-2">
                            <p className="text-xs font-semibold text-neutral-900 dark:text-white truncate">
                              {step.title}
                            </p>
                            <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-[10px] text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400 shrink-0">
                              {step.kind}
                            </span>
                          </div>

                          {step.error && (
                            <p className="mt-1 text-xs text-red-600 dark:text-red-400 font-mono">
                              {step.error}
                            </p>
                          )}

                          {/* CARD DE APROVAÇÃO IN-LINE NA TIMELINE */}
                          {approval && (
                            <div className="mt-3 rounded-lg border border-amber-300 bg-amber-50 p-3 dark:border-amber-700/60 dark:bg-amber-950/50">
                              <p className="text-xs font-medium text-amber-900 dark:text-amber-200">
                                Esta etapa requer aprovação humana para continuar.
                              </p>
                              {approval.policy && (
                                <p className="mt-0.5 text-[11px] text-amber-700 dark:text-amber-300">
                                  Política: {approval.policy}
                                </p>
                              )}
                              <textarea
                                value={approvalReasons[approval.id] || ""}
                                onChange={(e) =>
                                  setApprovalReasons({ ...approvalReasons, [approval.id]: e.target.value })
                                }
                                placeholder="Informe a justificativa para aprovar ou rejeitar..."
                                className="mt-2 w-full rounded-md border border-amber-300 bg-white/90 p-2 text-xs text-neutral-900 outline-none focus:border-amber-500 dark:border-amber-700 dark:bg-neutral-900 dark:text-neutral-100"
                                rows={2}
                              />
                              <div className="mt-2.5 flex items-center gap-2">
                                <button
                                  type="button"
                                  onClick={() => handleDecideApproval(approval, true)}
                                  disabled={actionLoading}
                                  className="rounded bg-emerald-600 px-3 py-1 text-xs font-medium text-white hover:bg-emerald-700"
                                >
                                  Aprovar e Prosseguir
                                </button>
                                <button
                                  type="button"
                                  onClick={() => handleDecideApproval(approval, false)}
                                  disabled={actionLoading}
                                  className="rounded bg-red-600 px-3 py-1 text-xs font-medium text-white hover:bg-red-700"
                                >
                                  Rejeitar
                                </button>
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Cards de Artefatos Gerados (padrão observável do Manus) */}
          {mission?.artifacts && mission.artifacts.length > 0 && (
            <div className="mt-4 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 shadow-sm">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-xs font-bold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                  Artefatos da Missão ({mission.artifacts.length})
                </h4>
                <button
                  type="button"
                  onClick={() => setActiveCanvasTab("artifacts")}
                  className="text-xs font-medium text-violet-600 hover:text-violet-700 dark:text-violet-400"
                >
                  Ver todos
                </button>
              </div>
              <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
                {mission.artifacts.map((art) => (
                  <button
                    key={art.id}
                    type="button"
                    onClick={() => setActiveCanvasTab("artifacts")}
                    className="flex items-center gap-3 rounded-xl border border-neutral-200/80 bg-neutral-50/80 p-2.5 text-left transition hover:border-neutral-300 hover:bg-neutral-100 dark:border-neutral-800 dark:bg-neutral-950/60 dark:hover:border-neutral-700"
                  >
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950/60 dark:text-blue-400">
                      <DocumentTextIcon className="h-5 w-5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium text-neutral-900 dark:text-white">
                        {art.name}
                      </p>
                      <p className="text-[10px] text-neutral-400">
                        {Math.round(art.size / 1024) || 1} KB · SHA-256 verificado
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Event Stream Log */}
          {events.length > 0 && (
            <div className="mt-4 rounded-xl border border-neutral-200 bg-neutral-50/50 p-3 dark:border-neutral-800 dark:bg-neutral-950/30">
              <h4 className="text-[11px] font-bold uppercase tracking-wider text-neutral-400 mb-2">
                Trilha de Execução (SSE)
              </h4>
              <div className="space-y-1.5 font-mono text-xs">
                {events.slice(-15).map((evt) => (
                  <div key={evt.id} className="flex items-start gap-2 text-neutral-600 dark:text-neutral-400">
                    <span className="text-neutral-400">[{new Date(evt.created_at).toLocaleTimeString()}]</span>
                    <span className="font-semibold text-neutral-800 dark:text-neutral-200">{evt.type}</span>
                    {evt.step_id && <span className="text-neutral-500 truncate">({evt.step_id})</span>}
                  </div>
                ))}
              </div>
            </div>
          )}

          <div ref={timelineEndRef} />
        </div>

        {/* Input Composer no rodapé */}
        <footer className="border-t border-neutral-200 p-3.5 dark:border-neutral-800">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void handleSendPrompt();
            }}
            className="flex items-center gap-2 rounded-xl border border-neutral-300 bg-neutral-50 px-3 py-2 focus-within:border-neutral-500 focus-within:bg-white dark:border-neutral-700 dark:bg-neutral-950/60 dark:focus-within:border-neutral-500"
          >
            <input
              type="text"
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              aria-label="Instrução da missão"
              placeholder="Envie uma instrução, pergunte ou crie uma nova missão..."
              className="flex-1 bg-transparent text-sm outline-none text-neutral-900 dark:text-white"
            />
            <button
              type="submit"
              disabled={!chatInput.trim() || busy || actionLoading}
              aria-label="Enviar instrução"
              className="rounded-lg bg-neutral-900 p-1.5 text-white hover:bg-neutral-800 disabled:opacity-30 dark:bg-white dark:text-neutral-900"
            >
              <PaperAirplaneIcon className="h-4 w-4" />
            </button>
          </form>
        </footer>
      </section>

      {/* PAINEL DIREITO: Active Canvas (Browser ao vivo / Editor de Código / Artefatos) */}
      <section className="flex flex-1 flex-col bg-neutral-50 dark:bg-neutral-950 lg:w-1/2">
        {/* Barra de Abas do Canvas */}
        <header className="flex items-center justify-between border-b border-neutral-200 px-4 py-2 bg-white dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => setActiveCanvasTab("browser")}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeCanvasTab === "browser"
                  ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-sm"
                  : "text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
              }`}
            >
              <GlobeAltIcon className="h-4 w-4" />
              Navegador ao Vivo
              {browserFrame && <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />}
            </button>

            <button
              type="button"
              onClick={() => setActiveCanvasTab("artifacts")}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeCanvasTab === "artifacts"
                  ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-sm"
                  : "text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
              }`}
            >
              <DocumentDuplicateIcon className="h-4 w-4" />
              Artefatos ({mission?.artifacts?.length ?? 0})
            </button>

            <button
              type="button"
              onClick={() => setActiveCanvasTab("code")}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeCanvasTab === "code"
                  ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900 shadow-sm"
                  : "text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
              }`}
            >
              <WindowIcon className="h-4 w-4" />
              Terminal / Canvas
            </button>
          </div>
        </header>

        {/* Área Central do Canvas Ativo */}
        <div className="min-h-0 flex-1 overflow-hidden">
          {activeCanvasTab === "browser" ? (
            /* Navegador ao Vivo */
            <div className="flex h-full flex-col bg-neutral-900 text-neutral-100">
              {/* Barra de Endereço Simulada */}
              <div className="flex items-center gap-2 border-b border-neutral-800 bg-neutral-950 px-4 py-2">
                <div className="flex gap-1.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-red-500/80" />
                  <span className="h-2.5 w-2.5 rounded-full bg-yellow-500/80" />
                  <span className="h-2.5 w-2.5 rounded-full bg-green-500/80" />
                </div>
                <div className="flex flex-1 items-center gap-1.5 rounded-md bg-neutral-900 px-3 py-1 text-xs text-neutral-300">
                  <GlobeAltIcon className="h-3.5 w-3.5 text-neutral-500" />
                  <span className="font-mono truncate">{browserFrame?.url || "about:blank"}</span>
                </div>
              </div>

              {/* Viewport do Navegador */}
              <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto p-4">
                {browserFrame?.screenshot ? (
                  <img
                    src={browserFrame.screenshot}
                    alt={browserFrame.title || "Browser Viewport"}
                    className="max-h-full max-w-full rounded-lg shadow-2xl object-contain border border-neutral-800"
                  />
                ) : (
                  <div className="text-center text-neutral-500">
                    <GlobeAltIcon className="mx-auto h-12 w-12 stroke-[1.2]" />
                    <p className="mt-3 text-sm font-medium">Navegador aguardando ação.</p>
                    <p className="mt-1 text-xs">Ações como navegação, cliques e preenchimentos serão espelhadas em tempo real aqui.</p>
                  </div>
                )}
              </div>
            </div>
          ) : activeCanvasTab === "artifacts" ? (
            /* Visualizador de Artefatos e Entrega */
            <ArtifactsViewer
              missionId={mission?.id ?? ""}
              artifacts={mission?.artifacts ?? []}
            />
          ) : (
            /* Terminal / Canvas de Código */
            <div className="flex h-full flex-col bg-neutral-950 p-4 font-mono text-xs text-neutral-300 overflow-auto">
              <p className="text-neutral-500 mb-2">// Registro de saída e canvas de execução</p>
              {events
                .filter((e) => e.type.startsWith("step.") || e.type.startsWith("mission."))
                .map((e) => (
                  <div key={e.id} className="py-0.5">
                    <span className="text-violet-400">{e.type}</span>:{" "}
                    <span className="text-neutral-300">
                      {e.payload ? JSON.stringify(e.payload) : "(sem payload)"}
                    </span>
                  </div>
                ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
