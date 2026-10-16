/**
 * E2E DO BFCACHE (`E-1038`). A ultima divida da familia `onUpdated`.
 *
 * No `E-1037` ficou registado: *"o BFCache (voltar atras) nao foi testado, e pode repor uma
 * pagina de outra origem sem disparar onUpdated com URL novo."*
 *
 * O que se mede, com instrumentacao do `onUpdated`:
 *   1. Concede A. Navega REALMENTE para B -> revoga (ja' sabido do `E-1035`).
 *   2. `goBack()` para A -> a pagina volta do BFCache. **A concessao continua revogada?**
 *      E o regresso trouxe URL no `onUpdated`, ou so' `status`?
 *   3. O teste decisivo: **o HADES re-concede sozinho ao voltar?** *Reconexao automatica de
 *      permissao seria exactamente o que o contrato proibe* -- `reconexao nunca restaura
 *      automaticamente permissao de alto risco expirada`.
 *   4. E o adversario do BFCache: com a concessao de A VIVA, ir a B e voltar -- a concessao
 *      de A revive sobre uma pagina que ja' foi de outra origem?
 *
 * Registo do que chega ao listener em CADA passo, para nao inferir nada.
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

function subir(nome) {
  const s = http.createServer((req, res) => {
    res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    res.end(`<!DOCTYPE html><html lang="pt-BR"><head><meta charset="utf-8"><title>${nome}</title></head>
<body><h1>${nome}</h1>
<script>
  window.__marcas = [];
  window.addEventListener("pageshow", (e) => window.__marcas.push({e:"pageshow", persisted: e.persisted, t: Date.now()}));
  window.addEventListener("pagehide", (e) => window.__marcas.push({e:"pagehide", persisted: e.persisted, t: Date.now()}));
</script></body></html>`);
  });
  return new Promise((r) => s.listen(0, "127.0.0.1", () => r({ s, base: "http://127.0.0.1:" + s.address().port + "/" })));
}

const A = await subir("origem-A");
const B = await subir("origem-B");
const origemA = new URL(A.base).origin;
const origemB = new URL(B.base).origin;
console.log("0) A=" + origemA + "  B=" + origemB);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-bf-"));
let ctx = null;
try {
  ctx = await chromium.launchPersistentContext(perfil, {
    executablePath: EDGE, headless: false,
    args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
      "--disable-features=DisableLoadExtensionCommandLineSwitch", "--no-first-run", "--no-default-browser-check"],
  });
  let sw = ctx.serviceWorkers()[0];
  if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 30000 }).catch(() => null);
  ok("service worker da extensao", !!sw);
  if (!sw) throw new Error("sem service worker");
  const id = new URL(sw.url()).host;

  await sw.evaluate(() => {
    self.__ev = [];
    chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
      self.__ev.push({
        chaves: Object.keys(info),
        url: info.url === undefined ? "<<AUSENTE>>" : info.url,
        status: info.status === undefined ? "<<AUSENTE>>" : info.status,
        tabUrl: tab?.url ?? "<<AUSENTE>>",
      });
    });
  });

  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(600);
  const pedir = (m) => popup.evaluate(async (mm) => chrome.runtime.sendMessage(mm), m);
  await pedir({ tipo: "optin:definir", ligado: true });

  const aba = await ctx.newPage();
  await aba.goto(A.base);
  await aba.waitForTimeout(1500);
  const tabId = await sw.evaluate(async () => {
    const t = await chrome.tabs.query({});
    const a = t.find((x) => String(x.url).startsWith("http://127.0.0.1"));
    return a ? a.id : null;
  });
  const limpar = () => sw.evaluate(() => { self.__ev = []; });
  const eventos = () => sw.evaluate(() => self.__ev);
  const concessoes = async () => ((await pedir({ tipo: "estado" }))?.concessoes ?? []).map((x) => x.tabId);
  const badge = () => sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  const despejar = async (titulo) => {
    const ev = await eventos();
    console.log("   " + titulo);
    for (const e of ev) console.log("     " + JSON.stringify(e));
  };

  console.log("\n1) concede A, navega REALMENTE para B, e volta com goBack");
  await pedir({ tipo: "conectar", tabId });
  ok("concedida em A", (await concessoes()).includes(tabId));

  await limpar();
  await aba.goto(B.base);
  await aba.waitForTimeout(1500);
  await despejar("eventos na ida para B:");
  ok("a concessao CAIU na ida", !(await concessoes()).includes(tabId), "ainda viva");

  await limpar();
  await aba.goBack();
  await aba.waitForTimeout(2000);
  await despejar("eventos no REGRESSO para A (goBack/BFCache):");
  const voltouA = await aba.evaluate(() => location.origin);
  ok("a pagina voltou a A", voltouA === origemA, String(voltouA));
  ok("a concessao NAO revive sozinha", !(await concessoes()).includes(tabId), "reviveu!");
  ok("badge continua limpo", !(await badge()), JSON.stringify(await badge()));

  const marcas = await aba.evaluate(() => window.__marcas ?? []);
  console.log("   marcas da pagina (pageshow/pagehide): " + JSON.stringify(marcas));

  console.log("\n2) e a accao segue recusada depois de voltar");
  const a1 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId, url: origemA + "/x" });
  ok("accao recusada", a1?.ok === false && a1?.motivo === "aba_nao_concedida", JSON.stringify(a1));

  console.log("\n3) o adversario do BFCache: ir a B e voltar NAO deixa concessao viva");
  await pedir({ tipo: "conectar", tabId });
  ok("re-concedida em A", (await concessoes()).includes(tabId));
  await aba.goto(B.base);
  await aba.waitForTimeout(1400);
  ok("caiu de novo na ida", !(await concessoes()).includes(tabId), "ainda viva");
  await aba.goBack();
  await aba.waitForTimeout(2000);
  const est = await pedir({ tipo: "estado" });
  console.log("   no regresso: " + JSON.stringify((est?.concessoes ?? []).map((c) => ({ tabId: c.tabId, origem: c.origem }))));
  ok("NAO ha' concessao viva no regresso", (est?.concessoes ?? []).length === 0, JSON.stringify(est?.concessoes));
  ok("e o badge esta' limpo", !(await badge()), JSON.stringify(await badge()));

  console.log("\n4) e re-conceder em A, depois ir a B: cai outra vez (sem excepcao de BFCache)");
  await pedir({ tipo: "conectar", tabId });
  ok("re-concedida", (await concessoes()).includes(tabId));
  await aba.goto(B.base + "?terceira=1");
  await aba.waitForTimeout(1500);
  ok("cai na terceira ida", !(await concessoes()).includes(tabId), "ainda viva");

  await popup.close();
} catch (e) {
  console.log("   ERRO: " + e.message);
  falhas.push("execucao: " + e.message);
} finally {
  if (ctx) await ctx.close().catch(() => {});
  fs.rmSync(perfil, { recursive: true, force: true });
  A.s.close(); B.s.close();
  if (/registado/.test(saidaInst)) desinstalar();
}

console.log("");
if (falhas.length) {
  console.log("FALHAS: " + falhas.length + " -> " + falhas.join(" | "));
  process.exit(1);
}
console.log("E2E DO BFCACHE: voltar atras NAO revive a concessao -- e o HADES nao re-concede sozinho.");
process.exit(0);
