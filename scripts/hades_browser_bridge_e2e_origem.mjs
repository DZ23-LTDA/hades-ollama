/**
 * E2E DA REVOGACAO POR MUDANCA DE ORIGEM (`E-1035`).
 *
 * O `tabs.onUpdated` tem o codigo que revoga quando a aba navega para outra origem --
 * e **nunca teve E2E**. Codigo de seguranca sem teste e' uma promessa.
 *
 * Duas portas diferentes sao duas ORIGENS diferentes (a porta faz parte da origem), logo
 * servem para testar a fronteira sem sair de loopback.
 *
 * O que discrimina:
 *   - aba A concedida em `:P1` -> badge ON, accao permitida;
 *   - aba B na MESMA origem `:P1` -> concedida tambem, e as duas coexistem;
 *   - a aba A NAVEGA para `:P2` -> a concessao e' REVOGADA, o badge limpa, a accao e'
 *     recusada -- e a concessao da aba B **continua intacta**;
 *   - voltar a `:P1` NAO restaura nada: exige nova concessao do utilizador.
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

const pagina = (nome) => `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>HADES: ${nome}</title></head>
<body><h1>${nome}</h1><input id="campo" type="text" autocomplete="off"></body></html>`;

function subir(nome) {
  const s = http.createServer((req, res) => {
    res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    res.end(pagina(nome));
  });
  return new Promise((r) => s.listen(0, "127.0.0.1", () => r({ s, base: "http://127.0.0.1:" + s.address().port + "/" })));
}

const A = await subir("origem-A");
const B = await subir("origem-B");
console.log("0) duas origens locais: A=" + A.base + "  B=" + B.base);
ok("as duas origens sao diferentes", new URL(A.base).origin !== new URL(B.base).origin);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-orig-"));
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

  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(600);
  const pedir = (m) => popup.evaluate(async (mm) => chrome.runtime.sendMessage(mm), m);
  await pedir({ tipo: "optin:definir", ligado: true });

  const idDaAba = (fragmento) => sw.evaluate(async (f) => {
    const abas = await chrome.tabs.query({});
    const alvo = abas.find((a) => String(a.url).includes(f));
    return alvo ? alvo.id : null;
  }, fragmento);

  // 1) Duas abas na MESMA origem A: as duas concedidas.
  console.log("1) duas abas na mesma origem A, as duas concedidas");
  const abas = await ctx.newPage();
  const abaA = await ctx.newPage();
  await abas.goto(A.base + "?aba=1");
  await abaA.goto(A.base + "?aba=2");
  await abas.waitForTimeout(1200);
  await abaA.waitForTimeout(1200);

  const id1 = await idDaAba("aba=1");
  const id2 = await idDaAba("aba=2");
  const c1 = await pedir({ tipo: "conectar", tabId: id1 });
  const c2 = await pedir({ tipo: "conectar", tabId: id2 });
  console.log("   aba1: " + JSON.stringify(c1) + "\n   aba2: " + JSON.stringify(c2));
  ok("aba 1 concedida", c1?.ok === true, JSON.stringify(c1));
  ok("aba 2 concedida", c2?.ok === true, JSON.stringify(c2));
  ok("as duas partilham a MESMA origem", c1?.origem === c2?.origem, `${c1?.origem} vs ${c2?.origem}`);

  const b1 = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), id1);
  const b2 = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), id2);
  ok("badge ON na aba 1", b1 === "ON", String(b1));
  ok("badge ON na aba 2", b2 === "ON", String(b2));

  // 2) A aba 1 NAVEGA para outra origem (B). A concessao TEM de cair.
  console.log("2) a aba 1 navega para a origem B");
  await abas.bringToFront();
  await abas.goto(B.base + "?aba=1");
  await abas.waitForTimeout(2000);

  const est = await pedir({ tipo: "estado" });
  const idsConcedidos = (est?.concessoes ?? []).map((c) => c.tabId);
  console.log("   concessoes agora: " + JSON.stringify(idsConcedidos));
  ok("a concessao da aba 1 foi REVOGADA", !idsConcedidos.includes(id1), JSON.stringify(idsConcedidos));
  ok("a concessao da aba 2 CONTINUA", idsConcedidos.includes(id2), JSON.stringify(idsConcedidos));

  const b1depois = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), id1);
  const b2depois = await sw.evaluate(async (t) => chrome.action.getBadgeText({ tabId: t }), id2);
  console.log("   badges: aba1=" + JSON.stringify(b1depois) + "  aba2=" + JSON.stringify(b2depois));
  ok("badge da aba 1 LIMPO", !b1depois, JSON.stringify(b1depois));
  ok("badge da aba 2 intacto", b2depois === "ON", JSON.stringify(b2depois));

  // 3) A accao na aba 1 e' RECUSADA; na aba 2 continua permitida.
  console.log("3) a accao depende da concessao, por aba");
  const a1 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId: id1, url: B.base });
  const a2 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId: id2, url: A.base });
  console.log("   aba1: " + JSON.stringify(a1) + "\n   aba2: " + JSON.stringify(a2));
  ok("aba 1 recusada (concessao caiu)", a1?.ok === false && a1?.motivo === "aba_nao_concedida", JSON.stringify(a1));
  ok("aba 2 permitida (concessao vive)", a2?.ok === true, JSON.stringify(a2));

  // 4) VOLTAR a origem original NAO restaura nada.
  console.log("4) voltar a origem A NAO restaura a concessao");
  await abas.goto(A.base + "?aba=1");
  await abas.waitForTimeout(2000);
  const est2 = await pedir({ tipo: "estado" });
  const ids2 = (est2?.concessoes ?? []).map((c) => c.tabId);
  console.log("   concessoes: " + JSON.stringify(ids2));
  ok("a aba 1 continua SEM concessao", !ids2.includes(id1), JSON.stringify(ids2));
  ok("a aba 2 continua concedida", ids2.includes(id2), JSON.stringify(ids2));
  const a3 = await pedir({ tipo: "accao", scope: "browser:navigate", tabId: id1, url: A.base });
  ok("accao na aba 1 segue recusada", a3?.ok === false && a3?.motivo === "aba_nao_concedida", JSON.stringify(a3));

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
console.log("E2E DE ORIGEM: navegar para outra origem revoga; a outra aba, na mesma origem, continua concedida.");
process.exit(0);
