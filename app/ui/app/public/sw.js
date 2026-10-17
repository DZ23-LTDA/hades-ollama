/**
 * HADES Service Worker (`E-1061`).
 *
 * ESCOPO DELIBERADO, E O QUE ELE **NAO** FAZ.
 *
 * O §FASE-6 pede «service worker/cache quando existir, offline honesto». Isto entrega o
 * service worker com o escopo **minimo e honesto**: o **shell** e os **assets estaticos**
 * ficam em cache; **as respostas de API nunca**.
 *
 * POR QUE A API FICA FORA -- e nao e' timidez:
 *
 *   - o HADES fala com um **servidor local** (`/api/*` -> 127.0.0.1). Servir `/api/chat` do
 *     cache devolveria uma resposta **de outro momento** como se fosse agora;
 *   - as **aprovacoes** sao a superficie mais sensivel do produto: uma aprovacao vinda do cache
 *     poderia autorizar uma accao que ja' nao esta' pendente. *Um cache que responde por uma
 *     aprovacao e' um aprovador que ninguem autorizou.*;
 *   - um `POST`/`PUT`/`DELETE` servido do cache nao e' offline: e' uma accao que **nao aconteceu**
 *     e que o utilizador julga ter acontecido.
 *
 * => **`/api/*` e' `network-only`. Sem excepcao.** Offline, o HADES mostra o shell e diz que
 *    esta' offline -- *nao inventa um servidor.*
 *
 * Estrategias, por tipo:
 *   - `/api/*`                    -> **network-only** (nunca toca no cache)
 *   - metodo != GET               -> **network-only**
 *   - navegacao (documento HTML)   -> network-first, fallback ao shell em cache, e se nao houver
 *                                    shell, a pagina offline estatica
 *   - assets com hash `/assets/*`  -> cache-first (imutaveis: o nome muda com o conteudo)
 *   - icones/manifest              -> cache-first
 *   - o resto                      -> network-first com fallback ao cache
 *
 * Sem `eval`, sem `importScripts` remoto, sem dependencia carregada da internet.
 */

const VERSAO = "hades-sw-v1";
const CACHE_SHELL = `${VERSAO}-shell`;
const CACHE_ASSETS = `${VERSAO}-assets`;

/** O minimo para o shell abrir offline. *Se um destes falhar, o install falha -- e e' propositado.* */
const SHELL = ["/", "/index.html", "/manifest.json"];

/** Icones: o PWA precisa deles offline para a instalacao nao partir. */
const ICONES = [
  "/hades-icon-192.png",
  "/hades-icon-512.png",
  "/hades-maskable-512.png",
  "/favicon.ico",
  "/favicon.svg",
  "/apple-touch-icon.png",
];

/** A pagina de fallback offline. Tem de ser estatica e sem dados -- *nada de estado real*. */
const OFFLINE_HTML = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>HADES — offline</title>
<style>
  :root { color-scheme: dark light; }
  body { margin: 0; min-height: 100vh; display: grid; place-items: center;
         font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
         background: #0B0F17; color: #E6EAF2; }
  main { max-width: 34rem; padding: 2rem; text-align: center; }
  h1 { font-size: 1.35rem; margin: 0 0 .6rem; }
  p { line-height: 1.5; margin: .5rem 0; color: #A9B4C7; }
  button { margin-top: 1.2rem; padding: .6rem 1.1rem; border-radius: .5rem;
           border: 1px solid #2A3446; background: #141B26; color: #E6EAF2;
           font-size: .95rem; cursor: pointer; }
  button:hover { background: #1B2432; }
  .nota { margin-top: 1.4rem; font-size: .8rem; color: #6E7A8F; }
</style>
</head>
<body>
<main>
  <h1>HADES está offline</h1>
  <p>O servidor local do HADES não respondeu. <strong>Nada foi inventado:</strong> o chat, as
     missões e as aprovações <em>não</em> vêm do cache — elas precisam do servidor.</p>
  <p>O que continua a funcionar offline é o <strong>shell</strong> da aplicação.</p>
  <button onclick="location.reload()">Tentar de novo</button>
  <p class="nota">As respostas de API nunca são guardadas em cache, por desenho.</p>
</main>
</body>
</html>`;

/* --------------------------------------------------------------- ciclo de vida */

self.addEventListener("install", (evento) => {
  evento.waitUntil(
    (async () => {
      const shell = await caches.open(CACHE_SHELL);
      // *`addAll` e' atomico: se um falhar, o install falha inteiro* -- e preferimos
      // um SW que nao instala a um SW que instala meio shell.
      await shell.addAll(SHELL);
      const assets = await caches.open(CACHE_ASSETS);
      // Icones: best-effort -- um icone em falta nao pode impedir o shell de funcionar,
      // mas fica **registado** em vez de engolido em silencio.
      for (const icone of ICONES) {
        try {
          await assets.add(icone);
        } catch (erro) {
          console.warn("[hades-sw] icone nao cacheado:", icone, erro);
        }
      }
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (evento) => {
  evento.waitUntil(
    (async () => {
      // Limpa TODAS as versoes antigas -- *um cache que ninguem apaga e' divida em disco.*
      const nomes = await caches.keys();
      await Promise.all(
        nomes
          .filter((nome) => !nome.startsWith(VERSAO))
          .map((nome) => caches.delete(nome)),
      );
      await self.clients.claim();
    })(),
  );
});

/* --------------------------------------------------------------- fetch */

/** `/api/*` e' a fronteira: *quem responde por uma aprovacao tem de ser o servidor.* */
function eApi(url) {
  return url.pathname === "/api" || url.pathname.startsWith("/api/");
}

/** Assets do build: o nome tem hash, logo o conteudo nao muda. */
function eAssetImutavel(url) {
  return url.pathname.startsWith("/assets/");
}

function eNavegacao(evento) {
  return evento.request.mode === "navigate";
}

self.addEventListener("fetch", (evento) => {
  const pedido = evento.request;
  const url = new URL(pedido.url);

  // (1) SO' GET entra no cache. POST/PUT/DELETE vao directos a rede.
  if (pedido.method !== "GET") return;

  // (2) A API NUNCA e' cacheada. Sem excepcao.
  if (eApi(url)) return;

  // (3) Outra origem? Nao gerimos.
  if (url.origin !== self.location.origin) return;

  // (4) Assets com hash: cache-first.
  if (eAssetImutavel(url)) {
    evento.respondWith(cachePrimeiro(pedido));
    return;
  }

  // (5) Navegacao: network-first, com o shell em cache como plano B.
  if (eNavegacao(evento)) {
    evento.respondWith(navegacao(pedido));
    return;
  }

  // (6) O resto: network-first com fallback ao cache.
  evento.respondWith(redePrimeiro(pedido));
});

async function cachePrimeiro(pedido) {
  const cache = await caches.open(CACHE_ASSETS);
  const guardado = await cache.match(pedido);
  if (guardado) return guardado;
  try {
    const resposta = await fetch(pedido);
    if (resposta && resposta.ok) cache.put(pedido, resposta.clone());
    return resposta;
  } catch (erro) {
    // *Um asset imutavel que nao esta' em cache e nao vem da rede: nao ha' como inventar.*
    return new Response("", { status: 504, statusText: "Gateway Timeout (offline)" });
  }
}

async function navegacao(pedido) {
  try {
    const resposta = await fetch(pedido);
    return resposta;
  } catch (erro) {
    const shell = await caches.open(CACHE_SHELL);
    const guardado = await shell.match("/index.html");
    if (guardado) return guardado;
    return new Response(OFFLINE_HTML, {
      status: 503,
      statusText: "Service Unavailable (offline)",
      headers: { "content-type": "text/html; charset=utf-8" },
    });
  }
}

async function redePrimeiro(pedido) {
  try {
    return await fetch(pedido);
  } catch (erro) {
    const cache = await caches.open(CACHE_SHELL);
    const guardado = await cache.match(pedido);
    if (guardado) return guardado;
    return new Response("", { status: 504, statusText: "Gateway Timeout (offline)" });
  }
}
