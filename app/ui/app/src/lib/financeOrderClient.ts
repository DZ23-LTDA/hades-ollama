// Lançamentos financeiros de uma ordem -- `POST /finance/orders/:id/{revenue,cost,reconcile}`.
//
// TRES ESCRITAS, TRES REGRAS DIFERENTES -- E A DA RECEITA E' A QUE MAIS IMPORTA
//
// O comentario do handler da receita:
//
//     "postOrderRevenue lanca a receita de um pedido PAGO. Pedido aguardando pagamento e' recusado
//      (409): lancar receita dele seria receita ficticia."
//
// **Receita ficticia e' o pior numero que um sistema financeiro pode produzir**: nao e' um erro de
// arredondamento, e' um valor que entra no relatorio e que ninguem vai cobrar. O servidor recusa; a
// rota nao "tenta na mesma".
//
//   *Um painel que mostrasse a receita como "lancada" para um pedido nao pago estaria a produzir o
//   numero que o servidor recusou -- e o utilizador acreditaria nele.*
//
// 201 vs 200 NA RECEITA, E A DIFERENCA E' REAL
//
//     "201 quando cria; 200 quando a receita ja' existia (idempotente)."
//
// A receita de uma ordem e' **um** lancamento. Reenviar nao cria um segundo. E' a mesma doutrina do
// E-458 (devolver o MESMO recurso), mas aqui o servidor **distingue** os dois casos pelo status.
//
// ⚠️ O `agentFetch` **lanca em erro HTTP e nao expoe o status nem o corpo** -- por isso o painel NAO
// consegue dizer «criada agora» vs «ja' existia». Diz o que sabe: que o lancamento existe. E' o
// mesmo limite do E-1352 (o 409 que trazia o agendamento).
//
// AS TRES NAO DEVOLVEM O MESMO STATUS
//
//   revenue    -> 201 (criou) ou 200 (ja' existia)
//   cost       -> 201 sempre
//   reconcile  -> 200 sempre  (liga a um lancamento que JA' existe, nao cria)
//
//   *Um cliente que assuma 201 para as tres esta' a assumir que reconciliar cria alguma coisa.*
//
// `amount_cents` E' INTEIRO EM CENTAVOS
//
// O campo e' `int64` em centavos. Um valor em reais aqui e' um erro de factor 100 -- e o tipo nao o
// apanha, porque `1250` e' um int valido quer signifique R$ 12,50 quer R$ 1.250,00.
import { agentFetch } from "@/lib/agenticClient";

export interface LancamentoFinanceiro {
  id?: string;
  order_id?: string;
  kind?: string;
  amount_cents?: number;
  category?: string;
  reference?: string;
  created?: boolean;
  created_at?: string;
  [k: string]: unknown;
}

/** O financeiro da empresa nao esta' ligado neste runtime. */
export class FinanceiroAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FinanceiroAusenteError";
  }
}

/**
 * O servidor RECUSOU o lancamento. O caso que mais importa: receita de pedido ainda nao pago.
 * *Lancar receita dele seria receita ficticia.*
 */
export class LancamentoRecusadoError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "LancamentoRecusadoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|financeiro|company missing|empresa/i.test(mensagem)) {
    throw new FinanceiroAusenteError("O financeiro não está ligado neste runtime.");
  }
  if (/409|n[aã]o pago|aguardando pagamento|fict[ií]cia|400|obrigat[oó]ri|inv[aá]lid|n[aã]o encontrad/i.test(mensagem)) {
    throw new LancamentoRecusadoError(mensagem);
  }
  throw causa;
}

/** O `amount_cents` e' inteiro em CENTAVOS -- nao em reais. */
export function validarMontante(amountCents: number | undefined): string | null {
  if (amountCents === undefined || amountCents === null) return "o montante é obrigatório (em centavos)";
  if (!Number.isInteger(amountCents)) return "o montante tem de ser um inteiro em CENTAVOS, não em reais";
  if (amountCents <= 0) return "o montante tem de ser positivo";
  return null;
}

/** Categoria e referencia identificam a natureza do custo. */
export function validarCusto(category: string | undefined, reference: string | undefined): string | null {
  const faltas: string[] = [];
  if ((category ?? "").trim() === "") faltas.push("a categoria");
  if ((reference ?? "").trim() === "") faltas.push("a referência");
  if (faltas.length > 0) return `falta ${faltas.join(" e ")}`;
  return null;
}

/**
 * O que a receita exige que o painel NAO pode verificar: o estado do pagamento.
 * A recusa e' do servidor -- o painel nao sabe se o pedido foi pago.
 */
export function descreverFronteiraDaReceita(): string {
  return (
    "O painel não sabe se o pedido foi pago — quem sabe é o servidor, e ele recusa a receita de um " +
    "pedido aguardando pagamento. Lançar receita dele seria receita fictícia."
  );
}

/** Formata centavos como moeda, para o erro de factor 100 ser visivel. */
export function formatarCentavos(centavos: number | undefined): string {
  if (centavos === undefined || centavos === null || !Number.isFinite(centavos)) return "—";
  const sinal = centavos < 0 ? "-" : "";
  const abs = Math.abs(Math.round(centavos));
  const reais = Math.floor(abs / 100);
  const resto = String(abs % 100).padStart(2, "0");
  return `${sinal}R$ ${reais.toLocaleString("pt-BR")},${resto}`;
}

/**
 * O que veio no lancamento. O transporte nao expoe o STATUS (201 vs 200), por isso o painel nao
 * distingue "criada agora" de "ja' existia" -- diz o que sabe, e nao inventa a diferenca.
 */
export function descreverLancamento(r: LancamentoFinanceiro | null): string {
  if (!r) return "o servidor respondeu sem lançamento";
  const id = (r.id ?? "").trim() === "" ? "(sem id)" : r.id;
  const partes = [`lançamento ${id}`];
  if (r.kind) partes.push(`tipo: ${r.kind}`);
  if (typeof r.amount_cents === "number") partes.push(`valor: ${formatarCentavos(r.amount_cents)}`);
  if (r.reference) partes.push(`referência: ${r.reference}`);
  // O transporte descarta o status: nao dizemos "criado" nem "ja' existia".
  partes.push("o status HTTP não chega ao painel: não distingue «criado agora» de «já existia»");
  return partes.join(" · ");
}

/** O que cada operacao promete devolver, dito antes de a chamar. */
export function descreverOperacao(op: OperacaoFinanceira): string {
  switch (op) {
    case "revenue":
      return "lança a receita · 201 se criou, 200 se já existia (idempotente)";
    case "cost":
      return "lança um custo · 201 sempre";
    case "reconcile":
      return "liga a uma linha do extrato que JÁ existe · 200 sempre, não cria nada";
    default:
      return "operação desconhecida";
  }
}

export type OperacaoFinanceira = "revenue" | "cost" | "reconcile";

const CAMINHOS: Record<OperacaoFinanceira, string> = {
  revenue: "/api/agent/v1/finance/orders/",
  cost: "/api/agent/v1/finance/orders/",
  reconcile: "/api/agent/v1/finance/orders/",
};

const SUFIXOS: Record<OperacaoFinanceira, string> = {
  revenue: "/revenue",
  cost: "/cost",
  reconcile: "/reconcile",
};

/**
 * Lanca. E' uma ESCRITA: so' com gesto do utilizador.
 * O id vai no CAMINHO porque a rota e' `:id` -- e por isso e' codificado.
 */
export async function lancar(entrada: {
  operacao: OperacaoFinanceira;
  orderId: string;
  amountCents?: number;
  category?: string;
  reference?: string;
}): Promise<LancamentoFinanceiro> {
  const id = (entrada.orderId ?? "").trim();
  if (id === "") throw new LancamentoRecusadoError("falta o identificador da ordem (id)");

  let corpo: Record<string, unknown> = {};

  if (entrada.operacao === "cost") {
    const erro = validarMontante(entrada.amountCents);
    if (erro) throw new LancamentoRecusadoError(erro);
    const erroCusto = validarCusto(entrada.category, entrada.reference);
    if (erroCusto) throw new LancamentoRecusadoError(erroCusto);
    corpo = {
      amount_cents: entrada.amountCents,
      category: (entrada.category ?? "").trim(),
      reference: (entrada.reference ?? "").trim(),
    };
  } else if (entrada.operacao === "reconcile") {
    const ref = (entrada.reference ?? "").trim();
    if (ref === "") throw new LancamentoRecusadoError("falta a referência do extrato");
    corpo = { reference: ref };
  }
  // revenue nao leva corpo: o servidor calcula o valor do pedido pago.

  const caminho = CAMINHOS[entrada.operacao] + encodeURIComponent(id) + SUFIXOS[entrada.operacao];

  try {
    return await agentFetch<LancamentoFinanceiro>(caminho, {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
