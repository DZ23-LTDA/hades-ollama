// Despertar de agentes -- `POST /wakeups`.
//
// E' A PRIMEIRA FATIA DESTA SERIE QUE ESCREVE COM SEGURANCA, e o motivo esta' no handler:
//
//   1. **Nao duplica.** *"`EnqueueMission` e' idempotente por missao (ha' teste no pacote do
//      agente) -- logo repetir o despertar nao cria job novo. E' o que torna esta rota segura
//      de chamar periodicamente."*
//   2. **Nao executa o que espera aprovacao.** *"Aprovacao pendente e recuperacao produzem
//      despertar INFORMATIVO: enfileirar uma missao que espera aprovacao seria executa-la sem
//      a decisao que ela propria pediu."*
//   3. **Falha de uma nao esconde as outras** -- vai em `failed`.
//   4. **Se nada foi enfileirado e houve falhas, a resposta e' 500.**
//      *"Sucesso parcial declarado nao e' sucesso, e silencio nao e' desfecho."*
//
// O QUE ISTO OBRIGA O PAINEL A DIZER
//
// `wakeups` (todos os despertares) e `enqueued` (os que viraram trabalho) **nao sao a mesma
// lista**. Um painel que mostre so' `enqueued` esconde que o sistema viu missoes -- e um painel
// que mostre so' `wakeups` sugere que todas vao correr. As duas listas, lado a lado, sao a
// informacao.
//
//   *Um despertar informativo e um trabalho enfileirado sao ambos "acordar" e nao sao a mesma
//   coisa: um pede uma pessoa, o outro pede um worker.*
import { agentFetch } from "@/lib/agenticClient";

export interface DespertarDeAgente {
  mission_id?: string;
  id?: string;
  agent_id?: string;
  agente?: string;
  reason?: string;
  motivo?: string;
  informative?: boolean;
  informational?: boolean;
}

export interface RespostaDeWakeups {
  wakeups?: DespertarDeAgente[] | null;
  /** Só o que virou trabalho. Não inclui os despertares informativos. */
  enqueued?: string[] | null;
  /** Missões que falharam ao enfileirar. Não esconde as outras. */
  failed?: string[] | null;
  /** O agregado é do chamador, não global. */
  scope?: string;
}

/** Falha parcial declarada: nada foi enfileirado e houve falhas. */
export class WakeupsSemEfeitoError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "WakeupsSemEfeitoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  // O servidor responde 500 quando `enqueued` esta' vazio E houve falhas. O `agentFetch` lanca
  // no erro HTTP sem expor o corpo, por isso o painel reconhece a assinatura da falha parcial --
  // e nomeia-a como tal, em vez de a mostrar como erro de rede.
  if (/500|internal|enfileir|wakeup|falha/i.test(mensagem)) {
    throw new WakeupsSemEfeitoError(
      "nada foi enfileirado e houve falhas -- isto não é sucesso parcial",
    );
  }
  throw causa;
}

/** Quantos despertares, quantos enfileirados, quantos falharam. */
export function resumirDespertares(r: RespostaDeWakeups): string {
  const despertares = (r.wakeups ?? []).length;
  const enfileirados = (r.enqueued ?? []).length;
  const falhas = (r.failed ?? []).length;

  if (despertares === 0 && enfileirados === 0 && falhas === 0) {
    return "nenhuma missão precisou de despertar";
  }
  const partes: string[] = [];
  if (despertares > 0) partes.push(`${despertares} desperto(s)`);
  if (enfileirados > 0) partes.push(`${enfileirados} enfileirado(s)`);
  if (falhas > 0) partes.push(`${falhas} FALHA(S)`);
  return partes.join(" · ");
}

/**
 * A diferenca que o painel nao pode colapsar: despertares informativos nao viram trabalho.
 */
export function resumirInformativo(r: RespostaDeWakeups): string | null {
  const despertares = (r.wakeups ?? []).length;
  const enfileirados = (r.enqueued ?? []).length;
  if (despertares === 0) return null;
  const informativos = Math.max(0, despertares - enfileirados);
  if (informativos === 0) return null;
  return `${informativos} despertar(es) informativo(s) — NÃO foram enfileirados (esperam decisão humana ou são só aviso)`;
}

/** Um despertar informativo declarado explicitamente pelo servidor. */
export function eInformativo(d: DespertarDeAgente): boolean {
  return d.informative === true || d.informational === true;
}

/** Identificador de um despertar, sem inventar. */
export function idDoDespertar(d: DespertarDeAgente, i: number): string {
  const id = (d.mission_id ?? d.id ?? "").trim();
  return id === "" ? `despertar-${i + 1}` : id;
}

/** Motivo do despertar, sem inventar. */
export function motivoDoDespertar(d: DespertarDeAgente): string {
  const m = (d.reason ?? d.motivo ?? "").trim();
  return m === "" ? "motivo não declarado" : m;
}

/** `scope` e' declarado, nunca presumido global. */
export function descreverEscopo(r: RespostaDeWakeups): string {
  const s = (r.scope ?? "").trim();
  return s === "" ? "o servidor não declarou o escopo deste agregado" : `escopo: ${s}`;
}

export async function despertarAgentes(): Promise<RespostaDeWakeups> {
  try {
    return await agentFetch<RespostaDeWakeups>("/api/agent/v1/wakeups", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
  } catch (causa) {
    traduzir(causa);
  }
}
