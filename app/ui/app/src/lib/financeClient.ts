// Painel financeiro -- a leitura de `GET /finance/report`.
//
// `finance` tinha 4 rotas: 3 sao POST (receita, custo, reconciliar) e esta e' a unica
// leitura. O painel entrega-so esta: nao inventa as escritas.
//
// PORQUE ESTE PAINEL NAO E' UM PAINEL DE "RECEITA"
//
// `FinanceReport` tem quatro numeros de dinheiro que nao sao a mesma coisa:
//
//   income_cents        -- o que entrou, contabilizado.
//   receivable_cents    -- o que AINDA NAO ENTROU. O proprio tipo:
//                          "ReceivableCents e' o que ainda NAO entrou (pedidos aguardando pagamento)."
//   unposted_revenue    -- receita EXISTENTE e ainda nao contabilizada.
//   refund_due_cents    -- dinheiro que se deve DEVOLVER.
//
// Somar os quatro e chamar-lhe "receita" e' a mentira mais facil deste ecra. O painel
// separa-os e nomeia cada um.
//
// E HA' UMA QUINTA ARMADILHA: `reconciliation_coverage_pct`.
//
//   net_cents = income - expense e' aritmetica correcta. Mas se a cobertura de
//   reconciliacao for 0, os lancamentos nao foram conferidos contra o extracto, e o
//   `net_cents` e' a soma dos lancamentos, nao o resultado conciliado.
//
//   *Uma receita liquida calculada sobre 0% de reconciliacao nao e' uma receita liquida:
//   e' uma soma.*
//
// O painel mostra a cobertura ao lado do liquido, e diz quando o liquido e' PROVISORIO.
import { centavosParaReais } from "@/lib/commerceClient";
import { agentFetch } from "@/lib/agenticClient";

export interface ResumoDoLivro {
  company_id: string;
  currency: string;
  from?: string;
  to?: string;
  income_cents: number;
  expense_cents: number;
  net_cents: number;
  entry_count: number;
  reconciled_count: number;
  unreconciled_count: number;
  unreconciled_income_cents: number;
  unreconciled_expense_cents: number;
  reconciliation_coverage_pct: number;
}

export interface Recebivel {
  order_id: string;
  order_number: string;
  contact_id?: string;
  total_cents: number;
  currency: string;
  created_at: string;
}

export interface Divergencia {
  order_id: string;
  order_number: string;
  kind: string;
  amount_cents: number;
  detail: string;
}

export interface RelatorioFinanceiro {
  company_id: string;
  from?: string;
  to?: string;
  ledger: ResumoDoLivro;
  /** O que ainda NAO entrou. Nao e' receita. */
  receivable_cents: number;
  receivables?: Recebivel[];
  unposted_revenue_cents: number;
  refund_due_cents: number;
  divergences?: Divergencia[];
  orders_paid: number;
  orders_fulfilled: number;
  orders_pending: number;
  orders_cancelled: number;
}

/** A empresa financeira nao esta' configurada (`ErrFinanceCompanyMissing`). */
export class FinanceiroAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FinanceiroAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/company|empresa|n[aã]o configurad|n[aã]o est[aá] dispon[íi]vel|503/i.test(mensagem)) {
    throw new FinanceiroAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Diz se o liquido e' final ou provisorio, e porquê.
 *
 * 0% de reconciliacao significa que nenhum lancamento foi conferido: o liquido e' a soma
 * dos lancamentos do livro, nao o resultado conciliado.
 */
export function descreverQualidadeDoLiquido(resumo: ResumoDoLivro | undefined): string {
  if (!resumo) return "sem informacao de reconciliacao";
  const cobertura = Number.isFinite(resumo.reconciliation_coverage_pct)
    ? resumo.reconciliation_coverage_pct
    : 0;
  if (cobertura <= 0) {
    return "liquido PROVISORIO -- 0% de reconciliacao: e' a soma dos lancamentos, nao o resultado conciliado";
  }
  if (cobertura < 100) {
    return `liquido provisorio -- ${formatarPercentagem(cobertura)} dos lancamentos conferidos`;
  }
  return "liquido conciliado -- todos os lancamentos foram conferidos";
}

/** Percentagem legivel, sem inventar casas. */
export function formatarPercentagem(valor: number | undefined): string {
  const n = Number.isFinite(valor) ? Math.max(0, valor as number) : 0;
  const arredondado = Math.round(n * 10) / 10;
  return `${arredondado.toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`;
}

/** O que ainda nao entrou, dito como tal. */
export function descreverRecebivel(centavos: number | undefined, quantidade: number): string {
  const valor = centavosParaReais(Math.max(0, centavos ?? 0));
  const n = Math.max(0, quantidade);
  if (n === 0) return "nada a receber";
  return `${valor} a receber de ${n} pedido(s) -- ainda nao entrou`;
}

/** Receita existente mas nao contabilizada. */
export function descreverNaoContabilizado(centavos: number | undefined): string {
  const valor = Math.max(0, centavos ?? 0);
  if (valor === 0) return "nenhuma receita por contabilizar";
  return `${centavosParaReais(valor)} de receita existente e ainda nao contabilizada`;
}

/** Reembolso devido. Zero nao e' "sem reembolso": e' um numero que se le'. */
export function descreverReembolso(centavos: number | undefined): string {
  const valor = Math.max(0, centavos ?? 0);
  if (valor === 0) return "nenhum reembolso devido";
  return `${centavosParaReais(valor)} a devolver`;
}

/** Uma divergencia: tipo, valor e detalhe. */
export function descreverDivergencia(d: Divergencia): string {
  const tipo = (d.kind ?? "").trim() === "" ? "tipo nao declarado" : d.kind.trim();
  const detalhe = (d.detail ?? "").trim();
  const valor = centavosParaReais(d.amount_cents ?? 0);
  return detalhe === ""
    ? `${d.order_number || d.order_id}: ${tipo} de ${valor}`
    : `${d.order_number || d.order_id}: ${tipo} de ${valor} -- ${detalhe}`;
}

export async function obterRelatorioFinanceiro(init?: RequestInit): Promise<RelatorioFinanceiro> {
  try {
    return await agentFetch<RelatorioFinanceiro>("/api/agent/v1/finance/report", init);
  } catch (causa) {
    traduzir(causa);
  }
}
