// Verificacao de CARREGABILIDADE da HADES Browser Bridge.
//
// Nao basta o manifest existir: um manifest MV3 que aponta para ficheiros
// ausentes NAO carrega. Este teste falha exactamente nesse caso.
//
// Correr: node --test apps/hades-browser-bridge/test/carregavel.test.js
import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync, existsSync, statSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));
const raiz = join(aqui, "..");
const ler = (p) => readFileSync(join(raiz, p), "utf8");
const manifest = JSON.parse(ler("manifest.json"));

/** Todos os caminhos que o manifest referencia. */
function referenciados(m) {
  const out = new Set();
  out.add(m.background?.service_worker);
  out.add(m.action?.default_popup);
  for (const cs of m.content_scripts ?? []) for (const j of cs.js ?? []) out.add(j);
  for (const v of Object.values(m.icons ?? {})) out.add(v);
  for (const v of Object.values(m.action?.default_icon ?? {})) out.add(v);
  return [...out].filter(Boolean);
}

test("TODO ficheiro referenciado pelo manifest existe em disco", () => {
  const faltam = referenciados(manifest).filter((p) => !existsSync(join(raiz, p)));
  assert.deepEqual(faltam, [], `ficheiros ausentes: ${faltam.join(", ")}`);
});

test("os ficheiros referenciados nao estao vazios", () => {
  for (const p of referenciados(manifest)) {
    assert.ok(statSync(join(raiz, p)).size > 0, `${p} esta' vazio`);
  }
});

test("os icones sao PNG de verdade (assinatura, nao extensao)", () => {
  for (const p of Object.values(manifest.icons ?? {})) {
    const b = readFileSync(join(raiz, p));
    assert.equal(b.subarray(0, 8).toString("hex"), "89504e470d0a1a0a", `${p} nao e' PNG`);
  }
});

test("nenhum ficheiro da extensao usa eval, new Function ou codigo remoto", () => {
  const ficheiros = [
    "src/protocol.js", "src/service_worker.js", "src/content_script.js", "popup/popup.js",
  ];
  for (const f of ficheiros) {
    const t = ler(f);
    assert.ok(!/\beval\s*\(/.test(t), `${f} usa eval`);
    assert.ok(!/new\s+Function\s*\(/.test(t), `${f} usa new Function`);
    // nenhuma URL absoluta de carregamento (import/script/fetch para fora)
    assert.ok(!/https?:\/\/[^"'`\s]+\/(src|dist|assets)\//.test(t), `${f} carrega codigo remoto`);
  }
});

test("o service worker e' um modulo e importa o protocolo por caminho relativo", () => {
  assert.equal(manifest.background.type, "module");
  const sw = ler("src/service_worker.js");
  assert.match(sw, /from\s+"\.\/protocol\.js"/, "tem de importar ./protocol.js");
  assert.ok(!/from\s+"https?:/.test(sw), "import remoto e' proibido");
});

test("o content script NAO usa modulos (MV3 classico) e nao guarda segredos", () => {
  const cs = ler("src/content_script.js");
  assert.ok(!/^\s*import\s/m.test(cs), "content_scripts classicos nao podem usar import");
  assert.ok(!/^\s*export\s/m.test(cs), "content_scripts classicos nao podem usar export");
  // Atribuicao de segredo, com o grupo explicito: o `|` sem parenteses daria
  // `token` OU `secret` OU `password...`, e apanharia a palavra "tokens" num
  // comentario. A regra e' a ATRIBUICAO, nao a mencao.
  const atribuicaoSegredo = /(token|secret|password|apikey|api_key)\s*[:=]\s*["'`]/im;
  assert.ok(!atribuicaoSegredo.test(cs), "content script nao guarda segredos");
});

test("o popup usa o icone da marca HADES e nao um asset remoto", () => {
  const html = ler("popup/popup.html");
  assert.match(html, /icons\/hades-icon-192\.png/);
  assert.ok(!/https?:\/\//.test(html), "o popup nao carrega nada remoto");
});

test("o CSS respeita prefers-reduced-motion", () => {
  assert.match(ler("popup/popup.css"), /prefers-reduced-motion/);
});

test("o manifest continua com o nucleo minimo e sem poder sobre o PC", () => {
  const todos = [...(manifest.permissions ?? []), ...(manifest.host_permissions ?? [])];
  for (const proibido of [
    "<all_urls>", "debugger", "nativeMessaging", "clipboardRead", "clipboardWrite",
    "downloads", "management", "proxy", "webRequestBlocking",
  ]) {
    assert.ok(!todos.includes(proibido), `${proibido} nao pode estar no nucleo`);
  }
  assert.equal(manifest.incognito, "not_allowed");
});

test("a extensao nao declara qualquer scope de sistema operativo", () => {
  // desktop:* pertence ao desktop.companion, nunca a esta extensao
  const todos = [
    ler("src/protocol.js"), ler("src/service_worker.js"),
    ler("src/content_script.js"), ler("popup/popup.js"),
  ].join("\n");
  assert.ok(!/"desktop:(screen|input|clipboard|process)"/.test(todos),
    "a extensao nao pode reclamar scopes de desktop");
});

test("nativeMessaging esta' em optional_permissions, NUNCA no nucleo", () => {
  const nucleo = [...(manifest.permissions ?? []), ...(manifest.host_permissions ?? [])];
  assert.ok(!nucleo.includes("nativeMessaging"),
    "nativeMessaging no nucleo seria poder desde a instalacao, sem consentimento");
  assert.ok((manifest.optional_permissions ?? []).includes("nativeMessaging"),
    "o canal `native` do contrato canal_local.js precisa desta permissao opcional");
  // e as opcionais nao podem alargar o poder sem pedido explicito do utilizador
  const opcionais = manifest.optional_permissions ?? [];
  for (const p of opcionais) {
    assert.ok(["nativeMessaging"].includes(p), "opcional nao prevista: " + p);
  }
});
