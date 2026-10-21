/**
 * FASE 28.7/28.8: cliente da rota `publish.rollback`.
 *
 * O QUE ESTE FICHEIRO LIGA
 *
 * A E-1276 criou a rota e provou-a com 6 testes. Mas **nenhum painel a consumia** -- e o defeito
 * recorrente desta missao ja' se reabriu catorze vezes assim:
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * O QUE ESTE CLIENTE NAO TEM, POR DESENHO
 *
 * **Nao tem parametro para o directorio.** A raiz vem de `HADES_ARTEFACTOS` no servidor, a mesma que
 * `build.smoke` usa:
 *
 *	*Duas rotas que leem o mesmo cofre nao podem ter duas fronteiras diferentes.*
 *
 * E NAO EXECUTA O ROLLBACK
 *
 * A rota devolve `executado: false` sempre. Nao ha' `executarRollback()` -- nem para devolver "nao
 * autorizado":
 *
 *	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Canal de rollback, como o backend o classifica. */
export type TipoDeRollback =
  | "WINDOWS_INSTALLER"
  | "GOOGLE_PLAY_ROLLOUT"
  | "APP_STORE_ROLLOUT"
  | "EAS_UPDATE_CHANNEL"
  | "OCI_TAG";

/** Uma versao anterior candidata a rollback. */
export interface VersaoAnterior {
  caminho: string;
  nome: string;
  tamanho_bytes: number;
  sha256: string;
  modificado_em: string;
  parece_instalador: boolean;
}

/** Resposta de `publish.rollback`. */
export interface PlanoDeRollback {
  executado: boolean;
  suportado: boolean;
  tipo: TipoDeRollback;
  alvo?: VersaoAnterior;
  total_de_candidatos: number;
  integro: boolean;
  requer_aprovacao?: string;
  explicacao: string;
  aviso: string;
  aviso_de_fronteira: string;
}

/** Lista os canais que o painel oferece. */
export function canaisDeRollback(): TipoDeRollback[] {
  return [
    "WINDOWS_INSTALLER",
    "GOOGLE_PLAY_ROLLOUT",
    "APP_STORE_ROLLOUT",
    "EAS_UPDATE_CHANNEL",
    "OCI_TAG",
  ];
}

/** Traduz o tipo para uma etiqueta legivel. */
export function descreverCanalDeRollback(tipo: TipoDeRollback): string {
  switch (tipo) {
    case "WINDOWS_INSTALLER":
      return "Instalador Windows";
    case "GOOGLE_PLAY_ROLLOUT":
      return "Google Play (rollout)";
    case "APP_STORE_ROLLOUT":
      return "App Store (rollout)";
    case "EAS_UPDATE_CHANNEL":
      return "EAS Update (canal)";
    case "OCI_TAG":
      return "Registry OCI (tag)";
    default: {
      const inesperado: never = tipo;
      throw new Error(`canal de rollback desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Pede o plano de rollback para um canal.
 *
 * O directorio **nao** vai no pedido: vem do ambiente do servidor.
 */
export async function planearRollback(tipo: TipoDeRollback): Promise<PlanoDeRollback> {
  return agentFetch<PlanoDeRollback>("/api/agent/v1/publish/rollback", {
    method: "POST",
    body: JSON.stringify({ tipo }),
  });
}

/**
 * Resume o plano numa frase.
 *
 * Nunca diz que o rollback foi feito: o plano **nao executa**.
 */
export function resumirPlanoDeRollback(plano: PlanoDeRollback): string {
  if (!plano.suportado) {
    return `Canal nao suportado neste ambiente: ${plano.explicacao}`;
  }
  if (plano.total_de_candidatos === 0) {
    return "Nenhuma versao anterior encontrada na raiz autorizada.";
  }
  const alvo = plano.alvo === undefined ? "sem alvo proposto" : plano.alvo.nome;
  return `${plano.total_de_candidatos} candidato(s); alvo proposto: ${alvo}. ` +
    "Nada foi restaurado.";
}

/**
 * Diz o que um plano aprovado prova -- e o que nao prova.
 *
 *	*Um plano nao e' uma restauracao: e' a lista do que falta antes de pedir autorizacao.*
 */
export function provaDoPlanoDeRollback(plano: PlanoDeRollback): string {
  if (plano.executado) {
    // Nao deveria acontecer: a rota garante `executado: false`. Mas se acontecer, o painel tem de o
    // dizer em vez de o esconder.
    return "ATENCAO: o servidor declarou que EXECUTOU o rollback.";
  }
  if (!plano.suportado) {
    return "nao planeavel: o canal nao existe neste ambiente";
  }
  if (!plano.integro) {
    return "plano incompleto: o alvo nao tem checksum medido";
  }
  return "planeavel: o alvo existe e tem SHA-256 medido; NAO restaurado";
}
