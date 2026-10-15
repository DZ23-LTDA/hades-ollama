// Fixture de MISSAO para a prova em browser do Estudio multimodal (item 45).
//
// Segue EXACTAMENTE o contrato do `studio-builder-fixture.mjs`: quem constroi passa
// `{ sendJSON, readBody }`; o handler recebe `{ route, request, response }` e devolve o
// STATUS HTTP quando tratou a rota, ou `null` para o chamador responder 404.
//
// *Porque nao `page.route`:* medido em 2026-10-15 (commit e899c5f77) — um spec com
// `page.route` foi REPROVADO pelo `Web E2E`, porque as chamadas do browser foram ao
// backend-stub do repositorio (`[e2e-backend] GET /api/agent/v1/missions -> 404`) e nao
// foram interceptadas. O caminho do repositorio e' este: fixture ligado ao stub.
//
// A missao declara `workspace` de proposito: o painel exige um workspace fixado para
// poder lancar, e o servidor recusa sem ele ("pinned media workspace is required").

export const MISSAO_E2E_ID = "mis_e2e";

export function createMediaStudioFixture({ sendJSON } = {}) {
  const missao = {
    id: MISSAO_E2E_ID,
    objective: "Prova E2E do estudio multimodal",
    state: "PENDING",
    workspace: "/workspace/e2e",
    plan: { steps: [] },
    approvals: [],
    artifacts: [],
    usage: {},
  };

  return async function handleMediaStudioRequest({ route, request, response }) {
    if (route === "GET /api/agent/v1/missions") {
      sendJSON(response, 200, { missions: [missao] });
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${MISSAO_E2E_ID}`) {
      sendJSON(response, 200, missao);
      return 200;
    }
    if (route === `GET /api/agent/v1/missions/${MISSAO_E2E_ID}/events`) {
      sendJSON(response, 200, { events: [] });
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
