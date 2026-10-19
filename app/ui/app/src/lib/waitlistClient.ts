// Lista de espera de agendamento -- `GET /scheduling/waitlist`.
//
// O CONTRATO TEM DOIS PARES DE TEMPO COM SIGNIFICADOS DIFERENTES
//
//     From / To            -- a JANELA que o cliente PEDIU
//     OfferedSlot          -- o slot que o motor OFERECEU
//     OfferExpiresAt       -- quando a oferta DEIXA DE VALER
//
//   *Um painel que mostre `From`/`To` como se fosse o horario marcado esta' a dizer ao
//   utilizador que tem hora quando ainda so' pediu uma.*
//
// E UMA OFERTA TEM PRAZO. `OfferedSlot` sem `OfferExpiresAt` seria uma promessa eterna: o
// motor pode declarar os dois, e o painel mostra o prazo. Quando o prazo passou, a oferta
// **ja' nao existe** -- mesmo que o campo continue preenchido.
//
// `Status` e' o estado na fila (`waiting`, `offered`, `booked`, ...). Nao e' booleano: uma
// entrada pode estar a espera, com oferta por aceitar, ou ja' convertida em agendamento
// (`AppointmentID` presente).
import { agentFetch } from "@/lib/agenticClient";

export interface EntradaNaFila {
  id: string;
  organization_id?: string;
  resource_id: string;
  service_id: string;
  contact_id?: string;
  channel?: string;
  /** A janela que o cliente PEDIU -- nao o slot oferecido. */
  from: string;
  to: string;
  created_at: string;
  /** Estado na fila. Nao e' booleano. */
  status: string;
  /** O slot que o motor OFERECEU, se houver. */
  offered_slot?: string;
  /** Quando a oferta deixa de valer. */
  offer_expires_at?: string;
  /** Presente quando a oferta ja' virou agendamento. */
  appointment_id?: string;
}

export interface RespostaDeFila {
  waitlist?: EntradaNaFila[] | null;
  count?: number;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class FilaAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FilaAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new FilaAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  throw causa;
}

/** Uma oferta sem prazo nao pode ser tratada como oferta valida. */
export function ofertaValida(e: EntradaNaFila, agora: Date = new Date()): boolean {
  if (!e.offered_slot) return false;
  if (!e.offer_expires_at) return false;
  const expira = Date.parse(e.offer_expires_at);
  if (Number.isNaN(expira)) return false;
  return expira > agora.getTime();
}

/** O estado da entrada, dito em portugues. */
export function descreverEstado(e: EntradaNaFila): string {
  if (e.appointment_id) return "já convertida em agendamento";
  if (e.offered_slot) {
    if (!e.offer_expires_at) return "com oferta, mas sem prazo declarado — oferta não é válida";
    return ofertaValida(e)
      ? `oferta válida até ${e.offer_expires_at}`
      : `oferta expirada em ${e.offer_expires_at}`;
  }
  const s = (e.status ?? "").trim();
  return s === "" ? "estado não declarado" : `na fila (${s})`;
}

/** A janela PEDIDA, distinta do slot oferecido. */
export function descreverJanelaPedida(e: EntradaNaFila): string {
  if (!e.from || !e.to) return "janela pedida não declarada";
  return `pediu de ${e.from} a ${e.to}`;
}

/** O que o cliente pode fazer agora, sem inventar. */
export function descreverAcao(e: EntradaNaFila): string {
  if (e.appointment_id) return "nada a fazer: já está marcado";
  if (e.offered_slot && ofertaValida(e)) {
    return `pode aceitar a oferta de ${e.offered_slot}`;
  }
  if (e.offered_slot) return "a oferta expirou — volta a aguardar";
  return "aguardar: sem oferta para aceitar";
}

export async function listarFilaDeEspera(): Promise<RespostaDeFila> {
  try {
    return await agentFetch<RespostaDeFila>("/api/agent/v1/scheduling/waitlist");
  } catch (causa) {
    traduzir(causa);
  }
}
