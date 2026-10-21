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
import { useT } from "@/i18n";

const field = "h-10 w-full rounded-xl border border-neutral-300 bg-transparent px-3 text-sm outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
const button = "rounded-xl bg-neutral-950 px-3 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-950";

export function CompanyGrowthPanel({ company, onCompanyChange }: { company: AgentCompany; onCompanyChange: (company: AgentCompany) => void }) {
  const { t } = useT();
  const [report, setReport] = useState<CompanyGrowthReport | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [campaignName, setCampaignName] = useState("");
  const [campaignObjective, setCampaignObjective] = useState("");
  const [campaignChannel, setCampaignChannel] = useState("social");
  const [programName, setProgramName] = useState("");
  const [programNetwork, setProgramNetwork] = useState("sandbox-network");
  const [sku, setSku] = useState("");
  const [productName, setProductName] = useState("");
  // Valor INICIAL do campo, nao rotulo: vai no payload `supplier` de addCompanyProduct,
  // logo nao pode mudar com o idioma da interface -- um utilizador em ingles passaria a
  // gravar "Supplier sandbox" nos dados do produto. O rotulo visivel e' o `placeholder`
  // do input, que continua a usar a chave traduzida.
  const [supplier, setSupplier] = useState("Fornecedor sandbox");
  const [price, setPrice] = useState("1200");
  const [inventory, setInventory] = useState("10");
  const [customer, setCustomer] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try { setReport(await getCompanyGrowthReport(company.id)); } catch (error) { setNotice(error instanceof Error ? error.message : t("companyGrowth.error.report")); }
  }, [company.id, t]);
  useEffect(() => { void refresh(); }, [refresh]);

  const run = async (action: () => Promise<AgentCompany>, message: string) => {
    setBusy(true);
    try { const updated = await action(); onCompanyChange(updated); setNotice(message); await refresh(); }
    catch (error) { setNotice(error instanceof Error ? error.message : t("companyGrowth.error.action")); }
    finally { setBusy(false); }
  };

  const campaigns = company.campaigns ?? [];
  const programs = company.affiliate_programs ?? [];
  const products = company.products ?? [];
	const orders = company.orders ?? [];
	const selectedProduct = products[0];
	const approvalFor = (resourceType: string, resourceID: string) => company.approvals?.find((approval) => approval.resource_type === resourceType && approval.resource_id === resourceID && approval.status === "pending");

  return <section className="mt-5 rounded-2xl border border-violet-200/80 bg-violet-50/40 p-5 dark:border-violet-900/60 dark:bg-violet-950/10">
    <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between"><div><div className="text-xs font-medium uppercase tracking-[0.14em] text-violet-700 dark:text-violet-300">Growth OS</div><h2 className="mt-1 text-lg font-semibold text-neutral-950 dark:text-white">{t("companyGrowth.title")}</h2><p className="mt-1 max-w-3xl text-xs leading-5 text-neutral-600 dark:text-neutral-400">{t("companyGrowth.intro")}</p></div><button type="button" className="rounded-xl border border-violet-300 px-3 py-2 text-xs dark:border-violet-800" onClick={() => void refresh()}>{t("companyGrowth.refresh")}</button></div>
    {notice && <div role="status" aria-live="polite" className="mt-3 rounded-xl border border-violet-200 bg-white px-3 py-2 text-xs text-violet-800 dark:border-violet-900 dark:bg-neutral-900 dark:text-violet-200">{notice}</div>}
    <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-6">{[[t("companyGrowth.stats.campaigns"), report?.campaigns_total ?? campaigns.length], [t("companyGrowth.stats.active"), report?.campaigns_active ?? 0], [t("companyGrowth.stats.programs"), report?.affiliate_programs ?? programs.length], [t("companyGrowth.stats.conversions"), report?.affiliate_conversions ?? 0], [t("companyGrowth.stats.products"), report?.products ?? products.length], [t("companyGrowth.stats.ordersDone"), report?.fulfilled_orders ?? 0]].map(([label, value]) => <div key={label} className="rounded-xl border border-violet-200 bg-white p-3 dark:border-violet-900/60 dark:bg-neutral-900"><div className="text-[10px] uppercase tracking-wider text-neutral-400">{label}</div><div className="mt-1 text-lg font-semibold text-neutral-900 dark:text-white">{value}</div></div>)}</div>
    <div className="mt-4 grid gap-4 xl:grid-cols-3">
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">{t("companyGrowth.campaign.title")}</h3><div className="mt-3 grid gap-2"><input className={field} value={campaignName} onChange={(event) => setCampaignName(event.target.value)} placeholder={t("companyGrowth.campaign.name")} aria-label={t("companyGrowth.campaign.name")} /><input className={field} value={campaignObjective} onChange={(event) => setCampaignObjective(event.target.value)} placeholder={t("companyGrowth.campaign.objectivePlaceholder")} aria-label={t("companyGrowth.campaign.objective")} /><select className={field} value={campaignChannel} onChange={(event) => setCampaignChannel(event.target.value)} aria-label={t("companyGrowth.campaign.channel")}><option value="social">{t("companyGrowth.campaign.channelSocial")}</option><option value="email">{t("companyGrowth.campaign.channelEmail")}</option><option value="affiliate">{t("companyGrowth.campaign.channelAffiliate")}</option><option value="content">{t("companyGrowth.campaign.channelContent")}</option></select><button type="button" className={button} disabled={busy || !campaignName.trim() || !campaignObjective.trim()} onClick={() => void run(() => addCompanyCampaign(company.id, { name: campaignName, objective: campaignObjective, channel: campaignChannel, daily_budget_cents: 0 }), t("companyGrowth.campaign.created"))}>{t("companyGrowth.campaign.create")}</button></div><div className="mt-3 space-y-2">{campaigns.map((campaign) => <div key={campaign.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{campaign.name}</span><span>{campaign.status}</span></div><div className="mt-2 flex gap-2">{!campaign.approved && <button type="button" className="rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyCampaign(company.id, campaign.id, approvalFor("campaign", campaign.id)?.nonce ?? ""), t("companyGrowth.campaign.approved"))}>{t("companyGrowth.campaign.approve")}</button>}{campaign.approved && campaign.status !== "active" && <button type="button" className="rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => launchCompanyCampaign(company.id, campaign.id), t("companyGrowth.campaign.launched"))}>{t("companyGrowth.campaign.launch")}</button>}</div></div>)}</div></div>
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">{t("companyGrowth.program.title")}</h3><div className="mt-3 grid gap-2"><input className={field} value={programName} onChange={(event) => setProgramName(event.target.value)} placeholder={t("companyGrowth.program.name")} aria-label={t("companyGrowth.program.name")} /><input className={field} value={programNetwork} onChange={(event) => setProgramNetwork(event.target.value)} placeholder={t("companyGrowth.program.networkPlaceholder")} aria-label={t("companyGrowth.program.network")} /><button type="button" className={button} disabled={busy || !programName.trim()} onClick={() => void run(() => addCompanyAffiliateProgram(company.id, { name: programName, network: programNetwork, commission_bps: 1000 }), t("companyGrowth.program.created"))}>{t("companyGrowth.program.create")}</button></div><div className="mt-3 space-y-2">{programs.map((program) => <div key={program.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{program.name}</span><span>{program.status}</span></div>{!program.approved && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyAffiliateProgram(company.id, program.id, approvalFor("affiliate_program", program.id)?.nonce ?? ""), t("companyGrowth.program.approved"))}>{t("companyGrowth.program.approve")}</button>}</div>)}</div></div>
      <div className="rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold">{t("companyGrowth.product.title")}</h3><div className="mt-3 grid gap-2"><input className={field} value={sku} onChange={(event) => setSku(event.target.value)} placeholder="SKU" aria-label="SKU" /><input className={field} value={productName} onChange={(event) => setProductName(event.target.value)} placeholder={t("companyGrowth.product.name")} aria-label={t("companyGrowth.product.name")} /><input className={field} value={supplier} onChange={(event) => setSupplier(event.target.value)} placeholder={t("companyGrowth.product.supplierPlaceholder")} aria-label={t("companyGrowth.product.supplier")} /><div className="grid grid-cols-2 gap-2"><input className={field} type="number" value={price} onChange={(event) => setPrice(event.target.value)} aria-label={t("companyGrowth.product.priceAria")} placeholder={t("companyGrowth.product.pricePlaceholder")} /><input className={field} type="number" value={inventory} onChange={(event) => setInventory(event.target.value)} aria-label={t("companyGrowth.product.inventory")} placeholder={t("companyGrowth.product.inventory")} /></div><button type="button" className={button} disabled={busy || !sku.trim() || !productName.trim()} onClick={() => void run(() => addCompanyProduct(company.id, { sku, name: productName, supplier, cost_cents: Math.round(Number(price) / 2), price_cents: Number(price), inventory: Number(inventory) }), t("companyGrowth.product.added"))}>{t("companyGrowth.product.add")}</button>{selectedProduct && <><input className={field} value={customer} onChange={(event) => setCustomer(event.target.value)} placeholder={t("companyGrowth.order.customer")} aria-label={t("companyGrowth.order.customer")} /><input className={field} type="number" min="1" value={quantity} onChange={(event) => setQuantity(event.target.value)} aria-label={t("companyGrowth.order.quantity")} /><button type="button" className="rounded-xl border border-neutral-300 px-3 py-2 text-xs dark:border-neutral-700" disabled={busy || !customer.trim()} onClick={() => void run(() => createCompanyOrder(company.id, { product_id: selectedProduct.id, customer_ref: customer, quantity: Number(quantity) }), t("companyGrowth.order.created"))}>{t("companyGrowth.order.create")}</button></>}</div><div className="mt-3 space-y-2">{orders.map((order) => <div key={order.id} className="rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800"><div className="flex justify-between"><span>{order.customer_ref}</span><span>{order.status}</span></div>{!order.approved && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => approveCompanyOrder(company.id, order.id, approvalFor("order", order.id)?.nonce ?? ""), t("companyGrowth.order.approved"))}>{t("companyGrowth.order.approve")}</button>}{order.approved && order.status !== "fulfilled" && <button type="button" className="mt-2 rounded-lg border px-2 py-1" disabled={busy} onClick={() => void run(() => fulfillCompanyOrder(company.id, order.id, `SANDBOX-${order.id.slice(-8)}`), t("companyGrowth.order.fulfilled"))}>{t("companyGrowth.order.fulfill")}</button>}</div>)}</div></div>
    </div>
    <CompanyCampaignSchedulingPanel company={company} onCompanyChange={onCompanyChange} />
  </section>;
}
