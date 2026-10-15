import { agentFetch } from "./agenticClient";

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
 * *PENDENTE DE CONSOLIDACAO (medido):* `agenticClient` ja' expoe `createMission`, `runMission` e
 * `decideMissionApproval` com estes mesmos contratos - **estas tres funcoes sao duplicacao minha e
 * devem passar a delegar**, para nao haver dois clientes da mesma API.
 */

const BASE = "/api/agent/v1";

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
  const resposta = await agentFetch<RespostaDeMissao>(`${BASE}/missions`, {
    method: "POST",
    body: JSON.stringify(corpo),
  });
  return { id: identificador(resposta) };
}

/** Executa a missao. *A midia nao corre por um pedido directo: corre dentro dos passos.* */
export async function executarMissaoDeMidia(missionID: string): Promise<void> {
  await agentFetch(`${BASE}/missions/${encodeURIComponent(missionID)}/run`, {
    method: "POST",
  });
}

/**
 * Decide uma aprovacao da missao.
 *
 * Contrato **medido** na rota real: `agent_routes.go:878` ⇒ `decideApproval` (`:4285`), cujo struct
 * (`:4287`/`:4289`) e' `{ decision, reason, nonce }` — ***nao* `approved`**. Enviar `approved`
 * deixava `decision` vazia e o servidor recusava com 400 antes de aprovar.
 * O `nonce` e' o que impede aprovar uma aprovacao que ja mudou.
 */
export async function decidirAprovacaoDeMidia(
  missionID: string,
  approvalID: string,
  aprovado: boolean,
  motivo: string,
  nonce = "",
): Promise<void> {
  if (approvalID.trim() === "") {
    throw new Error("a aprovacao precisa de identificador");
  }
  await agentFetch(
    `${BASE}/missions/${encodeURIComponent(missionID)}/approvals/${encodeURIComponent(approvalID)}`,
    {
      method: "POST",
      body: JSON.stringify({
        decision: aprovado ? "approve" : "reject",
        reason: motivo,
        nonce,
      }),
    },
  );
}
