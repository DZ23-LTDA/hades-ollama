// Fixture da rota do COMPANION para a prova em browser do item 46 (mascote/pet).
//
// Segue EXACTAMENTE o contrato de `media-studio-fixture.mjs` / `studio-builder-fixture.mjs`:
// quem constroi passa `{ sendJSON }`; o handler recebe `{ route, request, response }`
// e devolve o STATUS HTTP quando tratou a rota, ou `null` para o chamador responder 404.
//
// ROTA SERVIDA (existe no produto, medida em 2026-10-15):
//   `GET /api/agent/v1/missions/:mission_id/companion`
//   -> handlers em `server/agent_routes.go`; dominio em `internal/agent/companion*`
//   -> testes de servidor verdes: `go test ./server -run Companion` (6/6)
//
// CONTRATO QUE A UI EXIGE (lido em `src/lib/companionApi.ts:46-51`):
//   { mission_id: string, estado: <um dos 8>, razao?: string, scope?: string }
// Os 8 estados sao um conjunto FECHADO (`companionApi.ts:12-36`). A UI VALIDA o estado
// na fronteira e cai em `estadoIndisponivel` se ele vier fora do conjunto
// (`companionApi.ts:68-72`) — por isso o fixture NUNCA devolve um estado inventado.
//
// ISOLAMENTO POR `Referer` — *nao e' zelo, e' licao medida.* O stub do repositorio e'
// PARTILHADO por todos os specs: no job 116501392677, servir `/missions` a TODOS os testes
// fez nascer um segundo botao «Criar missao» no DOM que a `shell.spec.ts` observa e
// reprovou-a por strict mode violation. Aqui aplica-se a mesma cerca: este fixture so
// responde quando a navegacao e' a da prova do item 46 (a console aberta com
// `mission_id=mis_companion_e2e`). Qualquer outro teste recebe `null`.

/** Id da missao usada pela prova E2E do Companion. */
export const COMPANION_E2E_MISSAO_ID = "mis_companion_e2e";

/**
 * Estado servido por omissao: `a_trabalhar` — um estado que EXISTE no conjunto fechado
 * do servidor e que exercita o caminho "ha' trabalho real a decorrer", em vez do estado
 * neutro `ausente` (que a UI usa justamente para dizer "nao consegui saber").
 */
export const COMPANION_ESTADO_PADRAO = "a_trabalhar";

export function createCompanionFixture({
  sendJSON,
  estado = COMPANION_ESTADO_PADRAO,
  razao = "Fixtures E2E: a missao esta' a decorrer (estado servido pelo fixture).",
  scope = "missao",
} = {}) {
  if (typeof sendJSON !== "function") {
    // Falha alto: sem `sendJSON` o fixture nao consegue responder e o spec falharia de
    // forma confusa mais adiante. *Melhor rebentar aqui, com causa nomeada.*
    throw new Error(
      "createCompanionFixture exige `sendJSON` (injectado por first-run-backend.mjs)",
    );
  }

  /** Missao servida — o `AgenticSplitShell` monta o Companion a partir dela. */
  const missao = {
    id: COMPANION_E2E_MISSAO_ID,
    objective: "Prova E2E do Companion (item 46)",
    state: "RUNNING",
    workspace: "/workspace/e2e",
    plan: { steps: [] },
    approvals: [],
    artifacts: [],
    usage: {},
  };

  return async function handleCompanionRequest({ route, request, response }) {
    const referer = String(
      (request && request.headers && request.headers.referer) || "",
    );
    if (!referer.includes(`mission_id=${COMPANION_E2E_MISSAO_ID}`)) return null;

    if (route === `GET /api/agent/v1/missions/${COMPANION_E2E_MISSAO_ID}/companion`) {
      sendJSON(response, 200, {
        mission_id: COMPANION_E2E_MISSAO_ID,
        estado,
        razao,
        scope,
      });
      return 200;
    }

    if (route === "GET /api/agent/v1/missions") {
      sendJSON(response, 200, { missions: [missao] });
      return 200;
    }

    if (route === `GET /api/agent/v1/missions/${COMPANION_E2E_MISSAO_ID}`) {
      sendJSON(response, 200, missao);
      return 200;
    }

    // O `AgenticSplitShell` monta o Companion quando ha' missao; o console pede ainda
    // os eventos e o SSE. Sem estas rotas o stub responderia 404 e o painel nao aparecia
    // (mesma licao medida que obrigou o fixture do Estudio a servir `/events/stream`).
    if (route === `GET /api/agent/v1/missions/${COMPANION_E2E_MISSAO_ID}/events`) {
      sendJSON(response, 200, { events: [] });
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${COMPANION_E2E_MISSAO_ID}/events/stream`) {
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

    return null;
  };
}
