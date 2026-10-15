import { agentFetch } from "./agenticClient";

/**
 * Cliente das rotas de VERSOES de artefacto (Estagio 47 / item 47).
 *
 * A superficie HTTP existe e esta' testada no servidor (`GET .../versions` e `POST .../restore`,
 * com 404/409 e o invariante de tenant a' superficie). Este ficheiro e' o elo que a UI precisava
 * - e, como no Companion, **nao deriva nada**: le^ o que o servidor diz e declara a ausencia
 * quando a rota falha ou devolve algo que nao sabe desenhar.
 */

export type VersaoDeArtefacto = {
  id: string;
  /** A versao e' o numero que o servidor guarda; *nao* a posicao na lista. */
  version: number;
  sha256?: string;
  size?: number;
  media_type?: string;
  path?: string;
};

export type VersoesDoArtefacto = {
  artefactoID: string;
  versoes: VersaoDeArtefacto[];
  /** O historico e' POR MISSAO (o indice e' `mission.Artifacts`); o servidor declara-o. */
  scope: string;
};

type RespostaDeVersoes = {
  mission_id: string;
  artifact_id: string;
  versions?: unknown;
  scope?: string;
};

function versaoValida(valor: unknown): valor is VersaoDeArtefacto {
  if (!valor || typeof valor !== "object") return false;
  const v = valor as Record<string, unknown>;
  return typeof v.id === "string" && typeof v.version === "number";
}

/**
 * Le^ o historico de versoes de um artefacto.
 *
 * *O servidor pode recusar (400 quando o artefacto nao tem tenant, 404 quando nao e' desta
 * missao). Nesses casos o erro SOBE - a UI tem de o mostrar, nao inventar uma lista vazia.*
 */
export async function carregarVersoesDoArtefacto(
  missionID: string,
  artefactoID: string,
): Promise<VersoesDoArtefacto> {
  const resposta = await agentFetch<RespostaDeVersoes>(
    `/api/agent/v1/missions/${encodeURIComponent(missionID)}/artifacts/${encodeURIComponent(artefactoID)}/versions`,
  );
  const lista = Array.isArray(resposta.versions)
    ? resposta.versions.filter(versaoValida)
    : [];
  return {
    artefactoID: resposta.artifact_id ?? artefactoID,
    versoes: lista,
    scope: resposta.scope ?? "indisponivel (o servidor nao declarou o escopo)",
  };
}

/**
 * Restaura uma versao anterior.
 *
 * `expectedVersion` e' o CAS do ARTEFACTO (obrigatorio no corpo): *sem ele o servidor recusa com
 * 409, e e' isso que impede dois clientes de restaurarem sobre a mesma versao corrente.*
 */
export async function restaurarVersaoDoArtefacto(
  missionID: string,
  artefactoID: string,
  versao: number,
  expectedVersion: number,
): Promise<VersaoDeArtefacto | null> {
  const resposta = await agentFetch<{ restored?: unknown }>(
    `/api/agent/v1/missions/${encodeURIComponent(missionID)}/artifacts/${encodeURIComponent(artefactoID)}/restore`,
    {
      method: "POST",
      body: JSON.stringify({
        version: versao,
        expected_version: expectedVersion,
      }),
    },
  );
  return versaoValida(resposta.restored) ? resposta.restored : null;
}

/** Mensagem honesta quando a rota de versoes nao respondeu - *nunca uma lista inventada.* */
export function versoesIndisponiveis(motivo: string): VersoesDoArtefacto {
  return {
    artefactoID: "",
    versoes: [],
    scope: `indisponivel (nao foi possivel ler as versoes: ${motivo})`,
  };
}
