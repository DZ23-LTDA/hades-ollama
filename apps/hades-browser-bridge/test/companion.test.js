/**
 * O COMPANION COMO PROCESSO REAL (`E-1029`).
 *
 * A diferenca em relacao ao `transporte.test.js`: ali o fio era memoria; aqui e' um
 * SUBPROCESSO de verdade, com `stdin`/`stdout` do SO. O que se prova: o enquadramento
 * e o handshake atravessam um pipe real, com dados a chegar em pedacos que o SO
 * escolhe -- nao os que o teste gostaria.
 */
import test from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

globalThis.crypto ??= (await import("node:crypto")).webcrypto;

const C = await import("../src/canal_local.js");
const T = await import("../src/transporte.js");
const H = await import("../native-host/companion.mjs");

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const COMPANION = path.join(AQUI, "..", "native-host", "companion.mjs");

/**
 * Fala com o processo. Le' o `stdout` byte a byte, remonta com o MESMO
 * `desenquadrar` e resolve quando chega a resposta com o `id` esperado.
 */
function arrancar() {
  const p = spawn(process.execPath, [COMPANION], { stdio: ["pipe", "pipe", "pipe"] });
  let pendente = new Uint8Array(0);
  const recebidas = [];
  const esperando = new Map();
  let stderr = "";
  let id = 0;

  p.stdout.on("data", (b) => {
    const junto = new Uint8Array(pendente.length + b.length);
    junto.set(pendente, 0);
    junto.set(new Uint8Array(b), pendente.length);
    let r;
    try {
      r = T.desenquadrar(junto);
    } catch (e) {
      for (const [, rej] of esperando) rej(e);
      esperando.clear();
      return;
    }
    pendente = r.resto;
    for (const m of r.mensagens) {
      recebidas.push(m);
      const w = esperando.get(m.id);
      if (w) { esperando.delete(m.id); w(m); }
    }
  });
  p.stderr.on("data", (b) => { stderr += b.toString("utf8"); });

  return {
    proc: p,
    stderr: () => stderr,
    recebidas: () => recebidas.slice(),
    /** Manda uma mensagem e espera a resposta com o MESMO id. */
    pedir(cmd, extra = {}, timeoutMs = 8000) {
      const meu = ++id;
      const m = { id: meu, cmd, ...extra };
      return new Promise((res, rej) => {
        const t = setTimeout(() => { esperando.delete(meu); rej(new Error("timeout no " + cmd)); }, timeoutMs);
        esperando.set(meu, (r) => { clearTimeout(t); res(r); });
        p.stdin.write(Buffer.from(T.enquadrar(m)));
      });
    },
    /** Escreve bytes crus, para provar o corpo parcial no pipe do SO. */
    cru(bytes) { p.stdin.write(Buffer.from(bytes)); },
    fechar() { try { p.stdin.end(); } catch {} },
    matar() { try { p.kill(); } catch {} },
  };
}

test("o host arranca como PROCESSO e responde pelo pipe do SO", async () => {
  const c = arrancar();
  try {
    const r = await c.pedir("host:ola");
    assert.equal(r.ok, true);
    assert.equal(r.hostProtocolo, H.HOST_PROTOCOLO);
    assert.equal(r.protocolo, C.PROTOCOLO, "o host fala o MESMO protocolo do contrato");
    assert.deepEqual(r.capacidades, [], "este host nao controla o PC: zero capacidades");
    assert.equal(r.instalado, false);
    assert.equal(r.assinado, false);
  } finally { c.fechar(); c.matar(); }
});

test("o diagnostico vai para stderr e NAO polui o fio", async () => {
  const c = arrancar();
  try {
    await c.pedir("host:ola");
    // Se o diagnostico tivesse ido para o stdout, o desenquadrador teria lancado
    // antes de chegar aqui -- a resposta chegou inteira.
    assert.equal(c.recebidas().length, 1);
    assert.ok(c.stderr().includes("[companion]"), "o diagnostico tem de existir no stderr");
    assert.ok(!c.stderr().includes("{\"ok\""), "stderr nao e' o fio");
  } finally { c.fechar(); c.matar(); }
});

test("PEDIDO PARTIDO: escrever os bytes UM A UM pelo pipe real", async () => {
  const c = arrancar();
  try {
    const bytes = T.enquadrar({ id: 99, cmd: "host:ola" });
    for (const b of bytes) c.cru(new Uint8Array([b]));
    // espera a resposta aparecer
    await new Promise((r) => setTimeout(r, 400));
    const rs = c.recebidas();
    assert.equal(rs.length, 1, "uma resposta, apesar de 1 byte por escrita");
    assert.equal(rs[0].id, 99);
    assert.equal(rs[0].ok, true);
  } finally { c.fechar(); c.matar(); }
});

test("DOIS pedidos COLADOS numa so' escrita saem os dois", async () => {
  const c = arrancar();
  try {
    const a = T.enquadrar({ id: 1, cmd: "host:ola" });
    const b = T.enquadrar({ id: 2, cmd: "host:ola" });
    const junto = new Uint8Array(a.length + b.length);
    junto.set(a, 0); junto.set(b, a.length);
    c.cru(junto);
    await new Promise((r) => setTimeout(r, 400));
    const ids = c.recebidas().map((m) => m.id).sort();
    assert.deepEqual(ids, [1, 2]);
  } finally { c.fechar(); c.matar(); }
});

test("cmd desconhecido devolve erro PELO FIO, nao derruba o host", async () => {
  const c = arrancar();
  try {
    const r = await c.pedir("nao:existe");
    assert.equal(r.ok, false);
    assert.equal(r.motivo, "cmd_desconhecido");
    // e o host continua vivo e a responder
    const r2 = await c.pedir("host:ola");
    assert.equal(r2.ok, true);
  } finally { c.fechar(); c.matar(); }
});

test("FIM A FIM pelo PROCESSO: pairing -> chave -> desafio -> handshake", async () => {
  const c = arrancar();
  try {
    const ini = await c.pedir("pairing:iniciar");
    assert.equal(ini.ok, true);
    assert.match(ini.codigo, /^\d{6}$/);

    // a extensao deriva a chave do codigo curto e envia-a (na vida real, ja' cifrada)
    const chave = await C.derivarChave(ini.codigo, ini.pedido, "inst-1", "m1");
    const bytes = Array.from(await C.exportarChave(chave));
    assert.equal(bytes.length, 32);

    const conf = await c.pedir("pairing:confirmar", { codigo: ini.codigo, chave: bytes, instalacao: "inst-1" });
    assert.equal(conf.ok, true, JSON.stringify(conf));
    assert.equal(conf.instalacao, "inst-1");

    const d = await c.pedir("handshake:desafio", { instalacao: "inst-1" });
    assert.equal(d.ok, true);
    assert.match(d.nonce, /^[0-9a-f]{64}$/);

    const prova = await C.assinarDesafio(chave, d.nonce, "inst-1");
    const h = await c.pedir("handshake:aceitar", { instalacao: "inst-1", nonce: d.nonce, prova });
    assert.equal(h.ok, true, JSON.stringify(h));
    assert.equal(h.instalacao, "inst-1");

    // REPLAY do MESMO desafio: o host recusa
    const h2 = await c.pedir("handshake:aceitar", { instalacao: "inst-1", nonce: d.nonce, prova });
    assert.equal(h2.ok, false);

    // e uma prova ERRADA tambem
    const d3 = await c.pedir("handshake:desafio", { instalacao: "inst-1" });
    const ruim = await c.pedir("handshake:aceitar", { instalacao: "inst-1", nonce: d3.nonce, prova: "00".repeat(32) });
    assert.equal(ruim.ok, false);
    assert.equal(ruim.motivo, "prova_invalida");
  } finally { c.fechar(); c.matar(); }
});

test("envelope de trabalho pelo processo, e o REPLAY recusado", async () => {
  const c = arrancar();
  try {
    const ini = await c.pedir("pairing:iniciar");
    const chave = await C.derivarChave(ini.codigo, ini.pedido, "inst-1", "m1");
    const bytes = Array.from(await C.exportarChave(chave));
    await c.pedir("pairing:confirmar", { codigo: ini.codigo, chave: bytes, instalacao: "inst-1" });
    const d = await c.pedir("handshake:desafio", { instalacao: "inst-1" });
    const prova = await C.assinarDesafio(chave, d.nonce, "inst-1");
    await c.pedir("handshake:aceitar", { instalacao: "inst-1", nonce: d.nonce, prova });

    const env = C.envelope({
      tipoCanal: "native", instalacao: "inst-1", missao: "m1",
      sequencia: 1, corpo: "trabalho", agora: Date.now(),
    });
    env.mac = await C.assinar(chave, env);
    const r = await c.pedir("envelope:aceitar", { envelope: env });
    assert.equal(r.ok, true, JSON.stringify(r));
    assert.equal(r.corpo, "trabalho");

    const r2 = await c.pedir("envelope:aceitar", { envelope: env });
    assert.equal(r2.ok, false, "o mesmo envelope duas vezes e' replay");
  } finally { c.fechar(); c.matar(); }
});

test("o processo ENCERRA quando o fluxo dessincroniza -- nao adivinha", async () => {
  const c = arrancar();
  const fim = new Promise((res) => c.proc.on("exit", (code) => res(code)));
  try {
    // um corpo bem enquadrado com JSON invalido: o fluxo morre
    const lixo = new TextEncoder().encode("{nao e' json");
    const b = new Uint8Array(4 + lixo.length);
    new DataView(b.buffer).setUint32(0, lixo.length, true);
    b.set(lixo, 4);
    c.cru(b);
    const code = await Promise.race([fim, new Promise((r) => setTimeout(() => r("timeout"), 4000))]);
    assert.equal(code, 2, "sai com 2 quando o fluxo morre");
    assert.ok(c.stderr().includes("erro de fluxo"), "e diz por que no stderr");
  } finally { c.matar(); }
});

test("a unidade: `criarHost` recusa campo a campo sem subprocesso", async () => {
  const host = H.criarHost(() => 1_800_000_000_000);
  assert.equal((await host.tratar(null)).motivo, "forma_invalida");
  assert.equal((await host.tratar({ cmd: "pairing:confirmar", codigo: "000000" })).motivo, "sem_pedido");
  assert.equal((await host.tratar({ cmd: "handshake:desafio", instalacao: "" })).motivo, "instalacao_ausente");
  assert.equal((await host.tratar({ cmd: "handshake:desafio", instalacao: "inst-x" })).ok, true);
  // instalar a chave a mao e ver o handshake recusar prova ausente
  const host2 = H.criarHost();
  const ini = await host2.tratar({ cmd: "pairing:iniciar" });
  const chave = await C.derivarChave(ini.codigo, ini.pedido, "inst-1");
  const bytes = await C.exportarChave(chave);
  await host2.tratar({ cmd: "pairing:confirmar", codigo: ini.codigo, chave: Array.from(bytes), instalacao: "inst-1" });
  assert.equal((await host2.tratar({ cmd: "handshake:aceitar", instalacao: "inst-1", nonce: "x" })).motivo, "prova_ausente");
});

test("`E-1043`: a consulta de aprovacao RESPONDE -- e responde que NAO verificou", async () => {
  // *Uma rota que responde «nao posso» e' melhor do que uma rota que nao existe: quem pergunta
  // fica a saber que perguntou ao sitio certo.*
  const host = H.criarHost();

  // 1) sem par emparelhado nao ha' sequer direito de perguntar
  const semPar = await host.tratar({ cmd: "aprovacao:consultar", fingerprint: "a".repeat(64), instalacao: "inst-1" });
  assert.equal(semPar.ok, false);
  assert.equal(semPar.motivo, "sem_dispositivo_pareado");

  // 2) emparelhar
  const ini = await host.tratar({ cmd: "pairing:iniciar" });
  const chave = await C.derivarChave(ini.codigo, ini.pedido, "inst-1");
  const bytes = await C.exportarChave(chave);
  const conf = await host.tratar({
    cmd: "pairing:confirmar", codigo: ini.codigo, chave: Array.from(bytes), instalacao: "inst-1",
  });
  assert.equal(conf.ok, true);

  // 3) fingerprint invalido
  assert.equal((await host.tratar({ cmd: "aprovacao:consultar", fingerprint: "curto", instalacao: "inst-1" })).motivo,
    "fingerprint_invalido");
  assert.equal((await host.tratar({ cmd: "aprovacao:consultar", instalacao: "inst-1" })).motivo,
    "fingerprint_invalido");

  // 4) instalacao desconhecida
  assert.equal((await host.tratar({ cmd: "aprovacao:consultar", fingerprint: "a".repeat(64), instalacao: "outra" })).motivo,
    "instalacao_desconhecida");

  // 5) e a resposta HONESTA
  const fp = "b".repeat(64);
  const r = await host.tratar({ cmd: "aprovacao:consultar", fingerprint: fp, instalacao: "inst-1" });
  assert.equal(r.ok, true);
  assert.equal(r.fingerprint, fp, "ecoa o fingerprint perguntado");
  // *A LINHA QUE IMPORTA: o Companion NUNCA diz que verificou.*
  assert.equal(r.verificada, false, "O COMPANION NAO CONSULTOU O STORE -- NAO PODE DIZER QUE SIM");
  assert.equal(r.motivo, "sem_rota_autenticavel_no_runtime");
  assert.equal(r.motivo, H.MOTIVO_SEM_ROTA, "o motivo e' a constante, nao uma string solta");
  // e diz o passo de desbloqueio, em vez de deixar a pergunta sem resposta
  assert.equal(typeof r.desbloqueio, "string");
  assert.ok(r.desbloqueio.includes("FindApproved"), r.desbloqueio);
});

test("`E-1043`: o Companion NUNCA afirma `verificada: true` -- em nenhuma entrada", async () => {
  // *Um campo que diz `verificada` sem ter sido verificado e' pior do que nao ter campo.*
  // Este teste varre TODAS as formas de chamar e confirma que a afirmacao nao existe.
  const host = H.criarHost();
  const ini = await host.tratar({ cmd: "pairing:iniciar" });
  const chave = await C.derivarChave(ini.codigo, ini.pedido, "inst-1");
  await host.tratar({ cmd: "pairing:confirmar", codigo: ini.codigo, chave: Array.from(await C.exportarChave(chave)), instalacao: "inst-1" });

  const tentativas = [
    { cmd: "aprovacao:consultar", fingerprint: "c".repeat(64), instalacao: "inst-1" },
    { cmd: "aprovacao:consultar", fingerprint: "c".repeat(64), instalacao: "inst-1", verificada: true },
    { cmd: "aprovacao:consultar", fingerprint: "c".repeat(64), instalacao: "inst-1", forcar: true },
    { cmd: "aprovacao:consultar", fingerprint: "c".repeat(64), instalacao: "inst-1", aprovado: true },
  ];
  for (const t of tentativas) {
    const r = await host.tratar(t);
    assert.equal(r.verificada, false, "pedido com " + JSON.stringify(t) + " deu verificada=" + r.verificada);
  }
  // e a constante de capacidades continua vazia -- *o Companion nao controla o PC.*
  assert.deepEqual([...H.CAPACIDADES], []);
});
