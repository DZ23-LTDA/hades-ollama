import { useCallback, useEffect, useState } from "react";
import {
  addCompanySocialAccount,
  approveCompanySocialDraft,
  createCompanySocialDraft,
  getCompanySocialReport,
  getGrokStatus,
  pauseCompanyAgent,
  publishCompanySocialDraft,
  resumeCompanyAgent,
} from "@/lib/agenticClient";
import type { AgentCompany, CompanySocialReport, GrokStatus } from "@/lib/agenticClient";

const inputClass = "h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
const areaClass = "min-h-24 w-full rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return <section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><h2 className="text-sm font-semibold text-neutral-950 dark:text-white">{title}</h2>{children}</section>;
}

export function CompanyOperationsPanel({ company, onCompanyChange }: { company: AgentCompany; onCompanyChange: (company: AgentCompany) => void }) {
  const [social, setSocial] = useState<CompanySocialReport | null>(null);
  const [grok, setGrok] = useState<GrokStatus | null>(null);
  const [provider, setProvider] = useState("instagram");
  const [draftTitle, setDraftTitle] = useState("");
  const [draftBody, setDraftBody] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [socialReport, grokStatus] = await Promise.all([getCompanySocialReport(company.id), getGrokStatus()]);
      setSocial(socialReport);
      setGrok(grokStatus);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Não foi possível carregar Social OS/Grok.");
    }
  }, [company.id]);
  useEffect(() => { void refresh(); }, [refresh]);

  const mutate = async (action: () => Promise<AgentCompany>, message: string) => {
    setBusy(true);
    try { onCompanyChange(await action()); setNotice(message); await refresh(); }
    catch (error) { setNotice(error instanceof Error ? error.message : "Ação não concluída."); }
    finally { setBusy(false); }
  };

	const agents = company.agents ?? [];
	const drafts = company.social_drafts ?? [];
	const approvalFor = (resourceType: string, resourceID: string) => company.approvals?.find((approval) => approval.resource_type === resourceType && approval.resource_id === resourceID && approval.status === "pending");
  return <div className="mt-5 grid gap-5 xl:grid-cols-2">
    <Card title="Agentes departamentais e supervisão">
      <p className="mt-2 text-xs leading-5 text-neutral-500">Cada agente tem objetivo, escopos de ferramentas, memória, SLA, supervisor e condição de pausa. Estados pausados não são executados pelo painel.</p>
      <div className="mt-4 space-y-2">{agents.map((agent) => <div key={agent.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex items-start justify-between gap-3"><div><div className="text-sm font-medium text-neutral-900 dark:text-white">{agent.name}</div><div className="mt-1 text-xs text-neutral-500">{agent.objective}</div></div><span className={`rounded-full px-2 py-1 text-[10px] ${agent.status === "active" ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300" : "bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300"}`}>{agent.status}</span></div><div className="mt-2 flex flex-wrap gap-2 text-[10px] text-neutral-400"><span>memória: {agent.memory_scope}</span><span>SLA: {agent.sla ?? "—"}</span><span>tools: {agent.allowed_tools?.length ?? 0}</span></div><div className="mt-3 flex gap-2">{agent.status === "active" ? <button type="button" disabled={busy} onClick={() => void mutate(() => pauseCompanyAgent(company.id, agent.id, "Pausa manual pelo Company OS"), `${agent.name} pausado.`)} className="rounded-lg border border-amber-300 px-2.5 py-1.5 text-[11px] text-amber-700 dark:border-amber-800 dark:text-amber-300">Pausar agente</button> : <button type="button" disabled={busy} onClick={() => void mutate(() => resumeCompanyAgent(company.id, agent.id), `${agent.name} retomado.`)} className="rounded-lg border border-emerald-300 px-2.5 py-1.5 text-[11px] text-emerald-700 dark:border-emerald-800 dark:text-emerald-300">Retomar agente</button>}</div></div>)}</div>
    </Card>
    <Card title="Grok Live e saúde dos provedores">
      <div className="mt-3 rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex items-center justify-between"><span className="text-xs font-medium text-neutral-900 dark:text-white">xAI / Grok Responses</span><span className={`rounded-full px-2 py-1 text-[10px] ${grok?.healthy ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300" : "bg-neutral-100 text-neutral-500 dark:bg-neutral-800"}`}>{grok?.state ?? "cataloged"}</span></div><p className="mt-2 text-xs text-neutral-500">Modelo: {grok?.model ?? "grok-4"} · credencial: {grok?.authenticated ? "configurada" : "não configurada"} · latência: {grok?.last_latency_ms ? `${grok.last_latency_ms} ms` : "—"}</p>{grok?.last_error && <p className="mt-2 text-[11px] text-amber-700 dark:text-amber-300">{grok.last_error}</p>}</div><button type="button" onClick={() => void refresh()} className="mt-3 rounded-xl border border-neutral-300 px-3 py-2 text-xs dark:border-neutral-700">Atualizar saúde</button></Card>
    <Card title="Redes sociais: rascunhos, aprovação e publicação em sandbox">
      <p className="mt-2 text-xs leading-5 text-neutral-500">O adapter cobre catálogo e workflow. OAuth e publicação externa permanecem bloqueados até a conexão real da conta e aprovação explícita.</p>
      <div className="mt-3 grid gap-2 md:grid-cols-2"><select className={inputClass} value={provider} onChange={(event) => setProvider(event.target.value)} aria-label="Provedor social"><option value="instagram">Instagram</option><option value="facebook">Facebook</option><option value="x">X</option><option value="youtube">YouTube</option><option value="tiktok">TikTok</option><option value="linkedin">LinkedIn</option></select><input className={inputClass} value={draftTitle} onChange={(event) => setDraftTitle(event.target.value)} placeholder="Título do draft" aria-label="Título do draft" /></div><textarea className={`${areaClass} mt-2`} value={draftBody} onChange={(event) => setDraftBody(event.target.value)} placeholder="Conteúdo da publicação" aria-label="Conteúdo da publicação" /><div className="mt-2 flex gap-2"><button type="button" disabled={busy || !draftTitle.trim() || !draftBody.trim()} onClick={() => void mutate(() => createCompanySocialDraft(company.id, { provider, title: draftTitle, body: draftBody, mode: "sandbox" }), "Draft social criado; approval exigido.")} className="rounded-xl bg-neutral-950 px-3 py-2 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950">Criar draft</button><button type="button" disabled={busy} onClick={() => void mutate(() => addCompanySocialAccount(company.id, { provider, name: `${provider} brand account` }), "Conta registrada como pending_oauth.")} className="rounded-xl border border-neutral-300 px-3 py-2 text-xs dark:border-neutral-700">Registrar conta/OAuth</button></div><div className="mt-4 grid grid-cols-3 gap-2"><div className="rounded-xl bg-neutral-100 p-3 text-center dark:bg-neutral-800"><div className="text-lg font-semibold">{social?.drafts ?? drafts.length}</div><div className="text-[10px] text-neutral-500">rascunhos</div></div><div className="rounded-xl bg-neutral-100 p-3 text-center dark:bg-neutral-800"><div className="text-lg font-semibold">{social?.approved_drafts ?? 0}</div><div className="text-[10px] text-neutral-500">aprovados</div></div><div className="rounded-xl bg-neutral-100 p-3 text-center dark:bg-neutral-800"><div className="text-lg font-semibold">{social?.published_sandbox ?? 0}</div><div className="text-[10px] text-neutral-500">sandbox</div></div></div><div className="mt-3 space-y-2">{drafts.map((draft) => <div key={draft.id} className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-800"><div className="flex justify-between text-xs"><span>{draft.provider} · {draft.title}</span><span className="text-neutral-400">{draft.status}</span></div><div className="mt-2 flex gap-2">{!draft.approved && <button type="button" disabled={busy} onClick={() => void mutate(() => approveCompanySocialDraft(company.id, draft.id, approvalFor("social_draft", draft.id)?.nonce ?? ""), "Draft aprovado.")} className="rounded-lg border border-emerald-300 px-2 py-1 text-[10px] text-emerald-700 dark:border-emerald-800 dark:text-emerald-300">Aprovar</button>}{draft.approved && draft.status !== "published_sandbox" && <button type="button" disabled={busy} onClick={() => void mutate(() => publishCompanySocialDraft(company.id, draft.id), "Publicado em sandbox; nenhum canal externo foi acionado.")} className="rounded-lg border border-violet-300 px-2 py-1 text-[10px] text-violet-700 dark:border-violet-800 dark:text-violet-300">Publicar sandbox</button>}</div></div>)}</div>{notice && <div role="status" className="mt-3 rounded-lg bg-violet-50 px-3 py-2 text-xs text-violet-800 dark:bg-violet-950/30 dark:text-violet-200">{notice}</div>}</Card>
  </div>;
}
