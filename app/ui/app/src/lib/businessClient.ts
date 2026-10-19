// Acoes de negocio pendentes de decisao humana -- `GET /business/actions`.
//
// `business` tem 4 rotas e SO' UMA e' leitura: esta. As outras 3 sao escritas
// (certify, tick, decide) -- o `decide` aprova e EXECUTA, recusa e nao executa.
// O painel nao decide: mostra o que espera decisao.
//
// AS SEIS FASES DE UMA ACAO NAO SAO A MESMA COISA
//
//   AUTONOMA              -- correu SEM humano. O supervisor decidiu sozinho.
//   REQUER_APROVACAO      -- o tipo de acao exige aprovacao (declarado no tipo).
//   EXECUTADA             -- correu, e alguem (ou o supervisor) executou.
//   AGUARDANDO_APROVACAO  -- espera um humano. E' o que este painel lista.
//   RECUSADA              -- um humano disse nao. Nao executou.
//   BLOQUEADA_ORCAMENTO   -- o ORCAMENTO travou. Nao e' o mesmo que esperar humano:
//                            nao adianta aprovar, porque nao ha' verba.
//
// *Uma acao bloqueada por orcamento e uma acao a espera de aprovacao sao ambas "pendentes"
// e nao sao a mesma coisa: uma precisa de dinheiro, a outra precisa de uma pessoa.*
//
// E O CUSTO NAO E' DECORATIVO: `cost_cents` e' quanto a acao vai gastar. Aprovar uma acao
// sem ver o custo e' aprovar um numero que nao se leu.
import { centavosParaReais } from "@/lib/commerceClient";
import { agentFetch } from "@/lib/agenticClient";

export type EstadoDaAcaoDeNegocio =
  | "AUTONOMA"
  | "REQUER_APROVACAO"
  | "EXECUTADA"
  | "AGUARDANDO_APROVACAO"
  | "RECUSADA"
  | "BLOQUEADA_ORCAMENTO";

export interface AcaoDeNegocio {
  id: string;
  organization_id: string;
  /** `lancar_receita`, `repor_estoque`, `atribuir_fila_humana`, `estornar_pagamento`, `cancelar_pedido`. */
  kind: string;
  subject: string;
  reason: string;
  risk: string;
  /** Quanto a acao vai gastar. Aprovar sem ler isto e' aprovar um numero que nao se leu. */
  cost_cents: number;
  status: EstadoDaAcaoDeNegocio | string;
  effect?: string;
  decided_by?: string;
  decided_at?: string;
  created_at: string;
}

export interface RespostaDeAcoesDeNegocio {
  pending: AcaoDeNegocio[] | null;
  count: number;
  policy: Record<string, unknown>;
}

/** O supervisor de negocio nao esta' ligado neste runtime. */
export class NegocioAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "NegocioAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/business|neg[oó]cio|n[aã]o est[aá] ligado|n[aã]o est[aá] dispon[íi]vel|503/i.test(mensagem)) {
    throw new NegocioAusenteError(mensagem);
  }
  throw causa;
}

const ROTULOS: Record<string, string> = {
  AUTONOMA: "autónoma — correu sem humano",
  REQUER_APROVACAO: "requer aprovação",
  EXECUTADA: "executada",
  AGUARDANDO_APROVACAO: "aguardando aprovação humana",
  RECUSADA: "recusada por humano",
  BLOQUEADA_ORCAMENTO: "BLOQUEADA POR ORÇAMENTO — aprovar não resolve",
};

/** Traduz o estado, sem colapsar orçamento em aprovação. */
export function descreverEstadoDaAcao(status: string | undefined): string {
  const bruto = (status ?? "").trim();
  if (bruto === "") return "estado não declarado";
  return ROTULOS[bruto] ?? bruto;
}

/** Um estado que exige DINHEIRO, nao uma pessoa. */
export function bloqueadaPorOrcamento(acao: AcaoDeNegocio): boolean {
  return (acao.status ?? "").trim() === "BLOQUEADA_ORCAMENTO";
}

/** Um estado que exige uma PESSOA. */
export function aguardaHumano(acao: AcaoDeNegocio): boolean {
  const s = (acao.status ?? "").trim();
  return s === "AGUARDANDO_APROVACAO" || s === "REQUER_APROVACAO";
}

/** O custo, dito como custo. Zero nao e' "sem custo": e' um numero. */
export function descreverCusto(centavos: number | undefined): string {
  const valor = Math.max(0, centavos ?? 0);
  if (valor === 0) return "sem custo declarado";
  return `custo: ${centavosParaReais(valor)}`;
}

/** Risco declarado, legivel. */
export function descreverRisco(acao: AcaoDeNegocio): string {
  const risco = (acao.risk ?? "").trim();
  if (risco === "") return "risco não declarado";
  return `risco: ${risco.replace(/_/g, " ")}`;
}

/** O tipo de acao, legivel. */
export function descreverTipo(acao: AcaoDeNegocio): string {
  const tipo = (acao.kind ?? "").trim();
  if (tipo === "") return "tipo não declarado";
  return tipo.replace(/_/g, " ");
}

/** Separa o que precisa de dinheiro do que precisa de pessoa. */
export function resumirPendentes(acoes: AcaoDeNegocio[]): string {
  const orcamento = acoes.filter(bloqueadaPorOrcamento).length;
  const humano = acoes.filter(aguardaHumano).length;
  const outras = acoes.length - orcamento - humano;
  const partes: string[] = [];
  if (humano > 0) partes.push(`${humano} à espera de uma pessoa`);
  if (orcamento > 0) partes.push(`${orcamento} à espera de orçamento`);
  if (outras > 0) partes.push(`${outras} noutro estado`);
  if (partes.length === 0) return "nenhuma ação pendente";
  return partes.join(" · ");
}

export async function listarAcoesDeNegocio(): Promise<RespostaDeAcoesDeNegocio> {
  try {
    return await agentFetch<RespostaDeAcoesDeNegocio>("/api/agent/v1/business/actions");
  } catch (causa) {
    traduzir(causa);
  }
}
