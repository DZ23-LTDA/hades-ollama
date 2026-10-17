/**
 * E2E DE TAKEOVER HUMANO (`E-1034`).
 *
 * O scope `browser:takeover` estava nos 3 canonicos e **nao tinha uma linha de
 * implementacao**. E o contrato exige: *enquanto o utilizador estiver a interagir, o
 * agente nao disputa mouse/teclado nem sobrescreve campos.*
 *
 * O que discrimina:
 *   - sem interaccao humana, `preencher` e `clicar` funcionam (o agente tem o controlo);
 *   - depois de o HUMANO digitar, o agente e' RECUSADO com `takeover_humano_ativo`;
 *   - e o VALOR que o humano escreveu NAO e' sobrescrito;
 *   - o service worker regista o takeover com autor `human`;
 *   - `takeover:devolver` devolve a sessao e o agente volta a poder escrever.
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
// **E-1083: o browser vem do ambiente, e nao do codigo.** *O prompt exige Chrome E Edge; com o
// caminho do Edge fixo, so' um deles era testavel -- e a constante `EDGE` afirmava o browser.*
const BROWSERS = {
  edge: "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
  chrome: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
};
const QUAL = (process.env.HADES_E2E_BROWSER || "edge").trim().toLowerCase();
if (!(QUAL in BROWSERS)) {
  console.error(`  HADES_E2E_BROWSER="${QUAL}" nao suportado. Aceita: ${Object.keys(BROWSERS).join(", ")}.`);
  if (QUAL === "firefox" || QUAL === "webkit") {
    console.error("  Firefox/WebKit NAO sao suportados: a extensao e' Manifest V3, e o caminho de carga");
    console.error("  e' o de Chromium (--load-extension). Testa-la noutro motor exige outro manifesto.");
  }
  process.exit(2);
}
const EDGE = BROWSERS[QUAL];
const CHROMIUM = true; // *as duas entradas acima sao Chromium*

const falhas = [];
const ok = (n, c, d = "") => {
  console.log((c ? "  OK   " : "  FAIL ") + n + (d && !c ? "  -- " + d : ""));
  if (!c) falhas.push(n);
};

const PAGINA = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>HADES: takeover</title></head>
<body>
  <label>Campo <input id="campo" type="text" autocomplete="off"></label>
  <button id="ir" type="button">Ir</button>
  <p id="resultado">nada</p>
  <script>
    document.getElementById("ir").addEventListener("click", () => {
      document.getElementById("resultado").textContent = "clicado";
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
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-tk-"));
let ctx = null;
try {
  ctx = await chromium.launchPersistentContext(perfil, {
    executablePath: EDGE, headless: false,
    args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
      // *O Chrome 155 removeu `--load-extension`; o Edge ainda o aceita com este workaround.*
        // => *declarado, e nao silencioso: o leitor sabe que ha um switch a compensar.*
        "--disable-features=DisableLoadExtensionCommandLineSwitch",
      "--no-first-run", "--no-default-browser-check"],
  });
  let sw = ctx.serviceWorkers()[0];
  if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 30000 }).catch(() => null);
  ok("service worker da extensao", !!sw);
  if (!sw) throw new Error("sem service worker");
  const id = new URL(sw.url()).host;

  const pag = await ctx.newPage();
  await pag.goto(BASE, { waitUntil: "domcontentloaded" });
  await pag.waitForTimeout(1200);
  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(600);

  // Preparar: opt-in + concessao.
  await popup.evaluate(async () => chrome.runtime.sendMessage({ tipo: "optin:definir", ligado: true }));
  const idDaPagina = await sw.evaluate(async () => {
    const abas = await chrome.tabs.query({});
    const alvo = abas.find((a) => String(a.url).startsWith("http://127.0.0.1"));
    return alvo ? alvo.id : null;
  });
  const conf = await popup.evaluate(async (t) => chrome.runtime.sendMessage({ tipo: "conectar", tabId: t }), idDaPagina);
  ok("aba concedida", conf?.ok === true, JSON.stringify(conf));
  const tabId = conf.tabId;

  const falar = (msg) => sw.evaluate(async ([t, m]) => chrome.tabs.sendMessage(t, m), [tabId, msg]);
  const pedir = (m) => popup.evaluate(async (mm) => chrome.runtime.sendMessage(mm), m);

  // 1) SEM interaccao humana: o AGENTE tem o controlo.
  console.log("1) sem interaccao humana, o agente escreve e clica");
  const r1 = await falar({ tipo: "preencher", seletor: "#campo", valor: "escrito pelo agente" });
  const v1 = await pag.evaluate(() => document.getElementById("campo")?.value);
  console.log("   preencher: " + JSON.stringify(r1) + "  DOM=" + JSON.stringify(v1));
  ok("agente escreve sem takeover", r1?.ok === true, JSON.stringify(r1));
  ok("valor no DOM", v1 === "escrito pelo agente", String(v1));

  const c1 = await falar({ tipo: "clicar", seletor: "#ir" });
  await pag.waitForTimeout(300);
  const res1 = await pag.evaluate(() => document.getElementById("resultado")?.textContent);
  ok("agente clica sem takeover", c1?.ok === true, JSON.stringify(c1));
  ok("efeito do clique", res1 === "clicado", String(res1));

  // 2) O HUMANO age: digitar na pagina (gesto REAL do Playwright).
  console.log("2) o HUMANO digita na pagina");
  await pag.bringToFront();
  // Limpar SEM gerar gesto: o campo tinha o texto do agente e queremos medir o do humano.
  await pag.evaluate(() => { document.getElementById("campo").value = ""; });
  await pag.click("#campo");
  await pag.keyboard.type("escrito pelo humano", { delay: 30 });
  await pag.waitForTimeout(700);
  const vHumano = await pag.evaluate(() => document.getElementById("campo")?.value);
  console.log("   valor do humano: " + JSON.stringify(vHumano));
  ok("o humano escreveu", vHumano === "escrito pelo humano", String(vHumano));

  // 3) O SERVICE WORKER soube. Autor `human`.
  console.log("3) o takeover foi registado com autor `human`");
  const est = await pedir({ tipo: "takeover:estado", tabId });
  console.log("   estado: " + JSON.stringify(est));
  ok("takeover ATIVO", est?.ativo === true, JSON.stringify(est));
  ok("autor e' `human`", est?.autor === "human", JSON.stringify(est));

  // 4) O AGENTE E' RECUSADO -- e o valor do humano NAO e' sobrescrito.
  console.log("4) o agente e' recusado enquanto o humano tem o controlo");
  const r2 = await falar({ tipo: "preencher", seletor: "#campo", valor: "AGENTE A SOBRESCREVER" });
  const v2 = await pag.evaluate(() => document.getElementById("campo")?.value);
  console.log("   preencher: " + JSON.stringify(r2) + "  DOM=" + JSON.stringify(v2));
  ok("preencher recusado com `takeover_humano_ativo`", r2?.ok === false && r2?.motivo === "takeover_humano_ativo", JSON.stringify(r2));
  ok("O VALOR DO HUMANO NAO FOI SOBRESCRITO", v2 === "escrito pelo humano", String(v2));

  const c2 = await falar({ tipo: "clicar", seletor: "#ir" });
  console.log("   clicar: " + JSON.stringify(c2));
  ok("clicar tambem recusado", c2?.ok === false && c2?.motivo === "takeover_humano_ativo", JSON.stringify(c2));

  // 5) O UTILIZADOR devolve a sessao.
  console.log("5) o utilizador devolve a sessao ao agente");
  const dev = await pedir({ tipo: "takeover:devolver", tabId });
  await falar({ tipo: "takeover:devolver" });
  const est2 = await pedir({ tipo: "takeover:estado", tabId });
  console.log("   devolver: " + JSON.stringify(dev) + "  estado: " + JSON.stringify(est2));
  ok("takeover encerrado", est2?.ativo === false, JSON.stringify(est2));

  // 6) E o agente volta a poder escrever.
  console.log("6) o agente volta a poder escrever");
  const r3 = await falar({ tipo: "preencher", seletor: "#campo", valor: "agente de novo" });
  const v3 = await pag.evaluate(() => document.getElementById("campo")?.value);
  console.log("   preencher: " + JSON.stringify(r3) + "  DOM=" + JSON.stringify(v3));
  ok("agente escreve apos a devolucao", r3?.ok === true, JSON.stringify(r3));
  ok("valor novo no DOM", v3 === "agente de novo", String(v3));

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
console.log("E2E DE TAKEOVER: o humano assume, o agente e' recusado, o valor nao e' sobrescrito, e a sessao volta.");
process.exit(0);
