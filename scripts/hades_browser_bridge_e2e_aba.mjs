/**
 * E2E DE CONTROLE DE ABA: pagina local, concessao por aba, DOM, campo sensivel, badge,
 * e a recusa quando a aba NAO foi concedida (`E-1033`).
 *
 * O `E-1032` provou o CANAL (extensao <-> Companion). Este prova a ABERTURA: o HADES
 * consegue observar e interagir com uma pagina real -- e **nao** consegue tocar no que
 * nao deve. Tudo numa pagina servida por um servidor local: nada sai da maquina.
 *
 * O que discrimina:
 *   - `observar` devolve a arvore saneada da pagina REAL;
 *   - `preencher` num campo normal ESCREVE, e o efeito e' verificado no DOM;
 *   - `preencher` num campo de senha e' **RECUSADO** com `campo_sensivel_takeover_humano`;
 *   - sem concessao, a accao e' recusada com `aba_nao_concedida`;
 *   - o badge «ON» aparece e desaparece.
 */
import { fileURLToPath, pathToFileURL } from "node:url";
import { execFileSync } from "node:child_process";
import http from "node:http";
import path from "node:path";
import fs from "node:fs";
import os from "node:os";

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const RAIZ = path.resolve(AQUI, "..");
const { chromium } = await import(
  pathToFileURL(path.join(RAIZ, "app", "ui", "app", "node_modules", "playwright", "index.mjs")).href
);
const EXT = path.join(RAIZ, "apps", "hades-browser-bridge");
const EDGE = "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe";

const falhas = [];
const ok = (n, c, d = "") => {
  console.log((c ? "  OK   " : "  FAIL ") + n + (d && !c ? "  -- " + d : ""));
  if (!c) falhas.push(n);
};

// ---------------------------------------------------------------- pagina local
const PAGINA = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>HADES: pagina de teste local</title></head>
<body>
  <h1>Pagina de teste</h1>
  <form onsubmit="return false">
    <label>Nome <input id="nome" type="text" autocomplete="off"></label>
    <label>Senha <input id="senha" type="password" autocomplete="current-password"></label>
    <label>Cartao <input id="cartao" type="text" autocomplete="cc-number"></label>
    <button id="ir" type="button">Ir</button>
  </form>
  <p id="resultado">nada</p>
  <script>
    document.getElementById("ir").addEventListener("click", () => {
      document.getElementById("resultado").textContent = "clicado:" + document.getElementById("nome").value;
    });
  </script>
</body></html>`;

const servidor = http.createServer((req, res) => {
  res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
  res.end(PAGINA);
});
await new Promise((r) => servidor.listen(0, "127.0.0.1", r));
const BASE = "http://127.0.0.1:" + servidor.address().port + "/";
console.log("0) servidor local em " + BASE);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-tab-"));
let ctx = null;
try {
  ctx = await chromium.launchPersistentContext(perfil, {
    executablePath: EDGE, headless: false,
    args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
      "--disable-features=DisableLoadExtensionCommandLineSwitch",
      "--no-first-run", "--no-default-browser-check"],
  });
  let sw = ctx.serviceWorkers()[0];
  if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 30000 }).catch(() => null);
  ok("service worker da extensao", !!sw);
  if (!sw) throw new Error("sem service worker");
  const id = new URL(sw.url()).host;

  // 1) A PAGINA LOCAL, e a extensao injecta o content script nela.
  console.log("1) abrir a pagina local");
  const pag = await ctx.newPage();
  await pag.goto(BASE, { waitUntil: "domcontentloaded" });
  ok("pagina local abriu", (await pag.title()).includes("teste"), await pag.title());
  await pag.waitForTimeout(1500);

  const temContent = await pag.evaluate(() => typeof chrome !== "undefined" && !!chrome.runtime?.id);
  console.log("   content script presente (chrome.runtime na pagina): " + temContent);

  // 2) A aba ATIVA e' a da pagina local. Conceder pelo popup (gesto real).
  console.log("2) conceder ESTA aba pelo popup");
  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(800);

  // O service worker ve' a aba ATIVA da janela ativa. Trazer a pagina para a frente.
  await pag.bringToFront();
  await popup.bringToFront();
  // O popup de verdade e' uma janela sobre a aba; aqui e' uma aba -- entao concedemos
  // pela API do proprio service worker, que e' o MESMO caminho que o popup usa.
  // 2a) SEM OPT-IN: a rede privada e' RECUSADA. Medir a recusa antes de medir o sucesso.
  console.log("2a) sem opt-in, a rede privada e' recusada (protecao SSRF)");
  const idDaPagina = await sw.evaluate(async (prefixo) => {
    const abas = await chrome.tabs.query({});
    const alvo = abas.find((a) => String(a.url).startsWith(prefixo));
    return alvo ? alvo.id : null;
  }, "http://127.0.0.1");

  const semOptIn = await popup.evaluate(async (t) => ({
    resp: await chrome.runtime.sendMessage({ tipo: "conectar", tabId: t }),
  }), idDaPagina);
  console.log("   resposta: " + JSON.stringify(semOptIn));
  ok("SEM opt-in -> `rede_privada_sem_optin`", semOptIn?.resp?.motivo === "rede_privada_sem_optin", JSON.stringify(semOptIn));
  ok("e nada foi concedido", semOptIn?.resp?.ok === false);

  // 2b) O opt-in e' do UTILIZADOR: liga-lo muda o comportamento, e so' entao.
  console.log("2b) ligar o opt-in de rede privada (decisao explicita)");
  const ligou = await popup.evaluate(async () => chrome.runtime.sendMessage({ tipo: "optin:definir", ligado: true }));
  const estadoOptIn = await popup.evaluate(async () => chrome.runtime.sendMessage({ tipo: "optin:estado" }));
  console.log("   definir: " + JSON.stringify(ligou) + "  estado: " + JSON.stringify(estadoOptIn));
  ok("opt-in ficou LIGADO", estadoOptIn?.ligado === true, JSON.stringify(estadoOptIn));

  // 2c) Agora sim: conceder a aba.
  console.log("2c) conceder a aba local");
  const conceder = await popup.evaluate(async (t) => ({
    resp: await chrome.runtime.sendMessage({ tipo: "conectar", tabId: t }),
  }), idDaPagina);
  console.log("   resposta: " + JSON.stringify(conceder).slice(0, 200));

  let tabId = conceder.resp?.tabId;
  console.log("   tabId concedido: " + tabId);
  ok("a aba local foi concedida", Number.isFinite(tabId), String(tabId));
  ok("a concessao e' da origem local", String(conceder.resp?.origem ?? "").startsWith("http://127.0.0.1"), String(conceder.resp?.origem));

  // 3) O BADGE «ON».
  console.log("3) badge persistente enquanto a aba esta' concedida");
  const badge = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  console.log("   badge: " + JSON.stringify(badge));
  ok("badge diz ON", badge === "ON", String(badge));

  // 4) SEM CONCESSAO: a accao e' recusada. Medir a recusa ANTES de medir o sucesso.
  console.log("4) uma aba NAO concedida e' recusada");
  const outra = await ctx.newPage();
  await outra.goto(BASE + "?outra=1", { waitUntil: "domcontentloaded" });
  await outra.waitForTimeout(800);
  const idDaOutra = await sw.evaluate(async () => {
    const abas = await chrome.tabs.query({});
    const alvo = abas.find((a) => String(a.url).includes("outra=1"));
    return alvo ? alvo.id : null;
  });
  const recusa = await popup.evaluate(async ([t, url]) =>
    ({ resp: await chrome.runtime.sendMessage({ tipo: "accao", scope: "browser:navigate", tabId: t, url }) }),
    [idDaOutra, BASE]).catch((e) => ({ excecao: String(e.message).slice(0, 80) }));
  console.log("   -> " + JSON.stringify(recusa));
  ok("aba nao concedida -> recusada", recusa?.resp?.ok === false, JSON.stringify(recusa));
  await outra.close();

  // 5) OBSERVAR a pagina real.
  console.log("5) observar o DOM da pagina concedida");
  const obs = await sw.evaluate(async (t) => {
    try { return await chrome.tabs.sendMessage(t, { tipo: "observar" }); }
    catch (e) { return { erro: String(e.message).slice(0, 120) }; }
  }, tabId);
  const temArvore = !!(obs?.observacao);
  console.log("   observacao: " + JSON.stringify(obs).slice(0, 200));
  ok("observar devolve uma arvore", temArvore, JSON.stringify(obs).slice(0, 120));

  // 6) PREENCHER num campo normal: escreve de verdade.
  console.log("6) preencher um campo NAO sensivel");
  const preencheu = await sw.evaluate(async (t) => {
    const r = await chrome.tabs.sendMessage(t, { tipo: "preencher", seletor: "#nome", valor: "HADES teste" });
    return r;
  }, tabId);
  console.log("   -> " + JSON.stringify(preencheu));
  const valorNoDom = await pag.evaluate(() => document.getElementById("nome")?.value);
  console.log("   valor no DOM: " + JSON.stringify(valorNoDom));
  ok("preencher devolveu ok", preencheu?.ok === true, JSON.stringify(preencheu));
  ok("o valor CHEGOU ao DOM", valorNoDom === "HADES teste", String(valorNoDom));

  // 7) O CAMPO DE SENHA E O CARTAO: recusados. E' o teste que mais importa.
  console.log("7) campo de SENHA e de CARTAO sao recusados (takeover humano)");
  for (const [sel, nome] of [["#senha", "senha"], ["#cartao", "cartao"]]) {
    const r = await sw.evaluate(async ([t, s]) => chrome.tabs.sendMessage(t, { tipo: "preencher", seletor: s, valor: "SEGREDO" }), [tabId, sel]);
    const noDom = await pag.evaluate((s) => document.getElementById(s.slice(1))?.value, sel);
    console.log("   " + nome + ": " + JSON.stringify(r) + "  valor no DOM=" + JSON.stringify(noDom));
    ok(nome + " recusado com motivo proprio", r?.ok === false && r?.motivo === "campo_sensivel_takeover_humano", JSON.stringify(r));
    ok(nome + " NAO foi escrito no DOM", noDom === "", JSON.stringify(noDom));
  }

  // 8) CLICAR e verificar o EFEITO (nao basta o clique ter corrido).
  console.log("8) clicar e verificar o efeito observavel");
  const clicou = await sw.evaluate(async (t) => chrome.tabs.sendMessage(t, { tipo: "clicar", seletor: "#ir" }), tabId);
  await pag.waitForTimeout(400);
  const resultado = await pag.evaluate(() => document.getElementById("resultado")?.textContent);
  console.log("   clique: " + JSON.stringify(clicou) + "  resultado no DOM: " + JSON.stringify(resultado));
  ok("clicar devolveu ok", clicou?.ok === true, JSON.stringify(clicou));
  ok("o EFEITO aconteceu no DOM", String(resultado).startsWith("clicado:"), String(resultado));

  // 9) REVOGAR: badge limpo e accao recusada.
  console.log("9) revogar a concessao");
  const rev = await popup.evaluate(async (t) => chrome.runtime.sendMessage({ tipo: "desconectar", tabId: t }), tabId);
  await new Promise((r) => setTimeout(r, 600));
  const badgeDepois = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  console.log("   revogacao: " + JSON.stringify(rev) + "  badge: " + JSON.stringify(badgeDepois));
  ok("revogacao ok", rev?.ok === true, JSON.stringify(rev));
  ok("badge limpo", !badgeDepois, JSON.stringify(badgeDepois));
  const aposRevogar = await popup.evaluate(async (t) => chrome.runtime.sendMessage({ tipo: "accao", scope: "browser:navigate", tabId: t, url: "http://127.0.0.1/" }), tabId);
  console.log("   accao apos revogar: " + JSON.stringify(aposRevogar));
  ok("apos revogar, a accao e' recusada", aposRevogar?.ok === false && aposRevogar?.motivo === "aba_nao_concedida", JSON.stringify(aposRevogar));

  await popup.close();
} catch (e) {
  console.log("   ERRO: " + e.message);
  falhas.push("execucao: " + e.message);
} finally {
  if (ctx) await ctx.close().catch(() => {});
  fs.rmSync(perfil, { recursive: true, force: true });
  servidor.close();
  if (/registado/.test(saidaInst)) desinstalar();
}

console.log("");
if (falhas.length) {
  console.log("FALHAS: " + falhas.length + " -> " + falhas.join(" | "));
  process.exit(1);
}
console.log("E2E DE ABA: concessao, observacao, escrita, RECUSA de campo sensivel, badge e revogacao.");
process.exit(0);
