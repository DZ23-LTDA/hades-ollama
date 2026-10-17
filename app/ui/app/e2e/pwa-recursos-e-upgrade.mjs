/**
 * Mede os recursos do service worker e testa o UPGRADE entre versoes (`E-1064`).
 *
 * O QUE O `E-1061`/`E-1062`/`E-1063` NAO MEDIRAM.
 *
 * - `E-1061` provou a logica com `caches` simulado.
 * - `E-1062` provou a instalacao num Chromium real.
 * - `E-1063` provou os tres motores.
 *
 * Falta: **quanto pesa** e **o que acontece quando o SW muda de versao**. Um service worker
 * que nunca e' substituido e' um service worker que congela o produto no dia em que e' instalado.
 *
 * COMO O UPGRADE E' TESTADO SEM INVENTAR NADA.
 *
 * O `sw.js` deriva o nome do cache de `const VERSAO`. Este teste serve o MESMO `dist/` por um
 * servidor local que, no caminho `/sw.js`, **reescreve a constante** para `hades-sw-v2`. Nada
 * mais muda -- o corpo do SW e' o mesmo. Assim:
 *
 *   1. carrega v1, regista, confirma `hades-sw-v1-*` no `caches`;
 *   2. recarrega com o servidor a servir v2, o browser detecta o SW novo, e o `activate` da v2
 *      apaga a v1 -- porque o codigo de limpeza ja' existia e foi ele que o `E-1061` testou em
 *      simulacao. **Agora e' testado num browser.**
 *   3. confirma que `hades-sw-v1-*` **desapareceu** e `hades-sw-v2-*` existe.
 *
 * NAO prova: instalacao pela UI, nem Safari real, nem Android/iOS.
 */
import { chromium, firefox, webkit } from "playwright";

/** Os tres motores. *O upgrade e os recursos tem de ser medidos em cada um, ou nao sao medidos.* */
const MOTORES = { chromium, firefox, webkit };
const NOME_MOTOR = process.env.HADES_E2E_BROWSER || "chromium";
const motor = MOTORES[NOME_MOTOR];
if (!motor) {
  console.error("motor desconhecido: " + NOME_MOTOR + " (usa chromium|firefox|webkit)");
  process.exit(2);
}
import { createServer } from "node:http";
import { readFile, stat } from "node:fs/promises";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = fileURLToPath(new URL(".", import.meta.url));
const DIST = join(AQUI, "..", "dist");

const TIPOS = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".mjs": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".webmanifest": "application/manifest+json",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".woff2": "font/woff2",
};

let VERSAO_ATUAL = "hades-sw-v1";
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
      // *A unica coisa que muda entre versoes e' a constante.* O corpo e' o mesmo.
      corpo = Buffer.from(
        corpo.toString("utf8").replace(/const VERSAO = "[^"]+";/, `const VERSAO = "${VERSAO_ATUAL}";`),
        "utf8",
      );
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

await new Promise((r) => servidor.listen(5180, "127.0.0.1", r));
const BASE = "http://127.0.0.1:5180";
console.log(`\n  servidor de teste em ${BASE}  (dist = ${DIST})\n`);

const resultados = [];
function registar(nome, ok, detalhe) {
  resultados.push({ nome, ok, detalhe });
  console.log(`  ${ok ? "OK  " : "FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

console.log(`\n  ==== motor: ${NOME_MOTOR} ====`);
const browser = await motor.launch({ headless: true });
// *`serviceWorkers: "allow"` so existe no Chromium;* nos outros motores o default ja permite.
const contexto = await browser.newContext(
  NOME_MOTOR === "chromium" ? { serviceWorkers: "allow" } : {},
);

async function medirCacheESW() {
  const pagina = await contexto.newPage();
  await pagina.goto(BASE + "/", { waitUntil: "load" });
  const dados = await pagina.evaluate(async () => {
    const reg = await navigator.serviceWorker.ready;
    let ativo = reg.active;
    const limite = Date.now() + 15000;
    while ((!ativo || ativo.state !== "activated") && Date.now() < limite) {
      await new Promise((r) => setTimeout(r, 150));
      ativo = reg.active;
    }
    // conteúdo do cache: quantas entradas e quantos bytes (via Response.arrayBuffer)
    const nomes = await caches.keys();
    const detalhe = [];
    let totalBytes = 0;
    let totalEntradas = 0;
    for (const nome of nomes) {
      const c = await caches.open(nome);
      const chaves = await c.keys();
      let bytes = 0;
      for (const k of chaves) {
        const r = await c.match(k);
        if (r) bytes += (await r.arrayBuffer()).byteLength;
      }
      detalhe.push({ nome, entradas: chaves.length, bytes });
      totalBytes += bytes;
      totalEntradas += chaves.length;
    }
    return {
      estado: ativo ? ativo.state : null,
      scriptURL: ativo ? ativo.scriptURL : null,
      caches: detalhe,
      totalBytes,
      totalEntradas,
      memoriaJS: performance.memory
        ? {
            usedJSHeapSize: performance.memory.usedJSHeapSize,
            totalJSHeapSize: performance.memory.totalJSHeapSize,
          }
        : null,
    };
  });
  await pagina.close();
  return dados;
}

try {
  // ================= (1) v1: instalacao e MEDICAO DE RECURSOS =================
  const t0 = Date.now();
  const v1 = await medirCacheESW();
  const tempoAteAtivo = Date.now() - t0;

  registar("v1: service worker activo", v1.estado === "activated", "estado=" + v1.estado);
  registar("v1: caches com prefixo hades-sw-v1",
           v1.caches.length > 0 && v1.caches.every((c) => c.nome.startsWith("hades-sw-v1")),
           v1.caches.map((c) => `${c.nome}(${c.entradas})`).join(" · "));

  // orcamento declarado AQUI, antes de medir -- *um limite que se ajusta ao resultado nao e' um limite*
  const ORCAMENTO_ENTRADAS = 400;
  const ORCAMENTO_BYTES = 64 * 1024 * 1024; // 64 MB
  registar(`v1: entradas no cache <= ${ORCAMENTO_ENTRADAS}`,
           v1.totalEntradas <= ORCAMENTO_ENTRADAS,
           `${v1.totalEntradas} entradas, ${(v1.totalBytes / 1024).toFixed(1)} KB`);
  registar(`v1: bytes no cache <= ${(ORCAMENTO_BYTES / 1024 / 1024).toFixed(0)} MB`,
           v1.totalBytes <= ORCAMENTO_BYTES,
           `${(v1.totalBytes / 1024 / 1024).toFixed(2)} MB`);
  registar("v1: fica activo em menos de 20 s",
           tempoAteAtivo < 20000,
           `${tempoAteAtivo} ms (inclui abrir a pagina)`);

  // ================= (2) UPGRADE: v1 -> v2 =================
  VERSAO_ATUAL = "hades-sw-v2";
  const pagina2 = await contexto.newPage();
  await pagina2.goto(BASE + "/", { waitUntil: "load" });
  // forcar a verificacao de actualizacao do SW
  const upgrade = await pagina2.evaluate(async () => {
    const reg = await navigator.serviceWorker.getRegistration();
    if (!reg) return { ok: false, motivo: "sem registo" };
    await reg.update();
    // esperar que a v2 assuma
    const limite = Date.now() + 20000;
    while (Date.now() < limite) {
      await new Promise((r) => setTimeout(r, 250));
      const ativo = reg.active;
      const nomes = await caches.keys();
      if (ativo && ativo.state === "activated" && nomes.some((n) => n.startsWith("hades-sw-v2"))) {
        return { ok: true, estado: ativo.state, caches: nomes };
      }
    }
    const ativo = reg.active;
    return { ok: false, estado: ativo ? ativo.state : null, caches: await caches.keys() };
  });
  await pagina2.close();

  registar("upgrade: o browser aceitou a v2 e activou-a", upgrade.ok === true,
           `estado=${upgrade.estado} caches=${(upgrade.caches || []).join(",")}`);

  // ================= (3) a v1 foi APAGADA pelo activate da v2 =================
  const v2 = await medirCacheESW();
  const temV1 = v2.caches.some((c) => c.nome.startsWith("hades-sw-v1"));
  const temV2 = v2.caches.some((c) => c.nome.startsWith("hades-sw-v2"));
  registar("upgrade: a v1 foi APAGADA (o activate limpou)", temV1 === false,
           temV1 ? "*** hades-sw-v1-* ainda existe ***" : "nenhum cache hades-sw-v1-*");
  registar("upgrade: a v2 esta' no cache", temV2 === true,
           v2.caches.map((c) => c.nome).join(" · "));

  // ================= (4) o SW continua a funcionar depois do upgrade =================
  const pagina3 = await contexto.newPage();
  await pagina3.goto(BASE + "/", { waitUntil: "load" });
  const posUpgrade = await pagina3.evaluate(async () => {
    const r = await fetch("/api/pos-upgrade", { cache: "no-store" }).catch((e) => ({ erro: String(e) }));
    return { status: r.status, erro: r.erro };
  });
  await pagina3.close();
  registar("upgrade: /api/* continua a ir a' rede (nao vem do cache)",
           posUpgrade.status === 404 || posUpgrade.erro !== undefined,
           JSON.stringify(posUpgrade));
} finally {
  await browser.close();
  servidor.close();
}

const total = resultados.length;
const passaram = resultados.filter((r) => r.ok).length;
console.log("");
console.log(`  ==== ${passaram}/${total} verificacoes passaram ====`);
process.exit(passaram === total ? 0 : 1);
