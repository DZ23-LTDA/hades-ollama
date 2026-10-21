// Cancelar agendamento -- `POST /scheduling/cancel`.
//
// E' UMA ESCRITA DESTRUTIVA: O PAINEL NAO A DISPARA AO MONTAR.
//
// O MOTIVO E' OBRIGATORIO, E O MOTIVO E' O PROPRIO COMENTARIO DO HANDLER
//
//     "schedulingCancel cancela um agendamento. *Escrita com motivo obrigatorio - cancelamento sem
//      motivo nao e' auditavel.*"
//
// Nao e' validacao de conveniencia: e' a diferenca entre um registo e um boato. Um cancelamento sem
// motivo explica o QUE aconteceu e nao o PORQUE -- e o porque e' a unica parte que alguem vai querer
// ler seis meses depois.
//
//   *Um cancelamento sem motivo e' um facto sem explicacao guardado para sempre. Um cancelamento com
//   motivo e' uma decisao.*
//
// O `appointment_id` VAI NO CORPO, NAO NO CAMINHO
//
// A rota e' `POST /scheduling/cancel` com `{"appointment_id": ..., "reason": ...}` -- nao
// `POST /scheduling/appointments/:id/cancel`. O identificador viaja no corpo porque o corpo e' o
// contrato: um id concatenado num caminho e' um id que pode reescrever o caminho.
//
// A RESPOSTA E' 200, NAO 204
//
// O servidor devolve `{"appointment": ...}` com o agendamento JA' CANCELADO. Um 204 diria "correu
// bem" e nao diria o que ficou no lugar -- e o estado final e' a unica coisa que o cliente precisa
// de mostrar.
import { agentFetch } from "@/lib/agenticClient";
import type { AgendamentoCriado } from "@/lib/bookingClient";

export interface RespostaDeCancelamento {
  appointment: AgendamentoCriado;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class CancelamentoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "CancelamentoAusenteError";
  }
}

/** O pedido foi recusado (id em falta, motivo em falta, id inexistente, ja' cancelado). */
export class CancelamentoInvalidoError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "CancelamentoInvalidoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new CancelamentoAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  if (/400|obrigat[oó]ri|motivo|reason|n[aã]o encontrad|inexistente|j[aá] cancelad/i.test(mensagem)) {
    throw new CancelamentoInvalidoError(mensagem);
  }
  throw causa;
}

/** Sem identificador nao ha' o que cancelar. */
export function validarIdAgendamento(id: string | undefined): string | null {
  const v = (id ?? "").trim();
  if (v === "") return "falta o identificador do agendamento (appointment_id)";
  return null;
}

/**
 * Sem motivo, o cancelamento nao e' auditavel -- e o servidor recusa.
 * A validacao local existe para o utilizador saber ANTES de carregar no botao.
 */
export function validarMotivo(motivo: string | undefined): string | null {
  const v = (motivo ?? "").trim();
  if (v === "") return "o motivo é obrigatório: um cancelamento sem motivo não é auditável";
  return null;
}

/** O que o cancelamento deixou no lugar -- e o estado NAO e' "cancelado" por omissao. */
export function descreverCancelamento(r: RespostaDeCancelamento): string {
  const a = r.appointment;
  if (!a) return "o servidor respondeu sem agendamento";
  const estado = (a.status ?? "").trim() === "" ? "estado não declarado" : a.status;
  return `agendamento ${a.id} agora está ${estado}`;
}

/** Antes de cancelar, diz o que se perde: o horario que fica livre e o que ja' nao se recupera. */
export function descreverOQueSePerde(a: AgendamentoCriado | null, motivo: string): string {
  if (!a) return "sem agendamento carregado: o cancelamento actua só pelo identificador";
  const m = (motivo ?? "").trim();
  const partes = [`liberta ${a.starts_at} até ${a.busy_until}`];
  if (m === "") partes.push("sem motivo: o registo não vai dizer porque");
  else partes.push(`motivo registado: ${m}`);
  return partes.join(" · ");
}

/** Cancela. E' uma ESCRITA: so' com gesto do utilizador. */
export async function cancelarAgendamento(entrada: {
  appointmentId: string;
  reason: string;
}): Promise<RespostaDeCancelamento> {
  const erroId = validarIdAgendamento(entrada.appointmentId);
  if (erroId) throw new CancelamentoInvalidoError(erroId);

  const erroMotivo = validarMotivo(entrada.reason);
  if (erroMotivo) throw new CancelamentoInvalidoError(erroMotivo);

  try {
    return await agentFetch<RespostaDeCancelamento>("/api/agent/v1/scheduling/cancel", {
      method: "POST",
      body: JSON.stringify({
        appointment_id: entrada.appointmentId.trim(),
        reason: entrada.reason.trim(),
      }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
