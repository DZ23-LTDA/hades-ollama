// Fase de oportunidade -- `POST /crm/opportunities/:id/stage`.
//
// FASE TERMINAL NAO VOLTA ATRAS, E O SERVIDOR DIZ PORQUE
//
// Em `internal/agent/crm.go`:
//
//     ErrCRMStageTerminal = errors.New("crm opportunity is already closed; reopening would erase the outcome")
//
// `won` e `lost` sao terminais. Uma oportunidade fechada nao reabre.
//
//   *Reabrir uma oportunidade fechada apagaria o resultado -- e o resultado e' a unica coisa que o funil
//   existe para guardar. Um funil que se pode reabrir nao mede vendas: mede intencoes.*
//
// O painel recusa mover a partir de uma fase terminal sem chamar o servidor, e explica por que. Se o
// servidor recusar na mesma (o cliente nao sabe tudo), a recusa e' mostrada como recusa, nao como erro.
//
// O `:id` VAI NO CAMINHO, E O CORPO SO' LEVA A FASE
//
//     POST /crm/opportunities/:id/stage   { "stage": "..." }
//
// O actor vem do contexto autenticado, nao do corpo: quem mudou a fase nao se escolhe a si mesmo.
//
//   *Um actor que o cliente envia e' um actor que o cliente escolhe. Um actor que vem do token e' um
//   actor que o servidor sabe.*
import { agentFetch } from "@/lib/agenticClient";

/** As cinco fases. `won` e `lost` sao terminais. */
export const FASES = ["new", "qualified", "proposal", "won", "lost"] as const;
export type Fase = (typeof FASES)[number];

export interface OportunidadeDeCRM {
  id?: string;
  stage?: string;
  contact_id?: string;
  [k: string]: unknown;
}

/** A oportunidade ja' esta' fechada; reabrir apagaria o resultado. */
export class FaseTerminalError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FaseTerminalError";
  }
}

/** A fase pedida nao existe, ou o id nao existe, ou falta o id. */
export class FaseRecusadaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FaseRecusadaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/already closed|terminal|reopen|erase the outcome/i.test(mensagem)) {
    throw new FaseTerminalError(
      "A oportunidade já está fechada (ganha ou perdida). O servidor recusa reabrir: reabrir apagaria o resultado.",
    );
  }
  if (/400|404|inv[aá]lid|stage|not found|n[aã]o encontrad/i.test(mensagem)) {
    throw new FaseRecusadaError(mensagem);
  }
  throw causa;
}

/** `won` e `lost` fecham a oportunidade para sempre. */
export function faseTerminal(fase: string | undefined): boolean {
  const f = (fase ?? "").trim().toLowerCase();
  return f === "won" || f === "lost";
}

/** Uma fase inventada nao e' uma fase. */
export function validarFase(fase: string | undefined): string | null {
  const f = (fase ?? "").trim().toLowerCase();
  if (f === "") return "falta a fase (stage)";
  if (!(FASES as readonly string[]).includes(f)) {
    return `fase «${f}» não existe: o servidor conhece ${FASES.join(", ")}`;
  }
  return null;
}

/** O id vai no caminho -- um id vazio e' um caminho que nao identifica nada. */
export function validarOportunidade(id: string | undefined): string | null {
  if ((id ?? "").trim() === "") return "falta o id da oportunidade";
  return null;
}

/** A recusa de reabrir, dita como regra e nao como avaria. */
export function descreverRegraTerminal(): string {
  return (
    "Uma oportunidade em «ganha» ou «perdida» está fechada: o servidor recusa reabrir porque reabrir " +
    "apagaria o resultado. O painel recusa antes de chamar — a mesma regra, dita uma vez e não duas."
  );
}

/** O que mudou, dito sem inventar. */
export function descreverMudanca(o: OportunidadeDeCRM | null): string {
  if (!o) return "o servidor respondeu sem oportunidade";
  const id = (o.id ?? "").trim() === "" ? "(sem id)" : o.id;
  const fase = (o.stage ?? "").trim() === "" ? "fase não declarada" : o.stage;
  return `oportunidade ${id} · fase: ${fase}`;
}

/**
 * Muda a fase. `de` e' a fase actual conhecida pelo cliente: se for terminal, recusa localmente.
 * O servidor recusa de qualquer forma -- esta recusa e' a mesma regra, dita antes da rede.
 */
export async function mudarFase(entrada: {
  id: string;
  stage: string;
  faseActual?: string;
}): Promise<OportunidadeDeCRM> {
  const erroId = validarOportunidade(entrada.id);
  if (erroId) throw new FaseRecusadaError(erroId);
  const erroFase = validarFase(entrada.stage);
  if (erroFase) throw new FaseRecusadaError(erroFase);
  if (faseTerminal(entrada.faseActual)) {
    throw new FaseTerminalError(
      `A oportunidade já está em «${entrada.faseActual}». O servidor recusa reabrir: reabrir apagaria o resultado.`,
    );
  }

  const id = encodeURIComponent(entrada.id.trim());
  try {
    return await agentFetch<OportunidadeDeCRM>(`/api/agent/v1/crm/opportunities/${id}/stage`, {
      method: "POST",
      body: JSON.stringify({ stage: entrada.stage.trim().toLowerCase() }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
