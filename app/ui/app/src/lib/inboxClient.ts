// Inbox -- conversas, mensagens, metricas e bloqueio por consentimento.
//
// Decima terceira, decima quarta e decima quinta rotas do E-1319 a chegar ao produto:
// `/inbox/conversations`, `/inbox/conversations/:id` e `/inbox/metrics`.
//
// `inbox` era o maior dominio restante (**8 rotas**), com `inbox.go` dezenas de KB e um store
// com SLA, atribuicao, handoff e conhecimento. Nao havia componente `*nbox*`.
//
// OS CONTRATOS:
//
//	GET /inbox/conversations?status=  -> 200 {conversations:[...], count:N}   503
//	GET /inbox/conversations/:id      -> 200 {conversation, messages, reply_blocked?, reply_blocked_reason?}
//	GET /inbox/metrics                -> 200 InboxMetrics (objecto directo)   503
//
// QUATRO DECISOES QUE O PAINEL NAO PODE ERRAR:
//
//  1. **`reply_blocked` E' UM ESTADO, NAO UM ERRO.** O servidor anexa-o quando o
//     consentimento do contacto nao autoriza resposta naquele canal:
//
//         // O consentimento do contato e' anexado a visao para o atendente saber,
//         // sem consultar outro lugar, se pode responder.
//
//     A conversa continua visivel e legivel. O que esta' bloqueado e' RESPONDER.
//     *Uma conversa que se pode ler e nao se pode responder nao e' uma conversa
//     bloqueada: e' uma conversa com uma porta fechada.*
//
//  2. **`deduplicated` NAO E' UMA MENSAGEM NOVA.** O comentario do servidor:
//     «Deduplicated marca a mensagem devolvida quando a entrega se repetiu.» Contar
//     uma reentrega como mensagem nova inflaciona a conversa.
//
//  3. **`last_message` E' UMA PREVIA, NAO A CONVERSA.** O proprio tipo diz: «LastMessage
//     e' uma previa curta para a lista nao precisar carregar tudo.» O painel usa-a
//     como previa e diz que e' previa.
//
//  4. **SEGUNDOS NAO SAO MINUTOS.** `first_response_avg_seconds` e
//     `resolution_avg_seconds` sao segundos em ponto flutuante.
import { agentFetch } from "@/lib/agenticClient";

export interface ConversaDoInbox {
  id: string;
  organization_id: string;
  channel: string;
  contact_id: string;
  contact_name?: string;
  status: string;
  assigned_to?: string;
  assigned_kind?: string;
  tags?: string[];
  priority?: string;
  created_at: string;
  updated_at: string;
  first_inbound_at?: string;
  first_reply_at?: string;
  reply_due_at?: string;
  closed_at?: string;
  message_count: number;
  last_message?: string;
}

export interface MensagemDoInbox {
  id: string;
  conversation_id: string;
  organization_id: string;
  direction: string;
  author_type: string;
  author_id?: string;
  body: string;
  external_id?: string;
  created_at: string;
  deduplicated?: boolean;
}

export interface MetricasDoInbox {
  organization_id: string;
  total_conversations: number;
  open_conversations: number;
  pending_human: number;
  closed_conversations: number;
  sla_breached: number;
  awaiting_first_reply: number;
  first_response_avg_seconds: number;
  resolution_sla_breached: number;
  resolution_overdue: number;
  resolution_avg_seconds: number;
  messages_inbound: number;
  messages_outbound: number;
}

export interface RespostaDeConversas {
  conversations: ConversaDoInbox[] | null;
  count: number;
}

export interface DetalheDeConversa {
  conversation: ConversaDoInbox;
  messages: MensagemDoInbox[] | null;
  /** true quando o consentimento do contacto NAO autoriza responder naquele canal. */
  reply_blocked?: boolean;
  reply_blocked_reason?: string;
}

/** O inbox nao esta' ligado neste runtime (`errInboxUnavailable`). */
export class InboxAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "InboxAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/conversa n[aã]o existe|gone|503|n[aã]o est[aá] dispon[íi]vel/i.test(mensagem)) {
    throw new InboxAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Formata segundos em texto legivel: 125 -> `2 min 5 s`, 45 -> `45 s`, 0 -> `0 s`.
 *
 * O servidor mede em segundos. Mostrar `125` seria mostrar a unidade errada.
 */
export function formatarSegundos(segundos: number | undefined): string {
  const total = Math.max(0, Math.round(segundos ?? 0));
  if (total < 60) return `${total} s`;
  const minutos = Math.floor(total / 60);
  const resto = total % 60;
  if (minutos < 60) return resto === 0 ? `${minutos} min` : `${minutos} min ${resto} s`;
  const horas = Math.floor(minutos / 60);
  const minutosResto = minutos % 60;
  return minutosResto === 0 ? `${horas} h` : `${horas} h ${minutosResto} min`;
}

/** Descreve o bloqueio de resposta. Bloqueado nao e' o mesmo que ausente. */
export function descreverBloqueio(
  bloqueado: boolean | undefined,
  motivo: string | undefined,
): string {
  if (bloqueado === undefined) return "sem informacao de consentimento nesta resposta";
  if (!bloqueado) return "pode responder";
  const limpo = (motivo ?? "").trim();
  return limpo === "" ? "resposta bloqueada pelo consentimento" : `resposta bloqueada: ${limpo}`;
}

/** Etapa/estado da conversa em texto legivel, sem inventar o que nao conhece. */
export function descreverEstadoConversa(estado: string | undefined): string {
  const bruto = (estado ?? "").trim();
  if (bruto === "") return "sem estado";
  return bruto.replace(/_/g, " ");
}

/** Diz que a linha e' uma previa, e de que tamanho. */
export function descreverPrevia(conversa: ConversaDoInbox): string {
  const previa = (conversa.last_message ?? "").trim();
  if (previa === "") return "sem previa";
  return `${previa} (previa)`;
}

export async function listarConversas(
  estado?: string,
  init?: RequestInit,
): Promise<RespostaDeConversas> {
  const limpo = (estado ?? "").trim();
  const sufixo = limpo !== "" ? `?status=${encodeURIComponent(limpo)}` : "";
  try {
    // Prefixo `/api/agent/v1` explicito: o gate compara o path CRU (E-1321).
    return await agentFetch<RespostaDeConversas>(`/api/agent/v1/inbox/conversations${sufixo}`, init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function obterConversa(
  id: string,
  init?: RequestInit,
): Promise<DetalheDeConversa> {
  const limpo = id.trim();
  if (limpo === "") throw new Error("id da conversa vazio");
  try {
    return await agentFetch<DetalheDeConversa>(
      `/api/agent/v1/inbox/conversations/${encodeURIComponent(limpo)}`,
      init,
    );
  } catch (causa) {
    traduzir(causa);
  }
}

export async function metricasDoInbox(init?: RequestInit): Promise<MetricasDoInbox> {
  try {
    return await agentFetch<MetricasDoInbox>("/api/agent/v1/inbox/metrics", init);
  } catch (causa) {
    traduzir(causa);
  }
}
