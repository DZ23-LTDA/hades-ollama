// Fila de espera e no-show -- `POST /scheduling/waitlist/{offer,confirm}` e `/scheduling/no-show`.
//
// TRES ESCRITAS, E A PRIMEIRA COISA A DIZER E' QUE FALTAR NAO E' CANCELAR
//
// O comentario do handler do no-show:
//
//     "schedulingNoShow marca o no-show. *Quem faltou nao e' o mesmo que quem cancelou, e a agenda precisa
//      distinguir.*"
//
// Sao dois factos diferentes com consequencias diferentes: um cancelamento avisa, um no-show nao. Se a
// agenda os juntar num so' estado, o relatorio perde a unica informacao que distingue um cliente que
// desmarcou de um cliente que nao apareceu.
//
//   *Dois estados que o servidor distingue e o painel junta sao dois estados que o utilizador perdeu.*
//
// FILA VAZIA NAO E' ERRO -- E O STATUS TEM DE DIZER ISSO
//
// O comentario do handler da oferta:
//
//     "Fila vazia nao e' erro: e' ausencia de oferta - e o status tem de dizer isso, nao 500."
//
// O servidor responde **200 com `offered: false`**. NAO e' falha: e' o resultado honesto de uma operacao
// que correu bem e nao encontrou ninguem para oferecer.
//
//   *Uma fila vazia devolvida como erro ensina o utilizador a tentar outra vez -- e a segunda tentativa
//   tambem nao vai encontrar ninguem.*
//
// `now` NAO VEM DO CORPO
//
//     "`now` NAO vem do corpo: *um corpo que escolhe "agora" escolhe o proprio prazo de expiracao da oferta.*"
//
// A oferta **expira**. Quem decide se ainda esta' a tempo e' o servidor, com o seu relogio -- nao o cliente
// com o relogio que escolheu.
//
// A OFERTA EXPIRA, E A ROTA NAO DECIDE ISSO
//
//     "A oferta EXPIRA: o motor decide, nao a rota."
//
// Confirmar tarde e' recusado pelo motor. O painel nao verifica prazos -- nao tem o relogio do servidor.
import { agentFetch } from "@/lib/agenticClient";
import { validarIdAgendamento } from "@/lib/cancellationClient";
import type { AgendamentoCriado } from "@/lib/bookingClient";

export interface OfertaDeEspera {
  entry_id?: string;
  resource_id?: string;
  slot?: string;
  expires_at?: string;
  [k: string]: unknown;
}

export interface RespostaDeOferta {
  offered: boolean;
  offer?: OfertaDeEspera;
}

export interface RespostaDeConfirmacao {
  appointment: AgendamentoCriado;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class AgendaAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "AgendaAusenteError";
  }
}

/** O motor recusou: entrada inexistente, oferta expirada, agendamento invalido. */
export class AgendaRecusadaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "AgendaRecusadaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new AgendaAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  if (/400|RFC 3339|obrigat[oó]ri|expirad|n[aã]o encontrad|inv[aá]lid|inexistente/i.test(mensagem)) {
    throw new AgendaRecusadaError(mensagem);
  }
  throw causa;
}

/** `slot` tem de ser RFC 3339 -- o servidor recusa outro formato. */
export function validarSlot(slot: string | undefined): string | null {
  const s = (slot ?? "").trim();
  if (s === "") return "o horário é obrigatório (RFC 3339)";
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(s)) {
    return "o horário precisa ser RFC 3339 (ex.: 2026-11-06T14:00:00Z)";
  }
  return null;
}

/** Recurso e horario identificam a vaga que vagou. */
export function validarVaga(resourceId: string | undefined, slot: string | undefined): string | null {
  if ((resourceId ?? "").trim() === "") return "falta o recurso (resource_id)";
  return validarSlot(slot);
}

/**
 * O resultado da oferta, dito sem transformar ausencia em falha.
 * *Fila vazia nao e' erro: e' ausencia de oferta.*
 */
export function descreverOferta(r: RespostaDeOferta): string {
  if (!r.offered) {
    return "a fila está vazia — ninguém para oferecer (não é falha: é ausência de oferta)";
  }
  const o = r.offer;
  if (!o) return "a fila ofereceu alguém · o servidor não devolveu os detalhes da oferta";
  const partes = ["a fila ofereceu"];
  if (o.entry_id) partes.push(`entrada ${o.entry_id}`);
  if (o.slot) partes.push(`para ${o.slot}`);
  if (o.expires_at) partes.push(`expira em ${o.expires_at}`);
  return partes.join(" · ");
}

/**
 * A oferta TEM prazo, e o painel nao o pode calcular: nao tem o relogio do servidor.
 * O que o painel faz e' dizer que o prazo existe e que quem decide e' o motor.
 */
export function descreverPrazoDaOferta(expiraEm: string | undefined): string {
  const e = (expiraEm ?? "").trim();
  if (e === "") return "a oferta expira; o prazo é do servidor e o painel não o calcula";
  return `a oferta expira em ${e} — confirmar depois disso é recusado pelo motor`;
}

/** Faltar e cancelar sao dois estados, e o painel nao os junta. */
export function descreverNoShow(r: { appointment: AgendamentoCriado }): string {
  const a = r.appointment;
  if (!a) return "o servidor respondeu sem agendamento";
  const estado = (a.status ?? "").trim() === "" ? "estado não declarado" : a.status;
  return `agendamento ${a.id}: marcado como FALTOU · agora está ${estado} · isto não é um cancelamento`;
}

/** Oferece o proximo da fila. E' uma ESCRITA: so' com gesto do utilizador. */
export async function oferecerProximo(entrada: {
  resourceId: string;
  slot: string;
}): Promise<RespostaDeOferta> {
  const erro = validarVaga(entrada.resourceId, entrada.slot);
  if (erro) throw new AgendaRecusadaError(erro);

  try {
    return await agentFetch<RespostaDeOferta>("/api/agent/v1/scheduling/waitlist/offer", {
      method: "POST",
      body: JSON.stringify({
        resource_id: entrada.resourceId.trim(),
        slot: entrada.slot.trim(),
        // `now` NAO vai no corpo: um corpo que escolhe "agora" escolhe o proprio prazo da oferta.
      }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Confirma a oferta. E' o motor que decide se ainda esta' a tempo. */
export async function confirmarOferta(entrada: { entryId: string }): Promise<RespostaDeConfirmacao> {
  const id = (entrada.entryId ?? "").trim();
  if (id === "") throw new AgendaRecusadaError("falta o identificador da entrada na fila (entry_id)");

  try {
    return await agentFetch<RespostaDeConfirmacao>("/api/agent/v1/scheduling/waitlist/confirm", {
      method: "POST",
      body: JSON.stringify({ entry_id: id }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Marca no-show. Quem faltou nao e' quem cancelou. */
export async function marcarNoShow(entrada: { appointmentId: string }): Promise<RespostaDeConfirmacao> {
  const erro = validarIdAgendamento(entrada.appointmentId);
  if (erro) throw new AgendaRecusadaError(erro);

  try {
    return await agentFetch<RespostaDeConfirmacao>("/api/agent/v1/scheduling/no-show", {
      method: "POST",
      body: JSON.stringify({ appointment_id: entrada.appointmentId.trim() }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
