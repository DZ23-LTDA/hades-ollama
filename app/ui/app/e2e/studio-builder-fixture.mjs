// Fixture HTTP real do Studio (builder visual) para a suíte E2E.
//
// Por que existe: o canvas do Studio persiste a ordem dos componentes no
// SERVIDOR (`updateBuilderVisual` -> POST /api/agent/v1/builders/{id}/visual,
// ver `src/components/StudioCanvasPage.tsx`). Sem backend não há como provar
// que a reordenação sobrevive ao recarregamento — só que a tela mudou. Este
// módulo implementa o contrato do builder em um servidor `node:http` de
// verdade, sem dependências, para que a requisição seja HTTP real (nada de
// `page.route`).
//
// Escopo deliberado: apenas o contrato usado pela jornada de reordenação.
// - GET  /api/agent/v1/builders                -> lista com o projeto fixture
// - GET  /api/agent/v1/builders/{id}           -> projeto com componentes
// - POST /api/agent/v1/builders/{id}/visual    -> grava a nova ordem da lista
// - POST /api/agent/v1/builders                -> cria projeto (caminho do app)
// - POST /api/__e2e/studio/reset               -> volta o projeto ao estado base
// - GET  /api/__e2e/studio/state               -> estado observado pelo servidor
//
// O estado base são três componentes rotulados (`heading`, `paragraph`,
// `button`), os mesmos que o app cria como projeto inicial. Os endpoints
// `__e2e/*` são controle de teste e não fazem parte da API do produto; eles
// ficam sob `/api` apenas para atravessarem o proxy do preview do Vite, que é
// quem dá a mesma origem ao SPA.

const PROJECT_ID = "builder_e2e_reorder";
const BASE_VERSION = 7;

function baseComponents() {
  return [
    {
      id: "comp_hero",
      type: "heading",
      props: { text: "Bem-vindo ao Studio do Hades", level: "h1" },
      x: 40,
      y: 40,
      width: 700,
      height: 70,
    },
    {
      id: "comp_lead",
      type: "paragraph",
      props: { text: "Editor visual interativo de ponta a ponta." },
      x: 40,
      y: 120,
      width: 700,
      height: 50,
    },
    {
      id: "comp_cta",
      type: "button",
      props: { label: "Experimentar Agora" },
      x: 40,
      y: 190,
      width: 180,
      height: 48,
    },
  ];
}

export function createStudioBuilderFixture({ sendJSON, readBody }) {
  let components = baseComponents();
  let version = BASE_VERSION;
  let visualWrites = 0;
  let lastWriteIds = [];

  function project() {
    return {
      id: PROJECT_ID,
      organization_id: "local-org",
      name: "Hades Studio App",
      kind: "website",
      entry: "index.html",
      version,
      status: "draft",
      root: "/tmp/hades-studio-e2e",
      created_at: "2026-10-10T00:00:00Z",
      updated_at: new Date().toISOString(),
      components,
      undo_stack: [],
      redo_stack: [],
    };
  }

  function reset() {
    components = baseComponents();
    version = BASE_VERSION;
    visualWrites = 0;
    lastWriteIds = [];
  }

  // Devolve o status HTTP quando tratou a rota, ou `null` para deixar o
  // chamador responder 404.
  return async function handleStudioBuilderRequest({
    route,
    request,
    response,
  }) {
    if (route === "POST /api/__e2e/studio/reset") {
      reset();
      sendJSON(response, 200, {
        status: "reset",
        version,
        order: components.map((c) => c.type),
      });
      return 200;
    }

    if (route === "GET /api/__e2e/studio/state") {
      sendJSON(response, 200, {
        version,
        order: components.map((c) => c.type),
        ids: components.map((c) => c.id),
        visual_writes: visualWrites,
        last_write_ids: lastWriteIds,
      });
      return 200;
    }

    if (route === "GET /api/agent/v1/builders") {
      sendJSON(response, 200, {
        projects: [
          {
            id: PROJECT_ID,
            organization_id: "local-org",
            name: "Hades Studio App",
            kind: "website",
            entry: "index.html",
            version,
            status: "draft",
            root: "/tmp/hades-studio-e2e",
            created_at: "2026-10-10T00:00:00Z",
            updated_at: new Date().toISOString(),
          },
        ],
      });
      return 200;
    }

    if (route === `GET /api/agent/v1/builders/${PROJECT_ID}`) {
      sendJSON(response, 200, project());
      return 200;
    }

    if (route === `POST /api/agent/v1/builders/${PROJECT_ID}/visual`) {
      let parsed;
      try {
        parsed = JSON.parse((await readBody(request)) || "{}");
      } catch {
        parsed = null;
      }
      if (!parsed || !Array.isArray(parsed.components)) {
        sendJSON(response, 400, { error: "components are required" });
        return 400;
      }
      // Concorrência otimista igual à do servidor real: versão divergente é
      // recusada em vez de sobrescrever silenciosamente.
      if (
        typeof parsed.expected_version === "number" &&
        parsed.expected_version !== version
      ) {
        sendJSON(response, 409, { error: "version conflict", version });
        return 409;
      }
      components = parsed.components;
      version += 1;
      visualWrites += 1;
      lastWriteIds = components.map((c) => c.id);
      sendJSON(response, 200, project());
      return 200;
    }

    if (route === "POST /api/agent/v1/builders") {
      let parsed;
      try {
        parsed = JSON.parse((await readBody(request)) || "{}");
      } catch {
        parsed = null;
      }
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        sendJSON(response, 400, { error: "invalid request body" });
        return 400;
      }
      if (Array.isArray(parsed.components) && parsed.components.length > 0) {
        components = parsed.components;
      }
      version += 1;
      sendJSON(response, 200, project());
      return 200;
    }

    return null;
  };
}
