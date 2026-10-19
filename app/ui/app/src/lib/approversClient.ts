// Aprovadores de aprovação -- `PUT /service-agents/approval-approvers`.
//
// 🚨 E-429: DEFINIR QUEM APROVA E' O ATO MAIS SENSIVEL DESTA LINHA
//
// O comentario do handler e' explicito:
//
//     🚨 E-429 (CRÍTICO): definir quem aprova é o ato mais sensível desta linha, e era o único sem
//     portão de papel — a sonda de um `RoleViewer` redefiniu a lista para `["eu-mesmo"]` e recebeu
//     HTTP 200. ⇒ agora exige owner/admin e grava o ator da sessão na trilha.
//
//   *Definir quem aprova e' definir quem pode autorizar tudo o resto. Uma lista de aprovadores que
//   qualquer um reescreve nao e' uma lista de aprovadores: e' um campo de texto que decide permissoes.*
//
// O painel nao tenta adivinhar o papel: manda o pedido e mostra o 403 como 403.
//
// LISTA VAZIA NAO E' "SEM APROVADORES": E' "SEM CHECAGEM DE PAPEL"
//
// O handler devolve:
//
//     "role_checked": len(approvers) > 0
//
// Lista vazia significa que a decisao **nao passara' por checagem de papel**. Isso nao e' um estado
// neutro: e' a ausencia do portao.
//
//   *Uma lista vazia e um portao aberto sao o mesmo numero e duas afirmacoes diferentes — e o
//   servidor tem a honestidade de o dizer em vez de deixar o painel inferir.*
//
// DUAS IDENTIDADES NA MESMA RESPOSTA
//
//     "changed_by":     strings.TrimSpace(input.ActorID)   // o que o CLIENTE declarou
//     RecordApproversChange(..., agentActorID(c), ...)     // o que a SESSAO era
//
// O `changed_by` vem do corpo; a trilha grava o ator **da sessao**. Se divergirem, a resposta mostra
// uma identidade e a auditoria guarda outra.
//
//   *Um `changed_by` que o cliente escreve e' uma declaracao, nao um facto. O facto esta' na trilha,
//   e a trilha nao aceita o corpo.*
//
// `actor_id` E' OBRIGATORIO, E O ERRO DIZ PORQUE
//
//     errServiceAgentApproversActorRequired = errors.New("definir aprovadores exige ator identificado")
//
// A TRILHA PODE FALHAR, E A RESPOSTA DIZ ISSO
//
//     "audit_recorded": auditRecorded,   // false se RecordApproversChange falhou
//     "audit_reason":   auditReason
//
//   *Um ato de politica sem registo e' um ato invisivel para quem audita. A resposta declara a falha
//   em vez de a esconder atras de um 200.*
import { agentFetch } from "@/lib/agenticClient";

export interface ResultadoDeAprovadores {
  approvers?: string[];
  count?: number;
  role_checked?: boolean;
  changed_by?: string;
  audit_recorded?: boolean;
  audit_reason?: string;
  [k: string]: unknown;
}

/** O pedido exige owner/admin e a sessao nao tem esse papel (403). */
export class AprovadoresProibidosError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "AprovadoresProibidosError";
  }
}

/** Falta o actor, ou o corpo e' invalido. */
export class AprovadoresRecusadosError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "AprovadoresRecusadosError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/403|owner or admin|forbidden|n[aã]o autorizad|permiss/i.test(mensagem)) {
    throw new AprovadoresProibidosError(
      "Definir aprovadores exige owner ou admin da organização (403). Não é uma falha do pedido: é o portão de papel a funcionar.",
    );
  }
  if (/400|ator|actor|obrigat[oó]ri|inv[aá]lid/i.test(mensagem)) {
    throw new AprovadoresRecusadosError(mensagem);
  }
  throw causa;
}

/** `actor_id` e' obrigatorio: definir aprovadores exige ator identificado. */
export function validarAtor(actorId: string | undefined): string | null {
  if ((actorId ?? "").trim() === "") {
    return "falta o ator (actor_id): definir aprovadores exige ator identificado";
  }
  return null;
}

/** Lista vazia = sem checagem de papel. Diz-se, nao se deixa inferir. */
export function descreverPortao(approvers: string[] | undefined): string {
  const n = (approvers ?? []).length;
  if (n === 0) {
    return (
      "Lista vazia: a decisão NÃO passará por checagem de papel. Uma lista vazia e um portão aberto " +
      "são o mesmo número e duas afirmações diferentes — e esta é a segunda."
    );
  }
  return `${n} aprovador(es): a decisão passará por checagem de papel.`;
}

/** O resultado, incluindo a trilha e as duas identidades. */
export function descreverResultado(r: ResultadoDeAprovadores | null): string {
  if (!r) return "o servidor respondeu sem resultado";
  const n = (r.approvers ?? []).length;
  const portao = r.role_checked ? "com portão de papel" : "SEM portão de papel (lista vazia)";
  const quem = (r.changed_by ?? "").trim() === "" ? "(não declarado)" : r.changed_by;
  const trilha = r.audit_recorded
    ? "trilha gravada"
    : `TRILHA NÃO GRAVADA${r.audit_reason ? `: ${r.audit_reason}` : ""}`;
  return `${n} aprovador(es) · ${portao} · declarado por ${quem} · ${trilha}`;
}

/** A distincao entre o que o cliente declarou e o que a sessao era. */
export function descreverDuasIdentidades(): string {
  return (
    "O `changed_by` da resposta vem do corpo e é uma declaração do cliente; a trilha grava o ator da " +
    "SESSÃO. Se divergirem, a resposta mostra uma identidade e a auditoria guarda outra — o facto está " +
    "na trilha, e a trilha não aceita o corpo."
  );
}

/** Muda a lista de aprovadores. Ato de política, e como tal auditado. */
export async function definirAprovadores(entrada: {
  actorId: string;
  approvers: string[];
}): Promise<ResultadoDeAprovadores> {
  const erro = validarAtor(entrada.actorId);
  if (erro) throw new AprovadoresRecusadosError(erro);

  // A lista e' o que e': uma lista vazia e' um pedido legitimo de "sem portao", e o servidor diz isso.
  const limpos = entrada.approvers.map((a) => a.trim()).filter((a) => a !== "");

  try {
    return await agentFetch<ResultadoDeAprovadores>("/api/agent/v1/service-agents/approval-approvers", {
      method: "PUT",
      body: JSON.stringify({ actor_id: entrada.actorId.trim(), approvers: limpos }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
