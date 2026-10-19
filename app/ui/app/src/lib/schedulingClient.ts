// FASE 23: agendamento -- a superfície das rotas de `/scheduling` que faltava.
//
// Sexta e sétima rotas do E-1319 a chegar ao produto (`/scheduling/services` e
// `/scheduling/appointments`). O subsistema tem 14 rotas e um motor maduro
// (`scheduling.go`, `scheduling_waitlist.go`, `scheduling_hours.go`) -- e **nenhum
// componente no frontend**. `scheduling` era o maior domínio sem superfície: 14.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// OS CONTRATOS:
//
//	GET /scheduling/services     -> 200 {services:[...]}   503 motor nao ligado   400 erro
//	GET /scheduling/appointments -> 200 {appointments:[...], count:N}  503 / 400
//
// TRÊS DECISÕES QUE O PAINEL NÃO PODE ERRAR:
//
//  1. **503 NÃO É LISTA VAZIA.** O servidor diz, textualmente:
//     *"sem store não há superfície -- e dizer isso é melhor que devolver lista
//     vazia, que pareceria 'não há serviços' em vez de 'não há motor ligado'."*
//     O painel tem de mostrar os dois estados como diferentes.
//
//  2. **`busy_until` NÃO É `ends_at`.** O comentário do servidor:
//     *"BusyUntil inclui o buffer: é o intervalo REAL em que o recurso não pode
//     receber outro atendimento. Sem buffer, dois atendimentos colados viram
//     atraso em cascata -- e o sistema diria que havia vaga onde não havia."*
//
//  3. **O estado é fechado.** `booked`, `cancelled`, `no_show`. Um estado fora da
//     lista não ganha rótulo inventado.
import { agentFetch } from "@/lib/agenticClient";

export const ESTADOS_DO_AGENDAMENTO = ["booked", "cancelled", "no_show"] as const;

export interface ServicoDeAgendamento {
  id: string;
  organization_id: string;
  name: string;
  duration_minutes: number;
  buffer_minutes?: number;
}

export interface Agendamento {
  id: string;
  organization_id: string;
  resource_id: string;
  service_id: string;
  contact_id?: string;
  channel?: string;
  starts_at: string;
  ends_at: string;
  busy_until: string;
  idempotency_key?: string;
  status: string;
  created_at: string;
}

export interface RespostaDeServicos {
  services: ServicoDeAgendamento[] | null;
}

export interface RespostaDeAgendamentos {
  appointments: Agendamento[] | null;
  count: number;
}

/**
 * O motor de agendamento não está ligado neste runtime.
 *
 *	*Uma lista vazia e um motor ausente não são a mesma coisa, e mostrá-los
 *	iguais ensina o utilizador a desconfiar do que vê.*
 */
export class MotorDeAgendamentoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MotorDeAgendamentoAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/motor de agendamento n[aã]o est[aá] ligado|status(?:Code)?\s*=\s*503|\b503\b/i.test(mensagem)) {
    throw new MotorDeAgendamentoAusenteError(mensagem);
  }
  throw causa;
}

export async function listarServicosDeAgendamento(
  init?: RequestInit,
): Promise<RespostaDeServicos> {
  try {
    // Prefixo `/api/agent/v1` explícito: o gate compara o path CRU (E-1321).
    return await agentFetch<RespostaDeServicos>("/api/agent/v1/scheduling/services", init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarAgendamentos(init?: RequestInit): Promise<RespostaDeAgendamentos> {
  try {
    return await agentFetch<RespostaDeAgendamentos>("/api/agent/v1/scheduling/appointments", init);
  } catch (causa) {
    traduzir(causa);
  }
}

/**
 * O intervalo em que o RECURSO está ocupado, dito como intervalo.
 *
 * Quando há buffer, `busy_until` é depois de `ends_at`; o painel diz a diferença
 * em vez de a esconder -- o buffer é a razão pela qual dois atendimentos colados
 * não cabem.
 */
export function descreverBuffer(agendamento: Agendamento): string {
  const fim = Date.parse(agendamento.ends_at);
  const ocupadoAte = Date.parse(agendamento.busy_until);
  if (!Number.isFinite(fim) || !Number.isFinite(ocupadoAte)) return "";
  const minutos = Math.round((ocupadoAte - fim) / 60000);
  if (minutos <= 0) return "";
  return `+${minutos} min de buffer`;
}

/** A hora local do agendamento, no formato que se lê. */
export function descreverQuando(iso: string | undefined): string {
  const bruto = (iso ?? "").trim();
  if (bruto === "") return "—";
  const data = new Date(bruto);
  if (Number.isNaN(data.getTime())) return bruto;
  return data.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/**
 * Um estado fora da lista fechada aparece como tal.
 *
 *	*Um painel que inventa um rótulo para um estado que não conhece ensina o
 *	utilizador a confiar no que não foi verificado.*
 */
export function descreverEstadoDoAgendamento(status: string | undefined): string {
  const bruto = (status ?? "").trim();
  if (bruto === "") return "sem estado";
  if ((ESTADOS_DO_AGENDAMENTO as readonly string[]).includes(bruto)) {
    if (bruto === "no_show") return "falta";
    if (bruto === "cancelled") return "cancelado";
    return "marcado";
  }
  return `estado desconhecido (${bruto})`;
}

/** Conta por estado. Os três conhecidos, e os outros somados à parte. */
export function contarAgendamentos(lista: Agendamento[]): {
  total: number;
  marcados: number;
  cancelados: number;
  faltas: number;
  desconhecidos: number;
} {
  let marcados = 0;
  let cancelados = 0;
  let faltas = 0;
  let desconhecidos = 0;
  for (const a of lista) {
    if (a.status === "booked") marcados += 1;
    else if (a.status === "cancelled") cancelados += 1;
    else if (a.status === "no_show") faltas += 1;
    else desconhecidos += 1;
  }
  return { total: lista.length, marcados, cancelados, faltas, desconhecidos };
}
