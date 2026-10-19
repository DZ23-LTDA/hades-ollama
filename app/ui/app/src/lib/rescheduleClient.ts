// Horários livres para REMARCAR -- `GET /scheduling/reschedule/slots`.
//
// A PECA QUE QUASE SEMPRE FALTA, E O HANDLER DIZ PORQUE
//
//     "remarcar sem saber o que esta' livre obriga o cliente a adivinhar, e a adivinhacao vira
//      horario inexistente -- que o motor recusa DEPOIS, quando o cliente ja' escolheu. Aqui o
//      motor responde ANTES."
//
//   *Um fluxo de remarcacao que so' descobre o conflito quando o cliente confirma nao e' um fluxo:
//   e' uma armadilha com dois passos. O erro chega depois do esforco.*
//
// DOIS CAMPOS OBRIGATORIOS, E NAO UM
//
// `appointment_id` **e** `date`. Sem o primeiro nao se sabe o que remarcar (e o motor nao pode
// excluir o proprio agendamento da lista); sem o segundo nao ha' dia. Os dois em falta davam o
// mesmo 400 -- o painel valida os dois separadamente para dizer QUAL falta.
//
// `step` tem o mesmo tecto de 240 do `/scheduling/slots`, pelo mesmo motivo.
//
// DIFERENCA PARA `/scheduling/slots`
//
// `/slots` responde para um recurso/servico num dia -- serve para MARCAR.
// Esta rota responde para um **agendamento concreto** -- serve para REMARCAR, e por isso conhece o
// agendamento que esta' a ser movido. *A mesma pergunta com um sujeito diferente nao e' a mesma
// pergunta: uma escolhe onde caber, a outra escolhe para onde mover.*
import { agentFetch } from "@/lib/agenticClient";

export interface HorarioParaRemarcar {
  starts_at: string;
  ends_at: string;
  free: boolean;
}

export interface RespostaDeRemarcacao {
  slots?: HorarioParaRemarcar[] | null;
  count?: number;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class RemarcacaoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "RemarcacaoAusenteError";
  }
}

/** O pedido foi recusado (appointment_id/date em falta, step fora do intervalo). */
export class RemarcacaoInvalidaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "RemarcacaoInvalidaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new RemarcacaoAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  if (/400|appointment_id|date|step|obrigat[oó]ri|entre 1 e 240|AAAA-MM-DD/i.test(mensagem)) {
    throw new RemarcacaoInvalidaError(mensagem);
  }
  throw causa;
}

/** Os dois campos obrigatorios, ditos SEPARADAMENTE: qual falta. */
export function validarPedido(
  appointmentId: string | undefined,
  data: string | undefined,
): string | null {
  const id = (appointmentId ?? "").trim();
  const d = (data ?? "").trim();
  const faltas: string[] = [];
  if (id === "") faltas.push("o agendamento (appointment_id)");
  if (d === "") faltas.push("a data (AAAA-MM-DD)");
  if (faltas.length > 0) {
    return `falta ${faltas.join(" e ")} — os dois são obrigatórios para remarcar`;
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(d)) return "a data precisa estar no formato AAAA-MM-DD";
  return null;
}

/** O passo (granulado) dentro do intervalo do servidor. */
export function validarPasso(bruto: string | number | undefined): string | null {
  if (bruto === undefined || bruto === null || String(bruto).trim() === "") return null;
  const n = Number(bruto);
  if (!Number.isInteger(n) || n <= 0 || n > 240) {
    return "o passo precisa ser um inteiro entre 1 e 240 minutos";
  }
  return null;
}

/** O que a lista significa, dito sem prometer demais. */
export function descreverDisponibilidade(r: RespostaDeRemarcacao): string {
  const n = r.count ?? (r.slots ?? []).length;
  if (n === 0) {
    return "nenhum horário livre para remarcar neste dia — o motor respondeu antes de o cliente escolher";
  }
  return `${n} horário(s) livre(s) para remarcar — o motor responde ANTES de o cliente escolher`;
}

/**
 * A lista devia conter apenas horários livres. Um `free=false` aqui e' a mesma contradicao do
 * `/slots`: aparece numa lista de "livres" e declara-se ocupado.
 */
export function contarNaoLivres(r: RespostaDeRemarcacao): number {
  return (r.slots ?? []).filter((s) => s.free === false).length;
}

/** Duracoes, medidas -- nao assumidas do passo. */
export function descreverDuracoes(r: RespostaDeRemarcacao): string {
  const slots = r.slots ?? [];
  if (slots.length === 0) return "sem horários para medir";
  const duracoes: number[] = [];
  let invalidos = 0;
  for (const s of slots) {
    const ini = Date.parse(s.starts_at ?? "");
    const fim = Date.parse(s.ends_at ?? "");
    if (Number.isNaN(ini) || Number.isNaN(fim) || fim <= ini) {
      invalidos++;
      continue;
    }
    duracoes.push(Math.round((fim - ini) / 60000));
  }
  if (invalidos > 0) return `${invalidos} horário(s) com início/fim inválidos ou invertidos`;
  const distintas = Array.from(new Set(duracoes));
  return distintas.length === 1
    ? `todos os horários duram ${distintas[0]} min`
    : `durações variadas: ${distintas.join(', ')} min`;
}

/** O agendamento a remarcar, e o dia -- o par que a rota exige. */
export function descreverAlvo(appointmentId: string, data: string): string {
  const id = (appointmentId ?? "").trim();
  const d = (data ?? "").trim();
  if (id === "" || d === "") return "alvo incompleto: a rota exige agendamento e data";
  return `remarcar ${id} para um horário de ${d}`;
}

export async function listarHorariosParaRemarcar(
  appointmentId: string,
  data: string,
  passo?: number | string,
): Promise<RespostaDeRemarcacao> {
  const erroPedido = validarPedido(appointmentId, data);
  if (erroPedido) throw new RemarcacaoInvalidaError(erroPedido);

  const erroPasso = validarPasso(passo);
  if (erroPasso) throw new RemarcacaoInvalidaError(erroPasso);

  const params = new URLSearchParams({
    appointment_id: appointmentId.trim(),
    date: data.trim(),
  });
  if (passo !== undefined && String(passo).trim() !== "") params.set("step", String(passo).trim());

  try {
    return await agentFetch<RespostaDeRemarcacao>(
      `/api/agent/v1/scheduling/reschedule/slots?${params.toString()}`,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
