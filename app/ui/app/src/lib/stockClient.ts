// Stock por SKU -- `GET /commerce/stock/:sku`.
//
// `commerce` teve 6 rotas; 3 ja' eram consumidas pelo `CommercePanel` (products, orders,
// summary). Esta e' a **consulta dirigida**: precisa de um SKU para responder.
//
// PORQUE E' UM PAINEL PROPRIO E NAO UMA SECCAO DO `CommercePanel`
//
// O `CommercePanel` e' uma VISTA DE AGREGADOS (lista de produtos, lista de pedidos, resumo).
// Esta e' uma consulta **com input**: sem SKU nao ha' nada a mostrar. Juntar as duas obrigaria
// o painel de agregados a ter um campo vazio por preencher -- e um painel que arranca a pedir
// input nao e' o mesmo objecto que um que arranca a mostrar totais.
//
// O QUE ESTE CONTRATO ENSINA
//
// `CommerceStockBalance` tem TRES numeros e nao um:
//
//     OnHand    -- existe no armazem
//     Reserved  -- existe mas ja' tem dono
//     Available -- pode ser vendido AGORA
//
//   *Um painel que mostre so' `available` esconde o que esta' reservado; um que mostre
//   `on_hand` como "disponivel" vende mercadoria que ja' tem dono.*
//
// E o comentario do tipo diz o resto: **"O saldo e' DERIVADO"**. O saldo nao e' um campo
// guardado: e' a soma dos movimentos. O painel mostra os movimentos e verifica a soma.
//
//   *Um saldo que se anuncia e nao se deriva e' um numero que ninguem pode auditar. Aqui a
//   soma e' conferida contra os movimentos que o servidor devolveu no mesmo payload.*
import { agentFetch } from "@/lib/agenticClient";

export interface SaldoDeStock {
  sku: string;
  /** Existe no armazém. */
  on_hand: number;
  /** Existe, mas já tem dono. */
  reserved: number;
  /** Pode ser vendido agora. */
  available: number;
  movement_count: number;
}

export interface MovimentoDeStock {
  id: string;
  organization_id?: string;
  sku: string;
  /** Positivo = entrada; negativo = saída ou reserva. */
  delta: number;
  kind: string;
  reference?: string;
  created_at: string;
}

export interface RespostaDeStock {
  balance?: SaldoDeStock | null;
  movements?: MovimentoDeStock[] | null;
}

/** O store de comércio não está ligado neste runtime. */
export class StockAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "StockAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/commerce|com[eé]rcio|n[aã]o est[aá] ligado|n[aã]o dispon[íi]vel|503/i.test(mensagem)) {
    throw new StockAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Os três números, nunca um. `available` e' o unico que se pode vender.
 */
export function descreverSaldo(s: SaldoDeStock | null | undefined): string {
  if (!s) return "sem saldo declarado para este SKU";
  const onHand = Math.max(0, s.on_hand ?? 0);
  const reserved = Math.max(0, s.reserved ?? 0);
  const available = s.available ?? 0;
  return `${available} disponível(is) · ${onHand} em armazém · ${reserved} reservado(s)`;
}

/** O que NAO se pode vender, dito com todas as letras. */
export function descreverIndisponivel(s: SaldoDeStock | null | undefined): string | null {
  if (!s) return null;
  const reserved = Math.max(0, s.reserved ?? 0);
  if (reserved === 0) return null;
  return `${reserved} unidade(s) existem mas já têm dono — não estão disponíveis para venda`;
}

/** O sinal do movimento, dito por palavras. */
export function descreverMovimento(m: MovimentoDeStock): string {
  const delta = m.delta ?? 0;
  if (delta === 0) return "sem alteração";
  if (delta > 0) return `+${delta} (entrada)`;
  return `${delta} (saída)`;
}

/**
 * O saldo é DERIVADO dos movimentos. Aqui confere-se a soma.
 *
 * Devolve `null` quando não há movimentos para conferir (não é falha: é ausência de prova).
 */
export function conferirDerivacao(r: RespostaDeStock): string | null {
  const saldo = r.balance;
  const movimentos = r.movements ?? [];
  if (!saldo || movimentos.length === 0) return null;

  const soma = movimentos.reduce((acc, m) => acc + (m.delta ?? 0), 0);
  const esperado = Math.max(0, saldo.on_hand ?? 0);
  if (soma === esperado) {
    return `saldo conferido: a soma dos ${movimentos.length} movimento(s) bate com o armazém`;
  }
  return `saldo NÃO confere: ${movimentos.length} movimento(s) somam ${soma}, mas o armazém diz ${esperado}`;
}

/** `available` deveria ser `on_hand - reserved`. Quando não é, o painel diz. */
export function conferirDisponibilidade(s: SaldoDeStock | null | undefined): string | null {
  if (!s) return null;
  const calculado = Math.max(0, s.on_hand ?? 0) - Math.max(0, s.reserved ?? 0);
  const declarado = s.available ?? 0;
  if (calculado === declarado) return null;
  return `disponibilidade não confere: armazém − reservado = ${calculado}, mas o servidor diz ${declarado}`;
}

export async function obterStock(sku: string): Promise<RespostaDeStock> {
  const alvo = (sku ?? "").trim();
  if (alvo === "") {
    throw new StockAusenteError("sem SKU: nao ha' stock a consultar");
  }
  try {
    return await agentFetch<RespostaDeStock>(
      `/api/agent/v1/commerce/stock/${encodeURIComponent(alvo)}`,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
