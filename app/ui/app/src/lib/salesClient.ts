// Forca de vendas -- propostas, itens, precificacao aberta e pos-venda.
//
// Decima sexta, decima setima e decima oitava rotas do E-1319 a chegar ao produto:
// `/sales/proposals`, `/sales/proposals/:id` e `/sales/follow-ups`.
//
// `sales` era o maior dominio restante (**8 rotas**), com `sales_workforce.go` 23 KB e
// precificacao detalhada. Nao havia componente `*ales*`.
//
// OS CONTRATOS:
//
//	GET /sales/proposals?status=   -> 200 {proposals:[...], count:N}   503
//	GET /sales/proposals/:id       -> 200 SalesProposal (objecto directo)   503
//	GET /sales/follow-ups?overdue= -> 200 {follow_ups:[...], count:N}   503
//
// TRES DECISOES QUE O PAINEL NAO PODE ERRAR:
//
//  1. **A CONTA FICA ABERTA.** O proprio tipo diz, com o ID da evidencia:
//
//         // PRECIFICACAO (E-269): a conta fica ABERTA na proposta. Um total unico
//         // esconderia quanto foi desconto, quanto foi imposto e quanto foi frete.
//
//     O painel mostra subtotal, desconto, imposto, frete e total -- **nunca apenas o
//     total**. *Uma proposta que mostra so' o total nao mostra a proposta: mostra o
//     preco.*
//
//  2. **FRETE NAO DECLARADO NAO E' FRETE GRATIS.** `shipping_declared` existe porque,
//     no comentario do tipo: «Zero nao informado nao e zero cobrado.» O painel diz
//     «frete nao declarado» em vez de escrever `R$ 0,00`.
//
//  3. **CENTAVOS NAO SAO REAIS**, e `discount_basis_points`/`tax_basis_points` sao
//     pontos-base (10000 = 100%).
import { agentFetch } from "@/lib/agenticClient";
import { centavosParaReais } from "@/lib/commerceClient";

export interface ItemDeProposta {
  sku: string;
  name: string;
  quantity: number;
  unit_price_cents: number;
  total_cents: number;
}

export interface PropostaDeVenda {
  id: string;
  organization_id: string;
  contact_id: string;
  opportunity_id?: string;
  currency: string;
  items: ItemDeProposta[] | null;
  subtotal_cents: number;
  discount_basis_points: number;
  discount_cents: number;
  tax_basis_points: number;
  tax_cents: number;
  shipping_cents: number;
  shipping_declared: boolean;
  total_cents: number;
  status: string;
  valid_until: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  sent_at?: string;
  decided_at?: string;
  decided_by?: string;
}

export interface AcompanhamentoDeVenda {
  id: string;
  organization_id: string;
  contact_id: string;
  order_id?: string;
  reason: string;
  status: string;
  due_at: string;
  created_at: string;
  completed_at?: string;
  completed_by?: string;
}

export interface RespostaDePropostas {
  proposals: PropostaDeVenda[] | null;
  count: number;
}

export interface RespostaDeAcompanhamentos {
  follow_ups: AcompanhamentoDeVenda[] | null;
  count: number;
}

/** A forca de vendas nao esta' ligada neste runtime (`errSalesUnavailable`). */
export class VendasAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "VendasAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/n[aã]o est[aá] dispon[íi]vel|503|n[aã]o existe/i.test(mensagem)) {
    throw new VendasAusenteError(mensagem);
  }
  throw causa;
}

/** Valor em reais. Centavos nao sao reais. */
export function descreverValorDaProposta(centavos: number | undefined, moeda = "BRL"): string {
  return centavosParaReais(centavos, moeda);
}

/**
 * Descreve o frete: valor, ou «frete nao declarado» quando nao foi informado.
 *
 * `shipping_declared=false` e `shipping_cents=0` significa «ninguem disse», nao
 * «e' gratis». Escrever `R$ 0,00` seria afirmar o que a proposta nao afirma.
 */
export function descreverFrete(proposta: PropostaDeVenda): string {
  if (!proposta.shipping_declared) return "frete nao declarado";
  return descreverValorDaProposta(proposta.shipping_cents, proposta.currency);
}

/** Pontos-base em percentagem legivel: 500 -> `5%`, 1250 -> `12,5%`, 0 -> `0%`. */
export function formatarPontosBase(pontos: number | undefined): string {
  const valor = Math.max(0, pontos ?? 0);
  const percentagem = valor / 100;
  const texto = Number.isInteger(percentagem)
    ? String(percentagem)
    : percentagem.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
  return `${texto.replace(".", ",")}%`;
}

/**
 * A conta aberta: as linhas que compoem o total, por ordem.
 *
 * O total vem SEMPRE no fim e vem SEMPRE acompanhado das parcelas. Nunca ha' uma
 * versao desta funcao que devolva so' o total.
 */
export function abrirAConta(
  proposta: PropostaDeVenda,
): { etiqueta: string; valor: string }[] {
  const moeda = proposta.currency;
  const linhas: { etiqueta: string; valor: string }[] = [
    {
      etiqueta: `subtotal`,
      valor: descreverValorDaProposta(proposta.subtotal_cents, moeda),
    },
  ];
  if (proposta.discount_cents !== 0) {
    linhas.push({
      etiqueta: `desconto (${formatarPontosBase(proposta.discount_basis_points)})`,
      valor: `- ${descreverValorDaProposta(proposta.discount_cents, moeda)}`,
    });
  }
  if (proposta.tax_cents !== 0) {
    linhas.push({
      etiqueta: `imposto (${formatarPontosBase(proposta.tax_basis_points)})`,
      valor: descreverValorDaProposta(proposta.tax_cents, moeda),
    });
  }
  linhas.push({ etiqueta: "frete", valor: descreverFrete(proposta) });
  linhas.push({ etiqueta: "total", valor: descreverValorDaProposta(proposta.total_cents, moeda) });
  return linhas;
}

/** Estado da proposta em texto legivel, sem inventar o que nao conhece. */
export function descreverEstadoDaProposta(estado: string | undefined): string {
  const bruto = (estado ?? "").trim();
  if (bruto === "") return "sem estado";
  return bruto.replace(/_/g, " ");
}

/** Total de unidades pedidas. Item sem quantidade nao soma zero por engano. */
export function contarUnidades(itens: ItemDeProposta[] | null | undefined): number {
  return (itens ?? []).reduce((soma, item) => soma + (item.quantity ?? 0), 0);
}

export async function listarPropostas(
  estado?: string,
  init?: RequestInit,
): Promise<RespostaDePropostas> {
  const limpo = (estado ?? "").trim();
  const sufixo = limpo !== "" ? `?status=${encodeURIComponent(limpo)}` : "";
  try {
    // Prefixo `/api/agent/v1` explicito: o gate compara o path CRU (E-1321).
    return await agentFetch<RespostaDePropostas>(`/api/agent/v1/sales/proposals${sufixo}`, init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function obterProposta(id: string, init?: RequestInit): Promise<PropostaDeVenda> {
  const limpo = id.trim();
  if (limpo === "") throw new Error("id da proposta vazio");
  try {
    return await agentFetch<PropostaDeVenda>(
      `/api/agent/v1/sales/proposals/${encodeURIComponent(limpo)}`,
      init,
    );
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarAcompanhamentos(
  apenasAtrasados = false,
  init?: RequestInit,
): Promise<RespostaDeAcompanhamentos> {
  const sufixo = apenasAtrasados ? "?overdue=true" : "";
  try {
    return await agentFetch<RespostaDeAcompanhamentos>(
      `/api/agent/v1/sales/follow-ups${sufixo}`,
      init,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
