// Backend mínimo, real e sem dependências para a jornada de primeira execução.
//
// Por que existe: a suíte E2E do shell nasceu "offline-first" (sem backend) e o
// portão de primeira execução é decidido pelo SERVIDOR, não por localStorage:
//
//   GET  /api/v1/settings  -> { settings: { OnboardingVersion: n } }
//   POST /api/v1/settings  -> grava e devolve o estado persistido
//
// `src/routes/index.tsx` só redireciona para /onboarding quando a leitura do
// settings responde com `OnboardingVersion < CURRENT_ONBOARDING_VERSION`; quando
// a chamada falha, o shell segue utilizável e nunca mostra o onboarding. Para
// provar a jornada (instalação -> onboarding -> persistência -> reinício ->
// atualização -> rollback) é preciso um servidor HTTP de verdade respondendo o
// contrato acima. Este processo responde apenas esse contrato, mais
// `/api/me` (401, sessão local), `/api/version` (health check) e
// `/api/v1/models/pull` (stub de download, ver abaixo).
//
// O SPA alcança este processo pela MESMA origem do preview do Vite, porque
// `vite.config.ts` faz proxy de `/api` para `OLLAMA_PROXY_TARGET` (padrão
// http://127.0.0.1:11434). Nada de `page.route`: a requisição é HTTP real.
//
// Estado servido por padrão = `OnboardingVersion: 1`, ou seja, "já configurado".
// Isso é deliberado: as specs existentes (`shell`, `accessibility`) rodam contra
// este backend e precisam enxergar o mesmo shell que enxergavam sem backend. A
// spec de primeira execução zera a versão pela API pública antes de cada teste,
// simulando uma instalação nova.

import { createServer } from "node:http";

import { createStudioBuilderFixture } from "./studio-builder-fixture.mjs";

const PORT = Number(process.env.E2E_BACKEND_PORT || 43117);
const HOST = "127.0.0.1";

// Espelha `store.Settings` (app/store/store.go) com os campos que o cliente lê.
function defaultSettings() {
  return {
    Expose: false,
    Browser: false,
    Survey: false,
    Models: "",
    Agent: false,
    Tools: false,
    WorkingDir: "",
    ContextLength: 0,
    TurboEnabled: false,
    WebSearchEnabled: false,
    ThinkEnabled: false,
    ThinkLevel: "none",
    SelectedModel: "",
    SidebarOpen: false,
    LastHomeView: "chat",
    // Já configurado: é o estado que as specs sem backend enxergavam.
    OnboardingVersion: 1,
    AutoUpdateEnabled: false,
    ClaudeDesktopUsed: false,
  };
}

let settings = defaultSettings();

// Contrato do Studio (builder visual) fica em módulo próprio para manter este
// processo como o ÚNICO backend real da suíte: a spec de reordenação precisa
// de HTTP de verdade, não de `page.route`.
const handleStudioBuilderRequest = createStudioBuilderFixture({
  sendJSON,
  readBody,
});

function sendJSON(response, status, payload) {
  const body = Buffer.from(JSON.stringify(payload));
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": String(body.length),
    "Cache-Control": "no-store",
  });
  response.end(body);
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
    request.on("error", reject);
  });
}

// Mesma semântica de `app/ui/ui.go` (settings handler): `OnboardingVersion`
// ausente preserva o valor gravado; presente sobrescreve. As demais chaves
// enviadas são mescladas porque o cliente sempre envia o objeto completo.
function applySettingsUpdate(body) {
  let parsed;
  try {
    parsed = JSON.parse(body || "{}");
  } catch {
    return { ok: false, error: "invalid request body" };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { ok: false, error: "invalid request body" };
  }
  const update = {};
  for (const [key, value] of Object.entries(parsed)) {
    if (Object.prototype.hasOwnProperty.call(settings, key))
      update[key] = value;
  }
  if (typeof parsed.OnboardingVersion === "number") {
    update.OnboardingVersion = parsed.OnboardingVersion;
  }
  settings = { ...settings, ...update };
  return { ok: true };
}

const server = createServer(async (request, response) => {
  const url = new URL(request.url || "/", `http://${HOST}:${PORT}`);
  const route = `${request.method} ${url.pathname}`;
  let status = 404;

  if (route === "GET /api/version") {
    status = 200;
    sendJSON(response, status, { version: "0.0.0-e2e-primeira-execucao" });
  } else if (route === "GET /api/v1/settings") {
    status = 200;
    sendJSON(response, status, { settings });
  } else if (route === "POST /api/v1/settings") {
    const result = applySettingsUpdate(await readBody(request));
    if (!result.ok) {
      status = 400;
      sendJSON(response, status, { error: result.error });
    } else {
      status = 200;
      sendJSON(response, status, { settings });
    }
  } else if (route === "POST /api/v1/models/pull") {
    // Stub DECLARADO do download de modelo. `RunOllamaScreen` só habilita o
    // botão "Continuar" depois que o stream do pull termina com sucesso
    // (api.ts -> pullModel -> POST /api/v1/models/pull, NDJSON). Baixar um
    // modelo real (GBs) no CI não é viável, então este processo responde um
    // stream NDJSON curto e rotulado como stub — o comportamento verificado
    // continua sendo o real: antes do stream o botão NÃO existe, depois existe.
    let requested = "";
    try {
      const parsed = JSON.parse((await readBody(request)) || "{}");
      if (parsed && typeof parsed.name === "string")
        requested = parsed.name.trim();
    } catch {
      requested = "";
    }
    if (!requested) {
      status = 400;
      sendJSON(response, status, { error: "model name is required" });
    } else {
      status = 200;
      response.writeHead(status, {
        "Content-Type": "application/x-ndjson; charset=utf-8",
        "Cache-Control": "no-store",
      });
      response.write(
        `${JSON.stringify({ status: `preparando ${requested} (stub E2E)`, total: 1, completed: 0 })}\n`,
      );
      response.write(
        `${JSON.stringify({ status: "success", total: 1, completed: 1, done: true })}\n`,
      );
      response.end();
    }
  } else if (route === "POST /api/me") {
    // Sessão local, sem conta: `fetchUser` devolve null em 401.
    status = 401;
    sendJSON(response, status, {});
  } else {
    const studioStatus = await handleStudioBuilderRequest({
      route,
      request,
      response,
    });
    if (studioStatus === null) {
      sendJSON(response, status, { error: "not found" });
    } else {
      status = studioStatus;
    }
  }

  process.stdout.write(`[e2e-backend] ${route} -> ${status}\n`);
});

server.listen(PORT, HOST, () => {
  process.stdout.write(
    `[e2e-backend] ouvindo em http://${HOST}:${PORT} (OnboardingVersion=${settings.OnboardingVersion})\n`,
  );
});

function shutdown() {
  server.close(() => process.exit(0));
}

process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);
