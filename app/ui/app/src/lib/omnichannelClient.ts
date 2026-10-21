// Omnicanal -- conectores, eventos e fila de mortos (DLQ).
//
// Vigesima segunda, vigesima terceira e vigesima quarta rotas do E-1319 a chegar ao
// produto: `/omnichannel/connectors`, `/omnichannel/events` e `/omnichannel/dlq`.
//
// `omnichannel` tinha 6 rotas e nenhum componente `*omni*`/`*channel*`.
//
// OS CONTRATOS:
//
//	GET /omnichannel/connectors -> 200 {connectors, count}   503 nao configurado
//	GET /omnichannel/events     -> 200 {events, count}       503
//	GET /omnichannel/dlq        -> 200 {dead_letters, count, storage: OmnichannelDLQStatus}  503
//
// QUATRO DECISOES QUE O PAINEL NAO PODE ERRAR:
//
//  1. **`secret_env` E' O NOME DA VARIAVEL, NAO O SEGREDO.** O proprio tipo:
//
//         // SecretEnv e' o NOME da variavel de ambiente com o segredo. O valor nunca e'
//         // guardado nem devolvido pela API.
//
//     O painel mostra o NOME e diz que e' nome. *Um campo chamado `secret_env` que mostra
//     o nome da variavel e' lido como o segredo por quem tem pressa.*
//
//  2. **DURABILIDADE NAO SE PRESUME.** O handler do DLQ diz por que manda o estado junto:
//
//         // O ESTADO acompanha a lista: durabilidade nao se presume, e falha de persistencia
//         // precisa ser visivel para quem opera -- uma fila "em memoria" que se apresenta
//         // como duravel e' pior que a ausencia dela.
//
//     O painel diz **«fila em memória -- não sobrevive a reinício»** quando `persisted` for
//     falso, e mostra o ultimo erro de persistencia quando houver.
//
//  3. **`organization_id` DO EVENTO VEM DO CONECTOR, NAO DO PAYLOAD.** O tipo: *«OrganizationID
//     vem do CONECTOR, nao do payload.»* Isso e' isolamento de tenant na INGESTAO -- o painel
//     nao tem de o justificar, mas nao deve sugerir que o campo e' do remetente.
//
//  4. **A DLQ NAO E' UMA LISTA DE ERROS GENERICOS.** Cada entrada tem `reason` e `attempts`:
//     e' o que faltou para entregar, e quantas vezes se tentou.
import { agentFetch } from "@/lib/agenticClient";

export interface ConectorDeOmnicanal {
  id: string;
  organization_id: string;
  channel: string;
  name?: string;
  /** NOME da variavel de ambiente. O valor nunca e' devolvido pela API. */
  secret_env: string;
  enabled: boolean;
  created_at: string;
}

export interface EventoDeOmnicanal {
  id: string;
  organization_id: string;
  channel_id: string;
  connector_id: string;
  external_message_id: string;
  conversation_id?: string;
  contact_id: string;
  sender?: string;
  recipient?: string;
  message_type: string;
  content_reference?: string;
  attachments?: string[];
  timestamp: string;
  delivery_state: string;
}

export interface LetraMorta {
  id: string;
  organization_id: string;
  channel: string;
  event: EventoDeOmnicanal;
  reason: string;
  attempts: number;
  created_at: string;
  last_attempt_at: string;
}

export interface EstadoDaDLQ {
  organization_id: string;
  /** `false` = a fila vive em memoria e nao sobrevive a reinicio. */
  persisted: boolean;
  storage?: string;
  entries: number;
  per_organization: number;
  error?: string;
}

export interface RespostaDeConectores {
  connectors: ConectorDeOmnicanal[] | null;
  count: number;
}

export interface RespostaDeEventos {
  events: EventoDeOmnicanal[] | null;
  count: number;
}

export interface RespostaDaDLQ {
  dead_letters: LetraMorta[] | null;
  count: number;
  storage: EstadoDaDLQ;
}

/** O runtime omnicanal nao esta' configurado (`ErrOmnichannelNotConfigured`). */
export class OmnicanalAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "OmnicanalAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/n[aã]o est[aá] configurad|n[aã]o est[aá] dispon[íi]vel|503/i.test(mensagem)) {
    throw new OmnicanalAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Descreve o estado da fila de mortos, com a durabilidade declarada.
 *
 * Uma fila em memoria que se apresenta como duravel e' pior que a ausencia dela.
 */
export function descreverDurabilidade(estado: EstadoDaDLQ | undefined): string {
  if (!estado) return "sem informacao de durabilidade";
  if (!estado.persisted) return "fila em memoria -- nao sobrevive a reinicio";
  const erro = (estado.error ?? "").trim();
  if (erro !== "") return `fila persistida, mas a ultima escrita falhou: ${erro}`;
  return "fila persistida em disco";
}

/**
 * Descreve o conector: o nome do segredo, nunca o segredo.
 *
 * Devolve o NOME da variavel de ambiente, com a marca de que e' nome.
 */
export function descreverSegredo(secretEnv: string | undefined): string {
  const limpo = (secretEnv ?? "").trim();
  if (limpo === "") return "sem variavel de segredo configurada";
  return `segredo lido de ${limpo} (nome da variavel, nao o valor)`;
}

/** Limite por organizacao, legivel. */
export function descreverLimite(estado: EstadoDaDLQ | undefined): string {
  if (!estado) return "sem informacao de limite";
  const entradas = Math.max(0, estado.entries ?? 0);
  const teto = Math.max(0, estado.per_organization ?? 0);
  if (teto === 0) return `${entradas} entrada(s), sem teto declarado`;
  return `${entradas} de ${teto} entrada(s)`;
}

/** Estado de entrega em texto legivel, sem inventar o que nao conhece. */
export function descreverEntrega(estado: string | undefined): string {
  const bruto = (estado ?? "").trim();
  if (bruto === "") return "sem estado de entrega";
  return bruto.replace(/_/g, " ");
}

/** Falha de entrega: motivo e tentativas. */
export function descreverFalha(reason: string | undefined, attempts: number | undefined): string {
  const motivo = (reason ?? "").trim();
  const tentativas = Math.max(0, attempts ?? 0);
  const base = motivo === "" ? "motivo nao declarado" : motivo;
  return `${base} (${tentativas} tentativa(s))`;
}

export async function listarConectores(init?: RequestInit): Promise<RespostaDeConectores> {
  try {
    return await agentFetch<RespostaDeConectores>("/api/agent/v1/omnichannel/connectors", init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarEventos(init?: RequestInit): Promise<RespostaDeEventos> {
  try {
    return await agentFetch<RespostaDeEventos>("/api/agent/v1/omnichannel/events", init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarLetrasMortas(init?: RequestInit): Promise<RespostaDaDLQ> {
  try {
    return await agentFetch<RespostaDaDLQ>("/api/agent/v1/omnichannel/dlq", init);
  } catch (causa) {
    traduzir(causa);
  }
}
