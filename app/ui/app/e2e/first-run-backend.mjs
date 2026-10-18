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

// FASE 2 (E-1237): fixture DECLARADO das rotas de worktree.
//
// PORQUE E' UM FIXTURE E NAO O BACKEND REAL
//
//   O inventario de worktrees exige um repositorio git de verdade. O backend de fixture nao tem
//   um. Por isso estes dados sao INVENTADOS e dizem-no:
//
//       *Um fixture que se faz passar por backend real e' pior do que nenhum fixture: prova
//       integracao que nao aconteceu.*
//
//   O spec `worktrees-panel.spec.ts` classifica o modo (`UI_FIXTURE` aqui,
//   `BACKEND_REAL_AGENT_RUNTIME` quando `HADES_E2E_BACKEND_URL` esta' definido) e NAO finge que
//   um e' o outro.
const FONTE_DO_FIXTURE = "e2e-fixture";
const worktreesDoFixture = {
  repo: "/repo-fixture",
  resumo: "2 worktree(s): 1 do HADES, 1 externo",
  total: 2,
  do_hades: 1,
  externo: 1,
  lista: [
    {
      caminho_resolvido: "/repo-fixture/.agent-worktrees/missao-e2e",
      branch: "agent/missao-e2e",
      commit: "abcdef1",
      detached: false,
      criado_pelo_hades: true,
      principal: false,
      bloqueado: false,
    },
    {
      caminho_resolvido: "/repo-fixture-externo",
      branch: "externo",
      commit: "1234567",
      detached: false,
      criado_pelo_hades: false,
      principal: false,
      bloqueado: false,
    },
  ],
  avisos: [
    "existem worktrees que o HADES nao criou; remover um deles apaga trabalho que nao e' dele",
  ],
  somente_leitura: true,
  source: FONTE_DO_FIXTURE,
};
// O estado muda com POST /worktrees/state -- para o spec poder provar a RELEITURA.
let estadoDoWorktreeDoFixture = "preparado";

// FASE 28.8: os planos de build que o fixture serve ao painel.
//
//   *Um fixture que so' serve o caminho feliz nao prova que o painel distingue os outros.*
function planoDoFixture(repo) {
  const sondaSaudavel = (nome, motivo) => ({
    Requisito: { Nome: nome, Comando: nome, Obrigatorio: true, Motivo: motivo },
    Estado: "saudavel",
    Caminho: `/usr/local/bin/${nome}`,
    Versao: `${nome} version 1.2.3`,
  });
  const sondaAusente = (nome, motivo) => ({
    Requisito: { Nome: nome, Comando: nome, Obrigatorio: true, Motivo: motivo },
    Estado: "ausente",
    Detalhe: "nao encontrado no PATH",
  });
  return {
    somente_leitura: true,
    nada_instalado: true,
    nada_construido: true,
    repo,
    profundidade: 4,
    plano: {
      projeto: {
        raiz: repo,
        stacks: ["go", "node"],
        gestores: ["go-modules", "npm"],
        alvos: ["linux", "web"],
        manifests: ["go.mod", "package.json"],
      },
      alvos: [
        {
          alvo: "linux",
          plano: {
            Alvo: "linux",
            Probes: [
              sondaSaudavel("go", "compilar o binario"),
              sondaAusente("docker", "empacotar a imagem"),
            ],
            Blockers: ["docker"],
            TrabalhoIndependente: ["go mod tidy"],
          },
        },
        {
          alvo: "web",
          plano: {
            Alvo: "web",
            Probes: [sondaSaudavel("node", "construir o bundle")],
            Blockers: [],
            TrabalhoIndependente: [],
          },
        },
      ],
      alvos_prontos: ["web"],
      alvos_bloqueados: ["linux"],
      sem_alvos: false,
      trabalho_independente: ["go mod tidy"],
      avisos: [],
      explicacao: "1 alvo(s) pronto(s) e 1 bloqueado(s); 1 item(ns) de trabalho independente",
    },
    blockers: ["docker"],
    motivos: ["linux: docker -- empacotar a imagem (nao encontrado no PATH)"],
    aviso_de_fronteira:
      "esta rota apenas DETECTA o projecto e SONDA ferramentas; nao instala nada, nao constroi nada e nao assina nada",
    source: FONTE_DO_FIXTURE,
  };
}

function planoVazioDoFixture(repo) {
  return {
    somente_leitura: true,
    nada_instalado: true,
    nada_construido: true,
    repo,
    profundidade: 4,
    plano: {
      projeto: {
        raiz: repo,
        stacks: ["python"],
        gestores: [],
        alvos: [],
        manifests: ["pyproject.toml"],
      },
      alvos: [],
      alvos_prontos: [],
      alvos_bloqueados: [],
      sem_alvos: true,
      trabalho_independente: [],
      avisos: [],
      explicacao:
        "o projecto nao declara nenhum alvo que o catalogo do HADES conheca; nao ha' plano de build a construir",
    },
    blockers: [],
    motivos: [],
    aviso_de_fronteira:
      "o projecto nao declara alvos que o catalogo do HADES conheca; nada foi construido",
    source: FONTE_DO_FIXTURE,
  };
}

function diagnosticoDoFixture(missao) {
  return {
    mission_id: missao,
    estado: estadoDoWorktreeDoFixture,
    estado_valido: true,
    estado_traduzido: false,
    proximos: ["suspenso", "integrado", "removido"],
    alvo_seguro: true,
    commits_nao_integrados: 0,
    ficheiros_modificados: 0,
    tem_trabalho_nao_integrado: false,
    estado_observado: "fixture E2E declarado",
    estado_do_registo: false,
    registo: "ausente: nenhuma entrada persistida neste data root",
    pode_remover: true,
    motivo_remover: "",
    source: FONTE_DO_FIXTURE,
  };
}

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
  } else if (route === "GET /api/agent/v1/build/plan") {
      // FASE 28.8: o plano de release do PROJECTO. O fixture serve um plano com um alvo PRONTO e
      // um BLOQUEADO, e um segundo "projecto" sem alvos -- para o spec poder provar as duas
      // classificacoes de resultado.
      //
      //   *Um fixture que so' serve o caminho feliz nao prova que o painel distingue os outros.*
      const repoPedido = url.searchParams.get("repo") || "";
      if (!repoPedido) {
        status = 400;
        sendJSON(response, status, { erro: "parametro `repo` e' obrigatorio" });
      } else if (repoPedido.includes("sem-alvos")) {
        status = 200;
        sendJSON(response, status, planoVazioDoFixture(repoPedido));
      } else {
        status = 200;
        sendJSON(response, status, planoDoFixture(repoPedido));
      }
  } else if (route === "GET /api/agent/v1/worktrees") {
      status = 200;
      sendJSON(response, status, worktreesDoFixture);
    } else if (route === "GET /api/agent/v1/worktrees/diagnose") {
      const missao = url.searchParams.get("missao") || "";
      if (!missao) {
        status = 400;
        sendJSON(response, status, { erro: "parametro 'missao' e' obrigatorio" });
      } else {
        status = 200;
        sendJSON(response, status, diagnosticoDoFixture(missao));
      }
    } else if (route === "POST /api/agent/v1/worktrees/state") {
      let pedido = {};
      try {
        pedido = JSON.parse((await readBody(request)) || "{}");
      } catch {
        pedido = {};
      }
      const para = String(pedido.para || "").toLowerCase();
      // A MESMA allowlist da rota real: tres accoes reversiveis. `removido` NAO entra.
      const reversiveis = ["suspender", "retomar", "arquivar"];
      if (!reversiveis.includes(para)) {
        status = 400;
        sendJSON(response, status, {
          erro: "transicao nao permitida por esta rota",
          pedido: pedido.para,
          permitidas: reversiveis,
        });
      } else {
        const anterior = estadoDoWorktreeDoFixture;
        estadoDoWorktreeDoFixture =
          para === "suspender"
            ? "suspenso"
            : para === "retomar"
              ? "em_uso"
              : "arquivado";
        status = 200;
        sendJSON(response, status, {
          mission_id: pedido.missao,
          de: anterior,
          para: estadoDoWorktreeDoFixture,
          acao: para,
          estado_do_registo: true,
          registado_em: "/repo-fixture/_state/worktree-state.json",
          avisos: ["fixture E2E: estado mudado em memoria, nao em disco"],
          somente_leitura: false,
          source: FONTE_DO_FIXTURE,
        });
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
  } else if (route === "GET /api/agent/v1/toolchain/targets") {
    // FASE 28.7/28.8. FIXTURE DECLARADO.
    //
    // Alvos de fixture, com nomes que existem no catalogo real mas com exigencias
    // inventadas: *um fixture que copia a fonte de verdade desactualiza-se em silencio.*
    // O `source` diz a quem ler que estes dados nao sao medicao.
    status = 200;
    sendJSON(response, status, {
      host: "fixture-host",
      goos: "fixture",
      alvos: [
        {
          nome: "fixture-alfa",
          descricao: "Alvo de fixture alfa",
          exigencias: ["fixture-tool-a (obrigatoria) -- so' existe no fixture"],
        },
        {
          nome: "fixture-beta",
          descricao: "Alvo de fixture beta",
          exigencias: ["fixture-tool-b -- so' existe no fixture"],
        },
      ],
      count: 2,
      source: "e2e-fixture",
    });
  } else if (route === "GET /api/agent/v1/toolchain/doctor") {
    status = 200;
    sendJSON(response, status, {
      host: "fixture-host",
      goos: "fixture",
      alvos: [
        { alvo: "fixture-alfa", pronto: false, veredicto: "bloqueado", exigencias_bloqueantes: 1 },
        { alvo: "fixture-beta", pronto: false, veredicto: "incompleto", exigencias_bloqueantes: 0 },
      ],
      prontos: 0,
      bloqueados: 1,
      incompletos: 1,
      total: 2,
      resumo: "1 de 2 alvos bloqueados, 1 incompletos (ferramentas presentes, falta health funcional)",
      source: "e2e-fixture",
    });
  } else if (route.startsWith("GET /api/agent/v1/toolchain/plan/")) {
    // O alvo desconhecido devolve 404, como o servidor real. *Um fixture que aceita
    // qualquer alvo esconderia exactamente o comportamento que o teste quer medir.*
    const alvo = url.pathname.split("/").pop();
    if (alvo === "fixture-alfa") {
      status = 200;
      sendJSON(response, status, {
        host: "fixture-host",
        goos: "fixture",
        alvo: "fixture-alfa",
        pronto: false,
        veredicto: "bloqueado",
        probes: [
          {
            nome: "fixture-tool-a",
            estado: "ABSENT",
            obrigatorio: true,
            motivo: "so' existe no fixture",
            detalhe: "nao encontrado no PATH",
          },
        ],
        blockers: ["fixture-alfa exige host fixture; este host e' fixture"],
        trabalho_independente: ["preparar o resto do fixture"],
        source: "e2e-fixture",
      });
    } else {
      status = 404;
      sendJSON(response, status, { error: `alvo desconhecido: ${alvo}` });
    }
  } else if (route === "GET /api/agent/v1/install/profiles") {
    // FASE 28.6-A: catalogo de perfis e pacotes. Nomes `fixture-*` e `source: e2e-fixture`
    // para nao se confundir com o catalogo real. *Um fixture que se faz passar pelo produto
    // esconde exactamente a diferenca que o teste quer medir.*
    sendJSON(response, 200, {
      source: "e2e-fixture",
      host: "e2e-fixture-host",
      goos: "windows",
      perfis: [
        { id: "BASIC", descricao: "HADES Core e diagnostico (fixture)", pacotes: ["hades-core"], tamanho_mb: 180, escolhe_pacotes: false },
        { id: "CONTAINERS_AND_CLOUD", descricao: "Core + containers (fixture)", pacotes: ["hades-core", "containers"], tamanho_mb: 720, escolhe_pacotes: false },
        { id: "CUSTOM", descricao: "escolha componente a componente (fixture)", pacotes: [], tamanho_mb: 0, escolhe_pacotes: true },
      ],
      pacotes: [
        { id: "hades-core", nome: "HADES Core", dependencias: [], tamanho_mb: 180, licenca: "MIT", redistribuivel: true, exige_aprovacao: false, impacto: [] },
        { id: "containers", nome: "Containers e Buildx", dependencias: ["hades-core"], tamanho_mb: 540, licenca: "Apache-2.0", redistribuivel: false, exige_aprovacao: true, impacto: ["exige elevacao administrativa", "cria servico persistente"] },
      ],
      avisos: [],
      count: 3,
    });
  } else if (route.startsWith("GET /api/agent/v1/install/plan/")) {
    // O perfil desconhecido devolve 400, como o servidor real.
    // *Um fixture que aceita qualquer perfil esconderia o comportamento que o teste mede.*
    const perfil = decodeURIComponent(route.slice("GET /api/agent/v1/install/plan/".length)).toUpperCase();
    const planos = {
      BASIC: {
        itens: [
          { pacote_id: "hades-core", nome: "HADES Core", acao: "PULAR", motivo: "nao foi possivel determinar o estado nesta maquina (fixture)", requer_aprovacao: false, tamanho_mb: 180 },
        ],
        aprovacoes_necessarias: 0,
        bloqueios: ["hades-core: estado desconhecido (UNKNOWN nao e' ABSENT) [fixture]"],
        pronto: false,
        tamanho_total_mb: 0,
        downloads_evitados: 0,
      },
      CONTAINERS_AND_CLOUD: {
        itens: [
          { pacote_id: "containers", nome: "Containers e Buildx", acao: "APROVAR", motivo: "exige aprovacao: elevacao administrativa, servico persistente (fixture)", requer_aprovacao: true, tamanho_mb: 540 },
        ],
        aprovacoes_necessarias: 1,
        bloqueios: [],
        pronto: false,
        tamanho_total_mb: 540,
        downloads_evitados: 0,
      },
      CUSTOM: { itens: [], aprovacoes_necessarias: 0, bloqueios: [], pronto: true, tamanho_total_mb: 0, downloads_evitados: 0 },
    };
    const escolhido = planos[perfil];
    if (!escolhido) {
      sendJSON(response, 400, { source: "e2e-fixture", error: "perfil desconhecido; use GET /install/profiles para a lista" });
    } else {
      sendJSON(response, 200, {
        source: "e2e-fixture",
        perfil,
        host: "e2e-fixture-host",
        goos: "windows",
        itens: escolhido.itens,
        tamanho_total_mb: escolhido.tamanho_total_mb,
        downloads_evitados: escolhido.downloads_evitados,
        aprovacoes_necessarias: escolhido.aprovacoes_necessarias,
        bloqueios: escolhido.bloqueios,
        pronto: escolhido.pronto,
        resumo: `perfil ${perfil} (fixture): ${escolhido.itens.length} itens, ${escolhido.tamanho_total_mb} MB a baixar, ${escolhido.aprovacoes_necessarias} aprovacoes, ${escolhido.bloqueios.length} bloqueios`,
        executa_algo: false,
      });
    }
  } else if (route === "GET /api/agent/v1/signing/identities") {
    // FASE 28.5. FIXTURE DECLARADO. Nenhum valor privado, por construcao: a estrutura nao
    // tem campo para ele, e o fixture segue a mesma forma.
    status = 200;
    sendJSON(response, status, {
      estado: { configurado: true, carregado: true, mensagem: "2 identidade(s) registada(s)", total: 2 },
      identidades: [
        {
          nome: "fixture-producao",
          plataforma: "ios",
          ambiente: "producao",
          provedor: "dpapi",
          fingerprint_publico_fim: "FIXTURE1",
          expira_em: "2099-01-01T00:00:00Z",
          estado: "ativa",
          utilizavel_agora: true,
          exige_aprovacao: true,
          runners_autorizados: 1,
        },
        {
          nome: "fixture-preview",
          plataforma: "android",
          ambiente: "preview",
          provedor: "dpapi",
          fingerprint_publico_fim: "FIXTURE2",
          expira_em: "2099-01-01T00:00:00Z",
          estado: "expira_breve",
          utilizavel_agora: true,
          exige_aprovacao: false,
          runners_autorizados: 1,
        },
      ],
      problemas: [],
      resumo: { total: 2, por_estado: { ativa: 1, expira_breve: 1 }, por_ambiente: { producao: 1, preview: 1 }, problemas: 0 },
      verificado_em: "2099-01-01T00:00:00Z",
      aviso: "apenas metadados publicos (fixture)",
      source: "e2e-fixture",
    });
  } else if (route === "GET /api/agent/v1/signing/summary") {
    status = 200;
    sendJSON(response, status, {
      configurado: true,
      carregado: true,
      total: 2,
      mensagem: "2 identidade(s) registada(s)",
      resumo: { total: 2, por_estado: { ativa: 1, expira_breve: 1 }, por_ambiente: { producao: 1, preview: 1 }, problemas: 0 },
      problemas: 0,
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
