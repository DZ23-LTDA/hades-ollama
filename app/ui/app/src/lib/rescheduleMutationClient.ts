// Remarcar agendamento -- `POST /scheduling/reschedule`.
//
// E' UMA ESCRITA: O PAINEL NAO A DISPARA AO MONTAR.
//
// A ROTA NAO DECIDE SE CABE -- O MOTOR DECIDE, E A ROTA PRESERVA A RECUSA
//
// O comentario do handler:
//
//     "schedulingReschedule move um agendamento para outro instante. *A recusa do motor e' preservada:
//      sobreposicao, fechamento e turno continuam sendo decididos por ele, nao por esta rota.*"
//
// Isto importa porque a tentacao e' validar no cliente: "o horario esta' dentro do turno?",
// "nao ha' sobreposicao?". O cliente NAO sabe o turno, NAO sabe os fechamentos, NAO sabe quem mais
// marcou naquele minuto. Uma validacao local que aceite o que o motor recusa cria um painel que
// promete um horario que nao existe.
//
//   *Uma rota que decide o que cabe duplica o motor com menos informacao. Uma rota que PRESERVA a
//   recusa tem uma so' fonte de verdade -- e o erro chega ao utilizador com o motivo do motor.*
//
// REMARCAR NAO E' CANCELAR+CRIAR
//
// Escreve-se `appointment_id` e `starts_at`. O agendamento mantem a IDENTIDADE -- e' o mesmo
// agendamento noutro instante, nao um novo. Quem olhar para o historico ve' uma remarcacao, nao um
// cancelamento seguido de uma marcacao.
//
//   *Duas escritas para mover um agendamento sao duas oportunidades de deixar o cliente sem nenhum.
//   Uma escrita e' uma mudanca de estado; duas sao uma janela em que o estado nao existe.*
import { agentFetch } from "@/lib/agenticClient";
import type { AgendamentoCriado } from "@/lib/bookingClient";
import { validarIdAgendamento } from "@/lib/cancellationClient";
import { validarInicio } from "@/lib/bookingClient";

export interface RespostaDeRemarcacao {
  appointment: AgendamentoCriado;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class RemarcacaoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "RemarcacaoAusenteError";
  }
}

/**
 * O motor RECUSOU o novo instante -- sobreposicao, fechamento ou turno.
 * A recusa e' do motor, e o painel mostra-a em vez de a reinterpretar.
 */
export class RemarcacaoRecusadaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "RemarcacaoRecusadaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new RemarcacaoAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  if (/400|RFC 3339|obrigat[oó]ri|sobrepos|fechament|turno|ocupado|indispon|n[aã]o encontrad/i.test(mensagem)) {
    throw new RemarcacaoRecusadaError(mensagem);
  }
  throw causa;
}

function traduzirErroDoMotor(causa: unknown): never {
  traduzir(causa);
}

/** Antes de remarcar: o que a rota valida e o que ela NAO valida. */
export function descreverFronteiraDaRemarcacao(): string {
  return (
    "A rota valida o identificador e o formato RFC 3339. Se o novo instante CABE " +
    "(sobreposição, fechamento, turno) é o motor que decide — não o painel."
  );
}

/** O que mudou, dito com os dois instantes. */
export function descreverMovimento(de: string, para: string): string {
  const a = (de ?? "").trim();
  const b = (para ?? "").trim();
  if (a === "" && b === "") return "sem instantes: nada a comparar";
  if (a === "") return `move para ${b}`;
  if (b === "") return `sai de ${a}`;
  if (a === b) return `mantém ${a} -- o novo instante é o mesmo`;
  return `de ${a} para ${b}`;
}

/** O que a remarcacao devolveu. */
export function descreverRemarcacao(r: RespostaDeRemarcacao): string {
  const ag = r.appointment;
  if (!ag) return "o servidor respondeu sem agendamento";
  return `agendamento ${ag.id} remarcado para ${ag.starts_at} · estado: ${ag.status || "não declarado"}`;
}

/** Remarca. E' uma ESCRITA: so' com gesto do utilizador. */
export async function remarcarAgendamento(entrada: {
  appointmentId: string;
  startsAt: string;
}): Promise<RespostaDeRemarcacao> {
  const erroId = validarIdAgendamento(entrada.appointmentId);
  if (erroId) throw new RemarcacaoRecusadaError(erroId);

  const erroInicio = validarInicio(entrada.startsAt);
  if (erroInicio) throw new RemarcacaoRecusadaError(erroInicio);

  try {
    return await agentFetch<RespostaDeRemarcacao>("/api/agent/v1/scheduling/reschedule", {
      method: "POST",
      body: JSON.stringify({
        appointment_id: entrada.appointmentId.trim(),
        starts_at: entrada.startsAt.trim(),
      }),
    });
  } catch (causa) {
    traduzirErroDoMotor(causa);
  }
}
