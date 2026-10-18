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
// Fixture da missão do Estúdio multimodal (item 45): mesma disciplina — HTTP de verdade
// no stub, e NÃO `page.route` (medido: `page.route` não intercepta aqui).
import { createMediaStudioFixture } from "./media-studio-fixture.mjs";
// Fixture do Companion (item 46 / complemento 48): mesma disciplina — HTTP de verdade,
// e NAO `page.route`. So' responde quando o `Referer` traz `mission_id=mis_companion_e2e`,
// para nao mudar o DOM que os outros specs observam (ver comentario no proprio fixture).
import { createCompanionFixture } from "./companion-fixture.mjs";
// E-916: Fixture da SALA (complemento 48 — Cowork/Team Chat). Mesma disciplina do Companion
// (HTTP de verdade, NAO `page.route`) e o **mesmo isolamento por `Referer`**
// (`mission_id=mis_sala_e2e`). *O fixture existia desde os rounds 62-63 e **ninguem o importava**:
// sem esta linha ele nunca corria - era tao orfao como o componente que ele serve.*
import { createSalaFixture } from "./sala-fixture.mjs";
// E-926/E-927: Fixture do CHAT (familia `/api/v1/chat/...`). Mesma disciplina (HTTP de
// verdade, NAO `page.route`), mas a cerca e' a **ROTA** e nao o `Referer`: o
// `chat-ui-real.spec.ts` faz `page.goto("/")` SEM missao. O contrato servido e' o do
// **desktop** (`app/ui/ui.go:285-290`), o unico que serve esta familia; o `hades serve`
// responde 501 **de proposito** (`server/desktop_local_routes.go:141-145`).
import { createChatFixture } from "./chat-fixture.mjs";

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

// Fixture da missão (item 45): serve `/api/agent/v1/missions*` para o Agentic Console
// conseguir seleccionar a missão e montar o painel do Estúdio multimodal.
const handleMediaStudioRequest = createMediaStudioFixture({
  sendJSON,
  readBody,
});

// Fixture do Companion (item 46): serve
// `GET /api/agent/v1/missions/mis_companion_e2e/companion` para a prova do mascote.
const handleCompanionRequest = createCompanionFixture({
  sendJSON,
  readBody,
});

// E-916: Fixture da SALA. Serve
//   `GET /api/agent/v1/collab/proj_sala_e2e/rooms/sala_e2e/members` -> { membros }
//   `GET /api/agent/v1/collab/proj_sala_e2e/rooms`                  -> { salas }
// — o contrato real de `server/agent_routes.go:849-851`.
const handleSalaRequest = createSalaFixture({
  sendJSON,
});

// E-926/E-927: Fixture do CHAT. Serve a familia que o SPA chama de verdade
//   `POST /api/v1/chat/new`       -> stream JSONL (o `id` literal "new" CRIA o chat)
//   `POST /api/v1/chat/{id}`      -> stream JSONL
//   `GET  /api/v1/chat/{id}`      -> { chat }
// — o contrato real de `app/ui/ui.go:285-290`, o unico servidor que serve esta familia.
const handleChatRequest = createChatFixture({
  sendJSON,
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
  } else if (route === "GET /api/agent/v1/presets") {
    // FASE 23 do adendo V3. FIXTURE DECLARADO.
    //
    // Estes NAO sao os presets embutidos do HADES: sao dois presets de fixture, com
    // ids que nenhum catalogo real usa. A razao de nao copiar os 11 reais: *um fixture
    // que copia a fonte de verdade desactualiza-se em silencio e passa a medir uma
    // coisa que ja' nao existe.* Com ids `fixture-*` ninguem confunde os dois.
    //
    // O que este fixture permite medir: que a UI mostra o que o servidor devolve, que
    // a contagem de opcoes bate, e que "Sem preset" existe sempre. O que ele NAO mede:
    // o catalogo real. Para isso ha' `HADES_E2E_BACKEND_URL`.
    status = 200;
    sendJSON(response, status, {
      presets: [
        {
          id: "fixture-alfa",
          name: "Fixture Alfa",
          description: "Preset de fixture — nao existe em producao.",
          completion_criteria: ["o teste de fixture terminou"],
          budget_seconds: 60,
        },
        {
          id: "fixture-beta",
          name: "Fixture Beta",
          description: "Segundo preset de fixture.",
          completion_criteria: ["a UI mostrou os dois"],
        },
      ],
      count: 2,
      source: "e2e-fixture",
    });
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
    const mediaStatus =
      studioStatus === null
        ? await handleMediaStudioRequest({ route, request, response })
        : studioStatus;
    // Fixture do Companion (item 46): so' entra na cadeia se os anteriores devolverem null.
    const companionStatus =
      mediaStatus === null
        ? await handleCompanionRequest({ route, request, response })
        : mediaStatus;
    // E-916: Fixture da SALA (complemento 48), pela mesma regra de cadeia.
    const salaStatus =
      companionStatus === null
        ? await handleSalaRequest({ route, request, response })
        : companionStatus;
    if (salaStatus === null) {
      // E-926/E-927: Fixture do CHAT, pela mesma regra de cadeia. A cerca dele e' a ROTA
      // (`/api/v1/chat/...`), nao o `Referer` — ver o cabecalho do `chat-fixture.mjs`.
      const chatStatus = await handleChatRequest({ route, request, response });
      if (chatStatus === null) {
        sendJSON(response, status, { error: "not found" });
      } else {
        status = chatStatus;
      }
    } else {
      status = salaStatus;
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
