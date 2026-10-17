/**
 * Fixture de E2E para a SALA (complemento 48 — Cowork/Team Chat).
 *
 * PORQUE E' QUE ISTO EXISTE
 * A Sala tem dominio, rotas HTTP, cliente, vista e testes unitarios — mas durante os rounds 62-63
 * **NAO tinha ponto de montagem na UI** (nenhum ficheiro `.tsx` nem `.ts` sob `src/` a montava), e por
 * isso este fixture foi escrito e ficou **sem consumidor**: era a peca 3 de 7 de um brief, e o estado
 * do complemento 48 era `EXISTE x DESCONECTADO`.
 *
 * **O QUE MUDOU (E-915/E-916)**: a montagem passou a existir (`AgenticSplitShell` monta a `SalaLigada`
 * na aba `artifacts`, condicional a `project_id` + `room_id`) e este fixture passou a ser **registado**
 * em `first-run-backend.mjs`. *Correr de verdade revelou **tres** defeitos que a leitura nao apanhava* —
 * todos da mesma causa (o ficheiro tinha sido escrito contra a API do **Playwright**, nao contra o
 * contrato do dispatcher):
 *
 *   1. `request.headers()` -> `TypeError: request.headers is not a function`
 *   2. `new URL(request.url())` -> `TypeError: request.url is not a function`
 *   3. `request.method()` -> a mesma classe
 *
 * E um **quarto**, de desenho: faltavam as rotas da **missao**. *Sem elas o `AgenticSplitShell` nunca
 * monta a aba `artifacts` e a Sala nunca chega a ser pedida* — a mesma licao que o
 * `companion-fixture.mjs:87-89` ja' tinha registado por escrito.
 *
 * O CONTRATO SERVIDO E' O DO PRODUTO, NAO UM INVENTADO
 *   rota   : GET /api/agent/v1/collab/:project_id/rooms/:room_id/members
 *            (server/agent_routes.go:851 -> collabRoomMembers, :2427)
 *   corpo  : { membros: MembroDaSala[] }
 *            (SalaPainel.tsx:46  resultado: { membros: MembroDaSala[] } | { erro: unknown })
 *            (SalaMembros.tsx:21 MembroDaSala = { id, tipo: "humano" | "agente", nome })
 *   rotas  : GET /api/agent/v1/collab/:project_id/rooms  (agent_routes.go:849 -> collabRooms)
 *
 * A REGRA HERDADA DO COMPANION: ISOLAMENTO POR `Referer`
 * O backend de E2E serve varios fixtures no mesmo processo. Cada um responde APENAS quando o
 * `Referer` traz a sua propria `mission_id`; caso contrario devolve `null` e o pedido segue na
 * cadeia de dispatch. Assim um fixture nunca se torna, por acidente, fonte de outro teste.
 */

export const SALA_E2E_MISSAO_ID = "mis_sala_e2e";
export const SALA_E2E_PROJECT_ID = "proj_sala_e2e";
export const SALA_E2E_ROOM_ID = "sala_e2e";

/** Dois membros: uma pessoa e um agente. E' o caso que o complemento 48 existe para cobrir. */
export const SALA_MEMBROS_PADRAO = [
  { id: "ana", tipo: "humano", nome: "Ana" },
  { id: "agente-revisor", tipo: "agente", nome: "Revisor" },
];

/**
 * A missao que faz o `AgenticSplitShell` montar a aba `artifacts` — e portanto a Sala.
 * *Sem uma missao o shell nao renderiza canvas nenhum, e a rota `/members` nunca e' pedida.*
 */
const missao = {
  id: SALA_E2E_MISSAO_ID,
  objective: "Prova E2E da Sala (complemento 48)",
  state: "RUNNING",
  workspace: "/workspace/e2e",
  plan: { steps: [] },
  approvals: [],
  artifacts: [],
  usage: {},
};

/**
 * @param {{ sendJSON: Function, membros?: Array, erro?: string }} opcoes
 * @returns {(pedido: { route: string, request: any, response: any }) => Promise<number|null>}
 */
export function createSalaFixture({ sendJSON, membros = SALA_MEMBROS_PADRAO, erro } = {}) {
  if (typeof sendJSON !== "function") {
    throw new Error(
      "createSalaFixture exige `sendJSON`. Sem ele o fixture nao consegue responder e falharia em silencio.",
    );
  }

  return async function handleSalaRequest({ route, request, response }) {
    /**
     * O `Referer` vem do `request` porque o dispatcher passa o objecto do Node `http`, onde
     * `headers` e' um **objecto** (`request.headers.referer`). *Aceita-se tambem a forma do Playwright
     * (`headers()` como funcao), porque um fixture partilhado pode ser chamado das duas maneiras.*
     */
    const cabecalhos =
      typeof (request && request.headers) === "function"
        ? request.headers()
        : (request && request.headers) || {};
    const referer = String(cabecalhos.referer || cabecalhos.Referer || "");
    // Cerca de isolamento: sem a nossa mission_id no Referer, nao e' connosco.
    if (!referer.includes(`mission_id=${SALA_E2E_MISSAO_ID}`)) return null;

    /**
     * O METODO E O CAMINHO VEEM DO `route`, NAO DO `request`.
     *
     * O dispatcher chama cada fixture com **`{ route, request, response }`**, onde `route` ja' e'
     * `` `${request.method} ${url.pathname}` ``. *Usar o `route`* e' o contrato **documentado** do
     * dispatcher, e' o que o `studio-builder-fixture.mjs` (que corre) ja' faz, e elimina qualquer
     * dependencia da forma do objecto `request` — que difere entre o Node `http` e o Playwright.
     */
    const [metodo, ...partesDoCaminho] = String(route || "").split(" ");
    const caminho = partesDoCaminho.join(" ");

    // --- rotas da MISSAO: sem elas o shell nao monta a aba e a Sala nunca e' pedida ---------------
    if (route === "GET /api/agent/v1/missions") {
      sendJSON(response, 200, { missions: [missao] });
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${SALA_E2E_MISSAO_ID}`) {
      sendJSON(response, 200, missao);
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${SALA_E2E_MISSAO_ID}/events`) {
      sendJSON(response, 200, { events: [] });
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${SALA_E2E_MISSAO_ID}/events/stream`) {
      response.writeHead(200, {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      });
      response.write(": conectado\n\n");
      return 200;
    }
    if (route === "GET /api/agent/v1/metrics") {
      sendJSON(response, 200, {});
      return 200;
    }
    if (route === "GET /api/agent/v1/tools") {
      sendJSON(response, 200, { tools: [] });
      return 200;
    }

    // --- rotas da SALA ----------------------------------------------------------------------------
    // Membros de UMA sala -> e' a rota que a `SalaLigada` pede em `carregarEstado`.
    const membrosDaSala = new RegExp(
      `^/api/agent/v1/collab/[^/]+/rooms/[^/]+/members$`,
    );
    if (metodo === "GET" && membrosDaSala.test(caminho)) {
      if (erro) {
        // Uma FALHA tem de ser visivel. O `SalaMembros` distingue os tres casos (erro,
        // carregando, vazio) precisamente para uma falha nunca virar lista vazia.
        sendJSON(response, 500, { error: erro });
        return 500;
      }
      sendJSON(response, 200, { membros });
      // E-916: **devolver o STATUS, nao o resultado do `sendJSON`.**
      // *O dispatcher faz `status = salaStatus` e escreve-o no log*; o `sendJSON` do repo devolve
      // `undefined`, e foi isso que produziu `collab/.../members -> undefined` enquanto o teste do
      // 1.0 membro falhava sem causa aparente. O `companion-fixture.mjs` ja' fazia `return 200;`
      // **depois** do `sendJSON` — a forma correcta.
      return 200;
    }

    // A lista de salas. Necessaria para escolher o `roomID` — existe no produto
    // (server/agent_routes.go:849 -> collabRooms).
    const salasDoProjecto = new RegExp(`^/api/agent/v1/collab/[^/]+/rooms$`);
    if (metodo === "GET" && salasDoProjecto.test(caminho)) {
      sendJSON(response, 200, {
        salas: [{ id: SALA_E2E_ROOM_ID, nome: "Sala de teste" }],
      });
      return 200;
    }

    return null; // nao e' nosso: segue na cadeia de dispatch
  };
}
