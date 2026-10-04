import { useState, useEffect } from "react";
import { Link } from "@tanstack/react-router";
import {
  listSchedules,
  createSchedule,
  createMission,
  type AgentSchedule,
} from "@/lib/agenticClient";
import { API_BASE } from "@/lib/config";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { FlowEditor } from "@/components/FlowEditor";
import { humanizeApiError } from "@/lib/userFacingError";
import {
  CalendarDaysIcon,
  BoltIcon,
  SparklesIcon,
  PlusIcon,
  PlayIcon,
  TrashIcon,
  ClockIcon,
  CheckCircleIcon,
  PauseIcon,
} from "@heroicons/react/24/outline";

export function AutomationsPage() {
  const [schedules, setSchedules] = useState<AgentSchedule[]>([]);
  const [view, setView] = useState<"list" | "flow">("list");
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [naturalPrompt, setNaturalPrompt] = useState("");
  const [naturalInterval, setNaturalInterval] = useState("86400"); // 24h default
  const [newObjective, setNewObjective] = useState("");
  const [newInterval, setNewInterval] = useState("86400"); // 24h default
  const [triggerType, setTriggerType] = useState<"interval" | "webhook">("interval");
  const [webhookSecretEnv, setWebhookSecretEnv] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  // Após criar um webhook, guardamos o id e a variável do segredo para mostrar a
  // URL do endpoint e as instruções de acionamento (o webhook não tem UI até aqui).
  const [webhookInfo, setWebhookInfo] = useState<{ id: string; secretEnv: string } | null>(null);
  // Após "Executar agora", guardamos a missão criada para oferecer acompanhá-la.
  const [triggeredMission, setTriggeredMission] = useState<{ id: string; objective: string } | null>(null);

  const loadData = () => {
    setLoading(true);
    listSchedules()
      .then((data) => {
        setSchedules(data.schedules || []);
        setErrorMsg(null);
      })
      .catch((e) => {
        setSchedules([]);
        setErrorMsg(humanizeApiError(e, "Não foi possível carregar as automações.").message);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadData();
  }, []);

  // Webhooks precisam ser acionados SOMENTE pelo endpoint, nunca por timer. O
  // backend exige um intervalo válido (1s a 31 dias) e sempre marca enabled=true,
  // então empurramos next_run_at para o futuro distante: o agendador só reivindica
  // rotinas cujo next_run_at já passou, de modo que a rotina de webhook nunca roda
  // no relógio, enquanto o endpoint do webhook continua funcionando.
  const WEBHOOK_NEVER_RUN_AT = "9999-12-31T23:59:59Z";

  const handleCreate = async (objective: string, intervalSeconds: number, type = triggerType) => {
    if (!objective.trim()) {
      setErrorMsg("Informe o objetivo/instrução da automação.");
      return;
    }
    if (type === "webhook" && !webhookSecretEnv.trim()) {
      setErrorMsg("Informe a variável de ambiente com o segredo do webhook.");
      return;
    }
    setErrorMsg(null);
    setSaving(true);
    const isWebhook = type === "webhook";
    const secretEnv = webhookSecretEnv.trim();
    try {
      const payload: Partial<AgentSchedule> = {
        objective: objective.trim(),
        interval_seconds: intervalSeconds,
        enabled: true,
      };
      if (isWebhook) {
        payload.webhook_secret_env = secretEnv;
        payload.next_run_at = WEBHOOK_NEVER_RUN_AT;
      }
      const created = await createSchedule(payload);
      setSuccessMsg("Automação criada com sucesso!");
      setTimeout(() => setSuccessMsg(null), 3000);
      setNewObjective("");
      setWebhookSecretEnv("");
      setShowModal(false);
      setWebhookInfo(isWebhook ? { id: created.id, secretEnv } : null);
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, "Não foi possível criar a automação.").message);
    } finally {
      setSaving(false);
    }
  };

  const handleTriggerNow = async (sch: AgentSchedule) => {
    setErrorMsg(null);
    try {
      const mission = await createMission({
        objective: `[Execução Manual] ${sch.objective}`,
        auto_run: true,
      });
      setSuccessMsg(`Missão disparada: ${sch.objective}`);
      setTriggeredMission({ id: mission.id, objective: sch.objective });
    } catch (e) {
      setErrorMsg(humanizeApiError(e, "Não foi possível executar a automação.").message);
    }
  };

  const handleToggle = async (sch: AgentSchedule) => {
    try {
      await fetch(`${API_BASE}/api/agent/v1/schedules/${sch.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: !sch.enabled }),
      });
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, "Não foi possível alterar o estado da automação.").message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm("Tem certeza que deseja excluir esta automação?")) return;
    try {
      await fetch(`${API_BASE}/api/agent/v1/schedules/${id}`, {
        method: "DELETE",
      });
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, "Não foi possível excluir a automação.").message);
    }
  };

  const formatInterval = (sec: number) => {
    if (sec === 3600) return "A cada 1 hora";
    if (sec === 86400) return "Diariamente (a cada 24 horas)";
    if (sec === 604800) return "Semanalmente (a cada 7 dias)";
    const h = Math.round(sec / 3600);
    return `A cada ${h} horas`;
  };

  return (
    <SidebarLayout title="Automações" sidebar={<AppSidebar current="scheduled" />}>
      <div className="mx-auto max-w-6xl space-y-8 px-4 py-8 lg:px-8">
        {/* Cabeçalho */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white sm:text-3xl">
              Automações
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              Automatize fluxos recorrentes, configure gatilhos e deixe o agente trabalhar em segundo plano.
            </p>
          </div>
          {view === "list" && (
            <button
              type="button"
              onClick={() => setShowModal(true)}
              className="inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <PlusIcon className="h-4 w-4" />
              Nova automação
            </button>
          )}
        </div>

        {/* Alternância entre a lista de rotinas e o editor visual de fluxo */}
        <div className="inline-flex rounded-xl border border-neutral-200 p-1 dark:border-neutral-700">
          <button
            type="button"
            onClick={() => setView("list")}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium ${view === "list" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-600 dark:text-neutral-300"}`}
          >
            Lista
          </button>
          <button
            type="button"
            onClick={() => setView("flow")}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium ${view === "flow" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-600 dark:text-neutral-300"}`}
          >
            Editor visual (beta)
          </button>
        </div>

        {view === "flow" && (
          <section className="space-y-3">
            <p className="text-sm text-neutral-500 dark:text-neutral-400">
              Monte o fluxo arrastando nós da paleta, conecte as etapas e valide antes de publicar.
              O rascunho fica salvo neste navegador.
            </p>
            <FlowEditor
              onPublished={() => {
                loadData();
                setView("list");
                setSuccessMsg("Fluxo publicado como automação. Veja em Rotinas Ativas.");
              }}
            />
          </section>
        )}

        {view === "list" && (<>

        {/* Mensagem de sucesso */}
        {successMsg && (
          <div className="flex flex-col gap-2 rounded-xl bg-emerald-50 p-4 text-sm font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300 sm:flex-row sm:items-center sm:justify-between">
            <span className="flex items-center gap-2">
              <CheckCircleIcon className="h-5 w-5 text-emerald-500" />
              {successMsg}
            </span>
            {triggeredMission && (
              <Link
                to="/agentic"
                onClick={() => setTriggeredMission(null)}
                className="shrink-0 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-700"
              >
                Acompanhar no Console de Missões →
              </Link>
            )}
          </div>
        )}

        {/* Instruções de acionamento do webhook recém-criado (defeito: webhook era um beco sem saída) */}
        {webhookInfo && (
          <div className="rounded-2xl border border-amber-200 bg-amber-50 p-5 text-sm dark:border-amber-900/60 dark:bg-amber-950/30">
            <div className="flex items-start justify-between gap-3">
              <h3 className="font-bold text-amber-900 dark:text-amber-200">
                Gatilho de webhook criado — como acionar
              </h3>
              <button
                type="button"
                onClick={() => setWebhookInfo(null)}
                className="shrink-0 rounded-lg border border-amber-300 px-2 py-1 text-xs font-medium text-amber-800 hover:bg-amber-100 dark:border-amber-800 dark:text-amber-200"
              >
                Fechar
              </button>
            </div>
            <p className="mt-2 text-xs leading-relaxed text-amber-800 dark:text-amber-300">
              Este gatilho roda apenas quando o endpoint abaixo recebe uma requisição
              autenticada (não roda por tempo). Envie um <code className="font-mono">POST</code> com
              o cabeçalho do segredo e uma chave de idempotência única por evento.
            </p>
            <dl className="mt-3 space-y-2">
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  Endpoint (POST)
                </dt>
                <dd className="mt-0.5 break-all rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {`${window.location.origin}/api/agent/v1/webhooks/${webhookInfo.id}`}
                </dd>
              </div>
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  Cabeçalho do segredo
                </dt>
                <dd className="mt-0.5 rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {`X-Ollama-Agent-Secret: <valor da variável de ambiente ${webhookInfo.secretEnv}>`}
                </dd>
              </div>
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  Chave de idempotência (obrigatória, única por evento)
                </dt>
                <dd className="mt-0.5 rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {"Idempotency-Key: <id único do evento>"}
                </dd>
              </div>
            </dl>
            <p className="mt-3 text-[11px] leading-relaxed text-amber-700 dark:text-amber-400">
              O valor de <code className="font-mono">X-Ollama-Agent-Secret</code> deve ser igual ao
              conteúdo da variável de ambiente <code className="font-mono">{webhookInfo.secretEnv}</code>{" "}
              configurada no servidor. Requisições sem a chave de idempotência ou com segredo inválido
              são recusadas.
            </p>
          </div>
        )}

        {/* Os 3 Cards Oficiais de Automação do Manus */}
        <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
          {/* Card 1: Agendamento Recorrente */}
          <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm transition-all hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900">
            <div>
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-50 text-blue-600 dark:bg-blue-950/50 dark:text-blue-400">
                <CalendarDaysIcon className="h-6 w-6" />
              </div>
              <h3 className="mt-4 text-lg font-bold text-neutral-900 dark:text-white">
                Agendamento
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                Automatize trabalhos recorrentes escolhendo quando e com que frequência o Hades deve ser executado.
              </p>
              <div className="mt-4 space-y-2 border-t border-neutral-100 pt-3 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => handleCreate("Auditar arquivos e gerar relatório de progresso", 86400, "interval")}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  ⚡ Relatório diário de progresso (24h)
                </button>
                <button
                  type="button"
                  onClick={() => handleCreate("Verificar atualizações de dependências e segurança", 604800, "interval")}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  ⚡ Varredura semanal de segurança (7d)
                </button>
              </div>
            </div>
            <button
              type="button"
              onClick={() => { setTriggerType("interval"); setErrorMsg(null); setShowModal(true); }}
              className="mt-6 w-full rounded-xl border border-neutral-200 bg-white py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
            >
              Criar agendamento
            </button>
          </div>

          {/* Card 2: Gatilho Condicional */}
          <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm transition-all hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900">
            <div>
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400">
                <BoltIcon className="h-6 w-6" />
              </div>
              <h3 className="mt-4 text-lg font-bold text-neutral-900 dark:text-white">
                Gatilho
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                Configure um webhook autenticado; o endpoint exige segredo em variável de ambiente e chave de idempotência.
              </p>
              <div className="mt-4 space-y-2 border-t border-neutral-100 pt-3 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => { setTriggerType("webhook"); setNewObjective("Disparar testes e validar build quando novo commit for detectado"); setShowModal(true); }}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  ⚡ Testar repositório ao detectar commits
                </button>
                <button
                  type="button"
                  onClick={() => { setTriggerType("webhook"); setNewObjective("Processar arquivos e faturas adicionadas na pasta local"); setShowModal(true); }}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  ⚡ Extrair dados de faturas recebidas
                </button>
              </div>
            </div>
            <button
              type="button"
              onClick={() => { setTriggerType("webhook"); setErrorMsg(null); setShowModal(true); }}
              className="mt-6 w-full rounded-xl border border-neutral-200 bg-white py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
            >
              Criar gatilho
            </button>
          </div>

          {/* Card 3: Automação Avançada (Beta) */}
          <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm transition-all hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900">
            <div>
              <div className="flex items-center justify-between">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-purple-50 text-purple-600 dark:bg-purple-950/50 dark:text-purple-400">
                  <SparklesIcon className="h-6 w-6" />
                </div>
                <span className="rounded-full bg-purple-100 px-2 py-0.5 text-[10px] font-bold text-purple-700 dark:bg-purple-950/60 dark:text-purple-300">
                  Beta
                </span>
              </div>
              <h3 className="mt-4 text-lg font-bold text-neutral-900 dark:text-white">
                Automação avançada
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                Descreva com suas próprias palavras o que o Hades deve fazer e escolha a frequência
                abaixo. A interpretação de horários em linguagem natural ainda não está disponível:
                por ora a automação roda em intervalo fixo.
              </p>
              <div className="mt-4 space-y-2">
                <input
                  type="text"
                  value={naturalPrompt}
                  onChange={(e) => setNaturalPrompt(e.target.value)}
                  placeholder="Ex: Monitore meu repositório e gere um resumo..."
                  className="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-900 focus:bg-white focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                />
                <label className="block text-[11px] font-medium text-neutral-600 dark:text-neutral-400">
                  Frequência
                  <select
                    value={naturalInterval}
                    onChange={(e) => setNaturalInterval(e.target.value)}
                    className="mt-1 w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:bg-white focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  >
                    <option value="3600">A cada 1 hora</option>
                    <option value="21600">A cada 6 horas</option>
                    <option value="43200">A cada 12 horas</option>
                    <option value="86400">Diariamente (24 horas)</option>
                    <option value="604800">Semanalmente (7 dias)</option>
                  </select>
                </label>
              </div>
            </div>
            <button
              type="button"
              onClick={() => {
                if (!naturalPrompt.trim()) {
                  setErrorMsg("Descreva o que o Hades deve fazer antes de começar.");
                  return;
                }
                void handleCreate(naturalPrompt, parseInt(naturalInterval, 10), "interval");
                setNaturalPrompt("");
              }}
              className="mt-6 w-full rounded-xl bg-neutral-900 py-2 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              Começar
            </button>
          </div>
        </div>

        {/* Modal Inline de Criação de Automação */}
        {errorMsg && <div role="alert" className="rounded-xl border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">{errorMsg}</div>}
        {showModal && (
          <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-md dark:border-neutral-800 dark:bg-neutral-950">
            <h3 className="text-base font-bold text-neutral-900 dark:text-white">
              Configurar Nova Rotina de Automação
            </h3>
            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                  Objetivo / Instrução para o Agente
                </label>
                <input
                  type="text"
                  value={newObjective}
                  onChange={(e) => setNewObjective(e.target.value)}
                  placeholder="Ex: Comparar dados de concorrentes e atualizar relatório"
                  className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">Tipo de gatilho</label>
                <select value={triggerType} onChange={(e) => setTriggerType(e.target.value as "interval" | "webhook")} className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-800">
                  <option value="interval">Intervalo recorrente</option><option value="webhook">Webhook/evento</option>
                </select>
              </div>
              {triggerType === "webhook" && <div>
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">Variável do segredo webhook</label>
                <input value={webhookSecretEnv} onChange={(e) => setWebhookSecretEnv(e.target.value)} placeholder="NOME_DA_VARIAVEL" className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-800" />
              </div>}
              {triggerType === "webhook" ? (
                <div className="flex items-end">
                  <p className="text-[11px] leading-relaxed text-neutral-500 dark:text-neutral-400">
                    Gatilhos de webhook não rodam por tempo: são acionados apenas pelo endpoint
                    autenticado, cuja URL e instruções aparecem após salvar.
                  </p>
                </div>
              ) : (
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    Frequência de Execução
                  </label>
                  <select
                    value={newInterval}
                    onChange={(e) => setNewInterval(e.target.value)}
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  >
                    <option value="3600">A cada 1 hora</option>
                    <option value="21600">A cada 6 horas</option>
                    <option value="43200">A cada 12 horas</option>
                    <option value="86400">Diariamente (24 horas)</option>
                    <option value="604800">Semanalmente (7 dias)</option>
                  </select>
                </div>
              )}
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="rounded-xl border border-neutral-200 px-4 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300"
              >
                Cancelar
              </button>
              <button
                type="button"
                disabled={saving || !newObjective.trim() || (triggerType === "webhook" && !webhookSecretEnv.trim())}
                onClick={() => handleCreate(newObjective, parseInt(newInterval, 10), triggerType)}
                className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-neutral-900"
              >
                {saving ? "Salvando..." : "Salvar automação"}
              </button>
            </div>
          </div>
        )}

        {/* Lista de Automações Existentes */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-bold text-neutral-900 dark:text-white">
              Rotinas Ativas ({schedules.length})
            </h2>
            <Link
              to="/agentic"
              className="text-xs font-medium text-neutral-500 hover:text-neutral-900 dark:hover:text-white"
            >
              Abrir Console de Missões →
            </Link>
          </div>

          {loading ? (
            <div className="py-12 text-center text-sm text-neutral-500">
              Carregando automações...
            </div>
          ) : schedules.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-neutral-300 p-12 text-center dark:border-neutral-700">
              <ClockIcon className="mx-auto h-10 w-10 text-neutral-300 dark:text-neutral-600" />
              <h3 className="mt-3 text-sm font-semibold text-neutral-900 dark:text-white">
                Nenhuma rotina configurada
              </h3>
              <p className="mt-1 text-xs text-neutral-500">
                Use os cartões acima para criar seu primeiro agendamento ou gatilho autônomo.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-neutral-200 overflow-hidden rounded-2xl border border-neutral-200 bg-white dark:divide-neutral-800 dark:border-neutral-800 dark:bg-neutral-900">
              {schedules.map((sch) => (
                <div
                  key={sch.id}
                  className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span
                        className={`h-2.5 w-2.5 rounded-full ${
                          sch.enabled ? "bg-emerald-500 animate-pulse" : "bg-neutral-400"
                        }`}
                      />
                      <h4 className="text-sm font-bold text-neutral-900 dark:text-white">
                        {sch.objective}
                      </h4>
                    </div>
                    <p className="text-xs text-neutral-500 dark:text-neutral-400">
                      {sch.webhook_secret_env
                        ? "Acionada por webhook (não roda por tempo)"
                        : `${formatInterval(sch.interval_seconds)} • Próxima execução: ${new Date(sch.next_run_at).toLocaleString("pt-BR")}`}
                    </p>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => handleTriggerNow(sch)}
                      className="inline-flex items-center gap-1.5 rounded-xl bg-neutral-900 px-3 py-1.5 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
                      title="Executar imediatamente"
                    >
                      <PlayIcon className="h-3.5 w-3.5" />
                      Executar agora
                    </button>
                    <button
                      type="button"
                      onClick={() => handleToggle(sch)}
                      className="rounded-xl border border-neutral-200 p-2 text-neutral-600 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
                      title={sch.enabled ? "Pausar" : "Retomar"}
                    >
                      {sch.enabled ? <PauseIcon className="h-4 w-4" /> : <PlayIcon className="h-4 w-4" />}
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(sch.id)}
                      className="rounded-xl border border-neutral-200 p-2 text-red-600 hover:bg-red-50 dark:border-neutral-700 dark:hover:bg-red-950/30"
                      title="Excluir rotina"
                    >
                      <TrashIcon className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
        </>)}
      </div>
    </SidebarLayout>
  );
}
