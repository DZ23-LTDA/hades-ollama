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

// FASE 28.7: constantes da verificacao de artefacto.
//
// O SHA e' sintaticamente valido (64 hexadecimais) e FIXO, para o spec poder pedi-lo de volta e
// provocar tanto o caso integro como o adulterado. Nao e' o SHA de nenhum ficheiro real -- o fixture
// nao le' o disco, e *um fixture que se faz passar por backend real e' pior do que nenhum fixture:
// prova integracao que nao aconteceu.*
const SHA_DO_FIXTURE =
  "fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6";

// O aviso de fronteira VEM DO "BACKEND" (aqui, do fixture), para o spec provar que o painel mostra
// o texto do servidor e nao um texto que o componente inventou:
//
//   *Um painel que mostra "pronto" sem poder construir e' um painel que promete o que nao faz --
//   a menos que diga que so' le'.*
const AVISO_DE_FRONTEIRA_DO_ARTEFACTO =
  "estas rotas apenas MEDEM e COMPARAM o ficheiro indicado; nao constroem, nao assinam, nao " +
  "instalam e nao o procuram por ID. O caminho tem de ser absoluto e apontar para um ficheiro que " +
  "ja' exista.";

/**
 * A MESMA regra do servidor de producao: o caminho tem de ser absoluto.
 *
 * Repetida aqui de proposito -- se o fixture fosse mais permissivo do que o servidor, o spec passaria
 * num comportamento que a producao recusa.
 */
function ehCaminhoAbsoluto(caminho) {
  if (/^[A-Za-z]:[\\/]/.test(caminho)) return true; // Windows: C:\ ou C:/
  return caminho.startsWith("/") || caminho.startsWith("\\\\");
}

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
    Estado: "PRESENT_COMPATIBLE_HEALTHY",
    Caminho: `/usr/local/bin/${nome}`,
    Versao: `${nome} version 1.2.3`,
  });
  const sondaAusente = (nome, motivo) => ({
    Requisito: { Nome: nome, Comando: nome, Obrigatorio: true, Motivo: motivo },
    Estado: "ABSENT",
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
  } else if (route === "GET /api/agent/v1/build/artifact/checksum") {
      // FASE 28.7: `artifact.checksum` -- MEDE, nao compara.
      //
      // O fixture deriva o resultado do CAMINHO pedido, para o spec poder provocar as tres
      // respostas sem inventar rotas. *Um fixture que so' serve o caminho feliz nao prova que o
      // painel distingue os outros.*
      const caminho = url.searchParams.get("caminho") || "";
      if (!caminho) {
        status = 400;
        sendJSON(response, status, { erro: "o parametro `caminho` e' obrigatorio" });
      } else if (!ehCaminhoAbsoluto(caminho)) {
        // A mesma regra do servidor: *um caminho relativo e' relativo ao directorio de quem pede, e
        // o servidor nao sabe qual e'.*
        status = 400;
        sendJSON(response, status, {
          erro:
            "o parametro `caminho` tem de ser ABSOLUTO: um caminho relativo e' relativo ao " +
            "directorio de quem pede, e o servidor nao sabe qual e'",
        });
      } else if (caminho.includes("inexistente")) {
        status = 404;
        sendJSON(response, status, { erro: `ficheiro nao encontrado: ${caminho}` });
      } else {
        status = 200;
        sendJSON(response, status, {
          somente_leitura: true,
          nada_assinado: true,
          caminho,
          tamanho_bytes: 41111716,
          sha256: SHA_DO_FIXTURE,
          aviso_de_fronteira: AVISO_DE_FRONTEIRA_DO_ARTEFACTO,
        });
      }
    } else if (route === "GET /api/agent/v1/build/artifact/verify") {
      // FASE 28.7: `artifact.verify` -- COMPARA com uma referencia.
      const caminho = url.searchParams.get("caminho") || "";
      const esperado = url.searchParams.get("sha256") || "";
      if (!caminho) {
        status = 400;
        sendJSON(response, status, { erro: "o parametro `caminho` e' obrigatorio" });
      } else if (!esperado) {
        // Sem referencia NAO se verifica.
        status = 400;
        sendJSON(response, status, {
          erro:
            "o parametro `sha256` e' obrigatorio: sem referencia isto seria uma medicao, nao uma " +
            "verificacao (use /artifact/checksum para medir)",
        });
      } else if (esperado.length !== 64) {
        // Um PREFIXO de checksum nao e' uma referencia.
        status = 400;
        sendJSON(response, status, {
          erro: `referencia invalida: ${esperado.length} caracteres; esperados 64 hexadecimais`,
        });
      } else {
        const confere = esperado === SHA_DO_FIXTURE;
        status = 200;
        sendJSON(response, status, {
          somente_leitura: true,
          nada_assinado: false,
          aviso_de_fronteira: AVISO_DE_FRONTEIRA_DO_ARTEFACTO,
          resultado: {
            caminho,
            tamanho_bytes: 41111716,
            sha256_medido: SHA_DO_FIXTURE,
            sha256_esperado: esperado,
            integro: confere,
            adulterado: !confere,
            // `NAO_VERIFICADA` e' o estado honesto de um fixture que nao inspecciona assinaturas.
            // Fingir `AUSENTE` seria afirmar um facto que o fixture nao mediu.
            assinatura: "NAO_VERIFICADA",
            detalhe_da_assinatura: "fixture: nenhum adaptador de assinatura",
            aprovado: confere,
            duracao_ms: 2,
            explicacao: confere
              ? "integro; assinatura NAO verificada neste host"
              : "ADULTERADO: o checksum medido nao corresponde ao esperado",
          },
        });
      }
    } else if (route === "POST /api/agent/v1/build/smoke") {
      // FASE 28.7: `build.smoke` -- EXECUTA um binario. O fixture simula o resultado.
      //
      // O fixture devolve o que o servidor devolveria SE a raiz estivesse configurada e o artefacto
      // existisse. O `sem_sandbox` e o `aviso_de_sandbox` vao SEMPRE -- o painel tem de os mostrar.
      // `readBody` devolve uma STRING. Sem o `JSON.parse` o `pedido` ficava vazio e a rota respondia
      // 400 dizendo que o caminho faltava -- um sintoma que parecia erro do painel.
      const corpo = await readBody(request);
      let pedido = {};
      try {
        pedido = JSON.parse(corpo || "{}");
      } catch {
        sendJSON(response, 400, { erro: "corpo invalido: esperado JSON" });
        return;
      }
      if (pedido === null || typeof pedido !== "object") {
        sendJSON(response, 400, { erro: "corpo invalido: esperado um objecto JSON" });
        return;
      }
      const caminho = typeof pedido.caminho === "string" ? pedido.caminho : "";
      if (!caminho) {
        sendJSON(response, 400, { erro: "o campo `caminho` e' obrigatorio" });
      } else if (!Array.isArray(pedido.argumentos) || pedido.argumentos.length === 0) {
        // A mesma regra do provider: *executar sem argumentos e' executa-lo no modo que o
        // programador nao escolheu*.
        sendJSON(response, 400, {
          erro:
            "smoke sem argumentos: executar o artefacto sem argumentos e' executa-lo no modo por " +
            "omissao, que ninguem escolheu",
        });
      } else {
        sendJSON(response, 200, {
          executou: true,
          aprovado: true,
          codigo_de_saida: 0,
          duracao_ms: 42,
          saida: "ollama version is 0.40.1",
          saida_truncada: false,
          timeout: false,
          explicacao: "o artefacto arrancou, terminou com exit code 0 em 42 ms",
          sem_sandbox: true,
          aviso_de_sandbox:
            "este smoke executou o artefacto com os privilegios deste processo. NAO ha' sandbox, " +
            "contentor, VM, emulador ou simulador nesta maquina. A fronteira e' apenas a raiz " +
            "`HADES_ARTEFACTOS`: o caminho tem de estar dentro dela, com symlinks resolvidos.",
          raiz_autorizada: true,
        });
      }
    } else if (route === "POST /api/agent/v1/sign/plan") {
      // FASE 28.7: `sign.plan` -- AVALIA, nunca assina.
      //
      // `readBody` devolve uma STRING (lissao da E-1275): sem o JSON.parse o pedido ficava vazio.
      const corpoAssinatura = await readBody(request);
      let pedidoAssinatura = {};
      try {
        pedidoAssinatura = JSON.parse(corpoAssinatura || "{}");
      } catch {
        sendJSON(response, 400, { erro: "corpo invalido: esperado JSON" });
        return;
      }
      if (pedidoAssinatura === null || typeof pedidoAssinatura !== "object") {
        sendJSON(response, 400, { erro: "corpo invalido: esperado um objecto JSON" });
        return;
      }
      const tipoAssinatura = pedidoAssinatura.tipo;
      const identidade = pedidoAssinatura.identidade;
      if (typeof tipoAssinatura !== "string" || tipoAssinatura === "") {
        sendJSON(response, 400, { erro: "o campo `tipo` e' obrigatorio" });
      } else if (identidade === null || typeof identidade !== "object") {
        sendJSON(response, 400, { erro: "o campo `identidade` e' obrigatorio" });
      } else if (
        typeof identidade.caminho_do_certificado !== "string" ||
        identidade.caminho_do_certificado.trim() === ""
      ) {
        sendJSON(response, 400, {
          erro: "o campo `identidade.caminho_do_certificado` e' obrigatorio",
        });
      } else if (identidade.caminho_do_certificado.includes("sem-eku")) {
        // Certificado sem o EKU de assinatura de codigo.
        sendJSON(response, 200, {
          somente_leitura: true,
          nada_assinado: true,
          avaliacao: {
            tipo: tipoAssinatura,
            aptidao: "SEM_EKU_DE_ASSINATURA",
            apta: false,
            sujeito: "CN=DZ23 LTDA, O=DZ23 LTDA",
            emissor: "CN=Test CA",
            valido_de: "2026-01-01T00:00:00Z",
            valido_ate: "2027-01-08T00:00:00Z",
            ekus: ["1.3.6.1.5.5.7.3.2"],
            tem_eku_necessario: false,
            motivo: "o certificado nao tem o EKU de assinatura de codigo (1.3.6.1.5.5.7.3.3)",
          },
          alvos_suportados: [
            "AUTHENTICODE",
            "APPLE_DEVELOPER_ID",
            "ANDROID_APK",
            "OPENPGP",
          ],
          aviso_de_fronteira:
            "esta rota apenas AVALIA se a identidade serve para o tipo pedido; nao le' a chave " +
            "privada, nao assina e nao escreve. `apta: true` significa que os gates que este " +
            "provider sabe aplicar passaram -- nao significa que o artefacto foi assinado. A " +
            "cadeia criptografica e a revogacao NAO sao validadas aqui.",
        });
      } else {
        sendJSON(response, 200, {
          somente_leitura: true,
          nada_assinado: true,
          avaliacao: {
            tipo: tipoAssinatura,
            aptidao: "APTA",
            apta: true,
            sujeito: "CN=DZ23 LTDA, O=DZ23 LTDA",
            emissor: "CN=DZ23 Code Signing CA",
            valido_de: "2026-01-01T00:00:00Z",
            valido_ate: "2027-10-17T00:00:00Z",
            ekus: ["1.3.6.1.5.5.7.3.3"],
            tem_eku_necessario: true,
            motivo: "",
          },
          alvos_suportados: [
            "AUTHENTICODE",
            "APPLE_DEVELOPER_ID",
            "ANDROID_APK",
            "OPENPGP",
          ],
          aviso_de_fronteira:
            "esta rota apenas AVALIA se a identidade serve para o tipo pedido; nao le' a chave " +
            "privada, nao assina e nao escreve. `apta: true` significa que os gates que este " +
            "provider sabe aplicar passaram -- nao significa que o artefacto foi assinado. A " +
            "cadeia criptografica e a revogacao NAO sao validadas aqui.",
        });
      }
    } else if (route === "POST /api/agent/v1/publish/rollback") {

      // FASE 28.7: `publish.rollback` -- PLANEIA, nunca restaura.
      //
      // `readBody` devolve uma STRING (lissao da E-1275): sem o JSON.parse o pedido ficava vazio.
      const corpoRollback = await readBody(request);
      let pedidoRollback = {};
      try {
        pedidoRollback = JSON.parse(corpoRollback || "{}");
      } catch {
        sendJSON(response, 400, { erro: "corpo invalido: esperado JSON" });
        return;
      }
      if (pedidoRollback === null || typeof pedidoRollback !== "object") {
        sendJSON(response, 400, { erro: "corpo invalido: esperado um objecto JSON" });
        return;
      }
      const tipoRollback = pedidoRollback.tipo;
      if (typeof tipoRollback !== "string" || tipoRollback === "") {
        sendJSON(response, 400, {
          erro: "o campo `tipo` e' obrigatorio: o canal de rollback nao tem valor por omissao",
        });
      } else if (tipoRollback === "WINDOWS_INSTALLER") {
        sendJSON(response, 200, {
          executado: false,
          suportado: true,
          tipo: "WINDOWS_INSTALLER",
          alvo: {
            caminho: "HadesSetup-anterior.exe",
            nome: "HadesSetup-anterior.exe",
            tamanho_bytes: 73400320,
            sha256:
              "7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730",
            modificado_em: "2026-10-17T09:12:00Z",
            parece_instalador: true,
          },
          total_de_candidatos: 2,
          integro: true,
          requer_aprovacao: "escolher o alvo e autorizar a restauracao",
          explicacao: "2 candidatos; o mais recente e' HadesSetup-anterior.exe",
          aviso: "este plano NAO executou o rollback",
          aviso_de_fronteira:
            "esta rota apenas PLANEIA o rollback: mede o SHA-256 dos candidatos e propoe o mais " +
            "recente. `executado` e' sempre false -- nenhum ficheiro foi movido, copiado ou " +
            "apagado. Restaurar um artefacto de producao exige aprovacao humana especifica no " +
            "momento da accao. O directorio vem da variavel de ambiente `HADES_ARTEFACTOS`, " +
            "nunca do pedido.",
        });
      } else {
        // Canal sem credencial/host: 409, nao 500. O fixture simula Play.
        sendJSON(response, 409, {
          erro:
            "rollback para `GOOGLE_PLAY_ROLLOUT` exige uma service account JSON com acesso ao " +
            "Play Console; nenhuma esta' configurada",
          suportado: false,
          executado: false,
          explicacao: "o canal existe mas falta credencial",
          aviso_de_fronteira:
            "esta rota apenas PLANEIA o rollback: `executado` e' sempre false -- nenhum ficheiro " +
            "foi movido, copiado ou apagado. O directorio vem da variavel de ambiente " +
            "`HADES_ARTEFACTOS`, nunca do pedido.",
        });
      }
    } else if (route === "GET /api/agent/v1/credentials/status") {


      // FASE 28.7: `credentials.status` -- INVENTARIA o kit, nunca o abre.
      //
      // O fixture simula um kit com um certificado publico e um segredo. O segredo vem com
      // `conteudo_lido: false` -- e o painel tem de mostrar isso como INFORMACAO, nao como falha.
      sendJSON(response, 200, {
        somente_leitura: true,
        configurado: true,
        credenciais: [
          {
            caminho_relativo: "apple/distribution.cer",
            tipo: "CERTIFICADO_X509",
            tamanho_bytes: 1485,
            modificado_em: "2026-10-17T12:00:00Z",
            conteudo_lido: true,
            fingerprint_sha256: "e2e-fixture",
            sujeito: "Apple Distribution: leandro prado (K8G36CUP3S)",
            emissor: "Apple Worldwide Developer Relations CA",
            valido_ate: "2027-10-17T00:00:00Z",
            expirado: false,
          },
          {
            caminho_relativo: "apple/hades-apple-distribution.p12",
            tipo: "CHAVE_PRIVADA",
            tamanho_bytes: 3512,
            modificado_em: "2026-10-17T12:00:00Z",
            conteudo_lido: false,
            expirado: false,
            nota: "conteudo NAO lido: apenas tamanho e data",
          },
        ],
        total: 2,
        lidos: 1,
        nao_lidos: 1,
        certificados: 1,
        expirados: 0,
        aviso: "este inventario lista o que o fixture declarou",
        aviso_de_fronteira:
          "esta rota apenas INVENTARIA o kit: os ficheiros de segredo NAO sao abertos. " +
          "O caminho vem da variavel de ambiente, nunca do pedido.",
      })
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
