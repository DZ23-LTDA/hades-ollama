/**
 * E2E do **takeover humano** e da **revogação ao vivo** da extensão -- §ESTÁGIO 14 (`E-1117`).
 *
 * ***O QUE FALTAVA.*** *O `E-1105` provou que a extensão carrega; o `E-1115` provou que ela **opera** (DOM,
 * clique, preenchimento, navegação, e o negativo da senha). Mas o §FASE-3-B exige mais duas coisas que só um
 * teste **ao vivo** pode provar:*
 *
 *   - *«o usuário pode **pausar** instantaneamente, **assumir controle** e devolver a sessão»*;
 *   - *«botão de **desconectar/revogar** acessível na extensão e no HADES»*.
 *
 * ***E COMO ELE FALA COM A EXTENSÃO SEM INVENTAR UM ATALHO.*** *O `chrome.runtime.onMessage` do service worker
 * **recusa qualquer remetente que não seja a própria extensão** (`remetente.id !== chrome.runtime.id` =>
 * `remetente_nao_autorizado`). Logo o teste **não** pode chamar a função interna: tem de usar o **caminho real**.
 * => *abre o **popup** numa aba (`chrome-extension://ID/popup/popup.html`) e envia `chrome.runtime.sendMessage`
 * de lá -- que é exactamente o que o utilizador faz ao clicar no ícone.* ***O teste usa a porta, não a fechadura.***
 *
 * ***COMO CORRER.*** `node apps/hades-browser-bridge/test/e2e-takeover-real.mjs`
 */
import { chromium } from "playwright";
import { createServer } from "node:http";
import { mkdtempSync, rmSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = dirname(fileURLToPath(import.meta.url));
const RAIZ = resolve(AQUI, "..", "..", "..");
const EXTENSAO = resolve(RAIZ, "apps", "hades-browser-bridge");

const resultados = [];
function T(nome, ok, detalhe = "") {
  resultados.push({ nome, ok, detalhe });
  console.log(`${ok ? "  OK   " : "  FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

const PAGINA = `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>Takeover</title></head>
<body>
  <h1 id="titulo">Página de takeover</h1>
  <input id="campo" type="text" />
  <button id="btn">Botão</button>
</body></html>`;

function servidor() {
  const srv = createServer((req, res) => {
    res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    res.end(PAGINA);
  });
  return new Promise((ok) => srv.listen(0, "127.0.0.1", () => ok(srv)));
}

async function main() {
  if (!existsSync(EXTENSAO)) {
    console.log(`  BLOCKED: a extensao nao esta' em ${EXTENSAO}`);
    process.exit(2);
  }
  const srv = await servidor();
  const base = `http://127.0.0.1:${srv.address().port}`;
  const perfil = mkdtempSync(join(tmpdir(), "hades-e2e-takeover-"));
  console.log(`  servidor: ${base}`);
  console.log("");

  let ctx;
  try {
    // ***`E-1121`: O CANAL E' ESCOLHIDO POR AMBIENTE*** -- *o mesmo contrato do `e2e-extension-dom.mjs`,
    // para o takeover poder correr no **Edge real** e nao so' no Chromium do Playwright.*
    const canal = process.env.HADES_E2E_CHANNEL || "";
    ctx = await chromium.launchPersistentContext(perfil, {
      headless: false,
      ...(canal ? { channel: canal } : {}),
      args: [`--disable-extensions-except=${EXTENSAO}`, `--load-extension=${EXTENSAO}`,
        "--no-first-run", "--no-default-browser-check"],
    });
    T("(0) o motor: " + (canal || "chromium do Playwright"), true, "versao=" + (ctx.browser()?.version?.() || "?"));
    let sw = ctx.serviceWorkers()[0];
    if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 20000 }).catch(() => null);
    T("(1) a extensao carregou e o service_worker acordou", !!sw);
    if (!sw) throw new Error("sem service worker");
    const id = new URL(sw.url()).host;

    // ***O POPUP E' A PORTA.*** *e' o unico remetente autorizado (`remetente.id === chrome.runtime.id`),
    // alem do content script -- e o content script nao tem acesso a `chrome.runtime.sendMessage` com resposta.
    const popup = await ctx.newPage();
    await popup.goto(`chrome-extension://${id}/popup/popup.html`, { waitUntil: "domcontentloaded" });
    T("(1b) o popup abriu (o caminho REAL, nao um hook de teste)", popup.url().includes(id),
      popup.url().slice(0, 60));

    /** A porta real: `chrome.runtime.sendMessage` a partir do popup. */
    const enviar = async (msg) => popup.evaluate(async (m) => {
      try {
        return await chrome.runtime.sendMessage(m);
      } catch (e) {
        return { ok: false, motivo: "erro_no_sendMessage", detalhe: String(e?.message ?? e) };
      }
    }, msg);

    // ---------- (2) um remetente EXTERNO e' recusado ----------
    // *uma pagina normal tenta falar com o SW: `chrome.runtime` nao existe la', logo o teste prova que a
    // porta **nao** esta' aberta a paginas.*
    const paginaQualquer = await ctx.newPage();
    await paginaQualquer.goto("about:blank");
    const temRuntime = await paginaQualquer.evaluate(() => typeof chrome !== "undefined" && !!chrome.runtime);
    T("(2) uma pagina normal **nao** tem acesso a `chrome.runtime` (a porta esta' fechada)", temRuntime === false);
    await paginaQualquer.close();

    // ---------- (3) conectar ----------
    const aba = await ctx.newPage();
    await aba.goto(base + "/", { waitUntil: "domcontentloaded" });
    const tabId = await sw.evaluate(async (url) => {
      const abas = await chrome.tabs.query({ url });
      return abas[0] ? abas[0].id : null;
    }, base + "/*");
    T("(3) a aba existe e o service worker ve-a", tabId !== null, `tabId=${tabId}`);
    if (tabId === null) throw new Error("sem aba");

    // ***O `127.0.0.1` E' REDE PRIVADA -- e o `conectar` exige OPT-IN EXPLICITO.*** *Medido no `E-1117`: sem
    // o opt-in, o `conectar` devolve `rede_privada_sem_optin`. Isso NAO e' um defeito: e' o contrato -- o
    // prompt diz «nunca aceitar conexao publica/LAN por padrao». => *o teste tem de dar o opt-in, como o
    // utilizador faria.*
    const op = await enviar({ tipo: "optin:definir", ligado: true });
    T("(3a) o **opt-in** de rede privada pode ser ligado (e' o utilizador que decide)",
      op?.ok === true, JSON.stringify(op).slice(0, 70));

    const c = await enviar({ tipo: "conectar", tabId });
    T("(3b) **conectar** criou a concessao", c?.ok === true, JSON.stringify(c).slice(0, 80));

    const est = await enviar({ tipo: "estado" });
    const n = (est?.concessoes || []).length;
    T("(3c) e o **estado** lista a concessao", n >= 1, `${n} concessao(oes)`);
    const escopos = est?.escopos || [];
    T("(3d) e os scopes sao os **3 canonicos**", escopos.length === 3 && escopos.includes("browser:takeover"),
      escopos.join(","));

    // ---------- (4) accao numa aba concedida -> PEDIDO, nao aprovacao ----------
    const a1 = await enviar({ tipo: "accao", tabId, scope: "browser:navigate", action: "navegar",
      url: base + "/outra", params: {} });
    T("(4) **accao** numa aba concedida devolve um PEDIDO", a1?.ok === true && !!a1?.pedido,
      a1?.motivo || "pedido presente");
    T("(4b) e o pedido exige **aprovacao** (`requires_approval: true`)", a1?.pedido?.requires_approval === true,
      "a extensao NAO decide aprovacao: declara a exigencia");
    T("(4c) e o pedido carrega o scope e a origem", a1?.pedido?.scope === "browser:navigate" &&
      typeof a1?.pedido?.origem === "string", `origem=${a1?.pedido?.origem}`);

    // ---------- (5) scope NAO canonico ----------
    const a2 = await enviar({ tipo: "accao", tabId, scope: "browser:controlo_total", action: "navegar" });
    T("(5) um **scope fora dos canonicos** e' recusado", a2?.ok === false && a2?.motivo === "scope_nao_canonico",
      a2?.motivo);

    // ---------- (6) TAKEOVER ----------
    // ***A ORDEM IMPORTA, E EU ERREI-A.*** *Medido no `E-1117`: se o gesto humano correr ANTES das
    // verificacoes do estado inativo, o takeover ja' esta' ativo e o teste falha por **ordem**, nao por
    // defeito.* => *a sequencia correcta e': estado inativo -> escrita funciona -> GESTO -> escrita recusada
    // -> devolver -> inativo.*

    // *o `chrome.tabs.sendMessage` e' o caminho REAL de EXECUCAO: e' o content script que executa e recusa.*
    const paraAba = async (msg) => popup.evaluate(async ([tid, m]) => {
      try {
        return await chrome.tabs.sendMessage(tid, m);
      } catch (e) {
        return { ok: false, motivo: "erro_no_sendMessage", detalhe: String(e?.message ?? e) };
      }
    }, [tabId, msg]);

    const tk0 = await enviar({ tipo: "takeover:estado", tabId });
    T("(6a) no service worker, o takeover esta' **inativo** antes do gesto", tk0?.ok === true && tk0?.ativo !== true,
      `ativo=${tk0?.ativo}`);

    const cs0 = await paraAba({ tipo: "takeover:estado" });
    T("(6b) e no **content script** tambem", cs0?.ok === true && cs0?.ativo !== true, `ativo=${cs0?.ativo}`);

    // ***A ESCRITA DO AGENTE FUNCIONA ANTES.*** *sem isto, o (6f) nao provaria nada: um teste que so' verifica
    // a recusa nao distingue «recusou por takeover» de «nunca funcionou».*
    const p1 = await paraAba({ tipo: "preencher", seletor: "#campo", valor: "escrita do agente" });
    T("(6c) e a escrita do agente **funciona** antes do takeover", p1?.ok === true, p1?.motivo || "preencheu");

    // *um gesto HUMANO real: `pointerdown` na pagina*
    await aba.click("#campo");
    await aba.waitForTimeout(500);

    const tk1 = await enviar({ tipo: "takeover:estado", tabId });
    // *o `ativo` esta' no TOPO da resposta (`{ok, ativo, takeover, autor}`), nao dentro do `takeover` --
    // medido no `E-1117`, e era um erro MEU de leitura.*
    T("(6d) um **gesto humano** marca o takeover (no service worker)", tk1?.ok === true && tk1?.ativo === true,
      `ativo=${tk1?.ativo} autor=${tk1?.autor} motivo=${tk1?.takeover?.motivo}`);

    const cs1 = await paraAba({ tipo: "takeover:estado" });
    T("(6e) e o **content script** ve o takeover ativo", cs1?.ok === true && cs1?.ativo === true,
      `ativo=${cs1?.ativo} motivo=${cs1?.motivo}`);

    // ***E AGORA A ESCRITA E' RECUSADA -- o travao que existe para travar a automacao.***
    const p2 = await paraAba({ tipo: "preencher", seletor: "#campo", valor: "escrita durante o takeover" });
    T("(6f) e a **escrita do agente e' RECUSADA** durante o takeover",
      p2?.ok === false && p2?.motivo === "takeover_humano_ativo", p2?.motivo || JSON.stringify(p2).slice(0, 60));

    // *e o utilizador DEVOLVE a sessao -- o contrario do takeover*
    const dev = await enviar({ tipo: "takeover:devolver", tabId });
    T("(6g) e o utilizador **devolve a sessao** (`takeover:devolver`)", dev?.ok === true,
      JSON.stringify(dev).slice(0, 70));

    const tk2 = await enviar({ tipo: "takeover:estado", tabId });
    T("(6h) e o takeover volta a **inativo**", tk2?.ok === true && tk2?.ativo !== true, `ativo=${tk2?.ativo}`);

    // *e a escrita volta a funcionar -- a devolucao e' REAL, nao so' do estado*
    const p3 = await paraAba({ tipo: "preencher", seletor: "#campo", valor: "escrita depois da devolucao" });
    T("(6i) e a escrita do agente **volta a funcionar** depois de devolver", p3?.ok === true,
      p3?.motivo || "preencheu");


    // ---------- (7) REVOGAR ----------
    const d = await enviar({ tipo: "desconectar", tabId });
    T("(7) **desconectar** devolve ok", d?.ok === true, JSON.stringify(d).slice(0, 70));

    const est2 = await enviar({ tipo: "estado" });
    const n2 = (est2?.concessoes || []).length;
    T("(7b) e a concessao **desapareceu** do estado", n2 === 0, `${n2} concessao(oes)`);

    const a4 = await enviar({ tipo: "accao", tabId, scope: "browser:navigate", action: "navegar", url: base + "/x" });
    T("(7c) e uma accao depois de revogar e' **recusada**", a4?.ok === false && a4?.motivo === "aba_nao_concedida",
      a4?.motivo);

    await aba.close();
    await popup.close();
  } finally {
    if (ctx) await ctx.close().catch(() => {});
    srv.close();
    rmSync(perfil, { recursive: true, force: true });
  }

  const falhas = resultados.filter((r) => !r.ok);
  console.log("");
  if (falhas.length) {
    console.log(`  ==== ${falhas.length} FALHA(S) ====`);
    for (const f of falhas) console.log(`    - ${f.nome}`);
    process.exit(1);
  }
  console.log(`  ==== O E2E DE TAKEOVER E REVOGACAO PASSOU (${resultados.length} verificacoes) ====`);
}

main().catch((e) => {
  console.error("  ERRO: " + (e && e.message ? e.message : e));
  process.exit(1);
});
