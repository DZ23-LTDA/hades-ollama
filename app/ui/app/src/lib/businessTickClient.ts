// Ciclo de supervisão de negócio -- `POST /business/tick`.
//
// E' UMA ESCRITA, E O PAINEL NAO A DISPARA AO MONTAR.
//
// Um tick executa acções autónomas, regista incidentes e gasta orçamento. Montar o painel e
// disparar um ciclo seria *executar sem ninguém pedir* -- o oposto do que a supervisão promete.
// Por isso há um botão, e o painel começa em `pronto`.
//
// O HANDLER DIZ QUE O RESULTADO VAI JUNTO MESMO COM ERRO
//
//     "O resultado VAI junto mesmo com erro de dominio declarado: e' ele que mostra o que foi
//      observado, executado, bloqueado e o que aguarda decisao."
//
//   *Um tick que falha a meio e nao devolve o que fez obriga a adivinhar o estado do mundo. Aqui
//   o erro e' um CAMPO do resultado, nao uma ausencia de resultado.*
//
// QUATRO CONTADORES QUE NAO SAO O MESMO NUMERO
//
//     autonomous_executed  -- correu sem pessoa
//     awaiting_approval    -- precisa de uma PESSOA
//     blocked_by_budget    -- precisa de DINHEIRO
//     incidents_recorded   -- correu mal e ficou registado
//
//   *Somar isto num "N acções" e' a mesma mentira do E-1340: uma acção bloqueada por orçamento e
//   uma à espera de aprovação são ambas «pendentes» e não são a mesma coisa: uma precisa de
//   dinheiro, a outra precisa de uma pessoa.*
//
// O ORÇAMENTO TAMBÉM SÃO DOIS NÚMEROS: `budget_spent_cents` e `budget_left_cents`. O painel
// mostra os dois e confere que somam o mesmo total inicial quando o servidor o declara.
import { agentFetch } from "@/lib/agenticClient";

export interface SinalDeNegocio {
  kind?: string;
  subject?: string;
  detail?: string;
}

export interface AcaoDeNegocioDoTick {
  id: string;
  kind: string;
  subject: string;
  reason: string;
  risk: string;
  cost_cents: number;
  status: string;
  effect?: string;
  decided_by?: string;
  decided_at?: string;
  created_at: string;
}

export interface ResultadoDoTick {
  tick_number: number;
  organization_id?: string;
  executed_at: string;
  signals?: SinalDeNegocio[] | null;
  actions?: AcaoDeNegocioDoTick[] | null;
  budget_spent_cents: number;
  budget_left_cents: number;
  /** Correu sem pessoa. */
  autonomous_executed: number;
  /** Precisa de uma PESSOA. */
  awaiting_approval: number;
  /** Precisa de DINHEIRO. */
  blocked_by_budget: number;
  /** Correu mal e ficou registado. */
  incidents_recorded: number;
  /** Presente mesmo quando o tick "correu": o resultado vem junto. */
  errors?: string[] | null;
}

/** O supervisor de negócio não está ligado neste runtime. */
export class SupervisaoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "SupervisaoAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|supervisor/i.test(mensagem)) {
    throw new SupervisaoAusenteError(
      "O supervisor de negócio não está ligado neste runtime.",
    );
  }
  throw causa;
}

/** Os quatro contadores, cada um com o seu significado. Nunca somados. */
export function descreverDesfechos(r: ResultadoDoTick): string {
  return (
    `${r.autonomous_executed ?? 0} executada(s) sem pessoa · ` +
    `${r.awaiting_approval ?? 0} à espera de uma pessoa · ` +
    `${r.blocked_by_budget ?? 0} bloqueada(s) por orçamento · ` +
    `${r.incidents_recorded ?? 0} incidente(s) registado(s)`
  );
}

/** O total que precisa de ALGUÉM (pessoa ou dinheiro) -- distinto do que correu. */
export function precisaDeAlguem(r: ResultadoDoTick): number {
  return (r.awaiting_approval ?? 0) + (r.blocked_by_budget ?? 0);
}

/** O tick avançou? O número é sequencial e não pode recuar. */
export function descreverTick(r: ResultadoDoTick, anterior: number | null): string {
  const n = r.tick_number ?? 0;
  if (anterior !== null && n <= anterior) {
    return `tick #${n} NÃO avançou em relação ao anterior (#${anterior}) — o ciclo não correu de novo`;
  }
  return anterior === null ? `tick #${n}` : `tick #${n} (anterior: #${anterior})`;
}

/** O erro veio COM resultado: não é ausência de resultado. */
export function descreverErros(r: ResultadoDoTick): string | null {
  const erros = r.errors ?? [];
  if (erros.length === 0) return null;
  return `${erros.length} erro(s) durante o ciclo — o resultado abaixo é o que FOI observado apesar deles`;
}

/** O orçamento, com os dois números e a soma quando o total é conhecido. */
export function descreverOrcamento(r: ResultadoDoTick, totalInicial: number | null): string {
  const gasto = r.budget_spent_cents ?? 0;
  const restante = r.budget_left_cents ?? 0;
  const base = `gasto: ${gasto} centavo(s) · restante: ${restante} centavo(s)`;
  if (totalInicial === null) return base;
  if (gasto + restante === totalInicial) return `${base} · soma confere com ${totalInicial}`;
  return `${base} · soma NÃO confere com ${totalInicial} (${gasto} + ${restante} = ${gasto + restante})`;
}

/** As acções do ciclo, cada uma com o seu estado -- sem colapsar. */
export function resumirAcoes(r: ResultadoDoTick): string {
  const acoes = r.actions ?? [];
  if (acoes.length === 0) return "o ciclo não produziu acções";
  const porEstado = new Map<string, number>();
  for (const a of acoes) {
    const s = (a.status ?? "").trim() || "sem estado";
    porEstado.set(s, (porEstado.get(s) ?? 0) + 1);
  }
  return Array.from(porEstado.entries())
    .map(([estado, n]) => `${n}× ${estado}`)
    .join(" · ");
}

/** Dispara um ciclo. E' uma ESCRITA: só com gesto do utilizador. */
export async function dispararTick(): Promise<ResultadoDoTick> {
  try {
    return await agentFetch<ResultadoDoTick>("/api/agent/v1/business/tick", {
      method: "POST",
    });
  } catch (causa) {
    traduzir(causa);
  }
}
