/**
 * Onde e' que o Firefox gasta o +1,2 s na activacao do SW? (`E-1066`)
 *
 * O `E-1065` mediu: Chromium **423 ms**, WebKit **1041 ms**, Firefox **1646 ms** -- e nao
 * explicou a diferenca. *Um numero nao e' uma explicacao.*
 *
 * HIPOTESE DECLARADA ANTES DE MEDIR (obrigatoria pelo protocolo antiloop):
 *
 *   H1: o custo esta' no `install` -- o `addAll` do shell + os 6 icones. O Firefox pode ser mais
 *       lento a descarregar 6 imagens em paralelo que o Chromium.
 *   H2: o custo esta' no `activate` -- o `caches.keys()` + `delete` + `clients.claim()`.
 *   H3: o custo esta' FORA do `sw.js` -- no proprio browser a registar/arrancar o worker.
 *
 * COMO E' DISCRIMINADO, SEM TOCAR NO PRODUTO.
 *
 * O servidor deste teste serve o `dist/`, mas no `/sw.js` **injeta** uma versao instrumentada:
 * o mesmo corpo + um `postMessage` com `performance.now()` no inicio do `install`, no fim do
 * `install`, no inicio do `activate` e no fim do `activate`.
 *
 * Isso e' uma **variante de teste**, nao uma alteracao ao `sw.js` real. O `sw.js` do produto
 * continua sem telemetria -- *um service worker de producao nao leva cronometros dentro.*
 *
 * A medicao de referencia (`t_registo` -> `activated`) vem do lado da pagina, com
 * `performance.now()`, e e' o mesmo numero que o `E-1065` mediu.
 */
import { chromium, firefox, webkit } from "playwright";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = fileURLToPath(new URL(".", import.meta.url));
const DIST = join(AQUI, "..", "dist");

const MOTORES = { chromium, firefox, webkit };
const NOME_MOTOR = process.env.HADES_E2E_BROWSER || "chromium";
const motor = MOTORES[NOME_MOTOR];
if (!motor) {
  console.error("motor desconhecido: " + NOME_MOTOR);
  process.exit(2);
}

const TIPOS = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".woff2": "font/woff2",
};

/** Instrumenta o sw.js SEM tocar no ficheiro do produto. */
function instrumentar(corpo, porta) {
  // marcos no install
  let s = corpo.replace(
    'self.addEventListener("install", (evento) => {',
    'self.addEventListener("install", (evento) => {\n  const tInstallInicio = performance.now();',
  );
  s = s.replace(
    "      await self.skipWaiting();",
    '      self.__marcos = { installInicio: tInstallInicio, installFim: performance.now() };\n      await self.skipWaiting();',
  );
  // marcos no activate
  s = s.replace(
    'self.addEventListener("activate", (evento) => {',
    'self.addEventListener("activate", (evento) => {\n  const tActivateInicio = performance.now();\n  const marcos = self.__marcos || {};\n  marcos.activateInicio = tActivateInicio;',
  );
  s = s.replace(
    "      await self.clients.claim();",
    "      marcos.activateFim = performance.now();\n      // reporta os marcos a todas as paginas controladas\n      self.clients.matchAll({ includeUncontrolled: true }).then((cs) => {\n        for (const c of cs) c.postMessage({ tipo: \"hades-marcos-sw\", marcos });\n      });\n      await self.clients.claim();",
  );
  return s;
}

let corpoSw = null;
const servidor = createServer(async (req, res) => {
  try {
    let caminho = decodeURIComponent(new URL(req.url, "http://x").pathname);
    if (caminho === "/") caminho = "/index.html";
    const alvo = normalize(join(DIST, caminho));
    if (!alvo.startsWith(DIST)) {
      res.writeHead(403).end();
      return;
    }
    let corpo = await readFile(alvo);
    if (caminho === "/sw.js") {
      if (!corpoSw) corpoSw = instrumentar(corpo.toString("utf8"));
      corpo = Buffer.from(corpoSw, "utf8");
    }
    res.writeHead(200, {
      "content-type": TIPOS[extname(alvo)] || "application/octet-stream",
      "cache-control": caminho === "/sw.js" ? "no-store" : "no-cache",
    });
    res.end(corpo);
  } catch {
    res.writeHead(404).end();
  }
});
await new Promise((r) => servidor.listen(5181, "127.0.0.1", r));
const BASE = "http://127.0.0.1:5181";
console.log(`\n  ==== motor: ${NOME_MOTOR}   servidor: ${BASE} ====\n`);

const browser = await motor.launch({ headless: true });
const contexto = await browser.newContext(NOME_MOTOR === "chromium" ? { serviceWorkers: "allow" } : {});
const pagina = await contexto.newPage();

// recolhe os marcos que o SW instrumentado envia
const marcosSW = [];
await pagina.exposeFunction("__receberMarcos", (m) => marcosSW.push(m));
await pagina.addInitScript(() => {
  navigator.serviceWorker?.addEventListener("message", (e) => {
    if (e.data && e.data.tipo === "hades-marcos-sw") window.__receberMarcos(e.data.marcos);
  });
});

const t0 = Date.now();
await pagina.goto(BASE + "/", { waitUntil: "load" });
const dados = await pagina.evaluate(async () => {
  const inicio = performance.now();
  const reg = await navigator.serviceWorker.ready;
  let ativo = reg.active;
  const limite = performance.now() + 20000;
  while ((!ativo || ativo.state !== "activated") && performance.now() < limite) {
    await new Promise((r) => setTimeout(r, 100));
    ativo = reg.active;
  }
  return { tRegistoAoActivo: performance.now() - inicio, estado: ativo ? ativo.state : null };
});
const totalMs = Date.now() - t0;
await pagina.waitForTimeout(1200); // deixa chegar o postMessage

const m = marcosSW.length > 0 ? marcosSW[marcosSW.length - 1] : null;
console.log(`  estado do SW: ${dados.estado}`);
console.log(`  t_registo -> activated (lado da pagina): ${dados.tRegistoAoActivo.toFixed(1)} ms`);
console.log(`  t_navegacao total:                      ${totalMs} ms`);
console.log("");
if (m) {
  const install = m.installFim - m.installInicio;
  const activate = m.activateFim - m.activateInicio;
  const entre = m.activateInicio - m.installFim;
  console.log(`  install   (addAll shell + 6 icones):    ${install.toFixed(1)} ms`);
  console.log(`  entre install-fim e activate-inicio:    ${entre.toFixed(1)} ms`);
  console.log(`  activate  (keys + delete + claim):      ${activate.toFixed(1)} ms`);
  console.log(`  ------------------------------------------------`);
  console.log(`  soma dentro do sw.js:                   ${(install + entre + activate).toFixed(1)} ms`);
  console.log("");
  console.log(`  *** ONDE ESTA' O TEMPO ***`);
  console.log(`  install=${install.toFixed(0)} ms | activate=${activate.toFixed(0)} ms | fora(entre)=${entre.toFixed(0)} ms | total_pagina=${dados.tRegistoAoActivo.toFixed(0)} ms`);
  if (install > activate * 2) {
    console.log(`  => H1 ACEITE: o custo esta' no install (addAll do shell + icones)`);
  } else if (activate > install * 2) {
    console.log(`  => H2 ACEITE: o custo esta' no activate (keys + delete + claim)`);
  } else {
    console.log(`  => H3: o custo esta' FORA do sw.js -- no browser a registar/arrancar o worker`);
  }
} else {
  console.log("  (o SW instrumentado nao enviou marcos -- ver H3)");
}
await browser.close();
servidor.close();
