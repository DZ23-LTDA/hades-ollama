import { useCallback, useEffect, useState } from "react";
import {
  addCompanyBacklog,
  addCompanyGoal,
	addCompanyCycle,
	addCompanyRoadmap,
	createCompany,
	executeCompanyTelAgent,
	getCompanyReport,
	getCompanyTelAgentHistory,
  listCompanies,
  pauseCompany,
  recordCompanyAnomaly,
  recordCompanySpend,
  resumeCompany,
  updateCompany,
} from "@/lib/agenticClient";
import type { AgentCompany, AgentCompanyReport, TelAgentExchange } from "@/lib/agenticClient";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { CompanyGrowthPanel } from "@/components/CompanyGrowthPanel";
import { CompanyOperationsPanel } from "@/components/CompanyOperationsPanel";
import { CompanyApprovalQueue } from "@/components/CompanyApprovalQueue";
import { CompanySupervisorPanel } from "@/components/CompanySupervisorPanel";
import { ArrowPathIcon, BuildingOffice2Icon, CheckCircleIcon, ExclamationTriangleIcon, PauseCircleIcon, PlayCircleIcon, PlusIcon, ShieldCheckIcon } from "@heroicons/react/24/outline";

const inputClass = "h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
const areaClass = "min-h-20 w-full rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";

function Card({ title, children, className = "" }: { title: string; children: React.ReactNode; className?: string }) {
  return <section className={`rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900 ${className}`}><h2 className="text-sm font-semibold text-neutral-950 dark:text-white">{title}</h2>{children}</section>;
}
function Stat({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "green" | "amber" }) {
  const color = tone === "green" ? "text-emerald-700 bg-emerald-50 dark:text-emerald-300 dark:bg-emerald-950/30" : tone === "amber" ? "text-amber-700 bg-amber-50 dark:text-amber-300 dark:bg-amber-950/30" : "text-neutral-700 bg-neutral-100 dark:text-neutral-300 dark:bg-neutral-800";
  return <div className="rounded-xl border border-neutral-200/80 p-3 dark:border-neutral-800"><div className="text-[10px] uppercase tracking-[0.12em] text-neutral-400">{label}</div><div className={`mt-2 inline-flex rounded-full px-2 py-1 text-xs font-semibold ${color}`}>{value}</div></div>;
}

export function CompanyWorkspacePage() {
  const [companies, setCompanies] = useState<AgentCompany[]>([]);
  const [company, setCompany] = useState<AgentCompany | null>(null);
  const [report, setReport] = useState<AgentCompanyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [name, setName] = useState("");
  const [mission, setMission] = useState("");
  const [positioning, setPositioning] = useState("");
  const [businessModel, setBusinessModel] = useState("SaaS");
  const [audience, setAudience] = useState("");
  const [offer, setOffer] = useState("");
  const [budget, setBudget] = useState("0");
  const [roadmapTitle, setRoadmapTitle] = useState("");
  const [goalTitle, setGoalTitle] = useState("");
  const [goalMetric, setGoalMetric] = useState("");
  const [goalTarget, setGoalTarget] = useState("1");
  const [backlogTitle, setBacklogTitle] = useState("");
  const [backlogPriority, setBacklogPriority] = useState("50");
  const [cycleName, setCycleName] = useState("Ciclo diário");
  const [cycleObjective, setCycleObjective] = useState("Revisar métricas, roadmap e executar a próxima tarefa priorizada");
  const [cycleFrequency, setCycleFrequency] = useState("daily");
  const [spendAmount, setSpendAmount] = useState("");
  const [spendCategory, setSpendCategory] = useState("ads");
  const [telAgentMessage, setTelAgentMessage] = useState("");
  const [telAgentOperation, setTelAgentOperation] = useState<"report.read" | "backlog.create" | "campaign.draft">("report.read");
  const [telAgentTitle, setTelAgentTitle] = useState("");
  const [telAgentDescription, setTelAgentDescription] = useState("");
  const [telAgentPriority, setTelAgentPriority] = useState("50");
  const [telAgentHistory, setTelAgentHistory] = useState<TelAgentExchange[]>([]);
  const [telAgentReply, setTelAgentReply] = useState<string | null>(null);
  const [telAgentIdempotencyKey, setTelAgentIdempotencyKey] = useState<string | null>(null);

  const refresh = useCallback(async (selectedID?: string) => {
    setLoading(true);
    try {
      const result = await listCompanies();
      setCompanies(result.companies);
      const next = result.companies.find((item) => item.id === (selectedID ?? company?.id)) ?? result.companies[0] ?? null;
      setCompany(next);
      if (next) {
	        const [nextReport, history] = await Promise.all([getCompanyReport(next.id), getCompanyTelAgentHistory(next.id)]);
	        setReport(nextReport);
	        setTelAgentHistory(Array.isArray(history?.history) ? history.history : []);
	      } else {
	        setReport(null);
        setTelAgentHistory([]);
      }
    } catch (cause) {
      setNotice(cause instanceof Error ? cause.message : "Não foi possível carregar o Company OS.");
    } finally {
      setLoading(false);
    }
  }, [company?.id]);

  useEffect(() => { void refresh(); }, [refresh]);

  const run = async (action: () => Promise<AgentCompany>, message: string) => {
    setBusy(true);
    try { const updated = await action(); setNotice(message); await refresh(updated.id); }
    catch (cause) { setNotice(cause instanceof Error ? cause.message : "A operação falhou."); }
    finally { setBusy(false); }
  };
  const create = async () => {
    if (!name.trim()) return;
    await run(() => createCompany({ name, mission, positioning, business_model: businessModel, target_audience: audience, offer, currency: "BRL", budget: { currency: "BRL", monthly_limit_cents: Math.round(Number(budget || 0) * 100), approval_threshold_cents: Math.round(Number(budget || 0) * 10), require_approval_for_ads: true, require_approval_for_sales: true } }), "Empresa criada como tenant local.");
    setName("");
  };
  const selectedID = company?.id;
  const add = async (action: () => Promise<AgentCompany>, message: string, clear: () => void) => { if (!selectedID) return; await run(action, message); clear(); };
  const sendTelAgent = async () => {
    if (!selectedID || !telAgentMessage.trim() || (telAgentOperation !== "report.read" && !telAgentTitle.trim())) return;
    setBusy(true);
    const requestKey = telAgentIdempotencyKey ?? (globalThis.crypto?.randomUUID?.() ?? `tel-${Date.now()}-${Math.random().toString(36).slice(2)}`);
    setTelAgentIdempotencyKey(requestKey);
    try {
      const result = await executeCompanyTelAgent(selectedID, { message: telAgentMessage, operation: telAgentOperation, ...(telAgentTitle.trim() ? { title: telAgentTitle } : {}), ...(telAgentDescription.trim() ? { description: telAgentDescription } : {}), ...(telAgentOperation === "backlog.create" ? { priority: Number(telAgentPriority || 50) } : {}) }, requestKey);
      setCompany(result.company);
      setTelAgentHistory((current) => [...current, result.exchange].slice(-100));
      setTelAgentReply(result.exchange.reply);
      setTelAgentMessage("");
      setTelAgentIdempotencyKey(null);
      setNotice(result.exchange.status === "approval_pending" ? "Tel-Agent criou um rascunho local; approval continua obrigatório." : "Tel-Agent concluiu a operação local.");
      if (result.report) setReport(result.report);
    } catch (cause) { setNotice(cause instanceof Error ? cause.message : "Tel-Agent não conseguiu concluir a operação."); }
    finally { setBusy(false); }
  };
  const departments = company?.departments ?? [];
  const backlog = company?.backlog ?? [];
  const goals = company?.goals ?? [];
  const roadmap = company?.roadmap ?? [];
  const cycles = company?.cycles ?? [];
  const budgetPct = report?.budget_utilization_pct ?? 0;
  const statusLabel = company?.status === "paused" ? "pausada" : company?.status === "active" ? "ativa" : company?.status ?? "sem empresa";
  const cycleSeconds = cycleFrequency === "weekly" ? 7 * 24 * 60 * 60 : 24 * 60 * 60;

  return <SidebarLayout title="Empresa" sidebar={<AppSidebar current="company" />}><main className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-950"><div className="mx-auto max-w-7xl px-6 pb-16 pt-10 lg:px-12">
	    <header className="flex flex-col gap-5 border-b border-neutral-200 pb-8 dark:border-neutral-800 md:flex-row md:items-end md:justify-between"><div><div className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-[0.14em] text-violet-600"><BuildingOffice2Icon className="h-4 w-4" />Company OS</div><h1 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">Crie e opere sua empresa</h1><p className="mt-3 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">Estratégia, produto, engenharia, marketing, vendas, suporte e operações em um tenant com ciclos autônomos, métricas, approvals e pausa segura.</p></div><button type="button" onClick={() => void refresh()} className="inline-flex items-center gap-2 rounded-xl border border-neutral-300 px-3 py-2 text-xs font-medium dark:border-neutral-700"><ArrowPathIcon className="h-4 w-4" />Atualizar</button></header>
	    <div className="mt-7">
	      <CompanySupervisorPanel />
	    </div>
	    {notice && <div role="status" aria-live="polite" className="mt-5 rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-xs text-violet-800 dark:border-violet-900/50 dark:bg-violet-950/20 dark:text-violet-200">{notice}</div>}
    {!loading && !companies.length && <Card title="Comece com uma empresa"><div className="mt-4 grid gap-3 md:grid-cols-2"><input className={inputClass} value={name} onChange={(event) => setName(event.target.value)} placeholder="Nome da empresa" aria-label="Nome da empresa" /><input className={inputClass} value={businessModel} onChange={(event) => setBusinessModel(event.target.value)} placeholder="Modelo de negócio" aria-label="Modelo de negócio" /><textarea className={areaClass} value={mission} onChange={(event) => setMission(event.target.value)} placeholder="Missão e resultado que a empresa entrega" aria-label="Missão" /><textarea className={areaClass} value={positioning} onChange={(event) => setPositioning(event.target.value)} placeholder="Posicionamento e diferencial" aria-label="Posicionamento" /><input className={inputClass} value={audience} onChange={(event) => setAudience(event.target.value)} placeholder="Público-alvo" aria-label="Público-alvo" /><input className={inputClass} value={offer} onChange={(event) => setOffer(event.target.value)} placeholder="Oferta principal" aria-label="Oferta" /><input className={inputClass} type="number" min="0" value={budget} onChange={(event) => setBudget(event.target.value)} placeholder="Limite mensal em BRL" aria-label="Limite mensal" /></div><button type="button" disabled={busy || !name.trim()} onClick={() => void create()} className="mt-4 inline-flex items-center gap-2 rounded-xl bg-neutral-950 px-4 py-2.5 text-sm font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950"><PlusIcon className="h-4 w-4" />Criar empresa e tenant</button></Card>}
    {loading && <div className="mt-8 rounded-2xl border border-dashed border-neutral-300 px-6 py-12 text-center text-sm text-neutral-500 dark:border-neutral-700">Carregando Company OS…</div>}
    {!loading && company && <>
	      <Card title="Tel-Agent — canal textual operacional" className="mt-7 border-violet-200 shadow-sm dark:border-violet-900/60"><div className="mt-3 flex flex-wrap items-center justify-between gap-2"><p className="max-w-3xl text-xs leading-5 text-neutral-500 dark:text-neutral-400">Comande esta empresa por texto e receba o retorno persistido no tenant. As operações são allowlisted e locais; telefonia, WhatsApp e mensagens externas permanecem não configurados até uma conexão homologada.</p><span className="rounded-full bg-violet-50 px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">tel-agent.text · local</span></div><div className="mt-4 grid gap-3 lg:grid-cols-[0.8fr_1.2fr] lg:items-start"><div className="space-y-2"><select className={inputClass} value={telAgentOperation} onChange={(event) => setTelAgentOperation(event.target.value as typeof telAgentOperation)} aria-label="Operação Tel-Agent"><option value="report.read">Ler relatório operacional</option><option value="backlog.create">Criar item no backlog</option><option value="campaign.draft">Criar rascunho de campanha</option></select>{telAgentOperation !== "report.read" && <input className={inputClass} value={telAgentTitle} onChange={(event) => setTelAgentTitle(event.target.value)} placeholder="Título da operação" aria-label="Título da operação" />}{telAgentOperation === "backlog.create" && <input className={inputClass} type="number" min="1" max="1000" value={telAgentPriority} onChange={(event) => setTelAgentPriority(event.target.value)} placeholder="Prioridade" aria-label="Prioridade Tel-Agent" />}{telAgentOperation === "campaign.draft" && <textarea className={areaClass} value={telAgentDescription} onChange={(event) => setTelAgentDescription(event.target.value)} placeholder="Objetivo do rascunho (não publica)" aria-label="Objetivo da campanha" />}</div><div className="space-y-2"><textarea className={areaClass} value={telAgentMessage} onChange={(event) => setTelAgentMessage(event.target.value)} maxLength={2048} placeholder="Ex.: crie uma tarefa para revisar o onboarding" aria-label="Mensagem Tel-Agent" /><div className="flex items-center justify-between gap-3"><span className="text-[11px] text-neutral-400">Histórico limitado às últimas 100 trocas; entradas são redigidas antes da persistência.</span><button type="button" disabled={busy || !telAgentMessage.trim() || (telAgentOperation !== "report.read" && !telAgentTitle.trim())} onClick={() => void sendTelAgent()} className="rounded-xl bg-neutral-950 px-4 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Executar operação</button></div></div></div>{telAgentReply && <div role="status" aria-live="polite" className="mt-4 rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/20 dark:text-emerald-200">{telAgentReply}</div>}<div className="mt-4 space-y-2">{(telAgentHistory || []).slice(-4).reverse().map((entry) => <div key={entry.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex flex-wrap items-center justify-between gap-2 text-[11px] text-neutral-400"><span>{entry.operation} · {entry.status}</span><time dateTime={entry.created_at}>{new Date(entry.created_at).toLocaleString("pt-BR")}</time></div><p className="mt-1 text-xs text-neutral-700 dark:text-neutral-300">{entry.reply}</p></div>)}</div></Card>
		      <CompanyGrowthPanel company={company} onCompanyChange={setCompany} />
		      <CompanyOperationsPanel company={company} onCompanyChange={setCompany} />
		      <CompanyApprovalQueue company={company} onCompanyChange={setCompany} />
	      <div className="mt-7 flex flex-wrap items-center gap-2 rounded-2xl border border-neutral-200/80 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-900">{companies.map((item) => <button key={item.id} type="button" onClick={() => void refresh(item.id)} className={`rounded-xl px-3 py-2 text-xs font-medium ${item.id === company.id ? "bg-neutral-950 text-white dark:bg-white dark:text-neutral-950" : "text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"}`}>{item.name}</button>)}<span className="ml-auto text-[11px] text-neutral-400">tenant {company.organization_id}</span></div>
      <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-6"><Stat label="Estado" value={statusLabel} tone={company.status === "paused" ? "amber" : "green"} /><Stat label="Backlog aberto" value={String(report?.open_backlog ?? backlog.length)} /><Stat label="Metas em dia" value={String(report?.goals_on_track ?? 0)} tone="green" /><Stat label="Ciclos ativos" value={String(report?.enabled_cycles ?? cycles.length)} /><Stat label="Budget usado" value={`${budgetPct.toFixed(1)}%`} tone={budgetPct > 80 ? "amber" : "neutral"} /><Stat label="Anomalias" value={String(company.risk.anomaly_count)} tone={company.risk.anomaly_count ? "amber" : "green"} /></div>
      <div className="mt-5 grid gap-5 xl:grid-cols-[1.25fr_0.75fr]"><div className="space-y-5">
        <Card title="Identidade, posicionamento e modelo"><div className="mt-4 grid gap-3 md:grid-cols-2"><input className={inputClass} defaultValue={company.name} aria-label="Nome" onBlur={(event) => { if (event.currentTarget.value !== company.name) void run(() => updateCompany(company.id, { name: event.currentTarget.value }), "Nome atualizado."); }} /><textarea className={areaClass} defaultValue={company.mission} aria-label="Missão" placeholder="Missão" onBlur={(event) => { if (event.currentTarget.value !== (company.mission ?? "")) void run(() => updateCompany(company.id, { mission: event.currentTarget.value }), "Missão atualizada."); }} /><textarea className={areaClass} defaultValue={company.positioning} aria-label="Posicionamento" placeholder="Posicionamento" onBlur={(event) => { if (event.currentTarget.value !== (company.positioning ?? "")) void run(() => updateCompany(company.id, { positioning: event.currentTarget.value }), "Posicionamento atualizado."); }} /><input className={inputClass} defaultValue={company.business_model} aria-label="Modelo de negócio" placeholder="Modelo de negócio" onBlur={(event) => { if (event.currentTarget.value !== (company.business_model ?? "")) void run(() => updateCompany(company.id, { business_model: event.currentTarget.value }), "Modelo de negócio atualizado."); }} /></div></Card>
        <Card title="Departamentos virtuais"><div className="mt-4 grid gap-2 md:grid-cols-2">{departments.map((department) => <div key={department.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex items-center justify-between"><span className="text-sm font-medium text-neutral-900 dark:text-white">{department.name}</span><span className="rounded-full bg-violet-50 px-2 py-1 text-[10px] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">{department.autonomy}</span></div><p className="mt-2 text-xs leading-5 text-neutral-500">{department.mandate}</p></div>)}</div></Card>
        <Card title="Roadmap estratégico"><div className="mt-4 flex gap-2"><input className={inputClass} value={roadmapTitle} onChange={(event) => setRoadmapTitle(event.target.value)} placeholder="Próximo resultado estratégico" aria-label="Item de roadmap" /><button type="button" disabled={!roadmapTitle.trim() || busy} onClick={() => void add(() => addCompanyRoadmap(company.id, { title: roadmapTitle, owner_department: "product", priority: 20 }), "Item adicionado ao roadmap.", () => setRoadmapTitle(""))} className="rounded-xl bg-neutral-950 px-3 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Adicionar</button></div><div className="mt-4 space-y-2">{roadmap.length ? roadmap.map((item) => <div key={item.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex justify-between gap-3 text-sm"><span>{item.title}</span><span className="text-xs text-neutral-400">P{item.priority} · {item.status}</span></div></div>) : <p className="text-xs text-neutral-500">Nenhum marco definido.</p>}</div></Card>
        <Card title="Metas e KPIs"><div className="mt-4 grid gap-2 md:grid-cols-[1fr_1fr_0.5fr_auto]"><input className={inputClass} value={goalTitle} onChange={(event) => setGoalTitle(event.target.value)} placeholder="Meta" aria-label="Título da meta" /><input className={inputClass} value={goalMetric} onChange={(event) => setGoalMetric(event.target.value)} placeholder="KPI: leads, receita, clientes" aria-label="Métrica" /><input className={inputClass} type="number" value={goalTarget} onChange={(event) => setGoalTarget(event.target.value)} aria-label="Alvo" /><button type="button" disabled={!goalTitle.trim() || !goalMetric.trim() || busy} onClick={() => void add(() => addCompanyGoal(company.id, { title: goalTitle, metric: goalMetric, target: Number(goalTarget), owner_department: "ceo" }), "Meta criada.", () => { setGoalTitle(""); setGoalMetric(""); })} className="rounded-xl bg-neutral-950 px-3 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Criar</button></div><div className="mt-4 space-y-2">{goals.length ? goals.map((goal) => <div key={goal.id} className="flex justify-between rounded-xl border border-neutral-200 p-3 text-sm dark:border-neutral-800"><span>{goal.title}<small className="ml-2 text-xs text-neutral-400">{goal.metric}</small></span><span className="text-xs text-neutral-500">{goal.current}/{goal.target} · {goal.status}</span></div>) : <p className="text-xs text-neutral-500">Os KPIs da empresa aparecerão aqui.</p>}</div></Card>
        <Card title="Backlog priorizado"><div className="mt-4 flex gap-2"><input className={inputClass} value={backlogTitle} onChange={(event) => setBacklogTitle(event.target.value)} placeholder="Tarefa ou experimento" aria-label="Item de backlog" /><input className="h-10 w-24 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" type="number" value={backlogPriority} onChange={(event) => setBacklogPriority(event.target.value)} aria-label="Prioridade" /><button type="button" disabled={!backlogTitle.trim() || busy} onClick={() => void add(() => addCompanyBacklog(company.id, { title: backlogTitle, priority: Number(backlogPriority), owner_department: "product", source: "manual" }), "Item adicionado e priorizado.", () => setBacklogTitle(""))} className="rounded-xl bg-neutral-950 px-3 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Adicionar</button></div><div className="mt-4 space-y-2">{backlog.length ? backlog.map((item) => <div key={item.id} className="flex justify-between rounded-xl border border-neutral-200 p-3 text-sm dark:border-neutral-800"><span>{item.title}<small className="ml-2 text-xs text-neutral-400">{item.owner_department}</small></span><span className="text-xs text-neutral-500">P{item.priority} · {item.status}</span></div>) : <p className="text-xs text-neutral-500">Backlog vazio.</p>}</div></Card>
      </div><aside className="space-y-5">
        <Card title="Ciclos autônomos"><div className="mt-4 space-y-2"><input className={inputClass} value={cycleName} onChange={(event) => setCycleName(event.target.value)} aria-label="Nome do ciclo" /><textarea className={areaClass} value={cycleObjective} onChange={(event) => setCycleObjective(event.target.value)} aria-label="Objetivo do ciclo" /><div className="flex gap-2"><select className={inputClass} value={cycleFrequency} onChange={(event) => setCycleFrequency(event.target.value)} aria-label="Frequência"><option value="daily">Diário</option><option value="weekly">Semanal</option></select><button type="button" disabled={!cycleName.trim() || !cycleObjective.trim() || busy} onClick={() => void add(() => addCompanyCycle(company.id, { name: cycleName, objective: cycleObjective, frequency: cycleFrequency, interval_seconds: cycleSeconds }), "Ciclo criado e ligado ao scheduler.", () => undefined)} className="rounded-xl bg-neutral-950 px-3 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Criar ciclo</button></div></div><div className="mt-4 space-y-2">{cycles.map((cycle) => <div key={cycle.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex justify-between text-sm"><span>{cycle.name}</span><span className="text-xs text-emerald-600">{cycle.enabled ? "ativo" : "pausado"}</span></div><p className="mt-1 text-xs text-neutral-500">{cycle.objective}</p></div>)}</div></Card>
        <Card title="Segurança, budget e pausa"><div className="mt-4 flex items-center gap-2 text-xs"><ShieldCheckIcon className="h-4 w-4 text-emerald-500" />Ações de gasto, anúncio, contrato e mensagem exigem approval.</div><div className="mt-4 grid gap-2"><input className={inputClass} type="number" min="0" value={spendAmount} onChange={(event) => setSpendAmount(event.target.value)} placeholder="Valor em centavos" aria-label="Valor de gasto" /><select className={inputClass} value={spendCategory} onChange={(event) => setSpendCategory(event.target.value)} aria-label="Categoria"><option value="ads">Anúncios</option><option value="contract">Contrato</option><option value="infrastructure">Infraestrutura</option><option value="sales">Vendas</option></select><button type="button" disabled={!spendAmount || busy} onClick={() => void run(() => recordCompanySpend(company.id, spendCategory, Number(spendAmount)), "Gasto solicitado; decision pendente antes de atualizar o budget.")} className="rounded-xl border border-neutral-300 px-3 py-2 text-xs dark:border-neutral-700">Registrar gasto</button></div><div className="mt-4 flex flex-wrap gap-2"><button type="button" onClick={() => void run(() => company.status === "paused" ? resumeCompany(company.id) : pauseCompany(company.id, "Pausa manual pelo Company OS"), company.status === "paused" ? "Empresa retomada." : "Empresa pausada com segurança.")} className="inline-flex items-center gap-2 rounded-xl bg-neutral-950 px-3 py-2 text-xs text-white dark:bg-white dark:text-neutral-950">{company.status === "paused" ? <PlayCircleIcon className="h-4 w-4" /> : <PauseCircleIcon className="h-4 w-4" />}{company.status === "paused" ? "Retomar" : "Pausar"}</button><button type="button" onClick={() => void run(() => recordCompanyAnomaly(company.id, "high", "anomalia manual para teste de guardrail"), "Anomalia registrada; a política pode pausar a empresa.")} className="inline-flex items-center gap-2 rounded-xl border border-amber-300 px-3 py-2 text-xs text-amber-700 dark:border-amber-800 dark:text-amber-300"><ExclamationTriangleIcon className="h-4 w-4" />Simular anomalia</button></div>{company.risk.pause_reason && <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">{company.risk.pause_reason}</p>}</Card>
        <Card title="Sinal operacional"><div className="mt-4 space-y-3 text-xs text-neutral-500"><div className="flex justify-between"><span>Budget mensal</span><strong className="text-neutral-900 dark:text-white">{company.budget.currency} {(company.budget.monthly_limit_cents / 100).toFixed(2)}</strong></div><div className="flex justify-between"><span>Consumido</span><strong className="text-neutral-900 dark:text-white">{(company.budget.spent_cents / 100).toFixed(2)}</strong></div><div className="h-2 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"><div className={`h-full ${budgetPct > 80 ? "bg-amber-500" : "bg-emerald-500"}`} style={{ width: `${Math.min(100, budgetPct)}%` }} /></div><p className="flex items-center gap-2"><CheckCircleIcon className="h-4 w-4 text-emerald-500" />Persistência local, tenant e auditoria server-side.</p></div></Card>
      </aside></div>
    </>}
  </div></main></SidebarLayout>;
}
