// Marcar agendamento -- `POST /scheduling/book`.
//
// E' UMA ESCRITA: O PAINEL NAO A DISPARA AO MONTAR.
//
// E-456: O CONTRATO DA SUPERFICIE E' SEPARADO DO MOTOR, E O MOTIVO E' UM MODO DE FALHA
//
// O comentario do handler:
//
//     "Expor a struct INTERNA como contrato de API acopla a API a implementacao -- e o modo de
//      falha e' silencioso: o pedido chega vazio, e o erro culpa os campos que o cliente mandou."
//
// O corpo manda `resource_id`/`starts_at` (snake_case); o motor espera `ResourceID`/`StartsAt`
// porque a struct interna **nao tem json tags**. Ligar um ao outro faz o corpo virar um **zero
// silencioso** e o 400 culpar os campos que o cliente mandou CERTOS.
//
//   *Um contrato que traduz explicitamente pode ser corrigido num sitio. Um contrato que partilha a
//   struct interna parte-se em dois sitios ao mesmo tempo, e so' um deles da' erro.*
//
// E-458: REENTREGA NAO E' ERRO DO CLIENTE
//
// O handler responde **409 com o MESMO recurso** (`idempotent_replay: true`) quando o
// `idempotency_key` ja' foi usado -- nao um 400 que sugere pedido malformado.
//
//     "Idempotencia de verdade e' devolver o MESMO RECURSO. Um cliente que reentrega o pedido --
//      porque a rede caiu, porque o agente repetiu -- quer o agendamento, nao uma licao sobre o
//      formato do pedido."
//
// ⚠️ E AQUI HA' UM LIMITE REAL DA CAMADA DE TRANSPORTE: o `agentFetch` **lanca** em erro HTTP e
// **nao expoe o corpo**. O 409 traz o agendamento, mas o painel nao o consegue ler. Por isso o
// painel trata o 409 como **reentrega detectada** e diz isso -- sem inventar o id que nao recebeu.
//
//   *Um painel que inventasse o id do agendamento existente estaria a adivinhar exactamente aquilo
//   que a idempotencia existe para nao adivinhar.*
//
// `busy_until` NAO E' `ends_at`
//
// O agendamento tem os dois. `busy_until` **inclui o buffer** -- e' o intervalo real em que o
// recurso nao pode receber outro atendimento. Mostrar `ends_at` como se fosse o fim da ocupacao
// esconde o buffer.
import { agentFetch } from "@/lib/agenticClient";

export interface AgendamentoCriado {
  id: string;
  organization_id?: string;
  resource_id: string;
  service_id: string;
  contact_id?: string;
  channel?: string;
  starts_at: string;
  ends_at: string;
  /** Inclui o buffer: e' o intervalo REAL de ocupacao. Nao e' `ends_at`. */
  busy_until: string;
  idempotency_key?: string;
  status: string;
  created_at: string;
}

export interface RespostaDeMarcacao {
  appointment: AgendamentoCriado;
  idempotent_replay?: boolean;
}

/** O motor de agendamento nao esta' ligado neste runtime. */
export class MarcacaoAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MarcacaoAusenteError";
  }
}

/** O pedido foi recusado pelo servidor (campo invalido, conflito real, etc.). */
export class MarcacaoInvalidaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MarcacaoInvalidaError";
  }
}

/** A reentrega foi detectada: o agendamento ja' existia. NAO e' falha. */
export class ReentregaDetectadaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "ReentregaDetectadaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o est[aá] ligado|motor de agendamento/i.test(mensagem)) {
    throw new MarcacaoAusenteError("O motor de agendamento não está ligado neste runtime.");
  }
  // 409 = reentrega. O corpo traz o agendamento, mas o transporte descarta-o: dizemos o que sabemos.
  if (/409|conflit|idempot|duplicat|j[aá] existe/i.test(mensagem)) {
    throw new ReentregaDetectadaError(
      "Reentrega detectada: este pedido já tinha sido aceite antes. O agendamento existente NÃO foi duplicado.",
    );
  }
  if (/400|obrigat[oó]ri|starts_at|RFC 3339|inv[aá]lid/i.test(mensagem)) {
    throw new MarcacaoInvalidaError(mensagem);
  }
  throw causa;
}

/** `starts_at` tem de ser RFC 3339 -- o servidor recusa outro formato. */
export function validarInicio(startsAt: string | undefined): string | null {
  const s = (startsAt ?? "").trim();
  if (s === "") return "o início é obrigatório (RFC 3339, ex.: 2026-10-14T10:00:00Z)";
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(s)) {
    return "o início precisa ser RFC 3339 (ex.: 2026-10-14T10:00:00Z)";
  }
  return null;
}

/** Recurso e serviço identificam O QUE se marca: sem eles o motor nao sabe onde. */
export function validarAlvo(resourceId: string | undefined, serviceId: string | undefined): string | null {
  const faltas: string[] = [];
  if ((resourceId ?? "").trim() === "") faltas.push("o recurso (resource_id)");
  if ((serviceId ?? "").trim() === "") faltas.push("o serviço (service_id)");
  if (faltas.length > 0) return `falta ${faltas.join(" e ")}`;
  return null;
}

/** A chave de idempotência: sem ela, uma reentrega cria um SEGUNDO agendamento. */
export function descreverIdempotencia(chave: string | undefined): string {
  const k = (chave ?? "").trim();
  if (k === "") {
    return "sem chave de idempotência: uma reentrega vai criar um SEGUNDO agendamento";
  }
  return `reentrega com a mesma chave devolve o MESMO agendamento, não uma cópia`;
}

/** O buffer, medido: quanto tempo depois do fim o recurso continua ocupado. */
export function bufferEmMinutos(a: AgendamentoCriado): number | null {
  const fim = Date.parse(a.ends_at ?? "");
  const ocupadoAte = Date.parse(a.busy_until ?? "");
  if (Number.isNaN(fim) || Number.isNaN(ocupadoAte)) return null;
  return Math.round((ocupadoAte - fim) / 60000);
}

/** O que o agendamento ocupa, com o buffer dito e nao escondido. */
export function descreverOcupacao(a: AgendamentoCriado): string {
  const fim = a.ends_at ?? "";
  const ocupado = a.busy_until ?? "";
  const b = bufferEmMinutos(a);
  if (b === null) return `atende de ${a.starts_at} a ${fim}`;
  if (b === 0) return `atende de ${a.starts_at} a ${fim} · sem buffer`;
  return `atende de ${a.starts_at} a ${fim} · recurso ocupado até ${ocupado} (buffer de ${b} min)`;
}

/** O que a marcação devolveu, dito com o estado real. */
export function descreverMarcacao(r: RespostaDeMarcacao): string {
  if (r.idempotent_replay) {
    return `reentrega: o agendamento ${r.appointment?.id ?? "(sem id)"} já existia`;
  }
  return `agendamento ${r.appointment?.id ?? "(sem id)"} criado · estado: ${r.appointment?.status ?? "não declarado"}`;
}

/** Marca. E' uma ESCRITA: so' com gesto do utilizador. */
export async function marcarAgendamento(entrada: {
  resourceId: string;
  serviceId: string;
  startsAt: string;
  contactId?: string;
  channel?: string;
  idempotencyKey?: string;
}): Promise<RespostaDeMarcacao> {
  const erroAlvo = validarAlvo(entrada.resourceId, entrada.serviceId);
  if (erroAlvo) throw new MarcacaoInvalidaError(erroAlvo);

  const erroInicio = validarInicio(entrada.startsAt);
  if (erroInicio) throw new MarcacaoInvalidaError(erroInicio);

  const corpo: Record<string, string> = {
    resource_id: entrada.resourceId.trim(),
    service_id: entrada.serviceId.trim(),
    starts_at: entrada.startsAt.trim(),
  };
  if (entrada.contactId && entrada.contactId.trim() !== "") corpo.contact_id = entrada.contactId.trim();
  if (entrada.channel && entrada.channel.trim() !== "") corpo.channel = entrada.channel.trim();
  // A chave de idempotência viaja no corpo: e' ela que faz a reentrega devolver o mesmo recurso.
  if (entrada.idempotencyKey && entrada.idempotencyKey.trim() !== "") {
    corpo.idempotency_key = entrada.idempotencyKey.trim();
  }
  // A organizacao NAO vai no corpo: vem do middleware. *Um corpo que escolhe o tenant escolhe o tenant.*

  try {
    return await agentFetch<RespostaDeMarcacao>("/api/agent/v1/scheduling/book", {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
