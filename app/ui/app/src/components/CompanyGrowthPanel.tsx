import { useCallback, useEffect, useState } from "react";
import {
  addCompanyAffiliateProgram,
  addCompanyCampaign,
  addCompanyProduct,
  approveCompanyAffiliateProgram,
  approveCompanyCampaign,
  approveCompanyOrder,
  createCompanyOrder,
  fulfillCompanyOrder,
  getCompanyGrowthReport,
  launchCompanyCampaign,
} from "@/lib/agenticClient";
import { CompanyCampaignSchedulingPanel } from "./CompanyCampaignSchedulingPanel";
import type { AgentCompany, CompanyGrowthReport } from "@/lib/agenticClient";

const field = "h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
const button = "rounded-xl bg-neutral-950 px-3 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950";

export function CompanyGrowthPanel({ company, onCompanyChange }: { company: AgentCompany; onCompanyChange: (company: AgentCompany) => void }) {
  const [report, setReport] = useState<CompanyGrowthReport | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [campaignName, setCampaignName] = useState("");
  const [campaignObjective, setCampaignObjective] = useState("");
  const [campaignChannel, setCampaignChannel] = useState("social");
  const [programName, setProgramName] = useState("");
  const [programNetwork, setProgramNetwork] = useState("sandbox-network");
  const [sku, setSku] = useState("");
  const [productName, setProductName] = useState("");
  const [supplier, setSupplier] = useState("Fornecedor sandbox");
  const [price, setPrice] = useState("1200");
  const [inventory, setInventory] = useState("10");
  const [customer, setCustomer] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try { setReport(await getCompanyGrowthReport(company.id)); } catch (error) { setNotice(error instanceof Error ? error.message : "Growth OS indisponível."); }
  }, [company.id]);
  useEffect(() => { void refresh(); }, [refresh]);

  const run = async (action: () => Promise<AgentCompany>, message: string) => {
    setBusy(true);
    try { const updated = await action(); onCompanyChange(updated); setNotice(message); await refresh(); }
    catch (error) { setNotice(error instanceof Error ? error.message : "A operação de Growth OS falhou."); }
    finally { setBusy(false); }
  };

  const campaigns = company.campaigns ?? [];
  const programs = company.affiliate_programs ?? [];
  const products = company.products ?? [];
	const orders = company.orders ?? [];
	const selectedProduct = products[0];
	const approvalFor = (resourceType: string, resourceID: string) => company.approvals?.find((approval) => approval.resource_type === resourceType && approval.resource_id === resourceID && approval.status === "pending");

  return <section className="mt-5 rounded-2xl border border-violet-200/80 bg-violet-50/40 p-5 dark:border-violet-900/60 dark:bg-violet-950/10">
    <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between"><div><div className="text-xs font-medium uppercase tracking-[0.14em] text-violet-700 dark:text-violet-300">Growth OS</div><h2 className="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">Marketing, afiliados e dropshipping controlados</h2><p className="mt-1 max-w-3xl text-xs leading-5 text-neutral-600 dark:text-neutral-400">Tudo começa em sandbox local. Publicação, anúncio, parceiro e fulfillment exigem approval e connector autorizado.</p></div><button type="button" className="rounded-xl border border-violet-300 px-3 py-2 text-xs dark:border-violet-800" onClick={() => void refresh()}>Atualizar growth</button></div>
    {notice && <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-violet-200 bg-white px-3 py-2 text-xs text-violet-800 dark:border-violet-900 dark:bg-neutral-900 dark:text-violet-200">{notice}</div>}
    <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-6">{[["Campanhas", report?.campaigns_total ?? campaigns.length], ["Ativas", report?.campaigns_active ?? 0], ["Programas", report?.affiliate_programs ?? programs.length], ["Conversões", report?.affiliate_conversions ?? 0], ["Produtos", report?.products ?? products.length], ["Pedidos concluídos", report?.fulfilled_orders ?? 0]].map(([label, value]) => <div key={label} className="rounded-xl border border-violet-200 bg-white p-3 dark:border-violet-900/60 dark:bg-neutral-900"><div className="text-[10px] uppercase tracking-wider text-neutral-400">{label}</div><div className="mt-1 text-lg font-semibold text-neutral-900 dark:text-white">{value}</div></div>)}</div>
    <div className="mt-4 grid gap-4 xl:grid-cols-3">
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">Campanha</h3><div className="mt-3 grid gap-2"><input className={field} value={campaignName} onChange={(event) => setCampaignName(event.target.value)} placeholder="Nome da campanha" aria-label="Nome da campanha" /><input className={field} value={campaignObjective} onChange={(event) => setCampaignObjective(event.target.value)} placeholder="Objetivo: gerar leads" aria-label="Objetivo da campanha" /><select className={field} value={campaignChannel} onChange={(event) => setCampaignChannel(event.target.value)} aria-label="Canal da campanha"><option value="social">Redes sociais</option><option value="email">E-mail</option><option value="affiliate">Afiliados</option><option value="content">Conteúdo</option></select><button type="button" className={button} disabled={busy || !campaignName.trim() || !campaignObjective.trim()} onClick={() => void run(() => addCompanyCampaign(company.id, { name: campaignName, objective: campaignObjective, channel: campaignChannel, daily_budget_cents: 0 }), "Campanha criada como rascunho; approval pendente.")}>Criar campanha</button></div><div className="mt-3 space-y-2">{campaigns.map((campaign) => <div key={campaign.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{campaign.name}</span><span>{campaign.status}</span></div><div className="mt-2 flex gap-2">{!campaign.approved && <button type="button" className="rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyCampaign(company.id, campaign.id, approvalFor("campaign", campaign.id)?.nonce ?? ""), "Campanha aprovada no sandbox.")}>Aprovar</button>}{campaign.approved && campaign.status !== "active" && <button type="button" className="rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => launchCompanyCampaign(company.id, campaign.id), "Campanha iniciada no sandbox.")}>Iniciar</button>}</div></div>)}</div></div>
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">Programa de afiliados</h3><div className="mt-3 grid gap-2"><input className={field} value={programName} onChange={(event) => setProgramName(event.target.value)} placeholder="Nome do programa" aria-label="Nome do programa" /><input className={field} value={programNetwork} onChange={(event) => setProgramNetwork(event.target.value)} placeholder="Rede ou parceiro" aria-label="Rede de afiliados" /><button type="button" className={button} disabled={busy || !programName.trim()} onClick={() => void run(() => addCompanyAffiliateProgram(company.id, { name: programName, network: programNetwork, commission_bps: 1000 }), "Programa criado; aprovação pendente.")}>Criar programa</button></div><div className="mt-3 space-y-2">{programs.map((program) => <div key={program.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{program.name}</span><span>{program.status}</span></div>{!program.approved && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyAffiliateProgram(company.id, program.id, approvalFor("affiliate_program", program.id)?.nonce ?? ""), "Programa de afiliados aprovado no sandbox.")}>Aprovar programa</button>}</div>)}</div></div>
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">Produto e pedido</h3><div className="mt-3 grid gap-2"><input className={field} value={sku} onChange={(event) => setSku(event.target.value)} placeholder="SKU" aria-label="SKU" /><input className={field} value={productName} onChange={(event) => setProductName(event.target.value)} placeholder="Nome do produto" aria-label="Nome do produto" /><input className={field} value={supplier} onChange={(event) => setSupplier(event.target.value)} placeholder="Fornecedor sandbox" aria-label="Fornecedor" /><div className="grid grid-cols-2 gap-2"><input className={field} type="number" value={price} onChange={(event) => setPrice(event.target.value)} aria-label="Preço em centavos" placeholder="Preço cents" /><input className={field} type="number" value={inventory} onChange={(event) => setInventory(event.target.value)} aria-label="Estoque" placeholder="Estoque" /></div><button type="button" className={button} disabled={busy || !sku.trim() || !productName.trim()} onClick={() => void run(() => addCompanyProduct(company.id, { sku, name: productName, supplier, cost_cents: Math.round(Number(price) / 2), price_cents: Number(price), inventory: Number(inventory) }), "Produto criado no catálogo sandbox.")}>Adicionar produto</button>{selectedProduct && <><input className={field} value={customer} onChange={(event) => setCustomer(event.target.value)} placeholder="Referência do cliente" aria-label="Referência do cliente" /><input className={field} type="number" min="1" value={quantity} onChange={(event) => setQuantity(event.target.value)} aria-label="Quantidade" /><button type="button" className="rounded-xl border border-neutral-300 px-3 py-2 text-xs dark:border-neutral-700" disabled={busy || !customer.trim()} onClick={() => void run(() => createCompanyOrder(company.id, { product_id: selectedProduct.id, customer_ref: customer, quantity: Number(quantity) }), "Pedido criado; aprovação pendente.")}>Criar pedido</button></>}</div><div className="mt-3 space-y-2">{orders.map((order) => <div key={order.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{order.customer_ref}</span><span>{order.status}</span></div>{!order.approved && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyOrder(company.id, order.id, approvalFor("order", order.id)?.nonce ?? ""), "Pedido aprovado no sandbox.")}>Aprovar pedido</button>}{order.approved && order.status !== "fulfilled" && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => fulfillCompanyOrder(company.id, order.id, `SANDBOX-${order.id.slice(-8)}`), "Pedido fulfillado no sandbox.")}>Fulfill sandbox</button>}</div>)}</div></div>
    </div>
    <CompanyCampaignSchedulingPanel company={company} onCompanyChange={onCompanyChange} />
  </section>;
}
