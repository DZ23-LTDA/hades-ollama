// Contrato do CANAL LOCAL -- testes de contrato.
//
// O canal real (native host) ainda nao existe. Estes testes provam que o CONTRATO
// esta' fechado: quando o host existir, ele nao pode aceitar token estatico, replay,
// ligacao LAN, instalacao desconhecida, chave expirada ou mensagem fora de ordem.
//
// Cada teste aqui e' um defeito que o host futuro NAO pode ter.
//
// Correr: node --test apps/hades-browser-bridge/test/canal_local.test.js
import test from "node:test";
import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";

// O service worker tem `crypto` global; o node ESM nao. Isto NAO enfraquece o
// contrato: e' a MESMA implementacao WebCrypto que o navegador usa.
globalThis.crypto ??= webcrypto;

/** Remove comentarios de linha e de bloco, preservando o resto do codigo. */
function semComentarios(fonte) {
  return fonte
    .replace(/\/\*[\s\S]*?\*\//g, " ")
    .replace(/(^|[^:])\/\/[^\n]*/g, "$1 ");
}

const C = await import("../src/canal_local.js");

const AGORA = 1_700_000_000_000;

/** Cria um estado com um dispositivo pareado e devolve tudo para assinar. */
async function par(instalacao = "inst-teste") {
  const estado = C.estadoInicial();
  const chave = await C.gerarChave();
  const bytes = await C.exportarChave(chave);
  const r = C.registrarDispositivo(estado, instalacao, bytes, AGORA);
  assert.equal(r.ok, true, JSON.stringify(r));
  return { estado, chave, instalacao };
}

/** Assina e devolve o envelope pronto a ser aceito. */
async function enviar(chave, campos) {
  const e = C.envelope(campos);
  e.mac = await C.assinar(chave, e);
  return e;
}

/* --------------------------------------------------------------- aleatoriedade */

test("a aleatoriedade vem de WebCrypto, nunca de Math.random", () => {
  // Procurar SO' no CODIGO: o comentario do modulo MENCIONA Math.random para o
  // proibir, e um detector sobre o texto cru confundiria a mencao com o uso.
  // Esta e' a 4a vez nesta missao que essa confusao aparece -- e agora esta' isolada.
  const t = semComentarios(String(C.aleatorio.toString()));
  // A CHAMADA, nao a mencao: a mensagem de erro do modulo diz
  // "nao gere segredo com Math.random" -- ela menciona para PROIBIR, e a string
  // e' codigo (nao comentario). O uso real tem parenteses.
  assert.ok(!/Math\.random\s*\(/.test(t), "Math.random() chamado");
  assert.ok(t.includes("crypto.getRandomValues"), "tem de usar getRandomValues");
  // e o codigo curto tem o tamanho declarado
  for (let i = 0; i < 40; i++) {
    const c = C.gerarCodigoCurto();
    assert.equal(c.length, C.LIMITES.codigoCurtoDigitos);
    assert.match(c, /^[0-9]+$/);
  }
});

test("40 codigos curtos seguidos nao se repetem (nao ha contador previsivel)", () => {
  const vistos = new Set();
  for (let i = 0; i < 40; i++) vistos.add(C.gerarCodigoCurto());
  assert.ok(vistos.size > 30, "repeticao excessiva: " + vistos.size + "/40");
});

/* ---------------------------------------------------------------------- canal */

test("canal: native e loopback sao aceitos; LAN e rede publica NAO sao", () => {
  assert.equal(C.canalPermitido("native").ok, true);
  for (const h of ["http://127.0.0.1:1234", "http://localhost:1234", "http://[::1]:9"]) {
    assert.equal(C.canalPermitido(h).ok, true, h);
  }
  // LAN e publica: recusadas SEM opcao
  for (const h of ["http://192.168.1.10:1", "http://10.0.0.1:1", "https://evil.example.com",
                   "http://169.254.169.254/", "http://0.0.0.0:1", "http://[fe80::1]:1"]) {
    const r = C.canalPermitido(h);
    assert.equal(r.ok, false, h + " devia ser recusado");
    assert.equal(r.motivo, "canal_nao_e_loopback", h);
  }
});

test("canal: esquema estranho e credenciais embutidas sao recusados", () => {
  assert.equal(C.canalPermitido("ws://127.0.0.1:1").motivo, "canal_esquema_nao_permitido");
  assert.equal(C.canalPermitido("file:///etc/passwd").motivo, "canal_esquema_nao_permitido");
  assert.equal(C.canalPermitido("http://u:p@127.0.0.1:1").motivo, "canal_com_credenciais");
  assert.equal(C.canalPermitido("").motivo, "canal_ausente");
  assert.equal(C.canalPermitido("nao e url").motivo, "canal_invalido");
});

/* -------------------------------------------------------------------- pairing */

test("pairing: codigo certo passa; errado incrementa tentativas", () => {
  const p = C.abrirPairing(AGORA);
  assert.equal(p.tentativas, 0);
  assert.equal(String(p.codigo).length, 6);
  const mau = C.confirmarPairing(p, "000000" === p.codigo ? "111111" : "000000", AGORA);
  assert.equal(mau.ok, false);
  assert.equal(mau.motivo, "codigo_incorreto");
  assert.equal(p.tentativas, 1);
  assert.equal(C.confirmarPairing(p, p.codigo, AGORA).ok, true);
});

test("pairing: codigo expirado e' recusado mesmo se estiver certo", () => {
  const p = C.abrirPairing(AGORA);
  const tarde = AGORA + C.TTL_CODIGO_MS + 1;
  const r = C.confirmarPairing(p, p.codigo, tarde);
  assert.equal(r.ok, false);
  assert.equal(r.motivo, "codigo_expirado");
});

test("pairing: forca bruta tem teto -- a 6a tentativa morre mesmo com o codigo certo", () => {
  const p = C.abrirPairing(AGORA);
  const errado = p.codigo === "000000" ? "111111" : "000000";
  for (let i = 0; i < C.LIMITES.tentativasDePairing; i++) {
    assert.equal(C.confirmarPairing(p, errado, AGORA).ok, false);
  }
  const r = C.confirmarPairing(p, p.codigo, AGORA);
  assert.equal(r.ok, false);
  assert.equal(r.motivo, "tentativas_esgotadas");
});

/* ------------------------------------------------------------------ envelope */

test("envelope valido e' aceito, e a sequencia avanca", async () => {
  const { estado, chave, instalacao } = await par();
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: "m1",
                                  sequencia: 1, corpo: "ola", agora: AGORA });
  const r = await C.aceitar(e, estado, AGORA);
  assert.equal(r.ok, true, JSON.stringify(r));
  assert.equal(r.corpo, "ola");
  assert.equal(estado.dispositivos[instalacao].ultimaSequencia, 1);
});

test("ANTI-REPLAY: o MESMO envelope duas vezes e' recusado na segunda", async () => {
  const { estado, chave, instalacao } = await par();
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                  sequencia: 1, corpo: "x", agora: AGORA });
  assert.equal((await C.aceitar(e, estado, AGORA)).ok, true);
  const r2 = await C.aceitar(e, estado, AGORA);
  assert.equal(r2.ok, false);
  assert.equal(r2.motivo, "replay_ou_fora_de_ordem");
});

test("ANTI-REPLAY: sequencia para tras e' recusada; salto para a frente e' aceito", async () => {
  const { estado, chave, instalacao } = await par();
  await C.aceitar(await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                        sequencia: 5, corpo: "a", agora: AGORA }), estado, AGORA);
  const atras = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                      sequencia: 3, corpo: "b", agora: AGORA });
  assert.equal((await C.aceitar(atras, estado, AGORA)).motivo, "replay_ou_fora_de_ordem");
  const frente = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                       sequencia: 9, corpo: "c", agora: AGORA });
  assert.equal((await C.aceitar(frente, estado, AGORA)).ok, true);
});

test("um envelope RECUSADO nao consome numero de sequencia", async () => {
  const { estado, chave, instalacao } = await par();
  const mau = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                    sequencia: 1, corpo: "ok", agora: AGORA });
  mau.mac = "0".repeat(64); // assinatura invalida
  assert.equal((await C.aceitar(mau, estado, AGORA)).motivo, "assinatura_invalida");
  assert.equal(estado.dispositivos[instalacao].ultimaSequencia, 0, "nao pode avancar");
  const bom = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                    sequencia: 1, corpo: "ok", agora: AGORA });
  assert.equal((await C.aceitar(bom, estado, AGORA)).ok, true, "o 1 tem de continuar livre");
});

test("assinatura: chave de OUTRO dispositivo nao vale", async () => {
  const a = await par("inst-a");
  const b = await par("inst-b");
  const e = await enviar(b.chave, { tipoCanal: "native", instalacao: "inst-a", missao: null,
                                    sequencia: 1, corpo: "falso", agora: AGORA });
  assert.equal((await C.aceitar(e, a.estado, AGORA)).motivo, "assinatura_invalida");
});

test("assinatura: trocar o corpo, a missao ou a instalacao invalida o MAC", async () => {
  const { estado, chave, instalacao } = await par();
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: "m1",
                                  sequencia: 1, corpo: "original", agora: AGORA });
  const adulterado = { ...e, corpo: "alterado" };
  assert.equal((await C.aceitar(adulterado, estado, AGORA)).motivo, "assinatura_invalida");
  const trocado = { ...e, missao: "m2" };
  assert.equal((await C.aceitar(trocado, estado, AGORA)).motivo, "assinatura_invalida");
  const mudouInstalacao = { ...e, instalacao: "outra" };
  assert.equal((await C.aceitar(mudouInstalacao, estado, AGORA)).motivo, "instalacao_desconhecida");
});

/* ------------------------------------------------------------ ciclo de vida */

test("instalacao desconhecida e' recusada: o default e' FECHADO", async () => {
  const { chave, instalacao } = await par();
  const estado = C.estadoInicial();
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                  sequencia: 1, corpo: "x", agora: AGORA });
  assert.equal((await C.aceitar(e, estado, AGORA)).motivo, "instalacao_desconhecida");
});

test("chave expirada e' recusada, mesmo com assinatura valida", async () => {
  const { estado, chave, instalacao } = await par();
  const tarde = AGORA + C.TTL_CHAVE_MS + 1;
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                  sequencia: 1, corpo: "x", agora: tarde });
  assert.equal((await C.aceitar(e, estado, tarde)).motivo, "chave_expirada");
});

test("revogar e' instantaneo e apaga o material da chave", async () => {
  const { estado, chave, instalacao } = await par();
  assert.equal(C.revogarDispositivo(estado, instalacao).ok, true);
  assert.equal(estado.dispositivos[instalacao].chave.length, 0, "chave apagada");
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                  sequencia: 1, corpo: "x", agora: AGORA });
  assert.equal((await C.aceitar(e, estado, AGORA)).motivo, "instalacao_revogada");
});

test("rodar a chave invalida a antiga e reinicia a sequencia", async () => {
  const { estado, chave, instalacao } = await par();
  await C.aceitar(await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                        sequencia: 7, corpo: "a", agora: AGORA }), estado, AGORA);
  assert.equal((await C.rodarChave(estado, instalacao, AGORA)).ok, true);
  assert.equal(estado.dispositivos[instalacao].ultimaSequencia, 0);
  const comAntiga = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                          sequencia: 1, corpo: "b", agora: AGORA });
  assert.equal((await C.aceitar(comAntiga, estado, AGORA)).motivo, "assinatura_invalida");
  const nova = await C.importarChave(estado.dispositivos[instalacao].chave);
  const comNova = await enviar(nova, { tipoCanal: "native", instalacao, missao: null,
                                       sequencia: 1, corpo: "c", agora: AGORA });
  assert.equal((await C.aceitar(comNova, estado, AGORA)).ok, true);
});

test("rodar chave de instalacao revogada e' recusado", async () => {
  const { estado, instalacao } = await par();
  C.revogarDispositivo(estado, instalacao);
  assert.equal((await C.rodarChave(estado, instalacao, AGORA)).motivo, "instalacao_revogada");
});

/* ------------------------------------------------------------------ validacao */

test("relogio fora da janela e' recusado (passado e futuro)", async () => {
  const { estado, chave, instalacao } = await par();
  const velho = AGORA - C.LIMITES.desvioDeRelogioMs - 1;
  const e1 = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                   sequencia: 1, corpo: "x", agora: velho });
  assert.equal((await C.aceitar(e1, estado, AGORA)).motivo, "relogio_fora_da_janela");
  const futuro = AGORA + C.LIMITES.desvioDeRelogioMs + 1;
  const e2 = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                   sequencia: 1, corpo: "x", agora: futuro });
  assert.equal((await C.aceitar(e2, estado, AGORA)).motivo, "relogio_fora_da_janela");
});

test("mensagem gigante e' recusada ANTES de ser assinada", async () => {
  const { estado, chave, instalacao } = await par();
  const e = await enviar(chave, { tipoCanal: "native", instalacao, missao: null,
                                  sequencia: 1, corpo: "a".repeat(C.LIMITES.mensagemBytes + 1),
                                  agora: AGORA });
  assert.equal((await C.aceitar(e, estado, AGORA)).motivo, "mensagem_demasiado_grande");
});

test("envelope malformado e' recusado campo a campo", async () => {
  const { estado, chave, instalacao } = await par();
  const base = { tipoCanal: "native", instalacao, missao: null, sequencia: 1,
                 corpo: "x", agora: AGORA };
  const casos = [
    [null, "envelope_invalido"],
    [{}, "protocolo_incompativel"],
    [{ ...C.envelope(base), v: 99 }, "protocolo_incompativel"],
    [{ ...C.envelope(base), tipoCanal: "lan" }, "tipo_de_canal_invalido"],
    [{ ...C.envelope(base), corpo: 42 }, "corpo_invalido"],
    [{ ...C.envelope(base), instalacao: "" }, "instalacao_invalida"],
    [{ ...C.envelope(base), sequencia: 0 }, "sequencia_invalida"],
    [{ ...C.envelope(base), sequencia: 1.5 }, "sequencia_invalida"],
    [{ ...C.envelope(base), instante: "agora" }, "instante_invalido"],
  ];
  for (const [entrada, motivo] of casos) {
    const r = await C.aceitar(entrada, estado, AGORA);
    assert.equal(r.ok, false, JSON.stringify(entrada));
    assert.equal(r.motivo, motivo, JSON.stringify(entrada));
  }
});

test("a mensagem canonica e' estavel e cobre o contexto todo", () => {
  const base = { tipoCanal: "native", instalacao: "i", missao: "m", sequencia: 1,
                 instante: AGORA, corpo: "c" };
  const a = new TextDecoder().decode(C.canonico(base));
  assert.equal(a, a.normalize());
  // cada campo muda a mensagem canonica
  for (const campo of ["tipoCanal", "instalacao", "missao", "sequencia", "instante", "corpo"]) {
    const outro = { ...base, [campo]: campo === "sequencia" ? 2 : base[campo] + "X" };
    const b = new TextDecoder().decode(C.canonico(outro));
    assert.notEqual(a, b, campo + " nao entra na mensagem canonica");
  }
});

/* ------------------------------------------------------------------ regressao */

test("nao ha token estatico nem segredo literal no modulo", async () => {
  const fs = await import("node:fs");
  const path = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const raiz = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
  const bruto = fs.readFileSync(path.join(raiz, "src/canal_local.js"), "utf8");
  const t = semComentarios(bruto);
  assert.ok(!/Math\.random\s*\(/.test(t), "Math.random() chamado no codigo");
  // A ATRIBUICAO, nao a mencao: o modulo fala de "token estatico" nos comentarios.
  assert.ok(!/(token|secret|senha|password)\s*[:=]\s*["'][A-Za-z0-9_-]{8,}/i.test(t),
    "parece haver segredo literal");
  assert.ok(!/eval\s*\(/.test(t), "eval");
  // URL nao-loopback no CODIGO (a lista de hosts permitidos e' dados, nao URL)
  const urls = t.match(/https?:\/\/[^"'`\s)]+/g) ?? [];
  const fora = urls.filter((u) => !/^https?:\/\/(127\.|localhost|\[::1\]|::1)/.test(u));
  assert.deepEqual(fora, [], "URL nao-loopback no codigo: " + fora.join(", "));
});

/* ------------------------------------------------------- derivacao (E-1024) */

test("a chave do canal DERIVA do codigo curto, nao e' o codigo", async () => {
  const pedido = C.abrirPairing(AGORA);
  const a = await C.derivarChave(pedido.codigo, pedido.pedido, "inst-1");
  const b = await C.derivarChave(pedido.codigo, pedido.pedido, "inst-1");
  const bytes = await C.exportarChave(a);
  const bytes2 = await C.exportarChave(b);
  assert.equal(bytes.length, C.HKDF_BITS / 8, "a chave tem 256 bits");
  assert.deepEqual(Array.from(bytes), Array.from(bytes2), "derivacao determinista");
  // e a chave NAO e' o codigo curto
  assert.notEqual(Buffer.from(bytes).toString("utf8"), pedido.codigo);
});

test("o MESMO codigo em contextos diferentes da' chaves DIFERENTES", async () => {
  // Sem salt/info, um codigo de 6 digitos seria a mesma chave em todo o lado.
  const codigo = "123456";
  const k1 = await C.exportarChave(await C.derivarChave(codigo, "pr-a", "inst-1"));
  const k2 = await C.exportarChave(await C.derivarChave(codigo, "pr-b", "inst-1"));
  const k3 = await C.exportarChave(await C.derivarChave(codigo, "pr-a", "inst-2"));
  const k4 = await C.exportarChave(await C.derivarChave(codigo, "pr-a", "inst-1", "m1"));
  const todos = [k1, k2, k3, k4].map((x) => Buffer.from(x).toString("hex"));
  assert.equal(new Set(todos).size, 4, "os 4 contextos tem de dar 4 chaves");
  for (const t of todos) assert.notEqual(t, codigo);
});

test("derivar sem codigo LANCA em vez de gerar chave fraca", async () => {
  await assert.rejects(() => C.derivarChave("", "pr", "inst"), /codigo curto ausente/);
  await assert.rejects(() => C.derivarChave(null, "pr", "inst"), /codigo curto ausente/);
});

test("a chave derivada assina e verifica como qualquer outra", async () => {
  const pedido = C.abrirPairing(AGORA);
  const chave = await C.derivarChave(pedido.codigo, pedido.pedido, "inst-1", "m1");
  const estado = C.estadoInicial();
  const bytes = await C.exportarChave(chave);
  assert.equal(C.registrarDispositivo(estado, "inst-1", bytes, AGORA).ok, true);
  const e = await enviar(chave, { tipoCanal: "native", instalacao: "inst-1",
                                  missao: "m1", sequencia: 1, corpo: "ok", agora: AGORA });
  assert.equal((await C.aceitar(e, estado, AGORA)).ok, true);
});

test("o codigo curto e' de USO UNICO: consumido, nao volta a valer", () => {
  const pedido = C.abrirPairing(AGORA);
  assert.equal(C.confirmarPairing(pedido, pedido.codigo, AGORA).ok, true);
  assert.equal(C.consumirCodigo(pedido).ok, true);
  assert.equal(pedido.codigo, null, "o codigo e' apagado");
  const r = C.confirmarPairing(pedido, "000000", AGORA);
  assert.equal(r.ok, false);
  assert.equal(r.motivo, "codigo_ja_usado");
});

test("consumir duas vezes e' recusado", () => {
  const pedido = C.abrirPairing(AGORA);
  assert.equal(C.consumirCodigo(pedido).ok, true);
  assert.equal(C.consumirCodigo(pedido).motivo, "codigo_ja_usado");
  assert.equal(C.consumirCodigo(null).motivo, "pedido_ausente");
});

/* ------------------------------------------------------- native host (E-1024) */

test("o manifesto do native host existe, e' stdio e NAO tem origem curinga", async () => {
  const fs = await import("node:fs");
  const path = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const raiz = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
  const p = path.join(raiz, "native-host", "com.dz23ltd.hades.bridge.json");
  const m = JSON.parse(fs.readFileSync(p, "utf8"));

  assert.equal(m.type, "stdio", "native messaging usa stdio");
  assert.match(m.name, /^[a-z0-9._]+$/, "o nome tem de ser DNS-like");
  assert.ok(Array.isArray(m.allowed_origins) && m.allowed_origins.length >= 1);
  for (const o of m.allowed_origins) {
    // A REGRA: nada de "*", nada de <all_urls>, nada de http. So' a extensao HADES.
    assert.match(o, /^chrome-extension:\/\/[a-p]{32}\/$|^chrome-extension:\/\/REPLACE_WITH_EXTENSION_ID\/$/,
      "origem nao e' um id de extensao: " + o);
    assert.ok(!o.includes("*"), "origem curinga: " + o);
  }
  // o path e' um placeholder explicito, porque o Companion NAO EXISTE
  assert.match(m.path, /^REPLACE_WITH_/, "o path tem de ser um placeholder declarado");
});

test("a extensao NAO pede nativeMessaging no nucleo, mas declara-o como opcional", async () => {
  const fs = await import("node:fs");
  const path = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const raiz = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
  const m = JSON.parse(fs.readFileSync(path.join(raiz, "manifest.json"), "utf8"));
  assert.ok(!(m.permissions ?? []).includes("nativeMessaging"),
    "nativeMessaging no nucleo da' poder a toda a extensao desde a instalacao");
  assert.ok((m.optional_permissions ?? []).includes("nativeMessaging"),
    "sem optional_permissions, o tipo de canal `native` do contrato nao pode existir");
});
