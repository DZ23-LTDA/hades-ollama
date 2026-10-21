// FASE 19: roteamento multiobjectivo e fronteira de Pareto.
//
// Terceira das 93 rotas do E-1319 a chegar ao produto. `/routing` e' a rota que
// responde PORQUE e' que o HADES escolheu um modelo -- e ate' agora ninguem a
// podia perguntar.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// O CONTRATO, lido do servidor (`server/agent_routes.go:4676`):
//
//	GET /routing?capabilities=&preference=&local_only=&max_cost_cents=&path=
//	200 -> { winner:{model,provider,score,reason},
//	         candidates:[{rank,model,provider,score,cost_cents_per_1k,
//	                      quality_score,reason,factors}],
//	         frontier:[{model,score,quality,cost,latency,privacy}] }
//	400 -> preferencia / local_only / max_cost_cents invalidos
//	422 -> NENHUM modelo satisfaz as restricoes (decisao honesta, nao avaria)
//	503 -> registry de modelos indisponivel
//
//	*Um score sem os componentes que o somam obriga quem o le' a confiar em quem
//	o escreveu.*
import { agentFetch } from "@/lib/agenticClient";

/** As preferências que o servidor aceita. As chaves são o que vai na query. */
export const PREFERENCIAS = [
  { chave: "auto", rotulo: "Equilibrado (auto)" },
  { chave: "best_quality", rotulo: "Melhor qualidade" },
  { chave: "fastest", rotulo: "Mais rápido" },
  { chave: "cheapest", rotulo: "Mais barato" },
] as const;

export type ChaveDePreferencia = (typeof PREFERENCIAS)[number]["chave"];

export interface CandidatoDeRota {
  rank: number;
  model: string;
  provider: string;
  score: number;
  cost_cents_per_1k: number;
  quality_score: number;
  reason: string;
  factors?: Record<string, number>;
}

export interface AlternativaDePareto {
  model: string;
  score: number;
  quality: number;
  cost: number;
  latency: number;
  privacy: number;
}

export interface RespostaDeRota {
  winner: {
    model: string;
    provider: string;
    score: number;
    reason: string;
  };
  candidates: CandidatoDeRota[];
  frontier: AlternativaDePareto[];
}

export interface PedidoDeRota {
  capabilities?: string;
  preference?: ChaveDePreferencia;
  localOnly?: boolean;
  maxCostCents?: string;
  path?: string;
}

/**
 * Monta a query. Cada parâmetro só entra se tiver valor -- um `local_only=false`
 * explícito não muda nada no servidor, mas um `capabilities=` vazio mudaria o
 * sentido do pedido se fosse interpretado como "capacidade vazia".
 */
export function montarConsulta(pedido: PedidoDeRota): string {
  const partes: string[] = [];
  const capacidades = (pedido.capabilities ?? "").trim();
  if (capacidades !== "") partes.push(`capabilities=${encodeURIComponent(capacidades)}`);
  if (pedido.preference) partes.push(`preference=${encodeURIComponent(pedido.preference)}`);
  if (pedido.localOnly) partes.push("local_only=true");
  const custo = (pedido.maxCostCents ?? "").trim();
  if (custo !== "") partes.push(`max_cost_cents=${encodeURIComponent(custo)}`);
  const caminho = (pedido.path ?? "").trim();
  if (caminho !== "") partes.push(`path=${encodeURIComponent(caminho)}`);
  return partes.length ? `?${partes.join("&")}` : "";
}

/**
 * O 422 é RESULTADO, não avaria.
 *
 *	*Uma recusa que explica as restrições que a produziram é informação; a mesma
 *	recusa mostrada como "falhou" é ruído.*
 */
export class SemRotaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "SemRotaError";
  }
}

/**
 * O `agentFetch` lança em erro HTTP; a mensagem do 422 traz as restrições. O
 * prefixo `/api/agent/v1` é explícito porque o gate de contrato compara o path
 * CRU (a armadilha do E-1321).
 */
export async function consultarRota(
  pedido: PedidoDeRota,
  init?: RequestInit,
): Promise<RespostaDeRota> {
  try {
    return await agentFetch<RespostaDeRota>(
      `/api/agent/v1/routing${montarConsulta(pedido)}`,
      init,
    );
  } catch (causa) {
    const mensagem = causa instanceof Error ? causa.message : String(causa);
    if (/nenhum modelo satisfaz|no route/i.test(mensagem)) {
      throw new SemRotaError(mensagem);
    }
    throw causa;
  }
}

/** Custo somado por 1k de entrada + saída, dito em reais quando dá. */
export function descreverCusto(centavos: number): string {
  if (!Number.isFinite(centavos) || centavos <= 0) return "grátis";
  const reais = centavos / 100;
  return `R$ ${reais.toFixed(2).replace(".", ",")}/1k`;
}

/**
 * Um score sem os componentes visíveis é uma caixa negra. Os factores vêm do
 * servidor e somam o score; o painel mostra-os todos, não só o total.
 */
export function descreverFactores(factores: Record<string, number> | undefined): string {
  if (!factores) return "—";
  const entradas = Object.entries(factores)
    .filter(([, valor]) => typeof valor === "number")
    .sort(([a], [b]) => a.localeCompare(b));
  if (entradas.length === 0) return "—";
  return entradas.map(([nome, valor]) => `${nome}=${valor.toFixed(2)}`).join(", ");
}
