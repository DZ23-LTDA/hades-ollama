// FASE 23: comércio -- a superfície das rotas de leitura de `/commerce`.
//
// Oitava, nona e décima rotas do E-1319 a chegar ao produto:
// `/commerce/products`, `/commerce/orders` e `/commerce/summary`.
//
// `commerce` era o maior domínio restante sem UI (**12 rotas**), com um motor
// maduro (`commerce.go` 54 KB, `live_commerce.go` 29 KB, ~12 ficheiros de teste)
// e nenhum componente `Commerce*`.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// OS CONTRATOS:
//
//	GET /commerce/products -> 200 {products:[...], count:N}   503 store ausente
//	GET /commerce/orders   -> 200 {orders:[...], count:N}     503
//	GET /commerce/summary  -> 200 {organization_id, products, orders, pending,
//	                               paid, fulfilled, cancelled, revenue_cents,
//	                               low_stock_skus}            503
//
// TRÊS DECISÕES QUE O PAINEL NÃO PODE ERRAR:
//
//  1. **CENTAVOS NÃO SÃO REAIS.** `price_cents`, `total_cents` e `revenue_cents`
//     são inteiros de cêntimos. Mostrar `125000` como se fossem reais é mostrar o
//     número certo na unidade errada -- e num valor de dinheiro isso é grave.
//
//  2. **503 NÃO É LISTA VAZIA.** `errCommerceUnavailable` é devolvido quando o
//     store não está ligado. «Não há loja ligada» e «não há produtos» são estados
//     diferentes.
//
//  3. **`Duplicated` NÃO É UM PEDIDO NOVO.** O comentário do servidor: *"Duplicated
//     marca a devolução idempotente do MESMO pedido."* Um painel que o conta como
//     pedido novo inflaciona a contagem.
import { agentFetch } from "@/lib/agenticClient";

export interface ProdutoDeComercio {
  id: string;
  organization_id: string;
  sku: string;
  name: string;
  price_cents: number;
  currency: string;
  active: boolean;
  created_at: string;
}

export interface ItemDePedido {
  sku?: string;
  quantity?: number;
  unit_price_cents?: number;
  [chave: string]: unknown;
}

export interface PedidoDeComercio {
  id: string;
  organization_id: string;
  number: string;
  contact_id?: string;
  status: string;
  items: ItemDePedido[] | null;
  total_cents: number;
  currency: string;
  idempotency_key: string;
  reconciled_reference?: string;
  created_at: string;
  updated_at: string;
  cancelled_at?: string;
  /** Devolução idempotente do MESMO pedido -- não é um pedido novo. */
  duplicated?: boolean;
}

export interface ResumoDeComercio {
  organization_id: string;
  products: number;
  orders: number;
  pending: number;
  paid: number;
  fulfilled: number;
  cancelled: number;
  revenue_cents: number;
  low_stock_skus?: string[];
}

export interface RespostaDeProdutos {
  products: ProdutoDeComercio[] | null;
  count: number;
}

export interface RespostaDePedidos {
  orders: PedidoDeComercio[] | null;
  count: number;
}

/** O store de comércio não está ligado neste runtime. */
export class LojAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "LojAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|service unavailable|store de comercio|comercio nao (esta|está) ligado|ErrCommerceInvalid/i.test(mensagem)) {
    throw new LojAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Centavos ditos em reais. `125000` cêntimos são `R$ 1.250,00`, e não `125000`.
 *
 *	*Um valor de dinheiro na unidade errada não é um arredondamento: é uma
 *	mentira com duas casas decimais.*
 */
export function centavosParaReais(centavos: number | undefined, moeda = "BRL"): string {
  if (centavos === undefined || centavos === null || !Number.isFinite(centavos)) return "—";
  const unidade = centavos / 100;
  // `toLocaleString` da' o separador de milhar. `R$ 1250,00` esta' certo e e'
  // pior de ler que `R$ 1.250,00` -- num valor de dinheiro, legibilidade nao e'
  // estetica.
  const valor = unidade.toLocaleString("pt-BR", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
  const prefixo = (moeda || "BRL").toUpperCase() === "BRL" ? "R$" : moeda;
  return `${prefixo} ${valor}`;
}

/**
 * Diz quantos pedidos são de facto novos e quantos são devoluções idempotentes.
 *
 *	*Uma devolução idempotente contada como pedido novo inflaciona a contagem --
 *	e a contagem é o que o utilizador usa para decidir.*
 */
export function contarPedidos(pedidos: PedidoDeComercio[]): {
  total: number;
  novos: number;
  devolvidos: number;
} {
  let devolvidos = 0;
  for (const p of pedidos) if (p.duplicated) devolvidos += 1;
  return { total: pedidos.length, novos: pedidos.length - devolvidos, devolvidos };
}

/** O estado do pedido, dito sem inventar rótulo para o que não se conhece. */
export function descreverEstadoDoPedido(status: string | undefined): string {
  const bruto = (status ?? "").trim();
  if (bruto === "") return "sem estado";
  const conhecidos: Record<string, string> = {
    pending: "pendente",
    paid: "pago",
    fulfilled: "cumprido",
    cancelled: "cancelado",
  };
  return conhecidos[bruto] ?? `estado desconhecido (${bruto})`;
}

/** O preço em unidades, com duas casas -- e a moeda do próprio produto. */
export function descreverPreco(produto: ProdutoDeComercio): string {
  return centavosParaReais(produto.price_cents, produto.currency);
}

export async function listarProdutos(init?: RequestInit): Promise<RespostaDeProdutos> {
  try {
    // Prefixo `/api/agent/v1` explícito: o gate compara o path CRU (E-1321).
    return await agentFetch<RespostaDeProdutos>("/api/agent/v1/commerce/products", init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarPedidos(init?: RequestInit): Promise<RespostaDePedidos> {
  try {
    return await agentFetch<RespostaDePedidos>("/api/agent/v1/commerce/orders", init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function resumoDoComercio(init?: RequestInit): Promise<ResumoDeComercio> {
  try {
    // `/commerce/summary` devolve o objecto DIRECTO, sem wrapper `count`.
    return await agentFetch<ResumoDeComercio>("/api/agent/v1/commerce/summary", init);
  } catch (causa) {
    traduzir(causa);
  }
}
