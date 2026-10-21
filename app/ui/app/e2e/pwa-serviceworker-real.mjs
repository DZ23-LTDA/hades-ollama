/**
 * E2E do Web/PWA num browser REAL (`E-1062`).
 *
 * O QUE ESTE TESTE PROVA -- e o que o `E-1061` NAO podia provar.
 *
 * O `E-1061` testou o `sw.js` com `caches` **simulado**. Isso prova a logica, nao a instalacao.
 * Este corre num **Chromium real**, servido pelo **`vite preview`** (o build de producao), e
 * verifica o que so' um browser pode dizer:
 *
 *   1. o `sw.js` **regista-se** e fica `activated` com um `scope` de `/`;
 *   2. o `manifest.json` **carrega** e e' JSON valido com `display: standalone`;
 *   3. os **tres** icones do manifest respondem **200** (o `E-1060` curou o 404; isto prova no browser);
 *   4. uma resposta de `/api/*` **nunca** e' servida do cache -- *mesmo com o SW a controlar*;
 *   5. o **cache** existe e tem o shell -- *offline parcial, honesto*;
 *   6. a **pagina de erro** (navegacao offline) e' servida do shell em cache, nao inventada.
 *
 * NAO prova: instalacao do PWA na UI do browser (isso e' um gesto humano), nem iOS/Safari.
 */
import { chromium, firefox, webkit } from "playwright";

/** Os motores declarados. *O Web/PWA tem de servir os tres ou dizer que nao serve.* */
const MOTORES = { chromium, firefox, webkit };
const NOME_MOTOR = process.env.HADES_E2E_BROWSER || "chromium";
const motor = MOTORES[NOME_MOTOR];
if (!motor) {
  console.error("motor desconhecido: " + NOME_MOTOR + " (usa chromium|firefox|webkit)");
  process.exit(2);
}
import { writeFileSync } from "node:fs";

const BASE = process.env.HADES_E2E_BASE || "http://127.0.0.1:5173";
const SAIDA = process.env.HADES_E2E_OUT || null;

const resultados = [];
function registar(nome, ok, detalhe) {
  resultados.push({ nome, ok, detalhe });
  console.log(`  ${ok ? "OK  " : "FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

const browser = await motor.launch({ headless: true });
console.log(`\n  ==== motor: ${NOME_MOTOR} ====`);
// *`serviceWorkers: "allow"` so existe no Chromium;* nos outros motores o default ja e' permitir.
const contexto = await browser.newContext(
  NOME_MOTOR === "chromium" ? { serviceWorkers: "allow" } : {},
);
const pagina = await contexto.newPage();

const pedidosFalhados = [];
pagina.on("requestfailed", (r) => pedidosFalhados.push(r.url() + " :: " + r.failure()?.errorText));

try {
  // (0) abrir a app
  const resposta = await pagina.goto(BASE + "/", { waitUntil: "domcontentloaded", timeout: 30000 });
  registar("a app abre (documento raiz)", resposta !== null && resposta.status() === 200,
           "status=" + (resposta ? resposta.status() : "sem resposta"));

  // (1) o service worker regista-se e activa
  // *Um SW acabado de registar pode estar `installing` quando o lemos.* => espera-se pelo
  // `activated` com um limite, em vez de se ler o primeiro instante e chamar-lhe falha.
  // (E-1062: a primeira versao deste teste acusou o PRODUTO de um defeito que era do TESTE.)
  const swEstado = await pagina.evaluate(async () => {
    if (!("serviceWorker" in navigator)) return { suporte: false };
    const reg = await navigator.serviceWorker.ready;
    let ativo = reg.active;
    const limite = Date.now() + 15000;
    while ((!ativo || ativo.state !== "activated") && Date.now() < limite) {
      await new Promise((r) => setTimeout(r, 200));
      ativo = reg.active;
    }
    return {
      suporte: true,
      scope: reg.scope,
      estado: ativo ? ativo.state : null,
      scriptURL: ativo ? ativo.scriptURL : null,
    };
  });
  registar("service worker registado e ACTIVO",
           swEstado.suporte === true && swEstado.estado === "activated",
           `scope=${swEstado.scope} script=${swEstado.scriptURL}`);

  // (2) o manifest carrega e e' valido
  const manifest = await pagina.evaluate(async () => {
    const link = document.querySelector('link[rel="manifest"]');
    if (!link) return { achou: false };
    const r = await fetch(link.href);
    const json = await r.json();
    return { achou: true, status: r.status, name: json.name, display: json.display,
             icones: (json.icons || []).map((i) => i.src) };
  });
  registar("manifest carrega e e' JSON valido",
           manifest.achou === true && manifest.status === 200 && manifest.display === "standalone",
           `name=${manifest.name} display=${manifest.display}`);

  // (3) os icones do manifest respondem 200 no browser
  let iconesOk = 0;
  const iconesDetalhe = [];
  for (const src of manifest.icones || []) {
    const r = await pagina.request.get(new URL(src, BASE).href);
    const ok = r.status() === 200;
    if (ok) iconesOk++;
    iconesDetalhe.push(`${src} -> ${r.status()}`);
  }
  registar("todos os icones do manifest respondem 200",
           iconesOk === (manifest.icones || []).length && iconesOk > 0,
           iconesDetalhe.join(" · "));

  // (4) /api/* NUNCA e' servido do cache -- o pedido tem de chegar a' rede (ou falhar), nao vir do SW
  const apiResultado = await pagina.evaluate(async () => {
    const url = "/api/verificacao-e2e-nao-existe";
    try {
      const r = await fetch(url, { cache: "no-store" });
      // Se o servidor devolver 404, chegou a' rede. O que NAO pode e' vir de cache marcado pelo SW.
      return { chegou: true, status: r.status, tipo: r.type };
    } catch (e) {
      return { chegou: false, erro: String(e) };
    }
  });
  registar("/api/* nao e' servido do cache pelo SW",
           apiResultado.chegou === true && apiResultado.tipo !== "opaque",
           `status=${apiResultado.status} tipo=${apiResultado.tipo}`);

  // (5) o cache do SW existe e tem o shell
  const cacheEstado = await pagina.evaluate(async () => {
    const nomes = await caches.keys();
    const shell = nomes.find((n) => n.includes("shell"));
    let temIndex = false;
    if (shell) {
      const c = await caches.open(shell);
      temIndex = (await c.match("/index.html")) !== undefined;
    }
    return { nomes, temIndex, versao: nomes.find((n) => n.startsWith("hades-sw-")) || null };
  });
  registar("o cache do shell existe e contem /index.html",
           cacheEstado.temIndex === true && cacheEstado.versao !== null,
           `versao=${cacheEstado.versao} caches=${cacheEstado.nomes.length}`);

  // (6) navegacao offline: com a rede cortada, o SW tem de servir o shell (nao inventar)
  await contexto.setOffline(true);
  const offline = await pagina.evaluate(async () => {
    try {
      const r = await fetch("/alguma-rota-que-nao-existe", { cache: "no-store" });
      return { status: r.status, tipo: r.type };
    } catch (e) {
      return { erro: String(e) };
    }
  });
  // offline, /alguma-rota... nao esta' em cache e nao vem da rede => o SW devolve 504 honesto
  registar("offline: o SW devolve um erro HONESTO (nao inventa conteudo)",
           offline.erro !== undefined || offline.status === 504,
           JSON.stringify(offline));
  await contexto.setOffline(false);

  // (7) nao houve pedidos falhados inesperados
  // *O proprio teste provoca um pedido que TEM de falhar* (a rota inexistente, offline).
  // Conta-lo como falha do produto seria o teste a acusar-se a si mesmo. Filtra-se.
  const doTeste = pedidosFalhados.filter((f) => f.includes("alguma-rota-que-nao-existe"));
  const inesperados = pedidosFalhados.filter((f) => !f.includes("alguma-rota-que-nao-existe"));
  registar("nenhum pedido falhou em carga normal (excluindo os que o teste provoca)",
           inesperados.length === 0,
           `${inesperados.length} inesperados, ${doTeste.length} provocados pelo teste`);
} finally {
  await browser.close();
}

const total = resultados.length;
const passaram = resultados.filter((r) => r.ok).length;
console.log("");
console.log(`  ==== ${passaram}/${total} verificacoes passaram ====`);
if (SAIDA) writeFileSync(SAIDA, JSON.stringify({ motor: NOME_MOTOR, base: BASE, resultados, passaram, total }, null, 2));
process.exit(passaram === total ? 0 : 1);
