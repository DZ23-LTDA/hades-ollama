// Testes do protocolo da HADES Browser Bridge.
//
// Cada teste foi escrito para FALHAR se a proteccao correspondente for removida.
// Correr: node --test apps/hades-browser-bridge/test/
import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import {
  SCOPES, TOOL, LIMITES, validarUrl, validarCdpUrl, guardarRota,
  pedidoCanonico, campoSensivel, loopbackAutorizado,
} from "../src/protocol.js";

const aqui = dirname(fileURLToPath(import.meta.url));
const raiz = join(aqui, "..");
const manifest = JSON.parse(readFileSync(join(raiz, "manifest.json"), "utf8"));

test("os 3 scopes sao exactamente os do browser.operator", () => {
  assert.deepEqual([...SCOPES].sort(), ["browser:files", "browser:navigate", "browser:takeover"]);
  assert.equal(TOOL, "browser.operator");
});

test("o manifesto e' Manifest V3 e nao usa incognito", () => {
  assert.equal(manifest.manifest_version, 3);
  assert.equal(manifest.incognito, "not_allowed", "incognito tem de estar desligado por omissao");
  assert.ok(manifest.background.service_worker, "MV3 exige service_worker");
});

test("o manifesto nao pede <all_urls> nem permissoes de PC", () => {
  const todas = [...(manifest.permissions || []), ...(manifest.host_permissions || [])];
  assert.ok(!todas.includes("<all_urls>"), "<all_urls> permanente e' proibido");
  for (const p of ["debugger", "nativeMessaging", "clipboardRead", "clipboardWrite", "downloads", "management"]) {
    assert.ok(!todas.includes(p), `permissao ${p} nao pode estar no nucleo`);
  }
});

test("a CSP nao permite eval nem codigo remoto", () => {
  const csp = manifest.content_security_policy.extension_pages;
  assert.ok(!csp.includes("unsafe-eval"), "unsafe-eval e' proibido");
  assert.ok(!csp.includes("http://") && !csp.includes("https://"), "codigo remoto e' proibido");
  assert.ok(csp.includes("script-src 'self'"));
});

test("esquemas perigosos sao recusados antes de qualquer accao", () => {
  for (const mau of [
    "javascript:alert(1)",
    "file:///C:/Windows/System32/config/SAM",
    "data:text/html,<script>alert(1)</script>",
    "chrome-extension://abc/steal.html",
    "ftp://exemplo.com/x",
  ]) {
    const r = validarUrl(mau);
    assert.equal(r.ok, false, `${mau} devia ser recusado`);
  }
});

test("URLs com credenciais embutidas sao recusados", () => {
  const r = validarUrl("https://admin:segredo@exemplo.com/");
  assert.equal(r.ok, false);
  assert.equal(r.motivo, "credenciais_no_url");
});

test("rede privada e' recusada SEM opt-in explicito", () => {
  for (const privado of [
    "http://127.0.0.1:8080/admin",
    "http://10.0.0.5/",
    "http://192.168.1.1/",
    "http://169.254.169.254/latest/meta-data/",
    "http://localhost:11434/api/tags",
  ]) {
    const r = validarUrl(privado);
    assert.equal(r.ok, false, `${privado} devia ser recusado sem opt-in`);
    assert.equal(r.motivo, "rede_privada_sem_optin");
  }
});

test("rede privada e' aceite COM opt-in (mesma variavel do helper Python)", () => {
  assert.equal(loopbackAutorizado({ OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE: "1" }), true);
  assert.equal(loopbackAutorizado({}), false);
  assert.equal(loopbackAutorizado(null), false);
  const r = validarUrl("http://127.0.0.1:11434/", { permitirPrivado: true });
  assert.equal(r.ok, true);
});

test("o CDP so' pode ser loopback -- nunca LAN nem publico", () => {
  assert.equal(validarCdpUrl("http://127.0.0.1:9222/json/version").ok, true);
  assert.equal(validarCdpUrl("ws://localhost:9222/devtools/browser/abc").ok, true);
  const lan = validarCdpUrl("http://192.168.1.50:9222/json/version");
  assert.equal(lan.ok, false);
  assert.equal(lan.motivo, "cdp_fora_de_loopback");
  assert.equal(validarCdpUrl("https://cdp.exemplo.com/x").ok, false);
});

test("uma origem NAO concedida e' recusada mesmo sendo https valido", () => {
  const concedidos = ["https://exemplo.com"];
  assert.equal(guardarRota("https://exemplo.com/pagina", concedidos).ok, true);
  const outra = guardarRota("https://outro.com/pagina", concedidos);
  assert.equal(outra.ok, false);
  assert.equal(outra.motivo, "origem_nao_concedida");
  // a origem concedida nao pode ser alargada por um prefixo parecido
  assert.equal(guardarRota("https://exemplo.com.evil.net/", concedidos).ok, false);
});

test("o pedido canonico usa session_id = missao e exige aprovacao", () => {
  const bom = pedidoCanonico("missao-42", { scope: "browser:navigate", nome: "navigate" }, { url: "https://exemplo.com" });
  assert.equal(bom.ok, true);
  assert.equal(bom.pedido.session_id, "missao-42");
  assert.equal(bom.pedido.tool, "browser.operator");
  assert.equal(bom.pedido.requires_approval, true);
});

test("scopes inventados e missoes vazias sao recusados", () => {
  assert.equal(pedidoCanonico("", { scope: "browser:navigate", nome: "navigate" }).ok, false);
  assert.equal(pedidoCanonico("m", { scope: "browser:root", nome: "navigate" }).ok, false);
  assert.equal(pedidoCanonico("m", { scope: "desktop:input", nome: "mouse" }).ok, false,
    "desktop:input nao pertence a esta extensao");
  assert.equal(pedidoCanonico("m", { scope: "browser:navigate", nome: "   " }).ok, false);
});

test("campos sensiveis sao detectados por tipo, autocomplete E nome", () => {
  assert.equal(campoSensivel({ type: "password" }), true);
  assert.equal(campoSensivel({ type: "text", autocomplete: "cc-number" }), true);
  assert.equal(campoSensivel({ type: "text", autocomplete: "one-time-code" }), true);
  assert.equal(campoSensivel({ type: "text", id: "user-senha" }), true);
  assert.equal(campoSensivel({ type: "text", name: "card_cvc" }), true);
  assert.equal(campoSensivel({ type: "text", name: "pesquisa", id: "q" }), false);
});

test("os limites declarados sao aplicados, nao ignorados", () => {
  const longa = "https://exemplo.com/" + "a".repeat(LIMITES.url + 10);
  assert.equal(validarUrl(longa).motivo, "url_demasiado_longa");
  assert.ok(LIMITES.snapshotBytes > 0 && LIMITES.accoesPorMinuto > 0);
});
