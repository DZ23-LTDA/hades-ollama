// Lembretes de agendamento devidos -- `GET /scheduling/reminders/due`.
//
// O HANDLER TEM DOIS AVISOS QUE O PAINEL TEM DE RESPEITAR
//
//   1. **`now` NAO vem do corpo.** O comentario: *«um corpo que escolhe «agora» escolhe quem deve
//      ser avisado»*. O painel **nao envia** `now` -- quem decide o instante e' o servidor.
//   2. **A superficie tem de expor as DUAS metades.** O comentario:
//
//          *"a superficie tem de expor as DUAS metades: quem deve ser avisado, e o registo de que foi."*
//
//      **Esta rota e' SO' UMA metade.** Devolve quem deve ser avisado AGORA; o registo de que o
//      aviso foi enviado vive noutra rota (`POST /scheduling/reminders/sent`). *Um painel que
//      mostre esta lista e se chame "lembretes" esta' a chamar ao trabalho pendente o trabalho
//      feito -- e o utilizador acredita que ja' foram enviados.*
//
// `kind` E' OBRIGATORIO. Sem ele o servidor responde 400: nao ha' "todos os tipos" implicitos,
// porque um lembrete de 24h e um de 1h nao sao o mesmo aviso.
//
// `lead_minutes` e' opcional (1..100000, default 1440 = 24h). O painel deixa escolher, mas dentro
// do intervalo -- e nao o impoe por omissao silenciosa.
import { agentFetch } from "@/lib/agenticClient";

/** Um agendamento que precisa de lembrete agora (ou o registo de um ja' enviado). */
export interface LembreteDeAgendamento {
  organization_id?: string;
  appointment_id: string;
  kind: string;
  /** Presente APENAS no registo do que foi enviado, nao na lista de devidos. */
  sent_at?: string;
  /** Campos do agendamento que o motor pode juntar. */
  contact_id?: string;
  starts_at?: string;
  service_id?: string;
}

export interface RespostaDeLembretes {
  appointments?: LembreteDeAgendamento[] | null;
  count?: number;
  kind?: string;
  lead_minutes?: number;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class LembretesAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "LembretesAusenteError";
  }
}

/** O pedido foi recusado (kind em falta, lead_minutes invalido). */
export class LembretesInvalidosError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "LembretesInvalidosError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|n[aã]o est[aá] dispon[íi]vel|motor de agendamento/i.test(mensagem)) {
    throw new LembretesAusenteError(
      "O motor de agendamento não está ligado neste runtime.",
    );
  }
  if (/400|kind|lead_minutes|obrigat[oó]rio|entre 1 e 100000/i.test(mensagem)) {
    throw new LembretesInvalidosError(mensagem);
  }
  throw causa;
}

/** `kind` vazio NAO e' "todos": e' um pedido que o servidor recusa. */
export function validarTipo(kind: string | undefined): string | null {
  const t = (kind ?? "").trim();
  if (t === "") return "o tipo de lembrete é obrigatório (ex.: 24h)";
  return null;
}

/** `lead_minutes` dentro do intervalo do servidor, ou `null` para usar o default. */
export function validarAntecedencia(bruto: string | number | undefined): string | null {
  if (bruto === undefined || bruto === null || String(bruto).trim() === "") return null;
  const n = Number(bruto);
  if (!Number.isInteger(n) || n <= 0 || n > 100000) {
    return "a antecedência precisa ser um inteiro entre 1 e 100000 minutos";
  }
  return null;
}

/**
 * A metade que ESTA rota da'. A outra metade nao esta' aqui -- e o painel diz isso.
 */
export function descreverMetade(r: RespostaDeLembretes): string {
  const n = r.count ?? (r.appointments ?? []).length;
  if (n === 0) {
    return "nenhum agendamento precisa de lembrete agora — isto NÃO diz que os anteriores foram enviados";
  }
  return `${n} agendamento(s) precisam de lembrete agora — o registo do que já foi enviado está noutra rota`;
}

/** Um lembrete devido, legivel. */
export function descreverLembrete(l: LembreteDeAgendamento): string {
  const id = (l.appointment_id ?? "").trim() || "agendamento sem id";
  const partes = [id];
  if (l.starts_at) partes.push(`começa ${l.starts_at}`);
  if (l.service_id) partes.push(`serviço ${l.service_id}`);
  return partes.join(" · ");
}

/** O `kind` e a antecedencia com que o servidor respondeu. */
export function descreverJanela(r: RespostaDeLembretes): string {
  const k = (r.kind ?? "").trim();
  const m = r.lead_minutes;
  if (k === "" && m === undefined) return "o servidor não declarou o tipo nem a antecedência";
  const partes: string[] = [];
  if (k !== "") partes.push(`tipo: ${k}`);
  if (m !== undefined) partes.push(`antecedência: ${m} min`);
  return partes.join(" · ");
}

export async function listarLembretesDevidos(
  kind: string,
  leadMinutes?: number | string,
): Promise<RespostaDeLembretes> {
  const erroTipo = validarTipo(kind);
  if (erroTipo) throw new LembretesInvalidosError(erroTipo);

  const erroLead = validarAntecedencia(leadMinutes);
  if (erroLead) throw new LembretesInvalidosError(erroLead);

  const params = new URLSearchParams({ kind: kind.trim() });
  const lead = leadMinutes === undefined ? "" : String(leadMinutes).trim();
  if (lead !== "") params.set("lead_minutes", lead);
  // NAO enviamos `now`: quem escolhe "agora" e' o servidor.

  try {
    return await agentFetch<RespostaDeLembretes>(
      `/api/agent/v1/scheduling/reminders/due?${params.toString()}`,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
