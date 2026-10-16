/**
 * E2E DE FRAMES E SHADOW DOM (`E-1039`).
 *
 * O contrato exige *"suportar frames e shadow DOM com limites"*. O `observar` usa
 * `document.querySelectorAll` -- **que nao atravessa frames nem penetra shadow roots**.
 *
 * A pergunta que discrimina NAO e' "suporta?" -- e': **o `observar` DIZ que nao viu?**
 *
 * *Um observador que devolve uma lista parcial como se fosse completa faz o modelo agir sobre
 * uma pagina que nao e' a que existe.* O defeito nao seria nao suportar frames: seria **nao
 * declarar** que nao os viu.
 *
 * Pagina de teste, numa so origem:
 *   - documento principal:  `#raiz` (texto visivel)
 *   - shadow DOM ABERTO:   `#no-shadow`  (dentro do shadow root)
 *   - shadow DOM FECHADO:  `#no-fechado` (inacessivel por design)
 *   - IFRAME mesma origem: `#no-iframe`
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

const PRINCIPAL = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>frames</title></head>
<body>
  <h1>Principal</h1>
  <button id="raiz">botao da raiz</button>
  <div id="hospedeiro"></div>
  <div id="hospedeiro-fechado"></div>
  <iframe id="quadro" src="/interno" width="300" height="120"></iframe>
  <script>
    const raiz = document.getElementById("hospedeiro").attachShadow({ mode: "open" });
    raiz.innerHTML = '<button id="no-shadow">botao no shadow</button>';
    document.getElementById("hospedeiro-fechado").attachShadow({ mode: "closed" })
      .innerHTML = '<button id="no-fechado">botao no shadow fechado</button>';
  </script>
</body></html>`;

const INTERNA = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>interna</title></head>
<body><h1>Interna</h1><button id="no-iframe">botao no iframe</button></body></html>`;

const servidor = http.createServer((req, res) => {
  res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
  res.end(String(req.url).startsWith("/interno") ? INTERNA : PRINCIPAL);
});
await new Promise((r) => servidor.listen(0, "127.0.0.1", r));
const BASE = "http://127.0.0.1:" + servidor.address().port + "/";
console.log("0) servidor em " + BASE);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-fr-"));
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

  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(600);
  const pedir = (m) => popup.evaluate(async (mm) => chrome.runtime.sendMessage(mm), m);
  await pedir({ tipo: "optin:definir", ligado: true });

  const aba = await ctx.newPage();
  await aba.goto(BASE);
  await aba.waitForTimeout(1500);
  const tabId = await sw.evaluate(async () => {
    const t = await chrome.tabs.query({});
    const a = t.find((x) => String(x.url).startsWith("http://127.0.0.1"));
    return a ? a.id : null;
  });
  const conf = await pedir({ tipo: "conectar", tabId });
  ok("aba concedida", conf?.ok === true, JSON.stringify(conf));

  // A verdade do terreno: o que o DOM REALMENTE tem.
  const verdade = await aba.evaluate(() => {
    const sombraAberta = document.getElementById("hospedeiro").shadowRoot?.querySelector("#no-shadow");
    const quadro = document.getElementById("quadro");
    return {
      raiz: !!document.getElementById("raiz"),
      shadowAberto: !!sombraAberta,
      iframeInterno: !!quadro?.contentDocument?.getElementById("no-iframe"),
      raizAberta: !!document.getElementById("hospedeiro").shadowRoot,
    };
  });
  console.log("1) a verdade do terreno: " + JSON.stringify(verdade));
  ok("o botao da raiz existe", verdade.raiz);
  ok("o botao no shadow ABERTO existe", verdade.shadowAberto);
  ok("o botao no IFRAME de mesma origem existe", verdade.iframeInterno);

  // O que o `observar` devolve.
  const obs = await aba.evaluate(async (t) => {
    return chrome.runtime.sendMessage({ tipo: "observar", tabId: t });
  }, tabId).catch(() => null);

  // `observar` e' uma mensagem do content script; o popup fala com o worker. Aqui falamos
  // direto com o content script pelo service worker.
  const obs2 = await sw.evaluate(async (t) => chrome.tabs.sendMessage(t, { tipo: "observar" }), tabId);
  const O = obs2?.observacao ?? obs2 ?? {};
  console.log("2) o que o `observar` devolveu:");
  console.log("   chaves=" + JSON.stringify(Object.keys(O)));
  console.log("   total=" + O.total + "  limites=" + JSON.stringify(O.limites));
  const rotulos = JSON.stringify((O.nos ?? []).map((n) => n.rotulo));
  console.log("   rotulos: " + rotulos.slice(0, 300));
  void obs;

  const veRaiz = rotulos.includes("botao da raiz");
  const veShadow = rotulos.includes("botao no shadow");
  const veIframe = rotulos.includes("botao no iframe");
  console.log("   ve a raiz=" + veRaiz + "  ve o shadow aberto=" + veShadow + "  ve o iframe=" + veIframe);

  ok("ve a raiz", veRaiz, "nao viu o documento principal");
  ok("ve o shadow ABERTO (`E-1039`)", veShadow, "nao atravessou o shadow aberto");
  ok("ve o IFRAME de mesma origem (`E-1039`)", veIframe, "nao atravessou o iframe");

  console.log("\n3) E DECLARA OS LIMITES -- a pergunta que discrimina");
  const L = O.limites ?? null;
  console.log("   limites=" + JSON.stringify(L));
  ok("devolve o campo `limites`", !!L, "nao declara nada: a lista parece completa");
  ok("declara `parcial`", typeof L?.parcial === "boolean", JSON.stringify(L?.parcial));
  ok("conta os shadow roots ABERTOS incluidos", (L?.shadowRoots?.abertosIncluidos ?? 0) >= 1, JSON.stringify(L?.shadowRoots));
  ok("conta os frames existentes", (L?.frames?.existentes ?? 0) >= 1, JSON.stringify(L?.frames));
  ok("conta os frames de mesma origem incluidos", (L?.frames?.mesmaOrigemIncluidos ?? 0) >= 1, JSON.stringify(L?.frames));
  ok("declara a profundidade maxima (limite honesto)",
    (L?.shadowRoots?.profundidadeMaxima ?? 0) > 0 && (L?.frames?.profundidadeMaxima ?? 0) > 0,
    JSON.stringify(L));
  ok("o total e' o total VISTO, e ha' o total do documento principal",
    typeof L?.totalNoDocumentoPrincipal === "number", JSON.stringify(L?.totalNoDocumentoPrincipal));

  console.log("\n4) o shadow FECHADO: declarado como INDETECTAVEL, nunca lido");
  // A primeira versao contava custom elements com filhos -- *um palpite* -- e devolvia 0
  // para uma pagina que TINHA um shadow fechado. *Um palpite que falha e' pior do que uma
  // declaracao de ignorancia: da' uma contagem que parece medida.*
  ok("declara o shadow fechado como nao-detectavel",
    L?.shadowRoots?.fechados === "nao_detectaveis_por_design", JSON.stringify(L?.shadowRoots));
  ok("NAO inventa uma contagem de fechados",
    L?.shadowRoots?.fechadosNaoLidos === undefined, JSON.stringify(L?.shadowRoots));
  ok("declara a limitacao estrutural por escrito",
    typeof L?.limitacaoEstrutural === "string" && L.limitacaoEstrutural.includes("fechados"),
    String(L?.limitacaoEstrutural));
  ok("o conteudo do shadow fechado NAO aparece", !rotulos.includes("botao no shadow fechado"), "leu o shadow fechado!");
  ok("e declara o conteudo do shadow ABERTO (contraprova)", rotulos.includes("botao no shadow"), "nao viu o aberto");

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
console.log("E2E DE FRAMES: medido.");
process.exit(0);
