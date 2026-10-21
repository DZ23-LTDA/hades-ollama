import { useState, useEffect } from "react";
import { Link } from "@tanstack/react-router";
import {
  listSchedules,
  createSchedule,
  runScheduleNow,
  reprocessSchedule,
  decideScheduleApproval,
  updateSchedule,
  deleteSchedule,
  type AgentSchedule,
} from "@/lib/agenticClient";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { FlowEditor } from "@/components/FlowEditor";
import { ScheduleRunReportCard } from "@/components/ScheduleRunReportCard";
import { scheduleRunConcerns, schedulePendingApprovals, scheduleRunSummary, scheduleRunVerdict, scheduleRunVerdictLabel } from "@/lib/scheduleRun";
import { humanizeApiError } from "@/lib/userFacingError";
import { confirmDialog } from "@/lib/confirmDialog";
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
  ArrowPathIcon,
} from "@heroicons/react/24/outline";
import { useT } from "@/i18n";

export function AutomationsPage() {
  const { t } = useT();
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
        setErrorMsg(humanizeApiError(e, t("automations.error.load")).message);
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
      setErrorMsg(t("automations.error.missingObjective"));
      return;
    }
    if (type === "webhook" && !webhookSecretEnv.trim()) {
      setErrorMsg(t("automations.error.missingSecret"));
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
      setSuccessMsg(t("automations.success.created"));
      setTimeout(() => setSuccessMsg(null), 3000);
      setNewObjective("");
      setWebhookSecretEnv("");
      setShowModal(false);
      setWebhookInfo(isWebhook ? { id: created.id, secretEnv } : null);
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.create")).message);
    } finally {
      setSaving(false);
    }
  };

  const handleApproval = async (sch: AgentSchedule, stepId: string, runKey: string, approved: boolean) => {
    setErrorMsg(null);
    try {
      const result = await decideScheduleApproval(sch.id, stepId, approved, approved ? "" : "recusado na interface", runKey);
      setSchedules((previous) => previous.map((item) => (item.id === sch.id ? result.schedule : item)));
      setSuccessMsg(
        approved
          ? t("automations.success.approved", { step: stepId })
          : t("automations.success.rejected", { step: stepId }),
      );
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.approval")).message);
    }
  };

  const handleReprocess = async (sch: AgentSchedule) => {
    setErrorMsg(null);
    try {
      // Reprocessa apenas o que ficou pendente, reusando a execução anterior: o
      // que já criou missão não é duplicado.
      const result = await reprocessSchedule(sch.id);
      setSchedules((previous) => previous.map((item) => (item.id === sch.id ? result.schedule : item)));
      if (!result.succeeded) {
        setSuccessMsg(null);
        setErrorMsg(t("automations.reprocess.failed", { error: result.error ?? t("automations.reprocess.errorUnknown") }));
        return;
      }
      const pending = scheduleRunConcerns(result.report).length;
      setSuccessMsg(
        pending === 0
          ? t("automations.reprocess.clean", { summary: scheduleRunSummary(result.report) })
          : t("automations.reprocess.pending", { count: pending, summary: scheduleRunSummary(result.report) }),
      );
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.reprocess")).message);
    }
  };

  const handleTriggerNow = async (sch: AgentSchedule) => {
    setErrorMsg(null);
    try {
      // Executa o GRAFO persistido da rotina, não uma missão avulsa: um fluxo com
      // condição e várias ações precisa rodar como fluxo também no disparo manual.
      const result = await runScheduleNow(sch.id);
      setSchedules((previous) => previous.map((item) => (item.id === sch.id ? result.schedule : item)));
      const verdict = scheduleRunVerdict(result.report);
      if (!result.succeeded) {
        setSuccessMsg(null);
        setTriggeredMission(null);
        setErrorMsg(
          t("automations.run.failed", {
            verdict: scheduleRunVerdictLabel(verdict),
            error: result.error ?? t("automations.reprocess.errorUnknown"),
          }),
        );
        return;
      }
      if (verdict === "partial") {
        setSuccessMsg(t("automations.run.partial", { summary: scheduleRunSummary(result.report) }));
      } else {
        setSuccessMsg(t("automations.run.done", { summary: scheduleRunSummary(result.report) }));
      }
      setTriggeredMission({ id: sch.id, objective: sch.objective });
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.run")).message);
    }
  };

  const handleToggle = async (sch: AgentSchedule) => {
    try {
      await updateSchedule(sch.id, { enabled: !sch.enabled });
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.toggle")).message);
    }
  };

  const handleDelete = async (id: string) => {
    if (!(await confirmDialog(t("automations.delete.confirm"), { danger: true, confirmLabel: t("automations.delete.confirmLabel") }))) return;
    try {
      await deleteSchedule(id);
      loadData();
    } catch (e) {
      setErrorMsg(humanizeApiError(e, t("automations.error.delete")).message);
    }
  };

  const formatInterval = (sec: number) => {
    if (sec === 3600) return t("automations.interval.hour1");
    if (sec === 86400) return t("automations.interval.day24Long");
    if (sec === 604800) return t("automations.interval.week7Long");
    const h = Math.round(sec / 3600);
    return t("automations.interval.everyHours", { hours: h });
  };

  return (
    <SidebarLayout title={t("automations.title")} sidebar={<AppSidebar current="scheduled" />}>
      <div className="mx-auto max-w-6xl space-y-8 px-4 py-8 lg:px-8">
        {/* Cabeçalho */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white sm:text-3xl">
              {t("automations.title")}
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              {t("automations.description")}
            </p>
          </div>
          {view === "list" && (
            <button
              type="button"
              onClick={() => setShowModal(true)}
              className="inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              <PlusIcon className="h-4 w-4" />
              {t("automations.new")}
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
            {t("automations.view.list")}
          </button>
          <button
            type="button"
            onClick={() => setView("flow")}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium ${view === "flow" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-600 dark:text-neutral-300"}`}
          >
            {t("automations.view.flow")}
          </button>
        </div>

        {view === "flow" && (
          <section className="space-y-3">
            <p className="text-sm text-neutral-500 dark:text-neutral-400">
              {t("automations.flow.hint")}
            </p>
            <FlowEditor
              onPublished={() => {
                loadData();
                setView("list");
                setSuccessMsg(t("automations.flow.published"));
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
                {t("automations.followMission")}
              </Link>
            )}
          </div>
        )}

        {/* Instruções de acionamento do webhook recém-criado (defeito: webhook era um beco sem saída) */}
        {webhookInfo && (
          <div className="rounded-2xl border border-amber-200 bg-amber-50 p-5 text-sm dark:border-amber-900/60 dark:bg-amber-950/30">
            <div className="flex items-start justify-between gap-3">
              <h3 className="font-bold text-amber-900 dark:text-amber-200">
                {t("automations.webhook.title")}
              </h3>
              <button
                type="button"
                onClick={() => setWebhookInfo(null)}
                className="shrink-0 rounded-lg border border-amber-300 px-2 py-1 text-xs font-medium text-amber-800 hover:bg-amber-100 dark:border-amber-800 dark:text-amber-200"
              >
                {t("automations.webhook.close")}
              </button>
            </div>
            <p className="mt-2 text-xs leading-relaxed text-amber-800 dark:text-amber-300">
              {t("automations.webhook.intro")} <code className="font-mono">POST</code>{" "}
              {t("automations.webhook.introTail")}
            </p>
            <dl className="mt-3 space-y-2">
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  {t("automations.webhook.endpoint")}
                </dt>
                <dd className="mt-0.5 break-all rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {`${window.location.origin}/api/agent/v1/webhooks/${webhookInfo.id}`}
                </dd>
              </div>
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  {t("automations.webhook.secretHeader")}
                </dt>
                <dd className="mt-0.5 rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {t("automations.webhook.secretValue", { env: webhookInfo.secretEnv })}
                </dd>
              </div>
              <div>
                <dt className="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-400">
                  {t("automations.webhook.idempotency")}
                </dt>
                <dd className="mt-0.5 rounded-lg border border-amber-200 bg-white px-2.5 py-1.5 font-mono text-xs text-neutral-800 dark:border-amber-900/60 dark:bg-neutral-950 dark:text-neutral-200">
                  {t("automations.webhook.idempotencyExample")}
                </dd>
              </div>
            </dl>
            <p className="mt-3 text-[11px] leading-relaxed text-amber-700 dark:text-amber-400">
              {t("automations.webhook.secretExplain")} <code className="font-mono">X-Ollama-Agent-Secret</code>{" "}
              {t("automations.webhook.secretExplainMid")} <code className="font-mono">{webhookInfo.secretEnv}</code>{" "}
              {t("automations.webhook.secretExplainTail")}
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
                {t("automations.card.schedule.title")}
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                {t("automations.card.schedule.description")}
              </p>
              <div className="mt-4 space-y-2 border-t border-neutral-100 pt-3 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => handleCreate(t("automations.card.schedule.objectiveProgress"), 86400, "interval")}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  {t("automations.card.schedule.presetProgress")}
                </button>
                <button
                  type="button"
                  onClick={() => handleCreate(t("automations.card.schedule.objectiveSecurity"), 604800, "interval")}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  {t("automations.card.schedule.presetSecurity")}
                </button>
              </div>
            </div>
            <button
              type="button"
              onClick={() => { setTriggerType("interval"); setErrorMsg(null); setShowModal(true); }}
              className="mt-6 w-full rounded-xl border border-neutral-200 bg-white py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
            >
              {t("automations.card.schedule.action")}
            </button>
          </div>

          {/* Card 2: Gatilho Condicional */}
          <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm transition-all hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900">
            <div>
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-600 dark:bg-amber-950/50 dark:text-amber-400">
                <BoltIcon className="h-6 w-6" />
              </div>
              <h3 className="mt-4 text-lg font-bold text-neutral-900 dark:text-white">
                {t("automations.card.trigger.title")}
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                {t("automations.card.trigger.description")}
              </p>
              <div className="mt-4 space-y-2 border-t border-neutral-100 pt-3 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => { setTriggerType("webhook"); setNewObjective(t("automations.card.trigger.objectiveTests")); setShowModal(true); }}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  {t("automations.card.trigger.presetTests")}
                </button>
                <button
                  type="button"
                  onClick={() => { setTriggerType("webhook"); setNewObjective(t("automations.card.trigger.objectiveInvoices")); setShowModal(true); }}
                  className="w-full text-left rounded-lg bg-neutral-50 px-2.5 py-1.5 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  {t("automations.card.trigger.presetInvoices")}
                </button>
              </div>
            </div>
            <button
              type="button"
              onClick={() => { setTriggerType("webhook"); setErrorMsg(null); setShowModal(true); }}
              className="mt-6 w-full rounded-xl border border-neutral-200 bg-white py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
            >
              {t("automations.card.trigger.action")}
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
                {t("automations.card.advanced.title")}
              </h3>
              <p className="mt-2 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                {t("automations.card.advanced.description")}
              </p>
              <div className="mt-4 space-y-2">
                <input
                  type="text"
                  value={naturalPrompt}
                  onChange={(e) => setNaturalPrompt(e.target.value)}
                  placeholder={t("automations.card.advanced.placeholder")}
                  className="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-900 focus:bg-white focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                />
                <label className="block text-[11px] font-medium text-neutral-600 dark:text-neutral-400">
                  {t("automations.card.advanced.frequency")}
                  <select
                    value={naturalInterval}
                    onChange={(e) => setNaturalInterval(e.target.value)}
                    className="mt-1 w-full rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:bg-white focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  >
                    <option value="3600">{t("automations.interval.hour1")}</option>
                    <option value="21600">{t("automations.interval.hour6")}</option>
                    <option value="43200">{t("automations.interval.hour12")}</option>
                    <option value="86400">{t("automations.interval.day24")}</option>
                    <option value="604800">{t("automations.interval.week7")}</option>
                  </select>
                </label>
              </div>
            </div>
            <button
              type="button"
              onClick={() => {
                if (!naturalPrompt.trim()) {
                  setErrorMsg(t("automations.card.advanced.missingPrompt"));
                  return;
                }
                void handleCreate(naturalPrompt, parseInt(naturalInterval, 10), "interval");
                setNaturalPrompt("");
              }}
              className="mt-6 w-full rounded-xl bg-neutral-900 py-2 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
            >
              {t("automations.card.advanced.start")}
            </button>
          </div>
        </div>

        {/* Modal Inline de Criação de Automação */}
        {errorMsg && <div role="alert" className="rounded-xl border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">{errorMsg}</div>}
        {showModal && (
          <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-md dark:border-neutral-800 dark:bg-neutral-950">
            <h3 className="text-base font-bold text-neutral-900 dark:text-white">
              {t("automations.modal.title")}
            </h3>
            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                  {t("automations.modal.objective")}
                </label>
                <input
                  type="text"
                  value={newObjective}
                  onChange={(e) => setNewObjective(e.target.value)}
                  placeholder={t("automations.modal.objectivePlaceholder")}
                  className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">{t("automations.modal.triggerType")}</label>
                <select value={triggerType} onChange={(e) => setTriggerType(e.target.value as "interval" | "webhook")} className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-800">
                  <option value="interval">{t("automations.modal.triggerInterval")}</option><option value="webhook">{t("automations.modal.triggerWebhook")}</option>
                </select>
              </div>
              {triggerType === "webhook" && <div>
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">{t("automations.modal.secretEnv")}</label>
                <input value={webhookSecretEnv} onChange={(e) => setWebhookSecretEnv(e.target.value)} placeholder={t("automations.modal.secretEnvPlaceholder")} className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-800" />
              </div>}
              {triggerType === "webhook" ? (
                <div className="flex items-end">
                  <p className="text-[11px] leading-relaxed text-neutral-500 dark:text-neutral-400">
                    {t("automations.modal.webhookNote")}
                  </p>
                </div>
              ) : (
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    {t("automations.modal.frequency")}
                  </label>
                  <select
                    value={newInterval}
                    onChange={(e) => setNewInterval(e.target.value)}
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  >
                    <option value="3600">{t("automations.interval.hour1")}</option>
                    <option value="21600">{t("automations.interval.hour6")}</option>
                    <option value="43200">{t("automations.interval.hour12")}</option>
                    <option value="86400">{t("automations.interval.day24")}</option>
                    <option value="604800">{t("automations.interval.week7")}</option>
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
                {t("automations.modal.cancel")}
              </button>
              <button
                type="button"
                disabled={saving || !newObjective.trim() || (triggerType === "webhook" && !webhookSecretEnv.trim())}
                onClick={() => handleCreate(newObjective, parseInt(newInterval, 10), triggerType)}
                className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-neutral-900"
              >
                {saving ? t("automations.modal.saving") : t("automations.modal.save")}
              </button>
            </div>
          </div>
        )}

        {/* Lista de Automações Existentes */}
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-bold text-neutral-900 dark:text-white">
              {t("automations.list.title", { count: schedules.length })}
            </h2>
            <Link
              to="/agentic"
              className="text-xs font-medium text-neutral-500 hover:text-neutral-900 dark:hover:text-white"
            >
              {t("automations.list.openConsole")}
            </Link>
          </div>

          {loading ? (
            <div className="py-12 text-center text-sm text-neutral-500">
              {t("automations.list.loading")}
            </div>
          ) : schedules.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-neutral-300 p-12 text-center dark:border-neutral-700">
              <ClockIcon className="mx-auto h-10 w-10 text-neutral-300 dark:text-neutral-600" />
              <h3 className="mt-3 text-sm font-semibold text-neutral-900 dark:text-white">
                {t("automations.list.empty")}
              </h3>
              <p className="mt-1 text-xs text-neutral-500">
                {t("automations.list.emptyHint")}
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
                        ? t("automations.item.webhookOnly")
                        : t("automations.item.nextRun", {
                            interval: formatInterval(sch.interval_seconds),
                            when: new Date(sch.next_run_at).toLocaleString("pt-BR"),
                          })}
                    </p>
                    {/* Resultado por nó da última execução: uma rotina com passo
                        ignorado NÃO aparece como sucesso limpo. */}
                    <ScheduleRunReportCard
                      run={sch.last_run ?? null}
                      lastFailureCode={sch.last_failure_code ?? null}
                      lastRunAt={sch.last_run_at ?? null}
                    />
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => handleTriggerNow(sch)}
                      className="inline-flex items-center gap-1.5 rounded-xl bg-neutral-900 px-3 py-1.5 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
                      title={t("automations.item.runNowTitle")}
                    >
                      <PlayIcon className="h-3.5 w-3.5" />
                      {t("automations.item.runNow")}
                    </button>
                    {scheduleRunConcerns(sch.last_run ?? null).length > 0 && (
                      <button
                        type="button"
                        onClick={() => handleReprocess(sch)}
                        className="inline-flex items-center gap-1.5 rounded-xl border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-800 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200"
                        title={t("automations.item.reprocessTitle")}
                      >
                        <ArrowPathIcon className="h-3.5 w-3.5" />
                        {t("automations.item.reprocess")}
                      </button>
                    )}
                    {schedulePendingApprovals(sch.last_run ?? null).map((pending) => (
                      <span key={`${pending.step_id}-${pending.run_key}`} className="inline-flex items-center gap-1.5">
                        <button
                          type="button"
                          onClick={() => handleApproval(sch, pending.step_id, pending.run_key, true)}
                          className="inline-flex items-center gap-1.5 rounded-xl bg-sky-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-sky-700"
                          title={t("automations.item.approveTitle", { step: pending.step_id })}
                        >
                          <CheckCircleIcon className="h-3.5 w-3.5" />
                          {t("automations.item.approve", { step: pending.step_id })}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleApproval(sch, pending.step_id, pending.run_key, false)}
                          className="inline-flex items-center gap-1.5 rounded-xl border border-neutral-300 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                          title={t("automations.item.rejectTitle", { step: pending.step_id })}
                        >
                          {t("automations.item.reject")}
                        </button>
                      </span>
                    ))}
                    <button
                      type="button"
                      onClick={() => handleToggle(sch)}
                      className="rounded-xl border border-neutral-200 p-2 text-neutral-600 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
                      title={sch.enabled ? t("automations.item.pause") : t("automations.item.resume")}
                    >
                      {sch.enabled ? <PauseIcon className="h-4 w-4" /> : <PlayIcon className="h-4 w-4" />}
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(sch.id)}
                      className="rounded-xl border border-neutral-200 p-2 text-red-600 hover:bg-red-50 dark:border-neutral-700 dark:hover:bg-red-950/30"
                      title={t("automations.item.deleteTitle")}
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
