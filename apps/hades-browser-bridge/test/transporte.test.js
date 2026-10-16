/**
 * TRANSPORTE -- enquadramento e o protocolo completo pelo fio (`E-1027`).
 * O que se prova aqui: que o protocolo do `E-1023`/`E-1024`/`E-1025` atravessa
 * um transporte real, partido em pedacos de 1 byte, e continua correcto.
 */
import test from "node:test";
import assert from "node:assert/strict";

globalThis.crypto ??= (await import("node:crypto")).webcrypto;

const C = await import("../src/canal_local.js");
const T = await import("../src/transporte.js");

const AGORA = 1_800_000_000_000;

/* ------------------------------------------------------------- enquadramento */

test("o cabecalho e' 4 bytes LITTLE-ENDIAN -- e nao por acaso", () => {
  const m = { a: 1 };
  const n = new TextEncoder().encode(JSON.stringify(m)).length; // MEDIDO, nao contado a mao
  const b = T.enquadrar(m);
  assert.equal(n, 7, '`{"a":1}` tem 7 bytes');
  assert.equal(b.length, T.CABECALHO_BYTES + n);
  assert.deepEqual(Array.from(b.subarray(0, 4)), [n, 0, 0, 0], 'LE: o byte baixo primeiro');
  assert.equal(new DataView(b.buffer).getUint32(0, true), n);
  // se fosse big-endian, o outro lado leria 0x07000000 = 117 MB e chamaria "longo"
  assert.notEqual(new DataView(b.buffer).getUint32(0, false), n);
  // e um comprimento com bits altos prova a ordem nos 4 bytes
  // 3a vez que conto bytes a mao e erro: o corpo mede-se, nao se adivinha.
  const g = T.enquadrar({ t: "x".repeat(300) });
  const ng = new TextEncoder().encode(JSON.stringify({ t: "x".repeat(300) })).length;
  assert.ok(ng > 255, "o comprimento usa mais do que um byte");
  assert.deepEqual(Array.from(g.subarray(0, 4)),
    [ng & 0xff, (ng >> 8) & 0xff, (ng >> 16) & 0xff, (ng >> 24) & 0xff]);
  assert.equal(new DataView(g.buffer).getUint32(0, true), ng);
  assert.ok(Array.from(g.subarray(0, 4))[1] > 0, "o 2o byte nao e' zero: e' little-endian");
});

test("round-trip: o que entra e' o que sai", () => {
  const casos = [
    { a: 1 },
    { texto: "acentuacao: ção, ã, é, 😀" },
    { aninhado: { lista: [1, 2, { x: null }], vazio: {} } },
    { numero: 1234567890, booleano: false },
  ];
  for (const c of casos) {
    const r = T.desenquadrar(T.enquadrar(c));
    assert.deepEqual(r.mensagens, [c]);
    assert.equal(r.resto.length, 0, "nada fica pendente");
  }
});

test("UTF-8 de varios bytes: o comprimento e' em BYTES, nao em caracteres", () => {
  const m = { t: "ção" };
  const b = T.enquadrar(m);
  const texto = JSON.stringify(m);
  assert.ok(texto.length < new TextEncoder().encode(texto).length, "ha' multi-byte");
  assert.equal(new DataView(b.buffer).getUint32(0, true), new TextEncoder().encode(texto).length);
  assert.deepEqual(T.desenquadrar(b).mensagens, [m]);
});

test("recusa: nao-objeto, vazio e gigante", () => {
  assert.throws(() => T.enquadrar(null), (e) => e.codigo === T.ERROS.FORMA);
  assert.throws(() => T.enquadrar("texto"), (e) => e.codigo === T.ERROS.FORMA);
  assert.throws(() => T.enquadrar([1, 2]), (e) => e.codigo === T.ERROS.FORMA);
  const gigante = { t: "x".repeat(T.MAX_CORPO_BYTES) };
  assert.throws(() => T.enquadrar(gigante), (e) => e.codigo === T.ERROS.LONGO);
  assert.throws(() => T.desenquadrar("nao e' bytes"), (e) => e.codigo === T.ERROS.FORMA);
});

/* ------------------------------------------------------------ desenquadrar */

test("corpo PARCIAL nao e' erro: e' o normal de um fluxo", () => {
  const b = T.enquadrar({ oi: 1 });
  const r = T.desenquadrar(b.subarray(0, b.length - 1));
  assert.deepEqual(r.mensagens, [], "ainda nao ha' mensagem");
  assert.equal(r.resto.length, b.length - 1, "os bytes ficam guardados");
  // e quando o resto chega, a mensagem aparece
  const r2 = T.desenquadrar(new Uint8Array([...r.resto, b[b.length - 1]]));
  assert.deepEqual(r2.mensagens, [{ oi: 1 }]);
  assert.equal(r2.resto.length, 0);
});

test("cabecalho sozinho e' um corpo parcial de zero bytes -- espera, nao lanca", () => {
  const r = T.desenquadrar(new Uint8Array([5, 0, 0, 0]));
  assert.deepEqual(r.mensagens, []);
  assert.equal(r.resto.length, 4);
});

test("comprimento ZERO e' recusado: uma mensagem sem corpo nao tem JSON", () => {
  assert.throws(() => T.desenquadrar(new Uint8Array([0, 0, 0, 0])),
    (e) => e.codigo === T.ERROS.VAZIO);
});

test("comprimento GIGANTE e' recusado ANTES de esperar os bytes", () => {
  // 300 MB anunciados: sem este limite, o host alocaria/esperaria 300 MB
  const b = new Uint8Array(4);
  new DataView(b.buffer).setUint32(0, 300 * 1024 * 1024, true);
  assert.throws(() => T.desenquadrar(b), (e) => e.codigo === T.ERROS.LONGO);
});

test("JSON invalido e' erro do FLUXO -- nao ha como saber onde acaba a mensagem", () => {
  const corpo = new TextEncoder().encode("{isto nao e' json");
  const b = new Uint8Array(4 + corpo.length);
  new DataView(b.buffer).setUint32(0, corpo.length, true);
  b.set(corpo, 4);
  assert.throws(() => T.desenquadrar(b), (e) => e.codigo === T.ERROS.JSON);
});

test("UTF-8 invalido e' recusado (fatal: true), nao substituido por U+FFFD", () => {
  const corpo = new Uint8Array([0xff, 0xfe, 0xfd]);
  const b = new Uint8Array(4 + 3);
  new DataView(b.buffer).setUint32(0, 3, true);
  b.set(corpo, 4);
  assert.throws(() => T.desenquadrar(b), (e) => e.codigo === T.ERROS.JSON);
});

test("JSON VALIDO mas que nao e' objeto: recusado campo a campo", () => {
  for (const bruto of ['"texto"', "42", "null", "[1,2]"]) {
    const corpo = new TextEncoder().encode(bruto);
    const b = new Uint8Array(4 + corpo.length);
    new DataView(b.buffer).setUint32(0, corpo.length, true);
    b.set(corpo, 4);
    assert.throws(() => T.desenquadrar(b), (e) => e.codigo === T.ERROS.FORMA, bruto);
  }
});

/* ------------------------------------------------------------- canal em memoria */

test("canal em memoria: 5 mensagens seguidas chegam inteiras e em ordem", () => {
  const c = T.canalEmMemoria();
  for (let i = 0; i < 5; i++) c.enviar({ i });
  assert.deepEqual(c.recebidas(), [{ i: 0 }, { i: 1 }, { i: 2 }, { i: 3 }, { i: 4 }]);
  assert.equal(c.pendentes(), 0);
});

test("A PROVA DURA: o mesmo fluxo partido em PEDACOS DE 1 BYTE", () => {
  const c = T.canalEmMemoria();
  const tudo = T.enquadrar({ oi: "pedaco a pedaco, c/ acento" });
  for (const byte of tudo) c.escrever(new Uint8Array([byte]));
  assert.deepEqual(c.recebidas(), [{ oi: "pedaco a pedaco, c/ acento" }],
    "o desenquadrador tem de remontar o que chega a conta-gotas");
  assert.equal(c.pendentes(), 0);
});

test("duas mensagens COLADAS no mesmo pacote saem as duas", () => {
  const c = T.canalEmMemoria();
  const a = T.enquadrar({ a: 1 });
  const b = T.enquadrar({ b: 2 });
  const junto = new Uint8Array(a.length + b.length);
  junto.set(a, 0);
  junto.set(b, a.length);
  c.escrever(junto);
  assert.deepEqual(c.recebidas(), [{ a: 1 }, { b: 2 }]);
});

test("LIXO NO INICIO nao e' detectavel -- e o teste mede isso em vez de o exigir", () => {
  // Um enquadramento de comprimento nao tem numero magico: qualquer 4 bytes sao um
  // cabecalho plausivel. Quem dessincroniza so' o descobre pelo conteudo. Exigir
  // deteccao aqui seria exigir adivinhacao.
  const c = T.canalEmMemoria();
  const bom = T.enquadrar({ ok: 2 });
  c.escrever(new Uint8Array([0]));            // 1 byte de lixo
  assert.equal(c.recebidas().length, 0, "com 1 byte nao ha' cabecalho completo");
  c.escrever(bom);                            // o lixo + o bom
  // o que o desenquadrador NAO faz: devolver { ok: 2 } como se nada tivesse passado
  const r = c.recebidas();
  assert.ok(!r.some((m) => m.ok === 2), "nao adivinha a mensagem boa atras do lixo");
});

test("um corpo BEM enquadrado mas com JSON invalido LANCA -- o fluxo morre", () => {
  const c = T.canalEmMemoria();
  const lixo = new TextEncoder().encode("{nao e' json");
  const b = new Uint8Array(4 + lixo.length);
  new DataView(b.buffer).setUint32(0, lixo.length, true);
  b.set(lixo, 4);
  assert.throws(() => c.escrever(b), (e) => e.codigo === T.ERROS.JSON);
  // e depois desse erro o canal nao devolve uma mensagem inventada
  assert.deepEqual(c.recebidas(), []);
});

/* ------------------------------------- O PROTOCOLO COMPLETO, FIM A FIM, NO FIO */

test("FIM A FIM: pairing -> handshake -> envelope, tudo pelo transporte", async () => {
  const c = T.canalEmMemoria();

  // 1) quem fala: pairing e derivacao da chave
  const pedido = C.abrirPairing(AGORA);
  c.enviar({ passo: "inicio", pedido: pedido.pedido });
  const chave = await C.derivarChave(pedido.codigo, pedido.pedido, "inst-1", "m1");
  const bytes = await C.exportarChave(chave);
  const estado = C.estadoInicial();
  assert.equal(C.registrarDispositivo(estado, "inst-1", bytes, AGORA).ok, true);
  assert.equal(C.confirmarPairing(pedido, pedido.codigo, AGORA).ok, true);
  assert.equal(C.consumirCodigo(pedido).ok, true, "o codigo morre depois de usado");

  // 2) o host manda o desafio PELO FIO
  const d = C.abrirDesafio("inst-1", AGORA);
  c.enviar({ passo: "desafio", nonce: d.desafio.nonce });
  const recebido = c.recebidas().at(-1);
  assert.equal(recebido.nonce, d.desafio.nonce);

  // 3) a extensao prova posse e o host aceita
  estado.desafios[d.desafio.nonce] = d.desafio;
  const prova = await C.provarPosse(chave, d.desafio, AGORA);
  c.enviar({ passo: "handshake", instalacao: "inst-1", nonce: d.desafio.nonce, prova });
  const h = c.recebidas().at(-1);
  const r = await C.aceitarHandshake(h, estado, AGORA);
  assert.equal(r.ok, true, "o handshake passa pelo fio e e' aceito");
  assert.equal(r.instalacao, "inst-1");

  // 4) e so' entao um envelope de trabalho
  const e = await enviarEnvelope(chave, { corpo: "trabalho" });
  c.enviar(e);
  const chegado = c.recebidas().at(-1);
  const a = await C.aceitar(chegado, estado, AGORA);
  assert.equal(a.ok, true, "o envelope chega inteiro e e' aceito");
  assert.equal(a.corpo, "trabalho");

  // 5) REPLAY pelo fio: reenviar os MESMOS bytes e' recusado
  c.enviar(e);
  const segunda = c.recebidas().at(-1);
  const r2 = await C.aceitar(segunda, estado, AGORA);
  assert.equal(r2.ok, false);
  // o motivo real, lido do contrato: nao `replay` (nome que eu inventei)
  assert.equal(r2.motivo, "replay_ou_fora_de_ordem");

  // 6) nada ficou a meio
  assert.equal(c.pendentes(), 0, "o fluxo inteiro foi consumido");
  assert.equal(c.recebidas().length, 5);
});

async function enviarEnvelope(chave, { corpo }) {
  const base = C.envelope({
    tipoCanal: "native", instalacao: "inst-1", missao: "m1",
    sequencia: 1, corpo, agora: AGORA,
  });
  base.mac = await C.assinar(chave, base);
  return base;
}
