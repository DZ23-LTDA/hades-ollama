// Inbox -- `POST /inbox/messages`. A mensagem RECEBIDA, com deduplicação por `external_id`.
//
// 201 NOVA, 200 REENTREGA -- E OS DOIS SAO SUCESSO
//
//     status := http.StatusCreated
//     if message.Deduplicated {
//         status = http.StatusOK
//     }
//
// O mesmo pedido entregue duas vezes (webhook repetido, retry de rede) **nao cria mensagem duplicada**:
// devolve a original com `Deduplicated=true` e **200**. O 201 e' "criei uma mensagem"; o 200 e' "esta
// mensagem ja' existia e e' esta".
//
//   *Uma reentrega contada como mensagem nova inflaciona a caixa de entrada -- e a caixa inflacionada faz
//   o utilizador procurar uma conversa que nunca aconteceu.*
//
// O painel tem de mostrar os dois como **estados distintos**, e nao como "sucesso" generico: quem olha
// precisa de saber se a mensagem chegou agora ou se ja' ca' estava.
//
// A CHAVE DA DEDUPLICACAO E' O `external_id`, E SO' ELE
//
//     AppendInbound(org, channel, contactID, contactName, body, externalID, now)
//
// Sem `external_id` nao ha' deduplicacao -- e o servidor nao inventa uma. *Um id que o servidor inventa
// nao deduplica nada: deduplica o que o servidor ja' viu, e o que o servidor ja' viu e' o que ele
// proprio escreveu.*
import { agentFetch } from "@/lib/agenticClient";

export interface ConversaDeInbox {
  id?: string;
  channel?: string;
  contact_id?: string;
  [k: string]: unknown;
}

export interface MensagemDeInbox {
  id?: string;
  body?: string;
  deduplicated?: boolean;
  external_id?: string;
  [k: string]: unknown;
}

export interface ResultadoDeInbox {
  conversation?: ConversaDeInbox;
  message?: MensagemDeInbox;
  deduplicated?: boolean;
  [k: string]: unknown;
}

/** O store de inbox nao esta' ligado (503). */
export class InboxAusenteError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "InboxAusenteError";
  }
}

/** Faltam campos obrigatorios da mensagem recebida. */
export class InboxRecusadoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "InboxRecusadoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|inbox.*unavailable|n[aã]o est[aá']? ligado/i.test(mensagem)) {
    throw new InboxAusenteError("O armazém de conversas não está ligado neste runtime (503).");
  }
  if (/400|obrigat[oó]ri|inv[aá]lid|channel|body|contact/i.test(mensagem)) {
    throw new InboxRecusadoError(mensagem);
  }
  throw causa;
}

/** Canal e corpo: sem eles nao ha' mensagem. */
export function validarMensagem(canal: string | undefined, corpo: string | undefined): string | null {
  if ((canal ?? "").trim() === "") return "falta o canal (channel)";
  if ((corpo ?? "").trim() === "") return "falta o corpo da mensagem (body)";
  return null;
}

/** A deduplicacao depende do `external_id`; sem ele, nao ha'. */
export function descreverDeduplicacao(externalId: string | undefined): string {
  if ((externalId ?? "").trim() === "") {
    return (
      "Sem `external_id` não há deduplicação: o servidor não inventa a chave de uma mensagem que veio " +
      "de fora. Uma reentrega sem chave cria uma segunda mensagem."
    );
  }
  return `Deduplicação activa por external_id «${externalId}»: a mesma chave não cria segunda mensagem.`;
}

/** O estado real: 201 nova ou 200 reentrega. Nao e' "sucesso" generico. */
export function descreverEntrega(r: ResultadoDeInbox | null): string {
  if (!r) return "o servidor respondeu sem resultado";
  const id = (r.message?.id ?? "").trim() === "" ? "(sem id)" : r.message?.id;
  if (r.deduplicated === true) {
    return `200 · reentrega · a mensagem ${id} já existia e é esta (não foi criada outra)`;
  }
  return `201 · nova · mensagem ${id} criada`;
}

/** A mensagem recebida. 201 se nova, 200 se reentrega. */
export async function receberMensagem(entrada: {
  channel: string;
  body: string;
  contactId?: string;
  contactName?: string;
  externalId?: string;
}): Promise<ResultadoDeInbox> {
  const erro = validarMensagem(entrada.channel, entrada.body);
  if (erro) throw new InboxRecusadoError(erro);

  const corpo: Record<string, unknown> = {
    channel: entrada.channel.trim(),
    body: entrada.body,
  };
  if (entrada.contactId && entrada.contactId.trim() !== "") corpo.contact_id = entrada.contactId.trim();
  if (entrada.contactName && entrada.contactName.trim() !== "") corpo.contact_name = entrada.contactName.trim();
  if (entrada.externalId && entrada.externalId.trim() !== "") corpo.external_id = entrada.externalId.trim();

  try {
    return await agentFetch<ResultadoDeInbox>("/api/agent/v1/inbox/messages", {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
