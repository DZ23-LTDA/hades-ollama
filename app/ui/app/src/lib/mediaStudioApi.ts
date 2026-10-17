import { agentFetch, runMission } from "./agenticClient";

/**
 * Chamada do Multimodal Studio (Estagio 45 / item 45).
 *
 * *Ate' aqui eu so^ tinha o catalogo (o que existe). Aqui conduzo a operacao pelo UNICO caminho
 * medido no servidor* - uma missao com a capacidade de midia:
 *
 *   POST /missions                     {objective, workspace, capabilities:["media:execute"]}
 *   POST /missions/:id/run             executa os passos (e a tool `media.process`)
 *   POST /missions/:id/approvals/:aid  {decision: "approve"|"reject", reason, nonce}
 *
 * **NAO ha rota REST de geracao de midia** (medido: zero correspondencias em `server/*.go`), e **nao
 * a crio**: `media.process` exige aprovacao, escopo `media:execute` e workspace fixado - uma rota
 * directa contornaria exactamente essa politica.
 *
 * *CONSOLIDACAO (medido, uma parte feita nesta fatia):*
 *  - `executarMissaoDeMidia` **ja' delega** em `agenticClient.runMission`;
 *  - `criarMissaoDeMidia` ainda NAO delega: `createMission` aceita
 *    `{objective, workspace, capabilities}` mas o seu tipo de retorno e o nome do campo do
 *    identificador precisam de ser lidos antes (nao se troca um cliente por outro a' cegas);
 *  - `decidirAprovacaoDeMidia` ainda NAO delega **de proposito**: `decideMissionApproval` fixa o
 *    `reason` («Aprovado no Agentic Console»), e aqui o motivo e' dado de AUDITORIA da operacao de
 *    midia - delegar perderia informacao. Caminho correcto: acrescentar `reason?` a' funcao
 *    partilhada e depois delegar.
 */

// *Sem constante de base:* cada chamada usa o caminho LITERAL (`/api/agent/v1/...`), que e' o que
// o verificador de superficies Class A+ consegue comparar com as rotas do backend.

export type MissaoDeMidia = { id: string };

type RespostaDeMissao = { id?: unknown; mission_id?: unknown };

/** Aceita o identificador venha em `id` ou `mission_id` - *o que nao se aceita e' inventa'-lo.* */
function identificador(resposta: RespostaDeMissao): string {
  const bruto =
    typeof resposta.id === "string" ? resposta.id : resposta.mission_id;
  if (typeof bruto !== "string" || bruto.trim() === "") {
    throw new Error("o servidor nao devolveu o identificador da missao");
  }
  return bruto;
}

/**
 * Cria a missao **com a capacidade `media:execute`**, que e' o que faz o runtime registar a tool
 * `media.process` (`runtime.go:267`). *Sem esta capacidade a missao nao alcanca a midia - e' dado
 * de criacao, nao um detalhe de UI.*
 */
export async function criarMissaoDeMidia(
  objetivo: string,
  workspace: string,
  organizationID?: string,
): Promise<MissaoDeMidia> {
  if (objetivo.trim() === "") {
    throw new Error("a missao de midia precisa de um objetivo");
  }
  if (workspace.trim() === "") {
    // O servidor exige workspace fixado («pinned media workspace is required»).
    throw new Error("a missao de midia precisa de um workspace");
  }
  const corpo: Record<string, unknown> = {
    objective: objetivo,
    workspace,
    capabilities: ["media:execute"],
  };
  if (organizationID) {
    corpo.organization_id = organizationID;
  }
  // *Caminho LITERAL, nao `\${BASE}/missions`:* o verificador de superficies Class A+ compara o
  // texto do `agentFetch` com as rotas do backend e uma constante interpolada da' `NO MATCH`
  // (`Preserve Class A+ surfaces` falhou exactamente aqui). O servidor e' `/api/agent/v1/missions`.
  const resposta = await agentFetch<RespostaDeMissao>(
    "/api/agent/v1/missions",
    {
      method: "POST",
      body: JSON.stringify(corpo),
    },
  );
  return { id: identificador(resposta) };
}

/** Executa a missao. *A midia nao corre por um pedido directo: corre dentro dos passos.*
 * **Delega** em `agenticClient.runMission` (mesma rota, ja' existente) - *nao se mantem um
 * segundo cliente da mesma API.* */
export async function executarMissaoDeMidia(missionID: string): Promise<void> {
  await runMission(missionID);
}

/*
 * NAO ha aqui funcao de decidir aprovacao - **de proposito**.
 *
 * Medido: o painel cria a missao e lanca-a; *quem aprova e' a superficie de aprovacao que ja'
 * existe* (`AgenticSplitShell`/`AgenticConsole`), que recebe `MissionApproval.nonce` (`:43`) e
 * envia `{decision, nonce, reason}` (`:250`). Uma segunda funcao de aprovar - que cheguei a
 * escrever e a testar - **nao era chamada por ecra nenhum** e criaria um segundo caminho para a
 * mesma decisao. Foi removida com o seu teste: um so caminho de aprovacao e' requisito da meta.
 */
