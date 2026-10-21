// Horários oferecíveis -- `GET /scheduling/slots`.
//
// O HANDLER CORRIGE UMA PROMESSA ANTIGA (ADV-60)
//
// O comentario do handler:
//
//     "usa OpenSlots, nao Availability. Availability decide apenas por agendamentos -- nao olha
//      turno, fechamento por data, intervalo nem fuso; um slot marcado livre ali pode estar as
//      03:00 de um domingo com a empresa fechada."
//     "A funcao nao mudou de comportamento -- mudou de promessa."
//
// E o `OpenSlots` diz o que faz:
//
//     "horarios REALMENTE livres do recurso num dia local: dentro do turno, fora de fechamento e
//      descontando o que ja esta ocupado."
//     "Um horario que nao caiba inteiro no turno NAO e' oferecido."
//
//   *Um horario "livre" que esta' fora do turno nao e' livre: e' um buraco na agenda que ninguem
//   pode usar. A diferenca entre `Availability` e `OpenSlots` e' a diferenca entre "nao ha'
//   reuniao marcada" e "a empresa esta' aberta".*
//
// `date` E' OBRIGATORIO. Vazio nao significa "hoje": significa 400. Um painel que assuma "hoje"
// quando o utilizador nao escolheu data esta' a inventar o dia.
//
// `step` e' o granulado (1..240 min, default 30). O teto existe porque *"passo absurdo geraria
// centenas de milhares de slots"*.
//
// O campo `free` EXISTE no tipo. Numa lista de horarios abertos, um slot com `free=false` e' uma
// contradicao que o painel mostra em vez de esconder.
import { agentFetch } from "@/lib/agenticClient";

export interface HorarioOferecivel {
  starts_at: string;
  ends_at: string;
  /** Numa lista de horarios ABERTOS, `false` aqui e' contradicao -- mostrar, nao esconder. */
  free: boolean;
}

export interface RespostaDeHorarios {
  slots?: HorarioOferecivel[] | null;
  count?: number;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class HorariosAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "HorariosAusenteError";
  }
}

/** O pedido foi recusado (data em falta/invalida, step fora do intervalo). */
export class HorariosInvalidosError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "HorariosInvalidosError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new HorariosAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  if (/400|date|step|obrigat[oó]ri|entre 1 e 240|AAAA-MM-DD/i.test(mensagem)) {
    throw new HorariosInvalidosError(mensagem);
  }
  throw causa;
}

/** `date` vazia NAO e' "hoje": e' um pedido que o servidor recusa. */
export function validarData(data: string | undefined): string | null {
  const d = (data ?? "").trim();
  if (d === "") return "a data é obrigatória (AAAA-MM-DD) — vazio não significa hoje";
  if (!/^\d{4}-\d{2}-\d{2}$/.test(d)) return "a data precisa estar no formato AAAA-MM-DD";
  return null;
}

/** O passo (granulado) dentro do intervalo do servidor, ou `null` para usar o default. */
export function validarPasso(bruto: string | number | undefined): string | null {
  if (bruto === undefined || bruto === null || String(bruto).trim() === "") return null;
  const n = Number(bruto);
  if (!Number.isInteger(n) || n <= 0 || n > 240) {
    return "o passo precisa ser um inteiro entre 1 e 240 minutos";
  }
  return null;
}

/** A duração de cada slot, em minutos. */
export function duracaoEmMinutos(s: HorarioOferecivel): number | null {
  const ini = Date.parse(s.starts_at ?? "");
  const fim = Date.parse(s.ends_at ?? "");
  if (Number.isNaN(ini) || Number.isNaN(fim) || fim <= ini) return null;
  return Math.round((fim - ini) / 60000);
}

/** Horários que a lista diz abertos mas o campo contradiz. */
export function contarContradicoes(r: RespostaDeHorarios): number {
  return (r.slots ?? []).filter((s) => s.free === false).length;
}

/** O que a lista promete, dito sem exagerar. */
export function descreverHorarios(r: RespostaDeHorarios): string {
  const n = r.count ?? (r.slots ?? []).length;
  if (n === 0) {
    return "nenhum horário oferecível neste dia — não é o mesmo que o dia estar livre";
  }
  return `${n} horário(s) REALMENTE oferecível(is) — dentro do turno, fora de fechamento e livre de ocupação`;
}

/** A duração, conferida contra o passo que o servidor aplicou. */
export function descreverDuracoes(r: RespostaDeHorarios, passoMinutos: number): string {
  const slots = r.slots ?? [];
  if (slots.length === 0) return "sem horários para medir";
  const duracoes = slots.map(duracaoEmMinutos);
  const invalidas = duracoes.filter((d) => d === null).length;
  if (invalidas > 0) return `${invalidas} horário(s) com início/fim inválidos ou invertidos`;
  const distintas = Array.from(new Set(duracoes as number[]));
  return distintas.length === 1
    ? `todos os horários duram ${distintas[0]} min (passo pedido: ${passoMinutos})`
    : `durações variadas: ${distintas.join(', ')} min (passo pedido: ${passoMinutos})`;
}

export async function listarHorarios(
  data: string,
  passo?: number | string,
  resourceId?: string,
  serviceId?: string,
): Promise<RespostaDeHorarios> {
  const erroData = validarData(data);
  if (erroData) throw new HorariosInvalidosError(erroData);

  const erroPasso = validarPasso(passo);
  if (erroPasso) throw new HorariosInvalidosError(erroPasso);

  const params = new URLSearchParams({ date: data.trim() });
  if (passo !== undefined && String(passo).trim() !== "") params.set("step", String(passo).trim());
  if (resourceId && resourceId.trim() !== "") params.set("resource_id", resourceId.trim());
  if (serviceId && serviceId.trim() !== "") params.set("service_id", serviceId.trim());

  try {
    return await agentFetch<RespostaDeHorarios>(
      `/api/agent/v1/scheduling/slots?${params.toString()}`,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
