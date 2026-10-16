/**
 * E2E DO `pushState` (`E-1037`). A divida declarada no `E-1035` -- e a medicao.
 *
 * No `E-1035` registei: *"`history.pushState` nao dispara `tabs.onUpdated` com URL novo,
 * logo um SPA que mude de origem por History API NAO seria revogado. Nao foi testado, e pode
 * ser um buraco real."*
 *
 * `uma divida declarada e' uma medicao a fazer, nao uma ressalva.` Aqui mede-se.
 *
 * A hipotese adversarial: a pagina muda de origem por `pushState` e a concessao sobrevive.
 * O que se mede:
 *   1. `pushState` para OUTRA origem -> o BROWSER recusa (`SecurityError`). Nao e' politica
 *      do HADES: e' a same-origin policy. *Nao ha buraco onde o browser nao deixa passar.*
 *   2. `pushState` para outro CAMINHO na mesma origem -> a concessao MANTEM-SE, e isso esta'
 *      CORRETO: a origem nao mudou. *A unidade da politica e' a origem, nao o caminho.*
 *   3. `replaceState` -> mesmo comportamento.
 *   4. E o adversario a serio: navegacao REAL para outra origem (o que um SPA faz quando
 *      troca de host) -> a concessao CAI. Ja' provado no `E-1035`, reconfirmado aqui para
 *      nao deixar a divida em aberto por falta de contraprova no mesmo ficheiro.
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

const pagina = (nome, origem) => `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>HADES: ${nome}</title></head>
<body><h1>${nome}</h1><input id="campo" type="text" autocomplete="off">
<script>
  window.__tentarPush = (url) => {
    try { history.pushState({}, "", url); return { ok: true, href: location.href, origem: location.origin }; }
    catch (e) { return { ok: false, erro: e.name, mensagem: String(e.message).slice(0, 90) }; }
  };
  window.__tentarReplace = (url) => {
    try { history.replaceState({}, "", url); return { ok: true, href: location.href, origem: location.origin }; }
    catch (e) { return { ok: false, erro: e.name, mensagem: String(e.message).slice(0, 90) }; }
  };
</script></body></html>`;

function subir(nome) {
  const s = http.createServer((req, res) => {
    res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    res.end(pagina(nome, "http://127.0.0.1:" + s.address().port));
  });
  return new Promise((r) => s.listen(0, "127.0.0.1", () => r({ s, base: "http://127.0.0.1:" + s.address().port + "/" })));
}

const A = await subir("origem-A");
const B = await subir("origem-B");
const origemA = new URL(A.base).origin;
const origemB = new URL(B.base).origin;
console.log("0) duas origens: A=" + origemA + "  B=" + origemB);
ok("sao origens diferentes", origemA !== origemB);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-push-"));
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
  await aba.waitForTimeout(1200);
  const tabId = await sw.evaluate(async () => {
    const abas = await chrome.tabs.query({});
    const alvo = abas.find((a) => String(a.url).startsWith("http://127.0.0.1"));
    return alvo ? alvo.id : null;
  });
  const c = await pedir({ tipo: "conectar", tabId });
  console.log("   concessao: " + JSON.stringify(c));
  ok("aba concedida em A", c?.ok === true && c?.origem === origemA, JSON.stringify(c));

  // 1) pushState para OUTRA origem: o BROWSER recusa.
  console.log("1) `pushState` para OUTRA origem -- quem decide e' o browser, nao o HADES");
  const p1 = await aba.evaluate((u) => window.__tentarPush(u), origemB + "/invasao");
  console.log("   " + JSON.stringify(p1));
  ok("o browser RECUSA (`SecurityError`)", p1?.ok === false && p1?.erro === "SecurityError", JSON.stringify(p1));
  const origemAinda = await aba.evaluate(() => location.origin);
  ok("a origem NAO mudou", origemAinda === origemA, String(origemAinda));

  // 2) pushState para outro CAMINHO na mesma origem: mantem-se, e esta' correto.
  console.log("2) `pushState` para outro CAMINHO na mesma origem -- a concessao MANTEM-SE");
  const p2 = await aba.evaluate((u) => window.__tentarPush(u), origemA + "/outro/caminho?x=1#frag");
  console.log("   " + JSON.stringify(p2));
  ok("o pushState de mesmo origem e' ACEITE", p2?.ok === true, JSON.stringify(p2));
  ok("a origem continua A", p2?.origem === origemA, String(p2?.origem));

  const est1 = await pedir({ tipo: "estado" });
  const ids1 = (est1?.concessoes ?? []).map((x) => x.tabId);
  ok("a concessao MANTEM-SE (mesma origem)", ids1.includes(tabId), JSON.stringify(ids1));
  const b1 = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  ok("badge ON intacto", b1 === "ON", String(b1));
  const a1 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId, url: origemA + "/outro/caminho" });
  ok("e a accao na mesma origem continua PERMITIDA", a1?.ok === true, JSON.stringify(a1));

  // 3) replaceState: mesma regra.
  console.log("3) `replaceState` -- mesma regra");
  const r1 = await aba.evaluate((u) => window.__tentarReplace(u), origemB + "/invasao");
  ok("replaceState para outra origem TAMBEM e' recusado", r1?.ok === false, JSON.stringify(r1));
  const r2 = await aba.evaluate((u) => window.__tentarReplace(u), origemA + "/replace");
  ok("replaceState de mesma origem e' aceite", r2?.ok === true, JSON.stringify(r2));
  const est2 = await pedir({ tipo: "estado" });
  ok("concessao continua viva", (est2?.concessoes ?? []).map((x) => x.tabId).includes(tabId), JSON.stringify(est2?.concessoes));
  void pagina;

  // 3-B) `E-1037`: O RELOAD. O browser emite `onUpdated` com SO' `status` e SEM `url`.
  //      O codigo antigo validava a string VAZIA, falhava, e revogava. *A navegacao mais
  //      comum que existe matava a sessao.*
  console.log("3-B) o RELOAD -- o caso que o `E-1035` nao via, porque ali a mudanca era real");
  await aba.reload();
  await aba.waitForTimeout(1800);
  const estRl = await pedir({ tipo: "estado" });
  const idsRl = (estRl?.concessoes ?? []).map((x) => x.tabId);
  const bRl = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  console.log("   concessoes: " + JSON.stringify(idsRl) + "  badge=" + JSON.stringify(bRl));
  ok("a concessao SOBREVIVE ao reload", idsRl.includes(tabId), JSON.stringify(idsRl));
  ok("o badge continua ON", bRl === "ON", JSON.stringify(bRl));
  const aRl = await pedir({ tipo: "accao", scope: "browser:navigate", tabId, url: origemA + "/replace" });
  ok("e a accao continua PERMITIDA", aRl?.ok === true, JSON.stringify(aRl));

  // 3-C) e a mudanca de TITULO, que tambem emite `onUpdated` sem URL.
  console.log("3-C) uma mudanca de TITULO -- tambem emite `onUpdated` sem URL");
  await aba.evaluate(() => { document.title = "titulo novo " + Date.now(); });
  await aba.waitForTimeout(1500);
  const estT = await pedir({ tipo: "estado" });
  ok("a concessao sobrevive a mudanca de titulo", (estT?.concessoes ?? []).map((x) => x.tabId).includes(tabId), JSON.stringify(estT?.concessoes));

  // 4) O ADVERSARIO A SERIO: navegacao REAL para outra origem.
  console.log("4) o adversario a serio: navegacao REAL para outra origem");
  await aba.goto(B.base + "?real=1");
  await aba.waitForTimeout(1800);
  const est3 = await pedir({ tipo: "estado" });
  const ids3 = (est3?.concessoes ?? []).map((x) => x.tabId);
  const b3 = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), tabId);
  console.log("   concessoes: " + JSON.stringify(ids3) + "  badge=" + JSON.stringify(b3));
  ok("a concessao CAIU na navegacao real", !ids3.includes(tabId), JSON.stringify(ids3));
  ok("o badge limpou", !b3, JSON.stringify(b3));
  const a3 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId, url: origemB + "/x" });
  ok("e a accao e' recusada", a3?.ok === false && a3?.motivo === "aba_nao_concedida", JSON.stringify(a3));

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
console.log("E2E DE PUSHSTATE: mudar de origem por History API e' IMPOSSIVEL -- quem o proibe e' o browser.");
process.exit(0);
