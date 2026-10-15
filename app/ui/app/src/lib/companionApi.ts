import { agentFetch } from "./agenticClient";

/**
 * Cliente da rota do Companion (complemento 48 / item 46).
 *
 * A rota existe e esta' testada no servidor (`GET /api/agent/v1/missions/:id/companion`).
 * Este ficheiro e' o elo que faltava do lado da UI - e nada mais: *nao deriva estado,
 * nao inventa fallback bonito.*
 */

/** Os mesmos oito estados do conjunto FECHADO do servidor (`internal/agent/companion_estado.go`). */
export type EstadoCompanion =
  | "ausente"
  | "a_planear"
  | "a_trabalhar"
  | "a_aprovar"
  | "bloqueado"
  | "concluido"
  | "falhou"
  | "cancelado";

export type EstadoDoCompanion = {
  estado: EstadoCompanion;
  /** A razao vem SEMPRE do servidor. *Um estado sem razao e' indistinguivel de um palpite.* */
  razao: string;
  /** Escopo declarado pelo servidor: *um agregado sem escopo e' lido como global.* */
  scope: string;
};

type RespostaDoCompanion = {
  mission_id: string;
  estado: string;
  razao?: string;
  scope?: string;
};

/**
 * Le' o estado REAL do Companion para uma missao.
 *
 * Se a rota falhar, o erro sobe - **de proposito**: quem chama tem de declarar a ausencia
 * (`estadoIndisponivel`), e nao mostrar um estado que o servidor nunca afirmou.
 */
export async function carregarEstadoDoCompanion(
  missionID: string,
): Promise<EstadoDoCompanion> {
  const resposta = await agentFetch<RespostaDoCompanion>(
    `/api/agent/v1/missions/${encodeURIComponent(missionID)}/companion`,
  );
  return {
    estado: resposta.estado as EstadoCompanion,
    razao: resposta.razao ?? "",
    scope: resposta.scope ?? "",
  };
}

/**
 * O que a UI mostra quando **nao conseguiu** saber o estado.
 *
 * *Um mascote que sorri quando a ligacao falhou e' pior do que um mascote ausente:* o
 * estado `ausente` com esta razao diz a verdade - nao sabemos.
 */
export function estadoIndisponivel(motivo: string): EstadoDoCompanion {
  return {
    estado: "ausente",
    razao: `Nao foi possivel saber o estado: ${motivo}`,
    scope: "indisponivel (a rota do Companion nao respondeu)",
  };
}
