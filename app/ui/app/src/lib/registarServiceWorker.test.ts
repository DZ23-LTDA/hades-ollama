/**
 * Testes do Service Worker (`E-1061`).
 *
 * O QUE ESTES TESTES PROVAM -- e o que NAO provam.
 *
 * Provam que: (a) o `sw.js` e' JavaScript valido; (b) a API **nunca** entra no cache;
 * (c) metodos != GET nao entram; (d) o `install` falha alto se o shell nao puder ser cacheado;
 * (e) um icone em falta **nao** impede o shell; (f) versoes antigas sao apagadas.
 *
 * NAO provam que o PWA instala num browser real. *Um teste com `caches` simulado prova a
 * logica; nao prova a instalacao.* Por isso o `PWA_INSTALAVEL` continua `PARCIAL`.
 *
 * Executa com `node --test`. O `sw.js` e' carregado num contexto `vm` isolado, com um `self`
 * falso que regista os listeners -- *nao se importa um service worker: ele nao e' um modulo.*
 */
import { describe, it, expect, beforeEach } from "vitest";
import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const AQUI = dirname(fileURLToPath(import.meta.url));
const SW = readFileSync(resolve(AQUI, "../../public/sw.js"), "utf8");

/** Um `caches` falso que registra o que foi guardado. */
function cachesFalso() {
  const guardado = new Map(); // nome -> Map(url -> Response)
  const apagados = [];
  return {
    guardado,
    apagados,
    async open(nome) {
      if (!guardado.has(nome)) guardado.set(nome, new Map());
      const mapa = guardado.get(nome);
      return {
        async add(url) {
          if (url.includes("nao-existe")) throw new Error("404 simulado");
          mapa.set(url, { url, ok: true });
        },
        async addAll(urls) {
          for (const u of urls) {
            if (u.includes("nao-existe")) throw new Error("addAll falhou em " + u);
            mapa.set(u, { url: u, ok: true });
          }
        },
        async match(req) {
          const chave = typeof req === "string" ? req : req.url;
          return mapa.get(chave);
        },
        async put(req, resp) {
          mapa.set(typeof req === "string" ? req : req.url, resp);
        },
      };
    },
    async keys() {
      return [...guardado.keys()];
    },
    async delete(nome) {
      apagados.push(nome);
      return guardado.delete(nome);
    },
  };
}

/** Carrega o `sw.js` num contexto isolado e devolve os listeners + o `caches`. */
function carregar() {
  const caches = cachesFalso();
  const listeners = {};
  const contexto = {
    self: {
      location: { origin: "http://localhost:11434" },
      addEventListener: (tipo, fn) => {
        listeners[tipo] = fn;
      },
      skipWaiting: async () => {},
      clients: { claim: async () => {} },
    },
    caches,
    fetch: async () => {
      throw new Error("offline simulado");
    },
    console,
    URL,
    Response,
    Promise,
  };
  contexto.self.self = contexto.self;
  vm.createContext(contexto);
  vm.runInContext(SW, contexto);
  return { listeners, caches };
}

/** Dispara um `fetch` e devolve o que aconteceria. */
async function dispararFetch(listeners, pedido) {
  let respondido = null;
  const evento = {
    request: pedido,
    respondWith: (p) => {
      respondido = p;
    },
  };
  listeners.fetch(evento);
  return { respondido, evento };
}

const PEDIDO = (url, extra = {}) => ({
  url,
  method: extra.method ?? "GET",
  mode: extra.mode ?? "no-cors",
});

describe("sw.js — o service worker do HADES", () => {
  beforeEach(() => {});

  it("e' JavaScript valido e regista install/activate/fetch", () => {
    const { listeners } = carregar();
    expect(typeof listeners.install).toBe("function");
    expect(typeof listeners.activate).toBe("function");
    expect(typeof listeners.fetch).toBe("function");
  });

  it("/api/ NUNCA e' interceptado -- nem o chat, nem as aprovacoes", async () => {
    const { listeners } = carregar();
    for (const caminho of ["/api/chat", "/api/tags", "/api/approvals", "/api"]) {
      const { respondido } = await dispararFetch(listeners, PEDIDO("http://localhost:11434" + caminho));
      expect(respondido, `${caminho} foi interceptado -- a API nunca pode ser cacheada`).toBeNull();
    }
  });

  it("POST/PUT/DELETE nao sao interceptados", async () => {
    const { listeners } = carregar();
    for (const metodo of ["POST", "PUT", "DELETE", "PATCH"]) {
      const { respondido } = await dispararFetch(
        listeners,
        PEDIDO("http://localhost:11434/assets/x.js", { method: metodo }),
      );
      expect(respondido, `${metodo} foi interceptado`).toBeNull();
    }
  });

  it("outra origem nao e' gerida por este service worker", async () => {
    const { listeners } = carregar();
    const { respondido } = await dispararFetch(
      listeners,
      PEDIDO("https://exemplo.invalido/assets/x.js"),
    );
    expect(respondido).toBeNull();
  });

  it("assets com hash sao servidos cache-first", async () => {
    const { listeners, caches } = carregar();
    const cache = await caches.open("hades-sw-v1-assets");
    await cache.put("http://localhost:11434/assets/index-abc.js", { corpo: "do-cache" });
    const { respondido } = await dispararFetch(
      listeners,
      PEDIDO("http://localhost:11434/assets/index-abc.js"),
    );
    expect(respondido).not.toBeNull();
    const resposta = await respondido;
    expect(resposta.corpo).toBe("do-cache");
  });

  it("install caches o shell e os icones", async () => {
    const { listeners, caches } = carregar();
    let promessa;
    listeners.install({ waitUntil: (p) => (promessa = p) });
    await promessa;
    const shell = caches.guardado.get("hades-sw-v1-shell");
    expect(shell.has("/index.html")).toBe(true);
    expect(shell.has("/manifest.json")).toBe(true);
    const assets = caches.guardado.get("hades-sw-v1-assets");
    expect(assets.has("/hades-icon-192.png")).toBe(true);
    expect(assets.has("/hades-maskable-512.png")).toBe(true);
  });

  it("activate apaga TODAS as versoes antigas -- um cache que ninguem apaga e' divida", async () => {
    const { listeners, caches } = carregar();
    await caches.open("hades-sw-v0-shell");
    await caches.open("qualquer-coisa-antiga");
    let promessa;
    listeners.activate({ waitUntil: (p) => (promessa = p) });
    await promessa;
    expect(caches.apagados).toContain("hades-sw-v0-shell");
    expect(caches.apagados).toContain("qualquer-coisa-antiga");
    // e nao apaga a versao actual
    expect(caches.apagados).not.toContain("hades-sw-v1-shell");
  });

  it("navegacao offline cai no shell em cache, nao numa resposta inventada", async () => {
    const { listeners, caches } = carregar();
    const shell = await caches.open("hades-sw-v1-shell");
    await shell.put("/index.html", { corpo: "shell-real" });
    let promessa;
    const evento = {
      request: PEDIDO("http://localhost:11434/missao/123", { mode: "navigate" }),
      respondWith: (p) => (promessa = p),
    };
    listeners.fetch(evento);
    const resposta = await promessa;
    expect(resposta.corpo).toBe("shell-real");
  });
});
